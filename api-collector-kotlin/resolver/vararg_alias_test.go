package resolver

import (
	"fmt"
	"testing"

	ktparser "github.com/tangcent/apilot/api-collector-kotlin/parser"
)

// A typealias is the type it stands for, so a vararg of one is what a vararg of
// that type is: `vararg Octet` of `typealias Octet = Byte` is a ByteArray, where
// `vararg Octet?` and a vararg of an alias whose target is nullable, at any level
// of a chain of aliases, are an `Array` of boxed values. The alias is read where
// it is declared: what `Byte` denotes in its file, not in the file that uses it.

const (
	varargAliasPackage  = "com.example.kcvarargalias"
	varargShadowPackage = "com.example.kcvarargshadow"
)

func varargAlias(name, target string, nullable bool, params ...string) ktparser.TypeAlias {
	return ktparser.TypeAlias{
		Name: name, Package: varargAliasPackage, FilePath: "alias/Aliases.kt",
		Target: target, TargetNullable: nullable, TypeParameters: params,
	}
}

func varargAliases() []ktparser.TypeAlias {
	aliases := []ktparser.TypeAlias{
		varargAlias("Octet", "Byte", false),
		varargAlias("Letter", "Char", false),
		varargAlias("Whole", "Int", false),
		varargAlias("UOctet", "UByte", false),
		varargAlias("MaybeByte", "Byte", true),
		varargAlias("MaybeChar", "Char", true),
		varargAlias("Octet2", "Octet", false),
		varargAlias("Octet3", "Octet2", false),
		varargAlias("Bottom", "Byte", false),
		varargAlias("Middle", "Bottom", true),
		varargAlias("Top", "Middle", false),
		varargAlias("QualifiedByte", "kotlin.Byte", false),
		varargAlias("Const", "Byte", false, "T"),
		varargAlias("MaybeConst", "Byte", true, "T"),
		varargAlias("Via", "Const<Byte>", false),
		varargAlias("ViaMaybe", "MaybeConst<Byte>", false),
		varargAlias("Text", "String", false),
		varargAlias("Blob", "ByteArray", false),
		varargAlias("Bytes", "List<Byte>", false),
		varargAlias("Handler", "(Byte)->Unit", false),
		varargAlias("LoopA", "LoopB", false),
		varargAlias("LoopB", "LoopA", false),
		varargAlias("Self", "Self", false),
		{Name: "Secret", Package: varargAliasPackage, FilePath: "alias/Secret.kt", Target: "Byte", IsPrivate: true},
		// The file imports the class of the JDK, which is what Byte denotes there.
		{
			Name: "JavaByte", Package: varargAliasPackage, FilePath: "alias/Java.kt", Target: "Byte",
			Imports: []ktparser.Import{{Path: "java.lang.Byte"}},
		},
		// The package of the file declares a class that is called Byte.
		{Name: "Shadowed", Package: varargShadowPackage, FilePath: "shadow/Aliases.kt", Target: "Byte"},
	}
	// A chain longer than the walk follows is left as written.
	for i := 0; i < 12; i++ {
		target := "Byte"
		if i > 0 {
			target = fmt.Sprintf("Link%d", i-1)
		}
		aliases = append(aliases, varargAlias(fmt.Sprintf("Link%d", i), target, false))
	}
	return aliases
}

func varargAliasResolver() *Resolver {
	return NewWithAliases([]ktparser.Class{scopeShadow(varargShadowPackage, "Byte", "own")}, nil, varargAliases())
}

// varargAliasUnit is a file of the package of the aliases, so it sees them
// without an import.
func varargAliasUnit() *ktparser.Class {
	return &ktparser.Class{Name: "Handler", Package: varargAliasPackage, FilePath: "alias/Handler.kt"}
}

