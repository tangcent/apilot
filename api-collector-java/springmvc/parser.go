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
	warnings             []string
	requiredFields       string
	selection            *collector.FileSelection
}

// Warnings returns the diagnostics of the last extraction: handler
// parameters that are not exported, and why.
func (p *Parser) Warnings() []string {
	return p.warnings
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

// SetRequiredFields selects how resolved fields are marked required: one of
// collector.RequiredFieldsDefault or collector.RequiredFieldsConstraints.
func (p *Parser) SetRequiredFields(mode string) {
	p.requiredFields = mode
}

// SetSelection limits ExtractControllers to the controllers declared in the
// selected files. The type table still holds every class of results, so a
// selected controller resolves types declared anywhere in the project, but
// nothing is extracted or resolved for a controller outside the selection.
// Optional; a nil selection extracts every controller.
func (p *Parser) SetSelection(selection *collector.FileSelection) {
	p.selection = selection
}

func (p *Parser) ExtractControllers(results []parser.ParseResult) []Controller {
	typeResolver := resolver.NewTypeResolver(flattenClasses(results))
	if p.dependencyResolver != nil {
		typeResolver.SetDependencyResolver(p.dependencyResolver)
	} else if p.collectorDepResolver != nil {
		typeResolver.SetCollectorDependencyResolver(p.collectorDepResolver)
	}
	typeResolver.SetUnresolved(p.unresolved)
	typeResolver.SetRequiredFields(p.requiredFields)

	var controllers []Controller

	for _, result := range results {
		if result.Error != nil {
			continue
		}

		for _, class := range result.Classes {
			if !p.selection.Contains(class.FilePath) {
				continue
			}
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

	typeBindings := buildTypeBindings(class, nil, typeResolver)

	var endpoints []Endpoint
	for _, method := range class.Methods {
		if endpoint := p.extractEndpoint(method, basePath, class, class, typeResolver, typeBindings); endpoint != nil {
			endpoints = append(endpoints, *endpoint)
		}
	}

	inheritedEndpoints := p.extractInheritedEndpoints(class, basePath, typeResolver, typeBindings)
	endpoints = append(endpoints, inheritedEndpoints...)

	return &Controller{
		Name:        class.Name,
		Package:     class.Package,
		BasePath:    basePath,
		Endpoints:   endpoints,
		Description: class.JavaDoc,
	}
}

// buildTypeBindings binds the type parameters of class's superclass to the
// arguments of its extends clause. bindings binds the type parameters of class
// itself, for an argument that mentions them. Each argument is spelled so that
// the superclass, which resolves it in its own scope, reads the type class
// named (see TypeResolver.SuperClassBindings).
func buildTypeBindings(class parser.Class, bindings map[string]string, typeResolver *resolver.TypeResolver) map[string]string {
	if class.SuperClass == "" {
		return nil
	}

	superClass, found := typeResolver.Class(class, class.SuperClass)
	if !found {
		return nil
	}

	return typeResolver.SuperClassBindings(class, superClass, bindings)
}

// extractInheritedEndpoints extracts the endpoints class inherits. typeBindings
// binds the type parameters of class's superclass; the endpoints of that
// superclass resolve their types with it, and its own superclass is bound in
// turn from what the extends clause of the superclass says.
func (p *Parser) extractInheritedEndpoints(class parser.Class, basePath string, typeResolver *resolver.TypeResolver, typeBindings map[string]string) []Endpoint {
	if class.SuperClass == "" {
		return nil
	}

	superClass, found := typeResolver.Class(class, class.SuperClass)
	if !found {
		return nil
	}

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
		if endpoint := p.extractEndpoint(method, effectiveBasePath, class, superClass, typeResolver, typeBindings); endpoint != nil {
			inherited = append(inherited, *endpoint)
		}
	}

	grandparentBindings := buildTypeBindings(superClass, typeBindings, typeResolver)
	grandparentEndpoints := p.extractInheritedEndpoints(superClass, effectiveBasePath, typeResolver, grandparentBindings)
	inherited = append(inherited, grandparentEndpoints...)

	return inherited
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
	httpMethod, methodPath, consumes, produces := p.extractMethodInfo(method.Annotations)
	if httpMethod == "" {
		return nil
	}

	fullPath := p.combinePaths(basePath, methodPath)

	hasBody := false
	for _, param := range method.Parameters {
		if p.detectParameterType(param.Annotations) == "body" {
			hasBody = true
		}
	}
	implicitIn := ImplicitLocation(string(httpMethod), hasBody)

	var params []EndpointParameter
	var requestBodyType string
	for _, param := range method.Parameters {
		if ep := p.extractParameter(param, method, source, typeResolver); ep != nil {
			params = append(params, *ep)
			if ep.ParamType == "body" {
				requestBodyType = param.Type
			}
			continue
		}
		params = append(params, p.implicitParameters(param, implicitIn, method, source, typeResolver, typeBindings)...)
	}

	endpoint := &Endpoint{
		Path:        fullPath,
		Method:      httpMethod,
		MethodName:  method.Name,
		Description: javadoc.EndpointDescription(method.Annotations, method.JavaDoc),
		Parameters:  params,
		ReturnType:  method.ReturnType,
		Consumes:    consumes,
		Produces:    produces,
		ClassName:   class.Name,
		Package:     class.Package,
	}
	deprecation := javadoc.EndpointDeprecation(
		javadoc.DeprecationOf(method.Annotations, method.RawJavaDoc),
		javadoc.DeprecationOf(source.Annotations, source.RawJavaDoc),
		javadoc.DeprecationOf(class.Annotations, class.RawJavaDoc),
	)
	endpoint.Deprecated, endpoint.DeprecationNote = deprecation.Deprecated, deprecation.Note

	if requestBodyType != "" {
		endpoint.RequestBodySchema = typeResolver.ResolveFrom(source, requestBodyType, typeBindings)
	}

	if method.ReturnType != "" && method.ReturnType != "void" && method.ReturnType != "Void" {
		// A ResponseEntity<T> wrapper is unwrapped by the type resolver, the
		// same way it handles a reactor publisher. A wildcard argument
		// (`Result<?>`) is filled from the type the body returns; when the body
		// names none, the argument resolves to an open object rather than to a
		// literal "?" (issue #167).
		endpoint.ResponseSchema = typeResolver.ResolveReturn(source, method, typeBindings)
	}

	return endpoint
}

// extractMethodInfo reads the mapping annotation of a handler method: the HTTP
// method, the path, and the media types its consumes/produces attributes
// declare. A mapping annotation that declares neither media type leaves them
// nil, so the export can tell "declared nothing" from "declared json".
func (p *Parser) extractMethodInfo(annotations []parser.Annotation) (HTTPMethod, string, []string, []string) {
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
			method = p.extractHTTPMethodFromRequestMapping(ann, annotations)
		default:
			continue
		}
		return method, p.extractPathFromMapping(ann),
			parser.MediaTypes(ann.Params["consumes"]),
			parser.MediaTypes(ann.Params["produces"])
	}
	return "", "", nil, nil
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

// extractParameter builds a parameter bound by an annotation. method declares
// it and source is the class whose declaration carries method, the scope a
// constant naming the parameter resolves in.
func (p *Parser) extractParameter(param parser.Parameter, method parser.Method, source parser.Class, typeResolver *resolver.TypeResolver) *EndpointParameter {
	paramType := p.detectParameterType(param.Annotations)
	if paramType == "" {
		return nil
	}

	paramJavaDoc := param.JavaDoc
	if paramJavaDoc == "" && method.JavaDocParams != nil {
		paramJavaDoc = method.JavaDocParams[param.Name]
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
	names := make([]string, 0, len(param.Annotations))
	for _, ann := range param.Annotations {
		names = append(names, ann.Name)
	}
	if OptionalArgument(paramType, param.Type, names, false) {
		ep.Required = false
	}
	if doc.Default != "" {
		ep.DefaultValue = doc.Default
	}
	if doc.Required != nil {
		ep.Required = *doc.Required
	}

	// The name on the wire comes from the annotation, not the Java parameter
	// name: @RequestHeader("Authorization") String auth binds the
	// Authorization header, and @RequestParam(Params.PAGE) int p binds the
	// request parameter the constant names.
	name, unresolved := BindingName(param.Annotations, func(ref string) (string, bool) {
		return typeResolver.ConstantValue(source, ref)
	})
	switch {
	case name != "":
		ep.Name = name
	case unresolved != "":
		p.warnings = append(p.warnings, UnresolvedBindingNameWarning(source.Name, method.Name, param.Name, unresolved))
	}

	return ep
}

// implicitParameters exports a handler parameter that carries no binding
// annotation the way Spring MVC binds it by default: a simple value as an
// optional request parameter, an object as a model attribute whose properties
// are request parameters, a MultipartFile as a form part. Framework objects
// and parameters left to a custom argument resolver are not request data and
// export nothing.
func (p *Parser) implicitParameters(param parser.Parameter, in string, method parser.Method, source parser.Class, typeResolver *resolver.TypeResolver, typeBindings map[string]string) []EndpointParameter {
	names := make([]string, 0, len(param.Annotations))
	for _, ann := range param.Annotations {
		names = append(names, ann.Name)
	}
	isEnum := func(typeName string) bool {
		_, ok := typeResolver.EnumConstants(source, typeName)
		return ok
	}
	kind, blocker := ClassifyImplicit(param.Type, names, isEnum)
	if blocker != "" {
		p.warnings = append(p.warnings, ImplicitNoneWarning(source.Name, method.Name, param.Name, blocker))
		return nil
	}

	paramJavaDoc := param.JavaDoc
	if paramJavaDoc == "" && method.JavaDocParams != nil {
		paramJavaDoc = method.JavaDocParams[param.Name]
	}
	doc := javadoc.ParameterDocumentation(param.Annotations, paramJavaDoc)

	switch kind {
	case ImplicitParam:
		enum := doc.Enum
		if len(enum) == 0 {
			enum, _ = typeResolver.EnumConstants(source, ElementType(param.Type))
		}
		return []EndpointParameter{{
			Name:         param.Name,
			Type:         param.Type,
			ParamType:    in,
			Required:     ImplicitRequired(names),
			DefaultValue: doc.Default,
			Description:  doc.Description,
			Example:      doc.Example,
			Enum:         enum,
		}}
	case ImplicitFile:
		return []EndpointParameter{{
			Name:        param.Name,
			Type:        param.Type,
			ParamType:   "form",
			Required:    ImplicitRequired(names),
			Description: doc.Description,
		}}
	case ImplicitPageable:
		return boundParameters(PageableFields(), in)
	case ImplicitSort:
		return boundParameters(SortFields(), in)
	case ImplicitModel:
		resolved := typeResolver.ResolveForBinding(source, param.Type, typeBindings)
		return boundParameters(ResolvedModelFields(param.Name, resolved), in)
	}
	return nil
}

func boundParameters(fields []BoundField, in string) []EndpointParameter {
	out := make([]EndpointParameter, 0, len(fields))
	for _, f := range fields {
		out = append(out, EndpointParameter{
			Name:         f.Name,
			Type:         f.TypeName,
			ParamType:    in,
			Required:     f.Required,
			DefaultValue: f.Default,
			Description:  f.Description,
			Example:      f.Example,
			Enum:         f.Enum,
		})
	}
	return out
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
		case "RequestPart":
			// A part of a multipart request is a form field, never a query
			// argument.
			return "form"
		case "CookieValue":
			return "cookie"
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
