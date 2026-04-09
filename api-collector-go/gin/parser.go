// Package gin parses Gin route registrations using Go's standard go/ast package.
package gin

import "github.com/tangcent/apilot/api-collector/collector"

// Parse extracts endpoints from Gin route registrations in the given source directory.
// It walks the AST for gin.RouterGroup.GET/POST/PUT/DELETE/PATCH/HEAD/OPTIONS calls.
func Parse(sourceDir string) ([]collector.ApiEndpoint, error) {
	// TODO: implement using go/parser and go/ast
	return nil, nil
}
