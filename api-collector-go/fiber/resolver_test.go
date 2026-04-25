package fiber

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"

	model "github.com/tangcent/apilot/api-model"
)

func parseSource(t *testing.T, src string) *ast.File {
	t.Helper()
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "test.go", src, parser.ParseComments)
	if err != nil {
		t.Fatalf("Failed to parse source: %v", err)
	}
	return f
}

func TestExtractStructs(t *testing.T) {
	src := `package main

type User struct {
	Name  string  ` + "`json:\"name\" validate:\"required\"`" + `
	Email string  ` + "`json:\"email\"`" + `
	Age   int     ` + "`json:\"age,omitempty\"`" + `
}

type Admin struct {
	User
	Role string ` + "`json:\"role\"`" + `
}
`
	f := parseSource(t, src)
	structs := extractStructs(f)

	if len(structs) != 2 {
		t.Fatalf("Expected 2 structs, got %d", len(structs))
	}

	user := structs["User"]
	if len(user.Fields) != 3 {
		t.Fatalf("Expected 3 fields in User, got %d", len(user.Fields))
	}
	if user.Fields[0].JsonTag != "name" {
		t.Errorf("Expected json tag 'name', got '%s'", user.Fields[0].JsonTag)
	}
	if user.Fields[0].BindingTag != "required" {
		t.Errorf("Expected binding tag 'required' from validate tag, got '%s'", user.Fields[0].BindingTag)
	}
	if user.Fields[2].JsonTag != "age" {
		t.Errorf("Expected json tag 'age', got '%s'", user.Fields[2].JsonTag)
	}

	admin := structs["Admin"]
	if len(admin.EmbeddedTypes) != 1 {
		t.Fatalf("Expected 1 embedded type in Admin, got %d", len(admin.EmbeddedTypes))
	}
	if admin.EmbeddedTypes[0] != "User" {
		t.Errorf("Expected embedded type 'User', got '%s'", admin.EmbeddedTypes[0])
	}
}

func TestBuildVarTypeMap(t *testing.T) {
	src := `package main

func handler() {
	var req CreateUserReq
	var resp *UserResponse
	item := Item{}
	ptr := &Product{}
}
`
	f := parseSource(t, src)
	fn := f.Decls[0].(*ast.FuncDecl)
	varMap := buildVarTypeMap(fn)

	tests := []struct {
		varName  string
		typeName string
	}{
		{"req", "CreateUserReq"},
		{"resp", "*UserResponse"},
		{"item", "Item"},
		{"ptr", "Product"},
	}

	for _, tt := range tests {
		if got := varMap[tt.varName]; got != tt.typeName {
			t.Errorf("varMap[%q] = %q, want %q", tt.varName, got, tt.typeName)
		}
	}
}

func TestResolve_Primitives(t *testing.T) {
	resolver := NewTypeResolver(nil)

	tests := []struct {
		goType   string
		jsonType string
	}{
		{"string", model.JsonTypeString},
		{"int", model.JsonTypeInt},
		{"int64", model.JsonTypeLong},
		{"float64", model.JsonTypeDouble},
		{"bool", model.JsonTypeBoolean},
	}

	for _, tt := range tests {
		result := resolver.Resolve(tt.goType)
		if !result.IsSingle() {
			t.Errorf("Resolve(%q): expected single model, got kind=%s", tt.goType, result.Kind)
		}
		if result.TypeName != tt.jsonType {
			t.Errorf("Resolve(%q): got type %q, want %q", tt.goType, result.TypeName, tt.jsonType)
		}
	}
}

func TestResolve_Pointers(t *testing.T) {
	resolver := NewTypeResolver(nil)
	result := resolver.Resolve("*string")

	if !result.IsSingle() {
		t.Errorf("Expected single model for *string, got kind=%s", result.Kind)
	}
	if result.TypeName != model.JsonTypeString {
		t.Errorf("Expected string type, got %q", result.TypeName)
	}
}

func TestResolve_Slices(t *testing.T) {
	resolver := NewTypeResolver(nil)
	result := resolver.Resolve("[]string")

	if !result.IsArray() {
		t.Fatalf("Expected array model, got kind=%s", result.Kind)
	}
	if result.Items == nil || !result.Items.IsSingle() {
		t.Error("Expected array items to be single model")
	}
	if result.Items.TypeName != model.JsonTypeString {
		t.Errorf("Expected string items, got %q", result.Items.TypeName)
	}
}

func TestResolve_Maps(t *testing.T) {
	resolver := NewTypeResolver(nil)
	result := resolver.Resolve("map[string]int")

	if !result.IsMap() {
		t.Fatalf("Expected map model, got kind=%s", result.Kind)
	}
	if result.KeyModel == nil || result.KeyModel.TypeName != model.JsonTypeString {
		t.Error("Expected string key type")
	}
	if result.ValueModel == nil || result.ValueModel.TypeName != model.JsonTypeInt {
		t.Error("Expected int value type")
	}
}

