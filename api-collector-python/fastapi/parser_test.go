package fastapi

import (
	"path/filepath"
	"sort"
	"testing"

	collector "github.com/tangcent/apilot/api-collector"
)

func TestParse_EmptyDir(t *testing.T) {
	endpoints, err := Parse(filepath.Join("testdata", "empty"))
	if err != nil {
		t.Fatalf("empty dir should not error: %v", err)
	}
	if endpoints != nil {
		t.Fatalf("expected nil endpoints for empty dir, got %d", len(endpoints))
	}
}

func TestParse_NoRoutes(t *testing.T) {
	endpoints, err := Parse(filepath.Join("testdata", "noroutes"))
	if err != nil {
		t.Fatalf("no routes should not error: %v", err)
	}
	if endpoints != nil {
		t.Fatalf("expected nil endpoints for no routes, got %d", len(endpoints))
	}
}

func TestParse_NonexistentDir(t *testing.T) {
	endpoints, err := Parse(filepath.Join("testdata", "nonexistent"))
	if err != nil {
		t.Fatalf("nonexistent dir should not error: %v", err)
	}
	if endpoints != nil {
		t.Fatalf("expected nil endpoints for nonexistent dir, got %d", len(endpoints))
	}
}

func TestParse_BasicRoutes(t *testing.T) {
	endpoints, err := Parse(filepath.Join("testdata", "basic"))
	if err != nil {
		t.Fatalf("basic routes should not error: %v", err)
	}
	if len(endpoints) != 9 {
		t.Fatalf("expected 9 endpoints, got %d", len(endpoints))
	}

	sort.Slice(endpoints, func(i, j int) bool {
		if endpoints[i].Method != endpoints[j].Method {
			return endpoints[i].Method < endpoints[j].Method
		}
		return endpoints[i].Path < endpoints[j].Path
	})

	assertEndpoint(t, endpoints[0], collector.ApiEndpoint{
		Name: "delete_user", Path: "/users/{id}", Method: "DELETE", Protocol: "http",
		Description: "deleteUser removes a user by ID.",
		Parameters: []collector.ApiParameter{
			{Name: "id", In: "path", Required: true, Type: "text"},
		},
	})

	assertEndpoint(t, endpoints[1], collector.ApiEndpoint{
		Name: "list_users", Path: "/users", Method: "GET", Protocol: "http",
		Description: "listUsers returns all users.",
		Parameters: []collector.ApiParameter{
			{Name: "name", In: "query", Required: true, Type: "text"},
			{Name: "role", In: "query", Required: false, Type: "text"},
		},
	})

	assertEndpoint(t, endpoints[2], collector.ApiEndpoint{
		Name: "get_user", Path: "/users/{id}", Method: "GET", Protocol: "http",
		Description: "getUser returns a single user by ID.",
		Parameters: []collector.ApiParameter{
			{Name: "id", In: "path", Required: true, Type: "text"},
		},
	})

	assertEndpoint(t, endpoints[3], collector.ApiEndpoint{
		Name: "health_check", Path: "/health", Method: "HEAD", Protocol: "http",
		Description: "healthCheck returns service health status.",
	})

	assertEndpoint(t, endpoints[4], collector.ApiEndpoint{
		Name: "user_options", Path: "/users", Method: "OPTIONS", Protocol: "http",
		Description: "userOptions returns allowed methods for /users.",
	})

	assertEndpoint(t, endpoints[5], collector.ApiEndpoint{
		Name: "patch_user", Path: "/users/{id}", Method: "PATCH", Protocol: "http",
		Description: "patchUser partially updates a user.",
		Parameters: []collector.ApiParameter{
			{Name: "id", In: "path", Required: true, Type: "text"},
		},
	})

	assertEndpoint(t, endpoints[6], collector.ApiEndpoint{
		Name: "upload_file", Path: "/upload", Method: "POST", Protocol: "http",
		Description: "uploadFile handles file uploads.",
		Parameters: []collector.ApiParameter{
			{Name: "file", In: "form", Required: true, Type: "file"},
			{Name: "description", In: "form", Required: true, Type: "text"},
		},
	})

	assertEndpoint(t, endpoints[7], collector.ApiEndpoint{
		Name: "create_user", Path: "/users", Method: "POST", Protocol: "http",
		Description: "createUser creates a new user.",
	})

	assertEndpoint(t, endpoints[8], collector.ApiEndpoint{
		Name: "update_user", Path: "/users/{id}", Method: "PUT", Protocol: "http",
		Description: "updateUser updates an existing user.",
		Parameters: []collector.ApiParameter{
			{Name: "id", In: "path", Required: true, Type: "text"},
		},
	})
}

