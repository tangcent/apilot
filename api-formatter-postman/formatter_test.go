package postman

import (
	"encoding/json"
	"strings"
	"testing"

	formatter "github.com/tangcent/apilot/api-formatter"
	model "github.com/tangcent/apilot/api-model"
	postmanmodel "github.com/tangcent/apilot/api-formatter-postman/model"
)

func TestFormat_EmptyInput(t *testing.T) {
	f := New()
	endpoints := []model.ApiEndpoint{}
	opts := formatter.FormatOptions{}

	output, err := f.Format(endpoints, opts)
	if err != nil {
		t.Fatalf("Format() error = %v, want nil", err)
	}
	if output == nil {
		t.Fatal("Format() output = nil, want non-nil bytes")
	}

	var col postmanmodel.Collection
	if err := json.Unmarshal(output, &col); err != nil {
		t.Fatalf("Output is not valid JSON: %v", err)
	}
	if col.Info.Schema != postmanSchema {
		t.Errorf("Schema = %q, want %q", col.Info.Schema, postmanSchema)
	}
	if col.Info.Name != "APilot Export" {
		t.Errorf("Name = %q, want %q", col.Info.Name, "APilot Export")
	}
	if len(col.Item) != 0 {
		t.Errorf("Item count = %d, want 0", len(col.Item))
	}
}

func TestFormat_RoundTrip(t *testing.T) {
	f := New()
	endpoints := []model.ApiEndpoint{
		{
			Name:     "Get Users",
			Path:     "/users",
			Method:   "GET",
			Protocol: "http",
			Folder:   "Users",
		},
		{
			Name:     "Create User",
			Path:     "/users",
			Method:   "POST",
			Protocol: "http",
			Folder:   "Users",
			RequestBody: &model.ApiBody{
				MediaType: "application/json",
				Example:   map[string]interface{}{"name": "Alice"},
			},
		},
	}
	params := Params{CollectionName: "Test Collection", BaseURL: "https://api.example.com"}
	paramsJSON, _ := json.Marshal(params)
	opts := formatter.FormatOptions{Params: paramsJSON}

	output1, err := f.Format(endpoints, opts)
	if err != nil {
		t.Fatalf("First Format() error = %v", err)
	}

	var col postmanmodel.Collection
	if err := json.Unmarshal(output1, &col); err != nil {
		t.Fatalf("Unmarshal error = %v", err)
	}

	output2, err := json.MarshalIndent(col, "", "  ")
	if err != nil {
		t.Fatalf("Re-marshal error = %v", err)
	}

	var v1, v2 interface{}
	json.Unmarshal(output1, &v1)
	json.Unmarshal(output2, &v2)

	b1, _ := json.Marshal(v1)
	b2, _ := json.Marshal(v2)
	if string(b1) != string(b2) {
		t.Errorf("Round-trip mismatch:\nfirst:  %s\nsecond: %s", string(b1), string(b2))
	}
}

func TestFormat_PathParams(t *testing.T) {
	f := New()
	endpoints := []model.ApiEndpoint{
		{
			Name:     "Get User",
			Path:     "/users/{id}",
			Method:   "GET",
			Protocol: "http",
			Parameters: []model.ApiParameter{
				{Name: "id", In: "path", Type: "text", Required: true, Example: "123", Description: "User ID"},
			},
		},
	}
	opts := formatter.FormatOptions{}

	output, err := f.Format(endpoints, opts)
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	var col postmanmodel.Collection
	if err := json.Unmarshal(output, &col); err != nil {
		t.Fatalf("Unmarshal error = %v", err)
	}

	if len(col.Item) == 0 {
		t.Fatal("Expected at least one folder in collection")
	}
	folder := col.Item[0]
	if len(folder.Item) == 0 {
		t.Fatal("Expected at least one item in folder")
	}
	item := folder.Item[0]

	expectedPath := []string{"users", ":id"}
	if len(item.Request.URL.Path) != len(expectedPath) {
		t.Fatalf("URL.Path = %v, want %v", item.Request.URL.Path, expectedPath)
	}
	for i, seg := range item.Request.URL.Path {
		if seg != expectedPath[i] {
			t.Errorf("URL.Path[%d] = %q, want %q", i, seg, expectedPath[i])
		}
	}

	if len(item.Request.URL.Variable) != 1 {
		t.Fatalf("Expected 1 path variable, got %d", len(item.Request.URL.Variable))
	}
	pv := item.Request.URL.Variable[0]
	if pv.Key != "id" {
		t.Errorf("PathVariable.Key = %q, want %q", pv.Key, "id")
	}
	if pv.Value != "123" {
		t.Errorf("PathVariable.Value = %q, want %q", pv.Value, "123")
	}
	if pv.Description != "User ID" {
		t.Errorf("PathVariable.Description = %q, want %q", pv.Description, "User ID")
	}
}

