package springmvc

import (
	"path"
	"strings"

	"github.com/tangcent/apilot/api-collector-java/parser"
	"github.com/tangcent/apilot/api-collector-java/resolver"
)

type Parser struct {
	dependencyResolver resolver.DependencyResolver
}

func NewParser() *Parser {
	return &Parser{}
}

func (p *Parser) SetDependencyResolver(dr resolver.DependencyResolver) {
	p.dependencyResolver = dr
}

func (p *Parser) ExtractControllers(results []parser.ParseResult) []Controller {
	classRegistry := buildClassRegistry(results)
	typeResolver := resolver.NewTypeResolver(flattenClasses(results))
	if p.dependencyResolver != nil {
		typeResolver.SetDependencyResolver(p.dependencyResolver)
	}

	var controllers []Controller

	for _, result := range results {
		if result.Error != nil {
			continue
		}

		for _, class := range result.Classes {
			if controller := p.extractController(class, typeResolver, classRegistry); controller != nil {
				controllers = append(controllers, *controller)
			}
		}
	}

	return controllers
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

func (p *Parser) extractController(class parser.Class, typeResolver *resolver.TypeResolver, classRegistry map[string]parser.Class) *Controller {
	if !p.isController(class) {
		return nil
	}

	basePath := p.extractBasePath(class.Annotations)

	typeBindings := buildTypeBindings(class, classRegistry)

	var endpoints []Endpoint
	for _, method := range class.Methods {
		if endpoint := p.extractEndpoint(method, basePath, class, typeResolver, typeBindings); endpoint != nil {
			endpoints = append(endpoints, *endpoint)
		}
	}

	inheritedEndpoints := p.extractInheritedEndpoints(class, basePath, typeResolver, classRegistry, typeBindings)
	endpoints = append(endpoints, inheritedEndpoints...)

	return &Controller{
		Name:      class.Name,
		Package:   class.Package,
		BasePath:  basePath,
		Endpoints: endpoints,
	}
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

func (p *Parser) extractInheritedEndpoints(class parser.Class, basePath string, typeResolver *resolver.TypeResolver, classRegistry map[string]parser.Class, typeBindings map[string]string) []Endpoint {
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

		if endpoint := p.extractEndpoint(method, effectiveBasePath, class, typeResolver, mergedBindings); endpoint != nil {
			inherited = append(inherited, *endpoint)
		}
	}

	grandparentEndpoints := p.extractInheritedEndpoints(superClass, effectiveBasePath, typeResolver, classRegistry, mergedBindings)
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

func (p *Parser) extractEndpoint(method parser.Method, basePath string, class parser.Class, typeResolver *resolver.TypeResolver, typeBindings map[string]string) *Endpoint {
	httpMethod, methodPath := p.extractMethodInfo(method.Annotations)
	if httpMethod == "" {
		return nil
	}

	fullPath := p.combinePaths(basePath, methodPath)

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
		ClassName:  class.Name,
		Package:    class.Package,
	}

	if requestBodyType != "" {
		endpoint.RequestBodySchema = typeResolver.Resolve(requestBodyType, typeBindings)
	}

	if method.ReturnType != "" && method.ReturnType != "void" && method.ReturnType != "Void" {
		resolvedType := unwrapSpringResponseType(method.ReturnType)
		endpoint.ResponseSchema = typeResolver.Resolve(resolvedType, typeBindings)
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
			method := p.extractHTTPMethodFromRequestMapping(ann)
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

func (p *Parser) extractHTTPMethodFromRequestMapping(ann parser.Annotation) HTTPMethod {
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

func (p *Parser) extractParameter(param parser.Parameter) *EndpointParameter {
	paramType := p.detectParameterType(param.Annotations)
	if paramType == "" {
		return nil
	}

	ep := &EndpointParameter{
		Name:      param.Name,
		Type:      param.Type,
		ParamType: paramType,
		Required:  true,
	}

	for _, ann := range param.Annotations {
		switch ann.Name {
		case "RequestParam":
			if required, ok := ann.Params["required"]; ok {
				ep.Required = required != "false"
			}
			if defaultValue, ok := ann.Params["defaultValue"]; ok {
				ep.DefaultValue = defaultValue
				ep.Required = false
			}
		}
	}

	return ep
}

func (p *Parser) detectParameterType(annotations []parser.Annotation) string {
	for _, ann := range annotations {
		switch ann.Name {
		case "PathVariable":
			return "path"
		case "RequestParam":
			return "query"
		case "RequestBody":
			return "body"
		case "RequestHeader":
			return "header"
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

func unwrapSpringResponseType(rawType string) string {
	if strings.HasPrefix(rawType, "ResponseEntity<") && strings.HasSuffix(rawType, ">") {
		return rawType[len("ResponseEntity<") : len(rawType)-1]
	}
	return rawType
}
