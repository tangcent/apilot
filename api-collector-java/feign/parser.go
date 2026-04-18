// Package feign parses Feign client interfaces for API endpoints.
package feign

import (
	"strings"

	"github.com/tangcent/apilot/api-collector-java/parser"
)

// Parser extracts Feign client endpoints from parsed Java classes.
type Parser struct{}

// NewParser creates a new Feign client parser.
func NewParser() *Parser {
	return &Parser{}
}

// ExtractClients extracts Feign clients from parse results.
// It supports both Spring Cloud OpenFeign (Spring MVC annotations) and
// Netflix Feign (@RequestLine annotations).
func (p *Parser) ExtractClients(results []parser.ParseResult) []FeignClient {
	var clients []FeignClient
	for _, result := range results {
		if result.Error != nil {
			continue
		}
		for _, class := range result.Classes {
			if client := p.extractClient(class); client != nil {
				clients = append(clients, *client)
			}
		}
	}
	return clients
}

func (p *Parser) extractClient(class parser.Class) *FeignClient {
	ann := findAnnotation(class.Annotations, "FeignClient")
	if ann != nil {
		name := ann.Params["name"]
		if name == "" {
			name = ann.Params["value"]
		}

		var endpoints []Endpoint
		for _, method := range class.Methods {
			if ep := p.extractEndpoint(method, class); ep != nil {
				endpoints = append(endpoints, *ep)
			}
		}

		return &FeignClient{
			Name:        class.Name,
			Package:     class.Package,
			ServiceName: name,
			URL:         ann.Params["url"],
			Endpoints:   endpoints,
		}
	}

	if class.IsInterface && p.hasRequestLineMethods(class) {
		var endpoints []Endpoint
		for _, method := range class.Methods {
			if ep := p.extractEndpoint(method, class); ep != nil {
				endpoints = append(endpoints, *ep)
			}
		}

		if len(endpoints) > 0 {
			return &FeignClient{
				Name:      class.Name,
				Package:   class.Package,
				Endpoints: endpoints,
			}
		}
	}

	return nil
}

func (p *Parser) hasRequestLineMethods(class parser.Class) bool {
	for _, method := range class.Methods {
		if findAnnotation(method.Annotations, "RequestLine") != nil {
			return true
		}
	}
	return false
}

func (p *Parser) extractEndpoint(method parser.Method, class parser.Class) *Endpoint {
	// Try Spring Cloud OpenFeign style first (Spring MVC annotations)
	if ep := p.extractSpringStyleEndpoint(method, class); ep != nil {
		return ep
	}
	// Fall back to Netflix Feign style (@RequestLine)
	return p.extractRequestLineEndpoint(method, class)
}

func (p *Parser) extractSpringStyleEndpoint(method parser.Method, class parser.Class) *Endpoint {
	httpMethod, methodPath := p.extractSpringMappingInfo(method.Annotations)
	if httpMethod == "" {
		return nil
	}

	var params []EndpointParameter
	for _, param := range method.Parameters {
		if ep := p.extractSpringParameter(param); ep != nil {
			params = append(params, *ep)
		}
	}

	return &Endpoint{
		Path:       methodPath,
		Method:     httpMethod,
		MethodName: method.Name,
		Parameters: params,
		ReturnType: method.ReturnType,
		ClassName:  class.Name,
		Package:    class.Package,
	}
}

func (p *Parser) extractSpringMappingInfo(annotations []parser.Annotation) (HTTPMethod, string) {
	for _, ann := range annotations {
		switch ann.Name {
		case "GetMapping":
			return GET, extractSpringPath(ann)
		case "PostMapping":
			return POST, extractSpringPath(ann)
		case "PutMapping":
			return PUT, extractSpringPath(ann)
		case "DeleteMapping":
			return DELETE, extractSpringPath(ann)
		case "PatchMapping":
			return PATCH, extractSpringPath(ann)
		case "RequestMapping":
			return extractSpringRequestMappingMethod(ann), extractSpringPath(ann)
		}
	}
	return "", ""
}

func extractSpringPath(ann parser.Annotation) string {
	if v, ok := ann.Params["value"]; ok {
		return normalizePath(v)
	}
	if v, ok := ann.Params["path"]; ok {
		return normalizePath(v)
	}
	return ""
}