func TestFormat_QueryParams(t *testing.T) {
	f := New()
	endpoints := []model.ApiEndpoint{
		{
			Name:     "List Users",
			Path:     "/users",
			Method:   "GET",
			Protocol: "http",
			Parameters: []model.ApiParameter{
				{Name: "page", In: "query", Type: "text", Example: "1", Description: "Page number"},
				{Name: "size", In: "query", Type: "text", Default: "20"},
			},
		},
	}
	opts := formatter.FormatOptions{}

	output, err := f.Format(endpoints, opts)
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	var col postmanmodel.Collection
	if err := json.Unmarshal(output, &col); err != nil {
		t.Fatalf("Unmarshal error = %v", err)
	}

	item := col.Item[0].Item[0]
	if len(item.Request.URL.Query) != 2 {
		t.Fatalf("Expected 2 query params, got %d", len(item.Request.URL.Query))
	}

	q1 := item.Request.URL.Query[0]
	if q1.Key != "page" {
		t.Errorf("Query[0].Key = %q, want %q", q1.Key, "page")
	}
	if q1.Value != "1" {
		t.Errorf("Query[0].Value = %q, want %q", q1.Value, "1")
	}
	if q1.Description != "Page number" {
		t.Errorf("Query[0].Description = %q, want %q", q1.Description, "Page number")
	}

	q2 := item.Request.URL.Query[1]
	if q2.Key != "size" {
		t.Errorf("Query[1].Key = %q, want %q", q2.Key, "size")
	}
	if q2.Value != "20" {
		t.Errorf("Query[1].Value = %q, want %q", q2.Value, "20")
	}
}

func TestFormat_ResponseExample(t *testing.T) {
	f := New()
	endpoints := []model.ApiEndpoint{
		{
			Name:     "Get User",
			Path:     "/users/123",
			Method:   "GET",
			Protocol: "http",
			Response: &model.ApiBody{
				MediaType: "application/json",
				Example: map[string]interface{}{
					"id":   123,
					"name": "Alice",
				},
			},
		},
	}
	opts := formatter.FormatOptions{}

	output, err := f.Format(endpoints, opts)
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	var col postmanmodel.Collection
	if err := json.Unmarshal(output, &col); err != nil {
		t.Fatalf("Unmarshal error = %v", err)
	}

	folder := col.Item[0]
	item := folder.Item[0]

	if len(item.Response) == 0 {
		t.Fatal("Expected response examples to be populated")
	}

	resp := item.Response[0]
	if resp.Name != "Example response" {
		t.Errorf("Response.Name = %q, want %q", resp.Name, "Example response")
	}
	if resp.Status != "OK" {
		t.Errorf("Response.Status = %q, want %q", resp.Status, "OK")
	}
	if resp.Code != 200 {
		t.Errorf("Response.Code = %d, want 200", resp.Code)
	}
	if !strings.Contains(resp.Body, `"id"`) || !strings.Contains(resp.Body, `"name"`) {
		t.Errorf("Response.Body = %q, want JSON containing id and name fields", resp.Body)
	}
	if resp.PostmanPreviewLanguage != "json" {
		t.Errorf("Response._postman_previewlanguage = %q, want %q", resp.PostmanPreviewLanguage, "json")
	}
	if len(resp.Header) == 0 {
		t.Error("Expected response headers to be populated")
	}
}

