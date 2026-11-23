package resolver

import (
	"slices"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

	collector "github.com/tangcent/apilot/api-collector"
	javadoc "github.com/tangcent/apilot/api-collector-java/doc"
	"github.com/tangcent/apilot/api-collector-java/expr"
	"github.com/tangcent/apilot/api-collector-java/parser"
	model "github.com/tangcent/apilot/api-model"
)

// primitiveTypes maps the keyword of a primitive type to the JSON scalar it
// serialises to. A keyword names the same type in every compilation unit, so no
// declaration can hide it.
var primitiveTypes = map[string]string{
	"int":     model.JsonTypeInt,
	"short":   model.JsonTypeInt,
	"long":    model.JsonTypeLong,
	"float":   model.JsonTypeFloat,
	"double":  model.JsonTypeDouble,
	"boolean": model.JsonTypeBoolean,
	"byte":    model.JsonTypeInt,
	"char":    model.JsonTypeString,
	"void":    model.JsonTypeNull,
}

// javaLangScalars maps the simple name of a java.lang value class to the JSON
// scalar it serialises to.
//
// Unlike a keyword, a class name can be hidden by a declaration of the same
// name, so these names are read as the JDK classes only where java.lang is the
// scope that supplies them (see javaLangType). `Object` is java.lang too but
// has no scalar; it is an open object (see isJavaLangType).
var javaLangScalars = map[string]string{
	"Void":      model.JsonTypeNull,
	"String":    model.JsonTypeString,
	"Integer":   model.JsonTypeInt,
	"Short":     model.JsonTypeInt,
	"Long":      model.JsonTypeLong,
	"Float":     model.JsonTypeFloat,
	"Double":    model.JsonTypeDouble,
	"Boolean":   model.JsonTypeBoolean,
	"Number":    model.JsonTypeDouble,
	"Byte":      model.JsonTypeInt,
	"Character": model.JsonTypeString,
	// A CharSequence is text on the wire.
	"CharSequence": model.JsonTypeString,
}

// jdkQualifiedScalars maps a fully qualified JDK value type to the JSON scalar
// it serialises to.
//
// Names here are matched exactly, so short spellings that could belong to a
// project class (`Time`, `URL`) stay resolvable by the class registry: a bare
// name is read as the JDK class only when the source imports it by name or its
// package on demand (see declaredInScope). This is easy-yapi's
// `json.rule.convert[<fqn>]` layer.
var jdkQualifiedScalars = map[string]string{
	"java.sql.Time":         model.JsonTypeString,
	"java.sql.Date":         model.JsonTypeString,
	"java.sql.Timestamp":    model.JsonTypeString,
	"java.net.URL":          model.JsonTypeString,
	"java.net.URI":          model.JsonTypeString,
	"java.util.Date":        model.JsonTypeString,
	"java.util.Calendar":    model.JsonTypeString,
	"java.util.UUID":        model.JsonTypeString,
	"java.math.BigDecimal":  model.JsonTypeDouble,
	"java.math.BigInteger":  model.JsonTypeLong,
	"org.bson.types.ObjectId": model.JsonTypeString,
	// java.util.OptionalInt and its kin hold a number or nothing. Jackson's
	// Jdk8Module, which Spring Boot registers, writes the number or null.
	"java.util.OptionalInt":    model.JsonTypeInt,
	"java.util.OptionalLong":   model.JsonTypeLong,
	"java.util.OptionalDouble": model.JsonTypeDouble,
	// Locale, currency and time zone values: Jackson writes each as a string.
	"java.util.Locale":   model.JsonTypeString,
	"java.util.Currency": model.JsonTypeString,
	"java.util.TimeZone": model.JsonTypeString,
	// The Kotlin standard library's CharSequence is text as well.
	"kotlin.CharSequence": model.JsonTypeString,
}

// javaUtilOptional is the one generic class that holds a value or nothing.
// Jackson's Jdk8Module, which Spring Boot registers, writes the value it
// holds, or null: `Optional<Order>` is `Order`, and a property of that type may
// be left out.
const javaUtilOptional = "java.util.Optional"

// isOptionalType reports whether a platform type name is one of the java.util
// classes that hold a value or nothing.
func isOptionalType(platform string) bool {
	switch platform {
	case javaUtilOptional, "java.util.OptionalInt", "java.util.OptionalLong", "java.util.OptionalDouble":
		return true
	}
	return false
}

// platformEnums lists the constants of the JDK enums the resolver knows without
// reading their declarations. Like the JDK scalars they are known by their
// qualified names only: `Month` is as likely to be a project's enum as the
// JDK's, so a bare name denotes the JDK one only when the source imports it.
var platformEnums = map[string][]string{
	"java.time.DayOfWeek": {
		"MONDAY", "TUESDAY", "WEDNESDAY", "THURSDAY", "FRIDAY", "SATURDAY", "SUNDAY",
	},
	"java.time.Month": {
		"JANUARY", "FEBRUARY", "MARCH", "APRIL", "MAY", "JUNE",
		"JULY", "AUGUST", "SEPTEMBER", "OCTOBER", "NOVEMBER", "DECEMBER",
	},
}

// platformModels maps the qualified name of a library class to the shape it has
// on the wire, for the classes that are not a scalar.
//
// A Jackson tree is free-form: `JsonNode` and `ObjectNode` hold any object,
// `ArrayNode` an array of them. A `MultipartFile` is not JSON at all: Spring
// binds it from a part of a multipart request, never from a body, so nothing
// but its name says what it is. It is shown by its simple name, and the
// resolver remembers the model it answered it with (see IsUpload), which is how
// a bound property of this type is told to be a file.
var platformModels = map[string]func() *model.ObjectModel{
	"com.fasterxml.jackson.databind.JsonNode":        model.EmptyObject,
	"com.fasterxml.jackson.databind.node.ObjectNode": model.EmptyObject,
	"com.fasterxml.jackson.databind.node.ArrayNode": func() *model.ObjectModel {
		return model.ArrayModel(model.EmptyObject())
	},
	MultipartFileClass: func() *model.ObjectModel {
		return model.SingleModel("MultipartFile")
	},
}

// jdkScalarTypes maps the simple name of an unambiguous JDK value type to the
// JSON scalar it serialises to.
//
// Matching is package-insensitive on purpose: `java.time.LocalDateTime` and the
// imported bare `LocalDateTime` must give the same answer. This is easy-yapi's
// IrType.fromJavaType layer, which strips the package and reads the simple name.
var jdkScalarTypes = map[string]string{
	// java.math
	"biginteger": model.JsonTypeLong,
	"bigdecimal": model.JsonTypeDouble,
	// java.time
	"instant":       model.JsonTypeString,
	"localdate":     model.JsonTypeString,
	"localtime":     model.JsonTypeString,
	"localdatetime": model.JsonTypeString,
	"offsettime":    model.JsonTypeString,
	"offsetdatetime": model.JsonTypeString,
	"zoneddatetime": model.JsonTypeString,
	"duration":      model.JsonTypeString,
	"period":        model.JsonTypeString,
	"year":          model.JsonTypeString,
	"yearmonth":     model.JsonTypeString,
	"monthday":      model.JsonTypeString,
	"zoneid":        model.JsonTypeString,
	"zoneoffset":    model.JsonTypeString,
	// java.util
	"date":             model.JsonTypeString,
	"calendar":         model.JsonTypeString,
	"gregoriancalendar": model.JsonTypeString,
	"uuid":             model.JsonTypeString,
	// java.util.concurrent.atomic
	"atomicboolean": model.JsonTypeBoolean,
	"atomicinteger": model.JsonTypeInt,
	"atomiclong":    model.JsonTypeLong,
	"longadder":     model.JsonTypeLong,
}

// jdkValueTypes lists, by their qualified names, the classes of the JDK that
// jdkScalarTypes answers for by their simple names and jdkQualifiedScalars does
// not name. With the classes of jdkQualifiedScalars they are the JDK value types
// an import of their package supplies (see declaredInScope).
var jdkValueTypes = map[string]bool{
	"java.time.Instant":                         true,
	"java.time.LocalDate":                       true,
	"java.time.LocalTime":                       true,
	"java.time.LocalDateTime":                   true,
	"java.time.OffsetTime":                      true,
	"java.time.OffsetDateTime":                  true,
	"java.time.ZonedDateTime":                   true,
	"java.time.Duration":                        true,
	"java.time.Period":                          true,
	"java.time.Year":                            true,
	"java.time.YearMonth":                       true,
	"java.time.MonthDay":                        true,
	"java.time.ZoneId":                          true,
	"java.time.ZoneOffset":                      true,
	"java.util.GregorianCalendar":               true,
	"java.util.concurrent.atomic.AtomicBoolean": true,
	"java.util.concurrent.atomic.AtomicInteger": true,
	"java.util.concurrent.atomic.AtomicLong":    true,
	"java.util.concurrent.atomic.LongAdder":     true,
}

// simpleTypeName reduces a type spelling to its lower-cased simple name:
// `java.time.LocalDateTime` and `LocalDateTime` both become `localdatetime`,
// and the head of `List<Order>` becomes `list`.
func simpleTypeName(rawType string) string {
	name := rawType
	if i := strings.Index(name, "<"); i >= 0 {
		name = name[:i]
	}
	if i := strings.LastIndex(name, "."); i >= 0 {
		name = name[i+1:]
	}
	return strings.ToLower(name)
}

var collectionTypes = map[string]bool{
	"List":          true,
	"ArrayList":     true,
	"Collection":    true,
	"Set":           true,
	"HashSet":       true,
	"LinkedList":    true,
	"Iterable":      true,
	"LinkedHashSet": true,
	"SortedSet":     true,
	"TreeSet":       true,
}

var mapTypes = map[string]bool{
	"Map":           true,
	"HashMap":       true,
	"LinkedHashMap": true,
	"TreeMap":       true,
	"SortedMap":     true,
}

// unqualifiedJdkName strips the package from a name of java.util, java.lang or
// the Kotlin standard library's collections (`java.util.List` is `List`,
// `java.lang.Iterable` is `Iterable`, `kotlin.collections.Map` is `Map`). A name
// of any other package, or of a nested class (`java.util.Map.Entry`), is
// returned as written.
func unqualifiedJdkName(name string) string {
	for _, prefix := range []string{"java.util.", "java.lang.", "kotlin.collections."} {
		if rest, ok := strings.CutPrefix(name, prefix); ok && !strings.Contains(rest, ".") {
			return rest
		}
	}
	return name
}

// jdkClassName returns the simple name of the java.util or java.lang class that
// a type name denotes in source, and name as written when it denotes another
// type.
//
// A name spelled with its package is that class or it is not. A simple name is
// the class only when no declaration in scope hides it: a class of the project
// called `List` or `Iterable` is a class, and so is a single-type import of
// another library's.
func (r *TypeResolver) jdkClassName(source parser.Class, name string) string {
	if simple := unqualifiedJdkName(name); simple != name {
		return simple
	}
	if strings.Contains(name, ".") {
		return name
	}
	if key, declared := r.inScope(source, name, false); declared {
		return unqualifiedJdkName(key)
	}
	return name
}

// isCollectionType reports whether a type name, as written in source, is a
// collection serialised as an array.
func (r *TypeResolver) isCollectionType(source parser.Class, name string) bool {
	return collectionTypes[r.jdkClassName(source, name)]
}

// isMapType reports whether a type name, as written in source, is a map
// serialised as an object.
func (r *TypeResolver) isMapType(source parser.Class, name string) bool {
	return mapTypes[r.jdkClassName(source, name)]
}

// reactiveWrappers maps the identity of a reactor publisher to whether its
// payload is streamed as an array.
//
// A publisher is plumbing, not payload: nothing of `Mono` or `Flux` reaches
// the wire, so the exported schema is the type argument. `Mono<T>` is `T`;
// `Flux<T>` is a stream and serialises as an array of `T`. A class of the
// project that is called Mono is a class.
var reactiveWrappers = map[string]bool{
	"reactor.core.publisher.Mono":  false,
	"reactor.core.publisher.Flux": true,
}

// responseWrappers are the identities of the framework types that carry the
// payload as their single type argument: Spring's `ResponseEntity<Order>` and
// Feign's.
//
// They are plumbing in the same way a publisher is — nothing of the wrapper
// reaches the wire — so the unwrap lives here rather than in each framework
// parser, and a wrapper nested inside another type keeps working.
var responseWrappers = map[string]bool{
	"org.springframework.http.ResponseEntity": true,
}

type DependencyResolver interface {
	ResolveClass(className string) *parser.Class
}

