package fastapi

import (
	"testing"

	collector "github.com/tangcent/apilot/api-collector"
	model "github.com/tangcent/apilot/api-model"
)

func TestPythonTypeResolver_UnresolvedTypeIsRecorded(t *testing.T) {
	sink := collector.NewUnresolvedSet()
	r := NewPythonTypeResolver(nil)
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

func TestPythonTypeResolver_KnownModelIsNotRecorded(t *testing.T) {
	sink := collector.NewUnresolvedSet()
	r := NewPythonTypeResolver(map[string]PydanticModel{
		"User": {
			Name:   "User",
			Fields: []PydanticField{{Name: "name", Type: "str", Required: true}},
		},
	})
	r.SetUnresolved(sink)

	got := r.Resolve("User")

	if !got.IsObject() {
		t.Fatalf("Resolve(User) kind = %s, want object", got.Kind)
	}
	if len(sink.Counts()) != 0 {
		t.Errorf("resolved model must not be reported unresolved, got %v", sink.Counts())
	}
}

func TestPythonTypeResolver_MultiBranchUnionIsRecorded(t *testing.T) {
	sink := collector.NewUnresolvedSet()
	r := NewPythonTypeResolver(nil)
	r.SetUnresolved(sink)

	// Union[int, str] cannot collapse, so the union text is rendered as-is.
	r.Resolve("Union[int, str]")

	counts := sink.Counts()
	if len(counts) != 1 {
		t.Fatalf("unresolved counts = %v, want exactly one entry", counts)
	}
}

func TestPythonTypeResolver_PrimitiveIsNotRecorded(t *testing.T) {
	sink := collector.NewUnresolvedSet()
	r := NewPythonTypeResolver(nil)
	r.SetUnresolved(sink)

	r.Resolve("str")
	r.Resolve("List[str]")
	r.Resolve("Optional[str]")
	r.Resolve("")

	if len(sink.Counts()) != 0 {
		t.Errorf("primitives and empty types are not failures, got %v", sink.Counts())
	}
}

// `Generic[T]` is a typing declaration used as a base class
// (`class PageResult(BaseModel, Generic[T])`), so it is neither expandable nor
// a resolution failure.
func TestPythonTypeResolver_GenericBaseIsNotReported(t *testing.T) {
	sink := collector.NewUnresolvedSet()
	r := NewPythonTypeResolver(nil)
	r.SetUnresolved(sink)

	got := r.Resolve("Generic[T]")

	if !got.IsNull() {
		t.Fatalf("Resolve(Generic[T]) = %+v, want null", got)
	}
	if len(sink.Counts()) != 0 {
		t.Errorf("Generic[T] must not be reported unresolved, got %v", sink.Counts())
	}
}

func TestPythonTypeResolver_WithoutSinkIsSafe(t *testing.T) {
	r := NewPythonTypeResolver(nil)

	got := r.Resolve("PageResult") // must not panic

	if got.TypeName != "PageResult" {
		t.Fatalf("Resolve(PageResult).TypeName = %q", got.TypeName)
	}
	if len(r.Unresolved()) != 0 {
		t.Errorf("Unresolved() = %v, want empty", r.Unresolved())
	}
}
