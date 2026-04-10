// Package express parses Express route registrations.
package express

import "github.com/tangcent/apilot/api-collector"

// Parse extracts endpoints from Express route registrations (app.get, router.use, etc.).
func Parse(sourceDir string) ([]collector.ApiEndpoint, error) {
	// TODO: implement using tree-sitter-typescript or AST analysis
	return nil, nil
}
