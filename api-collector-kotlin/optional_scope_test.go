package kotlincollector

import (
	"reflect"
	"testing"

	collector "github.com/tangcent/apilot/api-collector"
)

// optionalScopeExpectation is what each handler of testdata/optional-scope takes,
// as name/location/type. The project declares a class named Optional, and a
// handler parameter of that name is a bean bound property by property when the
// file reaches the project's class (from its package, by name, or by an import
// on demand of its package), and java.util.Optional, bound as the type it holds,
// when the file reaches that: by name, or by an import on demand of java.util
// next to a project class that the file does not see.
//
// The Java collector asserts the same table for the same declarations written
// in Java (api-collector-java/optional_scope_test.go): keep the two in sync.
var optionalScopeExpectation = map[string][]string{
	"/optional-scope/own":      {"prefix/query/text", "value/query/text"},
	"/optional-scope/imported": {"prefix/query/text", "value/query/text"},
	"/optional-scope/demand":   {"prefix/query/text", "value/query/text"},

	"/optional-scope/jdk":        {"filter/query/text"},
	"/optional-scope/jdk-demand": {"filter/query/text"},
	"/optional-scope/jdk-upload": {"avatar/form/file"},
}

func TestOptionalIsTheJavaUtilClassOnlyWhereTheScopeSaysSo(t *testing.T) {
	endpoints, c := collectDir(t, "testdata/optional-scope")
	byPath := map[string]*collector.ApiEndpoint{}
	for i := range endpoints {
		byPath[endpoints[i].Path] = &endpoints[i]
	}

	for path, want := range optionalScopeExpectation {
		ep := byPath[path]
		if ep == nil {
			t.Fatalf("no endpoint %s in %d endpoints", path, len(endpoints))
		}
		var got []string
		for _, p := range ep.Parameters {
			got = append(got, p.Name+"/"+p.In+"/"+p.Type)
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("%s parameters:\n got  %v\n want %v", path, got, want)
		}
	}

	// An Optional of java.util is optional; the project's bean binds as it is
	// declared.
	for _, path := range []string{"/optional-scope/jdk", "/optional-scope/jdk-demand"} {
		for _, p := range byPath[path].Parameters {
			if p.Required {
				t.Errorf("%s: parameter %s is required, want an optional one", path, p.Name)
			}
		}
	}
	if unresolved := c.Unresolved(); len(unresolved) != 0 {
		t.Errorf("unresolved = %v, want none", unresolved)
	}
}
