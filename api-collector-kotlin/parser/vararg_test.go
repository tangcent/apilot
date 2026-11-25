package parser

import (
	"reflect"
	"testing"
)

// A typealias records whether its target is nullable as a whole: Target drops
// the `?`, and a vararg of the alias is an array of boxed values only when the
// target is nullable (see resolver.ParameterType).
func TestParse_ATypeAliasRecordsWhetherItsTargetIsNullable(t *testing.T) {
	result := parseKotlin(t, `package a.b

typealias Octet = Byte
typealias MaybeByte = Byte?
typealias Parenthesized = (Byte)?
typealias Inside = (Byte?)
typealias Qualified = kotlin.Byte
typealias QualifiedMaybe = kotlin.Byte?
typealias Ids<T> = List<T>
typealias Maybes<T> = List<T?>
typealias MaybeIds<T> = List<T>?
typealias Handler = (String) -> Unit
typealias Result = (String) -> Byte?
typealias MaybeHandler = ((String) -> Unit)?
`)
	type shape struct {
		Target   string
		Nullable bool
	}
	got := map[string]shape{}
	for _, alias := range result.TypeAliases {
		got[alias.Name] = shape{alias.Target, alias.TargetNullable}
	}
	want := map[string]shape{
		"Octet":          {"Byte", false},
		"MaybeByte":      {"Byte", true},
		"Parenthesized":  {"Byte", true},
		"Inside":         {"Byte", true},
		"Qualified":      {"kotlin.Byte", false},
		"QualifiedMaybe": {"kotlin.Byte", true},
		"Ids":            {"List<T>", false},
		"Maybes":         {"List<T>", false},
		"MaybeIds":       {"List<T>", true},
		"Handler":        {"(String)->Unit", false},
		"Result":         {"(String)->Byte", false},
		"MaybeHandler":   {"(String)->Unit", true},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("aliases = %v, want %v", got, want)
	}
}

// A vararg parameter of a function is the array it is received as: the element
// being nullable is kept, and the array itself is never null.
func TestParse_AVarargParameterIsItsArray(t *testing.T) {
	result := parseKotlin(t, `package a.b

class Handler {
    fun bytes(vararg data: Byte) {}
    fun boxed(vararg data: Byte?) {}
    fun octets(vararg data: Octet) {}
    fun plain(data: Byte?) {}
}
`)
	type shape struct {
		Type            string
		Nullable        bool
		Vararg          bool
		ElementNullable bool
	}
	got := map[string]shape{}
	for _, fn := range result.Classes[0].Methods {
		p := fn.Parameters[0]
		got[fn.Name] = shape{p.Type, p.Nullable, p.Vararg, p.ElementNullable}
	}
	want := map[string]shape{
		"bytes":  {"Array<Byte>", false, true, false},
		"boxed":  {"Array<Byte>", false, true, true},
		"octets": {"Array<Octet>", false, true, false},
		"plain":  {"Byte", true, false, false},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("parameters = %v, want %v", got, want)
	}
}
