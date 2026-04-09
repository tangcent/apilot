// Package formatter defines the stable interface contract for all API formatter implementations.
// A Formatter converts []ApiEndpoint into a specific output format (Markdown, cURL, Postman, etc.).
package formatter

import "github.com/tangcent/apilot/api-collector/collector"

// Formatter is the interface every output formatter must implement.
type Formatter interface {
	// Name returns the unique identifier for this formatter (e.g. "markdown", "postman").
	Name() string

	// SupportedFormats returns the list of format variant names this formatter handles.
	SupportedFormats() []string

	// Format converts the given endpoints into the target format.
	// An empty endpoints slice MUST return valid empty output, not an error.
	Format(endpoints []collector.ApiEndpoint, opts FormatOptions) ([]byte, error)
}

// FormatOptions carries the inputs needed by a Formatter.
type FormatOptions struct {
	// Format selects the output variant (e.g. "simple", "detailed").
	Format string `json:"format"`

	// Config holds formatter-specific key-value configuration.
	Config map[string]string `json:"config,omitempty"`
}
