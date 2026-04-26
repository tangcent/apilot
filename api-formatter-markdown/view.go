package markdown

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"unsafe"

	model "github.com/tangcent/apilot/api-model"
)

type MarkdownDoc struct {
	ModuleName string
	Groups     []EndpointGroup
}

type EndpointGroup struct {
	Folder    string
	Endpoints []EndpointView
}

type EndpointView struct {
	Name        string
	Protocol    string
	Description string

	Path      string
	Method    string
	Service   string
	Streaming string
	FullPath  string

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
	Description string
}

type HeaderRow struct {
	Name        string
	Value       string
	Required    string
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

func buildMarkdownDoc(endpoints []model.ApiEndpoint, moduleName string, outputDemo bool, maxVisits int) MarkdownDoc {
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

	for _, folder := range folderOrder {
		eps := grouped[folder]
		views := make([]EndpointView, 0, len(eps))
		for _, ep := range eps {
			views = append(views, buildEndpointView(ep, outputDemo, maxVisits))
		}
		groups = append(groups, EndpointGroup{
			Folder:    folder,
			Endpoints: views,
		})
	}

	return MarkdownDoc{
		ModuleName: moduleName,
		Groups:     groups,
	}
}

func groupByFolder(endpoints []model.ApiEndpoint) map[string][]model.ApiEndpoint {
	result := make(map[string][]model.ApiEndpoint)
	for _, ep := range endpoints {
		folder := ep.Folder
		result[folder] = append(result[folder], ep)
	}
	return result
}

func buildEndpointView(ep model.ApiEndpoint, outputDemo bool, maxVisits int) EndpointView {
	view := EndpointView{
		Name:        ep.Name,
		Protocol:    ep.Protocol,
		Description: ep.Description,
		Path:        ep.Path,
		Method:      ep.Method,
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
			Description: mdEscape(p.Description),
		}
		switch p.In {
		case "path":
			view.PathParams = append(view.PathParams, row)
		case "query":
			view.QueryParams = append(view.QueryParams, row)
		case "header":
			view.Headers = append(view.Headers, HeaderRow{
				Name:        mdEscape(p.Name),
				Value:       mdEscape(paramValue(p)),
				Required:    boolToStr(p.Required),
				Description: mdEscape(p.Description),
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
			view.BodyRows = flattenObjectModel(ep.RequestBody.Body, maxVisits)
		}
		if outputDemo {
			view.BodyDemo = generateDemo(ep.RequestBody)
		}
	}

	if ep.Response != nil {
		if ep.Response.Body != nil {
			view.ResponseBodyRows = flattenObjectModel(ep.Response.Body, maxVisits)
		}
		if outputDemo {
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

func flattenObjectModel(m *model.ObjectModel, maxVisits int) []BodyRow {
	if m == nil {
		return nil
	}
	visitCounts := make(map[uintptr]int)
	var rows []BodyRow
	flattenModel(m, 0, &rows, visitCounts, maxVisits)
	return rows
}

func objPtr(m *model.ObjectModel) uintptr {
	return uintptr(unsafe.Pointer(m))
}

func flattenModel(m *model.ObjectModel, depth int, rows *[]BodyRow, visitCounts map[uintptr]int, maxVisits int) {
	if m == nil {
		return
	}

	switch m.Kind {
	case model.KindObject:
		id := objPtr(m)
		if visitCounts[id] >= maxVisits {
			return
		}
		visitCounts[id]++
		names := sortedFieldNames(m.Fields)
		for _, name := range names {
			fm := m.Fields[name]
			flattenFieldRow(name, fm, depth, rows, visitCounts, maxVisits)
		}
		visitCounts[id]--

	case model.KindArray:
		if m.Items != nil {
			flattenArrayItem(m.Items, "[0]", depth, rows, visitCounts, maxVisits)
		}

	case model.KindSingle:
		*rows = append(*rows, BodyRow{
			Name: "",
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
			Name: "",
			Type: mdEscape(m.TypeName),
			Desc: "",
		})
	}
}

func flattenArrayItem(item *model.ObjectModel, prefix string, depth int, rows *[]BodyRow, visitCounts map[uintptr]int, maxVisits int) {
	if item == nil {
		return
	}

	switch item.Kind {
	case model.KindObject:
		id := objPtr(item)
		if visitCounts[id] >= maxVisits {
			return
		}
		visitCounts[id]++
		names := sortedFieldNames(item.Fields)
		for _, name := range names {
			fm := item.Fields[name]
			flattenFieldRow(prefix+"."+name, fm, depth, rows, visitCounts, maxVisits)
		}
		visitCounts[id]--

	case model.KindArray:
		if item.Items != nil {
			flattenArrayItem(item.Items, prefix+"[0]", depth, rows, visitCounts, maxVisits)
		}

	case model.KindSingle:
		*rows = append(*rows, BodyRow{
			Name: mdEscape(prefix),
			Type: mdEscape(item.TypeName + "[]"),
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
			Type: mdEscape(item.TypeName + "[]"),
			Desc: "",
		})
	}
}

func flattenFieldRow(fieldName string, fm *model.FieldModel, depth int, rows *[]BodyRow, visitCounts map[uintptr]int, maxVisits int) {
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
		if visitCounts[id] >= maxVisits {
			return
		}
		visitCounts[id]++
		names := sortedFieldNames(fm.Model.Fields)
		for _, name := range names {
			nestedFM := fm.Model.Fields[name]
			flattenFieldRow(name, nestedFM, depth+1, rows, visitCounts, maxVisits)
		}
		visitCounts[id]--
	} else if fm.Model.Kind == model.KindArray && fm.Model.Items != nil {
		if fm.Model.Items.Kind == model.KindObject {
			id := objPtr(fm.Model.Items)
			if visitCounts[id] >= maxVisits {
				return
			}
			visitCounts[id]++
			names := sortedFieldNames(fm.Model.Items.Fields)
			for _, name := range names {
				nestedFM := fm.Model.Items.Fields[name]
				flattenFieldRow(name, nestedFM, depth+1, rows, visitCounts, maxVisits)
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
	names := sortedFieldNames(m.Fields)
	for _, name := range names {
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
