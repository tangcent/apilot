// Package springmvc extracts Spring MVC endpoints from Kotlin declarations.
package springmvc

import (
	"path"
	"strings"

	javadoc "github.com/tangcent/apilot/api-collector-java/doc"
	javaparser "github.com/tangcent/apilot/api-collector-java/parser"
	ktparser "github.com/tangcent/apilot/api-collector-kotlin/parser"
	"github.com/tangcent/apilot/api-collector-kotlin/resolver"
)

// Parser extracts controllers from Kotlin declarations.
type Parser struct {
	typeResolver *resolver.Resolver
	warnings     []string
}

// NewParser creates a Spring MVC parser backed by the Kotlin type resolver.
func NewParser(typeResolver *resolver.Resolver) *Parser {
	return &Parser{typeResolver: typeResolver}
}

// Warnings returns non-fatal findings accumulated while extracting.
func (p *Parser) Warnings() []string {
	return p.warnings
}

// ExtractControllers returns every controller in the parsed files.
func (p *Parser) ExtractControllers(results []ktparser.ParseResult) []Controller {
	var controllers []Controller
	seen := map[string]bool{}
	for _, result := range results {
		if result.Error != nil {
			continue
		}
		for i := range result.Classes {
			class := &result.Classes[i]
			key := class.Package + "." + class.Name
			if seen[key] {
				continue
			}
			seen[key] = true
			if controller := p.extractController(class); controller != nil {
				controllers = append(controllers, *controller)
			}
		}
	}
	return controllers
}

func (p *Parser) extractController(class *ktparser.Class) *Controller {
	if !isController(class) {
		return nil
	}
	basePath := extractBasePath(class.Annotations)
	bindings := p.buildTypeBindings(class)

	var endpoints []Endpoint
	for i := range class.Methods {
		method := &class.Methods[i]
		if !isEndpointMethod(method) {
			continue
		}
		if endpoint := p.extractEndpoint(method, basePath, class, class, bindings); endpoint != nil {
			endpoints = append(endpoints, *endpoint)
		}
	}
	endpoints = append(endpoints, p.extractInheritedEndpoints(class, basePath, bindings, map[string]bool{})...)

	return &Controller{
		Name:      class.Name,
		Package:   class.Package,
		BasePath:  basePath,
		Endpoints: endpoints,
	}
}

func (p *Parser) buildTypeBindings(class *ktparser.Class) map[string]string {
	if class.SuperClass == "" {
		return nil
	}
	superClass, found := p.typeResolver.FindClass(class, class.SuperClass)
	if !found {
		return nil
	}
	bindings := map[string]string{}
	for i, typeParam := range superClass.TypeParameters {
		if i < len(class.SuperClassTypeArgs) {
			bindings[typeParam] = class.SuperClassTypeArgs[i]
		}
	}
	if len(bindings) == 0 {
		return nil
	}
	return bindings
}

func (p *Parser) extractInheritedEndpoints(class *ktparser.Class, basePath string, bindings map[string]string, visited map[string]bool) []Endpoint {
	if class.SuperClass == "" {
		return nil
	}
	key := class.Package + "." + class.Name
	if visited[key] {
		return nil
	}
	visited[key] = true

	superClass, found := p.typeResolver.FindClass(class, class.SuperClass)
	if !found {
		return nil
	}

	parentBindings := p.buildTypeBindings(superClass)
	mergedBindings := mergeBindings(parentBindings, bindings)

	parentBasePath := extractBasePath(superClass.Annotations)
	effectiveBasePath := basePath
	if effectiveBasePath == "" && parentBasePath != "" {
		effectiveBasePath = parentBasePath
	}

	var inherited []Endpoint
	for i := range superClass.Methods {
		method := &superClass.Methods[i]
		if !isEndpointMethod(method) {
			continue
		}
		if isMethodOverridden(method.Name, class) {
			continue
		}
		if endpoint := p.extractEndpoint(method, effectiveBasePath, class, superClass, mergedBindings); endpoint != nil {
			inherited = append(inherited, *endpoint)
		}
	}
	inherited = append(inherited, p.extractInheritedEndpoints(superClass, effectiveBasePath, mergedBindings, visited)...)
	return inherited
}

