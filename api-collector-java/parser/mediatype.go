package parser

import "strings"

// mediaTypeConstants maps the javax.ws.rs.core.MediaType,
// jakarta.ws.rs.core.MediaType and org.springframework.http.MediaType
// constants to the wire values they stand for. Source code almost always
// writes the constant (`@Consumes(MediaType.MULTIPART_FORM_DATA)`), and
// exporting that text verbatim would leak Java syntax into the document.
var mediaTypeConstants = map[string]string{
	"APPLICATION_JSON":            "application/json",
	"APPLICATION_XML":             "application/xml",
	"APPLICATION_ATOM_XML":        "application/atom+xml",
	"APPLICATION_FORM_URLENCODED": "application/x-www-form-urlencoded",
	"APPLICATION_OCTET_STREAM":    "application/octet-stream",
	"APPLICATION_PDF":             "application/pdf",
	"MULTIPART_FORM_DATA":         "multipart/form-data",
	"TEXT_PLAIN":                  "text/plain",
	"TEXT_XML":                    "text/xml",
	"TEXT_HTML":                   "text/html",
	"WILDCARD":                    "*/*",
}

// MediaTypes interprets the value of a media type annotation argument.
// Annotation values arrive as the raw source text, so an array literal is
// comma-joined (`"application/json,text/plain"`) and a constant reference
// keeps its qualifier (`MediaType.APPLICATION_JSON`). The result holds one
// wire media type per entry; a value that names no known constant is kept
// as written, so an unusual declaration still reaches the export.
func MediaTypes(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	var out []string
	for _, part := range strings.Split(raw, ",") {
		if v := NormalizeMediaType(part); v != "" {
			out = append(out, v)
		}
	}
	return out
}

// NormalizeMediaType resolves one annotation value to a wire media type.
// A value that already looks like a media type is returned trimmed;
// anything else is read as a constant reference and looked up by its
// last segment, so both MediaType.APPLICATION_JSON and a fully qualified
// jakarta.ws.rs.core.MediaType.APPLICATION_JSON resolve. A constant the
// table does not know is kept as written, so an unusual declaration still
// reaches the export.
func NormalizeMediaType(raw string) string {
	v := strings.TrimSpace(raw)
	v = strings.Trim(v, "\"'")
	if v == "" {
		return ""
	}
	if strings.Contains(v, "/") {
		return v
	}
	name := v
	if i := strings.LastIndex(name, "."); i >= 0 {
		name = name[i+1:]
	}
	if mapped, ok := mediaTypeConstants[strings.ToUpper(name)]; ok {
		return mapped
	}
	return v
}
