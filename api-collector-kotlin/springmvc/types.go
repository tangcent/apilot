package springmvc

import model "github.com/tangcent/apilot/api-model"

// HTTPMethod is an HTTP verb.
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

// Controller is one Spring MVC controller.
type Controller struct {
	Name      string
	Package   string
	BasePath  string
	Endpoints []Endpoint
}

// Endpoint is one handler method.
type Endpoint struct {
	Path              string
	Method            HTTPMethod
	MethodName        string
	Description       string
	Parameters        []EndpointParameter
	RequestBodySchema *model.ObjectModel
	ResponseSchema    *model.ObjectModel
	ReturnType        string
	Consumes          []string
	Produces          []string
	ClassName         string
	Package           string
}

// EndpointParameter is one bound request parameter.
type EndpointParameter struct {
	Name         string
	Type         string
	ParamType    string
	Description  string
	Required     bool
	DefaultValue string
	Example      string
	Enum         []string
}
