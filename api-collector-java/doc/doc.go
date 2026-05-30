package doc

import (
	"strings"

	"github.com/tangcent/apilot/api-collector-java/parser"
)

type FieldDocumentation struct {
	Description string
	Example     string
	Required    *bool
	Default     string
	Enum        []string
}

func CleanJavaDoc(raw string) string {
	return cleanJavaDoc(raw, false)
}

func ParseJavaDocParams(raw string) map[string]string {
	cleaned := cleanJavaDoc(raw, true)
	if cleaned == "" {
		return nil
	}

	params := make(map[string]string)
	for _, line := range strings.Split(cleaned, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "@param ") {
			continue
		}
		rest := strings.TrimSpace(strings.TrimPrefix(line, "@param "))
		parts := strings.Fields(rest)
		if len(parts) == 0 {
			continue
		}
		name := parts[0]
		params[name] = strings.TrimSpace(strings.TrimPrefix(rest, name))
	}
	if len(params) == 0 {
		return nil
	}
	return params
}

func EndpointDescription(annotations []parser.Annotation, javaDoc string) string {
	if ann := findAnnotation(annotations, "Operation"); ann != nil {
		return joinDistinct(ann.Params["summary"], ann.Params["description"])
	}
	if ann := findAnnotation(annotations, "ApiOperation"); ann != nil {
		return joinDistinct(ann.Params["value"], ann.Params["notes"])
	}
	return strings.TrimSpace(javaDoc)
}

func ParameterDocumentation(annotations []parser.Annotation, javaDoc string) FieldDocumentation {
	doc := FieldDocumentation{Description: strings.TrimSpace(javaDoc)}
	for _, name := range []string{"Parameter", "Schema"} {
		if ann := findAnnotation(annotations, name); ann != nil {
			applyCommonParams(&doc, ann.Params)
		}
	}
	return doc
}

func FieldDocumentationFor(annotations []parser.Annotation, javaDoc string) FieldDocumentation {
	doc := FieldDocumentation{Description: strings.TrimSpace(javaDoc)}
	if ann := findAnnotation(annotations, "ApiModelProperty"); ann != nil {
		applyApiModelProperty(&doc, ann.Params)
	}
	if ann := findAnnotation(annotations, "Schema"); ann != nil {
		applyCommonParams(&doc, ann.Params)
	}
	return doc
}

func cleanJavaDoc(raw string, includeTags bool) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	raw = strings.TrimPrefix(raw, "/**")
	raw = strings.TrimSuffix(raw, "*/")

	var lines []string
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		line = strings.TrimPrefix(line, "*")
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if !includeTags && strings.HasPrefix(line, "@") {
			continue
		}
		lines = append(lines, line)
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

func applyApiModelProperty(doc *FieldDocumentation, params map[string]string) {
	if value := firstNonEmpty(params["value"], params["notes"]); value != "" {
		doc.Description = value
	}
	applyCommonParams(doc, params)
}

func applyCommonParams(doc *FieldDocumentation, params map[string]string) {
	if desc := firstNonEmpty(params["description"], params["value"]); desc != "" {
		doc.Description = desc
	}
	if example := params["example"]; example != "" {
		doc.Example = example
	}
	if def := params["defaultValue"]; def != "" {
		doc.Default = def
	}
	if required, ok := parseBool(params["required"]); ok {
		doc.Required = &required
	}
	if values := splitCSV(params["allowableValues"]); len(values) > 0 {
		doc.Enum = values
	}
}

func findAnnotation(annotations []parser.Annotation, name string) *parser.Annotation {
	for i := range annotations {
		if annotations[i].Name == name {
			return &annotations[i]
		}
	}
	return nil
}

func joinDistinct(first, second string) string {
	first = strings.TrimSpace(first)
	second = strings.TrimSpace(second)
	if first == "" {
		return second
	}
	if second == "" || second == first {
		return first
	}
	return first + "\n\n" + second
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func parseBool(value string) (bool, bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "true":
		return true, true
	case "false":
		return false, true
	default:
		return false, false
	}
}

func splitCSV(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
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
