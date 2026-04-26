package model

// Response represents an example response saved in a Postman request.
type Response struct {
	Name                    string    `json:"name"`
	OriginalRequest         *Request  `json:"originalRequest,omitempty"`
	Status                  string    `json:"status"`
	Code                    int       `json:"code"`
	Header                  []Header  `json:"header,omitempty"`
	Body                    string    `json:"body,omitempty"`
	PostmanPreviewLanguage  string    `json:"_postman_previewlanguage,omitempty"`
}

// Event represents a pre-request or test script event.
type Event struct {
	Listen string `json:"listen"`
	Script Script `json:"script"`
}

// Script represents a Postman script (pre-request or test).
type Script struct {
	Type string   `json:"type"`
	Exec []string `json:"exec"`
}

// Variable represents a collection-level variable.
type Variable struct {
	Key         string `json:"key"`
	Value       string `json:"value"`
	Type        string `json:"type"`
	Description string `json:"description,omitempty"`
}
