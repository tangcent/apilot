package resolver

import (
	"sort"
	"strings"
	"unicode"

	collector "github.com/tangcent/apilot/api-collector"
	javadoc "github.com/tangcent/apilot/api-collector-java/doc"
	"github.com/tangcent/apilot/api-collector-java/expr"
	"github.com/tangcent/apilot/api-collector-java/parser"
	model "github.com/tangcent/apilot/api-model"
)

var primitiveTypes = map[string]string{
	"int":     model.JsonTypeInt,
	"long":    model.JsonTypeLong,
	"float":   model.JsonTypeFloat,
	"double":  model.JsonTypeDouble,
	"boolean": model.JsonTypeBoolean,
	"void":    model.JsonTypeNull,
	"Void":    model.JsonTypeNull,
	"String":  model.JsonTypeString,
	"Integer": model.JsonTypeInt,
	"Long":    model.JsonTypeLong,
	"Float":   model.JsonTypeFloat,
	"Double":  model.JsonTypeDouble,
	"Boolean": model.JsonTypeBoolean,
}

// jdkQualifiedScalars maps a fully qualified JDK value type to the JSON scalar
// it serialises to.
//
// Names here are matched exactly, so short spellings that could belong to a
// project class (`Time`, `URL`) stay resolvable by the class registry. This is
// easy-yapi's `json.rule.convert[<fqn>]` layer.
var jdkQualifiedScalars = map[string]string{
	"java.sql.Time":         model.JsonTypeString,
	"java.sql.Timestamp":    model.JsonTypeString,
	"java.net.URL":          model.JsonTypeString,
	"java.net.URI":          model.JsonTypeString,
	"java.util.Date":        model.JsonTypeString,
	"java.util.Calendar":    model.JsonTypeString,
	"java.util.UUID":        model.JsonTypeString,
	"java.math.BigDecimal":  model.JsonTypeDouble,
	"java.math.BigInteger":  model.JsonTypeLong,
	"org.bson.types.ObjectId": model.JsonTypeString,
}

// jdkScalarTypes maps the simple name of an unambiguous JDK value type to the
// JSON scalar it serialises to.
//
// Matching is package-insensitive on purpose: `java.time.LocalDateTime` and the
// imported bare `LocalDateTime` must give the same answer. This is easy-yapi's
// IrType.fromJavaType layer, which strips the package and reads the simple name.
var jdkScalarTypes = map[string]string{
	// java.lang
	"string":     model.JsonTypeString,
	"char":       model.JsonTypeString,
	"character":  model.JsonTypeString,
	"byte":       model.JsonTypeInt,
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
	"List":       true,
	"ArrayList":  true,
	"Collection": true,
	"Set":        true,
	"HashSet":    true,
	"LinkedList": true,
}

var mapTypes = map[string]bool{
	"Map":           true,
	"HashMap":       true,
	"LinkedHashMap": true,
	"TreeMap":       true,
}

// reactiveWrappers maps a reactor publisher's simple name to whether its
// payload is streamed as an array.
//
// A publisher is plumbing, not payload: nothing of `Mono` or `Flux` reaches
// the wire, so the exported schema is the type argument. `Mono<T>` is `T`;
// `Flux<T>` is a stream and serialises as an array of `T`.
var reactiveWrappers = map[string]bool{
	"mono": false,
	"flux": true,
}

// responseWrappers are the framework types that carry the payload as their
// single type argument: Spring's `ResponseEntity<Order>` and Feign's.
//
// They are plumbing in the same way a publisher is — nothing of the wrapper
// reaches the wire — so the unwrap lives here rather than in each framework
// parser, and a wrapper nested inside another type keeps working.
var responseWrappers = map[string]bool{
	"responseentity": true,
}

type DependencyResolver interface {
	ResolveClass(className string) *parser.Class
}

