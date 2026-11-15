package resolver

import (
	"testing"

	"github.com/tangcent/apilot/api-collector-java/parser"
	model "github.com/tangcent/apilot/api-model"
)

// An upper-bounded wildcard is its bound: every element of a
// `List<? extends Item>` is an Item. A lower bound and an open wildcard carry
// no element type, so those stay open objects (#167). Decompiled Kotlin
// declares each `List<T>` property as `List<? extends T>`.
func TestResolve_AnUpperBoundedWildcardIsItsBound(t *testing.T) {
	r := NewTypeResolver([]parser.Class{
		{Name: "Item", Package: "com.example", Fields: []parser.Field{{Name: "sku", Type: "String"}}},
		{Name: "PageData", Package: "com.example", TypeParameters: []string{"T"}, Fields: []parser.Field{
			{Name: "covariant", Type: "List<? extends T>"},
			{Name: "concrete", Type: "List<? extends Item>"},
			{Name: "contravariant", Type: "List<? super T>"},
			{Name: "open", Type: "List<?>"},
		}},
	})
	page := r.Resolve("com.example.PageData<com.example.Item>", nil)

	for _, name := range []string{"covariant", "concrete"} {
		if items := page.Fields[name].Model.Items; items == nil || items.Fields["sku"] == nil {
			t.Errorf("%s items = %+v, want Item", name, items)
		}
	}
	for _, name := range []string{"contravariant", "open"} {
		if items := page.Fields[name].Model.Items; items == nil || items.Kind != model.KindObject || len(items.Fields) != 0 {
			t.Errorf("%s items = %+v, want an open object", name, items)
		}
	}
	if unresolved := r.Unresolved(); len(unresolved) != 0 {
		t.Errorf("unresolved = %v", unresolved)
	}
}

func TestWildcardUpperBound(t *testing.T) {
	for typeName, want := range map[string]string{
		"? extends Item":             "Item",
		"?  extends  List<Item>":     "List<Item>",
		"? extends com.example.Item": "com.example.Item",
		"?":                          "",
		"? super Item":               "",
	} {
		got, ok := wildcardUpperBound(typeName)
		if got != want || ok != (want != "") {
			t.Errorf("wildcardUpperBound(%q) = %q, %v; want %q", typeName, got, ok, want)
		}
	}
}
