package resolver

import (
	"strings"
	"testing"

	collector "github.com/tangcent/apilot/api-collector"
	"github.com/tangcent/apilot/api-collector-java/parser"
	model "github.com/tangcent/apilot/api-model"
)

// Jackson writes a byte[] as one Base64 string. `Byte[]` and `List<Byte>` are
// arrays of numbers, as are the other primitive arrays; before, a byte[] was
// exported as an array of integers, which is not what a client reads.
func TestResolve_ByteArrayIsAString(t *testing.T) {
	r := NewTypeResolver([]parser.Class{
		{Name: "Order", Package: "com.example", Fields: []parser.Field{{Name: "id", Type: "long"}}},
	})
	r.SetUnresolved(collector.NewUnresolvedSet())

	for typeName, want := range map[string]string{
		"byte[]":                  "string",
		"byte[][]":                "array<string>",
		"List<byte[]>":            "array<string>",
		"Map<String, byte[]>":     "map<string>",
		"ResponseEntity<byte[]>":  "string",
		"Mono<byte[]>":            "string",
		"Flux<byte[]>":            "array<string>",
		"Byte[]":                  "array<int>",
		"java.lang.Byte[]":        "array<int>",
		"List<Byte>":              "array<int>",
		"Byte[][]":                "array<array<int>>",
		"byte":                    "int",
		"Byte":                    "int",
		"int[]":                   "array<int>",
		"long[]":                  "array<long>",
		"char[]":                  "array<string>",
		"Order[]":                 "array<object:Order>",
		"Map<String, List<Byte>>": "map<array<int>>",
		"? extends byte[]":        "string",
		"List<? extends Byte>":    "array<int>",
		"List<? extends byte[]>":  "array<string>",
	} {
		if got := arrayShape(r.Resolve(typeName, nil)); got != want {
			t.Errorf("%s = %s, want %s", typeName, got, want)
		}
	}
	if unresolved := r.Unresolved(); len(unresolved) != 0 {
		t.Errorf("unresolved = %v, want none", unresolved)
	}
}

// A byte[] field is a string wherever it stands: in a class, in a generic
// class, and as a getter-only property.
func TestResolve_ByteArrayFieldsAreStrings(t *testing.T) {
	r := NewTypeResolver([]parser.Class{
		{Name: "Blob", Package: "com.example", Fields: []parser.Field{
			{Name: "data", Type: "byte[]"},
			{Name: "boxed", Type: "Byte[]"},
			{Name: "chunks", Type: "byte[][]"},
		}, Methods: []parser.Method{
			{Name: "getDigest", ReturnType: "byte[]"},
		}},
		{Name: "Box", Package: "com.example", TypeParameters: []string{"T"}, Fields: []parser.Field{
			{Name: "content", Type: "T"},
			{Name: "raw", Type: "byte[]"},
		}},
	})
	r.SetUnresolved(collector.NewUnresolvedSet())

	blob := r.Resolve("com.example.Blob", nil)
	for name, want := range map[string]string{
		"data":   "string",
		"boxed":  "array<int>",
		"chunks": "array<string>",
		"digest": "string",
	} {
		field := blob.Fields[name]
		if field == nil {
			t.Errorf("Blob has no field %s: %v", name, blob.FieldOrder)
			continue
		}
		if got := arrayShape(field.Model); got != want {
			t.Errorf("Blob.%s = %s, want %s", name, got, want)
		}
	}

	// A type argument bound to a byte[] is a string too.
	box := r.Resolve("com.example.Box<byte[]>", nil)
	for name, want := range map[string]string{"content": "string", "raw": "string"} {
		field := box.Fields[name]
		if field == nil {
			t.Errorf("Box has no field %s: %v", name, box.FieldOrder)
			continue
		}
		if got := arrayShape(field.Model); got != want {
			t.Errorf("Box<byte[]>.%s = %s, want %s", name, got, want)
		}
	}
}

