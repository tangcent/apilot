package resolver

import (
	"testing"

	"github.com/tangcent/apilot/api-collector-java/parser"
	model "github.com/tangcent/apilot/api-model"
)

// A class binds its own type parameters from its own arguments, and nothing
// else: the arguments are spelled in the scope of the caller, so the bindings of
// the class that holds a field mean nothing inside the class the field is typed
// with. A parameter that no argument binds stays open, as it does in Java for a
// raw type, whichever name another class gives its own.

func unboundClasses() []parser.Class {
	f := typeArgField
	class := func(name string, params []string, superClass string, superArgs []string, fields ...parser.Field) parser.Class {
		return parser.Class{
			Name: name, Package: typeArgLibPackage, FilePath: "com/example/lib/" + name + ".java",
			TypeParameters: params, SuperClass: superClass, SuperClassTypeArgs: superArgs, Fields: fields,
		}
	}
	return []parser.Class{
		typeArgItem(typeArgDtoPackage, "name"),
		class("Box", []string{"T"}, "", nil, f("data", "T")),
		class("Wrapper", []string{"U"}, "", nil, f("data", "U")),
		// A raw use of a generic class inside a generic class, and the same name
		// for the parameter of each.
		class("Holder", []string{"T"}, "", nil, f("raw", "Box"), f("typed", "Box<T>"), f("wrapped", "Wrapper<T>")),
		class("RawBase", []string{"T"}, "", nil, f("value", "T")),
		// Extends a generic class without arguments.
		class("RawSub", []string{"T"}, "RawBase", nil, f("own", "T")),
		// A map subclass used without arguments inside a generic class.
		class("Bag", []string{"T"}, "HashMap", []string{"String", "T"}),
		class("BagHolder", []string{"T"}, "", nil, f("raw", "Bag"), f("typed", "Bag<T>")),
		// A class with no parameter, that has a field typed with a name that is the
		// parameter of the class it is held by. In Java only a non-static inner class
		// could write that; the resolver keeps no enclosing class's bindings for it.
		class("Plain", nil, "", nil, f("data", "T")),
		class("PlainHolder", []string{"T"}, "", nil, f("plain", "Plain")),
	}
}

func TestResolve_AParameterNoArgumentBindsStaysOpen(t *testing.T) {
	const item = "Item(name)"
	for view, resolveIn := range typeArgViews {
		t.Run(view, func(t *testing.T) {
			r := NewTypeResolver(unboundClasses())
			caller := recursionCaller()
			resolve := func(typeName string) *model.ObjectModel { return resolveIn(r, caller, typeName) }

			holder := resolve("Holder<Item>")
			for path, want := range map[string]string{
				"typed.data":   item,
				"wrapped.data": item,
				"raw.data":     "T", // the raw Box is not a Box<Item>
			} {
				if got := typeArgShape(typeArgAt(holder, path)); got != want {
					t.Errorf("Holder<Item>.%s = %s, want %s", path, got, want)
				}
			}
			if field := typeArgAt(holder, "raw").Fields["data"]; !field.Generic {
				t.Errorf("Holder<Item>.raw.data = %+v, want a generic field", field)
			}

			sub := resolve("RawSub<Item>")
			for path, want := range map[string]string{"own": item, "value": "T"} {
				if got := typeArgShape(typeArgAt(sub, path)); got != want {
					t.Errorf("RawSub<Item>.%s = %s, want %s", path, got, want)
				}
			}

			bags := resolve("BagHolder<Item>")
			for path, want := range map[string]string{"typed": "{" + item + "}", "raw": "{T}"} {
				if got := typeArgShape(typeArgAt(bags, path)); got != want {
					t.Errorf("BagHolder<Item>.%s = %s, want %s", path, got, want)
				}
			}

			plain := resolve("PlainHolder<Item>")
			if got := typeArgShape(typeArgAt(plain, "plain.data")); got != "T" {
				t.Errorf("PlainHolder<Item>.plain.data = %s, want T: Plain declares no parameter", got)
			}
		})
	}
}

// The bindings a caller hands to ResolveFrom bind the parameters of the class
// the type is declared in; a raw generic class named in that type does not
// inherit them.
func TestResolve_BindingsOfTheCallerDoNotReachARawClass(t *testing.T) {
	r := NewTypeResolver(unboundClasses())
	// The type of a method of a generic controller whose variable is T.
	controller := parser.Class{Name: "Controller", Package: typeArgAppPackage, FilePath: "com/example/app/Controller.java"}
	bindings := map[string]string{"T": "com.example.app.dto.Item"}

	if got := typeArgShape(r.ResolveFrom(controller, "T", bindings)); got != "Item(name)" {
		t.Errorf("T = %s, want Item(name)", got)
	}
	if got := typeArgShape(typeArgAt(r.ResolveFrom(controller, "com.example.lib.Box<T>", bindings), "data")); got != "Item(name)" {
		t.Errorf("Box<T>.data = %s, want Item(name)", got)
	}
	if got := typeArgShape(typeArgAt(r.ResolveFrom(controller, "com.example.lib.Box", bindings), "data")); got != "T" {
		t.Errorf("Box.data = %s, want T: a raw Box is not a Box<T> of the controller", got)
	}
}
