package model

import (
	"testing"
)

func TestSingleModel(t *testing.T) {
	tests := []struct {
		typeName     string
		expectedKind ObjectModelKind
	}{
		{JsonTypeString, KindSingle},
		{JsonTypeInt, KindSingle},
		{JsonTypeLong, KindSingle},
		{JsonTypeFloat, KindSingle},
		{JsonTypeDouble, KindSingle},
		{JsonTypeBoolean, KindSingle},
		{JsonTypeNull, KindSingle},
		{"CustomType", KindSingle},
	}

	for _, tt := range tests {
		t.Run(tt.typeName, func(t *testing.T) {
			m := SingleModel(tt.typeName)
			if m.Kind != tt.expectedKind {
				t.Errorf("Expected kind %s, got %s", tt.expectedKind, m.Kind)
			}
			if m.TypeName != tt.typeName {
				t.Errorf("Expected typeName %s, got %s", tt.typeName, m.TypeName)
			}
			if m.Fields != nil {
				t.Errorf("Expected nil Fields, got %v", m.Fields)
			}
			if m.Items != nil {
				t.Errorf("Expected nil Items, got %v", m.Items)
			}
		})
	}
}

func TestObjectModelFrom(t *testing.T) {
	fields := map[string]*FieldModel{
		"id":   {Model: SingleModel(JsonTypeLong)},
		"name": {Model: SingleModel(JsonTypeString)},
	}

	m := ObjectModelFrom(fields)

	if m.Kind != KindObject {
		t.Errorf("Expected kind %s, got %s", KindObject, m.Kind)
	}
	if m.TypeName != "object" {
		t.Errorf("Expected typeName 'object', got %s", m.TypeName)
	}
	if len(m.Fields) != 2 {
		t.Errorf("Expected 2 fields, got %d", len(m.Fields))
	}
	if m.Fields["id"] == nil || m.Fields["name"] == nil {
		t.Error("Expected id and name fields to be present")
	}
}

func TestObjectModelFrom_Empty(t *testing.T) {
	m := ObjectModelFrom(nil)

	if m.Kind != KindObject {
		t.Errorf("Expected kind %s, got %s", KindObject, m.Kind)
	}
	if m.Fields != nil {
		t.Errorf("Expected nil Fields for nil input, got %v", m.Fields)
	}
}

func TestArrayModel(t *testing.T) {
	item := SingleModel(JsonTypeString)
	m := ArrayModel(item)

	if m.Kind != KindArray {
		t.Errorf("Expected kind %s, got %s", KindArray, m.Kind)
	}
	if m.TypeName != "array" {
		t.Errorf("Expected typeName 'array', got %s", m.TypeName)
	}
	if m.Items != item {
		t.Error("Expected Items to be the provided item model")
	}
	if m.Fields != nil {
		t.Errorf("Expected nil Fields, got %v", m.Fields)
	}
}

func TestArrayModel_NilItem(t *testing.T) {
	m := ArrayModel(nil)

	if m.Kind != KindArray {
		t.Errorf("Expected kind %s, got %s", KindArray, m.Kind)
	}
	if m.Items != nil {
		t.Errorf("Expected nil Items, got %v", m.Items)
	}
}

func TestMapModel(t *testing.T) {
	key := SingleModel(JsonTypeString)
	value := SingleModel(JsonTypeInt)
	m := MapModel(key, value)

	if m.Kind != KindMap {
		t.Errorf("Expected kind %s, got %s", KindMap, m.Kind)
	}
	if m.TypeName != "map" {
		t.Errorf("Expected typeName 'map', got %s", m.TypeName)
	}
	if m.KeyModel != key {
		t.Error("Expected KeyModel to be the provided key model")
	}
	if m.ValueModel != value {
		t.Error("Expected ValueModel to be the provided value model")
	}
}

func TestRefModel(t *testing.T) {
	m := RefModel("User")

	if m.Kind != KindRef {
		t.Errorf("Expected kind %s, got %s", KindRef, m.Kind)
	}
	if m.TypeName != "User" {
		t.Errorf("Expected typeName 'User', got %s", m.TypeName)
	}
}

