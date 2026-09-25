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
//
// Required is never set here: whether a field is required is decided by the
// binding/validate tag handling in each framework resolver, and validation
// keywords such as `validate:"required"` or `oneof=` are not parsed.
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
