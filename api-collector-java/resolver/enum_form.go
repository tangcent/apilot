package resolver

import (
	"strings"

	"github.com/tangcent/apilot/api-collector-java/parser"
)

// The form Jackson writes an enum in. Most enums are written as the name of a
// constant, and a field typed with one is a string whose options are the names.
// Jackson writes an enum in another form, which no schema of its constants tells,
// when:
//
//   - @JsonValue is on a member of the enum, or the nearest @JsonValue on a
//     no-argument method an enum overrides is switched on. A declaration of
//     @JsonValue(false) wins over one further up the same method. A method with
//     parameters is a different member. A field is not that method: @JsonValue
//     on one does not turn the other off. Jackson also reads an annotation
//     meta-annotated @JacksonAnnotationsInside when it is retained at runtime,
//     and a @JsonFormat or @JsonSerialize an interface declares, the first
//     supertype that declares it;
//   - @JsonProperty gives a name to a constant;
//   - @JsonFormat gives the enum, or the field that holds it, the shape OBJECT,
//     ARRAY or a number;
//   - @JsonSerialize names a serializer or a converter for the enum, or for the
//     field that holds it.
//
// The JSON view exports such an enum by its type name, as it does any type whose
// declaration was not read. The binding view is not concerned: Spring binds a
// request parameter to an enum by the name of a constant, whatever its JSON form.

// fieldFormat is what the annotations of a field say about the enums it holds, a
// collection or an array of them included.
type fieldFormat struct {
	shape  string // the last segment of the shape of @JsonFormat, "" when there is none
	custom bool   // @JsonSerialize names a serializer or a converter
}

// formatOf reads the annotations of a field or of an enum.
func formatOf(annotations []parser.Annotation) fieldFormat {
	var format fieldFormat
	for _, ann := range annotations {
		switch lastSegment(ann.Name) {
		case "JsonFormat":
			format.shape = lastSegment(strings.TrimSpace(ann.Params["shape"]))
		case "JsonSerialize":
			format.custom = format.custom || namesSerializer(ann)
		}
	}
	return format
}

// namesSerializer reports whether an @JsonSerialize replaces the serializer of
// its target: with `using`, `contentUsing`, `converter` or `contentConverter`
// naming a class other than the one that stands for none. Kotlin writes that
// class `JsonSerializer.None::class`.
func namesSerializer(ann parser.Annotation) bool {
	for _, param := range []string{"using", "contentUsing", "converter", "contentConverter"} {
		value := strings.TrimSpace(ann.Params[param])
		if value != "" && !isNoneSerializer(value) {
			return true
		}
	}
	return false
}

// isNoneSerializer reports the class Jackson uses to say an annotation names
// no serializer: JsonSerializer.None, written as Java or as Kotlin sees it.
func isNoneSerializer(value string) bool {
	return strings.HasSuffix(value, "None.class") || strings.HasSuffix(value, "None::class")
}

// writesOtherThanName reports whether a shape of @JsonFormat makes Jackson write
// an enum as something else than the name of its constant.
func writesOtherThanName(shape string) bool {
	switch shape {
	case "OBJECT", "ARRAY", "NUMBER", "NUMBER_INT", "NUMBER_FLOAT":
		return true
	}
	return false
}

// writesConstantName reports whether Jackson writes enum as the name of its
// constant, in the field being resolved. What the enum declares about itself and
// what the annotations of the field say are both asked.
func (r *TypeResolver) writesConstantName(enum parser.Class) bool {
	return !r.writesCustomForm(enum) && !r.format.custom && !writesOtherThanName(r.format.shape)
}

// writesCustomForm reports whether the enum declares a form of its own, whatever
// field holds it.
func (r *TypeResolver) writesCustomForm(enum parser.Class) bool {
	key := qualifiedName(enum)
	memo := key + "\x00" + enum.FilePath
	if custom, known := r.customEnums[memo]; known {
		return custom
	}
	if r.customEnums == nil {
		r.customEnums = make(map[string]bool)
	}
	// Finding the interfaces of the enum looks names up, which may ask for this
	// enum again: until the answer is known it is the one a plain enum gets.
	r.customEnums[memo] = false
	custom := enum.EnumJsonProperty || r.writesJsonValue(enum, key) || r.writesInheritedFormat(enum, key)
	r.customEnums[memo] = custom
	return custom
}

