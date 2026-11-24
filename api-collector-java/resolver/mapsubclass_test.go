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

// A map subclass whose value type refers to the subclass again is expanded once
// and then named: `Tree extends HashMap<String, Tree>` is a map of trees, and the
// tree inside it is a reference, not the same map expanded without end. The value
// comes from the superclass's arguments, which may name the class with the
// arguments it is expanded as, so the class is registered as being expanded
// before the map's value is read.
func selfMapClasses() []parser.Class {
	f := typeArgField
	class := func(name string, params []string, value string) parser.Class {
		return parser.Class{
			Name: name, Package: typeArgLibPackage, FilePath: "com/example/lib/" + name + ".java",
			TypeParameters: params, SuperClass: "HashMap", SuperClassTypeArgs: []string{"String", value},
		}
	}
	return []parser.Class{
		typeArgItem(typeArgDtoPackage, "name"),
		class("Tree", nil, "Tree"),
		class("Node", []string{"V"}, "Node<V>"),
		class("Forest", []string{"V"}, "List<Forest<V>>"),
		class("Grow", []string{"V"}, "Grow<List<V>>"),
		class("Left", nil, "Right"),
		class("Right", nil, "Left"),
		{
			Name: "Holder", Package: typeArgLibPackage, FilePath: "com/example/lib/Holder.java",
			Fields: []parser.Field{f("tree", "Tree"), f("node", "Node<Item>"), f("raw", "Node"), f("trees", "List<Tree>"), f("forest", "Forest<Item>")},
		},
	}
}

func TestResolve_AMapSubclassThatContainsItselfEnds(t *testing.T) {
	isMapOfRef := func(m *model.ObjectModel) bool { return m != nil && m.IsMap() && isReference(m.ValueModel) }
	for view, resolveIn := range typeArgViews {
		t.Run(view, func(t *testing.T) {
			r := NewTypeResolver(selfMapClasses())
			caller := recursionCaller()
			for _, typeName := range []string{"Tree", "Node<Item>", "Node", "Grow<Item>"} {
				if got := resolveIn(r, caller, typeName); !isMapOfRef(got) {
					t.Errorf("%s = %s, want a map whose value is a reference", typeName, typeArgShape(got))
				}
			}

			// A map of lists of the map, and two maps that hold each other.
			forest := resolveIn(r, caller, "Forest<Item>")
			if !forest.IsMap() || !forest.ValueModel.IsArray() || !isReference(forest.ValueModel.Items) {
				t.Errorf("Forest<Item> = %s, want a map of arrays of references", typeArgShape(forest))
			}
			left := resolveIn(r, caller, "Left")
			if !left.IsMap() || !left.ValueModel.IsMap() || !isReference(left.ValueModel.ValueModel) {
				t.Errorf("Left = %s, want a map of Right, a map of references", typeArgShape(left))
			}

			// As the type of a field, the map ends at its own reference too.
			holder := resolveIn(r, caller, "com.example.lib.Holder")
			for _, name := range []string{"tree", "node", "raw"} {
				if got := typeArgAt(holder, name); !isMapOfRef(got) {
					t.Errorf("Holder.%s = %s, want a map whose value is a reference", name, typeArgShape(got))
				}
			}
			if trees := typeArgAt(holder, "trees"); !trees.IsArray() || !isMapOfRef(trees.Items) {
				t.Errorf("Holder.trees = %s, want an array of maps whose value is a reference", typeArgShape(trees))
			}
			if got := typeArgAt(holder, "forest"); !got.IsMap() || !got.ValueModel.IsArray() || !isReference(got.ValueModel.Items) {
				t.Errorf("Holder.forest = %s, want a map of arrays of references", typeArgShape(got))
			}
		})
	}
}

// The superclass that a class writes in full is a map as well: `extends
// java.util.HashMap<String, Item>` is the map `extends HashMap<String, Item>` is,
// and a class of another package that shares the simple name is not.
func TestResolve_AMapSubclassMayWriteItsSuperclassInFull(t *testing.T) {
	dtoItem := parser.Import{Path: typeArgDtoPackage + ".Item"}
	full := func(name, super string) parser.Class {
		return parser.Class{
			Name: name, Package: typeArgLibPackage, FilePath: "com/example/lib/" + name + ".java",
			SuperClass: super, SuperClassTypeArgs: []string{"String", "Item"}, Imports: []parser.Import{dtoItem},
		}
	}
	r := NewTypeResolver(append(mapSubclasses(),
		full("FullHash", "java.util.HashMap"),
		full("FullLinked", "java.util.LinkedHashMap"),
		full("Foreign", "com.example.other.HashMap"),
	))
	caller := recursionCaller()

	for _, name := range []string{"FullHash", "FullLinked"} {
		if got := typeArgShape(r.ResolveFrom(caller, name, nil)); got != "{Item(name)}" {
			t.Errorf("%s = %s, want a map of Item", name, got)
		}
	}
	if got := r.ResolveFrom(caller, "Foreign", nil); got.Kind == model.KindMap {
		t.Errorf("Foreign = %+v, want a class: com.example.other.HashMap is not java.util.HashMap", got)
	}
}
