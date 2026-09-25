// Package gocollector implements the Collector interface for Go projects.
// Supported frameworks: Gin, Echo, Fiber.
package gocollector

import (
	"log"
	"sync"

	"github.com/tangcent/apilot/api-collector"
	"github.com/tangcent/apilot/api-collector-go/echo"
	"github.com/tangcent/apilot/api-collector-go/fiber"
	"github.com/tangcent/apilot/api-collector-go/gin"
)

// GoCollector parses Go source trees for API route registrations.
type GoCollector struct {
	dependencyResolver collector.DependencyResolver
	mu                 sync.Mutex
	unresolved         map[string]int
}

func New() collector.Collector { return &GoCollector{} }

func (c *GoCollector) Name() string { return "go" }

func (c *GoCollector) SupportedLanguages() []string { return []string{"go"} }

func (c *GoCollector) SetDependencyResolver(dr collector.DependencyResolver) {
	c.dependencyResolver = dr
}

// Unresolved reports type names the framework resolvers could not expand during
// the last Collect call, mapped to occurrence counts.
func (c *GoCollector) Unresolved() map[string]int {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make(map[string]int, len(c.unresolved))
	for name, n := range c.unresolved {
		out[name] = n
	}
	return out
}

// Collect walks the source directory and extracts endpoints from Gin, Echo,
// and Fiber route registrations.
//
// Each framework parser is invoked concurrently. Results are merged into a
// single slice. If a parser returns an error, a warning is logged and
// collection continues with the remaining parsers — the overall Collect call
// does not fail.
func (c *GoCollector) Collect(ctx collector.CollectContext) ([]collector.ApiEndpoint, error) {
	type parseResult struct {
		endpoints []collector.ApiEndpoint
		err       error
		framework string
	}

	var depResolver collector.DependencyResolver
	if c.dependencyResolver != nil {
		depResolver = c.dependencyResolver
	} else {
		depResolver = NewGoDependencyResolver(ctx.SourceDir)
	}

	// Shared across framework parsers: a type may be referenced by handlers
	// written for more than one framework.
	unresolved := collector.NewUnresolvedSet()

	parsers := []struct {
		name  string
		parse func(string, *collector.UnresolvedSet, ...collector.DependencyResolver) ([]collector.ApiEndpoint, error)
	}{
		{"gin", gin.ParseWithUnresolved},
		{"echo", echo.ParseWithUnresolved},
		{"fiber", fiber.ParseWithUnresolved},
	}

	ch := make(chan parseResult, len(parsers))
	var wg sync.WaitGroup

	for _, p := range parsers {
		wg.Add(1)
		go func(name string, fn func(string, *collector.UnresolvedSet, ...collector.DependencyResolver) ([]collector.ApiEndpoint, error)) {
			defer wg.Done()
			endpoints, err := fn(ctx.SourceDir, unresolved, depResolver)
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

	c.mu.Lock()
	c.unresolved = unresolved.Counts()
	c.mu.Unlock()

	if len(all) == 0 {
		return nil, nil
	}

	return all, nil
}
