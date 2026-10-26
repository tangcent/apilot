package markdown

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"unsafe"

	model "github.com/tangcent/apilot/api-model"
)

// DocMeta carries what the pipeline knows about the export as a whole. A
// document is otherwise unidentifiable: the filename is the only thing naming
// it, and an export to stdout has not even that (#170).
type DocMeta struct {
	// ProjectName is the project root the endpoints were collected from.
	ProjectName string
	// Collector is the collector that read the source, e.g. "java".
	Collector string
	// Frameworks are the frameworks the collector actually matched, e.g.
	// ["spring-mvc"]. Empty when the collector cannot tell.
	Frameworks []string
}

// MarkdownDoc is the template's data. Title is the H1 text and Summary is the
// metadata that follows it on the same line; both are computed in Go rather
// than assembled in the template, so the punctuation and the plural of
// "endpoint" are testable.
type MarkdownDoc struct {
	Title   string
	Summary string
	Groups  []EndpointGroup
}

type EndpointGroup struct {
	Folder    string
	Endpoints []EndpointView
}

type EndpointView struct {
	// Heading is the endpoint heading, unique within the document. The name
	// alone is not: resources that share method names produce identical `###`
	// headings, so no anchor can address one of them (#170).
	Heading string
	// RouteHeading is the same, built from the method and path. The simple
	// variant headings on the route rather than on the name.
	RouteHeading string

	Name        string
	Protocol    string
	Description string

	Path      string
	Method    string
	Service   string
	Streaming string
	FullPath  string

	// ContentType and Accept carry the media types the source declares
	// (@Consumes / @Produces). Empty when it declares none, so a collector
	// default never poses as a declaration.
	ContentType string
	Accept      string

	PathParams  []ParamRow
	QueryParams []ParamRow
	Headers     []HeaderRow
	FormParams  []FormRow
	BodyRows    []BodyRow
	BodyDemo    string

	ResponseBodyRows []BodyRow
	ResponseDemo     string

	HasRequest  bool
	HasResponse bool
}

type ParamRow struct {
	Name        string
	Value       string
	Required    string
	Type        string
	Description string
}

type HeaderRow struct {
	Name        string
	Value       string
	Required    string
	Type        string
	Description string
}

type FormRow struct {
	Name        string
	Value       string
	Required    string
	Type        string
	Description string
}

type BodyRow struct {
	Name string
	Type string
	Desc string
}

// renderOptions collects the choices that shape the rendered tables.
type renderOptions struct {
	outputDemo bool
	maxVisits  int
	// sortFields renders object fields alphabetically instead of in the order
	// the source declares them. Opt-in: declaration order is the default
	// because it is what the source says, because the Query and Path tables
	// already keep it, and because a field added at the end of a DTO should
	// appear at the end of the table in review (#170).
	sortFields bool
}

func buildMarkdownDoc(endpoints []model.ApiEndpoint, moduleName string, meta DocMeta, opts renderOptions) MarkdownDoc {
	grouped := groupByFolder(endpoints)
	groups := make([]EndpointGroup, 0, len(grouped))

	folderOrder := make([]string, 0, len(grouped))
	for folder := range grouped {
		folderOrder = append(folderOrder, folder)
	}
	sort.Slice(folderOrder, func(i, j int) bool {
		if folderOrder[i] == "" {
			return false
		}
		if folderOrder[j] == "" {
			return true
		}
		return folderOrder[i] < folderOrder[j]
	})

	// placed holds every view with the folder it sits under, so the headings
	// can be assigned after the whole document is known: a name that repeats
	// anywhere is qualified everywhere, rather than the first occurrence
	// keeping the bare name and the rest carrying a qualification.
	type placed struct {
		view   *EndpointView
		folder string
	}
	var all []placed

	for _, folder := range folderOrder {
		eps := grouped[folder]
		// Sized up front: the views are addressed by pointer below, and a
		// slice that grows out of its array would leave earlier pointers
		// pointing at the old one.
		views := make([]EndpointView, len(eps))
		for i, ep := range eps {
			views[i] = buildEndpointView(ep, opts)
			all = append(all, placed{view: &views[i], folder: folder})
		}
		groups = append(groups, EndpointGroup{
			Folder:    folder,
			Endpoints: views,
		})
	}

	nameCount := make(map[string]int, len(all))
	routeCount := make(map[string]int, len(all))
	for _, p := range all {
		nameCount[endpointLabel(p.view)]++
		routeCount[routeText(p.view)]++
	}

	// One deduplicator per heading namespace. The document is walked in output
	// order, so the assignment is stable across runs.
	nameHeadings := newHeadingDeduplicator()
	routeHeadings := newHeadingDeduplicator()
	for _, p := range all {
		label := endpointLabel(p.view)
		route := routeText(p.view)
		p.view.Heading = nameHeadings.assign(
			headingCandidates(label, p.folder, route, nameCount[label] > 1)...)
		p.view.RouteHeading = routeHeadings.assign(
			headingCandidates(route, p.folder, label, routeCount[route] > 1)...)
	}

	return MarkdownDoc{
		Title:   docTitle(moduleName, meta),
		Summary: docSummary(len(endpoints), meta),
		Groups:  groups,
	}
}

