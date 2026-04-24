package feign

import (
	"testing"

	"github.com/tangcent/apilot/api-collector-java/parser"
)

func TestParser_SpringCloudFeignClient(t *testing.T) {
	results := []parser.ParseResult{
		{
			FilePath: "UserClient.java",
			Classes: []parser.Class{
				{
					Name:        "UserClient",
					Package:     "com.example.client",
					IsInterface: true,
					Annotations: []parser.Annotation{
						{Name: "FeignClient", Params: map[string]string{"name": "user-service"}},
					},
					Methods: []parser.Method{
						{
							Name: "getUser",
							Annotations: []parser.Annotation{
								{Name: "GetMapping", Params: map[string]string{"value": "/api/users/{id}"}},
							},
							Parameters: []parser.Parameter{
								{
									Name: "id",
									Type: "Long",
									Annotations: []parser.Annotation{
										{Name: "PathVariable"},
									},
								},
							},
							ReturnType: "User",
						},
						{
							Name: "createUser",
							Annotations: []parser.Annotation{
								{Name: "PostMapping", Params: map[string]string{"value": "/api/users"}},
							},
							Parameters: []parser.Parameter{
								{
									Name: "user",
									Type: "User",
									Annotations: []parser.Annotation{
										{Name: "RequestBody"},
									},
								},
							},
							ReturnType: "User",
						},
					},
				},
			},
		},
	}

	p := NewParser()
	clients := p.ExtractClients(results)

	if len(clients) != 1 {
		t.Fatalf("Expected 1 client, got %d", len(clients))
	}

	client := clients[0]
	if client.Name != "UserClient" {
		t.Errorf("Expected 'UserClient', got '%s'", client.Name)
	}
	if client.ServiceName != "user-service" {
		t.Errorf("Expected service name 'user-service', got '%s'", client.ServiceName)
	}
	if len(client.Endpoints) != 2 {
		t.Fatalf("Expected 2 endpoints, got %d", len(client.Endpoints))
	}

	ep0 := client.Endpoints[0]
	if ep0.Method != GET {
		t.Errorf("Expected GET, got %s", ep0.Method)
	}
	if ep0.Path != "/api/users/{id}" {
		t.Errorf("Expected '/api/users/{id}', got '%s'", ep0.Path)
	}
	if len(ep0.Parameters) != 1 || ep0.Parameters[0].ParamType != "path" {
		t.Errorf("Expected path parameter, got %v", ep0.Parameters)
	}

	ep1 := client.Endpoints[1]
	if ep1.Method != POST {
		t.Errorf("Expected POST, got %s", ep1.Method)
	}
	if len(ep1.Parameters) != 1 || ep1.Parameters[0].ParamType != "body" {
		t.Errorf("Expected body parameter, got %v", ep1.Parameters)
	}
}

