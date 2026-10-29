// Package resolver adapts Kotlin declarations to the Java type resolver so a
// structurally identical Kotlin source and Java source produce the same schema.
package resolver

import (
	"sort"
	"strings"

	collector "github.com/tangcent/apilot/api-collector"
	javaparser "github.com/tangcent/apilot/api-collector-java/parser"
	javaresolver "github.com/tangcent/apilot/api-collector-java/resolver"
	ktparser "github.com/tangcent/apilot/api-collector-kotlin/parser"
	model "github.com/tangcent/apilot/api-model"
)

// Resolver wraps the Java type resolver with Kotlin spelling and enum
// handling.
type Resolver struct {
	classes    []ktparser.Class
	byKey      map[string]*ktparser.Class
	bySimple   map[string][]string
	byFile     map[string][]string
	enumValues map[string][]string
	tr         *javaresolver.TypeResolver
	unresolved *collector.UnresolvedSet
}

// New builds a resolver over all Kotlin declarations in the project.
func New(classes []ktparser.Class) *Resolver {
	r := &Resolver{
		classes:    classes,
		byKey:      make(map[string]*ktparser.Class, len(classes)),
		bySimple:   make(map[string][]string),
		byFile:     make(map[string][]string),
		enumValues: make(map[string][]string),
		unresolved: collector.NewUnresolvedSet(),
	}

	var javaClasses []javaparser.Class
	for i := range classes {
		class := &classes[i]
		key := qualifiedName(*class)
		r.byKey[key] = class
		r.bySimple[class.Name] = appendUnique(r.bySimple[class.Name], key)
		if class.FilePath != "" {
			r.byFile[class.FilePath] = appendUnique(r.byFile[class.FilePath], key)
		}
		if class.IsEnum {
			r.enumValues[class.Name] = class.EnumValues
			r.enumValues[key] = class.EnumValues
			// Enums are rendered as string options, not as empty objects.
			continue
		}
		javaClasses = append(javaClasses, toJavaClass(*class))
	}

	r.tr = javaresolver.NewTypeResolver(javaClasses)
	r.tr.SetUnresolved(r.unresolved)
	return r
}

// Unresolved returns type names the resolver could not expand, with enums
// filtered out: an enum is intentionally rendered as a string, so it is not a
// missing type.
func (r *Resolver) Unresolved() map[string]int {
	counts := r.unresolved.Counts()
	for name := range counts {
		if _, ok := r.enumValues[lastSegment(name)]; ok {
			delete(counts, name)
		}
	}
	return counts
}

// IsEnum reports whether name denotes an enum class.
func (r *Resolver) IsEnum(name string) bool {
	_, ok := r.enumValues[lastSegment(name)]
	return ok
}

// EnumValues returns the entries of an enum class.
func (r *Resolver) EnumValues(name string) []string {
	return r.enumValues[lastSegment(name)]
}

// ResolveFrom expands a Kotlin type spelling from the given source class.
func (r *Resolver) ResolveFrom(source *ktparser.Class, rawType string, bindings map[string]string) *model.ObjectModel {
	if rawType == "" {
		return model.NullModel()
	}
	javaSource := toJavaClass(*source)
	// An empty but non-nil binding map makes the shared resolver rewrite type
	// arguments from their resolved model (see resolver.ResolveFrom), which
	// flattens a nested generic such as PageResult<User> to just "PageResult"
	// and loses the argument. Pass nil when there is nothing to bind.
	var normalizedBindings map[string]string
	if len(bindings) > 0 {
		normalizedBindings = make(map[string]string, len(bindings))
		for name, value := range bindings {
			normalizedBindings[name] = NormalizeModelType(value)
		}
	}
	resolved := r.tr.ResolveFrom(javaSource, NormalizeModelType(rawType), normalizedBindings)
	r.applyEnums(resolved)
	return resolved
}

