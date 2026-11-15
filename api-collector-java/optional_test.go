package javacollector

import (
	"path/filepath"
	"reflect"
	"testing"

	collector "github.com/tangcent/apilot/api-collector"
)

// optionalExpectation is whether each parameter of testdata/optional is
// required. Spring rejects a request without a @RequestParam or
// @RequestHeader unless the parameter is optional: required = false, a
// defaultValue, a java.util.Optional or @Nullable (and in Kotlin a nullable
// type or a default value). An optional parameter stays optional even with
// an explicit required = true, since Spring checks both. A validation
// constraint makes it mandatory again: binding leaves it null, validation
// rejects that.
//
// The Kotlin collector asserts the same table for the same handler written
// in Kotlin (api-collector-kotlin/optional_test.go): keep the two in sync.
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
	abs, err := filepath.Abs("testdata/optional")
	if err != nil {
		t.Fatal(err)
	}
	endpoints, err := New().Collect(collector.CollectContext{SourceDir: abs, NoDeps: true})
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}
	checkOptionalExport(t, endpoints)
}

func checkOptionalExport(t *testing.T, endpoints []collector.ApiEndpoint) {
	t.Helper()
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
