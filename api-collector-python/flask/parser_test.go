package flask

import (
	"path/filepath"
	"sort"
	"testing"

	collector "github.com/tangcent/apilot/api-collector"
	model "github.com/tangcent/apilot/api-model"
)

func TestParse_BasicRoutes(t *testing.T) {
	endpoints, err := Parse(filepath.Join("testdata", "basic"))
	if err != nil {
		t.Fatalf("basic routes should not error: %v", err)
	}
	if len(endpoints) != 12 {
		t.Fatalf("expected 12 endpoints, got %d", len(endpoints))
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
		Name: "get_post", Path: "/items/{item_id}/posts/{post_id}", Method: "GET", Protocol: "http",
		Description: "getPost returns a specific post.",
		Parameters: []collector.ApiParameter{
			{Name: "item_id", In: "path", Required: true, Type: "text"},
			{Name: "post_id", In: "path", Required: true, Type: "text"},
		},
	})

	assertEndpoint(t, endpoints[2], collector.ApiEndpoint{
		Name: "list_products", Path: "/products", Method: "GET", Protocol: "http",
		Description: "listProducts returns all products.",
	})

	assertEndpoint(t, endpoints[3], collector.ApiEndpoint{
		Name: "product_detail", Path: "/products/{id}", Method: "GET", Protocol: "http",
		Description: "productDetail handles GET and POST for a product.",
		Parameters: []collector.ApiParameter{
			{Name: "id", In: "path", Required: true, Type: "text"},
		},
	})

	assertEndpoint(t, endpoints[4], collector.ApiEndpoint{
		Name: "list_users", Path: "/users", Method: "GET", Protocol: "http",
		Description: "listUsers returns all users.",
	})

	assertEndpoint(t, endpoints[5], collector.ApiEndpoint{
		Name: "get_user", Path: "/users/{id}", Method: "GET", Protocol: "http",
		Description: "getUser returns a single user by ID.",
		Parameters: []collector.ApiParameter{
			{Name: "id", In: "path", Required: true, Type: "text"},
		},
	})

	assertEndpoint(t, endpoints[6], collector.ApiEndpoint{
		Name: "health_check", Path: "/health", Method: "HEAD", Protocol: "http",
		Description: "healthCheck returns service health status.",
	})

	assertEndpoint(t, endpoints[7], collector.ApiEndpoint{
		Name: "user_options", Path: "/users", Method: "OPTIONS", Protocol: "http",
		Description: "userOptions returns allowed methods for /users.",
	})

	assertEndpoint(t, endpoints[8], collector.ApiEndpoint{
		Name: "patch_user", Path: "/users/{id}", Method: "PATCH", Protocol: "http",
		Description: "patchUser partially updates a user.",
		Parameters: []collector.ApiParameter{
			{Name: "id", In: "path", Required: true, Type: "text"},
		},
	})

	assertEndpoint(t, endpoints[9], collector.ApiEndpoint{
		Name: "product_detail", Path: "/products/{id}", Method: "POST", Protocol: "http",
		Description: "productDetail handles GET and POST for a product.",
		Parameters: []collector.ApiParameter{
			{Name: "id", In: "path", Required: true, Type: "text"},
		},
	})

	assertEndpoint(t, endpoints[10], collector.ApiEndpoint{
		Name: "create_user", Path: "/users", Method: "POST", Protocol: "http",
		Description: "createUser creates a new user.",
	})

	assertEndpoint(t, endpoints[11], collector.ApiEndpoint{
		Name: "update_user", Path: "/users/{id}", Method: "PUT", Protocol: "http",
		Description: "updateUser updates an existing user.",
		Parameters: []collector.ApiParameter{
			{Name: "id", In: "path", Required: true, Type: "text"},
		},
	})
}

