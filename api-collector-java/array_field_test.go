package javacollector

import (
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	collector "github.com/tangcent/apilot/api-collector"
	model "github.com/tangcent/apilot/api-model"
)

// arrayFieldExpectation is the export of testdata/array-field: the fields of
// the request body and of the response, each as its kind. An array is an
// array of its element type, whichever way it is spelled, and a name qualified
// with its package is the type it names.
//
// The Kotlin collector asserts the same table for the same declarations
// written in Kotlin (api-collector-kotlin/array_field_test.go), except for
// `legacy`, which only Java can spell: keep the two in sync.
var arrayFieldExpectation = map[string]string{
	"names":   "array<string>",
	"ids":     "array<int>",
	"longs":   "array<long>",
	"items":   "array<object{name}>",
	"groups":  "array<array<string>>",
	"grid":    "array<array<int>>",
	"legacy":  "array<string>",
	"day":     "string",
	"amounts": "array<double>",
	"codes":   "array<string>",
}

func TestArrayFieldsAreExportedAsArrays(t *testing.T) {
	abs, err := filepath.Abs("testdata/array-field")
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

	save := byPath["/array-fields/save"]
	if save == nil || save.RequestBody == nil || save.RequestBody.Body == nil || save.Response == nil {
		t.Fatalf("endpoints = %+v, want the save endpoint with a request body and a response", endpoints)
	}
	for what, body := range map[string]*model.ObjectModel{
		"request body": save.RequestBody.Body,
		"response":     save.Response.Body,
	} {
		if got := arrayFieldKinds(body); !reflect.DeepEqual(got, arrayFieldExpectation) {
			t.Errorf("%s:\n got  %v\n want %v", what, got, arrayFieldExpectation)
		}
		// A constraint on an array field is read like one on any other.
		for name, field := range body.Fields {
			if want := name == "codes"; field.Required != want {
				t.Errorf("%s %s: required = %v, want %v", what, name, field.Required, want)
			}
		}
	}

	// An array is also what a handler takes and returns as a whole.
	names := byPath["/array-fields/names"]
	if names == nil || names.RequestBody == nil || names.Response == nil {
		t.Fatalf("names endpoint = %+v, want a request body and a response", names)
	}
	if got := arrayFieldKind(names.RequestBody.Body); got != "array<string>" {
		t.Errorf("names request body = %s, want array<string>", got)
	}
	if got := arrayFieldKind(names.Response.Body); got != "array<string>" {
		t.Errorf("names response = %s, want array<string>", got)
	}
	items := byPath["/array-fields/items"]
	if items == nil || items.Response == nil {
		t.Fatalf("items endpoint = %+v, want a response", items)
	}
	if got := arrayFieldKind(items.Response.Body); got != "array<object{name}>" {
		t.Errorf("items response = %s, want array<object{name}>", got)
	}

	if unresolved := c.Unresolved(); len(unresolved) != 0 {
		t.Errorf("unresolved = %v, want none", unresolved)
	}
}

// arrayFieldKinds spells the fields of an object model as their kinds.
func arrayFieldKinds(m *model.ObjectModel) map[string]string {
	out := map[string]string{}
	for name, field := range m.Fields {
		out[name] = arrayFieldKind(field.Model)
	}
	return out
}

func arrayFieldKind(m *model.ObjectModel) string {
	switch {
	case m == nil:
		return "nil"
	case m.IsArray():
		return "array<" + arrayFieldKind(m.Items) + ">"
	case m.IsMap():
		return "map<" + arrayFieldKind(m.KeyModel) + "," + arrayFieldKind(m.ValueModel) + ">"
	case m.IsObject():
		names := make([]string, 0, len(m.Fields))
		for name := range m.Fields {
			names = append(names, name)
		}
		sort.Strings(names)
		return "object{" + strings.Join(names, ",") + "}"
	}
	return m.TypeName
}
