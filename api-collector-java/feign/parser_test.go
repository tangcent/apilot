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

	ep1 := client.Endpoints[1]
	if ep1.Method != DELETE {
		t.Errorf("Expected DELETE, got %s", ep1.Method)
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