func TestParameterType_AVarargOfATypealiasIsTheArrayItsTargetMakes(t *testing.T) {
	tests := []struct{ element, want string }{
		{"Octet", "ByteArray"},
		{"Letter", "CharArray"},
		{"Whole", "IntArray"},
		{"UOctet", "UByteArray"},
		// A chain of aliases.
		{"Octet2", "ByteArray"},
		{"Octet3", "ByteArray"},
		{"Link5", "ByteArray"},
		// A target written with its package.
		{"QualifiedByte", "ByteArray"},
		// The alias written with its package.
		{varargAliasPackage + ".Octet", "ByteArray"},
		// The arguments of a generic alias are of no matter to the type it stands for.
		{"Const<String>", "ByteArray"},
		{"Const<Byte?>", "ByteArray"},
		{"Const<List<String>>", "ByteArray"},
		{"Via", "ByteArray"},
		// A target that is nullable, as a whole or at one level of a chain.
		{"MaybeByte", "Array<MaybeByte>"},
		{"MaybeChar", "Array<MaybeChar>"},
		{"Middle", "Array<Middle>"},
		{"Top", "Array<Top>"},
		{"MaybeConst<String>", "Array<MaybeConst<String>>"},
		{"ViaMaybe", "Array<ViaMaybe>"},
		// An alias of a type that is not a primitive type of Kotlin.
		{"Text", "Array<Text>"},
		{"Blob", "Array<Blob>"},
		{"Bytes", "Array<Bytes>"},
		{"Handler", "Array<Handler>"},
		{"JavaByte", "Array<JavaByte>"},
		{"Shadowed", "Array<Shadowed>"},
		// A chain too long to follow, and aliases that name one another.
		{"Link11", "Array<Link11>"},
		{"LoopA", "Array<LoopA>"},
		{"Self", "Array<Self>"},
		// A name that is no alias.
		{"Unknown", "Array<Unknown>"},
	}
	r := varargAliasResolver()
	for _, tt := range tests {
		t.Run(tt.element, func(t *testing.T) {
			if got := r.ParameterType(varargAliasUnit(), varargParameter(tt.element)); got != tt.want {
				t.Errorf("ParameterType(vararg %s) = %q, want %q", tt.element, got, tt.want)
			}
		})
	}
}

// An element written nullable is an array of boxed values whatever it names, and
// a parameter that is not a vararg is the type it is written with.
func TestParameterType_ANullableElementOrAParameterOfATypealiasIsAsWritten(t *testing.T) {
	r := varargAliasResolver()
	unit := varargAliasUnit()

	for _, element := range []string{"Octet", "Letter", "Octet3", "Const<String>"} {
		param := varargParameter(element)
		param.ElementNullable = true
		if got, want := r.ParameterType(unit, param), "Array<"+element+">"; got != want {
			t.Errorf("ParameterType(vararg %s?) = %q, want %q", element, got, want)
		}
	}

	for _, written := range []string{"Octet", "MaybeByte", "Array<Octet>", "List<Octet>"} {
		param := ktparser.Parameter{Name: "values", Type: written}
		if got := r.ParameterType(unit, param); got != written {
			t.Errorf("ParameterType(%s) = %q, want it as written", written, got)
		}
	}

	// A resolver that was never built knows no alias.
	var none *Resolver
	if got := none.ParameterType(unit, varargParameter("Octet")); got != "Array<Octet>" {
		t.Errorf("ParameterType(vararg Octet) without a resolver = %q, want Array<Octet>", got)
	}
}

