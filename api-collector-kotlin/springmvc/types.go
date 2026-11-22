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
	// ValueType is the type the parameter holds: the type inside a
	// java.util.Optional, which is how Spring binds one, and Type itself for
	// any other. It is empty for a parameter built from a property of a bound
	// object.
	ValueType    string
	ParamType    string
	Description  string
	Required     bool
	DefaultValue string
	Example      string
	Enum         []string
}

// HeldType returns the type the parameter holds (see ValueType).
func (p EndpointParameter) HeldType() string {
	if p.ValueType != "" {
		return p.ValueType
	}
	return p.Type
}