func TestNullModel(t *testing.T) {
	m := NullModel()

	if m.Kind != KindSingle {
		t.Errorf("Expected kind %s, got %s", KindSingle, m.Kind)
	}
	if m.TypeName != JsonTypeNull {
		t.Errorf("Expected typeName %s, got %s", JsonTypeNull, m.TypeName)
	}
}

func TestEmptyObject(t *testing.T) {
	m := EmptyObject()

	if m.Kind != KindObject {
		t.Errorf("Expected kind %s, got %s", KindObject, m.Kind)
	}
	if m.TypeName != "object" {
		t.Errorf("Expected typeName 'object', got %s", m.TypeName)
	}
	if m.Fields == nil {
		t.Error("Expected non-nil Fields for EmptyObject")
	}
	if len(m.Fields) != 0 {
		t.Errorf("Expected 0 fields, got %d", len(m.Fields))
	}
}

func TestObjectModel_IsMethods(t *testing.T) {
	tests := []struct {
		name     string
		model    *ObjectModel
		isSingle bool
		isObject bool
		isArray  bool
		isMap    bool
		isRef    bool
	}{
		{"single", SingleModel("string"), true, false, false, false, false},
		{"object", EmptyObject(), false, true, false, false, false},
		{"array", ArrayModel(SingleModel("int")), false, false, true, false, false},
		{"map", MapModel(SingleModel("string"), SingleModel("int")), false, false, false, true, false},
		{"ref", RefModel("User"), false, false, false, false, true},
		{"nil", nil, false, false, false, false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.model.IsSingle() != tt.isSingle {
				t.Errorf("IsSingle() = %v, want %v", tt.model.IsSingle(), tt.isSingle)
			}
			if tt.model.IsObject() != tt.isObject {
				t.Errorf("IsObject() = %v, want %v", tt.model.IsObject(), tt.isObject)
			}
			if tt.model.IsArray() != tt.isArray {
				t.Errorf("IsArray() = %v, want %v", tt.model.IsArray(), tt.isArray)
			}
			if tt.model.IsMap() != tt.isMap {
				t.Errorf("IsMap() = %v, want %v", tt.model.IsMap(), tt.isMap)
			}
			if tt.model.IsRef() != tt.isRef {
				t.Errorf("IsRef() = %v, want %v", tt.model.IsRef(), tt.isRef)
			}
		})
	}
}

func TestFieldOpts(t *testing.T) {
	t.Run("WithComment", func(t *testing.T) {
		fm := &FieldModel{}
		WithComment("test comment")(fm)
		if fm.Comment != "test comment" {
			t.Errorf("Expected comment 'test comment', got %s", fm.Comment)
		}
	})

	t.Run("WithRequired", func(t *testing.T) {
		fm := &FieldModel{}
		WithRequired(true)(fm)
		if fm.Required != true {
			t.Errorf("Expected Required true, got %v", fm.Required)
		}
		WithRequired(false)(fm)
		if fm.Required != false {
			t.Errorf("Expected Required false, got %v", fm.Required)
		}
	})

	t.Run("WithDefault", func(t *testing.T) {
		fm := &FieldModel{}
		WithDefault("default_value")(fm)
		if fm.DefaultValue != "default_value" {
			t.Errorf("Expected DefaultValue 'default_value', got %s", fm.DefaultValue)
		}
	})

	t.Run("WithDemo", func(t *testing.T) {
		fm := &FieldModel{}
		WithDemo("demo_value")(fm)
		if fm.Demo != "demo_value" {
			t.Errorf("Expected Demo 'demo_value', got %s", fm.Demo)
		}
	})

	t.Run("WithGeneric", func(t *testing.T) {
		fm := &FieldModel{}
		WithGeneric(true)(fm)
		if fm.Generic != true {
			t.Errorf("Expected Generic true, got %v", fm.Generic)
		}
		WithGeneric(false)(fm)
		if fm.Generic != false {
			t.Errorf("Expected Generic false, got %v", fm.Generic)
		}
	})
}

