// Package echo parses Echo route registrations using Go's standard go/ast package.
// It walks Go source files for echo.Echo and echo.Group method calls
// (GET, POST, PUT, DELETE, PATCH) and extracts endpoint metadata
// including path, HTTP method, handler name, doc comments, parameters,
// request body, and response body.
package echo

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"sync"

	collector "github.com/tangcent/apilot/api-collector"
	model "github.com/tangcent/apilot/api-model"
)

// echoMethods maps Echo route-registration method names that we recognize.
var echoMethods = map[string]bool{
	"GET":    true,
	"POST":   true,
	"PUT":    true,
	"DELETE": true,
	"PATCH":  true,
}

// Parse extracts endpoints from Echo route registrations in the given source directory.
//
// It performs the following extractions per file:
//  1. Function doc comments — builds a map from function name to its
//     Go doc comment text, used later to populate ApiEndpoint.Description.
//  2. Handler body analysis — inspects each function's body for echo.Context
//     method calls to discover query params, form params, path params,
//     request body bindings, and response writes.
//  3. Group prefixes — finds assignments like
//     `v1 := e.Group("/v1")` and records the variable-to-prefix mapping,
//     so that subsequent route calls on that variable (e.g. `v1.GET("/users", ...)`)
//     resolve to the full path `/v1/users`.
//  4. Route registrations — finds method calls matching the Echo HTTP
//     method names (GET, POST, PUT, DELETE, PATCH) and extracts
//     the path (first argument as a string literal), HTTP method (from the call
//     selector name), handler function name, and the handler's doc comment.
//
// After merging per-file results, path parameters are extracted from the route
// path string (e.g. `/users/:id` → path param "id"), and the handler body
// analysis is applied to populate Parameters, RequestBody, and Response.
//
// Memory optimization:
//   - Files are discovered first via filepath.Walk, then parsed and processed
//     one at a time in goroutines. Each goroutine processes a single file and
//     sends its partial results through channels, so we never hold all ASTs in
//     memory simultaneously beyond what is needed for the current file.
//   - Maps are merged incrementally as results arrive.
//
// Per the collector contract:
//   - Returns nil, nil (not an error) when no endpoints are found.
//   - Skips unparseable files silently.
func Parse(sourceDir string) ([]collector.ApiEndpoint, error) {
	goFiles, err := discoverGoFiles(sourceDir)
	if err != nil || len(goFiles) == 0 {
		return nil, nil
	}

	ch := make(chan fileResult, len(goFiles))
	var wg sync.WaitGroup

	for _, path := range goFiles {
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

	funcDocs := make(map[string]string)
	groupPrefixes := make(map[string]string)
	handlerAnalyses := make(map[string]handlerAnalysis)
	var allRaw []rawEndpoint

	for res := range ch {
		for k, v := range res.funcDocs {
			funcDocs[k] = v
		}
		for k, v := range res.groupPrefixes {
			groupPrefixes[k] = v
		}
		for k, v := range res.handlerAnalyses {
			handlerAnalyses[k] = v
		}
		allRaw = append(allRaw, res.rawEndpoints...)
	}

	if len(allRaw) == 0 {
		return nil, nil
	}

	endpoints := make([]collector.ApiEndpoint, 0, len(allRaw))
	for _, raw := range allRaw {
		path := raw.path
		prefix := groupPrefixes[raw.receiverVar]
		if prefix != "" {
			path = prefix + path
		}

		handlerKey := raw.handlerName
		if idx := strings.LastIndex(handlerKey, "."); idx >= 0 {
			handlerKey = handlerKey[idx+1:]
		}

		ep := collector.ApiEndpoint{
			Name:        raw.handlerName,
			Path:        path,
			Method:      raw.method,
			Protocol:    "http",
			Description: funcDocs[handlerKey],
		}

		pathParams := extractPathParams(path)

		paramSet := make(map[string]bool)
		var params []collector.ApiParameter
		for _, p := range pathParams {
			params = append(params, collector.ApiParameter{
				Name:     p.name,
				In:       p.in,
				Required: p.required,
				Type:     p.typ,
			})
			paramSet[p.name+"|"+p.in] = true
		}

		analysis := handlerAnalyses[handlerKey]
		for _, p := range analysis.params {
			key := p.name + "|" + p.in
			if !paramSet[key] {
				params = append(params, collector.ApiParameter{
					Name:     p.name,
					In:       p.in,
					Required: p.required,
					Type:     p.typ,
					Default:  p.def,
				})
				paramSet[key] = true
			}
		}

		if len(params) > 0 {
			ep.Parameters = params
		}

		if analysis.requestBody != nil {
			ep.RequestBody = &collector.ApiBody{
				MediaType: analysis.requestBody.mediaType,
			}
			if analysis.requestBody.typeName != "" {
				ep.RequestBody.Body = model.SingleModel(analysis.requestBody.typeName)
			}
		}

		if analysis.response != nil {
			ep.Response = &collector.ApiBody{
				MediaType: analysis.response.mediaType,
			}
			if analysis.response.typeName != "" {
				ep.Response.Body = model.SingleModel(analysis.response.typeName)
			}
		}

		endpoints = append(endpoints, ep)
	}

	return endpoints, nil
}

// rawParam holds parameter info extracted from handler body or path string.
type rawParam struct {
	name     string
	in       string // "path", "query", "form", "header"
	required bool
	def      string
	typ      string // "text" or "file"
}

// rawBody holds request/response body info extracted from handler body.
type rawBody struct {
	mediaType string
	typeName  string
}

// rawEndpoint holds the raw data extracted from a single route-registration call
// before group-prefix resolution and doc-comment lookup.
type rawEndpoint struct {
	method      string
	path        string
	handlerName string
	receiverVar string
}

// handlerAnalysis holds the extracted info from analyzing a handler function body.
type handlerAnalysis struct {
	params      []rawParam
	requestBody *rawBody
	response    *rawBody
}

// fileResult holds the per-file extraction output produced by processFile.
type fileResult struct {
	funcDocs        map[string]string
	groupPrefixes   map[string]string
	rawEndpoints    []rawEndpoint
	handlerAnalyses map[string]handlerAnalysis
}

// discoverGoFiles walks sourceDir and returns paths to all non-test .go files.
// Returns nil on any filesystem error (per collector contract: skip gracefully).
func discoverGoFiles(sourceDir string) ([]string, error) {
	var goFiles []string
	err := filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() && strings.HasSuffix(path, ".go") && !strings.HasSuffix(path, "_test.go") {
			goFiles = append(goFiles, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return goFiles, nil
}

// processFile parses a single Go source file and extracts:
//   - funcDocs: map of function names to their doc comment text
//   - handlerAnalyses: map of function names to their handler body analysis
//   - groupPrefixes: map of variable names to their Echo Group prefix
//   - rawEndpoints: route registrations found in this file
func processFile(filePath string) fileResult {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		return fileResult{}
	}

	if !importsPackage(f, "github.com/labstack/echo/v4") {
		return fileResult{}
	}

	echoHandlers := make(map[string]bool)
	funcDocs := make(map[string]string)
	handlerAnalyses := make(map[string]handlerAnalysis)
	for _, decl := range f.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}
		name := fn.Name.Name
		if fn.Doc != nil {
			funcDocs[name] = strings.TrimSpace(fn.Doc.Text())
		}
		if findContextParamName(fn) != "" {
			echoHandlers[name] = true
		}
		params, reqBody, respBody := analyzeHandlerBody(fn)
		if len(params) > 0 || reqBody != nil || respBody != nil {
			handlerAnalyses[name] = handlerAnalysis{
				params:      params,
				requestBody: reqBody,
				response:    respBody,
			}
		}
	}

	groupPrefixes := make(map[string]string)
	ast.Inspect(f, func(n ast.Node) bool {
		assign, ok := n.(*ast.AssignStmt)
		if !ok {
			return true
		}
		for i, lhs := range assign.Lhs {
			ident, ok := lhs.(*ast.Ident)
			if !ok {
				continue
			}
			if i >= len(assign.Rhs) {
				continue
			}
			prefix := extractGroupPrefix(assign.Rhs[i])
			if prefix != "" {
				groupPrefixes[ident.Name] = prefix
			}
		}
		return true
	})

	var rawEndpoints []rawEndpoint
	ast.Inspect(f, func(n ast.Node) bool {
		callExpr, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}

		selExpr, ok := callExpr.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}

		methodName := selExpr.Sel.Name
		if !echoMethods[methodName] {
			return true
		}

		if len(callExpr.Args) < 2 {
			return true
		}

		path := extractStringLiteral(callExpr.Args[0])
		handlerName := extractHandlerName(callExpr.Args[1])
		receiverVar := resolveReceiverVar(selExpr.X)

		if !echoHandlers[handlerName] {
			return true
		}

		rawEndpoints = append(rawEndpoints, rawEndpoint{
			method:      methodName,
			path:        path,
			handlerName: handlerName,
			receiverVar: receiverVar,
		})
		return true
	})

	return fileResult{
		funcDocs:        funcDocs,
		groupPrefixes:   groupPrefixes,
		rawEndpoints:    rawEndpoints,
		handlerAnalyses: handlerAnalyses,
	}
}

