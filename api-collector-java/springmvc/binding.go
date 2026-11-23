package springmvc

import (
	"strings"

	"github.com/tangcent/apilot/api-collector-java/parser"
	"github.com/tangcent/apilot/api-collector-java/resolver"
	model "github.com/tangcent/apilot/api-model"
)

// ImplicitKind says how Spring MVC binds a handler parameter that carries no
// binding annotation (@RequestParam, @RequestBody, @PathVariable, ...).
type ImplicitKind int

const (
	// ImplicitNone: the argument does not come from the request data, or is
	// left to a resolver the collector cannot see. Framework objects such as
	// HttpServletRequest, BindingResult or Locale belong here, and so does a
	// parameter with an annotation no known binding covers (@CurrentUser),
	// which a custom argument resolver handles.
	ImplicitNone ImplicitKind = iota
	// ImplicitParam: a simple value (string, number, boolean, date, enum, or a
	// collection or array of them), bound as a request parameter the way an
	// optional @RequestParam is.
	ImplicitParam
	// ImplicitFile: a MultipartFile, or a collection or array of them, bound as
	// parts of a multipart request.
	ImplicitFile
	// ImplicitModel: an object, bound property by property from request
	// parameters the way @ModelAttribute is.
	ImplicitModel
	// ImplicitPageable: a Spring Data Pageable, bound from the page, size and
	// sort request parameters.
	ImplicitPageable
	// ImplicitSort: a Spring Data Sort, bound from the sort request parameter,
	// the same parameter a Pageable takes its order from.
	ImplicitSort
)

// ClassifyImplicit decides how Spring binds a parameter of the given type that
// has none of the binding annotations. annotations are the simple names of the
// annotations the parameter does carry; isEnum reports whether a type name is
// an enum of the project. When the result is ImplicitNone because of an
// annotation, that annotation's name is returned so the caller can say why the
// parameter is not exported.
//
// The rules follow Spring's default argument resolution: framework types are
// resolved from the request context, simple values become request parameters,
// and any other object is a model attribute.
//
// typeName is read as written, and every class it names is the class its scope
// gives it: identity returns the qualified name a class name denotes where the
// type is written (see resolver.TypeResolver.Identity). A framework type, a
// simple value, a collection, a Pageable, a Sort or a MultipartFile is one by
// that identity, not by its simple name, so a class or an enum of the project
// that bears such a name is not. Spring binds a java.util.Optional parameter as
// the type it holds, but only the scope of the name says whether an `Optional`
// is that class or the project's own, so the caller looks through it first (see
// resolver.TypeResolver.UnwrapOptional).
func ClassifyImplicit(typeName string, annotations []string, isEnum func(string) bool, identity func(string) string) (ImplicitKind, string) {
	// Every annotation is looked at before deciding, so the outcome does not
	// depend on the order they are written in: an explicit @ModelAttribute wins
	// wherever it stands.
	unknown := ""
	for _, name := range annotations {
		if name == "ModelAttribute" {
			return ImplicitModel, ""
		}
		if !transparentAnnotations[name] && unknown == "" {
			unknown = name
		}
	}
	if unknown != "" {
		return ImplicitNone, unknown
	}

	if identity == nil {
		identity = func(name string) string { return name }
	}
	// is reports whether the class a name denotes is one of table's: a class of
	// the JDK or of a library the resolver knows, and a class of the same simple
	// name that is not one of them is not.
	is := func(name string, table map[string]bool) bool {
		denoted := identity(name)
		return table[simpleName(denoted)] && resolver.IsLibraryClass(denoted)
	}
	simpleOrEnum := func(t string) bool {
		t = strings.TrimSpace(t)
		return primitiveKeywords[t] || is(t, simpleValueTypes) || (isEnum != nil && isEnum(t))
	}
	base, args := splitType(typeName)
	element := ""
	switch {
	case strings.HasSuffix(base, "[]"):
		element = strings.TrimSpace(strings.TrimSuffix(base, "[]"))
	case is(base, collectionTypes) && len(args) == 1:
		element, _ = splitType(args[0])
	}

	switch {
	case is(base, frameworkTypes):
		return ImplicitNone, ""
	case identity(base) == resolver.MultipartFileClass, element != "" && identity(element) == resolver.MultipartFileClass:
		// Spring binds a collection or an array of files as parts, as it binds
		// a single one.
		return ImplicitFile, ""
	case identity(base) == pageableClass:
		return ImplicitPageable, ""
	case identity(base) == sortClass:
		return ImplicitSort, ""
	case simpleOrEnum(base):
		return ImplicitParam, ""
	case element != "":
		if simpleOrEnum(element) {
			return ImplicitParam, ""
		}
		return ImplicitNone, ""
	case is(base, collectionTypes):
		return ImplicitNone, ""
	}
	return ImplicitModel, ""
}