// The binding view is Spring's data binder, not Jackson, and does not read a
// byte[] as Base64 (the request value "YWJj" binds as the four bytes of that
// text, "1,2,3" as three numbers), so a byte[] stays an array there.
func TestResolveForBinding_ByteArrayStaysAnArray(t *testing.T) {
	r := NewTypeResolver([]parser.Class{
		{Name: "Upload", Package: "com.example", Fields: []parser.Field{
			{Name: "data", Type: "byte[]"},
			{Name: "boxed", Type: "Byte[]"},
		}},
	})
	r.SetUnresolved(collector.NewUnresolvedSet())

	if got := arrayShape(r.ResolveForBinding(parser.Class{}, "byte[]", nil)); got != "array<int>" {
		t.Errorf("byte[] for binding = %s, want array<int>", got)
	}
	upload := r.ResolveForBinding(parser.Class{}, "com.example.Upload", nil)
	for name, want := range map[string]string{"data": "array<int>", "boxed": "array<int>"} {
		field := upload.Fields[name]
		if field == nil {
			t.Errorf("Upload has no field %s: %v", name, upload.FieldOrder)
			continue
		}
		if got := arrayShape(field.Model); got != want {
			t.Errorf("Upload.%s for binding = %s, want %s", name, got, want)
		}
	}

	// The same class is a string where Jackson writes it, in the same resolver.
	if got := arrayShape(r.Resolve("com.example.Upload", nil).Fields["data"].Model); got != "string" {
		t.Errorf("Upload.data as JSON = %s, want string", got)
	}
}

// byteArrayGenerics are classes that carry a type argument to a field in
// several ways: straight through, one generic class deeper, as an element, as a
// map value, and as the argument of a superclass.
func byteArrayGenerics() []parser.Class {
	return []parser.Class{
		{Name: "Wrapper", Package: "app.dto", TypeParameters: []string{"U"}, Fields: []parser.Field{
			{Name: "data", Type: "U"},
			{Name: "items", Type: "List<U>"},
			{Name: "grid", Type: "U[]"},
			{Name: "byName", Type: "Map<String, U>"},
			{Name: "nested", Type: "List<List<U>>"},
		}},
		{Name: "Outer", Package: "app.dto", TypeParameters: []string{"T"}, Fields: []parser.Field{
			{Name: "wrapped", Type: "Wrapper<T>"},
			{Name: "wrappedItems", Type: "List<Wrapper<T>>"},
			{Name: "wrappedGrid", Type: "Wrapper<T[]>"},
			{Name: "wrappedByName", Type: "Map<String, Wrapper<T>>"},
			{Name: "direct", Type: "T"},
		}},
		{Name: "Holder", Package: "app.dto", TypeParameters: []string{"H"}, Fields: []parser.Field{
			{Name: "value", Type: "H"},
			{Name: "values", Type: "List<H>"},
		}},
		{Name: "Deep", Package: "app.dto", TypeParameters: []string{"S"}, Fields: []parser.Field{
			{Name: "outer", Type: "Outer<S>"},
		}},
		{Name: "Blobs", Package: "app.dto", SuperClass: "HashMap", SuperClassTypeArgs: []string{"String", "byte[]"}},
		{Name: "BlobsOf", Package: "app.dto", TypeParameters: []string{"V"}, SuperClass: "HashMap", SuperClassTypeArgs: []string{"String", "V"}},
		{Name: "Holds", Package: "app.dto", TypeParameters: []string{"K"}, Fields: []parser.Field{
			{Name: "blobs", Type: "BlobsOf<K>"},
			{Name: "byName", Type: "Map<String, K>"},
		}},
		{Name: "Base", Package: "app.dto", TypeParameters: []string{"B"}, Fields: []parser.Field{
			{Name: "payload", Type: "B"},
			{Name: "wrapped", Type: "Wrapper<B>"},
		}},
		{Name: "Sub", Package: "app.dto", SuperClass: "Base", SuperClassTypeArgs: []string{"byte[]"}},
		{Name: "SubBoxed", Package: "app.dto", SuperClass: "Base", SuperClassTypeArgs: []string{"Byte[]"}},
		{Name: "SubList", Package: "app.dto", SuperClass: "Base", SuperClassTypeArgs: []string{"List<byte[]>"}},
		{Name: "SubOpen", Package: "app.dto", TypeParameters: []string{"X"}, SuperClass: "Base", SuperClassTypeArgs: []string{"X"}},
	}
}

