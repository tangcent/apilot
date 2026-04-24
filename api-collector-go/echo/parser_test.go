package echo

import (
	"path/filepath"
	"sort"
	"testing"

	"github.com/tangcent/apilot/api-collector"
	model "github.com/tangcent/apilot/api-model"
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

func TestParse_BasicRoutes(t *testing.T) {
	endpoints, err := Parse(filepath.Join("testdata", "basic"))
	if err != nil {
		t.Fatalf("basic routes should not error: %v", err)
	}
	if len(endpoints) != 7 {
		t.Fatalf("expected 7 endpoints, got %d", len(endpoints))
	}

	sort.Slice(endpoints, func(i, j int) bool {
		if endpoints[i].Method != endpoints[j].Method {
			return endpoints[i].Method < endpoints[j].Method
		}
		return endpoints[i].Path < endpoints[j].Path
	})

	assertEndpoint(t, endpoints[0], collector.ApiEndpoint{
		Name: "deleteUser", Path: "/users/:id", Method: "DELETE", Protocol: "http",
		Description: "deleteUser removes a user by ID.",
		Parameters: []collector.ApiParameter{
			{Name: "id", In: "path", Required: true, Type: "text"},
		},
	})

	assertEndpoint(t, endpoints[1], collector.ApiEndpoint{
		Name: "listUsers", Path: "/users", Method: "GET", Protocol: "http",
		Description: "listUsers returns all users.",
		Parameters: []collector.ApiParameter{
			{Name: "name", In: "query", Required: true, Type: "text"},
		},
		Response: &collector.ApiBody{MediaType: "application/json", Body: model.SingleModel("map")},
	})

	assertEndpoint(t, endpoints[2], collector.ApiEndpoint{
		Name: "getUser", Path: "/users/:id", Method: "GET", Protocol: "http",
		Description: "getUser returns a single user by ID.",
		Parameters: []collector.ApiParameter{
			{Name: "id", In: "path", Required: true, Type: "text"},
		},
		Response: &collector.ApiBody{MediaType: "application/json", Body: model.SingleModel("map")},
	})

	assertEndpoint(t, endpoints[3], collector.ApiEndpoint{
		Name: "patchUser", Path: "/users/:id", Method: "PATCH", Protocol: "http",
		Description: "patchUser partially updates a user.",
		Parameters: []collector.ApiParameter{
			{Name: "id", In: "path", Required: true, Type: "text"},
			{Name: "name", In: "query", Required: true, Type: "text"},
		},
		Response: &collector.ApiBody{MediaType: "application/json", Body: model.SingleModel("map")},
	})

	assertEndpoint(t, endpoints[4], collector.ApiEndpoint{
		Name: "uploadFile", Path: "/upload", Method: "POST", Protocol: "http",
		Description: "uploadFile handles file uploads.",
		Parameters: []collector.ApiParameter{
			{Name: "file", In: "form", Required: true, Type: "file"},
			{Name: "description", In: "form", Required: true, Type: "text"},
		},
		Response: &collector.ApiBody{MediaType: "application/json", Body: model.SingleModel("map")},
	})

	assertEndpoint(t, endpoints[5], collector.ApiEndpoint{
		Name: "createUser", Path: "/users", Method: "POST", Protocol: "http",
		Description: "createUser creates a new user.",
		RequestBody: &collector.ApiBody{MediaType: "application/json", Body: model.SingleModel("req")},
		Response:   &collector.ApiBody{MediaType: "application/json", Body: model.SingleModel("req")},
	})

	assertEndpoint(t, endpoints[6], collector.ApiEndpoint{
		Name: "updateUser", Path: "/users/:id", Method: "PUT", Protocol: "http",
		Description: "updateUser updates an existing user.",
		Parameters: []collector.ApiParameter{
			{Name: "id", In: "path", Required: true, Type: "text"},
		},
		RequestBody: &collector.ApiBody{MediaType: "application/json", Body: model.SingleModel("req")},
		Response:   &collector.ApiBody{MediaType: "application/json", Body: model.SingleModel("req")},
	})
}

func TestParse_GroupRoutes(t *testing.T) {
	endpoints, err := Parse(filepath.Join("testdata", "groups"))
	if err != nil {
		t.Fatalf("group routes should not error: %v", err)
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
		Name: "listItems", Path: "/api/items", Method: "GET", Protocol: "http",
		Description: "listItems returns all items.",
		Response:    &collector.ApiBody{MediaType: "application/json", Body: model.SingleModel("map")},
	})

	assertEndpoint(t, endpoints[1], collector.ApiEndpoint{
		Name: "deleteItem", Path: "/api/items/:id", Method: "DELETE", Protocol: "http",
		Description: "deleteItem removes an item by ID.",
		Parameters: []collector.ApiParameter{
			{Name: "id", In: "path", Required: true, Type: "text"},
		},
	})

	assertEndpoint(t, endpoints[2], collector.ApiEndpoint{
		Name: "healthCheck", Path: "/health", Method: "GET", Protocol: "http",
		Description: "healthCheck returns service health status.",
		Response:    &collector.ApiBody{MediaType: "text/plain"},
	})

	assertEndpoint(t, endpoints[3], collector.ApiEndpoint{
		Name: "listUsers", Path: "/v1/users", Method: "GET", Protocol: "http",
		Description: "listUsers returns all users.",
		Parameters: []collector.ApiParameter{
			{Name: "name", In: "query", Required: true, Type: "text"},
		},
		Response: &collector.ApiBody{MediaType: "application/json", Body: model.SingleModel("map")},
	})

	assertEndpoint(t, endpoints[4], collector.ApiEndpoint{
		Name: "createUser", Path: "/v1/users", Method: "POST", Protocol: "http",
		Description: "createUser creates a new user.",
		RequestBody: &collector.ApiBody{MediaType: "application/json", Body: model.SingleModel("req")},
		Response:   &collector.ApiBody{MediaType: "application/json", Body: model.SingleModel("map")},
	})
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

func TestExtractPathParams(t *testing.T) {
	tests := []struct {
		path     string
		expected []rawParam
	}{
		{"/users", nil},
		{"/users/:id", []rawParam{{name: "id", in: "path", required: true, typ: "text"}}},
		{"/users/:id/posts/:postId", []rawParam{
			{name: "id", in: "path", required: true, typ: "text"},
			{name: "postId", in: "path", required: true, typ: "text"},
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
			if p.in != tt.expected[i].in {
				t.Errorf("extractPathParams(%q)[%d].in = %q, want %q", tt.path, i, p.in, tt.expected[i].in)
			}
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
	assertBody(t, "RequestBody", got.RequestBody, want.RequestBody)
	assertBody(t, "Response", got.Response, want.Response)
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
		if g.Default != w.Default {
			t.Errorf("Parameters[%d].Default = %q, want %q", i, g.Default, w.Default)
		}
	}
}

func assertBody(t *testing.T, field string, got, want *collector.ApiBody) {
	t.Helper()

	if got == nil && want == nil {
		return
	}
	if got == nil {
		t.Errorf("%s: got nil, want %+v", field, want)
		return
	}
	if want == nil {
		t.Errorf("%s: got %+v, want nil", field, got)
		return
	}
	if got.MediaType != want.MediaType {
		t.Errorf("%s.MediaType = %q, want %q", field, got.MediaType, want.MediaType)
	}
	if want.Body != nil {
		if got.Body == nil {
			t.Errorf("%s.Body = nil, want %+v", field, want.Body)
		}
	}
}
