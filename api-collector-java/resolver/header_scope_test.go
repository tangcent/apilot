package resolver

import (
	"testing"

	"github.com/tangcent/apilot/api-collector-java/parser"
	model "github.com/tangcent/apilot/api-model"
)

// The extends and implements clauses of a class are outside its body, and the
// classes nested in the class are in scope in the body only. Every name the
// header writes is read there: the superclass, the interfaces and the arguments
// of both, and whether the superclass is a map. A class nested in the class that
// is called like one of them does not hide the import or the class of the
// package that the header means.

func headerScopeClasses() []parser.Class {
	nested := func(outer, name string) parser.Class {
		return parser.Class{
			Name: name, Package: "p", Outer: outer, FilePath: "p/" + outer + ".java",
			Fields: []parser.Field{typeArgField("wrong", "String")},
		}
	}
	mapClass := func(name, superClass string, args []string, imports ...parser.Import) parser.Class {
		return parser.Class{
			Name: name, Package: "p", FilePath: "p/" + name + ".java",
			SuperClass: superClass, SuperClassTypeArgs: args, Imports: imports,
		}
	}
	hashMap := parser.Import{Path: "java.util.HashMap"}
	return []parser.Class{
		{Name: "Item", Package: "q", FilePath: "q/Item.java", Fields: []parser.Field{typeArgField("right", "String")}},
		// Each of these has a class nested in it that is called like a name its
		// header writes.
		mapClass("Counts", "HashMap", []string{"String", "Integer"}, hashMap),
		nested("Counts", "HashMap"),
		mapClass("Starred", "HashMap", []string{"String", "Integer"}, parser.Import{Path: "java.util", OnDemand: true}),
		nested("Starred", "HashMap"),
		mapClass("Names", "HashMap", []string{"String", "Item"}, hashMap, parser.Import{Path: "q.Item"}),
		nested("Names", "Item"),
		mapClass("Texts", "HashMap", []string{"String", "String"}, hashMap),
		nested("Texts", "String"),
		mapClass("Qualified", "java.util.HashMap", []string{"String", "Integer"}),
		nested("Qualified", "HashMap"),
		// The body of the class does see its member types.
		{
			Name: "Holder", Package: "p", FilePath: "p/Holder.java", Imports: []parser.Import{hashMap},
			Fields: []parser.Field{typeArgField("map", "HashMap<String, Integer>")},
		},
		nested("Holder", "HashMap"),
	}
}

func TestResolve_AMapSubclassIsClassifiedInTheScopeOfItsHeader(t *testing.T) {
	caller := parser.Class{Name: "Caller", Package: "app", FilePath: "app/Caller.java"}
	for view, resolveIn := range typeArgViews {
		t.Run(view, func(t *testing.T) {
			r := NewTypeResolver(headerScopeClasses())
			for typeName, want := range map[string]string{
				"p.Counts":    "{int}",
				"p.Starred":   "{int}",
				"p.Names":     "{Item(right)}",
				"p.Texts":     "{string}",
				"p.Qualified": "{int}",
			} {
				got := resolveIn(r, caller, typeName)
				if got.Kind != model.KindMap || typeArgShape(got) != want {
					t.Errorf("%s = %s (%s), want a map %s", typeName, typeArgShape(got), got.Kind, want)
				}
			}
			// In the body of a class the nested class hides the import.
			holder := resolveIn(r, caller, "p.Holder")
			if got := typeArgShape(typeArgAt(holder, "map")); got != "HashMap(wrong)" {
				t.Errorf("Holder.map = %s, want the class nested in Holder", got)
			}
		})
	}
}
