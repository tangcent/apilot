package javacollector

import (
	"path/filepath"
	"reflect"
	"testing"

	collector "github.com/tangcent/apilot/api-collector"
	model "github.com/tangcent/apilot/api-model"
)

// platformTypesExpectation is the export of the form of testdata/platform-types
// that names every type in full or by an import of its own: each field of the
// request body and of the response, as its kind. A collection and a map are one
// whichever way they are spelled, an Optional is the value it holds, the JDK
// values are scalars, a JDK enum is a string, a Jackson tree is free-form, and
// a multipart file keeps its name, which is how a parameter of that type is told
// to be a file.
//
// The Kotlin collector asserts the same table for the same declarations
// written in Kotlin (api-collector-kotlin/platform_types_test.go): keep the two
// in sync.
var platformTypesExpectation = map[string]string{
	"qualifiedList":       "array<string>",
	"qualifiedSet":        "array<long>",
	"qualifiedCollection": "array<int>",
	"qualifiedIterable":   "array<string>",
	"qualifiedMap":        "map<string,long>",
	"iterable":            "array<string>",
	"linkedSet":           "array<string>",
	"sortedMap":           "map<string,int>",
	"optionalName":        "string",
	"qualifiedOptional":   "long",
	"optionalItem":        "object{name,qty}",
	"optionalTags":        "array<string>",
	"optionalInt":         "int",
	"optionalLong":        "long",
	"optionalDouble":      "double",
	"text":                "string",
	"locale":              "string",
	"currency":            "string",
	"timeZone":            "string",
	"zoneId":              "string",
	"day":                 "string",
	"month":               "string",
	"atomicInteger":       "int",
	"atomicLong":          "long",
	"atomicBoolean":       "boolean",
	"longAdder":           "long",
	"timestamp":           "string",
	"sqlDate":             "string",
	"sqlTime":             "string",
	"jsonNode":            "object{}",
	"objectNode":          "object{}",
	"arrayNode":           "array<object{}>",
	"file":                "MultipartFile",
}

// platformWildcardExpectation is the export of the form whose every type is
// imported on demand. A bare `Timestamp` or `Month` is the JDK's because the
// source imports its package.
var platformWildcardExpectation = map[string]string{
	"stamp":    "string",
	"clock":    "string",
	"month":    "string",
	"day":      "string",
	"months":   "array<string>",
	"note":     "string",
	"count":    "int",
	"currency": "string",
	"locale":   "string",
	"node":     "object{}",
	"file":     "MultipartFile",
}

// platformOptional lists the properties that hold a value or nothing: the
// others are required, as every property is in the default mode.
var platformOptional = map[string]bool{
	"optionalName": true, "qualifiedOptional": true, "optionalItem": true, "optionalTags": true,
	"optionalInt": true, "optionalLong": true, "optionalDouble": true,
}

var platformOptions = map[string][]any{
	"day":   {"MONDAY", "TUESDAY", "WEDNESDAY", "THURSDAY", "FRIDAY", "SATURDAY", "SUNDAY"},
	"month": {"JANUARY", "FEBRUARY", "MARCH", "APRIL", "MAY", "JUNE", "JULY", "AUGUST", "SEPTEMBER", "OCTOBER", "NOVEMBER", "DECEMBER"},
}

// platformParameters is what each handler of the fixture takes, as
// name/location/type in declaration order. An Optional parameter is bound as the
// type it holds and is optional, and a file is a part of a multipart form.
var platformParameters = map[string][]string{
	"/platform-types/search": {
		"keyword/query/text", "page/query/text", "name/query/text", "qty/query/text",
		"currency/query/text", "day/query/text",
	},
	"/platform-types/upload": {
		"file/form/file", "title/form/text", "cover/form/file", "attachments/form/file", "extras/form/file",
		"avatar/form/file",
	},
}