func TestFormat_NoResponseWhenNil(t *testing.T) {
	f := New()
	endpoints := []model.ApiEndpoint{
		{
			Name:     "Delete User",
			Path:     "/users/{id}",
			Method:   "DELETE",
			Protocol: "http",
		},
	}
	opts := formatter.FormatOptions{}

	output, err := f.Format(endpoints, opts)
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	var col postmanmodel.Collection
	if err := json.Unmarshal(output, &col); err != nil {
		t.Fatalf("Unmarshal error = %v", err)
	}

	folder := col.Item[0]
	item := folder.Item[0]

	if len(item.Response) != 0 {
		t.Errorf("Expected no response examples, got %d", len(item.Response))
	}
}

func TestFormat_RequestBodyFromObjectModel(t *testing.T) {
	f := New()
	userModel := model.NewObjectModelBuilder().
		StringField("name", model.WithDemo("Alice")).
		IntField("age", model.WithDemo("30")).
		Build()

	endpoints := []model.ApiEndpoint{
		{
			Name:     "Create User",
			Path:     "/users",
			Method:   "POST",
			Protocol: "http",
			RequestBody: &model.ApiBody{
				MediaType: "application/json",
				Body:      userModel,
			},
		},
	}
	opts := formatter.FormatOptions{}

	output, err := f.Format(endpoints, opts)
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	var col postmanmodel.Collection
	if err := json.Unmarshal(output, &col); err != nil {
		t.Fatalf("Unmarshal error = %v", err)
	}

	item := col.Item[0].Item[0]
	if item.Request.Body == nil {
		t.Fatal("Expected request body to be populated")
	}
	if item.Request.Body.Mode != "raw" {
		t.Errorf("Body.Mode = %q, want %q", item.Request.Body.Mode, "raw")
	}
	if !strings.Contains(item.Request.Body.Raw, `"name"`) || !strings.Contains(item.Request.Body.Raw, `"age"`) {
		t.Errorf("Body.Raw = %q, want JSON containing name and age fields", item.Request.Body.Raw)
	}
	if !strings.Contains(item.Request.Body.Raw, `"Alice"`) {
		t.Errorf("Body.Raw = %q, want JSON containing demo value Alice", item.Request.Body.Raw)
	}
}

func TestFormat_ResponseBodyFromObjectModel(t *testing.T) {
	f := New()
	resultModel := model.NewObjectModelBuilder().
		IntField("code").
		StringField("message").
		ObjectField("data", model.NewObjectModelBuilder().
			LongField("id", model.WithDemo("1")).
			StringField("name", model.WithDemo("Alice")).
			Build()).
		Build()

	endpoints := []model.ApiEndpoint{
		{
			Name:     "Get User",
			Path:     "/users/1",
			Method:   "GET",
			Protocol: "http",
			Response: &model.ApiBody{
				MediaType: "application/json",
				Body:      resultModel,
			},
		},
	}
	opts := formatter.FormatOptions{}

	output, err := f.Format(endpoints, opts)
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	var col postmanmodel.Collection
	if err := json.Unmarshal(output, &col); err != nil {
		t.Fatalf("Unmarshal error = %v", err)
	}

	item := col.Item[0].Item[0]
	if len(item.Response) == 0 {
		t.Fatal("Expected response to be populated")
	}
	resp := item.Response[0]
	if !strings.Contains(resp.Body, `"code"`) {
		t.Errorf("Response.Body = %q, want JSON containing code field", resp.Body)
	}
	if !strings.Contains(resp.Body, `"data"`) {
		t.Errorf("Response.Body = %q, want JSON containing data field", resp.Body)
	}
	if !strings.Contains(resp.Body, `"Alice"`) {
		t.Errorf("Response.Body = %q, want JSON containing demo value Alice", resp.Body)
	}
}

