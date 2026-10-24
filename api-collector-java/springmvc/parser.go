package springmvc

import (
	"path"
	"strings"

	collector "github.com/tangcent/apilot/api-collector"
	javadoc "github.com/tangcent/apilot/api-collector-java/doc"
	"github.com/tangcent/apilot/api-collector-java/parser"
	"github.com/tangcent/apilot/api-collector-java/resolver"
)

type Parser struct {
	dependencyResolver   resolver.DependencyResolver
	collectorDepResolver collector.DependencyResolver
	unresolved           *collector.UnresolvedSet
}

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

func (p *Parser) ExtractControllers(results []parser.ParseResult) []Controller {
	typeResolver := resolver.NewTypeResolver(flattenClasses(results))
	if p.dependencyResolver != nil {
		typeResolver.SetDependencyResolver(p.dependencyResolver)
	} else if p.collectorDepResolver != nil {
		typeResolver.SetCollectorDependencyResolver(p.collectorDepResolver)
	}
	typeResolver.SetUnresolved(p.unresolved)

	var controllers []Controller

	for _, result := range results {
		if result.Error != nil {
			continue
		}

		for _, class := range result.Classes {
			if controller := p.extractController(class, typeResolver); controller != nil {
				controllers = append(controllers, *controller)
			}
		}
	}

	return controllers
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

func (p *Parser) extractController(class parser.Class, typeResolver *resolver.TypeResolver) *Controller {
	if !p.isController(class) {
		return nil
	}

	basePath := p.extractBasePath(class.Annotations)

	typeBindings := buildTypeBindings(class, typeResolver)

	var endpoints []Endpoint
	for _, method := range class.Methods {
		if endpoint := p.extractEndpoint(method, basePath, class, class, typeResolver, typeBindings); endpoint != nil {
			endpoints = append(endpoints, *endpoint)
		}
	}

	inheritedEndpoints := p.extractInheritedEndpoints(class, basePath, typeResolver, typeBindings)
	endpoints = append(endpoints, inheritedEndpoints...)

	return &Controller{
		Name:      class.Name,
		Package:   class.Package,
		BasePath:  basePath,
		Endpoints: endpoints,
	}
}

func buildTypeBindings(class parser.Class, typeResolver *resolver.TypeResolver) map[string]string {
	if class.SuperClass == "" {
		return nil
	}

	superClass, found := typeResolver.Class(class, class.SuperClass)
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

func (p *Parser) extractInheritedEndpoints(class parser.Class, basePath string, typeResolver *resolver.TypeResolver, typeBindings map[string]string) []Endpoint {
	if class.SuperClass == "" {
		return nil
	}

	superClass, found := typeResolver.Class(class, class.SuperClass)
	if !found {
		return nil
	}

	parentTypeBindings := buildTypeBindings(superClass, typeResolver)
	mergedBindings := mergeBindings(parentTypeBindings, typeBindings)

	var inherited []Endpoint

	parentBasePath := p.extractBasePath(superClass.Annotations)
	effectiveBasePath := basePath
	if effectiveBasePath == "" && parentBasePath != "" {
		effectiveBasePath = parentBasePath
	}

	for _, method := range superClass.Methods {
		if !p.isEndpointMethod(method) {
			continue
		}

		if p.isMethodOverridden(method.Name, class) {
			continue
		}

		// The method is declared in the superclass's compilation unit, so its
		// parameter and return types resolve in that unit's scope.
		if endpoint := p.extractEndpoint(method, effectiveBasePath, class, superClass, typeResolver, mergedBindings); endpoint != nil {
			inherited = append(inherited, *endpoint)
		}
	}

	grandparentEndpoints := p.extractInheritedEndpoints(superClass, effectiveBasePath, typeResolver, mergedBindings)
	inherited = append(inherited, grandparentEndpoints...)

	return inherited
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

func (p *Parser) isEndpointMethod(method parser.Method) bool {
	for _, ann := range method.Annotations {
		switch ann.Name {
		case "GetMapping", "PostMapping", "PutMapping", "DeleteMapping", "PatchMapping", "RequestMapping":
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

func (p *Parser) isController(class parser.Class) bool {
	for _, ann := range class.Annotations {
		if ann.Name == "RestController" || ann.Name == "Controller" {
			return true
		}
	}
	return false
}

func (p *Parser) extractBasePath(annotations []parser.Annotation) string {
	for _, ann := range annotations {
		if ann.Name == "RequestMapping" {
			if value, ok := ann.Params["value"]; ok {
				return p.normalizePath(value)
			}
			if path, ok := ann.Params["path"]; ok {
				return p.normalizePath(path)
			}
		}
	}
	return ""
}

// extractEndpoint builds one endpoint. class names the owner of the endpoint;
// source is the class whose declaration carries the method, which is the
// superclass for inherited endpoints and supplies the scope its types resolve
// in.
func (p *Parser) extractEndpoint(method parser.Method, basePath string, class, source parser.Class, typeResolver *resolver.TypeResolver, typeBindings map[string]string) *Endpoint {
	httpMethod, methodPath := p.extractMethodInfo(method.Annotations)
	if httpMethod == "" {
		return nil
	}

	fullPath := p.combinePaths(basePath, methodPath)

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
		ClassName:   class.Name,
		Package:     class.Package,
	}

	if requestBodyType != "" {
		endpoint.RequestBodySchema = typeResolver.ResolveFrom(source, requestBodyType, typeBindings)
	}

	if method.ReturnType != "" && method.ReturnType != "void" && method.ReturnType != "Void" {
		// A ResponseEntity<T> wrapper is unwrapped by the type resolver, the
		// same way it handles a reactor publisher.
		endpoint.ResponseSchema = typeResolver.ResolveFrom(source, method.ReturnType, typeBindings)
	}

	return endpoint
}

func (p *Parser) extractMethodInfo(annotations []parser.Annotation) (HTTPMethod, string) {
	for _, ann := range annotations {
		switch ann.Name {
		case "GetMapping":
			return GET, p.extractPathFromMapping(ann)
		case "PostMapping":
			return POST, p.extractPathFromMapping(ann)
		case "PutMapping":
			return PUT, p.extractPathFromMapping(ann)
		case "DeleteMapping":
			return DELETE, p.extractPathFromMapping(ann)
		case "PatchMapping":
			return PATCH, p.extractPathFromMapping(ann)
		case "RequestMapping":
			method := p.extractHTTPMethodFromRequestMapping(ann, annotations)
			path := p.extractPathFromMapping(ann)
			return method, path
		}
	}
	return "", ""
}

func (p *Parser) extractPathFromMapping(ann parser.Annotation) string {
	if value, ok := ann.Params["value"]; ok {
		return p.normalizePath(value)
	}
	if path, ok := ann.Params["path"]; ok {
		return p.normalizePath(path)
	}
	return ""
}

func (p *Parser) extractHTTPMethodFromRequestMapping(ann parser.Annotation, allAnnotations []parser.Annotation) HTTPMethod {
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
	// Fallback: check @ApiOperation(httpMethod = "POST") from Swagger 2.
	for _, a := range allAnnotations {
		if a.Name == "ApiOperation" {
			if httpMethod, ok := a.Params["httpMethod"]; ok {
				switch strings.ToUpper(strings.TrimSpace(httpMethod)) {
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
		}
	}
	return GET
}

func (p *Parser) extractParameter(param parser.Parameter, methodJavaDocParams map[string]string) *EndpointParameter {
	paramType := p.detectParameterType(param.Annotations)
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
		Required:    true,
		Description: doc.Description,
		Example:     doc.Example,
		Enum:        doc.Enum,
	}

	for _, ann := range param.Annotations {
		switch ann.Name {
		case "RequestParam", "RequestHeader":
			if required, ok := ann.Params["required"]; ok {
				ep.Required = required != "false"
			}
			if defaultValue, ok := ann.Params["defaultValue"]; ok {
				ep.DefaultValue = defaultValue
				ep.Required = false
			}
		}
	}
	if doc.Default != "" {
		ep.DefaultValue = doc.Default
	}
	if doc.Required != nil {
		ep.Required = *doc.Required
	}

	// The header name on the wire comes from the annotation, not the Java
	// parameter name: @RequestHeader("Authorization") String auth binds the
	// Authorization header.
	if ep.ParamType == "header" {
		if name := parser.ExplicitAnnotationName(param.Annotations, "RequestHeader"); name != "" {
			ep.Name = name
		}
	}

	return ep
}

func (p *Parser) detectParameterType(annotations []parser.Annotation) string {
	// Check definitive parameter type annotations first (highest priority).
	for _, ann := range annotations {
		switch ann.Name {
		case "PathVariable":
			return "path"
		case "RequestBody":
			return "body"
		case "RequestHeader":
			return "header"
		case "RequestParam":
			return "query"
		}
	}
	// @ApiParam as fallback: treat as query parameter when no explicit
	// Spring binding annotation is present.
	for _, ann := range annotations {
		if ann.Name == "ApiParam" {
			return "query"
		}
	}
	return ""
}

func (p *Parser) normalizePath(pathStr string) string {
	pathStr = strings.Trim(pathStr, "\"")
	pathStr = strings.Trim(pathStr, "'")

	if pathStr != "" && !strings.HasPrefix(pathStr, "/") {
		pathStr = "/" + pathStr
	}

	return pathStr
}

func (p *Parser) combinePaths(basePath, methodPath string) string {
	if basePath == "" {
		return methodPath
	}
	if methodPath == "" {
		return basePath
	}
	return path.Join(basePath, methodPath)
}
