package resolver

import (
	"testing"

	collector "github.com/tangcent/apilot/api-collector"
	"github.com/tangcent/apilot/api-collector-java/parser"
	model "github.com/tangcent/apilot/api-model"
)

func TestResolve_UnresolvedTypeIsRecorded(t *testing.T) {
	sink := collector.NewUnresolvedSet()
	r := NewTypeResolver(nil)
	r.SetUnresolved(sink)

	got := r.Resolve("PageResult", nil)

	if got.Kind != model.KindSingle || got.TypeName != "PageResult" {
		t.Fatalf("Resolve(PageResult) = %+v, want opaque single model", got)
	}
	if sink.Counts()["PageResult"] != 1 {
		t.Errorf("unresolved counts = %v, want PageResult:1", sink.Counts())
	}
}

func TestResolve_UnresolvedTypeCountsEveryOccurrence(t *testing.T) {
	sink := collector.NewUnresolvedSet()
	r := NewTypeResolver(nil)
	r.SetUnresolved(sink)

	for i := 0; i < 3; i++ {
		r.Resolve("PageResult", nil)
	}
	r.Resolve("ApiResponse", nil)

	counts := sink.Counts()
	if counts["PageResult"] != 3 {
		t.Errorf("PageResult count = %d, want 3", counts["PageResult"])
	}
	if counts["ApiResponse"] != 1 {
		t.Errorf("ApiResponse count = %d, want 1", counts["ApiResponse"])
	}
}

func TestResolve_KnownClassIsNotRecorded(t *testing.T) {
	sink := collector.NewUnresolvedSet()
	r := NewTypeResolver([]parser.Class{{
		Name: "UserDTO",
		Fields: []parser.Field{
			{Name: "name", Type: "String"},
		},
	}})
	r.SetUnresolved(sink)

	got := r.Resolve("UserDTO", nil)

	if got.Kind != model.KindObject {
		t.Fatalf("Resolve(UserDTO) kind = %s, want object", got.Kind)
	}
	if len(sink.Counts()) != 0 {
		t.Errorf("resolved class must not be reported unresolved, got %v", sink.Counts())
	}
}

func TestResolve_PrimitiveAndEmptyTypeAreNotRecorded(t *testing.T) {
	sink := collector.NewUnresolvedSet()
	r := NewTypeResolver(nil)
	r.SetUnresolved(sink)

	r.Resolve("String", nil)
	r.Resolve("int", nil)
	r.Resolve("", nil)
	r.Resolve("List<String>", nil)

	if len(sink.Counts()) != 0 {
		t.Errorf("primitives and empty types are not failures, got %v", sink.Counts())
	}
}

// The TypeName of an array model is the synthetic placeholder "array". If a
// bound type parameter resolved to one fed that placeholder back into Resolve,
// the argument would collapse and "array" would be recorded as a bogus
// unresolved type.
func TestResolve_CompositeTypeArgumentKeepsSourceSpelling(t *testing.T) {
	sink := collector.NewUnresolvedSet()
	r := NewTypeResolver([]parser.Class{
		{Name: "Order", Fields: []parser.Field{{Name: "id", Type: "long"}}},
		{
			Name:           "Page",
			TypeParameters: []string{"T"},
			Fields:         []parser.Field{{Name: "items", Type: "List<T>"}},
		},
	})
	r.SetUnresolved(sink)

	got := r.Resolve("Page<List<Order>>", nil)

	items := got.Fields["items"].Model
	if !items.IsArray() {
		t.Fatalf("items kind = %s, want array", items.Kind)
	}
	inner := items.Items
	if !inner.IsArray() {
		t.Fatalf("items element kind = %s, want nested array", inner.Kind)
	}
	order := inner.Items
	if !order.IsObject() {
		t.Fatalf("nested element kind = %s, want object", order.Kind)
	}
	if _, ok := order.Fields["id"]; !ok {
		t.Errorf("Order fields = %v, want id", order.Fields)
	}
	if counts := sink.Counts(); len(counts) != 0 {
		t.Errorf("composite type argument must not be reported unresolved: %v", counts)
	}
}

func TestResolve_WithoutSinkIsSafe(t *testing.T) {
	r := NewTypeResolver(nil)

	got := r.Resolve("PageResult", nil) // must not panic

	if got.TypeName != "PageResult" {
		t.Fatalf("Resolve(PageResult).TypeName = %q", got.TypeName)
	}
	if len(r.Unresolved()) != 0 {
		t.Errorf("Unresolved() = %v, want empty", r.Unresolved())
	}
}
