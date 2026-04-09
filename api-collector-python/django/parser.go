// Package django parses Django REST Framework views and URL patterns.
package django

import "github.com/tangcent/apilot/api-collector/collector"

// Parse extracts endpoints from Django REST Framework @api_view, APIView, ViewSet classes
// and urlpatterns definitions.
func Parse(sourceDir string) ([]collector.ApiEndpoint, error) {
	// TODO: implement using tree-sitter-python
	return nil, nil
}
