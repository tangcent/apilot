package express

import (
	"strings"

	model "github.com/tangcent/apilot/api-model"
)

var tsPrimitiveTypes = map[string]string{
	"string":          model.JsonTypeString,
	"number":          model.JsonTypeInt,
	"boolean":         model.JsonTypeBoolean,
	"null":            model.JsonTypeNull,
	"undefined":       model.JsonTypeNull,
	"void":            model.JsonTypeNull,
	"any":             model.JsonTypeString,
	"unknown":         model.JsonTypeString,
	"never":           model.JsonTypeNull,
	"object":          model.JsonTypeString,
	"Date":            model.JsonTypeString,
	"BigInt":          model.JsonTypeLong,
	"symbol":          model.JsonTypeString,
	"string[]":        "string[]",
	"number[]":        "int[]",
	"boolean[]":       "boolean[]",
	"Record":          "map",
	"Partial":         "object",
	"Required":        "object",
	"Readonly":        "object",
	"Pick":            "object",
	"Omit":            "object",
	"Exclude":         "object",
	"Extract":         "object",
	"NonNullable":     "object",
	"ReturnType":      "object",
	"InstanceType":    "object",
	"ParamsDictionary": "",
	"ParsedQs":       "",
	"qs.ParsedQs":    "",
}

var tsCollectionTypes = map[string]bool{
	"Array":      true,
	"ReadonlyArray": true,
	"Set":        true,
	"ReadonlySet": true,
	"Map":        true,
	"ReadonlyMap": true,
	"Iterable":   true,
	"AsyncIterable": true,
}

type TSTypeResolver struct {
	registry   *TSTypeRegistry
	resolving  map[string]bool
}

func NewTSTypeResolver(registry *TSTypeRegistry) *TSTypeResolver {
	return &TSTypeResolver{
		registry:  registry,
		resolving: make(map[string]bool),
	}
}

func (r *TSTypeResolver) Resolve(rawType string, typeBindings map[string]string) *model.ObjectModel {
	if rawType == "" {
		return model.NullModel()
	}

	rawType = strings.TrimSpace(rawType)

	if typeBindings != nil {
		if bound, ok := typeBindings[rawType]; ok {
			return r.Resolve(bound, nil)
		}
	}

	if jsonType, ok := tsPrimitiveTypes[rawType]; ok {
		if jsonType == "" {
			return model.NullModel()
		}
		if strings.HasSuffix(jsonType, "[]") {
			itemType := strings.TrimSuffix(jsonType, "[]")
			return model.ArrayModel(model.SingleModel(itemType))
		}
		if jsonType == "map" {
			return model.MapModel(model.SingleModel(model.JsonTypeString), model.SingleModel(model.JsonTypeString))
		}
		return model.SingleModel(jsonType)
	}

	if strings.HasSuffix(rawType, "[]") {
		itemType := strings.TrimSuffix(rawType, "[]")
		itemModel := r.Resolve(itemType, typeBindings)
		return model.ArrayModel(itemModel)
	}

	if strings.Contains(rawType, " | ") {
		return r.resolveUnionType(rawType, typeBindings)
	}

	if strings.Contains(rawType, " & ") {
		return r.resolveIntersectionType(rawType, typeBindings)
	}

	baseName, typeArgs := parseTSTypeArgs(rawType)

	if baseName != rawType && len(typeArgs) > 0 {
		return r.resolveGenericType(baseName, typeArgs, typeBindings)
	}

	if enum, found := r.registry.Enums[baseName]; found {
		return r.resolveEnum(enum)
	}

	if iface, found := r.registry.Interfaces[baseName]; found {
		return r.resolveInterface(iface, typeArgs, typeBindings)
	}

	if alias, found := r.registry.TypeAliases[baseName]; found {
		return r.resolveTypeAlias(alias, typeArgs, typeBindings)
	}

	return model.SingleModel(rawType)
}

func (r *TSTypeResolver) resolveUnionType(rawType string, typeBindings map[string]string) *model.ObjectModel {
	parts := strings.Split(rawType, " | ")
	var nonNullParts []string

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "null" || part == "undefined" {
			continue
		}
		nonNullParts = append(nonNullParts, part)
	}

	if len(nonNullParts) == 1 {
		result := r.Resolve(nonNullParts[0], typeBindings)
		return result
	}

	return model.SingleModel(rawType)
}

func (r *TSTypeResolver) resolveIntersectionType(rawType string, typeBindings map[string]string) *model.ObjectModel {
	parts := strings.Split(rawType, " & ")
	mergedFields := make(map[string]*model.FieldModel)

	for _, part := range parts {
		part = strings.TrimSpace(part)
		resolved := r.Resolve(part, typeBindings)
		if resolved != nil && resolved.IsObject() {
			for name, field := range resolved.Fields {
				if _, exists := mergedFields[name]; !exists {
					mergedFields[name] = field
				}
			}
		}
	}

	if len(mergedFields) > 0 {
		return &model.ObjectModel{
			Kind:     model.KindObject,
			TypeName: "object",
			Fields:   mergedFields,
		}
	}

	return model.SingleModel("object")
}