// TypeResolver expands Java type spellings into object models.
//
// Classes are indexed under their qualified name, the package followed by the
// classes a nested class is declared in, so two declarations sharing a simple
// name in different packages or in different classes coexist instead of
// overwriting each other. Which one a simple name denotes is decided per
// compilation unit by resolveName, following JLS 6.5.5.1 and 6.4.1.
type TypeResolver struct {
	classRegistry      map[string]parser.Class // qualified name -> declaration
	bySimpleName       map[string][]string     // simple name -> qualified names, sorted
	byFile             map[string][]string     // compilation unit -> qualified names declared there
	dependencyKeys     map[string]bool         // registry keys of the classes read from dependencies (see registerDependency)
	nestedByName       map[string][]string     // simple name -> qualified names of the nested classes carrying it, scanned or read, sorted
	scannedKeys        map[string]bool         // qualified names of the classes of the scanned sources
	scannedPackages    map[string]bool         // packages that a class of the scanned sources declares
	supertypeOf        map[string]bool         // classes whose header is being read: their member types are out of scope (see inHeader)
	supertypesHeld     map[string]bool         // classes whose supertypes were read from the dependencies (see holdSupertypes)
	allTypeParams      map[string]bool
	resolving          map[string][]string // class -> the types it is being expanded as, innermost last (see isBeingExpanded)
	dependencyResolver DependencyResolver
	unresolved         *collector.UnresolvedSet
	// bindingView is set while ResolveForBinding expands a model attribute.
	// Every enum takes part in that view; the JSON view reads only some (see
	// sees).
	bindingView bool
	// keepBinaryArray is set while a property is resolved whose annotations
	// replace Jackson's serializer for a byte[] or a char[]. Those stay arrays,
	// which is what main exports: the replacement is not a Base64 string, and
	// what it writes instead cannot be read off the source.
	keepBinaryArray bool
	// requiredFields is a collector.RequiredFields mode; empty means default.
	requiredFields string
	// dependencyAnswers holds what dependencyResolver answered for each name,
	// a miss as nil, so that no name is asked for twice.
	dependencyAnswers map[string]*parser.Class
	// shown holds, for a class that a type argument names in full, the name the
	// source wrote for it, so that a type that is displayed or reported reads as
	// the source wrote it (see shortForm).
	shown map[string]string
	// uploads holds the models the resolver answered Spring's MultipartFile with:
	// a single value named MultipartFile, which a class of the project or a name
	// nothing binds that is called the same is not (see IsUpload).
	uploads map[*model.ObjectModel]bool
	// variables holds the models the resolver answered a type variable that nothing
	// binds with: a single value named like the variable, which an enum or a class of
	// the project that is called the same is not (see IsOpenVariable).
	variables map[*model.ObjectModel]bool
}

// SetRequiredFields selects how resolved fields are marked required: one of
// collector.RequiredFieldsDefault or collector.RequiredFieldsConstraints.
func (r *TypeResolver) SetRequiredFields(mode string) {
	r.requiredFields = mode
}

func NewTypeResolver(classes []parser.Class) *TypeResolver {
	r := &TypeResolver{
		classRegistry: make(map[string]parser.Class, len(classes)),
		bySimpleName:  make(map[string][]string),
		byFile:        make(map[string][]string),
		allTypeParams: make(map[string]bool),
		resolving:     make(map[string][]string),

		dependencyKeys:  make(map[string]bool),
		nestedByName:    make(map[string][]string),
		scannedKeys:     make(map[string]bool),
		scannedPackages: make(map[string]bool),
		supertypeOf:     make(map[string]bool),
		supertypesHeld:  make(map[string]bool),
	}
	for _, c := range classes {
		// A local class is in scope in its block only: no name that the resolver
		// looks up denotes it (see parser.Class.Local).
		if !c.Local {
			r.register(c, qualifiedName(c))
		}
	}
	return r
}

// qualifiedName returns the registry key of a declaration: its package, the
// classes it is nested in and its name.
func qualifiedName(c parser.Class) string {
	return c.QualifiedName()
}

// register indexes a declaration of the scanned sources under key.
func (r *TypeResolver) register(c parser.Class, key string) {
	r.classRegistry[key] = c
	r.bySimpleName[c.Name] = appendKey(r.bySimpleName[c.Name], key)
	if c.Outer != "" {
		r.nestedByName[c.Name] = appendKey(r.nestedByName[c.Name], key)
	}
	r.scannedKeys[key] = true
	r.scannedPackages[c.Package] = true
	if c.FilePath != "" {
		r.byFile[c.FilePath] = appendKey(r.byFile[c.FilePath], key)
	}
	for _, tp := range c.TypeParameters {
		r.allTypeParams[tp] = true
	}
}

// registerDependency indexes a class read from the dependencies under its
// qualified name, so a later lookup of that name is answered from the registry.
//
// The class is not entered under its simple name: a name that nothing in scope
// declares falls back to a declaration of the scanned sources carrying it (see
// declaredElsewhere), and a class a library declares must not stand in for it, nor
// for a class that another source reaches under the same simple name. The key is
// remembered so that a class of the dependencies can be told from one of the
// scanned sources (see declares).
//
// The source a class was read from declares the classes nested in it as well,
// and a name that denotes one of them is not asked of the dependencies again:
// the indexer holds top-level classes only. A resolver that kept them says so
// through siblingReader, and they are registered under their own qualified
// names with the class, a local class excepted. Another top-level class of the
// same source is not: it is asked for on its own, as it always was.
func (r *TypeResolver) registerDependency(c parser.Class, key string) {
	r.indexDependency(c, key)
	reader, ok := r.dependencyResolver.(siblingReader)
	if !ok {
		return
	}
	for _, sibling := range reader.Siblings(key) {
		// The declarations already held win: a sibling that the scanned sources
		// or an earlier answer declare under the same name is not replaced.
		if siblingKey := qualifiedName(sibling); siblingKey != "" && !sibling.Local {
			if _, held := r.classRegistry[siblingKey]; !held {
				r.indexDependency(sibling, siblingKey)
			}
		}
	}
}

// siblingReader is implemented by a dependency resolver that keeps the classes
// nested in a class when it reads the class from a source.
type siblingReader interface {
	// Siblings returns the classes nested in the class className was read from,
	// or, for a nested class, those of the class that declares it and that class
	// too. Each carries its package and Outer, so it can be registered under its
	// qualified name.
	Siblings(className string) []parser.Class
}

// indexDependency holds a class read from the dependencies under key.
func (r *TypeResolver) indexDependency(c parser.Class, key string) {
	r.classRegistry[key] = c
	r.dependencyKeys[key] = true
	if c.Outer != "" {
		r.nestedByName[c.Name] = appendKey(r.nestedByName[c.Name], key)
	}
	for _, tp := range c.TypeParameters {
		r.allTypeParams[tp] = true
	}
}

func appendKey(keys []string, key string) []string {
	for _, existing := range keys {
		if existing == key {
			return keys
		}
	}
	keys = append(keys, key)
	sort.Strings(keys)
	return keys
}

// Class returns the declaration a type name denotes as seen from source.
// It applies the same precedence as Resolve, without expanding the fields, and
// reads the class from the dependencies when the resolver does not hold it yet:
// the answer is the same whether or not something read the class before.
func (r *TypeResolver) Class(source parser.Class, name string) (parser.Class, bool) {
	return r.classNamed(r.resolveName(source, name))
}

// classNamed returns the class registered under key if the current view sees it.
// A class that only a dependency declares is read from it, once (see
// readQualified).
func (r *TypeResolver) classNamed(key string) (parser.Class, bool) {
	if c, found := r.classRegistry[key]; found {
		return c, r.sees(c)
	}
	if class, _ := r.readQualified(key); class != nil {
		return *class, r.sees(*class)
	}
	return parser.Class{}, false
}

// readQualified reads the class a qualified name denotes from the dependencies
// and registers it, or returns nil, with the key it is held under. A name that no scope qualified is not asked
// for: the index knows a class by its qualified name only. Nor is a class of the
// JDK or of the Kotlin standard library, which are reached as scalars and
// collections and as supertypes (Serializable) far more often than as a class
// worth reading. A class nested in a class of the scanned sources is one of the
// project's: the dependencies would read the class again from some release of
// it, and answer with that one's.
func (r *TypeResolver) readQualified(key string) (*parser.Class, string) {
	if r.dependencyResolver == nil || !strings.Contains(key, ".") || isPlatformName(key) || r.memberOfScanned(key) {
		return nil, ""
	}
	return r.readDependency(key)
}

// Supertype returns the declaration that name denotes when class writes it in
// its extends or implements clause. Class is the lookup for a name written in
// the body of a class; the clause is outside the body, where the class's own
// member types are not in scope (JLS 6.3), so `Base` in
// `class Child extends Base` is the class that Child's file imports, not a
// `Child.Base` that Child declares.
//
// The name goes through the same scope as any other, the dependencies included:
// a supertype that only a dependency declares is read from it, whether it is
// imported, imported on demand, a neighbour of class in its package, or written
// in full.
func (r *TypeResolver) Supertype(class parser.Class, name string) (parser.Class, bool) {
	super, _, found := r.supertypeClass(class, qualifiedName(class), name)
	return super, found && r.sees(super)
}

// supertypeClass returns the class that name denotes when c, registered under
// key, writes it in its extends or implements clause, with the key it is held
// under. A class the resolver does not hold is read from the dependencies.
func (r *TypeResolver) supertypeClass(c parser.Class, key, name string) (parser.Class, string, bool) {
	superKey := r.supertypeKey(c, key, name)
	if held, found := r.classRegistry[superKey]; found {
		return held, superKey, true
	}
	// A class of the JDK or of the project is not asked of the dependencies, nor is
	// a name that no scope qualified.
	if r.dependencyResolver == nil || !strings.Contains(superKey, ".") || isPlatformName(superKey) ||
		r.isRegisteredEnum(superKey) || r.memberOfScanned(superKey) {
		return parser.Class{}, "", false
	}
	class, heldAs := r.readDependency(superKey)
	if class == nil {
		return parser.Class{}, "", false
	}
	return *class, heldAs, true
}

// holdSupertypes reads the superclass and the interfaces of the class registered
// under key from the dependencies when the resolver does not hold them yet, and
// theirs in turn. A class inherits the member types of its supertypes, and a name
// in its body denotes one of them before anything the file declares or imports
// (JLS 8.5, 9.5), so they have to be known before such a name is looked up: the
// member types a library declares are registered with its class, and a name is
// looked for among the member types that something holds (see memberTypeInScope).
// A class is held once; without dependencies there is nothing to read.
func (r *TypeResolver) holdSupertypes(c parser.Class, key string) {
	if r.dependencyResolver == nil || r.supertypesHeld[key] {
		return
	}
	r.supertypesHeld[key] = true
	for _, super := range supertypeNames(c) {
		if superClass, superKey, found := r.supertypeClass(c, key, super); found {
			r.holdSupertypes(superClass, superKey)
		}
	}
}

// supertypeNames lists the types that c names in its extends and implements
// clauses, the superclass first.
func supertypeNames(c parser.Class) []string {
	names := make([]string, 0, 1+len(c.Interfaces))
	for _, name := range append([]string{c.SuperClass}, c.Interfaces...) {
		if name = strings.TrimSpace(name); name != "" {
			names = append(names, name)
		}
	}
	return names
}

// sees reports whether the current view takes a declaration into account.
//
// The binding view reads every enum: Spring converts a request parameter to an
// enum by the name of a constant, so the constants are the values the parameter
// accepts. The JSON view reads an enum that Jackson writes as the name of its
// constant, so that the field is a string of those constants. It leaves out an
// enum Jackson writes in another form (see writesConstantName): the value such
// a field takes cannot be told from the source, so the JSON view exports that
// enum as it does any type whose declaration was not read.
func (r *TypeResolver) sees(c parser.Class) bool {
	return r.bindingView || !c.IsEnum || writesConstantName(c)
}

// writesConstantName reports whether Jackson writes an enum as the name of its
// constant. It does not when the enum declares @JsonValue, which writes the
// value of a member instead, nor when @JsonFormat gives it the shape of an
// object or a number.
func writesConstantName(enum parser.Class) bool {
	if enum.EnumJsonValue {
		return false
	}
	for _, ann := range enum.Annotations {
		if lastSegment(ann.Name) != "JsonFormat" {
			continue
		}
		switch lastSegment(strings.TrimSpace(ann.Params["shape"])) {
		case "OBJECT", "NUMBER", "NUMBER_INT", "NUMBER_FLOAT":
			return false
		}
	}
	return true
}

// lookup returns the declaration registered under key, if the current view
// sees it.
func (r *TypeResolver) lookup(key string) (parser.Class, bool) {
	c, found := r.classRegistry[key]
	if !found || !r.sees(c) {
		return parser.Class{}, false
	}
	return c, true
}

// isRegisteredEnum reports whether key names an enum the resolver already
// holds, whether or not the current view sees it.
func (r *TypeResolver) isRegisteredEnum(key string) bool {
	c, found := r.classRegistry[key]
	return found && c.IsEnum
}

// enterBindingView switches to the binding view and returns the function that
// restores the previous view.
func (r *TypeResolver) enterBindingView() func() {
	previous := r.bindingView
	r.bindingView = true
	return func() { r.bindingView = previous }
}

