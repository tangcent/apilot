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
)

// PythonCollector parses Python source trees for API route definitions.
type PythonCollector struct{}

// New returns a new PythonCollector.
func New() collector.Collector { return &PythonCollector{} }

func (c *PythonCollector) Name() string { return "python" }

func (c *PythonCollector) SupportedLanguages() []string { return []string{"python"} }

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

	parsers := []struct {
		name  string
		parse func(string) ([]collector.ApiEndpoint, error)
	}{
		{"fastapi", fastapi.Parse},
		{"django", django.Parse},
		{"flask", flask.Parse},
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