// analyzeHandlerBody inspects the handler function body to extract query params,
// form params, path params, request body bindings, and response writes
// by walking echo.Context method calls.
//
// Recognized echo.Context methods:
//   - Param                           → path parameter (from c.Param("name"))
//   - QueryParam                      → query parameter (required)
//   - QueryParams                     → query parameter (for multi-value)
//   - FormValue                       → form parameter
//   - FormFile                        → file parameter (type="file")
//   - Request().Header.Get / Header   → header parameter
//   - Bind                            → request body (auto-detect media type)
//   - JSON                            → JSON response
//   - XML                             → XML response
//   - String                          → text/plain response
//   - Blob                            → binary response (application/octet-stream)
func analyzeHandlerBody(fn *ast.FuncDecl) ([]rawParam, *rawBody, *rawBody) {
	if fn.Body == nil {
		return nil, nil, nil
	}

	ctxVar := findContextParamName(fn)
	if ctxVar == "" {
		return nil, nil, nil
	}

	var params []rawParam
	var requestBody *rawBody
	var response *rawBody

	ast.Inspect(fn.Body, func(n ast.Node) bool {
		callExpr, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}

		selExpr, ok := callExpr.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}

		ident, ok := selExpr.X.(*ast.Ident)
		if !ok || ident.Name != ctxVar {
			return true
		}

		methodName := selExpr.Sel.Name

		switch methodName {
		case "Param":
			if name := extractStringLiteral(callExpr.Args[0]); name != "" {
				params = append(params, rawParam{name: name, in: "path", required: true, typ: "text"})
			}
		case "QueryParam", "QueryParams":
			if name := extractStringLiteral(callExpr.Args[0]); name != "" {
				params = append(params, rawParam{name: name, in: "query", required: true, typ: "text"})
			}
		case "FormValue":
			if name := extractStringLiteral(callExpr.Args[0]); name != "" {
				params = append(params, rawParam{name: name, in: "form", required: true, typ: "text"})
			}
		case "FormFile":
			if name := extractStringLiteral(callExpr.Args[0]); name != "" {
				params = append(params, rawParam{name: name, in: "form", required: true, typ: "file"})
			}
		case "Bind":
			typeName := ""
			if len(callExpr.Args) >= 1 {
				typeName = extractTypeName(callExpr.Args[0])
			}
			requestBody = &rawBody{mediaType: "application/json", typeName: typeName}
		case "JSON":
			typeName := ""
			if len(callExpr.Args) >= 2 {
				typeName = extractTypeName(callExpr.Args[1])
			}
			response = &rawBody{mediaType: "application/json", typeName: typeName}
		case "XML":
			typeName := ""
			if len(callExpr.Args) >= 2 {
				typeName = extractTypeName(callExpr.Args[1])
			}
			response = &rawBody{mediaType: "application/xml", typeName: typeName}
		case "String":
			response = &rawBody{mediaType: "text/plain"}
		case "Blob":
			response = &rawBody{mediaType: "application/octet-stream"}
		}

		return true
	})

	return params, requestBody, response
}

