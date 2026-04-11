// Package markdown implements the Formatter interface producing Markdown output.
// Supports two variants via Params: "simple" (default, compact) and "detailed" (full schema expansion).
package markdown

import (
	_ "embed"
	"bytes"
	"encoding/json"
	"text/template"

	"github.com/tangcent/apilot/api-formatter"
	"github.com/tangcent/apilot/api-model"
)

//go:embed templates/simple.md.tmpl
var simpleTmpl string

//go:embed templates/detailed.md.tmpl
var detailedTmpl string

// Params holds markdown-specific formatting options.
type Params struct {
	// Variant selects the output template: "simple" (default) or "detailed".
	Variant string `json:"variant"`
}

// MarkdownFormatter formats endpoints as Markdown documents.
type MarkdownFormatter struct{}

// New returns a new MarkdownFormatter.
func New() formatter.Formatter { return &MarkdownFormatter{} }

func (f *MarkdownFormatter) Name() string { return "markdown" }

// Format renders endpoints using the selected template variant.
// An empty endpoints slice returns an empty Markdown document.
func (f *MarkdownFormatter) Format(endpoints []model.ApiEndpoint, opts formatter.FormatOptions) ([]byte, error) {
	var p Params
	if err := opts.DecodeParams(&p); err != nil {
		return nil, err
	}

	var tmplSrc string
	switch p.Variant {
	case "detailed":
		tmplSrc = detailedTmpl
	default:
		tmplSrc = simpleTmpl
	}

	funcMap := template.FuncMap{
		"json": func(v interface{}) (string, error) {
			b, err := json.MarshalIndent(v, "", "  ")
			if err != nil {
				return "", err
			}
			return string(b), nil
		},
	}

	t, err := template.New("md").Funcs(funcMap).Parse(tmplSrc)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, endpoints); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
