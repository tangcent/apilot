package resolver

import (
	"fmt"
	"testing"

	"github.com/tangcent/apilot/api-collector-java/parser"
)

// libraryOfMemberTypes is a library whose classes declare member types and
// inherit some: Base has Row, Iface has Item, Mid is an interface that extends
// Iface, and Child is a class that extends Base.
func libraryOfMemberTypes() *nestedLibrary {
	const pkg = "com.example.lib"
	base := parser.Class{Name: "Base", Package: pkg}
	row := parser.Class{Name: "Row", Package: pkg, Outer: "Base", Fields: []parser.Field{{Name: "row", Type: "String"}}}
	iface := parser.Class{Name: "Iface", Package: pkg, IsInterface: true}
	item := parser.Class{Name: "Item", Package: pkg, Outer: "Iface", Fields: []parser.Field{{Name: "iface", Type: "String"}}}
	mid := parser.Class{Name: "Mid", Package: pkg, IsInterface: true, Interfaces: []string{"Iface"}}
	child := parser.Class{Name: "Child", Package: pkg, SuperClass: "Base"}
	return &nestedLibrary{
		classes: map[string]parser.Class{
			pkg + ".Base": base, pkg + ".Base.Row": row,
			pkg + ".Iface": iface, pkg + ".Iface.Item": item,
			pkg + ".Mid": mid, pkg + ".Child": child,
		},
		siblings: map[string][]parser.Class{pkg + ".Base": {row}, pkg + ".Iface": {item}},
	}
}

// A class inherits the member types of every supertype, a library's included, and
// a name in its body denotes one before anything the file declares or imports.
// The supertypes of the class, and of the classes around it, are read from the
// dependencies before the name is looked up: nothing has to have read them.
func TestResolve_TheMemberTypesALibraryClassInheritsAreFoundBeforeAnythingIsRead(t *testing.T) {
	const pkg = "com.example.lib"
	caller := parser.Class{Name: "Caller", Package: "com.example.app", FilePath: "Caller.java"}
	implementing := parser.Class{
		Name: "Impl", Package: "com.example.app", FilePath: "Impl.java", Interfaces: []string{"Iface"},
		Imports: []parser.Import{{Path: pkg + ".Iface"}}, Fields: []parser.Field{{Name: "item", Type: "Item"}},
	}
	throughMid := parser.Class{
		Name: "ViaMid", Package: "com.example.app", FilePath: "ViaMid.java", Interfaces: []string{"Mid"},
		Imports: []parser.Import{{Path: pkg + ".Mid"}}, Fields: []parser.Field{{Name: "item", Type: "Item"}},
	}
	outer := parser.Class{
		Name: "Outer", Package: "com.example.app", FilePath: "Outer.java", SuperClass: "Base",
		Imports: []parser.Import{{Path: pkg + ".Base"}},
	}
	inner := parser.Class{
		Name: "Inner", Package: "com.example.app", Outer: "Outer", FilePath: "Outer.java",
		Fields: []parser.Field{{Name: "row", Type: "Row"}},
	}
	star := parser.Class{
		Name: "Star", Package: "com.example.app", FilePath: "Star.java",
		Imports: []parser.Import{{Path: pkg + ".Child", OnDemand: true}}, Fields: []parser.Field{{Name: "row", Type: "Row"}},
	}

	for _, tc := range []struct {
		name   string
		source parser.Class
		scan   []parser.Class
		ref    string
		want   string
	}{
		{"a library interface the class implements", implementing, nil, "com.example.app.Impl", "{item:{iface:string}}"},
		{"a library interface that extends another", throughMid, nil, "com.example.app.ViaMid", "{item:{iface:string}}"},
		{"a library superclass of the class a nested class is declared in", inner, []parser.Class{outer}, "com.example.app.Outer.Inner", "{row:{row:string}}"},
		{"an import on demand of a class that inherits the member type", star, nil, "com.example.app.Star", "{row:{row:string}}"},
	} {
		// The answer is the same whatever the library classes the name depends on were
		// read before.
		for _, warm := range [][]string{nil, {pkg + ".Base"}, {pkg + ".Iface"}, {pkg + ".Iface", pkg + ".Mid"}, {pkg + ".Child"}} {
			t.Run(tc.name+fmt.Sprint(" read first ", warm), func(t *testing.T) {
				library := libraryOfMemberTypes()
				r := NewTypeResolver(append([]parser.Class{caller, tc.source}, tc.scan...))
				r.SetDependencyResolver(library)
				for _, name := range warm {
					r.ResolveFrom(caller, name, nil)
				}

				assertShape(t, tc.ref, r.ResolveFrom(caller, tc.ref, nil), tc.want)
				seen := map[string]int{}
				for _, name := range library.asked {
					seen[name]++
				}
				for name, count := range seen {
					if count > 1 {
						t.Errorf("%s was asked %d times: %v", name, count, library.asked)
					}
				}
			})
		}
	}

	t.Run("a member type that the class declares shadows an inherited one", func(t *testing.T) {
		own := parser.Class{Name: "Row", Package: "com.example.app", Outer: "Impl2", FilePath: "Impl2.java", Fields: []parser.Field{{Name: "own", Type: "String"}}}
		impl2 := parser.Class{
			Name: "Impl2", Package: "com.example.app", FilePath: "Impl2.java", SuperClass: "Base",
			Imports: []parser.Import{{Path: pkg + ".Base"}}, Fields: []parser.Field{{Name: "row", Type: "Row"}},
		}
		r := NewTypeResolver([]parser.Class{caller, impl2, own})
		r.SetDependencyResolver(libraryOfMemberTypes())

		assertShape(t, "Impl2", r.ResolveFrom(caller, "com.example.app.Impl2", nil), "{row:{own:string}}")
	})
}

// A class of the scanned sources imported on demand names the member types it
// inherits too.
func TestResolve_AnImportOnDemandOfAScannedClassNamesTheMemberTypesItInherits(t *testing.T) {
	classes := []parser.Class{
		{Name: "Base", Package: "p", FilePath: "Base.java"},
		{Name: "Inherited", Package: "p", Outer: "Base", FilePath: "Base.java", Fields: []parser.Field{{Name: "inherited", Type: "String"}}},
		{Name: "Outer", Package: "p", FilePath: "Outer.java", SuperClass: "Base"},
		{Name: "Inner", Package: "p", Outer: "Outer", FilePath: "Outer.java"},
		{Name: "Other", Package: "p", FilePath: "Other.java"},
		{Name: "Inherited", Package: "p", Outer: "Other", FilePath: "Other.java", Fields: []parser.Field{{Name: "bait", Type: "String"}}},
		{Name: "Star", Package: "app", FilePath: "Star.java", Imports: []parser.Import{{Path: "p.Outer", OnDemand: true}}},
	}
	r := NewTypeResolver(classes)

	if got := r.resolveName(classes[6], "Inherited"); got != "p.Base.Inherited" {
		t.Errorf("resolveName(Star, Inherited) = %q, want the member type Outer inherits from Base", got)
	}
	if got := r.resolveName(classes[6], "Inner"); got != "p.Outer.Inner" {
		t.Errorf("resolveName(Star, Inner) = %q, want the member type Outer declares", got)
	}
}