func TestResolve_FiberMap(t *testing.T) {
	resolver := NewTypeResolver(nil)
	result := resolver.Resolve("fiber.Map")

	if !result.IsMap() {
		t.Fatalf("Expected map model for fiber.Map, got kind=%s", result.Kind)
	}
}

func TestResolve_Struct(t *testing.T) {
	structs := map[string]StructDef{
		"User": {
			Name: "User",
			Fields: []StructField{
				{Name: "Name", Type: "string", JsonTag: "name", BindingTag: "required"},
				{Name: "Email", Type: "string", JsonTag: "email"},
			},
		},
	}

	resolver := NewTypeResolver(structs)
	result := resolver.Resolve("User")

	if !result.IsObject() {
		t.Fatalf("Expected object model, got kind=%s", result.Kind)
	}
	if len(result.Fields) != 2 {
		t.Fatalf("Expected 2 fields, got %d", len(result.Fields))
	}
	if result.Fields["name"] == nil {
		t.Error("Expected 'name' field")
	}
	if !result.Fields["name"].Required {
		t.Error("Expected 'name' field to be required")
	}
	if result.Fields["email"] == nil {
		t.Error("Expected 'email' field")
	}
}

func TestResolve_EmbeddedStruct(t *testing.T) {
	structs := map[string]StructDef{
		"Base": {
			Name: "Base",
			Fields: []StructField{
				{Name: "ID", Type: "int64", JsonTag: "id"},
			},
		},
		"User": {
			Name:          "User",
			EmbeddedTypes: []string{"Base"},
			Fields: []StructField{
				{Name: "Name", Type: "string", JsonTag: "name"},
			},
		},
	}

	resolver := NewTypeResolver(structs)
	result := resolver.Resolve("User")

	if !result.IsObject() {
		t.Fatalf("Expected object model, got kind=%s", result.Kind)
	}
	if len(result.Fields) != 2 {
		t.Fatalf("Expected 2 fields (1 own + 1 embedded), got %d", len(result.Fields))
	}
	if result.Fields["id"] == nil {
		t.Error("Expected embedded 'id' field from Base")
	}
	if result.Fields["name"] == nil {
		t.Error("Expected 'name' field")
	}
}

func TestResolve_CircularReference(t *testing.T) {
	structs := map[string]StructDef{
		"Node": {
			Name: "Node",
			Fields: []StructField{
				{Name: "Value", Type: "string", JsonTag: "value"},
				{Name: "Next", Type: "*Node", JsonTag: "next"},
			},
		},
	}

	resolver := NewTypeResolver(structs)
	result := resolver.Resolve("Node")

	if !result.IsObject() {
		t.Fatalf("Expected object model, got kind=%s", result.Kind)
	}
	nextField := result.Fields["next"]
	if nextField == nil {
		t.Fatal("Expected 'next' field")
	}
	if !nextField.Model.IsRef() {
		t.Errorf("Expected ref model for circular reference, got kind=%s", nextField.Model.Kind)
	}
}

func TestResolve_EmptyAndNull(t *testing.T) {
	resolver := NewTypeResolver(nil)

	if result := resolver.Resolve(""); !result.IsSingle() || result.TypeName != model.JsonTypeNull {
		t.Errorf("Expected null for empty type, got kind=%s typeName=%s", result.Kind, result.TypeName)
	}

	if result := resolver.Resolve("interface{}"); !result.IsSingle() || result.TypeName != model.JsonTypeNull {
		t.Errorf("Expected null for interface{}, got kind=%s typeName=%s", result.Kind, result.TypeName)
	}

	if result := resolver.Resolve("struct{}"); !result.IsObject() {
		t.Errorf("Expected object for struct{}, got kind=%s", result.Kind)
	}
}

func TestResolve_UnknownType(t *testing.T) {
	resolver := NewTypeResolver(nil)
	result := resolver.Resolve("CustomType")

	if !result.IsSingle() {
		t.Errorf("Expected single model for unknown type, got kind=%s", result.Kind)
	}
	if result.TypeName != "CustomType" {
		t.Errorf("Expected typeName 'CustomType', got %q", result.TypeName)
	}
}

func TestResolveVarType(t *testing.T) {
	varTypeMaps := map[string]map[string]string{
		"handler": {
			"req": "CreateUserReq",
		},
	}

	if got := resolveVarType("req", "handler", varTypeMaps); got != "CreateUserReq" {
		t.Errorf("resolveVarType: got %q, want %q", got, "CreateUserReq")
	}

	if got := resolveVarType("unknown", "handler", varTypeMaps); got != "unknown" {
		t.Errorf("resolveVarType: got %q, want %q", got, "unknown")
	}

	if got := resolveVarType("req", "otherHandler", varTypeMaps); got != "req" {
		t.Errorf("resolveVarType: got %q, want %q", got, "req")
	}
}
