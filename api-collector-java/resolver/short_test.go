package resolver

import (
	"testing"

	collector "github.com/tangcent/apilot/api-collector"
	"github.com/tangcent/apilot/api-collector-java/parser"
)

// A short is an integer on the wire, as an int is. Before, `short` and `Short`
// were taken for the names of types that no declaration has, and the field was
// one opaque value reported as unresolved.
func TestResolve_ShortIsAnInt(t *testing.T) {
	r := NewTypeResolver(nil)
	r.SetUnresolved(collector.NewUnresolvedSet())

	for typeName, want := range map[string]string{
		"short":                           "int",
		"Short":                           "int",
		"java.lang.Short":                 "int",
		"short[]":                         "array<int>",
		"Short[]":                         "array<int>",
		"java.lang.Short[]":               "array<int>",
		"short[][]":                       "array<array<int>>",
		"List<Short>":                     "array<int>",
		"Set<java.lang.Short>":            "array<int>",
		"Map<String, Short>":              "map<int>",
		"Mono<Short>":                     "int",
		"ResponseEntity<short[]>":         "array<int>",
		"? extends Short":                 "int",
		"List<? extends java.lang.Short>": "array<int>",
	} {
		if got := arrayShape(r.Resolve(typeName, nil)); got != want {
			t.Errorf("%s = %s, want %s", typeName, got, want)
		}
	}
	if unresolved := r.Unresolved(); len(unresolved) != 0 {
		t.Errorf("unresolved = %v, want none", unresolved)
	}
}

// A class of java.lang is imported by every compilation unit, so its name
// qualified with the package is the same type as the simple one. Spelling the
// package is how a field reaches the resolver as `java.lang.Integer`.
func TestResolve_QualifiedJavaLangTypesAreTheSimpleOnes(t *testing.T) {
	r := NewTypeResolver(nil)
	r.SetUnresolved(collector.NewUnresolvedSet())

	for typeName, want := range map[string]string{
		"java.lang.Integer": "int",
		"java.lang.Long":    "long",
		"java.lang.Float":   "float",
		"java.lang.Double":  "double",
		"java.lang.Boolean": "boolean",
		"java.lang.Void":    "null",
		"java.lang.String":  "string",
	} {
		if got := arrayShape(r.Resolve(typeName, nil)); got != want {
			t.Errorf("%s = %s, want %s", typeName, got, want)
		}
	}
	if unresolved := r.Unresolved(); len(unresolved) != 0 {
		t.Errorf("unresolved = %v, want none", unresolved)
	}
}

// Number is written as a JSON number, and so is a Double. The Kotlin collector
// has always answered a double for it.
func TestResolve_NumberIsADouble(t *testing.T) {
	r := NewTypeResolver(nil)
	r.SetUnresolved(collector.NewUnresolvedSet())

	for _, typeName := range []string{"Number", "java.lang.Number"} {
		if got := arrayShape(r.Resolve(typeName, nil)); got != "double" {
			t.Errorf("%s = %s, want double", typeName, got)
		}
	}
	if unresolved := r.Unresolved(); len(unresolved) != 0 {
		t.Errorf("unresolved = %v, want none", unresolved)
	}
}

// A project class that shares the simple name of a JDK type still resolves from
// the registry, as one named Date does.
func TestResolve_ProjectClassWinsOverNumber(t *testing.T) {
	r := NewTypeResolver([]parser.Class{
		{Name: "Number", Fields: []parser.Field{{Name: "digits", Type: "String"}}},
	})

	if got := arrayShape(r.Resolve("Number", nil)); got != "object:Number" {
		t.Errorf("Number = %s, want object:Number from the registry", got)
	}
}