func TestFormat_FormUrlencoded(t *testing.T) {
	f := New()
	endpoints := []model.ApiEndpoint{
		{
			Name:     "Login",
			Path:     "/login",
			Method:   "POST",
			Protocol: "http",
			RequestBody: &model.ApiBody{
				MediaType: "application/x-www-form-urlencoded",
			},
			Parameters: []model.ApiParameter{
				{Name: "username", In: "form", Type: "text", Example: "admin"},
				{Name: "password", In: "form", Type: "text", Example: "secret"},
			},
		},
	}
	opts := formatter.FormatOptions{}

	output, err := f.Format(endpoints, opts)
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	var col postmanmodel.Collection
	if err := json.Unmarshal(output, &col); err != nil {
		t.Fatalf("Unmarshal error = %v", err)
	}

	item := col.Item[0].Item[0]
	if item.Request.Body == nil {
		t.Fatal("Expected request body")
	}
	if item.Request.Body.Mode != "urlencoded" {
		t.Errorf("Body.Mode = %q, want %q", item.Request.Body.Mode, "urlencoded")
	}
	if len(item.Request.Body.Urlencoded) != 2 {
		t.Fatalf("Expected 2 urlencoded params, got %d", len(item.Request.Body.Urlencoded))
	}
	if item.Request.Body.Urlencoded[0].Key != "username" {
		t.Errorf("Param[0].Key = %q, want %q", item.Request.Body.Urlencoded[0].Key, "username")
	}
	if item.Request.Body.Urlencoded[0].Value != "admin" {
		t.Errorf("Param[0].Value = %q, want %q", item.Request.Body.Urlencoded[0].Value, "admin")
	}
}

func TestFormat_MultipartFormData(t *testing.T) {
	f := New()
	endpoints := []model.ApiEndpoint{
		{
			Name:     "Upload File",
			Path:     "/upload",
			Method:   "POST",
			Protocol: "http",
			RequestBody: &model.ApiBody{
				MediaType: "multipart/form-data",
			},
			Parameters: []model.ApiParameter{
				{Name: "file", In: "form", Type: "file"},
				{Name: "description", In: "form", Type: "text", Example: "A file"},
			},
		},
	}
	opts := formatter.FormatOptions{}

	output, err := f.Format(endpoints, opts)
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	var col postmanmodel.Collection
	if err := json.Unmarshal(output, &col); err != nil {
		t.Fatalf("Unmarshal error = %v", err)
	}

	item := col.Item[0].Item[0]
	if item.Request.Body == nil {
		t.Fatal("Expected request body")
	}
	if item.Request.Body.Mode != "formdata" {
		t.Errorf("Body.Mode = %q, want %q", item.Request.Body.Mode, "formdata")
	}
	if len(item.Request.Body.Formdata) != 2 {
		t.Fatalf("Expected 2 formdata params, got %d", len(item.Request.Body.Formdata))
	}
	if item.Request.Body.Formdata[0].Type != "file" {
		t.Errorf("Param[0].Type = %q, want %q", item.Request.Body.Formdata[0].Type, "file")
	}
	if item.Request.Body.Formdata[1].Type != "text" {
		t.Errorf("Param[1].Type = %q, want %q", item.Request.Body.Formdata[1].Type, "text")
	}
}

func TestFormat_Description(t *testing.T) {
	f := New()
	endpoints := []model.ApiEndpoint{
		{
			Name:        "Get User",
			Path:        "/users/{id}",
			Method:      "GET",
			Protocol:    "http",
			Description: "Retrieves a user by their unique identifier",
		},
	}
	opts := formatter.FormatOptions{}

	output, err := f.Format(endpoints, opts)
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	var col postmanmodel.Collection
	if err := json.Unmarshal(output, &col); err != nil {
		t.Fatalf("Unmarshal error = %v", err)
	}

	item := col.Item[0].Item[0]
	if item.Request.Description != "Retrieves a user by their unique identifier" {
		t.Errorf("Request.Description = %q, want description", item.Request.Description)
	}
	if item.Description != "Retrieves a user by their unique identifier" {
		t.Errorf("Item.Description = %q, want description", item.Description)
	}
}

