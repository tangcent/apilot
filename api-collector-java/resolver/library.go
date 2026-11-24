package resolver

import (
	"slices"
	"strings"

	"github.com/tangcent/apilot/api-collector-java/parser"
	model "github.com/tangcent/apilot/api-model"
)

// MultipartFileClass is the identity of Spring's uploaded file. A handler
// parameter is an upload when the class its type names has this identity, and
// the resolver answers the class with a single value it remembers (see
// IsUpload): a class of the project, or one an import names, that is also called
// MultipartFile is a class.
const MultipartFileClass = "org.springframework.web.multipart.MultipartFile"

// libraryPackages lists, by simple name, the packages that declare a class of the
// JDK or of a library that the collectors read by what it is: a handler argument
// that Spring resolves from the request context, a value it binds from one request
// parameter, a wrapper that carries the payload of a response.
//
// A class is one of them by its identity, the qualified name its scope gives it
// (see Identity), never by its simple name: a class of the project that has the
// same simple name is another class, and so is a name that nothing in scope
// binds.
//
// The index of dependencies holds the packages of the project's own libraries
// only, so the classes of a third-party library are known from this table, from a
// single-type import, or from a name written in full. The table gives an import on demand its types: a
// name that `import org.springframework.web.multipart.*;` brings in is the class
// of that package.
var libraryPackages = map[string][]string{
	// The request context.
	"HttpServletRequest":  {"jakarta.servlet.http", "javax.servlet.http"},
	"HttpServletResponse": {"jakarta.servlet.http", "javax.servlet.http"},
	"HttpSession":         {"jakarta.servlet.http", "javax.servlet.http"},
	"PushBuilder":         {"jakarta.servlet.http", "javax.servlet.http"},
	"ServletRequest":      {"jakarta.servlet", "javax.servlet"},
	"ServletResponse":     {"jakarta.servlet", "javax.servlet"},
	"WebRequest":          {"org.springframework.web.context.request"},
	"NativeWebRequest":    {"org.springframework.web.context.request"},
	"MultipartRequest":    {"org.springframework.web.multipart"},
	// The uploaded file.
	"MultipartFile":               {"org.springframework.web.multipart"},
	"MultipartHttpServletRequest": {"org.springframework.web.multipart"},
	"Principal":                   {"java.security"},
	"Authentication":              {"org.springframework.security.core"},
	"Locale":                      {"java.util"},
	"TimeZone":                    {"java.util"},
	"ZoneId":                      {"java.time"},
	"InputStream":                 {"java.io"},
	"OutputStream":                {"java.io"},
	"Reader":                      {"java.io"},
	"Writer":                      {"java.io"},
	"Model":                       {"org.springframework.ui"},
	"ModelMap":                    {"org.springframework.ui"},
	"Map":                         {"java.util"},
	"MultiValueMap":               {"org.springframework.util"},
	"RedirectAttributes":          {"org.springframework.web.servlet.mvc.support"},
	"BindingResult":               {"org.springframework.validation"},
	"Errors":                      {"org.springframework.validation"},
	"SessionStatus":               {"org.springframework.web.bind.support"},
	"UriComponentsBuilder":        {"org.springframework.web.util"},
	"ServletUriComponentsBuilder": {"org.springframework.web.servlet.support"},
	"HttpEntity":                  {"org.springframework.http"},
	"RequestEntity":               {"org.springframework.http"},
	"HttpMethod":                  {"org.springframework.http"},
	"HttpHeaders":                 {"org.springframework.http"},
	"ServerWebExchange":           {"org.springframework.web.server"},
	"WebSession":                  {"org.springframework.web.server"},
	"ServerHttpRequest":           {"org.springframework.http.server.reactive", "org.springframework.http.server"},
	"ServerHttpResponse":          {"org.springframework.http.server.reactive", "org.springframework.http.server"},
	"Continuation":                {"kotlin.coroutines"},
	// Spring Data.
	"Pageable": {"org.springframework.data.domain"},
	"Sort":     {"org.springframework.data.domain"},
	// Values bound from one request parameter.
	"String":         {"java.lang"},
	"CharSequence":   {"java.lang"},
	"Integer":        {"java.lang"},
	"Long":           {"java.lang"},
	"Short":          {"java.lang"},
	"Byte":           {"java.lang"},
	"Double":         {"java.lang"},
	"Float":          {"java.lang"},
	"Boolean":        {"java.lang"},
	"Character":      {"java.lang"},
	"Number":         {"java.lang"},
	"BigDecimal":     {"java.math"},
	"BigInteger":     {"java.math"},
	"Date":           {"java.util", "java.sql"},
	"LocalDate":      {"java.time", "org.joda.time"},
	"LocalDateTime":  {"java.time", "org.joda.time"},
	"LocalTime":      {"java.time", "org.joda.time"},
	"Instant":        {"java.time", "org.joda.time"},
	"OffsetDateTime": {"java.time"},
	"OffsetTime":     {"java.time"},
	"ZonedDateTime":  {"java.time"},
	"YearMonth":      {"java.time", "org.joda.time"},
	"Year":           {"java.time"},
	"Duration":       {"java.time", "org.joda.time"},
	"Period":         {"java.time", "org.joda.time"},
	"UUID":           {"java.util"},
	"URI":            {"java.net"},
	"URL":            {"java.net"},
	// Collections.
	"List":          {"java.util"},
	"Set":           {"java.util"},
	"Collection":    {"java.util"},
	"Iterable":      {"java.lang"},
	"ArrayList":     {"java.util"},
	"LinkedList":    {"java.util"},
	"HashSet":       {"java.util"},
	"LinkedHashSet": {"java.util"},
	"SortedSet":     {"java.util"},
	"TreeSet":       {"java.util"},
	// Holders of a value or nothing.
	"Optional":       {"java.util"},
	"OptionalInt":    {"java.util"},
	"OptionalLong":   {"java.util"},
	"OptionalDouble": {"java.util"},
	// Wrappers that carry the payload of a response.
	"ResponseEntity": {"org.springframework.http"},
	"Mono":           {"reactor.core.publisher"},
	"Flux":           {"reactor.core.publisher"},
	"Response":       {"jakarta.ws.rs.core", "javax.ws.rs.core"},
}