// FindClass resolves a Kotlin class in the same precedence order the Java
// resolver uses for simple names.
func (r *Resolver) FindClass(source *ktparser.Class, name string) (*ktparser.Class, bool) {
	if name == "" {
		return nil, false
	}
	if strings.Contains(name, ".") {
		c, ok := r.byKey[name]
		return c, ok
	}
	for _, key := range r.byFile[source.FilePath] {
		if c := r.byKey[key]; c != nil && c.Name == name {
			return c, true
		}
	}
	for _, imp := range source.Imports {
		if !imp.OnDemand && lastSegment(imp.Path) == name {
			if c, ok := r.byKey[imp.Path]; ok {
				return c, true
			}
		}
	}
	if key := source.Package + "." + name; source.Package != "" {
		if c, ok := r.byKey[key]; ok {
			return c, true
		}
	}
	for _, imp := range source.Imports {
		if imp.OnDemand {
			if c, ok := r.byKey[imp.Path+"."+name]; ok {
				return c, true
			}
		}
	}
	if keys := r.bySimple[name]; len(keys) == 1 {
		return r.byKey[keys[0]], true
	}
	if keys := r.bySimple[name]; len(keys) > 1 {
		sort.Strings(keys)
		return r.byKey[keys[0]], true
	}
	return nil, false
}

func (r *Resolver) applyEnums(m *model.ObjectModel) {
	if m == nil {
		return
	}
	switch m.Kind {
	case model.KindSingle:
		if r.IsEnum(m.TypeName) {
			m.TypeName = model.JsonTypeString
		}
	case model.KindObject:
		for _, field := range m.Fields {
			if field == nil || field.Model == nil {
				continue
			}
			if field.Model.Kind == model.KindSingle {
				if values := r.EnumValues(field.Model.TypeName); len(values) > 0 {
					field.Model = model.SingleModel(model.JsonTypeString)
					field.Options = enumOptions(values)
					continue
				}
			}
			r.applyEnums(field.Model)
		}
	case model.KindArray:
		r.applyEnums(m.Items)
	case model.KindMap:
		r.applyEnums(m.KeyModel)
		r.applyEnums(m.ValueModel)
	}
}

func enumOptions(values []string) []model.FieldOption {
	out := make([]model.FieldOption, 0, len(values))
	for _, value := range values {
		out = append(out, model.FieldOption{Value: value})
	}
	return out
}

func toJavaClass(class ktparser.Class) javaparser.Class {
	out := javaparser.Class{
		Name:           class.Name,
		Package:        class.Package,
		FilePath:       class.FilePath,
		JavaDoc:        class.KDoc,
		Annotations:    toJavaAnnotations(class.Annotations),
		TypeParameters: append([]string(nil), class.TypeParameters...),
		SuperClass:     normalizeClassName(class.SuperClass),
		Interfaces:     make([]string, 0, len(class.Interfaces)),
		IsInterface:    class.IsInterface || class.IsSealed,
	}
	for _, arg := range class.SuperClassTypeArgs {
		out.SuperClassTypeArgs = append(out.SuperClassTypeArgs, NormalizeModelType(arg))
	}
	for _, iface := range class.Interfaces {
		out.Interfaces = append(out.Interfaces, normalizeClassName(iface))
	}
	for _, imp := range class.Imports {
		out.Imports = append(out.Imports, javaparser.Import{Path: imp.Path, OnDemand: imp.OnDemand})
	}
	for _, prop := range class.Fields {
		if annotationValue(prop.Annotations, "JsonIgnore") == "true" || hasAnnotation(prop.Annotations, "JsonIgnore") {
			continue
		}
		name := prop.Name
		if jsonName := propertyName(prop.Annotations); jsonName != "" {
			name = jsonName
		}
		out.Fields = append(out.Fields, javaparser.Field{
			Name:        name,
			Type:        NormalizeModelType(prop.Type),
			Annotations: fieldAnnotations(prop.Annotations),
			JavaDoc:     prop.KDoc,
			IsFinal:     !prop.IsMutable,
		})
	}
	// Methods are deliberately not copied: Kotlin data-class getters,
	// componentN, copy and toString are compiler-generated. Properties are the
	// only JSON fields.
	return out
}

func normalizeClassName(name string) string {
	if name == "" {
		return ""
	}
	base, args := splitGenericType(name)
	if args == nil {
		return base
	}
	for i := range args {
		args[i] = NormalizeModelType(args[i])
	}
	return base + "<" + strings.Join(args, ",") + ">"
}

