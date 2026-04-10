// Package formatter defines the stable interface contract for all API formatter implementations.
// A Formatter converts []model.ApiEndpoint into a specific output format (Markdown, cURL, Postman, etc.).
package formatter

import (
	"encoding/json"

	"github.com/tangcent/apilot/api-model"
)

// Formatter is the interface every output formatter must implement.
type Formatter interface {
	// Name returns the unique identifier for this formatter (e.g. "markdown", "postman").
	Name() string

	// Format converts the given endpoints into the target format.
	// An empty endpoints slice MUST return valid empty output, not an error.
	Format(endpoints []model.ApiEndpoint, opts FormatOptions) ([]byte, error)
}

// FormatOptions carries formatter-specific configuration as raw JSON.
// Each formatter implementation decodes Params into its own typed options struct
// via DecodeParams, so there are no magic string keys and options are self-documenting.
//
// Example — passing options from the CLI or IDE:
//
//	FormatOptions{Params: json.RawMessage(`{"variant":"detailed","baseURL":"https://api.example.com"}`)}
type FormatOptions struct {
	// Params holds formatter-specific configuration as raw JSON.
	// Formatters unmarshal this into their own typed struct using DecodeParams.
	Params json.RawMessage `json:"params,omitempty"`
}

// DecodeParams unmarshals Params into v.
// If Params is empty, v is left unchanged and nil is returned.
func (o FormatOptions) DecodeParams(v any) error {
	if len(o.Params) == 0 {
		return nil
	}
	return json.Unmarshal(o.Params, v)
}
