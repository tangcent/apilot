package jsdoc

import (
	"testing"
)

func TestExtractJSDocBlock(t *testing.T) {
	raw := `/**
 * The user name.
 * Spans two lines.
 * @example John
 * @default anon
 */`
	d := Extract(raw)
	if d.Comment != "The user name. Spans two lines." {
		t.Errorf("Comment = %q", d.Comment)
	}
	if d.Demo != "John" {
		t.Errorf("Demo = %q, want %q", d.Demo, "John")
	}
	if d.DefaultValue != "anon" {
		t.Errorf("DefaultValue = %q, want %q", d.DefaultValue, "anon")
	}
	if d.Required != nil || d.Options != nil {
		t.Errorf("unexpected %+v", d)
	}
}

func TestExtractLineComment(t *testing.T) {
	d := Extract("// just a note")
	if d.Comment != "just a note" {
		t.Errorf("Comment = %q, want %q", d.Comment, "just a note")
	}
}

func TestExtractPlainDescription(t *testing.T) {
	d := Extract("from decorator")
	if d.Comment != "from decorator" {
		t.Errorf("Comment = %q, want %q", d.Comment, "from decorator")
	}
}

func TestExtractEmpty(t *testing.T) {
	d := Extract("   ")
	if d.Comment != "" || d.Demo != "" || d.DefaultValue != "" {
		t.Errorf("expected empty documentation, got %+v", d)
	}
}

func TestExtractUnknownTagsIgnored(t *testing.T) {
	d := Extract("/** desc\n * @param x ignored\n */")
	if d.Comment != "desc" {
		t.Errorf("Comment = %q, want %q", d.Comment, "desc")
	}
}

func TestExtractDefaultValueTag(t *testing.T) {
	d := Extract("/** desc\n * @defaultValue 7\n */")
	if d.DefaultValue != "7" {
		t.Errorf("DefaultValue = %q, want %q", d.DefaultValue, "7")
	}
}
