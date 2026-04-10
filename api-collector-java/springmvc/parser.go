// Package springmvc parses Spring MVC controller classes for API endpoints.
package springmvc

import "github.com/tangcent/apilot/api-collector"

// Parse extracts endpoints from Spring MVC annotated source files under sourceDir.
// It looks for @RestController, @RequestMapping, @GetMapping, @PostMapping, etc.
func Parse(sourceDir string) ([]collector.ApiEndpoint, error) {
	// TODO: implement
	return nil, nil
}
