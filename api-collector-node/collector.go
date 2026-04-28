// Package nodecollector implements the Collector interface for Node.js/TypeScript projects.
// Supported frameworks: Express, Fastify, NestJS.
package nodecollector

import (
	"log"
	"sync"

	collector "github.com/tangcent/apilot/api-collector"
	"github.com/tangcent/apilot/api-collector-node/express"
	"github.com/tangcent/apilot/api-collector-node/fastify"
	"github.com/tangcent/apilot/api-collector-node/nestjs"
)

// NodeCollector parses TypeScript/JavaScript source trees for API route definitions.
type NodeCollector struct {
	dependencyResolver collector.DependencyResolver
}

func New() collector.Collector { return &NodeCollector{} }

func (c *NodeCollector) Name() string { return "node" }

func (c *NodeCollector) SupportedLanguages() []string { return []string{"typescript", "javascript"} }

func (c *NodeCollector) SetDependencyResolver(dr collector.DependencyResolver) {
	c.dependencyResolver = dr
}

// Collect walks the source directory and extracts endpoints from Express, Fastify, and NestJS sources.
// Each framework parser is invoked concurrently. Results are merged into a
// single slice. If a parser returns an error, a warning is logged and
// collection continues with the remaining parsers.
func (c *NodeCollector) Collect(ctx collector.CollectContext) ([]collector.ApiEndpoint, error) {
	type parseResult struct {
		endpoints []collector.ApiEndpoint
		err       error
		framework string
	}

	parsers := []struct {
		name  string
		parse func(string) ([]collector.ApiEndpoint, error)
	}{
		{"express", func(dir string) ([]collector.ApiEndpoint, error) {
			if c.dependencyResolver != nil {
				return express.ParseWithDependencyResolver(dir, c.dependencyResolver)
			}
			return express.Parse(dir)
		}},
		{"fastify", func(dir string) ([]collector.ApiEndpoint, error) {
			if c.dependencyResolver != nil {
				return fastify.ParseWithDependencyResolver(dir, c.dependencyResolver)
			}
			return fastify.Parse(dir)
		}},
		{"nestjs", func(dir string) ([]collector.ApiEndpoint, error) {
			if c.dependencyResolver != nil {
				return nestjs.ParseWithDependencyResolver(dir, c.dependencyResolver)
			}
			return nestjs.Parse(dir)
		}},
	}

	ch := make(chan parseResult, len(parsers))
	var wg sync.WaitGroup

	for _, p := range parsers {
		wg.Add(1)
		go func(name string, fn func(string) ([]collector.ApiEndpoint, error)) {
			defer wg.Done()
			endpoints, err := fn(ctx.SourceDir)
			ch <- parseResult{endpoints: endpoints, err: err, framework: name}
		}(p.name, p.parse)
	}

	go func() {
		wg.Wait()
		close(ch)
	}()

	var all []collector.ApiEndpoint
	for res := range ch {
		if res.err != nil {
			log.Printf("warning: %s parser failed: %v", res.framework, res.err)
			continue
		}
		all = append(all, res.endpoints...)
	}

	if len(all) == 0 {
		return nil, nil
	}

	return all, nil
}