// pageableClass and sortClass are the identities of the Spring Data types a
// handler parameter binds from the page, size and sort request parameters.
const (
	pageableClass = "org.springframework.data.domain.Pageable"
	sortClass     = "org.springframework.data.domain.Sort"
)

// ElementType returns the element type of a collection or array type, or the
// type itself when it is neither. identity is the one ClassifyImplicit takes: a
// collection is the JDK's by the class its name denotes.
func ElementType(typeName string, identity func(string) string) string {
	base, args := splitType(typeName)
	switch {
	case strings.HasSuffix(base, "[]"):
		return strings.TrimSpace(strings.TrimSuffix(base, "[]"))
	case len(args) == 1 && identity != nil && collectionTypes[simpleName(identity(base))] && resolver.IsLibraryClass(identity(base)):
		element, _ := splitType(args[0])
		return element
	}
	return base
}

// ImplicitNoneWarning says why a handler parameter is not exported: it
// carries an annotation no known Spring binding covers, which in practice is
// a custom argument resolver's (@CurrentUser) and fills the argument from
// something other than request data. Exporting it would invent request
// fields; dropping it silently would hide the decision, so it is reported.
func ImplicitNoneWarning(class, method, param, annotation string) string {
	return class + "." + method + ": parameter " + param + " carries @" + annotation +
		", which no Spring binding covers; it is left to a custom argument resolver and not exported"
}

// bindingAnnotations give the request value a handler parameter is bound to a
// name of their own, in their value or name attribute.
var bindingAnnotations = map[string]bool{
	"PathVariable": true, "RequestParam": true, "RequestHeader": true, "CookieValue": true, "RequestPart": true,
}

// BindingName returns the name a binding annotation gives the request value a
// handler parameter is bound to: @RequestHeader("X-Tenant") binds the
// X-Tenant header whatever the parameter is called. An empty name means the
// annotation names nothing, and Spring binds the parameter's own name.
//
// Spring reads the name from the compiled annotation, where the compiler has
// already substituted a constant for its value, so the name may be written as
// a reference to a String constant: @RequestHeader(Headers.TENANT). constant
// resolves such a reference. When it cannot, name is empty and unresolved is
// the reference as written, so the caller can fall back to the parameter's
// own name and say why.
func BindingName(annotations []parser.Annotation, constant func(ref string) (string, bool)) (name, unresolved string) {
	for _, ann := range annotations {
		if !bindingAnnotations[ann.Name] {
			continue
		}
		for _, key := range []string{"value", "name"} {
			value := strings.TrimSpace(ann.Params[key])
			if value == "" {
				continue
			}
			if !ann.Refs[key] {
				return value, ""
			}
			if constant != nil {
				if resolved, ok := constant(value); ok && resolved != "" {
					return resolved, ""
				}
			}
			return "", value
		}
	}
	return "", ""
}

// UnresolvedBindingNameWarning says why a handler parameter is exported under
// its own name although its binding annotation names another: the name is a
// constant whose value could not be found, neither among the scanned sources
// nor in the dependencies. The reference itself is source code, never a name
// on the wire; the parameter's name is what Spring binds when the annotation
// names nothing, and is often what the constant holds. The warning keeps that
// guess from passing for a fact.
func UnresolvedBindingNameWarning(class, method, param, ref string) string {
	return class + "." + method + ": parameter " + param + " is bound by the name in " + ref +
		", a constant whose value could not be resolved; it is exported under the parameter's own name"
}

// ImplicitRequired reports whether an implicitly bound simple parameter must
// be present. Spring binds it as an optional request parameter; a validation
// constraint on it is what makes it mandatory.
func ImplicitRequired(annotations []string) bool {
	for _, name := range annotations {
		switch name {
		case "NotNull", "NotBlank", "NotEmpty", "Nonnull":
			return true
		}
	}
	return false
}

// OptionalArgument reports whether Spring leaves a missing request parameter,
// header or multipart part unbound instead of rejecting the request, although
// its annotation does not say required = false. Spring checks
// MethodParameter.isOptional, which holds for a java.util.Optional, for a
// parameter annotated @Nullable (any annotation of that simple name), and in
// Kotlin for a nullable type or a parameter with a default value; the
// collector passes a java.util.Optional (see resolver.TypeResolver.IsOptionalType,
// which reads the class its name denotes), and the last two, as optional. A
// validation constraint such as @NotNull still makes the value mandatory:
// binding leaves it null, and validation rejects the request.
func OptionalArgument(paramType string, annotations []string, optional bool) bool {
	switch paramType {
	case "query", "header", "form":
	default:
		return false
	}
	if ImplicitRequired(annotations) {
		return false
	}
	if optional {
		return true
	}
	for _, name := range annotations {
		if name == "Nullable" {
			return true
		}
	}
	return false
}

