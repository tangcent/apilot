package jaxrs

import (
	"testing"

	"github.com/tangcent/apilot/api-collector-java/parser"
)

func makeJaxrsResults() []parser.ParseResult {
	return []parser.ParseResult{
		{
			FilePath: "UserResource.java",
			Classes: []parser.Class{
				{
					Name:    "UserResource",
					Package: "com.example.api",
					Annotations: []parser.Annotation{
						{Name: "Path", Params: map[string]string{"value": "/users"}},
						{Name: "Produces", Params: map[string]string{"value": "application/json"}},
					},
					Methods: []parser.Method{
						{
							Name: "getUser",
							Annotations: []parser.Annotation{
								{Name: "GET"},
								{Name: "Path", Params: map[string]string{"value": "/{id}"}},
							},
							Parameters: []parser.Parameter{
								{
									Name: "id",
									Type: "Long",
									Annotations: []parser.Annotation{
										{Name: "PathParam"},
									},
								},
							},
							ReturnType: "User",
						},
						{
							Name: "listUsers",
							Annotations: []parser.Annotation{
								{Name: "GET"},
							},
							Parameters: []parser.Parameter{
								{
									Name: "page",
									Type: "int",
									Annotations: []parser.Annotation{
										{Name: "QueryParam"},
									},
								},
								{
									Name: "size",
									Type: "int",
									Annotations: []parser.Annotation{
										{Name: "QueryParam"},
									},
								},
							},
							ReturnType: "List<User>",
						},
						{
							Name: "createUser",
							Annotations: []parser.Annotation{
								{Name: "POST"},
								{Name: "Consumes", Params: map[string]string{"value": "application/json"}},
							},
							Parameters: []parser.Parameter{
								{
									Name:        "user",
									Type:        "User",
									Annotations: []parser.Annotation{},
								},
							},
							ReturnType: "Response",
						},
						{
							Name: "updateUser",
							Annotations: []parser.Annotation{
								{Name: "PUT"},
								{Name: "Path", Params: map[string]string{"value": "/{id}"}},
							},
							Parameters: []parser.Parameter{
								{
									Name: "id",
									Type: "Long",
									Annotations: []parser.Annotation{
										{Name: "PathParam"},
									},
								},
							},
							ReturnType: "Response",
						},
						{
							Name: "deleteUser",
							Annotations: []parser.Annotation{
								{Name: "DELETE"},
								{Name: "Path", Params: map[string]string{"value": "/{id}"}},
							},
							Parameters: []parser.Parameter{
								{
									Name: "id",
									Type: "Long",
									Annotations: []parser.Annotation{
										{Name: "PathParam"},
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
}

func TestParser_ExtractResources(t *testing.T) {
	p := NewParser()
	resources := p.ExtractResources(makeJaxrsResults())

	if len(resources) != 1 {
		t.Fatalf("Expected 1 resource, got %d", len(resources))
	}

	res := resources[0]
	if res.Name != "UserResource" {
		t.Errorf("Expected 'UserResource', got '%s'", res.Name)
	}
	if res.BasePath != "/users" {
		t.Errorf("Expected '/users', got '%s'", res.BasePath)
	}
	if res.Package != "com.example.api" {
		t.Errorf("Expected 'com.example.api', got '%s'", res.Package)
	}
	if len(res.Produces) != 1 || res.Produces[0] != "application/json" {
		t.Errorf("Expected produces ['application/json'], got %v", res.Produces)
	}

	if len(res.Endpoints) != 5 {
		t.Fatalf("Expected 5 endpoints, got %d", len(res.Endpoints))
	}
}

func TestParser_HTTPMethods(t *testing.T) {
	tests := []struct {
		annotation string
		expected   HTTPMethod
	}{
		{"GET", GET},
		{"POST", POST},
		{"PUT", PUT},
		{"DELETE", DELETE},
		{"HEAD", HEAD},
		{"OPTIONS", OPTIONS},
	}

	for _, tt := range tests {
		t.Run(tt.annotation, func(t *testing.T) {
			results := []parser.ParseResult{
				{
					Classes: []parser.Class{
						{
							Name: "TestResource",
							Annotations: []parser.Annotation{
								{Name: "Path", Params: map[string]string{"value": "/test"}},
							},
							Methods: []parser.Method{
								{
									Name: "testMethod",
									Annotations: []parser.Annotation{
										{Name: tt.annotation},
									},
								},
							},
						},
					},
				},
			}

			p := NewParser()
			resources := p.ExtractResources(results)

			if len(resources) != 1 || len(resources[0].Endpoints) != 1 {
				t.Fatal("Failed to extract endpoint")
			}
			if resources[0].Endpoints[0].Method != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, resources[0].Endpoints[0].Method)
			}
		})
	}
}

func TestParser_PathCombination(t *testing.T) {
	tests := []struct {
		name       string
		classPath  string
		methodPath string
		expected   string
	}{
		{"both paths", "/users", "/{id}", "/users/{id}"},
		{"only class", "/users", "", "/users"},
		{"only method", "", "/{id}", "/{id}"},
	}

	p := NewParser()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := []parser.ParseResult{
				{
					Classes: []parser.Class{
						{
							Name: "TestResource",
							Annotations: []parser.Annotation{
								{Name: "Path", Params: map[string]string{"value": tt.classPath}},
							},
							Methods: []parser.Method{
								{
									Name: "testMethod",
									Annotations: func() []parser.Annotation {
										anns := []parser.Annotation{{Name: "GET"}}
										if tt.methodPath != "" {
											anns = append(anns, parser.Annotation{
												Name:   "Path",
												Params: map[string]string{"value": tt.methodPath},
											})
										}
										return anns
									}(),
								},
							},
						},
					},
				},
			}

			resources := p.ExtractResources(results)
			if len(resources) != 1 || len(resources[0].Endpoints) != 1 {
				t.Fatal("Failed to extract endpoint")
			}
			if resources[0].Endpoints[0].Path != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, resources[0].Endpoints[0].Path)
			}
		})
	}
}

func TestParser_ParameterTypes(t *testing.T) {
	results := []parser.ParseResult{
		{
			Classes: []parser.Class{
				{
					Name: "TestResource",
					Annotations: []parser.Annotation{
						{Name: "Path", Params: map[string]string{"value": "/test"}},
					},
					Methods: []parser.Method{
						{
							Name: "testMethod",
							Annotations: []parser.Annotation{
								{Name: "POST"},
							},
							Parameters: []parser.Parameter{
								{Name: "id", Type: "Long", Annotations: []parser.Annotation{{Name: "PathParam"}}},
								{Name: "q", Type: "String", Annotations: []parser.Annotation{{Name: "QueryParam"}}},
								{Name: "form", Type: "String", Annotations: []parser.Annotation{{Name: "FormParam"}}},
								{Name: "auth", Type: "String", Annotations: []parser.Annotation{{Name: "HeaderParam"}}},
							},
						},
					},
				},
			},
		},
	}

	p := NewParser()
	resources := p.ExtractResources(results)

	if len(resources) != 1 || len(resources[0].Endpoints) != 1 {
		t.Fatal("Failed to extract endpoint")
	}

	params := resources[0].Endpoints[0].Parameters
	if len(params) != 4 {
		t.Fatalf("Expected 4 parameters, got %d", len(params))
	}

	expected := []string{"path", "query", "form", "header"}
	for i, exp := range expected {
		if params[i].ParamType != exp {
			t.Errorf("param[%d]: expected type '%s', got '%s'", i, exp, params[i].ParamType)
		}
	}
	if !params[0].Required {
		t.Error("path param should be required")
	}
	if params[1].Required {
		t.Error("query param should not be required")
	}
}

func TestParser_NonResourceClass(t *testing.T) {
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
	resources := p.ExtractResources(results)

	if len(resources) != 0 {
		t.Errorf("Expected 0 resources, got %d", len(resources))
	}
}

func TestParser_ClassLevelProducesInherited(t *testing.T) {
	results := []parser.ParseResult{
		{
			Classes: []parser.Class{
				{
					Name: "TestResource",
					Annotations: []parser.Annotation{
						{Name: "Path", Params: map[string]string{"value": "/test"}},
						{Name: "Produces", Params: map[string]string{"value": "application/json"}},
					},
					Methods: []parser.Method{
						{
							Name:        "get",
							Annotations: []parser.Annotation{{Name: "GET"}},
						},
					},
				},
			},
		},
	}

	p := NewParser()
	resources := p.ExtractResources(results)

	if len(resources) != 1 || len(resources[0].Endpoints) != 1 {
		t.Fatal("Failed to extract endpoint")
	}
	ep := resources[0].Endpoints[0]
	if len(ep.Produces) != 1 || ep.Produces[0] != "application/json" {
		t.Errorf("Expected inherited produces, got %v", ep.Produces)
	}
}