// LibraryPackages returns the packages that declare the class of that simple
// name among the classes of libraryPackages, or nil when it is not one of them.
func LibraryPackages(simple string) []string {
	return slices.Clone(libraryPackages[simple])
}

// IsLibraryClass reports whether a qualified name is the identity of one of the
// classes of libraryPackages. A bare name is not: it is a class of the unnamed
// package, or a name nothing binds.
func IsLibraryClass(identity string) bool {
	simple := lastSegment(identity)
	if simple == identity {
		return false
	}
	return slices.Contains(libraryPackages[simple], strings.TrimSuffix(identity, "."+simple))
}

// Identity returns the qualified name that a class name written in source
// denotes. The name is bound once, where it is written, by the order of JLS
// 6.4.1: a declaration of the same compilation unit, a single-type import, a
// declaration of the same package, the imports on demand (the classes of the
// project and the JDK and library classes of libraryPackages that the package
// supplies), and java.lang. A class of the project, an enum included, answers
// its registry key, an import its path, a name written with its package itself.
//
// A name that nothing in scope binds is a class of the project of that name, as
// a source parsed without its imports can only say, and otherwise stays as
// written. No dependency is asked: the classes of a library are known from the
// table, an import or the name itself.
//
// A source that is no compilation unit at all (see unscoped) has no scope to
// bind a name by: a name that libraryPackages knows is the class it is known for.
//
// Identity is for the name of a class, not for a type spelled with arguments or
// dimensions: ask it for the base name, and for the element of an array.
func (r *TypeResolver) Identity(source parser.Class, name string) string {
	name = strings.TrimSpace(name)
	if name == "" || strings.Contains(name, ".") {
		return name
	}
	if _, keyword := primitiveTypes[name]; keyword {
		return name
	}
	// A type variable of source is no class: it is what the name denotes in the
	// body of the class, before any class of the scope.
	if r.isTypeVariable(source, name) {
		return name
	}
	if key, found := r.inScope(source, name, false); found {
		return key
	}
	if isJavaLangType(name) {
		return "java.lang." + name
	}
	if unscoped(source) {
		if packages, ok := libraryPackages[name]; ok {
			return packages[0] + "." + name
		}
	}
	return r.scopedName(source, name, false)
}

// unscoped reports whether source is no compilation unit: it has no file, no
// package and no imports, as the source of Resolve, which resolves a type
// without any context. Nothing binds a name there, so a name of a library is
// read as the class it is known for.
func unscoped(source parser.Class) bool {
	return source.FilePath == "" && source.Package == "" && len(source.Imports) == 0
}

// IsOptionalType reports whether a type, as written in source, is one of the
// java.util classes that hold a value or nothing.
func (r *TypeResolver) IsOptionalType(source parser.Class, typeName string) bool {
	base, _ := parseGenericType(strings.TrimSpace(typeName))
	return isOptionalType(r.Identity(source, base))
}

// IsUpload reports whether m is a model this resolver answered Spring's
// MultipartFile with. The model is shown as MultipartFile, as a class of the
// project or a name that cannot be resolved that is called so is, so the model
// itself is what tells them apart, not its name.
func (r *TypeResolver) IsUpload(m *model.ObjectModel) bool {
	return m != nil && r.uploads[m]
}

// IsMultipartFile reports whether a handler parameter of the given type is
// uploaded as parts of a multipart request: Spring's MultipartFile, or a
// collection or an array of them, held by a java.util.Optional or not. The
// element is read by its identity, so a class of the project that is called
// MultipartFile is not an upload.
func (r *TypeResolver) IsMultipartFile(source parser.Class, typeName string) bool {
	element := r.UnwrapOptional(source, typeName)
	if inner, ok := strings.CutSuffix(strings.TrimSpace(element), "[]"); ok {
		element = inner
	} else if base, args := parseGenericType(element); len(args) == 1 && r.isCollectionType(source, base) {
		element = args[0]
	}
	base, _ := parseGenericType(strings.TrimSpace(element))
	return r.Identity(source, base) == MultipartFileClass
}

// IsClass reports whether a type, as written in source, is one of the classes
// with the given qualified names.
func (r *TypeResolver) IsClass(source parser.Class, typeName string, qualified ...string) bool {
	base, _ := parseGenericType(strings.TrimSpace(typeName))
	return slices.Contains(qualified, r.Identity(source, base))
}
