package resolver

import (
	"sort"
	"strings"

	collector "github.com/tangcent/apilot/api-collector"
	javadoc "github.com/tangcent/apilot/api-collector-java/doc"
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
	allTypeParams      map[string]bool
	resolving          map[string]bool
	dependencyResolver DependencyResolver
	unresolved         *collector.UnresolvedSet
}

func NewTypeResolver(classes []parser.Class) *TypeResolver {
	r := &TypeResolver{
		classRegistry: make(map[string]parser.Class, len(classes)),
		bySimpleName:  make(map[string][]string),
		byFile:        make(map[string][]string),
		allTypeParams: make(map[string]bool),
		resolving:     make(map[string]bool),
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

// register indexes a declaration under key. Classes resolved from
// dependencies are registered under the name that was looked up, so a
// subsequent lookup of the same name is answered from the registry.
func (r *TypeResolver) register(c parser.Class, key string) {
	r.classRegistry[key] = c
	r.bySimpleName[c.Name] = appendKey(r.bySimpleName[c.Name], key)
	if c.FilePath != "" {
		r.byFile[c.FilePath] = appendKey(r.byFile[c.FilePath], key)
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
// It applies the same precedence as Resolve, without expanding the fields.
func (r *TypeResolver) Class(source parser.Class, name string) (parser.Class, bool) {
	c, found := r.classRegistry[r.resolveSimpleName(source, name)]
	return c, found
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
	if name == "" || strings.Contains(name, ".") {
		return name
	}

	// 1. Same compilation unit: a class declared in the same file.
	for _, key := range r.byFile[source.FilePath] {
		if r.classRegistry[key].Name == name {
			return key
		}
	}

	// 2. Single-type import. The import names the type even when its body is
	// not in the scanned sources, so it is returned without a registry check.
	for _, imp := range source.Imports {
		if imp.Static || imp.OnDemand {
			continue
		}
		if lastSegment(imp.Path) == name {
			return imp.Path
		}
	}

	// 3. Same package.
	if source.Package != "" {
		if key := source.Package + "." + name; r.isRegistered(key) {
			return key
		}
	}

	// 4. Import-on-demand.
	for _, imp := range source.Imports {
		if imp.Static || !imp.OnDemand {
			continue
		}
		if key := imp.Path + "." + name; r.isRegistered(key) {
			return key
		}
	}

	// 5. Nothing in scope declared the name. Sources parsed without import
	// information (a single file, a dependency source) can only offer the
	// declarations themselves, so accept the only one — or the first of
	// several, in a stable order, which is what an ambiguous reference would
	// have to fall back to anyway.
	if keys := r.bySimpleName[name]; len(keys) > 0 {
		return keys[0]
	}

	return name
}

func (r *TypeResolver) isRegistered(key string) bool {
	_, found := r.classRegistry[key]
	return found
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
}

func (r *TypeResolver) SetCollectorDependencyResolver(cdr collector.DependencyResolver) {
	r.dependencyResolver = &collectorDependencyAdapter{resolver: cdr}
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

	name := r.resolveSimpleName(source, baseName)

	if class, found := r.classRegistry[name]; found {
		return r.resolveClass(class, name, typeArgs, typeBindings)
	}

	if r.dependencyResolver != nil {
		if depClass := r.dependencyResolver.ResolveClass(name); depClass != nil {
			r.register(*depClass, name)
			return r.resolveClass(*depClass, name, typeArgs, typeBindings)
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

// resolveClass expands one declaration into an object model.
func (r *TypeResolver) resolveClass(class parser.Class, key string, typeArgs []string, typeBindings map[string]string) *model.ObjectModel {
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

	fields := make(map[string]*model.FieldModel, len(class.Fields))
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
		fields[f.Name] = fm
	}

	r.resolveInheritedFields(class, localBindings, fields)

	return &model.ObjectModel{
		Kind:     model.KindObject,
		TypeName: class.Name,
		Fields:   fields,
	}
}

func (r *TypeResolver) resolveField(source parser.Class, f parser.Field, localBindings map[string]string, typeParamSet map[string]bool) *model.FieldModel {
	fieldModel := r.ResolveFrom(source, f.Type, localBindings)
	fm := &model.FieldModel{
		Model:    fieldModel,
		Required: true,
	}
	for _, ann := range f.Annotations {
		if ann.Name == "Nullable" || ann.Name == "Null" {
			fm.Required = false
		}
	}
	javadoc.FieldDocumentationFor(f.Annotations, f.JavaDoc).Normalize().ApplyTo(fm)
	if r.isUnboundTypeParam(f.Type, localBindings, typeParamSet) {
		fm.Generic = true
	} else if fieldModel != nil && fieldModel.Kind == model.KindSingle && r.allTypeParams[fieldModel.TypeName] {
		fm.Generic = true
	}
	return fm
}

func (r *TypeResolver) isUnboundTypeParam(typeName string, localBindings map[string]string, typeParamSet map[string]bool) bool {
	if typeParamSet == nil {
		return false
	}
	if !typeParamSet[typeName] {
		return false
	}
	if bound, exists := localBindings[typeName]; exists {
		return r.allTypeParams[bound] && r.classRegistry[bound].Name == ""
	}
	return true
}

func (r *TypeResolver) resolveInheritedFields(class parser.Class, localBindings map[string]string, fields map[string]*model.FieldModel) {
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

	for _, f := range superClass.Fields {
		if _, exists := fields[f.Name]; !exists {
			// The field type is written in the superclass's own compilation
			// unit, so that is the scope its simple names resolve in.
			fm := r.resolveField(superClass, f, superBindings, superTypeParamSet)
			fields[f.Name] = fm
		}
	}

	r.resolveInheritedFields(superClass, superBindings, fields)
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
		if class.IsInterface {
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