// TypeResolver expands Java type spellings into object models.
//
// Classes are indexed under their package-qualified name, so two declarations
// sharing a simple name in different packages coexist instead of overwriting
// each other. Which one a simple name denotes is decided per compilation unit
// by resolveSimpleName, following JLS 6.5.5.1 and 6.4.1.
type TypeResolver struct {
	classRegistry      map[string]parser.Class // qualified name -> declaration
	bySimpleName       map[string][]string     // simple name -> qualified names, sorted
	byFile             map[string][]string     // compilation unit -> qualified names declared there
	depBySimpleName    map[string][]string     // bySimpleName for the classes read from dependencies (see registerDependency)
	scannedPackages    map[string]bool         // packages that a class of the scanned sources declares
	allTypeParams      map[string]bool
	resolving          map[string]bool
	dependencyResolver DependencyResolver
	unresolved         *collector.UnresolvedSet
	// bindingView is set while ResolveForBinding expands a model attribute.
	// Enums take part in that view only (see sees).
	bindingView bool
	// requiredFields is a collector.RequiredFields mode; empty means default.
	requiredFields string
	// dependencyAnswers holds what dependencyResolver answered for each name,
	// a miss as nil, so that no name is asked for twice.
	dependencyAnswers map[string]*parser.Class
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
		resolving:     make(map[string]bool),

		depBySimpleName: make(map[string][]string),
		scannedPackages: make(map[string]bool),
	}
	for _, c := range classes {
		r.register(c, qualifiedName(c))
	}
	return r
}

// qualifiedName returns the registry key of a declaration.
func qualifiedName(c parser.Class) string {
	if c.Package == "" {
		return c.Name
	}
	return c.Package + "." + c.Name
}

