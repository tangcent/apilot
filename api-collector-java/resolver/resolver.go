package resolver

import (
	"strings"

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

type TypeResolver struct {
	classRegistry map[string]parser.Class
	allTypeParams map[string]bool
	resolving     map[string]bool
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