// BoundField is one request parameter a model attribute is bound from.
type BoundField struct {
	Name     string
	TypeName string
	// File marks a property that is uploaded as a part of a multipart request: a
	// MultipartFile, or a collection or an array of them.
	File        bool
	Required    bool
	Default     string
	Description string
	Example     string
	Enum        []string
}

// ModelAttributeFields lists the request parameters Spring binds a model
// attribute from: one per property, in declaration order. A nested object's
// properties are bound with a dotted path (address.city), which is how Spring's
// data binder addresses them.
func ModelAttributeFields(m *model.ObjectModel) []BoundField {
	var out []BoundField
	appendModelFields(&out, "", m, 0)
	return out
}

// ResolvedModelFields lists the request parameters of an implicit model
// attribute once its type has been resolved. A type that resolves to a plain
// value instead of an object (an enum the type table models as a string) is
// one parameter named after the handler parameter. A type left unresolved
// yields nothing here: the resolver has recorded it, and the unresolved-types
// report names it, which is better than inventing a parameter for it.
func ResolvedModelFields(paramName string, resolved *model.ObjectModel) []BoundField {
	if resolved == nil {
		return nil
	}
	if resolved.Kind == model.KindObject {
		return ModelAttributeFields(resolved)
	}
	if resolved.Kind == model.KindSingle && jsonPrimitive[resolved.TypeName] {
		return []BoundField{{Name: paramName, TypeName: resolved.TypeName}}
	}
	return nil
}

var jsonPrimitive = map[string]bool{
	model.JsonTypeString: true, model.JsonTypeInt: true, model.JsonTypeLong: true,
	model.JsonTypeFloat: true, model.JsonTypeDouble: true, model.JsonTypeBoolean: true,
}

// ImplicitLocation is where the request parameters of an implicitly bound
// argument travel. Spring binds them from the query string and, for a form
// post, from the body; a handler that reads its body as @RequestBody leaves
// only the query string.
func ImplicitLocation(httpMethod string, hasBody bool) string {
	switch httpMethod {
	case "POST", "PUT", "PATCH":
		if !hasBody {
			return "form"
		}
	}
	return "query"
}

// PageableFields lists the request parameters Spring Data binds a Pageable
// from, with Spring Data's defaults.
func PageableFields() []BoundField {
	return []BoundField{
		{Name: "page", TypeName: "int", Default: "0", Description: "zero-based page index"},
		{Name: "size", TypeName: "int", Default: "20", Description: "page size"},
		{Name: "sort", TypeName: "string", Description: "sorting criteria: property(,asc|desc)"},
	}
}

// SortFields lists the request parameter Spring Data binds a Sort from.
func SortFields() []BoundField {
	return PageableFields()[2:]
}

// isFile reports whether a resolved property is a multipart file, or a
// collection or an array of them. The resolver answers Spring's MultipartFile
// with a single value that carries its identity, which neither a class of the
// project that is called MultipartFile nor a name that cannot be resolved does.
func isFile(m *model.ObjectModel) bool {
	if m.IsArray() {
		m = m.Items
	}
	return m.IsSingle() && m.TypeName == resolver.MultipartFileClass
}

// maxModelDepth bounds how deep nested objects are flattened; a type that
// refers back to itself would otherwise expand forever.
const maxModelDepth = 3

func appendModelFields(out *[]BoundField, prefix string, m *model.ObjectModel, depth int) {
	if m == nil || m.Kind != model.KindObject {
		return
	}
	for _, name := range m.OrderedFieldNames() {
		field := m.Fields[name]
		if field == nil || field.Model == nil {
			continue
		}
		path := prefix + name
		if field.Model.Kind == model.KindObject && len(field.Model.Fields) > 0 && depth < maxModelDepth {
			appendModelFields(out, path+".", field.Model, depth+1)
			continue
		}
		var enum []string
		for _, option := range field.Options {
			if value, ok := option.Value.(string); ok {
				enum = append(enum, value)
			}
		}
		typeName := field.Model.TypeName
		file := isFile(field.Model)
		if file {
			// A file is shown by its simple name, a collection or an array of
			// them as the array of that: `array` would hide what they hold.
			typeName = "MultipartFile"
			if field.Model.IsArray() {
				typeName = "MultipartFile[]"
			}
		}
		*out = append(*out, BoundField{
			Name:        path,
			TypeName:    typeName,
			Required:    field.Required,
			Default:     field.DefaultValue,
			Description: field.Comment,
			Example:     field.Demo,
			Enum:        enum,
			File:        file,
		})
	}
}

