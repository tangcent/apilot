// Package fastapi parses FastAPI decorated route functions using tree-sitter-python.
// It walks Python source files for @app.get, @app.post, @router.get, etc. decorators
// and extracts endpoint metadata including path, HTTP method, function parameters,
// docstrings, and structured request/response type schemas resolved from Pydantic
// BaseModel definitions and Python type annotations.
package fastapi

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	tree_sitter "github.com/tree-sitter/go-tree-sitter"
	python "github.com/tree-sitter/tree-sitter-python/bindings/go"

	collector "github.com/tangcent/apilot/api-collector"
	model "github.com/tangcent/apilot/api-model"
)

var httpMethods = map[string]bool{
	"get":     true,
	"post":    true,
	"put":     true,
	"delete":  true,
	"patch":   true,
	"head":    true,
	"options": true,
}

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

	allModels := make(map[string]PydanticModel)
	var allRawEndpoints []rawEndpointInfo

	for res := range ch {
		if res.err != nil {
			continue
		}
		for k, v := range res.models {
			allModels[k] = v
		}
		allRawEndpoints = append(allRawEndpoints, res.rawEndpoints...)
	}

	if len(allRawEndpoints) == 0 {
		return nil, nil
	}

	typeResolver := NewPythonTypeResolver(allModels)

	var allEndpoints []collector.ApiEndpoint
	for _, raw := range allRawEndpoints {
		ep := buildEndpoint(raw, typeResolver)
		if ep != nil {
			allEndpoints = append(allEndpoints, *ep)
		}
	}

	if len(allEndpoints) == 0 {
		return nil, nil
	}

	return allEndpoints, nil
}

type rawEndpointInfo struct {
	method        string
	path          string
	funcName      string
	description   string
	params        []funcParam
	responseModel string
	returnType    string
}

type fileResult struct {
	rawEndpoints []rawEndpointInfo
	models       map[string]PydanticModel
	err          error
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
	models := ExtractPydanticModels(rootNode, source)
	rawEndpoints := extractRawEndpoints(rootNode, source)

	return fileResult{
		models:       models,
		rawEndpoints: rawEndpoints,
	}
}

func extractRawEndpoints(rootNode *tree_sitter.Node, source []byte) []rawEndpointInfo {
	var endpoints []rawEndpointInfo

	for i := uint(0); i < rootNode.ChildCount(); i++ {
		child := rootNode.Child(i)
		if child.Kind() != "decorated_definition" {
			continue
		}

		raw := extractRawDecoratedDefinition(child, source)
		if raw != nil {
			endpoints = append(endpoints, *raw)
		}
	}

	return endpoints
}

func extractRawDecoratedDefinition(node *tree_sitter.Node, source []byte) *rawEndpointInfo {
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

	method, path := extractDecoratorInfo(decorator, source)
	if method == "" || path == "" {
		return nil
	}

	funcName := extractFunctionName(funcDef, source)
	description := extractDocstring(funcDef, source)
	params := extractFunctionParameters(funcDef, source, path)
	responseModel := extractResponseModelFromDecorator(decorator, source)
	returnType := extractReturnType(funcDef, source)

	return &rawEndpointInfo{
		method:        method,
		path:          path,
		funcName:      funcName,
		description:   description,
		params:        params,
		responseModel: responseModel,
		returnType:    returnType,
	}
}

