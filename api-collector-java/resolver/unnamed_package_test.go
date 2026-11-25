package resolver

import (
	"testing"

	"github.com/tangcent/apilot/api-collector-java/parser"
)

// The unnamed package is a package like any other (JLS 7.4.2): a class declared
// in it is in scope for the other classes of it, ahead of an import on demand and
// of a class that another package declares under the same name.
func TestResolveName_AClassOfTheUnnamedPackageIsInScopeForTheOthers(t *testing.T) {
	item := parser.Class{Name: "Item", FilePath: "src/Item.java", Fields: []parser.Field{{Name: "own", Type: "int"}}}
	// "A.Item" sorts before "Item", so the fallback that takes the first class
	// of a name would pick it.
	other := parser.Class{Name: "Item", Package: "A", FilePath: "src/A/Item.java", Fields: []parser.Field{{Name: "other", Type: "int"}}}
	nested := parser.Class{Name: "Item", Outer: "Holder", FilePath: "src/Holder.java", Fields: []parser.Field{{Name: "nested", Type: "int"}}}
	holder := parser.Class{Name: "Holder", FilePath: "src/Holder.java"}
	form := parser.Class{Name: "Form", FilePath: "src/Form.java"}
	r := NewTypeResolver([]parser.Class{other, item, nested, holder, form})

	if got := typeArgShape(r.ResolveFrom(form, "Item", nil)); got != "Item(own)" {
		t.Errorf("Item in the unnamed package = %s, want the class of that package", got)
	}

	// A class of another package does not see it: it is not imported.
	app := parser.Class{Name: "Form", Package: "app", FilePath: "src/app/Form.java",
		Imports: []parser.Import{{Path: "A.Item"}}}
	if got := typeArgShape(r.ResolveFrom(app, "Item", nil)); got != "Item(other)" {
		t.Errorf("Item imported by name = %s, want the class the import names", got)
	}

	// A class nested in one of the unnamed package is named through its outer
	// class, not by its simple name from the others.
	if got := r.resolveName(form, "Item"); got != "Item" {
		t.Errorf("resolveName(Item) = %q, want the top-level class", got)
	}
}
