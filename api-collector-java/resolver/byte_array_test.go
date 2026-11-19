package resolver

import (
	"testing"

	collector "github.com/tangcent/apilot/api-collector"
	"github.com/tangcent/apilot/api-collector-java/parser"
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
