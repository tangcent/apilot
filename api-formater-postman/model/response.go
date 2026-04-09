package model

// Response represents an example response saved in a Postman request.
type Response struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Code   int    `json:"code"`
	Body   string `json:"body,omitempty"`
}