// routeText is the "METHOD /path" label of an endpoint. A gRPC endpoint has no
// method, so its path is the label.
func routeText(view *EndpointView) string {
	return strings.TrimSpace(view.Method + " " + view.Path)
}

// endpointLabel is the name an endpoint is headed by. An endpoint the source
// did not name is headed by its route.
func endpointLabel(view *EndpointView) string {
	if view.Name != "" {
		return view.Name
	}
	return routeText(view)
}

// headingCandidates lists a heading from the bare one to the most qualified
// one. The folder comes before the route: a name repeated across resources is
// told apart by the resource first, which is what a reader looks for.
// duplicated drops the bare candidate, so every endpoint sharing a name is
// qualified the same way rather than one of them staying bare.
func headingCandidates(primary, folder, secondary string, duplicated bool) []string {
	if primary == "" {
		primary = secondary
	}
	if primary == "" {
		primary = "endpoint"
	}
	var candidates []string
	if !duplicated {
		candidates = append(candidates, primary)
	}
	if folder != "" {
		candidates = append(candidates, folder+" / "+primary)
	}
	if secondary != "" && secondary != primary {
		candidates = append(candidates, secondary+" / "+primary)
	}
	return candidates
}

// headingDeduplicator hands out headings that are unique within one document.
type headingDeduplicator struct {
	used map[string]bool
}

func newHeadingDeduplicator() *headingDeduplicator {
	return &headingDeduplicator{used: make(map[string]bool)}
}

// assign returns the first candidate nobody has taken. When every candidate is
// taken — the same name in the same resource — the most qualified one gets a
// counter, which keeps the heading unique and guessable.
func (d *headingDeduplicator) assign(candidates ...string) string {
	for _, c := range candidates {
		if c == "" || d.used[c] {
			continue
		}
		d.used[c] = true
		return c
	}
	last := "endpoint"
	for _, c := range candidates {
		if c != "" {
			last = c
		}
	}
	for i := 2; ; i++ {
		c := fmt.Sprintf("%s (%d)", last, i)
		if !d.used[c] {
			d.used[c] = true
			return c
		}
	}
}

// docTitle is the H1 text. An explicit module name wins; otherwise the project
// the endpoints were collected from names the document.
func docTitle(moduleName string, meta DocMeta) string {
	if moduleName != "" {
		return moduleName
	}
	if meta.ProjectName != "" {
		return meta.ProjectName + " API"
	}
	return "API"
}

// docSummary is the metadata after the title: where the document came from and
// how much of the project it covers. Empty when the pipeline supplied nothing,
// so a document rendered without it is unchanged.
func docSummary(endpointCount int, meta DocMeta) string {
	var parts []string
	if meta.Collector != "" {
		source := meta.Collector
		if len(meta.Frameworks) > 0 {
			source += " / " + strings.Join(meta.Frameworks, ", ")
		}
		parts = append(parts, source)
	}
	if endpointCount > 0 {
		noun := "endpoints"
		if endpointCount == 1 {
			noun = "endpoint"
		}
		parts = append(parts, fmt.Sprintf("%d %s", endpointCount, noun))
	}
	return strings.Join(parts, ", ")
}

func groupByFolder(endpoints []model.ApiEndpoint) map[string][]model.ApiEndpoint {
	result := make(map[string][]model.ApiEndpoint)
	for _, ep := range endpoints {
		folder := ep.Folder
		result[folder] = append(result[folder], ep)
	}
	return result
}

