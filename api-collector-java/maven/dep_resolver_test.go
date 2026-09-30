package maven

import (
	"testing"
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
