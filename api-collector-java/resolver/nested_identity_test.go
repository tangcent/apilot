package resolver

import (
	"reflect"
	"testing"

	"github.com/tangcent/apilot/api-collector-java/parser"
	model "github.com/tangcent/apilot/api-model"
)

// nestedWorld is the setting of the identity tests: the application's Order, with
// a Status nested in it, and the library's own Order, which nests a Status too, and
// a top-level Status next to it. The library's generic classes sit in the package
// of the second, where Order, Order.Status and Status mean the library's.
type nestedWorld struct {
	scanned []parser.Class
	library *nestedLibrary
}

func newNestedWorld() *nestedWorld {
	appOrder := parser.Class{Name: "Order", Package: "p", FilePath: "p/Order.java"}
	appStatus := parser.Class{Name: "Status", Package: "p", Outer: "Order", FilePath: "p/Order.java", Fields: []parser.Field{{Name: "fromP", Type: "String"}}}
	// Another class of the application named Order.Status, in a package nobody here
	// imports: the fallback by simple name must not pick it.
	otherOrder := parser.Class{Name: "Order", Package: "other", FilePath: "other/Order.java"}
	otherStatus := parser.Class{Name: "Status", Package: "other", Outer: "Order", FilePath: "other/Order.java", Fields: []parser.Field{{Name: "fromOther", Type: "String"}}}

	libOrder := parser.Class{Name: "Order", Package: "lib"}
	libStatus := parser.Class{Name: "Status", Package: "lib", Outer: "Order", Fields: []parser.Field{{Name: "fromLib", Type: "String"}}}
	libTop := parser.Class{Name: "Status", Package: "lib", Fields: []parser.Field{{Name: "fromLibTop", Type: "String"}}}
	box := parser.Class{Name: "Box", Package: "lib", TypeParameters: []string{"T"}, Fields: []parser.Field{
		{Name: "value", Type: "T"}, {Name: "list", Type: "List<T>"}, {Name: "array", Type: "T[]"}, {Name: "byKey", Type: "Map<String, T>"},
	}}
	page := parser.Class{Name: "Page", Package: "lib", TypeParameters: []string{"U"}, Fields: []parser.Field{{Name: "inner", Type: "Box<U>"}}}
	base := parser.Class{Name: "Base", Package: "lib", TypeParameters: []string{"T"}, Fields: []parser.Field{{Name: "data", Type: "T"}}}

	return &nestedWorld{
		scanned: []parser.Class{appOrder, appStatus, otherOrder, otherStatus},
		library: &nestedLibrary{
			classes: map[string]parser.Class{
				"lib.Order": libOrder, "lib.Order.Status": libStatus, "lib.Status": libTop,
				"lib.Box": box, "lib.Page": page, "lib.Base": base,
			},
			siblings: map[string][]parser.Class{"lib.Order": {libStatus}},
		},
	}
}

