package kotlincollector

import (
	"path/filepath"
	"reflect"
	"testing"

	collector "github.com/tangcent/apilot/api-collector"
)

// varargsBodyExpectation is the export of testdata/vararg: a `vararg`
// parameter is exported as the array it is, the same as one declared with
// Array<T>. The Java collector asserts the same table for the same handlers
// written with `...` (api-collector-java/declarator_test.go): keep the two in
// sync.
var varargsBodyExpectation = map[string]string{
	"/declarators/items":       "array<object{name}>",
	"/declarators/items-array": "array<object{name}>",
}

func TestVarargParametersAreExportedAsArrays(t *testing.T) {
	abs, err := filepath.Abs("testdata/vararg")
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

	// The type of `vararg ids: String` is one element, but what the function
	// receives is an array: it is exported as the array parameter declared
	// the same way.
	search := byPath["/declarators/search"]
	twin := byPath["/declarators/search-array"]
	if search == nil || twin == nil || len(search.Parameters) != 1 || len(twin.Parameters) != 1 {
		t.Fatalf("search = %+v, search-array = %+v, want one parameter each", search, twin)
	}
	if got := search.Parameters[0]; got.Name != "ids" || got.In != "query" || got.TypeName != "Array<String>" || got.Required {
		t.Errorf("search parameter = %+v, want optional ids in query as Array<String>", got)
	}
	if !reflect.DeepEqual(search.Parameters[0], twin.Parameters[0]) {
		t.Errorf("vararg parameter:\n got  %+v\n want %+v", search.Parameters[0], twin.Parameters[0])
	}

	implicit := byPath["/declarators/implicit"]
	if implicit == nil || len(implicit.Parameters) != 1 {
		t.Fatalf("implicit endpoint = %+v, want one parameter", implicit)
	}
	if p := implicit.Parameters[0]; p.Name != "tags" || p.In != "query" || p.TypeName != "Array<String>" {
		t.Errorf("implicit parameter = %+v, want tags in query as Array<String>", p)
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
}
