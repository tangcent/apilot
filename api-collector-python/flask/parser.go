package flask

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	tree_sitter "github.com/tree-sitter/go-tree-sitter"
	python "github.com/tree-sitter/tree-sitter-python/bindings/go"

	collector "github.com/tangcent/apilot/api-collector"
)

func Parse(sourceDir string) ([]collector.ApiEndpoint, error) {
	pyFiles, err := discoverPythonFiles(sourceDir)
	if err != nil || len(pyFiles) == 0 {
		return nil, nil
	}

	ch := make(chan fileResult, len(pyFiles))
	var wg sync.WaitGroup

	for _, path := range pyFiles {
		wg.Add(1)
		go func(filePath string) {
			defer wg.Done()
			res := processFile(filePath)
			ch <- res
		}(path)
	}

	go func() {
		wg.Wait()
		close(ch)
	}()

	var allEndpoints []collector.ApiEndpoint
	for res := range ch {
		if res.err != nil {
			continue
		}
		allEndpoints = append(allEndpoints, res.endpoints...)
	}

	if len(allEndpoints) == 0 {
		return nil, nil
	}

	return allEndpoints, nil
}

type fileResult struct {
	endpoints []collector.ApiEndpoint
	err       error
}

func discoverPythonFiles(sourceDir string) ([]string, error) {
	var pyFiles []string
	err := filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() && strings.HasSuffix(path, ".py") {
			pyFiles = append(pyFiles, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return pyFiles, nil
}

func processFile(filePath string) fileResult {
	source, err := os.ReadFile(filePath)
	if err != nil {
		return fileResult{err: fmt.Errorf("failed to read file: %w", err)}
	}

	p := tree_sitter.NewParser()
	defer p.Close()

	lang := tree_sitter.NewLanguage(python.Language())
	if err := p.SetLanguage(lang); err != nil {
		return fileResult{err: fmt.Errorf("failed to set language: %w", err)}
	}

	tree := p.Parse(source, nil)
	if tree == nil {
		return fileResult{}
	}
	defer tree.Close()

	rootNode := tree.RootNode()
	endpoints := extractEndpoints(rootNode, source)

	return fileResult{endpoints: endpoints}
}

func extractEndpoints(rootNode *tree_sitter.Node, source []byte) []collector.ApiEndpoint {
	var endpoints []collector.ApiEndpoint

	for i := uint(0); i < rootNode.ChildCount(); i++ {
		child := rootNode.Child(i)
		if child.Kind() != "decorated_definition" {
			continue
		}

		eps := extractDecoratedDefinition(child, source)
		endpoints = append(endpoints, eps...)
	}

	return endpoints
}

func extractDecoratedDefinition(node *tree_sitter.Node, source []byte) []collector.ApiEndpoint {
	var decorator *tree_sitter.Node
	var funcDef *tree_sitter.Node

	for i := uint(0); i < node.ChildCount(); i++ {
		child := node.Child(i)
		switch child.Kind() {
		case "decorator":
			decorator = child
		case "function_definition":
			funcDef = child
		}
	}

	if decorator == nil || funcDef == nil {
		return nil
	}

	routeInfo := extractDecoratorInfo(decorator, source)
	if routeInfo.path == "" {
		return nil
	}

	funcName := extractFunctionName(funcDef, source)
	description := extractDocstring(funcDef, source)

	var endpoints []collector.ApiEndpoint

	for _, method := range routeInfo.methods {
		path := convertFlaskPathToStandard(routeInfo.path)
		ep := &collector.ApiEndpoint{
			Name:        funcName,
			Path:        path,
			Method:      strings.ToUpper(method),
			Protocol:    "http",
			Description: description,
		}

		pathParams := extractPathParams(path)
		pathParamNames := make(map[string]bool)
		for _, p := range pathParams {
			pathParamNames[p.name] = true
		}

		params := extractFunctionParameters(funcDef, source, path)

		paramSet := make(map[string]bool)
		var allParams []collector.ApiParameter

		for _, p := range pathParams {
			allParams = append(allParams, collector.ApiParameter{
				Name:     p.name,
				In:       "path",
				Required: true,
				Type:     "text",
			})
			paramSet[p.name+"|path"] = true
		}

		for _, p := range params {
			if pathParamNames[p.name] && p.in == "query" {
				p.in = "path"
				p.required = true
			}
			key := p.name + "|" + p.in
			if !paramSet[key] {
				allParams = append(allParams, collector.ApiParameter{
					Name:     p.name,
					In:       p.in,
					Required: p.required,
					Type:     p.typ,
				})
				paramSet[key] = true
			}
		}

		if len(allParams) > 0 {
			ep.Parameters = allParams
		}

		endpoints = append(endpoints, *ep)
	}

	return endpoints
}

type routeInfo struct {
	path    string
	methods []string
}

func extractDecoratorInfo(decorator *tree_sitter.Node, source []byte) routeInfo {
	for i := uint(0); i < decorator.ChildCount(); i++ {
		child := decorator.Child(i)
		if child.Kind() == "@" {
			continue
		}

		info := resolveDecoratorCall(child, source)
		if info.path != "" {
			return info
		}
	}
	return routeInfo{}
}

func resolveDecoratorCall(node *tree_sitter.Node, source []byte) routeInfo {
	if node.Kind() == "call" {
		return resolveCallExpression(node, source)
	}
	if node.Kind() == "attribute" {
		return resolveAttribute(node, source)
	}
	for i := uint(0); i < node.ChildCount(); i++ {
		child := node.Child(i)
		info := resolveDecoratorCall(child, source)
		if info.path != "" {
			return info
		}
	}
	return routeInfo{}
}

func resolveCallExpression(callNode *tree_sitter.Node, source []byte) routeInfo {
	var isRoute bool
	var path string
	var methods []string

	for i := uint(0); i < callNode.ChildCount(); i++ {
		child := callNode.Child(i)
		if child.Kind() == "attribute" {
			attr := extractAttributeName(child, source)
			if attr == "route" {
				isRoute = true
			}
		}
		if child.Kind() == "argument_list" {
			path, methods = extractArguments(child, source)
		}
	}

	if !isRoute || path == "" {
		return routeInfo{}
	}

	if len(methods) == 0 {
		methods = []string{"GET"}
	}

	return routeInfo{path: path, methods: methods}
}

func resolveAttribute(attrNode *tree_sitter.Node, source []byte) routeInfo {
	attr := extractAttributeName(attrNode, source)
	if attr == "route" {
		return routeInfo{}
	}
	return routeInfo{}
}

func extractAttributeName(attrNode *tree_sitter.Node, source []byte) string {
	var attr string

	for i := uint(0); i < attrNode.ChildCount(); i++ {
		child := attrNode.Child(i)
		if child.Kind() == "." {
			continue
		}
		if child.Kind() == "identifier" {
			attr = child.Utf8Text(source)
		}
	}

	return attr
}

func extractArguments(argListNode *tree_sitter.Node, source []byte) (path string, methods []string) {
	for i := uint(0); i < argListNode.ChildCount(); i++ {
		child := argListNode.Child(i)
		if child.Kind() == "(" || child.Kind() == ")" || child.Kind() == "," {
			continue
		}

		if child.Kind() == "string" && path == "" {
			path = unquotePythonString(child.Utf8Text(source))
		}

		if child.Kind() == "keyword_argument" {
			methods = extractMethodsArgument(child, source)
		}
	}

	return path, methods
}

func extractMethodsArgument(kwArgNode *tree_sitter.Node, source []byte) []string {
	var methods []string

	for i := uint(0); i < kwArgNode.ChildCount(); i++ {
		child := kwArgNode.Child(i)
		if child.Kind() == "identifier" {
			if child.Utf8Text(source) != "methods" {
				return nil
			}
		}
		if child.Kind() == "list" {
			methods = extractMethodList(child, source)
		}
	}

	return methods
}

func extractMethodList(listNode *tree_sitter.Node, source []byte) []string {
	var methods []string

	for i := uint(0); i < listNode.ChildCount(); i++ {
		child := listNode.Child(i)
		if child.Kind() == "[" || child.Kind() == "]" || child.Kind() == "," {
			continue
		}
		if child.Kind() == "string" {
			method := unquotePythonString(child.Utf8Text(source))
			methods = append(methods, strings.ToUpper(method))
		}
	}

	return methods
}

func extractFunctionName(funcDef *tree_sitter.Node, source []byte) string {
	for i := uint(0); i < funcDef.ChildCount(); i++ {
		child := funcDef.Child(i)
		if child.Kind() == "identifier" {
			return child.Utf8Text(source)
		}
	}
	return ""
}

func extractDocstring(funcDef *tree_sitter.Node, source []byte) string {
	for i := uint(0); i < funcDef.ChildCount(); i++ {
		child := funcDef.Child(i)
		if child.Kind() == "block" {
			return extractDocstringFromBlock(child, source)
		}
	}
	return ""
}

func extractDocstringFromBlock(blockNode *tree_sitter.Node, source []byte) string {
	for i := uint(0); i < blockNode.ChildCount(); i++ {
		child := blockNode.Child(i)
		if child.Kind() == "expression_statement" {
			for j := uint(0); j < child.ChildCount(); j++ {
				subChild := child.Child(j)
				if subChild.Kind() == "string" {
					return unquotePythonString(subChild.Utf8Text(source))
				}
			}
		}
		break
	}
	return ""
}

type funcParam struct {
	name     string
	in       string
	required bool
	typ      string
}

func extractFunctionParameters(funcDef *tree_sitter.Node, source []byte, path string) []funcParam {
	var params []funcParam

	for i := uint(0); i < funcDef.ChildCount(); i++ {
		child := funcDef.Child(i)
		if child.Kind() == "parameters" {
			params = extractParameters(child, source, path)
			break
		}
	}

	return params
}

func extractParameters(paramsNode *tree_sitter.Node, source []byte, path string) []funcParam {
	var params []funcParam

	for i := uint(0); i < paramsNode.ChildCount(); i++ {
		child := paramsNode.Child(i)
		if child.Kind() == "(" || child.Kind() == ")" || child.Kind() == "," {
			continue
		}

		if child.Kind() == "identifier" {
			name := child.Utf8Text(source)
			in := "query"
			if isNameInPath(name, path) {
				in = "path"
			}
			params = append(params, funcParam{
				name:     name,
				in:       in,
				required: in == "path",
				typ:      "text",
			})
		} else if child.Kind() == "typed_parameter" || child.Kind() == "default_parameter" || child.Kind() == "typed_default_parameter" {
			param := extractSingleParameter(child, source, path)
			if param.name != "" {
				params = append(params, param)
			}
		}
	}

	return params
}

func extractSingleParameter(paramNode *tree_sitter.Node, source []byte, path string) funcParam {
	p := funcParam{
		in:       "query",
		required: true,
		typ:      "text",
	}

	for i := uint(0); i < paramNode.ChildCount(); i++ {
		child := paramNode.Child(i)

		switch child.Kind() {
		case "identifier":
			if p.name == "" {
				p.name = child.Utf8Text(source)
			}
		case "=":
			p.required = false
		}
	}

	if p.name != "" && isNameInPath(p.name, path) && p.in == "query" {
		p.in = "path"
		p.required = true
	}

	return p
}

type pathParamInfo struct {
	name string
}

func extractPathParams(path string) []pathParamInfo {
	var params []pathParamInfo
	for _, segment := range strings.Split(path, "/") {
		if strings.HasPrefix(segment, "{") && strings.HasSuffix(segment, "}") {
			name := segment[1 : len(segment)-1]
			params = append(params, pathParamInfo{name: name})
		}
	}
	return params
}

func isNameInPath(name string, path string) bool {
	target := "{" + name + "}"
	return strings.Contains(path, target)
}

func convertFlaskPathToStandard(flaskPath string) string {
	result := flaskPath

	result = strings.ReplaceAll(result, "<string:", "{")
	result = strings.ReplaceAll(result, "<int:", "{")
	result = strings.ReplaceAll(result, "<float:", "{")
	result = strings.ReplaceAll(result, "<path:", "{")
	result = strings.ReplaceAll(result, "<uuid:", "{")
	result = strings.ReplaceAll(result, "<", "{")
	result = strings.ReplaceAll(result, ">", "}")

	return result
}

func unquotePythonString(s string) string {
	if len(s) >= 3 && (strings.HasPrefix(s, `"""`) || strings.HasPrefix(s, `'''`)) {
		quote := s[:3]
		if strings.HasSuffix(s, quote) {
			return s[3 : len(s)-3]
		}
	}
	if len(s) >= 2 {
		if (s[0] == '"' && s[len(s)-1] == '"') || (s[0] == '\'' && s[len(s)-1] == '\'') {
			return s[1 : len(s)-1]
		}
	}
	return s
}
