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

// A type argument that is an array of a type parameter keeps the binding of that
// parameter: `Wrapper<T[]>` written in `Box<T>` is `Wrapper<Order[]>` once the
// box is resolved as `Box<Order>`, so the wrapper's `U data` is an array of
// Order. Before, the argument stayed the spelling `T[]` and was resolved inside
// the wrapper, where T is not bound, and the element was reported as an
// unresolved `T`.
func TestResolve_ArrayOfATypeParameterAsATypeArgumentFollowsItsBinding(t *testing.T) {
	r := NewTypeResolver([]parser.Class{
		{Name: "Order", Package: "com.example", Fields: []parser.Field{{Name: "id", Type: "long"}}},
		{Name: "Wrapper", Package: "com.example", TypeParameters: []string{"U"}, Fields: []parser.Field{
			{Name: "data", Type: "U"},
		}},
		{Name: "Box", Package: "com.example", TypeParameters: []string{"T"}, Fields: []parser.Field{
			{Name: "payload", Type: "Wrapper<T[]>"},
			{Name: "grid", Type: "Wrapper<T[][]>"},
			{Name: "lists", Type: "Wrapper<List<T[]>>"},
			{Name: "byName", Type: "Wrapper<Map<String, T[]>>"},
			{Name: "direct", Type: "Map<String, T[]>"},
		}},
	})
	r.SetUnresolved(collector.NewUnresolvedSet())
	box := r.Resolve("com.example.Box<com.example.Order>", nil)

	// The wrapper carries the argument in its `data` field.
	for field, want := range map[string]string{
		"payload": "array<object:Order>",
		"grid":    "array<array<object:Order>>",
		"lists":   "array<array<object:Order>>",
		"byName":  "map<array<object:Order>>",
	} {
		fm := box.Fields[field]
		if fm == nil || fm.Model.Fields["data"] == nil {
			t.Errorf("%s = %+v, want a wrapper with a data field", field, fm)
			continue
		}
		if got := arrayShape(fm.Model.Fields["data"].Model); got != want {
			t.Errorf("%s.data = %s, want %s", field, got, want)
		}
	}
	if got := arrayShape(box.Fields["direct"].Model); got != "map<array<object:Order>>" {
		t.Errorf("direct = %s, want map<array<object:Order>>", got)
	}
	// The element is Order, with its fields, not a name that stands for it.
	element := box.Fields["payload"].Model.Fields["data"].Model.Items
	if element == nil || element.Fields["id"] == nil {
		t.Errorf("payload.data element = %+v, want Order with its fields", element)
	}
	if unresolved := r.Unresolved(); len(unresolved) != 0 {
		t.Errorf("unresolved = %v, want none", unresolved)
	}
}

// The same holds for a superclass: `Sub<T> extends Base<T[]>` binds Base's
// parameter to an array of whatever T is bound to in the subclass.
func TestResolve_ArrayOfATypeParameterAsASuperclassArgumentFollowsItsBinding(t *testing.T) {
	r := NewTypeResolver([]parser.Class{
		{Name: "Order", Package: "com.example", Fields: []parser.Field{{Name: "id", Type: "long"}}},
		{Name: "Base", Package: "com.example", TypeParameters: []string{"E"}, Fields: []parser.Field{
			{Name: "data", Type: "E"},
		}},
		{Name: "Sub", Package: "com.example", TypeParameters: []string{"T"}, SuperClass: "Base", SuperClassTypeArgs: []string{"T[]"}},
	})
	r.SetUnresolved(collector.NewUnresolvedSet())
	sub := r.Resolve("com.example.Sub<com.example.Order>", nil)
	if sub.Fields["data"] == nil {
		t.Fatalf("Sub = %+v, want the data field of Base", sub)
	}
	if got := arrayShape(sub.Fields["data"].Model); got != "array<object:Order>" {
		t.Errorf("data = %s, want array<object:Order>", got)
	}
	if unresolved := r.Unresolved(); len(unresolved) != 0 {
		t.Errorf("unresolved = %v, want none", unresolved)
	}
}
