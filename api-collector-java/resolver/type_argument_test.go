package resolver

import (
	"reflect"
	"sort"
	"strings"
	"testing"

	collector "github.com/tangcent/apilot/api-collector"
	"github.com/tangcent/apilot/api-collector-java/parser"
	model "github.com/tangcent/apilot/api-model"
)

// A type argument is bound to a type parameter and the parameter is resolved
// inside the generic class. The argument is written in the scope of the caller,
// which the generic class does not share: it may sit in another package, and
// that package may declare a class of the same simple name. The tests below
// give the same simple name, Item, to a class of the application and to a class
// of the library the generic classes come from, and ask which one an argument
// ends up as.
//
// Each case is run against a library of scanned sources and against a library
// read through a dependency resolver, with a resolver that has not read the
// library's own Item and one that has, in the JSON view and in the binding view.

const (
	typeArgAppPackage = "com.example.app"
	typeArgDtoPackage = "com.example.app.dto"
	typeArgLibPackage = "com.example.lib"
)

func typeArgField(name, typeName string) parser.Field {
	return parser.Field{Name: name, Type: typeName}
}

func typeArgItem(pkg, field string) parser.Class {
	return parser.Class{
		Name: "Item", Package: pkg, FilePath: strings.ReplaceAll(pkg, ".", "/") + "/Item.java",
		Fields: []parser.Field{typeArgField(field, "String")},
	}
}

// typeArgLibraryClass is a class of the library, which the library's own
// package shares with its Item. It imports the library's generic classes by
// name, so that a library class read from a dependency finds them without the
// resolver reading the package.
func typeArgLibraryClass(name string, params []string, superClass string, superArgs []string, fields ...parser.Field) parser.Class {
	var imports []parser.Import
	for _, neighbour := range []string{"Box", "Wrapper", "Pair", "SameInner", "Base"} {
		imports = append(imports, parser.Import{Path: typeArgLibPackage + "." + neighbour})
	}
	return parser.Class{
		Name: name, Package: typeArgLibPackage, FilePath: "com/example/lib/" + name + ".java",
		TypeParameters: params, SuperClass: superClass, SuperClassTypeArgs: superArgs, Fields: fields,
		Imports: imports,
	}
}

// typeArgLibrary declares the generic classes of the library. `Wrapper<U>` is
// the second level of `Outer<T>`, which passes its parameter on in several
// shapes; `Holder` spells an argument in full inside the library.
func typeArgLibrary() []parser.Class {
	f := typeArgField
	return []parser.Class{
		typeArgItem(typeArgLibPackage, "code"),
		typeArgLibraryClass("Box", []string{"T"}, "", nil,
			f("data", "T"), f("list", "List<T>"), f("array", "T[]"),
			f("byKey", "Map<String, T>"), f("bounded", "List<? extends T>")),
		typeArgLibraryClass("Wrapper", []string{"U"}, "", nil, f("data", "U")),
		typeArgLibraryClass("Outer", []string{"T"}, "", nil,
			f("wrapped", "Wrapper<T>"), f("wrappedList", "Wrapper<List<T>>"),
			f("wrappedArray", "Wrapper<T[]>"), f("wrappedMap", "Wrapper<Map<String, T>>"),
			f("paired", "Wrapper<Pair<T, List<T>>>")),
		// The parameter has the name of the one it is passed to.
		typeArgLibraryClass("Same", []string{"T"}, "", nil, f("inner", "SameInner<T>")),
		typeArgLibraryClass("SameInner", []string{"T"}, "", nil, f("data", "T")),
		typeArgLibraryClass("Pair", []string{"K", "V"}, "", nil, f("first", "K"), f("second", "V")),
		typeArgLibraryClass("Holder", nil, "", nil,
			f("fixedList", "List<com.example.app.dto.Item>"),
			f("fixedBox", "Box<com.example.app.dto.Item>"),
			f("fixedArray", "com.example.app.dto.Item[]"),
			f("fixedPair", "Pair<com.example.app.dto.Item, com.example.lib.Item>")),
		typeArgLibraryClass("Base", []string{"B"}, "", nil,
			f("value", "B"), f("list", "List<B>"), f("array", "B[]"), f("wrapped", "Wrapper<B>")),
		typeArgLibraryClass("Mid", []string{"M"}, "Base", []string{"M"}, f("midValue", "M")),
		typeArgLibraryClass("Listed", []string{"L"}, "Base", []string{"List<L>"}),
		// A subclass declared by the library names its argument in full.
		typeArgLibraryClass("DepSub", nil, "Base", []string{"com.example.app.dto.Item"}),
	}
}

