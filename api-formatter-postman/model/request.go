package model

// Request describes a Postman HTTP request.
type Request struct {
	Method      string   `json:"method"`
	Header      []Header `json:"header,omitempty"`
	Body        *Body    `json:"body,omitempty"`
	URL         URL      `json:"url"`
	Description string   `json:"description,omitempty"`
}

// Header is a single HTTP header in a Postman request or response.
type Header struct {
	Key         string `json:"key"`
	Value       string `json:"value"`
	Type        string `json:"type,omitempty"`
	Description string `json:"description,omitempty"`
	Name        string `json:"name,omitempty"`
}

// URL holds the parsed URL components of a Postman request.
type URL struct {
	Raw      string         `json:"raw"`
	Host     []string       `json:"host,omitempty"`
	Path     []string       `json:"path,omitempty"`
	Query    []Query        `json:"query,omitempty"`
	Variable []PathVariable `json:"variable,omitempty"`
}

// Query is a single query parameter.
type Query struct {
	Key         string `json:"key"`
	Value       string `json:"value"`
	Equals      bool   `json:"equals"`
	Description string `json:"description,omitempty"`
	Disabled    bool   `json:"disabled,omitempty"`
}

// PathVariable is a single path variable in a Postman URL.
type PathVariable struct {
	Key         string `json:"key"`
	Value       string `json:"value"`
	Description string `json:"description,omitempty"`
}

// Body holds the request body definition.
type Body struct {
	Mode       string       `json:"mode"`
	Raw        string       `json:"raw,omitempty"`
	Urlencoded []FormParam  `json:"urlencoded,omitempty"`
	Formdata   []FormParam  `json:"formdata,omitempty"`
	Options    *BodyOptions `json:"options,omitempty"`
}

// BodyOptions specifies the language for raw body mode.
type BodyOptions struct {
	Raw RawOptions `json:"raw"`
}

// RawOptions specifies the raw body language.
type RawOptions struct {
	Language string `json:"language"`
}

// FormParam is a form parameter for urlencoded or formdata body modes.
type FormParam struct {
	Key         string `json:"key"`
	Value       string `json:"value"`
	Type        string `json:"type"`
	Description string `json:"description,omitempty"`
	Disabled    bool   `json:"disabled,omitempty"`
	Src         string `json:"src,omitempty"`
}