func buildEndpointView(ep model.ApiEndpoint, opts renderOptions) EndpointView {
	view := EndpointView{
		Name:        ep.Name,
		Protocol:    ep.Protocol,
		Description: ep.Description,
		Path:        ep.Path,
		Method:      ep.Method,
		ContentType: joinMediaTypes(ep.Consumes),
		Accept:      joinMediaTypes(ep.Produces),
	}

	if ep.Protocol == "grpc" {
		view.Service = strVal(ep.Metadata, "serviceName")
		view.Streaming = strVal(ep.Metadata, "streamingType")
		view.FullPath = ep.Path
	}

	for _, p := range ep.Parameters {
		row := ParamRow{
			Name:        mdEscape(p.Name),
			Value:       mdEscape(paramValue(p)),
			Required:    boolToStr(p.Required),
			Type:        mdEscape(p.TypeName),
			Description: mdEscape(p.Description),
		}
		switch p.In {
		case "path":
			view.PathParams = append(view.PathParams, row)
		case "query":
			view.QueryParams = append(view.QueryParams, row)
		case "header":
			view.Headers = append(view.Headers, HeaderRow{
				Name:        row.Name,
				Value:       row.Value,
				Required:    row.Required,
				Type:        row.Type,
				Description: row.Description,
			})
		case "form":
			view.FormParams = append(view.FormParams, FormRow{
				Name:        mdEscape(p.Name),
				Value:       mdEscape(paramValue(p)),
				Required:    boolToStr(p.Required),
				Type:        strings.ToLower(p.Type),
				Description: mdEscape(p.Description),
			})
		}
	}

	for _, h := range ep.Headers {
		view.Headers = append(view.Headers, HeaderRow{
			Name:        mdEscape(h.Name),
			Value:       mdEscape(h.Value),
			Required:    boolToStr(h.Required),
			Description: mdEscape(h.Description),
		})
	}

	if ep.RequestBody != nil {
		if ep.RequestBody.Body != nil {
			view.BodyRows = flattenObjectModel(ep.RequestBody.Body, ep.RequestBody.Name, opts)
		}
		if opts.outputDemo {
			view.BodyDemo = generateDemo(ep.RequestBody)
		}
	}

	if ep.Response != nil {
		if ep.Response.Body != nil {
			view.ResponseBodyRows = flattenObjectModel(ep.Response.Body, "", opts)
		}
		if opts.outputDemo {
			view.ResponseDemo = generateDemo(ep.Response)
		}
	}

	view.HasRequest = len(view.PathParams) > 0 ||
		len(view.QueryParams) > 0 ||
		len(view.Headers) > 0 ||
		len(view.FormParams) > 0 ||
		ep.RequestBody != nil

	view.HasResponse = ep.Response != nil

	return view
}

func paramValue(p model.ApiParameter) string {
	if p.Default != "" {
		return p.Default
	}
	if p.Example != "" {
		return p.Example
	}
	return ""
}

// joinMediaTypes renders the declared media types of an endpoint as one cell
// value. An endpoint that declares none renders nothing: a collector default
// must not pose as a declaration.
func joinMediaTypes(mediaTypes []string) string {
	return strings.Join(mediaTypes, ", ")
}

func strVal(m map[string]any, key string) string {
	if m == nil {
		return ""
	}
	v, ok := m[key]
	if !ok {
		return ""
	}
	return fmt.Sprintf("%v", v)
}

func boolToStr(b bool) string {
	if b {
		return "YES"
	}
	return "NO"
}

func mdEscape(s string) string {
	s = strings.ReplaceAll(s, "\n", "<br>")
	s = strings.ReplaceAll(s, "|", "\\|")
	return s
}

// flattenObjectModel turns a body schema into table rows. name is the parameter
// that supplies the body, empty when the schema stands on its own (a response,
// or a collector that reports no parameter). A body with fields of its own
// ignores name: the field rows speak for themselves at the top level. A body
// without fields keeps it, because otherwise nothing in the table says which
// payload the endpoint expects.
func flattenObjectModel(m *model.ObjectModel, name string, opts renderOptions) []BodyRow {
	if m == nil {
		return nil
	}
	visitCounts := make(map[uintptr]int)
	var rows []BodyRow
	flattenModel(m, name, 0, &rows, visitCounts, opts)
	return rows
}

// fieldNames lists the fields of an object in the order the table renders
// them: declaration order, or alphabetical when sortFields is set.
func fieldNames(m *model.ObjectModel, opts renderOptions) []string {
	if opts.sortFields {
		return sortedFieldNames(m.Fields)
	}
	return m.OrderedFieldNames()
}

func objPtr(m *model.ObjectModel) uintptr {
	return uintptr(unsafe.Pointer(m))
}

