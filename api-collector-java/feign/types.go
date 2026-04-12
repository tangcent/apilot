// Package feign provides Feign client specific API extraction.
package feign

// HTTPMethod represents HTTP methods.
type HTTPMethod string

const (
	GET    HTTPMethod = "GET"
	POST   HTTPMethod = "POST"
	PUT    HTTPMethod = "PUT"
	DELETE HTTPMethod = "DELETE"
	PATCH  HTTPMethod = "PATCH"
)

// EndpointParameter describes a single parameter of a Feign endpoint.
type EndpointParameter struct {
	Name      string
	Type      string
	ParamType string // path, query, body, header, form
	Required  bool
}

// Endpoint represents a Feign client endpoint.
type Endpoint struct {
	Path       string
	Method     HTTPMethod
	MethodName string
	Parameters []EndpointParameter
	ReturnType string
	ClassName  string
	Package    string
}

// FeignClient represents a Feign client interface with its endpoints.
type FeignClient struct {
	Name        string
	Package     string
	ServiceName string // from @FeignClient(name = "...")
	URL         string // from @FeignClient(url = "...")
	Endpoints   []Endpoint
}