func buildEndpoint(raw rawEndpointInfo, typeResolver *PythonTypeResolver) *collector.ApiEndpoint {
	ep := &collector.ApiEndpoint{
		Name:        raw.funcName,
		Path:        raw.path,
		Method:      strings.ToUpper(raw.method),
		Protocol:    "http",
		Description: raw.description,
	}

	pathParams := extractPathParams(raw.path)
	pathParamNames := make(map[string]bool)
	for _, p := range pathParams {
		pathParamNames[p.name] = true
	}

	paramSet := make(map[string]bool)
	var allParams []collector.ApiParameter
	var bodyParams []funcParam

	for _, p := range pathParams {
		allParams = append(allParams, collector.ApiParameter{
			Name:     p.name,
			In:       "path",
			Required: true,
			Type:     "text",
		})
		paramSet[p.name+"|path"] = true
	}

	for _, p := range raw.params {
		if pathParamNames[p.name] && p.in == "query" {
			p.in = "path"
			p.required = true
		}

		if p.in == "body" && p.typeAnnotation != "" {
			bodyParams = append(bodyParams, p)
			continue
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

	if len(bodyParams) == 1 && bodyParams[0].typeAnnotation != "" {
		resolvedBody := typeResolver.Resolve(bodyParams[0].typeAnnotation)
		ep.RequestBody = &collector.ApiBody{
			MediaType: "application/json",
			Body:      resolvedBody,
		}
	} else if len(bodyParams) > 1 {
		fields := make(map[string]*model.FieldModel)
		for _, bp := range bodyParams {
			var fieldModel *model.ObjectModel
			if bp.typeAnnotation != "" {
				fieldModel = typeResolver.Resolve(bp.typeAnnotation)
			} else {
				fieldModel = model.SingleModel(model.JsonTypeString)
			}
			fields[bp.name] = &model.FieldModel{
				Model:    fieldModel,
				Required: bp.required,
			}
		}
		ep.RequestBody = &collector.ApiBody{
			MediaType: "application/json",
			Body: &model.ObjectModel{
				Kind:     model.KindObject,
				TypeName: "object",
				Fields:   fields,
			},
		}
	}

	responseType := raw.responseModel
	if responseType == "" && raw.returnType != "" {
		responseType = raw.returnType
	}

	if responseType != "" {
		resolvedResponse := typeResolver.Resolve(responseType)
		if resolvedResponse != nil && !resolvedResponse.IsNull() {
			ep.Response = &collector.ApiBody{
				MediaType: "application/json",
				Body:      resolvedResponse,
			}
		}
	}

	return ep
}

func extractDecoratorInfo(decorator *tree_sitter.Node, source []byte) (method string, path string) {
	for i := uint(0); i < decorator.ChildCount(); i++ {
		child := decorator.Child(i)
		if child.Kind() == "@" {
			continue
		}

		method, path = resolveDecoratorCall(child, source)
		if method != "" {
			return method, path
		}
	}
	return "", ""
}

func resolveDecoratorCall(node *tree_sitter.Node, source []byte) (method string, path string) {
	if node.Kind() == "call" {
		return resolveCallExpression(node, source)
	}
	if node.Kind() == "attribute" {
		return resolveAttribute(node, source)
	}
	if node.Kind() == "identifier" {
		return "", ""
	}
	for i := uint(0); i < node.ChildCount(); i++ {
		child := node.Child(i)
		m, p := resolveDecoratorCall(child, source)
		if m != "" {
			return m, p
		}
	}
	return "", ""
}

func resolveCallExpression(callNode *tree_sitter.Node, source []byte) (method string, path string) {
	for i := uint(0); i < callNode.ChildCount(); i++ {
		child := callNode.Child(i)
		if child.Kind() == "attribute" {
			m, _ := resolveAttribute(child, source)
			if m != "" {
				method = m
			}
		}
		if child.Kind() == "argument_list" {
			path = extractFirstStringArgument(child, source)
		}
	}
	return method, path
}

func resolveAttribute(attrNode *tree_sitter.Node, source []byte) (method string, path string) {
	var obj string
	var attr string

	for i := uint(0); i < attrNode.ChildCount(); i++ {
		child := attrNode.Child(i)
		if child.Kind() == "." {
			continue
		}
		if child.Kind() == "identifier" {
			if obj == "" {
				obj = child.Utf8Text(source)
			} else {
				attr = child.Utf8Text(source)
			}
		}
		if child.Kind() == "attribute" {
			_, _ = resolveAttribute(child, source)
		}
	}

	lowerAttr := strings.ToLower(attr)
	if httpMethods[lowerAttr] {
		return lowerAttr, ""
	}
	return "", ""
}

func extractFirstStringArgument(argListNode *tree_sitter.Node, source []byte) string {
	for i := uint(0); i < argListNode.ChildCount(); i++ {
		child := argListNode.Child(i)
		if child.Kind() == "(" || child.Kind() == ")" || child.Kind() == "," {
			continue
		}
		if child.Kind() == "string" {
			return unquotePythonString(child.Utf8Text(source))
		}
		if child.Kind() == "keyword_argument" {
			for j := uint(0); j < child.ChildCount(); j++ {
				kwChild := child.Child(j)
				if kwChild.Kind() == "string" {
					text := kwChild.Utf8Text(source)
					if strings.Contains(text, "/") {
						return unquotePythonString(text)
					}
				}
			}
		}
	}
	return ""
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
	name           string
	in             string
	required       bool
	typ            string
	typeAnnotation string
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

		if child.Kind() == "typed_parameter" || child.Kind() == "default_parameter" || child.Kind() == "typed_default_parameter" {
			param := extractSingleParameter(child, source, path)
			if param.name != "" && !isSpecialParam(param.name) {
				params = append(params, param)
			}
		} else if child.Kind() == "identifier" {
			name := child.Utf8Text(source)
			if !isSpecialParam(name) {
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
			}
		} else if child.Kind() == "list_splat_pattern" {
			for j := uint(0); j < child.ChildCount(); j++ {
				subChild := child.Child(j)
				if subChild.Kind() == "identifier" {
					name := subChild.Utf8Text(source)
					if !isSpecialParam(name) {
						params = append(params, funcParam{
							name:     name,
							in:       "query",
							required: false,
							typ:      "text",
						})
					}
				}
			}
		} else if child.Kind() == "dictionary_splat_pattern" {
			for j := uint(0); j < child.ChildCount(); j++ {
				subChild := child.Child(j)
				if subChild.Kind() == "identifier" {
					name := subChild.Utf8Text(source)
					if !isSpecialParam(name) {
						params = append(params, funcParam{
							name:     name,
							in:       "query",
							required: false,
							typ:      "text",
						})
					}
				}
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

	var defaultCall string
	var defaultHasEllipsis bool

	for i := uint(0); i < paramNode.ChildCount(); i++ {
		child := paramNode.Child(i)

		switch child.Kind() {
		case "identifier":
			if p.name == "" {
				p.name = child.Utf8Text(source)
			}
		case "type":
			typeText := child.Utf8Text(source)
			p = applyTypeAnnotation(p, typeText)
			if p.typeAnnotation == "" {
				p.typeAnnotation = typeText
			}
		case "call":
			defaultCall, defaultHasEllipsis = extractCallInfo(child, source)
		case "=":
			p.required = false
		}
	}

	if defaultCall != "" {
		p = applyDefaultCall(p, defaultCall)
		if defaultHasEllipsis {
			p.required = true
		}
	} else if p.name != "" && isNameInPath(p.name, path) && p.in == "query" {
		p.in = "path"
		p.required = true
	}

	if p.typeAnnotation != "" && p.in != "body" && p.in != "path" && p.in != "header" && p.in != "cookie" && p.in != "form" {
		if IsPydanticModelType(p.typeAnnotation) {
			p.in = "body"
		}
	}

	return p
}

func IsPydanticModelType(typeText string) bool {
	baseName, _ := ParsePythonGenericType(typeText)
	if pythonPrimitives[baseName] != "" {
		return false
	}
	if pythonCollectionTypes[baseName] || pythonMapTypes[baseName] {
		return false
	}
	if baseName == "list" || baseName == "dict" || baseName == "set" || baseName == "tuple" {
		return false
	}
	if baseName == "Optional" || baseName == "Union" {
		return false
	}
	lower := strings.ToLower(baseName)
	if lower == "str" || lower == "int" || lower == "float" || lower == "bool" {
		return false
	}
	if lower == "string" || lower == "bytes" {
		return false
	}
	return true
}

func extractCallInfo(callNode *tree_sitter.Node, source []byte) (identifier string, hasEllipsis bool) {
	for i := uint(0); i < callNode.ChildCount(); i++ {
		child := callNode.Child(i)
		if child.Kind() == "identifier" && identifier == "" {
			identifier = child.Utf8Text(source)
		}
		if child.Kind() == "attribute" {
			for j := uint(0); j < child.ChildCount(); j++ {
				attrChild := child.Child(j)
				if attrChild.Kind() == "identifier" {
					identifier = attrChild.Utf8Text(source)
				}
			}
		}
		if child.Kind() == "argument_list" {
			hasEllipsis = checkEllipsisArgument(child)
		}
	}
	return identifier, hasEllipsis
}

func checkEllipsisArgument(argListNode *tree_sitter.Node) bool {
	for i := uint(0); i < argListNode.ChildCount(); i++ {
		child := argListNode.Child(i)
		if child.Kind() == "ellipsis" {
			return true
		}
	}
	return false
}

func applyDefaultCall(p funcParam, callName string) funcParam {
	lower := strings.ToLower(callName)

	switch lower {
	case "path":
		p.in = "path"
		p.required = true
	case "query":
		p.in = "query"
	case "header":
		p.in = "header"
	case "cookie":
		p.in = "cookie"
	case "body":
		p.in = "body"
	case "form":
		p.in = "form"
	case "file":
		p.typ = "file"
		p.in = "form"
	}

	return p
}

func isNameInPath(name string, path string) bool {
	target := "{" + name + "}"
	return strings.Contains(path, target)
}

func applyTypeAnnotation(p funcParam, typeText string) funcParam {
	lower := strings.ToLower(typeText)

	if strings.Contains(lower, "path") && !strings.Contains(lower, "pathlib") {
		p.in = "path"
		p.required = true
	} else if strings.Contains(lower, "query") {
		p.in = "query"
	} else if strings.Contains(lower, "header") {
		p.in = "header"
	} else if strings.Contains(lower, "cookie") {
		p.in = "cookie"
	} else if strings.Contains(lower, "body") || strings.Contains(lower, "form") {
		p.in = "body"
	} else if strings.Contains(lower, "uploadfile") || strings.Contains(lower, "file") {
		p.typ = "file"
		p.in = "form"
	}

	return p
}

func isSpecialParam(name string) bool {
	switch name {
	case "self", "cls", "request", "response", "db", "session", "background_tasks":
		return true
	}
	return false
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