func TestParser_NetflixFeignRequestLine(t *testing.T) {
	results := []parser.ParseResult{
		{
			FilePath: "OrderClient.java",
			Classes: []parser.Class{
				{
					Name:        "OrderClient",
					Package:     "com.example.client",
					IsInterface: true,
					Annotations: []parser.Annotation{
						{Name: "FeignClient", Params: map[string]string{"value": "order-service", "url": "http://order-service"}},
					},
					Methods: []parser.Method{
						{
							Name: "getOrder",
							Annotations: []parser.Annotation{
								{Name: "RequestLine", Params: map[string]string{"value": "GET /orders/{id}"}},
							},
							Parameters: []parser.Parameter{
								{
									Name: "id",
									Type: "Long",
									Annotations: []parser.Annotation{
										{Name: "Param", Params: map[string]string{"value": "id"}},
									},
								},
							},
							ReturnType: "Order",
						},
						{
							Name: "deleteOrder",
							Annotations: []parser.Annotation{
								{Name: "RequestLine", Params: map[string]string{"value": "DELETE /orders/{id}"}},
							},
							Parameters: []parser.Parameter{
								{
									Name: "id",
									Type: "Long",
									Annotations: []parser.Annotation{
										{Name: "Param", Params: map[string]string{"value": "id"}},
									},
								},
							},
							ReturnType: "void",
						},
					},
				},
			},
		},
	}

	p := NewParser()
	clients := p.ExtractClients(results)

	if len(clients) != 1 {
		t.Fatalf("Expected 1 client, got %d", len(clients))
	}

	client := clients[0]
	if client.ServiceName != "order-service" {
		t.Errorf("Expected 'order-service', got '%s'", client.ServiceName)
	}
	if client.URL != "http://order-service" {
		t.Errorf("Expected URL 'http://order-service', got '%s'", client.URL)
	}
	if len(client.Endpoints) != 2 {
		t.Fatalf("Expected 2 endpoints, got %d", len(client.Endpoints))
	}

	ep0 := client.Endpoints[0]
	if ep0.Method != GET {
		t.Errorf("Expected GET, got %s", ep0.Method)
	}
	if ep0.Path != "/orders/{id}" {
		t.Errorf("Expected '/orders/{id}', got '%s'", ep0.Path)
	}
	if len(ep0.Parameters) != 1 || ep0.Parameters[0].ParamType != "path" {
		t.Errorf("Expected path parameter, got %v", ep0.Parameters)
	}

	ep1 := client.Endpoints[1]
	if ep1.Method != DELETE {
		t.Errorf("Expected DELETE, got %s", ep1.Method)
	}
	if len(ep1.Parameters) != 1 || ep1.Parameters[0].ParamType != "path" {
		t.Errorf("Expected path parameter, got %v", ep1.Parameters)
	}
}

func TestParser_NetflixFeignWithoutFeignClientAnnotation(t *testing.T) {
	results := []parser.ParseResult{
		{
			FilePath: "UserClient.java",
			Classes: []parser.Class{
				{
					Name:        "UserClient",
					Package:     "com.example",
					IsInterface: true,
					Annotations: []parser.Annotation{},
					Methods: []parser.Method{
						{
							Name: "listUsers",
							Annotations: []parser.Annotation{
								{Name: "RequestLine", Params: map[string]string{"value": "GET /users?name={name}&role={role}"}},
							},
							Parameters: []parser.Parameter{
								{
									Name: "name",
									Type: "String",
									Annotations: []parser.Annotation{
										{Name: "Param", Params: map[string]string{"value": "name"}},
									},
								},
								{
									Name: "role",
									Type: "String",
									Annotations: []parser.Annotation{
										{Name: "Param", Params: map[string]string{"value": "role"}},
									},
								},
							},
							ReturnType: "String",
						},
						{
							Name: "createUser",
							Annotations: []parser.Annotation{
								{Name: "RequestLine", Params: map[string]string{"value": "POST /users"}},
							},
							Parameters: []parser.Parameter{
								{
									Name: "req",
									Type: "CreateUserReq",
									Annotations: []parser.Annotation{},
								},
							},
							ReturnType: "String",
						},
						{
							Name: "getUser",
							Annotations: []parser.Annotation{
								{Name: "RequestLine", Params: map[string]string{"value": "GET /users/{id}"}},
							},
							Parameters: []parser.Parameter{
								{
									Name: "id",
									Type: "String",
									Annotations: []parser.Annotation{
										{Name: "Param", Params: map[string]string{"value": "id"}},
									},
								},
							},
							ReturnType: "String",
						},
					},
				},
			},
		},
	}

	p := NewParser()
	clients := p.ExtractClients(results)

	if len(clients) != 1 {
		t.Fatalf("Expected 1 client, got %d", len(clients))
	}

	client := clients[0]
	if client.Name != "UserClient" {
		t.Errorf("Expected 'UserClient', got '%s'", client.Name)
	}
	if client.ServiceName != "" {
		t.Errorf("Expected empty service name, got '%s'", client.ServiceName)
	}
	if len(client.Endpoints) != 3 {
		t.Fatalf("Expected 3 endpoints, got %d", len(client.Endpoints))
	}

	ep0 := client.Endpoints[0]
	if ep0.Method != GET {
		t.Errorf("Expected GET, got %s", ep0.Method)
	}
	if len(ep0.Parameters) != 2 {
		t.Errorf("Expected 2 query params, got %d", len(ep0.Parameters))
	} else {
		if ep0.Parameters[0].ParamType != "query" {
			t.Errorf("Expected query param, got %s", ep0.Parameters[0].ParamType)
		}
		if ep0.Parameters[1].ParamType != "query" {
			t.Errorf("Expected query param, got %s", ep0.Parameters[1].ParamType)
		}
	}

	ep1 := client.Endpoints[1]
	if ep1.Method != POST {
		t.Errorf("Expected POST, got %s", ep1.Method)
	}
	if len(ep1.Parameters) != 1 || ep1.Parameters[0].ParamType != "body" {
		t.Errorf("Expected body parameter, got %v", ep1.Parameters)
	}

	ep2 := client.Endpoints[2]
	if ep2.Method != GET {
		t.Errorf("Expected GET, got %s", ep2.Method)
	}
	if len(ep2.Parameters) != 1 || ep2.Parameters[0].ParamType != "path" {
		t.Errorf("Expected path parameter, got %v", ep2.Parameters)
	}
}