// The type argument of a generic class is the class its caller names, and a
// nested class keeps its identity: `Order.Status` written in a source is the
// Status nested in the Order that source's scope names, and neither the Status
// nested in the Order of the library's package, nor the top-level Status there, nor
// a Status nested in an Order of the project that the source does not reach, takes
// its place. It holds however the name is bound, spelled, and wrapped.
func TestResolve_ANestedClassKeepsItsIdentityAsATypeArgument(t *testing.T) {
	bindings := []struct {
		name   string
		source parser.Class
		order  string // how the source spells the outer class
	}{
		{"imported", parser.Class{Name: "Ctl", Package: "app", FilePath: "Ctl.java", Imports: []parser.Import{{Path: "p.Order"}, {Path: "lib.Box"}, {Path: "lib.Page"}, {Path: "lib.Base"}}}, "Order"},
		{"imported on demand", parser.Class{Name: "Ctl", Package: "app", FilePath: "Ctl.java", Imports: []parser.Import{{Path: "p", OnDemand: true}, {Path: "lib.Box"}, {Path: "lib.Page"}, {Path: "lib.Base"}}}, "Order"},
		{"in the package of the source", parser.Class{Name: "Ctl", Package: "p", FilePath: "p/Ctl.java", Imports: []parser.Import{{Path: "lib.Box"}, {Path: "lib.Page"}, {Path: "lib.Base"}}}, "Order"},
		{"imported by the path of the nested class", parser.Class{Name: "Ctl", Package: "app", FilePath: "Ctl.java", Imports: []parser.Import{{Path: "p.Order.Status"}, {Path: "lib.Box"}, {Path: "lib.Page"}, {Path: "lib.Base"}}}, "Status"},
		{"imported on demand from the outer class", parser.Class{Name: "Ctl", Package: "app", FilePath: "Ctl.java", Imports: []parser.Import{{Path: "p.Order", OnDemand: true}, {Path: "lib.Box"}, {Path: "lib.Page"}, {Path: "lib.Base"}}}, "Status"},
	}
	spellings := []struct {
		name string
		make func(order string) string
	}{
		{"simple", func(order string) string { return order + ".Status" }},
		{"binary", func(order string) string { return order + "$Status" }},
		{"qualified", func(string) string { return "p.Order.Status" }},
		{"qualified binary", func(string) string { return "p.Order$Status" }},
	}
	for _, binding := range bindings {
		for _, spelling := range spellings {
			arg := spelling.make(binding.order)
			if binding.order == "Status" {
				// Imported by path: the nested class is named by its own name.
				if spelling.name != "simple" {
					continue
				}
				arg = "Status"
			}
			for _, withDependencies := range []bool{false, true} {
				name := binding.name + ", " + spelling.name
				if withDependencies {
					name += ", with dependencies"
				}
				t.Run(name, func(t *testing.T) {
					world := newNestedWorld()
					source := binding.source
					r := NewTypeResolver(append(append([]parser.Class{}, world.scanned...), source))
					if withDependencies {
						r.SetDependencyResolver(world.library)
					} else {
						// The library's classes are declared in the scanned sources.
						r = NewTypeResolver(append(append(append([]parser.Class{}, world.scanned...), source), libraryAsScanned(world)...))
					}

					for what, rawType := range map[string]string{
						"Box<arg>":           "Box<" + arg + ">",
						"Box<arg[]>":         "Box<" + arg + "[]>",
						"Box<? extends arg>": "Box<? extends " + arg + ">",
						"Page<arg>":          "Page<" + arg + ">",
						"Page<List<arg>>":    "Page<List<" + arg + ">>",
						"List<Box<arg>>":     "List<Box<" + arg + ">>",
					} {
						got := r.ResolveFrom(source, rawType, nil)
						for _, leaf := range statusLeaves(got) {
							if !reflect.DeepEqual(leaf, []string{"fromP"}) {
								t.Errorf("%s = %s: a leaf has %v, want the application's Order.Status [fromP]", what, shapeOf(got), leaf)
							}
						}
						if len(statusLeaves(got)) == 0 {
							t.Errorf("%s = %s, want the application's Order.Status inside", what, shapeOf(got))
						}
					}
				})
			}
		}
	}
}

// libraryAsScanned declares the classes of the library among the scanned sources,
// each in a file of its own and the classes nested in a class in its file.
func libraryAsScanned(world *nestedWorld) []parser.Class {
	var out []parser.Class
	for _, key := range []string{"lib.Order", "lib.Order.Status", "lib.Status", "lib.Box", "lib.Page", "lib.Base"} {
		c := world.library.classes[key]
		if c.Outer != "" {
			c.FilePath = "lib/" + c.Outer + ".java"
		} else {
			c.FilePath = "lib/" + c.Name + ".java"
		}
		out = append(out, c)
	}
	return out
}

// statusLeaves lists the field names of every object in a model that is one of
// the Status classes of nestedWorld, told apart by the field each declares.
func statusLeaves(m *model.ObjectModel) [][]string {
	if m == nil {
		return nil
	}
	var out [][]string
	switch {
	case m.IsArray():
		return statusLeaves(m.Items)
	case m.IsMap():
		return statusLeaves(m.ValueModel)
	case m.IsObject():
		names := fieldsOf(m)
		for _, name := range names {
			switch name {
			case "fromP", "fromLib", "fromLibTop", "fromOther":
				return [][]string{names}
			}
		}
		for _, name := range names {
			out = append(out, statusLeaves(m.Fields[name].Model)...)
		}
	}
	return out
}

