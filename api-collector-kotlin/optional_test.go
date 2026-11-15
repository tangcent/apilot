package kotlincollector

import (
	"reflect"
	"testing"
)

// optionalExpectation is whether each parameter of testdata/optional is
// required. Spring rejects a request without a @RequestParam or
// @RequestHeader unless the parameter is optional: required = false, a
// defaultValue, @Nullable, a nullable type or a default value (and in Java a
// java.util.Optional). A nullable parameter stays optional even with an
// explicit required = true, since Spring checks both. A validation
// constraint makes it mandatory again: binding leaves it null, validation
// rejects that.
//
// The Java collector asserts the same table for the same handler written in
// Java (api-collector-java/optional_test.go): keep the two in sync.
var optionalExpectation = map[string]bool{
	"keyword":  true,
	"tag":      false,
	"size":     false,
	"cursor":   false,
	"region":   false,
	"ownerId":  true,
	"X-Trace":  false,
	"X-Tenant": true,
}

func TestOptionalParametersAreNotRequired(t *testing.T) {
	endpoints, _ := collectDir(t, "testdata/optional")
	if len(endpoints) != 1 {
		t.Fatalf("endpoints = %+v, want the search endpoint", endpoints)
	}
	got := map[string]bool{}
	for _, p := range endpoints[0].Parameters {
		got[p.Name] = p.Required
	}
	for _, h := range endpoints[0].Headers {
		got[h.Name] = h.Required
	}
	if !reflect.DeepEqual(got, optionalExpectation) {
		t.Errorf("required:\n got  %v\n want %v", got, optionalExpectation)
	}
}
