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
}

func New() collector.Collector { return &GoCollector{} }

func (c *GoCollector) Name() string { return "go" }

func (c *GoCollector) SupportedLanguages() []string { return []string{"go"} }

func (c *GoCollector) SetDependencyResolver(dr collector.DependencyResolver) {
	c.dependencyResolver = dr
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

	parsers := []struct {
		name    string
		parse   func(string) ([]collector.ApiEndpoint, error)
	}{
		{"gin", gin.Parse},
		{"echo", echo.Parse},
		{"fiber", fiber.Parse},
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