// EnumConstants returns the constants of the enum a type name denotes as seen
// from source, and whether it denotes an enum of the scanned sources at all.
// Every enum takes part in the binding view, so the name is looked up there.
func (r *TypeResolver) EnumConstants(source parser.Class, name string) ([]string, bool) {
	c, found := r.enumClass(source, name)
	if !found {
		return nil, false
	}
	return c.EnumConstants, true
}

// enumClass returns the enum a type name denotes as seen from source.
func (r *TypeResolver) enumClass(source parser.Class, name string) (parser.Class, bool) {
	defer r.enterBindingView()()
	c, found := r.Class(source, strings.TrimSpace(name))
	if !found || !c.IsEnum {
		return parser.Class{}, false
	}
	return c, true
}

// EnumOptions lists the constants of an enum as the values a field takes,
// each described by the constant's doc comment.
func EnumOptions(enum parser.Class) []model.FieldOption {
	options := make([]model.FieldOption, 0, len(enum.EnumConstants))
	for i, constant := range enum.EnumConstants {
		option := model.FieldOption{Value: constant}
		if i < len(enum.EnumConstantDocs) {
			option.Desc = enum.EnumConstantDocs[i]
		}
		options = append(options, option)
	}
	return options
}

// seeOptions returns the values a field documented with `@see SomeEnum`
// takes: the constants of that enum, as EasyYapi reads them. A field such as
// `String state` whose doc points at the enum holds one of its constant
// names. A reference to a member, `SomeEnum#code`, names another value set
// and is left alone, as is a reference to anything but an enum.
func (r *TypeResolver) seeOptions(source parser.Class, rawDoc string) []model.FieldOption {
	for _, ref := range javadoc.SeeReferences(rawDoc) {
		if strings.Contains(ref, "#") {
			continue
		}
		if enum, ok := r.seeEnum(source, ref); ok {
			return EnumOptions(enum)
		}
	}
	return nil
}

// seeEnum returns the enum a `@see` reference names as seen from source. The name
// goes through the scope of source first (see Class). When no tier of the scope
// declares a simple name, which is how an author writes a reference to an enum of
// another package without importing it, the one enum of the scanned sources that
// carries that name is taken; with two there is nothing to say which is meant. A
// field or a parameter written so names no enum: this leniency is the doc
// comment's alone.
//
// An enum the scanned sources do not declare is read from the project's
// dependencies: the enums such a reference names usually live in a shared model
// library. Neither a class of the dependencies nor one of the JDK is taken for a
// name the scope does not declare.
func (r *TypeResolver) seeEnum(source parser.Class, ref string) (parser.Class, bool) {
	ref = strings.TrimSpace(ref)
	if !strings.Contains(ref, ".") {
		if _, found := r.inScope(source, ref, true); !found {
			return r.onlyEnumNamed(ref)
		}
	}
	return r.enumClass(source, ref)
}

// onlyEnumNamed returns the enum of the scanned sources that carries a simple
// name, when there is exactly one.
func (r *TypeResolver) onlyEnumNamed(name string) (parser.Class, bool) {
	var found parser.Class
	count := 0
	for _, key := range r.bySimpleName[name] {
		if c := r.classRegistry[key]; c.IsEnum {
			found = c
			count++
		}
	}
	return found, count == 1
}

// dependencyClass returns the class the dependency resolver reads for name,
// or nil when it has none. Each name is asked for once.
func (r *TypeResolver) dependencyClass(name string) *parser.Class {
	if class, asked := r.dependencyAnswers[name]; asked {
		return class
	}
	class := r.dependencyResolver.ResolveClass(name)
	if r.dependencyAnswers == nil {
		r.dependencyAnswers = make(map[string]*parser.Class)
	}
	r.dependencyAnswers[name] = class
	return class
}

// readDependency asks the dependencies for the class name denotes, registers it
// and returns it with the key it is held under. A name with a `$` is asked for as
// written: the dependency resolver reads it as the binary name of a nested class
// and as the name of a top-level class, whichever it finds. A resolver that does
// not tell the two apart is asked for the binary name too.
func (r *TypeResolver) readDependency(name string) (*parser.Class, string) {
	class := r.dependencyClass(name)
	if class == nil {
		if canonical := canonicalName(name); canonical != name {
			class, name = r.dependencyClass(canonical), canonical
		}
	}
	if class == nil {
		return nil, ""
	}
	key := heldKey(*class, name)
	r.registerDependency(*class, key)
	return class, key
}

// fieldEnum returns the enum whose constants a field takes, which is the enum
// its type names. The JSON view looks through a collection to the enum it
// holds, and through a type parameter to the enum it is bound to, because the
// model of the field does: its items are strings of the constants. The values
// of a map are strings of them as well, but the map lists none, as EasyYapi
// does: a constraint on the map is not one on its values. The binding view reads
// the same types: Spring binds each element of a collection or an array as it binds
// a single value.
//
// The field type has been resolved by then, so an enum of the dependencies is
// registered.
func (r *TypeResolver) fieldEnum(source parser.Class, typeName string, bindings map[string]string) (parser.Class, bool) {
	return r.valueEnum(source, typeName, bindings)
}

// valueEnum follows a type the way ResolveFrom does to the enum it ends in.
func (r *TypeResolver) valueEnum(source parser.Class, typeName string, bindings map[string]string) (parser.Class, bool) {
	typeName = strings.TrimSpace(typeName)
	if bound, ok := bindings[typeName]; ok {
		return r.valueEnum(source, bound, nil)
	}
	if isWildcard(typeName) {
		if bound, ok := wildcardUpperBound(typeName); ok {
			return r.valueEnum(source, bound, bindings)
		}
		return parser.Class{}, false
	}
	// An array holds its elements as a collection does (see ResolveFrom).
	if element, ok := strings.CutSuffix(typeName, "[]"); ok {
		return r.valueEnum(source, element, bindings)
	}
	// A type variable that nothing binds is no enum, whatever class carries its
	// name (see ResolveFrom).
	if _, marked := variableName(typeName); marked || r.isTypeVariable(source, typeName) {
		return parser.Class{}, false
	}
	base, args := parseGenericType(typeName)
	switch {
	case base == typeName:
		c, found := r.lookup(r.resolveName(source, typeName))
		return c, found && c.IsEnum
	case r.isCollectionType(source, base) && len(args) > 0:
		return r.valueEnum(source, args[0], bindings)
	}
	return parser.Class{}, false
}

// ResolveForBinding expands a type the way Spring's data binder sees it when
// it binds a model attribute from request parameters, rather than the way
// Jackson writes it: getter-only properties cannot be bound, so they are left
// out, and an enum is a string holding one of its constants.
func (r *TypeResolver) ResolveForBinding(source parser.Class, rawType string, typeBindings map[string]string) *model.ObjectModel {
	defer r.enterBindingView()()
	return r.ResolveFrom(source, rawType, typeBindings)
}

// resolveName maps a type name as written in source to the registry key it
// denotes, following JLS 6.5.5.1 and 6.4.1 (see scopedName). It is the one lookup
// of a name: every path that turns a name into a class goes through it, and
// through the one cache of what the dependencies answered, so the answer does
// not depend on which path came first.
//
// The result doubles as the name to keep resolving with, so an import of a
// type whose body is not among the scanned sources still selects the imported
// type instead of a same-named class that happens to sit elsewhere in the
// project. A name that nothing in the scope of source declares among the classes
// held may still be a class of a library: one the package of source declares, or
// one that an import-on-demand brings in, as in `import com.example.model.*`.
// Such a class is asked of the dependencies by its qualified name (see declares).
func (r *TypeResolver) resolveName(source parser.Class, name string) string {
	return r.scopedName(source, name, true)
}

// scopedName is the lookup behind resolveName. It walks the scope of source tier
// by tier, as JLS 6.4.1 orders it, and the first tier that
// declares the name answers:
//
//  0. the member types of source and of the classes around it, declared or
//     inherited from a supertype (JLS 8.5), the innermost class first;
//  1. the top-level classes of the compilation unit of source;
//  2. a single-type import;
//  3. the package of source;
//  4. the imports on demand, in the order they are written, where an import of a
//     class names the types nested in it.
//
// Every kind of declaration takes part in a tier at the same place: a class of
// the scanned sources, a class already read from the dependencies, and a class
// of the dependencies not read yet, which is asked for when ask is set (and a
// resolver without dependencies has none to ask). What a
// name denotes therefore does not depend on where its declaration comes from,
// nor on what the resolver read before, nor on whether source is a class of the
// project or one read from a library. The one exception is the split package
// (see declares).
//
// Only a name that no tier declares falls back to the declaration of the scanned
// sources that carries the simple name wherever it sits (see declaredElsewhere):
// the last resort of sources parsed without scope information, which never takes
// the place of a class that a tier would have named, nor answers for a source that
// imports a package on demand which the project does not hold (see
// mayComeFromAnUnscannedPackage). A class read from the
// dependencies answers only through a tier, never by its simple name alone, so
// that a source which cannot reach it does not get it because another source read
// it first. And a class read from the dependencies is not given the fallback at
// all: a library was built without the project, so no name it writes can be a
// class of the project, and a name that no tier declares (a class nested in it,
// say) stays unresolved rather than naming a project class that shares it.
func (r *TypeResolver) scopedName(source parser.Class, name string, ask bool) string {
	if name == "" {
		return name
	}
	if canonical := canonicalName(name); canonical != name {
		return r.dollarName(source, name, canonical, ask)
	}
	return r.scopedPlain(source, name, ask)
}

// scopedPlain is scopedName for a name that is read as written: a dotted name is
// a path to a member type (see resolveQualified), any other the simple name of a
// class in scope.
func (r *TypeResolver) scopedPlain(source parser.Class, name string, ask bool) string {
	if strings.Contains(name, ".") {
		return r.resolveQualified(source, name, func(s parser.Class, head string) string {
			return r.scopedName(s, head, ask)
		})
	}
	if key, found := r.inScope(source, name, ask); found {
		return key
	}
	// java.lang is the last scope of every compilation unit, so a name it
	// declares is declared in scope: the fallback below, which exists for a name
	// that nothing declares, must not offer a project class instead.
	if isJavaLangType(name) {
		return "java.lang." + name
	}
	if r.isDependencySource(source) || r.mayComeFromAnUnscannedPackage(source) {
		return name
	}
	if key, found := r.declaredElsewhere(name); found {
		return key
	}
	return name
}

// mayComeFromAnUnscannedPackage reports whether source imports a package on demand
// that the scanned sources do not hold. A name that no tier declared may be the
// name of a class of that package, one that the dependencies are not able to say:
// the class of the project that shares the name is not the one the source means,
// and the fallback to it is a guess that the source has no scope information, which
// an import says it has.
func (r *TypeResolver) mayComeFromAnUnscannedPackage(source parser.Class) bool {
	for _, imp := range source.Imports {
		if imp.OnDemand && !imp.Static && !r.scannedPackages[imp.Path] && !r.scannedKeys[imp.Path] {
			return true
		}
	}
	return false
}

// isDependencySource reports whether source is a class that was read from the
// dependencies, as against a class of the scanned sources or no class at all.
func (r *TypeResolver) isDependencySource(source parser.Class) bool {
	return source.Name != "" && r.dependencyKeys[qualifiedName(source)]
}

// dollarName resolves a name with a `$` in it. The `$` is the separator of a
// binary name, `Outer$Inner`, or a character of an identifier, `Price$Value`, and
// the name alone does not say which. Written by hand it is a character of the name
// (JLS 3.8): `Price$Value` is the class of that name, whatever class `Price.Value`
// is. Only bytecode spells a nested class with the `$` of its binary name, and so
// does a source that a decompiler printed from it.
//
// A name is read as written first, in the scope of source, and as the binary name,
// `canonical` (see canonicalName), when that finds no class. The source of a
// decompiled class reads it the other way round: as the binary name first, which
// costs no call for a spelling that nothing is named so, and as written when that
// finds no class. A single-type import that spells the name with the `$` says the
// class is named so, and the name is read as written only.
//
// A name that neither reading finds a class for is returned as it was written,
// not as the binary name, so that it reaches the dependencies in the spelling the
// source used: only the dependency resolver can say whether `lib.Price$Value` is
// the class Value nested in Price or a class of that name.
func (r *TypeResolver) dollarName(source parser.Class, name, canonical string, ask bool) string {
	imported := importedAsWritten(source, name)
	spellings := []string{name, canonical}
	switch {
	case imported:
		spellings = []string{name}
	case source.Decompiled:
		spellings = []string{canonical, name}
	}
	keys := make(map[string]string, 2)
	for _, spelling := range spellings {
		key := r.scopedPlain(source, spelling, ask)
		if r.isDeclared(key) {
			return key
		}
		// A key that no tier read a class for, because it is written in full or a
		// single-type import names it, is asked of the dependencies here: whether
		// the class is read must not depend on what was read before.
		if _, held := r.readNamed(key, ask); held != "" {
			return held
		}
		keys[spelling] = key
	}
	if binary := keys[canonical]; binary != "" && binary != canonical {
		// The name of the class that encloses it was found: the path to the member
		// is that class's key followed by the name, to be asked for.
		return binary
	}
	return keys[name]
}

