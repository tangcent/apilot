// Package echo parses Echo route registrations using Go's standard go/ast package.
package echo

import "github.com/tangcent/apilot/api-collector"

// Parse extracts endpoints from Echo route registrations in the given source directory.
func Parse(sourceDir string) ([]collector.ApiEndpoint, error) {
	// TODO: implement using go/parser and go/ast
	return nil, nil
}
