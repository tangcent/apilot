// Package springmvc provides Spring MVC specific API extraction.
package springmvc

import model "github.com/tangcent/apilot/api-model"

// HTTPMethod represents HTTP methods
type HTTPMethod string

const (
	GET    HTTPMethod = "GET"
	POST   HTTPMethod = "POST"
	PUT    HTTPMethod = "PUT"
	DELETE HTTPMethod = "DELETE"
	PATCH  HTTPMethod = "PATCH"
)

// EndpointParameter represents a parameter in an API endpoint
type EndpointParameter struct {
	Name         string
	Type         string
	ParamType    string
	Required     bool
	DefaultValue string
}

// Endpoint represents a REST API endpoint
type Endpoint struct {
	Path              string
	Method            HTTPMethod
	MethodName        string
	Parameters        []EndpointParameter
	ReturnType        string
	ClassName         string
	Package           string
	RequestBodySchema *model.ObjectModel
	ResponseSchema    *model.ObjectModel
}

// Controller represents a Spring MVC controller with its endpoints
type Controller struct {
	Name      string
	Package   string
	BasePath  string
	Endpoints []Endpoint
}