func typeArgSubclass(name, superClass string, params []string, superArgs ...string) parser.Class {
	return parser.Class{
		Name: name, Package: typeArgAppPackage, FilePath: "com/example/app/" + name + ".java",
		TypeParameters: params, SuperClass: superClass, SuperClassTypeArgs: superArgs,
		Imports: []parser.Import{
			{Path: typeArgDtoPackage + ".Item"}, {Path: typeArgLibPackage + ".Base"}, {Path: typeArgLibPackage + ".Mid"},
			{Path: typeArgLibPackage + ".Listed"},
		},
	}
}

// typeArgApplication declares what the caller sees: the application's Item, the
// controller that writes the arguments, and the subclasses of the library's
// generic classes.
func typeArgApplication() []parser.Class {
	return []parser.Class{
		typeArgItem(typeArgDtoPackage, "name"),
		{
			Name: "Controller", Package: typeArgAppPackage, FilePath: "com/example/app/Controller.java",
			Imports: []parser.Import{
				{Path: typeArgDtoPackage + ".Item"},
				{Path: typeArgLibPackage + ".Box"}, {Path: typeArgLibPackage + ".Outer"}, {Path: typeArgLibPackage + ".Same"},
				{Path: typeArgLibPackage + ".Pair"}, {Path: typeArgLibPackage + ".Holder"}, {Path: typeArgLibPackage + ".Wrapper"},
				{Path: "java.util.List"}, {Path: "java.util.Map"},
			},
		},
		typeArgSubclass("Sub", "Base", nil, "Item"),
		typeArgSubclass("SubQualified", "Base", nil, "com.example.app.dto.Item"),
		typeArgSubclass("SubArray", "Base", nil, "Item[]"),
		typeArgSubclass("SubList", "Base", nil, "List<Item>"),
		typeArgSubclass("SubInt", "Base", nil, "Integer"),
		typeArgSubclass("SubString", "Base", nil, "String"),
		typeArgSubclass("SubGeneric", "Base", []string{"S"}, "List<S>"),
		typeArgSubclass("Leaf", "Mid", nil, "Item"),
		typeArgSubclass("LeafListed", "Listed", nil, "Item"),
	}
}

// typeArgShape spells a model as the identity of what it holds: an object by
// its name and fields, so that the application's Item is `Item(name)` and the
// library's is `Item(code)`.
func typeArgShape(m *model.ObjectModel) string {
	switch {
	case m == nil:
		return "nil"
	case m.IsArray():
		return "[" + typeArgShape(m.Items) + "]"
	case m.IsMap():
		return "{" + typeArgShape(m.ValueModel) + "}"
	case m.IsObject():
		names := make([]string, 0, len(m.Fields))
		for name := range m.Fields {
			names = append(names, name)
		}
		sort.Strings(names)
		return m.TypeName + "(" + strings.Join(names, ",") + ")"
	}
	return m.TypeName
}

// typeArgAt follows a dotted path of field names.
func typeArgAt(m *model.ObjectModel, path string) *model.ObjectModel {
	for _, name := range strings.Split(path, ".") {
		if m == nil || m.Fields[name] == nil {
			return nil
		}
		m = m.Fields[name].Model
	}
	return m
}

type typeArgWorld struct {
	name string
	// build returns a resolver over the application and the library, and the
	// dependency resolver it asks, if any.
	build func(t *testing.T) (*TypeResolver, *libraryClasses)
}

