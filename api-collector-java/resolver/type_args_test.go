package resolver

import (
	"reflect"
	"testing"

	"github.com/tangcent/apilot/api-collector-java/parser"
	model "github.com/tangcent/apilot/api-model"
)

// libraryClasses answers dependency lookups by qualified name only, as the
// indexer does: a simple name finds nothing.
type libraryClasses struct {
	classes map[string]parser.Class
	asked   []string
}

func (l *libraryClasses) ResolveClass(name string) *parser.Class {
	l.asked = append(l.asked, name)
	if c, ok := l.classes[name]; ok {
		return &c
	}
	return nil
}

func TestResolve_TypeArgumentsAreLookedUpByTheNameTheCallerImports(t *testing.T) {
	library := &libraryClasses{classes: map[string]parser.Class{
		"com.example.lib.Wrapper": {
			Name: "Wrapper", Package: "com.example.lib", TypeParameters: []string{"T"},
			Fields: []parser.Field{{Name: "code", Type: "int"}, {Name: "data", Type: "T"}},
		},
		"com.example.lib.PageData": {
			Name: "PageData", Package: "com.example.lib", TypeParameters: []string{"T"},
			Fields: []parser.Field{{Name: "total", Type: "long"}, {Name: "rows", Type: "List<T>"}},
		},
	}}
	// The controller whose return types are resolved, as an endpoint's are.
	controller := parser.Class{
		Name: "OrderController", Package: "com.example.app", FilePath: "OrderController.java",
		Imports: []parser.Import{
			{Path: "com.example.lib.Wrapper"}, {Path: "com.example.lib.PageData"}, {Path: "java.util.List"},
		},
	}
	order := parser.Class{
		Name: "Order", Package: "com.example.app", FilePath: "Order.java",
		Fields: []parser.Field{{Name: "id", Type: "long"}},
	}
	r := NewTypeResolver([]parser.Class{controller, order})
	r.SetDependencyResolver(library)

	page := r.ResolveFrom(controller, "Wrapper<PageData<Order>>", nil)
	data := page.Fields["data"].Model
	if data == nil || data.Kind != model.KindObject {
		t.Fatalf("data = %+v, want PageData expanded", data)
	}
	rows := data.Fields["rows"].Model
	if rows.Kind != model.KindArray || rows.Items.Fields["id"] == nil {
		t.Fatalf("data.rows = %+v, want an array of Order", rows)
	}
	list := r.ResolveFrom(controller, "Wrapper<List<Order>>", nil).Fields["data"].Model
	if list.Kind != model.KindArray || list.Items.Fields["id"] == nil {
		t.Fatalf("data = %+v, want an array of Order: java.util.List stays a collection", list)
	}
	for _, name := range library.asked {
		if name == "PageData" {
			t.Errorf("PageData was asked for by its simple name; asked %v", library.asked)
		}
	}
}

func TestQualifyTypeArgs(t *testing.T) {
	source := parser.Class{
		Name: "Holder", Package: "com.example.app",
		Imports: []parser.Import{{Path: "com.example.lib.PageData"}, {Path: "java.util.List"}, {Path: "kotlin.collections.Map"}},
	}
	r := NewTypeResolver(nil)
	// A class of a dependency can only be asked for by its qualified name.
	r.SetDependencyResolver(&libraryClasses{})
	got := r.qualifyTypeArgs(source, nil, []string{"PageData<Unknown>", "List<PageData>", "Map<String, PageData[]>", "?", "T", "? extends PageData"})
	want := []string{
		"com.example.lib.PageData<Unknown>",
		"List<com.example.lib.PageData>",
		"Map<String, com.example.lib.PageData[]>",
		"?",
		"T",
		"? extends com.example.lib.PageData",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("qualifyTypeArgs:\n got  %q\n want %q", got, want)
	}
}

// A name that no body is read for is written in full only where it needs to
// be: with a dependency resolver, which asks for a class by its qualified name,
// or when another class shares the simple name. Otherwise it is shown, and
// reported when it cannot be resolved, as it is written.
func TestQualifyTypeName_ANameWithoutABodyIsWrittenInFullWhereItNeedsToBe(t *testing.T) {
	source := parser.Class{
		Name: "Holder", Package: "com.example.app",
		Imports: []parser.Import{{Path: "com.example.lib.Lost"}, {Path: "com.example.lib.Shared"}},
	}
	classes := []parser.Class{
		{Name: "Shared", Package: "com.example.other"},
		{Name: "Status", Package: "com.example.lib", IsEnum: true, EnumConstants: []string{"OPEN"}},
	}
	for name, tc := range map[string]struct {
		dependencies bool
		want         map[string]string
	}{
		"without a dependency resolver": {false, map[string]string{
			"Lost":   "Lost",                   // nothing else is called Lost
			"Shared": "com.example.lib.Shared", // the class of another package is called Shared too
			"Status": "Status",                 // not imported: nothing in scope names it
		}},
		"with a dependency resolver": {true, map[string]string{
			"Lost":   "com.example.lib.Lost",
			"Shared": "com.example.lib.Shared",
			"Status": "Status",
		}},
	} {
		t.Run(name, func(t *testing.T) {
			r := NewTypeResolver(classes)
			if tc.dependencies {
				r.SetDependencyResolver(&libraryClasses{})
			}
			for typeName, want := range tc.want {
				if got := r.qualifyTypeName(source, nil, typeName); got != want {
					t.Errorf("qualifyTypeName(%q) = %q, want %q", typeName, got, want)
				}
			}
		})
	}
}

