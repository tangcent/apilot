package flask

import (
	"testing"

	collector "github.com/tangcent/apilot/api-collector"
)

func TestFlaskTypeResolver_UnresolvedTypeIsRecorded(t *testing.T) {
	sink := collector.NewUnresolvedSet()
	r := NewFlaskTypeResolver(nil, nil)
	r.SetUnresolved(sink)

	got := r.Resolve("PageResult")

	if got.TypeName != "PageResult" {
		t.Fatalf("Resolve(PageResult).TypeName = %q", got.TypeName)
	}
	// The Flask resolver delegates to the FastAPI resolver, which owns the sink.
	if sink.Counts()["PageResult"] != 1 {
		t.Errorf("unresolved counts = %v, want PageResult:1", sink.Counts())
	}
}

func TestFlaskTypeResolver_KnownMarshmallowSchemaIsNotRecorded(t *testing.T) {
	sink := collector.NewUnresolvedSet()
	r := NewFlaskTypeResolver(nil, map[string]MarshmallowModel{
		"UserSchema": {
			Name:   "UserSchema",
			Fields: []MarshmallowField{{Name: "name", FieldType: "String", Required: true}},
		},
	})
	r.SetUnresolved(sink)

	got := r.Resolve("UserSchema")

	if !got.IsObject() {
		t.Fatalf("Resolve(UserSchema) kind = %s, want object", got.Kind)
	}
	if len(sink.Counts()) != 0 {
		t.Errorf("resolved schema must not be reported unresolved, got %v", sink.Counts())
	}
}

func TestFlaskTypeResolver_MultiBranchUnionIsRecorded(t *testing.T) {
	sink := collector.NewUnresolvedSet()
	r := NewFlaskTypeResolver(nil, nil)
	r.SetUnresolved(sink)

	r.Resolve("int | str")

	if sink.Counts()["int | str"] != 1 {
		t.Errorf("unresolved counts = %v, want 'int | str':1", sink.Counts())
	}
}

func TestFlaskTypeResolver_WithoutSinkIsSafe(t *testing.T) {
	r := NewFlaskTypeResolver(nil, nil)

	got := r.Resolve("PageResult") // must not panic

	if got.TypeName != "PageResult" {
		t.Fatalf("Resolve(PageResult).TypeName = %q", got.TypeName)
	}
	if len(r.Unresolved()) != 0 {
		t.Errorf("Unresolved() = %v, want empty", r.Unresolved())
	}
}
