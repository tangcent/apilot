package echo

import (
	"testing"

	collector "github.com/tangcent/apilot/api-collector"
	model "github.com/tangcent/apilot/api-model"
)

func TestResolve_UnresolvedTypeIsRecorded(t *testing.T) {
	sink := collector.NewUnresolvedSet()
	r := NewTypeResolver(nil)
	r.SetUnresolved(sink)

	got := r.Resolve("PageResult")

	if got.Kind != model.KindSingle || got.TypeName != "PageResult" {
		t.Fatalf("Resolve(PageResult) = %+v, want opaque single model", got)
	}
	if sink.Counts()["PageResult"] != 1 {
		t.Errorf("unresolved counts = %v, want PageResult:1", sink.Counts())
	}

	r.Resolve("PageResult")
	if sink.Counts()["PageResult"] != 2 {
		t.Errorf("PageResult count = %d, want 2", sink.Counts()["PageResult"])
	}
}

func TestResolve_KnownStructIsNotRecorded(t *testing.T) {
	sink := collector.NewUnresolvedSet()
	r := NewTypeResolver(map[string]StructDef{
		"User": {
			Name: "User",
			Fields: []StructField{
				{Name: "Name", Type: "string", JsonTag: "name"},
			},
		},
	})
	r.SetUnresolved(sink)

	got := r.Resolve("User")

	if !got.IsObject() {
		t.Fatalf("Resolve(User) kind = %s, want object", got.Kind)
	}
	if len(sink.Counts()) != 0 {
		t.Errorf("resolved struct must not be reported unresolved, got %v", sink.Counts())
	}
}

func TestResolve_PrimitiveAndEmptyTypeAreNotRecorded(t *testing.T) {
	sink := collector.NewUnresolvedSet()
	r := NewTypeResolver(nil)
	r.SetUnresolved(sink)

	r.Resolve("string")
	r.Resolve("[]string")
	r.Resolve("")
	r.Resolve("interface{}")

	if len(sink.Counts()) != 0 {
		t.Errorf("primitives and empty types are not failures, got %v", sink.Counts())
	}
}

func TestResolve_WithoutSinkIsSafe(t *testing.T) {
	r := NewTypeResolver(nil)

	got := r.Resolve("PageResult") // must not panic

	if got.TypeName != "PageResult" {
		t.Fatalf("Resolve(PageResult).TypeName = %q", got.TypeName)
	}
	if len(r.Unresolved()) != 0 {
		t.Errorf("Unresolved() = %v, want empty", r.Unresolved())
	}
}
