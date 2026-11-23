package kotlincollector

import (
	"reflect"
	"testing"

	ktparser "github.com/tangcent/apilot/api-collector-kotlin/parser"
	"github.com/tangcent/apilot/api-collector-kotlin/resolver"
)

// A superclass that only a dependency declares is read from it whatever the
// spelling: its qualified name, an import, an import on demand. The fields it
// declares are the subclass's, and the argument the supertype list gives it is the
// class that list names, not the class of the same name next to the superclass.
func TestASuperclassThatOnlyADependencyDeclaresIsRead(t *testing.T) {
	sources := map[string]string{
		"lib.Parent": `package lib

open class Parent<T> {
    var inherited: String = ""
    var payload: T? = null
    var ownItem: Item? = null
}
`,
		"lib.Item": `package lib

class Item {
    var libraryItem: String = ""
}
`,
	}
	item := ktparser.Class{Name: "Item", Package: "app.dto", FilePath: "Item.kt", Fields: []ktparser.Property{{Name: "projectItem", Type: "String"}}}
	caller := ktparser.Class{Name: "Caller", Package: "app", FilePath: "Caller.kt"}

	for _, tc := range []struct {
		name  string
		child ktparser.Class
	}{
		{"qualified", ktparser.Class{Name: "Child", Package: "app.dto", FilePath: "Child.kt", SuperClass: "lib.Parent", SuperClassTypeArgs: []string{"Item"}}},
		{"imported", ktparser.Class{Name: "Child", Package: "app.dto", FilePath: "Child.kt", SuperClass: "Parent", SuperClassTypeArgs: []string{"Item"}, Imports: []ktparser.Import{{Path: "lib.Parent"}}}},
		{"imported on demand", ktparser.Class{Name: "Child", Package: "app.dto", FilePath: "Child.kt", SuperClass: "Parent", SuperClassTypeArgs: []string{"Item"}, Imports: []ktparser.Import{{Path: "lib", OnDemand: true}}}},
	} {
		for _, readFirst := range []string{"", "lib.Parent"} {
			t.Run(tc.name+", read first "+readFirst, func(t *testing.T) {
				typeResolver := resolver.New([]ktparser.Class{item, tc.child, caller})
				typeResolver.SetDependencyResolver(&starLibrary{t: t, sources: sources})
				if readFirst != "" {
					typeResolver.ResolveFrom(&caller, readFirst, nil)
				}

				child := typeResolver.ResolveFrom(&caller, "app.dto.Child", nil)
				want := map[string]string{"inherited": "string", "payload": "object{projectItem}", "ownItem": "object{libraryItem}"}
				if got := starFieldKinds(child); !reflect.DeepEqual(got, want) {
					t.Errorf("Child:\n got  %v\n want %v", got, want)
				}
			})
		}
	}
}
