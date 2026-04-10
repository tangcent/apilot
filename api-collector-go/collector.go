// Package gocollector implements the Collector interface for Go projects.
// Supported frameworks: Gin, Echo, Fiber.
package gocollector

import "github.com/tangcent/apilot/api-collector"

// GoCollector parses Go source trees for API route registrations.
type GoCollector struct{}

// New returns a new GoCollector.
func New() collector.Collector { return &GoCollector{} }

func (c *GoCollector) Name() string { return "go" }

func (c *GoCollector) SupportedLanguages() []string { return []string{"go"} }

// Collect walks the source directory and extracts endpoints from Gin, Echo, and Fiber route registrations.
func (c *GoCollector) Collect(ctx collector.CollectContext) ([]collector.ApiEndpoint, error) {
	// TODO: implement
	// 1. Walk .go files under ctx.SourceDir using go/parser
	// 2. Delegate to gin/, echo/, fiber/ sub-packages
	return nil, nil
}
