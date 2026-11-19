package kotlincollector

import (
	"path/filepath"
	"reflect"
	"testing"

	collector "github.com/tangcent/apilot/api-collector"
	model "github.com/tangcent/apilot/api-model"
)

// jdkScalarExpectation is the export of testdata/jdk-scalar: the fields of the
// request body and of the response, each as its kind. A short is an integer,
// whichever way it is spelled, and a boxed type or a Number spelled with its
// package is the type it names.
//
// The Java collector asserts the same table for the same declarations written
// in Java (api-collector-java/jdk_scalar_test.go); the unsigned shorts are
// added here because only Kotlin has them: keep the two in sync.
var jdkScalarExpectation = map[string]string{
	"primitiveShort":   "int",
	"boxedShort":       "int",
	"qualifiedShort":   "int",
	"shorts":           "array<int>",
	"boxedShorts":      "array<int>",
	"qualifiedShorts":  "array<int>",
	"shortList":        "array<int>",
	"shortsByName":     "map<string,int>",
	"unsignedShort":    "int",
	"unsignedShorts":   "array<int>",
	"qualifiedInteger": "int",
	"qualifiedLong":    "long",
	"qualifiedFloat":   "float",
	"qualifiedDouble":  "double",
	"qualifiedBoolean": "boolean",
	"number":           "double",
	"qualifiedNumber":  "double",
}

func TestJdkScalarFieldsAreExportedAsScalars(t *testing.T) {
	abs, err := filepath.Abs("testdata/jdk-scalar")
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

	save := byPath["/jdk-scalars/save"]
	if save == nil || save.RequestBody == nil || save.RequestBody.Body == nil || save.Response == nil {
		t.Fatalf("endpoints = %+v, want the save endpoint with a request body and a response", endpoints)
	}
	for what, body := range map[string]*model.ObjectModel{
		"request body": save.RequestBody.Body,
		"response":     save.Response.Body,
	} {
		if got := arrayFieldKinds(body); !reflect.DeepEqual(got, jdkScalarExpectation) {
			t.Errorf("%s:\n got  %v\n want %v", what, got, jdkScalarExpectation)
		}
	}

	// A short is also what a handler takes and returns as a whole.
	shorts := byPath["/jdk-scalars/shorts"]
	if shorts == nil || shorts.RequestBody == nil || shorts.Response == nil {
		t.Fatalf("shorts endpoint = %+v, want a request body and a response", shorts)
	}
	if got := arrayFieldKind(shorts.RequestBody.Body); got != "array<int>" {
		t.Errorf("shorts request body = %s, want array<int>", got)
	}
	if got := arrayFieldKind(shorts.Response.Body); got != "array<int>" {
		t.Errorf("shorts response = %s, want array<int>", got)
	}
	count := byPath["/jdk-scalars/count"]
	if count == nil || count.RequestBody == nil || count.Response == nil {
		t.Fatalf("count endpoint = %+v, want a request body and a response", count)
	}
	if got := arrayFieldKind(count.RequestBody.Body); got != "int" {
		t.Errorf("count request body = %s, want int", got)
	}
	if got := arrayFieldKind(count.Response.Body); got != "int" {
		t.Errorf("count response = %s, want int", got)
	}

	if unresolved := c.Unresolved(); len(unresolved) != 0 {
		t.Errorf("unresolved = %v, want none", unresolved)
	}
}