// A @JsonValue on one member. Absent means the member does not declare it, so a
// supertype's declaration still applies. Off is @JsonValue(false): it wins over
// a declaration further up and the member is not a value.
const (
	valueAbsent = iota
	valueOff
	valueOn
)

// writesJsonValue reports whether Jackson writes the enum as the value of a
// member rather than as the name of a constant.
//
// For each no-argument instance method, the nearest declaration of @JsonValue
// wins: the enum's own, else the first supertype in source order that declares
// it, and that supertype's own supertypes before the next one. A method with
// parameters is not that method. A field is not a method either, so @JsonValue
// on a field does not turn a getter off and @JsonValue(false) on a getter does
// not turn a field off. Either one being on is enough.
//
// EnumJsonValue is the same fact for a declaration that records it and does not
// carry the member: a test, or a shape read before members were kept.
func (r *TypeResolver) writesJsonValue(enum parser.Class, key string) bool {
	if r.fieldValue(enum) == valueOn {
		return true
	}
	names := map[string]struct{}{}
	r.collectNoArgMethods(enum, key, map[string]bool{}, names)
	for name := range names {
		if r.nearestValue(enum, key, name, map[string]bool{}) == valueOn {
			return true
		}
	}
	return enum.EnumJsonValue && !r.recordsJsonValue(enum)
}

// writesInheritedFormat reports whether the nearest @JsonFormat writes the enum
// as something other than the name of a constant, or the nearest @JsonSerialize
// names a serializer. The two are independent: a shape of STRING does not clear
// a serializer, and a serializer of None does not clear a shape. The nearest
// declaration of each wins, including one that names nothing, which is how an
// enum's own @JsonFormat() or @JsonSerialize(using = None.class) wins over an
// interface.
func (r *TypeResolver) writesInheritedFormat(enum parser.Class, key string) bool {
	if shape, found := r.nearestShape(enum, key, map[string]bool{}); found && writesOtherThanName(shape) {
		return true
	}
	custom, _ := r.nearestSerializer(enum, key, map[string]bool{})
	return custom
}

// fieldValue is the @JsonValue of an instance field the enum itself declares.
// On wins over off when two fields disagree; a field does not inherit one.
func (r *TypeResolver) fieldValue(enum parser.Class) int {
	state := valueAbsent
	for _, field := range enum.Fields {
		if field.IsStatic {
			continue
		}
		switch jsonValueState(field.Annotations, r.expandBundles(enum, field.Annotations, map[string]bool{}, 0)) {
		case valueOn:
			return valueOn
		case valueOff:
			state = valueOff
		}
	}
	return state
}

// recordsJsonValue reports whether a member the enum carries declares @JsonValue,
// on or off. The flag EnumJsonValue is then the same fact and must not also be
// read, or an explicit false could not win over it.
func (r *TypeResolver) recordsJsonValue(enum parser.Class) bool {
	if r.fieldValue(enum) != valueAbsent {
		return true
	}
	for _, method := range enum.Methods {
		if method.IsStatic || len(method.Parameters) > 0 {
			continue
		}
		if jsonValueState(method.Annotations, r.expandBundles(enum, method.Annotations, map[string]bool{}, 0)) != valueAbsent {
			return true
		}
	}
	return false
}

// collectNoArgMethods adds the names of the no-argument instance methods of c
// and of every type it extends or implements.
func (r *TypeResolver) collectNoArgMethods(c parser.Class, key string, visited map[string]bool, names map[string]struct{}) {
	if visited[key] {
		return
	}
	visited[key] = true
	for _, method := range c.Methods {
		if method.IsStatic || len(method.Parameters) > 0 || method.Name == "" {
			continue
		}
		names[method.Name] = struct{}{}
	}
	r.eachSupertype(c, key, func(super parser.Class, superKey string) bool {
		r.collectNoArgMethods(super, superKey, visited, names)
		return false
	})
}

