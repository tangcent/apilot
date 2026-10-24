package json

import (
	stdjson "encoding/json"
	"strings"
	"testing"

	formatter "github.com/tangcent/apilot/api-formatter"
	model "github.com/tangcent/apilot/api-model"
)

func decode(t *testing.T, data []byte) document {
	t.Helper()
	var doc document
	if err := stdjson.Unmarshal(data, &doc); err != nil {
		t.Fatalf("output is not valid JSON: %v\n%s", err, data)
	}
	return doc
}

func TestName(t *testing.T) {
	if got := New().Name(); got != "json" {
		t.Fatalf("Name() = %q, want %q", got, "json")
	}
}

func TestFormat_EmptyInput(t *testing.T) {
	out, err := New().Format(nil, formatter.FormatOptions{})
	if err != nil {
		t.Fatalf("Format() error = %v, want nil", err)
	}
	if len(out) == 0 {
		t.Fatal("empty input should produce valid empty output, not empty bytes")
	}

	doc := decode(t, out)
	if doc.Endpoints == nil {
		t.Fatal("endpoints should be an empty array, not null")
	}
	if len(doc.Endpoints) != 0 {
		t.Fatalf("endpoints length = %d, want 0", len(doc.Endpoints))
	}
	if !strings.Contains(string(out), `"endpoints": []`) {
		t.Errorf("empty output should hold an empty endpoints array, got %s", out)
	}
}

// TestFormat_DocumentedShape pins the exact document a consumer can gate on:
// key names, key order and nesting are part of the contract.
func TestFormat_DocumentedShape(t *testing.T) {
	endpoints := []model.ApiEndpoint{
		{
			Name:        "getUser",
			Path:        "/users/:id",
			Method:      "GET",
			Protocol:    "http",
			Description: "getUser returns a single user.",
			Parameters: []model.ApiParameter{
				{Name: "id", In: "path", Type: "text", Required: true, Description: "user id"},
			},
			Response: &model.ApiBody{
				MediaType: "application/json",
				Body: model.ObjectModelFrom(map[string]*model.FieldModel{
					"id": {Model: model.SingleModel(model.JsonTypeLong), Required: true},
				}),
			},
		},
	}

	out, err := New().Format(endpoints, formatter.FormatOptions{})
	if err != nil {
		t.Fatalf("Format() error = %v, want nil", err)
	}

	want := `{
  "endpoints": [
    {
      "name": "getUser",
      "method": "GET",
      "path": "/users/:id",
      "protocol": "http",
      "description": "getUser returns a single user.",
      "request": {
        "parameters": {
          "path": [
            {
              "name": "id",
              "type": "text",
              "required": true,
              "description": "user id"
            }
          ]
        }
      },
      "response": {
        "mediaType": "application/json",
        "schema": {
          "kind": "object",
          "fields": [
            {
              "name": "id",
              "kind": "single",
              "type": "long",
              "required": true
            }
          ]
        }
      }
    }
  ]
}`
	if string(out) != want {
		t.Errorf("document shape changed.\n--- got ---\n%s\n--- want ---\n%s", out, want)
	}
}

func TestFormat_ParametersGroupedByLocation(t *testing.T) {
	endpoints := []model.ApiEndpoint{
		{
			Name: "upload", Path: "/files", Method: "POST", Protocol: "http",
			Parameters: []model.ApiParameter{
				{Name: "id", In: "path", Type: "text", Required: true},
				{Name: "page", In: "query", Type: "text", Default: "1"},
				{Name: "X-Request-Id", In: "header", Type: "text"},
				{Name: "session", In: "cookie", Type: "text"},
				{Name: "file", In: "form", Type: "file", Required: true},
				{Name: "raw", In: "body", Type: "text"},
			},
			Headers: []model.ApiHeader{
				{Name: "X-Trace", Value: "on", Required: true, Description: "trace flag"},
			},
		},
	}

	doc := decode(t, mustFormat(t, endpoints))
	params := doc.Endpoints[0].Request.Parameters
	if params == nil {
		t.Fatal("parameters missing from output")
	}

	for name, got := range map[string]int{
		"path":   len(params.Path),
		"query":  len(params.Query),
		"cookie": len(params.Cookie),
		"form":   len(params.Form),
		"body":   len(params.Body),
	} {
		if got != 1 {
			t.Errorf("%s group length = %d, want 1", name, got)
		}
	}

	// Header parameters and endpoint headers share one group so a header diff
	// has a single place to look.
	if len(params.Header) != 2 {
		t.Fatalf("header group length = %d, want 2", len(params.Header))
	}
	if params.Header[0].Name != "X-Request-Id" {
		t.Errorf("header[0].name = %q, want %q", params.Header[0].Name, "X-Request-Id")
	}
	if params.Header[1].Name != "X-Trace" || params.Header[1].Value != "on" || !params.Header[1].Required {
		t.Errorf("header[1] = %+v, want the declared X-Trace header", params.Header[1])
	}

	if params.Query[0].Default != "1" {
		t.Errorf("query[0].default = %q, want %q", params.Query[0].Default, "1")
	}
	if params.Form[0].Type != "file" || !params.Form[0].Required {
		t.Errorf("form[0] = %+v, want a required file parameter", params.Form[0])
	}
}

