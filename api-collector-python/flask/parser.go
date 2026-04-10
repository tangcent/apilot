// Package flask parses Flask route decorators.
package flask

import "github.com/tangcent/apilot/api-collector"

// Parse extracts endpoints from Flask @app.route and @blueprint.route decorated functions.
func Parse(sourceDir string) ([]collector.ApiEndpoint, error) {
	// TODO: implement using tree-sitter-python
	return nil, nil
}