// The argument of a superclass is spelled once, in the header of the subclass: the
// member types of the subclass are not in scope there, the classes around it are,
// and the class that reads the argument (the superclass, in its package) sees none
// of that. The superclass may be a class of the project or a dependency's, and a
// nested class named in the argument keeps its identity.
func TestResolve_ASuperclassArgumentIsSpelledInTheHeaderOfTheSubclass(t *testing.T) {
	wrapper := parser.Class{Name: "Wrapper", Package: "q", TypeParameters: []string{"T"}, Fields: []parser.Field{{Name: "data", Type: "T"}, {Name: "own", Type: "Item"}}}
	libItem := marked("q", "Item", "fromLibrary")
	projectItem := marked("app.dto", "Item", "fromProject")
	projectItem.FilePath = "Item.java"
	// A class nested in Holder that the header does not see.
	nestedItem := parser.Class{Name: "Item", Package: "p", Outer: "Holder", FilePath: "Holder.java", Fields: []parser.Field{{Name: "fromNested", Type: "String"}}}
	// A class nested in the class around Holder, which the header does see.
	outer := parser.Class{Name: "Outer", Package: "p", FilePath: "Holder.java"}
	outerRow := parser.Class{Name: "Row", Package: "p", Outer: "Outer", FilePath: "Holder.java", Fields: []parser.Field{{Name: "fromOuter", Type: "String"}}}
	holder := parser.Class{
		Name: "Holder", Package: "p", Outer: "Outer", FilePath: "Holder.java", SuperClass: "Wrapper", SuperClassTypeArgs: []string{"Item"},
		Imports: []parser.Import{{Path: "q.Wrapper"}, {Path: "app.dto.Item"}},
	}
	rowHolder := holder
	rowHolder.Name, rowHolder.SuperClassTypeArgs = "RowHolder", []string{"Row"}
	caller := parser.Class{Name: "Caller", Package: "app", FilePath: "Caller.java"}

	for _, dependency := range []bool{false, true} {
		name := "the superclass is a class of the project"
		if dependency {
			name = "the superclass is a class of the dependencies"
		}
		t.Run(name, func(t *testing.T) {
			scanned := []parser.Class{caller, outer, outerRow, holder, rowHolder, nestedItem, projectItem}
			var r *TypeResolver
			if dependency {
				r = NewTypeResolver(scanned)
				r.SetDependencyResolver(&nestedLibrary{classes: map[string]parser.Class{"q.Wrapper": wrapper, "q.Item": libItem}})
			} else {
				wrapper.FilePath, libItem.FilePath = "Wrapper.java", "QItem.java"
				r = NewTypeResolver(append(scanned, wrapper, libItem))
			}

			got := r.ResolveFrom(caller, "p.Outer.Holder", nil)
			if fields := fieldsOf(got.Fields["data"].Model); !reflect.DeepEqual(fields, []string{"fromProject"}) {
				t.Errorf("Holder.data has %v, want the imported Item [fromProject], not Holder's own", fields)
			}
			if fields := fieldsOf(got.Fields["own"].Model); !reflect.DeepEqual(fields, []string{"fromLibrary"}) {
				t.Errorf("Holder.own has %v, want the superclass's own Item [fromLibrary]", fields)
			}
			got = r.ResolveFrom(caller, "p.Outer.RowHolder", nil)
			if fields := fieldsOf(got.Fields["data"].Model); !reflect.DeepEqual(fields, []string{"fromOuter"}) {
				t.Errorf("RowHolder.data has %v, want Outer.Row [fromOuter]: the classes around the header are in scope", fields)
			}
		})
	}
}

