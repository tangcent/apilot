package kotlincollector

import (
	"reflect"
	"testing"
)

// byteScopeScenario is one vararg of testdata/byte-array whose element is a
// type parameter of the function, or a name that parameter does not hide, and
// the explicit declaration beside it. The two export the same kind and, as a
// request parameter, the same listing. A type parameter keeps the vararg an
// Array of that name; it does not become the primitive array.
type byteScopeScenario struct {
	base, name, kind, typeName string
}

// The first group is the parameter hiding Byte, Char, Int or UByte, including
// a bound, a where clause, reified and a second parameter. The rest is what
// that parameter hides (an alias, an import) and what it does not.
var byteScopeScenarios = []byteScopeScenario{
	{"/byte-scope", "byte", "array<int>", "Array<Byte>"},
	{"/byte-scope", "char", "array<string>", "Array<Char>"},
	{"/byte-scope", "int", "array<int>", "Array<Int>"},
	{"/byte-scope", "ubyte", "array<int>", "Array<UByte>"},
	{"/byte-scope", "bounded", "array<int>", "Array<Byte>"},
	{"/byte-scope", "where", "array<int>", "Array<Byte>"},
	{"/byte-scope", "reified", "array<int>", "Array<Byte>"},
	{"/byte-scope", "multi", "array<int>", "Array<Byte>"},
	{"/byte-scope", "shadow-alias", "array<int>", "Array<Octet>"},
	{"/byte-scope", "alias-target", "string", "ByteArray"},
	{"/byte-scope", "qualified", "string", "ByteArray"},
	{"/byte-scope", "beside", "string", "CharArray"},
	{"/byte-scope-import", "project-byte", "array<object{code}>", "Array<Byte>"},
	{"/byte-scope-import", "project-plain", "array<object{code}>", "Array<Byte>"},
	{"/byte-scope-import", "plain-import", "array<int>", "Array<PlainOctet>"},
	{"/byte-scope-import", "renamed", "array<int>", "Array<Renamed>"},
	{"/byte-scope-import", "renamed-kept", "string", "ByteArray"},
	{"/byte-scope-star", "star-octet", "array<int>", "Array<StarOctet>"},
	{"/byte-scope-star", "star-kept", "string", "ByteArray"},
	{"/byte-scope-star", "star-letter", "array<string>", "Array<StarLetter>"},
	{"/byte-scope-star", "star-other", "string", "CharArray"},
}

func TestAMethodTypeParameterKeepsAVarargBoxed(t *testing.T) {
	c, byPath := collectByteArrays(t)

	for _, scenario := range byteScopeScenarios {
		scenario := scenario
		t.Run(scenario.base+"/"+scenario.name+"/body", func(t *testing.T) {
			for _, form := range []string{"explicit", "vararg"} {
				path := scenario.base + "/" + form + "/" + scenario.name
				endpoint := byPath[path]
				if endpoint == nil || endpoint.RequestBody == nil || endpoint.Response == nil {
					t.Errorf("%s = %+v, want a request body and a response", path, endpoint)
					continue
				}
				if got := arrayFieldKind(endpoint.RequestBody.Body); got != scenario.kind {
					t.Errorf("%s request body = %s, want %s", path, got, scenario.kind)
				}
				if got := arrayFieldKind(endpoint.Response.Body); got != scenario.kind {
					t.Errorf("%s response = %s, want %s", path, got, scenario.kind)
				}
			}
		})
		t.Run(scenario.base+"/"+scenario.name+"/query", func(t *testing.T) {
			explicit := byPath[scenario.base+"/explicit-query/"+scenario.name]
			vararg := byPath[scenario.base+"/vararg-query/"+scenario.name]
			if explicit == nil || vararg == nil || len(explicit.Parameters) != 1 || len(vararg.Parameters) != 1 {
				t.Fatalf("%s: explicit = %+v, vararg = %+v, want one parameter each", scenario.name, explicit, vararg)
			}
			got, want := vararg.Parameters[0], explicit.Parameters[0]
			if got.Name != "data" || got.In != "query" || got.TypeName != scenario.typeName {
				t.Errorf("%s: vararg parameter = %+v, want data in query as %s", scenario.name, got, scenario.typeName)
			}
			got.TypeName, want.TypeName = "", ""
			if !reflect.DeepEqual(got, want) {
				t.Errorf("%s: vararg parameter:\n got  %+v\n want %+v", scenario.name, got, want)
			}
		})
	}

	if unresolved := c.Unresolved(); len(unresolved) != 0 {
		t.Errorf("unresolved = %v, want none", unresolved)
	}
}

// A function that declares no type parameter is not affected by one declared
// on another function of the same class.
func TestAMethodWithoutATypeParameterIsStillThePrimitiveArray(t *testing.T) {
	_, byPath := collectByteArrays(t)

	endpoint := byPath["/byte-scope/leak/plain"]
	if endpoint == nil || endpoint.RequestBody == nil || endpoint.Response == nil {
		t.Fatalf("leak = %+v, want a request body and a response", endpoint)
	}
	if got := arrayFieldKind(endpoint.RequestBody.Body); got != "string" {
		t.Errorf("request body = %s, want string", got)
	}
	if got := arrayFieldKind(endpoint.Response.Body); got != "string" {
		t.Errorf("response = %s, want string", got)
	}
}

