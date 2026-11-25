package resolver

import (
	"testing"

	ktparser "github.com/tangcent/apilot/api-collector-kotlin/parser"
)

// Kotlin types `vararg n: Byte` as a ByteArray and `vararg c: Char` as a
// CharArray, the primitive arrays, and `vararg n: Byte?` as an `Array` of the
// boxed type, as it types `vararg s: String` as an `Array<String>`. Which one a
// vararg is depends on what the name of its element denotes where it is written.

const varargPackage = "com.example.kcvararg"

func varargParameter(element string) ktparser.Parameter {
	return ktparser.Parameter{Name: "values", Type: "Array<" + element + ">", Vararg: true}
}

func varargUnit(imports ...ktparser.Import) *ktparser.Class {
	return &ktparser.Class{Name: "Handler", Package: varargPackage, FilePath: "vararg/Handler.kt", Imports: imports}
}

func TestParameterType_AVarargOfAPrimitiveTypeIsThePrimitiveArray(t *testing.T) {
	tests := []struct{ element, want string }{
		{"Byte", "ByteArray"},
		{"Char", "CharArray"},
		{"Short", "ShortArray"},
		{"Int", "IntArray"},
		{"Long", "LongArray"},
		{"Float", "FloatArray"},
		{"Double", "DoubleArray"},
		{"Boolean", "BooleanArray"},
		{"UByte", "UByteArray"},
		{"UShort", "UShortArray"},
		{"UInt", "UIntArray"},
		{"ULong", "ULongArray"},
		{"kotlin.Byte", "ByteArray"},
		{" Byte ", "ByteArray"},
		// Not a primitive type of Kotlin: an Array of it.
		{"String", "Array<String>"},
		{"Number", "Array<Number>"},
		{"Any", "Array<Any>"},
		{"Item", "Array<Item>"},
		{"java.lang.Byte", "Array<java.lang.Byte>"},
		{"List<Byte>", "Array<List<Byte>>"},
		{"ByteArray", "Array<ByteArray>"},
	}
	r := New(nil)
	for _, tt := range tests {
		t.Run(tt.element, func(t *testing.T) {
			param := varargParameter(tt.element)
			if got := r.ParameterType(varargUnit(), param); got != tt.want {
				t.Errorf("ParameterType(vararg %s) = %q, want %q", tt.element, got, tt.want)
			}
		})
	}
}

// A vararg of a nullable element is an array of boxed values, and a parameter
// that is not a vararg is the type it is written with.
func TestParameterType_ANullableVarargAndAnArrayAreAsWritten(t *testing.T) {
	r := New(nil)
	unit := varargUnit()

	nullable := varargParameter("Byte")
	nullable.ElementNullable = true
	if got := r.ParameterType(unit, nullable); got != "Array<Byte>" {
		t.Errorf("ParameterType(vararg Byte?) = %q, want Array<Byte>", got)
	}

	for _, written := range []string{"Array<Byte>", "ByteArray", "Byte", "List<Byte>"} {
		param := ktparser.Parameter{Name: "values", Type: written}
		if got := r.ParameterType(unit, param); got != written {
			t.Errorf("ParameterType(%s) = %q, want it as written", written, got)
		}
	}

	// A resolver that was never built reads the same way.
	var none *Resolver
	if got := none.ParameterType(unit, varargParameter("Char")); got != "CharArray" {
		t.Errorf("ParameterType(vararg Char) = %q, want CharArray", got)
	}
}

