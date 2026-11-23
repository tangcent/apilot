package resolver

import (
	"reflect"
	"testing"

	"github.com/tangcent/apilot/api-collector-java/parser"
	model "github.com/tangcent/apilot/api-model"
)

// A superclass that only a dependency declares is read through the same scope as
// any other name, in the header of the subclass: by a single-type import, an
// import on demand, the package of the subclass, or its qualified name, with or
// without type arguments, whether the subclass is a class of the project or one
// read from a library, and whether or not the superclass was read before.
func TestResolve_ASuperclassThatOnlyADependencyDeclaresIsRead(t *testing.T) {
	base := parser.Class{
		Name: "Base", Package: "q", TypeParameters: []string{"T"},
		Fields: []parser.Field{{Name: "inherited", Type: "String"}, {Name: "payload", Type: "T"}, {Name: "own", Type: "Item"}},
	}
	root := parser.Class{Name: "Root", Package: "q", Fields: []parser.Field{{Name: "rootField", Type: "String"}}}
	middle := parser.Class{Name: "Middle", Package: "q", SuperClass: "Root", Fields: []parser.Field{{Name: "middleField", Type: "String"}}}
	libraryItem := marked("q", "Item", "fromLibrary")
	projectItem := marked("app.dto", "Item", "fromProject")
	projectItem.FilePath = "Item.java"

	for _, tc := range []struct {
		name  string
		child parser.Class
	}{
		{"single-type import", parser.Class{Name: "Child", Package: "p", SuperClass: "Base", SuperClassTypeArgs: []string{"Item"}, Imports: []parser.Import{{Path: "q.Base"}, {Path: "app.dto.Item"}}}},
		{"import on demand", parser.Class{Name: "Child", Package: "p", SuperClass: "Base", SuperClassTypeArgs: []string{"Item"}, Imports: []parser.Import{{Path: "q", OnDemand: true}, {Path: "app.dto.Item"}}}},
		{"qualified", parser.Class{Name: "Child", Package: "p", SuperClass: "q.Base", SuperClassTypeArgs: []string{"Item"}, Imports: []parser.Import{{Path: "app.dto.Item"}}}},
		{"qualified type argument", parser.Class{Name: "Child", Package: "p", SuperClass: "q.Base", SuperClassTypeArgs: []string{"app.dto.Item"}}},
		{"a superclass of the superclass", parser.Class{Name: "Child", Package: "p", SuperClass: "q.Middle"}},
	} {
		for _, warm := range []bool{false, true} {
			name := tc.name
			if warm {
				name += ", read before"
			}
			t.Run(name, func(t *testing.T) {
				library := &libraryClasses{classes: map[string]parser.Class{"q.Base": base, "q.Middle": middle, "q.Root": root, "q.Item": libraryItem}}
				child := tc.child
				child.FilePath = "Child.java"
				caller := parser.Class{Name: "Caller", Package: "app", FilePath: "Caller.java"}
				r := NewTypeResolver([]parser.Class{caller, child, projectItem})
				r.SetDependencyResolver(library)
				if warm {
					r.ResolveFrom(caller, "q.Base", nil)
					r.ResolveFrom(caller, "q.Middle", nil)
				}

				got := r.ResolveFrom(caller, "p.Child", nil)
				if tc.name == "a superclass of the superclass" {
					if want := []string{"middleField", "rootField"}; !reflect.DeepEqual(fieldsOf(got), want) {
						t.Errorf("Child has %v, want %v", fieldsOf(got), want)
					}
					return
				}
				if want := []string{"inherited", "own", "payload"}; !reflect.DeepEqual(fieldsOf(got), want) {
					t.Fatalf("Child has %v, want %v", fieldsOf(got), want)
				}
				// The argument is the class the subclass names; the superclass's own
				// Item is its own.
				if fields := fieldsOf(got.Fields["payload"].Model); !reflect.DeepEqual(fields, []string{"fromProject"}) {
					t.Errorf("payload has %v, want the project's Item", fields)
				}
				if fields := fieldsOf(got.Fields["own"].Model); !reflect.DeepEqual(fields, []string{"fromLibrary"}) {
					t.Errorf("own has %v, want the library's Item", fields)
				}
			})
		}
	}

	t.Run("a class read from a library, in the package of its superclass", func(t *testing.T) {
		child := parser.Class{Name: "Child", Package: "q", SuperClass: "Base", SuperClassTypeArgs: []string{"String"}}
		library := &libraryClasses{classes: map[string]parser.Class{"q.Child": child, "q.Base": base, "q.Item": libraryItem}}
		r := NewTypeResolver(nil)
		r.SetDependencyResolver(library)

		got := r.Resolve("q.Child", nil)
		if want := []string{"inherited", "own", "payload"}; !reflect.DeepEqual(fieldsOf(got), want) {
			t.Errorf("Child has %v, want %v", fieldsOf(got), want)
		}
	})

	t.Run("Supertype", func(t *testing.T) {
		library := &libraryClasses{classes: map[string]parser.Class{"q.Base": base}}
		child := parser.Class{Name: "Child", Package: "p", FilePath: "Child.java", SuperClass: "q.Base"}
		r := NewTypeResolver([]parser.Class{child})
		r.SetDependencyResolver(library)

		if got, found := r.Supertype(child, "q.Base"); !found || got.Name != "Base" {
			t.Errorf("Supertype(Child, q.Base) = %+v, %v, want q.Base", got, found)
		}
		if got, found := r.Supertype(child, "q.Gone"); found {
			t.Errorf("Supertype(Child, q.Gone) = %+v, want none", got)
		}
		// A class of the JDK is not asked of the dependencies.
		library.asked = nil
		r.Supertype(child, "java.io.Serializable")
		r.Supertype(child, "Serializable")
		for _, name := range library.asked {
			if name == "java.io.Serializable" {
				t.Errorf("the dependencies were asked for %s", name)
			}
		}
	})
}