func typeArgWorlds() []typeArgWorld {
	byName := func(classes []parser.Class) map[string]parser.Class {
		out := map[string]parser.Class{}
		for _, c := range classes {
			out[c.Package+"."+c.Name] = c
		}
		return out
	}
	// A class read from a dependency has no compilation unit of the project.
	dependencies := func() map[string]parser.Class {
		out := byName(typeArgLibrary())
		for name, c := range out {
			c.FilePath = ""
			out[name] = c
		}
		return out
	}
	return []typeArgWorld{
		{"scanned library", func(t *testing.T) (*TypeResolver, *libraryClasses) {
			return NewTypeResolver(append(typeArgApplication(), typeArgLibrary()...)), nil
		}},
		{"scanned library, dependency resolver set", func(t *testing.T) (*TypeResolver, *libraryClasses) {
			r := NewTypeResolver(append(typeArgApplication(), typeArgLibrary()...))
			library := &libraryClasses{}
			r.SetDependencyResolver(library)
			return r, library
		}},
		{"dependency library", func(t *testing.T) (*TypeResolver, *libraryClasses) {
			r := NewTypeResolver(typeArgApplication())
			library := &libraryClasses{classes: dependencies()}
			r.SetDependencyResolver(library)
			return r, library
		}},
		{"dependency library, its Item read first", func(t *testing.T) (*TypeResolver, *libraryClasses) {
			r := NewTypeResolver(typeArgApplication())
			library := &libraryClasses{classes: dependencies()}
			r.SetDependencyResolver(library)
			if got := typeArgShape(r.Resolve("com.example.lib.Item", nil)); got != "Item(code)" {
				t.Fatalf("the library's Item = %s", got)
			}
			return r, library
		}},
	}
}

func (w typeArgWorld) isDependency() bool { return strings.HasPrefix(w.name, "dependency") }

// typeArgViews resolve a type the way a JSON body is written and the way a
// request parameter is bound.
var typeArgViews = map[string]func(r *TypeResolver, source parser.Class, typeName string) *model.ObjectModel{
	"json": func(r *TypeResolver, source parser.Class, typeName string) *model.ObjectModel {
		return r.ResolveFrom(source, typeName, nil)
	},
	"binding": func(r *TypeResolver, source parser.Class, typeName string) *model.ObjectModel {
		return r.ResolveForBinding(source, typeName, nil)
	},
}

func typeArgController() parser.Class {
	for _, c := range typeArgApplication() {
		if c.Name == "Controller" {
			return c
		}
	}
	panic("no controller")
}

func typeArgClass(name string) parser.Class {
	for _, c := range append(typeArgApplication(), typeArgLibrary()...) {
		if c.Name == name && c.Package != typeArgDtoPackage {
			return c
		}
	}
	panic("no class " + name)
}

func forEachTypeArgCase(t *testing.T, run func(t *testing.T, w typeArgWorld, view string, r *TypeResolver, library *libraryClasses)) {
	t.Helper()
	for _, w := range typeArgWorlds() {
		for view := range typeArgViews {
			t.Run(w.name+"/"+view, func(t *testing.T) {
				r, library := w.build(t)
				r.SetUnresolved(collector.NewUnresolvedSet())
				run(t, w, view, r, library)
			})
		}
	}
}

