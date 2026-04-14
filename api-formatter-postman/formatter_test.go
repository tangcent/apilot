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

func TestRequiredSettings(t *testing.T) {
	f := New()
	sp, ok := f.(formatter.SettingsProvider)
	if !ok {
		t.Fatal("PostmanFormatter should implement SettingsProvider")
	}

	settings := sp.RequiredSettings()
	if len(settings) != 1 {
		t.Fatalf("Expected 1 setting, got %d", len(settings))
	}
	if settings[0].Key != "postman.api.key" {
		t.Errorf("Expected key 'postman.api.key', got %q", settings[0].Key)
	}
	if settings[0].Required {
		t.Error("postman.api.key should not be required (formatter works in file mode)")
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

type mapSettings map[string]string

func (s mapSettings) Get(key string) string { return s[key] }