// A type parameter on a parameter that is not a vararg is out of scope: Byte
// is still kotlin.Byte, the same as a function that does not declare it.
func TestAMethodTypeParameterOnAValueIsLeftAsItWas(t *testing.T) {
	_, byPath := collectByteArrays(t)

	typed := byPath["/byte-scope/plain-param/typed"]
	plain := byPath["/byte-scope/plain-param/plain"]
	if typed == nil || plain == nil || typed.RequestBody == nil || plain.RequestBody == nil {
		t.Fatalf("typed = %+v, plain = %+v, want a request body each", typed, plain)
	}
	if got, want := arrayFieldKind(typed.RequestBody.Body), arrayFieldKind(plain.RequestBody.Body); got != want {
		t.Errorf("typed request body = %s, want %s as the plain one is", got, want)
	}
	if got, want := arrayFieldKind(typed.Response.Body), arrayFieldKind(plain.Response.Body); got != want {
		t.Errorf("typed response = %s, want %s as the plain one is", got, want)
	}
	if got := arrayFieldKind(typed.RequestBody.Body); got != "int" {
		t.Errorf("typed request body = %s, want int", got)
	}
}

// A typealias of ByteArray, used as a body or a field rather than as a vararg
// element, is the string a ByteArray is.
func TestATypealiasOfByteArrayIsAString(t *testing.T) {
	c, byPath := collectByteArrays(t)

	body := byPath["/byte-scope/blob"]
	if body == nil || body.RequestBody == nil || body.Response == nil {
		t.Fatalf("blob = %+v, want a request body and a response", body)
	}
	if got := arrayFieldKind(body.RequestBody.Body); got != "string" {
		t.Errorf("blob request body = %s, want string", got)
	}
	if got := arrayFieldKind(body.Response.Body); got != "string" {
		t.Errorf("blob response = %s, want string", got)
	}

	field := byPath["/byte-scope/blob-field"]
	if field == nil || field.RequestBody == nil || field.Response == nil {
		t.Fatalf("blob field = %+v, want a request body and a response", field)
	}
	want := map[string]string{"data": "string"}
	if got := arrayFieldKinds(field.RequestBody.Body); !reflect.DeepEqual(got, want) {
		t.Errorf("blob field request body = %v, want %v", got, want)
	}
	if got := arrayFieldKinds(field.Response.Body); !reflect.DeepEqual(got, want) {
		t.Errorf("blob field response = %v, want %v", got, want)
	}

	if unresolved := c.Unresolved(); len(unresolved) != 0 {
		t.Errorf("unresolved = %v, want none", unresolved)
	}
}

// A type parameter of the class, of an outer class read by an inner class, and
// of a class whose primary constructor holds a vararg property, already hides
// the primitive type. A nested class that is not inner does not see it.
func TestAClassTypeParameterStillHidesAVararg(t *testing.T) {
	c, byPath := collectByteArrays(t)

	for _, base := range []string{"/byte-scope/class", "/byte-scope/inner"} {
		explicit := byPath[base+"/explicit"]
		vararg := byPath[base+"/vararg"]
		if explicit == nil || vararg == nil || explicit.RequestBody == nil || vararg.RequestBody == nil {
			t.Errorf("%s: explicit = %+v, vararg = %+v, want a request body each", base, explicit, vararg)
			continue
		}
		got := arrayFieldKind(vararg.RequestBody.Body)
		if got == "string" {
			t.Errorf("%s request body = string, want the boxed array", base)
		}
		if want := arrayFieldKind(explicit.RequestBody.Body); got != want {
			t.Errorf("%s request body = %s, want %s", base, got, want)
		}
		if got, want := arrayFieldKind(vararg.Response.Body), arrayFieldKind(explicit.Response.Body); got != want {
			t.Errorf("%s response = %s, want %s", base, got, want)
		}
	}

	nested := byPath["/byte-scope/nested/vararg"]
	if nested == nil || nested.RequestBody == nil {
		t.Fatalf("nested = %+v, want a request body", nested)
	}
	if got := arrayFieldKind(nested.RequestBody.Body); got != "string" {
		t.Errorf("nested request body = %s, want string", got)
	}

	ctorExplicit := byPath["/byte-scope/ctor/explicit"]
	ctorVararg := byPath["/byte-scope/ctor/vararg"]
	if ctorExplicit == nil || ctorVararg == nil || ctorExplicit.RequestBody == nil || ctorVararg.RequestBody == nil {
		t.Fatalf("ctor explicit = %+v, vararg = %+v, want a request body each", ctorExplicit, ctorVararg)
	}
	gotFields := arrayFieldKinds(ctorVararg.RequestBody.Body)
	wantFields := arrayFieldKinds(ctorExplicit.RequestBody.Body)
	if !reflect.DeepEqual(gotFields, wantFields) {
		t.Errorf("ctor fields:\n got  %v\n want %v", gotFields, wantFields)
	}
	// Bound to String, the property is an array of strings. A ByteArray would
	// ignore the argument and be a string.
	if gotFields["data"] != "array<string>" {
		t.Errorf("ctor data = %q, want array<string>", gotFields["data"])
	}

	if unresolved := c.Unresolved(); len(unresolved) != 0 {
		t.Errorf("unresolved = %v, want none", unresolved)
	}
}
