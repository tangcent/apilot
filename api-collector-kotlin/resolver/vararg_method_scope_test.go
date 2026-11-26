package resolver

import (
	"testing"

	ktparser "github.com/tangcent/apilot/api-collector-kotlin/parser"
)

// A type parameter of the function is the innermost scope of a vararg element.
// It hides a primitive type, an alias and a class of the project, and only in
// that function. The target of an alias is read where the alias is declared,
// so a parameter called Byte does not change an alias of Byte.
func TestParameterType_AMethodTypeParameterHidesTheName(t *testing.T) {
	byteClass := scopeShadow(varargPackage, "Byte", "own")
	r := NewWithAliases([]ktparser.Class{byteClass}, nil, varargAliases())
	unit := varargAliasUnit()
	imported := &ktparser.Class{
		Name: "Handler", Package: "com.example.kcvarargapp", FilePath: "app/Handler.kt",
		Imports: []ktparser.Import{{Path: varargAliasPackage + ".Octet"}},
	}
	starred := &ktparser.Class{
		Name: "Handler", Package: "com.example.kcvarargapp", FilePath: "app/Handler.kt",
		Imports: []ktparser.Import{{Path: varargAliasPackage, OnDemand: true}},
	}
	renamed := &ktparser.Class{
		Name: "Handler", Package: "com.example.kcvarargapp", FilePath: "app/Handler.kt",
		Imports: []ktparser.Import{{Path: varargAliasPackage + ".Octet", Alias: "Renamed"}},
	}
	project := &ktparser.Class{Name: "Handler", Package: varargPackage, FilePath: "vararg/Handler.kt"}

	tests := []struct {
		name, element, want string
		unit                *ktparser.Class
		params              []string
	}{
		{"Byte", "Byte", "Array<Byte>", unit, []string{"Byte"}},
		{"Char", "Char", "Array<Char>", unit, []string{"Char"}},
		{"Int", "Int", "Array<Int>", unit, []string{"Int"}},
		{"UByte", "UByte", "Array<UByte>", unit, []string{"UByte"}},
		{"a second parameter", "Byte", "Array<Byte>", unit, []string{"T", "Byte"}},
		{"the other parameter", "T", "Array<T>", unit, []string{"T", "Byte"}},
		{"does not hide Char", "Char", "CharArray", unit, []string{"Byte"}},
		{"a qualified name", "kotlin.Byte", "ByteArray", unit, []string{"Byte"}},
		{"an alias of the same name", "Octet", "Array<Octet>", unit, []string{"Octet"}},
		{"an alias whose target is the parameter", "Octet", "ByteArray", unit, []string{"Byte"}},
		{"another alias is not hidden", "Letter", "CharArray", unit, []string{"Octet"}},
		{"an explicit import", "Octet", "Array<Octet>", imported, []string{"Octet"}},
		{"a star import", "Octet", "Array<Octet>", starred, []string{"Octet"}},
		{"a star import of another name", "Letter", "CharArray", starred, []string{"Octet"}},
		{"a renaming import", "Renamed", "Array<Renamed>", renamed, []string{"Renamed"}},
		{"the old name beside a renaming import", "Octet", "Array<Octet>", renamed, []string{"Renamed"}},
		{"a class of the project", "Byte", "Array<Byte>", project, []string{"Byte"}},
		{"no parameter, the class stays", "Byte", "Array<Byte>", project, nil},
		{"no parameter, Byte is the primitive", "Byte", "ByteArray", unit, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := r.ParameterType(tt.unit, varargParameter(tt.element), tt.params); got != tt.want {
				t.Errorf("ParameterType(vararg %s, %v) = %q, want %q", tt.element, tt.params, got, tt.want)
			}
		})
	}

	// A parameter that is not a vararg is the type it is written with. The
	// function's type parameters are not applied to it.
	plain := ktparser.Parameter{Name: "body", Type: "Byte"}
	if got := r.ParameterType(unit, plain, []string{"Byte"}); got != "Byte" {
		t.Errorf("ParameterType(Byte) = %q, want Byte", got)
	}
}

// An inner class sees the type parameters of the class around it. A nested
// class that is not inner does not, and a vararg property has no function
// type parameters of its own.
func TestParameterType_AnInnerClassSeesTheOuterTypeParameter(t *testing.T) {
	outer := varargUnit()
	outer.Name = "Outer"
	outer.TypeParameters = []string{"Byte"}
	inner := varargUnit()
	inner.Name = "Inner"
	inner.Outer = "Outer"
	inner.IsInner = true
	nested := varargUnit()
	nested.Name = "Nested"
	nested.Outer = "Outer"
	classes := []ktparser.Class{*outer, *inner, *nested}
	r := New(classes)

	if got := r.ParameterType(&classes[1], varargParameter("Byte"), nil); got != "Array<Byte>" {
		t.Errorf("inner vararg Byte = %q, want Array<Byte>", got)
	}
	if got := r.ParameterType(&classes[1], varargParameter("Char"), nil); got != "CharArray" {
		t.Errorf("inner vararg Char = %q, want CharArray", got)
	}
	if got := r.ParameterType(&classes[2], varargParameter("Byte"), nil); got != "ByteArray" {
		t.Errorf("nested vararg Byte = %q, want ByteArray", got)
	}
	// The function's parameter is in front of the class's.
	if got := r.ParameterType(&classes[1], varargParameter("Byte"), []string{"T"}); got != "Array<Byte>" {
		t.Errorf("inner vararg Byte beside <T> = %q, want Array<Byte>", got)
	}
	if got := r.ParameterType(&classes[1], varargParameter("T"), []string{"T"}); got != "Array<T>" {
		t.Errorf("inner vararg T = %q, want Array<T>", got)
	}
}
