package fastapi

import (
	"testing"

	tree_sitter "github.com/tree-sitter/go-tree-sitter"
	python "github.com/tree-sitter/tree-sitter-python/bindings/go"

	model "github.com/tangcent/apilot/api-model"
)

func TestPythonTypeResolver_Primitives(t *testing.T) {
	resolver := NewPythonTypeResolver(nil)

	tests := []struct {
		input    string
		kind     model.ObjectModelKind
		typeName string
	}{
		{"str", model.KindSingle, model.JsonTypeString},
		{"int", model.KindSingle, model.JsonTypeInt},
		{"float", model.KindSingle, model.JsonTypeFloat},
		{"bool", model.KindSingle, model.JsonTypeBoolean},
		{"bytes", model.KindSingle, model.JsonTypeString},
		{"None", model.KindSingle, model.JsonTypeNull},
		{"Any", model.KindSingle, model.JsonTypeString},
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

func TestPythonTypeResolver_Optional(t *testing.T) {
	resolver := NewPythonTypeResolver(nil)

	result := resolver.Resolve("Optional[str]")
	if result.Kind != model.KindSingle {
		t.Errorf("Resolve(Optional[str]).Kind = %q, want %q", result.Kind, model.KindSingle)
	}
	if result.TypeName != model.JsonTypeString {
		t.Errorf("Resolve(Optional[str]).TypeName = %q, want %q", result.TypeName, model.JsonTypeString)
	}
}

func TestPythonTypeResolver_Union(t *testing.T) {
	resolver := NewPythonTypeResolver(nil)

	result := resolver.Resolve("Union[str, int]")
	if result.Kind != model.KindSingle {
		t.Errorf("Resolve(Union[str, int]).Kind = %q, want %q", result.Kind, model.KindSingle)
	}

	resultNone := resolver.Resolve("Union[str, None]")
	if resultNone.Kind != model.KindSingle {
		t.Errorf("Resolve(Union[str, None]).Kind = %q, want %q", resultNone.Kind, model.KindSingle)
	}
	if resultNone.TypeName != model.JsonTypeString {
		t.Errorf("Resolve(Union[str, None]).TypeName = %q, want %q", resultNone.TypeName, model.JsonTypeString)
	}
}

func TestPythonTypeResolver_Collections(t *testing.T) {
	resolver := NewPythonTypeResolver(nil)

	result := resolver.Resolve("List[str]")
	if result.Kind != model.KindArray {
		t.Errorf("Resolve(List[str]).Kind = %q, want %q", result.Kind, model.KindArray)
	}
	if result.Items == nil || result.Items.TypeName != model.JsonTypeString {
		t.Errorf("Resolve(List[str]).Items.TypeName = %q, want %q", result.Items.TypeName, model.JsonTypeString)
	}

	resultLower := resolver.Resolve("list[int]")
	if resultLower.Kind != model.KindArray {
		t.Errorf("Resolve(list[int]).Kind = %q, want %q", resultLower.Kind, model.KindArray)
	}
}

func TestPythonTypeResolver_Maps(t *testing.T) {
	resolver := NewPythonTypeResolver(nil)

	result := resolver.Resolve("Dict[str, int]")
	if result.Kind != model.KindMap {
		t.Errorf("Resolve(Dict[str, int]).Kind = %q, want %q", result.Kind, model.KindMap)
	}
	if result.KeyModel == nil || result.KeyModel.TypeName != model.JsonTypeString {
		t.Errorf("Resolve(Dict[str, int]).KeyModel.TypeName = %q, want %q", result.KeyModel.TypeName, model.JsonTypeString)
	}
	if result.ValueModel == nil || result.ValueModel.TypeName != model.JsonTypeInt {
		t.Errorf("Resolve(Dict[str, int]).ValueModel.TypeName = %q, want %q", result.ValueModel.TypeName, model.JsonTypeInt)
	}

	resultLower := resolver.Resolve("dict[str, str]")
	if resultLower.Kind != model.KindMap {
		t.Errorf("Resolve(dict[str, str]).Kind = %q, want %q", resultLower.Kind, model.KindMap)
	}
}

func TestPythonTypeResolver_PydanticModel(t *testing.T) {
	models := map[string]PydanticModel{
		"User": {
			Name: "User",
			Fields: []PydanticField{
				{Name: "name", Type: "str", Required: true},
				{Name: "email", Type: "str", Required: true},
				{Name: "age", Type: "int", Required: true},
			},
		},
	}

	resolver := NewPythonTypeResolver(models)

	result := resolver.Resolve("User")
	if result.Kind != model.KindObject {
		t.Fatalf("Resolve(User).Kind = %q, want %q", result.Kind, model.KindObject)
	}
	if result.TypeName != "User" {
		t.Errorf("Resolve(User).TypeName = %q, want %q", result.TypeName, "User")
	}
	if len(result.Fields) != 3 {
		t.Fatalf("Resolve(User).Fields count = %d, want 3", len(result.Fields))
	}

	nameField, ok := result.Fields["name"]
	if !ok {
		t.Fatal("expected 'name' field")
	}
	if nameField.Model.TypeName != model.JsonTypeString {
		t.Errorf("User.name model type = %q, want %q", nameField.Model.TypeName, model.JsonTypeString)
	}

	ageField, ok := result.Fields["age"]
	if !ok {
		t.Fatal("expected 'age' field")
	}
	if ageField.Model.TypeName != model.JsonTypeInt {
		t.Errorf("User.age model type = %q, want %q", ageField.Model.TypeName, model.JsonTypeInt)
	}
}

func TestPythonTypeResolver_NestedModel(t *testing.T) {
	models := map[string]PydanticModel{
		"Address": {
			Name: "Address",
			Fields: []PydanticField{
				{Name: "street", Type: "str", Required: true},
				{Name: "city", Type: "str", Required: true},
			},
		},
		"User": {
			Name: "User",
			Fields: []PydanticField{
				{Name: "name", Type: "str", Required: true},
				{Name: "address", Type: "Address", Required: true},
			},
		},
	}

	resolver := NewPythonTypeResolver(models)

	result := resolver.Resolve("User")
	if result.Kind != model.KindObject {
		t.Fatalf("Resolve(User).Kind = %q, want %q", result.Kind, model.KindObject)
	}

	addrField, ok := result.Fields["address"]
	if !ok {
		t.Fatal("expected 'address' field")
	}
	if addrField.Model.Kind != model.KindObject {
		t.Errorf("User.address kind = %q, want %q", addrField.Model.Kind, model.KindObject)
	}
	if addrField.Model.TypeName != "Address" {
		t.Errorf("User.address typeName = %q, want %q", addrField.Model.TypeName, "Address")
	}
	if len(addrField.Model.Fields) != 2 {
		t.Errorf("User.address fields count = %d, want 2", len(addrField.Model.Fields))
	}
}

func TestPythonTypeResolver_ListOfModel(t *testing.T) {
	models := map[string]PydanticModel{
		"Item": {
			Name: "Item",
			Fields: []PydanticField{
				{Name: "id", Type: "int", Required: true},
				{Name: "name", Type: "str", Required: true},
			},
		},
	}

	resolver := NewPythonTypeResolver(models)

	result := resolver.Resolve("List[Item]")
	if result.Kind != model.KindArray {
		t.Fatalf("Resolve(List[Item]).Kind = %q, want %q", result.Kind, model.KindArray)
	}
	if result.Items == nil {
		t.Fatal("expected Items to be non-nil")
	}
	if result.Items.Kind != model.KindObject {
		t.Errorf("List[Item].Items.Kind = %q, want %q", result.Items.Kind, model.KindObject)
	}
	if result.Items.TypeName != "Item" {
		t.Errorf("List[Item].Items.TypeName = %q, want %q", result.Items.TypeName, "Item")
	}
}

func TestPythonTypeResolver_CircularReference(t *testing.T) {
	models := map[string]PydanticModel{
		"Node": {
			Name: "Node",
			Fields: []PydanticField{
				{Name: "value", Type: "str", Required: true},
				{Name: "child", Type: "Node", Required: false},
			},
		},
	}

	resolver := NewPythonTypeResolver(models)

	result := resolver.Resolve("Node")
	if result.Kind != model.KindObject {
		t.Fatalf("Resolve(Node).Kind = %q, want %q", result.Kind, model.KindObject)
	}

	childField, ok := result.Fields["child"]
	if !ok {
		t.Fatal("expected 'child' field")
	}
	if childField.Model.Kind != model.KindRef {
		t.Errorf("Node.child kind = %q, want %q (circular ref)", childField.Model.Kind, model.KindRef)
	}
}

func TestPythonTypeResolver_UnknownType(t *testing.T) {
	resolver := NewPythonTypeResolver(nil)

	result := resolver.Resolve("UnknownType")
	if result.Kind != model.KindSingle {
		t.Errorf("Resolve(UnknownType).Kind = %q, want %q", result.Kind, model.KindSingle)
	}
	if result.TypeName != "UnknownType" {
		t.Errorf("Resolve(UnknownType).TypeName = %q, want %q", result.TypeName, "UnknownType")
	}
}

func TestPythonTypeResolver_EmptyType(t *testing.T) {
	resolver := NewPythonTypeResolver(nil)

	result := resolver.Resolve("")
	if !result.IsNull() {
		t.Errorf("Resolve('').should be null, got kind=%q typeName=%q", result.Kind, result.TypeName)
	}
}

func TestParsePythonGenericType(t *testing.T) {
	tests := []struct {
		input    string
		base     string
		args     []string
	}{
		{"str", "str", nil},
		{"List[str]", "List", []string{"str"}},
		{"Dict[str, int]", "Dict", []string{"str", "int"}},
		{"Optional[str]", "Optional", []string{"str"}},
		{"Union[str, int, None]", "Union", []string{"str", "int", "None"}},
		{"List[Dict[str, int]]", "List", []string{"Dict[str, int]"}},
		{"Result[List[User]]", "Result", []string{"List[User]"}},
	}

	for _, tt := range tests {
		base, args := ParsePythonGenericType(tt.input)
		if base != tt.base {
			t.Errorf("ParsePythonGenericType(%q) base = %q, want %q", tt.input, base, tt.base)
		}
		if len(args) != len(tt.args) {
			t.Errorf("ParsePythonGenericType(%q) args count = %d, want %d", tt.input, len(args), len(tt.args))
			continue
		}
		for i, arg := range args {
			if arg != tt.args[i] {
				t.Errorf("ParsePythonGenericType(%q) args[%d] = %q, want %q", tt.input, i, arg, tt.args[i])
			}
		}
	}
}

func TestPythonTypeResolver_Inheritance(t *testing.T) {
	models := map[string]PydanticModel{
		"BaseEntity": {
			Name: "BaseEntity",
			Fields: []PydanticField{
				{Name: "id", Type: "int", Required: true},
				{Name: "created_at", Type: "str", Required: true},
			},
		},
		"User": {
			Name: "User",
			Fields: []PydanticField{
				{Name: "name", Type: "str", Required: true},
				{Name: "email", Type: "str", Required: true},
			},
			EmbeddedTypes: []string{"BaseEntity"},
		},
	}

	resolver := NewPythonTypeResolver(models)

	result := resolver.Resolve("User")
	if result.Kind != model.KindObject {
		t.Fatalf("Resolve(User).Kind = %q, want %q", result.Kind, model.KindObject)
	}

	if len(result.Fields) != 4 {
		t.Fatalf("Resolve(User).Fields count = %d, want 4 (2 own + 2 inherited)", len(result.Fields))
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

func TestExtractPydanticModels(t *testing.T) {
	source := []byte(`
from pydantic import BaseModel
from typing import List

class User(BaseModel):
    name: str
    email: str
    age: int = 0

class Address(BaseModel):
    street: str
    city: str

class NotAModel:
    pass
`)

	p := createTestParser(t)
	tree := p.Parse(source, nil)
	defer tree.Close()

	rootNode := tree.RootNode()
	models := ExtractPydanticModels(rootNode, source)

	if len(models) != 2 {
		t.Fatalf("expected 2 Pydantic models, got %d", len(models))
	}

	userModel, ok := models["User"]
	if !ok {
		t.Fatal("expected 'User' model")
	}
	if len(userModel.Fields) != 3 {
		t.Errorf("User fields count = %d, want 3", len(userModel.Fields))
	}

	addrModel, ok := models["Address"]
	if !ok {
		t.Fatal("expected 'Address' model")
	}
	if len(addrModel.Fields) != 2 {
		t.Errorf("Address fields count = %d, want 2", len(addrModel.Fields))
	}

	if _, ok := models["NotAModel"]; ok {
		t.Error("NotAModel should not be extracted as it doesn't extend BaseModel")
	}
}

func TestExtractPydanticModels_Inheritance(t *testing.T) {
	source := []byte(`
from pydantic import BaseModel

class BaseEntity(BaseModel):
    id: int
    created_at: str

class User(BaseEntity):
    name: str
    email: str
`)

	p := createTestParser(t)
	tree := p.Parse(source, nil)
	defer tree.Close()

	rootNode := tree.RootNode()
	models := ExtractPydanticModels(rootNode, source)

	if len(models) != 2 {
		t.Fatalf("expected 2 models, got %d", len(models))
	}

	userModel, ok := models["User"]
	if !ok {
		t.Fatal("expected 'User' model")
	}
	if len(userModel.EmbeddedTypes) != 1 || userModel.EmbeddedTypes[0] != "BaseEntity" {
		t.Errorf("User.EmbeddedTypes = %v, want [BaseEntity]", userModel.EmbeddedTypes)
	}
}

func createTestParser(t *testing.T) *tree_sitter.Parser {
	t.Helper()
	p := tree_sitter.NewParser()
	lang := tree_sitter.NewLanguage(python.Language())
	if err := p.SetLanguage(lang); err != nil {
		t.Fatalf("failed to set language: %v", err)
	}
	return p
}
