package javacollector

import (
	"path/filepath"
	"reflect"
	"testing"

	collector "github.com/tangcent/apilot/api-collector"
	model "github.com/tangcent/apilot/api-model"
)

// A class of the project may carry the simple name of a java.lang class, and in
// the files that see it the name is that class: through a single-type import,
// or through the package. Everywhere else the name is still the JDK class and
// exports as the scalar it is. testdata/java-lang-scope declares Short and
// Number in a package of their own and Short and Integer next to LocalForm.
func TestCollect_ProjectClassHidesJavaLangName(t *testing.T) {
	abs, err := filepath.Abs("testdata/java-lang-scope")
	if err != nil {
		t.Fatal(err)
	}
	c := New().(*JavaCollector)
	endpoints, err := c.Collect(collector.CollectContext{SourceDir: abs, NoDeps: true})
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}
	byPath := map[string]*collector.ApiEndpoint{}
	for i := range endpoints {
		byPath[endpoints[i].Path] = &endpoints[i]
	}

	const projectClass = "object{digits}"
	for path, want := range map[string]map[string]string{
		// Short is the class named by the import.
		"/lang-scope/imported": {
			"value":        projectClass,
			"values":       "array<" + projectClass + ">",
			"valueList":    "array<" + projectClass + ">",
			"valuesByName": "map<string," + projectClass + ">",
		},
		// Short and Integer are the classes of the package; Long and the
		// qualified java.lang.Short are not hidden by them.
		"/lang-scope/local": {
			"value":     projectClass,
			"count":     projectClass,
			"valueList": "array<" + projectClass + ">",
			"total":     "long",
			"jdkValue":  "int",
		},
		// The project has a Short and a Number, but not in sight of this file.
		"/lang-scope/plain": {
			"value":     "int",
			"count":     "int",
			"amount":    "double",
			"flag":      "int",
			"letter":    "string",
			"extra":     "object{}",
			"valueList": "array<int>",
		},
	} {
		endpoint := byPath[path]
		if endpoint == nil || endpoint.RequestBody == nil || endpoint.RequestBody.Body == nil || endpoint.Response == nil {
			t.Fatalf("%s: endpoints = %+v, want an endpoint with a request body and a response", path, endpoints)
		}
		for what, body := range map[string]*model.ObjectModel{
			"request body": endpoint.RequestBody.Body,
			"response":     endpoint.Response.Body,
		} {
			if got := arrayFieldKinds(body); !reflect.DeepEqual(got, want) {
				t.Errorf("%s %s:\n got  %v\n want %v", path, what, got, want)
			}
		}
	}

	if unresolved := c.Unresolved(); len(unresolved) != 0 {
		t.Errorf("unresolved = %v, want none", unresolved)
	}
}
