package kotlincollector

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	collector "github.com/tangcent/apilot/api-collector"
	model "github.com/tangcent/apilot/api-model"
)

// byteArrayExpectation is the export of testdata/byte-array: the fields of the
// request body and of the response, each as its kind. Jackson writes a
// ByteArray, the JVM's byte[], as one Base64 string; `Array<Byte>` and
// `List<Byte>` are arrays of numbers, as are the other primitive arrays.
//
// The Java collector asserts the same table for the same declarations written
// in Java (api-collector-java/byte_array_test.go), where `legacy` and
// `signature` are also declared, which only Java can spell: keep the two in
// sync.
var byteArrayExpectation = map[string]string{
	"data":    "string",
	"chunks":  "array<string>",
	"blobs":   "array<string>",
	"boxed":   "array<int>",
	"numbers": "array<int>",
	"single":  "int",
	"ints":    "array<int>",
}

// How Spring binds a request value is not how Jackson writes one: the data
// binder reads a byte array from the text of the value, so the binding view
// keeps it as an array and the parameters are listed as before.
var byteArrayBindingExpectation = map[string]string{
	"data":  "array",
	"boxed": "array",
}

func collectByteArrays(t *testing.T) (*KotlinCollector, map[string]*collector.ApiEndpoint) {
	t.Helper()
	abs, err := filepath.Abs("testdata/byte-array")
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
	return c, byPath
}

func TestByteArraysAreExportedAsStrings(t *testing.T) {
	c, byPath := collectByteArrays(t)

	save := byPath["/byte-arrays/save"]
	if save == nil || save.RequestBody == nil || save.RequestBody.Body == nil || save.Response == nil {
		t.Fatalf("save endpoint = %+v, want a request body and a response", save)
	}
	for what, body := range map[string]*model.ObjectModel{
		"request body": save.RequestBody.Body,
		"response":     save.Response.Body,
	} {
		if got := arrayFieldKinds(body); !reflect.DeepEqual(got, byteArrayExpectation) {
			t.Errorf("%s:\n got  %v\n want %v", what, got, byteArrayExpectation)
		}
	}

	// A ByteArray taken or returned by a handler as a whole is the same scalar.
	blob := byPath["/byte-arrays/blob"]
	if blob == nil || blob.RequestBody == nil || blob.Response == nil {
		t.Fatalf("blob endpoint = %+v, want a request body and a response", blob)
	}
	if got := arrayFieldKind(blob.RequestBody.Body); got != "string" {
		t.Errorf("blob request body = %s, want string", got)
	}
	if got := arrayFieldKind(blob.Response.Body); got != "string" {
		t.Errorf("blob response = %s, want string", got)
	}

	if unresolved := c.Unresolved(); len(unresolved) != 0 {
		t.Errorf("unresolved = %v, want none", unresolved)
	}
}