// The argument is the class the caller's scope gives its name, however the
// argument is written, and at whatever depth the generic classes pass it on.
func TestResolve_ATypeArgumentIsTheClassTheCallerNames(t *testing.T) {
	// What the argument is, by how the caller writes it.
	arguments := []struct{ spelling, shape string }{
		{"Item", "Item(name)"},
		{"com.example.app.dto.Item", "Item(name)"},
		{"com.example.lib.Item", "Item(code)"},
		{"Item[]", "[Item(name)]"},
		{"Item[][]", "[[Item(name)]]"},
		{"com.example.app.dto.Item[]", "[Item(name)]"},
		{"List<Item>", "[Item(name)]"},
		{"List<com.example.lib.Item>", "[Item(code)]"},
		{"Map<String, Item>", "{Item(name)}"},
		{"List<Item>[]", "[[Item(name)]]"},
		{"? extends Item", "Item(name)"},
		{"? extends com.example.app.dto.Item", "Item(name)"},
		{"Integer", "int"},
		{"String", "string"},
		{"Long[]", "[long]"},
		{"int[]", "[int]"},
		{"List<Integer>", "[int]"},
	}
	forEachTypeArgCase(t, func(t *testing.T, w typeArgWorld, view string, r *TypeResolver, library *libraryClasses) {
		caller := typeArgController()
		for _, arg := range arguments {
			box := typeArgViews[view](r, caller, "Box<"+arg.spelling+">")
			for path, want := range map[string]string{
				"data":    arg.shape,
				"list":    "[" + arg.shape + "]",
				"array":   "[" + arg.shape + "]",
				"byKey":   "{" + arg.shape + "}",
				"bounded": "[" + arg.shape + "]",
			} {
				if got := typeArgShape(typeArgAt(box, path)); got != want {
					t.Errorf("Box<%s>.%s = %s, want %s", arg.spelling, path, got, want)
				}
			}

			// Two levels: Outer<T> hands its parameter to Wrapper<U>.
			outer := typeArgViews[view](r, caller, "Outer<"+arg.spelling+">")
			for path, want := range map[string]string{
				"wrapped.data":       arg.shape,
				"wrappedList.data":   "[" + arg.shape + "]",
				"wrappedArray.data":  "[" + arg.shape + "]",
				"wrappedMap.data":    "{" + arg.shape + "}",
				"paired.data.first":  arg.shape,
				"paired.data.second": "[" + arg.shape + "]",
			} {
				if got := typeArgShape(typeArgAt(outer, path)); got != want {
					t.Errorf("Outer<%s>.%s = %s, want %s", arg.spelling, path, got, want)
				}
			}

			// A parameter named like the one it is passed to.
			same := typeArgViews[view](r, caller, "Same<"+arg.spelling+">")
			if got := typeArgShape(typeArgAt(same, "inner.data")); got != arg.shape {
				t.Errorf("Same<%s>.inner.data = %s, want %s", arg.spelling, got, arg.shape)
			}

			// Three levels: the argument is itself a generic class.
			nested := typeArgViews[view](r, caller, "Box<Outer<"+arg.spelling+">>")
			if got := typeArgShape(typeArgAt(nested, "data.wrapped.data")); got != arg.shape {
				t.Errorf("Box<Outer<%s>>.data.wrapped.data = %s, want %s", arg.spelling, got, arg.shape)
			}
		}

		pair := typeArgViews[view](r, caller, "Pair<Item, com.example.lib.Item>")
		if first, second := typeArgShape(typeArgAt(pair, "first")), typeArgShape(typeArgAt(pair, "second")); first != "Item(name)" || second != "Item(code)" {
			t.Errorf("Pair<Item, com.example.lib.Item> = %s, %s; want Item(name), Item(code)", first, second)
		}
		// An argument that is a generic class itself, whose own arguments are
		// the parameter of the outer one.
		for spelling, want := range map[string][2]string{
			"Pair<Item, Integer>":                {"Item(name)", "int"},
			"Pair<String, Item>":                 {"string", "Item(name)"},
			"Pair<com.example.lib.Item, String>": {"Item(code)", "string"},
		} {
			nested := typeArgViews[view](r, caller, "Outer<"+spelling+">")
			first, second := typeArgShape(typeArgAt(nested, "wrapped.data.first")), typeArgShape(typeArgAt(nested, "wrapped.data.second"))
			if first != want[0] || second != want[1] {
				t.Errorf("Outer<%s>.wrapped.data = %s, %s; want %s, %s", spelling, first, second, want[0], want[1])
			}
		}
		swapped := typeArgViews[view](r, caller, "Pair<com.example.lib.Item, Item>")
		if first, second := typeArgShape(typeArgAt(swapped, "first")), typeArgShape(typeArgAt(swapped, "second")); first != "Item(code)" || second != "Item(name)" {
			t.Errorf("Pair<com.example.lib.Item, Item> = %s, %s; want Item(code), Item(name)", first, second)
		}

		if unresolved := r.Unresolved(); len(unresolved) != 0 {
			t.Errorf("unresolved = %v, want none", unresolved)
		}
	})
}