// A class named with a `$` and a class nested in another that spells the same way
// are told apart by how the name is written: `Price$Value` is the class a project
// declares under that name, and `Price.Value` is the class nested in Price,
// wherever each is reached from.
func TestResolve_ADollarNameAndANestedClassOfTheSameSpellingAreTwoClasses(t *testing.T) {
	classes := []parser.Class{
		{Name: "Price", Package: "p", FilePath: "Price.java"},
		{Name: "Value", Package: "p", Outer: "Price", FilePath: "Price.java", Fields: []parser.Field{{Name: "fromNested", Type: "String"}}},
		{Name: "Price$Value", Package: "p", FilePath: "Price$Value.java", Fields: []parser.Field{{Name: "fromLiteral", Type: "String"}}},
		{Name: "Order", Package: "p", FilePath: "Order.java"},
		{Name: "Holder", Package: "app", FilePath: "Holder.java", Imports: []parser.Import{{Path: "p.Price"}, {Path: "p.Price$Value"}}},
		{Name: "Starred", Package: "app", FilePath: "Starred.java", Imports: []parser.Import{{Path: "p", OnDemand: true}}},
	}
	for _, source := range []parser.Class{classes[3], classes[4], classes[5]} {
		t.Run(source.Name, func(t *testing.T) {
			r := NewTypeResolver(classes)
			for rawType, want := range map[string]string{
				"Price$Value":              "fromLiteral",
				"p.Price$Value":            "fromLiteral",
				"List<Price$Value>":        "fromLiteral",
				"List<p.Price$Value>":      "fromLiteral",
				"Price.Value":              "fromNested",
				"p.Price.Value":            "fromNested",
				"List<Price.Value>":        "fromNested",
				"List<p.Price.Value>":      "fromNested",
				"Price.Value[]":            "fromNested",
				"Price$Value[]":            "fromLiteral",
				"Map<String, Price.Value>": "fromNested",
			} {
				got := r.ResolveFrom(source, rawType, nil)
				leaf := got
				for leaf != nil && (leaf.IsArray() || leaf.IsMap()) {
					if leaf.IsArray() {
						leaf = leaf.Items
					} else {
						leaf = leaf.ValueModel
					}
				}
				if !reflect.DeepEqual(fieldsOf(leaf), []string{want}) {
					t.Errorf("%s from %s = %s, want %s", rawType, source.Name, shapeOf(got), want)
				}
			}
		})
	}
}

// A member type is inherited from the superclass through the header of the
// subclass (JLS 8.5), an enum as a class: a class of the same name in the package
// of the subclass, or one an import on demand brings in, does not take its place.
func TestResolve_AnEnumNestedInASuperclassIsInherited(t *testing.T) {
	classes := []parser.Class{
		{Name: "Base", Package: "q", FilePath: "Base.java"},
		{Name: "Mode", Package: "q", Outer: "Base", FilePath: "Base.java", IsEnum: true, EnumConstants: []string{"ON", "OFF"}},
		{Name: "Row", Package: "q", Outer: "Base", FilePath: "Base.java", Fields: []parser.Field{{Name: "fromBase", Type: "String"}}},
		marked("p", "Mode", "fromPackage"),
		marked("w", "Row", "fromWildcard"),
		{
			Name: "Child", Package: "p", FilePath: "Child.java", SuperClass: "Base",
			Imports: []parser.Import{{Path: "q.Base"}, {Path: "w", OnDemand: true}},
			Fields:  []parser.Field{{Name: "mode", Type: "Mode"}, {Name: "row", Type: "Row"}, {Name: "rows", Type: "List<Row>"}},
		},
	}
	classes[3].FilePath, classes[4].FilePath = "p/Mode.java", "w/Row.java"
	r := NewTypeResolver(classes)
	child := classes[5]

	got := r.ResolveFrom(child, "p.Child", nil)
	if mode := got.Fields["mode"].Model; mode.Kind != model.KindSingle {
		t.Errorf("Child.mode = %s, want the inherited enum Base.Mode (an opaque value in the JSON view)", shapeOf(mode))
	}
	assertShape(t, "Child.row", got.Fields["row"].Model, "{fromBase:string}")
	assertShape(t, "Child.rows", got.Fields["rows"].Model, "[{fromBase:string}]")
	if constants, ok := r.EnumConstants(child, "Mode"); !ok || !reflect.DeepEqual(constants, []string{"ON", "OFF"}) {
		t.Errorf("EnumConstants(Child, Mode) = %v, %v, want the constants of Base.Mode", constants, ok)
	}
}

