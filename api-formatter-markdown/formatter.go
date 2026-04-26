package markdown

import (
	_ "embed"
	"bytes"
	"path"
	"text/template"

	"github.com/tangcent/apilot/api-formatter"
	"github.com/tangcent/apilot/api-model"
)

//go:embed templates/simple.md.tmpl
var simpleTmpl string

//go:embed templates/detailed.md.tmpl
var detailedTmpl string

type Params struct {
	Variant    string `json:"variant"`
	OutputDemo *bool  `json:"outputDemo,omitempty"`
	MaxVisits  int    `json:"maxVisits,omitempty"`
	ModuleName string `json:"moduleName,omitempty"`
}

type MarkdownFormatter struct{}

func New() formatter.Formatter { return &MarkdownFormatter{} }

func (f *MarkdownFormatter) Name() string { return "markdown" }

func (f *MarkdownFormatter) Format(endpoints []model.ApiEndpoint, opts formatter.FormatOptions) ([]byte, error) {
	var p Params
	if err := opts.DecodeParams(&p); err != nil {
		return nil, err
	}

	outputDemo := true
	if p.OutputDemo != nil {
		outputDemo = *p.OutputDemo
	}

	maxVisits := p.MaxVisits
	if maxVisits <= 0 {
		maxVisits = 2
	}

	moduleName := p.ModuleName
	if moduleName == "" {
		moduleName = "API"
	}

	var tmplSrc string
	switch p.Variant {
	case "detailed":
		tmplSrc = detailedTmpl
	default:
		tmplSrc = simpleTmpl
	}

	funcMap := template.FuncMap{
		"baseName": func(s string) string {
			return path.Base(s)
		},
	}

	t, err := template.New("md").Funcs(funcMap).Parse(tmplSrc)
	if err != nil {
		return nil, err
	}

	doc := buildMarkdownDoc(endpoints, moduleName, outputDemo, maxVisits)

	var buf bytes.Buffer
	if err := t.Execute(&buf, doc); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
