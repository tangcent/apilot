package yapi

import (
	"encoding/json"
	"fmt"
	"strings"

	model "github.com/tangcent/apilot/api-model"
)

const jsonSchemaDraft4 = "http://json-schema.org/draft-04/schema#"

// maxSchemaDepth bounds nesting; the model already ends a self-reference in a
// ref node, so this only guards against a malformed model.
const maxSchemaDepth = 64

// schemaBuilder turns an object model into the JSON Schema YApi stores for a
// request or response body: draft-04, an object's required properties listed
// on the object, and the YApi extensions "mock" and "enumDesc".
type schemaBuilder struct {
	rules *Rules
}

// document builds the schema of a whole body.
func (b *schemaBuilder) document(m *model.ObjectModel) *object {
	root := newObject()
	root.set("$schema", jsonSchemaDraft4)
	if m == nil {
		root.set("type", "object")
		return root
	}
	s := b.node(m, "", 0)
	b.applyMockRule(s, "")
	for _, key := range s.keys {
		root.set(key, s.vals[key])
	}
	return root
}

// documentString is document serialised: YApi takes the schema as a string.
func (b *schemaBuilder) documentString(m *model.ObjectModel) string {
	data, err := json.Marshal(b.document(m))
	if err != nil {
		return `{"type":"object"}`
	}
	return string(data)
}

// node builds the schema of one value. path is its JSON path, written the
// way EasyYapi's mock rules address it: "data.list[].name".
func (b *schemaBuilder) node(m *model.ObjectModel, path string, depth int) *object {
	s := newObject()
	if m == nil || depth > maxSchemaDepth {
		s.set("type", "object")
		return s
	}
	switch m.Kind {
	case model.KindObject:
		s.set("type", "object")
		props := newObject()
		var required []string
		for _, name := range m.OrderedFieldNames() {
			field := m.Fields[name]
			if field == nil {
				continue
			}
			props.set(name, b.field(field, joinPath(path, name), depth+1))
			if field.Required {
				required = append(required, name)
			}
		}
		s.set("properties", props)
		if len(required) > 0 {
			s.set("required", required)
		}
	case model.KindArray:
		s.set("type", "array")
		items := b.node(m.Items, path+"[]", depth+1)
		b.applyMockRule(items, path+"[]")
		s.set("items", items)
	case model.KindMap:
		s.set("type", "object")
		if m.ValueModel != nil {
			s.set("additionalProperties", b.node(m.ValueModel, path, depth+1))
		}
	case model.KindSingle:
		s.set("type", b.jsonType(m.TypeName))
	default:
		// A reference back to an enclosing type.
		s.set("type", "object")
	}
	return s
}

// field builds the schema of an object member: its value, then its
// documentation, then a mock.
func (b *schemaBuilder) field(f *model.FieldModel, path string, depth int) *object {
	s := b.node(f.Model, path, depth)
	if f.Comment != "" {
		s.set("description", f.Comment)
	}
	if f.DefaultValue != "" {
		s.set("default", f.DefaultValue)
	}
	if f.Demo != "" {
		s.set("mock", mockValue(f.Demo))
	}
	if len(f.Options) > 0 {
		// An enum applies to the values, so a list of them carries it on its
		// innermost items.
		target := s
		for target.get("type") == "array" {
			items, ok := target.get("items").(*object)
			if !ok {
				break
			}
			target = items
		}
		values := make([]any, 0, len(f.Options))
		var descs []string
		for _, option := range f.Options {
			values = append(values, option.Value)
			if option.Desc != "" {
				descs = append(descs, fmt.Sprintf("%v: %s", option.Value, option.Desc))
			}
		}
		target.set("enum", values)
		if len(descs) > 0 {
			target.set("enumDesc", strings.Join(descs, "\n"))
		}
		// The enum's own pick mock wins over a generic mock rule that node()
		// may already have applied to an array's items. A mock from the field's
		// own documentation lives on the scalar field and is kept instead.
		if target != s || !target.has("mock") {
			pick, _ := json.Marshal(values)
			target.set("mock", mockValue("@pick("+string(pick)+")"))
		}
	}
	b.applyMockRule(s, path)
	return s
}

// applyMockRule attaches the mock of the first matching rule, unless the node
// already has one from its own documentation.
func (b *schemaBuilder) applyMockRule(s *object, path string) {
	if s.has("mock") {
		return
	}
	jsonType, _ := s.get("type").(string)
	if value, ok := b.rules.mockFor(path, jsonType); ok {
		s.set("mock", mockValue(value))
	}
}

func mockValue(v string) map[string]string { return map[string]string{"mock": v} }

// jsonType maps a model type, after the conversion rules, to a JSON Schema
// type. A type the collector could not resolve is an object.
func (b *schemaBuilder) jsonType(typeName string) string {
	switch b.rules.convertType(typeName) {
	case model.JsonTypeString:
		return "string"
	case model.JsonTypeInt, model.JsonTypeLong:
		return "integer"
	case model.JsonTypeFloat, model.JsonTypeDouble:
		return "number"
	case model.JsonTypeBoolean:
		return "boolean"
	case model.JsonTypeNull:
		return "null"
	}
	return "object"
}

func joinPath(path, name string) string {
	if path == "" {
		return name
	}
	return path + "." + name
}