func TestFormat_HeaderDescription(t *testing.T) {
	f := New()
	endpoints := []model.ApiEndpoint{
		{
			Name:     "Get User",
			Path:     "/users",
			Method:   "GET",
			Protocol: "http",
			Headers: []model.ApiHeader{
				{Name: "Authorization", Value: "Bearer token", Description: "Auth header"},
			},
		},
	}
	opts := formatter.FormatOptions{}

	output, err := f.Format(endpoints, opts)
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	var col postmanmodel.Collection
	if err := json.Unmarshal(output, &col); err != nil {
		t.Fatalf("Unmarshal error = %v", err)
	}

	item := col.Item[0].Item[0]
	if len(item.Request.Header) != 1 {
		t.Fatalf("Expected 1 header, got %d", len(item.Request.Header))
	}
	h := item.Request.Header[0]
	if h.Type != "text" {
		t.Errorf("Header.Type = %q, want %q", h.Type, "text")
	}
	if h.Description != "Auth header" {
		t.Errorf("Header.Description = %q, want %q", h.Description, "Auth header")
	}
}

func TestFormat_ResponseHeaders(t *testing.T) {
	f := New()
	endpoints := []model.ApiEndpoint{
		{
			Name:     "Get User",
			Path:     "/users/1",
			Method:   "GET",
			Protocol: "http",
			Response: &model.ApiBody{
				MediaType: "application/json",
				Example:   map[string]interface{}{"id": 1},
			},
		},
	}
	opts := formatter.FormatOptions{}

	output, err := f.Format(endpoints, opts)
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	var col postmanmodel.Collection
	if err := json.Unmarshal(output, &col); err != nil {
		t.Fatalf("Unmarshal error = %v", err)
	}

	item := col.Item[0].Item[0]
	if len(item.Response) == 0 {
		t.Fatal("Expected response")
	}
	resp := item.Response[0]

	hasContentType := false
	hasServer := false
	for _, h := range resp.Header {
		if h.Key == "content-type" {
			hasContentType = true
		}
		if h.Key == "server" {
			hasServer = true
		}
	}
	if !hasContentType {
		t.Error("Expected content-type header in response")
	}
	if !hasServer {
		t.Error("Expected server header in response")
	}
}

func TestRequiredSettings(t *testing.T) {
	f := New()
	sp, ok := f.(formatter.SettingsProvider)
	if !ok {
		t.Fatal("PostmanFormatter should implement SettingsProvider")
	}

	settings := sp.RequiredSettings()
	if len(settings) != 2 {
		t.Fatalf("Expected 2 settings, got %d", len(settings))
	}

	expectedKeys := []string{"postman.api.key", "postman.export.mode"}
	for i, expected := range expectedKeys {
		if settings[i].Key != expected {
			t.Errorf("settings[%d].Key = %q, want %q", i, settings[i].Key, expected)
		}
	}

	for _, s := range settings {
		if s.Required {
			t.Errorf("setting %q should not be required (formatter works in file mode)", s.Key)
		}
	}
}

func TestResolveAPIKey_FromSettings(t *testing.T) {
	p := Params{PostmanAPIKey: "from-params"}
	opts := formatter.FormatOptions{
		Settings: &mapSettings{"postman.api.key": "from-settings"},
	}

	key := resolveAPIKey(p, opts)
	if key != "from-settings" {
		t.Errorf("Expected 'from-settings', got %q", key)
	}
}

func TestResolveAPIKey_FromParams(t *testing.T) {
	p := Params{PostmanAPIKey: "from-params"}
	opts := formatter.FormatOptions{}

	key := resolveAPIKey(p, opts)
	if key != "from-params" {
		t.Errorf("Expected 'from-params', got %q", key)
	}
}

func TestResolveAPIKey_SettingsTakesPrecedence(t *testing.T) {
	p := Params{PostmanAPIKey: "from-params"}
	opts := formatter.FormatOptions{
		Settings: &mapSettings{"postman.api.key": "from-settings"},
	}

	key := resolveAPIKey(p, opts)
	if key != "from-settings" {
		t.Errorf("Settings should take precedence over params, got %q", key)
	}
}