// findContextParamName returns the variable name of the echo.Context parameter
// in the given function declaration. Returns "" if no echo.Context parameter is found.
func findContextParamName(fn *ast.FuncDecl) string {
	if fn.Type.Params == nil || len(fn.Type.Params.List) == 0 {
		return ""
	}

	for _, param := range fn.Type.Params.List {
		if isEchoContext(param.Type) && len(param.Names) > 0 {
			return param.Names[0].Name
		}
	}

	return ""
}

// isEchoContext checks if the expression represents echo.Context.
func isEchoContext(expr ast.Expr) bool {
	selExpr, ok := expr.(*ast.SelectorExpr)
	if !ok {
		return false
	}

	if selExpr.Sel.Name != "Context" {
		return false
	}

	ident, ok := selExpr.X.(*ast.Ident)
	return ok && ident.Name == "echo"
}

// extractTypeName returns a human-readable type name from an expression.
// Handles:
//   - &obj  → "obj"
//   - obj   → "obj"
//   - Type{} → "Type"
//   - pkg.Type{} → "pkg.Type"
func extractTypeName(expr ast.Expr) string {
	if unary, ok := expr.(*ast.UnaryExpr); ok && unary.Op == token.AND {
		expr = unary.X
	}

	switch e := expr.(type) {
	case *ast.Ident:
		return e.Name
	case *ast.SelectorExpr:
		if x, ok := e.X.(*ast.Ident); ok {
			return x.Name + "." + e.Sel.Name
		}
		return e.Sel.Name
	case *ast.CompositeLit:
		return extractTypeName(e.Type)
	case *ast.MapType:
		return "map"
	}

	return ""
}

