package resolver

import (
	"testing"

	"github.com/tangcent/apilot/api-collector-java/parser"
)

// A class implements the interface that the name in its implements clause denotes,
// read in the scope of its header. The simple name of the interface alone does not
// say which interface it is: an interface of another package that shares the name is
// not implemented by the class, and neither is one the class imports from a library.
// A class nested in the implementing class that is called like the interface does
// not stand for it, as it is not in scope in the clause.

func implementationClasses() []parser.Class {
	iface := func(pkg string) parser.Class {
		return parser.Class{Name: "Handler", Package: pkg, FilePath: pkg + "/Handler.java", IsInterface: true}
	}
	impl := func(pkg, name, field string, imports ...parser.Import) parser.Class {
		return parser.Class{
			Name: name, Package: pkg, FilePath: pkg + "/" + name + ".java", Interfaces: []string{"Handler"}, Imports: imports,
			Fields: []parser.Field{typeArgField(field, "String")},
		}
	}
	holder := func(pkg string) parser.Class {
		return parser.Class{
			Name: "Holder", Package: pkg, FilePath: pkg + "/Holder.java",
			Fields: []parser.Field{typeArgField("handler", "Handler")},
		}
	}
	return []parser.Class{
		iface("a"), iface("b"), iface("c"), iface("d"),
		holder("a"), holder("b"), holder("c"), holder("d"),
		// AImpl has a Handler of its own nested in it, which its header does not see.
		impl("a", "AImpl", "ofA"),
		{Name: "Handler", Package: "a", Outer: "AImpl", FilePath: "a/AImpl.java", IsInterface: true},
		impl("b", "BImpl", "ofB"),
		// Implements the Handler of a library that is not among the sources.
		impl("e", "LibImpl", "ofLib", parser.Import{Path: "com.lib.Handler"}),
		// Implements the Handler of a by its qualified name.
		{
			Name: "Qualified", Package: "e", FilePath: "e/Qualified.java", Interfaces: []string{"b.Handler"},
			Fields: []parser.Field{typeArgField("ofQualified", "String")},
		},
	}
}

func TestResolve_AnInterfaceIsImplementedByTheClassThatNamesItInItsHeader(t *testing.T) {
	caller := parser.Class{Name: "Caller", Package: "app", FilePath: "app/Caller.java"}
	for view, resolveIn := range typeArgViews {
		t.Run(view, func(t *testing.T) {
			r := NewTypeResolver(implementationClasses())
			handler := func(pkg string) string {
				return typeArgShape(typeArgAt(resolveIn(r, caller, pkg+".Holder"), "handler"))
			}
			for pkg, want := range map[string]string{
				"a": "AImpl(ofA)",
				"b": "BImpl(ofB)", // not AImpl or LibImpl: they implement other interfaces called Handler
			} {
				if got := handler(pkg); got != want {
					t.Errorf("%s.Holder.handler = %s, want its own implementation %s", pkg, got, want)
				}
			}
			// Nothing implements these: the classes that implement an interface called
			// Handler implement another.
			for _, pkg := range []string{"c", "d"} {
				if got := resolveIn(r, caller, pkg+".Holder").Fields["handler"].Model; !got.IsRef() {
					t.Errorf("%s.Holder.handler = %s, want a reference: no class implements %s.Handler", pkg, typeArgShape(got), pkg)
				}
			}
		})
	}
}
