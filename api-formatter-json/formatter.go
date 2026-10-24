// Package json implements the Formatter interface producing machine-readable
// JSON, so a pipeline can assert on the exported endpoints instead of parsing
// markdown.
package json

import (
	stdjson "encoding/json"
	"sort"

	formatter "github.com/tangcent/apilot/api-formatter"
	model "github.com/tangcent/apilot/api-model"
)

// JSONFormatter renders endpoints as a deterministic JSON document.
type JSONFormatter struct{}

// New returns a new JSONFormatter.
func New() formatter.Formatter { return &JSONFormatter{} }

func (f *JSONFormatter) Name() string { return "json" }

// Format produces a document holding one entry per endpoint. The output depends
// only on the collected endpoints — no timestamps, no absolute paths, no
// host-specific data — so the same input always produces byte-identical output.
func (f *JSONFormatter) Format(endpoints []model.ApiEndpoint, _ formatter.FormatOptions) ([]byte, error) {
	doc := document{Endpoints: make([]endpoint, 0, len(endpoints))}
	for _, ep := range sortedEndpoints(endpoints) {
		doc.Endpoints = append(doc.Endpoints, buildEndpoint(ep))
	}
	return stdjson.MarshalIndent(doc, "", "  ")
}

// document is the top-level object. Consumers gate on the length of
// endpoints, e.g. `apilot <dir> --formatter json | jq '.endpoints | length'`.
type document struct {
	Endpoints []endpoint `json:"endpoints"`
}

type endpoint struct {
	Name        string   `json:"name"`
	Method      string   `json:"method,omitempty"` // empty for non-HTTP protocols
	Path        string   `json:"path"`
	Protocol    string   `json:"protocol"`
	Description string   `json:"description,omitempty"`
	Folder      string   `json:"folder,omitempty"`
	Request     *request `json:"request,omitempty"`
	Response    *body    `json:"response,omitempty"`
}

type request struct {
	Parameters *parameters `json:"parameters,omitempty"`
	Body       *body       `json:"body,omitempty"`
}

// parameters groups input parameters by location, in the order the collector
// reported them. Groups holding no parameter are omitted; each parameter keeps
// its own declaration order.
type parameters struct {
	Path   []parameter `json:"path,omitempty"`
	Query  []parameter `json:"query,omitempty"`
	Header []parameter `json:"header,omitempty"`
	Cookie []parameter `json:"cookie,omitempty"`
	Form   []parameter `json:"form,omitempty"`
	Body   []parameter `json:"body,omitempty"`
}

type parameter struct {
	Name        string   `json:"name"`
	Type        string   `json:"type"`
	Required    bool     `json:"required"`
	Default     string   `json:"default,omitempty"`
	Description string   `json:"description,omitempty"`
	Example     string   `json:"example,omitempty"`
	Enum        []string `json:"enum,omitempty"`
	Value       string   `json:"value,omitempty"` // fixed value declared by the source, e.g. a header
}

type body struct {
	MediaType string `json:"mediaType,omitempty"`
	Schema    *node  `json:"schema,omitempty"`
}

// node is one node of a schema tree. A body root is a node without a name; a
// member of an object is a node carrying a name plus the documentation that
// belongs to that member. Kind is one of single, object, array, map, ref.
type node struct {
	Name        string   `json:"name,omitempty"`
	Kind        string   `json:"kind"`
	Type        string   `json:"type,omitempty"` // primitive name, or the referred type name
	Required    bool     `json:"required,omitempty"`
	Description string   `json:"description,omitempty"`
	Default     string   `json:"default,omitempty"`
	Demo        string   `json:"demo,omitempty"`
	Generic     bool     `json:"generic,omitempty"`
	Options     []option `json:"options,omitempty"`
	Fields      []*node  `json:"fields,omitempty"` // object, sorted by name
	Items       *node    `json:"items,omitempty"`  // array
	Key         *node    `json:"key,omitempty"`    // map
	Value       *node    `json:"value,omitempty"`  // map
}

type option struct {
	Value any    `json:"value"`
	Desc  string `json:"desc,omitempty"`
}

// sortedEndpoints orders endpoints by folder, name, method and path, with the
// ones that belong to no folder last — the same grouping the markdown formatter
// uses. Collectors report endpoints in traversal order, which is not a
// documented contract; sorting keeps the document identical no matter how the
// tree was walked.
func sortedEndpoints(endpoints []model.ApiEndpoint) []model.ApiEndpoint {
	sorted := make([]model.ApiEndpoint, len(endpoints))
	copy(sorted, endpoints)
	sort.SliceStable(sorted, func(i, j int) bool {
		a, b := sorted[i], sorted[j]
		if (a.Folder == "") != (b.Folder == "") {
			return b.Folder == ""
		}
		if a.Folder != b.Folder {
			return a.Folder < b.Folder
		}
		if a.Name != b.Name {
			return a.Name < b.Name
		}
		if a.Method != b.Method {
			return a.Method < b.Method
		}
		return a.Path < b.Path
	})
	return sorted
}

