package maven

import (
	"testing"

	"github.com/tangcent/apilot/api-collector-java/parser"
)

func TestExtractSourceFromMarkdown(t *testing.T) {
	t.Run("extracts java code block", func(t *testing.T) {
		input := "### Class: com.example.Result\nArtifact: com.example:lib:1.0\n\n```java\npackage com.example;\n\npublic class Result {\n    private int code;\n}\n```\n"
		expected := "package com.example;\n\npublic class Result {\n    private int code;\n}"
		result, err := extractSourceFromMarkdown(input)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != expected {
			t.Errorf("expected:\n%s\ngot:\n%s", expected, result)
		}
	})

	t.Run("returns raw if no code block", func(t *testing.T) {
		input := "public class Foo { }"
		result, err := extractSourceFromMarkdown(input)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != input {
			t.Errorf("expected raw input returned, got:\n%s", result)
		}
	})

	t.Run("handles unclosed code block", func(t *testing.T) {
		input := "```java\npublic class Foo { }"
		expected := "public class Foo { }"
		result, err := extractSourceFromMarkdown(input)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != expected {
			t.Errorf("expected:\n%s\ngot:\n%s", expected, result)
		}
	})

	t.Run("handles empty code block", func(t *testing.T) {
		input := "```java\n```"
		expected := ""
		result, err := extractSourceFromMarkdown(input)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != expected {
			t.Errorf("expected empty string, got:\n%s", result)
		}
	})

	t.Run("handles code block with generics", func(t *testing.T) {
		input := "```java\npublic class Result<T> {\n    private T data;\n}\n```"
		expected := "public class Result<T> {\n    private T data;\n}"
		result, err := extractSourceFromMarkdown(input)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != expected {
			t.Errorf("expected:\n%s\ngot:\n%s", expected, result)
		}
	})
}

func TestMavenDependencyResolver_New(t *testing.T) {
	// Construction does not require the CLI: availability gating lives in the
	// collectors, which report a tool-missing outcome. Without the CLI the
	// resolver simply records misses.
	resolver, err := NewMavenDependencyResolver()
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if resolver == nil {
		t.Fatal("Expected non-nil resolver")
	}
	defer resolver.Close()

	if !isCLIAvailable() {
		if resolver.ResolveClass("java.lang.String") != nil {
			t.Error("Expected nil class when CLI unavailable")
		}
	}
}

func TestMavenDependencyResolver_ResolveTypeFieldFiltering(t *testing.T) {
	// Dependency classes go through the same static/final rules as workspace
	// classes: statics are dropped, final instance fields (immutable DTOs,
	// Lombok @Value) are kept.
	r := &MavenDependencyResolver{cache: map[string]*parser.Class{
		"com.example.lib.ImmutableProduct": {
			Name: "com.example.lib.ImmutableProduct",
			Fields: []parser.Field{
				{Name: "TYPE", Type: "String", IsStatic: true, IsFinal: true, HasInitializer: true},
				{Name: "sku", Type: "String", IsFinal: true},
				{Name: "price", Type: "long", IsFinal: true},
				{Name: "memo", Type: "String"},
			},
		},
	}}

	rt := r.ResolveType("com.example.lib.ImmutableProduct")
	if rt == nil {
		t.Fatal("Expected resolved type")
	}
	if len(rt.Fields) != 3 {
		t.Fatalf("Expected 3 fields, got %d: %#v", len(rt.Fields), rt.Fields)
	}

	names := make(map[string]bool, len(rt.Fields))
	for _, f := range rt.Fields {
		names[f.Name] = true
	}
	if names["TYPE"] {
		t.Error("Expected static final constant 'TYPE' to stay excluded")
	}
	for _, want := range []string{"sku", "price", "memo"} {
		if !names[want] {
			t.Errorf("Expected field '%s' to be kept", want)
		}
	}
}
