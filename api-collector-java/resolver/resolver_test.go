package resolver

import (
	"testing"

	"github.com/tangcent/apilot/api-collector-java/parser"
	model "github.com/tangcent/apilot/api-model"
)

func TestResolve_Primitives(t *testing.T) {
	r := NewTypeResolver(nil)

	tests := []struct {
		typeName   string
		expectKind model.ObjectModelKind
		expectJson string
	}{
		{"int", model.KindSingle, model.JsonTypeInt},
		{"long", model.KindSingle, model.JsonTypeLong},
		{"float", model.KindSingle, model.JsonTypeFloat},
		{"double", model.KindSingle, model.JsonTypeDouble},
		{"boolean", model.KindSingle, model.JsonTypeBoolean},
		{"String", model.KindSingle, model.JsonTypeString},
		{"Integer", model.KindSingle, model.JsonTypeInt},
		{"Long", model.KindSingle, model.JsonTypeLong},
		{"Float", model.KindSingle, model.JsonTypeFloat},
		{"Double", model.KindSingle, model.JsonTypeDouble},
		{"Boolean", model.KindSingle, model.JsonTypeBoolean},
		{"void", model.KindSingle, model.JsonTypeNull},
		{"Void", model.KindSingle, model.JsonTypeNull},
	}

	for _, tt := range tests {
		t.Run(tt.typeName, func(t *testing.T) {
			result := r.Resolve(tt.typeName, nil)
			if result.Kind != tt.expectKind {
				t.Errorf("Expected kind %s, got %s", tt.expectKind, result.Kind)
			}
			if result.TypeName != tt.expectJson {
				t.Errorf("Expected typeName %s, got %s", tt.expectJson, result.TypeName)
			}
		})
	}
}

func TestResolve_SimpleClass(t *testing.T) {
	classes := []parser.Class{
		{
			Name: "User",
			Fields: []parser.Field{
				{Name: "id", Type: "Long"},
				{Name: "name", Type: "String"},
				{Name: "age", Type: "int"},
			},
		},
	}

	r := NewTypeResolver(classes)
	result := r.Resolve("User", nil)

	if result.Kind != model.KindObject {
		t.Fatalf("Expected KindObject, got %s", result.Kind)
	}
	if result.TypeName != "User" {
		t.Errorf("Expected typeName 'User', got '%s'", result.TypeName)
	}
	if len(result.Fields) != 3 {
		t.Fatalf("Expected 3 fields, got %d", len(result.Fields))
	}

	idField := result.Fields["id"]
	if idField == nil {
		t.Fatal("Expected 'id' field")
	}
	if idField.Model.TypeName != model.JsonTypeLong {
		t.Errorf("Expected id type 'long', got '%s'", idField.Model.TypeName)
	}

	nameField := result.Fields["name"]
	if nameField == nil {
		t.Fatal("Expected 'name' field")
	}
	if nameField.Model.TypeName != model.JsonTypeString {
		t.Errorf("Expected name type 'string', got '%s'", nameField.Model.TypeName)
	}
}

func TestResolve_GenericClass(t *testing.T) {
	classes := []parser.Class{
		{
			Name:           "Result",
			TypeParameters: []string{"T"},
			Fields: []parser.Field{
				{Name: "code", Type: "int"},
				{Name: "message", Type: "String"},
				{Name: "data", Type: "T"},
			},
		},
		{
			Name: "User",
			Fields: []parser.Field{
				{Name: "id", Type: "Long"},
				{Name: "name", Type: "String"},
			},
		},
	}

	r := NewTypeResolver(classes)
	result := r.Resolve("Result<User>", nil)

	if result.Kind != model.KindObject {
		t.Fatalf("Expected KindObject, got %s", result.Kind)
	}
	if result.TypeName != "Result" {
		t.Errorf("Expected typeName 'Result', got '%s'", result.TypeName)
	}

	dataField := result.Fields["data"]
	if dataField == nil {
		t.Fatal("Expected 'data' field")
	}
	if dataField.Model.Kind != model.KindObject {
		t.Errorf("Expected data model KindObject, got %s", dataField.Model.Kind)
	}
	if dataField.Model.TypeName != "User" {
		t.Errorf("Expected data model typeName 'User', got '%s'", dataField.Model.TypeName)
	}
}

