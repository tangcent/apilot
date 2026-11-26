package parser

import (
	"reflect"
	"testing"
)

// A function records its type parameters, reified and the upper bounds, both
// the one written after the colon and those of a where clause. A function
// that declares none has none.
func TestParse_AFunctionRecordsItsTypeParameters(t *testing.T) {
	result := parseKotlin(t, `package a.b

class Handler {
    fun plain(data: Byte) {}
    fun <Byte> bytes(vararg data: Byte) {}
    fun <reified Char> letters(vararg data: Char) {}
    fun <Byte : Number> bounded(vararg data: Byte) {}
    fun <Byte> whereBound(vararg data: Byte) where Byte : Number {}
    fun <T : CharSequence, Byte : Comparable<Byte>> multi(vararg data: Byte) {}
    fun <T> several(data: T) where T : CharSequence, T : Comparable<T> {}
    inline fun <reified Byte : Number> reifiedBound(vararg data: Byte) {}
}
`)
	type shape struct {
		Reified bool
		Bounds  []string
	}
	got := map[string][]shape{}
	for _, fn := range result.Classes[0].Methods {
		var params []shape
		for _, param := range fn.TypeParameters {
			params = append(params, shape{param.Reified, param.Bounds})
			if param.Name == "" {
				t.Errorf("%s has a type parameter with no name", fn.Name)
			}
		}
		got[fn.Name] = params
	}
	want := map[string][]shape{
		"plain":        nil,
		"bytes":        {{false, nil}},
		"letters":      {{true, nil}},
		"bounded":      {{false, []string{"Number"}}},
		"whereBound":   {{false, []string{"Number"}}},
		"multi":        {{false, []string{"CharSequence"}}, {false, []string{"Comparable<Byte>"}}},
		"several":      {{false, []string{"CharSequence", "Comparable<T>"}}},
		"reifiedBound": {{true, []string{"Number"}}},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("type parameters = %#v, want %#v", got, want)
	}
	names := map[string][]string{}
	for _, fn := range result.Classes[0].Methods {
		names[fn.Name] = fn.TypeParameterNames()
	}
	wantNames := map[string][]string{
		"plain":        nil,
		"bytes":        {"Byte"},
		"letters":      {"Char"},
		"bounded":      {"Byte"},
		"whereBound":   {"Byte"},
		"multi":        {"T", "Byte"},
		"several":      {"T"},
		"reifiedBound": {"Byte"},
	}
	if !reflect.DeepEqual(names, wantNames) {
		t.Errorf("names = %#v, want %#v", names, wantNames)
	}
}