// A superclass that is a class of the scanned sources is found by its qualified
// name, as a class of the project is by any other.
func TestResolve_AQualifiedSuperclassOfTheProjectIsRead(t *testing.T) {
	parent := parser.Class{Name: "Parent", Package: "common", FilePath: "Parent.java", Fields: []parser.Field{{Name: "inherited", Type: "String"}}}
	child := parser.Class{Name: "Child", Package: "p", FilePath: "Child.java", SuperClass: "common.Parent", Fields: []parser.Field{{Name: "own", Type: "String"}}}
	r := NewTypeResolver([]parser.Class{parent, child})

	if got := fieldsOf(r.ResolveFrom(child, "p.Child", nil)); !reflect.DeepEqual(got, []string{"inherited", "own"}) {
		t.Errorf("Child has %v, want the fields of common.Parent", got)
	}
	if got, found := r.Supertype(child, "common.Parent"); !found || got.Name != "Parent" {
		t.Errorf("Supertype(Child, common.Parent) = %+v, %v", got, found)
	}
}

// The member types a class inherits from a superclass that only a dependency
// declares are in scope in the class, as those of a class of the project are
// (JLS 8.5): the class, a nested one or an enum, is the one the superclass
// declares and not a class of the same name that the package of the subclass or
// an import on demand brings in.
func TestResolve_AMemberTypeOfADependencySuperclassIsInherited(t *testing.T) {
	base := parser.Class{Name: "Base", Package: "q"}
	nestedClass := parser.Class{Name: "Row", Package: "q", Outer: "Base", Fields: []parser.Field{{Name: "ofBase", Type: "String"}}}
	nestedEnum := parser.Class{Name: "Mode", Package: "q", Outer: "Base", IsEnum: true, EnumConstants: []string{"ON", "OFF"}}
	otherRow := marked("p", "Row", "fromPackage")
	otherRow.FilePath = "Row.java"
	wildcardRow := marked("w", "Row", "fromWildcard")
	wildcardMode := parser.Class{Name: "Mode", Package: "w", Fields: []parser.Field{{Name: "fromWildcard", Type: "String"}}}

	for _, warm := range []bool{false, true} {
		name := "the superclass is read when the subclass is expanded"
		if warm {
			name = "the superclass was read before"
		}
		t.Run(name, func(t *testing.T) {
			library := &nestedLibrary{
				classes: map[string]parser.Class{
					"q.Base": base, "q.Base.Row": nestedClass, "q.Base.Mode": nestedEnum,
					"w.Row": wildcardRow, "w.Mode": wildcardMode,
				},
				siblings: map[string][]parser.Class{"q.Base": {nestedClass, nestedEnum}},
			}
			child := parser.Class{
				Name: "Child", Package: "p", FilePath: "Child.java", SuperClass: "Base",
				Imports: []parser.Import{{Path: "q.Base"}, {Path: "w", OnDemand: true}},
				Fields:  []parser.Field{{Name: "row", Type: "Row"}, {Name: "mode", Type: "Mode"}},
			}
			caller := parser.Class{Name: "Caller", Package: "app", FilePath: "Caller.java"}
			r := NewTypeResolver([]parser.Class{caller, child, otherRow})
			r.SetDependencyResolver(library)
			if warm {
				r.ResolveFrom(caller, "q.Base", nil)
			}

			got := r.ResolveFrom(caller, "p.Child", nil)
			assertShape(t, "Child.row", got.Fields["row"].Model, "{ofBase:string}")
			// The enum is what Mode denotes, so it is exported as the JSON view
			// exports an enum, not as the class that w brings in.
			if mode := got.Fields["mode"].Model; mode.Kind != model.KindSingle {
				t.Errorf("Child.mode = %s, want the enum Base.Mode (an opaque value)", shapeOf(mode))
			}
		})
	}
}
