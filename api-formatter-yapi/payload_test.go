package yapi

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	model "github.com/tangcent/apilot/api-model"
)

func orderEndpoint() model.ApiEndpoint {
	body := model.ObjectModelFrom(map[string]*model.FieldModel{})
	body.SetField("orderNo", &model.FieldModel{Model: model.SingleModel(model.JsonTypeString), Required: true, Comment: "Order number."})
	body.SetField("amount", &model.FieldModel{Model: model.SingleModel(model.JsonTypeLong)})
	body.SetField("status", &model.FieldModel{
		Model:   model.SingleModel(model.JsonTypeString),
		Options: []model.FieldOption{{Value: "OPEN", Desc: "open"}, {Value: "CLOSED"}},
	})
	body.SetField("tags", &model.FieldModel{
		Model:   model.ArrayModel(model.SingleModel(model.JsonTypeString)),
		Options: []model.FieldOption{{Value: "A"}, {Value: "B"}},
	})

	resp := model.ObjectModelFrom(map[string]*model.FieldModel{})
	resp.SetField("code", &model.FieldModel{Model: model.SingleModel(model.JsonTypeInt)})
	data := model.ObjectModelFrom(map[string]*model.FieldModel{})
	data.SetField("createTime", &model.FieldModel{Model: model.SingleModel(model.JsonTypeLong)})
	data.SetField("items", &model.FieldModel{Model: model.ArrayModel(func() *model.ObjectModel {
		item := model.ObjectModelFrom(map[string]*model.FieldModel{})
		item.SetField("name", &model.FieldModel{Model: model.SingleModel(model.JsonTypeString)})
		return item
	}())})
	resp.SetField("data", &model.FieldModel{Model: data})

	return model.ApiEndpoint{
		Name:              "create",
		Folder:            "OrderController",
		FolderDescription: "Order management.\n\nCreates and lists orders.",
		Description:       "Create an order\nValidates the items first.\n\nThen reserves stock.",
		Path:              "orders/{orderNo}",
		Method:            "POST",
		Parameters: []model.ApiParameter{
			{Name: "orderNo", In: "path", Description: "order number"},
			{Name: "dryRun", In: "query", Default: "false", Description: "validate only"},
			{Name: "X-Tenant", In: "header", Required: true},
		},
		RequestBody: &model.ApiBody{MediaType: "application/json", Body: body},
		Response:    &model.ApiBody{MediaType: "application/json", Body: resp},
	}
}

func TestBuildInterface(t *testing.T) {
	it := buildInterface(orderEndpoint(), &schemaBuilder{})

	if it.Title != "Create an order" {
		t.Errorf("title = %q", it.Title)
	}
	if it.Markdown != "Validates the items first.\n\nThen reserves stock." {
		t.Errorf("markdown = %q", it.Markdown)
	}
	if it.Desc != "<body><p>Validates the items first.</p><p>Then reserves stock.</p></body>" {
		t.Errorf("desc = %q", it.Desc)
	}
	if it.Category() != "Order management." {
		t.Errorf("category = %q", it.Category())
	}
	if it.Path != "/orders/{orderNo}" || it.Method != "post" || it.Status != "done" {
		t.Errorf("path/method/status = %q %q %q", it.Path, it.Method, it.Status)
	}
	// A parameter without an example gets EasyYapi's mock for its name; a
	// query parameter with a default shows the default.
	if !reflect.DeepEqual(it.ReqParams, []PathParam{{Name: "orderNo", Example: "@string", Desc: "order number"}}) {
		t.Errorf("req_params = %#v", it.ReqParams)
	}
	if !reflect.DeepEqual(it.ReqQuery, []Query{{Name: "dryRun", Value: "false", Example: "false", Desc: "validate only", Required: "0"}}) {
		t.Errorf("req_query = %#v", it.ReqQuery)
	}
	wantHeaders := []Header{
		{Name: "X-Tenant", Required: "1"},
		{Name: "Content-Type", Value: "application/json", Example: "application/json", Required: "0"},
	}
	if !reflect.DeepEqual(it.ReqHeaders, wantHeaders) {
		t.Errorf("req_headers = %#v", it.ReqHeaders)
	}
	if it.ReqBodyType != "json" || it.ResBodyType != "json" || !it.ReqBodyIsJSONSchema || !it.ResBodyIsJSONSchema {
		t.Errorf("body types = %q %q", it.ReqBodyType, it.ResBodyType)
	}

	// The request schema keeps declaration order, lists required members on
	// the object, and carries enums with a pick mock.
	wantReq := `{"$schema":"http://json-schema.org/draft-04/schema#","type":"object","properties":{` +
		`"orderNo":{"type":"string","description":"Order number."},` +
		`"amount":{"type":"integer"},` +
		`"status":{"type":"string","enum":["OPEN","CLOSED"],"enumDesc":"OPEN: open","mock":{"mock":"@pick([\"OPEN\",\"CLOSED\"])"}},` +
		`"tags":{"type":"array","items":{"type":"string","enum":["A","B"],"mock":{"mock":"@pick([\"A\",\"B\"])"}}}},` +
		`"required":["orderNo"]}`
	if it.ReqBodyOther != wantReq {
		t.Errorf("req_body_other =\n%s\nwant\n%s", it.ReqBodyOther, wantReq)
	}
}