func (r *TSTypeResolver) resolveGenericType(baseName string, typeArgs []string, typeBindings map[string]string) *model.ObjectModel {
	if tsCollectionTypes[baseName] {
		if baseName == "Map" || baseName == "ReadonlyMap" {
			keyModel := model.SingleModel(model.JsonTypeString)
			valueModel := model.NullModel()
			if len(typeArgs) >= 2 {
				valueModel = r.Resolve(typeArgs[1], typeBindings)
			} else if len(typeArgs) == 1 {
				valueModel = r.Resolve(typeArgs[0], typeBindings)
			}
			return model.MapModel(keyModel, valueModel)
		}
		if len(typeArgs) > 0 {
			itemModel := r.Resolve(typeArgs[0], typeBindings)
			return model.ArrayModel(itemModel)
		}
		return model.ArrayModel(model.NullModel())
	}

	if baseName == "Record" {
		valueModel := model.NullModel()
		if len(typeArgs) >= 2 {
			valueModel = r.Resolve(typeArgs[1], typeBindings)
		} else if len(typeArgs) == 1 {
			valueModel = r.Resolve(typeArgs[0], typeBindings)
		}
		return model.MapModel(model.SingleModel(model.JsonTypeString), valueModel)
	}

	if baseName == "Promise" {
		if len(typeArgs) > 0 {
			return r.Resolve(typeArgs[0], typeBindings)
		}
		return model.NullModel()
	}

	if baseName == "Array" || baseName == "ReadonlyArray" {
		if len(typeArgs) > 0 {
			itemModel := r.Resolve(typeArgs[0], typeBindings)
			return model.ArrayModel(itemModel)
		}
		return model.ArrayModel(model.NullModel())
	}

	if baseName == "Partial" || baseName == "Required" || baseName == "Readonly" {
		if len(typeArgs) > 0 {
			return r.Resolve(typeArgs[0], typeBindings)
		}
		return model.EmptyObject()
	}

	if baseName == "Pick" || baseName == "Omit" {
		if len(typeArgs) > 0 {
			return r.Resolve(typeArgs[0], typeBindings)
		}
		return model.EmptyObject()
	}

	if baseName == "Exclude" || baseName == "Extract" {
		if len(typeArgs) > 0 {
			return r.Resolve(typeArgs[0], typeBindings)
		}
		return model.NullModel()
	}

	if baseName == "NonNullable" {
		if len(typeArgs) > 0 {
			return r.Resolve(typeArgs[0], typeBindings)
		}
		return model.NullModel()
	}

	if baseName == "ReturnType" || baseName == "InstanceType" {
		return model.EmptyObject()
	}

	if enum, found := r.registry.Enums[baseName]; found {
		return r.resolveEnum(enum)
	}

	if iface, found := r.registry.Interfaces[baseName]; found {
		return r.resolveInterface(iface, typeArgs, typeBindings)
	}

	if alias, found := r.registry.TypeAliases[baseName]; found {
		return r.resolveTypeAlias(alias, typeArgs, typeBindings)
	}

	return model.SingleModel(baseName)
}

func (r *TSTypeResolver) resolveEnum(enum *TSEnum) *model.ObjectModel {
	options := make([]model.FieldOption, 0, len(enum.Members))
	for _, m := range enum.Members {
		opt := model.FieldOption{Value: m.Name}
		if m.Value != "" {
			opt.Value = m.Value
		}
		options = append(options, opt)
	}

	return &model.ObjectModel{
		Kind:     model.KindSingle,
		TypeName: enum.Name,
	}
}

func (r *TSTypeResolver) resolveInterface(iface *TSInterface, typeArgs []string, typeBindings map[string]string) *model.ObjectModel {
	if r.resolving[iface.Name] {
		return model.RefModel(iface.Name)
	}

	r.resolving[iface.Name] = true
	defer func() { delete(r.resolving, iface.Name) }()

	localBindings := make(map[string]string)
	for i, tp := range iface.TypeParameters {
		if i < len(typeArgs) {
			localBindings[tp] = typeArgs[i]
		}
	}
	if typeBindings != nil {
		for k, v := range typeBindings {
			if _, exists := localBindings[k]; !exists {
				localBindings[k] = v
			}
		}
	}

	fields := make(map[string]*model.FieldModel, len(iface.Fields))
	for _, f := range iface.Fields {
		fieldModel := r.Resolve(f.Type, localBindings)
		fm := &model.FieldModel{
			Model:    fieldModel,
			Required: f.Required,
			Comment:  f.Comment,
		}
		fields[f.Name] = fm
	}

	return &model.ObjectModel{
		Kind:     model.KindObject,
		TypeName: iface.Name,
		Fields:   fields,
	}
}

func (r *TSTypeResolver) resolveTypeAlias(alias *TSTypeAlias, typeArgs []string, typeBindings map[string]string) *model.ObjectModel {
	if r.resolving[alias.Name] {
		return model.RefModel(alias.Name)
	}

	r.resolving[alias.Name] = true
	defer func() { delete(r.resolving, alias.Name) }()

	localBindings := make(map[string]string)
	for i, tp := range alias.TypeParameters {
		if i < len(typeArgs) {
			localBindings[tp] = typeArgs[i]
		}
	}
	if typeBindings != nil {
		for k, v := range typeBindings {
			if _, exists := localBindings[k]; !exists {
				localBindings[k] = v
			}
		}
	}

	return r.Resolve(alias.TypeDef, localBindings)
}

func parseTSTypeArgs(rawType string) (string, []string) {
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

func ResolveHandlerTypes(handlerInfo *ExpressHandlerInfo, registry *TSTypeRegistry) (reqBody *model.ObjectModel, resBody *model.ObjectModel) {
	resolver := NewTSTypeResolver(registry)

	if handlerInfo.ReqBodyType != "" {
		reqBody = resolver.Resolve(handlerInfo.ReqBodyType, nil)
	}

	if handlerInfo.ResBodyType != "" {
		resBody = resolver.Resolve(handlerInfo.ResBodyType, nil)
	}

	return reqBody, resBody
}
