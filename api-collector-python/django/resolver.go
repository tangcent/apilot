package django

import (
	"strings"

	model "github.com/tangcent/apilot/api-model"
)

type DRFTypeResolver struct {
	serializerRegistry map[string]SerializerModel
	resolving          map[string]bool
}

func NewDRFTypeResolver(serializers map[string]SerializerModel) *DRFTypeResolver {
	return &DRFTypeResolver{
		serializerRegistry: serializers,
		resolving:          make(map[string]bool),
	}
}

func (r *DRFTypeResolver) ResolveSerializer(serializerName string) *model.ObjectModel {
	md, found := r.serializerRegistry[serializerName]
	if !found {
		return model.SingleModel(serializerName)
	}

	if r.resolving[serializerName] {
		return model.RefModel(serializerName)
	}

	r.resolving[serializerName] = true
	defer func() { delete(r.resolving, serializerName) }()

	fields := make(map[string]*model.FieldModel, len(md.Fields))
	for _, f := range md.Fields {
		fieldModel := r.resolveSerializerField(f)
		fields[f.Name] = fieldModel
	}

	for _, embedded := range md.EmbeddedTypes {
		embeddedModel := r.ResolveSerializer(embedded)
		if embeddedModel != nil && embeddedModel.IsObject() {
			for k, v := range embeddedModel.Fields {
				if _, exists := fields[k]; !exists {
					fields[k] = v
				}
			}
		}
	}

	return &model.ObjectModel{
		Kind:     model.KindObject,
		TypeName: serializerName,
		Fields:   fields,
	}
}

func (r *DRFTypeResolver) resolveSerializerField(f SerializerField) *model.FieldModel {
	var fieldModel *model.ObjectModel

	if f.Many {
		itemModel := r.resolveDRFFieldType(f.DRFType)
		fieldModel = model.ArrayModel(itemModel)
	} else {
		fieldModel = r.resolveDRFFieldType(f.DRFType)
	}

	return &model.FieldModel{
		Model:    fieldModel,
		Required: f.Required && !f.ReadOnly,
	}
}

func (r *DRFTypeResolver) resolveDRFFieldType(drfType string) *model.ObjectModel {
	if jsonType, ok := drfFieldTypeMap[drfType]; ok {
		if jsonType == "array" {
			return model.ArrayModel(model.NullModel())
		}
		if jsonType == "map" {
			return model.MapModel(model.SingleModel(model.JsonTypeString), model.NullModel())
		}
		return model.SingleModel(jsonType)
	}

	nestedModel := r.ResolveSerializer(drfType)
	if nestedModel != nil && (nestedModel.IsObject() || nestedModel.IsRef()) {
		return nestedModel
	}

	return model.SingleModel(drfType)
}

func (r *DRFTypeResolver) ResolveActionSerializer(viewClassName string, action string, serializerClassName string) (requestSerializer string, responseSerializer string) {
	if serializerClassName == "" {
		return "", ""
	}

	switch action {
	case "list":
		return "", serializerClassName
	case "create":
		return serializerClassName, serializerClassName
	case "retrieve":
		return "", serializerClassName
	case "update":
		return serializerClassName, serializerClassName
	case "partial_update":
		return serializerClassName, serializerClassName
	case "destroy":
		return "", ""
	default:
		return "", serializerClassName
	}
}

func (r *DRFTypeResolver) ResolveHTTPMethodSerializer(viewClassName string, httpMethod string, serializerClassName string) (requestSerializer string, responseSerializer string) {
	if serializerClassName == "" {
		return "", ""
	}

	switch httpMethod {
	case "GET":
		return "", serializerClassName
	case "POST":
		return serializerClassName, serializerClassName
	case "PUT":
		return serializerClassName, serializerClassName
	case "PATCH":
		return serializerClassName, serializerClassName
	case "DELETE":
		return "", ""
	default:
		return "", serializerClassName
	}
}

func (r *DRFTypeResolver) BuildRequestBody(serializerName string) *model.ObjectModel {
	if serializerName == "" {
		return nil
	}

	resolved := r.ResolveSerializer(serializerName)
	if resolved == nil || resolved.IsNull() {
		return nil
	}

	writeFields := make(map[string]*model.FieldModel)
	for name, field := range resolved.Fields {
		if !field.Required && field.DefaultValue == "" {
			writeFields[name] = field
			continue
		}
		writeFields[name] = field
	}

	if len(writeFields) == 0 {
		return resolved
	}

	return &model.ObjectModel{
		Kind:     model.KindObject,
		TypeName: serializerName,
		Fields:   writeFields,
	}
}

func (r *DRFTypeResolver) BuildResponseBody(serializerName string) *model.ObjectModel {
	if serializerName == "" {
		return nil
	}

	return r.ResolveSerializer(serializerName)
}

func isDRFSerializerType(typeName string) bool {
	_, ok := drfFieldTypeMap[typeName]
	if ok {
		return true
	}
	if strings.HasSuffix(typeName, "Serializer") {
		return true
	}
	if strings.HasSuffix(typeName, "Field") {
		return true
	}
	return false
}