func TestParser_RequestLineParsing(t *testing.T) {
	tests := []struct {
		value          string
		expectedMethod HTTPMethod
		expectedPath   string
	}{
		{"GET /users", GET, "/users"},
		{"POST /users", POST, "/users"},
		{"PUT /users/{id}", PUT, "/users/{id}"},
		{"DELETE /users/{id}", DELETE, "/users/{id}"},
		{"PATCH /users/{id}", PATCH, "/users/{id}"},
		{"UNKNOWN", GET, ""},  // malformed: no space → empty path
	}

	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			method, path := parseRequestLine(tt.value)
			if method != tt.expectedMethod {
				t.Errorf("Expected method %s, got %s", tt.expectedMethod, method)
			}
			if path != tt.expectedPath {
				t.Errorf("Expected path '%s', got '%s'", tt.expectedPath, path)
			}
		})
	}
}

func TestParser_NonFeignInterface(t *testing.T) {
	results := []parser.ParseResult{
		{
			Classes: []parser.Class{
				{
					Name:        "UserRepository",
					IsInterface: true,
					Annotations: []parser.Annotation{
						{Name: "Repository"},
					},
				},
			},
		},
	}

	p := NewParser()
	clients := p.ExtractClients(results)

	if len(clients) != 0 {
		t.Errorf("Expected 0 clients, got %d", len(clients))
	}
}