func TestFormat_OmitsUnknownLocation(t *testing.T) {
	endpoints := []model.ApiEndpoint{
		{
			Name: "weird", Path: "/weird", Method: "GET", Protocol: "http",
			Parameters: []model.ApiParameter{
				{Name: "dropped", In: "unknown", Type: "text"},
			},
		},
	}

	doc := decode(t, mustFormat(t, endpoints))
	if doc.Endpoints[0].Request != nil {
		t.Errorf("request should be omitted when no parameter location is known, got %+v", doc.Endpoints[0].Request)
	}
}

func TestFormat_SchemaTree(t *testing.T) {
	address := model.ObjectModelFrom(map[string]*model.FieldModel{
		"city": {Model: model.SingleModel(model.JsonTypeString), Required: true, Comment: "city name"},
	})
	request := model.ObjectModelFrom(map[string]*model.FieldModel{
		"tags": {
			Model:    model.ArrayModel(model.SingleModel(model.JsonTypeString)),
			Required: true,
		},
		"address": {
			Model: address,
			Options: []model.FieldOption{
				{Value: "home", Desc: "residence"},
				{Value: "work"},
			},
		},
		"meta": {
			Model: model.MapModel(
				model.SingleModel(model.JsonTypeString),
				model.SingleModel(model.JsonTypeString),
			),
			Demo: "{}",
		},
		"owner":   {Model: model.RefModel("User"), Generic: true},
		"retries": {Model: model.SingleModel(model.JsonTypeInt), DefaultValue: "3"},
	})

	endpoints := []model.ApiEndpoint{
		{
			Name: "createOrder", Path: "/orders", Method: "POST", Protocol: "http",
			RequestBody: &model.ApiBody{MediaType: "application/json", Body: request},
		},
	}

	doc := decode(t, mustFormat(t, endpoints))
	schema := doc.Endpoints[0].Request.Body.Schema

	// Fields are sorted by name: the model stores them in a map, so this is the
	// only order that is stable across runs and machines.
	var names []string
	for _, f := range schema.Fields {
		names = append(names, f.Name)
	}
	wantOrder := []string{"address", "meta", "owner", "retries", "tags"}
	if strings.Join(names, ",") != strings.Join(wantOrder, ",") {
		t.Fatalf("field order = %v, want %v", names, wantOrder)
	}

	byName := make(map[string]*node, len(schema.Fields))
	for _, f := range schema.Fields {
		byName[f.Name] = f
	}

	tags := byName["tags"]
	if tags.Kind != "array" || !tags.Required || tags.Items == nil || tags.Items.Type != model.JsonTypeString {
		t.Errorf("tags = %+v, want a required array of string", tags)
	}

	addressNode := byName["address"]
	if addressNode.Kind != "object" || len(addressNode.Fields) != 1 {
		t.Fatalf("address = %+v, want an object with one field", addressNode)
	}
	if city := addressNode.Fields[0]; city.Name != "city" || !city.Required || city.Description != "city name" {
		t.Errorf("address.city = %+v, want a required string documented as city name", city)
	}
	if len(addressNode.Options) != 2 || addressNode.Options[0].Value != "home" || addressNode.Options[0].Desc != "residence" {
		t.Errorf("address.options = %+v, want the declared options", addressNode.Options)
	}

	meta := byName["meta"]
	if meta.Kind != "map" || meta.Key == nil || meta.Value == nil || meta.Demo != "{}" {
		t.Errorf("meta = %+v, want a map with key and value nodes", meta)
	}

	owner := byName["owner"]
	if owner.Kind != "ref" || owner.Type != "User" || !owner.Generic {
		t.Errorf("owner = %+v, want a generic ref to User", owner)
	}

	if retries := byName["retries"]; retries.Default != "3" || retries.Type != model.JsonTypeInt {
		t.Errorf("retries = %+v, want an int with default 3", retries)
	}
}

