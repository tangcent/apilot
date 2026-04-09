// Package fiber parses Fiber route registrations using Go's standard go/ast package.
package fiber

import "github.com/tangcent/apilot/api-collector/collector"

// Parse extracts endpoints from Fiber route registrations in the given source directory.
func Parse(sourceDir string) ([]collector.ApiEndpoint, error) {
	// TODO: implement using go/parser and go/ast
	return nil, nil
}