// A type argument bound to a byte[] is the Base64 string wherever the field
// that takes it stands. `Outer<T>` hands T to `Wrapper<T>`, which hands it on
// as U to `U data`: the argument of the inner class is spelled by T, and T is
// bound by the outer one. Before, a scalar result kept the raw `T`, so the
// outer binding was lost and `data` was exported as the unresolved type T.
func TestResolve_ByteArrayThroughTypeVariables(t *testing.T) {
	r := NewTypeResolver(byteArrayGenerics())
	r.SetUnresolved(collector.NewUnresolvedSet())

	type check struct{ typeName, path, want string }
	for _, c := range []check{
		// One level: the argument is the type of the field.
		{"app.dto.Wrapper<byte[]>", "data", "string"},
		{"app.dto.Wrapper<byte[]>", "items", "array<string>"},
		{"app.dto.Wrapper<byte[]>", "grid", "array<string>"},
		{"app.dto.Wrapper<byte[]>", "byName", "map<string>"},
		{"app.dto.Wrapper<byte[]>", "nested", "array<array<string>>"},
		// Two levels: Outer<T> -> Wrapper<T> -> U.
		{"app.dto.Outer<byte[]>", "wrapped.data", "string"},
		{"app.dto.Outer<byte[]>", "wrapped.items", "array<string>"},
		{"app.dto.Outer<byte[]>", "wrapped.grid", "array<string>"},
		{"app.dto.Outer<byte[]>", "wrapped.byName", "map<string>"},
		{"app.dto.Outer<byte[]>", "wrapped.nested", "array<array<string>>"},
		{"app.dto.Outer<byte[]>", "wrappedItems", "array<object:Wrapper>"},
		{"app.dto.Outer<byte[]>", "wrappedGrid.data", "array<string>"},
		{"app.dto.Outer<byte[]>", "wrappedByName", "map<object:Wrapper>"},
		{"app.dto.Outer<byte[]>", "direct", "string"},
		// Three levels: Deep<S> -> Outer<S> -> Wrapper<S> -> U.
		{"app.dto.Deep<byte[]>", "outer.wrapped.data", "string"},
		// The same shapes with an element type that is not a byte[].
		{"app.dto.Outer<Byte[]>", "wrapped.data", "array<int>"},
		{"app.dto.Outer<byte[][]>", "wrapped.data", "array<string>"},
		{"app.dto.Outer<List<byte[]>>", "wrapped.data", "array<string>"},
		{"app.dto.Outer<Map<String, byte[]>>", "wrapped.data", "map<string>"},
		// A scalar bound to the variable is kept the same way.
		{"app.dto.Outer<Integer>", "wrapped.data", "int"},
		{"app.dto.Outer<String>", "wrapped.data", "string"},
		{"app.dto.Outer<Long>", "wrapped.items", "array<long>"},
		{"app.dto.Deep<Integer>", "outer.wrapped.data", "int"},
		// An argument that is itself generic keeps what it is bound to.
		{"app.dto.Outer<app.dto.Holder<byte[]>>", "wrapped.data.value", "string"},
		{"app.dto.Outer<app.dto.Holder<byte[]>>", "wrapped.items", "array<object:Holder>"},
		{"app.dto.Deep<app.dto.Holder<byte[]>>", "outer.wrapped.data.values", "array<string>"},
		// A wildcard argument is read by its bound.
		{"app.dto.Outer<? extends byte[]>", "wrapped.data", "string"},
		{"app.dto.Outer<? extends byte[]>", "wrapped.items", "array<string>"},
		// A map is its value type, however the class that is one is declared.
		{"app.dto.Blobs", "", "map<string>"},
		{"app.dto.BlobsOf<byte[]>", "", "map<string>"},
		{"app.dto.Holds<byte[]>", "blobs", "map<string>"},
		{"app.dto.Holds<byte[]>", "byName", "map<string>"},
		{"app.dto.Holds<Byte[]>", "blobs", "map<array<int>>"},
		// A superclass argument is bound the same way.
		{"app.dto.Sub", "payload", "string"},
		{"app.dto.Sub", "wrapped.data", "string"},
		{"app.dto.Sub", "wrapped.items", "array<string>"},
		{"app.dto.SubBoxed", "payload", "array<int>"},
		{"app.dto.SubBoxed", "wrapped.data", "array<int>"},
		{"app.dto.SubList", "payload", "array<string>"},
		{"app.dto.SubList", "wrapped.data", "array<string>"},
		{"app.dto.SubOpen<byte[]>", "payload", "string"},
		{"app.dto.SubOpen<byte[]>", "wrapped.data", "string"},
		{"app.dto.SubOpen<Byte[]>", "wrapped.data", "array<int>"},
	} {
		if got := arrayShape(fieldAt(r.Resolve(c.typeName, nil), c.path)); got != c.want {
			t.Errorf("%s %s = %s, want %s", c.typeName, c.path, got, c.want)
		}
	}
	if unresolved := r.Unresolved(); len(unresolved) != 0 {
		t.Errorf("unresolved = %v, want none", unresolved)
	}
}