func flattenModel(m *model.ObjectModel, name string, depth int, rows *[]BodyRow, visitCounts map[uintptr]int, opts renderOptions) {
	if m == nil {
		return
	}

	switch m.Kind {
	case model.KindObject:
		id := objPtr(m)
		if visitCounts[id] >= opts.maxVisits {
			return
		}
		visitCounts[id]++
		for _, fieldName := range fieldNames(m, opts) {
			fm := m.Fields[fieldName]
			flattenFieldRow(fieldName, fm, depth, rows, visitCounts, opts)
		}
		visitCounts[id]--

	case model.KindArray:
		// A collection has no field of its own, so the parameter names it and
		// the index picks one element out of it: rows are ids[0] with long,
		// never [0] with long[]. An unnamed collection keeps the bare index.
		if m.Items != nil {
			flattenArrayItem(m.Items, name+"[0]", depth, rows, visitCounts, opts)
		}

	case model.KindSingle:
		*rows = append(*rows, BodyRow{
			Name: mdEscape(name),
			Type: mdEscape(m.TypeName),
			Desc: "",
		})

	case model.KindMap:
		keyType := formatType(m.KeyModel)
		valType := formatType(m.ValueModel)
		*rows = append(*rows,
			BodyRow{Name: "key", Type: mdEscape(keyType), Desc: ""},
			BodyRow{Name: "value", Type: mdEscape(valType), Desc: ""},
		)

	case model.KindRef:
		*rows = append(*rows, BodyRow{
			Name: mdEscape(name),
			Type: mdEscape(m.TypeName),
			Desc: "",
		})
	}
}

func flattenArrayItem(item *model.ObjectModel, prefix string, depth int, rows *[]BodyRow, visitCounts map[uintptr]int, opts renderOptions) {
	if item == nil {
		return
	}

	switch item.Kind {
	case model.KindObject:
		id := objPtr(item)
		if visitCounts[id] >= opts.maxVisits {
			return
		}
		visitCounts[id]++
		for _, name := range fieldNames(item, opts) {
			fm := item.Fields[name]
			flattenFieldRow(prefix+"."+name, fm, depth, rows, visitCounts, opts)
		}
		visitCounts[id]--

	case model.KindArray:
		if item.Items != nil {
			flattenArrayItem(item.Items, prefix+"[0]", depth, rows, visitCounts, opts)
		}

	case model.KindSingle:
		// The row is one element, so the type cell is the element type. The
		// container type belongs on a row that names the collection as a whole,
		// which is how a field row reports it.
		*rows = append(*rows, BodyRow{
			Name: mdEscape(prefix),
			Type: mdEscape(item.TypeName),
			Desc: "",
		})

	case model.KindMap:
		keyType := formatType(item.KeyModel)
		valType := formatType(item.ValueModel)
		*rows = append(*rows,
			BodyRow{Name: mdEscape(prefix + ".key"), Type: mdEscape(keyType), Desc: ""},
			BodyRow{Name: mdEscape(prefix + ".value"), Type: mdEscape(valType), Desc: ""},
		)

	case model.KindRef:
		*rows = append(*rows, BodyRow{
			Name: mdEscape(prefix),
			Type: mdEscape(item.TypeName),
			Desc: "",
		})
	}
}

func flattenFieldRow(fieldName string, fm *model.FieldModel, depth int, rows *[]BodyRow, visitCounts map[uintptr]int, opts renderOptions) {
	if fm == nil || fm.Model == nil {
		return
	}

	indent := ""
	if depth > 0 {
		indent = strings.Repeat("&ensp;&ensp;", depth) + "&#124;─"
	}

	typeStr := formatType(fm.Model)
	desc := buildFieldDescription(fm)

	*rows = append(*rows, BodyRow{
		Name: indent + mdEscape(fieldName),
		Type: mdEscape(typeStr),
		Desc: mdEscape(desc),
	})

	if fm.Model.Kind == model.KindObject {
		id := objPtr(fm.Model)
		if visitCounts[id] >= opts.maxVisits {
			return
		}
		visitCounts[id]++
		for _, name := range fieldNames(fm.Model, opts) {
			nestedFM := fm.Model.Fields[name]
			flattenFieldRow(name, nestedFM, depth+1, rows, visitCounts, opts)
		}
		visitCounts[id]--
	} else if fm.Model.Kind == model.KindArray && fm.Model.Items != nil {
		if fm.Model.Items.Kind == model.KindObject {
			id := objPtr(fm.Model.Items)
			if visitCounts[id] >= opts.maxVisits {
				return
			}
			visitCounts[id]++
			for _, name := range fieldNames(fm.Model.Items, opts) {
				nestedFM := fm.Model.Items.Fields[name]
				flattenFieldRow(name, nestedFM, depth+1, rows, visitCounts, opts)
			}
			visitCounts[id]--
		}
	}
}

