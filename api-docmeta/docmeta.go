// Package docmeta defines the shared documentation-metadata contract used by
// all collectors. Each language extracts its own documentation sources
// (JavaDoc and Swagger annotations, Go doc comments and struct tags, JSDoc,
// Python docstrings and Pydantic fields) into a normalized Documentation
// value, which is then applied to a model.FieldModel via ApplyTo.
package docmeta

import (
	model "github.com/tangcent/apilot/api-model"
)

// Documentation is the normalized form of a field's documentation metadata,
// independent of the source language.
type Documentation struct {
	// Comment is the human-readable description of the field.
	Comment string
	// Demo is an example value rendered as the field's demo.
	Demo string
	// DefaultValue is the documented default of the field.
	DefaultValue string
	// Required overrides the field's required flag when non-nil.
	Required *bool
	// Options lists the allowed values of the field, if documented as an enum.
	Options []model.FieldOption
}

// Bool returns a pointer to value, for building Documentation.Required.
func Bool(value bool) *bool {
	return &value
}

// OptionsFromValues converts a list of enum values into field options.
// It returns nil when values is empty.
func OptionsFromValues(values []string) []model.FieldOption {
	if len(values) == 0 {
		return nil
	}
	options := make([]model.FieldOption, 0, len(values))
	for _, value := range values {
		options = append(options, model.FieldOption{Value: value})
	}
	return options
}

// ApplyTo writes the documented metadata onto fm. Only non-empty entries
// override what fm already carries, so callers may pre-populate fallbacks
// (e.g. a language-specific default for Required) before applying.
// ApplyTo is a no-op when fm is nil.
func (d Documentation) ApplyTo(fm *model.FieldModel) {
	if fm == nil {
		return
	}
	if d.Comment != "" {
		fm.Comment = d.Comment
	}
	if d.Demo != "" {
		fm.Demo = d.Demo
	}
	if d.DefaultValue != "" {
		fm.DefaultValue = d.DefaultValue
	}
	if d.Required != nil {
		fm.Required = *d.Required
	}
	if len(d.Options) > 0 {
		fm.Options = make([]model.FieldOption, 0, len(d.Options))
		fm.Options = append(fm.Options, d.Options...)
	}
}
