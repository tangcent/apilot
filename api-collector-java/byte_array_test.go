package javacollector

import (
	"path/filepath"
	"reflect"
	"testing"

	collector "github.com/tangcent/apilot/api-collector"
	model "github.com/tangcent/apilot/api-model"
)

// byteArrayExpectation is the export of testdata/byte-array: the fields of the
// request body and of the response, each as its kind. Jackson writes a byte[]
// as one Base64 string, whichever way it is spelled; `Byte[]` and
// `List<Byte>` are arrays of numbers, as are the other primitive arrays.
//
// The Kotlin collector asserts the same table for the same declarations
// written in Kotlin (api-collector-kotlin/byte_array_test.go), except for
// `legacy` (brackets after the name) and `signature` (a getter without a
// field), which only Java can spell: keep the two in sync.
var byteArrayExpectation = map[string]string{
	"data":      "string",
	"legacy":    "string",
	"signature": "string",
	"chunks":    "array<string>",
	"blobs":     "array<string>",
	"boxed":     "array<int>",
	"numbers":   "array<int>",
	"single":    "int",
	"ints":      "array<int>",
}

// How Spring binds a request value is not how Jackson writes one: the data
// binder reads a byte[] from the text of the value, so the binding view keeps
// it as an array and the parameters are listed as before.
var byteArrayBindingExpectation = map[string]string{
	"data":  "array",
	"boxed": "array",
}

func collectByteArrays(t *testing.T) (*JavaCollector, map[string]*collector.ApiEndpoint) {
	t.Helper()
	abs, err := filepath.Abs("testdata/byte-array")
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

	// A byte[] taken or returned by a handler as a whole is the same scalar.
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
		{"data", "query", "text", "byte[]"},
		{"boxed", "query", "text", "Byte[]"},
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
