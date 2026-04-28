// Package feign parses Feign client interfaces for API endpoints.
package feign

import (
	"strings"

	"github.com/tangcent/apilot/api-collector-java/parser"
	"github.com/tangcent/apilot/api-collector-java/resolver"
)

// Parser extracts Feign client endpoints from parsed Java classes.
type Parser struct {
	dependencyResolver resolver.DependencyResolver
}

// NewParser creates a new Feign client parser.
func NewParser() *Parser {
	return &Parser{}
}

func (p *Parser) SetDependencyResolver(dr resolver.DependencyResolver) {
	p.dependencyResolver = dr
}

// ExtractClients extracts Feign clients from parse results.
// It supports both Spring Cloud OpenFeign (Spring MVC annotations) and
// Netflix Feign (@RequestLine annotations).
func (p *Parser) ExtractClients(results []parser.ParseResult) []FeignClient {
	classRegistry := buildClassRegistry(results)
	typeResolver := resolver.NewTypeResolver(flattenClasses(results))
	if p.dependencyResolver != nil {
		typeResolver.SetDependencyResolver(p.dependencyResolver)
	}

	var clients []FeignClient
	for _, result := range results {
		if result.Error != nil {
			continue
		}
		for _, class := range result.Classes {
			if client := p.extractClient(class, typeResolver, classRegistry); client != nil {
				clients = append(clients, *client)
			}
		}
	}
	return clients
}

func buildClassRegistry(results []parser.ParseResult) map[string]parser.Class {
	registry := make(map[string]parser.Class)
	for _, result := range results {
		if result.Error != nil {
			continue
		}
		for _, class := range result.Classes {
			registry[class.Name] = class
		}
	}
	return registry
}

func flattenClasses(results []parser.ParseResult) []parser.Class {
	var classes []parser.Class
	for _, result := range results {
		if result.Error != nil {
			continue
		}
		classes = append(classes, result.Classes...)
	}
	return classes
}

func buildTypeBindings(class parser.Class, classRegistry map[string]parser.Class) map[string]string {
	if class.SuperClass == "" {
		return nil
	}

	superClass, found := classRegistry[class.SuperClass]
	if !found {
		return nil
	}

	bindings := make(map[string]string)
	for i, tp := range superClass.TypeParameters {
		if i < len(class.SuperClassTypeArgs) {
			bindings[tp] = class.SuperClassTypeArgs[i]
		}
	}

	return bindings
}

func mergeBindings(parent, child map[string]string) map[string]string {
	merged := make(map[string]string)
	for k, v := range parent {
		merged[k] = v
	}
	for k, v := range child {
		merged[k] = v
	}
	return merged
}

