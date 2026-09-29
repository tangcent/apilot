// Package feign parses Feign client interfaces for API endpoints.
package feign

import (
	"strings"

	collector "github.com/tangcent/apilot/api-collector"
	javadoc "github.com/tangcent/apilot/api-collector-java/doc"
	"github.com/tangcent/apilot/api-collector-java/parser"
	"github.com/tangcent/apilot/api-collector-java/resolver"
)

// Parser extracts Feign client endpoints from parsed Java classes.
type Parser struct {
	dependencyResolver resolver.DependencyResolver
	collectorDepResolver collector.DependencyResolver
	unresolved           *collector.UnresolvedSet
}

// NewParser creates a new Feign client parser.
func NewParser() *Parser {
	return &Parser{}
}

func (p *Parser) SetDependencyResolver(dr resolver.DependencyResolver) {
	p.dependencyResolver = dr
}

func (p *Parser) SetCollectorDependencyResolver(cdr collector.DependencyResolver) {
	p.collectorDepResolver = cdr
}

// SetUnresolved attaches a shared sink collecting types the resolver could not
// expand. Optional; type resolution behaviour is unchanged without it.
func (p *Parser) SetUnresolved(u *collector.UnresolvedSet) {
	p.unresolved = u
}

// ExtractClients extracts Feign clients from parse results.
// It supports both Spring Cloud OpenFeign (Spring MVC annotations) and
// Netflix Feign (@RequestLine annotations).
func (p *Parser) ExtractClients(results []parser.ParseResult) []FeignClient {
	classRegistry := buildClassRegistry(results)
	typeResolver := resolver.NewTypeResolver(flattenClasses(results))
	if p.dependencyResolver != nil {
		typeResolver.SetDependencyResolver(p.dependencyResolver)
	} else if p.collectorDepResolver != nil {
		typeResolver.SetCollectorDependencyResolver(p.collectorDepResolver)
	}
	typeResolver.SetUnresolved(p.unresolved)

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
		if ep := p.extractSpringParameter(param, method.JavaDocParams); ep != nil {
			params = append(params, *ep)
			if ep.ParamType == "body" {
				requestBodyType = param.Type
			}
		}
	}

	endpoint := &Endpoint{
		Path:        methodPath,
		Method:      httpMethod,
		MethodName:  method.Name,
		Description: javadoc.EndpointDescription(method.Annotations, method.JavaDoc),
		Parameters:  params,
		ReturnType:  method.ReturnType,
		ClassName:   class.Name,
		Package:     class.Package,
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

// extractSpringParameter resolves the Spring MVC binding of a Feign method
// parameter and layers any documented metadata on top of it.
func (p *Parser) extractSpringParameter(param parser.Parameter, methodJavaDocParams map[string]string) *EndpointParameter {
	paramType, required := detectSpringParameterType(param.Annotations)
	if paramType == "" {
		return nil
	}
	return documentedParameter(param, methodJavaDocParams, paramType, required, "RequestParam")
}

// detectSpringParameterType maps Spring MVC binding annotations to a canonical
// location and the required flag the annotations imply. An empty location means
// the parameter has no Spring binding annotation and is not part of the API.
func detectSpringParameterType(annotations []parser.Annotation) (string, bool) {
	for _, ann := range annotations {
		switch ann.Name {
		case "PathVariable":
			return "path", true
		case "RequestParam":
			required := true
			if r, ok := ann.Params["required"]; ok {
				required = r != "false"
			}
			return "query", required
		case "RequestBody":
			return "body", true
		case "RequestHeader":
			return "header", true
		case "SpringQueryMap":
			return "body", true
		}
	}
	return "", false
}

// documentedParameter applies JavaDoc and Swagger metadata to a parameter
// already resolved to a location. defaultValueAnn names the annotation carrying
// a native default value, which also relaxes the required flag.
func documentedParameter(param parser.Parameter, methodJavaDocParams map[string]string, paramType string, required bool, defaultValueAnn string) *EndpointParameter {
	paramJavaDoc := param.JavaDoc
	if paramJavaDoc == "" && methodJavaDocParams != nil {
		paramJavaDoc = methodJavaDocParams[param.Name]
	}
	doc := javadoc.ParameterDocumentation(param.Annotations, paramJavaDoc)

	ep := &EndpointParameter{
		Name:        param.Name,
		Type:        param.Type,
		ParamType:   paramType,
		Required:    required,
		Description: doc.Description,
		Example:     doc.Example,
		Enum:        doc.Enum,
	}

	if defaultValueAnn != "" {
		if value := annotationValue(param.Annotations, defaultValueAnn, "defaultValue"); value != "" {
			ep.DefaultValue = value
			ep.Required = false
		}
	}
	if doc.Default != "" {
		ep.DefaultValue = doc.Default
	}
	if doc.Required != nil {
		ep.Required = *doc.Required
	}

	return ep
}

func annotationValue(annotations []parser.Annotation, annName, key string) string {
	for _, ann := range annotations {
		if ann.Name == annName {
			return strings.Trim(ann.Params[key], "\"'")
		}
	}
	return ""
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
		if ep := p.extractFeignParam(param, methodPath, method.JavaDocParams); ep != nil {
			params = append(params, *ep)
			if ep.ParamType == "body" {
				requestBodyType = param.Type
			}
		}
	}

	endpoint := &Endpoint{
		Path:        methodPath,
		Method:      httpMethod,
		MethodName:  method.Name,
		Description: javadoc.EndpointDescription(method.Annotations, method.JavaDoc),
		Parameters:  params,
		ReturnType:  method.ReturnType,
		ClassName:   class.Name,
		Package:     class.Package,
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

func (p *Parser) extractFeignParam(param parser.Parameter, methodPath string, methodJavaDocParams map[string]string) *EndpointParameter {
	ann := findAnnotation(param.Annotations, "Param")
	if ann == nil {
		if param.Type != "" && !isJavaPrimitive(param.Type) {
			return documentedParameter(param, methodJavaDocParams, "body", true, "")
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
	// Documentation sources address the Java identifier, so resolve them with
	// param.Name and only then expose the @Param alias.
	ep := documentedParameter(param, methodJavaDocParams, paramType, paramType == "path", "")
	ep.Name = name
	return ep
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