func TestParser_TypeResolution_SpringStyle(t *testing.T) {
	results := []parser.ParseResult{
		{
			Classes: []parser.Class{
				{
					Name: "User",
					Fields: []parser.Field{
						{Name: "name", Type: "String"},
						{Name: "email", Type: "String"},
					},
				},
				{
					Name:        "UserClient",
					IsInterface: true,
					Annotations: []parser.Annotation{
						{Name: "FeignClient", Params: map[string]string{"name": "user-service"}},
					},
					Methods: []parser.Method{
						{
							Name: "getUser",
							Annotations: []parser.Annotation{
								{Name: "GetMapping", Params: map[string]string{"value": "/users/{id}"}},
							},
							Parameters: []parser.Parameter{
								{Name: "id", Type: "Long", Annotations: []parser.Annotation{{Name: "PathVariable"}}},
							},
							ReturnType: "User",
						},
						{
							Name: "createUser",
							Annotations: []parser.Annotation{
								{Name: "PostMapping", Params: map[string]string{"value": "/users"}},
							},
							Parameters: []parser.Parameter{
								{Name: "user", Type: "User", Annotations: []parser.Annotation{{Name: "RequestBody"}}},
							},
							ReturnType: "User",
						},
						{
							Name: "listUsers",
							Annotations: []parser.Annotation{
								{Name: "GetMapping", Params: map[string]string{"value": "/users"}},
							},
							Parameters:  []parser.Parameter{},
							ReturnType: "List<User>",
						},
						{
							Name: "deleteUser",
							Annotations: []parser.Annotation{
								{Name: "DeleteMapping", Params: map[string]string{"value": "/users/{id}"}},
							},
							Parameters: []parser.Parameter{
								{Name: "id", Type: "Long", Annotations: []parser.Annotation{{Name: "PathVariable"}}},
							},
							ReturnType: "void",
						},
					},
				},
			},
		},
	}

	p := NewParser()
	clients := p.ExtractClients(results)

	if len(clients) != 1 {
		t.Fatalf("Expected 1 client, got %d", len(clients))
	}

	client := clients[0]
	if len(client.Endpoints) != 4 {
		t.Fatalf("Expected 4 endpoints, got %d", len(client.Endpoints))
	}

	// getUser returns User
	ep0 := client.Endpoints[0]
	if ep0.ResponseSchema == nil {
		t.Fatal("Expected ResponseSchema for getUser")
	}
	if !ep0.ResponseSchema.IsObject() {
		t.Errorf("Expected object model for User, got kind=%s", ep0.ResponseSchema.Kind)
	}
	if _, ok := ep0.ResponseSchema.Fields["name"]; !ok {
		t.Error("Expected 'name' field in User response schema")
	}

	// createUser has User request body and returns User
	ep1 := client.Endpoints[1]
	if ep1.RequestBodySchema == nil {
		t.Fatal("Expected RequestBodySchema for createUser")
	}
	if !ep1.RequestBodySchema.IsObject() {
		t.Errorf("Expected object model for User request, got kind=%s", ep1.RequestBodySchema.Kind)
	}
	if ep1.ResponseSchema == nil {
		t.Fatal("Expected ResponseSchema for createUser")
	}

	// listUsers returns List<User>
	ep2 := client.Endpoints[2]
	if ep2.ResponseSchema == nil {
		t.Fatal("Expected ResponseSchema for listUsers")
	}
	if !ep2.ResponseSchema.IsArray() {
		t.Errorf("Expected array model for List<User>, got kind=%s", ep2.ResponseSchema.Kind)
	}
	if ep2.ResponseSchema.Items == nil || !ep2.ResponseSchema.Items.IsObject() {
		t.Error("Expected array items to be User object")
	}

	// deleteUser returns void
	ep3 := client.Endpoints[3]
	if ep3.ResponseSchema != nil {
		t.Error("void return should not produce a ResponseSchema")
	}
}

func TestParser_TypeResolution_RequestLineStyle(t *testing.T) {
	results := []parser.ParseResult{
		{
			Classes: []parser.Class{
				{
					Name: "Order",
					Fields: []parser.Field{
						{Name: "orderId", Type: "String"},
						{Name: "amount", Type: "double"},
					},
				},
				{
					Name:        "OrderClient",
					IsInterface: true,
					Annotations: []parser.Annotation{
						{Name: "FeignClient", Params: map[string]string{"name": "order-service"}},
					},
					Methods: []parser.Method{
						{
							Name: "createOrder",
							Annotations: []parser.Annotation{
								{Name: "RequestLine", Params: map[string]string{"value": "POST /orders"}},
							},
							Parameters: []parser.Parameter{
								{Name: "order", Type: "Order", Annotations: []parser.Annotation{}},
							},
							ReturnType: "Order",
						},
					},
				},
			},
		},
	}

	p := NewParser()
	clients := p.ExtractClients(results)

	if len(clients) != 1 || len(clients[0].Endpoints) != 1 {
		t.Fatal("Expected 1 client with 1 endpoint")
	}

	ep := clients[0].Endpoints[0]
	if ep.RequestBodySchema == nil {
		t.Fatal("Expected RequestBodySchema for createOrder")
	}
	if !ep.RequestBodySchema.IsObject() {
		t.Errorf("Expected object model, got kind=%s", ep.RequestBodySchema.Kind)
	}
	if _, ok := ep.RequestBodySchema.Fields["orderId"]; !ok {
		t.Error("Expected 'orderId' field in Order request body")
	}

	if ep.ResponseSchema == nil {
		t.Fatal("Expected ResponseSchema for createOrder")
	}
	if !ep.ResponseSchema.IsObject() {
		t.Errorf("Expected object model, got kind=%s", ep.ResponseSchema.Kind)
	}
}

