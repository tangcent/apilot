package docmeta

import (
	"testing"

	model "github.com/tangcent/apilot/api-model"
)

func newFieldModel() *model.FieldModel {
	return &model.FieldModel{
		Model:    model.SingleModel(model.JsonTypeString),
		Required: true,
	}
}

func TestApplyToWritesAllDocumentedFields(t *testing.T) {
	fm := newFieldModel()
	d := Documentation{
		Comment:      "user name",
		Demo:         "John",
		DefaultValue: "anon",
		Required:     Bool(false),
		Options:      OptionsFromValues([]string{"a", "b"}),
	}
	d.ApplyTo(fm)

	if fm.Comment != "user name" {
		t.Errorf("Comment = %q, want %q", fm.Comment, "user name")
	}
	if fm.Demo != "John" {
		t.Errorf("Demo = %q, want %q", fm.Demo, "John")
	}
	if fm.DefaultValue != "anon" {
		t.Errorf("DefaultValue = %q, want %q", fm.DefaultValue, "anon")
	}
	if fm.Required != false {
		t.Errorf("Required = %v, want false", fm.Required)
	}
	if len(fm.Options) != 2 || fm.Options[0].Value != "a" || fm.Options[1].Value != "b" {
		t.Errorf("Options = %+v, want values a,b", fm.Options)
	}
}

func TestApplyToSkipsEmptyEntries(t *testing.T) {
	fm := newFieldModel()
	fm.Comment = "existing"
	fm.Demo = "existing demo"
	fm.DefaultValue = "existing default"
	existing := OptionsFromValues([]string{"keep"})
	fm.Options = existing

	Documentation{}.ApplyTo(fm)

	if fm.Comment != "existing" {
		t.Errorf("Comment = %q, want %q", fm.Comment, "existing")
	}
	if fm.Demo != "existing demo" {
		t.Errorf("Demo = %q, want it unchanged", fm.Demo)
	}
	if fm.DefaultValue != "existing default" {
		t.Errorf("DefaultValue = %q, want it unchanged", fm.DefaultValue)
	}
	if fm.Required != true {
		t.Errorf("Required = %v, want it unchanged (true)", fm.Required)
	}
	if len(fm.Options) != 1 || fm.Options[0].Value != "keep" {
		t.Errorf("Options = %+v, want it unchanged", fm.Options)
	}
}

func TestApplyToRequiredOverride(t *testing.T) {
	fm := newFieldModel()
	fm.Required = false
	Documentation{Required: Bool(true)}.ApplyTo(fm)
	if fm.Required != true {
		t.Errorf("Required = %v, want true", fm.Required)
	}

	Documentation{Required: Bool(false)}.ApplyTo(newFieldModel())
	fm2 := newFieldModel()
	Documentation{Required: Bool(false)}.ApplyTo(fm2)
	if fm2.Required != false {
		t.Errorf("Required = %v, want false", fm2.Required)
	}
}

func TestApplyToNilFieldModelIsNoOp(t *testing.T) {
	d := Documentation{Comment: "x"}
	d.ApplyTo(nil)
}

func TestApplyToCopiesOptions(t *testing.T) {
	options := OptionsFromValues([]string{"a"})
	d := Documentation{Options: options}
	fm := newFieldModel()
	d.ApplyTo(fm)

	fm.Options[0].Value = "mutated"
	if options[0].Value != "a" {
		t.Error("ApplyTo must not share the Options slice with the caller")
	}
}

func TestOptionsFromValues(t *testing.T) {
	if got := OptionsFromValues(nil); got != nil {
		t.Errorf("OptionsFromValues(nil) = %+v, want nil", got)
	}
	if got := OptionsFromValues([]string{}); got != nil {
		t.Errorf("OptionsFromValues(empty) = %+v, want nil", got)
	}
	got := OptionsFromValues([]string{"x", "y"})
	if len(got) != 2 || got[0].Value != "x" || got[1].Value != "y" {
		t.Errorf("OptionsFromValues = %+v, want x,y", got)
	}
}