// transparentAnnotations do not change how Spring binds a parameter: Bean
// Validation and Hibernate Validator constraints, nullability markers, and
// documentation annotations.
var transparentAnnotations = map[string]bool{
	"Valid": true, "Validated": true,
	"NotNull": true, "NotBlank": true, "NotEmpty": true, "Null": true,
	"Size": true, "Min": true, "Max": true, "DecimalMin": true, "DecimalMax": true,
	"Digits": true, "Pattern": true, "Email": true,
	"Positive": true, "PositiveOrZero": true, "Negative": true, "NegativeOrZero": true,
	"Past": true, "PastOrPresent": true, "Future": true, "FutureOrPresent": true,
	"AssertTrue": true, "AssertFalse": true,
	"Range": true, "Length": true, "URL": true,
	"Nullable": true, "NonNull": true, "Nonnull": true,
	"Deprecated": true,
	"Parameter":  true, "Schema": true,
	// Spring annotations that format or default a bound value without
	// changing where it is bound from.
	"DateTimeFormat": true, "NumberFormat": true,
	"PageableDefault": true, "SortDefault": true, "Qualifier": true,
}

// frameworkTypes are resolved from the request context, not from request data.
var frameworkTypes = map[string]bool{
	"HttpServletRequest": true, "HttpServletResponse": true, "ServletRequest": true, "ServletResponse": true,
	"HttpSession": true, "WebRequest": true, "NativeWebRequest": true, "PushBuilder": true,
	"MultipartRequest": true, "MultipartHttpServletRequest": true,
	"Principal": true, "Authentication": true,
	"Locale": true, "TimeZone": true, "ZoneId": true,
	"InputStream": true, "OutputStream": true, "Reader": true, "Writer": true,
	"Model": true, "ModelMap": true, "Map": true, "MultiValueMap": true, "RedirectAttributes": true,
	"BindingResult": true, "Errors": true, "SessionStatus": true,
	"UriComponentsBuilder": true, "ServletUriComponentsBuilder": true,
	"HttpEntity": true, "RequestEntity": true, "HttpMethod": true, "HttpHeaders": true,
	"ServerWebExchange": true, "ServerHttpRequest": true, "ServerHttpResponse": true, "WebSession": true,
	"Continuation": true,
}

// primitiveKeywords name the types no declaration can hide.
var primitiveKeywords = map[string]bool{
	"int": true, "long": true, "short": true, "byte": true, "double": true, "float": true, "boolean": true, "char": true,
}

// The three tables below hold simple names, and a name is one of them only when
// the class it denotes is a class of the JDK or of a library the resolver knows
// (see resolver.IsLibraryClass): the names are the keys of the table of those
// classes, which lists their packages.

// simpleValueTypes are bound from a single request parameter value.
var simpleValueTypes = map[string]bool{
	"String": true, "CharSequence": true,
	"Integer": true, "Long": true, "Short": true, "Byte": true, "Double": true, "Float": true, "Boolean": true,
	"Character": true, "Number": true, "BigDecimal": true, "BigInteger": true,
	"Date": true, "LocalDate": true, "LocalDateTime": true, "LocalTime": true, "Instant": true,
	"OffsetDateTime": true, "OffsetTime": true, "ZonedDateTime": true, "YearMonth": true, "Year": true,
	"Duration": true, "Period": true, "UUID": true, "URI": true, "URL": true,
}

var collectionTypes = map[string]bool{
	"List": true, "Set": true, "Collection": true, "Iterable": true,
	"ArrayList": true, "LinkedList": true, "HashSet": true, "LinkedHashSet": true, "SortedSet": true, "TreeSet": true,
}

func simpleName(name string) string {
	name = strings.TrimSpace(name)
	if idx := strings.LastIndex(name, "."); idx >= 0 {
		return name[idx+1:]
	}
	return name
}

// splitType splits "List<String>" into "List" and its type arguments.
func splitType(raw string) (string, []string) {
	raw = strings.TrimSpace(raw)
	open := strings.Index(raw, "<")
	if open < 0 || !strings.HasSuffix(raw, ">") {
		return raw, nil
	}
	inner := raw[open+1 : len(raw)-1]
	var args []string
	depth, start := 0, 0
	for i, r := range inner {
		switch r {
		case '<':
			depth++
		case '>':
			depth--
		case ',':
			if depth == 0 {
				args = append(args, strings.TrimSpace(inner[start:i]))
				start = i + 1
			}
		}
	}
	args = append(args, strings.TrimSpace(inner[start:]))
	return strings.TrimSpace(raw[:open]), args
}
