// Package gin parses Gin route registrations using Go's standard go/ast package.
// It walks Go source files for gin.RouterGroup and gin.Engine method calls
// (GET, POST, PUT, DELETE, PATCH, HEAD, OPTIONS) and extracts endpoint metadata
// including path, HTTP method, handler name, and doc comments.
package gin

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"sync"

	collector "github.com/tangcent/apilot/api-collector"
)

// httpMethods maps Gin route-registration method names that we recognize.
var httpMethods = map[string]bool{
	"GET":     true,
	"POST":    true,
	"PUT":     true,
	"DELETE":  true,
	"PATCH":   true,
	"HEAD":    true,
	"OPTIONS": true,
}

// Parse extracts endpoints from Gin route registrations in the given source directory.
//
// It performs three passes over the parsed AST files:
//  1. Collect function doc comments — builds a map from function name to its
//     Go doc comment text, used later to populate ApiEndpoint.Description.
//  2. Collect RouterGroup prefixes — finds assignments like
//     `v1 := r.Group("/v1")` and records the variable-to-prefix mapping,
//     so that subsequent route calls on that variable (e.g. `v1.GET("/users", ...)`)
//     resolve to the full path `/v1/users`.
//  3. Extract route registrations — finds method calls matching the Gin HTTP
//     method names (GET, POST, PUT, DELETE, PATCH, HEAD, OPTIONS) and extracts
//     the path (first argument as a string literal), HTTP method (from the call
//     selector name), handler function name, and the handler's doc comment.
//
// Memory optimization:
//   - Files are discovered first via filepath.Walk, then parsed and processed
//     one at a time in goroutines. Each goroutine processes a single file and
//     sends its partial results (func docs, group prefixes, endpoints) through
//     channels, so we never hold all ASTs in memory simultaneously beyond what
//     is needed for the current file.
//   - The func-doc and group-prefix maps are merged incrementally as results
//     arrive; endpoint extraction is deferred until all files have contributed
//     their prefix information.
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
	var allRaw []rawEndpoint

	for res := range ch {
		for k, v := range res.funcDocs {
			funcDocs[k] = v
		}
		for k, v := range res.groupPrefixes {
			groupPrefixes[k] = v
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

		description := ""
		if raw.handlerName != "" {
			shortName := raw.handlerName
			if idx := strings.LastIndex(shortName, "."); idx >= 0 {
				shortName = shortName[idx+1:]
			}
			description = funcDocs[shortName]
		}

		endpoints = append(endpoints, collector.ApiEndpoint{
			Name:        raw.handlerName,
			Path:        path,
			Method:      raw.method,
			Protocol:    "http",
			Description: description,
		})
	}

	return endpoints, nil
}

// rawEndpoint holds the raw data extracted from a single route-registration call
// before group-prefix resolution and doc-comment lookup.
type rawEndpoint struct {
	method      string
	path        string
	handlerName string
	receiverVar string
}

// fileResult holds the per-file extraction output produced by processFile.
type fileResult struct {
	funcDocs      map[string]string
	groupPrefixes map[string]string
	rawEndpoints  []rawEndpoint
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
//   - groupPrefixes: map of variable names to their RouterGroup prefix
//   - rawEndpoints: route registrations found in this file
func processFile(filePath string) fileResult {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		return fileResult{}
	}

	funcDocs := make(map[string]string)
	for _, decl := range f.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Doc == nil {
			continue
		}
		funcDocs[fn.Name.Name] = strings.TrimSpace(fn.Doc.Text())
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
		if !httpMethods[methodName] {
			return true
		}

		if len(callExpr.Args) < 2 {
			return true
		}

		path := extractStringLiteral(callExpr.Args[0])
		handlerName := extractHandlerName(callExpr.Args[1])
		receiverVar := resolveReceiverVar(selExpr.X)

		rawEndpoints = append(rawEndpoints, rawEndpoint{
			method:      methodName,
			path:        path,
			handlerName: handlerName,
			receiverVar: receiverVar,
		})
		return true
	})

	return fileResult{
		funcDocs:      funcDocs,
		groupPrefixes: groupPrefixes,
		rawEndpoints:  rawEndpoints,
	}
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
// of a Gin route-registration call. Supports:
//   - Simple identifier: listUsers -> "listUsers"
//   - Selector expression: handler.ListUsers -> "handler.ListUsers"
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

// extractGroupPrefix checks if an expression is a RouterGroup.Group() call
// and returns the prefix string literal from its first argument.
// For example, `r.Group("/v1")` returns "/v1".
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