// readNamed reads the class that a key names from the dependencies, when the key is
// a name that a package qualifies, and returns it with the key it is held under.
func (r *TypeResolver) readNamed(key string, ask bool) (*parser.Class, string) {
	if first, _ := utf8.DecodeRuneInString(key); !ask || !unicode.IsLower(first) {
		return nil, ""
	}
	return r.readQualified(key)
}

// importedAsWritten reports whether source imports a class by a name that spells a
// segment of name with its `$`: the import says that the class is named so.
func importedAsWritten(source parser.Class, name string) bool {
	for _, segment := range strings.Split(name, ".") {
		if !strings.Contains(segment, "$") {
			continue
		}
		for _, imp := range source.Imports {
			if !imp.Static && !imp.OnDemand && lastSegment(imp.Path) == segment {
				return true
			}
		}
	}
	return false
}

// canonicalName spells the name of a nested class the way source does. A
// decompiler may print the class Inner of Outer as `Outer$Inner`, the binary
// name, and the Kotlin compiler names it so; a name written with `.` is the
// one the registry holds. A `$` that the name does not use between two names, as
// in `$Gson$Types` or `Outer$1`, is not a separator and the name is left alone.
func canonicalName(name string) string {
	if !strings.Contains(name, "$") {
		return name
	}
	parts := strings.Split(name, "$")
	for _, part := range parts {
		if part == "" || strings.HasSuffix(part, ".") {
			return name
		}
		if first, _ := utf8.DecodeRuneInString(part); !unicode.IsLetter(first) && first != '_' {
			return name
		}
	}
	return strings.Join(parts, ".")
}

// resolveQualified resolves a dotted name. A name that starts with a package, as
// com.example.Order does, denotes itself. A name that starts with a class,
// Outer.Inner, denotes the member type Inner of the class Outer stands for in
// source, however Outer is reached: resolveHead says which class that is, and
// the registry key of the member is the key of the class followed by its name.
// A package starts with a lower-case letter and a class does not, which is as
// far as the spelling goes: a name whose first part resolves to nothing denotes
// itself.
func (r *TypeResolver) resolveQualified(source parser.Class, name string, resolveHead func(parser.Class, string) string) string {
	head, rest, _ := strings.Cut(name, ".")
	if first, _ := utf8.DecodeRuneInString(head); head == "" || unicode.IsLower(first) {
		return name
	}
	key := resolveHead(source, head)
	if key == head {
		return name
	}
	for _, segment := range strings.Split(rest, ".") {
		key = r.memberKey(key, segment)
	}
	return key
}

// memberKey returns the registry key of the member type that segment names in
// the class registered under key, declared there or inherited from a
// supertype. A member that nothing holds is spelled as a nested declaration
// would be, so that the dependencies can be asked for it.
func (r *TypeResolver) memberKey(key, segment string) string {
	if class, found := r.classRegistry[key]; found {
		if member, found := r.memberOf(class, key, segment, map[string]bool{}); found {
			return member
		}
	}
	return key + "." + segment
}

// memberTypeInScope returns the registry key of the member type that name
// denotes inside source: one declared or inherited by the class that declares
// source, or by a class around it, the innermost first (JLS 6.4.1 and 8.5). A
// member type shadows everything the compilation unit declares or imports under
// the same name, so it is looked for first.
//
// A declaration shadows by being there, whether or not the current view reads
// it: an enum nested in a class is what its name denotes inside that class, and
// the top-level class of the same name is not.
func (r *TypeResolver) memberTypeInScope(source parser.Class, name string) (string, bool) {
	if source.Name == "" {
		return "", false
	}
	// The supertypes of the classes around source are read first, so that a member
	// type that only a library declares is held. A member type is a nested
	// declaration that something holds: without one of that name there is nothing
	// to find, and most names have none.
	r.holdMemberScope(source)
	if len(r.nestedByName[name]) == 0 {
		return "", false
	}
	for c := source; ; c = r.enclosing(c) {
		key := qualifiedName(c)
		// The name of a supertype in the header of a class is outside the body
		// of the class, where its own member types are not in scope.
		if !r.supertypeOf[key] {
			if member, found := r.memberOf(c, key, name, map[string]bool{}); found {
				return member, true
			}
		}
		if c.Outer == "" {
			return "", false
		}
	}
}

// holdMemberScope holds the supertypes of source and of the classes around it,
// the classes whose member types are in scope in the body of source, and, in the
// header of one of them, those of the classes around it.
func (r *TypeResolver) holdMemberScope(source parser.Class) {
	if r.dependencyResolver == nil {
		return
	}
	for c := source; ; c = r.enclosing(c) {
		if key := qualifiedName(c); !r.supertypeOf[key] {
			r.holdSupertypes(c, key)
		}
		if c.Outer == "" {
			return
		}
	}
}

// enclosing returns the class that a nested class is declared in: its
// declaration when the resolver holds it, otherwise a declaration that carries
// the name and the scope of the nested class, which is all that finding the
// member types of the enclosing class around it needs.
func (r *TypeResolver) enclosing(c parser.Class) parser.Class {
	if held, found := r.classRegistry[qualify(c.Package, c.Outer)]; found {
		return held
	}
	outer := parser.Class{Name: c.Outer, Package: c.Package, FilePath: c.FilePath, Imports: c.Imports, Decompiled: c.Decompiled}
	if dot := strings.LastIndex(c.Outer, "."); dot >= 0 {
		outer.Name, outer.Outer = c.Outer[dot+1:], c.Outer[:dot]
	}
	return outer
}

// memberOf returns the member type named name that the class c, registered
// under key, declares or inherits: the key of a nested declaration, or the key
// of a member of a superclass or an interface. seen guards a class that names
// itself among its supertypes.
func (r *TypeResolver) memberOf(c parser.Class, key, name string, seen map[string]bool) (string, bool) {
	if seen[key] {
		return "", false
	}
	seen[key] = true
	if member := key + "." + name; r.isDeclared(member) {
		return member, true
	}
	if r.supertypeOf[key] {
		return "", false
	}
	for _, super := range supertypeNames(c) {
		if superClass, superKey, found := r.supertypeClass(c, key, super); found {
			if member, found := r.memberOf(superClass, superKey, name, seen); found {
				return member, true
			}
		}
	}
	return "", false
}

// supertypeKey resolves the name of a supertype of c, registered under key, as
// c's header spells it: in the scope around c, not in c.
func (r *TypeResolver) supertypeKey(c parser.Class, key, name string) string {
	var resolved string
	r.inHeader(key, func() { resolved = r.resolveName(c, name) })
	return resolved
}

// inHeader runs f while the header of the class registered under key is being
// read: its extends and implements clauses, and what they carry, a type
// argument included. The member types of the class, declared or inherited, are
// out of scope there; those of the classes around it are not.
func (r *TypeResolver) inHeader(key string, f func()) {
	if !r.supertypeOf[key] {
		r.supertypeOf[key] = true
		defer delete(r.supertypeOf, key)
	}
	f()
}

// isDeclared reports whether the resolver holds a declaration under key,
// whatever the current view reads of it.
func (r *TypeResolver) isDeclared(key string) bool {
	_, found := r.classRegistry[key]
	return found
}

// memberOfScanned reports whether key names a member of a class of the scanned
// sources, as com.example.Order.Item does. Such a class holds the types nested
// in it: one that is not registered is not there, and the dependencies are not
// asked for it. They would read the class of the project again, from some
// installed release of it, and answer with that one's.
func (r *TypeResolver) memberOfScanned(key string) bool {
	dot := strings.LastIndex(key, ".")
	return dot >= 0 && r.scannedKeys[key[:dot]]
}

// inScope returns the registry key an unqualified name denotes through the
// scope of source (the tiers of scopedName), and whether any tier declares it.
//
// A class that the current view does not see, an enum of the JSON view, still
// answers: it is what the name denotes, so no later tier can supply another
// class for it. The caller finds the class not seen and exports it as it does
// any type whose declaration was not read.
func (r *TypeResolver) inScope(source parser.Class, name string, ask bool) (string, bool) {
	// 0. Member type: a class nested in the class that declares source, in a class
	// around it, or in a supertype of either.
	if key, found := r.memberTypeInScope(source, name); found {
		return key, true
	}

	// 1. Same compilation unit: a top-level class declared in the same file. A
	// class nested in one of them is named through it or from inside it.
	for _, key := range r.byFile[source.FilePath] {
		if c, found := r.classRegistry[key]; found && c.Name == name && c.Outer == "" {
			return key, true
		}
	}

	// 2. Single-type import. The import names the type even when its body is
	// not in the scanned sources, so it is returned without a registry check.
	for _, imp := range source.Imports {
		if imp.Static || imp.OnDemand {
			continue
		}
		if lastSegment(imp.Path) == name {
			return imp.Path, true
		}
	}

	ask = ask && r.dependencyResolver != nil && r.mayBeDependencyClass(source, name)

	// 3. Same package. The unnamed package is a package too (JLS 7.4.2): a class
	// declared in it is in scope for the other classes of it. It has no name to
	// ask the dependencies for, so only the classes held answer.
	if source.Package != "" {
		if key, found := r.declares(source, source.Package+"."+name, ask); found {
			return key, true
		}
	} else if class, held := r.classRegistry[name]; held && class.Outer == "" {
		return name, true
	}

	// 4. Import-on-demand. An import of a class (`import com.example.Outer.*`)
	// brings in the types nested in that class, which are keyed by the class
	// followed by the name.
	for _, imp := range source.Imports {
		if imp.Static || !imp.OnDemand {
			continue
		}
		if key, found := r.declares(source, imp.Path+"."+name, ask); found {
			return key, true
		}
		if key, found := r.inheritedMember(imp.Path, name, ask); found {
			return key, true
		}
	}

	// An import on demand of a package of the JDK supplies the JDK types the
	// resolver answers for itself, as it supplies a class of the project, once no
	// declaration of the project has.
	for _, imp := range source.Imports {
		if imp.Static || !imp.OnDemand {
			continue
		}
		if key := imp.Path + "." + name; isPlatformType(key) {
			return key, true
		}
	}

	return "", false
}

// inheritedMember returns the member type name that the class path stands for
// inherits from a supertype. An import on demand of a class brings in its member
// types, declared or inherited (JLS 7.5.2); the declared ones were looked for
// already.
func (r *TypeResolver) inheritedMember(path, name string, ask bool) (string, bool) {
	class, key, found := r.importedClass(path, ask)
	if !found {
		return "", false
	}
	r.holdSupertypes(class, key)
	if len(r.nestedByName[name]) == 0 {
		return "", false
	}
	return r.memberOf(class, key, name, map[string]bool{})
}

// importedClass returns the class that an import on demand names, if it names
// one: a class the resolver holds, or, when ask is set, one that the dependencies
// read for it. A package is not asked for as if it were a class: as far as the
// spelling goes, the name of a class starts with a capital letter and the name of
// a package does not.
func (r *TypeResolver) importedClass(path string, ask bool) (parser.Class, string, bool) {
	if class, found := r.classRegistry[path]; found {
		return class, path, true
	}
	if !ask || r.dependencyResolver == nil {
		return parser.Class{}, "", false
	}
	if first, _ := utf8.DecodeRuneInString(lastSegment(path)); !unicode.IsUpper(first) {
		return parser.Class{}, "", false
	}
	class, key := r.readQualified(path)
	if class == nil {
		return parser.Class{}, "", false
	}
	return *class, key, true
}

// declares reports whether a class stands under key, as seen from source: one of
// the scanned sources, one already read from the dependencies, or, when ask is
// set, one the dependencies hold, which is then read and registered (see
// registerDependency) so that it is asked for once and answers later lookups from
// the registry. The JDK and the Kotlin standard library are not asked: they are
// reached as scalars and collections, and every ask is a call to the indexer.
//
// A source of the project does not see a class of the dependencies in a package
// that the scanned sources declare, whether or not the class has been read: the
// project owns that package, and a name its classes do not carry is taken not to
// be in it. This is the one tier that is not complete, and it is deliberate. The
// same-package candidate comes before every import on demand, so completing it
// asks the dependencies for `pkg.Name` for each name that an import on demand
// supplies, once per package of the project. Measured on the export of one
// service, that took the calls to the indexer from 88 to 277, and each call is a
// process. The price is a split package, a package that the project and a
// dependency both declare: a class of the dependency in it is not found by its
// simple name from a source of the project, which can still spell it in full. A
// class read from a library asks for every package, its own included.
//
// A class the resolver holds has its member types registered with it (see
// registerDependency), so a name under a held class that is not registered is not
// a member: it is not asked for, which would only read the owner again.
//
// It returns the key the class is held under, which is the key asked for unless
// the name has a `$`: `lib.Price$Value` may be the binary name of the class Value
// nested in Price, which the dependencies answer with and which is held under
// `lib.Price.Value`.
func (r *TypeResolver) declares(source parser.Class, key string, ask bool) (string, bool) {
	splitPackage := r.isScannedSource(source) && r.scannedPackages[packageOf(key)]
	if _, found := r.classRegistry[key]; found {
		return key, !splitPackage || !r.dependencyKeys[key]
	}
	if !ask || splitPackage || isPlatformName(key) || r.isDeclared(packageOf(key)) {
		return "", false
	}
	class := r.dependencyClass(key)
	if class == nil {
		return "", false
	}
	held := heldKey(*class, key)
	r.registerDependency(*class, held)
	return held, true
}

