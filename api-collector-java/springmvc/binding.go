package springmvc

import (
	"strings"

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
	// ImplicitFile: a MultipartFile, bound as a part of a multipart request.
	ImplicitFile
	// ImplicitModel: an object, bound property by property from request
	// parameters the way @ModelAttribute is.
	ImplicitModel
	// ImplicitPageable: a Spring Data Pageable, bound from the page, size and
	// sort request parameters.
	ImplicitPageable
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
func ClassifyImplicit(typeName string, annotations []string, isEnum func(string) bool) (ImplicitKind, string) {
	for _, name := range annotations {
		if name == "ModelAttribute" {
			return ImplicitModel, ""
		}
		if !transparentAnnotations[name] {
			return ImplicitNone, name
		}
	}
	base, args := splitType(typeName)
	simple := simpleName(base)
	switch {
	case frameworkTypes[simple]:
		return ImplicitNone, ""
	case simple == "MultipartFile":
		return ImplicitFile, ""
	case simple == "Pageable":
		return ImplicitPageable, ""
	case simpleValueTypes[simple] || (isEnum != nil && isEnum(base)):
		return ImplicitParam, ""
	case strings.HasSuffix(base, "[]"):
		element := strings.TrimSpace(strings.TrimSuffix(base, "[]"))
		if simpleValueTypes[simpleName(element)] || (isEnum != nil && isEnum(element)) {
			return ImplicitParam, ""
		}
		return ImplicitNone, ""
	case collectionTypes[simple]:
		if len(args) == 1 {
			element, _ := splitType(args[0])
			if simpleValueTypes[simpleName(element)] || (isEnum != nil && isEnum(element)) {
				return ImplicitParam, ""
			}
		}
		return ImplicitNone, ""
	}
	return ImplicitModel, ""
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

// BoundField is one request parameter a model attribute is bound from.
type BoundField struct {
	Name        string
	TypeName    string
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
		*out = append(*out, BoundField{
			Name:        path,
			TypeName:    field.Model.TypeName,
			Required:    field.Required,
			Default:     field.DefaultValue,
			Description: field.Comment,
			Example:     field.Demo,
			Enum:        enum,
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
	"Sort": true, "Continuation": true,
}

// simpleValueTypes are bound from a single request parameter value.
var simpleValueTypes = map[string]bool{
	"String": true, "CharSequence": true,
	"int": true, "long": true, "short": true, "byte": true, "double": true, "float": true, "boolean": true, "char": true,
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
