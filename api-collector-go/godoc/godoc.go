// Package godoc normalizes Go struct field documentation (doc comments and
// struct tags) into the shared docmeta contract.
package godoc

import (
	"reflect"
	"strings"

	docmeta "github.com/tangcent/apilot/api-docmeta"
)

// Extract converts a struct field's doc comment and raw struct tag into
// normalized documentation. Tag values take precedence over the doc comment.
//
// Recognized tags:
//   - description:"..." — field description (overrides the doc comment)
//   - example:"..."     — example value
//   - default:"..."     — documented default
//   - enums:"a,b,c"     — comma-separated allowed values
//   - validate:"required,oneof=a b c" — go-playground/validator keywords;
//     `required` marks the field required and `oneof` supplies the allowed
//     values. Any other keyword (min, max, omitempty, ...) is ignored.
//
// `enums` wins over `oneof` when a field carries both: enums exists purely to
// document options, while oneof is a validation constraint.
func Extract(comment string, tag reflect.StructTag) docmeta.Documentation {
	d := docmeta.Documentation{Comment: strings.TrimSpace(comment)}
	if v := tag.Get("description"); v != "" {
		d.Comment = v
	}
	if v := tag.Get("example"); v != "" {
		d.Demo = v
	}
	if v := tag.Get("default"); v != "" {
		d.DefaultValue = v
	}
	if v := tag.Get("validate"); v != "" {
		for _, keyword := range splitCSV(v) {
			switch {
			case keyword == "required":
				d.Required = docmeta.Bool(true)
			case strings.HasPrefix(keyword, "oneof="):
				d.Options = docmeta.OptionsFromValues(strings.Fields(strings.TrimPrefix(keyword, "oneof=")))
			}
		}
	}
	if v := tag.Get("enums"); v != "" {
		d.Options = docmeta.OptionsFromValues(splitCSV(v))
	}
	return d
}

func splitCSV(value string) []string {
	parts := strings.Split(value, ",")
	values := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			values = append(values, part)
		}
	}
	return values
}
