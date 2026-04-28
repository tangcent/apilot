// Package jaxrs parses JAX-RS annotated classes for API endpoints.
package jaxrs

import (
	"path"
	"strings"

	"github.com/tangcent/apilot/api-collector-java/parser"
	"github.com/tangcent/apilot/api-collector-java/resolver"
)

type Parser struct {
	dependencyResolver resolver.DependencyResolver
}

// NewParser creates a new JAX-RS parser.
func NewParser() *Parser {
	return &Parser{}
}

func (p *Parser) SetDependencyResolver(dr resolver.DependencyResolver) {
	p.dependencyResolver = dr
}

// ExtractResources extracts JAX-RS resources from parse results.
func (p *Parser) ExtractResources(results []parser.ParseResult) []Resource {
	classRegistry := buildClassRegistry(results)
	typeResolver := resolver.NewTypeResolver(flattenClasses(results))
	if p.dependencyResolver != nil {
		typeResolver.SetDependencyResolver(p.dependencyResolver)
	}

	var resources []Resource
	for _, result := range results {
		if result.Error != nil {
			continue
		}
		for _, class := range result.Classes {
			if resource := p.extractResource(class, typeResolver, classRegistry); resource != nil {
				resources = append(resources, *resource)
			}
		}
	}
	return resources
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

func (p *Parser) extractResource(class parser.Class, typeResolver *resolver.TypeResolver, classRegistry map[string]parser.Class) *Resource {
	basePath, hasPath := p.extractPath(class.Annotations)
	if !hasPath {
		return nil
	}

	classProduces := p.extractMediaTypes(class.Annotations, "Produces")
	classConsumes := p.extractMediaTypes(class.Annotations, "Consumes")

	typeBindings := buildTypeBindings(class, classRegistry)

	var endpoints []Endpoint
	for _, method := range class.Methods {
		ep := p.extractEndpoint(method, basePath, class, typeResolver, typeBindings)
		if ep == nil {
			continue
		}
		if len(ep.Produces) == 0 {
			ep.Produces = classProduces
		}
		if len(ep.Consumes) == 0 {
			ep.Consumes = classConsumes
		}
		endpoints = append(endpoints, *ep)
	}

	inheritedEndpoints := p.extractInheritedEndpoints(class, basePath, classProduces, classConsumes, typeResolver, classRegistry, typeBindings)
	endpoints = append(endpoints, inheritedEndpoints...)

	return &Resource{
		Name:      class.Name,
		Package:   class.Package,
		BasePath:  basePath,
		Endpoints: endpoints,
		Produces:  classProduces,
		Consumes:  classConsumes,
	}
}

func (p *Parser) extractInheritedEndpoints(class parser.Class, basePath string, classProduces, classConsumes []string, typeResolver *resolver.TypeResolver, classRegistry map[string]parser.Class, typeBindings map[string]string) []Endpoint {
	if class.SuperClass == "" {
		return nil
	}

	superClass, found := classRegistry[class.SuperClass]
	if !found {
		return nil
	}

	parentTypeBindings := buildTypeBindings(superClass, classRegistry)
	mergedBindings := mergeBindings(parentTypeBindings, typeBindings)

	parentBasePath, _ := p.extractPath(superClass.Annotations)
	effectiveBasePath := basePath
	if effectiveBasePath == "" && parentBasePath != "" {
		effectiveBasePath = parentBasePath
	}

	var inherited []Endpoint

	for _, method := range superClass.Methods {
		if !p.isEndpointMethod(method) {
			continue
		}
		if p.isMethodOverridden(method.Name, class) {
			continue
		}

		ep := p.extractEndpoint(method, effectiveBasePath, class, typeResolver, mergedBindings)
		if ep == nil {
			continue
		}
		if len(ep.Produces) == 0 {
			ep.Produces = classProduces
		}
		if len(ep.Consumes) == 0 {
			ep.Consumes = classConsumes
		}
		inherited = append(inherited, *ep)
	}

	grandparentEndpoints := p.extractInheritedEndpoints(superClass, effectiveBasePath, classProduces, classConsumes, typeResolver, classRegistry, mergedBindings)
	inherited = append(inherited, grandparentEndpoints...)

	return inherited
}

func (p *Parser) isEndpointMethod(method parser.Method) bool {
	for _, ann := range method.Annotations {
		switch ann.Name {
		case "GET", "POST", "PUT", "DELETE", "PATCH", "HEAD", "OPTIONS":
			return true
		}
	}
	return false
}

func (p *Parser) isMethodOverridden(methodName string, class parser.Class) bool {
	for _, m := range class.Methods {
		if m.Name == methodName {
			return true
		}
	}
	return false
}

func (p *Parser) extractEndpoint(method parser.Method, basePath string, class parser.Class, typeResolver *resolver.TypeResolver, typeBindings map[string]string) *Endpoint {
	httpMethod, ok := p.extractHTTPMethod(method.Annotations)
	if !ok {
		return nil
	}

	methodPath := ""
	if mp, hasPath := p.extractPath(method.Annotations); hasPath {
		methodPath = mp
	}

	fullPath := combinePaths(basePath, methodPath)

	var params []EndpointParameter
	var requestBodyType string
	for _, param := range method.Parameters {
		if ep := p.extractParameter(param); ep != nil {
			params = append(params, *ep)
			if ep.ParamType == "body" {
				requestBodyType = param.Type
			}
		}
	}

	endpoint := &Endpoint{
		Path:       fullPath,
		Method:     httpMethod,
		MethodName: method.Name,
		Parameters: params,
		ReturnType: method.ReturnType,
		Produces:   p.extractMediaTypes(method.Annotations, "Produces"),
		Consumes:   p.extractMediaTypes(method.Annotations, "Consumes"),
		ClassName:  class.Name,
		Package:    class.Package,
	}

	if requestBodyType != "" {
		endpoint.RequestBodySchema = typeResolver.Resolve(requestBodyType, typeBindings)
	}

	if method.ReturnType != "" && method.ReturnType != "void" && method.ReturnType != "Void" {
		resolvedType := unwrapJaxrsResponseType(method.ReturnType)
		if resolvedType != "" {
			endpoint.ResponseSchema = typeResolver.Resolve(resolvedType, typeBindings)
		}
	}

	return endpoint
}

func (p *Parser) extractPath(annotations []parser.Annotation) (string, bool) {
	for _, ann := range annotations {
		if ann.Name == "Path" {
			if value, ok := ann.Params["value"]; ok {
				return normalizePath(value), true
			}
			return "", true
		}
	}
	return "", false
}

func (p *Parser) extractHTTPMethod(annotations []parser.Annotation) (HTTPMethod, bool) {
	for _, ann := range annotations {
		switch ann.Name {
		case "GET":
			return GET, true
		case "POST":
			return POST, true
		case "PUT":
			return PUT, true
		case "DELETE":
			return DELETE, true
		case "PATCH":
			return PATCH, true
		case "HEAD":
			return HEAD, true
		case "OPTIONS":
			return OPTIONS, true
		}
	}
	return "", false
}

func (p *Parser) extractParameter(param parser.Parameter) *EndpointParameter {
	for _, ann := range param.Annotations {
		switch ann.Name {
		case "PathParam":
			return &EndpointParameter{Name: param.Name, Type: param.Type, ParamType: "path", Required: true}
		case "QueryParam":
			return &EndpointParameter{Name: param.Name, Type: param.Type, ParamType: "query", Required: false}
		case "FormParam":
			return &EndpointParameter{Name: param.Name, Type: param.Type, ParamType: "form", Required: false}
		case "HeaderParam":
			return &EndpointParameter{Name: param.Name, Type: param.Type, ParamType: "header", Required: false}
		case "CookieParam":
			return &EndpointParameter{Name: param.Name, Type: param.Type, ParamType: "cookie", Required: false}
		case "BeanParam":
			return &EndpointParameter{Name: param.Name, Type: param.Type, ParamType: "body", Required: true}
		}
	}
	if !hasJaxrsParamAnnotation(param) {
		return &EndpointParameter{Name: param.Name, Type: param.Type, ParamType: "body", Required: true}
	}
	return nil
}

func hasJaxrsParamAnnotation(param parser.Parameter) bool {
	for _, ann := range param.Annotations {
		switch ann.Name {
		case "PathParam", "QueryParam", "FormParam", "HeaderParam", "CookieParam",
			"BeanParam", "MatrixParam", "Context", "Suspended":
			return true
		}
	}
	return false
}

func (p *Parser) extractMediaTypes(annotations []parser.Annotation, annName string) []string {
	for _, ann := range annotations {
		if ann.Name == annName {
			if value, ok := ann.Params["value"]; ok {
				return []string{strings.Trim(value, "\"'")}
			}
		}
	}
	return nil
}

func unwrapJaxrsResponseType(rawType string) string {
	if rawType == "Response" {
		return ""
	}
	if strings.HasPrefix(rawType, "Response<") && strings.HasSuffix(rawType, ">") {
		return rawType[len("Response<") : len(rawType)-1]
	}
	return rawType
}

func normalizePath(s string) string {
	s = strings.Trim(s, "\"'")
	if s != "" && !strings.HasPrefix(s, "/") {
		s = "/" + s
	}
	return s
}

func combinePaths(basePath, methodPath string) string {
	if basePath == "" {
		return methodPath
	}
	if methodPath == "" {
		return basePath
	}
	return path.Join(basePath, methodPath)
}
