package gin

import (
	"path/filepath"
	"sort"
	"testing"

	"github.com/tangcent/apilot/api-collector"
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
	if len(endpoints) != 8 {
		t.Fatalf("expected 8 endpoints, got %d", len(endpoints))
	}

	sort.Slice(endpoints, func(i, j int) bool {
		if endpoints[i].Method != endpoints[j].Method {
			return endpoints[i].Method < endpoints[j].Method
		}
		return endpoints[i].Path < endpoints[j].Path
	})

	expected := []collector.ApiEndpoint{
		{Name: "deleteUser", Path: "/users/:id", Method: "DELETE", Protocol: "http", Description: "deleteUser removes a user by ID."},
		{Name: "listUsers", Path: "/users", Method: "GET", Protocol: "http", Description: "listUsers returns all users."},
		{Name: "getUser", Path: "/users/:id", Method: "GET", Protocol: "http", Description: "getUser returns a single user by ID."},
		{Name: "healthCheck", Path: "/health", Method: "HEAD", Protocol: "http", Description: "healthCheck returns service health status."},
		{Name: "userOptions", Path: "/users", Method: "OPTIONS", Protocol: "http", Description: "userOptions returns allowed methods for /users."},
		{Name: "patchUser", Path: "/users/:id", Method: "PATCH", Protocol: "http", Description: "patchUser partially updates a user."},
		{Name: "createUser", Path: "/users", Method: "POST", Protocol: "http", Description: "createUser creates a new user."},
		{Name: "updateUser", Path: "/users/:id", Method: "PUT", Protocol: "http", Description: "updateUser updates an existing user."},
	}

	for i, exp := range expected {
		if endpoints[i].Name != exp.Name {
			t.Errorf("endpoint[%d].Name = %q, want %q", i, endpoints[i].Name, exp.Name)
		}
		if endpoints[i].Path != exp.Path {
			t.Errorf("endpoint[%d].Path = %q, want %q", i, endpoints[i].Path, exp.Path)
		}
		if endpoints[i].Method != exp.Method {
			t.Errorf("endpoint[%d].Method = %q, want %q", i, endpoints[i].Method, exp.Method)
		}
		if endpoints[i].Protocol != exp.Protocol {
			t.Errorf("endpoint[%d].Protocol = %q, want %q", i, endpoints[i].Protocol, exp.Protocol)
		}
		if endpoints[i].Description != exp.Description {
			t.Errorf("endpoint[%d].Description = %q, want %q", i, endpoints[i].Description, exp.Description)
		}
	}
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

	expected := []collector.ApiEndpoint{
		{Name: "listItems", Path: "/api/items", Method: "GET", Protocol: "http", Description: "listItems returns all items."},
		{Name: "deleteItem", Path: "/api/items/:id", Method: "DELETE", Protocol: "http", Description: "deleteItem removes an item by ID."},
		{Name: "healthCheck", Path: "/health", Method: "GET", Protocol: "http", Description: "healthCheck returns service health status."},
		{Name: "listUsers", Path: "/v1/users", Method: "GET", Protocol: "http", Description: "listUsers returns all users."},
		{Name: "createUser", Path: "/v1/users", Method: "POST", Protocol: "http", Description: "createUser creates a new user."},
	}

	for i, exp := range expected {
		if endpoints[i].Name != exp.Name {
			t.Errorf("endpoint[%d].Name = %q, want %q", i, endpoints[i].Name, exp.Name)
		}
		if endpoints[i].Path != exp.Path {
			t.Errorf("endpoint[%d].Path = %q, want %q", i, endpoints[i].Path, exp.Path)
		}
		if endpoints[i].Method != exp.Method {
			t.Errorf("endpoint[%d].Method = %q, want %q", i, endpoints[i].Method, exp.Method)
		}
		if endpoints[i].Protocol != exp.Protocol {
			t.Errorf("endpoint[%d].Protocol = %q, want %q", i, endpoints[i].Protocol, exp.Protocol)
		}
		if endpoints[i].Description != exp.Description {
			t.Errorf("endpoint[%d].Description = %q, want %q", i, endpoints[i].Description, exp.Description)
		}
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
