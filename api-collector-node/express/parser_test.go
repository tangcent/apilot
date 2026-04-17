package express

import (
	"path/filepath"
	"sort"
	"testing"

	collector "github.com/tangcent/apilot/api-collector"
)

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
	if len(endpoints) != 8 {
		t.Fatalf("expected 8 endpoints, got %d", len(endpoints))
	}

	sort.Slice(endpoints, func(i, j int) bool {
		if endpoints[i].Method != endpoints[j].Method {
			return endpoints[i].Method < endpoints[j].Method
		}
		return endpoints[i].Path < endpoints[j].Path
	})

	assertEndpoint(t, endpoints[0], collector.ApiEndpoint{
		Name: "deleteUser", Path: "/users/{id}", Method: "DELETE", Protocol: "http",
		Description: "deleteUser removes a user by ID.",
		Parameters: []collector.ApiParameter{
			{Name: "id", In: "path", Required: true, Type: "text"},
		},
	})

	assertEndpoint(t, endpoints[1], collector.ApiEndpoint{
		Name: "listUsers", Path: "/users", Method: "GET", Protocol: "http",
		Description: "listUsers returns all users.",
	})

	assertEndpoint(t, endpoints[2], collector.ApiEndpoint{
		Name: "getUser", Path: "/users/{id}", Method: "GET", Protocol: "http",
		Description: "getUser returns a single user by ID.",
		Parameters: []collector.ApiParameter{
			{Name: "id", In: "path", Required: true, Type: "text"},
		},
	})

	assertEndpoint(t, endpoints[3], collector.ApiEndpoint{
		Name: "healthCheck", Path: "/health", Method: "HEAD", Protocol: "http",
		Description: "healthCheck returns service health status.",
	})

	assertEndpoint(t, endpoints[4], collector.ApiEndpoint{
		Name: "userOptions", Path: "/users", Method: "OPTIONS", Protocol: "http",
		Description: "userOptions returns allowed methods for /users.",
	})

	assertEndpoint(t, endpoints[5], collector.ApiEndpoint{
		Name: "patchUser", Path: "/users/{id}", Method: "PATCH", Protocol: "http",
		Description: "patchUser partially updates a user.",
		Parameters: []collector.ApiParameter{
			{Name: "id", In: "path", Required: true, Type: "text"},
		},
	})

	assertEndpoint(t, endpoints[6], collector.ApiEndpoint{
		Name: "createUser", Path: "/users", Method: "POST", Protocol: "http",
		Description: "createUser creates a new user.",
	})

	assertEndpoint(t, endpoints[7], collector.ApiEndpoint{
		Name: "updateUser", Path: "/users/{id}", Method: "PUT", Protocol: "http",
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
		Name: "healthCheck", Path: "/health", Method: "GET", Protocol: "http",
		Description: "healthCheck returns service health status.",
	})

	assertEndpoint(t, endpoints[1], collector.ApiEndpoint{
		Name: "listItems", Path: "/items", Method: "GET", Protocol: "http",
		Description: "listItems returns all items.",
	})

	assertEndpoint(t, endpoints[2], collector.ApiEndpoint{
		Name: "createItem", Path: "/items", Method: "POST", Protocol: "http",
		Description: "createItem creates a new item.",
	})

	assertEndpoint(t, endpoints[3], collector.ApiEndpoint{
		Name: "deleteItem", Path: "/items/{id}", Method: "DELETE", Protocol: "http",
		Description: "deleteItem removes an item by ID.",
		Parameters: []collector.ApiParameter{
			{Name: "id", In: "path", Required: true, Type: "text"},
		},
	})

	assertEndpoint(t, endpoints[4], collector.ApiEndpoint{
		Name: "getItem", Path: "/items/{id}", Method: "GET", Protocol: "http",
		Description: "getItem returns a single item by ID.",
		Parameters: []collector.ApiParameter{
			{Name: "id", In: "path", Required: true, Type: "text"},
		},
	})
}

func TestConvertExpressPath(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"/users", "/users"},
		{"/users/:id", "/users/{id}"},
		{"/users/:id/posts/:postId", "/users/{id}/posts/{postId}"},
		{"/:category/:item", "/{category}/{item}"},
		{"/api/v1/users/:id", "/api/v1/users/{id}"},
	}

	for _, tt := range tests {
		result := convertExpressPath(tt.input)
		if result != tt.expected {
			t.Errorf("convertExpressPath(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
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

func TestUnquoteJSString(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`"hello"`, "hello"},
		{`'hello'`, "hello"},
		{"`hello`", "hello"},
		{`hello`, "hello"},
		{`""`, ""},
		{`"a/b/c"`, "a/b/c"},
	}

	for _, tt := range tests {
		result := unquoteJSString(tt.input)
		if result != tt.expected {
			t.Errorf("unquoteJSString(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestCleanJSDocComment(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"/** hello */", "hello"},
		{"/**\n * line1\n * line2\n */", "line1 line2"},
		{"/**\n * listUsers returns all users.\n */", "listUsers returns all users."},
	}

	for _, tt := range tests {
		result := cleanJSDocComment(tt.input)
		if result != tt.expected {
			t.Errorf("cleanJSDocComment(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
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
