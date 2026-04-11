// Package springmvc provides Spring MVC specific API extraction.
package springmvc

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
	Name        string
	Type        string
	ParamType   string // path, query, body, header
	Required    bool
	DefaultValue string
}

// Endpoint represents a REST API endpoint
type Endpoint struct {
	Path       string
	Method     HTTPMethod
	MethodName string
	Parameters []EndpointParameter
	ReturnType string
	ClassName  string
	Package    string
}

// Controller represents a Spring MVC controller with its endpoints
type Controller struct {
	Name      string
	Package   string
	BasePath  string
	Endpoints []Endpoint
}
