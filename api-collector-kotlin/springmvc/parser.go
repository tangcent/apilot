// Package springmvc extracts Spring MVC endpoints from Kotlin declarations.
package springmvc

import (
	"path"
	"strings"

	collector "github.com/tangcent/apilot/api-collector"
	javadoc "github.com/tangcent/apilot/api-collector-java/doc"
	javaparser "github.com/tangcent/apilot/api-collector-java/parser"
	javaspringmvc "github.com/tangcent/apilot/api-collector-java/springmvc"
	ktparser "github.com/tangcent/apilot/api-collector-kotlin/parser"
	"github.com/tangcent/apilot/api-collector-kotlin/resolver"
)

// Parser extracts controllers from Kotlin declarations.
type Parser struct {
	typeResolver *resolver.Resolver
	warnings     []string
	selection    *collector.FileSelection
}

// NewParser creates a Spring MVC parser backed by the Kotlin type resolver.
func NewParser(typeResolver *resolver.Resolver) *Parser {
	return &Parser{typeResolver: typeResolver}
}

// SetSelection limits ExtractControllers to the controllers declared in the
// selected files. The type resolver still holds every class of the project, so
// a selected controller resolves types declared anywhere in it, but nothing is
// extracted or resolved for a controller outside the selection. Optional; a nil
// selection extracts every controller.
func (p *Parser) SetSelection(selection *collector.FileSelection) {
	p.selection = selection
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
			// Only the selected files take part: a same-named class of another
			// file must not shadow the one that was asked for.
			if !p.selection.Contains(class.FilePath) {
				continue
			}
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
	bindings := p.buildTypeBindings(class, nil)

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
		Name:        class.Name,
		Package:     class.Package,
		BasePath:    basePath,
		Endpoints:   endpoints,
		Description: javadoc.CleanJavaDoc(class.KDoc),
	}
}

// buildTypeBindings binds the type parameters of class's superclass to the
// arguments of its extends clause. bindings binds the type parameters of class
// itself, for an argument that mentions them. Each argument is spelled so that
// the superclass, which resolves it in its own scope, reads the type class
// named (see Resolver.SuperClassBindings).
func (p *Parser) buildTypeBindings(class *ktparser.Class, bindings map[string]string) map[string]string {
	if class.SuperClass == "" {
		return nil
	}
	superClass, found := p.typeResolver.FindSupertype(class, class.SuperClass)
	if !found {
		return nil
	}
	superBindings := p.typeResolver.SuperClassBindings(class, superClass, bindings)
	if len(superBindings) == 0 {
		return nil
	}
	return superBindings
}

// extractInheritedEndpoints extracts the endpoints class inherits. bindings
// binds the type parameters of class's superclass; the endpoints of that
// superclass resolve their types with it, and its own superclass is bound in
// turn from what the extends clause of the superclass says.
func (p *Parser) extractInheritedEndpoints(class *ktparser.Class, basePath string, bindings map[string]string, visited map[string]bool) []Endpoint {
	if class.SuperClass == "" {
		return nil
	}
	key := class.Package + "." + class.Name
	if visited[key] {
		return nil
	}
	visited[key] = true

	superClass, found := p.typeResolver.FindSupertype(class, class.SuperClass)
	if !found {
		return nil
	}

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
		if endpoint := p.extractEndpoint(method, effectiveBasePath, class, superClass, bindings); endpoint != nil {
			inherited = append(inherited, *endpoint)
		}
	}
	grandparentBindings := p.buildTypeBindings(superClass, bindings)
	inherited = append(inherited, p.extractInheritedEndpoints(superClass, effectiveBasePath, grandparentBindings, visited)...)
	return inherited
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

