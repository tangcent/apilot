// Package collector defines the stable interface contract for all API collector implementations.
// A Collector parses source code for a specific language/framework and produces []ApiEndpoint.
package collector

// Collector is the interface every language/framework collector must implement.
type Collector interface {
	// Name returns the unique identifier for this collector (e.g. "java", "go").
	Name() string

	// SupportedLanguages returns the list of language identifiers this collector handles.
	SupportedLanguages() []string

	// Collect parses the source directory described by ctx and returns the discovered endpoints.
	Collect(ctx CollectContext) ([]ApiEndpoint, error)
}

// CollectContext carries the inputs needed by a Collector to perform collection.
type CollectContext struct {
	// SourceDir is the absolute path to the root of the source tree to parse.
	SourceDir string `json:"sourceDir"`

	// Frameworks is an optional list of framework hints (e.g. ["spring-mvc", "feign"]).
	Frameworks []string `json:"frameworks,omitempty"`

	// Config holds collector-specific key-value configuration.
	Config map[string]string `json:"config,omitempty"`
}