// The binding view reads the same bindings: a byte[] stays an array there, and
// it is the array that reaches a field two generic classes deep.
func TestResolveForBinding_ByteArrayThroughTypeVariables(t *testing.T) {
	r := NewTypeResolver(byteArrayGenerics())
	r.SetUnresolved(collector.NewUnresolvedSet())

	for _, c := range []struct{ typeName, path, want string }{
		{"app.dto.Outer<byte[]>", "wrapped.data", "array<int>"},
		{"app.dto.Outer<byte[]>", "wrapped.items", "array<array<int>>"},
		{"app.dto.Outer<byte[]>", "direct", "array<int>"},
		{"app.dto.Outer<Integer>", "wrapped.data", "int"},
		{"app.dto.Sub", "wrapped.data", "array<int>"},
		{"app.dto.Holds<byte[]>", "blobs", "map<array<int>>"},
		{"app.dto.Blobs", "", "map<array<int>>"},
	} {
		got := arrayShape(fieldAt(r.ResolveForBinding(parser.Class{}, c.typeName, nil), c.path))
		if got != c.want {
			t.Errorf("%s %s for binding = %s, want %s", c.typeName, c.path, got, c.want)
		}
	}
	// The same resolver answers the JSON view with the string afterwards.
	if got := arrayShape(fieldAt(r.Resolve("app.dto.Outer<byte[]>", nil), "wrapped.data")); got != "string" {
		t.Errorf("Outer<byte[]> wrapped.data as JSON = %s, want string", got)
	}
	if unresolved := r.Unresolved(); len(unresolved) != 0 {
		t.Errorf("unresolved = %v, want none", unresolved)
	}
}

// fieldAt follows a dotted path of field names from an object model; an empty
// path is the model itself.
func fieldAt(m *model.ObjectModel, path string) *model.ObjectModel {
	if path == "" {
		return m
	}
	for _, name := range strings.Split(path, ".") {
		if m == nil || m.Fields[name] == nil {
			return nil
		}
		m = m.Fields[name].Model
	}
	return m
}
