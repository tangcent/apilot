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
// ByteArray, the JVM's byte[], as one Base64 string and a CharArray, the JVM's
// char[], as one string of its characters; `Array<Byte>` and `List<Byte>` are
// arrays of numbers and `Array<Char>` and `List<Char>` arrays of one-character
// strings, as are the other primitive arrays.
//
// The Java collector asserts the same table for the same declarations written
// in Java (api-collector-java/byte_array_test.go), where `legacy`,
// `legacyLetters`, `signature`, `password` and the `spaced` and `annotated`
// fields are also declared, which only Java can spell: keep the two in sync.
var byteArrayExpectation = map[string]string{
	"data":    "string",
	"chunks":  "array<string>",
	"blobs":   "array<string>",
	"boxed":   "array<int>",
	"numbers": "array<int>",
	"single":  "int",
	"ints":    "array<int>",

	"letters":       "string",
	"lines":         "array<string>",
	"words":         "array<string>",
	"lettersByName": "map<string,string>",
	"boxedLetters":  "array<string>",
	"letterList":    "array<string>",
	"letter":        "string",
}

// How Spring binds a request value is not how Jackson writes one: the data
// binder reads a byte or char array from the text of the value, so the binding
// view keeps them as arrays and the parameters are listed as before.
var byteArrayBindingExpectation = map[string]string{
	"data":         "array",
	"boxed":        "array",
	"letters":      "array",
	"boxedLetters": "array",
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

	// A ByteArray taken or returned by a handler as a whole is not Jackson's:
	// Spring's ByteArrayHttpMessageConverter reads and writes the raw bytes.
	// It is exported as a string all the same, the closest schema of an opaque
	// body, where it used to be an array of integers. A CharArray as a whole is
	// Jackson's, one string of its characters.
	for path, what := range map[string]string{"/byte-arrays/blob": "blob", "/byte-arrays/text": "text"} {
		whole := byPath[path]
		if whole == nil || whole.RequestBody == nil || whole.Response == nil {
			t.Errorf("%s endpoint = %+v, want a request body and a response", what, whole)
			continue
		}
		if got := arrayFieldKind(whole.RequestBody.Body); got != "string" {
			t.Errorf("%s request body = %s, want string", what, got)
		}
		if got := arrayFieldKind(whole.Response.Body); got != "string" {
			t.Errorf("%s response = %s, want string", what, got)
		}
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
		{"letters", "query", "text", "CharArray"},
		{"boxedLetters", "query", "text", "Array<Char>"},
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
// of a map. The type argument that reaches each of them is a ByteArray or a
// CharArray, handed on through one or two generic classes, a superclass or the
// handler's own base class: where it is a ByteArray or a CharArray the field is
// a string, and where it is `Array<Byte>` an array of numbers or `Array<Char>`
// an array of one-character strings.
//
// The Java collector asserts the same table for the same declarations written
// in Java (api-collector-java/byte_array_test.go): keep the two in sync.
var byteArrayGenericExpectation = map[string]map[string]string{
	"/byte-arrays/outer-chars": {
		"wrapped.data":         "string",
		"wrapped.items":        "array<string>",
		"wrapped.grid":         "array<string>",
		"wrapped.byName":       "map<string,string>",
		"wrappedItems.[].data": "string",
		"direct":               "string",
	},
	"/byte-arrays/outer-boxed-chars": {
		"wrapped.data":         "array<string>",
		"wrapped.items":        "array<array<string>>",
		"wrapped.grid":         "array<array<string>>",
		"wrapped.byName":       "map<string,array<string>>",
		"wrappedItems.[].data": "array<string>",
		"direct":               "array<string>",
	},
	"/byte-arrays/char-sub": {
		"payload":        "string",
		"wrapped.data":   "string",
		"wrapped.items":  "array<string>",
		"wrapped.grid":   "array<string>",
		"wrapped.byName": "map<string,string>",
	},
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
	// extending it with ByteArray or CharArray, takes and returns the array as
	// a string.
	for _, path := range []string{"/byte-arrays/typed/echo", "/byte-arrays/typed-chars/echo"} {
		echo := byPath[path]
		if echo == nil || echo.RequestBody == nil || echo.Response == nil {
			t.Errorf("%s = %+v, want a request body and a response", path, echo)
			continue
		}
		if got := arrayFieldKind(echo.RequestBody.Body); got != "string" {
			t.Errorf("%s request body = %s, want string", path, got)
		}
		if got := arrayFieldKind(byteArrayModelAt(echo.Response.Body, "data")); got != "string" {
			t.Errorf("%s response data = %s, want string", path, got)
		}
	}

	if unresolved := c.Unresolved(); len(unresolved) != 0 {
		t.Errorf("unresolved = %v, want none", unresolved)
	}
}

// The binding view reads the same bindings and keeps a byte or char array an
// array, down to the field two generic classes deep.
func TestByteArraysThroughGenericClassesWhenBound(t *testing.T) {
	c, byPath := collectByteArrays(t)

	want := map[string]string{
		"wrapped.data":   "array",
		"wrapped.items":  "array",
		"wrapped.grid":   "array",
		"wrapped.byName": "map",
		"wrappedItems":   "array",
		"direct":         "array",
	}
	for _, path := range []string{"/byte-arrays/bound-outer", "/byte-arrays/bound-outer-chars"} {
		bound := byPath[path]
		if bound == nil {
			t.Errorf("%s endpoint missing: %v", path, byPath)
			continue
		}
		got := map[string]string{}
		for _, p := range bound.Parameters {
			got[p.Name] = p.TypeName
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("%s parameters:\n got  %v\n want %v", path, got, want)
		}
	}

	if unresolved := c.Unresolved(); len(unresolved) != 0 {
		t.Errorf("unresolved = %v, want none", unresolved)
	}
}

// byteArrayVarargExpectation is, per declaration of testdata/byte-array that
// has an explicit form and a `vararg` form, the kind both export as. Kotlin
// types `vararg n: Byte` as a ByteArray and `vararg c: Char` as a CharArray, the
// arrays the explicit forms are, and `vararg n: Byte?` as an `Array` of boxed
// values, as `Array<Byte?>` is. The Java collector asserts the same table for
// `byte...` and `Byte...` (api-collector-java/byte_array_test.go): keep the two
// in sync.
var byteArrayVarargExpectation = map[string]string{
	"bytes":       "string",
	"chars":       "string",
	"boxed-bytes": "array<int>",
	"boxed-chars": "array<string>",
	"ints":        "array<int>",
}

// A vararg of a primitive type is exported as the primitive array it is: the
// body of a handler that takes `vararg Byte` is the string a ByteArray is, and
// `vararg Byte?` is the array of numbers `Array<Byte?>` is.
func TestVarargsOfBytesAndCharsAreExportedAsThePrimitiveArraysTheyAre(t *testing.T) {
	c, byPath := collectByteArrays(t)

	for name, kind := range byteArrayVarargExpectation {
		for _, form := range []string{"explicit", "vararg"} {
			path := "/byte-arrays/" + form + "/" + name
			endpoint := byPath[path]
			if endpoint == nil || endpoint.RequestBody == nil || endpoint.Response == nil {
				t.Errorf("%s = %+v, want a request body and a response", path, endpoint)
				continue
			}
			if got := arrayFieldKind(endpoint.RequestBody.Body); got != kind {
				t.Errorf("%s request body = %s, want %s", path, got, kind)
			}
			if got := arrayFieldKind(endpoint.Response.Body); got != kind {
				t.Errorf("%s response = %s, want %s", path, got, kind)
			}
		}
	}

	if unresolved := c.Unresolved(); len(unresolved) != 0 {
		t.Errorf("unresolved = %v, want none", unresolved)
	}
}

// A request parameter of a vararg is listed as the one declared with the array
// the vararg is: the same type name and the same binding.
func TestVarargsOfBytesAndCharsAreListedAsThePrimitiveArraysTheyAre(t *testing.T) {
	_, byPath := collectByteArrays(t)

	wantType := map[string]string{
		"bytes":       "ByteArray",
		"chars":       "CharArray",
		"boxed-bytes": "Array<Byte>",
		"boxed-chars": "Array<Char>",
	}
	for name, typeName := range wantType {
		explicit := byPath["/byte-arrays/explicit-query/"+name]
		vararg := byPath["/byte-arrays/vararg-query/"+name]
		if explicit == nil || vararg == nil || len(explicit.Parameters) != 1 || len(vararg.Parameters) != 1 {
			t.Errorf("%s: explicit = %+v, vararg = %+v, want one parameter each", name, explicit, vararg)
			continue
		}
		if got := vararg.Parameters[0]; got.Name != "data" || got.In != "query" || got.TypeName != typeName {
			t.Errorf("%s: vararg parameter = %+v, want data in query as %s", name, got, typeName)
		}
		if !reflect.DeepEqual(vararg.Parameters[0], explicit.Parameters[0]) {
			t.Errorf("%s: vararg parameter:\n got  %+v\n want %+v", name, vararg.Parameters[0], explicit.Parameters[0])
		}
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
