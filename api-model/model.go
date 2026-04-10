// Package model defines the canonical, language-agnostic data types shared
// between all API collectors and formatters.
// This is the only module both sides of the pipeline need to import for types.
package model

// ApiEndpoint is the canonical, language-agnostic model for a single API endpoint.
type ApiEndpoint struct {
	Name        string         `json:"name"`
	Folder      string         `json:"folder,omitempty"`
	Description string         `json:"description,omitempty"`
	Tags        []string       `json:"tags,omitempty"`
	Path        string         `json:"path"`
	Method      string         `json:"method,omitempty"`   // empty for non-HTTP protocols
	Protocol    string         `json:"protocol"`           // "http", "grpc", "websocket", etc.
	Parameters  []ApiParameter `json:"parameters,omitempty"`
	Headers     []ApiHeader    `json:"headers,omitempty"`
	RequestBody *ApiBody       `json:"requestBody,omitempty"`
	Response    *ApiBody       `json:"response,omitempty"`
	// Metadata holds protocol-specific extensions without polluting the core struct.
	Metadata map[string]any `json:"metadata,omitempty"`
}

// ApiParameter describes a single input parameter of an endpoint.
type ApiParameter struct {
	Name        string   `json:"name"`
	Type        string   `json:"type"`              // "text" | "file"
	Required    bool     `json:"required"`
	In          string   `json:"in"`                // "query" | "path" | "header" | "cookie" | "body" | "form"
	Default     string   `json:"default,omitempty"`
	Description string   `json:"description,omitempty"`
	Example     string   `json:"example,omitempty"`
	Enum        []string `json:"enum,omitempty"`
}

// ApiHeader describes an HTTP header associated with an endpoint.
type ApiHeader struct {
	Name        string `json:"name"`
	Value       string `json:"value,omitempty"`
	Description string `json:"description,omitempty"`
	Example     string `json:"example,omitempty"`
	Required    bool   `json:"required"`
}

// ApiBody describes the request or response body of an endpoint.
type ApiBody struct {
	MediaType string `json:"mediaType,omitempty"` // e.g. "application/json"
	Schema    any    `json:"schema,omitempty"`    // JSON Schema object
	Example   any    `json:"example,omitempty"`
}
