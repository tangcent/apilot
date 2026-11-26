package javacollector

import (
	"path/filepath"
	"testing"

	collector "github.com/tangcent/apilot/api-collector"
)

// byteAnnotatedFields is the Java spelling of the same form the Kotlin
// collector exports, without the use-site targets only Kotlin can write.
var byteAnnotatedFields = map[string]string{
	"plain":        "string",
	"letters":      "string",
	"using":        "array<int>",
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
	c := New().(*JavaCollector)
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
	}
	for _, path := range []string{"/byte-annotated/whole", "/byte-annotated/entity", "/byte-annotated/mono"} {
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
	flux := byPath["/byte-annotated/flux"]
	if flux == nil || flux.Response == nil {
		t.Fatal("flux endpoint missing")
	}
	if got := arrayFieldKind(flux.Response.Body); got != "array<string>" {
		t.Errorf("flux response = %s, want array<string>", got)
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
	// A default method of an implemented interface is not an endpoint of the
	// class. The interface itself is not a controller, so the method is absent.
	if ep := byPath["/byte-annotated/default"]; ep != nil {
		t.Errorf("default method was exported: %+v", ep)
	}
}
