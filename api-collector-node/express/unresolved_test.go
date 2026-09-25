package express

import (
	"testing"

	collector "github.com/tangcent/apilot/api-collector"
	model "github.com/tangcent/apilot/api-model"
)

func TestTSTypeResolver_UnresolvedTypeIsRecorded(t *testing.T) {
	sink := collector.NewUnresolvedSet()
	r := NewTSTypeResolver(NewTSTypeRegistry())
	r.SetUnresolved(sink)

	got := r.Resolve("PageResult", nil)

	if got.Kind != model.KindSingle || got.TypeName != "PageResult" {
		t.Fatalf("Resolve(PageResult) = %+v, want opaque single model", got)
	}
	if sink.Counts()["PageResult"] != 1 {
		t.Errorf("unresolved counts = %v, want PageResult:1", sink.Counts())
	}

	r.Resolve("PageResult", nil)
	if sink.Counts()["PageResult"] != 2 {
		t.Errorf("PageResult count = %d, want 2", sink.Counts()["PageResult"])
	}
}

func TestTSTypeResolver_KnownInterfaceIsNotRecorded(t *testing.T) {
	sink := collector.NewUnresolvedSet()
	registry := NewTSTypeRegistry()
	registry.Interfaces["User"] = &TSInterface{
		Name:   "User",
		Fields: []TSField{{Name: "name", Type: "string", Required: true}},
	}
	r := NewTSTypeResolver(registry)
	r.SetUnresolved(sink)

	got := r.Resolve("User", nil)

	if !got.IsObject() {
		t.Fatalf("Resolve(User) kind = %s, want object", got.Kind)
	}
	if len(sink.Counts()) != 0 {
		t.Errorf("resolved interface must not be reported unresolved, got %v", sink.Counts())
	}
}

func TestTSTypeResolver_UnresolvableUnionIsRecorded(t *testing.T) {
	sink := collector.NewUnresolvedSet()
	r := NewTSTypeResolver(NewTSTypeRegistry())
	r.SetUnresolved(sink)

	// Neither member is known, so the union cannot collapse to a single model.
	r.Resolve("Admin | Guest", nil)

	if sink.Counts()["Admin | Guest"] != 1 {
		t.Errorf("unresolved counts = %v, want 'Admin | Guest':1", sink.Counts())
	}
}

func TestTSTypeResolver_PrimitiveIsNotRecorded(t *testing.T) {
	sink := collector.NewUnresolvedSet()
	r := NewTSTypeResolver(NewTSTypeRegistry())
	r.SetUnresolved(sink)

	r.Resolve("string", nil)
	r.Resolve("string[]", nil)
	r.Resolve("string | null", nil)
	r.Resolve("", nil)

	if len(sink.Counts()) != 0 {
		t.Errorf("primitives are not failures, got %v", sink.Counts())
	}
}

func TestTSTypeResolver_WithoutSinkIsSafe(t *testing.T) {
	r := NewTSTypeResolver(NewTSTypeRegistry())

	got := r.Resolve("PageResult", nil) // must not panic

	if got.TypeName != "PageResult" {
		t.Fatalf("Resolve(PageResult).TypeName = %q", got.TypeName)
	}
	if len(r.Unresolved()) != 0 {
		t.Errorf("Unresolved() = %v, want empty", r.Unresolved())
	}
}
