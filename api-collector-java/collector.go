// Package javacollector implements the Collector interface for Java/Kotlin projects.
// Supported frameworks: Spring MVC, JAX-RS, Feign.
package javacollector

import "github.com/tangcent/apilot/api-collector/collector"

// JavaCollector parses Java/Kotlin source trees for API endpoints.
type JavaCollector struct{}

// New returns a new JavaCollector.
func New() collector.Collector { return &JavaCollector{} }

func (c *JavaCollector) Name() string { return "java" }

func (c *JavaCollector) SupportedLanguages() []string { return []string{"java", "kotlin"} }

// Collect walks the source directory and extracts endpoints from Spring MVC, JAX-RS, and Feign sources.
// @requires ReadAction context (PSI reads happen inside the collector)
func (c *JavaCollector) Collect(ctx collector.CollectContext) ([]collector.ApiEndpoint, error) {
	// TODO: implement
	// 1. Discover .java / .kt files under ctx.SourceDir
	// 2. Optionally invoke maven-indexer-cli for dependency resolution
	// 3. Parse each file with a Java grammar (tree-sitter-java or antlr4)
	// 4. Delegate to springmvc/, jaxrs/, feign/ sub-packages
	return nil, nil
}