func TestParser_SpringQueryMap(t *testing.T) {
	results := []parser.ParseResult{
		{
			Classes: []parser.Class{
				{
					Name: "SearchFilter",
					Fields: []parser.Field{
						{Name: "keyword", Type: "String"},
						{Name: "page", Type: "int"},
					},
				},
				{
					Name:        "SearchClient",
					IsInterface: true,
					Annotations: []parser.Annotation{
						{Name: "FeignClient", Params: map[string]string{"name": "search-service"}},
					},
					Methods: []parser.Method{
						{
							Name: "search",
							Annotations: []parser.Annotation{
								{Name: "GetMapping", Params: map[string]string{"value": "/search"}},
							},
							Parameters: []parser.Parameter{
								{Name: "filter", Type: "SearchFilter", Annotations: []parser.Annotation{{Name: "SpringQueryMap"}}},
							},
							ReturnType: "List<String>",
						},
					},
				},
			},
		},
	}

	p := NewParser()
	clients := p.ExtractClients(results)

	if len(clients) != 1 || len(clients[0].Endpoints) != 1 {
		t.Fatal("Expected 1 client with 1 endpoint")
	}

	ep := clients[0].Endpoints[0]
	if len(ep.Parameters) != 1 {
		t.Fatalf("Expected 1 parameter, got %d", len(ep.Parameters))
	}
	if ep.Parameters[0].ParamType != "body" {
		t.Errorf("Expected body param for @SpringQueryMap, got '%s'", ep.Parameters[0].ParamType)
	}
	if ep.RequestBodySchema == nil {
		t.Fatal("Expected RequestBodySchema for @SpringQueryMap parameter")
	}
	if !ep.RequestBodySchema.IsObject() {
		t.Errorf("Expected object model for SearchFilter, got kind=%s", ep.RequestBodySchema.Kind)
	}
	if _, ok := ep.RequestBodySchema.Fields["keyword"]; !ok {
		t.Error("Expected 'keyword' field in SearchFilter schema")
	}
}

func TestParser_GenericReturnType(t *testing.T) {
	results := []parser.ParseResult{
		{
			Classes: []parser.Class{
				{
					Name:           "Result",
					TypeParameters: []string{"T"},
					Fields: []parser.Field{
						{Name: "code", Type: "int"},
						{Name: "message", Type: "String"},
						{Name: "data", Type: "T"},
					},
				},
				{
					Name: "User",
					Fields: []parser.Field{
						{Name: "name", Type: "String"},
					},
				},
				{
					Name:        "UserClient",
					IsInterface: true,
					Annotations: []parser.Annotation{
						{Name: "FeignClient", Params: map[string]string{"name": "user-service"}},
					},
					Methods: []parser.Method{
						{
							Name: "getUser",
							Annotations: []parser.Annotation{
								{Name: "GetMapping", Params: map[string]string{"value": "/users/{id}"}},
							},
							Parameters: []parser.Parameter{
								{Name: "id", Type: "Long", Annotations: []parser.Annotation{{Name: "PathVariable"}}},
							},
							ReturnType: "Result<User>",
						},
					},
				},
			},
		},
	}

	p := NewParser()
	clients := p.ExtractClients(results)

	if len(clients) != 1 || len(clients[0].Endpoints) != 1 {
		t.Fatal("Expected 1 client with 1 endpoint")
	}

	ep := clients[0].Endpoints[0]
	if ep.ResponseSchema == nil {
		t.Fatal("Expected ResponseSchema for getUser")
	}
	if !ep.ResponseSchema.IsObject() {
		t.Fatalf("Expected object model for Result<User>, got kind=%s", ep.ResponseSchema.Kind)
	}

	dataField, ok := ep.ResponseSchema.Fields["data"]
	if !ok {
		t.Fatal("Expected 'data' field in Result<User>")
	}
	if dataField.Model == nil || !dataField.Model.IsObject() {
		t.Error("Expected 'data' field to be resolved as User object")
	}
	if dataField.Model != nil {
		if _, ok := dataField.Model.Fields["name"]; !ok {
			t.Error("Expected 'name' field in resolved User data")
		}
	}
}