// NormalizeModelType maps Kotlin spelling to the Java names the shared resolver
// understands. Enum names are left untouched so enum options can be attached
// after resolution.
func NormalizeModelType(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if raw == "*" {
		return "?"
	}
	base, args := splitGenericType(raw)
	switch lastSegment(base) {
	case "Int", "Byte", "Short", "UByte", "UShort", "UInt":
		base = "Integer"
	case "Long", "ULong":
		base = "Long"
	case "Float":
		base = "Float"
	case "Double":
		base = "Double"
	case "Boolean":
		base = "Boolean"
	case "Char":
		base = "String"
	case "String":
		base = "String"
	case "Any":
		base = "Object"
	case "Unit", "Nothing":
		base = "Void"
	case "Number":
		base = "Double"
	case "Integer", "int":
		base = "Integer"
	case "Void", "void":
		base = "Void"
	case "Object":
		base = "Object"
	case "List", "Set", "Collection", "ArrayList", "LinkedList", "HashSet", "LinkedHashSet":
		base = lastSegment(base)
	case "Map", "HashMap", "LinkedHashMap", "TreeMap", "SortedMap":
		base = "Map"
	case "ResponseEntity", "Mono", "Flux":
		base = lastSegment(base)
	case "MutableList", "MutableCollection", "Iterable", "Sequence":
		base = "List"
	case "Array":
		base = "List"
	case "IntArray", "ByteArray", "ShortArray", "UByteArray", "UShortArray", "UIntArray":
		base = "List"
		if args == nil {
			args = []string{"Integer"}
		}
	case "LongArray", "ULongArray":
		base = "List"
		if args == nil {
			args = []string{"Long"}
		}
	case "FloatArray":
		base = "List"
		if args == nil {
			args = []string{"Float"}
		}
	case "DoubleArray":
		base = "List"
		if args == nil {
			args = []string{"Double"}
		}
	case "BooleanArray":
		base = "List"
		if args == nil {
			args = []string{"Boolean"}
		}
	case "CharArray":
		base = "List"
		if args == nil {
			args = []string{"String"}
		}
	case "MutableSet":
		base = "Set"
	case "MutableMap":
		base = "Map"
	}
	if args == nil {
		return base
	}
	for i := range args {
		args[i] = NormalizeModelType(args[i])
	}
	return base + "<" + strings.Join(args, ",") + ">"
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

func fieldAnnotations(annotations []ktparser.Annotation) []javaparser.Annotation {
	out := toJavaAnnotations(annotations)
	for _, ann := range annotations {
		if ann.Name == "JsonProperty" && ann.Params["required"] == "false" {
			out = append(out, javaparser.Annotation{Name: "Nullable"})
		}
	}
	return out
}

func propertyName(annotations []ktparser.Annotation) string {
	for _, ann := range annotations {
		if ann.Name == "JsonProperty" {
			if name := ann.Params["value"]; name != "" {
				return name
			}
		}
	}
	return ""
}

func hasAnnotation(annotations []ktparser.Annotation, name string) bool {
	for _, ann := range annotations {
		if ann.Name == name {
			return true
		}
	}
	return false
}

func annotationValue(annotations []ktparser.Annotation, name string) string {
	base := lastSegment(name)
	param := ""
	if idx := strings.Index(name, "."); idx >= 0 {
		param = name[idx+1:]
	}
	for _, ann := range annotations {
		if ann.Name != base {
			continue
		}
		if param == "" {
			if v, ok := ann.Params["value"]; ok {
				return v
			}
			return "true"
		}
		if v, ok := ann.Params[param]; ok {
			return v
		}
	}
	return ""
}

func qualifiedName(class ktparser.Class) string {
	if class.Package == "" {
		return class.Name
	}
	return class.Package + "." + class.Name
}

func lastSegment(name string) string {
	if idx := strings.LastIndex(name, "."); idx >= 0 {
		return name[idx+1:]
	}
	return name
}

func appendUnique(list []string, value string) []string {
	for _, item := range list {
		if item == value {
			return list
		}
	}
	return append(list, value)
}

func splitGenericType(raw string) (string, []string) {
	raw = strings.TrimSpace(raw)
	open := strings.Index(raw, "<")
	if open < 0 || !strings.HasSuffix(raw, ">") {
		return raw, nil
	}
	base := strings.TrimSpace(raw[:open])
	return base, splitTopLevel(raw[open+1 : len(raw)-1])
}

func splitTopLevel(raw string) []string {
	var out []string
	depth := 0
	start := 0
	for i, r := range raw {
		switch r {
		case '<', '(', '[':
			depth++
		case '>', ')', ']':
			depth--
		case ',':
			if depth == 0 {
				out = append(out, strings.TrimSpace(raw[start:i]))
				start = i + 1
			}
		}
	}
	out = append(out, strings.TrimSpace(raw[start:]))
	return out
}
