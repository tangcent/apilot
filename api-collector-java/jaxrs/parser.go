// Package jaxrs parses JAX-RS annotated classes for API endpoints.
package jaxrs

import (
	"path"
	"strings"

	collector "github.com/tangcent/apilot/api-collector"
	javadoc "github.com/tangcent/apilot/api-collector-java/doc"
	"github.com/tangcent/apilot/api-collector-java/parser"
	"github.com/tangcent/apilot/api-collector-java/resolver"
)

type Parser struct {
	dependencyResolver resolver.DependencyResolver
	collectorDepResolver collector.DependencyResolver
	unresolved           *collector.UnresolvedSet
}

// NewParser creates a new JAX-RS parser.
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

// ExtractResources extracts JAX-RS resources from parse results.
func (p *Parser) ExtractResources(results []parser.ParseResult) []Resource {
	classRegistry := buildClassRegistry(results)
	typeResolver := resolver.NewTypeResolver(flattenClasses(results))
	if p.dependencyResolver != nil {
		typeResolver.SetDependencyResolver(p.dependencyResolver)
	} else if p.collectorDepResolver != nil {
		typeResolver.SetCollectorDependencyResolver(p.collectorDepResolver)
	}
	typeResolver.SetUnresolved(p.unresolved)

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
		if ep := p.extractParameter(param, method.JavaDocParams); ep != nil {
			params = append(params, *ep)
			if ep.ParamType == "body" {
				requestBodyType = param.Type
			}
		}
	}

	endpoint := &Endpoint{
		Path:        fullPath,
		Method:      httpMethod,
		MethodName:  method.Name,
		Description: javadoc.EndpointDescription(method.Annotations, method.JavaDoc),
		Parameters:  params,
		ReturnType:  method.ReturnType,
		Produces:    p.extractMediaTypes(method.Annotations, "Produces"),
		Consumes:    p.extractMediaTypes(method.Annotations, "Consumes"),
		ClassName:   class.Name,
		Package:     class.Package,
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

// extractParameter resolves the JAX-RS binding of a method parameter and
// layers any documented metadata (JavaDoc, Swagger annotations) on top of it.
func (p *Parser) extractParameter(param parser.Parameter, methodJavaDocParams map[string]string) *EndpointParameter {
	paramType, required := detectParameterType(param.Annotations)
	if paramType == "" {
		return nil
	}

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

	if defaultValue := defaultValueFrom(param.Annotations); defaultValue != "" {
		ep.DefaultValue = defaultValue
		ep.Required = false
	}
	if doc.Default != "" {
		ep.DefaultValue = doc.Default
	}
	if doc.Required != nil {
		ep.Required = *doc.Required
	}

	// The header name on the wire comes from the annotation, not the Java
	// parameter name: @HeaderParam("Authorization") String auth binds the
	// Authorization header.
	if ep.ParamType == "header" {
		if name := parser.ExplicitAnnotationName(param.Annotations, "HeaderParam"); name != "" {
			ep.Name = name
		}
	}

	return ep
}

// detectParameterType maps JAX-RS binding annotations to a canonical location
// and the required flag that binding implies. An empty location means the
// parameter is not part of the API surface (e.g. @Context, @Suspended).
func detectParameterType(annotations []parser.Annotation) (string, bool) {
	for _, ann := range annotations {
		switch ann.Name {
		case "PathParam":
			return "path", true
		case "QueryParam":
			return "query", false
		case "FormParam":
			return "form", false
		case "HeaderParam":
			return "header", false
		case "CookieParam":
			return "cookie", false
		case "BeanParam":
			return "body", true
		}
	}
	// A parameter carrying only @MatrixParam/@Context/@Suspended is injected by
	// the runtime rather than supplied by the caller. Anything without a JAX-RS
	// binding annotation at all is the request body.
	if hasJaxrsParamAnnotation(annotations) {
		return "", false
	}
	return "body", true
}

func hasJaxrsParamAnnotation(annotations []parser.Annotation) bool {
	for _, ann := range annotations {
		switch ann.Name {
		case "MatrixParam", "Context", "Suspended":
			return true
		}
	}
	return false
}

// defaultValueFrom reads the JAX-RS @DefaultValue annotation.
func defaultValueFrom(annotations []parser.Annotation) string {
	for _, ann := range annotations {
		if ann.Name == "DefaultValue" {
			return strings.Trim(ann.Params["value"], "\"'")
		}
	}
	return ""
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
