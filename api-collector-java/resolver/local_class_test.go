package resolver

import (
	"testing"

	"github.com/tangcent/apilot/api-collector-java/parser"
)

// A class declared in a method is in scope in that method only, so it is neither a
// member type of the class around it nor a class of the package: a field of that
// class typed by the same name is the class the file or the package declares.
func TestResolve_AClassDeclaredInAMethodIsNotInScopeOutsideIt(t *testing.T) {
	classes := []parser.Class{
		{Name: "Item", Package: "p", FilePath: "Item.java", Fields: []parser.Field{{Name: "top", Type: "String"}}},
		{Name: "C", Package: "p", FilePath: "C.java", Fields: []parser.Field{{Name: "item", Type: "Item"}}},
		{Name: "Item", Package: "p", Outer: "C", Local: true, FilePath: "C.java", Fields: []parser.Field{{Name: "local", Type: "String"}}},
		{Name: "Helper", Package: "p", Outer: "C", Local: true, FilePath: "C.java", Fields: []parser.Field{{Name: "helper", Type: "String"}}},
		{Name: "Part", Package: "p", Outer: "C.Helper", Local: true, FilePath: "C.java", Fields: []parser.Field{{Name: "part", Type: "String"}}},
		{Name: "D", Package: "p", FilePath: "D.java", Fields: []parser.Field{{Name: "item", Type: "Item"}, {Name: "helper", Type: "Helper"}}},
	}
	caller := parser.Class{Name: "Caller", Package: "app", FilePath: "Caller.java"}

	for _, name := range []string{"scanned before the top-level class", "scanned after it"} {
		t.Run(name, func(t *testing.T) {
			ordered := classes
			if name == "scanned before the top-level class" {
				ordered = append([]parser.Class{classes[2]}, classes...)
			}
			r := NewTypeResolver(ordered)

			assertShape(t, "C", r.ResolveFrom(caller, "p.C", nil), "{item:{top:string}}")
			assertShape(t, "D", r.ResolveFrom(caller, "p.D", nil), "{helper:Helper,item:{top:string}}")
			if got := r.resolveName(classes[1], "Item"); got != "p.Item" {
				t.Errorf("resolveName(C, Item) = %q, want the class of the package", got)
			}
			if r.isDeclared("p.C.Item") || r.isDeclared("p.C.Helper") || r.isDeclared("p.C.Helper.Part") {
				t.Error("a local class is held as a declaration of the class around it")
			}
			if got := r.resolveName(caller, "Helper"); got == "p.C.Helper" {
				t.Errorf("resolveName(Caller, Helper) = %q, a local class is no class of any other scope", got)
			}
		})
	}
}

// The classes of a source read from a library are held with the class that was
// asked for, a local one among them: it is not a member type either.
func TestResolve_AClassDeclaredInAMethodOfALibraryClassIsNotAMember(t *testing.T) {
	const pkg = "com.example.lib"
	page := parser.Class{Name: "Page", Package: pkg, Fields: []parser.Field{{Name: "row", Type: "Row"}}}
	local := parser.Class{Name: "Row", Package: pkg, Outer: "Page", Local: true, Fields: []parser.Field{{Name: "local", Type: "String"}}}
	row := parser.Class{Name: "Row", Package: pkg, Fields: []parser.Field{{Name: "top", Type: "String"}}}
	library := &nestedLibrary{
		classes:  map[string]parser.Class{pkg + ".Page": page, pkg + ".Page.Row": local, pkg + ".Row": row},
		siblings: map[string][]parser.Class{pkg + ".Page": {local}},
	}
	r := NewTypeResolver(nil)
	r.SetDependencyResolver(library)

	assertShape(t, "Page", r.ResolveFrom(parser.Class{}, pkg+".Page", nil), "{row:{top:string}}")
}