// heldKey returns the key a class that the dependencies answered for name is held
// under: the key of the class itself when name is its binary name, name otherwise.
func heldKey(class parser.Class, name string) string {
	if canonical := canonicalName(name); canonical != name && qualifiedName(class) == canonical {
		return canonical
	}
	return name
}

// isScannedSource reports whether source is a compilation unit of the scanned
// sources, as against a class read from a library, whose file is not one of them.
func (r *TypeResolver) isScannedSource(source parser.Class) bool {
	return len(r.byFile[source.FilePath]) > 0
}

// packageOf returns the package part of a qualified name.
func packageOf(name string) string {
	if i := strings.LastIndex(name, "."); i >= 0 {
		return name[:i]
	}
	return ""
}

// declaredElsewhere returns the declaration of the scanned sources that carries
// a simple name, wherever it sits. Nothing in scope declared the name: sources
// parsed without import information (a single file) can only offer the
// declarations themselves, so accept the only one — or the first of several, in
// a stable order, which is what an ambiguous reference would have to fall back
// to anyway. A class nested in another is a different matter, see pickByName.
func (r *TypeResolver) declaredElsewhere(name string) (string, bool) {
	return r.pickByName(r.bySimpleName[name])
}

// pickByName chooses the declaration that a simple name falls back to among the
// registered ones that carry it, keys in a stable order.
//
// A top-level class is what the name denotes unless some scope says otherwise,
// and the first of several is the answer, as it always was. A class nested in
// another is named from inside its outer class or through it, so a bare name
// that no scope declared falls back to one only when it is the only candidate:
// with two, nothing says which the name means, and the first would be a guess
// that exports the fields of the wrong class.
//
// An enum never answers here. A name binds to an enum through the scope only:
// an import on demand of a package the sources do not hold, java.lang and the
// package of a class read from a library may carry the name too, and the enum
// of some other package that happens to share it is not the type written.
func (r *TypeResolver) pickByName(keys []string) (string, bool) {
	nested, candidates := "", 0
	for _, key := range keys {
		c, found := r.lookup(key)
		if !found || c.IsEnum {
			continue
		}
		if c.Outer == "" {
			return key, true
		}
		nested, candidates = key, candidates+1
	}
	return nested, candidates == 1
}

func (r *TypeResolver) isRegistered(key string) bool {
	_, found := r.lookup(key)
	return found
}

// IsVoid reports whether a declared return type, as written in source, says the
// method returns nothing: the keyword `void`, or java.lang.Void. A class of the
// project that is called Void is a class like any other.
func (r *TypeResolver) IsVoid(source parser.Class, typeName string) bool {
	typeName = strings.TrimSpace(typeName)
	if typeName == "void" {
		return true
	}
	simple, ok := r.javaLangType(source, typeName)
	return ok && simple == "Void"
}

// isPlatformType reports whether the resolver answers for a qualified name of
// the JDK or of a library without reading a declaration of it.
func isPlatformType(qualified string) bool {
	if _, ok := jdkQualifiedScalars[qualified]; ok {
		return true
	}
	if jdkValueTypes[qualified] {
		return true
	}
	if _, ok := platformEnums[qualified]; ok {
		return true
	}
	if _, ok := platformModels[qualified]; ok {
		return true
	}
	if IsLibraryClass(qualified) {
		return true
	}
	if simple, ok := strings.CutPrefix(qualified, "java.lang."); ok && isJavaLangType(simple) {
		return true
	}
	return qualified == javaUtilOptional
}

// isJavaLangType reports whether a simple name is one of the java.lang classes
// the resolver answers for itself.
func isJavaLangType(name string) bool {
	_, scalar := javaLangScalars[name]
	return scalar || name == "Object"
}

// javaLangType reports which java.lang class typeName denotes as seen from
// source, by its simple name (`Short`), when it denotes one the resolver
// answers for itself.
//
// A qualified spelling names its own package and so needs no scope. A simple
// name denotes the JDK class only when nothing closer declares it: java.lang is
// imported by every compilation unit, but only as the last on-demand import
// (JLS 6.4.1), so a class of the same file, a single-type import, a class of
// the same package or another on-demand import of that name comes first, and a
// type variable of source hides it before any of them. Importing the JDK class
// by name is no such declaration.
func (r *TypeResolver) javaLangType(source parser.Class, typeName string) (string, bool) {
	if simple, qualified := strings.CutPrefix(typeName, "java.lang."); qualified {
		return simple, isJavaLangType(simple)
	}
	if !isJavaLangType(typeName) || r.isTypeVariable(source, typeName) {
		return "", false
	}
	if key, declared := r.inScope(source, typeName, true); declared && key != "java.lang."+typeName {
		return "", false
	}
	return typeName, true
}

// mayBeDependencyClass reports whether an unqualified name could denote a class
// of the dependencies, as seen from source. A type variable in scope in source
// does not, and neither does a name the resolver answers for before it consults
// any scope (a scalar, a collection, a map): every lookup is a call to the
// indexer, and a class of a library that carries such a name does not take its
// place. The names of JDK value types (`Date`, `Period`, `BigDecimal`) and
// `Object` are not among them: they are answered for after the scope, and a class
// that an import on demand brings in is what they denote.
//
// A type variable is in scope in the class that declares it, not in the project:
// a class that declares `<Item>` does not make `Item` any less a class
// elsewhere. The type variables of a method are not recorded, so one of them is
// asked for like a class, as a name no class of the project declares, and
// answers nothing.
func (r *TypeResolver) mayBeDependencyClass(source parser.Class, name string) bool {
	if name == "" || strings.Contains(name, ".") || r.isTypeVariable(source, name) {
		return false
	}
	if _, ok := primitiveTypes[name]; ok || javaLangScalars[name] != "" || collectionTypes[name] || mapTypes[name] {
		return false
	}
	return true
}

// isTypeVariable reports whether name is a type variable in scope in the body of
// the class source: a type parameter of source, or, when source is an inner
// class, of a class around it, the innermost first (see parser.Class.Inner). A
// type variable is the first thing a name denotes there: it hides every class
// that carries the name, whichever the scope declares, imports or inherits.
func (r *TypeResolver) isTypeVariable(source parser.Class, name string) bool {
	for c := source; ; c = r.enclosing(c) {
		if slices.Contains(c.TypeParameters, name) {
			return true
		}
		if !c.Inner || c.Outer == "" {
			return false
		}
	}
}

// openVariableMark begins the spelling of a type variable of the caller that no
// argument binds, as it is bound to the parameter of the class it is passed to
// (see qualifyTypeName). The caller knows that the name is its own variable, and
// the class the argument reaches does not: there, the same name is looked up as a
// class, and a class of any package that carries it would be the answer. The mark
// is not a character of any name, so the spelling cannot be mistaken for one.
const openVariableMark = "'"

// openVariable spells name as a type variable of the caller that nothing binds.
func openVariable(name string) string {
	return openVariableMark + name
}

// variableName returns the name of the type variable that a spelling stands for,
// and whether it stands for one (see openVariableMark).
func variableName(spelled string) (string, bool) {
	return strings.CutPrefix(strings.TrimSpace(spelled), openVariableMark)
}

// openType is the model of a type variable that nothing binds: a single value
// named like the variable. The model is remembered, as the one that tells the
// variable apart from a class or an enum that carries the name (see IsOpenVariable).
func (r *TypeResolver) openType(name string) *model.ObjectModel {
	open := model.SingleModel(name)
	if r.variables == nil {
		r.variables = make(map[*model.ObjectModel]bool)
	}
	r.variables[open] = true
	return open
}

// IsOpenVariable reports whether m is a model this resolver answered a type
// variable that nothing binds with. The model is shown by the name of the
// variable, as a class of the project or an enum that is called so is, so the
// model itself is what tells them apart, not its name.
func (r *TypeResolver) IsOpenVariable(m *model.ObjectModel) bool {
	return m != nil && r.variables[m]
}

// isOpenBinding reports whether the argument a type parameter is bound to is a
// type variable that nothing binds, so that the field typed with the parameter is
// generic.
//
// The caller marks a variable of its own class (see openVariableMark). One of a
// method is not marked: the parser does not record them, so the name reaches here
// as written, a name that no scope declared. It is a variable when some class
// declares it as a type parameter and no class of the project carries it.
func (r *TypeResolver) isOpenBinding(bound string) bool {
	if name, marked := variableName(bound); marked {
		// An array of the variable is an array, not a variable.
		return !strings.HasSuffix(name, "[]")
	}
	return r.allTypeParams[bound] && len(r.bySimpleName[bound]) == 0
}

// isJdkValueName reports whether a simple name is one that ResolveFrom answers
// for from a table of JDK value types once no class is found for it.
func isJdkValueName(name string) bool {
	simple := simpleTypeName(name)
	_, ok := jdkScalarTypes[simple]
	return ok || simple == "object"
}

// lastSegment returns the last dot-separated segment of a name.
func lastSegment(name string) string {
	if i := strings.LastIndex(name, "."); i >= 0 {
		return name[i+1:]
	}
	return name
}

// optionalValue reports whether a field type holds a value or nothing and, for
// `Optional<T>`, returns T. OptionalInt and its kin have no type argument.
// What a name denotes is decided by its scope (resolveSimpleName), so a class of
// the project or of another library called `Optional` holds nothing.
func (r *TypeResolver) optionalValue(source parser.Class, typeName string) (string, bool) {
	base, args := parseGenericType(strings.TrimSpace(typeName))
	if !isOptionalType(r.Identity(source, base)) {
		return "", false
	}
	if len(args) == 1 {
		return args[0], true
	}
	return "", true
}

// UnwrapOptional returns T of a type written `Optional<T>` in source when the
// name denotes java.util.Optional, and any other type as written. A caller that
// classifies a type by its spelling (how Spring binds a handler parameter) asks
// here first, so that the rule reaches java.util.Optional alone: a project
// class named Optional is an object like any other.
func (r *TypeResolver) UnwrapOptional(source parser.Class, typeName string) string {
	base, args := parseGenericType(strings.TrimSpace(typeName))
	if len(args) == 1 && r.Identity(source, base) == javaUtilOptional {
		return args[0]
	}
	return strings.TrimSpace(typeName)
}

// platformEnum returns the JDK enum that a type name denotes in source, as a
// declaration that holds its constants.
func (r *TypeResolver) platformEnum(source parser.Class, typeName string) (parser.Class, bool) {
	name := strings.TrimSpace(typeName)
	if name == "" || strings.ContainsAny(name, "<[") {
		return parser.Class{}, false
	}
	qualified := r.resolveName(source, name)
	constants, ok := platformEnums[qualified]
	if !ok {
		return parser.Class{}, false
	}
	return parser.Class{Name: lastSegment(qualified), IsEnum: true, EnumConstants: constants}, true
}

func (r *TypeResolver) SetDependencyResolver(dr DependencyResolver) {
	r.dependencyResolver = dr
	r.dependencyAnswers = nil
	r.supertypesHeld = make(map[string]bool)
}

func (r *TypeResolver) SetCollectorDependencyResolver(cdr collector.DependencyResolver) {
	r.dependencyResolver = &collectorDependencyAdapter{resolver: cdr}
	r.dependencyAnswers = nil
	r.supertypesHeld = make(map[string]bool)
}

// SetUnresolved attaches a shared sink that records type names this resolver
// could not expand. It is optional; without it Resolve behaves as before.
func (r *TypeResolver) SetUnresolved(u *collector.UnresolvedSet) {
	r.unresolved = u
}

// Unresolved returns the type names this resolver failed to resolve, mapped to
// their occurrence counts.
func (r *TypeResolver) Unresolved() map[string]int {
	return r.unresolved.Counts()
}

// recordUnresolved notes that typeName could not be expanded into fields and
// was rendered as an opaque single value.
func (r *TypeResolver) recordUnresolved(typeName string) {
	r.unresolved.Record(typeName)
}

type collectorDependencyAdapter struct {
	resolver collector.DependencyResolver
}

func (a *collectorDependencyAdapter) ResolveClass(className string) *parser.Class {
	rt := a.resolver.ResolveType(className)
	if rt == nil {
		return nil
	}
	return resolvedTypeToClass(rt)
}

