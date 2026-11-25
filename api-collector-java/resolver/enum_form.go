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
//   - @JsonValue is on a member of the enum, or on a method of an interface it
//     implements, however far up: Jackson merges the annotations of the methods an
//     enum overrides;
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
// naming a class other than the one that stands for none.
func namesSerializer(ann parser.Annotation) bool {
	for _, param := range []string{"using", "contentUsing", "converter", "contentConverter"} {
		value := strings.TrimSpace(ann.Params[param])
		if value != "" && !strings.HasSuffix(value, "None.class") {
			return true
		}
	}
	return false
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
	custom := enum.EnumJsonValue || enum.EnumJsonProperty || r.inheritsJsonValue(enum, key, map[string]bool{key: true})
	if !custom {
		format := formatOf(enum.Annotations)
		custom = format.custom || writesOtherThanName(format.shape)
	}
	r.customEnums[memo] = custom
	return custom
}

// inheritsJsonValue reports whether a type the class implements or extends, however
// far up, carries @JsonValue on a method: Jackson merges the annotations of the
// methods that an enum overrides, so it writes the value the method returns. The
// supertypes are named as the header of the class writes them, and one that only a
// dependency declares is read from it. key is the registry key of c, and visited
// holds the classes already looked at.
func (r *TypeResolver) inheritsJsonValue(c parser.Class, key string, visited map[string]bool) bool {
	named := c
	if c.IsEnum {
		// The superclass of an enum is java.lang.Enum, which a decompiler prints
		// and which carries no @JsonValue: it is not asked of the dependencies.
		named.SuperClass = ""
	}
	for _, name := range supertypeNames(named) {
		super, superKey, found := r.supertypeClass(c, key, name)
		if !found || visited[superKey] {
			continue
		}
		visited[superKey] = true
		if declaresJsonValue(super) || r.inheritsJsonValue(super, superKey, visited) {
			return true
		}
	}
	return false
}

// declaresJsonValue reports whether a method of the class carries @JsonValue. An
// enum inherits no field from a type it implements, and no constant of an
// interface is a value Jackson writes.
func declaresJsonValue(c parser.Class) bool {
	for _, m := range c.Methods {
		if parser.HasJsonValue(m.Annotations) {
			return true
		}
	}
	return false
}