func TestParser_InheritedEndpoints(t *testing.T) {
	results := []parser.ParseResult{
		{
			Classes: []parser.Class{
				{
					Name: "User",
					Fields: []parser.Field{
						{Name: "name", Type: "String"},
					},
				},
				{
					Name:           "BaseCrudClient",
					IsInterface:    true,
					TypeParameters: []string{"T"},
					Methods: []parser.Method{
						{
							Name: "getById",
							Annotations: []parser.Annotation{
								{Name: "GetMapping", Params: map[string]string{"value": "/{id}"}},
							},
							Parameters: []parser.Parameter{
								{Name: "id", Type: "Long", Annotations: []parser.Annotation{{Name: "PathVariable"}}},
							},
							ReturnType: "T",
						},
						{
							Name: "create",
							Annotations: []parser.Annotation{
								{Name: "PostMapping"},
							},
							Parameters: []parser.Parameter{
								{Name: "entity", Type: "T", Annotations: []parser.Annotation{{Name: "RequestBody"}}},
							},
							ReturnType: "T",
						},
					},
				},
				{
					Name:        "UserClient",
					IsInterface: true,
					Annotations: []parser.Annotation{
						{Name: "FeignClient", Params: map[string]string{"name": "user-service"}},
					},
					SuperClass:         "BaseCrudClient",
					SuperClassTypeArgs: []string{"User"},
					Methods:            []parser.Method{},
				},
			},
		},
	}

	p := NewParser()
	clients := p.ExtractClients(results)

	if len(clients) != 1 {
		t.Fatalf("Expected 1 client, got %d", len(clients))
	}

	client := clients[0]
	if len(client.Endpoints) != 2 {
		t.Fatalf("Expected 2 inherited endpoints, got %d", len(client.Endpoints))
	}

	methodNames := make(map[string]*Endpoint)
	for i := range client.Endpoints {
		methodNames[client.Endpoints[i].MethodName] = &client.Endpoints[i]
	}

	getEp := methodNames["getById"]
	if getEp == nil {
		t.Fatal("Expected inherited 'getById' endpoint")
	}
	if getEp.ResponseSchema == nil {
		t.Fatal("Expected ResponseSchema for inherited getById")
	}
	if !getEp.ResponseSchema.IsObject() {
		t.Errorf("Expected object model for User (T=User), got kind=%s", getEp.ResponseSchema.Kind)
	}
	if _, ok := getEp.ResponseSchema.Fields["name"]; !ok {
		t.Error("Expected 'name' field in resolved User response")
	}

	createEp := methodNames["create"]
	if createEp == nil {
		t.Fatal("Expected inherited 'create' endpoint")
	}
	if createEp.RequestBodySchema == nil {
		t.Fatal("Expected RequestBodySchema for inherited create")
	}
	if !createEp.RequestBodySchema.IsObject() {
		t.Errorf("Expected object model for User (T=User), got kind=%s", createEp.RequestBodySchema.Kind)
	}
}

func TestParser_ResponseEntityUnwrapping(t *testing.T) {
	results := []parser.ParseResult{
		{
			Classes: []parser.Class{
				{
					Name:        "TestClient",
					IsInterface: true,
					Annotations: []parser.Annotation{
						{Name: "FeignClient", Params: map[string]string{"name": "test-service"}},
					},
					Methods: []parser.Method{
						{
							Name: "get",
							Annotations: []parser.Annotation{
								{Name: "GetMapping", Params: map[string]string{"value": "/test"}},
							},
							ReturnType: "ResponseEntity<String>",
						},
					},
				},
			},
		},
	}

	p := NewParser()
	clients := p.ExtractClients(results)

	if len(clients) != 1 || len(clients[0].Endpoints) != 1 {
		t.Fatal("Expected 1 client with 1 endpoint")
	}

	ep := clients[0].Endpoints[0]
	if ep.ResponseSchema == nil {
		t.Fatal("Expected ResponseSchema")
	}
	if ep.ResponseSchema.TypeName != "string" {
		t.Errorf("Expected unwrapped String type, got '%s'", ep.ResponseSchema.TypeName)
	}
}
