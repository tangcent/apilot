package engine

import (
	"fmt"

	"github.com/tangcent/apilot/api-collector/collector"
	"github.com/tangcent/apilot/api-formatter/formatter"
)

var (
	collectors = map[string]collector.Collector{}
	formatters = map[string]formatter.Formatter{}
)

// RegisterCollector adds a Collector to the in-process registry.
func RegisterCollector(c collector.Collector) {
	collectors[c.Name()] = c
}

// RegisterFormatter adds a Formatter to the in-process registry.
func RegisterFormatter(f formatter.Formatter) {
	formatters[f.Name()] = f
}

// LookupCollector returns the named Collector or an error if not found.
func LookupCollector(name string) (collector.Collector, error) {
	c, ok := collectors[name]
	if !ok {
		return nil, fmt.Errorf("collector %q not registered", name)
	}
	return c, nil
}

// LookupFormatter returns the named Formatter or an error if not found.
func LookupFormatter(name string) (formatter.Formatter, error) {
	f, ok := formatters[name]
	if !ok {
		return nil, fmt.Errorf("formatter %q not registered", name)
	}
	return f, nil
}

// ListCollectors returns all registered collector names and their supported languages.
func ListCollectors() map[string][]string {
	out := make(map[string][]string, len(collectors))
	for name, c := range collectors {
		out[name] = c.SupportedLanguages()
	}
	return out
}

// ListFormatters returns all registered formatter names and their supported formats.
func ListFormatters() map[string][]string {
	out := make(map[string][]string, len(formatters))
	for name, f := range formatters {
		out[name] = f.SupportedFormats()
	}
	return out
}
