package kotlincollector

import (
	"reflect"
	"testing"

	model "github.com/tangcent/apilot/api-model"
)

// byteArrayVarargFormExpectation is, per class of testdata/byte-array with a
// `vararg val` property, the kind of the property and of the one its explicit
// twin declares: a ByteArray for `Byte` and an alias of it, a CharArray for
// `Char`, an `Array` of the element for a nullable element and for `String`.
var byteArrayVarargFormExpectation = map[string]string{
	"bytes":       "string",
	"chars":       "string",
	"boxed-bytes": "array<int>",
	"boxed-chars": "array<string>",
	"names":       "array<string>",
	"ints":        "array<int>",
	"octets":      "string",
	"maybe-bytes": "array<int>",
	"mutable":     "string",
}

// A property of the primary constructor declared `vararg val` is the array it
// holds, in a request body and in a response, as the property declared with that
// array is.
func TestVarargPropertiesAreExportedAsTheArraysTheyAre(t *testing.T) {
	c, byPath := collectByteArrays(t)

	for name, kind := range byteArrayVarargFormExpectation {
		for _, form := range []string{"explicit", "vararg"} {
			path := "/byte-vararg-forms/" + form + "/" + name
			endpoint := byPath[path]
			if endpoint == nil || endpoint.RequestBody == nil || endpoint.Response == nil {
				t.Errorf("%s = %+v, want a request body and a response", path, endpoint)
				continue
			}
			for what, body := range map[string]*model.ObjectModel{
				"request body": endpoint.RequestBody.Body,
				"response":     endpoint.Response.Body,
			} {
				want := map[string]string{"data": kind}
				if got := arrayFieldKinds(body); !reflect.DeepEqual(got, want) {
					t.Errorf("%s %s:\n got  %v\n want %v", path, what, got, want)
				}
			}
		}
	}

	if unresolved := c.Unresolved(); len(unresolved) != 0 {
		t.Errorf("unresolved = %v, want none", unresolved)
	}
}

// The data binder sets a `vararg val` property as it does one declared with the
// array: it is listed as a request parameter holding an array.
func TestVarargPropertiesAreBoundAsTheArraysTheyAre(t *testing.T) {
	_, byPath := collectByteArrays(t)

	for _, name := range []string{"bytes", "chars", "boxed-bytes", "names", "ints", "octets", "mutable"} {
		explicit := byPath["/byte-vararg-forms/explicit-bound/"+name]
		vararg := byPath["/byte-vararg-forms/vararg-bound/"+name]
		if explicit == nil || vararg == nil {
			t.Errorf("%s: explicit = %+v, vararg = %+v, want both", name, explicit, vararg)
			continue
		}
		got, want := map[string]string{}, map[string]string{"data": "array"}
		for _, p := range vararg.Parameters {
			got[p.Name] = p.TypeName
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("%s: vararg parameters:\n got  %v\n want %v", name, got, want)
		}
		if !reflect.DeepEqual(vararg.Parameters, explicit.Parameters) {
			t.Errorf("%s: vararg parameters:\n got  %+v\n want %+v", name, vararg.Parameters, explicit.Parameters)
		}
	}
}