func TestByteArrayRequestParametersAreListedAsDeclared(t *testing.T) {
	_, byPath := collectByteArrays(t)

	query := byPath["/byte-arrays/query"]
	if query == nil {
		t.Fatalf("query endpoint missing: %v", byPath)
	}
	type param struct{ Name, In, Type, TypeName string }
	var got []param
	for _, p := range query.Parameters {
		got = append(got, param{p.Name, p.In, p.Type, p.TypeName})
	}
	want := []param{
		{"data", "query", "text", "ByteArray"},
		{"boxed", "query", "text", "Array<Byte>"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("query parameters:\n got  %v\n want %v", got, want)
	}

	// The properties of a bound object are request parameters in the binding view.
	bound := byPath["/byte-arrays/bound"]
	if bound == nil {
		t.Fatalf("bound endpoint missing: %v", byPath)
	}
	gotBound := map[string]string{}
	for _, p := range bound.Parameters {
		gotBound[p.Name] = p.TypeName
	}
	if !reflect.DeepEqual(gotBound, byteArrayBindingExpectation) {
		t.Errorf("bound parameters:\n got  %v\n want %v", gotBound, byteArrayBindingExpectation)
	}
}

// byteArrayGenericExpectation is, per endpoint of testdata/byte-array, the kind
// of the fields at a path of the request body and of the response. A path
// follows field names, `[]` for the elements of an array and `*` for the value
// of a map. The type argument that reaches each of them is a ByteArray, handed
// on through one or two generic classes, a superclass or the handler's own base
// class: where it is a ByteArray the field is the Base64 string, and where it
// is `Array<Byte>` an array of numbers.
//
// The Java collector asserts the same table for the same declarations written
// in Java (api-collector-java/byte_array_test.go): keep the two in sync.
var byteArrayGenericExpectation = map[string]map[string]string{
	"/byte-arrays/outer": {
		"wrapped.data":         "string",
		"wrapped.items":        "array<string>",
		"wrapped.grid":         "array<string>",
		"wrapped.byName":       "map<string,string>",
		"wrappedItems.[].data": "string",
		"direct":               "string",
	},
	"/byte-arrays/outer-boxed": {
		"wrapped.data":         "array<int>",
		"wrapped.items":        "array<array<int>>",
		"wrapped.grid":         "array<array<int>>",
		"wrapped.byName":       "map<string,array<int>>",
		"wrappedItems.[].data": "array<int>",
		"direct":               "array<int>",
	},
	"/byte-arrays/outer-chunks": {
		"wrapped.data":   "array<string>",
		"wrapped.items":  "array<array<string>>",
		"wrapped.grid":   "array<array<string>>",
		"wrapped.byName": "map<string,array<string>>",
		"direct":         "array<string>",
	},
	"/byte-arrays/outer-list": {
		"wrapped.data":  "array<string>",
		"wrapped.items": "array<array<string>>",
		"direct":        "array<string>",
	},
	"/byte-arrays/outer-wildcard": {
		"wrapped.data":  "string",
		"wrapped.items": "array<string>",
		"direct":        "string",
	},
	"/byte-arrays/sub": {
		"payload":        "string",
		"wrapped.data":   "string",
		"wrapped.items":  "array<string>",
		"wrapped.grid":   "array<string>",
		"wrapped.byName": "map<string,string>",
	},
	"/byte-arrays/sub-boxed": {
		"payload":       "array<int>",
		"wrapped.data":  "array<int>",
		"wrapped.items": "array<array<int>>",
	},
	"/byte-arrays/open-sub": {
		"payload":       "string",
		"wrapped.data":  "string",
		"wrapped.items": "array<string>",
	},
	"/byte-arrays/by-name": {
		"*.data":   "string",
		"*.items":  "array<string>",
		"*.byName": "map<string,string>",
	},
}

func TestByteArraysThroughGenericClasses(t *testing.T) {
	c, byPath := collectByteArrays(t)

	for path, want := range byteArrayGenericExpectation {
		endpoint := byPath[path]
		if endpoint == nil || endpoint.RequestBody == nil || endpoint.Response == nil {
			t.Errorf("%s = %+v, want a request body and a response", path, endpoint)
			continue
		}
		for what, body := range map[string]*model.ObjectModel{
			"request body": endpoint.RequestBody.Body,
			"response":     endpoint.Response.Body,
		} {
			for fieldPath, kind := range want {
				if got := arrayFieldKind(byteArrayModelAt(body, fieldPath)); got != kind {
					t.Errorf("%s %s %s = %s, want %s", path, what, fieldPath, got, kind)
				}
			}
		}
	}

	// A handler that inherits `echo(body: T)` from a base class of `T`,
	// extending it with ByteArray, takes and returns the byte array as a string.
	echo := byPath["/byte-arrays/typed/echo"]
	if echo == nil || echo.RequestBody == nil || echo.Response == nil {
		t.Fatalf("echo endpoint = %+v, want a request body and a response", echo)
	}
	if got := arrayFieldKind(echo.RequestBody.Body); got != "string" {
		t.Errorf("echo request body = %s, want string", got)
	}
	if got := arrayFieldKind(byteArrayModelAt(echo.Response.Body, "data")); got != "string" {
		t.Errorf("echo response data = %s, want string", got)
	}

	if unresolved := c.Unresolved(); len(unresolved) != 0 {
		t.Errorf("unresolved = %v, want none", unresolved)
	}
}

// The binding view reads the same bindings and keeps a byte array an array,
// down to the field two generic classes deep.
func TestByteArraysThroughGenericClassesWhenBound(t *testing.T) {
	c, byPath := collectByteArrays(t)

	bound := byPath["/byte-arrays/bound-outer"]
	if bound == nil {
		t.Fatalf("bound-outer endpoint missing: %v", byPath)
	}
	got := map[string]string{}
	for _, p := range bound.Parameters {
		got[p.Name] = p.TypeName
	}
	want := map[string]string{
		"wrapped.data":   "array",
		"wrapped.items":  "array",
		"wrapped.grid":   "array",
		"wrapped.byName": "map",
		"wrappedItems":   "array",
		"direct":         "array",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("bound-outer parameters:\n got  %v\n want %v", got, want)
	}

	if unresolved := c.Unresolved(); len(unresolved) != 0 {
		t.Errorf("unresolved = %v, want none", unresolved)
	}
}

// byteArrayModelAt follows a path of field names, `[]` and `*` from a model.
func byteArrayModelAt(m *model.ObjectModel, path string) *model.ObjectModel {
	for _, step := range strings.Split(path, ".") {
		switch {
		case m == nil:
			return nil
		case step == "[]":
			m = m.Items
		case step == "*":
			m = m.ValueModel
		case m.Fields[step] == nil:
			return nil
		default:
			m = m.Fields[step].Model
		}
	}
	return m
}