func TestResolveAPIKey_EmptySettings(t *testing.T) {
	p := Params{PostmanAPIKey: "from-params"}
	opts := formatter.FormatOptions{
		Settings: &mapSettings{"postman.api.key": ""},
	}

	key := resolveAPIKey(p, opts)
	if key != "from-params" {
		t.Errorf("Empty settings value should fall back to params, got %q", key)
	}
}

func TestResolveAPIKey_None(t *testing.T) {
	p := Params{}
	opts := formatter.FormatOptions{}

	key := resolveAPIKey(p, opts)
	if key != "" {
		t.Errorf("Expected empty string when no key configured, got %q", key)
	}
}

func TestResolveExportMode_Default(t *testing.T) {
	p := Params{}
	opts := formatter.FormatOptions{}

	mode := resolveExportMode(p, opts)
	if mode != ExportModeCreateNew {
		t.Errorf("Expected %q, got %q", ExportModeCreateNew, mode)
	}
}

func TestResolveExportMode_FromParams(t *testing.T) {
	p := Params{ExportMode: "UPDATE_EXISTING"}
	opts := formatter.FormatOptions{}

	mode := resolveExportMode(p, opts)
	if mode != ExportModeUpdateExisting {
		t.Errorf("Expected %q, got %q", ExportModeUpdateExisting, mode)
	}
}

func TestResolveExportMode_FromSettings(t *testing.T) {
	p := Params{}
	opts := formatter.FormatOptions{
		Settings: &mapSettings{"postman.export.mode": "UPDATE_EXISTING"},
	}

	mode := resolveExportMode(p, opts)
	if mode != ExportModeUpdateExisting {
		t.Errorf("Expected %q, got %q", ExportModeUpdateExisting, mode)
	}
}

func TestResolveExportMode_ParamsTakesPrecedence(t *testing.T) {
	p := Params{ExportMode: "CREATE_NEW"}
	opts := formatter.FormatOptions{
		Settings: &mapSettings{"postman.export.mode": "UPDATE_EXISTING"},
	}

	mode := resolveExportMode(p, opts)
	if mode != ExportModeCreateNew {
		t.Errorf("Params should take precedence over settings, got %q", mode)
	}
}

func TestResolveExportMode_CaseInsensitive(t *testing.T) {
	p := Params{ExportMode: "update_existing"}
	opts := formatter.FormatOptions{}

	mode := resolveExportMode(p, opts)
	if mode != ExportModeUpdateExisting {
		t.Errorf("Expected %q (uppercased), got %q", ExportModeUpdateExisting, mode)
	}
}

func TestFormat_APIMode_NoAPIKey(t *testing.T) {
	f := New()
	endpoints := []model.ApiEndpoint{
		{Name: "Get Users", Path: "/users", Method: "GET"},
	}
	params := Params{Mode: "api"}
	paramsJSON, _ := json.Marshal(params)
	opts := formatter.FormatOptions{Params: paramsJSON}

	_, err := f.Format(endpoints, opts)
	if err == nil {
		t.Fatal("Expected error for api mode without API key, got nil")
	}
	if !strings.Contains(err.Error(), "postman api key is required") {
		t.Errorf("Expected 'postman api key is required' in error, got: %v", err)
	}
}

func TestFormat_FileMode_Default(t *testing.T) {
	f := New()
	endpoints := []model.ApiEndpoint{
		{Name: "Get Users", Path: "/users", Method: "GET"},
	}
	opts := formatter.FormatOptions{}

	output, err := f.Format(endpoints, opts)
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	var col postmanmodel.Collection
	if err := json.Unmarshal(output, &col); err != nil {
		t.Fatalf("Output should be valid collection JSON: %v", err)
	}
}

