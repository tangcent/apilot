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
	model "github.com/tangcent/apilot/api-model"

	"github.com/tangcent/apilot/api-collector-python/fastapi"
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

	allPydanticModels := make(map[string]fastapi.PydanticModel)
	allMarshmallowSchemas := make(map[string]MarshmallowModel)
	var allRawEndpoints []rawEndpointInfo

	for res := range ch {
		if res.err != nil {
			continue
		}
		for k, v := range res.pydanticModels {
			allPydanticModels[k] = v
		}
		for k, v := range res.marshmallowSchemas {
			allMarshmallowSchemas[k] = v
		}
		allRawEndpoints = append(allRawEndpoints, res.rawEndpoints...)
	}

	if len(allRawEndpoints) == 0 {
		return nil, nil
	}

	typeResolver := NewFlaskTypeResolver(allPydanticModels, allMarshmallowSchemas)

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
	method          string
	path            string
	funcName        string
	description     string
	params          []funcParam
	requestBodyType string
	responseType    string
	expectModel     string
	marshalModel    string
	requestPattern  string
}

type fileResult struct {
	rawEndpoints       []rawEndpointInfo
	pydanticModels     map[string]fastapi.PydanticModel
	marshmallowSchemas map[string]MarshmallowModel
	err                error
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
	pydanticModels := fastapi.ExtractPydanticModels(rootNode, source)
	marshmallowSchemas := extractMarshmallowSchemas(rootNode, source)
	rawEndpoints := extractRawEndpoints(rootNode, source)

	return fileResult{
		pydanticModels:     pydanticModels,
		marshmallowSchemas: marshmallowSchemas,
		rawEndpoints:       rawEndpoints,
	}
}

func extractRawEndpoints(rootNode *tree_sitter.Node, source []byte) []rawEndpointInfo {
	var endpoints []rawEndpointInfo

	for i := uint(0); i < rootNode.ChildCount(); i++ {
		child := rootNode.Child(i)
		switch child.Kind() {
		case "decorated_definition":
			eps := extractDecoratedDefinition(child, source)
			endpoints = append(endpoints, eps...)
		case "class_definition":
			eps := extractClassDefinition(child, source)
			endpoints = append(endpoints, eps...)
		}
	}

	return endpoints
}

func extractDecoratedDefinition(node *tree_sitter.Node, source []byte) []rawEndpointInfo {
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

	routeInfo := extractDecoratorRouteInfo(decorator, source)
	if routeInfo.path == "" {
		return nil
	}

	funcName := extractFunctionName(funcDef, source)
	description := extractDocstring(funcDef, source)
	params := extractFunctionParameters(funcDef, source, routeInfo.path)
	returnType := extractReturnType(funcDef, source)
	requestPattern := detectRequestPattern(funcDef, source)
	expectModel, marshalModel := extractRESTXDecoratorInfo(decorator, source)

	var requestBodyType string
	requestBodyType = detectRequestBodyType(funcDef, source, params)

	var endpoints []rawEndpointInfo

	for _, method := range routeInfo.methods {
		path := convertFlaskPathToStandard(routeInfo.path)
		raw := rawEndpointInfo{
			method:          strings.ToUpper(method),
			path:            path,
			funcName:        funcName,
			description:     description,
			params:          params,
			requestBodyType: requestBodyType,
			responseType:    returnType,
			expectModel:     expectModel,
			marshalModel:    marshalModel,
			requestPattern:  requestPattern,
		}
		endpoints = append(endpoints, raw)
	}

	return endpoints
}

func extractClassDefinition(node *tree_sitter.Node, source []byte) []rawEndpointInfo {
	var className string
	var body *tree_sitter.Node
	var argList *tree_sitter.Node

	for i := uint(0); i < node.ChildCount(); i++ {
		child := node.Child(i)
		switch child.Kind() {
		case "identifier":
			className = child.Utf8Text(source)
		case "argument_list":
			argList = child
		case "block":
			body = child
		}
	}

	if className == "" || body == nil {
		return nil
	}

	if !isRESTXResourceClass(argList, source) {
		return nil
	}

	return extractRESTXResourceEndpoints(className, body, source)
}

func isRESTXResourceClass(argList *tree_sitter.Node, source []byte) bool {
	if argList == nil {
		return false
	}

	for i := uint(0); i < argList.ChildCount(); i++ {
		child := argList.Child(i)
		if child.Kind() == "(" || child.Kind() == ")" || child.Kind() == "," {
			continue
		}
		text := strings.TrimSpace(child.Utf8Text(source))
		if text == "Resource" || strings.HasSuffix(text, ".Resource") {
			return true
		}
	}
	return false
}

