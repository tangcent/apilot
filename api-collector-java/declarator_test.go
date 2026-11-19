package javacollector

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	collector "github.com/tangcent/apilot/api-collector"
	model "github.com/tangcent/apilot/api-model"
)

// declaratorFieldExpectation is the export of testdata/declarator: the fields
// of the request body and of the response, each as its kind, in the order the
// source declares them. Several fields share one declaration; each keeps its
// own name, brackets and initializer, and the annotation and the doc comment
// of the declaration apply to every one of them.
var declaratorFieldExpectation = map[string]string{
	"firstName": "string",
	"lastName":  "string",
	"width":     "int",
	"height":    "int",
	"depth":     "array<int>",
	"note":      "string",
	"tags":      "array<string>",
	"remark":    "string",
	"aliases":   "array<string>",
	"nicknames": "array<string>",
}

var declaratorFieldOrder = []string{
	"firstName", "lastName", "width", "height", "depth", "note", "tags", "remark", "aliases", "nicknames",
}

// A varargs parameter is exported as the array it is: `String... ids` as
// `String[] ids`, and `Item... items` as a body of items. The Kotlin collector
// asserts the same for `vararg` (api-collector-kotlin/vararg_test.go).
var varargsBodyExpectation = map[string]string{
	"/declarators/items":       "array<object{name}>",
	"/declarators/items-array": "array<object{name}>",
}

func collectDeclarators(t *testing.T) (*JavaCollector, map[string]*collector.ApiEndpoint) {
	t.Helper()
	abs, err := filepath.Abs("testdata/declarator")
	if err != nil {
		t.Fatal(err)
	}
	c := New().(*JavaCollector)
	endpoints, err := c.Collect(collector.CollectContext{
		SourceDir: abs,
		NoDeps:    true,
		Config:    map[string]string{collector.ConfigRequiredFields: collector.RequiredFieldsConstraints},
	})
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}
	byPath := map[string]*collector.ApiEndpoint{}
	for i := range endpoints {
		byPath[endpoints[i].Path] = &endpoints[i]
	}
	return c, byPath
}

func TestSeveralFieldsInOneDeclarationAreAllExported(t *testing.T) {
	c, byPath := collectDeclarators(t)

	save := byPath["/declarators/save"]
	if save == nil || save.RequestBody == nil || save.RequestBody.Body == nil || save.Response == nil {
		t.Fatalf("save endpoint = %+v, want a request body and a response", save)
	}
	for what, body := range map[string]*model.ObjectModel{
		"request body": save.RequestBody.Body,
		"response":     save.Response.Body,
	} {
		if got := arrayFieldKinds(body); !reflect.DeepEqual(got, declaratorFieldExpectation) {
			t.Errorf("%s:\n got  %v\n want %v", what, got, declaratorFieldExpectation)
		}
		if !reflect.DeepEqual(body.FieldOrder, declaratorFieldOrder) {
			t.Errorf("%s field order:\n got  %v\n want %v", what, body.FieldOrder, declaratorFieldOrder)
		}
		for name, field := range body.Fields {
			wantRequired := name == "firstName" || name == "lastName"
			if field.Required != wantRequired {
				t.Errorf("%s %s: required = %v, want %v", what, name, field.Required, wantRequired)
			}
			wantComment := ""
			if wantRequired {
				wantComment = "The name of a person."
			}
			if field.Comment != wantComment {
				t.Errorf("%s %s: comment = %q, want %q", what, name, field.Comment, wantComment)
			}
		}
	}
	if unresolved := c.Unresolved(); len(unresolved) != 0 {
		t.Errorf("unresolved = %v, want none", unresolved)
	}
}

func TestVarargsParametersAreExportedAsArrays(t *testing.T) {
	c, byPath := collectDeclarators(t)

	// The names on the wire are two constants declared in one statement; the
	// varargs parameter that follows is optional by its annotation.
	search := byPath["/declarators/search"]
	if search == nil {
		t.Fatalf("search endpoint missing: %v", byPath)
	}
	type param struct{ Name, In, TypeName string }
	var got []param
	for _, p := range search.Parameters {
		got = append(got, param{p.Name, p.In, p.TypeName})
	}
	want := []param{
		{"page_no", "query", "int"},
		{"page_size", "query", "int"},
		{"ids", "query", "String[]"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("search parameters:\n got  %v\n want %v", got, want)
	}
	if len(search.Parameters) != len(want) {
		t.Fatalf("search has %d parameters, want %d", len(search.Parameters), len(want))
	}
	if ids := search.Parameters[2]; ids.Required {
		t.Errorf("ids: required = true, want false")
	}

	// A varargs parameter is exported as the array parameter declared the same way.
	twin := byPath["/declarators/search-array"]
	if twin == nil || len(twin.Parameters) != 1 {
		t.Fatalf("search-array endpoint = %+v, want one parameter", twin)
	}
	if !reflect.DeepEqual(twin.Parameters[0], search.Parameters[2]) {
		t.Errorf("varargs parameter:\n got  %+v\n want %+v", search.Parameters[2], twin.Parameters[0])
	}

	implicit := byPath["/declarators/implicit"]
	if implicit == nil || len(implicit.Parameters) != 1 {
		t.Fatalf("implicit endpoint = %+v, want one parameter", implicit)
	}
	if p := implicit.Parameters[0]; p.Name != "tags" || p.In != "query" || p.TypeName != "String[]" {
		t.Errorf("implicit parameter = %+v, want tags in query as String[]", p)
	}

	for path, wantKind := range varargsBodyExpectation {
		endpoint := byPath[path]
		if endpoint == nil || endpoint.RequestBody == nil || endpoint.RequestBody.Body == nil {
			t.Fatalf("%s: endpoint = %+v, want a request body", path, endpoint)
		}
		if got := arrayFieldKind(endpoint.RequestBody.Body); got != wantKind {
			t.Errorf("%s request body = %s, want %s", path, got, wantKind)
		}
	}

	if unresolved := c.Unresolved(); len(unresolved) != 0 {
		t.Errorf("unresolved = %v, want none", unresolved)
	}
	for _, w := range c.Warnings() {
		if strings.Contains(w, "could not be resolved") {
			t.Errorf("warning: %s", w)
		}
	}
}
