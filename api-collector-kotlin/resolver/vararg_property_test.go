package resolver

import (
	"testing"

	ktparser "github.com/tangcent/apilot/api-collector-kotlin/parser"
)

// A property of the primary constructor declared `vararg val` holds the array a
// vararg parameter is received as, in the JSON view and the binding view: the
// shape of the property is the shape of one declared with that array.
func TestResolve_AVarargPropertyIsTheArrayItsElementsMake(t *testing.T) {
	property := func(name, element string, nullable bool) ktparser.Property {
		return ktparser.Property{Name: name, Type: "Array<" + element + ">", Vararg: true, ElementNullable: nullable, InConstructor: true}
	}
	cases := []struct {
		name, element string
		nullable      bool
		twin          string
	}{
		{"bytes", "Byte", false, "ByteArray"},
		{"chars", "Char", false, "CharArray"},
		{"ints", "Int", false, "IntArray"},
		{"unsigned", "UByte", false, "UByteArray"},
		{"boxed", "Byte", true, "Array<Byte>"},
		{"boxedChars", "Char", true, "Array<Char>"},
		{"names", "String", false, "Array<String>"},
		{"lists", "List<Byte>", false, "Array<List<Byte>>"},
		{"octets", "Octet", false, "ByteArray"},
		{"octets3", "Octet3", false, "ByteArray"},
		{"generic", "Const<String>", false, "ByteArray"},
		{"maybe", "MaybeByte", false, "Array<MaybeByte>"},
		{"maybeOctets", "Octet", true, "Array<Octet>"},
		{"top", "Top", false, "Array<Top>"},
		{"shadowed", "Shadowed", false, "Array<Shadowed>"},
	}
	form := ktparser.Class{Name: "Form", Package: varargAliasPackage, FilePath: "alias/Form.kt"}
	twin := ktparser.Class{Name: "Twin", Package: varargAliasPackage, FilePath: "alias/Twin.kt"}
	for _, c := range cases {
		form.Fields = append(form.Fields, property(c.name, c.element, c.nullable))
		twin.Fields = append(twin.Fields, scopeProp(c.name, c.twin))
	}
	shadowByte := scopeShadow(varargShadowPackage, "Byte", "own")

	for view := range scopeViews {
		t.Run(view, func(t *testing.T) {
			r := NewWithAliases([]ktparser.Class{form, twin, shadowByte}, nil, varargAliases())
			got := scopeViews[view](r, &form, "Form")
			want := scopeViews[view](r, &twin, "Twin")
			for _, c := range cases {
				if g, w := scopeModelShape(scopeAt(got, c.name)), scopeModelShape(scopeAt(want, c.name)); g != w {
					t.Errorf("vararg val %s: %s = %s, want %s as %s is", c.name, c.element, g, w, c.twin)
				}
			}
		})
	}

	// A ByteArray is one string in the JSON view, and the binding view reads it as
	// an array of numbers.
	r := NewWithAliases([]ktparser.Class{form, twin, shadowByte}, nil, varargAliases())
	for view, want := range map[string]map[string]string{
		"json":    {"bytes": "string", "chars": "string", "ints": "[int]", "boxed": "[int]", "names": "[string]"},
		"binding": {"bytes": "[int]", "chars": "[string]", "ints": "[int]", "boxed": "[int]", "names": "[string]"},
	} {
		got := scopeViews[view](r, &form, "Form")
		for name, shape := range want {
			if actual := scopeModelShape(scopeAt(got, name)); actual != shape {
				t.Errorf("%s view: vararg val %s = %s, want %s", view, name, actual, shape)
			}
		}
	}
}