func resolvedTypeToClass(rt *collector.ResolvedType) *parser.Class {
	class := &parser.Class{
		Name:               rt.Name,
		TypeParameters:     rt.TypeParameters,
		SuperClass:         rt.SuperClass,
		SuperClassTypeArgs: rt.SuperClassTypeArgs,
		IsInterface:        rt.IsInterface,
		Interfaces:         rt.Interfaces,
	}

	for _, f := range rt.Fields {
		var annotations []parser.Annotation
		if !f.Required {
			annotations = append(annotations, parser.Annotation{Name: "Nullable"})
		}
		class.Fields = append(class.Fields, parser.Field{
			Name:        f.Name,
			Type:        f.Type,
			Annotations: annotations,
		})
	}

	return class
}

// Resolve resolves rawType without a compilation-unit context: a simple name
// falls back to a declaration carrying it, which is all a source parsed
// without scope information can offer (see resolveName).
func (r *TypeResolver) Resolve(rawType string, typeBindings map[string]string) *model.ObjectModel {
	return r.ResolveFrom(parser.Class{}, rawType, typeBindings)
}

// ResolveFrom resolves rawType as written in source, honouring that
// compilation unit's package and imports when a simple name has to be matched
// to a declaration.
func (r *TypeResolver) ResolveFrom(source parser.Class, rawType string, typeBindings map[string]string) *model.ObjectModel {
	if rawType == "" {
		return model.NullModel()
	}

	if typeBindings != nil {
		if bound, ok := typeBindings[rawType]; ok {
			return r.ResolveFrom(source, bound, nil)
		}
	}

	// A wildcard is a type argument, not a type. `Result<?>` declares an open
	// payload type, and rendering the argument as the payload name produced a
	// table that said the type was literally "?" with an empty demo (issue
	// #167). Without evidence of a concrete type the argument stays open, so
	// the schema carries an empty object. An upper bound is that evidence:
	// every element of a `List<? extends Order>` is an Order. A lower bound
	// is not: a `List<? super Order>` may hold any supertype of Order.
	if isWildcard(rawType) {
		if bound, ok := wildcardUpperBound(rawType); ok {
			return r.ResolveFrom(source, bound, typeBindings)
		}
		return model.EmptyObject()
	}

	// An array serialises as a JSON array of its element type, the same as a
	// collection does: `String[]` and `List<String>` are one shape, and
	// `Order[][]` is an array of arrays of Order.
	if element, ok := strings.CutSuffix(strings.TrimSpace(rawType), "[]"); ok {
		// A byte[] and a char[] are the exceptions: Jackson writes a byte[] as
		// one Base64 string and a char[] as one string of its characters, where
		// `Byte[]`, `Character[]` and the lists of them are arrays. Only the
		// JSON view is Jackson's; the binding view leaves the array as it is,
		// since Spring's data binder reads neither as one string. A byte[] that
		// is a whole request or response body is written raw by Spring rather
		// than by Jackson; the string is still the closest schema of it.
		if !r.bindingView && !r.keepBinaryArray && (element == "byte" || element == "char") {
			return model.SingleModel(model.JsonTypeString)
		}
		return model.ArrayModel(r.ResolveFrom(source, element, typeBindings))
	}

	// A type variable that nothing binds is an open type, whatever class carries
	// its name: the name is not looked up. One that an argument passed on from the
	// caller is spelled as such (see openVariableMark); one of source's own class
	// is in scope here (see isTypeVariable).
	if name, marked := variableName(rawType); marked {
		return r.openType(name)
	}
	if name := strings.TrimSpace(rawType); r.isTypeVariable(source, name) {
		return r.openType(name)
	}

	if jsonType, ok := primitiveTypes[rawType]; ok {
		return model.SingleModel(jsonType)
	}

	// Every compilation unit imports java.lang, so `java.lang.Short` is the
	// type `Short` names, however the source spells it. The simple name is
	// that type only where no declaration in scope hides it: a project class
	// named `Short` is a class, not a number.
	if simple, ok := r.javaLangType(source, rawType); ok {
		if simple == "Object" {
			// `java.lang.Object` carries no fields, so an empty object is its
			// true shape rather than an opaque scalar.
			return model.EmptyObject()
		}
		return model.SingleModel(javaLangScalars[simple])
	}

	baseName, typeArgs := parseGenericType(rawType)

	if baseName != rawType {
		// Reactor publishers used as a return type (`Mono<Result<User>>`)
		// carry no data of their own, so the schema is the type argument.
		// easy-yapi resolves these through a handled rule of the same shape.
		if isStream, ok := reactiveWrappers[r.Identity(source, baseName)]; ok && len(typeArgs) == 1 {
			payload := r.ResolveFrom(source, typeArgs[0], typeBindings)
			if isStream {
				return model.ArrayModel(payload)
			}
			return payload
		}

		// Response wrappers are the same shape: `ResponseEntity<Order>` is
		// `Order`, and the wrapper itself never appears in the schema.
		if responseWrappers[r.Identity(source, baseName)] && len(typeArgs) == 1 {
			return r.ResolveFrom(source, typeArgs[0], typeBindings)
		}

		// An Optional is plumbing too: what Jackson writes is the value it
		// holds. Only java.util.Optional is, which the scope of the name
		// decides: a project's own `Optional` or another library's is a class.
		if len(typeArgs) == 1 && r.Identity(source, baseName) == javaUtilOptional {
			return r.ResolveFrom(source, typeArgs[0], typeBindings)
		}

		// A collection or a map holds its arguments' models, which resolve where
		// they are written.
		if r.isCollectionType(source, baseName) {
			if len(typeArgs) > 0 {
				itemModel := r.ResolveFrom(source, typeArgs[0], typeBindings)
				return model.ArrayModel(itemModel)
			}
			return model.ArrayModel(model.NullModel())
		}

		if r.isMapType(source, baseName) {
			keyModel := model.SingleModel(model.JsonTypeString)
			valueModel := model.NullModel()
			if len(typeArgs) >= 2 {
				valueModel = r.ResolveFrom(source, typeArgs[1], typeBindings)
			} else if len(typeArgs) == 1 {
				valueModel = r.ResolveFrom(source, typeArgs[0], typeBindings)
			}
			return model.MapModel(keyModel, valueModel)
		}

		// The arguments of a generic class are bound to its type parameters and
		// resolved again inside it, in its scope, where none of source's names
		// apply. They therefore travel as spellings that mean the same wherever
		// they are read (see qualifyTypeName), and are never rebuilt from the
		// models they resolve to, which name an object by its simple name only.
		typeArgs = r.qualifyTypeArgs(source, typeBindings, typeArgs)
	}

	name := r.resolveName(source, baseName)

	if class, found := r.lookup(name); found {
		return r.resolveClass(class, name, typeArgs)
	}

	// A JDK enum and a library class the resolver has an answer for are read
	// before the dependencies are asked: the answer does not depend on the
	// version of a library the project uses, and what a dependency's copy of a
	// Jackson tree or a multipart file expands to is its internals, not the wire.
	platform := name
	if _, ok := platformEnums[platform]; ok {
		return model.SingleModel(model.JsonTypeString)
	}
	if build, ok := platformModels[platform]; ok {
		resolved := build()
		if platform == MultipartFileClass {
			if r.uploads == nil {
				r.uploads = make(map[*model.ObjectModel]bool)
			}
			r.uploads[resolved] = true
		}
		return resolved
	}

	// An enum of the scanned sources is already known; the JSON view just
	// does not read it when Jackson writes it in a form the constants do not
	// tell, and a dependency's copy of it would not be read either.
	// A name that is still unqualified here found no class in the packages
	// resolveName tried; it is asked as written for a resolver that knows
	// classes by their simple names, unless it is one that cannot be a class of
	// a library (see mayBeDependencyClass) or is the name of a JDK value type,
	// which the table below answers for. A class of the JDK is no dependency of
	// the project: asking for it would only cost a call to the indexer.
	if r.dependencyResolver != nil && (strings.Contains(name, ".") || r.mayBeDependencyClass(source, name) && !isJdkValueName(name)) && !r.isRegisteredEnum(name) && !strings.HasPrefix(name, "java.") && !r.memberOfScanned(name) {
		if depClass, key := r.readDependency(name); depClass != nil {
			if r.sees(*depClass) {
				return r.resolveClass(*depClass, key, typeArgs)
			}
		}
	}

	// No class body is available. Well-known JDK value types are still scalars
	// on the wire, so answer for them instead of reporting a failure.
	// easy-yapi reaches the same answer with `json.rule.convert` rules over the
	// qualified name plus an IrType lookup over the simple name.
	if jsonType, ok := jdkQualifiedScalars[platform]; ok {
		return model.SingleModel(jsonType)
	}
	// A declaration of the project is the class its name says, not a JDK type
	// that shares its simple name: an enum of the JSON view is declared, though
	// not read.
	if jsonType, ok := jdkScalarTypes[simpleTypeName(name)]; ok && r.denotesJdkClass(source, name) {
		return model.SingleModel(jsonType)
	}
	// An open object names itself `object` (see model.EmptyObject), and a type
	// argument bound to one is resolved by that name.
	if name == model.EmptyObject().TypeName {
		return model.EmptyObject()
	}

	shown := r.shortForm(rawType)
	r.recordUnresolved(shown)
	return model.SingleModel(shown)
}

// denotesJdkClass reports whether name, as resolveName answered it for a type
// written in source, is a class of the JDK. The scalar a JDK value type
// serialises to belongs to that class: a `LocalDate` of another library, or of
// the project, is no such value, whatever its simple name says.
//
// A qualified name is the JDK's when it is in a package of java, and no
// declaration of the project is under it. A simple name that nothing declared
// or imported is the JDK's only in a source that tells nothing of its scope: in
// a compilation unit that has a package or imports, every simple name it could
// mean has been tried, and one that none supplied is no class of the JDK.
func (r *TypeResolver) denotesJdkClass(source parser.Class, name string) bool {
	if r.isDeclared(name) {
		return false
	}
	if strings.Contains(name, ".") {
		return strings.HasPrefix(name, "java.")
	}
	return source.Package == "" && len(source.Imports) == 0
}

// ResolveReturn resolves the declared return type of method as written in
// source.
//
// A wildcard type argument (`Result<?>`) names no payload type by itself, but
// the method body usually does: `Result.success(new PageResult<>())` says the
// open argument was a PageResult. When the returns in the body agree on a type
// that fits the declared shape, that type fills the argument; otherwise the
// declared type resolves with the argument left open (see ResolveFrom).
func (r *TypeResolver) ResolveReturn(source parser.Class, method parser.Method, typeBindings map[string]string) *model.ObjectModel {
	if !hasWildcardArg(method.ReturnType) {
		return r.ResolveFrom(source, method.ReturnType, typeBindings)
	}

	bodyType, found := expr.ReturnedType(method, func(name string) (parser.Class, bool) {
		return r.Class(source, name)
	})
	if found {
		if filled, ok := fillWildcardArgs(method.ReturnType, bodyType); ok {
			return r.ResolveFrom(source, filled, typeBindings)
		}
	}
	return r.ResolveFrom(source, method.ReturnType, typeBindings)
}

// hasWildcardArg reports whether a type spelling carries a wildcard type
// argument. "?" cannot occur in a Java type name any other way.
func hasWildcardArg(rawType string) bool {
	return strings.Contains(rawType, "?")
}

// isWildcard reports whether a type spelling is a wildcard type argument,
// bounded (`? extends Order`) or not.
func isWildcard(typeName string) bool {
	return strings.HasPrefix(strings.TrimSpace(typeName), "?")
}

// wildcardUpperBound returns X of a wildcard `? extends X`.
func wildcardUpperBound(typeName string) (string, bool) {
	rest := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(typeName), "?"))
	if !strings.HasPrefix(rest, "extends ") {
		return "", false
	}
	return strings.TrimSpace(strings.TrimPrefix(rest, "extends ")), true
}

// fillWildcardArgs replaces the open type arguments of declared with the
// corresponding arguments of inferred, and only when both spell the same base
// type: `Result<?>` filled from `Result<PageResult>` is `Result<PageResult>`.
func fillWildcardArgs(declared, inferred string) (string, bool) {
	declBase, declArgs := parseGenericType(declared)
	infBase, infArgs := parseGenericType(inferred)
	if len(declArgs) == 0 || len(declArgs) != len(infArgs) {
		return "", false
	}
	if lastSegment(declBase) != lastSegment(infBase) {
		return "", false
	}

	args := make([]string, len(declArgs))
	changed := false
	for i, arg := range declArgs {
		if isWildcard(arg) && !isWildcard(infArgs[i]) {
			args[i] = infArgs[i]
			changed = true
			continue
		}
		args[i] = arg
	}
	if !changed {
		return "", false
	}
	return declBase + "<" + strings.Join(args, ", ") + ">", true
}