func TestObjectModelBuilder(t *testing.T) {
	t.Run("basic field", func(t *testing.T) {
		b := NewObjectModelBuilder()
		b.Field("custom", SingleModel("CustomType"))
		m := b.Build()

		if m.Kind != KindObject {
			t.Errorf("Expected kind %s, got %s", KindObject, m.Kind)
		}
		if len(m.Fields) != 1 {
			t.Errorf("Expected 1 field, got %d", len(m.Fields))
		}
		if m.Fields["custom"] == nil {
			t.Fatal("Expected 'custom' field")
		}
		if m.Fields["custom"].Model.TypeName != "CustomType" {
			t.Errorf("Expected typeName 'CustomType', got %s", m.Fields["custom"].Model.TypeName)
		}
	})

	t.Run("StringField", func(t *testing.T) {
		b := NewObjectModelBuilder()
		b.StringField("name")
		m := b.Build()

		if m.Fields["name"] == nil {
			t.Fatal("Expected 'name' field")
		}
		if m.Fields["name"].Model.Kind != KindSingle {
			t.Errorf("Expected KindSingle, got %s", m.Fields["name"].Model.Kind)
		}
		if m.Fields["name"].Model.TypeName != JsonTypeString {
			t.Errorf("Expected typeName %s, got %s", JsonTypeString, m.Fields["name"].Model.TypeName)
		}
	})

	t.Run("IntField", func(t *testing.T) {
		b := NewObjectModelBuilder()
		b.IntField("age")
		m := b.Build()

		if m.Fields["age"].Model.TypeName != JsonTypeInt {
			t.Errorf("Expected typeName %s, got %s", JsonTypeInt, m.Fields["age"].Model.TypeName)
		}
	})

	t.Run("LongField", func(t *testing.T) {
		b := NewObjectModelBuilder()
		b.LongField("id")
		m := b.Build()

		if m.Fields["id"].Model.TypeName != JsonTypeLong {
			t.Errorf("Expected typeName %s, got %s", JsonTypeLong, m.Fields["id"].Model.TypeName)
		}
	})

	t.Run("FloatField", func(t *testing.T) {
		b := NewObjectModelBuilder()
		b.FloatField("price")
		m := b.Build()

		if m.Fields["price"].Model.TypeName != JsonTypeFloat {
			t.Errorf("Expected typeName %s, got %s", JsonTypeFloat, m.Fields["price"].Model.TypeName)
		}
	})

	t.Run("DoubleField", func(t *testing.T) {
		b := NewObjectModelBuilder()
		b.DoubleField("amount")
		m := b.Build()

		if m.Fields["amount"].Model.TypeName != JsonTypeDouble {
			t.Errorf("Expected typeName %s, got %s", JsonTypeDouble, m.Fields["amount"].Model.TypeName)
		}
	})

	t.Run("BoolField", func(t *testing.T) {
		b := NewObjectModelBuilder()
		b.BoolField("active")
		m := b.Build()

		if m.Fields["active"].Model.TypeName != JsonTypeBoolean {
			t.Errorf("Expected typeName %s, got %s", JsonTypeBoolean, m.Fields["active"].Model.TypeName)
		}
	})

	t.Run("ArrayField", func(t *testing.T) {
		b := NewObjectModelBuilder()
		b.ArrayField("tags", SingleModel(JsonTypeString))
		m := b.Build()

		if m.Fields["tags"].Model.Kind != KindArray {
			t.Errorf("Expected KindArray, got %s", m.Fields["tags"].Model.Kind)
		}
		if m.Fields["tags"].Model.Items == nil {
			t.Fatal("Expected non-nil Items")
		}
		if m.Fields["tags"].Model.Items.TypeName != JsonTypeString {
			t.Errorf("Expected Items typeName %s, got %s", JsonTypeString, m.Fields["tags"].Model.Items.TypeName)
		}
	})

	t.Run("ObjectField", func(t *testing.T) {
		nested := NewObjectModelBuilder().
			StringField("street").
			StringField("city").
			Build()

		b := NewObjectModelBuilder()
		b.ObjectField("address", nested)
		m := b.Build()

		if m.Fields["address"].Model.Kind != KindObject {
			t.Errorf("Expected KindObject, got %s", m.Fields["address"].Model.Kind)
		}
		if len(m.Fields["address"].Model.Fields) != 2 {
			t.Errorf("Expected 2 nested fields, got %d", len(m.Fields["address"].Model.Fields))
		}
	})

	t.Run("MapField", func(t *testing.T) {
		b := NewObjectModelBuilder()
		b.MapField("metadata", SingleModel(JsonTypeString), SingleModel(JsonTypeString))
		m := b.Build()

		if m.Fields["metadata"].Model.Kind != KindMap {
			t.Errorf("Expected KindMap, got %s", m.Fields["metadata"].Model.Kind)
		}
		if m.Fields["metadata"].Model.KeyModel.TypeName != JsonTypeString {
			t.Errorf("Expected KeyModel typeName %s, got %s", JsonTypeString, m.Fields["metadata"].Model.KeyModel.TypeName)
		}
		if m.Fields["metadata"].Model.ValueModel.TypeName != JsonTypeString {
			t.Errorf("Expected ValueModel typeName %s, got %s", JsonTypeString, m.Fields["metadata"].Model.ValueModel.TypeName)
		}
	})

	t.Run("with options", func(t *testing.T) {
		b := NewObjectModelBuilder()
		b.StringField("name", WithComment("user name"), WithRequired(true), WithDemo("John"))
		m := b.Build()

		f := m.Fields["name"]
		if f.Comment != "user name" {
			t.Errorf("Expected comment 'user name', got %s", f.Comment)
		}
		if f.Required != true {
			t.Errorf("Expected Required true, got %v", f.Required)
		}
		if f.Demo != "John" {
			t.Errorf("Expected Demo 'John', got %s", f.Demo)
		}
	})

	t.Run("multiple fields", func(t *testing.T) {
		m := NewObjectModelBuilder().
			LongField("id").
			StringField("name", WithComment("user name")).
			BoolField("active", WithDefault("true")).
			ArrayField("tags", SingleModel(JsonTypeString)).
			Build()

		if len(m.Fields) != 4 {
			t.Errorf("Expected 4 fields, got %d", len(m.Fields))
		}
	})

	t.Run("build returns copy", func(t *testing.T) {
		b := NewObjectModelBuilder()
		b.StringField("name")
		m1 := b.Build()
		b.StringField("email")
		m2 := b.Build()

		if len(m1.Fields) != 1 {
			t.Errorf("First build should have 1 field, got %d", len(m1.Fields))
		}
		if len(m2.Fields) != 2 {
			t.Errorf("Second build should have 2 fields, got %d", len(m2.Fields))
		}
	})
}

