package resolver

import (
	"reflect"
	"testing"

	collector "github.com/tangcent/apilot/api-collector"
	"github.com/tangcent/apilot/api-collector-java/parser"
	model "github.com/tangcent/apilot/api-model"
)

// arrayShape spells a model as the shape the export carries.
func arrayShape(m *model.ObjectModel) string {
	switch {
	case m == nil:
		return "nil"
	case m.IsArray():
		return "array<" + arrayShape(m.Items) + ">"
	case m.IsMap():
		return "map<" + arrayShape(m.ValueModel) + ">"
	case m.IsObject():
		return "object:" + m.TypeName
	}
	return m.TypeName
}

// An array is a JSON array of its element type, however the element is
// spelled; before, the whole spelling was taken for the name of a type that no
// declaration has, and `String[]` was exported as a single opaque value.
func TestResolve_ArraysAreArraysOfTheirElementType(t *testing.T) {
	r := NewTypeResolver([]parser.Class{
		{Name: "Order", Package: "com.example", Fields: []parser.Field{{Name: "id", Type: "long"}}},
	})
	r.SetUnresolved(collector.NewUnresolvedSet())

	for typeName, want := range map[string]string{
		"String[]":               "array<string>",
		"java.lang.String[]":     "array<string>",
		"int[]":                  "array<int>",
		"Long[]":                 "array<long>",
		"java.time.LocalDate[]":  "array<string>",
		"int[][]":                "array<array<int>>",
		"Order[]":                "array<object:Order>",
		"List<String>[]":         "array<array<string>>",
		"List<Order[]>":          "array<array<object:Order>>",
		"Map<String, Order[]>":   "map<array<object:Order>>",
		"Map<String, Order>[]":   "array<map<object:Order>>",
		"? extends Order[]":      "array<object:Order>",
		"Mono<String[]>":         "array<string>",
		"ResponseEntity<Long[]>": "array<long>",
	} {
		if got := arrayShape(r.Resolve(typeName, nil)); got != want {
			t.Errorf("%s = %s, want %s", typeName, got, want)
		}
	}
	if unresolved := r.Unresolved(); len(unresolved) != 0 {
		t.Errorf("unresolved = %v, want none", unresolved)
	}
}

// The element of an array whose declaration is missing is reported by its own
// name, not by the spelling of the array.
func TestResolve_AnUnknownArrayElementIsReportedByItsName(t *testing.T) {
	r := NewTypeResolver(nil)
	r.SetUnresolved(collector.NewUnresolvedSet())

	if got := arrayShape(r.Resolve("Missing[]", nil)); got != "array<Missing>" {
		t.Errorf("Missing[] = %s, want array<Missing>", got)
	}
	if got, want := r.Unresolved(), map[string]int{"Missing": 1}; !reflect.DeepEqual(got, want) {
		t.Errorf("unresolved = %v, want %v", got, want)
	}
}

// A type parameter in front of the brackets is bound like one anywhere else.
func TestResolve_ArrayOfATypeParameterFollowsItsBinding(t *testing.T) {
	r := NewTypeResolver([]parser.Class{
		{Name: "Order", Package: "com.example", Fields: []parser.Field{{Name: "id", Type: "long"}}},
		{Name: "Batch", Package: "com.example", TypeParameters: []string{"T"}, Fields: []parser.Field{
			{Name: "items", Type: "T[]"},
			{Name: "tags", Type: "String[]"},
		}},
	})
	batch := r.Resolve("com.example.Batch<com.example.Order>", nil)
	if got := arrayShape(batch.Fields["items"].Model); got != "array<object:Order>" {
		t.Errorf("items = %s, want array<object:Order>", got)
	}
	if got := arrayShape(batch.Fields["tags"].Model); got != "array<string>" {
		t.Errorf("tags = %s, want array<string>", got)
	}
}