func TestPlatformTypesAreResolved(t *testing.T) {
	abs, err := filepath.Abs("testdata/platform-types")
	if err != nil {
		t.Fatal(err)
	}
	c := New().(*JavaCollector)
	endpoints, err := c.Collect(collector.CollectContext{SourceDir: abs, NoDeps: true})
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}
	checkPlatformTypesExport(t, endpoints, c.Unresolved())
}

func checkPlatformTypesExport(t *testing.T, endpoints []collector.ApiEndpoint, unresolved map[string]int) {
	t.Helper()
	byPath := map[string]*collector.ApiEndpoint{}
	for i := range endpoints {
		byPath[endpoints[i].Path] = &endpoints[i]
	}
	endpoint := func(path string) *collector.ApiEndpoint {
		t.Helper()
		ep := byPath[path]
		if ep == nil {
			t.Fatalf("no endpoint %s in %d endpoints", path, len(endpoints))
		}
		return ep
	}

	for path, want := range map[string]map[string]string{
		"/platform-types/save":     platformTypesExpectation,
		"/platform-types/wildcard": platformWildcardExpectation,
	} {
		ep := endpoint(path)
		if ep.RequestBody == nil || ep.RequestBody.Body == nil || ep.Response == nil || ep.Response.Body == nil {
			t.Fatalf("%s = %+v, want a request body and a response", path, ep)
		}
		for what, body := range map[string]*model.ObjectModel{
			"request body": ep.RequestBody.Body,
			"response":     ep.Response.Body,
		} {
			if got := arrayFieldKinds(body); !reflect.DeepEqual(got, want) {
				t.Errorf("%s %s:\n got  %v\n want %v", path, what, got, want)
			}
		}
	}

	// A property that holds a value or nothing may be left out; the JDK enums
	// carry their constants.
	form := endpoint("/platform-types/save").RequestBody.Body
	for name, field := range form.Fields {
		if want := !platformOptional[name]; field.Required != want {
			t.Errorf("%s: required = %v, want %v", name, field.Required, want)
		}
		var options []any
		for _, option := range field.Options {
			options = append(options, option.Value)
		}
		if want := platformOptions[name]; !reflect.DeepEqual(options, want) {
			t.Errorf("%s: options = %v, want %v", name, options, want)
		}
	}
	if note := endpoint("/platform-types/wildcard").RequestBody.Body.Fields["note"]; note.Required {
		t.Errorf("note: required, want a property that holds a value or nothing to be optional")
	}

	// Handlers: what is taken and returned as a whole.
	for path, want := range map[string]struct{ request, response string }{
		"/platform-types/nodes": {"object{}", "object{}"},
		"/platform-types/names": {"array<string>", "array<object{name,qty}>"},
		"/platform-types/maybe": {"object{name,qty}", "object{name,qty}"},
	} {
		ep := endpoint(path)
		if ep.RequestBody == nil || ep.Response == nil {
			t.Fatalf("%s = %+v, want a request body and a response", path, ep)
		}
		if got := arrayFieldKind(ep.RequestBody.Body); got != want.request {
			t.Errorf("%s request body = %s, want %s", path, got, want.request)
		}
		if got := arrayFieldKind(ep.Response.Body); got != want.response {
			t.Errorf("%s response = %s, want %s", path, got, want.response)
		}
	}

	for path, want := range platformParameters {
		var got []string
		for _, p := range endpoint(path).Parameters {
			got = append(got, p.Name+"/"+p.In+"/"+p.Type)
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("%s parameters:\n got  %v\n want %v", path, got, want)
		}
	}
	search := map[string]bool{}
	for _, p := range endpoint("/platform-types/search").Parameters {
		search[p.Name] = p.Required
	}
	if search["keyword"] || search["page"] {
		t.Errorf("an Optional parameter is optional: keyword=%v page=%v", search["keyword"], search["page"])
	}

	if len(unresolved) != 0 {
		t.Errorf("unresolved = %v, want none", unresolved)
	}
}
