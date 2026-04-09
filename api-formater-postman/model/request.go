package model

// Request describes a Postman HTTP request.
type Request struct {
	Method string   `json:"method"`
	Header []Header `json:"header,omitempty"`
	URL    URL      `json:"url"`
	Body   *Body    `json:"body,omitempty"`
}

// Header is a single HTTP header in a Postman request.
type Header struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// URL holds the parsed URL components of a Postman request.
type URL struct {
	Raw   string   `json:"raw"`
	Path  []string `json:"path,omitempty"`
	Query []Query  `json:"query,omitempty"`
}

// Query is a single query parameter.
type Query struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// Body holds the request body definition.
type Body struct {
	Mode string `json:"mode"` // "raw"
	Raw  string `json:"raw,omitempty"`
	Options *BodyOptions `json:"options,omitempty"`
}

// BodyOptions specifies the language for raw body mode.
type BodyOptions struct {
	Raw RawOptions `json:"raw"`
}

// RawOptions specifies the raw body language.
type RawOptions struct {
	Language string `json:"language"` // "json"
}
