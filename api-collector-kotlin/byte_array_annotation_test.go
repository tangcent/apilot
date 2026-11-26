package kotlincollector

import (
	"path/filepath"
	"testing"

	collector "github.com/tangcent/apilot/api-collector"
)

// byteAnnotatedFields is the JSON export of ByteAnnotatedForm. A serializer or
// a raw value replaces Jackson's Base64 string, so the field stays the array
// main exports. @JsonFormat does not, and an annotation on the constructor
// parameter does not either: plain Jackson still writes Base64 for @param:.
var byteAnnotatedFields = map[string]string{
	"plain":        "string",
	"letters":      "string",
	"using":        "array<int>",
	"getter":       "array<int>",
	"field":        "array<int>",
	"param":        "string",
	"converter":    "array<int>",
	"none":         "string",
	"contentSelf":  "string",
	"chunks":       "array<array<int>>",
	"blobs":        "array<array<int>>",
	"byName":       "map<string,array<int>>",
	"raw":          "array<int>",
	"fmtArray":     "string",
	"fmtNumber":    "string",
	"fmtString":    "string",
	"fmtObject":    "string",
	"lettersUsing": "array<string>",
	"nullable":     "string",
	"optional":     "string",
}

func TestAnnotatedByteArraysKeepTheArrayWhenTheSerializerIsReplaced(t *testing.T) {
	abs, err := filepath.Abs("testdata/byte-annotated")
	if err != nil {
		t.Fatal(err)
	}
	c := New().(*KotlinCollector)
	endpoints, err := c.Collect(collector.CollectContext{SourceDir: abs, NoDeps: true})
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}
	byPath := map[string]*collector.ApiEndpoint{}
	for i := range endpoints {
		byPath[endpoints[i].Path] = &endpoints[i]
	}
	form := byPath["/byte-annotated/form"]
	if form == nil || form.RequestBody == nil || form.Response == nil {
		t.Fatalf("form = %+v, want a request body and a response", form)
	}
	for name, want := range byteAnnotatedFields {
		field := form.RequestBody.Body.Fields[name]
		if field == nil {
			t.Errorf("form request has no %s", name)
			continue
		}
		if got := arrayFieldKind(field.Model); got != want {
			t.Errorf("form %s = %s, want %s", name, got, want)
		}
		resp := form.Response.Body.Fields[name]
		if resp == nil {
			t.Errorf("form response has no %s", name)
			continue
		}
		if got := arrayFieldKind(resp.Model); got != want {
			t.Errorf("form response %s = %s, want %s", name, got, want)
		}
	}
	classFmt := byPath["/byte-annotated/class-format"]
	if classFmt == nil || classFmt.RequestBody == nil || classFmt.RequestBody.Body.Fields["data"] == nil {
		t.Fatalf("class-format = %+v, want a data field", classFmt)
	}
	if got := arrayFieldKind(classFmt.RequestBody.Body.Fields["data"].Model); got != "string" {
		t.Errorf("class-level @JsonFormat field = %s, want string", got)
	}
	for _, path := range []string{"/byte-annotated/whole", "/byte-annotated/suspend", "/byte-annotated/object/whole"} {
		ep := byPath[path]
		if ep == nil || ep.RequestBody == nil || ep.Response == nil {
			t.Errorf("%s = %+v, want a request body and a response", path, ep)
			continue
		}
		if got := arrayFieldKind(ep.RequestBody.Body); got != "string" {
			t.Errorf("%s request = %s, want string", path, got)
		}
		if got := arrayFieldKind(ep.Response.Body); got != "string" {
			t.Errorf("%s response = %s, want string", path, got)
		}
	}
	declared := byPath["/byte-annotated/sub/declared"]
	if declared == nil || declared.RequestBody == nil {
		t.Fatalf("inherited declared = %+v, want a request body", declared)
	}
	if got := arrayFieldKind(declared.RequestBody.Body); got != "string" {
		t.Errorf("inherited vararg Byte = %s, want string", got)
	}
	typed := byPath["/byte-annotated/sub/typed"]
	if typed == nil || typed.RequestBody == nil {
		t.Fatalf("inherited typed = %+v, want a request body", typed)
	}
	if got := arrayFieldKind(typed.RequestBody.Body); got == "string" {
		t.Errorf("inherited vararg of a type parameter bound to Byte = string, want the boxed array")
	}
	bound := byPath["/byte-annotated/bound"]
	if bound == nil {
		t.Fatal("bound endpoint missing")
	}
	gotBound := map[string]string{}
	for _, p := range bound.Parameters {
		gotBound[p.Name] = p.TypeName
	}
	for _, name := range []string{"data", "letters"} {
		if gotBound[name] != "array" {
			t.Errorf("bound %s = %s, want array", name, gotBound[name])
		}
	}
}
