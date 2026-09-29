// Package pycollector implements the Collector interface for Python projects.
// Supported frameworks: FastAPI, Django REST Framework, Flask.
package pycollector

import (
	"log"
	"sync"

	"github.com/tangcent/apilot/api-collector"
	"github.com/tangcent/apilot/api-collector-python/django"
	"github.com/tangcent/apilot/api-collector-python/fastapi"
	"github.com/tangcent/apilot/api-collector-python/flask"
	"github.com/tangcent/apilot/api-collector-python/pip"
)

// PythonCollector parses Python source trees for API route definitions.
type PythonCollector struct {
	dependencyResolver collector.DependencyResolver
	mu                 sync.Mutex
	unresolved         map[string]int
	depResolution      collector.DependencyResolution
}

// findSitePackages locates the Python environment to read dependency types
// from. It is a variable so tests can force the tool-missing branch without
// uninstalling Python from the machine.
var findSitePackages = pip.FindSitePackages

// resolvedCounter is implemented by dependency resolvers that can report how
// many distinct types they expanded.
type resolvedCounter interface {
	ResolvedCount() int
}

func New() collector.Collector { return &PythonCollector{} }

func (c *PythonCollector) Name() string { return "python" }

func (c *PythonCollector) SupportedLanguages() []string { return []string{"python"} }

func (c *PythonCollector) SetDependencyResolver(dr collector.DependencyResolver) {
	c.dependencyResolver = dr
}

// Unresolved reports type names the framework resolvers could not expand during
// the last Collect call, mapped to occurrence counts.
func (c *PythonCollector) Unresolved() map[string]int {
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
func (c *PythonCollector) DependencyResolution() collector.DependencyResolution {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.depResolution
}

// Collect walks the source directory and extracts endpoints from FastAPI, Django REST, and Flask sources.
// Each framework parser is invoked concurrently. Results are merged into a
// single slice. If a parser returns an error, a warning is logged and
// collection continues with the remaining parsers.
func (c *PythonCollector) Collect(ctx collector.CollectContext) ([]collector.ApiEndpoint, error) {
	type parseResult struct {
		endpoints []collector.ApiEndpoint
		err       error
		framework string
	}

	depResolution, depResolver := c.setupDependencyResolution(ctx)

	// Shared across framework parsers: a type may be referenced by endpoints
	// written for more than one framework.
	unresolved := collector.NewUnresolvedSet()

	parsers := []struct {
		name  string
		parse func(string, *collector.UnresolvedSet) ([]collector.ApiEndpoint, error)
	}{
		{"fastapi", func(dir string, u *collector.UnresolvedSet) ([]collector.ApiEndpoint, error) {
			return fastapi.ParseWithUnresolved(dir, depResolver, u)
		}},
		{"django", func(dir string, u *collector.UnresolvedSet) ([]collector.ApiEndpoint, error) {
			return django.ParseWithUnresolved(dir, depResolver, u)
		}},
		{"flask", func(dir string, u *collector.UnresolvedSet) ([]collector.ApiEndpoint, error) {
			return flask.ParseWithUnresolved(dir, depResolver, u)
		}},
	}

	ch := make(chan parseResult, len(parsers))
	var wg sync.WaitGroup

	for _, p := range parsers {
		wg.Add(1)
		go func(name string, fn func(string, *collector.UnresolvedSet) ([]collector.ApiEndpoint, error)) {
			defer wg.Done()
			endpoints, err := fn(ctx.SourceDir, unresolved)
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
// Types outside the project come from pip dependency packages, read from the
// site-packages directory of the project's Python environment. Without a
// manifest there is nothing to resolve; without a discoverable environment
// the resolver has nowhere to read from.
func (c *PythonCollector) setupDependencyResolution(ctx collector.CollectContext) (collector.DependencyResolution, collector.DependencyResolver) {
	if ctx.NoDeps {
		return collector.DependencyResolution{State: collector.DependencyResolutionDisabled}, nil
	}

	if c.dependencyResolver != nil {
		// Wired by a host application; its availability is the host's concern.
		return collector.DependencyResolution{State: collector.DependencyResolutionActive}, c.dependencyResolver
	}

	if !pip.HasDependencyManifest(ctx.SourceDir) {
		return collector.DependencyResolution{State: collector.DependencyResolutionNoDeps}, nil
	}

	if _, err := findSitePackages(ctx.SourceDir); err != nil {
		return collector.DependencyResolution{
			State: collector.DependencyResolutionToolMissing,
			Detail: err.Error(),
		}, nil
	}

	return collector.DependencyResolution{State: collector.DependencyResolutionActive}, NewPythonDependencyResolver(ctx.SourceDir)
}