func extractRESTXResourceEndpoints(className string, body *tree_sitter.Node, source []byte) []rawEndpointInfo {
	var endpoints []rawEndpointInfo

	for i := uint(0); i < body.ChildCount(); i++ {
		child := body.Child(i)
		if child.Kind() != "decorated_definition" {
			continue
		}

		var decorator *tree_sitter.Node
		var funcDef *tree_sitter.Node

		for j := uint(0); j < child.ChildCount(); j++ {
			subChild := child.Child(j)
			switch subChild.Kind() {
			case "decorator":
				decorator = subChild
			case "function_definition":
				funcDef = subChild
			}
		}

		if funcDef == nil {
			continue
		}

		methodName := extractFunctionName(funcDef, source)
		method := restxMethodToHTTP(methodName)
		if method == "" {
			continue
		}

		description := extractDocstring(funcDef, source)
		params := extractFunctionParameters(funcDef, source, "")
		returnType := extractReturnType(funcDef, source)
		requestPattern := detectRequestPattern(funcDef, source)
		requestBodyType := detectRequestBodyType(funcDef, source, params)

		var expectModel, marshalModel string
		if decorator != nil {
			expectModel, marshalModel = extractRESTXDecoratorInfo(decorator, source)
		}

		raw := rawEndpointInfo{
			method:          method,
			path:            "/" + toSnakeCase(className),
			funcName:        methodName,
			description:     description,
			params:          params,
			requestBodyType: requestBodyType,
			responseType:    returnType,
			expectModel:     expectModel,
			marshalModel:    marshalModel,
			requestPattern:  requestPattern,
		}
		endpoints = append(endpoints, raw)
	}

	return endpoints
}

func restxMethodToHTTP(methodName string) string {
	switch strings.ToLower(methodName) {
	case "get":
		return "GET"
	case "post":
		return "POST"
	case "put":
		return "PUT"
	case "delete":
		return "DELETE"
	case "patch":
		return "PATCH"
	default:
		return ""
	}
}

func toSnakeCase(s string) string {
	var result []byte
	for i, c := range s {
		if c >= 'A' && c <= 'Z' {
			if i > 0 {
				result = append(result, '_')
			}
			result = append(result, byte(c+'a'-'A'))
		} else {
			result = append(result, byte(c))
		}
	}
	return string(result)
}