// An argument a library class spells in full inside its own package is the
// class it names, not the library's class of the same simple name.
func TestResolve_AQualifiedArgumentInALibraryClassKeepsItsIdentity(t *testing.T) {
	forEachTypeArgCase(t, func(t *testing.T, w typeArgWorld, view string, r *TypeResolver, library *libraryClasses) {
		holder := typeArgViews[view](r, typeArgController(), "Holder")
		for path, want := range map[string]string{
			"fixedList":        "[Item(name)]",
			"fixedBox.data":    "Item(name)",
			"fixedBox.list":    "[Item(name)]",
			"fixedArray":       "[Item(name)]",
			"fixedPair.first":  "Item(name)",
			"fixedPair.second": "Item(code)",
		} {
			if got := typeArgShape(typeArgAt(holder, path)); got != want {
				t.Errorf("Holder.%s = %s, want %s", path, got, want)
			}
		}
	})
}

// A superclass argument is written in the scope of the subclass, and binds a
// parameter that the superclass resolves in its own.
func TestResolve_ASuperclassArgumentIsTheClassTheSubclassNames(t *testing.T) {
	cases := []struct{ class, typeName, shape string }{
		{"Sub", "Sub", "Item(name)"},
		{"SubQualified", "SubQualified", "Item(name)"},
		{"SubArray", "SubArray", "[Item(name)]"},
		{"SubList", "SubList", "[Item(name)]"},
		{"SubInt", "SubInt", "int"},
		{"SubString", "SubString", "string"},
		{"SubGeneric", "SubGeneric<Item>", "[Item(name)]"},
		{"SubGeneric", "SubGeneric<com.example.lib.Item>", "[Item(code)]"},
		{"SubGeneric", "SubGeneric<Item[]>", "[[Item(name)]]"},
		// Through a second superclass of the library.
		{"Leaf", "Leaf", "Item(name)"},
		{"LeafListed", "LeafListed", "[Item(name)]"},
	}
	forEachTypeArgCase(t, func(t *testing.T, w typeArgWorld, view string, r *TypeResolver, library *libraryClasses) {
		if w.isDependency() {
			// The resolver reads the superclass of a class from the classes it
			// already holds, so the library's classes are read first.
			for _, name := range []string{"Base", "Mid", "Listed", "DepSub"} {
				r.Resolve("com.example.lib."+name, nil)
			}
		}
		for _, tc := range cases {
			source := typeArgClass(tc.class)
			got := typeArgViews[view](r, source, tc.typeName)
			for path, want := range map[string]string{
				"value":        tc.shape,
				"list":         "[" + tc.shape + "]",
				"array":        "[" + tc.shape + "]",
				"wrapped.data": tc.shape, // the parameter handed on to a second generic class
			} {
				if shape := typeArgShape(typeArgAt(got, path)); shape != want {
					t.Errorf("%s.%s = %s, want %s", tc.typeName, path, shape, want)
				}
			}
		}
		if got := typeArgShape(typeArgAt(typeArgViews[view](r, typeArgController(), "Leaf"), "midValue")); got != "Item(name)" {
			t.Errorf("Leaf.midValue = %s, want Item(name)", got)
		}

		// A subclass the library declares names its argument in full.
		depSub := typeArgViews[view](r, typeArgController(), "com.example.lib.DepSub")
		if got := typeArgShape(typeArgAt(depSub, "value")); got != "Item(name)" {
			t.Errorf("DepSub.value = %s, want Item(name)", got)
		}
	})
}

