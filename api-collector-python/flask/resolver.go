package flask

import (
	"strings"

	model "github.com/tangcent/apilot/api-model"

	"github.com/tangcent/apilot/api-collector-python/fastapi"
)

type FlaskTypeResolver struct {
	pythonResolver      *fastapi.PythonTypeResolver
	marshmallowRegistry map[string]MarshmallowModel
	resolving           map[string]bool
}

func NewFlaskTypeResolver(pydanticModels map[string]fastapi.PydanticModel, marshmallowSchemas map[string]MarshmallowModel) *FlaskTypeResolver {
	return &FlaskTypeResolver{
		pythonResolver:      fastapi.NewPythonTypeResolver(pydanticModels),
		marshmallowRegistry: marshmallowSchemas,
		resolving:           make(map[string]bool),
	}
}

func (r *FlaskTypeResolver) Resolve(typeText string) *model.ObjectModel {
	if typeText == "" {
		return model.NullModel()
	}

	typeText = strings.TrimSpace(typeText)

	if md, found := r.marshmallowRegistry[typeText]; found {
		return r.resolveMarshmallowSchema(typeText, md)
	}

	if strings.Contains(typeText, " | ") {
		return r.resolveUnionSyntax(typeText)
	}

	return r.pythonResolver.Resolve(typeText)
}

func (r *FlaskTypeResolver) resolveUnionSyntax(typeText string) *model.ObjectModel {
	parts := strings.Split(typeText, " | ")
	nonNoneParts := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" && p != "None" && p != "NoneType" {
			nonNoneParts = append(nonNoneParts, p)
		}
	}
	if len(nonNoneParts) == 1 {
		return r.Resolve(nonNoneParts[0])
	}
	if len(nonNoneParts) > 1 {
		return model.SingleModel(strings.Join(nonNoneParts, " | "))
	}
	return model.NullModel()
}

func (r *FlaskTypeResolver) resolveMarshmallowSchema(name string, md MarshmallowModel) *model.ObjectModel {
	if r.resolving[name] {
		return model.RefModel(name)
	}

	r.resolving[name] = true
	defer func() { delete(r.resolving, name) }()

	fields := make(map[string]*model.FieldModel, len(md.Fields))
	for _, f := range md.Fields {
		fieldModel := r.resolveMarshmallowField(f)
		fields[f.Name] = fieldModel
	}

	for _, embedded := range md.EmbeddedTypes {
		embeddedModel := r.Resolve(embedded)
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
		TypeName: name,
		Fields:   fields,
	}
}

func (r *FlaskTypeResolver) resolveMarshmallowField(f MarshmallowField) *model.FieldModel {
	var fieldModel *model.ObjectModel

	if f.Nested != "" {
		nestedModel := r.Resolve(f.Nested)
		if f.Many {
			fieldModel = model.ArrayModel(nestedModel)
		} else {
			fieldModel = nestedModel
		}
	} else if f.Many {
		itemModel := r.resolveMarshmallowFieldType(f.FieldType)
		fieldModel = model.ArrayModel(itemModel)
	} else {
		fieldModel = r.resolveMarshmallowFieldType(f.FieldType)
	}

	return &model.FieldModel{
		Model:    fieldModel,
		Required: f.Required,
	}
}

func (r *FlaskTypeResolver) resolveMarshmallowFieldType(fieldType string) *model.ObjectModel {
	if jsonType, ok := marshmallowFieldTypeMap[fieldType]; ok {
		if jsonType == "array" {
			return model.ArrayModel(model.NullModel())
		}
		if jsonType == "map" {
			return model.MapModel(model.SingleModel(model.JsonTypeString), model.NullModel())
		}
		return model.SingleModel(jsonType)
	}

	nestedModel := r.Resolve(fieldType)
	if nestedModel != nil && (nestedModel.IsObject() || nestedModel.IsRef()) {
		return nestedModel
	}

	return model.SingleModel(fieldType)
}
