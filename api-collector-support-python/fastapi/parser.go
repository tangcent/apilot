// Package fastapi parses FastAPI decorated route functions.
package fastapi

import "github.com/tangcent/apilot/api-collector/collector"

// Parse extracts endpoints from FastAPI @app.get, @router.post, etc. decorated functions.
func Parse(sourceDir string) ([]collector.ApiEndpoint, error) {
	// TODO: implement using tree-sitter-python
	return nil, nil
}
