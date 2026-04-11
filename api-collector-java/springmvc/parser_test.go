package springmvc

import (
	"testing"

	"github.com/tangcent/apilot/api-collector-java/parser"
)

func TestParser_ExtractControllers(t *testing.T) {
	// Create test parse results
	results := []parser.ParseResult{
		{
			FilePath: "UserController.java",
			Classes: []parser.Class{
				{
					Name:    "UserController",
					Package: "com.example.api",
					Annotations: []parser.Annotation{
						{Name: "RestController"},
						{Name: "RequestMapping", Params: map[string]string{"value": "/api/users"}},
					},
					Methods: []parser.Method{
						{
							Name: "getUser",
							Annotations: []parser.Annotation{
								{Name: "GetMapping", Params: map[string]string{"value": "/{id}"}},
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
							ReturnType: "ResponseEntity<User>",
						},
						{
							Name: "listUsers",
							Annotations: []parser.Annotation{
								{Name: "GetMapping"},
							},
							Parameters: []parser.Parameter{
								{
									Name: "page",
									Type: "int",
									Annotations: []parser.Annotation{
										{Name: "RequestParam", Params: map[string]string{"defaultValue": "0"}},
									},
								},
								{
									Name: "size",
									Type: "int",
									Annotations: []parser.Annotation{
										{Name: "RequestParam", Params: map[string]string{"defaultValue": "10"}},
									},
								},
							},
							ReturnType: "ResponseEntity<List<User>>",
						},
						{
							Name: "createUser",
							Annotations: []parser.Annotation{
								{Name: "PostMapping"},
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
							ReturnType: "ResponseEntity<User>",
						},
					},
				},
			},
		},
	}

	p := NewParser()
	controllers := p.ExtractControllers(results)

	if len(controllers) != 1 {
		t.Fatalf("Expected 1 controller, got %d", len(controllers))
	}

	controller := controllers[0]

	// Verify controller properties
	if controller.Name != "UserController" {
		t.Errorf("Expected controller name 'UserController', got '%s'", controller.Name)
	}

	if controller.Package != "com.example.api" {
		t.Errorf("Expected package 'com.example.api', got '%s'", controller.Package)
	}

	if controller.BasePath != "/api/users" {
		t.Errorf("Expected base path '/api/users', got '%s'", controller.BasePath)
	}

	if len(controller.Endpoints) != 3 {
		t.Fatalf("Expected 3 endpoints, got %d", len(controller.Endpoints))
	}

	// Verify getUser endpoint
	getUser := controller.Endpoints[0]
	if getUser.Path != "/api/users/{id}" {
		t.Errorf("Expected path '/api/users/{id}', got '%s'", getUser.Path)
	}
	if getUser.Method != GET {
		t.Errorf("Expected method GET, got %s", getUser.Method)
	}
	if len(getUser.Parameters) != 1 {
		t.Errorf("Expected 1 parameter, got %d", len(getUser.Parameters))
	}
	if getUser.Parameters[0].ParamType != "path" {
		t.Errorf("Expected param type 'path', got '%s'", getUser.Parameters[0].ParamType)
	}

	// Verify listUsers endpoint
	listUsers := controller.Endpoints[1]
	if listUsers.Path != "/api/users" {
		t.Errorf("Expected path '/api/users', got '%s'", listUsers.Path)
	}
	if listUsers.Method != GET {
		t.Errorf("Expected method GET, got %s", listUsers.Method)
	}
	if len(listUsers.Parameters) != 2 {
		t.Errorf("Expected 2 parameters, got %d", len(listUsers.Parameters))
	}
	if listUsers.Parameters[0].DefaultValue != "0" {
		t.Errorf("Expected default value '0', got '%s'", listUsers.Parameters[0].DefaultValue)
	}
	if listUsers.Parameters[0].Required {
		t.Error("Expected parameter to be optional (has default value)")
	}

	// Verify createUser endpoint
	createUser := controller.Endpoints[2]
	if createUser.Method != POST {
		t.Errorf("Expected method POST, got %s", createUser.Method)
	}
	if len(createUser.Parameters) != 1 {
		t.Errorf("Expected 1 parameter, got %d", len(createUser.Parameters))
	}
	if createUser.Parameters[0].ParamType != "body" {
		t.Errorf("Expected param type 'body', got '%s'", createUser.Parameters[0].ParamType)
	}
}

func TestParser_HTTPMethods(t *testing.T) {
	tests := []struct {
		annotation string
		expected   HTTPMethod
	}{
		{"GetMapping", GET},
		{"PostMapping", POST},
		{"PutMapping", PUT},
		{"DeleteMapping", DELETE},
		{"PatchMapping", PATCH},
	}

	for _, tt := range tests {
		t.Run(tt.annotation, func(t *testing.T) {
			results := []parser.ParseResult{
				{
					Classes: []parser.Class{
						{
							Name: "TestController",
							Annotations: []parser.Annotation{
								{Name: "RestController"},
							},
							Methods: []parser.Method{
								{
									Name: "testMethod",
									Annotations: []parser.Annotation{
										{Name: tt.annotation, Params: map[string]string{"value": "/test"}},
									},
								},
							},
						},
					},
				},
			}

			p := NewParser()
			controllers := p.ExtractControllers(results)

			if len(controllers) != 1 || len(controllers[0].Endpoints) != 1 {
				t.Fatal("Failed to extract endpoint")
			}

			if controllers[0].Endpoints[0].Method != tt.expected {
				t.Errorf("Expected method %s, got %s", tt.expected, controllers[0].Endpoints[0].Method)
			}
		})
	}
}

func TestParser_RequestMappingWithMethod(t *testing.T) {
	results := []parser.ParseResult{
		{
			Classes: []parser.Class{
				{
					Name: "TestController",
					Annotations: []parser.Annotation{
						{Name: "RestController"},
					},
					Methods: []parser.Method{
						{
							Name: "testMethod",
							Annotations: []parser.Annotation{
								{
									Name: "RequestMapping",
									Params: map[string]string{
										"value":  "/test",
										"method": "RequestMethod.POST",
									},
								},
							},
						},
					},
				},
			},
		},
	}

	p := NewParser()
	controllers := p.ExtractControllers(results)

	if len(controllers) != 1 || len(controllers[0].Endpoints) != 1 {
		t.Fatal("Failed to extract endpoint")
	}

	endpoint := controllers[0].Endpoints[0]
	if endpoint.Method != POST {
		t.Errorf("Expected method POST, got %s", endpoint.Method)
	}
	if endpoint.Path != "/test" {
		t.Errorf("Expected path '/test', got '%s'", endpoint.Path)
	}
}

func TestParser_PathCombination(t *testing.T) {
	tests := []struct {
		name       string
		basePath   string
		methodPath string
		expected   string
	}{
		{"both paths", "/api", "/users", "/api/users"},
		{"only base", "/api", "", "/api"},
		{"only method", "", "/users", "/users"},
		{"neither", "", "", ""},
		{"trailing slash", "/api/", "/users", "/api/users"},
	}

	p := NewParser()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := p.combinePaths(tt.basePath, tt.methodPath)
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestParser_NonControllerClass(t *testing.T) {
	results := []parser.ParseResult{
		{
			Classes: []parser.Class{
				{
					Name: "UserService",
					Annotations: []parser.Annotation{
						{Name: "Service"},
					},
				},
			},
		},
	}

	p := NewParser()
	controllers := p.ExtractControllers(results)

	if len(controllers) != 0 {
		t.Errorf("Expected 0 controllers, got %d", len(controllers))
	}
}