// extractPathParams parses path parameters from an Echo route path.
// Echo uses `:paramName` syntax for path parameters (e.g. `/users/:id` → param "id").
func extractPathParams(path string) []rawParam {
	var params []rawParam
	for _, segment := range strings.Split(path, "/") {
		if strings.HasPrefix(segment, ":") {
			name := segment[1:]
			params = append(params, rawParam{
				name:     name,
				in:       "path",
				required: true,
				typ:      "text",
			})
		}
	}
	return params
}

// extractStringLiteral returns the unquoted string value of a BasicLit node,
// supporting both double-quoted and backtick-quoted strings.
func extractStringLiteral(expr ast.Expr) string {
	lit, ok := expr.(*ast.BasicLit)
	if !ok || lit.Kind != token.STRING {
		return ""
	}
	s := lit.Value
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		return s[1 : len(s)-1]
	}
	if len(s) >= 2 && s[0] == '`' && s[len(s)-1] == '`' {
		return s[1 : len(s)-1]
	}
	return s
}

// extractHandlerName returns the handler function name from the second argument
// of an Echo route-registration call. Supports:
//   - Simple identifier: listUsers → "listUsers"
//   - Selector expression: handler.ListUsers → "handler.ListUsers"
//   - Any other form (closure, call): returns ""
func extractHandlerName(expr ast.Expr) string {
	switch arg := expr.(type) {
	case *ast.Ident:
		return arg.Name
	case *ast.SelectorExpr:
		if x, ok := arg.X.(*ast.Ident); ok {
			return x.Name + "." + arg.Sel.Name
		}
		return arg.Sel.Name
	default:
		return ""
	}
}

// extractGroupPrefix checks if an expression is an echo.Group() call
// and returns the prefix string literal from its first argument.
// For example, `e.Group("/v1")` returns "/v1".
func extractGroupPrefix(expr ast.Expr) string {
	callExpr, ok := expr.(*ast.CallExpr)
	if !ok {
		return ""
	}

	selExpr, ok := callExpr.Fun.(*ast.SelectorExpr)
	if !ok {
		return ""
	}

	if selExpr.Sel.Name != "Group" {
		return ""
	}

	if len(callExpr.Args) < 1 {
		return ""
	}

	return extractStringLiteral(callExpr.Args[0])
}

// resolveReceiverVar returns the variable name of a selector expression's
// receiver (the X part of X.Method). Used to look up group prefixes later.
// Returns "" if the receiver is not a simple identifier.
func resolveReceiverVar(expr ast.Expr) string {
	ident, ok := expr.(*ast.Ident)
	if !ok {
		return ""
	}
	return ident.Name
}

// importsPackage checks whether the given file imports the specified package path.
func importsPackage(f *ast.File, pkgPath string) bool {
	for _, imp := range f.Imports {
		path := strings.Trim(imp.Path.Value, `"`)
		if path == pkgPath {
			return true
		}
	}
	return false
}