// What the resolver answers for itself before it consults any scope keeps its
// meaning when it is passed as an argument, even where the generic class's
// package declares a class of that name.
func TestResolve_BuiltInNamesAsArgumentsKeepTheirMeaning(t *testing.T) {
	f := typeArgField
	classes := []parser.Class{
		typeArgItem(typeArgDtoPackage, "name"),
		// The library declares classes with the names of a scalar, a collection
		// and a map.
		{Name: "String", Package: typeArgLibPackage, FilePath: "lib/String.java", Fields: []parser.Field{f("shadow", "int")}},
		{Name: "Integer", Package: typeArgLibPackage, FilePath: "lib/Integer.java", Fields: []parser.Field{f("shadow", "int")}},
		{Name: "List", Package: typeArgLibPackage, FilePath: "lib/List.java", Fields: []parser.Field{f("shadow", "int")}},
		{Name: "Map", Package: typeArgLibPackage, FilePath: "lib/Map.java", Fields: []parser.Field{f("shadow", "int")}},
		typeArgLibraryClass("Box", []string{"T"}, "", nil, f("data", "T")),
		// The caller's package declares them too.
		{Name: "List", Package: typeArgAppPackage, FilePath: "app/List.java", Fields: []parser.Field{f("own", "int")}},
		{Name: "Map", Package: typeArgAppPackage, FilePath: "app/Map.java", Fields: []parser.Field{f("own", "int")}},
		{Name: "Controller", Package: typeArgAppPackage, FilePath: "app/Controller.java",
			Imports: []parser.Import{{Path: typeArgDtoPackage + ".Item"}, {Path: typeArgLibPackage + ".Box"}}},
	}
	r := NewTypeResolver(classes)
	caller := classes[len(classes)-1]
	for spelling, want := range map[string]string{
		"String":            "string",
		"Integer":           "int",
		"List<Item>":        "[Item(name)]",
		"Map<String, Item>": "{Item(name)}",
		"List<String>":      "[string]",
	} {
		if got := typeArgShape(typeArgAt(r.ResolveFrom(caller, "Box<"+spelling+">", nil), "data")); got != want {
			t.Errorf("Box<%s>.data = %s, want %s", spelling, got, want)
		}
	}
}

// A type that no body is read for, an enum of the JSON view or a class nothing
// declares, is shown as it is written wherever the caller uses it, and reported
// under that name: the argument of a generic class, of a collection or of a map
// is no reason to show it another way, nor to report it twice. With a
// dependency resolver, which asks for a class by its qualified name, the
// argument of a generic class is written in full.
func TestResolve_ATypeWithoutABodyIsShownAsWrittenInATypeArgument(t *testing.T) {
	f := typeArgField
	classes := []parser.Class{
		{Name: "Status", Package: "com.example.other", IsEnum: true, EnumConstants: []string{"OPEN"}},
		typeArgLibraryClass("Box", []string{"T"}, "", nil, f("data", "T"), f("list", "List<T>")),
		{Name: "Order", Package: typeArgAppPackage, FilePath: "app/Order.java",
			Imports: []parser.Import{
				{Path: "com.example.other.Status"}, {Path: "com.example.other.Lost"}, {Path: typeArgLibPackage + ".Box"},
				{Path: "java.util.List"}, {Path: "java.util.Map"},
			},
			Fields: []parser.Field{
				f("direct", "Status"), f("statuses", "List<Status>"), f("byKey", "Map<String, Lost>"),
				f("boxed", "Box<Status>"), f("lost", "Box<List<Lost>>"),
			}},
	}
	for name, withDependencies := range map[string]bool{"without a dependency resolver": false, "with a dependency resolver": true} {
		t.Run(name, func(t *testing.T) {
			r := NewTypeResolver(classes)
			r.SetUnresolved(collector.NewUnresolvedSet())
			if withDependencies {
				r.SetDependencyResolver(&libraryClasses{})
			}
			order := r.Resolve("com.example.app.Order", nil)

			boxedName, lostName := "Status", "Lost"
			if withDependencies {
				boxedName, lostName = "com.example.other.Status", "com.example.other.Lost"
			}
			for path, want := range map[string]string{
				"direct":     "Status",
				"statuses":   "[Status]",
				"byKey":      "{Lost}",
				"boxed.data": boxedName,
				"boxed.list": "[" + boxedName + "]",
				"lost.data":  "[" + lostName + "]",
				"lost.list":  "[[" + lostName + "]]",
			} {
				if got := typeArgShape(typeArgAt(order, path)); got != want {
					t.Errorf("Order.%s = %s, want %s", path, got, want)
				}
			}
			names := map[string]bool{}
			for unresolved := range r.Unresolved() {
				names[unresolved] = true
			}
			want := map[string]bool{"Status": true, "Lost": true}
			if withDependencies {
				// The map's value is not the argument of a generic class.
				want = map[string]bool{"Status": true, "Lost": true, "com.example.other.Status": true, "com.example.other.Lost": true}
			}
			if !reflect.DeepEqual(names, want) {
				t.Errorf("unresolved = %v, want %v", names, want)
			}
		})
	}
}