func TestParse_RouterRoutes(t *testing.T) {
	endpoints, err := Parse(filepath.Join("testdata", "router"))
	if err != nil {
		t.Fatalf("router routes should not error: %v", err)
	}
	if len(endpoints) != 5 {
		t.Fatalf("expected 5 endpoints, got %d", len(endpoints))
	}

	sort.Slice(endpoints, func(i, j int) bool {
		if endpoints[i].Path != endpoints[j].Path {
			return endpoints[i].Path < endpoints[j].Path
		}
		return endpoints[i].Method < endpoints[j].Method
	})

	assertEndpoint(t, endpoints[0], collector.ApiEndpoint{
		Name: "health_check", Path: "/health", Method: "GET", Protocol: "http",
		Description: "healthCheck returns service health status.",
	})

	assertEndpoint(t, endpoints[1], collector.ApiEndpoint{
		Name: "list_items", Path: "/items", Method: "GET", Protocol: "http",
		Description: "listItems returns all items.",
	})

	assertEndpoint(t, endpoints[2], collector.ApiEndpoint{
		Name: "create_item", Path: "/items", Method: "POST", Protocol: "http",
		Description: "createItem creates a new item.",
	})

	assertEndpoint(t, endpoints[3], collector.ApiEndpoint{
		Name: "delete_item", Path: "/items/{id}", Method: "DELETE", Protocol: "http",
		Description: "deleteItem removes an item by ID.",
		Parameters: []collector.ApiParameter{
			{Name: "id", In: "path", Required: true, Type: "text"},
		},
	})

	assertEndpoint(t, endpoints[4], collector.ApiEndpoint{
		Name: "get_item", Path: "/items/{id}", Method: "GET", Protocol: "http",
		Description: "getItem returns a single item by ID.",
		Parameters: []collector.ApiParameter{
			{Name: "id", In: "path", Required: true, Type: "text"},
		},
	})
}

func TestExtractPathParams(t *testing.T) {
	tests := []struct {
		path     string
		expected []pathParamInfo
	}{
		{"/users", nil},
		{"/users/{id}", []pathParamInfo{{name: "id"}}},
		{"/users/{id}/posts/{postId}", []pathParamInfo{
			{name: "id"},
			{name: "postId"},
		}},
		{"/{category}/{item}", []pathParamInfo{
			{name: "category"},
			{name: "item"},
		}},
	}

	for _, tt := range tests {
		result := extractPathParams(tt.path)
		if len(result) != len(tt.expected) {
			t.Errorf("extractPathParams(%q): got %d params, want %d", tt.path, len(result), len(tt.expected))
			continue
		}
		for i, p := range result {
			if p.name != tt.expected[i].name {
				t.Errorf("extractPathParams(%q)[%d].name = %q, want %q", tt.path, i, p.name, tt.expected[i].name)
			}
		}
	}
}