func TestBuildInterfaceFallbacks(t *testing.T) {
	it := buildInterface(model.ApiEndpoint{Name: "ping", Folder: "HealthController", Path: "/ping", Method: "GET"}, &schemaBuilder{})
	if it.Title != "ping" || it.Category() != "HealthController" || it.Desc != "" || it.Markdown != "" {
		t.Errorf("fallbacks = %q %q %q %q", it.Title, it.Category(), it.Desc, it.Markdown)
	}
	if it.ReqBodyType != "" || it.ResBodyType != "" || len(it.ReqHeaders) != 0 {
		t.Errorf("an endpoint without bodies has no body types or content type: %#v", it)
	}
	// Lists are sent as [] rather than null.
	data, _ := json.Marshal(it)
	for _, key := range []string{`"req_query":[]`, `"req_params":[]`, `"req_headers":[]`, `"req_body_form":[]`} {
		if !strings.Contains(string(data), key) {
			t.Errorf("payload lacks %s: %s", key, data)
		}
	}

	form := buildInterface(model.ApiEndpoint{Path: "/upload", Method: "POST", Parameters: []model.ApiParameter{
		{Name: "file", In: "form", Type: "file", Required: true},
		{Name: "note", In: "form"},
	}}, &schemaBuilder{})
	if form.Title != "POST /upload" || form.ReqBodyType != "form" {
		t.Errorf("form endpoint = %q %q", form.Title, form.ReqBodyType)
	}
	if form.ReqBodyForm[0] != (FormField{Name: "file", Type: "file", Required: "1"}) || form.ReqBodyForm[1].Type != "text" {
		t.Errorf("req_body_form = %#v", form.ReqBodyForm)
	}
	if ct := form.ReqHeaders[len(form.ReqHeaders)-1]; ct.Name != "Content-Type" || ct.Value != "multipart/form-data" {
		t.Errorf("content type = %#v", ct)
	}
}