// register indexes a declaration of the scanned sources under key.
func (r *TypeResolver) register(c parser.Class, key string) {
	r.classRegistry[key] = c
	r.bySimpleName[c.Name] = appendKey(r.bySimpleName[c.Name], key)
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
// A name that nothing in scope declares falls back to a declaration of the
// scanned sources carrying it (see declaredElsewhere). The classes read from
// dependencies are indexed apart from those, and answer only after them: a
// class a library declares must not stand in for a class of the project that
// has the same simple name.
func (r *TypeResolver) registerDependency(c parser.Class, key string) {
	r.classRegistry[key] = c
	r.depBySimpleName[c.Name] = appendKey(r.depBySimpleName[c.Name], key)
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
// It applies the same precedence as Resolve, without expanding the fields.
func (r *TypeResolver) Class(source parser.Class, name string) (parser.Class, bool) {
	return r.lookup(r.resolveSimpleName(source, name))
}

// sees reports whether the current view takes a declaration into account.
// Enums are read for the binding view, where Spring converts a request
// parameter to an enum by the name of a constant, so the constants are the
// values the parameter accepts. The JSON view does not see them: it exports an
// enum type as it does any type whose declaration was not read.
func (r *TypeResolver) sees(c parser.Class) bool {
	return r.bindingView || !c.IsEnum
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
// Enums take part in the binding view only, so the name is looked up there.
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

// seeEnum returns the enum a `@see` reference names as seen from source. An
// enum the scanned sources do not declare is read from the project's
// dependencies: the enums such a reference names usually live in a shared
// model library.
//
// Only a name the source qualifies, by writing it in full or importing it, is
// looked up there: the index knows a class by its qualified name only. Names
// of the JDK and the Kotlin standard library are skipped, as they are when
// type arguments are qualified (see isPlatformName).
//
// The class read is not registered. A doc comment does not put a class in
// scope, and once registered it would answer for its simple name wherever
// nothing in scope declares that name (see resolveSimpleName), in place of a
// project class of the same name.
func (r *TypeResolver) seeEnum(source parser.Class, ref string) (parser.Class, bool) {
	if enum, ok := r.enumClass(source, ref); ok || r.dependencyResolver == nil {
		return enum, ok
	}
	// The name enumClass looked up, in the view that sees enums.
	defer r.enterBindingView()()
	key := r.resolveSimpleName(source, strings.TrimSpace(ref))
	if _, known := r.classRegistry[key]; known || !strings.Contains(key, ".") || isPlatformName(key) {
		return parser.Class{}, false
	}
	class := r.dependencyClass(key)
	if class == nil || !class.IsEnum {
		return parser.Class{}, false
	}
	return *class, true
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

// ResolveForBinding expands a type the way Spring's data binder sees it when
// it binds a model attribute from request parameters, rather than the way
// Jackson writes it: getter-only properties cannot be bound, so they are left
// out, and an enum is a string holding one of its constants.
func (r *TypeResolver) ResolveForBinding(source parser.Class, rawType string, typeBindings map[string]string) *model.ObjectModel {
	defer r.enterBindingView()()
	return r.ResolveFrom(source, rawType, typeBindings)
}

// resolveSimpleName maps a type name as written in source to the registry key
// it denotes, following JLS 6.5.5.1 and 6.4.1: a declaration in the same
// compilation unit shadows a single-type import, which shadows a declaration
// in the same package, which shadows an import-on-demand.
//
// The result doubles as the name to keep resolving with, so an import of a
// type whose body is not among the scanned sources still selects the imported
// type instead of a same-named class that happens to sit elsewhere in the
// project.
func (r *TypeResolver) resolveSimpleName(source parser.Class, name string) string {
	if key, found := r.inScope(source, name); found {
		return key
	}
	if key, found := r.declaredElsewhere(name); found {
		return key
	}
	if key, found := r.readElsewhere(name); found {
		return key
	}
	return name
}

// inScope returns the registry key an unqualified name denotes through the
// scope of source: the four steps of JLS 6.4.1 that need no more than the
// declarations already held. A qualified name denotes itself; a name that
// nothing in scope declares is not found.
func (r *TypeResolver) inScope(source parser.Class, name string) (string, bool) {
	if name == "" || strings.Contains(name, ".") {
		return name, true
	}

	// 1. Same compilation unit: a class declared in the same file.
	for _, key := range r.byFile[source.FilePath] {
		if c, found := r.lookup(key); found && c.Name == name {
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

	// 3. Same package.
	if source.Package != "" {
		if key := source.Package + "." + name; r.isRegistered(key) {
			return key, true
		}
	}

	// 4. Import-on-demand.
	for _, imp := range source.Imports {
		if imp.Static || !imp.OnDemand {
			continue
		}
		if key := imp.Path + "." + name; r.isRegistered(key) {
			return key, true
		}
	}

	return "", false
}

// declaredElsewhere returns the declaration of the scanned sources that carries
// a simple name, wherever it sits. Nothing in scope declared the name: sources
// parsed without import information (a single file) can only offer the
// declarations themselves, so accept the only one — or the first of several, in
// a stable order, which is what an ambiguous reference would have to fall back
// to anyway.
func (r *TypeResolver) declaredElsewhere(name string) (string, bool) {
	return r.firstRegistered(r.bySimpleName[name])
}

// readElsewhere is declaredElsewhere over the classes already read from the
// dependencies. A name reaches it when a class read earlier is spelled by its
// simple name again, as a type argument is once the class it was bound to has
// been expanded.
//
// A simple name that a class of the scanned sources carries belongs to the
// project, whether or not the current view sees that class (the JSON view does
// not see an enum): no library class answers for it.
func (r *TypeResolver) readElsewhere(name string) (string, bool) {
	if len(r.bySimpleName[name]) > 0 {
		return "", false
	}
	return r.firstRegistered(r.depBySimpleName[name])
}

func (r *TypeResolver) firstRegistered(keys []string) (string, bool) {
	for _, key := range keys {
		if r.isRegistered(key) {
			return key, true
		}
	}
	return "", false
}

func (r *TypeResolver) isRegistered(key string) bool {
	_, found := r.lookup(key)
	return found
}

// resolveName is resolveSimpleName that also reads the dependencies. A name
// that nothing in the scope of source declares among the scanned sources may
// still be a class of a library: one the package of source declares, or one
// that an import-on-demand brings in, as in `import com.example.model.*`. Such
// a class is asked of the dependencies by each qualified name the name may
// denote, the package of source first and then each import-on-demand in the
// order they are written (JLS 6.4.1), until one answers.
//
// A source of the project does not ask for a package that the scanned sources
// declare: the project owns it, and a name its classes do not carry is not in
// it. Without that, every name that a library declares costs a call for the
// package of each controller and for each import of the project's own model
// packages, and every call is a process of the indexer. The price is a split
// package, a library class in a package of the project, which a project source
// does not find. A class read from a library asks for every package, its own
// included: its neighbours need no import, whoever else declares the package.
//
// A class found is registered under its qualified name (see
// registerDependency), so each candidate is asked of the dependencies once and
// later answered from the registry.
func (r *TypeResolver) resolveName(source parser.Class, name string) string {
	if key, found := r.inScope(source, name); found {
		return key
	}
	// A class of the project that carries the name wins over a library's: the
	// resolver spells a type by its simple name again when a bound type
	// argument is expanded inside a generic class, where no import of the
	// project applies, and that name must still denote the project's class.
	if key, found := r.declaredElsewhere(name); found {
		return key
	}
	if r.dependencyResolver != nil && r.mayBeDependencyClass(name) {
		for _, candidate := range dependencyCandidates(source, name) {
			if _, held := r.classRegistry[candidate]; held {
				// The resolver already holds a declaration of that name that the
				// current view does not see: an enum of the JSON view. It is what
				// the name denotes, so a package imported after it cannot supply
				// another.
				break
			}
			if r.isScannedSource(source) && r.scannedPackages[candidate[:strings.LastIndex(candidate, ".")]] {
				continue
			}
			if class := r.dependencyClass(candidate); class != nil {
				r.registerDependency(*class, candidate)
				return candidate
			}
		}
	}
	if key, found := r.readElsewhere(name); found {
		return key
	}
	return name
}

// isScannedSource reports whether source is a compilation unit of the scanned
// sources, as against a class read from a library, whose file is not one of
// them.
func (r *TypeResolver) isScannedSource(source parser.Class) bool {
	return len(r.byFile[source.FilePath]) > 0
}

// mayBeDependencyClass reports whether an unqualified name could denote a class
// of the dependencies. A type variable and a name the resolver answers for
// itself (a scalar, a collection, Object) are not looked up: every lookup is a
// call to the indexer, and none of these is a class of a library.
func (r *TypeResolver) mayBeDependencyClass(name string) bool {
	if name == "" || strings.Contains(name, ".") || r.allTypeParams[name] {
		return false
	}
	if _, ok := primitiveTypes[name]; ok || collectionTypes[name] || mapTypes[name] {
		return false
	}
	simple := simpleTypeName(name)
	if _, ok := jdkScalarTypes[simple]; ok {
		return false
	}
	return simple != "object"
}

// dependencyCandidates lists the qualified names an unqualified name may denote
// through the package of source and the packages it imports on demand, in the
// order JLS 6.4.1 gives them. The JDK and the Kotlin standard library are left
// out, as they are when type arguments are qualified (see isPlatformName), and
// so is an on-demand import of a class (`import com.example.Outer.*`): it
// imports the types nested in that class, which the index, holding top-level
// classes only, does not know.
func dependencyCandidates(source parser.Class, name string) []string {
	var candidates []string
	add := func(pkg string) {
		if pkg == "" {
			return
		}
		candidate := pkg + "." + name
		if isPlatformName(candidate) {
			return
		}
		for _, existing := range candidates {
			if existing == candidate {
				return
			}
		}
		candidates = append(candidates, candidate)
	}
	add(source.Package)
	for _, imp := range source.Imports {
		if imp.Static || !imp.OnDemand {
			continue
		}
		if last := lastSegment(imp.Path); last != "" && unicode.IsUpper(rune(last[0])) {
			continue
		}
		add(imp.Path)
	}
	return candidates
}

// lastSegment returns the last dot-separated segment of a name.
func lastSegment(name string) string {
	if i := strings.LastIndex(name, "."); i >= 0 {
		return name[i+1:]
	}
	return name
}

func (r *TypeResolver) SetDependencyResolver(dr DependencyResolver) {
	r.dependencyResolver = dr
	r.dependencyAnswers = nil
}

func (r *TypeResolver) SetCollectorDependencyResolver(cdr collector.DependencyResolver) {
	r.dependencyResolver = &collectorDependencyAdapter{resolver: cdr}
	r.dependencyAnswers = nil
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
// without scope information can offer (see resolveSimpleName).
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

	if jsonType, ok := primitiveTypes[rawType]; ok {
		return model.SingleModel(jsonType)
	}

	baseName, typeArgs := parseGenericType(rawType)

	if baseName != rawType {
		// Reactor publishers used as a return type (`Mono<Result<User>>`)
		// carry no data of their own, so the schema is the type argument.
		// easy-yapi resolves these through a handled rule of the same shape.
		if isStream, ok := reactiveWrappers[simpleTypeName(baseName)]; ok && len(typeArgs) == 1 {
			payload := r.ResolveFrom(source, typeArgs[0], typeBindings)
			if isStream {
				return model.ArrayModel(payload)
			}
			return payload
		}

		// Response wrappers are the same shape: `ResponseEntity<Order>` is
		// `Order`, and the wrapper itself never appears in the schema.
		if responseWrappers[simpleTypeName(baseName)] && len(typeArgs) == 1 {
			return r.ResolveFrom(source, typeArgs[0], typeBindings)
		}

		if typeBindings != nil {
			resolvedArgs := make([]string, len(typeArgs))
			for i, arg := range typeArgs {
				// A generic argument keeps its own arguments: the model of
				// PageData<H> is named just PageData, so spelling the argument
				// by the model's name would lose what H is bound to.
				if strings.Contains(arg, "<") {
					resolvedArgs[i] = bindTypeParams(arg, typeBindings)
					continue
				}
				resolvedModel := r.ResolveFrom(source, arg, typeBindings)
				if resolvedModel != nil {
					resolvedArgs[i] = resolvedModel.TypeName
					if resolvedModel.Kind == model.KindSingle && resolvedModel.TypeName != arg {
						resolvedArgs[i] = arg
					}
					// A composite model (array/map/object) keeps its structure in
					// the model, not in TypeName: `array` and `map` are synthetic
					// placeholders. Binding a type parameter to one of them feeds
					// a placeholder back into Resolve and destroys the argument.
					//
					// easy-yapi avoids this structurally: its generic context maps
					// a parameter to a ResolvedType, so `List<Foo>` is bound as the
					// structure ClassType(List, [Foo]) and never re-parsed as a
					// name. This resolver carries type arguments as strings, so the
					// equivalent rule is to keep the source spelling for composites
					// and only substitute names.
					if resolvedModel.IsArray() || resolvedModel.IsMap() {
						resolvedArgs[i] = arg
					}
				} else {
					resolvedArgs[i] = arg
				}
			}
			typeArgs = resolvedArgs
		}

		if collectionTypes[baseName] {
			if len(typeArgs) > 0 {
				itemModel := r.ResolveFrom(source, typeArgs[0], typeBindings)
				return model.ArrayModel(itemModel)
			}
			return model.ArrayModel(model.NullModel())
		}

		if mapTypes[baseName] {
			keyModel := model.SingleModel(model.JsonTypeString)
			valueModel := model.NullModel()
			if len(typeArgs) >= 2 {
				valueModel = r.ResolveFrom(source, typeArgs[1], typeBindings)
			} else if len(typeArgs) == 1 {
				valueModel = r.ResolveFrom(source, typeArgs[0], typeBindings)
			}
			return model.MapModel(keyModel, valueModel)
		}
	}

	name := r.resolveName(source, baseName)

	if r.dependencyResolver != nil {
		typeArgs = r.qualifyTypeArgs(source, typeArgs)
	}

	if class, found := r.lookup(name); found {
		return r.resolveClass(class, name, typeArgs, typeBindings)
	}

	// An enum of the scanned sources is already known; the JSON view just
	// does not see it, and a dependency's copy of it would not be seen either.
	// A name that is still unqualified here found no class in the packages
	// resolveName tried; it is asked as written for a resolver that knows
	// classes by their simple names, unless it is one that cannot be a class of
	// a library (see mayBeDependencyClass).
	if r.dependencyResolver != nil && (strings.Contains(name, ".") || r.mayBeDependencyClass(name)) && !r.isRegisteredEnum(name) {
		if depClass := r.dependencyClass(name); depClass != nil {
			r.registerDependency(*depClass, name)
			if r.sees(*depClass) {
				return r.resolveClass(*depClass, name, typeArgs, typeBindings)
			}
		}
	}

	// No class body is available. Well-known JDK value types are still scalars
	// on the wire, so answer for them instead of reporting a failure.
	// easy-yapi reaches the same answer with `json.rule.convert` rules over the
	// qualified name plus an IrType lookup over the simple name.
	if jsonType, ok := jdkQualifiedScalars[name]; ok {
		return model.SingleModel(jsonType)
	}
	if jsonType, ok := jdkScalarTypes[simpleTypeName(name)]; ok {
		return model.SingleModel(jsonType)
	}
	// `java.lang.Object` carries no fields, so an empty object is its true
	// shape rather than an opaque scalar.
	if simpleTypeName(name) == "object" {
		return model.EmptyObject()
	}

	r.recordUnresolved(rawType)
	return model.SingleModel(rawType)
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

// resolveClass expands one declaration into an object model.
func (r *TypeResolver) resolveClass(class parser.Class, key string, typeArgs []string, typeBindings map[string]string) *model.ObjectModel {
	// Only the binding view reaches an enum: a request parameter names one
	// of its constants.
	if class.IsEnum {
		return model.SingleModel(model.JsonTypeString)
	}
	if r.resolving[key] {
		return model.RefModel(class.Name)
	}

	if class.IsInterface {
		if impl, implKey, found := r.findImplementation(class, key); found {
			return r.resolveClass(impl, implKey, nil, nil)
		}
		return model.RefModel(class.Name)
	}

	if isMapSubclass(class) {
		return r.resolveMapSubclass(class, typeArgs, typeBindings)
	}

	r.resolving[key] = true
	defer func() { delete(r.resolving, key) }()

	localBindings := make(map[string]string)
	for i, tp := range class.TypeParameters {
		if i < len(typeArgs) {
			localBindings[tp] = typeArgs[i]
		}
	}
	for k, v := range typeBindings {
		if _, exists := localBindings[k]; !exists {
			localBindings[k] = v
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
	fieldModel := r.ResolveFrom(source, f.Type, localBindings)
	fm := &model.FieldModel{
		Model:    fieldModel,
		Required: r.fieldRequired(source, f.Annotations),
	}
	// In the binding view, an enum field is one of the enum's constants.
	if r.bindingView {
		if enum, ok := r.enumClass(source, f.Type); ok {
			fm.Options = EnumOptions(enum)
		}
	}
	javadoc.FieldDocumentationFor(f.Annotations, f.JavaDoc).Normalize().ApplyTo(fm)
	// Values the field declares in its own way win over the ones its doc
	// points at.
	if len(fm.Options) == 0 {
		fm.Options = r.seeOptions(source, f.RawJavaDoc)
	}
	if r.isUnboundTypeParam(f.Type, localBindings, typeParamSet) {
		fm.Generic = true
	} else if fieldModel != nil && fieldModel.Kind == model.KindSingle && r.allTypeParams[fieldModel.TypeName] {
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
		_, declared := r.lookup(bound)
		return r.allTypeParams[bound] && !declared
	}
	return true
}

func (r *TypeResolver) resolveInheritedFields(class parser.Class, localBindings map[string]string, obj *model.ObjectModel) {
	if class.SuperClass == "" {
		return
	}

	superClass, found := r.Class(class, class.SuperClass)
	if !found {
		return
	}

	superBindings := make(map[string]string)
	for i, tp := range superClass.TypeParameters {
		if i < len(class.SuperClassTypeArgs) {
			resolvedArg := class.SuperClassTypeArgs[i]
			if bound, ok := localBindings[resolvedArg]; ok {
				resolvedArg = bound
			}
			superBindings[tp] = resolvedArg
		}
	}
	for k, v := range localBindings {
		if _, exists := superBindings[k]; !exists {
			superBindings[k] = v
		}
	}

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

// qualifyTypeArgs spells each type argument by the qualified name it denotes
// in source. An argument is bound to a type parameter and resolved later,
// inside the generic class, where source's imports no longer apply: for
// `Wrapper<PageData<Order>>` written in a controller, PageData would be looked
// up among Wrapper's imports, and a dependency class could only be asked for by
// its simple name. A declaration of the scanned sources, a single-type import,
// and a dependency class that the package of source or an import-on-demand
// declares (see resolveName) qualify a name; JDK and Kotlin types stay as
// written, since the resolver knows collections and scalars by their simple
// names.
func (r *TypeResolver) qualifyTypeArgs(source parser.Class, args []string) []string {
	if len(args) == 0 {
		return args
	}
	out := make([]string, len(args))
	for i, arg := range args {
		out[i] = r.qualifyTypeName(source, strings.TrimSpace(arg))
	}
	return out
}

func (r *TypeResolver) qualifyTypeName(source parser.Class, typeName string) string {
	if typeName == "" {
		return typeName
	}
	if isWildcard(typeName) {
		if bound, ok := wildcardUpperBound(typeName); ok {
			return "? extends " + r.qualifyTypeName(source, bound)
		}
		return typeName
	}
	suffix := ""
	for strings.HasSuffix(typeName, "[]") {
		typeName = strings.TrimSuffix(typeName, "[]")
		suffix += "[]"
	}
	base, args := parseGenericType(typeName)
	if !strings.Contains(base, ".") {
		if key := r.resolveName(source, base); key != base && !isPlatformName(key) {
			base = key
		}
	}
	if len(args) > 0 {
		base += "<" + strings.Join(r.qualifyTypeArgs(source, args), ", ") + ">"
	}
	return base + suffix
}

// isPlatformName reports whether a qualified name belongs to the JDK or the
// Kotlin standard library.
func isPlatformName(name string) bool {
	for _, prefix := range []string{"java.", "javax.", "kotlin."} {
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

// bindTypeParams spells a type with the type parameters it mentions replaced
// by their bindings: under H=Order, `PageData<H>` is `PageData<Order>` and
// `Map<String, H[]>` is `Map<String, Order[]>`. Only names are substituted;
// the structure of the spelling is kept.
func bindTypeParams(typeName string, bindings map[string]string) string {
	typeName = strings.TrimSpace(typeName)
	if isWildcard(typeName) {
		rest := strings.TrimSpace(strings.TrimPrefix(typeName, "?"))
		for _, bound := range []string{"extends ", "super "} {
			if strings.HasPrefix(rest, bound) {
				return "? " + bound + bindTypeParams(strings.TrimPrefix(rest, bound), bindings)
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
		return base + suffix
	}
	bound := make([]string, len(args))
	for i, arg := range args {
		bound[i] = bindTypeParams(arg, bindings)
	}
	return base + "<" + strings.Join(bound, ", ") + ">" + suffix
}

func isMapSubclass(class parser.Class) bool {
	if class.SuperClass == "" {
		return false
	}
	baseName, _ := parseGenericType(class.SuperClass)
	return mapTypes[baseName]
}

// findImplementation returns the class in the scanned sources that implements
// the interface registered under key, preferring a declaration whose own
// implements clause resolves to that interface.
func (r *TypeResolver) findImplementation(iface parser.Class, key string) (parser.Class, string, bool) {
	keys := make([]string, 0, len(r.classRegistry))
	for candidate := range r.classRegistry {
		keys = append(keys, candidate)
	}
	sort.Strings(keys) // stable pick when several classes implement the interface

	for _, candidate := range keys {
		class := r.classRegistry[candidate]
		if class.IsInterface || !r.sees(class) {
			continue
		}
		for _, ifaceName := range class.Interfaces {
			if r.resolveSimpleName(class, ifaceName) == key || ifaceName == iface.Name {
				return class, candidate, true
			}
		}
	}
	return parser.Class{}, "", false
}

func (r *TypeResolver) resolveMapSubclass(class parser.Class, typeArgs []string, typeBindings map[string]string) *model.ObjectModel {
	keyModel := model.SingleModel(model.JsonTypeString)

	localBindings := make(map[string]string)
	for i, tp := range class.TypeParameters {
		if i < len(typeArgs) {
			localBindings[tp] = typeArgs[i]
		}
	}
	for k, v := range typeBindings {
		if _, exists := localBindings[k]; !exists {
			localBindings[k] = v
		}
	}

	var valueModel *model.ObjectModel
	if len(class.TypeParameters) > 0 {
		firstTypeParam := class.TypeParameters[0]
		if resolved, ok := localBindings[firstTypeParam]; ok {
			valueModel = r.ResolveFrom(class, resolved, localBindings)
		} else {
			valueModel = r.ResolveFrom(class, firstTypeParam, localBindings)
		}
	} else if len(class.SuperClassTypeArgs) >= 2 {
		valueModel = r.ResolveFrom(class, class.SuperClassTypeArgs[1], typeBindings)
	} else {
		valueModel = model.NullModel()
	}

	return &model.ObjectModel{
		Kind:       model.KindMap,
		TypeName:   class.Name,
		KeyModel:   keyModel,
		ValueModel: valueModel,
	}
}