// A file sees an alias the way it sees a class: by the package it is declared in,
// an import by name, a renaming import, a star import, or its own file when the
// alias is private. The array is written as the file spells it.
func TestParameterType_ATypealiasIsFoundWhereTheFileCanSeeIt(t *testing.T) {
	file := func(pkg, path string, imports ...ktparser.Import) *ktparser.Class {
		return &ktparser.Class{Name: "Handler", Package: pkg, FilePath: path, Imports: imports}
	}
	const app = "com.example.kcvarargapp"

	tests := []struct {
		name    string
		unit    *ktparser.Class
		element string
		want    string
	}{
		{"same package", varargAliasUnit(), "Octet", "ByteArray"},
		{"import", file(app, "app/Handler.kt", ktparser.Import{Path: varargAliasPackage + ".Octet"}), "Octet", "ByteArray"},
		{"import of a nullable alias", file(app, "app/Handler.kt", ktparser.Import{Path: varargAliasPackage + ".MaybeByte"}), "MaybeByte", "Array<MaybeByte>"},
		{"renaming import", file(app, "app/Handler.kt", ktparser.Import{Path: varargAliasPackage + ".Octet", Alias: "Renamed"}), "Renamed", "ByteArray"},
		{"renaming import of a nullable alias", file(app, "app/Handler.kt", ktparser.Import{Path: varargAliasPackage + ".MaybeByte", Alias: "Renamed"}), "Renamed", "Array<Renamed>"},
		{"star import", file(app, "app/Handler.kt", ktparser.Import{Path: varargAliasPackage, OnDemand: true}), "Octet", "ByteArray"},
		{"star import of a nullable alias", file(app, "app/Handler.kt", ktparser.Import{Path: varargAliasPackage, OnDemand: true}), "MaybeChar", "Array<MaybeChar>"},
		{"not imported", file(app, "app/Handler.kt"), "Octet", "Array<Octet>"},
		{"the old name of a renamed alias", file(app, "app/Handler.kt", ktparser.Import{Path: varargAliasPackage + ".Octet", Alias: "Renamed"}), "Octet", "Array<Octet>"},
		{"private alias of the same file", file(varargAliasPackage, "alias/Secret.kt"), "Secret", "ByteArray"},
		{"private alias of another file", file(varargAliasPackage, "alias/Other.kt"), "Secret", "Array<Secret>"},
		{"private alias, imported", file(app, "app/Handler.kt", ktparser.Import{Path: varargAliasPackage + ".Secret"}), "Secret", "Array<Secret>"},
		// The ByteArray is written by the name the file gives it: its own simple
		// name, or in full where a class of the file's package hides that.
		{"a class called ByteArray", file(varargShadowPackage, "shadow/Handler.kt", ktparser.Import{Path: varargAliasPackage + ".Octet"}), "Octet", "kotlin.ByteArray"},
	}
	classes := []ktparser.Class{
		scopeShadow(varargShadowPackage, "Byte", "own"),
		scopeShadow(varargShadowPackage, "ByteArray", "own"),
	}
	r := NewWithAliases(classes, nil, varargAliases())
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := r.ParameterType(tt.unit, varargParameter(tt.element)); got != tt.want {
				t.Errorf("ParameterType(vararg %s) = %q, want %q", tt.element, got, tt.want)
			}
		})
	}
}

// The target is read in the file that declares the alias: a class called Byte in
// the package of the file that uses the alias does not change what the alias
// stands for, nor does a type parameter of the class that uses it hide a name the
// alias is declared with. A class of the package, a type parameter and a nested
// class that are called like the alias hide it, as they hide any name.
func TestParameterType_ATypealiasIsReadWhereItIsDeclared(t *testing.T) {
	classes := []ktparser.Class{
		scopeShadow(varargShadowPackage, "Byte", "own"),
		scopeShadow(varargShadowPackage, "Octet", "own"),
		scopeShadow("com.example.kcvarargother", "Octet", "other"),
	}
	r := NewWithAliases(classes, nil, varargAliases())

	// Byte is a class of the package of the file that uses the alias: the alias
	// still stands for kotlin.Byte, and the alias of the shadow package for the
	// class its package declares.
	shadowFile := &ktparser.Class{Name: "Handler", Package: varargShadowPackage, FilePath: "shadow/Handler.kt"}
	if got := r.ParameterType(shadowFile, varargParameter("Byte")); got != "Array<Byte>" {
		t.Errorf("Byte beside a class called Byte = %q, want Array<Byte>", got)
	}
	if got := r.ParameterType(shadowFile, varargParameter("Shadowed")); got != "Array<Shadowed>" {
		t.Errorf("an alias of the class called Byte = %q, want Array<Shadowed>", got)
	}
	starred := &ktparser.Class{
		Name: "Handler", Package: varargShadowPackage, FilePath: "shadow/Handler.kt",
		Imports: []ktparser.Import{{Path: varargAliasPackage, OnDemand: true}},
	}
	// Octet is a class of the package here: it wins over the star import.
	if got := r.ParameterType(starred, varargParameter("Octet")); got != "Array<Octet>" {
		t.Errorf("a class of the package called like a star-imported alias = %q, want Array<Octet>", got)
	}
	if got := r.ParameterType(starred, varargParameter("Octet2")); got != "ByteArray" {
		t.Errorf("an alias of Byte beside a class called Byte = %q, want ByteArray", got)
	}

	// The class that uses the alias has a type parameter, or a name, of its own.
	unit := varargAliasUnit()
	unit.TypeParameters = []string{"Octet"}
	if got := r.ParameterType(unit, varargParameter("Octet")); got != "Array<Octet>" {
		t.Errorf("a type parameter called like the alias = %q, want Array<Octet>", got)
	}
	if got := r.ParameterType(unit, varargParameter("Letter")); got != "CharArray" {
		t.Errorf("a type parameter does not hide another alias = %q, want CharArray", got)
	}
}