// resolveClass expands one declaration into an object model. Its type
// parameters are bound to typeArgs, which are spelled in full (see
// qualifyTypeName), and to nothing else: a parameter that no argument binds
// stays open, as it does for a raw type.
func (r *TypeResolver) resolveClass(class parser.Class, key string, typeArgs []string) *model.ObjectModel {
	// An enum is the name of one of its constants, in the binding view and in
	// the JSON view of an enum Jackson writes that way (see sees).
	if class.IsEnum {
		return model.SingleModel(model.JsonTypeString)
	}
	expansion := expansionName(key, typeArgs)
	if r.isBeingExpanded(key, expansion) {
		return model.RefModel(class.Name)
	}

	if class.IsInterface {
		if impl, implKey, found := r.findImplementation(key); found {
			return r.resolveClass(impl, implKey, nil)
		}
		return model.RefModel(class.Name)
	}

	// The class is being expanded from here on, a map subclass too: the value of
	// its map is read from the arguments of its extends clause, which may name
	// the class again (`Tree extends HashMap<String, Tree>`).
	r.resolving[key] = append(r.resolving[key], expansion)
	defer func() {
		if active := r.resolving[key]; len(active) > 1 {
			r.resolving[key] = active[:len(active)-1]
		} else {
			delete(r.resolving, key)
		}
	}()

	if r.isMapSubclass(class) {
		return r.resolveMapSubclass(class, typeArgs)
	}

	localBindings := make(map[string]string)
	for i, tp := range class.TypeParameters {
		if i < len(typeArgs) {
			localBindings[tp] = typeArgs[i]
		}
	}

	typeParamSet := make(map[string]bool, len(class.TypeParameters))
	for _, tp := range class.TypeParameters {
		typeParamSet[tp] = true
	}

	obj := &model.ObjectModel{
		Kind:     model.KindObject,
		TypeName: class.Name,
		Fields:   make(map[string]*model.FieldModel, len(class.Fields)),
	}
	for _, f := range class.Fields {
		// Static fields never belong to a serialized instance, so they are
		// skipped — that also covers static final constants. Instance final
		// fields are ordinary data: immutable DTOs and Lombok @Value classes
		// declare every field final, and dropping them here silently
		// exported empty bodies.
		if f.IsStatic {
			continue
		}
		fm := r.resolveField(class, f, localBindings, typeParamSet)
		obj.SetField(f.Name, fm)
	}

	// A property may come from code rather than from a field: Jackson
	// serialises `isSuccess()` as `success` even when no field backs it.
	// Declared fields are added first, so a getter for a field that already
	// exists cannot add a second copy under the same name.
	r.addGetterProperties(class, localBindings, typeParamSet, obj)

	r.resolveInheritedFields(class, localBindings, obj)

	return obj
}

// expansionName spells the type that a class is expanded as: the class with the
// arguments it is bound to, which are spelled in full (see qualifyTypeName).
func expansionName(key string, typeArgs []string) string {
	if len(typeArgs) == 0 {
		return key
	}
	return key + "<" + strings.Join(typeArgs, ", ") + ">"
}

// isBeingExpanded reports whether expanding the type named expansion, of the
// class registered under key, would come back to a type that is being expanded
// already. A type that refers to itself would otherwise never end.
//
// A class may be expanded again while it is being expanded when the type it is
// expanded as is not the same as one under way and no larger than any of them:
// `Box<Item>` inside `Box<Box<Item>>`, or `Wrapper<Wrapper<Item>>` reached from
// `Outer<Item>` inside `Wrapper<Outer<Item>>`. The same type again (`Node<T>`
// inside `Node<Item>`), and a type whose arguments grow (`Node<List<T>>` inside
// `Node<T>`), are not expanded. Along any path the types a class is expanded as
// get no larger and never repeat, and there are finitely many of each size, so
// every class is expanded a finite number of times however the classes refer to
// one another.
func (r *TypeResolver) isBeingExpanded(key, expansion string) bool {
	size := typeSize(expansion)
	for _, active := range r.resolving[key] {
		if expansion == active || size > typeSize(active) {
			return true
		}
	}
	return false
}

// typeSize measures a spelled type by what a type can grow by: each argument,
// array dimension and wildcard adds one.
func typeSize(typeName string) int {
	return 1 + strings.Count(typeName, "<") + strings.Count(typeName, ",") + strings.Count(typeName, "[]") + strings.Count(typeName, "?")
}

// fieldRequired decides whether a field is required from its annotations.
//
// In the default mode a field is required unless it is marked nullable. In the
// constraints mode it is required only when a Bean Validation constraint
// demands a value, which is what the constraint enforces at runtime: a field
// without one may be left out of a request. A nullable marker wins in both
// modes.
//
// Only Bean Validation's @NotNull is a constraint. JetBrains' @NotNull is a
// nullability marker the Kotlin compiler puts on every non-null property, so
// it would make every field of a compiled Kotlin class required; it is told
// apart by the import of the declaring compilation unit.
func (r *TypeResolver) fieldRequired(source parser.Class, annotations []parser.Annotation) bool {
	required := r.requiredFields != collector.RequiredFieldsConstraints
	for _, ann := range annotations {
		switch lastSegment(ann.Name) {
		case "NotNull":
			if ann.Name == jetBrainsNotNull || importsAnnotation(source, jetBrainsNotNull) {
				continue
			}
			fallthrough
		case "NotBlank", "NotEmpty":
			if r.requiredFields == collector.RequiredFieldsConstraints {
				required = true
			}
		}
	}
	for _, ann := range annotations {
		switch lastSegment(ann.Name) {
		case "Nullable", "Null":
			return false
		}
	}
	return required
}

const jetBrainsNotNull = "org.jetbrains.annotations.NotNull"

// importsAnnotation reports whether a compilation unit binds the named
// annotation, either by a single-type import of the annotation itself or by
// an on-demand import of the package that declares it.
func importsAnnotation(source parser.Class, qualified string) bool {
	pkg := qualified[:strings.LastIndex(qualified, ".")]
	for _, imp := range source.Imports {
		if !imp.OnDemand && imp.Path == qualified {
			return true
		}
		if imp.OnDemand && imp.Path == pkg {
			return true
		}
	}
	return false
}

func (r *TypeResolver) resolveField(source parser.Class, f parser.Field, localBindings map[string]string, typeParamSet map[string]bool) *model.FieldModel {
	fieldModel := r.resolvePropertySchema(source, f.Type, f.Annotations, localBindings)
	fm := &model.FieldModel{
		Model:    fieldModel,
		Required: r.fieldRequired(source, f.Annotations),
	}
	// A field typed with a type variable is typed with what the variable is bound
	// to, not with the class its name could be: the rules below read a class by the
	// name the field is written with.
	variable := r.isTypeVariable(source, strings.TrimSpace(f.Type))
	// An enum field is one of the enum's constants.
	if enum, ok := r.fieldEnum(source, f.Type, localBindings); ok {
		fm.Options = EnumOptions(enum)
	}
	// A property that holds a value or nothing may be left out, whatever the
	// view and the constraints say about the value. A JDK enum is a string
	// holding one of its constants in either view: its declaration is known
	// without being read, so there is nothing for the JSON view not to see.
	valueType := f.Type
	if !variable {
		if inner, optional := r.optionalValue(source, f.Type); optional {
			fm.Required = false
			valueType = inner
		}
		if enum, ok := r.platformEnum(source, valueType); ok {
			fm.Options = EnumOptions(enum)
		}
	}
	javadoc.FieldDocumentationFor(f.Annotations, f.JavaDoc).Normalize().ApplyTo(fm)
	// Values the field declares in its own way win over the ones its doc
	// points at.
	if len(fm.Options) == 0 {
		fm.Options = r.seeOptions(source, f.RawJavaDoc)
	}
	// The field is generic when its type is a variable that nothing binds: one of
	// source's own parameters that no argument binds, or that is bound to a variable
	// the caller left open (see isOpenBinding), or, in an inner class, one of the
	// classes around it, which no argument can reach (see isTypeVariable).
	if r.isUnboundTypeParam(f.Type, localBindings, typeParamSet) {
		fm.Generic = true
	} else if variable && !slices.Contains(source.TypeParameters, strings.TrimSpace(f.Type)) {
		fm.Generic = true
	}
	return fm
}

// addGetterProperties records the properties a class exposes through
// zero-argument getters. Jackson serialises `getX()` as `x` and `isX()` as `x`
// for a boolean return type, whether or not a field backs them, so a response
// envelope's computed flags belong in the exported schema.
//
// Names already present in fields win: a getter paired with its field is that
// same property, not a second one. @JsonIgnore drops the property and
// @JsonProperty("other") renames it.
func (r *TypeResolver) addGetterProperties(source parser.Class, localBindings map[string]string, typeParamSet map[string]bool, obj *model.ObjectModel) {
	// A getter-only property can be read, not bound: Spring's data binder sets
	// properties through setters and fields, so the binding view leaves it out.
	if r.bindingView {
		return
	}
	for _, m := range source.Methods {
		property, ok := getterPropertyName(m)
		if !ok || isJsonIgnored(m.Annotations) {
			continue
		}
		if renamed := jsonPropertyName(m.Annotations); renamed != "" {
			property = renamed
		}
		if _, exists := obj.Fields[property]; exists {
			continue
		}

		fm := r.resolveField(source, parser.Field{
			Name:        property,
			Type:        m.ReturnType,
			Annotations: m.Annotations,
			JavaDoc:     m.JavaDoc,
			RawJavaDoc:  m.RawJavaDoc,
		}, localBindings, typeParamSet)

		// A derived boolean is computed state rather than stored data. The
		// example shows the enabling case — `"success": true` for an
		// `isSuccess()` — because a `false` example would document the outcome
		// the envelope exists to confirm.
		if fm.Model != nil && fm.Model.IsSingle() && fm.Model.TypeName == model.JsonTypeBoolean {
			fm.Demo = "true"
		}

		obj.SetField(property, fm)
	}
}

// getterPropertyName maps a method to the JSON property it exposes, if it is a
// state getter: zero arguments, a non-void return type, and a `get`/`is` name.
// Static methods are not instance state and are rejected.
func getterPropertyName(m parser.Method) (string, bool) {
	if m.IsStatic || len(m.Parameters) > 0 {
		return "", false
	}
	returnType := strings.TrimSpace(m.ReturnType)
	if simpleTypeName(returnType) == "" || simpleTypeName(returnType) == "void" {
		return "", false
	}

	if len(m.Name) > 3 && strings.HasPrefix(m.Name, "get") {
		if property := decapitalize(m.Name[3:]); property != "" {
			return property, true
		}
	}
	// `isX` names boolean state only. A helper such as `isValid(String)` is
	// rejected above by its argument, and one returning something else is not
	// a state accessor.
	if len(m.Name) > 2 && strings.HasPrefix(m.Name, "is") && simpleTypeName(returnType) == "boolean" {
		if property := decapitalize(m.Name[2:]); property != "" {
			return property, true
		}
	}
	return "", false
}

// decapitalize lowercases the leading run of capitals, which is how Jackson
// turns `getURL()` into `url`.
func decapitalize(name string) string {
	i := 0
	for i < len(name) && name[i] >= 'A' && name[i] <= 'Z' {
		i++
	}
	if i == 0 {
		return name
	}
	return strings.ToLower(name[:i]) + name[i:]
}

// jsonPropertyName returns the name @JsonProperty binds the member to, or ""
// when the annotation is absent or names nothing.
func jsonPropertyName(annotations []parser.Annotation) string {
	for _, ann := range annotations {
		if lastSegment(ann.Name) != "JsonProperty" {
			continue
		}
		return strings.TrimSpace(ann.Params["value"])
	}
	return ""
}

// isJsonIgnored reports whether @JsonIgnore suppresses the member. The
// annotation's own value can re-enable it: @JsonIgnore(false).
func isJsonIgnored(annotations []parser.Annotation) bool {
	for _, ann := range annotations {
		if lastSegment(ann.Name) != "JsonIgnore" {
			continue
		}
		return strings.TrimSpace(ann.Params["value"]) != "false"
	}
	return false
}

func (r *TypeResolver) isUnboundTypeParam(typeName string, localBindings map[string]string, typeParamSet map[string]bool) bool {
	if typeParamSet == nil {
		return false
	}
	if !typeParamSet[typeName] {
		return false
	}
	if bound, exists := localBindings[typeName]; exists {
		// Bound to another type variable: one the caller left open.
		return r.isOpenBinding(bound)
	}
	return true
}

func (r *TypeResolver) resolveInheritedFields(class parser.Class, localBindings map[string]string, obj *model.ObjectModel) {
	if class.SuperClass == "" {
		return
	}

	superClass, found := r.Supertype(class, class.SuperClass)
	if !found {
		return
	}

	superBindings := r.SuperClassBindings(class, superClass, localBindings)

	superTypeParamSet := make(map[string]bool, len(superClass.TypeParameters))
	for _, tp := range superClass.TypeParameters {
		superTypeParamSet[tp] = true
	}

	// A superclass's fields follow the subclass's own, which is the order a
	// reader of the subclass meets them in. Its static fields are no more part
	// of an instance than the subclass's own (see resolveClass).
	for _, f := range superClass.Fields {
		if f.IsStatic {
			continue
		}
		if _, exists := obj.Fields[f.Name]; !exists {
			// The field type is written in the superclass's own compilation
			// unit, so that is the scope its simple names resolve in.
			fm := r.resolveField(superClass, f, superBindings, superTypeParamSet)
			obj.SetField(f.Name, fm)
		}
	}

	// The superclass's getters are part of the subclass's wire format too, and
	// are resolved in the compilation unit that declared them.
	r.addGetterProperties(superClass, superBindings, superTypeParamSet, obj)

	r.resolveInheritedFields(superClass, superBindings, obj)
}

