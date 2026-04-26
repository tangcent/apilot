package flask

import (
	"testing"

	model "github.com/tangcent/apilot/api-model"

	"github.com/tangcent/apilot/api-collector-python/fastapi"
)

func TestFlaskTypeResolver_PydanticModel(t *testing.T) {
	pydanticModels := map[string]fastapi.PydanticModel{
		"UserCreate": {
			Name: "UserCreate",
			Fields: []fastapi.PydanticField{
				{Name: "name", Type: "str", Required: true},
				{Name: "email", Type: "str", Required: true},
				{Name: "age", Type: "int", Required: true},
			},
		},
	}

	resolver := NewFlaskTypeResolver(pydanticModels, nil)

	result := resolver.Resolve("UserCreate")
	if result.Kind != model.KindObject {
		t.Fatalf("Resolve(UserCreate).Kind = %q, want %q", result.Kind, model.KindObject)
	}
	if result.TypeName != "UserCreate" {
		t.Errorf("Resolve(UserCreate).TypeName = %q, want %q", result.TypeName, "UserCreate")
	}
	if len(result.Fields) != 3 {
		t.Fatalf("Resolve(UserCreate).Fields count = %d, want 3", len(result.Fields))
	}

	nameField, ok := result.Fields["name"]
	if !ok {
		t.Fatal("expected 'name' field")
	}
	if nameField.Model.TypeName != model.JsonTypeString {
		t.Errorf("UserCreate.name type = %q, want %q", nameField.Model.TypeName, model.JsonTypeString)
	}

	ageField, ok := result.Fields["age"]
	if !ok {
		t.Fatal("expected 'age' field")
	}
	if ageField.Model.TypeName != model.JsonTypeInt {
		t.Errorf("UserCreate.age type = %q, want %q", ageField.Model.TypeName, model.JsonTypeInt)
	}
}

func TestFlaskTypeResolver_MarshmallowSchema(t *testing.T) {
	marshmallowSchemas := map[string]MarshmallowModel{
		"UserSchema": {
			Name: "UserSchema",
			Fields: []MarshmallowField{
				{Name: "id", FieldType: "Int", Required: false},
				{Name: "name", FieldType: "Str", Required: true},
				{Name: "email", FieldType: "Email", Required: true},
			},
		},
	}

	resolver := NewFlaskTypeResolver(nil, marshmallowSchemas)

	result := resolver.Resolve("UserSchema")
	if result.Kind != model.KindObject {
		t.Fatalf("Resolve(UserSchema).Kind = %q, want %q", result.Kind, model.KindObject)
	}
	if result.TypeName != "UserSchema" {
		t.Errorf("Resolve(UserSchema).TypeName = %q, want %q", result.TypeName, "UserSchema")
	}
	if len(result.Fields) != 3 {
		t.Fatalf("Resolve(UserSchema).Fields count = %d, want 3", len(result.Fields))
	}

	nameField, ok := result.Fields["name"]
	if !ok {
		t.Fatal("expected 'name' field")
	}
	if nameField.Model.TypeName != model.JsonTypeString {
		t.Errorf("UserSchema.name type = %q, want %q", nameField.Model.TypeName, model.JsonTypeString)
	}
	if !nameField.Required {
		t.Error("UserSchema.name should be required")
	}

	idField, ok := result.Fields["id"]
	if !ok {
		t.Fatal("expected 'id' field")
	}
	if idField.Model.TypeName != model.JsonTypeInt {
		t.Errorf("UserSchema.id type = %q, want %q", idField.Model.TypeName, model.JsonTypeInt)
	}
	if idField.Required {
		t.Error("UserSchema.id should not be required")
	}
}

func TestFlaskTypeResolver_MarshmallowNested(t *testing.T) {
	marshmallowSchemas := map[string]MarshmallowModel{
		"AddressSchema": {
			Name: "AddressSchema",
			Fields: []MarshmallowField{
				{Name: "street", FieldType: "Str", Required: true},
				{Name: "city", FieldType: "Str", Required: true},
			},
		},
		"UserSchema": {
			Name: "UserSchema",
			Fields: []MarshmallowField{
				{Name: "name", FieldType: "Str", Required: true},
				{Name: "address", FieldType: "Nested", Required: true, Nested: "AddressSchema"},
			},
		},
	}

	resolver := NewFlaskTypeResolver(nil, marshmallowSchemas)

	result := resolver.Resolve("UserSchema")
	if result.Kind != model.KindObject {
		t.Fatalf("Resolve(UserSchema).Kind = %q, want %q", result.Kind, model.KindObject)
	}

	addrField, ok := result.Fields["address"]
	if !ok {
		t.Fatal("expected 'address' field")
	}
	if addrField.Model.Kind != model.KindObject {
		t.Errorf("UserSchema.address kind = %q, want %q", addrField.Model.Kind, model.KindObject)
	}
	if addrField.Model.TypeName != "AddressSchema" {
		t.Errorf("UserSchema.address typeName = %q, want %q", addrField.Model.TypeName, "AddressSchema")
	}
}

