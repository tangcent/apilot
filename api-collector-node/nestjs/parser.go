package nestjs

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	tree_sitter "github.com/tree-sitter/go-tree-sitter"
	typescript "github.com/tree-sitter/tree-sitter-typescript/bindings/go"

	"github.com/tangcent/apilot/api-collector-node/express"
	collector "github.com/tangcent/apilot/api-collector"
)

var httpMethodDecorators = map[string]bool{
	"Get":     true,
	"Post":    true,
	"Put":     true,
	"Delete":  true,
	"Patch":   true,
	"Head":    true,
	"Options": true,
}

var paramDecorators = map[string]string{
	"Param":     "path",
	"Query":     "query",
	"Body":      "body",
	"Headers":   "header",
	"Header":    "header",
	"Session":   "cookie",
	"HostParam": "path",
	"RawBody":   "body",
}

func Parse(sourceDir string) ([]collector.ApiEndpoint, error) {
	return ParseWithDependencyResolver(sourceDir, nil)
}

func ParseWithDependencyResolver(sourceDir string, depResolver collector.DependencyResolver) ([]collector.ApiEndpoint, error) {
	tsFiles, err := discoverTSFiles(sourceDir)
	if err != nil || len(tsFiles) == 0 {
		return nil, nil
	}

	typeRegistry := express.NewTSTypeRegistry()
	reg, err := express.ParseTSTypes(sourceDir)
	if err == nil && reg != nil {
		typeRegistry = reg
	}

	ctx := &parseContext{
		typeRegistry:       typeRegistry,
		dependencyResolver: depResolver,
	}

	ch := make(chan fileResult, len(tsFiles))
	var wg sync.WaitGroup

	for _, path := range tsFiles {
		wg.Add(1)
		go func(filePath string) {
			defer wg.Done()
			res := ctx.processFile(filePath)
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

type parseContext struct {
	typeRegistry       *express.TSTypeRegistry
	dependencyResolver collector.DependencyResolver
}

type fileResult struct {
	endpoints []collector.ApiEndpoint
	err       error
}

func discoverTSFiles(sourceDir string) ([]string, error) {
	var tsFiles []string
	err := filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() && (strings.HasSuffix(path, ".ts") || strings.HasSuffix(path, ".tsx")) {
			tsFiles = append(tsFiles, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return tsFiles, nil
}

func (ctx *parseContext) processFile(filePath string) fileResult {
	source, err := os.ReadFile(filePath)
	if err != nil {
		return fileResult{err: fmt.Errorf("failed to read file: %w", err)}
	}

	p := tree_sitter.NewParser()
	defer p.Close()

	lang := tree_sitter.NewLanguage(typescript.LanguageTypescript())
	if err := p.SetLanguage(lang); err != nil {
		return fileResult{err: fmt.Errorf("failed to set language: %w", err)}
	}

	tree := p.Parse(source, nil)
	if tree == nil {
		return fileResult{}
	}
	defer tree.Close()

	rootNode := tree.RootNode()
	endpoints := ctx.extractEndpoints(rootNode, source)

	return fileResult{endpoints: endpoints}
}

func (ctx *parseContext) extractEndpoints(rootNode *tree_sitter.Node, source []byte) []collector.ApiEndpoint {
	var endpoints []collector.ApiEndpoint

	for i := uint(0); i < rootNode.ChildCount(); i++ {
		child := rootNode.Child(i)

		if child.Kind() == "class_declaration" {
			eps := ctx.extractFromClassNode(child, nil, source)
			endpoints = append(endpoints, eps...)
		}

		if child.Kind() == "export_statement" {
			classNode := findChildByKind(child, "class_declaration")
			if classNode != nil {
				controllerDecorator := findControllerDecorator(child, source)
				eps := ctx.extractFromClassNode(classNode, controllerDecorator, source)
				endpoints = append(endpoints, eps...)
			}
		}
	}

	return endpoints
}

func findControllerDecorator(exportNode *tree_sitter.Node, source []byte) *decoratorInfo {
	for i := uint(0); i < exportNode.ChildCount(); i++ {
		child := exportNode.Child(i)
		if child.Kind() == "decorator" {
			di := parseDecorator(child, source)
			if di != nil && isControllerDecorator(di.name) {
				return di
			}
		}
	}
	return nil
}

func isControllerDecorator(name string) bool {
	return name == "Controller" || name == "RestController" || name == "Api" || name == "WebSocketGateway"
}

func (ctx *parseContext) extractFromClassNode(classNode *tree_sitter.Node, controllerDecorator *decoratorInfo, source []byte) []collector.ApiEndpoint {
	basePath := ""
	classComment := ""

	if controllerDecorator != nil {
		basePath = controllerDecorator.firstArg
	}

	for i := uint(0); i < classNode.ChildCount(); i++ {
		child := classNode.Child(i)
		if child.Kind() == "decorator" {
			di := parseDecorator(child, source)
			if di != nil && isControllerDecorator(di.name) {
				basePath = di.firstArg
			}
		}
	}

	classComment = extractPrevComment(classNode, source)

	classBody := findChildByKind(classNode, "class_body")
	if classBody == nil {
		return nil
	}

	return ctx.extractFromClassBody(classBody, basePath, classComment, source)
}

func (ctx *parseContext) extractFromClassBody(classBody *tree_sitter.Node, basePath string, classComment string, source []byte) []collector.ApiEndpoint {
	var endpoints []collector.ApiEndpoint

	var pendingDecorator *decoratorInfo
	var pendingComment string

	for i := uint(0); i < classBody.ChildCount(); i++ {
		child := classBody.Child(i)

		switch child.Kind() {
		case "comment":
			if isJSDocComment(child.Utf8Text(source)) {
				pendingComment = cleanJSDocComment(child.Utf8Text(source))
			} else {
				pendingComment = ""
			}

		case "decorator":
			di := parseDecorator(child, source)
			if di != nil && httpMethodDecorators[di.name] {
				pendingDecorator = di
			}

		case "method_definition":
			if pendingDecorator != nil {
				ep := ctx.buildEndpoint(pendingDecorator, child, basePath, pendingComment, classComment, source)
				if ep != nil {
					endpoints = append(endpoints, *ep)
				}
				pendingDecorator = nil
				pendingComment = ""
			}

		default:
			pendingComment = ""
		}
	}

	return endpoints
}

func (ctx *parseContext) buildEndpoint(methodDecorator *decoratorInfo, methodNode *tree_sitter.Node, basePath string, methodComment string, classComment string, source []byte) *collector.ApiEndpoint {
	handlerName := extractHandlerName(methodNode, source)
	paramBindings := extractParamBindings(methodNode, source)

	description := methodComment
	if description == "" {
		description = classComment
	}

	fullPath := joinPath(basePath, methodDecorator.firstArg)

	ep := &collector.ApiEndpoint{
		Name:        handlerName,
		Path:        fullPath,
		Method:      strings.ToUpper(methodDecorator.name),
		Protocol:    "http",
		Description: description,
	}

	if len(paramBindings) > 0 {
		var params []collector.ApiParameter
		for _, pb := range paramBindings {
			required := pb.in == "path" || pb.in == "body"
			params = append(params, collector.ApiParameter{
				Name:     pb.name,
				In:       pb.in,
				Required: required,
				Type:     "text",
			})
		}
		ep.Parameters = params
	}

	pathParams := extractPathParams(fullPath)
	if len(pathParams) > 0 {
		existingParams := make(map[string]bool)
		for _, p := range ep.Parameters {
			existingParams[p.Name] = true
		}
		for _, pp := range pathParams {
			if !existingParams[pp.name] {
				ep.Parameters = append(ep.Parameters, collector.ApiParameter{
					Name:     pp.name,
					In:       "path",
					Required: true,
					Type:     "text",
				})
			}
		}
	}

	if ctx.typeRegistry != nil {
		handlerInfo := AnalyzeNestJSHandler(methodNode, source)
		if handlerInfo != nil {
			reqBody, resBody := ResolveNestJSHandlerTypesWithDepResolver(handlerInfo, ctx.typeRegistry, ctx.dependencyResolver)
			if reqBody != nil && !reqBody.IsNull() {
				ep.RequestBody = &collector.ApiBody{
					MediaType: "application/json",
					Body:      reqBody,
				}
			}
			if resBody != nil && !resBody.IsNull() {
				ep.Response = &collector.ApiBody{
					MediaType: "application/json",
					Body:      resBody,
				}
			}
		}
	}

	return ep
}

func extractHandlerName(methodNode *tree_sitter.Node, source []byte) string {
	for i := uint(0); i < methodNode.ChildCount(); i++ {
		child := methodNode.Child(i)
		if child.Kind() == "property_identifier" {
			return child.Utf8Text(source)
		}
	}
	return ""
}

func extractParamBindings(methodNode *tree_sitter.Node, source []byte) []paramBinding {
	var bindings []paramBinding

	paramsNode := findChildByKind(methodNode, "formal_parameters")
	if paramsNode == nil {
		return nil
	}

	for i := uint(0); i < paramsNode.ChildCount(); i++ {
		child := paramsNode.Child(i)
		if child.Kind() == "required_parameter" || child.Kind() == "optional_parameter" {
			pb := extractParamBindingFromParameter(child, source)
			if pb != nil {
				bindings = append(bindings, *pb)
			}
		}
	}

	return bindings
}

func extractParamBindingFromParameter(paramNode *tree_sitter.Node, source []byte) *paramBinding {
	var decoratorName string
	var decoratorArg string

	for i := uint(0); i < paramNode.ChildCount(); i++ {
		child := paramNode.Child(i)
		if child.Kind() == "decorator" {
			di := parseDecorator(child, source)
			if di != nil {
				decoratorName = di.name
				decoratorArg = di.firstArg
			}
		}
	}

	if decoratorName == "" {
		return nil
	}

	paramIn, ok := paramDecorators[decoratorName]
	if !ok || paramIn == "" {
		return nil
	}

	paramName := decoratorArg
	if paramName == "" {
		paramName = extractParamIdentifier(paramNode, source)
	}

	return &paramBinding{
		name: paramName,
		in:   paramIn,
	}
}

func extractParamIdentifier(paramNode *tree_sitter.Node, source []byte) string {
	for i := uint(0); i < paramNode.ChildCount(); i++ {
		child := paramNode.Child(i)
		if child.Kind() == "identifier" {
			return child.Utf8Text(source)
		}
	}
	return ""
}

type decoratorInfo struct {
	name     string
	firstArg string
}

type paramBinding struct {
	name string
	in   string
}

func parseDecorator(decoratorNode *tree_sitter.Node, source []byte) *decoratorInfo {
	callNode := findChildByKind(decoratorNode, "call_expression")
	if callNode != nil {
		name := extractDecoratorName(callNode, source)
		if name == "" {
			return nil
		}
		firstArg := extractFirstStringArgument(callNode, source)
		return &decoratorInfo{name: name, firstArg: firstArg}
	}

	for i := uint(0); i < decoratorNode.ChildCount(); i++ {
		child := decoratorNode.Child(i)
		if child.Kind() == "identifier" {
			return &decoratorInfo{name: child.Utf8Text(source)}
		}
	}

	return nil
}

func extractDecoratorName(callNode *tree_sitter.Node, source []byte) string {
	for i := uint(0); i < callNode.ChildCount(); i++ {
		child := callNode.Child(i)
		if child.Kind() == "identifier" {
			return child.Utf8Text(source)
		}
		if child.Kind() == "member_expression" {
			return extractMemberProperty(child, source)
		}
	}
	return ""
}

func extractMemberProperty(memberNode *tree_sitter.Node, source []byte) string {
	for i := uint(0); i < memberNode.ChildCount(); i++ {
		child := memberNode.Child(i)
		if child.Kind() == "property_identifier" {
			return child.Utf8Text(source)
		}
	}
	return ""
}

func extractFirstStringArgument(callNode *tree_sitter.Node, source []byte) string {
	argsNode := findChildByKind(callNode, "arguments")
	if argsNode == nil {
		return ""
	}
	return extractFirstStringFromArguments(argsNode, source)
}

func extractFirstStringFromArguments(argsNode *tree_sitter.Node, source []byte) string {
	for i := uint(0); i < argsNode.ChildCount(); i++ {
		child := argsNode.Child(i)
		if child.Kind() == "string" {
			return extractStringValue(child, source)
		}
	}
	return ""
}

func extractStringValue(stringNode *tree_sitter.Node, source []byte) string {
	for i := uint(0); i < stringNode.ChildCount(); i++ {
		child := stringNode.Child(i)
		if child.Kind() == "string_fragment" {
			return child.Utf8Text(source)
		}
	}
	raw := stringNode.Utf8Text(source)
	return unquoteTSString(raw)
}

func findChildByKind(node *tree_sitter.Node, kind string) *tree_sitter.Node {
	for i := uint(0); i < node.ChildCount(); i++ {
		child := node.Child(i)
		if child.Kind() == kind {
			return child
		}
	}
	return nil
}

func extractPrevComment(node *tree_sitter.Node, source []byte) string {
	prev := node.PrevNamedSibling()
	if prev != nil && prev.Kind() == "comment" {
		commentText := prev.Utf8Text(source)
		if isJSDocComment(commentText) {
			return cleanJSDocComment(commentText)
		}
	}
	return ""
}

func isJSDocComment(comment string) bool {
	return strings.HasPrefix(comment, "/**") || strings.HasPrefix(comment, "/*")
}

func cleanJSDocComment(comment string) string {
	comment = strings.TrimPrefix(comment, "/**")
	comment = strings.TrimPrefix(comment, "/*")
	comment = strings.TrimSuffix(comment, "*/")

	var lines []string
	for _, line := range strings.Split(comment, "\n") {
		line = strings.TrimSpace(line)
		line = strings.TrimPrefix(line, "* ")
		line = strings.TrimPrefix(line, "*")
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "@") {
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
		if strings.HasPrefix(segment, ":") {
			name := segment[1:]
			params = append(params, pathParamInfo{name: name})
		}
		if strings.HasPrefix(segment, "{") && strings.HasSuffix(segment, "}") {
			name := segment[1 : len(segment)-1]
			params = append(params, pathParamInfo{name: name})
		}
	}
	return params
}

func joinPath(basePath, methodPath string) string {
	if basePath == "" {
		return normalizePath(methodPath)
	}
	if methodPath == "" {
		return normalizePath(basePath)
	}
	return normalizePath(basePath + "/" + methodPath)
}

func normalizePath(path string) string {
	path = strings.ReplaceAll(path, "//", "/")
	if !strings.HasPrefix(path, "/") && path != "" {
		path = "/" + path
	}
	return path
}

func unquoteTSString(s string) string {
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