func buildFieldDescription(fm *model.FieldModel) string {
	if fm == nil {
		return ""
	}
	var parts []string

	if fm.Comment != "" {
		parts = append(parts, fm.Comment)
	}

	if len(fm.Options) > 0 {
		var optParts []string
		for _, opt := range fm.Options {
			s := fmt.Sprintf("%v", opt.Value)
			if opt.Desc != "" {
				s += " :" + opt.Desc
			}
			optParts = append(optParts, s)
		}
		parts = append(parts, strings.Join(optParts, "<br>"))
	}

	return strings.Join(parts, "<br>")
}

func formatType(m *model.ObjectModel) string {
	if m == nil {
		return ""
	}
	switch m.Kind {
	case model.KindSingle:
		return m.TypeName
	case model.KindArray:
		return formatType(m.Items) + "[]"
	case model.KindObject:
		return "object"
	case model.KindMap:
		return "map"
	case model.KindRef:
		return m.TypeName
	default:
		return m.TypeName
	}
}

func sortedFieldNames(fields map[string]*model.FieldModel) []string {
	names := make([]string, 0, len(fields))
	for name := range fields {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func generateDemo(body *model.ApiBody) string {
	if body == nil {
		return ""
	}
	if body.Example != nil {
		b, err := json.MarshalIndent(body.Example, "", "  ")
		if err != nil {
			return ""
		}
		return string(b)
	}
	if body.Body != nil {
		return objectModelToJSON(body.Body)
	}
	return ""
}

func objectModelToJSON(m *model.ObjectModel) string {
	if m == nil {
		return "{}"
	}
	val := modelToValue(m)
	b, err := json.MarshalIndent(val, "", "  ")
	if err != nil {
		return "{}"
	}
	return string(b)
}

func modelToValue(m *model.ObjectModel) any {
	if m == nil {
		return nil
	}
	switch m.Kind {
	case model.KindSingle:
		return singleModelValue(m.TypeName)
	case model.KindObject:
		return objectModelValue(m)
	case model.KindArray:
		return arrayModelValue(m)
	case model.KindMap:
		return mapModelValue(m)
	case model.KindRef:
		return map[string]any{"$ref": m.TypeName}
	default:
		return nil
	}
}

func singleModelValue(typeName string) any {
	switch typeName {
	case model.JsonTypeString:
		return ""
	case model.JsonTypeInt, model.JsonTypeLong:
		return 0
	case model.JsonTypeFloat, model.JsonTypeDouble:
		return 0.0
	case model.JsonTypeBoolean:
		return false
	case model.JsonTypeNull:
		return nil
	default:
		return ""
	}
}

func objectModelValue(m *model.ObjectModel) any {
	if m.Fields == nil || len(m.Fields) == 0 {
		return map[string]any{}
	}
	obj := make(map[string]any, len(m.Fields))
	// The demo follows the table: a reader comparing them should meet the same
	// fields in the same order.
	for _, name := range m.OrderedFieldNames() {
		fm := m.Fields[name]
		obj[name] = fieldModelValue(fm)
	}
	return obj
}

func fieldModelValue(fm *model.FieldModel) any {
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

func arrayModelValue(m *model.ObjectModel) any {
	if m.Items == nil {
		return []any{}
	}
	return []any{modelToValue(m.Items)}
}

func mapModelValue(m *model.ObjectModel) any {
	if m.ValueModel == nil {
		return map[string]any{}
	}
	return map[string]any{"key": modelToValue(m.ValueModel)}
}

func parseDemoValue(demo string, m *model.ObjectModel) any {
	if m == nil {
		return demo
	}
	if m.Kind == model.KindSingle {
		switch m.TypeName {
		case model.JsonTypeInt, model.JsonTypeLong:
			var v int64
			if _, err := fmt.Sscanf(demo, "%d", &v); err == nil {
				return v
			}
		case model.JsonTypeFloat, model.JsonTypeDouble:
			var v float64
			if _, err := fmt.Sscanf(demo, "%f", &v); err == nil {
				return v
			}
		case model.JsonTypeBoolean:
			if demo == "true" {
				return true
			}
			if demo == "false" {
				return false
			}
		}
	}
	if m.Kind == model.KindObject || m.Kind == model.KindArray || m.Kind == model.KindMap {
		var v any
		if err := json.Unmarshal([]byte(demo), &v); err == nil {
			return v
		}
	}
	return demo
}
