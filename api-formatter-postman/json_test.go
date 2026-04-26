package postman

import (
	"encoding/json"
	"strings"
	"testing"

	apimodel "github.com/tangcent/apilot/api-model"
)

func TestObjectModelToJSON_SingleTypes(t *testing.T) {
	tests := []struct {
		name     string
		model    *apimodel.ObjectModel
		expected string
	}{
		{"string", apimodel.SingleModel(apimodel.JsonTypeString), `""`},
		{"int", apimodel.SingleModel(apimodel.JsonTypeInt), `0`},
		{"long", apimodel.SingleModel(apimodel.JsonTypeLong), `0`},
		{"float", apimodel.SingleModel(apimodel.JsonTypeFloat), `0`},
		{"double", apimodel.SingleModel(apimodel.JsonTypeDouble), `0`},
		{"boolean", apimodel.SingleModel(apimodel.JsonTypeBoolean), `false`},
		{"null", apimodel.SingleModel(apimodel.JsonTypeNull), `null`},
		{"unknown", apimodel.SingleModel("CustomType"), `""`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := objectModelToJSON(tt.model)
			if strings.TrimSpace(result) != tt.expected {
				t.Errorf("objectModelToJSON(%s) = %q, want %q", tt.name, result, tt.expected)
			}
		})
	}
}

func TestObjectModelToJSON_NilModel(t *testing.T) {
	result := objectModelToJSON(nil)
	if result != "{}" {
		t.Errorf("objectModelToJSON(nil) = %q, want %q", result, "{}")
	}
}

func TestObjectModelToJSON_EmptyObject(t *testing.T) {
	m := apimodel.EmptyObject()
	result := objectModelToJSON(m)
	if result != "{}" {
		t.Errorf("objectModelToJSON(EmptyObject) = %q, want %q", result, "{}")
	}
}

func TestObjectModelToJSON_ObjectWithFields(t *testing.T) {
	m := apimodel.NewObjectModelBuilder().
		StringField("name", apimodel.WithDemo("Alice")).
		IntField("age", apimodel.WithDemo("30")).
		Build()

	result := objectModelToJSON(m)

	var obj map[string]any
	if err := json.Unmarshal([]byte(result), &obj); err != nil {
		t.Fatalf("Result is not valid JSON: %v", err)
	}
	if obj["name"] != "Alice" {
		t.Errorf("name = %v, want Alice", obj["name"])
	}
	if obj["age"] != float64(30) {
		t.Errorf("age = %v, want 30", obj["age"])
	}
}

func TestObjectModelToJSON_ObjectWithDefaultValues(t *testing.T) {
	m := apimodel.NewObjectModelBuilder().
		StringField("status", apimodel.WithDefault("active")).
		BoolField("enabled", apimodel.WithDefault("true")).
		Build()

	result := objectModelToJSON(m)

	var obj map[string]any
	if err := json.Unmarshal([]byte(result), &obj); err != nil {
		t.Fatalf("Result is not valid JSON: %v", err)
	}
	if obj["status"] != "active" {
		t.Errorf("status = %v, want active", obj["status"])
	}
	if obj["enabled"] != true {
		t.Errorf("enabled = %v, want true", obj["enabled"])
	}
}

func TestObjectModelToJSON_NestedObject(t *testing.T) {
	addressModel := apimodel.NewObjectModelBuilder().
		StringField("street", apimodel.WithDemo("123 Main St")).
		StringField("city", apimodel.WithDemo("Springfield")).
		Build()
	userModel := apimodel.NewObjectModelBuilder().
		StringField("name", apimodel.WithDemo("Alice")).
		ObjectField("address", addressModel).
		Build()

	result := objectModelToJSON(userModel)

	var obj map[string]any
	if err := json.Unmarshal([]byte(result), &obj); err != nil {
		t.Fatalf("Result is not valid JSON: %v", err)
	}
	if obj["name"] != "Alice" {
		t.Errorf("name = %v, want Alice", obj["name"])
	}
	addr, ok := obj["address"].(map[string]any)
	if !ok {
		t.Fatal("address should be an object")
	}
	if addr["street"] != "123 Main St" {
		t.Errorf("address.street = %v, want 123 Main St", addr["street"])
	}
	if addr["city"] != "Springfield" {
		t.Errorf("address.city = %v, want Springfield", addr["city"])
	}
}

func TestObjectModelToJSON_Array(t *testing.T) {
	itemModel := apimodel.NewObjectModelBuilder().
		StringField("name", apimodel.WithDemo("Alice")).
		Build()
	m := apimodel.ArrayModel(itemModel)

	result := objectModelToJSON(m)

	var arr []any
	if err := json.Unmarshal([]byte(result), &arr); err != nil {
		t.Fatalf("Result is not valid JSON array: %v", err)
	}
	if len(arr) != 1 {
		t.Fatalf("Array length = %d, want 1", len(arr))
	}
	item, ok := arr[0].(map[string]any)
	if !ok {
		t.Fatal("Array item should be an object")
	}
	if item["name"] != "Alice" {
		t.Errorf("item.name = %v, want Alice", item["name"])
	}
}