// A deprecated endpoint is tagged "deprecated" and its description ends with
// 「已废弃」 and the deprecation note, as in an EasyYapi export.
func TestBuildInterfaceDeprecated(t *testing.T) {
	for _, tc := range []struct {
		name                  string
		ep                    model.ApiEndpoint
		title, markdown, desc string
	}{
		{
			"note after the description",
			model.ApiEndpoint{Name: "old", Description: "Old orders.\nKept for the app.", Deprecated: true, DeprecationNote: "Use /orders/v2"},
			"Old orders.", "Kept for the app.\n「已废弃」Use /orders/v2", "<body><p>Kept for the app.\n「已废弃」Use /orders/v2</p></body>",
		},
		{
			"a one-line description leaves the line alone in the description",
			model.ApiEndpoint{Name: "old", Description: "Old orders.", Deprecated: true, DeprecationNote: "Use /orders/v2"},
			"Old orders.", "「已废弃」Use /orders/v2", "<body><p>「已废弃」Use /orders/v2</p></body>",
		},
		{
			"without a note, as for Java's @Deprecated",
			model.ApiEndpoint{Name: "old", Description: "Old orders.", Deprecated: true},
			"Old orders.", "「已废弃」", "<body><p>「已废弃」</p></body>",
		},
		{
			"without a description the line is the title",
			model.ApiEndpoint{Name: "old", Deprecated: true, DeprecationNote: "Use /orders/v2"},
			"「已废弃」Use /orders/v2", "", "",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			it := buildInterface(tc.ep, &schemaBuilder{})
			if it.Title != tc.title || it.Markdown != tc.markdown || it.Desc != tc.desc {
				t.Errorf("title %q, markdown %q, desc %q\nwant  %q, %q, %q", it.Title, it.Markdown, it.Desc, tc.title, tc.markdown, tc.desc)
			}
			if !reflect.DeepEqual(it.Tag, []string{"deprecated"}) {
				t.Errorf("tag = %q", it.Tag)
			}
		})
	}

	current := buildInterface(model.ApiEndpoint{Name: "list", Description: "Orders."}, &schemaBuilder{})
	data, _ := json.Marshal(current)
	if current.Tag != nil || strings.Contains(string(data), `"tag"`) {
		t.Errorf("a current endpoint carries no tag, so tags set in YApi survive an upload: %s", data)
	}
}

// Descriptions render as EasyYapi renders them: GitHub-flavoured Markdown
// inside <body>, HTML from a doc comment passed through, and no line break
// between blocks.
func TestRenderHTML(t *testing.T) {
	for _, tc := range []struct{ name, in, want string }{
		{"paragraphs and a wrapped line",
			"A\nB\n\nC",
			"<body><p>A\nB</p><p>C</p></body>"},
		{"emphasis",
			"Read **only** this",
			"<body><p>Read <strong>only</strong> this</p></body>"},
		{"HTML of a Javadoc comment is kept",
			"<p>Same service method.",
			"<body><p>Same service method.</body>"},
		{"a list",
			"Steps:\n\n- one\n- two",
			"<body><p>Steps:</p><ul><li>one</li><li>two</li></ul></body>"},
		{"a code block keeps its lines",
			"```\na\nb\n```",
			"<body><pre><code>a\nb\n</code></pre></body>"},
		{"nothing",
			"  ",
			""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := renderHTML(tc.in); got != tc.want {
				t.Errorf("renderHTML(%q) =\n %q\nwant\n %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestParameterMock(t *testing.T) {
	for name, want := range map[string]string{
		"operatorAccountId": "@id",
		"serialNo":          "@string",
		"createTime":        "@datetime",
		"startDate":         "@datetime",
		"birthDate":         "@date('yyyy-MM-dd')",
		"userName":          "@cname",
		"warehouseName":     "@string",
		"province":          "@province",
		"token":             "@string(32)",
		"modelCode":         "@string(6)",
		"email":             "@email",
		// Substring tests, as EasyYapi's: "page" and "usageStatus" hold "age".
		"page":        "@integer(0, 120)",
		"usageStatus": "@integer(0, 120)",
		"identity":    "@string",
		"rows":        "@string",
	} {
		if got := parameterMock(name); got != want {
			t.Errorf("parameterMock(%q) = %q, want %q", name, got, want)
		}
	}
}

func TestNormalizePath(t *testing.T) {
	for in, want := range map[string]string{
		"orders":            "/orders",
		"/orders//{id}":     "/orders/{id}",
		"/a b/c":            "/a/b/c",
		"/v1/items/{id}:go": "/v1/items/{id}:go",
	} {
		if got := normalizePath(in); got != want {
			t.Errorf("normalizePath(%q) = %q, want %q", in, got, want)
		}
	}
}