// A name that cannot be resolved where the caller writes it stays as written.
// A class the caller imports without its body being known is still that class,
// not a class of the generic class's package that has the same simple name.
func TestResolve_AnUnresolvedTypeArgumentKeepsItsSpelling(t *testing.T) {
	f := typeArgField
	classes := []parser.Class{
		typeArgLibraryClass("Box", []string{"T"}, "", nil, f("data", "T")),
		typeArgLibraryClass("Gone", nil, "", nil, f("own", "String")),
		{Name: "Controller", Package: typeArgAppPackage, FilePath: "app/Controller.java",
			Imports: []parser.Import{{Path: typeArgLibPackage + ".Box"}, {Path: "com.example.elsewhere.Gone"}}},
	}
	r := NewTypeResolver(classes)
	r.SetUnresolved(collector.NewUnresolvedSet())
	caller := classes[len(classes)-1]

	missing := r.ResolveFrom(caller, "Box<Missing>", nil)
	if got := typeArgShape(typeArgAt(missing, "data")); got != "Missing" {
		t.Errorf("Box<Missing>.data = %s, want the name as written", got)
	}
	// The import names a class of another package; the library's Gone is not it.
	gone := r.ResolveFrom(caller, "Box<Gone>", nil)
	if got := typeArgShape(typeArgAt(gone, "data")); got != "com.example.elsewhere.Gone" {
		t.Errorf("Box<Gone>.data = %s, want the imported class, whose body is not known", got)
	}
	unresolved := r.Unresolved()
	if unresolved["Missing"] == 0 || unresolved["com.example.elsewhere.Gone"] == 0 || len(unresolved) != 2 {
		t.Errorf("unresolved = %v, want Missing and com.example.elsewhere.Gone", unresolved)
	}
}

// The type variables of the caller that nothing binds stay open, as they are
// when the arguments are not looked at: the field is marked generic.
func TestResolve_AnUnboundTypeVariableOfTheCallerStaysOpen(t *testing.T) {
	r := NewTypeResolver(append(typeArgApplication(), typeArgLibrary()...))
	r.SetUnresolved(collector.NewUnresolvedSet())
	outer := r.ResolveFrom(typeArgController(), "Outer", nil)
	data := outer.Fields["wrapped"].Model.Fields["data"]
	if data == nil || !data.Generic {
		t.Errorf("Outer.wrapped.data = %+v, want a generic field", data)
	}
}

// Qualifying an argument consults what the resolver already knows and nothing
// else: a name bound in scope costs no call to the dependencies.
func TestResolve_QualifyingTypeArgumentsAsksNothingOfTheDependencies(t *testing.T) {
	r := NewTypeResolver(append(typeArgApplication(), typeArgLibrary()...))
	library := &libraryClasses{}
	r.SetDependencyResolver(library)

	for _, typeName := range []string{
		"Box<Item>", "Outer<Item[]>", "Box<List<Item>>", "Box<Map<String, Item>>", "Box<? extends Item>",
		"Box<Integer>", "Outer<Integer>", "Pair<Item, com.example.lib.Item>", "Sub", "Leaf",
	} {
		r.ResolveFrom(typeArgController(), typeName, nil)
	}
	if len(library.asked) != 0 {
		t.Errorf("the dependencies were asked for %v; every name was bound by a class the resolver holds", library.asked)
	}
}
