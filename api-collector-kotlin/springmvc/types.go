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
	// Description is the controller's KDoc, cleaned.
	Description string
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

	// Deprecated and DeprecationNote report the deprecation of the handler
	// function or its class (see doc.DeprecationOf).
	Deprecated      bool
	DeprecationNote string
}

// EndpointParameter is one bound request parameter.
type EndpointParameter struct {
	Name string
	Type string
	// File marks a parameter that is uploaded as file parts: Spring's
	// MultipartFile, or a collection or an array of them, held by a
	// java.util.Optional or not. The parser decides it where the type is
	// written, so that the scope of its names is known.
	File bool

	ParamType    string
	Description  string
	Required     bool
	DefaultValue string
	Example      string
	Enum         []string
}
