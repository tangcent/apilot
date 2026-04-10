// Package nodecollector implements the Collector interface for Node.js/TypeScript projects.
// Supported frameworks: Express, Fastify, NestJS.
package nodecollector

import "github.com/tangcent/apilot/api-collector"

// NodeCollector parses TypeScript/JavaScript source trees for API route definitions.
type NodeCollector struct{}

// New returns a new NodeCollector.
func New() collector.Collector { return &NodeCollector{} }

func (c *NodeCollector) Name() string { return "node" }

func (c *NodeCollector) SupportedLanguages() []string { return []string{"typescript", "javascript"} }

// Collect walks the source directory and extracts endpoints from Express, Fastify, and NestJS sources.
func (c *NodeCollector) Collect(ctx collector.CollectContext) ([]collector.ApiEndpoint, error) {
	// TODO: implement
	// Preferred: tree-sitter-typescript Go bindings (no external runtime dependency)
	// Fallback: invoke node/ts-node subprocess
	return nil, nil
}