func TestObjectModelToJSON_EmptyArray(t *testing.T) {
	m := apimodel.ArrayModel(nil)
	result := objectModelToJSON(m)
	if result != "[]" {
		t.Errorf("objectModelToJSON(ArrayModel(nil)) = %q, want %q", result, "[]")
	}
}

func TestObjectModelToJSON_Map(t *testing.T) {
	m := apimodel.MapModel(
		apimodel.SingleModel(apimodel.JsonTypeString),
		apimodel.SingleModel(apimodel.JsonTypeInt),
	)

	result := objectModelToJSON(m)

	var obj map[string]any
	if err := json.Unmarshal([]byte(result), &obj); err != nil {
		t.Fatalf("Result is not valid JSON: %v", err)
	}
	if len(obj) != 1 {
		t.Fatalf("Map should have 1 entry, got %d", len(obj))
	}
	if _, ok := obj["key"]; !ok {
		t.Error("Map should have 'key' entry")
	}
}

func TestObjectModelToJSON_Ref(t *testing.T) {
	m := apimodel.RefModel("User")
	result := objectModelToJSON(m)

	var obj map[string]any
	if err := json.Unmarshal([]byte(result), &obj); err != nil {
		t.Fatalf("Result is not valid JSON: %v", err)
	}
	if obj["$ref"] != "User" {
		t.Errorf("$ref = %v, want User", obj["$ref"])
	}
}

func TestObjectModelToJSON_ComplexNested(t *testing.T) {
	userModel := apimodel.NewObjectModelBuilder().
		LongField("id", apimodel.WithDemo("1")).
		StringField("name", apimodel.WithDemo("Alice")).
		Build()
	resultModel := apimodel.NewObjectModelBuilder().
		IntField("code").
		StringField("message").
		ObjectField("data", userModel).
		ArrayField("items", userModel).
		Build()

	result := objectModelToJSON(resultModel)

	var obj map[string]any
	if err := json.Unmarshal([]byte(result), &obj); err != nil {
		t.Fatalf("Result is not valid JSON: %v", err)
	}
	if obj["code"] != float64(0) {
		t.Errorf("code = %v, want 0", obj["code"])
	}
	data, ok := obj["data"].(map[string]any)
	if !ok {
		t.Fatal("data should be an object")
	}
	if data["name"] != "Alice" {
		t.Errorf("data.name = %v, want Alice", data["name"])
	}
	items, ok := obj["items"].([]any)
	if !ok {
		t.Fatal("items should be an array")
	}
	if len(items) != 1 {
		t.Fatalf("items length = %d, want 1", len(items))
	}
}

func TestObjectModelToJSONWithComments(t *testing.T) {
	m := apimodel.NewObjectModelBuilder().
		StringField("name", apimodel.WithComment("user name")).
		IntField("age", apimodel.WithComment("user age"), apimodel.WithDemo("25")).
		Build()

	result := objectModelToJSONWithComments(m)

	if !strings.Contains(result, `"name"`) {
		t.Errorf("Should contain name field: %s", result)
	}
	if !strings.Contains(result, "// user name") {
		t.Errorf("Should contain comment for name: %s", result)
	}
	if !strings.Contains(result, "// user age") {
		t.Errorf("Should contain comment for age: %s", result)
	}
	if !strings.Contains(result, "25") {
		t.Errorf("Should contain demo value 25: %s", result)
	}
}

func TestParseDemoValue_Boolean(t *testing.T) {
	tests := []struct {
		demo     string
		expected any
	}{
		{"true", true},
		{"false", false},
		{"yes", "yes"},
	}

	for _, tt := range tests {
		t.Run(tt.demo, func(t *testing.T) {
			result := parseDemoValue(tt.demo, apimodel.SingleModel(apimodel.JsonTypeBoolean))
			if result != tt.expected {
				t.Errorf("parseDemoValue(%q, boolean) = %v, want %v", tt.demo, result, tt.expected)
			}
		})
	}
}

func TestParseDemoValue_Integer(t *testing.T) {
	result := parseDemoValue("42", apimodel.SingleModel(apimodel.JsonTypeInt))
	if result != int64(42) {
		t.Errorf("parseDemoValue(\"42\", int) = %v, want 42", result)
	}
}

func TestParseDemoValue_Float(t *testing.T) {
	result := parseDemoValue("3.14", apimodel.SingleModel(apimodel.JsonTypeDouble))
	if result != 3.14 {
		t.Errorf("parseDemoValue(\"3.14\", double) = %v, want 3.14", result)
	}
}

func TestParseDemoValue_JSON(t *testing.T) {
	m := apimodel.NewObjectModelBuilder().
		StringField("name").
		Build()
	result := parseDemoValue(`{"name":"test"}`, m)
	obj, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("Expected map, got %T", result)
	}
	if obj["name"] != "test" {
		t.Errorf("name = %v, want test", obj["name"])
	}
}
