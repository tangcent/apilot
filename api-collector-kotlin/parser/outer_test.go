package parser

import (
	"reflect"
	"testing"
)

func parseKotlin(t *testing.T, source string) *ParseResult {
	t.Helper()
	p, err := NewParser()
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()
	result, err := p.ParseSource("Sample.kt", []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	if result.Error != nil {
		t.Fatal(result.Error)
	}
	return result
}

// A class nested in another is a declaration of its own, and Outer says where it
// belongs; a private declaration is marked, since at the top level that makes it
// visible in its file only.
func TestParse_ANestedClassRecordsItsOuterAndAPrivateClassIsMarked(t *testing.T) {
	result := parseKotlin(t, `package a.b

class Holder {
    class Int(val own: kotlin.Int)
    class Mid {
        enum class Deep { A }
    }
}

private class Hidden

internal class Shown
`)
	got := map[string][2]any{}
	for _, class := range result.Classes {
		got[class.Name] = [2]any{class.Outer, class.IsPrivate}
	}
	want := map[string][2]any{
		"Holder": {"", false},
		"Int":    {"Holder", false},
		"Mid":    {"Holder", false},
		"Deep":   {"Holder.Mid", false},
		"Hidden": {"", true},
		"Shown":  {"", false},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("classes = %v, want %v", got, want)
	}
}

// A typealias is read with its name, its type parameters and the type it names,
// which is read in the scope of the file that declares it.
func TestParse_ATypeAliasIsRead(t *testing.T) {
	result := parseKotlin(t, `package a.b

import c.d.Item

typealias Number = Long
typealias Ids<T> = List<T>
private typealias Dto = Item
typealias Handler = (String) -> Unit
typealias Qualified = c.d.Item?
`)
	type shape struct {
		Target  string
		Params  []string
		Private bool
	}
	got := map[string]shape{}
	for _, alias := range result.TypeAliases {
		if alias.Package != "a.b" || alias.FilePath != "Sample.kt" || len(alias.Imports) != 1 {
			t.Errorf("alias %s = %+v, want its package, file and imports", alias.Name, alias)
		}
		got[alias.Name] = shape{alias.Target, alias.TypeParameters, alias.IsPrivate}
	}
	want := map[string]shape{
		"Number":    {"Long", nil, false},
		"Ids":       {"List<T>", []string{"T"}, false},
		"Dto":       {"Item", nil, true},
		"Handler":   {"(String)->Unit", nil, false},
		"Qualified": {"c.d.Item", nil, false},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("aliases = %v, want %v", got, want)
	}
	if len(result.Classes) != 0 {
		t.Errorf("classes = %v, want none: an alias declares no class", result.Classes)
	}
}

// The return type of a function written without one and with a block body is
// Unit by the language. It is spelled as the class of the standard library,
// since no file wrote it: a class of the package called Unit is not it.
func TestParse_AFunctionWithoutAReturnTypeReturnsTheStandardUnit(t *testing.T) {
	result := parseKotlin(t, `package a

class C {
    fun block() { }
    fun declared(): Unit { }
    fun expression() = 1
}
`)
	got := map[string][2]any{}
	for _, fn := range result.Classes[0].Methods {
		got[fn.Name] = [2]any{fn.ReturnType, fn.HasDeclaredReturn}
	}
	want := map[string][2]any{
		"block":      {"kotlin.Unit", false},
		"declared":   {"Unit", true},
		"expression": {"", false},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("functions = %v, want %v", got, want)
	}
}
