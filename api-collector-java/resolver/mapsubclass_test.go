package resolver

import (
	"testing"

	"github.com/tangcent/apilot/api-collector-java/parser"
	model "github.com/tangcent/apilot/api-model"
)

// A class that extends a map is a map of what the arguments of its extends
// clause say, which are written in the scope of the class, and the value is the
// second of them: not the first type parameter of the class, whatever it is.

func mapSubclasses() []parser.Class {
	class := func(name string, params []string, superArgs []string, imports ...parser.Import) parser.Class {
		return parser.Class{
			Name: name, Package: typeArgLibPackage, FilePath: "com/example/lib/" + name + ".java",
			TypeParameters: params, SuperClass: "HashMap", SuperClassTypeArgs: superArgs, Imports: imports,
		}
	}
	dtoItem := parser.Import{Path: typeArgDtoPackage + ".Item"}
	return []parser.Class{
		typeArgItem(typeArgDtoPackage, "name"),
		typeArgItem(typeArgLibPackage, "code"),
		class("Bag", []string{"V"}, []string{"String", "List<V>"}),
		class("Nested", []string{"V"}, []string{"String", "Map<String, V>"}),
		class("Pairs", []string{"K", "V"}, []string{"K", "V"}),
		class("Counter", []string{"T"}, []string{"String", "Integer"}),
		// Written in the library's package, which has an Item of its own.
		class("Named", nil, []string{"String", "Item"}, dtoItem),
		class("NamedInFull", nil, []string{"String", "com.example.app.dto.Item"}),
		class("NamedOwn", nil, []string{"String", "Item"}),
		class("NamedList", nil, []string{"String", "List<Item>"}, dtoItem),
		class("Raw", nil, nil),
		class("Holder", []string{"T"}, nil),
	}
}

func TestResolve_AMapSubclassIsAMapOfItsSuperclassArgument(t *testing.T) {
	const item = "Item(name)"
	caller := recursionCaller()
	for view, resolveIn := range typeArgViews {
		t.Run(view, func(t *testing.T) {
			r := NewTypeResolver(mapSubclasses())
			for typeName, want := range map[string]string{
				"Bag<Item>":                 "{[" + item + "]}",
				"Bag<com.example.lib.Item>": "{[Item(code)]}",
				"Bag<Integer>":              "{[int]}",
				"Nested<Item>":              "{{" + item + "}}",
				"Pairs<String, Item>":       "{" + item + "}",
				"Pairs<Item, String>":       "{string}",
				"Counter<Item>":             "{int}",
				"Named":                     "{" + item + "}",
				"NamedInFull":               "{" + item + "}",
				"NamedOwn":                  "{Item(code)}",
				"NamedList":                 "{[" + item + "]}",
				"Raw":                       "{null}",
				"Bag":                       "{[V]}",
			} {
				got := resolveIn(r, caller, typeName)
				if shape := typeArgShape(got); shape != want {
					t.Errorf("%s = %s, want %s", typeName, shape, want)
				}
			}
		})
	}
}

// A map subclass is held as a field like any other class, and its parameter
// is bound by the argument of the field's type.
func TestResolve_AMapSubclassAsTheArgumentOfAGenericClass(t *testing.T) {
	f := typeArgField
	classes := append(mapSubclasses(),
		parser.Class{
			Name: "Box", Package: typeArgLibPackage, FilePath: "com/example/lib/Box.java", TypeParameters: []string{"T"},
			Fields: []parser.Field{f("data", "T"), f("bag", "Bag<T>"), f("raw", "Bag")},
		})
	r := NewTypeResolver(classes)
	box := r.ResolveFrom(recursionCaller(), "Box<Item>", nil)
	for path, want := range map[string]string{
		"data": "Item(name)",
		"bag":  "{[Item(name)]}",
		"raw":  "{[V]}", // not Bag<Item>: the parameter of a raw type stays open
	} {
		if got := typeArgShape(typeArgAt(box, path)); got != want {
			t.Errorf("Box<Item>.%s = %s, want %s", path, got, want)
		}
	}
	if bag := box.Fields["bag"].Model; bag.Kind != model.KindMap {
		t.Errorf("bag = %+v, want a map", bag)
	}
	// A map of Box<Item>, as an argument.
	data := r.ResolveFrom(recursionCaller(), "Box<Bag<Item>>", nil)
	if got := typeArgShape(typeArgAt(data, "data")); got != "{[Item(name)]}" {
		t.Errorf("Box<Bag<Item>>.data = %s, want {[Item(name)]}", got)
	}
}
