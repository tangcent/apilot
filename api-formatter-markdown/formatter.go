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
	// ModuleName overrides the document title. Left empty, the title comes
	// from ProjectName, and failing that is just "API".
	ModuleName string `json:"moduleName,omitempty"`
	// ProjectName is the project the endpoints were collected from, injected
	// by the engine. It names the document so two exports can be told apart.
	ProjectName string `json:"projectName,omitempty"`
	// Collector is the collector that read the source, e.g. "java".
	Collector string `json:"collector,omitempty"`
	// Frameworks are the frameworks the collector matched, e.g. ["spring-mvc"].
	Frameworks []string `json:"frameworks,omitempty"`
	// SortFields renders object fields alphabetically instead of in the order
	// the source declares them.
	SortFields bool `json:"sortFields,omitempty"`
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

	render := renderOptions{
		outputDemo: outputDemo,
		maxVisits:  p.MaxVisits,
		sortFields: p.SortFields,
	}
	if render.maxVisits <= 0 {
		render.maxVisits = 2
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

	doc := buildMarkdownDoc(endpoints, p.ModuleName, DocMeta{
		ProjectName: p.ProjectName,
		Collector:   p.Collector,
		Frameworks:  p.Frameworks,
	}, render)

	var buf bytes.Buffer
	if err := t.Execute(&buf, doc); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
