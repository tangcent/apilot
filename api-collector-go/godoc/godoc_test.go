package godoc

import (
	"reflect"
	"testing"
)

func TestExtractDocCommentOnly(t *testing.T) {
	d := Extract("  Name is the user name.  ", "")
	if d.Comment != "Name is the user name." {
		t.Errorf("Comment = %q, want %q", d.Comment, "Name is the user name.")
	}
	if d.Demo != "" || d.DefaultValue != "" || d.Required != nil || d.Options != nil {
		t.Errorf("expected only Comment to be set, got %+v", d)
	}
}

func TestExtractDescriptionTagOverridesComment(t *testing.T) {
	d := Extract("fallback comment", reflect.StructTag(`description:"explicit doc"`))
	if d.Comment != "explicit doc" {
		t.Errorf("Comment = %q, want %q", d.Comment, "explicit doc")
	}
}

func TestExtractTags(t *testing.T) {
	tag := reflect.StructTag(`example:"John" default:"anon" enums:"red, green ,,"`)
	d := Extract("", tag)
	if d.Demo != "John" {
		t.Errorf("Demo = %q, want %q", d.Demo, "John")
	}
	if d.DefaultValue != "anon" {
		t.Errorf("DefaultValue = %q, want %q", d.DefaultValue, "anon")
	}
	if d.Required != nil {
		t.Errorf("Required = %v, want nil", d.Required)
	}
	if len(d.Options) != 2 || d.Options[0].Value != "red" || d.Options[1].Value != "green" {
		t.Errorf("Options = %+v, want red,green", d.Options)
	}
}

func TestExtractValidateTag(t *testing.T) {
	tag := reflect.StructTag(`validate:"required,oneof=admin editor viewer"`)
	d := Extract("", tag)
	if d.Required == nil || !*d.Required {
		t.Errorf("Required = %v, want true", d.Required)
	}
	if len(d.Options) != 3 || d.Options[0].Value != "admin" || d.Options[1].Value != "editor" || d.Options[2].Value != "viewer" {
		t.Errorf("Options = %+v, want admin,editor,viewer", d.Options)
	}
}

func TestExtractValidateTagIgnoresOtherKeywords(t *testing.T) {
	tag := reflect.StructTag(`validate:"omitempty,min=0,max=120"`)
	d := Extract("", tag)
	if d.Required != nil {
		t.Errorf("Required = %v, want nil", d.Required)
	}
	if d.Options != nil {
		t.Errorf("Options = %+v, want nil", d.Options)
	}
}

func TestExtractEnumsWinsOverOneof(t *testing.T) {
	tag := reflect.StructTag(`enums:"a,b" validate:"oneof=x y z"`)
	d := Extract("", tag)
	if len(d.Options) != 2 || d.Options[0].Value != "a" || d.Options[1].Value != "b" {
		t.Errorf("Options = %+v, want a,b", d.Options)
	}
}

func TestExtractEmpty(t *testing.T) {
	d := Extract("", "")
	if d.Comment != "" || d.Demo != "" || d.DefaultValue != "" || d.Required != nil || d.Options != nil {
		t.Errorf("expected empty Documentation, got %+v", d)
	}
}
