package resolver

import (
	"testing"

	"github.com/tangcent/apilot/api-collector-java/parser"
	model "github.com/tangcent/apilot/api-model"
)

// A field whose type passes the class's type parameter on to another generic
// class keeps the argument: in Holder<Order>, the field `PageData<H> page` is
// a PageData<Order>, so its rows are Orders. The argument used to be resolved
// to a model and then spelled by the model's type name, which dropped the
// nested `<H>` and left the rows unbound.
func TestResolve_ANestedGenericFieldKeepsItsArguments(t *testing.T) {
	r := NewTypeResolver([]parser.Class{
		{Name: "Order", Package: "com.example", Fields: []parser.Field{{Name: "id", Type: "long"}}},
		{Name: "PageData", Package: "com.example", TypeParameters: []string{"T"}, Fields: []parser.Field{
			{Name: "total", Type: "long"},
			{Name: "rows", Type: "List<T>"},
		}},
		{Name: "Pair", Package: "com.example", TypeParameters: []string{"A", "B"}, Fields: []parser.Field{
			{Name: "first", Type: "A"},
			{Name: "second", Type: "B"},
		}},
		// H, not T: a flattened PageData would otherwise pick up the outer
		// binding of a parameter that merely shares its name.
		{Name: "Holder", Package: "com.example", TypeParameters: []string{"H"}, Fields: []parser.Field{
			{Name: "page", Type: "PageData<H>"},
			{Name: "pages", Type: "List<PageData<H>>"},
			{Name: "byKey", Type: "Map<String, PageData<H>>"},
			{Name: "pair", Type: "Pair<H, PageData<H>>"},
			{Name: "fixed", Type: "PageData<Order>"},
		}},
	})
	holder := r.Resolve("com.example.Holder<com.example.Order>", nil)

	rowsOf := func(name string, page *model.ObjectModel) {
		t.Helper()
		if page == nil || page.Fields["rows"] == nil {
			t.Fatalf("%s = %+v, want a PageData", name, page)
		}
		items := page.Fields["rows"].Model.Items
		if items == nil || items.Fields["id"] == nil {
			t.Errorf("%s.rows items = %+v, want Order", name, items)
		}
	}
	rowsOf("page", holder.Fields["page"].Model)
	rowsOf("pages[]", holder.Fields["pages"].Model.Items)
	rowsOf("byKey{}", holder.Fields["byKey"].Model.ValueModel)
	pair := holder.Fields["pair"].Model
	if first := pair.Fields["first"].Model; first.Fields["id"] == nil {
		t.Errorf("pair.first = %+v, want Order", first)
	}
	rowsOf("pair.second", pair.Fields["second"].Model)
	rowsOf("fixed", holder.Fields["fixed"].Model)

	if unresolved := r.Unresolved(); len(unresolved) != 0 {
		t.Errorf("unresolved = %v", unresolved)
	}
}