func mergeBindings(parent, child map[string]string) map[string]string {
	if len(parent) == 0 && len(child) == 0 {
		return nil
	}
	merged := map[string]string{}
	for key, value := range parent {
		merged[key] = value
	}
	for key, value := range child {
		merged[key] = value
	}
	return merged
}

func isEndpointMethod(method *ktparser.Function) bool {
	for _, ann := range method.Annotations {
		switch ann.Name {
		case "GetMapping", "PostMapping", "PutMapping", "DeleteMapping", "PatchMapping", "RequestMapping":
			return true
		}
	}
	return false
}

func isMethodOverridden(methodName string, class *ktparser.Class) bool {
	for i := range class.Methods {
		if class.Methods[i].Name == methodName {
			return true
		}
	}
	return false
}

func isController(class *ktparser.Class) bool {
	for _, ann := range class.Annotations {
		if ann.Name == "RestController" || ann.Name == "Controller" {
			return true
		}
	}
	return false
}

func extractBasePath(annotations []ktparser.Annotation) string {
	for _, ann := range annotations {
		if ann.Name != "RequestMapping" {
			continue
		}
		if value, ok := ann.Params["value"]; ok {
			return normalizePath(value)
		}
		if value, ok := ann.Params["path"]; ok {
			return normalizePath(value)
		}
	}
	return ""
}