func (p *Parser) extractClient(class parser.Class, typeResolver *resolver.TypeResolver, classRegistry map[string]parser.Class) *FeignClient {
	ann := findAnnotation(class.Annotations, "FeignClient")
	if ann != nil {
		name := ann.Params["name"]
		if name == "" {
			name = ann.Params["value"]
		}

		typeBindings := buildTypeBindings(class, classRegistry)

		var endpoints []Endpoint
		for _, method := range class.Methods {
			if ep := p.extractEndpoint(method, class, typeResolver, typeBindings); ep != nil {
				endpoints = append(endpoints, *ep)
			}
		}

		inheritedEndpoints := p.extractInheritedEndpoints(class, typeResolver, classRegistry, typeBindings)
		endpoints = append(endpoints, inheritedEndpoints...)

		return &FeignClient{
			Name:        class.Name,
			Package:     class.Package,
			ServiceName: name,
			URL:         ann.Params["url"],
			Endpoints:   endpoints,
		}
	}

	if class.IsInterface && p.hasRequestLineMethods(class) {
		typeBindings := buildTypeBindings(class, classRegistry)

		var endpoints []Endpoint
		for _, method := range class.Methods {
			if ep := p.extractEndpoint(method, class, typeResolver, typeBindings); ep != nil {
				endpoints = append(endpoints, *ep)
			}
		}

		inheritedEndpoints := p.extractInheritedEndpoints(class, typeResolver, classRegistry, typeBindings)
		endpoints = append(endpoints, inheritedEndpoints...)

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

func (p *Parser) extractInheritedEndpoints(class parser.Class, typeResolver *resolver.TypeResolver, classRegistry map[string]parser.Class, typeBindings map[string]string) []Endpoint {
	if class.SuperClass == "" {
		return nil
	}

	superClass, found := classRegistry[class.SuperClass]
	if !found {
		return nil
	}

	parentTypeBindings := buildTypeBindings(superClass, classRegistry)
	mergedBindings := mergeBindings(parentTypeBindings, typeBindings)

	var inherited []Endpoint
	for _, method := range superClass.Methods {
		if p.isMethodOverridden(method.Name, class) {
			continue
		}
		if ep := p.extractEndpoint(method, class, typeResolver, mergedBindings); ep != nil {
			inherited = append(inherited, *ep)
		}
	}

	grandparentEndpoints := p.extractInheritedEndpoints(superClass, typeResolver, classRegistry, mergedBindings)
	inherited = append(inherited, grandparentEndpoints...)

	return inherited
}

func (p *Parser) isMethodOverridden(methodName string, class parser.Class) bool {
	for _, m := range class.Methods {
		if m.Name == methodName {
			return true
		}
	}
	return false
}

func (p *Parser) hasRequestLineMethods(class parser.Class) bool {
	for _, method := range class.Methods {
		if findAnnotation(method.Annotations, "RequestLine") != nil {
			return true
		}
	}
	return false
}

func (p *Parser) extractEndpoint(method parser.Method, class parser.Class, typeResolver *resolver.TypeResolver, typeBindings map[string]string) *Endpoint {
	if ep := p.extractSpringStyleEndpoint(method, class, typeResolver, typeBindings); ep != nil {
		return ep
	}
	return p.extractRequestLineEndpoint(method, class, typeResolver, typeBindings)
}

func (p *Parser) extractSpringStyleEndpoint(method parser.Method, class parser.Class, typeResolver *resolver.TypeResolver, typeBindings map[string]string) *Endpoint {
	httpMethod, methodPath := p.extractSpringMappingInfo(method.Annotations)
	if httpMethod == "" {
		return nil
	}

	var params []EndpointParameter
	var requestBodyType string
	for _, param := range method.Parameters {
		if ep := p.extractSpringParameter(param); ep != nil {
			params = append(params, *ep)
			if ep.ParamType == "body" {
				requestBodyType = param.Type
			}
		}
	}

	endpoint := &Endpoint{
		Path:       methodPath,
		Method:     httpMethod,
		MethodName: method.Name,
		Parameters: params,
		ReturnType: method.ReturnType,
		ClassName:  class.Name,
		Package:    class.Package,
	}

	if requestBodyType != "" {
		endpoint.RequestBodySchema = typeResolver.Resolve(requestBodyType, typeBindings)
	}

	if method.ReturnType != "" && method.ReturnType != "void" && method.ReturnType != "Void" {
		resolvedType := unwrapResponseType(method.ReturnType)
		endpoint.ResponseSchema = typeResolver.Resolve(resolvedType, typeBindings)
	}

	return endpoint
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
		case "SpringQueryMap":
			return &EndpointParameter{Name: param.Name, Type: param.Type, ParamType: "body", Required: true}
		}
	}
	return nil
}

func (p *Parser) extractRequestLineEndpoint(method parser.Method, class parser.Class, typeResolver *resolver.TypeResolver, typeBindings map[string]string) *Endpoint {
	ann := findAnnotation(method.Annotations, "RequestLine")
	if ann == nil {
		return nil
	}

	httpMethod, methodPath := parseRequestLine(ann.Params["value"])

	var params []EndpointParameter
	var requestBodyType string
	for _, param := range method.Parameters {
		if ep := p.extractFeignParam(param, methodPath); ep != nil {
			params = append(params, *ep)
			if ep.ParamType == "body" {
				requestBodyType = param.Type
			}
		}
	}

	endpoint := &Endpoint{
		Path:       methodPath,
		Method:     httpMethod,
		MethodName: method.Name,
		Parameters: params,
		ReturnType: method.ReturnType,
		ClassName:  class.Name,
		Package:    class.Package,
	}

	if requestBodyType != "" {
		endpoint.RequestBodySchema = typeResolver.Resolve(requestBodyType, typeBindings)
	}

	if method.ReturnType != "" && method.ReturnType != "void" && method.ReturnType != "Void" {
		endpoint.ResponseSchema = typeResolver.Resolve(method.ReturnType, typeBindings)
	}

	return endpoint
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

func unwrapResponseType(rawType string) string {
	if strings.HasPrefix(rawType, "ResponseEntity<") && strings.HasSuffix(rawType, ">") {
		return rawType[len("ResponseEntity<") : len(rawType)-1]
	}
	return rawType
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
