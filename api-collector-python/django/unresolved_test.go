package django

import (
	"testing"

	collector "github.com/tangcent/apilot/api-collector"
)

func TestDRFTypeResolver_UnresolvedSerializerIsRecorded(t *testing.T) {
	sink := collector.NewUnresolvedSet()
	r := NewDRFTypeResolver(nil)
	r.SetUnresolved(sink)

	got := r.ResolveSerializer("UserSerializer")

	if got.TypeName != "UserSerializer" {
		t.Fatalf("ResolveSerializer(UserSerializer).TypeName = %q", got.TypeName)
	}
	if sink.Counts()["UserSerializer"] != 1 {
		t.Errorf("unresolved counts = %v, want UserSerializer:1", sink.Counts())
	}
}

func TestDRFTypeResolver_KnownSerializerIsNotRecorded(t *testing.T) {
	sink := collector.NewUnresolvedSet()
	r := NewDRFTypeResolver(map[string]SerializerModel{
		"UserSerializer": {
			Name:   "UserSerializer",
			Fields: []SerializerField{{Name: "name", DRFType: "CharField", Required: true}},
		},
	})
	r.SetUnresolved(sink)

	got := r.ResolveSerializer("UserSerializer")

	if !got.IsObject() {
		t.Fatalf("ResolveSerializer(UserSerializer) kind = %s, want object", got.Kind)
	}
	if len(sink.Counts()) != 0 {
		t.Errorf("resolved serializer must not be reported unresolved, got %v", sink.Counts())
	}
}

func TestDRFTypeResolver_UnresolvedFieldTypeIsRecorded(t *testing.T) {
	sink := collector.NewUnresolvedSet()
	r := NewDRFTypeResolver(nil)
	r.SetUnresolved(sink)

	// A DRF field type that is neither a known field class nor a serializer.
	field := r.resolveSerializerField(SerializerField{Name: "profile", DRFType: "ProfileSerializer", Required: true})

	if field.Model == nil || field.Model.TypeName != "ProfileSerializer" {
		t.Fatalf("resolveSerializerField model = %+v", field.Model)
	}
	if sink.Counts()["ProfileSerializer"] != 1 {
		t.Errorf("unresolved counts = %v, want ProfileSerializer:1", sink.Counts())
	}
}

func TestDRFTypeResolver_WithoutSinkIsSafe(t *testing.T) {
	r := NewDRFTypeResolver(nil)

	got := r.ResolveSerializer("UserSerializer") // must not panic

	if got.TypeName != "UserSerializer" {
		t.Fatalf("ResolveSerializer(UserSerializer).TypeName = %q", got.TypeName)
	}
	if len(r.Unresolved()) != 0 {
		t.Errorf("Unresolved() = %v, want empty", r.Unresolved())
	}
}