func TestResolve_CollectionTypes(t *testing.T) {
	classes := []parser.Class{
		{
			Name: "User",
			Fields: []parser.Field{
				{Name: "id", Type: "Long"},
			},
		},
	}

	r := NewTypeResolver(classes)

	t.Run("List<User>", func(t *testing.T) {
		result := r.Resolve("List<User>", nil)
		if result.Kind != model.KindArray {
			t.Fatalf("Expected KindArray, got %s", result.Kind)
		}
		if result.Items == nil {
			t.Fatal("Expected non-nil Items")
		}
		if result.Items.Kind != model.KindObject {
			t.Errorf("Expected item KindObject, got %s", result.Items.Kind)
		}
		if result.Items.TypeName != "User" {
			t.Errorf("Expected item typeName 'User', got '%s'", result.Items.TypeName)
		}
	})

	t.Run("ArrayList<User>", func(t *testing.T) {
		result := r.Resolve("ArrayList<User>", nil)
		if result.Kind != model.KindArray {
			t.Fatalf("Expected KindArray, got %s", result.Kind)
		}
	})

	t.Run("Set<String>", func(t *testing.T) {
		result := r.Resolve("Set<String>", nil)
		if result.Kind != model.KindArray {
			t.Fatalf("Expected KindArray, got %s", result.Kind)
		}
	})
}

func TestResolve_MapTypes(t *testing.T) {
	classes := []parser.Class{
		{
			Name: "User",
			Fields: []parser.Field{
				{Name: "id", Type: "Long"},
			},
		},
	}

	r := NewTypeResolver(classes)

	t.Run("Map<String, User>", func(t *testing.T) {
		result := r.Resolve("Map<String, User>", nil)
		if result.Kind != model.KindMap {
			t.Fatalf("Expected KindMap, got %s", result.Kind)
		}
		if result.KeyModel == nil || result.ValueModel == nil {
			t.Fatal("Expected non-nil KeyModel and ValueModel")
		}
		if result.KeyModel.TypeName != model.JsonTypeString {
			t.Errorf("Expected key typeName 'string', got '%s'", result.KeyModel.TypeName)
		}
		if result.ValueModel.TypeName != "User" {
			t.Errorf("Expected value typeName 'User', got '%s'", result.ValueModel.TypeName)
		}
	})

	t.Run("HashMap<String, String>", func(t *testing.T) {
		result := r.Resolve("HashMap<String, String>", nil)
		if result.Kind != model.KindMap {
			t.Fatalf("Expected KindMap, got %s", result.Kind)
		}
	})
}

func TestResolve_UnknownType(t *testing.T) {
	r := NewTypeResolver(nil)
	result := r.Resolve("UnknownType", nil)

	if result.Kind != model.KindSingle {
		t.Fatalf("Expected KindSingle for unknown type, got %s", result.Kind)
	}
	if result.TypeName != "UnknownType" {
		t.Errorf("Expected typeName 'UnknownType', got '%s'", result.TypeName)
	}
}

func TestResolve_CircularReference(t *testing.T) {
	classes := []parser.Class{
		{
			Name: "Node",
			Fields: []parser.Field{
				{Name: "value", Type: "String"},
				{Name: "next", Type: "Node"},
			},
		},
	}

	r := NewTypeResolver(classes)
	result := r.Resolve("Node", nil)

	if result.Kind != model.KindObject {
		t.Fatalf("Expected KindObject, got %s", result.Kind)
	}

	nextField := result.Fields["next"]
	if nextField == nil {
		t.Fatal("Expected 'next' field")
	}
	if nextField.Model.Kind != model.KindRef {
		t.Errorf("Expected 'next' field KindRef for circular reference, got %s", nextField.Model.Kind)
	}
	if nextField.Model.TypeName != "Node" {
		t.Errorf("Expected 'next' field typeName 'Node', got '%s'", nextField.Model.TypeName)
	}
}

func TestResolve_NestedGenerics(t *testing.T) {
	classes := []parser.Class{
		{
			Name:           "Result",
			TypeParameters: []string{"T"},
			Fields: []parser.Field{
				{Name: "code", Type: "int"},
				{Name: "message", Type: "String"},
				{Name: "data", Type: "T"},
			},
		},
		{
			Name: "User",
			Fields: []parser.Field{
				{Name: "id", Type: "Long"},
				{Name: "name", Type: "String"},
			},
		},
	}

	r := NewTypeResolver(classes)

	t.Run("Result<List<User>>", func(t *testing.T) {
		result := r.Resolve("Result<List<User>>", nil)
		if result.Kind != model.KindObject {
			t.Fatalf("Expected KindObject, got %s", result.Kind)
		}

		dataField := result.Fields["data"]
		if dataField == nil {
			t.Fatal("Expected 'data' field")
		}
		if dataField.Model.Kind != model.KindArray {
			t.Fatalf("Expected data KindArray, got %s", dataField.Model.Kind)
		}
		if dataField.Model.Items == nil || dataField.Model.Items.TypeName != "User" {
			t.Errorf("Expected data items typeName 'User', got '%v'", dataField.Model.Items)
		}
	})
}