func TestUnquotePythonString(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`"hello"`, "hello"},
		{`'hello'`, "hello"},
		{`"""hello"""`, "hello"},
		{`'''hello'''`, "hello"},
		{`hello`, "hello"},
		{`""`, ""},
		{`"a/b/c"`, "a/b/c"},
	}

	for _, tt := range tests {
		result := unquotePythonString(tt.input)
		if result != tt.expected {
			t.Errorf("unquotePythonString(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestParse_TypedRoutes(t *testing.T) {
	endpoints, err := Parse(filepath.Join("testdata", "typed"))
	if err != nil {
		t.Fatalf("typed routes should not error: %v", err)
	}
	if len(endpoints) == 0 {
		t.Fatal("expected endpoints for typed routes")
	}

	sort.Slice(endpoints, func(i, j int) bool {
		if endpoints[i].Path != endpoints[j].Path {
			return endpoints[i].Path < endpoints[j].Path
		}
		return endpoints[i].Method < endpoints[j].Method
	})

	epMap := make(map[string]collector.ApiEndpoint)
	for _, ep := range endpoints {
		key := ep.Method + " " + ep.Path
		epMap[key] = ep
	}

	t.Run("POST /users has request body with CreateUserReq schema", func(t *testing.T) {
		ep, ok := epMap["POST /users"]
		if !ok {
			t.Fatal("missing POST /users endpoint")
		}
		if ep.RequestBody == nil {
			t.Fatal("expected request body for POST /users")
		}
		body := ep.RequestBody
		if body.Body == nil {
			t.Fatal("expected Body for request body")
		}
		if body.Body.Kind != "object" {
			t.Errorf("request body Body.Kind = %q, want %q", body.Body.Kind, "object")
		}
		if body.Body.TypeName != "CreateUserReq" {
			t.Errorf("request body Body.TypeName = %q, want %q", body.Body.TypeName, "CreateUserReq")
		}
		if len(body.Body.Fields) != 3 {
			t.Fatalf("request body Body.Fields count = %d, want 3", len(body.Body.Fields))
		}
		nameField, ok := body.Body.Fields["name"]
		if !ok {
			t.Fatal("expected 'name' field in CreateUserReq")
		}
		if nameField.Model.TypeName != "string" {
			t.Errorf("CreateUserReq.name type = %q, want %q", nameField.Model.TypeName, "string")
		}
	})

	t.Run("POST /users has response body with UserResult schema", func(t *testing.T) {
		ep, ok := epMap["POST /users"]
		if !ok {
			t.Fatal("missing POST /users endpoint")
		}
		if ep.Response == nil {
			t.Fatal("expected response body for POST /users")
		}
		body := ep.Response
		if body.Body == nil {
			t.Fatal("expected Body for response body")
		}
		if body.Body.Kind != "object" {
			t.Errorf("response body Body.Kind = %q, want %q", body.Body.Kind, "object")
		}
		if body.Body.TypeName != "UserResult" {
			t.Errorf("response body Body.TypeName = %q, want %q", body.Body.TypeName, "UserResult")
		}
		dataField, ok := body.Body.Fields["data"]
		if !ok {
			t.Fatal("expected 'data' field in UserResult")
		}
		if dataField.Model.Kind != "object" {
			t.Errorf("UserResult.data kind = %q, want %q", dataField.Model.Kind, "object")
		}
		if dataField.Model.TypeName != "User" {
			t.Errorf("UserResult.data typeName = %q, want %q", dataField.Model.TypeName, "User")
		}
	})

	t.Run("GET /users has response body with PaginatedUsers schema", func(t *testing.T) {
		ep, ok := epMap["GET /users"]
		if !ok {
			t.Fatal("missing GET /users endpoint")
		}
		if ep.Response == nil {
			t.Fatal("expected response body for GET /users")
		}
		body := ep.Response
		if body.Body == nil {
			t.Fatal("expected Body for response body")
		}
		if body.Body.TypeName != "PaginatedUsers" {
			t.Errorf("response body Body.TypeName = %q, want %q", body.Body.TypeName, "PaginatedUsers")
		}
		itemsField, ok := body.Body.Fields["items"]
		if !ok {
			t.Fatal("expected 'items' field in PaginatedUsers")
		}
		if itemsField.Model.Kind != "array" {
			t.Errorf("PaginatedUsers.items kind = %q, want %q", itemsField.Model.Kind, "array")
		}
	})

	t.Run("GET /items has return type List[Item]", func(t *testing.T) {
		ep, ok := epMap["GET /items"]
		if !ok {
			t.Fatal("missing GET /items endpoint")
		}
		if ep.Response == nil {
			t.Fatal("expected response body for GET /items")
		}
		body := ep.Response
		if body.Body == nil {
			t.Fatal("expected Body for response body")
		}
		if body.Body.Kind != "array" {
			t.Errorf("response body Body.Kind = %q, want %q", body.Body.Kind, "array")
		}
	})

	t.Run("GET /config has return type Dict[str, str]", func(t *testing.T) {
		ep, ok := epMap["GET /config"]
		if !ok {
			t.Fatal("missing GET /config endpoint")
		}
		if ep.Response == nil {
			t.Fatal("expected response body for GET /config")
		}
		body := ep.Response
		if body.Body == nil {
			t.Fatal("expected Body for response body")
		}
		if body.Body.Kind != "map" {
			t.Errorf("response body Body.Kind = %q, want %q", body.Body.Kind, "map")
		}
	})
}

func TestParse_Inheritance(t *testing.T) {
	endpoints, err := Parse(filepath.Join("testdata", "inheritance"))
	if err != nil {
		t.Fatalf("inheritance routes should not error: %v", err)
	}
	if len(endpoints) == 0 {
		t.Fatal("expected endpoints for inheritance routes")
	}

	epMap := make(map[string]collector.ApiEndpoint)
	for _, ep := range endpoints {
		key := ep.Method + " " + ep.Path
		epMap[key] = ep
	}

	t.Run("POST /users has User with inherited fields", func(t *testing.T) {
		ep, ok := epMap["POST /users"]
		if !ok {
			t.Fatal("missing POST /users endpoint")
		}
		if ep.Response == nil {
			t.Fatal("expected response body for POST /users")
		}
		body := ep.Response
		if body.Body == nil {
			t.Fatal("expected Body for response body")
		}
		if body.Body.TypeName != "User" {
			t.Errorf("response body Body.TypeName = %q, want %q", body.Body.TypeName, "User")
		}
		if len(body.Body.Fields) < 4 {
			t.Fatalf("User should have at least 4 fields (2 own + 2 inherited), got %d", len(body.Body.Fields))
		}
		if _, ok := body.Body.Fields["id"]; !ok {
			t.Error("expected inherited 'id' field in User")
		}
		if _, ok := body.Body.Fields["created_at"]; !ok {
			t.Error("expected inherited 'created_at' field in User")
		}
		if _, ok := body.Body.Fields["name"]; !ok {
			t.Error("expected own 'name' field in User")
		}
		if _, ok := body.Body.Fields["email"]; !ok {
			t.Error("expected own 'email' field in User")
		}
	})
}

func assertEndpoint(t *testing.T, got collector.ApiEndpoint, want collector.ApiEndpoint) {
	t.Helper()

	if got.Name != want.Name {
		t.Errorf("Name = %q, want %q", got.Name, want.Name)
	}
	if got.Path != want.Path {
		t.Errorf("Path = %q, want %q", got.Path, want.Path)
	}
	if got.Method != want.Method {
		t.Errorf("Method = %q, want %q", got.Method, want.Method)
	}
	if got.Protocol != want.Protocol {
		t.Errorf("Protocol = %q, want %q", got.Protocol, want.Protocol)
	}
	if got.Description != want.Description {
		t.Errorf("Description = %q, want %q", got.Description, want.Description)
	}

	assertParams(t, got.Parameters, want.Parameters)
}

func assertParams(t *testing.T, got, want []collector.ApiParameter) {
	t.Helper()

	if len(got) != len(want) {
		t.Errorf("Parameters: got %d, want %d", len(got), len(want))
		return
	}

	for i, g := range got {
		w := want[i]
		if g.Name != w.Name {
			t.Errorf("Parameters[%d].Name = %q, want %q", i, g.Name, w.Name)
		}
		if g.In != w.In {
			t.Errorf("Parameters[%d].In = %q, want %q", i, g.In, w.In)
		}
		if g.Required != w.Required {
			t.Errorf("Parameters[%d].Required = %v, want %v", i, g.Required, w.Required)
		}
		if g.Type != w.Type {
			t.Errorf("Parameters[%d].Type = %q, want %q", i, g.Type, w.Type)
		}
	}
}
