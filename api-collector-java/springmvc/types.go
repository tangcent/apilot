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
	Name string
	Type string
	// ValueType is the type the parameter holds: the type inside a
	// java.util.Optional, which is how Spring binds one, and Type itself for
	// any other. It is empty for a parameter built from a property of a bound
	// object.
	ValueType    string
	ParamType    string
	Required     bool
	DefaultValue string
	Description  string
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

// Endpoint represents a REST API endpoint
type Endpoint struct {
	Path              string
	Method            HTTPMethod
	MethodName        string
	Description       string
	Parameters        []EndpointParameter
	ReturnType        string
	// Consumes and Produces hold the media types the mapping annotation
	// declares (`@PostMapping(consumes = "multipart/form-data")`), resolved
	// to wire media types. Empty when the annotation declares none.
	Consumes          []string
	Produces          []string
	ClassName         string
	Package           string
	RequestBodySchema *model.ObjectModel
	ResponseSchema    *model.ObjectModel

	// Deprecated and DeprecationNote report the deprecation of the handler
	// method or its class (see doc.DeprecationOf).
	Deprecated      bool
	DeprecationNote string
}

// Controller represents a Spring MVC controller with its endpoints
type Controller struct {
	Name      string
	Package   string
	BasePath  string
	Endpoints []Endpoint
	// Description is the controller's Javadoc.
	Description string
}
