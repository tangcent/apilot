// Package nodecollector implements the Collector interface for Node.js/TypeScript projects.
// Supported frameworks: Express, Fastify, NestJS.
package nodecollector

import (
	"log"
	"os"
	"path/filepath"
	"sync"

	collector "github.com/tangcent/apilot/api-collector"
	"github.com/tangcent/apilot/api-collector-node/express"
	"github.com/tangcent/apilot/api-collector-node/fastify"
	"github.com/tangcent/apilot/api-collector-node/nestjs"
	"github.com/tangcent/apilot/api-collector-node/npm"
)

// NodeCollector parses TypeScript/JavaScript source trees for API route definitions.
type NodeCollector struct {
	dependencyResolver collector.DependencyResolver
	mu                 sync.Mutex
	unresolved         map[string]int
	depResolution      collector.DependencyResolution
}

// resolvedCounter is implemented by dependency resolvers that can report how
// many distinct types they expanded.
type resolvedCounter interface {
	ResolvedCount() int
}

func New() collector.Collector { return &NodeCollector{} }

func (c *NodeCollector) Name() string { return "node" }

func (c *NodeCollector) SupportedLanguages() []string { return []string{"typescript", "javascript"} }

func (c *NodeCollector) SetDependencyResolver(dr collector.DependencyResolver) {
	c.dependencyResolver = dr
}

// Unresolved reports type names the framework resolvers could not expand during
// the last Collect call, mapped to occurrence counts.
func (c *NodeCollector) Unresolved() map[string]int {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make(map[string]int, len(c.unresolved))
	for name, n := range c.unresolved {
		out[name] = n
	}
	return out
}

// DependencyResolution reports whether dependency-based type resolution ran
// during the last Collect call, and why not when it did not.
func (c *NodeCollector) DependencyResolution() collector.DependencyResolution {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.depResolution
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

	depResolution, depResolver := c.setupDependencyResolution(ctx)

	// Shared across framework parsers: a type may be referenced by handlers
	// written for more than one framework.
	unresolved := collector.NewUnresolvedSet()

	parsers := []struct {
		name  string
		parse func(string) ([]collector.ApiEndpoint, error)
	}{
		{"express", func(dir string) ([]collector.ApiEndpoint, error) {
			return express.ParseWithUnresolved(dir, depResolver, unresolved)
		}},
		{"fastify", func(dir string) ([]collector.ApiEndpoint, error) {
			return fastify.ParseWithUnresolved(dir, depResolver, unresolved)
		}},
		{"nestjs", func(dir string) ([]collector.ApiEndpoint, error) {
			return nestjs.ParseWithUnresolved(dir, depResolver, unresolved)
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

	c.mu.Lock()
	c.unresolved = unresolved.Counts()
	if rc, ok := depResolver.(resolvedCounter); ok {
		depResolution.ResolvedTypes = rc.ResolvedCount()
	}
	c.depResolution = depResolution
	c.mu.Unlock()

	if len(all) == 0 {
		return nil, nil
	}

	return all, nil
}

// setupDependencyResolution decides whether dependency-based type resolution
// runs for this Collect call, and why not when it does not.
//
// npm has no external tool requirement of its own: the resolvers read .d.ts
// files straight out of node_modules. The prerequisites are therefore a
// package.json declaring dependencies and an npm install having run.
func (c *NodeCollector) setupDependencyResolution(ctx collector.CollectContext) (collector.DependencyResolution, collector.DependencyResolver) {
	if ctx.NoDeps {
		return collector.DependencyResolution{State: collector.DependencyResolutionDisabled}, nil
	}

	if c.dependencyResolver != nil {
		// Wired by a host application; its availability is the host's concern.
		return collector.DependencyResolution{State: collector.DependencyResolutionActive}, c.dependencyResolver
	}

	if _, err := os.Stat(filepath.Join(ctx.SourceDir, "package.json")); err != nil {
		return collector.DependencyResolution{State: collector.DependencyResolutionNoDeps}, nil
	}

	deps, err := npm.DetectNpmDependencies(ctx.SourceDir)
	if err != nil || len(deps) == 0 {
		return collector.DependencyResolution{State: collector.DependencyResolutionNoDeps}, nil
	}

	if !dirExists(filepath.Join(ctx.SourceDir, "node_modules")) {
		return collector.DependencyResolution{
			State: collector.DependencyResolutionToolMissing,
			Detail: "node_modules not found — request and response types from the declared packages cannot be resolved. " +
				"Run npm install (or your package manager's install command) and retry",
		}, nil
	}

	return collector.DependencyResolution{State: collector.DependencyResolutionActive}, NewNodeDependencyResolver(ctx.SourceDir)
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