// nearestValue is the @JsonValue that wins for one no-argument method: the
// first declaration of it, starting at c.
func (r *TypeResolver) nearestValue(c parser.Class, key, name string, visited map[string]bool) int {
	if visited[key] {
		return valueAbsent
	}
	visited[key] = true
	if state, declared := r.declaredValue(c, name); declared {
		return state
	}
	found := valueAbsent
	r.eachSupertype(c, key, func(super parser.Class, superKey string) bool {
		if state := r.nearestValue(super, superKey, name, visited); state != valueAbsent {
			found = state
			return true
		}
		return false
	})
	return found
}

// declaredValue is the @JsonValue a class writes on its own no-argument method
// of that name. A method that is present without the annotation does not
// declare it.
func (r *TypeResolver) declaredValue(c parser.Class, name string) (int, bool) {
	state := valueAbsent
	declared := false
	for _, method := range c.Methods {
		if method.IsStatic || method.Name != name || len(method.Parameters) > 0 {
			continue
		}
		switch jsonValueState(method.Annotations, r.expandBundles(c, method.Annotations, map[string]bool{}, 0)) {
		case valueOn:
			return valueOn, true
		case valueOff:
			state = valueOff
			declared = true
		}
	}
	return state, declared
}

// nearestShape is the shape of the nearest @JsonFormat, and whether any type
// declares one.
func (r *TypeResolver) nearestShape(c parser.Class, key string, visited map[string]bool) (string, bool) {
	if visited[key] {
		return "", false
	}
	visited[key] = true
	if shape, found := jsonFormatShape(r.expandBundles(c, c.Annotations, map[string]bool{}, 0)); found {
		return shape, true
	}
	var shape string
	found := false
	r.eachSupertype(c, key, func(super parser.Class, superKey string) bool {
		if got, ok := r.nearestShape(super, superKey, visited); ok {
			shape, found = got, true
			return true
		}
		return false
	})
	return shape, found
}

// nearestSerializer reports whether the nearest @JsonSerialize names a
// serializer, and whether any type declares the annotation. A declaration that
// names none still counts as found, so it wins over one further up.
func (r *TypeResolver) nearestSerializer(c parser.Class, key string, visited map[string]bool) (bool, bool) {
	if visited[key] {
		return false, false
	}
	visited[key] = true
	if custom, found := jsonSerializerState(r.expandBundles(c, c.Annotations, map[string]bool{}, 0)); found {
		return custom, true
	}
	custom, found := false, false
	r.eachSupertype(c, key, func(super parser.Class, superKey string) bool {
		if got, ok := r.nearestSerializer(super, superKey, visited); ok {
			custom, found = got, true
			return true
		}
		return false
	})
	return custom, found
}

// eachSupertype calls fn for every type c extends or implements, in source
// order, and stops when fn returns true. The superclass of an enum is
// java.lang.Enum, which a decompiler prints and which carries none of these
// annotations, so it is not asked of the dependencies. A type only a dependency
// declares is read from it.
func (r *TypeResolver) eachSupertype(c parser.Class, key string, fn func(parser.Class, string) bool) {
	named := c
	if c.IsEnum {
		named.SuperClass = ""
	}
	for _, name := range supertypeNames(named) {
		super, superKey, found := r.supertypeClass(c, key, name)
		if !found {
			continue
		}
		if fn(super, superKey) {
			return
		}
	}
}

// jsonValueState is the @JsonValue written on a member. A declaration on the
// member itself wins over one a composed annotation unwraps to.
func jsonValueState(direct, expanded []parser.Annotation) int {
	if state, ok := declaredJsonValue(direct); ok {
		return state
	}
	if state, ok := declaredJsonValue(expanded); ok {
		return state
	}
	return valueAbsent
}