// An argument is spelled so that it means the same wherever it is read: the
// classes it names are written in full, and what the resolver answers for by
// the spelling alone (scalars, collections, maps) is left as written.
func TestQualifyTypeName_Shapes(t *testing.T) {
	source := parser.Class{
		Name: "Holder", Package: typeArgAppPackage,
		Imports: []parser.Import{{Path: typeArgDtoPackage + ".Item"}, {Path: "java.util.List"}},
	}
	r := NewTypeResolver([]parser.Class{typeArgItem(typeArgDtoPackage, "name")})
	for typeName, want := range map[string]string{
		"Item":                      "com.example.app.dto.Item",
		"Item[]":                    "com.example.app.dto.Item[]",
		"Item[][]":                  "com.example.app.dto.Item[][]",
		"List<Item>":                "List<com.example.app.dto.Item>",
		"List<Item>[]":              "List<com.example.app.dto.Item>[]",
		"Map<String,Item>":          "Map<String, com.example.app.dto.Item>",
		"Map<String, List<Item[]>>": "Map<String, List<com.example.app.dto.Item[]>>",
		"? extends Item":            "? extends com.example.app.dto.Item",
		"? super Item":              "? super com.example.app.dto.Item",
		"?":                         "?",
		"com.example.lib.Item":      "com.example.lib.Item",
		"Integer":                   "Integer",
		"int[]":                     "int[]",
		"Unknown":                   "Unknown",
		"Unknown<Item>":             "Unknown<com.example.app.dto.Item>",
	} {
		if got := r.qualifyTypeName(source, nil, typeName); got != want {
			t.Errorf("qualifyTypeName(%q) = %q, want %q", typeName, got, want)
		}
	}
}

// A type variable of the caller is replaced by what it is bound to, once: the
// binding is an argument already spelled in full. One that nothing binds is
// left alone, whatever class shares its name.
func TestQualifyTypeName_TypeVariablesOfTheCaller(t *testing.T) {
	r := NewTypeResolver([]parser.Class{typeArgItem(typeArgDtoPackage, "name")})
	caller := parser.Class{Name: "Result", Package: typeArgAppPackage, TypeParameters: []string{"T", "Item"}}
	got := r.qualifyTypeArgs(caller, map[string]string{"T": "com.example.app.dto.Item[]"}, []string{"T", "List<T>", "Item", "Map<String, T[]>"})
	want := []string{"com.example.app.dto.Item[]", "List<com.example.app.dto.Item[]>", "Item", "Map<String, com.example.app.dto.Item[][]>"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("qualifyTypeArgs:\n got  %q\n want %q", got, want)
	}
}

// The bindings a controller applies to the methods it inherits are the
// superclass's parameters bound to the arguments of the extends clause, spelled
// so that the superclass can resolve them.
func TestSuperClassBindings(t *testing.T) {
	r := NewTypeResolver(append(typeArgApplication(), typeArgLibrary()...))
	base := typeArgClass("Base")
	for _, tc := range []struct {
		class    string
		bindings map[string]string
		want     map[string]string
	}{
		{"Sub", nil, map[string]string{"B": "com.example.app.dto.Item"}},
		{"SubQualified", nil, map[string]string{"B": "com.example.app.dto.Item"}},
		{"SubArray", nil, map[string]string{"B": "com.example.app.dto.Item[]"}},
		{"SubList", nil, map[string]string{"B": "List<com.example.app.dto.Item>"}},
		{"SubInt", nil, map[string]string{"B": "Integer"}},
		{"SubGeneric", map[string]string{"S": "com.example.lib.Item"}, map[string]string{"B": "List<com.example.lib.Item>"}},
		// A parameter nobody bound stays a name.
		{"SubGeneric", nil, map[string]string{"B": "List<S>"}},
	} {
		got := r.SuperClassBindings(typeArgClass(tc.class), base, tc.bindings)
		if !reflect.DeepEqual(got, tc.want) {
			t.Errorf("SuperClassBindings(%s, %v) = %v, want %v", tc.class, tc.bindings, got, tc.want)
		}
	}
}

// The JSON view does not see enums, so an enum field of a scanned class misses
// the class table. It is still a declaration of the project: asking the
// dependency index for it would read some installed release of the project
// itself and report it as a dependency type.
func TestResolve_AProjectEnumIsNotAskedOfTheDependencies(t *testing.T) {
	library := &libraryClasses{classes: map[string]parser.Class{
		"com.example.app.Status": {Name: "Status", Package: "com.example.app", IsEnum: true, EnumConstants: []string{"OPEN"}},
	}}
	r := NewTypeResolver([]parser.Class{
		{Name: "Status", Package: "com.example.app", IsEnum: true, EnumConstants: []string{"OPEN"}},
		{Name: "Order", Package: "com.example.app", Fields: []parser.Field{{Name: "status", Type: "com.example.app.Status"}}},
	})
	r.SetDependencyResolver(library)

	order := r.Resolve("com.example.app.Order", nil)
	if status := order.Fields["status"].Model; status.TypeName != "com.example.app.Status" {
		t.Errorf("status = %+v, want the enum's name, as the JSON view exports it", status)
	}
	if len(library.asked) != 0 {
		t.Errorf("the dependencies were asked for %v", library.asked)
	}
}
