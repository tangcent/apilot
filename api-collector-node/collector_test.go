package nodecollector

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

func TestCollect_ExpressOnly(t *testing.T) {
	c := New()
	endpoints, err := c.Collect(collector.CollectContext{
		SourceDir: filepath.Join("testdata", "expressonly"),
	})
	if err != nil {
		t.Fatalf("expressonly should not error: %v", err)
	}

	assertContainsEndpoint(t, endpoints, collector.ApiEndpoint{
		Name: "listUsers", Path: "/users", Method: "GET", Protocol: "http",
		Description: "listUsers returns all users.",
	})
	assertContainsEndpoint(t, endpoints, collector.ApiEndpoint{
		Name: "createUser", Path: "/users", Method: "POST", Protocol: "http",
		Description: "createUser creates a new user.",
	})
	assertContainsEndpoint(t, endpoints, collector.ApiEndpoint{
		Name: "getUser", Path: "/users/{id}", Method: "GET", Protocol: "http",
		Description: "getUser returns a single user by ID.",
		Parameters: []collector.ApiParameter{
			{Name: "id", In: "path", Required: true, Type: "text"},
		},
	})
}

func TestCollect_FastifyOnly(t *testing.T) {
	c := New()
	endpoints, err := c.Collect(collector.CollectContext{
		SourceDir: filepath.Join("testdata", "fastifyonly"),
	})
	if err != nil {
		t.Fatalf("fastifyonly should not error: %v", err)
	}
	if len(endpoints) != 3 {
		t.Fatalf("expected 3 endpoints, got %d", len(endpoints))
	}

	sort.Slice(endpoints, func(i, j int) bool {
		if endpoints[i].Path != endpoints[j].Path {
			return endpoints[i].Path < endpoints[j].Path
		}
		return endpoints[i].Method < endpoints[j].Method
	})

	assertEndpoint(t, endpoints[0], collector.ApiEndpoint{
		Name: "listItems", Path: "/items", Method: "GET", Protocol: "http",
		Description: "listItems returns all items.",
	})

	assertEndpoint(t, endpoints[1], collector.ApiEndpoint{
		Name: "createItem", Path: "/items", Method: "POST", Protocol: "http",
		Description: "createItem creates a new item.",
	})

	assertEndpoint(t, endpoints[2], collector.ApiEndpoint{
		Name: "getItem", Path: "/items/{id}", Method: "GET", Protocol: "http",
		Description: "getItem returns a single item by ID.",
		Parameters: []collector.ApiParameter{
			{Name: "id", In: "path", Required: true, Type: "text"},
		},
	})
}

func TestCollect_NestJSOnly(t *testing.T) {
	c := New()
	endpoints, err := c.Collect(collector.CollectContext{
		SourceDir: filepath.Join("testdata", "nestjsonly"),
	})
	if err != nil {
		t.Fatalf("nestjsonly should not error: %v", err)
	}
	if len(endpoints) != 3 {
		t.Fatalf("expected 3 endpoints, got %d", len(endpoints))
	}

	sort.Slice(endpoints, func(i, j int) bool {
		if endpoints[i].Path != endpoints[j].Path {
			return endpoints[i].Path < endpoints[j].Path
		}
		return endpoints[i].Method < endpoints[j].Method
	})

	assertEndpoint(t, endpoints[0], collector.ApiEndpoint{
		Name: "listProducts", Path: "/products", Method: "GET", Protocol: "http",
		Description: "listProducts returns all products.",
		Parameters: []collector.ApiParameter{
			{Name: "page", In: "query", Required: false, Type: "text"},
		},
	})

	assertEndpoint(t, endpoints[1], collector.ApiEndpoint{
		Name: "createProduct", Path: "/products", Method: "POST", Protocol: "http",
		Description: "createProduct creates a new product.",
		Parameters: []collector.ApiParameter{
			{Name: "body", In: "body", Required: true, Type: "text"},
		},
	})

	assertEndpoint(t, endpoints[2], collector.ApiEndpoint{
		Name: "getProduct", Path: "/products/:id", Method: "GET", Protocol: "http",
		Description: "getProduct returns a single product by ID.",
		Parameters: []collector.ApiParameter{
			{Name: "id", In: "path", Required: true, Type: "text"},
		},
	})
}

func TestCollect_Mixed(t *testing.T) {
	c := New()
	endpoints, err := c.Collect(collector.CollectContext{
		SourceDir: filepath.Join("testdata", "mixed"),
	})
	if err != nil {
		t.Fatalf("mixed should not error: %v", err)
	}

	assertContainsEndpoint(t, endpoints, collector.ApiEndpoint{
		Name: "expressHello", Path: "/express/hello", Method: "GET", Protocol: "http",
		Description: "expressHello returns a greeting from Express.",
	})
	assertContainsEndpoint(t, endpoints, collector.ApiEndpoint{
		Name: "expressCreateUser", Path: "/express/users", Method: "POST", Protocol: "http",
		Description: "expressCreateUser creates a new user via Express.",
	})
	assertContainsEndpoint(t, endpoints, collector.ApiEndpoint{
		Name: "fastifyHello", Path: "/fastify/hello", Method: "GET", Protocol: "http",
		Description: "fastifyHello returns a greeting from Fastify.",
	})
	assertContainsEndpoint(t, endpoints, collector.ApiEndpoint{
		Name: "fastifyDeleteUser", Path: "/fastify/users/{id}", Method: "DELETE", Protocol: "http",
		Description: "fastifyDeleteUser deletes a user via Fastify.",
		Parameters: []collector.ApiParameter{
			{Name: "id", In: "path", Required: true, Type: "text"},
		},
	})
	assertContainsEndpoint(t, endpoints, collector.ApiEndpoint{
		Name: "listOrders", Path: "/orders", Method: "GET", Protocol: "http",
		Description: "listOrders returns all orders.",
	})
	assertContainsEndpoint(t, endpoints, collector.ApiEndpoint{
		Name: "createOrder", Path: "/orders", Method: "POST", Protocol: "http",
		Description: "createOrder creates a new order.",
		Parameters: []collector.ApiParameter{
			{Name: "body", In: "body", Required: true, Type: "text"},
		},
	})
	assertContainsEndpoint(t, endpoints, collector.ApiEndpoint{
		Name: "getOrder", Path: "/orders/:id", Method: "GET", Protocol: "http",
		Description: "getOrder returns a single order by ID.",
		Parameters: []collector.ApiParameter{
			{Name: "id", In: "path", Required: true, Type: "text"},
		},
	})
}

func TestNodeCollector_Interface(t *testing.T) {
	c := New()

	if c.Name() != "node" {
		t.Errorf("Name() = %q, want %q", c.Name(), "node")
	}

	langs := c.SupportedLanguages()
	if len(langs) != 2 || langs[0] != "typescript" || langs[1] != "javascript" {
		t.Errorf("SupportedLanguages() = %v, want [typescript javascript]", langs)
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

func assertContainsEndpoint(t *testing.T, endpoints []collector.ApiEndpoint, want collector.ApiEndpoint) {
	t.Helper()

	for _, got := range endpoints {
		if got.Name == want.Name && got.Path == want.Path && got.Method == want.Method {
			assertEndpoint(t, got, want)
			return
		}
	}

	t.Errorf("endpoint not found: %s %s %s", want.Method, want.Path, want.Name)
}
