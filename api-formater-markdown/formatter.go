// Package markdown implements the Formatter interface producing Markdown output.
// Supports two format variants: "simple" (compact) and "detailed" (full schema expansion).
package markdown

import (
	_ "embed"
	"bytes"
	"text/template"

	"github.com/tangcent/apilot/api-collector/collector"
	"github.com/tangcent/apilot/api-formater/formater"
)

//go:embed templates/simple.md.tmpl
var simpleTmpl string

//go:embed templates/detailed.md.tmpl
var detailedTmpl string

// MarkdownFormatter formats endpoints as Markdown documents.
type MarkdownFormatter struct{}

// New returns a new MarkdownFormatter.
func New() formater.Formatter { return &MarkdownFormatter{} }

func (f *MarkdownFormatter) Name() string { return "markdown" }

func (f *MarkdownFormatter) SupportedFormats() []string { return []string{"simple", "detailed"} }

// Format renders endpoints using the selected template variant.
// An empty endpoints slice returns an empty Markdown document.
func (f *MarkdownFormatter) Format(endpoints []collector.ApiEndpoint, opts formater.FormatOptions) ([]byte, error) {
	variant := opts.Format
	if variant == "" {
		variant = "simple"
	}

	var tmplSrc string
	switch variant {
	case "detailed":
		tmplSrc = detailedTmpl
	default:
		tmplSrc = simpleTmpl
	}

	t, err := template.New("md").Parse(tmplSrc)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, endpoints); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
