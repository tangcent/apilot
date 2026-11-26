package resolver

import (
	"strings"

	"github.com/tangcent/apilot/api-collector-java/parser"
	model "github.com/tangcent/apilot/api-model"
)

// resolvePropertySchema resolves a field or a getter. Jackson writes a byte[]
// as Base64 and a char[] as one string, unless an annotation on the property
// replaces that serializer. The replacement is not a string whose contents the
// source states, so the array is kept, which is what main exports.
//
// @JsonSerialize(using) or converter, and @JsonRawValue, replace the serializer
// of the whole value. contentUsing and contentConverter replace only the
// content of a container; on a byte[] or a char[] itself they do nothing.
// @JsonFormat does not change either array.
func (r *TypeResolver) resolvePropertySchema(source parser.Class, rawType string, anns []parser.Annotation, bindings map[string]string) *model.ObjectModel {
	if r.bindingView {
		return r.ResolveFrom(source, rawType, bindings)
	}
	if replacesValueSerializer(anns) {
		return r.keepingBinaryArrays(func() *model.ObjectModel {
			return r.ResolveFrom(source, rawType, bindings)
		})
	}
	if replacesContentSerializer(anns) {
		return r.resolveReplacedContent(source, rawType, bindings)
	}
	return r.ResolveFrom(source, rawType, bindings)
}

func (r *TypeResolver) keepingBinaryArrays(fn func() *model.ObjectModel) *model.ObjectModel {
	previous := r.keepBinaryArray
	r.keepBinaryArray = true
	got := fn()
	r.keepBinaryArray = previous
	return got
}

// resolveReplacedContent resolves a container whose content serializer was
// replaced. A byte[] or a char[] is not a container of itself.
func (r *TypeResolver) resolveReplacedContent(source parser.Class, rawType string, bindings map[string]string) *model.ObjectModel {
	rawType = strings.TrimSpace(rawType)
	if element, ok := strings.CutSuffix(rawType, "[]"); ok {
		element = strings.TrimSpace(element)
		if element == "byte" || element == "char" {
			return r.ResolveFrom(source, rawType, bindings)
		}
		item := r.keepingBinaryArrays(func() *model.ObjectModel {
			return r.ResolveFrom(source, element, bindings)
		})
		return model.ArrayModel(item)
	}
	base, args := parseGenericType(rawType)
	if r.isCollectionType(source, base) {
		if len(args) == 0 {
			return model.ArrayModel(model.NullModel())
		}
		item := r.keepingBinaryArrays(func() *model.ObjectModel {
			return r.ResolveFrom(source, args[0], bindings)
		})
		return model.ArrayModel(item)
	}
	if r.isMapType(source, base) {
		value := model.NullModel()
		switch {
		case len(args) >= 2:
			value = r.keepingBinaryArrays(func() *model.ObjectModel {
				return r.ResolveFrom(source, args[1], bindings)
			})
		case len(args) == 1:
			value = r.keepingBinaryArrays(func() *model.ObjectModel {
				return r.ResolveFrom(source, args[0], bindings)
			})
		}
		return model.MapModel(model.SingleModel(model.JsonTypeString), value)
	}
	return r.ResolveFrom(source, rawType, bindings)
}

// replacesValueSerializer reports whether an annotation replaces the serializer
// of the property itself. A serializer named None is Jackson's way of saying
// the annotation names none.
func replacesValueSerializer(anns []parser.Annotation) bool {
	for _, ann := range anns {
		switch lastSegment(ann.Name) {
		case "JsonSerialize":
			if serializerArg(ann, "using") || serializerArg(ann, "converter") {
				return true
			}
		case "JsonRawValue":
			if strings.TrimSpace(ann.Params["value"]) != "false" {
				return true
			}
		}
	}
	return false
}

// replacesContentSerializer reports whether an annotation replaces the
// serializer of the content of a container.
func replacesContentSerializer(anns []parser.Annotation) bool {
	for _, ann := range anns {
		if lastSegment(ann.Name) != "JsonSerialize" {
			continue
		}
		if serializerArg(ann, "contentUsing") || serializerArg(ann, "contentConverter") {
			return true
		}
	}
	return false
}

func serializerArg(ann parser.Annotation, key string) bool {
	value := strings.TrimSpace(ann.Params[key])
	if value == "" {
		return false
	}
	return !strings.HasSuffix(value, "None.class") && !strings.HasSuffix(value, "None::class")
}