func TestParse_TypedRoutes(t *testing.T) {
	endpoints, err := Parse(filepath.Join("testdata", "typed"))
	if err != nil {
		t.Fatalf("typed routes should not error: %v", err)
	}

	endpointMap := make(map[string]*collector.ApiEndpoint)
	for i := range endpoints {
		key := endpoints[i].Method + " " + endpoints[i].Path
		endpointMap[key] = &endpoints[i]
	}

	t.Run("POST /users - Pydantic request/response", func(t *testing.T) {
		ep, ok := endpointMap["POST /users"]
		if !ok {
			t.Fatal("expected POST /users endpoint")
		}
		if ep.RequestBody == nil {
			t.Fatal("expected request body")
		}
		if ep.RequestBody.MediaType != "application/json" {
			t.Errorf("RequestBody.MediaType = %q, want %q", ep.RequestBody.MediaType, "application/json")
		}
		if ep.RequestBody.Body == nil || ep.RequestBody.Body.TypeName != "UserCreate" {
			t.Errorf("RequestBody.Body.TypeName = %q, want %q", ep.RequestBody.Body.TypeName, "UserCreate")
		}
		if ep.Response == nil {
			t.Fatal("expected response body")
		}
		if ep.Response.Body == nil || ep.Response.Body.TypeName != "UserResponse" {
			t.Errorf("Response.Body.TypeName = %q, want %q", ep.Response.Body.TypeName, "UserResponse")
		}
	})

	t.Run("GET /users/{id} - typed path param + response", func(t *testing.T) {
		ep, ok := endpointMap["GET /users/{id}"]
		if !ok {
			t.Fatal("expected GET /users/{id} endpoint")
		}
		if ep.Response == nil {
			t.Fatal("expected response body")
		}
		if ep.Response.Body == nil || ep.Response.Body.TypeName != "UserResponse" {
			t.Errorf("Response.Body.TypeName = %q, want %q", ep.Response.Body.TypeName, "UserResponse")
		}
		hasPathParam := false
		for _, p := range ep.Parameters {
			if p.Name == "id" && p.In == "path" {
				hasPathParam = true
				break
			}
		}
		if !hasPathParam {
			t.Error("expected 'id' path parameter")
		}
	})

	t.Run("PUT /users/{id} - typed path param + body + response", func(t *testing.T) {
		ep, ok := endpointMap["PUT /users/{id}"]
		if !ok {
			t.Fatal("expected PUT /users/{id} endpoint")
		}
		if ep.RequestBody == nil {
			t.Fatal("expected request body")
		}
		if ep.RequestBody.Body == nil || ep.RequestBody.Body.TypeName != "UserCreate" {
			t.Errorf("RequestBody.Body.TypeName = %q, want %q", ep.RequestBody.Body.TypeName, "UserCreate")
		}
		if ep.Response == nil || ep.Response.Body.TypeName != "UserResponse" {
			t.Errorf("Response.Body.TypeName = %q, want %q", ep.Response.Body.TypeName, "UserResponse")
		}
	})

	t.Run("POST /marshmallow/users - Marshmallow request/response", func(t *testing.T) {
		ep, ok := endpointMap["POST /marshmallow/users"]
		if !ok {
			t.Fatal("expected POST /marshmallow/users endpoint")
		}
		if ep.RequestBody == nil {
			t.Fatal("expected request body")
		}
		if ep.RequestBody.Body == nil || ep.RequestBody.Body.TypeName != "UserSchema" {
			t.Errorf("RequestBody.Body.TypeName = %q, want %q", ep.RequestBody.Body.TypeName, "UserSchema")
		}
		if ep.Response == nil {
			t.Fatal("expected response body")
		}
		if ep.Response.Body == nil || ep.Response.Body.TypeName != "UserSchema" {
			t.Errorf("Response.Body.TypeName = %q, want %q", ep.Response.Body.TypeName, "UserSchema")
		}
	})

	t.Run("GET /marshmallow/items - Marshmallow response only", func(t *testing.T) {
		ep, ok := endpointMap["GET /marshmallow/items"]
		if !ok {
			t.Fatal("expected GET /marshmallow/items endpoint")
		}
		if ep.Response == nil {
			t.Fatal("expected response body")
		}
		if ep.Response.Body == nil || ep.Response.Body.TypeName != "ItemSchema" {
			t.Errorf("Response.Body.TypeName = %q, want %q", ep.Response.Body.TypeName, "ItemSchema")
		}
	})

	t.Run("GET /users/{id}/items - list response type", func(t *testing.T) {
		ep, ok := endpointMap["GET /users/{id}/items"]
		if !ok {
			t.Fatal("expected GET /users/{id}/items endpoint")
		}
		if ep.Response == nil {
			t.Fatal("expected response body")
		}
		if ep.Response.Body.Kind != model.KindArray {
			t.Errorf("Response.Body.Kind = %q, want %q", ep.Response.Body.Kind, model.KindArray)
		}
		if ep.Response.Body.Items == nil || ep.Response.Body.Items.TypeName != "ItemResponse" {
			t.Errorf("Response.Body.Items.TypeName = %q, want %q", ep.Response.Body.Items.TypeName, "ItemResponse")
		}
	})

	t.Run("POST /users/batch - list request/response type", func(t *testing.T) {
		ep, ok := endpointMap["POST /users/batch"]
		if !ok {
			t.Fatal("expected POST /users/batch endpoint")
		}
		if ep.RequestBody == nil {
			t.Fatal("expected request body")
		}
		if ep.RequestBody.Body.Kind != model.KindArray {
			t.Errorf("RequestBody.Body.Kind = %q, want %q", ep.RequestBody.Body.Kind, model.KindArray)
		}
		if ep.Response == nil {
			t.Fatal("expected response body")
		}
		if ep.Response.Body.Kind != model.KindArray {
			t.Errorf("Response.Body.Kind = %q, want %q", ep.Response.Body.Kind, model.KindArray)
		}
	})

	t.Run("GET /health - dict response type", func(t *testing.T) {
		ep, ok := endpointMap["GET /health"]
		if !ok {
			t.Fatal("expected GET /health endpoint")
		}
		if ep.Response == nil {
			t.Fatal("expected response body")
		}
		if ep.Response.Body.Kind != model.KindMap {
			t.Errorf("Response.Body.Kind = %q, want %q", ep.Response.Body.Kind, model.KindMap)
		}
	})

	t.Run("POST /no-type - no type annotations", func(t *testing.T) {
		ep, ok := endpointMap["POST /no-type"]
		if !ok {
			t.Fatal("expected POST /no-type endpoint")
		}
		if ep.RequestBody != nil {
			t.Error("expected no request body for untyped endpoint")
		}
		if ep.Response != nil {
			t.Error("expected no response body for untyped endpoint")
		}
	})

	t.Run("GET /optional-response - union response type", func(t *testing.T) {
		ep, ok := endpointMap["GET /optional-response"]
		if !ok {
			t.Fatal("expected GET /optional-response endpoint")
		}
		if ep.Response == nil {
			t.Fatal("expected response body")
		}
		if ep.Response.Body.TypeName != "UserResponse" {
			t.Errorf("Response.Body.TypeName = %q, want %q", ep.Response.Body.TypeName, "UserResponse")
		}
	})

	t.Run("Blueprint endpoints with types", func(t *testing.T) {
		ep, ok := endpointMap["POST /products"]
		if !ok {
			t.Fatal("expected POST /products endpoint")
		}
		if ep.RequestBody == nil {
			t.Fatal("expected request body")
		}
		if ep.RequestBody.Body == nil || ep.RequestBody.Body.TypeName != "ItemCreate" {
			t.Errorf("RequestBody.Body.TypeName = %q, want %q", ep.RequestBody.Body.TypeName, "ItemCreate")
		}
		if ep.Response == nil || ep.Response.Body.TypeName != "ItemResponse" {
			t.Errorf("Response.Body.TypeName = %q, want %q", ep.Response.Body.TypeName, "ItemResponse")
		}
	})
}

func TestConvertFlaskPathToStandard(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"/users", "/users"},
		{"/users/<id>", "/users/{id}"},
		{"/users/<string:id>", "/users/{id}"},
		{"/users/<int:id>", "/users/{id}"},
		{"/users/<float:id>", "/users/{id}"},
		{"/users/<path:id>", "/users/{id}"},
		{"/users/<uuid:id>", "/users/{id}"},
		{"/items/<item_id>/posts/<post_id>", "/items/{item_id}/posts/{post_id}"},
	}

	for _, tt := range tests {
		result := convertFlaskPathToStandard(tt.input)
		if result != tt.expected {
			t.Errorf("convertFlaskPathToStandard(%q) = %q, want %q", tt.input, result, tt.expected)
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