func TestResolve_TypeBindings(t *testing.T) {
	classes := []parser.Class{
		{
			Name:           "BaseController",
			TypeParameters: []string{"Req", "Res"},
			Fields: []parser.Field{
				{Name: "request", Type: "Req"},
				{Name: "response", Type: "Res"},
			},
		},
		{
			Name: "CreateOrderReq",
			Fields: []parser.Field{
				{Name: "orderId", Type: "String"},
			},
		},
		{
			Name: "OrderVO",
			Fields: []parser.Field{
				{Name: "id", Type: "Long"},
				{Name: "total", Type: "double"},
			},
		},
	}

	r := NewTypeResolver(classes)

	typeBindings := map[string]string{
		"Req": "CreateOrderReq",
		"Res": "OrderVO",
	}

	result := r.Resolve("BaseController", typeBindings)

	if result.Kind != model.KindObject {
		t.Fatalf("Expected KindObject, got %s", result.Kind)
	}

	reqField := result.Fields["request"]
	if reqField == nil {
		t.Fatal("Expected 'request' field")
	}
	if reqField.Model.Kind != model.KindObject {
		t.Errorf("Expected request KindObject, got %s", reqField.Model.Kind)
	}
	if reqField.Model.TypeName != "CreateOrderReq" {
		t.Errorf("Expected request typeName 'CreateOrderReq', got '%s'", reqField.Model.TypeName)
	}

	resField := result.Fields["response"]
	if resField == nil {
		t.Fatal("Expected 'response' field")
	}
	if resField.Model.TypeName != "OrderVO" {
		t.Errorf("Expected response typeName 'OrderVO', got '%s'", resField.Model.TypeName)
	}
}

func TestParseGenericType(t *testing.T) {
	tests := []struct {
		input    string
		expected string
		args     []string
	}{
		{"String", "String", nil},
		{"List<User>", "List", []string{"User"}},
		{"Map<String, User>", "Map", []string{"String", "User"}},
		{"ResponseEntity<List<User>>", "ResponseEntity", []string{"List<User>"}},
		{"Result<Map<String, User>>", "Result", []string{"Map<String, User>"}},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			baseName, args := parseGenericType(tt.input)
			if baseName != tt.expected {
				t.Errorf("Expected base '%s', got '%s'", tt.expected, baseName)
			}
			if len(args) != len(tt.args) {
				t.Errorf("Expected %d args, got %d", len(tt.args), len(args))
				return
			}
			for i, arg := range args {
				if arg != tt.args[i] {
					t.Errorf("Arg %d: expected '%s', got '%s'", i, tt.args[i], arg)
				}
			}
		})
	}
}

func TestResolve_EmptyType(t *testing.T) {
	r := NewTypeResolver(nil)
	result := r.Resolve("", nil)
	if result.Kind != model.KindSingle {
		t.Fatalf("Expected KindSingle for empty type, got %s", result.Kind)
	}
	if result.TypeName != model.JsonTypeNull {
		t.Errorf("Expected typeName 'null', got '%s'", result.TypeName)
	}
}

func TestResolve_PageResult(t *testing.T) {
	classes := []parser.Class{
		{
			Name:           "PageResult",
			TypeParameters: []string{"T"},
			Fields: []parser.Field{
				{Name: "total", Type: "long"},
				{Name: "page", Type: "int"},
				{Name: "size", Type: "int"},
				{Name: "items", Type: "List<T>"},
			},
		},
		{
			Name: "Order",
			Fields: []parser.Field{
				{Name: "id", Type: "Long"},
				{Name: "amount", Type: "double"},
			},
		},
	}

	r := NewTypeResolver(classes)
	result := r.Resolve("PageResult<Order>", nil)

	if result.Kind != model.KindObject {
		t.Fatalf("Expected KindObject, got %s", result.Kind)
	}
	if result.TypeName != "PageResult" {
		t.Errorf("Expected typeName 'PageResult', got '%s'", result.TypeName)
	}

	itemsField := result.Fields["items"]
	if itemsField == nil {
		t.Fatal("Expected 'items' field")
	}
	if itemsField.Model.Kind != model.KindArray {
		t.Fatalf("Expected items KindArray, got %s", itemsField.Model.Kind)
	}
	if itemsField.Model.Items == nil || itemsField.Model.Items.TypeName != "Order" {
		t.Errorf("Expected items element typeName 'Order', got '%v'", itemsField.Model.Items)
	}

	totalField := result.Fields["total"]
	if totalField == nil {
		t.Fatal("Expected 'total' field")
	}
	if totalField.Model.TypeName != model.JsonTypeLong {
		t.Errorf("Expected total typeName 'long', got '%s'", totalField.Model.TypeName)
	}
}
