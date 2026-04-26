package postman

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	apimodel "github.com/tangcent/apilot/api-model"
)

// objectModelToJSON converts an ObjectModel into a pretty-printed JSON string
// suitable for use as a Postman request or response body.
// It generates demo values based on type information.
func objectModelToJSON(m *apimodel.ObjectModel) string {
	if m == nil {
		return "{}"
	}
	val := modelToValue(m)
	b, err := json.MarshalIndent(val, "", "    ")
	if err != nil {
		return "{}"
	}
	return string(b)
}

// modelToValue converts an ObjectModel into a Go value suitable for json.Marshal.
func modelToValue(m *apimodel.ObjectModel) any {
	if m == nil {
		return nil
	}

	switch m.Kind {
	case apimodel.KindSingle:
		return singleModelValue(m.TypeName)
	case apimodel.KindObject:
		return objectModelValue(m)
	case apimodel.KindArray:
		return arrayModelValue(m)
	case apimodel.KindMap:
		return mapModelValue(m)
	case apimodel.KindRef:
		return map[string]any{"$ref": m.TypeName}
	default:
		return nil
	}
}

func singleModelValue(typeName string) any {
	switch typeName {
	case apimodel.JsonTypeString:
		return ""
	case apimodel.JsonTypeInt, apimodel.JsonTypeLong:
		return 0
	case apimodel.JsonTypeFloat, apimodel.JsonTypeDouble:
		return 0.0
	case apimodel.JsonTypeBoolean:
		return false
	case apimodel.JsonTypeNull:
		return nil
	default:
		return ""
	}
}

func objectModelValue(m *apimodel.ObjectModel) any {
	if m.Fields == nil || len(m.Fields) == 0 {
		return map[string]any{}
	}

	obj := make(map[string]any, len(m.Fields))

	names := make([]string, 0, len(m.Fields))
	for name := range m.Fields {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		fm := m.Fields[name]
		obj[name] = fieldModelValue(fm)
	}
	return obj
}

func fieldModelValue(fm *apimodel.FieldModel) any {
	if fm == nil || fm.Model == nil {
		return nil
	}

	if fm.Demo != "" {
		return parseDemoValue(fm.Demo, fm.Model)
	}

	if fm.DefaultValue != "" {
		return parseDemoValue(fm.DefaultValue, fm.Model)
	}

	return modelToValue(fm.Model)
}

func arrayModelValue(m *apimodel.ObjectModel) any {
	if m.Items == nil {
		return []any{}
	}
	return []any{modelToValue(m.Items)}
}

func mapModelValue(m *apimodel.ObjectModel) any {
	if m.ValueModel == nil {
		return map[string]any{}
	}
	keyStr := "key"
	if m.KeyModel != nil && m.KeyModel.TypeName == apimodel.JsonTypeString {
		keyStr = "key"
	}
	return map[string]any{keyStr: modelToValue(m.ValueModel)}
}

// parseDemoValue attempts to parse a demo/default string into the appropriate Go type
// based on the model's kind. Falls back to the string value if parsing fails.
func parseDemoValue(demo string, m *apimodel.ObjectModel) any {
	if m == nil {
		return demo
	}

	if m.Kind == apimodel.KindSingle {
		switch m.TypeName {
		case apimodel.JsonTypeInt, apimodel.JsonTypeLong:
			var v int64
			if _, err := fmt.Sscanf(demo, "%d", &v); err == nil {
				return v
			}
		case apimodel.JsonTypeFloat, apimodel.JsonTypeDouble:
			var v float64
			if _, err := fmt.Sscanf(demo, "%f", &v); err == nil {
				return v
			}
		case apimodel.JsonTypeBoolean:
			if demo == "true" {
				return true
			}
			if demo == "false" {
				return false
			}
		}
	}

	if m.Kind == apimodel.KindObject || m.Kind == apimodel.KindArray || m.Kind == apimodel.KindMap {
		var v any
		if err := json.Unmarshal([]byte(demo), &v); err == nil {
			return v
		}
	}

	return demo
}

// objectModelToJSONWithComments converts an ObjectModel into a JSON string
// with field comments appended as `// comment` suffixes.
// This produces a JSON5-like format that Postman can display.
func objectModelToJSONWithComments(m *apimodel.ObjectModel) string {
	if m == nil {
		return "{}"
	}
	return modelToCommentedJSON(m, 0)
}

func modelToCommentedJSON(m *apimodel.ObjectModel, indent int) string {
	if m == nil {
		return "null"
	}

	switch m.Kind {
	case apimodel.KindSingle:
		val := singleModelValue(m.TypeName)
		b, _ := json.Marshal(val)
		return string(b)
	case apimodel.KindObject:
		return objectModelToCommentedJSON(m, indent)
	case apimodel.KindArray:
		return arrayModelToCommentedJSON(m, indent)
	case apimodel.KindMap:
		return mapModelToCommentedJSON(m, indent)
	case apimodel.KindRef:
		return fmt.Sprintf(`{"$ref": "%s"}`, m.TypeName)
	default:
		return "null"
	}
}

func objectModelToCommentedJSON(m *apimodel.ObjectModel, indent int) string {
	if m.Fields == nil || len(m.Fields) == 0 {
		return "{}"
	}

	pad := strings.Repeat("    ", indent)
	innerPad := strings.Repeat("    ", indent+1)

	names := make([]string, 0, len(m.Fields))
	for name := range m.Fields {
		names = append(names, name)
	}
	sort.Strings(names)

	lines := []string{"{"}
	for i, name := range names {
		fm := m.Fields[name]
		val := fieldModelToCommentedJSON(fm, indent+1)
		comma := ","
		if i == len(names)-1 {
			comma = ""
		}

		comment := ""
		if fm != nil && fm.Comment != "" {
			comment = fmt.Sprintf(" // %s", fm.Comment)
		}

		lines = append(lines, fmt.Sprintf("%s%q: %s%s%s", innerPad, name, val, comma, comment))
	}
	lines = append(lines, pad+"}")
	return strings.Join(lines, "\n")
}

func fieldModelToCommentedJSON(fm *apimodel.FieldModel, indent int) string {
	if fm == nil || fm.Model == nil {
		return "null"
	}

	if fm.Demo != "" {
		val := parseDemoValue(fm.Demo, fm.Model)
		b, _ := json.Marshal(val)
		return string(b)
	}

	if fm.DefaultValue != "" {
		val := parseDemoValue(fm.DefaultValue, fm.Model)
		b, _ := json.Marshal(val)
		return string(b)
	}

	return modelToCommentedJSON(fm.Model, indent)
}

func arrayModelToCommentedJSON(m *apimodel.ObjectModel, indent int) string {
	if m.Items == nil {
		return "[]"
	}
	pad := strings.Repeat("    ", indent+1)
	item := modelToCommentedJSON(m.Items, indent+1)
	return fmt.Sprintf("[\n%s%s\n%s]", pad, item, strings.Repeat("    ", indent))
}

func mapModelToCommentedJSON(m *apimodel.ObjectModel, indent int) string {
	if m.ValueModel == nil {
		return "{}"
	}
	pad := strings.Repeat("    ", indent+1)
	val := modelToCommentedJSON(m.ValueModel, indent+1)
	return fmt.Sprintf("{\n%s\"key\": %s\n%s}", pad, val, strings.Repeat("    ", indent))
}
