package parser

import (
	"reflect"
	"testing"
)

// A property of the primary constructor declared `vararg val` holds the array of
// its elements, as a vararg parameter is received: Type is `Array<T>`, and the
// element being nullable is kept apart, as for a parameter.
func TestParse_AVarargPropertyOfThePrimaryConstructorIsItsArray(t *testing.T) {
	result := parseKotlin(t, `package a.b

class Payload(vararg val bytes: Byte, val plain: Byte)
class Boxed(vararg val bytes: Byte?)
class Names(@get:JsonProperty("n") vararg var names: String)
class Octets(vararg val octets: Octet)
class Lists<T>(vararg val lists: List<T>)
class Skipped(vararg bytes: Byte)
`)
	type shape struct {
		Type            string
		Vararg          bool
		ElementNullable bool
		Mutable         bool
	}
	got := map[string]map[string]shape{}
	for _, class := range result.Classes {
		got[class.Name] = map[string]shape{}
		for _, field := range class.Fields {
			if !field.InConstructor {
				t.Errorf("%s.%s is not marked as a property of the constructor", class.Name, field.Name)
			}
			got[class.Name][field.Name] = shape{field.Type, field.Vararg, field.ElementNullable, field.IsMutable}
		}
	}
	want := map[string]map[string]shape{
		"Payload": {"bytes": {"Array<Byte>", true, false, false}, "plain": {"Byte", false, false, false}},
		"Boxed":   {"bytes": {"Array<Byte>", true, true, false}},
		"Names":   {"names": {"Array<String>", true, false, true}},
		"Octets":  {"octets": {"Array<Octet>", true, false, false}},
		"Lists":   {"lists": {"Array<List<T>>", true, false, false}},
		// A parameter that is not a property is no field.
		"Skipped": {},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("fields = %v, want %v", got, want)
	}
}