func TestFieldModel(t *testing.T) {
	t.Run("all fields", func(t *testing.T) {
		fm := &FieldModel{
			Model:        SingleModel(JsonTypeString),
			Comment:      "test comment",
			Required:     true,
			DefaultValue: "default",
			Options: []FieldOption{
				{Value: "opt1", Desc: "option 1"},
				{Value: "opt2", Desc: "option 2"},
			},
			Demo:     "demo",
			Advanced: map[string]any{"key": "value"},
			Generic:  true,
		}

		if fm.Model == nil {
			t.Error("Expected non-nil Model")
		}
		if fm.Comment != "test comment" {
			t.Errorf("Expected comment 'test comment', got %s", fm.Comment)
		}
		if !fm.Required {
			t.Error("Expected Required true")
		}
		if fm.DefaultValue != "default" {
			t.Errorf("Expected DefaultValue 'default', got %s", fm.DefaultValue)
		}
		if len(fm.Options) != 2 {
			t.Errorf("Expected 2 options, got %d", len(fm.Options))
		}
		if fm.Demo != "demo" {
			t.Errorf("Expected Demo 'demo', got %s", fm.Demo)
		}
		if fm.Advanced["key"] != "value" {
			t.Error("Expected Advanced key=value")
		}
		if !fm.Generic {
			t.Error("Expected Generic true")
		}
	})
}