func buildEndpoint(ep model.ApiEndpoint) endpoint {
	return endpoint{
		Name:        ep.Name,
		Method:      ep.Method,
		Path:        ep.Path,
		Protocol:    ep.Protocol,
		Folder:      ep.Folder,
		Description: ep.Description,
		Request:     buildRequest(ep),
		Response:    buildBody(ep.Response),
	}
}

func buildRequest(ep model.ApiEndpoint) *request {
	params := buildParameters(ep)
	body := buildBody(ep.RequestBody)
	if params == nil && body == nil {
		return nil
	}
	return &request{Parameters: params, Body: body}
}

func buildParameters(ep model.ApiEndpoint) *parameters {
	var out parameters

	for _, p := range ep.Parameters {
		entry := parameter{
			Name:        p.Name,
			Type:        p.Type,
			Required:    p.Required,
			Default:     p.Default,
			Description: p.Description,
			Example:     p.Example,
			Enum:        p.Enum,
		}
		switch p.In {
		case "path":
			out.Path = append(out.Path, entry)
		case "query":
			out.Query = append(out.Query, entry)
		case "header":
			out.Header = append(out.Header, entry)
		case "cookie":
			out.Cookie = append(out.Cookie, entry)
		case "form":
			out.Form = append(out.Form, entry)
		case "body":
			out.Body = append(out.Body, entry)
		}
	}

	// Headers declared outside the parameter list belong in the same group, so
	// a header diff does not have to look in two places.
	for _, h := range ep.Headers {
		out.Header = append(out.Header, parameter{
			Name:        h.Name,
			Type:        "text",
			Required:    h.Required,
			Value:       h.Value,
			Description: h.Description,
			Example:     h.Example,
		})
	}

	if len(out.Path) == 0 && len(out.Query) == 0 && len(out.Header) == 0 &&
		len(out.Cookie) == 0 && len(out.Form) == 0 && len(out.Body) == 0 {
		return nil
	}
	return &out
}

func buildBody(b *model.ApiBody) *body {
	if b == nil || (b.Body == nil && b.MediaType == "") {
		return nil
	}
	return &body{
		MediaType: b.MediaType,
		Schema:    buildNode(b.Body, nil),
	}
}

// buildNode converts an object model into a schema tree. onPath holds the
// models already expanded on the current branch: a model that refers back to
// one of them terminates as a ref node instead of expanding forever.
func buildNode(m *model.ObjectModel, onPath []*model.ObjectModel) *node {
	if m == nil {
		return nil
	}

	switch m.Kind {
	case model.KindObject:
		if onPathContains(onPath, m) {
			return &node{Kind: string(model.KindRef), Type: m.TypeName}
		}
		out := &node{Kind: string(m.Kind)}
		branch := make([]*model.ObjectModel, len(onPath)+1)
		copy(branch, onPath)
		branch[len(onPath)] = m
		for _, name := range sortedFieldNames(m.Fields) {
			if field := buildField(name, m.Fields[name], branch); field != nil {
				out.Fields = append(out.Fields, field)
			}
		}
		return out

	case model.KindArray:
		return &node{Kind: string(m.Kind), Items: buildNode(m.Items, onPath)}

	case model.KindMap:
		return &node{
			Kind:  string(m.Kind),
			Key:   buildNode(m.KeyModel, onPath),
			Value: buildNode(m.ValueModel, onPath),
		}

	default: // single, ref
		return &node{Kind: string(m.Kind), Type: m.TypeName}
	}
}

func buildField(name string, fm *model.FieldModel, onPath []*model.ObjectModel) *node {
	if fm == nil || fm.Model == nil {
		return nil
	}
	out := buildNode(fm.Model, onPath)
	if out == nil {
		return nil
	}
	out.Name = name
	out.Required = fm.Required
	out.Description = fm.Comment
	out.Default = fm.DefaultValue
	out.Demo = fm.Demo
	out.Generic = fm.Generic
	for _, opt := range fm.Options {
		out.Options = append(out.Options, option{Value: opt.Value, Desc: opt.Desc})
	}
	return out
}

func onPathContains(onPath []*model.ObjectModel, m *model.ObjectModel) bool {
	for _, seen := range onPath {
		if seen == m {
			return true
		}
	}
	return false
}

// sortedFieldNames orders an object's members. The model stores them in a map,
// so the declaration order is gone by the time a formatter sees them; sorting
// is the only order that is stable across runs and machines.
func sortedFieldNames(fields map[string]*model.FieldModel) []string {
	names := make([]string, 0, len(fields))
	for name := range fields {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