// TestFormat_RecursiveModelTerminates guards the export against a type that
// refers to itself: the tree must end as a ref node instead of expanding until
// the stack gives out.
func TestFormat_RecursiveModelTerminates(t *testing.T) {
	treeNode := model.ObjectModelFrom(map[string]*model.FieldModel{})
	treeNode.Fields["name"] = &model.FieldModel{Model: model.SingleModel(model.JsonTypeString)}
	treeNode.Fields["children"] = &model.FieldModel{Model: model.ArrayModel(treeNode)}

	endpoints := []model.ApiEndpoint{
		{
			Name: "tree", Path: "/tree", Method: "GET", Protocol: "http",
			Response: &model.ApiBody{Body: treeNode},
		},
	}

	out, err := New().Format(endpoints, formatter.FormatOptions{})
	if err != nil {
		t.Fatalf("Format() error = %v, want nil", err)
	}

	doc := decode(t, out)
	root := doc.Endpoints[0].Response.Schema
	if root == nil || len(root.Fields) != 2 {
		t.Fatalf("root schema = %+v, want two fields", root)
	}
	var children *node
	for _, f := range root.Fields {
		if f.Name == "children" {
			children = f
		}
	}
	if children == nil || children.Kind != "array" || children.Items == nil || children.Items.Kind != "ref" {
		t.Errorf("children = %+v, want an array terminated by a ref", children)
	}
}

func TestFormat_EndpointOrderIsStable(t *testing.T) {
	endpoints := []model.ApiEndpoint{
		{Name: "zeta", Path: "/z", Method: "GET", Protocol: "http"},
		{Name: "alpha", Path: "/a", Method: "GET", Protocol: "http", Folder: "second"},
		{Name: "alpha", Path: "/a", Method: "DELETE", Protocol: "http", Folder: "first"},
		{Name: "alpha", Path: "/a", Method: "GET", Protocol: "http", Folder: "first"},
	}

	doc := decode(t, mustFormat(t, endpoints))

	var got []string
	for _, ep := range doc.Endpoints {
		got = append(got, ep.Folder+"/"+ep.Name+"/"+ep.Method)
	}
	want := []string{"first/alpha/DELETE", "first/alpha/GET", "second/alpha/GET", "/zeta/GET"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("endpoint order = %v, want %v", got, want)
	}
}

// TestFormat_ByteIdenticalAcrossRuns covers the stability requirement: the same
// input must produce the same bytes even when the members were inserted into the
// model in a different order, as a different collector implementation might.
func TestFormat_ByteIdenticalAcrossRuns(t *testing.T) {
	build := func(order ...string) []model.ApiEndpoint {
		fields := make(map[string]*model.FieldModel, len(order))
		for _, name := range order {
			fields[name] = &model.FieldModel{Model: model.SingleModel(model.JsonTypeString)}
		}
		return []model.ApiEndpoint{
			{
				Name: "getUser", Path: "/users/:id", Method: "GET", Protocol: "http",
				Response: &model.ApiBody{Body: model.ObjectModelFrom(fields)},
			},
		}
	}

	first := mustFormat(t, build("id", "name", "age"))
	second := mustFormat(t, build("age", "id", "name"))
	if string(first) != string(second) {
		t.Errorf("insertion order changed the output:\n%s\n---\n%s", first, second)
	}
}

func mustFormat(t *testing.T, endpoints []model.ApiEndpoint) []byte {
	t.Helper()
	out, err := New().Format(endpoints, formatter.FormatOptions{})
	if err != nil {
		t.Fatalf("Format() error = %v, want nil", err)
	}
	return out
}