func TestFormat_NestedObjectModelBody(t *testing.T) {
	f := New()
	addressModel := model.NewObjectModelBuilder().
		StringField("street", model.WithDemo("123 Main St")).
		StringField("city", model.WithDemo("Springfield")).
		Build()
	userModel := model.NewObjectModelBuilder().
		StringField("name", model.WithDemo("Alice")).
		ObjectField("address", addressModel).
		ArrayField("tags", model.SingleModel(model.JsonTypeString)).
		Build()

	endpoints := []model.ApiEndpoint{
		{
			Name:     "Create User",
			Path:     "/users",
			Method:   "POST",
			Protocol: "http",
			RequestBody: &model.ApiBody{
				MediaType: "application/json",
				Body:      userModel,
			},
		},
	}
	opts := formatter.FormatOptions{}

	output, err := f.Format(endpoints, opts)
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	var col postmanmodel.Collection
	if err := json.Unmarshal(output, &col); err != nil {
		t.Fatalf("Unmarshal error = %v", err)
	}

	item := col.Item[0].Item[0]
	body := item.Request.Body.Raw

	if !strings.Contains(body, `"address"`) {
		t.Errorf("Body should contain nested address field: %s", body)
	}
	if !strings.Contains(body, `"street"`) {
		t.Errorf("Body should contain nested street field: %s", body)
	}
	if !strings.Contains(body, `"tags"`) {
		t.Errorf("Body should contain tags array field: %s", body)
	}
}

func TestFormat_ArrayModelBody(t *testing.T) {
	f := New()
	userModel := model.NewObjectModelBuilder().
		StringField("name", model.WithDemo("Alice")).
		Build()
	arrayModel := model.ArrayModel(userModel)

	endpoints := []model.ApiEndpoint{
		{
			Name:     "Batch Create Users",
			Path:     "/users/batch",
			Method:   "POST",
			Protocol: "http",
			RequestBody: &model.ApiBody{
				MediaType: "application/json",
				Body:      arrayModel,
			},
		},
	}
	opts := formatter.FormatOptions{}

	output, err := f.Format(endpoints, opts)
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	var col postmanmodel.Collection
	if err := json.Unmarshal(output, &col); err != nil {
		t.Fatalf("Unmarshal error = %v", err)
	}

	item := col.Item[0].Item[0]
	body := item.Request.Body.Raw

	if !strings.HasPrefix(body, "[") {
		t.Errorf("Array body should start with [: %s", body)
	}
	if !strings.Contains(body, `"name"`) {
		t.Errorf("Array body should contain name field: %s", body)
	}
}

type mapSettings map[string]string

func (s mapSettings) Get(key string) string { return s[key] }

func TestResolveMode_ExplicitMode(t *testing.T) {
	p := Params{Mode: "api"}
	mode := resolveMode(p, "some-key")
	if mode != "api" {
		t.Errorf("Expected 'api', got %q", mode)
	}
}

func TestResolveMode_ExplicitFileMode(t *testing.T) {
	p := Params{Mode: "file"}
	mode := resolveMode(p, "some-key")
	if mode != "file" {
		t.Errorf("Expected 'file', got %q", mode)
	}
}

func TestResolveMode_AutoAPIKey(t *testing.T) {
	p := Params{}
	mode := resolveMode(p, "PMAK-xxxx")
	if mode != "api" {
		t.Errorf("Expected 'api' when API key is available, got %q", mode)
	}
}

func TestResolveMode_AutoNoKey(t *testing.T) {
	p := Params{}
	mode := resolveMode(p, "")
	if mode != "file" {
		t.Errorf("Expected 'file' when no API key, got %q", mode)
	}
}

func TestResolveMode_ExplicitOverridesAuto(t *testing.T) {
	p := Params{Mode: "file"}
	mode := resolveMode(p, "PMAK-xxxx")
	if mode != "file" {
		t.Errorf("Explicit mode should override auto-detection, got %q", mode)
	}
}

func TestResolveMode_OutputPathForcesFile(t *testing.T) {
	p := Params{OutputPath: "collection.json"}
	mode := resolveMode(p, "PMAK-xxxx")
	if mode != "file" {
		t.Errorf("OutputPath should force file mode even with API key, got %q", mode)
	}
}

func TestResolveMode_ExplicitAPIOverridesOutputPath(t *testing.T) {
	p := Params{Mode: "api", OutputPath: "collection.json"}
	mode := resolveMode(p, "PMAK-xxxx")
	if mode != "api" {
		t.Errorf("Explicit api mode should override OutputPath, got %q", mode)
	}
}