func TestFlaskTypeResolver_MarshmallowListField(t *testing.T) {
	marshmallowSchemas := map[string]MarshmallowModel{
		"ItemSchema": {
			Name: "ItemSchema",
			Fields: []MarshmallowField{
				{Name: "id", FieldType: "Int", Required: false},
				{Name: "title", FieldType: "Str", Required: true},
				{Name: "tags", FieldType: "List", Required: false, Many: true},
			},
		},
	}

	resolver := NewFlaskTypeResolver(nil, marshmallowSchemas)

	result := resolver.Resolve("ItemSchema")
	if result.Kind != model.KindObject {
		t.Fatalf("Resolve(ItemSchema).Kind = %q, want %q", result.Kind, model.KindObject)
	}

	tagsField, ok := result.Fields["tags"]
	if !ok {
		t.Fatal("expected 'tags' field")
	}
	if tagsField.Model.Kind != model.KindArray {
		t.Errorf("ItemSchema.tags kind = %q, want %q", tagsField.Model.Kind, model.KindArray)
	}
}

func TestFlaskTypeResolver_MarshmallowNestedMany(t *testing.T) {
	marshmallowSchemas := map[string]MarshmallowModel{
		"TagSchema": {
			Name: "TagSchema",
			Fields: []MarshmallowField{
				{Name: "name", FieldType: "Str", Required: true},
			},
		},
		"ItemSchema": {
			Name: "ItemSchema",
			Fields: []MarshmallowField{
				{Name: "title", FieldType: "Str", Required: true},
				{Name: "tags", FieldType: "Nested", Required: false, Many: true, Nested: "TagSchema"},
			},
		},
	}

	resolver := NewFlaskTypeResolver(nil, marshmallowSchemas)

	result := resolver.Resolve("ItemSchema")
	if result.Kind != model.KindObject {
		t.Fatalf("Resolve(ItemSchema).Kind = %q, want %q", result.Kind, model.KindObject)
	}

	tagsField, ok := result.Fields["tags"]
	if !ok {
		t.Fatal("expected 'tags' field")
	}
	if tagsField.Model.Kind != model.KindArray {
		t.Errorf("ItemSchema.tags kind = %q, want %q", tagsField.Model.Kind, model.KindArray)
	}
	if tagsField.Model.Items == nil || tagsField.Model.Items.TypeName != "TagSchema" {
		t.Errorf("ItemSchema.tags items typeName = %q, want %q", tagsField.Model.Items.TypeName, "TagSchema")
	}
}

func TestFlaskTypeResolver_MixedPydanticAndMarshmallow(t *testing.T) {
	pydanticModels := map[string]fastapi.PydanticModel{
		"UserCreate": {
			Name: "UserCreate",
			Fields: []fastapi.PydanticField{
				{Name: "name", Type: "str", Required: true},
				{Name: "email", Type: "str", Required: true},
			},
		},
	}
	marshmallowSchemas := map[string]MarshmallowModel{
		"UserSchema": {
			Name: "UserSchema",
			Fields: []MarshmallowField{
				{Name: "id", FieldType: "Int", Required: false},
				{Name: "name", FieldType: "Str", Required: true},
			},
		},
	}

	resolver := NewFlaskTypeResolver(pydanticModels, marshmallowSchemas)

	pydanticResult := resolver.Resolve("UserCreate")
	if pydanticResult.Kind != model.KindObject {
		t.Fatalf("Resolve(UserCreate).Kind = %q, want %q", pydanticResult.Kind, model.KindObject)
	}
	if pydanticResult.TypeName != "UserCreate" {
		t.Errorf("Resolve(UserCreate).TypeName = %q, want %q", pydanticResult.TypeName, "UserCreate")
	}

	marshmallowResult := resolver.Resolve("UserSchema")
	if marshmallowResult.Kind != model.KindObject {
		t.Fatalf("Resolve(UserSchema).Kind = %q, want %q", marshmallowResult.Kind, model.KindObject)
	}
	if marshmallowResult.TypeName != "UserSchema" {
		t.Errorf("Resolve(UserSchema).TypeName = %q, want %q", marshmallowResult.TypeName, "UserSchema")
	}
}

