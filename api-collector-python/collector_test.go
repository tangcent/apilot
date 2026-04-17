package pycollector

import (
	"path/filepath"
	"sort"
	"testing"

	collector "github.com/tangcent/apilot/api-collector"
)

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

func TestCollect_FastAPIOnly(t *testing.T) {
	c := New()
	endpoints, err := c.Collect(collector.CollectContext{
		SourceDir: filepath.Join("testdata", "fastapionly"),
	})
	if err != nil {
		t.Fatalf("fastapionly should not error: %v", err)
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

func TestCollect_DjangoOnly(t *testing.T) {
	c := New()
	endpoints, err := c.Collect(collector.CollectContext{
		SourceDir: filepath.Join("testdata", "djangoonly"),
	})
	if err != nil {
		t.Fatalf("djangoonly should not error: %v", err)
	}
	if len(endpoints) != 1 {
		t.Fatalf("expected 1 endpoint, got %d", len(endpoints))
	}

	ep := endpoints[0]
	if ep.Name != "health_check" {
		t.Errorf("Name = %q, want %q", ep.Name, "health_check")
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

func TestCollect_FlaskOnly(t *testing.T) {
	c := New()
	endpoints, err := c.Collect(collector.CollectContext{
		SourceDir: filepath.Join("testdata", "flaskonly"),
	})
	if err != nil {
		t.Fatalf("flaskonly should not error: %v", err)
	}
	if len(endpoints) != 1 {
		t.Fatalf("expected 1 endpoint, got %d", len(endpoints))
	}

	ep := endpoints[0]
	if ep.Name != "status_handler" {
		t.Errorf("Name = %q, want %q", ep.Name, "status_handler")
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
		Name: "django_delete_user", Path: "/django_delete_user", Method: "DELETE", Protocol: "http",
		Description: "djangoDeleteUser deletes a user via Django.",
	})

	assertEndpoint(t, endpoints[1], collector.ApiEndpoint{
		Name: "django_hello", Path: "/django_hello", Method: "GET", Protocol: "http",
		Description: "djangoHello returns a greeting from Django.",
	})

	assertEndpoint(t, endpoints[2], collector.ApiEndpoint{
		Name: "fastapi_hello", Path: "/fastapi/hello", Method: "GET", Protocol: "http",
		Description: "fastapiHello returns a greeting from FastAPI.",
	})

	assertEndpoint(t, endpoints[3], collector.ApiEndpoint{
		Name: "fastapi_create_user", Path: "/fastapi/users", Method: "POST", Protocol: "http",
		Description: "fastapiCreateUser creates a new user via FastAPI.",
	})

	assertEndpoint(t, endpoints[4], collector.ApiEndpoint{
		Name: "flask_hello", Path: "/flask/hello", Method: "GET", Protocol: "http",
		Description: "flaskHello returns a greeting from Flask.",
	})

	assertEndpoint(t, endpoints[5], collector.ApiEndpoint{
		Name: "flask_update_user", Path: "/flask/users/{id}", Method: "PUT", Protocol: "http",
		Description: "flaskUpdateUser updates a user via Flask.",
		Parameters: []collector.ApiParameter{
			{Name: "id", In: "path", Required: true, Type: "text"},
		},
	})
}

func TestPythonCollector_Interface(t *testing.T) {
	c := New()

	if c.Name() != "python" {
		t.Errorf("Name() = %q, want %q", c.Name(), "python")
	}

	langs := c.SupportedLanguages()
	if len(langs) != 1 || langs[0] != "python" {
		t.Errorf("SupportedLanguages() = %v, want [python]", langs)
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