func extractSpringRequestMappingMethod(ann parser.Annotation) HTTPMethod {
	if method, ok := ann.Params["method"]; ok {
		method = strings.TrimPrefix(method, "RequestMethod.")
		switch method {
		case "GET":
			return GET
		case "POST":
			return POST
		case "PUT":
			return PUT
		case "DELETE":
			return DELETE
		case "PATCH":
			return PATCH
		}
	}
	return GET
}

func (p *Parser) extractSpringParameter(param parser.Parameter) *EndpointParameter {
	for _, ann := range param.Annotations {
		switch ann.Name {
		case "PathVariable":
			return &EndpointParameter{Name: param.Name, Type: param.Type, ParamType: "path", Required: true}
		case "RequestParam":
			required := true
			if r, ok := ann.Params["required"]; ok {
				required = r != "false"
			}
			return &EndpointParameter{Name: param.Name, Type: param.Type, ParamType: "query", Required: required}
		case "RequestBody":
			return &EndpointParameter{Name: param.Name, Type: param.Type, ParamType: "body", Required: true}
		case "RequestHeader":
			return &EndpointParameter{Name: param.Name, Type: param.Type, ParamType: "header", Required: true}
		}
	}
	return nil
}

func (p *Parser) extractRequestLineEndpoint(method parser.Method, class parser.Class) *Endpoint {
	ann := findAnnotation(method.Annotations, "RequestLine")
	if ann == nil {
		return nil
	}

	httpMethod, methodPath := parseRequestLine(ann.Params["value"])

	var params []EndpointParameter
	for _, param := range method.Parameters {
		if ep := p.extractFeignParam(param, methodPath); ep != nil {
			params = append(params, *ep)
		}
	}

	return &Endpoint{
		Path:       methodPath,
		Method:     httpMethod,
		MethodName: method.Name,
		Parameters: params,
		ReturnType: method.ReturnType,
		ClassName:  class.Name,
		Package:    class.Package,
	}
}

// parseRequestLine parses a value like "GET /users/{id}" into method and path.
// Returns GET with empty path for malformed input.
func parseRequestLine(value string) (HTTPMethod, string) {
	value = strings.TrimSpace(value)
	parts := strings.SplitN(value, " ", 2)
	if len(parts) != 2 {
		return GET, ""
	}
	methodPath := normalizePath(parts[1])
	switch parts[0] {
	case "GET":
		return GET, methodPath
	case "POST":
		return POST, methodPath
	case "PUT":
		return PUT, methodPath
	case "DELETE":
		return DELETE, methodPath
	case "PATCH":
		return PATCH, methodPath
	default:
		return GET, methodPath
	}
}

func (p *Parser) extractFeignParam(param parser.Parameter, methodPath string) *EndpointParameter {
	ann := findAnnotation(param.Annotations, "Param")
	if ann == nil {
		if param.Type != "" && !isJavaPrimitive(param.Type) {
			return &EndpointParameter{Name: param.Name, Type: param.Type, ParamType: "body", Required: true}
		}
		return nil
	}
	name := ann.Params["value"]
	if name == "" {
		name = param.Name
	}
	paramType := "query"
	pathPart := methodPath
	if idx := strings.Index(methodPath, "?"); idx >= 0 {
		pathPart = methodPath[:idx]
	}
	if strings.Contains(pathPart, "{"+name+"}") {
		paramType = "path"
	}
	return &EndpointParameter{Name: name, Type: param.Type, ParamType: paramType, Required: paramType == "path"}
}

func isJavaPrimitive(typ string) bool {
	switch typ {
	case "byte", "short", "int", "long", "float", "double", "boolean", "char",
		"Byte", "Short", "Integer", "Long", "Float", "Double", "Boolean", "Character",
		"String", "void", "Void":
		return true
	}
	return false
}

func findAnnotation(annotations []parser.Annotation, name string) *parser.Annotation {
	for i := range annotations {
		if annotations[i].Name == name {
			return &annotations[i]
		}
	}
	return nil
}

func normalizePath(s string) string {
	s = strings.Trim(s, "\"'")
	if s != "" && !strings.HasPrefix(s, "/") {
		s = "/" + s
	}
	return s
}