// SuperClassBindings binds each type parameter of super to the argument that
// sub's extends clause gives it, spelled so that it means the same in the scope
// of super: `Sub extends Base<Item>` binds Base's parameter to the class Item
// names where Sub is written, not to whichever class Item names beside Base.
// bindings binds the type parameters of sub itself, for an argument that
// mentions them: `Sub<S> extends Base<S[]>` under S=Order binds `Order[]`.
//
// The extends clause is outside the body of sub, so the arguments are spelled in
// its header: the member types of sub are not in scope there, those of the
// classes around it are (see inHeader).
func (r *TypeResolver) SuperClassBindings(sub, super parser.Class, bindings map[string]string) map[string]string {
	out := make(map[string]string, len(super.TypeParameters))
	r.inHeader(qualifiedName(sub), func() {
		for i, tp := range super.TypeParameters {
			if i < len(sub.SuperClassTypeArgs) {
				out[tp] = r.qualifyTypeName(sub, bindings, sub.SuperClassTypeArgs[i])
			}
		}
	})
	return out
}

// qualifyTypeArgs spells each type argument so that it means the same wherever
// it is read (see qualifyTypeName).
func (r *TypeResolver) qualifyTypeArgs(source parser.Class, bindings map[string]string, args []string) []string {
	if len(args) == 0 {
		return args
	}
	out := make([]string, len(args))
	for i, arg := range args {
		out[i] = r.qualifyTypeName(source, bindings, arg)
	}
	return out
}

// qualifyTypeName spells a type as written in source so that it denotes the same
// type wherever it is resolved. An argument is bound to a type parameter and
// resolved later inside the generic class, whose imports and package are not
// source's: a name that a class of that package shares would otherwise stand in
// for the one source meant, and a dependency class could only be asked for by a
// simple name.
//
// A type variable of source that bindings binds is replaced by its binding,
// which is already spelled this way; one that nothing binds is spelled as an open
// variable (see openVariableMark), which keeps it apart from every class that
// carries its name. Every other class name goes through the same scope order as a direct
// reference (see resolveName) and is written as the class it denotes, in full, in
// nested type arguments, array dimensions and wildcard bounds alike: whatever the
// class is, of the project, of a library or of the JDK, the spelling names it by
// identity, and no class of the generic class's package or import can stand in
// for it. A dotted name goes through it too: `Outer.Inner` is the member type of
// the class Outer stands for in source, and `lib.Outer.Inner` or `lib.Outer$Inner`
// is itself.
//
// What ResolveFrom answers for by the spelling alone, before it consults a
// scope, keeps that meaning wherever the argument is read: the primitive types
// stay as written. A collection or a map is the java.util class its name denotes
// in the caller's scope, and is written as such (see collectionIdentity). A name
// that is already qualified, and a name nothing in scope declares, stay as
// written too.
//
// The written form is remembered (see shortForm), because the type is still the
// one the source wrote when it is displayed or reported, unless it is shown by
// its full name (see shownInFull).
func (r *TypeResolver) qualifyTypeName(source parser.Class, bindings map[string]string, typeName string) string {
	typeName = strings.TrimSpace(typeName)
	if typeName == "" {
		return typeName
	}
	// A variable the caller left open is spelled already, and is no name to look up.
	if _, marked := variableName(typeName); marked {
		return typeName
	}
	if isWildcard(typeName) {
		rest := strings.TrimSpace(strings.TrimPrefix(typeName, "?"))
		for _, bound := range []string{"extends ", "super "} {
			if strings.HasPrefix(rest, bound) {
				return "? " + bound + r.qualifyTypeName(source, bindings, strings.TrimPrefix(rest, bound))
			}
		}
		return typeName
	}
	suffix := ""
	for strings.HasSuffix(typeName, "[]") {
		typeName = strings.TrimSpace(strings.TrimSuffix(typeName, "[]"))
		suffix += "[]"
	}
	base, args := parseGenericType(typeName)
	if len(args) == 0 {
		if bound, ok := bindings[base]; ok {
			return bound + suffix
		}
		if r.isTypeVariable(source, base) {
			return openVariable(base) + suffix
		}
	}
	if spelled, ok := scopeFreeName(base); ok {
		base = spelled
	} else if jdk, ok := r.collectionIdentity(source, base); ok {
		if !r.shownInFull(jdk, base) {
			r.showAs(jdk, base)
		}
		base = jdk
	} else if key := r.resolveName(source, base); key != base {
		if !r.shownInFull(key, base) {
			r.showAs(key, base)
		}
		base = key
	}
	if len(args) > 0 {
		// The arguments are separated as the source wrote them, so a type that is
		// shown reads the same.
		separator := ","
		if strings.Contains(typeName, ", ") {
			separator = ", "
		}
		base += "<" + strings.Join(r.qualifyTypeArgs(source, bindings, args), separator) + ">"
	}
	return base + suffix
}

// shownInFull reports whether a type argument that resolveName gave key is
// displayed and reported by that name, and not as the source wrote it: any name
// when a dependency resolver is asked for classes (the package is what a missing
// dependency is looked up by), and a name that another class shares, which the
// full name tells apart.
func (r *TypeResolver) shownInFull(key, name string) bool {
	if r.dependencyResolver != nil {
		return true
	}
	// A class nested in another is read by the simple name it ends in.
	for _, other := range r.bySimpleName[lastSegment(canonicalName(name))] {
		if other != key {
			return true
		}
	}
	return false
}

// showAs records that a class written in full was written as name in source.
// The first spelling stays: the class is one, and the forms differ only in the
// scope that wrote them.
func (r *TypeResolver) showAs(identity, name string) {
	if r.shown == nil {
		r.shown = make(map[string]string)
	}
	if _, ok := r.shown[identity]; !ok {
		r.shown[identity] = name
	}
}

// ShowAs tells the resolver how a class is to be displayed when a type that
// names it in full is shown or reported: the name its sources write.
func (r *TypeResolver) ShowAs(identity, name string) {
	r.showAs(identity, name)
}

// shortForm returns a spelled type with each class that was written in full for
// its identity (see qualifyTypeName) back as the source wrote it. The type that
// is shown on a model and reported as unresolved is a name for people, so
// `Optional<com.pr.lib.Item>` reads as `Optional<Item>`, as it was written.
func (r *TypeResolver) shortForm(spelled string) string {
	// A variable the caller left open is shown by its name (see openVariableMark).
	spelled = strings.ReplaceAll(spelled, openVariableMark, "")
	if len(r.shown) == 0 || !strings.Contains(spelled, ".") {
		return spelled
	}
	var out strings.Builder
	for i := 0; i < len(spelled); {
		if !isTypeNameChar(spelled[i]) {
			out.WriteByte(spelled[i])
			i++
			continue
		}
		start := i
		for i < len(spelled) && isTypeNameChar(spelled[i]) {
			i++
		}
		word := spelled[start:i]
		if short, ok := r.shown[word]; ok {
			word = short
		}
		out.WriteString(word)
	}
	return out.String()
}

func isTypeNameChar(c byte) bool {
	return c == '.' || c == '_' || c == '$' || c >= '0' && c <= '9' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= 0x80
}

// scopeFreeName reports whether ResolveFrom answers for a bare name from its
// spelling alone, before it consults any scope: the keyword of a primitive type,
// which no declaration can hide. It returns the spelling that keeps that meaning
// wherever it is read, the name itself.
func scopeFreeName(name string) (string, bool) {
	_, ok := primitiveTypes[name]
	return name, ok
}

// collectionIdentity returns the qualified name of the java.util collection or
// map, or of java.lang.Iterable, that a simple name denotes in source, and
// reports false for any other name: a class of the project, or a single-type
// import of another library's, that is called List is a class (see
// jdkClassName).
func (r *TypeResolver) collectionIdentity(source parser.Class, name string) (string, bool) {
	simple := r.jdkClassName(source, name)
	if strings.Contains(simple, ".") || (!collectionTypes[simple] && !mapTypes[simple]) {
		return "", false
	}
	if simple == "Iterable" {
		return "java.lang.Iterable", true
	}
	return "java.util." + simple, true
}

// isPlatformName reports whether a qualified name belongs to the JDK or the
// Kotlin standard library. The javax packages are not all the JDK's: javax.money
// and javax.inject, among others, are libraries, and a class of one is a
// dependency like any other.
func isPlatformName(name string) bool {
	for _, prefix := range []string{"java.", "kotlin."} {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}
	return false
}

func parseGenericType(rawType string) (string, []string) {
	idx := strings.Index(rawType, "<")
	if idx == -1 {
		return rawType, nil
	}

	baseName := rawType[:idx]
	rest := rawType[idx:]

	if len(rest) < 2 || rest[0] != '<' || rest[len(rest)-1] != '>' {
		return rawType, nil
	}

	inner := rest[1 : len(rest)-1]
	args := splitTypeArgs(inner)
	return baseName, args
}

func splitTypeArgs(s string) []string {
	var args []string
	depth := 0
	start := 0

	for i, c := range s {
		switch c {
		case '<':
			depth++
		case '>':
			depth--
		case ',':
			if depth == 0 {
				arg := strings.TrimSpace(s[start:i])
				if arg != "" {
					args = append(args, arg)
				}
				start = i + 1
			}
		}
	}

	if start < len(s) {
		arg := strings.TrimSpace(s[start:])
		if arg != "" {
			args = append(args, arg)
		}
	}

	return args
}

// isMapSubclass reports whether class extends a map. The extends clause is outside
// the body of the class, where its own member types are not in scope, so the
// superclass is read in the scope of the header: a class nested in the class that is
// called HashMap does not hide the import of java.util.HashMap that the clause means.
func (r *TypeResolver) isMapSubclass(class parser.Class) bool {
	if class.SuperClass == "" {
		return false
	}
	baseName, _ := parseGenericType(class.SuperClass)
	// A map written in full is a class of java.util: another package's HashMap is a
	// class of its own.
	var isMap bool
	r.inHeader(qualifiedName(class), func() { isMap = r.isMapType(class, baseName) })
	return isMap
}

// findImplementation returns the class in the scanned sources that implements
// the interface registered under key: the first, in the order of the keys, whose
// implements clause names it.
//
// The clause names the interface as the header of the class reads the name (see
// supertypeKey), which is the whole of what makes a name the interface: the simple
// name alone does not say which interface it is, and one that another package, a
// library or a class around the implementing class declares under it is another.
func (r *TypeResolver) findImplementation(key string) (parser.Class, string, bool) {
	keys := make([]string, 0, len(r.classRegistry))
	for candidate := range r.classRegistry {
		keys = append(keys, candidate)
	}
	sort.Strings(keys) // stable pick when several classes implement the interface

	for _, candidate := range keys {
		class := r.classRegistry[candidate]
		// The JSON view never took an enum for the implementation of an
		// interface, and an enum it now reads does not become one: it is one
		// value of the interface, not the shape every value has.
		if class.IsInterface || !r.sees(class) || (class.IsEnum && !r.bindingView) {
			continue
		}
		for _, ifaceName := range class.Interfaces {
			if r.supertypeKey(class, candidate, ifaceName) == key {
				return class, candidate, true
			}
		}
	}
	return parser.Class{}, "", false
}

// mapSuperClass stands for the superclass of a map subclass, Map<K, V> or one
// of its implementations, whose arguments are the key and the value type.
var mapSuperClass = parser.Class{TypeParameters: []string{"K", "V"}}

// resolveMapSubclass expands a class that extends a map. It is a map of what the
// arguments of its extends clause say, written in its own scope: the value is
// the second of them, with the class's own parameters bound to typeArgs.
func (r *TypeResolver) resolveMapSubclass(class parser.Class, typeArgs []string) *model.ObjectModel {
	keyModel := model.SingleModel(model.JsonTypeString)

	localBindings := make(map[string]string)
	for i, tp := range class.TypeParameters {
		if i < len(typeArgs) {
			localBindings[tp] = typeArgs[i]
		}
	}

	valueModel := model.NullModel()
	if value, ok := r.SuperClassBindings(class, mapSuperClass, localBindings)["V"]; ok {
		valueModel = r.ResolveFrom(class, value, nil)
	}

	return &model.ObjectModel{
		Kind:       model.KindMap,
		TypeName:   class.Name,
		KeyModel:   keyModel,
		ValueModel: valueModel,
	}
}
