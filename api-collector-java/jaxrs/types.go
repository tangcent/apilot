// Package jaxrs provides JAX-RS specific API extraction.
package jaxrs

import model "github.com/tangcent/apilot/api-model"

// HTTPMethod represents HTTP methods.
type HTTPMethod string

const (
	GET     HTTPMethod = "GET"
	POST    HTTPMethod = "POST"
	PUT     HTTPMethod = "PUT"
	DELETE  HTTPMethod = "DELETE"
	PATCH   HTTPMethod = "PATCH"
	HEAD    HTTPMethod = "HEAD"
	OPTIONS HTTPMethod = "OPTIONS"
)

// EndpointParameter describes a single parameter of a JAX-RS endpoint.
type EndpointParameter struct {
	Name      string
	Type      string
	ParamType string // path, query, form, header, cookie, body
	Required  bool
}

// Endpoint represents a JAX-RS REST endpoint.
type Endpoint struct {
	Path              string
	Method            HTTPMethod
	MethodName        string
	Parameters        []EndpointParameter
	ReturnType        string
	Produces          []string
	Consumes          []string
	ClassName         string
	Package           string
	RequestBodySchema *model.ObjectModel
	ResponseSchema    *model.ObjectModel
}

// Resource represents a JAX-RS resource class with its endpoints.
type Resource struct {
	Name      string
	Package   string
	BasePath  string
	Endpoints []Endpoint
	Produces  []string
	Consumes  []string
}