func TestFlaskTypeResolver_Primitives(t *testing.T) {
	resolver := NewFlaskTypeResolver(nil, nil)

	tests := []struct {
		input    string
		kind     model.ObjectModelKind
		typeName string
	}{
		{"str", model.KindSingle, model.JsonTypeString},
		{"int", model.KindSingle, model.JsonTypeInt},
		{"float", model.KindSingle, model.JsonTypeFloat},
		{"bool", model.KindSingle, model.JsonTypeBoolean},
	}

	for _, tt := range tests {
		result := resolver.Resolve(tt.input)
		if result.Kind != tt.kind {
			t.Errorf("Resolve(%q).Kind = %q, want %q", tt.input, result.Kind, tt.kind)
		}
		if result.TypeName != tt.typeName {
			t.Errorf("Resolve(%q).TypeName = %q, want %q", tt.input, result.TypeName, tt.typeName)
		}
	}
}

func TestFlaskTypeResolver_ListOfPydanticModel(t *testing.T) {
	pydanticModels := map[string]fastapi.PydanticModel{
		"Item": {
			Name: "Item",
			Fields: []fastapi.PydanticField{
				{Name: "id", Type: "int", Required: true},
				{Name: "name", Type: "str", Required: true},
			},
		},
	}

	resolver := NewFlaskTypeResolver(pydanticModels, nil)

	result := resolver.Resolve("list[Item]")
	if result.Kind != model.KindArray {
		t.Fatalf("Resolve(list[Item]).Kind = %q, want %q", result.Kind, model.KindArray)
	}
	if result.Items == nil {
		t.Fatal("expected Items to be non-nil")
	}
	if result.Items.Kind != model.KindObject {
		t.Errorf("list[Item].Items.Kind = %q, want %q", result.Items.Kind, model.KindObject)
	}
	if result.Items.TypeName != "Item" {
		t.Errorf("list[Item].Items.TypeName = %q, want %q", result.Items.TypeName, "Item")
	}
}

func TestFlaskTypeResolver_EmptyType(t *testing.T) {
	resolver := NewFlaskTypeResolver(nil, nil)

	result := resolver.Resolve("")
	if !result.IsNull() {
		t.Errorf("Resolve('') should be null, got kind=%q typeName=%q", result.Kind, result.TypeName)
	}
}

func TestFlaskTypeResolver_MarshmallowInheritance(t *testing.T) {
	marshmallowSchemas := map[string]MarshmallowModel{
		"BaseSchema": {
			Name: "BaseSchema",
			Fields: []MarshmallowField{
				{Name: "id", FieldType: "Int", Required: true},
				{Name: "created_at", FieldType: "DateTime", Required: true},
			},
		},
		"UserSchema": {
			Name: "UserSchema",
			Fields: []MarshmallowField{
				{Name: "name", FieldType: "Str", Required: true},
				{Name: "email", FieldType: "Email", Required: true},
			},
			EmbeddedTypes: []string{"BaseSchema"},
		},
	}

	resolver := NewFlaskTypeResolver(nil, marshmallowSchemas)

	result := resolver.Resolve("UserSchema")
	if result.Kind != model.KindObject {
		t.Fatalf("Resolve(UserSchema).Kind = %q, want %q", result.Kind, model.KindObject)
	}

	if len(result.Fields) != 4 {
		t.Fatalf("Resolve(UserSchema).Fields count = %d, want 4 (2 own + 2 inherited)", len(result.Fields))
	}

	if _, ok := result.Fields["id"]; !ok {
		t.Error("expected inherited 'id' field")
	}
	if _, ok := result.Fields["created_at"]; !ok {
		t.Error("expected inherited 'created_at' field")
	}
	if _, ok := result.Fields["name"]; !ok {
		t.Error("expected own 'name' field")
	}
	if _, ok := result.Fields["email"]; !ok {
		t.Error("expected own 'email' field")
	}
}
