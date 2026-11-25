package kotlincollector

import (
	"reflect"
	"testing"
)

// byteArrayAliasScenarios is, per vararg of testdata/byte-array that takes a
// typealias, the kind it and its explicit twin export as and the type name the
// vararg parameter is listed under. A vararg of an alias is what a vararg of the
// type the alias stands for is: `vararg Octet` of `typealias Octet = Byte` is a
// ByteArray, as `vararg Byte` is, and one whose alias is nullable at any level of
// a chain, or whose element is written nullable, is an `Array` of boxed values.
// An alias that stands for something that is not a primitive type is the `Array`
// of it, and is listed as written. The explicit twin is declared with the type
// the compiler gives the vararg.
var byteArrayAliasScenarios = []struct {
	name, kind, typeName string
}{
	{"octet", "string", "ByteArray"},
	{"letter", "string", "CharArray"},
	{"whole", "array<int>", "IntArray"},
	{"ubyte", "array<int>", "UByteArray"},
	{"maybe-byte", "array<int>", "Array<MaybeByte>"},
	{"maybe-char", "array<string>", "Array<MaybeChar>"},
	{"octet-nullable", "array<int>", "Array<Octet>"},
	{"chain", "string", "ByteArray"},
	{"chain-nullable", "array<int>", "Array<Top>"},
	{"qualified", "string", "ByteArray"},
	{"generic", "string", "ByteArray"},
	{"generic-nullable", "array<int>", "Array<MaybeConst<String>>"},
	{"private", "string", "ByteArray"},
	{"plain-import", "string", "ByteArray"},
	{"renamed-import", "string", "ByteArray"},
	{"star-import", "string", "ByteArray"},
	{"star-letter", "string", "CharArray"},
	{"other-nullable", "array<int>", "Array<OtherMaybe>"},
	{"text", "array<string>", "Array<Text>"},
	{"blob", "array<string>", "Array<Blob>"},
	{"shadowed", "array<object{code}>", "Array<Shadowed>"},
}

// A vararg of a typealias is exported as the array the alias makes it: the body
// of a handler that takes `vararg Octet` is the string a ByteArray is, and
// `vararg MaybeByte` the array of numbers `Array<Byte?>` is.
func TestVarargsOfTypealiasesAreExportedAsTheArraysTheyStandFor(t *testing.T) {
	c, byPath := collectByteArrays(t)

	for _, scenario := range byteArrayAliasScenarios {
		for _, form := range []string{"explicit", "vararg"} {
			path := "/byte-aliases/" + form + "/" + scenario.name
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
	}

	if unresolved := c.Unresolved(); len(unresolved) != 0 {
		t.Errorf("unresolved = %v, want none", unresolved)
	}
}

// A request parameter of a vararg of a typealias is listed as the one declared
// with the array the vararg is, but for the type name of an alias that is no
// primitive array, which is as written.
func TestVarargsOfTypealiasesAreListedAsTheArraysTheyStandFor(t *testing.T) {
	_, byPath := collectByteArrays(t)

	for _, scenario := range byteArrayAliasScenarios {
		explicit := byPath["/byte-aliases/explicit-query/"+scenario.name]
		vararg := byPath["/byte-aliases/vararg-query/"+scenario.name]
		if explicit == nil || vararg == nil || len(explicit.Parameters) != 1 || len(vararg.Parameters) != 1 {
			t.Errorf("%s: explicit = %+v, vararg = %+v, want one parameter each", scenario.name, explicit, vararg)
			continue
		}
		got, want := vararg.Parameters[0], explicit.Parameters[0]
		if got.Name != "data" || got.In != "query" || got.TypeName != scenario.typeName {
			t.Errorf("%s: vararg parameter = %+v, want data in query as %s", scenario.name, got, scenario.typeName)
		}
		got.TypeName, want.TypeName = "", ""
		if !reflect.DeepEqual(got, want) {
			t.Errorf("%s: vararg parameter:\n got  %+v\n want %+v", scenario.name, got, want)
		}
	}
}

// A typealias that is not a vararg is the type it stands for, as it was: an alias
// of Byte is a number and one of Char a string, nullable or not.
func TestTypealiasesThatAreNotVarargsAreExportedAsTheTypeTheyStandFor(t *testing.T) {
	c, byPath := collectByteArrays(t)

	for _, pair := range [][2]string{
		{"octet", "byte"},
		{"maybe-byte", "nullable-byte"},
		{"letter", "char"},
	} {
		alias, plain := byPath["/byte-aliases/plain/"+pair[0]], byPath["/byte-aliases/plain/"+pair[1]]
		if alias == nil || plain == nil || alias.RequestBody == nil || plain.RequestBody == nil {
			t.Errorf("%s: alias = %+v, plain = %+v, want a request body each", pair[0], alias, plain)
			continue
		}
		if got, want := arrayFieldKind(alias.RequestBody.Body), arrayFieldKind(plain.RequestBody.Body); got != want {
			t.Errorf("%s request body = %s, want %s as %s is", pair[0], got, want, pair[1])
		}
		if got, want := arrayFieldKind(alias.Response.Body), arrayFieldKind(plain.Response.Body); got != want {
			t.Errorf("%s response = %s, want %s as %s is", pair[0], got, want, pair[1])
		}
	}
	if got := arrayFieldKind(byPath["/byte-aliases/plain/octet"].RequestBody.Body); got != "int" {
		t.Errorf("octet request body = %s, want int", got)
	}
	if got := arrayFieldKind(byPath["/byte-aliases/plain/letter"].RequestBody.Body); got != "string" {
		t.Errorf("letter request body = %s, want string", got)
	}

	if unresolved := c.Unresolved(); len(unresolved) != 0 {
		t.Errorf("unresolved = %v, want none", unresolved)
	}
}
