package javacollector

import (
	"path/filepath"
	"testing"

	collector "github.com/tangcent/apilot/api-collector"
)

// jdkIdentityExpectation is the export of testdata/jdk-identity: the kind of what
// sits at a path of an endpoint's request body ("request.") or response
// ("response."). The library's generic classes and its controller sit in a
// package that declares its own String, Integer and Date; every argument is the
// class the caller names where it writes it: the JDK class where nothing closer
// declares the name, imported by name or on demand, and the application's class
// in the package that declares one.
//
// The Kotlin collector asserts the same table for the same declarations
// (api-collector-kotlin/builtin_scope_test.go).
var jdkIdentityExpectation = map[string]map[string]string{
	"/jdk-args/box-string": {
		"request.data": "string", "request.list": "array<string>", "request.array": "array<string>",
		"request.byKey": "map<string,string>", "request.bounded": "array<string>",
		"response.data": "string", "response.list": "array<string>", "response.array": "array<string>",
		"response.byKey": "map<string,string>", "response.bounded": "array<string>",
	},
	"/jdk-args/box-integer": {
		"response.data": "int", "response.list": "array<int>", "response.array": "array<int>",
	},
	"/jdk-args/box-date": {
		"response.data": "string", "response.list": "array<string>", "response.array": "array<string>",
	},
	"/jdk-args/box-list": {
		"response.data": "array<string>", "response.list": "array<array<string>>",
	},
	"/jdk-args/box-array": {
		"response.data": "array<string>", "response.array": "array<array<string>>",
	},
	"/jdk-args/box-map": {
		"response.data": "map<string,int>", "response.list": "array<map<string,int>>",
	},
	"/jdk-args/outer-string": {
		"response.wrapped.data": "string", "response.wrappedList.data": "array<string>", "response.wrappedArray.data": "array<string>",
	},
	"/jdk-args/outer-date": {
		"response.wrapped.data": "string", "response.wrappedList.data": "array<string>", "response.wrappedArray.data": "array<string>",
	},
	// The library writes the JDK classes in full, and its own String simply.
	"/jdk-args/holder": {
		"response.text": "string", "response.when": "string", "response.boxed.data": "string",
		"response.numbers": "array<int>", "response.own": "object{shadow}",
	},
	// The superclass argument is written in the controller's scope.
	"/jdk-strings/one":      {"response": "string"},
	"/jdk-strings/all":      {"response": "array<string>"},
	"/jdk-strings/save":     {"request": "string", "response": "string"},
	"/jdk-dates/one":        {"response": "string"},
	"/jdk-dates/all":        {"response": "array<string>"},
	"/jdk-dates/save":       {"request": "string", "response": "string"},
	"/jdk-demand/box-date":  {"response.data": "string", "response.list": "array<string>"},
	"/jdk-demand/box-uuid":  {"response.data": "string", "response.list": "array<string>"},
	"/jdk-demand-dates/one": {"response": "string"},
	// The application's own String and Integer.
	"/jdk-own/box-string":  {"response.data": "object{own}", "response.list": "array<object{own}>", "response.byKey": "map<string,object{own}>"},
	"/jdk-own/box-integer": {"response.data": "object{own}", "response.list": "array<object{own}>"},
	// Void is the application's class here, so the method has a response.
	"/jdk-own/void":        {"response": "object{own}"},
	"/jdk-own-strings/one": {"response": "object{own}"},
	"/jdk-own-strings/all": {"response": "array<object{own}>"},
}

// jdkIdentityParameters are the parameters each endpoint binds from the request,
// by the binding view: `search(T query)` binds the argument of its generic
// controller, which for a String is one value.
var jdkIdentityParameters = map[string][]string{
	"/jdk-strings/search":     {"query/query"},
	"/jdk-dates/search":       {"query/query"},
	"/jdk-own-strings/search": {"own/query"},
}

func TestJdkClassesAsArgumentsAreTheClassesTheCallerNames(t *testing.T) {
	abs, err := filepath.Abs("testdata/jdk-identity")
	if err != nil {
		t.Fatal(err)
	}
	c := New().(*JavaCollector)
	endpoints, err := c.Collect(collector.CollectContext{SourceDir: abs, NoDeps: true})
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}
	checkTypeArgumentExport(t, endpoints, jdkIdentityExpectation, jdkIdentityParameters)
	if unresolved := c.Unresolved(); len(unresolved) != 0 {
		t.Errorf("unresolved = %v, want none", unresolved)
	}
}
