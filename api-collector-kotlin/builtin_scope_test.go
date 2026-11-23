package kotlincollector

import (
	"path/filepath"
	"testing"

	collector "github.com/tangcent/apilot/api-collector"
)

// builtinScopeExpectation is the export of testdata/builtin-scope. Kotlin binds a
// name by its own scopes (explicit imports, the package, star-imports, then the
// packages every file imports), so `Int` is kotlin.Int in a package that
// declares `Integer`, and it is the project's class in a package, an import or
// a star-import that declares an `Int`. The generic classes and the controller
// the arguments are handed to sit in a package that declares Int, Integer and
// String of its own.
var builtinScopeExpectation = map[string]map[string]string{
	"/own/save": {
		"request.count": "int", "request.total": "long", "request.name": "string", "request.flag": "boolean",
		"request.counts": "array<int>", "request.bytes": "array<int>",
		"request.boxed": "object{digits}", "request.weight": "object{digits}", "request.unit": "string",
		"response.count": "int", "response.total": "long", "response.name": "string", "response.flag": "boolean",
		"response.counts": "array<int>", "response.bytes": "array<int>",
		"response.boxed": "object{digits}", "response.weight": "object{digits}", "response.unit": "string",
	},
	"/own/unit": {"response": "string"},
	"/own/box-int": {
		"response.data": "int", "response.list": "array<int>", "response.byKey": "map<string,int>",
	},
	"/own/box-string": {
		"response.data": "string", "response.list": "array<string>", "response.byKey": "map<string,string>",
	},
	"/own/box-integer": {
		"response.data": "object{digits}", "response.list": "array<object{digits}>",
	},
	"/own/box-list": {
		"response.data": "array<int>", "response.list": "array<array<int>>",
	},
	"/own/outer-int": {
		"response.wrapped.data": "int", "response.wrappedList.data": "array<int>",
	},
	// The superclass argument is written in the controller's scope.
	"/own-items/one":  {"response": "int"},
	"/own-items/all":  {"response": "array<int>"},
	"/own-items/save": {"request": "int", "response": "int"},
	"/own-names/one":  {"response": "string"},
	"/own-names/save": {"request": "string", "response": "string"},
	"/shadow/save":    builtinScopeShadowForm(),
	"/imported/save":  builtinScopeShadowForm(),
	"/star/save":      builtinScopeShadowForm(),
	"/shadow/box":     {"response.data": "object{digits}", "response.list": "array<object{digits}>"},
	"/imported/box":   {"response.data": "object{digits}", "response.list": "array<object{digits}>"},
	"/star/box":       {"response.data": "object{digits}", "response.list": "array<object{digits}>"},
	// A renaming import binds only the new name.
	"/alias/save": {
		"request.count": "object{digits}", "request.bytes": "object{raw}", "request.total": "long",
		"response.count": "object{digits}", "response.bytes": "object{raw}", "response.total": "long",
	},
}

// builtinScopeShadowForm is the form of the packages whose Int, ByteArray and
// Number are classes of the project: the request body and the response.
func builtinScopeShadowForm() map[string]string {
	out := map[string]string{}
	for _, side := range []string{"request.", "response."} {
		out[side+"count"] = "object{digits}"
		out[side+"bytes"] = "object{raw}"
		out[side+"weight"] = "object{digits}"
		out[side+"total"] = "long"
	}
	out["request.label"] = "string"
	out["response.label"] = "string"
	return out
}

// builtinScopeParameters are the request parameters a model attribute is bound
// from, by the binding view.
var builtinScopeParameters = map[string][]string{
	"/own/search": {
		"count/query", "total/query", "name/query", "flag/query", "counts/query", "bytes/query",
		"boxed.digits/query", "weight.digits/query", "unit/query",
	},
}

func TestKotlinNamesAreBoundByKotlinScopes(t *testing.T) {
	abs, err := filepath.Abs("testdata/builtin-scope")
	if err != nil {
		t.Fatal(err)
	}
	c := New().(*KotlinCollector)
	endpoints, err := c.Collect(collector.CollectContext{SourceDir: abs, NoDeps: true})
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}
	checkTypeArgumentExport(t, endpoints, builtinScopeExpectation, builtinScopeParameters)
	if unresolved := c.Unresolved(); len(unresolved) != 0 {
		t.Errorf("unresolved = %v, want none", unresolved)
	}
}