// extractEndpoint builds one endpoint. class names the controller that owns
// the endpoint; source is the declaration carrying the method, which is the
// superclass for inherited endpoints.
func (p *Parser) extractEndpoint(method *ktparser.Function, basePath string, class, source *ktparser.Class, bindings map[string]string) *Endpoint {
	httpMethod, methodPath, consumes, produces := extractMethodInfo(method.Annotations)
	if httpMethod == "" {
		return nil
	}

	fullPath := combinePaths(basePath, methodPath)

	var params []EndpointParameter
	var requestBodyType string
	methodDocParams := javadoc.ParseJavaDocParams(method.KDoc)
	for i := range method.Parameters {
		param := &method.Parameters[i]
		if ep := p.extractParameter(param, methodDocParams); ep != nil {
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
		Description: javadoc.EndpointDescription(toJavaAnnotations(method.Annotations), javadoc.CleanJavaDoc(method.KDoc)),
		Parameters:  params,
		ReturnType:  method.ReturnType,
		Consumes:    consumes,
		Produces:    produces,
		ClassName:   class.Name,
		Package:     class.Package,
	}

	if requestBodyType != "" {
		endpoint.RequestBodySchema = p.typeResolver.ResolveFrom(source, requestBodyType, bindings)
	}

	if isVoidReturn(method.ReturnType) {
		if !method.HasDeclaredReturn && method.ReturnType == "" {
			p.warnings = append(p.warnings, class.FilePath+": endpoint "+class.Name+"."+method.Name+
				" uses an expression body without a declared return type; its response schema is not exported")
		}
	} else {
		endpoint.ResponseSchema = p.typeResolver.ResolveFrom(source, method.ReturnType, bindings)
	}

	return endpoint
}

func isVoidReturn(returnType string) bool {
	switch strings.TrimSpace(returnType) {
	case "", "Unit", "kotlin.Unit", "Void", "java.lang.Void", "Nothing", "kotlin.Nothing":
		return true
	}
	return false
}

func extractMethodInfo(annotations []ktparser.Annotation) (HTTPMethod, string, []string, []string) {
	for _, ann := range annotations {
		var method HTTPMethod
		switch ann.Name {
		case "GetMapping":
			method = GET
		case "PostMapping":
			method = POST
		case "PutMapping":
			method = PUT
		case "DeleteMapping":
			method = DELETE
		case "PatchMapping":
			method = PATCH
		case "RequestMapping":
			method = extractHTTPMethodFromRequestMapping(ann, annotations)
		default:
			continue
		}
		return method, extractPathFromMapping(ann),
			mediaTypes(ann.Params["consumes"]),
			mediaTypes(ann.Params["produces"])
	}
	return "", "", nil, nil
}

func extractPathFromMapping(ann ktparser.Annotation) string {
	if value, ok := ann.Params["value"]; ok {
		return normalizePath(value)
	}
	if value, ok := ann.Params["path"]; ok {
		return normalizePath(value)
	}
	return ""
}

func extractHTTPMethodFromRequestMapping(ann ktparser.Annotation, allAnnotations []ktparser.Annotation) HTTPMethod {
	if method, ok := ann.Params["method"]; ok {
		method = strings.TrimPrefix(method, "RequestMethod.")
		if idx := strings.Index(method, ","); idx >= 0 {
			method = method[:idx]
		}
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
	for _, a := range allAnnotations {
		if a.Name != "ApiOperation" {
			continue
		}
		switch strings.ToUpper(strings.TrimSpace(a.Params["httpMethod"])) {
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

func (p *Parser) extractParameter(param *ktparser.Parameter, methodDocParams map[string]string) *EndpointParameter {
	paramType := detectParameterType(param.Annotations)
	if paramType == "" {
		return nil
	}

	paramDoc := methodDocParams[param.Name]
	doc := javadoc.ParameterDocumentation(toJavaAnnotations(param.Annotations), paramDoc)

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

	// Kotlin never loses parameter names, so the identifier is the fallback.
	// An explicit annotation name still wins: @RequestHeader("X-Tenant").
	if name := explicitAnnotationName(param.Annotations); name != "" {
		ep.Name = name
	}
	return ep
}

func detectParameterType(annotations []ktparser.Annotation) string {
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
		case "RequestPart":
			return "form"
		case "CookieValue":
			return "cookie"
		}
	}
	for _, ann := range annotations {
		if ann.Name == "ApiParam" {
			return "query"
		}
	}
	return ""
}

func explicitAnnotationName(annotations []ktparser.Annotation) string {
	for _, ann := range annotations {
		switch ann.Name {
		case "PathVariable", "RequestParam", "RequestHeader", "CookieValue", "RequestPart":
			if value := firstNonEmpty(ann.Params["value"], ann.Params["name"]); value != "" {
				return value
			}
		}
	}
	return ""
}

func toJavaAnnotations(annotations []ktparser.Annotation) []javaparser.Annotation {
	if len(annotations) == 0 {
		return nil
	}
	out := make([]javaparser.Annotation, 0, len(annotations))
	for _, ann := range annotations {
		out = append(out, javaparser.Annotation{Name: ann.Name, Params: ann.Params})
	}
	return out
}

func normalizePath(value string) string {
	value = strings.Trim(value, "\"")
	value = strings.Trim(value, "'")
	if idx := strings.Index(value, ","); idx >= 0 {
		value = value[:idx]
	}
	if value != "" && !strings.HasPrefix(value, "/") {
		value = "/" + value
	}
	return value
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

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

var mediaTypeConstants = map[string]string{
	"APPLICATION_JSON":            "application/json",
	"APPLICATION_XML":             "application/xml",
	"APPLICATION_ATOM_XML":        "application/atom+xml",
	"APPLICATION_FORM_URLENCODED": "application/x-www-form-urlencoded",
	"APPLICATION_OCTET_STREAM":    "application/octet-stream",
	"APPLICATION_PDF":             "application/pdf",
	"MULTIPART_FORM_DATA":         "multipart/form-data",
	"TEXT_PLAIN":                  "text/plain",
	"TEXT_XML":                    "text/xml",
	"TEXT_HTML":                   "text/html",
	"WILDCARD":                    "*/*",
}

func mediaTypes(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	var out []string
	for _, part := range strings.Split(raw, ",") {
		if value := normalizeMediaType(part); value != "" {
			out = append(out, value)
		}
	}
	return out
}

func normalizeMediaType(raw string) string {
	value := strings.TrimSpace(raw)
	value = strings.Trim(value, "\"'")
	if value == "" {
		return ""
	}
	if strings.Contains(value, "/") {
		return value
	}
	name := value
	if idx := strings.LastIndex(name, "."); idx >= 0 {
		name = name[idx+1:]
	}
	if mapped, ok := mediaTypeConstants[strings.ToUpper(name)]; ok {
		return mapped
	}
	return value
}
