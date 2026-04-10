// Package pycollector implements the Collector interface for Python projects.
// Supported frameworks: FastAPI, Django REST Framework, Flask.
package pycollector

import "github.com/tangcent/apilot/api-collector"

// PythonCollector parses Python source trees for API route definitions.
type PythonCollector struct{}

// New returns a new PythonCollector.
func New() collector.Collector { return &PythonCollector{} }

func (c *PythonCollector) Name() string { return "python" }

func (c *PythonCollector) SupportedLanguages() []string { return []string{"python"} }

// Collect walks the source directory and extracts endpoints from FastAPI, Django REST, and Flask sources.
func (c *PythonCollector) Collect(ctx collector.CollectContext) ([]collector.ApiEndpoint, error) {
	// TODO: implement using tree-sitter-python Go bindings
	return nil, nil
}