// isController decides with the same predicate the parser's recovery uses, so
// a file the recovery treats as a controller is one this parser reads as one.
func isController(class *ktparser.Class) bool {
	for _, ann := range class.Annotations {
		if ktparser.IsControllerAnnotation(ann.Name) {
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

	hasBody := false
	for i := range method.Parameters {
		if detectParameterType(method.Parameters[i].Annotations) == "body" {
			hasBody = true
		}
	}
	implicitIn := javaspringmvc.ImplicitLocation(string(httpMethod), hasBody)

	var params []EndpointParameter
	var requestBodyType string
	methodDocParams := javadoc.ParseJavaDocParams(method.KDoc)
	for i := range method.Parameters {
		param := &method.Parameters[i]
		if ep := p.extractParameter(param, methodDocParams, method.Name, source); ep != nil {
			params = append(params, *ep)
			if ep.ParamType == "body" {
				requestBodyType = param.Type
			}
			continue
		}
		params = append(params, p.implicitParameters(param, implicitIn, method.Name, methodDocParams, source, bindings)...)
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
	deprecation := javadoc.EndpointDeprecation(
		javadoc.DeprecationOf(toJavaAnnotations(method.Annotations), method.KDoc),
		javadoc.DeprecationOf(toJavaAnnotations(source.Annotations), source.KDoc),
		javadoc.DeprecationOf(toJavaAnnotations(class.Annotations), class.KDoc),
	)
	endpoint.Deprecated, endpoint.DeprecationNote = deprecation.Deprecated, deprecation.Note

	if requestBodyType != "" {
		endpoint.RequestBodySchema = p.typeResolver.ResolveFrom(source, requestBodyType, bindings)
	}

	if p.isVoidReturn(source, method.ReturnType) {
		if !method.HasDeclaredReturn && method.ReturnType == "" {
			p.warnings = append(p.warnings, class.FilePath+": endpoint "+class.Name+"."+method.Name+
				" uses an expression body without a declared return type; its response schema is not exported")
		}
	} else {
		endpoint.ResponseSchema = p.typeResolver.ResolveFrom(source, method.ReturnType, bindings)
	}

	return endpoint
}

// isVoidReturn reports whether a function's declared return type, as written in
// source, says it returns nothing: Unit, Nothing or java.lang.Void, which the
// scope of the name decides. A class of the project that is called Unit, an
// enum for one, is a class like any other.
func (p *Parser) isVoidReturn(source *ktparser.Class, returnType string) bool {
	returnType = strings.TrimSpace(returnType)
	return returnType == "" || p.typeResolver.NormalizeType(source, returnType) == "java.lang.Void"
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

// extractParameter builds a parameter bound by an annotation. source is the
// declaration carrying the function, the scope a constant naming the
// parameter resolves in.
func (p *Parser) extractParameter(param *ktparser.Parameter, methodDocParams map[string]string, methodName string, source *ktparser.Class) *EndpointParameter {
	paramType := detectParameterType(param.Annotations)
	if paramType == "" {
		return nil
	}

	paramDoc := methodDocParams[param.Name]
	doc := javadoc.ParameterDocumentation(toJavaAnnotations(param.Annotations), paramDoc)

	ep := &EndpointParameter{
		Name:        param.Name,
		Type:        param.Type,
		File:        p.typeResolver != nil && p.typeResolver.IsMultipartFile(source, param.Type),
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
	if javaspringmvc.OptionalArgument(paramType, names, (p.typeResolver != nil && p.typeResolver.IsOptionalType(source, param.Type)) || param.Nullable || param.HasDefault) {
		ep.Required = false
	}
	if doc.Default != "" {
		ep.DefaultValue = doc.Default
	}
	if doc.Required != nil {
		ep.Required = *doc.Required
	}

	// Kotlin never loses parameter names, so the identifier is the fallback.
	// An explicit annotation name still wins: @RequestHeader("X-Tenant"), or
	// @RequestHeader(Headers.TENANT) for the string the constant holds.
	name, unresolved := javaspringmvc.BindingName(toJavaAnnotations(param.Annotations), func(ref string) (string, bool) {
		if p.typeResolver == nil {
			return "", false
		}
		return p.typeResolver.ConstantValue(source, ref)
	})
	switch {
	case name != "":
		ep.Name = name
	case unresolved != "":
		className := ""
		if source != nil {
			className = source.Name
		}
		p.warnings = append(p.warnings, javaspringmvc.UnresolvedBindingNameWarning(className, methodName, param.Name, unresolved))
	}
	return ep
}

// implicitParameters exports a handler parameter that carries no binding
// annotation the way Spring MVC binds it by default; the rules are shared with
// the Java collector so structurally identical sources export the same
// parameters.
func (p *Parser) implicitParameters(param *ktparser.Parameter, in, methodName string, methodDocParams map[string]string, source *ktparser.Class, bindings map[string]string) []EndpointParameter {
	names := make([]string, 0, len(param.Annotations))
	for _, ann := range param.Annotations {
		names = append(names, ann.Name)
	}
	// Spring binds a java.util.Optional parameter as the type it holds, and
	// leaves it empty when the request has no value for it.
	typeName := p.valueType(source, param.Type)
	isEnum := func(name string) bool { return p.typeResolver.IsEnumIn(source, name) }
	identity := func(name string) string { return p.typeResolver.Identity(source, name) }
	kind, blocker := javaspringmvc.ClassifyImplicit(typeName, names, isEnum, identity)
	if blocker != "" {
		p.warnings = append(p.warnings, javaspringmvc.ImplicitNoneWarning(source.Name, methodName, param.Name, blocker))
		return nil
	}
	doc := javadoc.ParameterDocumentation(toJavaAnnotations(param.Annotations), methodDocParams[param.Name])

	switch kind {
	case javaspringmvc.ImplicitParam:
		enum := doc.Enum
		if len(enum) == 0 {
			enum = p.typeResolver.EnumValues(javaspringmvc.ElementType(typeName, identity))
		}
		return []EndpointParameter{{
			Name:         param.Name,
			Type:         param.Type,
			ParamType:    in,
			Required:     javaspringmvc.ImplicitRequired(names),
			DefaultValue: doc.Default,
			Description:  doc.Description,
			Example:      doc.Example,
			Enum:         enum,
		}}
	case javaspringmvc.ImplicitFile:
		return []EndpointParameter{{
			Name:        param.Name,
			Type:        param.Type,
			File:        true,
			ParamType:   "form",
			Required:    javaspringmvc.ImplicitRequired(names),
			Description: doc.Description,
		}}
	case javaspringmvc.ImplicitPageable:
		return boundParameters(javaspringmvc.PageableFields(), in)
	case javaspringmvc.ImplicitSort:
		return boundParameters(javaspringmvc.SortFields(), in)
	case javaspringmvc.ImplicitModel:
		resolved := p.typeResolver.ResolveForBinding(source, param.Type, bindings)
		return boundParameters(javaspringmvc.ResolvedModelFields(param.Name, resolved), in)
	}
	return nil
}

// valueType returns the type a parameter holds in the spelling the shared
// resolver reads: the type inside a java.util.Optional, and the type itself
// otherwise.
func (p *Parser) valueType(source *ktparser.Class, rawType string) string {
	if p.typeResolver == nil {
		return p.typeResolver.NormalizeType(source, rawType)
	}
	return p.typeResolver.UnwrapOptional(source, rawType)
}

func boundParameters(fields []javaspringmvc.BoundField, in string) []EndpointParameter {
	out := make([]EndpointParameter, 0, len(fields))
	for _, f := range fields {
		out = append(out, EndpointParameter{
			Name:         f.Name,
			Type:         f.TypeName,
			File:         f.File,
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

func toJavaAnnotations(annotations []ktparser.Annotation) []javaparser.Annotation {
	if len(annotations) == 0 {
		return nil
	}
	out := make([]javaparser.Annotation, 0, len(annotations))
	for _, ann := range annotations {
		out = append(out, javaparser.Annotation{Name: ann.Name, Params: ann.Params, Refs: ann.Refs})
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