func buildEndpoint(raw rawEndpointInfo, typeResolver *FlaskTypeResolver) *collector.ApiEndpoint {
	ep := &collector.ApiEndpoint{
		Name:        raw.funcName,
		Path:        raw.path,
		Method:      raw.method,
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

	requestBodyType := raw.expectModel
	if requestBodyType == "" {
		requestBodyType = raw.requestBodyType
	}
	if requestBodyType == "" && len(bodyParams) == 1 && bodyParams[0].typeAnnotation != "" {
		requestBodyType = bodyParams[0].typeAnnotation
	}

	if requestBodyType != "" {
		resolvedBody := typeResolver.Resolve(requestBodyType)
		if resolvedBody != nil && !resolvedBody.IsNull() {
			mediaType := "application/json"
			if raw.requestPattern == "form" {
				mediaType = "application/x-www-form-urlencoded"
			}
			ep.RequestBody = &collector.ApiBody{
				MediaType: mediaType,
				Body:      resolvedBody,
			}
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

	responseType := raw.marshalModel
	if responseType == "" {
		responseType = raw.responseType
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

type routeInfo struct {
	path    string
	methods []string
}

func extractDecoratorRouteInfo(decorator *tree_sitter.Node, source []byte) routeInfo {
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
			p, m := extractArguments(child, source)
			if p != "" {
				path = p
			}
			if len(m) > 0 {
				methods = m
			}
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

func extractRESTXDecoratorInfo(decorator *tree_sitter.Node, source []byte) (expectModel string, marshalModel string) {
	for i := uint(0); i < decorator.ChildCount(); i++ {
		child := decorator.Child(i)
		if child.Kind() == "@" {
			continue
		}
		if child.Kind() == "call" {
			e, m := extractRESTXCallInfo(child, source)
			if e != "" {
				expectModel = e
			}
			if m != "" {
				marshalModel = m
			}
		}
		for j := uint(0); j < child.ChildCount(); j++ {
			subChild := child.Child(j)
			if subChild.Kind() == "call" {
				e, m := extractRESTXCallInfo(subChild, source)
				if e != "" {
					expectModel = e
				}
				if m != "" {
					marshalModel = m
				}
			}
		}
	}
	return expectModel, marshalModel
}

func extractRESTXCallInfo(callNode *tree_sitter.Node, source []byte) (expectModel string, marshalModel string) {
	var funcName string

	for i := uint(0); i < callNode.ChildCount(); i++ {
		child := callNode.Child(i)
		if child.Kind() == "identifier" {
			funcName = child.Utf8Text(source)
		}
		if child.Kind() == "attribute" {
			for j := uint(0); j < child.ChildCount(); j++ {
				attrChild := child.Child(j)
				if attrChild.Kind() == "identifier" {
					funcName = attrChild.Utf8Text(source)
				}
			}
		}
		if child.Kind() == "argument_list" {
			switch funcName {
			case "expect":
				expectModel = extractFirstIdentifierArg(child, source)
			case "marshal_with":
				marshalModel = extractFirstIdentifierArg(child, source)
			}
		}
	}

	return expectModel, marshalModel
}

func extractFirstIdentifierArg(argList *tree_sitter.Node, source []byte) string {
	for i := uint(0); i < argList.ChildCount(); i++ {
		child := argList.Child(i)
		if child.Kind() == "(" || child.Kind() == ")" || child.Kind() == "," {
			continue
		}
		if child.Kind() == "identifier" {
			return child.Utf8Text(source)
		}
		if child.Kind() == "attribute" {
			return child.Utf8Text(source)
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

		if child.Kind() == "identifier" {
			name := child.Utf8Text(source)
			if isSpecialFlaskParam(name) {
				continue
			}
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
			if param.name != "" && !isSpecialFlaskParam(param.name) {
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
		case "type":
			typeText := child.Utf8Text(source)
			p.typeAnnotation = typeText
			p.typ = simplifyType(typeText)
			if isComplexType(typeText) {
				p.in = "body"
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

func isSpecialFlaskParam(name string) bool {
	switch name {
	case "self", "cls", "request", "response", "db", "session":
		return true
	}
	return false
}

func isComplexType(typeText string) bool {
	if fastapi.IsPydanticModelType(typeText) {
		return true
	}
	if strings.HasSuffix(typeText, "Schema") {
		return true
	}
	base, args := fastapi.ParsePythonGenericType(typeText)
	if len(args) > 0 {
		for _, arg := range args {
			if fastapi.IsPydanticModelType(arg) || strings.HasSuffix(arg, "Schema") {
				return true
			}
		}
	}
	if base == "list" || base == "List" || base == "dict" || base == "Dict" {
		return true
	}
	return false
}

func simplifyType(typeText string) string {
	switch strings.ToLower(typeText) {
	case "str", "string":
		return "text"
	case "int", "integer":
		return "int"
	case "float":
		return "float"
	case "bool", "boolean":
		return "boolean"
	default:
		return typeText
	}
}

func extractReturnType(funcDef *tree_sitter.Node, source []byte) string {
	for i := uint(0); i < funcDef.ChildCount(); i++ {
		child := funcDef.Child(i)
		if child.Kind() == "type" {
			return child.Utf8Text(source)
		}
	}
	return ""
}

func detectRequestPattern(funcDef *tree_sitter.Node, source []byte) string {
	block := findBlock(funcDef)
	if block == nil {
		return ""
	}

	pattern := scanBlockForRequestPattern(block, source)
	return pattern
}

func findBlock(funcDef *tree_sitter.Node) *tree_sitter.Node {
	for i := uint(0); i < funcDef.ChildCount(); i++ {
		child := funcDef.Child(i)
		if child.Kind() == "block" {
			return child
		}
	}
	return nil
}

func scanBlockForRequestPattern(block *tree_sitter.Node, source []byte) string {
	var found string

	for i := uint(0); i < block.ChildCount(); i++ {
		child := block.Child(i)
		text := child.Utf8Text(source)

		if strings.Contains(text, "request.json") || strings.Contains(text, "request.get_json()") {
			if found == "" || found == "json" {
				found = "json"
			}
		}
		if strings.Contains(text, "request.form") {
			if found == "" || found == "form" {
				found = "form"
			}
		}
		if strings.Contains(text, "request.args") {
			if found == "" {
				found = "query"
			}
		}
		if strings.Contains(text, "request.files") {
			if found == "" || found == "files" {
				found = "files"
			}
		}
	}

	return found
}

func detectRequestBodyType(funcDef *tree_sitter.Node, source []byte, params []funcParam) string {
	for _, p := range params {
		if p.in == "body" && p.typeAnnotation != "" {
			return p.typeAnnotation
		}
	}

	block := findBlock(funcDef)
	if block == nil {
		return ""
	}

	return inferRequestBodyFromPattern(block, source)
}

func inferRequestBodyFromPattern(block *tree_sitter.Node, source []byte) string {
	for i := uint(0); i < block.ChildCount(); i++ {
		child := block.Child(i)
		text := child.Utf8Text(source)

		if strings.Contains(text, "request.json") || strings.Contains(text, "request.get_json()") {
			return inferTypeFromAssignment(child, source)
		}
	}
	return ""
}

func inferTypeFromAssignment(stmt *tree_sitter.Node, source []byte) string {
	for i := uint(0); i < stmt.ChildCount(); i++ {
		child := stmt.Child(i)
		if child.Kind() == "assignment" {
			for j := uint(0); j < child.ChildCount(); j++ {
				assignChild := child.Child(j)
				if assignChild.Kind() == "type" {
					return assignChild.Utf8Text(source)
				}
			}
		}
		if child.Kind() == "annotated_assignment" {
			for j := uint(0); j < child.ChildCount(); j++ {
				assignChild := child.Child(j)
				if assignChild.Kind() == "type" {
					return assignChild.Utf8Text(source)
				}
			}
		}
	}
	return ""
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
