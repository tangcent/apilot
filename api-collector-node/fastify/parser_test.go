package fastify

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
	if len(endpoints) != 6 {
		t.Fatalf("expected 6 endpoints, got %d", len(endpoints))
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
		Name: "patchUser", Path: "/users/{id}", Method: "PATCH", Protocol: "http",
		Description: "patchUser partially updates a user.",
		Parameters: []collector.ApiParameter{
			{Name: "id", In: "path", Required: true, Type: "text"},
		},
	})

	assertEndpoint(t, endpoints[4], collector.ApiEndpoint{
		Name: "createUser", Path: "/users", Method: "POST", Protocol: "http",
		Description: "createUser creates a new user.",
	})

	assertEndpoint(t, endpoints[5], collector.ApiEndpoint{
		Name: "updateUser", Path: "/users/{id}", Method: "PUT", Protocol: "http",
		Description: "updateUser updates an existing user.",
		Parameters: []collector.ApiParameter{
			{Name: "id", In: "path", Required: true, Type: "text"},
		},
	})
}

func TestParse_RouteObject(t *testing.T) {
	endpoints, err := Parse(filepath.Join("testdata", "route-object"))
	if err != nil {
		t.Fatalf("route-object should not error: %v", err)
	}
	if len(endpoints) != 4 {
		t.Fatalf("expected 4 endpoints, got %d", len(endpoints))
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
		Name: "deleteItem", Path: "/items/{id}", Method: "DELETE", Protocol: "http",
		Description: "deleteItem removes an item by ID.",
		Parameters: []collector.ApiParameter{
			{Name: "id", In: "path", Required: true, Type: "text"},
		},
	})

	assertEndpoint(t, endpoints[3], collector.ApiEndpoint{
		Name: "getItem", Path: "/items/{id}", Method: "GET", Protocol: "http",
		Description: "getItem returns a single item by ID.",
		Parameters: []collector.ApiParameter{
			{Name: "id", In: "path", Required: true, Type: "text"},
		},
	})
}

func TestParse_SchemaRoutes(t *testing.T) {
	endpoints, err := Parse(filepath.Join("testdata", "schema"))
	if err != nil {
		t.Fatalf("schema routes should not error: %v", err)
	}
	if len(endpoints) != 3 {
		t.Fatalf("expected 3 endpoints, got %d", len(endpoints))
	}

	sort.Slice(endpoints, func(i, j int) bool {
		if endpoints[i].Method != endpoints[j].Method {
			return endpoints[i].Method < endpoints[j].Method
		}
		return endpoints[i].Path < endpoints[j].Path
	})

	getUsers := endpoints[0]
	if getUsers.Method != "GET" || getUsers.Path != "/users" {
		t.Fatalf("expected GET /users, got %s %s", getUsers.Method, getUsers.Path)
	}

	if getUsers.Response == nil || getUsers.Response.Body == nil {
		t.Fatalf("GET /users should have response with schema")
	} else {
		body := getUsers.Response.Body
		if !body.IsObject() {
			t.Errorf("GET /users response should be object, got %s", body.Kind)
		}
		if _, ok := body.Fields["users"]; !ok {
			t.Errorf("GET /users response should have 'users' field")
		}
		if _, ok := body.Fields["total"]; !ok {
			t.Errorf("GET /users response should have 'total' field")
		}
	}

	getUserById := endpoints[1]
	if getUserById.Method != "GET" || getUserById.Path != "/users/{id}" {
		t.Fatalf("expected GET /users/{id}, got %s %s", getUserById.Method, getUserById.Path)
	}

	if getUserById.Response == nil || getUserById.Response.Body == nil {
		t.Fatalf("GET /users/{id} should have response with schema")
	} else {
		body := getUserById.Response.Body
		if !body.IsObject() {
			t.Errorf("GET /users/{id} response should be object, got %s", body.Kind)
		}
		if _, ok := body.Fields["id"]; !ok {
			t.Errorf("GET /users/{id} response should have 'id' field")
		}
	}

	postUsers := endpoints[2]
	if postUsers.Method != "POST" || postUsers.Path != "/users" {
		t.Fatalf("expected POST /users, got %s %s", postUsers.Method, postUsers.Path)
	}

	if postUsers.RequestBody == nil || postUsers.RequestBody.Body == nil {
		t.Fatalf("POST /users should have request body with schema")
	} else {
		body := postUsers.RequestBody.Body
		if !body.IsObject() {
			t.Errorf("POST /users request body should be object, got %s", body.Kind)
		}
		if _, ok := body.Fields["name"]; !ok {
			t.Errorf("POST /users request body should have 'name' field")
		}
		if _, ok := body.Fields["email"]; !ok {
			t.Errorf("POST /users request body should have 'email' field")
		}
		nameField := body.Fields["name"]
		if !nameField.Required {
			t.Errorf("POST /users request body 'name' should be required")
		}
	}

	if postUsers.Response == nil || postUsers.Response.Body == nil {
		t.Fatalf("POST /users should have response with schema")
	} else {
		body := postUsers.Response.Body
		if !body.IsObject() {
			t.Errorf("POST /users response should be object, got %s", body.Kind)
		}
		if _, ok := body.Fields["id"]; !ok {
			t.Errorf("POST /users response should have 'id' field")
		}
	}
}