// declaredJsonValue reports the @JsonValue among annotations. On wins when one
// annotation switches it on and another switches it off.
func declaredJsonValue(annotations []parser.Annotation) (int, bool) {
	state := valueAbsent
	for _, ann := range annotations {
		if lastSegment(ann.Name) != "JsonValue" {
			continue
		}
		if strings.TrimSpace(ann.Params["value"]) != "false" {
			return valueOn, true
		}
		state = valueOff
	}
	return state, state != valueAbsent
}

// jsonFormatShape is the shape of the last @JsonFormat, and whether there is
// one. The last wins, as it does for a single declaration.
func jsonFormatShape(annotations []parser.Annotation) (string, bool) {
	shape := ""
	found := false
	for _, ann := range annotations {
		if lastSegment(ann.Name) != "JsonFormat" {
			continue
		}
		shape = lastSegment(strings.TrimSpace(ann.Params["shape"]))
		found = true
	}
	return shape, found
}

// jsonSerializerState reports whether an @JsonSerialize names a serializer,
// and whether there is one at all.
func jsonSerializerState(annotations []parser.Annotation) (bool, bool) {
	custom := false
	found := false
	for _, ann := range annotations {
		if lastSegment(ann.Name) != "JsonSerialize" {
			continue
		}
		found = true
		custom = custom || namesSerializer(ann)
	}
	return custom, found
}

// expandBundles unwraps annotations meta-annotated @JacksonAnnotationsInside
// whose retention is runtime, and the composed annotations those carry. Jackson
// does not see a composed annotation retained only in the class file. One it
// cannot be resolved to is left as written.
func (r *TypeResolver) expandBundles(owner parser.Class, annotations []parser.Annotation, seen map[string]bool, depth int) []parser.Annotation {
	if depth > 8 || len(annotations) == 0 {
		out := make([]parser.Annotation, len(annotations))
		copy(out, annotations)
		return out
	}
	if seen == nil {
		seen = map[string]bool{}
	}
	out := make([]parser.Annotation, 0, len(annotations))
	for _, ann := range annotations {
		out = append(out, ann)
		decl, key, ok := r.runtimeBundle(owner, ann, seen)
		if !ok {
			continue
		}
		seen[key] = true
		out = append(out, r.expandBundles(decl, decl.Annotations, seen, depth+1)...)
	}
	return out
}

// runtimeBundle returns the annotation type ann names when Jackson would unwrap
// it, and the key it is held under. Only a type the resolver already holds is
// considered: asking the dependencies would change which names a lookup asks
// for, and a composed annotation they do not show stays as written.
func (r *TypeResolver) runtimeBundle(owner parser.Class, ann parser.Annotation, seen map[string]bool) (parser.Class, string, bool) {
	switch lastSegment(ann.Name) {
	case "JsonValue", "JsonFormat", "JsonSerialize", "JacksonAnnotationsInside",
		"Retention", "Target", "Inherited", "Documented", "Repeatable",
		"NotNull", "Nullable", "NonNull", "NotBlank", "Override", "Deprecated",
		"SuppressWarnings", "Suppress", "Metadata":
		return parser.Class{}, "", false
	}
	key := ann.Name
	if !strings.Contains(key, ".") {
		var found bool
		key, found = r.inScope(owner, ann.Name, false)
		if !found {
			return parser.Class{}, "", false
		}
	}
	decl, ok := r.classRegistry[key]
	if !ok || decl.AnnotationRetention != "RUNTIME" || !hasSimpleName(decl.Annotations, "JacksonAnnotationsInside") {
		return parser.Class{}, "", false
	}
	if seen[key] {
		return parser.Class{}, "", false
	}
	return decl, key, true
}

// hasSimpleName reports whether annotations include one whose last segment is name.
func hasSimpleName(annotations []parser.Annotation, name string) bool {
	for _, ann := range annotations {
		if lastSegment(ann.Name) == name {
			return true
		}
	}
	return false
}
