package gocollector

import (
	"path/filepath"
	"sort"
	"testing"

	collector "github.com/tangcent/apilot/api-collector"
)

func TestCollect_EmptyDir(t *testing.T) {
	c := New()
	endpoints, err := c.Collect(collector.CollectContext{
		SourceDir: filepath.Join("testdata", "empty"),
	})
	if err != nil {
		t.Fatalf("empty dir should not error: %v", err)
	}
	if endpoints != nil {
		t.Fatalf("expected nil endpoints for empty dir, got %d", len(endpoints))
	}
}

func TestCollect_NonexistentDir(t *testing.T) {
	c := New()
	endpoints, err := c.Collect(collector.CollectContext{
		SourceDir: filepath.Join("testdata", "nonexistent"),
	})
	if err != nil {
		t.Fatalf("nonexistent dir should not error: %v", err)
	}
	if endpoints != nil {
		t.Fatalf("expected nil endpoints for nonexistent dir, got %d", len(endpoints))
	}
}

func TestCollect_GinOnly(t *testing.T) {
	c := New()
	endpoints, err := c.Collect(collector.CollectContext{
		SourceDir: filepath.Join("testdata", "ginonly"),
	})
	if err != nil {
		t.Fatalf("ginonly should not error: %v", err)
	}
	if len(endpoints) != 1 {
		t.Fatalf("expected 1 endpoint, got %d", len(endpoints))
	}

	ep := endpoints[0]
	if ep.Name != "ping" {
		t.Errorf("Name = %q, want %q", ep.Name, "ping")
	}
	if ep.Path != "/ping" {
		t.Errorf("Path = %q, want %q", ep.Path, "/ping")
	}
	if ep.Method != "GET" {
		t.Errorf("Method = %q, want %q", ep.Method, "GET")
	}
	if ep.Protocol != "http" {
		t.Errorf("Protocol = %q, want %q", ep.Protocol, "http")
	}
	if ep.Description != "ping returns pong." {
		t.Errorf("Description = %q, want %q", ep.Description, "ping returns pong.")
	}
}

func TestCollect_EchoOnly(t *testing.T) {
	c := New()
	endpoints, err := c.Collect(collector.CollectContext{
		SourceDir: filepath.Join("testdata", "echonly"),
	})
	if err != nil {
		t.Fatalf("echonly should not error: %v", err)
	}
	if len(endpoints) != 1 {
		t.Fatalf("expected 1 endpoint, got %d", len(endpoints))
	}

	ep := endpoints[0]
	if ep.Name != "healthCheck" {
		t.Errorf("Name = %q, want %q", ep.Name, "healthCheck")
	}
	if ep.Path != "/health" {
		t.Errorf("Path = %q, want %q", ep.Path, "/health")
	}
	if ep.Method != "GET" {
		t.Errorf("Method = %q, want %q", ep.Method, "GET")
	}
	if ep.Protocol != "http" {
		t.Errorf("Protocol = %q, want %q", ep.Protocol, "http")
	}
	if ep.Description != "healthCheck returns service health." {
		t.Errorf("Description = %q, want %q", ep.Description, "healthCheck returns service health.")
	}
}

func TestCollect_FiberOnly(t *testing.T) {
	c := New()
	endpoints, err := c.Collect(collector.CollectContext{
		SourceDir: filepath.Join("testdata", "fiberonly"),
	})
	if err != nil {
		t.Fatalf("fiberonly should not error: %v", err)
	}
	if len(endpoints) != 1 {
		t.Fatalf("expected 1 endpoint, got %d", len(endpoints))
	}

	ep := endpoints[0]
	if ep.Name != "statusHandler" {
		t.Errorf("Name = %q, want %q", ep.Name, "statusHandler")
	}
	if ep.Path != "/status" {
		t.Errorf("Path = %q, want %q", ep.Path, "/status")
	}
	if ep.Method != "GET" {
		t.Errorf("Method = %q, want %q", ep.Method, "GET")
	}
	if ep.Protocol != "http" {
		t.Errorf("Protocol = %q, want %q", ep.Protocol, "http")
	}
	if ep.Description != "statusHandler returns service status." {
		t.Errorf("Description = %q, want %q", ep.Description, "statusHandler returns service status.")
	}
}

func TestCollect_Mixed(t *testing.T) {
	c := New()
	endpoints, err := c.Collect(collector.CollectContext{
		SourceDir: filepath.Join("testdata", "mixed"),
	})
	if err != nil {
		t.Fatalf("mixed should not error: %v", err)
	}
	if len(endpoints) != 6 {
		t.Fatalf("expected 6 endpoints, got %d", len(endpoints))
	}

	sort.Slice(endpoints, func(i, j int) bool {
		if endpoints[i].Path != endpoints[j].Path {
			return endpoints[i].Path < endpoints[j].Path
		}
		return endpoints[i].Method < endpoints[j].Method
	})

	assertEndpoint(t, endpoints[0], collector.ApiEndpoint{
		Name: "echoHello", Path: "/echo/hello", Method: "GET", Protocol: "http",
		Description: "echoHello returns a greeting from Echo.",
	})

	assertEndpoint(t, endpoints[1], collector.ApiEndpoint{
		Name: "echoDeleteUser", Path: "/echo/users/:id", Method: "DELETE", Protocol: "http",
		Description: "echoDeleteUser deletes a user via Echo.",
		Parameters: []collector.ApiParameter{
			{Name: "id", In: "path", Required: true, Type: "text"},
		},
	})

	assertEndpoint(t, endpoints[2], collector.ApiEndpoint{
		Name: "fiberHello", Path: "/fiber/hello", Method: "GET", Protocol: "http",
		Description: "fiberHello returns a greeting from Fiber.",
	})

	assertEndpoint(t, endpoints[3], collector.ApiEndpoint{
		Name: "fiberUpdateUser", Path: "/fiber/users/:id", Method: "PUT", Protocol: "http",
		Description: "fiberUpdateUser updates a user via Fiber.",
		Parameters: []collector.ApiParameter{
			{Name: "id", In: "path", Required: true, Type: "text"},
		},
	})

	assertEndpoint(t, endpoints[4], collector.ApiEndpoint{
		Name: "ginHello", Path: "/gin/hello", Method: "GET", Protocol: "http",
		Description: "ginHello returns a greeting from Gin.",
	})

	assertEndpoint(t, endpoints[5], collector.ApiEndpoint{
		Name: "ginCreateUser", Path: "/gin/users", Method: "POST", Protocol: "http",
		Description: "ginCreateUser creates a new user via Gin.",
	})
}

func TestGoCollector_Interface(t *testing.T) {
	c := New()

	if c.Name() != "go" {
		t.Errorf("Name() = %q, want %q", c.Name(), "go")
	}

	langs := c.SupportedLanguages()
	if len(langs) != 1 || langs[0] != "go" {
		t.Errorf("SupportedLanguages() = %v, want [go]", langs)
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

	if len(got.Parameters) != len(want.Parameters) {
		t.Errorf("Parameters: got %d, want %d", len(got.Parameters), len(want.Parameters))
		return
	}
	for i, g := range got.Parameters {
		w := want.Parameters[i]
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
