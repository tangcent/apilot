package resolver

import (
	"testing"

	"github.com/tangcent/apilot/api-collector-java/parser"
)

// A field is marked generic when its type is an open type variable. The names of
// the type parameters of every class of the project are known, but a class that
// carries the name of one of them, an enum the JSON view does not read, a class
// the field's file imports, is that class and not a variable.

func genericFieldClasses() []parser.Class {
	f := typeArgField
	other := func(name string, isEnum bool) parser.Class {
		return parser.Class{Name: name, Package: "com.example.other", FilePath: "com/example/other/" + name + ".java", IsEnum: isEnum}
	}
	kind := other("Kind", true)
	kind.EnumConstants = []string{"A"}
	return []parser.Class{
		kind,
		// Classes whose parameters carry the names of the types the order uses.
		{Name: "Weird", Package: typeArgLibPackage, FilePath: "com/example/lib/Weird.java", TypeParameters: []string{"Kind", "Tag"},
			Fields: []parser.Field{f("kind", "Kind"), f("tag", "Tag")}},
		{Name: "Box", Package: typeArgLibPackage, FilePath: "com/example/lib/Box.java", TypeParameters: []string{"T"},
			Fields: []parser.Field{f("data", "T")}},
		{Name: "Order", Package: typeArgAppPackage, FilePath: "com/example/app/Order.java",
			Imports: []parser.Import{{Path: "com.example.other.Kind"}, {Path: "com.example.other.Tag"}, {Path: typeArgLibPackage + ".Box"}, {Path: "java.util.List"}},
			Fields: []parser.Field{
				f("kind", "Kind"), f("tag", "Tag"), f("kinds", "List<Kind>"),
				f("boxedKind", "Box<Kind>"), f("boxedTag", "Box<Tag>"),
			}},
	}
}

func TestResolve_AClassNamedLikeATypeParameterIsNotAGenericField(t *testing.T) {
	for view, resolveIn := range typeArgViews {
		t.Run(view, func(t *testing.T) {
			r := NewTypeResolver(genericFieldClasses())
			order := resolveIn(r, parser.Class{}, "com.example.app.Order")
			for path, want := range map[string]bool{
				"kind":           false, // an enum of the project
				"tag":            false, // a class the file imports, whose body is not known
				"boxedKind":      false,
				"boxedTag":       false,
				"boxedKind.data": false,
				"boxedTag.data":  false,
			} {
				parent, name := order, path
				if i := lastDot(path); i >= 0 {
					parent, name = typeArgAt(order, path[:i]), path[i+1:]
				}
				field := parent.Fields[name]
				if field == nil {
					t.Fatalf("Order.%s: no such field in %+v", path, parent)
				}
				if field.Generic != want {
					t.Errorf("Order.%s generic = %v, want %v", path, field.Generic, want)
				}
			}
		})
	}
}

// What is a type variable still is one.
func TestResolve_AnOpenTypeVariableIsAGenericField(t *testing.T) {
	r := NewTypeResolver(genericFieldClasses())
	weird := r.Resolve("com.example.lib.Weird", nil)
	for _, name := range []string{"kind", "tag"} {
		if field := weird.Fields[name]; field == nil || !field.Generic {
			t.Errorf("Weird.%s = %+v, want a generic field: its type is the class's own parameter", name, field)
		}
	}
	// Bound to an open variable of the caller.
	caller := parser.Class{Name: "Holder", Package: typeArgAppPackage, TypeParameters: []string{"Tag"}}
	boxed := r.ResolveFrom(caller, "com.example.lib.Box<Tag>", nil)
	if field := boxed.Fields["data"]; field == nil || !field.Generic {
		t.Errorf("Box<Tag>.data = %+v, want a generic field: Tag is a variable of the caller", field)
	}
}

func lastDot(s string) int {
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == '.' {
			return i
		}
	}
	return -1
}
