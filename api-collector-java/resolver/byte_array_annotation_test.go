package resolver

import (
	"testing"

	"github.com/tangcent/apilot/api-collector-java/parser"
)

// Jackson writes a byte[] as Base64 and a char[] as one string. An annotation
// that replaces that serializer does not, and the field stays the array it was
// on main. @JsonFormat does not replace it, nor does contentUsing on the array
// itself, nor a serializer named None.
func TestResolve_AReplacedByteArraySerializerStaysAnArray(t *testing.T) {
	ser := func(params map[string]string) []parser.Annotation {
		return []parser.Annotation{{Name: "JsonSerialize", Params: params}}
	}
	format := func(shape string) []parser.Annotation {
		return []parser.Annotation{{Name: "JsonFormat", Params: map[string]string{"shape": "JsonFormat.Shape." + shape}}}
	}
	form := parser.Class{
		Name: "AnnotatedBytes", Package: "p",
		Annotations: format("ARRAY"),
		Fields: []parser.Field{
			{Name: "plain", Type: "byte[]"},
			{Name: "using", Type: "byte[]", Annotations: ser(map[string]string{"using": "Nums.class"})},
			{Name: "converter", Type: "byte[]", Annotations: ser(map[string]string{"converter": "Conv.class"})},
			{Name: "none", Type: "byte[]", Annotations: ser(map[string]string{"using": "JsonSerializer.None.class"})},
			{Name: "kotlinNone", Type: "byte[]", Annotations: ser(map[string]string{"using": "JsonSerializer.None::class"})},
			{Name: "contentSelf", Type: "byte[]", Annotations: ser(map[string]string{"contentUsing": "Nums.class"})},
			{Name: "chunks", Type: "byte[][]", Annotations: ser(map[string]string{"contentUsing": "Nums.class"})},
			{Name: "blobs", Type: "List<byte[]>", Annotations: ser(map[string]string{"contentUsing": "Nums.class"})},
			{Name: "byName", Type: "Map<String, byte[]>", Annotations: ser(map[string]string{"contentUsing": "Nums.class"})},
			{Name: "deep", Type: "List<List<byte[]>>", Annotations: ser(map[string]string{"contentUsing": "Nums.class"})},
			{Name: "contentConv", Type: "List<byte[]>", Annotations: ser(map[string]string{"contentConverter": "Conv.class"})},
			{Name: "usingList", Type: "List<byte[]>", Annotations: ser(map[string]string{"using": "Nums.class"})},
			{Name: "raw", Type: "byte[]", Annotations: []parser.Annotation{{Name: "JsonRawValue"}}},
			{Name: "rawOff", Type: "byte[]", Annotations: []parser.Annotation{{Name: "JsonRawValue", Params: map[string]string{"value": "false"}}}},
			{Name: "fmtArray", Type: "byte[]", Annotations: format("ARRAY")},
			{Name: "fmtNumber", Type: "byte[]", Annotations: format("NUMBER")},
			{Name: "fmtString", Type: "byte[]", Annotations: format("STRING")},
			{Name: "fmtObject", Type: "byte[]", Annotations: format("OBJECT")},
			{Name: "letters", Type: "char[]"},
			{Name: "lettersUsing", Type: "char[]", Annotations: ser(map[string]string{"using": "Nums.class"})},
			{Name: "letterChunks", Type: "char[][]", Annotations: ser(map[string]string{"contentUsing": "Nums.class"})},
		},
	}
	r := NewTypeResolver([]parser.Class{form})
	got := map[string]string{}
	for name, field := range r.Resolve("p.AnnotatedBytes", nil).Fields {
		got[name] = arrayShape(field.Model)
	}
	want := map[string]string{
		"plain":        "string",
		"using":        "array<int>",
		"converter":    "array<int>",
		"none":         "string",
		"kotlinNone":   "string",
		"contentSelf":  "string",
		"chunks":       "array<array<int>>",
		"blobs":        "array<array<int>>",
		"byName":       "map<array<int>>",
		"deep":         "array<array<array<int>>>",
		"contentConv":  "array<array<int>>",
		"usingList":    "array<array<int>>",
		"raw":          "array<int>",
		"rawOff":       "string",
		"fmtArray":     "string",
		"fmtNumber":    "string",
		"fmtString":    "string",
		"fmtObject":    "string",
		"letters":      "string",
		"lettersUsing": "array<string>",
		"letterChunks": "array<array<string>>",
	}
	for name, wantShape := range want {
		if got[name] != wantShape {
			t.Errorf("%s = %s, want %s", name, got[name], wantShape)
		}
	}

	// The binding view never writes these as strings, annotation or not.
	bound := r.ResolveForBinding(form, "p.AnnotatedBytes", nil)
	if shape := arrayShape(bound.Fields["using"].Model); shape != "array<int>" {
		t.Errorf("binding using = %s, want array<int>", shape)
	}
	if shape := arrayShape(bound.Fields["plain"].Model); shape != "array<int>" {
		t.Errorf("binding plain = %s, want array<int>", shape)
	}
}
