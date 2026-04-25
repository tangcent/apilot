// Package django parses Django REST Framework views and URL patterns.
package django

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	tree_sitter "github.com/tree-sitter/go-tree-sitter"
	python "github.com/tree-sitter/tree-sitter-python/bindings/go"

	collector "github.com/tangcent/apilot/api-collector"
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

var viewSetMethods = map[string]string{
	"list":     "GET",
	"create":   "POST",
	"retrieve": "GET",
	"update":   "PUT",
	"partial_update": "PATCH",
	"destroy":  "DELETE",
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

	allSerializers := make(map[string]SerializerModel)
	var allRawEndpoints []rawEndpointInfo

	for res := range ch {
		if res.err != nil {
			continue
		}
		for k, v := range res.serializers {
			allSerializers[k] = v
		}
		allRawEndpoints = append(allRawEndpoints, res.rawEndpoints...)
	}

	if len(allRawEndpoints) == 0 {
		return nil, nil
	}

	typeResolver := NewDRFTypeResolver(allSerializers)

	var allEndpoints []collector.ApiEndpoint
	for _, raw := range allRawEndpoints {
		ep := buildDjangoEndpoint(raw, typeResolver)
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
	name               string
	path               string
	method             string
	protocol           string
	description        string
	parameters         []collector.ApiParameter
	serializerClass    string
	action             string
	isViewSet          bool
}

type fileResult struct {
	rawEndpoints []rawEndpointInfo
	serializers  map[string]SerializerModel
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
	serializers := extractSerializers(rootNode, source)
	rawEndpoints := extractEndpoints(rootNode, source)

	return fileResult{
		serializers:  serializers,
		rawEndpoints: rawEndpoints,
	}
}

func extractEndpoints(rootNode *tree_sitter.Node, source []byte) []rawEndpointInfo {
	var endpoints []rawEndpointInfo

	urlPatterns := extractUrlPatterns(rootNode, source)
	classEndpoints := extractClassBasedViews(rootNode, source)
	funcEndpoints := extractFunctionBasedViews(rootNode, source)

	endpoints = append(endpoints, urlPatterns...)
	endpoints = append(endpoints, classEndpoints...)
	endpoints = append(endpoints, funcEndpoints...)

	return endpoints
}

func extractUrlPatterns(rootNode *tree_sitter.Node, source []byte) []rawEndpointInfo {
	var endpoints []rawEndpointInfo

	for i := uint(0); i < rootNode.ChildCount(); i++ {
		child := rootNode.Child(i)
		if child.Kind() != "expression_statement" {
			continue
		}

		for j := uint(0); j < child.ChildCount(); j++ {
			assignment := child.Child(j)
			if assignment.Kind() != "assignment" {
				continue
			}

			var leftIdent *tree_sitter.Node
			var rightList *tree_sitter.Node

			for k := uint(0); k < assignment.ChildCount(); k++ {
				assignChild := assignment.Child(k)
				if assignChild.Kind() == "identifier" {
					leftIdent = assignChild
				} else if assignChild.Kind() == "list" {
					rightList = assignChild
				}
			}

			if leftIdent == nil || rightList == nil {
				continue
			}

			varName := leftIdent.Utf8Text(source)
			if varName != "urlpatterns" {
				continue
			}

			patterns := parseUrlPatternList(rightList, source)
			endpoints = append(endpoints, patterns...)
		}
	}

	return endpoints
}

func parseUrlPatternList(listNode *tree_sitter.Node, source []byte) []rawEndpointInfo {
	var endpoints []rawEndpointInfo

	for i := uint(0); i < listNode.ChildCount(); i++ {
		child := listNode.Child(i)
		if child.Kind() == "call" {
			ep := parsePathCall(child, source)
			if ep != nil {
				endpoints = append(endpoints, *ep)
			}
		}
	}

	return endpoints
}

func parsePathCall(callNode *tree_sitter.Node, source []byte) *rawEndpointInfo {
	var funcName string
	var args []string

	for i := uint(0); i < callNode.ChildCount(); i++ {
		child := callNode.Child(i)
		if child.Kind() == "identifier" {
			funcName = child.Utf8Text(source)
		} else if child.Kind() == "attribute" {
			funcName = child.Utf8Text(source)
		} else if child.Kind() == "argument_list" {
			args = extractCallArguments(child, source)
		}
	}

	if funcName != "path" && funcName != "re_path" && funcName != "url" {
		return nil
	}

	if len(args) < 2 {
		return nil
	}

	pathPattern := args[0]
	viewName := args[1]

	if funcName == "re_path" || funcName == "url" {
		pathPattern = convertRegexToPath(pathPattern)
	}

	// Normalize path format
	pathPattern = normalizePath(pathPattern)

	method := "GET"
	if strings.Contains(strings.ToLower(viewName), "post") {
		method = "POST"
	} else if strings.Contains(strings.ToLower(viewName), "put") {
		method = "PUT"
	} else if strings.Contains(strings.ToLower(viewName), "delete") {
		method = "DELETE"
	} else if strings.Contains(strings.ToLower(viewName), "patch") {
		method = "PATCH"
	}

	ep := &rawEndpointInfo{
		name:     viewName,
		path:     pathPattern,
		method:   method,
		protocol: "http",
	}

	pathParams := extractPathParams(pathPattern)
	if len(pathParams) > 0 {
		var params []collector.ApiParameter
		for _, p := range pathParams {
			params = append(params, collector.ApiParameter{
				Name:     p.name,
				In:       "path",
				Required: true,
				Type:     "text",
			})
		}
		ep.parameters = params
	}

	return ep
}

func extractCallArguments(argListNode *tree_sitter.Node, source []byte) []string {
	var args []string

	for i := uint(0); i < argListNode.ChildCount(); i++ {
		child := argListNode.Child(i)
		if child.Kind() == "(" || child.Kind() == ")" || child.Kind() == "," {
			continue
		}

		if child.Kind() == "string" {
			args = append(args, unquotePythonString(child.Utf8Text(source)))
		} else if child.Kind() == "identifier" {
			args = append(args, child.Utf8Text(source))
		} else if child.Kind() == "attribute" {
			args = append(args, child.Utf8Text(source))
		} else if child.Kind() == "call" {
			args = append(args, child.Utf8Text(source))
		}
	}

	return args
}

func extractClassBasedViews(rootNode *tree_sitter.Node, source []byte) []rawEndpointInfo {
	var endpoints []rawEndpointInfo

	for i := uint(0); i < rootNode.ChildCount(); i++ {
		child := rootNode.Child(i)
		if child.Kind() != "class_definition" {
			continue
		}

		className := extractClassName(child, source)
		if className == "" {
			continue
		}

		baseClasses := extractBaseClasses(child, source)
		if !isAPIView(baseClasses) && !isViewSet(baseClasses) {
			continue
		}

		serializerClass := extractSerializerClassFromView(child, source)
		isVS := isViewSet(baseClasses)

		methods := extractClassMethods(child, source)
		for _, method := range methods {
			httpMethod := method.name
			action := ""
			if isVS {
				if mapped, ok := viewSetMethods[strings.ToLower(method.name)]; ok {
					httpMethod = mapped
					action = strings.ToLower(method.name)
				} else {
					continue
				}
			} else {
				if !httpMethods[strings.ToLower(method.name)] {
					continue
				}
				httpMethod = strings.ToUpper(method.name)
			}

			ep := rawEndpointInfo{
				name:            className + "." + method.name,
				path:            "/" + className,
				method:          httpMethod,
				protocol:        "http",
				description:     method.docstring,
				serializerClass: serializerClass,
				action:          action,
				isViewSet:       isVS,
			}
			endpoints = append(endpoints, ep)
		}
	}

	return endpoints
}

func extractClassName(classNode *tree_sitter.Node, source []byte) string {
	for i := uint(0); i < classNode.ChildCount(); i++ {
		child := classNode.Child(i)
		if child.Kind() == "identifier" {
			return child.Utf8Text(source)
		}
	}
	return ""
}

func extractBaseClasses(classNode *tree_sitter.Node, source []byte) []string {
	var baseClasses []string

	for i := uint(0); i < classNode.ChildCount(); i++ {
		child := classNode.Child(i)
		if child.Kind() == "argument_list" {
			for j := uint(0); j < child.ChildCount(); j++ {
				arg := child.Child(j)
				if arg.Kind() == "(" || arg.Kind() == ")" || arg.Kind() == "," {
					continue
				}
				if arg.Kind() == "identifier" || arg.Kind() == "attribute" {
					baseClasses = append(baseClasses, arg.Utf8Text(source))
				}
			}
		}
	}

	return baseClasses
}

func isAPIView(baseClasses []string) bool {
	for _, bc := range baseClasses {
		if bc == "APIView" || strings.HasSuffix(bc, ".APIView") {
			return true
		}
	}
	return false
}

func isViewSet(baseClasses []string) bool {
	for _, bc := range baseClasses {
		if strings.Contains(bc, "ViewSet") || strings.HasSuffix(bc, ".ViewSet") {
			return true
		}
	}
	return false
}

type classMethod struct {
	name     string
	docstring string
}

func extractClassMethods(classNode *tree_sitter.Node, source []byte) []classMethod {
	var methods []classMethod

	for i := uint(0); i < classNode.ChildCount(); i++ {
		child := classNode.Child(i)
		if child.Kind() != "block" {
			continue
		}

		for j := uint(0); j < child.ChildCount(); j++ {
			blockChild := child.Child(j)
			if blockChild.Kind() != "function_definition" {
				continue
			}

			methodName := extractFunctionName(blockChild, source)
			
			// Check if it's an HTTP method or a ViewSet method
			if !httpMethods[strings.ToLower(methodName)] && viewSetMethods[strings.ToLower(methodName)] == "" {
				continue
			}

			docstring := extractDocstring(blockChild, source)
			methods = append(methods, classMethod{
				name:     methodName,
				docstring: docstring,
			})
		}
	}

	return methods
}

func extractFunctionBasedViews(rootNode *tree_sitter.Node, source []byte) []rawEndpointInfo {
	var endpoints []rawEndpointInfo

	for i := uint(0); i < rootNode.ChildCount(); i++ {
		child := rootNode.Child(i)
		if child.Kind() != "decorated_definition" {
			continue
		}

		eps := extractApiViewDecorator(child, source)
		if len(eps) > 0 {
			endpoints = append(endpoints, eps...)
		}
	}

	return endpoints
}

func extractApiViewDecorator(node *tree_sitter.Node, source []byte) []rawEndpointInfo {
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

	methods := extractApiViewMethods(decorator, source)
	if len(methods) == 0 {
		return nil
	}

	funcName := extractFunctionName(funcDef, source)
	description := extractDocstring(funcDef, source)

	var endpoints []rawEndpointInfo
	for _, method := range methods {
		ep := rawEndpointInfo{
			name:        funcName,
			path:        "/" + funcName,
			method:      strings.ToUpper(method),
			protocol:    "http",
			description: description,
		}
		endpoints = append(endpoints, ep)
	}

	return endpoints
}

func extractApiViewMethods(decorator *tree_sitter.Node, source []byte) []string {
	var methods []string

	for i := uint(0); i < decorator.ChildCount(); i++ {
		child := decorator.Child(i)
		if child.Kind() == "@" {
			continue
		}

		if child.Kind() == "call" {
			funcName := extractCallFunctionName(child, source)
			if funcName == "api_view" {
				methods = extractApiViewCallArguments(child, source)
			}
		}
	}

	return methods
}

func extractCallFunctionName(callNode *tree_sitter.Node, source []byte) string {
	for i := uint(0); i < callNode.ChildCount(); i++ {
		child := callNode.Child(i)
		if child.Kind() == "identifier" {
			return child.Utf8Text(source)
		} else if child.Kind() == "attribute" {
			return child.Utf8Text(source)
		}
	}
	return ""
}

func extractApiViewCallArguments(callNode *tree_sitter.Node, source []byte) []string {
	var methods []string

	for i := uint(0); i < callNode.ChildCount(); i++ {
		child := callNode.Child(i)
		if child.Kind() != "argument_list" {
			continue
		}

		for j := uint(0); j < child.ChildCount(); j++ {
			arg := child.Child(j)
			if arg.Kind() == "(" || arg.Kind() == ")" || arg.Kind() == "," {
				continue
			}

			if arg.Kind() == "list" {
				methods = extractListElements(arg, source)
			}
		}
	}

	return methods
}

func extractListElements(listNode *tree_sitter.Node, source []byte) []string {
	var elements []string

	for i := uint(0); i < listNode.ChildCount(); i++ {
		child := listNode.Child(i)
		if child.Kind() == "[" || child.Kind() == "]" || child.Kind() == "," {
			continue
		}

		if child.Kind() == "string" {
			elements = append(elements, unquotePythonString(child.Utf8Text(source)))
		}
	}

	return elements
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

func buildDjangoEndpoint(raw rawEndpointInfo, typeResolver *DRFTypeResolver) *collector.ApiEndpoint {
	ep := &collector.ApiEndpoint{
		Name:        raw.name,
		Path:        raw.path,
		Method:      raw.method,
		Protocol:    raw.protocol,
		Description: raw.description,
	}

	if len(raw.parameters) > 0 {
		ep.Parameters = raw.parameters
	}

	if raw.serializerClass == "" {
		return ep
	}

	var requestSerializer string
	var responseSerializer string

	if raw.isViewSet && raw.action != "" {
		requestSerializer, responseSerializer = typeResolver.ResolveActionSerializer(raw.name, raw.action, raw.serializerClass)
	} else {
		requestSerializer, responseSerializer = typeResolver.ResolveHTTPMethodSerializer(raw.name, raw.method, raw.serializerClass)
	}

	if requestSerializer != "" {
		reqBody := typeResolver.BuildRequestBody(requestSerializer)
		if reqBody != nil {
			ep.RequestBody = &collector.ApiBody{
				MediaType: "application/json",
				Body:      reqBody,
			}
		}
	}

	if responseSerializer != "" {
		respBody := typeResolver.BuildResponseBody(responseSerializer)
		if respBody != nil {
			ep.Response = &collector.ApiBody{
				MediaType: "application/json",
				Body:      respBody,
			}
		}
	}

	return ep
}

type pathParamInfo struct {
	name string
}

func normalizePath(path string) string {
	// Ensure path starts with /
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	// Convert Django path parameters from <type:name> to {name}
	re := regexp.MustCompile(`<[^>]+>`)
	result := re.ReplaceAllStringFunc(path, func(match string) string {
		// Remove < and >
		content := match[1 : len(match)-1]
		// If it contains a colon, extract the name part
		if strings.Contains(content, ":") {
			parts := strings.SplitN(content, ":", 2)
			return "{" + parts[1] + "}"
		}
		return "{" + content + "}"
	})

	return result
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

func convertRegexToPath(regex string) string {
	regex = strings.TrimPrefix(regex, "^")
	regex = strings.TrimSuffix(regex, "$")

	re := regexp.MustCompile(`\(\?P<(\w+)>[^)]*\)`)
	result := re.ReplaceAllString(regex, "<$1>")

	re = regexp.MustCompile(`\(([^)]+)\)`)
	result = re.ReplaceAllString(result, "<param>")

	result = strings.ReplaceAll(result, `\d+`, "<id>")
	result = strings.ReplaceAll(result, `\w+`, "<name>")
	result = strings.ReplaceAll(result, `\.`, ".")
	result = strings.ReplaceAll(result, `\/`, "/")

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