func TestParse_TypedRoutes(t *testing.T) {
	endpoints, err := Parse(filepath.Join("testdata", "typed"))
	if err != nil {
		t.Fatalf("typed routes should not error: %v", err)
	}
	if len(endpoints) == 0 {
		t.Fatalf("expected endpoints for typed routes, got 0")
	}

	sort.Slice(endpoints, func(i, j int) bool {
		if endpoints[i].Method != endpoints[j].Method {
			return endpoints[i].Method < endpoints[j].Method
		}
		return endpoints[i].Path < endpoints[j].Path
	})

	var postUsers, getUserById, putUserById []collector.ApiEndpoint
	for _, ep := range endpoints {
		switch {
		case ep.Method == "POST" && ep.Path == "/users":
			postUsers = append(postUsers, ep)
		case ep.Method == "GET" && ep.Path == "/users/{id}":
			getUserById = append(getUserById, ep)
		case ep.Method == "PUT" && ep.Path == "/users/{id}":
			putUserById = append(putUserById, ep)
		}
	}

	if len(postUsers) > 0 {
		ep := postUsers[0]
		if ep.RequestBody == nil || ep.RequestBody.Body == nil {
			t.Errorf("POST /users should have request body with type info")
		} else {
			body := ep.RequestBody.Body
			if !body.IsObject() {
				t.Errorf("POST /users request body should be object, got %s", body.Kind)
			}
			if _, ok := body.Fields["name"]; !ok {
				t.Errorf("POST /users request body should have 'name' field")
			}
			if _, ok := body.Fields["email"]; !ok {
				t.Errorf("POST /users request body should have 'email' field")
			}
		}
	}

	if len(putUserById) > 0 {
		ep := putUserById[0]
		if ep.RequestBody == nil || ep.RequestBody.Body == nil {
			t.Errorf("PUT /users/{id} should have request body with type info")
		} else {
			body := ep.RequestBody.Body
			if !body.IsObject() {
				t.Errorf("PUT /users/{id} request body should be object, got %s", body.Kind)
			}
		}
	}
}

func TestConvertPath(t *testing.T) {
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
		result := convertPath(tt.input)
		if result != tt.expected {
			t.Errorf("convertPath(%q) = %q, want %q", tt.input, result, tt.expected)
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

func TestParseFastifyRequestGenerics(t *testing.T) {
	tests := []struct {
		input       string
		wantBody    string
		wantQuery   string
		wantParams  string
	}{
		{
			"FastifyRequest<{ Body: CreateUserRequest }>",
			"CreateUserRequest", "", "",
		},
		{
			"FastifyRequest<{ Params: { id: string } }>",
			"", "", "{ id: string }",
		},
		{
			"FastifyRequest<{ Params: { id: string }; Body: CreateUserRequest }>",
			"CreateUserRequest", "", "{ id: string }",
		},
		{
			"FastifyRequest<{ Body: CreateUserRequest; Querystring: UserQuery }>",
			"CreateUserRequest", "UserQuery", "",
		},
		{
			"FastifyRequest",
			"", "", "",
		},
		{
			"FastifyRequest<{ Body: any }>",
			"", "", "",
		},
		{
			"FastifyRequest<RawServerDefault, CreateUserRequest>",
			"CreateUserRequest", "", "",
		},
		{
			"FastifyRequest<RawServerDefault, CreateUserRequest, UserQuery>",
			"CreateUserRequest", "UserQuery", "",
		},
	}

	for _, tt := range tests {
		reqBody, query, params := parseFastifyRequestGenerics(tt.input)
		if reqBody != tt.wantBody {
			t.Errorf("parseFastifyRequestGenerics(%q) reqBody = %q, want %q", tt.input, reqBody, tt.wantBody)
		}
		if query != tt.wantQuery {
			t.Errorf("parseFastifyRequestGenerics(%q) query = %q, want %q", tt.input, query, tt.wantQuery)
		}
		if params != tt.wantParams {
			t.Errorf("parseFastifyRequestGenerics(%q) params = %q, want %q", tt.input, params, tt.wantParams)
		}
	}
}

func TestParseFastifyReplyGenerics(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"FastifyReply<UserResponse>", "UserResponse"},
		{"FastifyReply", ""},
		{"FastifyReply<any>", ""},
	}
	for _, tt := range tests {
		result := parseFastifyReplyGenerics(tt.input)
		if result != tt.expected {
			t.Errorf("parseFastifyReplyGenerics(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestMapJSONSchemaType(t *testing.T) {
	tests := []struct {
		input    string
		kind     string
		typeName string
	}{
		{"string", "single", "string"},
		{"integer", "single", "int"},
		{"number", "single", "double"},
		{"boolean", "single", "boolean"},
		{"null", "single", "null"},
		{"array", "array", "array"},
		{"object", "object", "object"},
	}

	for _, tt := range tests {
		result := mapJSONSchemaType(tt.input)
		if string(result.Kind) != tt.kind {
			t.Errorf("mapJSONSchemaType(%q).Kind = %q, want %q", tt.input, result.Kind, tt.kind)
		}
		if result.TypeName != tt.typeName {
			t.Errorf("mapJSONSchemaType(%q).TypeName = %q, want %q", tt.input, result.TypeName, tt.typeName)
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