func TestApiBody(t *testing.T) {
	t.Run("with body", func(t *testing.T) {
		body := &ApiBody{
			MediaType: "application/json",
			Body:      SingleModel("User"),
			Example:   map[string]string{"id": "1", "name": "test"},
		}

		if body.MediaType != "application/json" {
			t.Errorf("Expected MediaType 'application/json', got %s", body.MediaType)
		}
		if body.Body == nil {
			t.Error("Expected non-nil Body")
		}
		if body.Body.TypeName != "User" {
			t.Errorf("Expected Body typeName 'User', got %s", body.Body.TypeName)
		}
	})

	t.Run("empty", func(t *testing.T) {
		body := &ApiBody{}

		if body.MediaType != "" {
			t.Errorf("Expected empty MediaType, got %s", body.MediaType)
		}
		if body.Body != nil {
			t.Error("Expected nil Body")
		}
	})
}

func TestApiEndpoint(t *testing.T) {
	ep := ApiEndpoint{
		Name:        "getUser",
		Folder:      "UserController",
		Description: "Get user by ID",
		Tags:        []string{"user", "read"},
		Path:        "/users/{id}",
		Method:      "GET",
		Protocol:    "http",
		Parameters: []ApiParameter{
			{Name: "id", Type: "text", Required: true, In: "path"},
		},
		Headers: []ApiHeader{
			{Name: "Authorization", Value: "Bearer token", Required: true},
		},
		RequestBody: &ApiBody{MediaType: "application/json"},
		Response:    &ApiBody{MediaType: "application/json", Body: SingleModel("User")},
		Metadata:    map[string]any{"version": "1.0"},
	}

	if ep.Name != "getUser" {
		t.Errorf("Expected Name 'getUser', got %s", ep.Name)
	}
	if len(ep.Tags) != 2 {
		t.Errorf("Expected 2 tags, got %d", len(ep.Tags))
	}
	if len(ep.Parameters) != 1 {
		t.Errorf("Expected 1 parameter, got %d", len(ep.Parameters))
	}
	if len(ep.Headers) != 1 {
		t.Errorf("Expected 1 header, got %d", len(ep.Headers))
	}
	if ep.Metadata["version"] != "1.0" {
		t.Error("Expected Metadata version=1.0")
	}
}

func TestApiParameter(t *testing.T) {
	p := ApiParameter{
		Name:        "id",
		Type:        "text",
		Required:    true,
		In:          "path",
		Default:     "1",
		Description: "User ID",
		Example:     "123",
		Enum:        []string{"1", "2", "3"},
	}

	if p.Name != "id" {
		t.Errorf("Expected Name 'id', got %s", p.Name)
	}
	if p.Type != "text" {
		t.Errorf("Expected Type 'text', got %s", p.Type)
	}
	if !p.Required {
		t.Error("Expected Required true")
	}
	if p.In != "path" {
		t.Errorf("Expected In 'path', got %s", p.In)
	}
	if len(p.Enum) != 3 {
		t.Errorf("Expected 3 enum values, got %d", len(p.Enum))
	}
}

func TestApiHeader(t *testing.T) {
	h := ApiHeader{
		Name:        "Authorization",
		Value:       "Bearer token",
		Description: "Auth header",
		Example:     "Bearer xxx",
		Required:    true,
	}

	if h.Name != "Authorization" {
		t.Errorf("Expected Name 'Authorization', got %s", h.Name)
	}
	if h.Value != "Bearer token" {
		t.Errorf("Expected Value 'Bearer token', got %s", h.Value)
	}
	if !h.Required {
		t.Error("Expected Required true")
	}
}

func TestFieldOption(t *testing.T) {
	opt := FieldOption{
		Value: "active",
		Desc:  "Active status",
	}

	if opt.Value != "active" {
		t.Errorf("Expected Value 'active', got %v", opt.Value)
	}
	if opt.Desc != "Active status" {
		t.Errorf("Expected Desc 'Active status', got %s", opt.Desc)
	}
}
