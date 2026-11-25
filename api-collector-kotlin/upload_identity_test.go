package kotlincollector

import (
	"path/filepath"
	"reflect"
	"sort"
	"testing"

	collector "github.com/tangcent/apilot/api-collector"
)

// uploadIdentityExpectation is the export of testdata/upload-identity: the
// parameters of each endpoint as `name in type`, where the type is `file` for a
// part of a multipart request. A parameter is an upload by the class its type
// names, not by its simple name: the application's own MultipartFile is a bean
// bound property by property, held by a java.util.Optional or not, and Spring's
// is a file however the file reaches it.
//
// The Java collector asserts the same table for the same declarations
// (api-collector-java/upload_identity_test.go).
var uploadIdentityExpectation = map[string][]string{
	"/own/single":       {"name form text", "size form text"},
	"/own/optional":     {"name form text", "size form text"},
	"/own/list":         {},
	"/own/names":        {"method query text", "label query text", "by query text", "locale query text"},
	"/own/own-optional": {"prefix query text", "value.label query text"},
	"/lib/single":       {"file form file"},
	"/lib/optional":     {"avatar form file"},
	"/lib/list":         {"files form file", "docs form file"},
	"/lib/annotated":    {"cover form file", "parts form file"},
	"/demand/single":    {"file form file", "avatar form file"},
	"/demand/qualified": {"file form file"},
}

// uploadIdentityRequired is whether the parameter of an endpoint is required: a
// java.util.Optional is left unbound when the request has no value, and a class of
// the application that is called Optional is not one.
var uploadIdentityRequired = map[string]bool{
	"/own/optional-param": true,
	"/lib/optional-param": false,
}

func TestAnUploadIsTheClassItsTypeNames(t *testing.T) {
	abs, err := filepath.Abs("testdata/upload-identity")
	if err != nil {
		t.Fatal(err)
	}
	c := New().(*KotlinCollector)
	endpoints, err := c.Collect(collector.CollectContext{SourceDir: abs, NoDeps: true})
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}
	byPath := map[string]*collector.ApiEndpoint{}
	var exported []string
	for i := range endpoints {
		byPath[endpoints[i].Path] = &endpoints[i]
		exported = append(exported, endpoints[i].Path)
	}
	sort.Strings(exported)
	for path, want := range uploadIdentityExpectation {
		ep := byPath[path]
		if ep == nil {
			t.Errorf("endpoint %s was not exported; exported %v", path, exported)
			continue
		}
		got := []string{}
		for _, p := range ep.Parameters {
			got = append(got, p.Name+" "+p.In+" "+p.Type)
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("%s parameters = %v, want %v", path, got, want)
		}
	}
	for path, want := range uploadIdentityRequired {
		ep := byPath[path]
		if ep == nil || len(ep.Parameters) != 1 {
			t.Errorf("%s = %+v, want one parameter", path, ep)
			continue
		}
		if got := ep.Parameters[0].Required; got != want {
			t.Errorf("%s parameter required = %v, want %v", path, got, want)
		}
	}
}
