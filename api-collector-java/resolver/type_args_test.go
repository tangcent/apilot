package resolver

import (
	"reflect"
	"testing"

	"github.com/tangcent/apilot/api-collector-java/parser"
	model "github.com/tangcent/apilot/api-model"
)

// libraryClasses answers dependency lookups by qualified name only, as the
// indexer does: a simple name finds nothing.
type libraryClasses struct {
	classes map[string]parser.Class
	asked   []string
}

func (l *libraryClasses) ResolveClass(name string) *parser.Class {
	l.asked = append(l.asked, name)
	if c, ok := l.classes[name]; ok {
		return &c
	}
	return nil
}

func TestResolve_TypeArgumentsAreLookedUpByTheNameTheCallerImports(t *testing.T) {
	library := &libraryClasses{classes: map[string]parser.Class{
		"com.example.lib.Wrapper": {
			Name: "Wrapper", Package: "com.example.lib", TypeParameters: []string{"T"},
			Fields: []parser.Field{{Name: "code", Type: "int"}, {Name: "data", Type: "T"}},
		},
		"com.example.lib.PageData": {
			Name: "PageData", Package: "com.example.lib", TypeParameters: []string{"T"},
			Fields: []parser.Field{{Name: "total", Type: "long"}, {Name: "rows", Type: "List<T>"}},
		},
	}}
	// The controller whose return types are resolved, as an endpoint's are.
	controller := parser.Class{
		Name: "OrderController", Package: "com.example.app", FilePath: "OrderController.java",
		Imports: []parser.Import{
			{Path: "com.example.lib.Wrapper"}, {Path: "com.example.lib.PageData"}, {Path: "java.util.List"},
		},
	}
	order := parser.Class{
		Name: "Order", Package: "com.example.app", FilePath: "Order.java",
		Fields: []parser.Field{{Name: "id", Type: "long"}},
	}
	r := NewTypeResolver([]parser.Class{controller, order})
	r.SetDependencyResolver(library)

	page := r.ResolveFrom(controller, "Wrapper<PageData<Order>>", nil)
	data := page.Fields["data"].Model
	if data == nil || data.Kind != model.KindObject {
		t.Fatalf("data = %+v, want PageData expanded", data)
	}
	rows := data.Fields["rows"].Model
	if rows.Kind != model.KindArray || rows.Items.Fields["id"] == nil {
		t.Fatalf("data.rows = %+v, want an array of Order", rows)
	}
	list := r.ResolveFrom(controller, "Wrapper<List<Order>>", nil).Fields["data"].Model
	if list.Kind != model.KindArray || list.Items.Fields["id"] == nil {
		t.Fatalf("data = %+v, want an array of Order: java.util.List stays a collection", list)
	}
	for _, name := range library.asked {
		if name == "PageData" {
			t.Errorf("PageData was asked for by its simple name; asked %v", library.asked)
		}
	}
}

func TestQualifyTypeArgs(t *testing.T) {
	source := parser.Class{
		Name: "Holder", Package: "com.example.app",
		Imports: []parser.Import{{Path: "com.example.lib.PageData"}, {Path: "java.util.List"}, {Path: "kotlin.collections.Map"}},
	}
	r := NewTypeResolver(nil)
	got := r.qualifyTypeArgs(source, []string{"PageData<Unknown>", "List<PageData>", "Map<String, PageData[]>", "?", "T", "? extends PageData"})
	want := []string{
		"com.example.lib.PageData<Unknown>",
		"List<com.example.lib.PageData>",
		"Map<String, com.example.lib.PageData[]>",
		"?",
		"T",
		"? extends com.example.lib.PageData",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("qualifyTypeArgs:\n got  %q\n want %q", got, want)
	}
}

// Decompiled Kotlin declares a `List<T>` property as `List<? extends T>`.
func TestResolve_AnUpperBoundedWildcardIsItsBound(t *testing.T) {
	r := NewTypeResolver([]parser.Class{
		{Name: "Item", Package: "com.example", Fields: []parser.Field{{Name: "sku", Type: "String"}}},
		{Name: "PageData", Package: "com.example", TypeParameters: []string{"T"}, Fields: []parser.Field{
			{Name: "covariant", Type: "List<? extends T>"},
			{Name: "contravariant", Type: "List<? super T>"},
			{Name: "open", Type: "List<?>"},
		}},
	})
	page := r.Resolve("com.example.PageData<com.example.Item>", nil)

	if items := page.Fields["covariant"].Model.Items; items == nil || items.Fields["sku"] == nil {
		t.Errorf("covariant items = %+v, want Item", items)
	}
	for _, name := range []string{"contravariant", "open"} {
		if items := page.Fields[name].Model.Items; items == nil || items.Kind != model.KindObject || len(items.Fields) != 0 {
			t.Errorf("%s items = %+v, want an open object", name, items)
		}
	}
}
