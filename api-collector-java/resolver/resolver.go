package resolver

import (
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

type DependencyResolver interface {
	ResolveClass(className string) *parser.Class
}

type TypeResolver struct {
	classRegistry      map[string]parser.Class
	allTypeParams      map[string]bool
	resolving          map[string]bool
	dependencyResolver DependencyResolver
	unresolved         *collector.UnresolvedSet
}

func NewTypeResolver(classes []parser.Class) *TypeResolver {
	registry := make(map[string]parser.Class, len(classes))
	allTypeParams := make(map[string]bool)
	for _, c := range classes {
		registry[c.Name] = c
		for _, tp := range c.TypeParameters {
			allTypeParams[tp] = true
		}
	}
	return &TypeResolver{
		classRegistry: registry,
		allTypeParams: allTypeParams,
		resolving:     make(map[string]bool),
	}
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

func (r *TypeResolver) Resolve(rawType string, typeBindings map[string]string) *model.ObjectModel {
	if rawType == "" {
		return model.NullModel()
	}

	if typeBindings != nil {
		if bound, ok := typeBindings[rawType]; ok {
			return r.Resolve(bound, nil)
		}
	}

	if jsonType, ok := primitiveTypes[rawType]; ok {
		return model.SingleModel(jsonType)
	}

	baseName, typeArgs := parseGenericType(rawType)

	if baseName != rawType {
		if typeBindings != nil {
			resolvedArgs := make([]string, len(typeArgs))
			for i, arg := range typeArgs {
				resolvedModel := r.Resolve(arg, typeBindings)
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
				itemModel := r.Resolve(typeArgs[0], typeBindings)
				return model.ArrayModel(itemModel)
			}
			return model.ArrayModel(model.NullModel())
		}

		if mapTypes[baseName] {
			keyModel := model.SingleModel(model.JsonTypeString)
			valueModel := model.NullModel()
			if len(typeArgs) >= 2 {
				valueModel = r.Resolve(typeArgs[1], typeBindings)
			} else if len(typeArgs) == 1 {
				valueModel = r.Resolve(typeArgs[0], typeBindings)
			}
			return model.MapModel(keyModel, valueModel)
		}
	}

	if class, found := r.classRegistry[baseName]; found {
		if r.resolving[baseName] {
			return model.RefModel(baseName)
		}

		if class.IsInterface {
			if impl := r.findImplementation(class.Name, typeArgs); impl != nil {
				return impl
			}
			return model.RefModel(baseName)
		}

		if isMapSubclass(class) {
			return r.resolveMapSubclass(class, typeArgs, typeBindings)
		}

		r.resolving[baseName] = true
		defer func() { delete(r.resolving, baseName) }()

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
			if f.IsStatic || f.IsFinal {
				continue
			}
			fm := r.resolveField(f, localBindings, typeParamSet)
			fields[f.Name] = fm
		}

		r.resolveInheritedFields(class, localBindings, fields)

		return &model.ObjectModel{
			Kind:     model.KindObject,
			TypeName: baseName,
			Fields:   fields,
		}
	}

	if r.dependencyResolver != nil {
		if depClass := r.dependencyResolver.ResolveClass(baseName); depClass != nil {
			r.classRegistry[baseName] = *depClass
			for _, tp := range depClass.TypeParameters {
				r.allTypeParams[tp] = true
			}
			return r.Resolve(rawType, typeBindings)
		}
	}

	// No class body is available. Well-known JDK value types are still scalars
	// on the wire, so answer for them instead of reporting a failure.
	// easy-yapi reaches the same answer with `json.rule.convert` rules over the
	// qualified name plus an IrType lookup over the simple name.
	if jsonType, ok := jdkQualifiedScalars[rawType]; ok {
		return model.SingleModel(jsonType)
	}
	if jsonType, ok := jdkScalarTypes[simpleTypeName(rawType)]; ok {
		return model.SingleModel(jsonType)
	}
	// `java.lang.Object` carries no fields, so an empty object is its true
	// shape rather than an opaque scalar.
	if simpleTypeName(rawType) == "object" {
		return model.EmptyObject()
	}

	r.recordUnresolved(rawType)
	return model.SingleModel(rawType)
}

func (r *TypeResolver) resolveField(f parser.Field, localBindings map[string]string, typeParamSet map[string]bool) *model.FieldModel {
	fieldModel := r.Resolve(f.Type, localBindings)
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

	superClass, found := r.classRegistry[class.SuperClass]
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
			fm := r.resolveField(f, superBindings, superTypeParamSet)
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

func (r *TypeResolver) findImplementation(interfaceName string, typeArgs []string) *model.ObjectModel {
	for _, class := range r.classRegistry {
		if class.IsInterface {
			continue
		}
		for _, iface := range class.Interfaces {
			if iface == interfaceName {
				return r.Resolve(class.Name, nil)
			}
		}
	}
	return nil
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
			valueModel = r.Resolve(resolved, localBindings)
		} else {
			valueModel = r.Resolve(firstTypeParam, localBindings)
		}
	} else if len(class.SuperClassTypeArgs) >= 2 {
		valueModel = r.Resolve(class.SuperClassTypeArgs[1], typeBindings)
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