// A name is what Kotlin binds it to: a class of the project that is called Byte,
// or a type parameter, is not the primitive type, and one that is called
// ByteArray does not hide the array, which is then written in full.
func TestParameterType_TheElementIsWhatItsNameDenotesWhereItIsWritten(t *testing.T) {
	byteClass := scopeShadow(varargPackage, "Byte", "own")
	otherByte := scopeShadow("com.example.kcvarargother", "Byte", "other")
	arrayClass := scopeShadow(varargPackage, "ByteArray", "own")
	arrayClassElsewhere := scopeShadow("com.example.kcvarargother", "ByteArray", "other")

	r := New([]ktparser.Class{byteClass, otherByte, arrayClass, arrayClassElsewhere})
	if got := r.ParameterType(varargUnit(), varargParameter("Byte")); got != "Array<Byte>" {
		t.Errorf("a class of the package called Byte: got %q, want Array<Byte>", got)
	}
	if got := r.ParameterType(varargUnit(), varargParameter("Short")); got != "ShortArray" {
		t.Errorf("a class called Byte does not hide Short: got %q, want ShortArray", got)
	}
	if got := r.ParameterType(varargUnit(), varargParameter("kotlin.Byte")); got != "kotlin.ByteArray" {
		t.Errorf("kotlin.Byte beside a class called ByteArray: got %q, want kotlin.ByteArray", got)
	}

	// Another package's class is in scope only where it is imported.
	other := &ktparser.Class{Name: "Handler", Package: "com.example.kcvarargapp", FilePath: "app/Handler.kt"}
	if got := r.ParameterType(other, varargParameter("Byte")); got != "ByteArray" {
		t.Errorf("a class of another package called Byte: got %q, want ByteArray", got)
	}
	imported := &ktparser.Class{
		Name: "Handler", Package: "com.example.kcvarargapp", FilePath: "app/Handler.kt",
		Imports: []ktparser.Import{{Path: "com.example.kcvarargother.Byte"}},
	}
	if got := r.ParameterType(imported, varargParameter("Byte")); got != "Array<Byte>" {
		t.Errorf("an imported class called Byte: got %q, want Array<Byte>", got)
	}
	starred := &ktparser.Class{
		Name: "Handler", Package: "com.example.kcvarargapp", FilePath: "app/Handler.kt",
		Imports: []ktparser.Import{{Path: "com.example.kcvarargother", OnDemand: true}},
	}
	if got := r.ParameterType(starred, varargParameter("Byte")); got != "Array<Byte>" {
		t.Errorf("a star-imported class called Byte: got %q, want Array<Byte>", got)
	}

	// A renaming import binds only the new name.
	renamed := &ktparser.Class{
		Name: "Handler", Package: "com.example.kcvarargapp", FilePath: "app/Handler.kt",
		Imports: []ktparser.Import{{Path: "com.example.kcvarargother.Byte", Alias: "Octet"}},
	}
	if got := r.ParameterType(renamed, varargParameter("Octet")); got != "Array<Octet>" {
		t.Errorf("a renamed class: got %q, want Array<Octet>", got)
	}
	if got := r.ParameterType(renamed, varargParameter("Byte")); got != "ByteArray" {
		t.Errorf("Byte beside a renamed class: got %q, want ByteArray", got)
	}

	// An explicit import of java.lang.Byte binds the name to that class, which
	// Kotlin does not type a vararg as a primitive array.
	javaByte := &ktparser.Class{
		Name: "Handler", Package: "com.example.kcvarargapp", FilePath: "app/Handler.kt",
		Imports: []ktparser.Import{{Path: "java.lang.Byte"}},
	}
	if got := r.ParameterType(javaByte, varargParameter("Byte")); got != "Array<Byte>" {
		t.Errorf("an imported java.lang.Byte: got %q, want Array<Byte>", got)
	}
}

// A type parameter hides every class: `vararg values: Byte` of a class that
// declares a type parameter called Byte is an array of that.
func TestParameterType_ATypeParameterIsNotAPrimitiveType(t *testing.T) {
	r := New(nil)
	unit := varargUnit()
	unit.TypeParameters = []string{"T", "Byte"}
	for _, element := range []string{"T", "Byte"} {
		param := varargParameter(element)
		if got := r.ParameterType(unit, param); got != param.Type {
			t.Errorf("ParameterType(vararg %s) = %q, want %q", element, got, param.Type)
		}
	}
	if got := r.ParameterType(unit, varargParameter("Char")); got != "CharArray" {
		t.Errorf("ParameterType(vararg Char) = %q, want CharArray", got)
	}
}