// An import of a class that a library nests in another names that class, and the
// library is asked for it by its path; the class that declares it comes with it.
func TestResolve_AnImportOfANestedClassOfALibraryNamesIt(t *testing.T) {
	const pkg = "com.example.lib"
	controller := parser.Class{
		Name: "SortController", Package: nestedPkg, FilePath: "SortController.java",
		Imports: []parser.Import{{Path: pkg + ".Page.Sort"}},
	}
	library := newNestedLibrary()
	r := NewTypeResolver([]parser.Class{controller})
	r.SetDependencyResolver(library)

	assertShape(t, "Sort", r.ResolveFrom(controller, "Sort", nil), "{field:string,order:{asc:boolean}}")
	assertShape(t, "List<Sort>", r.ResolveFrom(controller, "List<Sort>", nil), "[{field:string,order:{asc:boolean}}]")
	if want := []string{pkg + ".Page.Sort"}; !reflect.DeepEqual(library.asked, want) {
		t.Errorf("asked = %v, want %v", library.asked, want)
	}
}

// An enum is a declaration like any other: as a type argument it stays the enum its
// caller names, and the JSON view, which does not read an enum, exports it by that
// name in full rather than by a simple name that the generic class's package may
// give to another enum. It holds one and two levels down, through a superclass
// argument, and for an enum nested in a class.
func TestResolve_AnEnumKeepsItsIdentityAsATypeArgument(t *testing.T) {
	classes := []parser.Class{
		{Name: "Envelope", Package: "a", FilePath: "a/Envelope.java", TypeParameters: []string{"T"}, Fields: []parser.Field{{Name: "data", Type: "T"}}},
		{Name: "Page", Package: "a", FilePath: "a/Page.java", TypeParameters: []string{"U"}, Fields: []parser.Field{{Name: "inner", Type: "Envelope<U>"}}},
		{Name: "Base", Package: "a", FilePath: "a/Base.java", TypeParameters: []string{"T"}, Fields: []parser.Field{{Name: "data", Type: "T"}}},
		{Name: "Status", Package: "a", FilePath: "a/Status.java", IsEnum: true, EnumConstants: []string{"FROM_A"}},
		{Name: "Status", Package: "b", FilePath: "b/Status.java", IsEnum: true, EnumConstants: []string{"FROM_B"}},
		{Name: "Order", Package: "b", FilePath: "b/Order.java"},
		{Name: "Kind", Package: "b", Outer: "Order", FilePath: "b/Order.java", IsEnum: true, EnumConstants: []string{"FROM_ORDER"}},
		{Name: "Kind", Package: "a", FilePath: "a/Kind.java", IsEnum: true, EnumConstants: []string{"FROM_A_KIND"}},
		{
			Name: "Sub", Package: "b", FilePath: "b/Sub.java", SuperClass: "Base", SuperClassTypeArgs: []string{"Status"},
			Imports: []parser.Import{{Path: "a.Base"}},
		},
		{
			Name: "Holder", Package: "b", FilePath: "b/Holder.java",
			Imports: []parser.Import{{Path: "a.Envelope"}, {Path: "a.Page"}},
			Fields: []parser.Field{
				{Name: "one", Type: "Envelope<Status>"},
				{Name: "twoLevels", Type: "Page<Status>"},
				{Name: "sub", Type: "Sub"},
				{Name: "nested", Type: "Envelope<Order.Kind>"},
				{Name: "nestedList", Type: "Envelope<List<Order.Kind>>"},
			},
		},
	}
	for name, library := range map[string]DependencyResolver{"without a dependency resolver": nil, "with a dependency resolver": &libraryClasses{}} {
		t.Run(name, func(t *testing.T) {
			r := NewTypeResolver(classes)
			if library != nil {
				r.SetDependencyResolver(library)
			}
			holder := r.Resolve("b.Holder", nil)
			for what, got := range map[string]string{
				"Envelope<Status>":                 holder.Fields["one"].Model.Fields["data"].Model.TypeName,
				"Page<Status> two levels down":     holder.Fields["twoLevels"].Model.Fields["inner"].Model.Fields["data"].Model.TypeName,
				"Sub extends Base<Status>":         holder.Fields["sub"].Model.Fields["data"].Model.TypeName,
				"Envelope<Order.Kind>":             holder.Fields["nested"].Model.Fields["data"].Model.TypeName,
				"Envelope<List<Order.Kind>> items": holder.Fields["nestedList"].Model.Fields["data"].Model.Items.TypeName,
			} {
				want := "b.Status"
				switch what {
				case "Envelope<Order.Kind>", "Envelope<List<Order.Kind>> items":
					want = "b.Order.Kind"
				}
				if got != want {
					t.Errorf("%s is exported as %q, want %q", what, got, want)
				}
			}
		})
	}
}
