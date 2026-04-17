// Package fastify parses Fastify route registrations using tree-sitter-javascript.
// It walks JavaScript source files for fastify.get, fastify.post, etc. calls
// and extracts endpoint metadata including path, HTTP method, handler function name,
// and JSDoc comments.
//
// Supported patterns:
//   - Shorthand: fastify.get('/path', handler)
//   - Shorthand with options: fastify.get('/path', { schema: ... }, handler)
//   - Route object: fastify.route({ method: 'GET', url: '/path', handler: fn })
package fastify

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	tree_sitter "github.com/tree-sitter/go-tree-sitter"
	javascript "github.com/tree-sitter/tree-sitter-javascript/bindings/go"

	collector "github.com/tangcent/apilot/api-collector"
)

var httpMethods = map[string]bool{
	"get":    true,
	"post":   true,
	"put":    true,
	"delete": true,
	"patch":  true,
}

func Parse(sourceDir string) ([]collector.ApiEndpoint, error) {
	jsFiles, err := discoverJSFiles(sourceDir)
	if err != nil || len(jsFiles) == 0 {
		return nil, nil
	}

	ch := make(chan fileResult, len(jsFiles))
	var wg sync.WaitGroup

	for _, path := range jsFiles {
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

func discoverJSFiles(sourceDir string) ([]string, error) {
	var jsFiles []string
	err := filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() && (strings.HasSuffix(path, ".js") || strings.HasSuffix(path, ".mjs")) {
			jsFiles = append(jsFiles, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return jsFiles, nil
}

func processFile(filePath string) fileResult {
	source, err := os.ReadFile(filePath)
	if err != nil {
		return fileResult{err: fmt.Errorf("failed to read file: %w", err)}
	}

	p := tree_sitter.NewParser()
	defer p.Close()

	lang := tree_sitter.NewLanguage(javascript.Language())
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
	var lastComment string

	for i := uint(0); i < rootNode.ChildCount(); i++ {
		child := rootNode.Child(i)

		if child.Kind() == "comment" {
			commentText := child.Utf8Text(source)
			if isJSDocComment(commentText) {
				lastComment = cleanJSDocComment(commentText)
			} else {
				lastComment = ""
			}
			continue
		}

		if child.Kind() == "expression_statement" {
			ep := extractFromExpressionStatement(child, source, lastComment)
			if ep != nil {
				endpoints = append(endpoints, ep...)
			}
			lastComment = ""
		} else {
			lastComment = ""
		}
	}

	return endpoints
}

func extractFromExpressionStatement(node *tree_sitter.Node, source []byte, description string) []collector.ApiEndpoint {
	for i := uint(0); i < node.ChildCount(); i++ {
		child := node.Child(i)
		if child.Kind() == "call_expression" {
			return extractFromCallExpression(child, source, description)
		}
	}
	return nil
}

func extractFromCallExpression(callNode *tree_sitter.Node, source []byte, description string) []collector.ApiEndpoint {
	method, isRoute := extractMethodInfo(callNode, source)
	if isRoute {
		ep := extractShorthandRoute(callNode, source, method, description)
		if ep != nil {
			return []collector.ApiEndpoint{*ep}
		}
		return nil
	}

	if isRouteObjectCall(callNode, source) {
		return extractRouteObject(callNode, source, description)
	}

	return nil
}

func extractMethodInfo(callNode *tree_sitter.Node, source []byte) (method string, isRoute bool) {
	for i := uint(0); i < callNode.ChildCount(); i++ {
		child := callNode.Child(i)
		if child.Kind() == "member_expression" {
			return extractMemberExpressionMethod(child, source)
		}
	}
	return "", false
}

func extractMemberExpressionMethod(memberNode *tree_sitter.Node, source []byte) (method string, isRoute bool) {
	var prop string

	for i := uint(0); i < memberNode.ChildCount(); i++ {
		child := memberNode.Child(i)
		if child.Kind() == "property_identifier" {
			prop = child.Utf8Text(source)
		}
	}

	lowerProp := strings.ToLower(prop)
	if httpMethods[lowerProp] {
		return lowerProp, true
	}
	return "", false
}

func isRouteObjectCall(callNode *tree_sitter.Node, source []byte) bool {
	for i := uint(0); i < callNode.ChildCount(); i++ {
		child := callNode.Child(i)
		if child.Kind() == "member_expression" {
			return isRouteProperty(child, source)
		}
	}
	return false
}

func isRouteProperty(memberNode *tree_sitter.Node, source []byte) bool {
	for i := uint(0); i < memberNode.ChildCount(); i++ {
		child := memberNode.Child(i)
		if child.Kind() == "property_identifier" {
			prop := child.Utf8Text(source)
			if prop == "route" {
				return true
			}
		}
	}
	return false
}

func extractShorthandRoute(callNode *tree_sitter.Node, source []byte, method string, description string) *collector.ApiEndpoint {
	path, handlerName := extractShorthandArguments(callNode, source)
	if path == "" {
		return nil
	}

	standardPath := convertPath(path)

	ep := &collector.ApiEndpoint{
		Name:        handlerName,
		Path:        standardPath,
		Method:      strings.ToUpper(method),
		Protocol:    "http",
		Description: description,
	}

	pathParams := extractPathParams(standardPath)
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
		ep.Parameters = params
	}

	return ep
}

func extractShorthandArguments(callNode *tree_sitter.Node, source []byte) (path string, handlerName string) {
	for i := uint(0); i < callNode.ChildCount(); i++ {
		child := callNode.Child(i)
		if child.Kind() == "arguments" {
			return extractShorthandArgsFromNode(child, source)
		}
	}
	return "", ""
}

func extractShorthandArgsFromNode(argsNode *tree_sitter.Node, source []byte) (path string, handlerName string) {
	pathFound := false

	for i := uint(0); i < argsNode.ChildCount(); i++ {
		child := argsNode.Child(i)
		if child.Kind() == "(" || child.Kind() == ")" || child.Kind() == "," {
			continue
		}

		if !pathFound {
			if child.Kind() == "string" {
				path = unquoteJSString(child.Utf8Text(source))
				pathFound = true
			}
			continue
		}

		if handlerName == "" {
			handlerName = extractHandlerName(child, source)
		}
	}

	return path, handlerName
}

func extractRouteObject(callNode *tree_sitter.Node, source []byte, description string) []collector.ApiEndpoint {
	for i := uint(0); i < callNode.ChildCount(); i++ {
		child := callNode.Child(i)
		if child.Kind() == "arguments" {
			return extractRouteObjectArgs(child, source, description)
		}
	}
	return nil
}

func extractRouteObjectArgs(argsNode *tree_sitter.Node, source []byte, description string) []collector.ApiEndpoint {
	for i := uint(0); i < argsNode.ChildCount(); i++ {
		child := argsNode.Child(i)
		if child.Kind() == "object" {
			return extractRouteObjectFromObject(child, source, description)
		}
	}
	return nil
}

func extractRouteObjectFromObject(objNode *tree_sitter.Node, source []byte, description string) []collector.ApiEndpoint {
	var method, path, handlerName string

	for i := uint(0); i < objNode.ChildCount(); i++ {
		pair := objNode.Child(i)
		if pair.Kind() != "pair" {
			continue
		}

		var key string
		for j := uint(0); j < pair.ChildCount(); j++ {
			pairChild := pair.Child(j)
			if pairChild.Kind() == "property_identifier" {
				key = pairChild.Utf8Text(source)
			}
			if pairChild.Kind() == "string" && key == "" {
				key = unquoteJSString(pairChild.Utf8Text(source))
			}
		}

		switch key {
		case "method":
			method = extractPairValue(pair, source)
		case "url", "path":
			path = extractPairValue(pair, source)
		case "handler":
			handlerName = extractPairHandlerValue(pair, source)
		}
	}

	if method == "" || path == "" {
		return nil
	}

	standardPath := convertPath(path)
	method = strings.ToUpper(method)

	ep := collector.ApiEndpoint{
		Name:        handlerName,
		Path:        standardPath,
		Method:      method,
		Protocol:    "http",
		Description: description,
	}

	pathParams := extractPathParams(standardPath)
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
		ep.Parameters = params
	}

	return []collector.ApiEndpoint{ep}
}

func extractPairValue(pair *tree_sitter.Node, source []byte) string {
	for i := uint(0); i < pair.ChildCount(); i++ {
		child := pair.Child(i)
		if child.Kind() == "string" {
			return unquoteJSString(child.Utf8Text(source))
		}
		if child.Kind() == "identifier" {
			return child.Utf8Text(source)
		}
	}
	return ""
}

func extractPairHandlerValue(pair *tree_sitter.Node, source []byte) string {
	for i := uint(0); i < pair.ChildCount(); i++ {
		child := pair.Child(i)
		if child.Kind() == "identifier" {
			return child.Utf8Text(source)
		}
		if child.Kind() == "member_expression" {
			return child.Utf8Text(source)
		}
		if child.Kind() == "function_expression" {
			return extractFunctionExpressionName(child, source)
		}
		if child.Kind() == "arrow_function" {
			return "anonymous"
		}
	}
	return "anonymous"
}

func extractHandlerName(node *tree_sitter.Node, source []byte) string {
	switch node.Kind() {
	case "identifier":
		return node.Utf8Text(source)
	case "member_expression":
		return node.Utf8Text(source)
	case "function_expression":
		return extractFunctionExpressionName(node, source)
	case "arrow_function":
		return "anonymous"
	case "object":
		return ""
	default:
		return "anonymous"
	}
}

func extractFunctionExpressionName(funcNode *tree_sitter.Node, source []byte) string {
	for i := uint(0); i < funcNode.ChildCount(); i++ {
		child := funcNode.Child(i)
		if child.Kind() == "identifier" {
			return child.Utf8Text(source)
		}
	}
	return "anonymous"
}

func isJSDocComment(comment string) bool {
	return strings.HasPrefix(comment, "/**")
}

func cleanJSDocComment(comment string) string {
	comment = strings.TrimPrefix(comment, "/**")
	comment = strings.TrimSuffix(comment, "*/")

	var lines []string
	for _, line := range strings.Split(comment, "\n") {
		line = strings.TrimSpace(line)
		line = strings.TrimPrefix(line, "* ")
		line = strings.TrimPrefix(line, "*")
		line = strings.TrimSpace(line)
		if line != "" {
			lines = append(lines, line)
		}
	}

	return strings.Join(lines, " ")
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

func convertPath(fastifyPath string) string {
	var parts []string
	for _, segment := range strings.Split(fastifyPath, "/") {
		if strings.HasPrefix(segment, ":") {
			parts = append(parts, "{"+segment[1:]+"}")
		} else {
			parts = append(parts, segment)
		}
	}
	return strings.Join(parts, "/")
}

func unquoteJSString(s string) string {
	if len(s) >= 2 {
		if (s[0] == '"' && s[len(s)-1] == '"') || (s[0] == '\'' && s[len(s)-1] == '\'') {
			return s[1 : len(s)-1]
		}
	}
	if len(s) >= 2 && s[0] == '`' && s[len(s)-1] == '`' {
		return s[1 : len(s)-1]
	}
	return s
}
