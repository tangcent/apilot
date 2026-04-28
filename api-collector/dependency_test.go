package collector

import (
	"testing"
)

type mockDependencyResolver struct {
	types map[string]*ResolvedType
	deps  []Dependency
}

func (m *mockDependencyResolver) DetectDependencies(sourceDir string) ([]Dependency, error) {
	return m.deps, nil
}

func (m *mockDependencyResolver) ResolveType(typeName string) *ResolvedType {
	if rt, ok := m.types[typeName]; ok {
		return rt
	}
	return nil
}

func TestDependencyResolver_Interface(t *testing.T) {
	resolver := &mockDependencyResolver{
		types: map[string]*ResolvedType{
			"User": {
				Name: "User",
				Fields: []ResolvedField{
					{Name: "id", Type: "int", Required: true},
					{Name: "name", Type: "string", Required: true},
					{Name: "email", Type: "string", Required: false},
				},
			},
		},
		deps: []Dependency{
			{Name: "mylib", Version: "1.0.0"},
		},
	}

	var _ DependencyResolver = resolver

	deps, err := resolver.DetectDependencies("/tmp")
	if err != nil {
		t.Fatalf("DetectDependencies returned error: %v", err)
	}
	if len(deps) != 1 {
		t.Fatalf("expected 1 dependency, got %d", len(deps))
	}
	if deps[0].Name != "mylib" || deps[0].Version != "1.0.0" {
		t.Errorf("unexpected dependency: %+v", deps[0])
	}

	rt := resolver.ResolveType("User")
	if rt == nil {
		t.Fatal("expected resolved type, got nil")
	}
	if rt.Name != "User" {
		t.Errorf("expected name=User, got %s", rt.Name)
	}
	if len(rt.Fields) != 3 {
		t.Fatalf("expected 3 fields, got %d", len(rt.Fields))
	}
	if rt.Fields[0].Name != "id" || !rt.Fields[0].Required {
		t.Errorf("unexpected field[0]: %+v", rt.Fields[0])
	}
	if rt.Fields[2].Name != "email" || rt.Fields[2].Required {
		t.Errorf("expected email not required, got %+v", rt.Fields[2])
	}

	if resolver.ResolveType("Unknown") != nil {
		t.Error("expected nil for unknown type")
	}
}

func TestResolvedType_Fields(t *testing.T) {
	rt := &ResolvedType{
		Name:           "Page",
		TypeParameters: []string{"T"},
		SuperClass:     "BasePage",
		IsInterface:    false,
		Interfaces:     []string{"Serializable"},
		Fields: []ResolvedField{
			{Name: "content", Type: "List<T>", Required: true},
			{Name: "total", Type: "int", Required: true},
		},
	}

	if len(rt.TypeParameters) != 1 || rt.TypeParameters[0] != "T" {
		t.Errorf("unexpected type parameters: %v", rt.TypeParameters)
	}
	if rt.SuperClass != "BasePage" {
		t.Errorf("unexpected super class: %s", rt.SuperClass)
	}
	if len(rt.Interfaces) != 1 || rt.Interfaces[0] != "Serializable" {
		t.Errorf("unexpected interfaces: %v", rt.Interfaces)
	}
	if !rt.Fields[0].Required {
		t.Error("expected content to be required")
	}
}

func TestDependency_Fields(t *testing.T) {
	dep := Dependency{
		Name:    "com.example:my-lib",
		Version: "2.1.0",
	}
	if dep.Name != "com.example:my-lib" {
		t.Errorf("unexpected name: %s", dep.Name)
	}
	if dep.Version != "2.1.0" {
		t.Errorf("unexpected version: %s", dep.Version)
	}
}
