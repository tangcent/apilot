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

func TestParser_ImplicitBodyParameter(t *testing.T) {
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
							Name:        "create",
							Annotations: []parser.Annotation{{Name: "POST"}},
							Parameters: []parser.Parameter{
								{Name: "body", Type: "User", Annotations: []parser.Annotation{}},
							},
							ReturnType: "Response",
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
	if len(ep.Parameters) != 1 {
		t.Fatalf("Expected 1 parameter, got %d", len(ep.Parameters))
	}
	if ep.Parameters[0].ParamType != "body" {
		t.Errorf("Expected body parameter, got '%s'", ep.Parameters[0].ParamType)
	}
	if !ep.Parameters[0].Required {
		t.Error("Implicit body parameter should be required")
	}
}

func TestParser_BeanParamAsBody(t *testing.T) {
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
							Name:        "search",
							Annotations: []parser.Annotation{{Name: "GET"}},
							Parameters: []parser.Parameter{
								{Name: "filter", Type: "SearchFilter", Annotations: []parser.Annotation{{Name: "BeanParam"}}},
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
	ep := resources[0].Endpoints[0]
	if len(ep.Parameters) != 1 {
		t.Fatalf("Expected 1 parameter, got %d", len(ep.Parameters))
	}
	if ep.Parameters[0].ParamType != "body" {
		t.Errorf("Expected body parameter for @BeanParam, got '%s'", ep.Parameters[0].ParamType)
	}
}

func TestParser_ResponseTypeUnwrapping(t *testing.T) {
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
							Name:        "getPlainResponse",
							Annotations: []parser.Annotation{{Name: "GET"}},
							ReturnType:  "Response",
						},
						{
							Name:        "getDirectType",
							Annotations: []parser.Annotation{{Name: "GET"}, {Name: "Path", Params: map[string]string{"value": "/direct"}}},
							ReturnType:  "String",
						},
						{
							Name:        "getVoid",
							Annotations: []parser.Annotation{{Name: "DELETE"}, {Name: "Path", Params: map[string]string{"value": "/void"}}},
							ReturnType:  "void",
						},
					},
				},
			},
		},
	}

	p := NewParser()
	resources := p.ExtractResources(results)

	if len(resources) != 1 || len(resources[0].Endpoints) != 3 {
		t.Fatalf("Expected 3 endpoints, got %d", len(resources[0].Endpoints))
	}

	if resources[0].Endpoints[0].ResponseSchema != nil {
		t.Error("Plain Response should not produce a schema")
	}

	if resources[0].Endpoints[1].ResponseSchema == nil {
		t.Error("Direct String return should produce a schema")
	} else if resources[0].Endpoints[1].ResponseSchema.TypeName != "string" {
		t.Errorf("Expected string schema, got %s", resources[0].Endpoints[1].ResponseSchema.TypeName)
	}

	if resources[0].Endpoints[2].ResponseSchema != nil {
		t.Error("void return should not produce a schema")
	}
}

func TestParser_TypeResolution(t *testing.T) {
	results := []parser.ParseResult{
		{
			Classes: []parser.Class{
				{
					Name: "User",
					Fields: []parser.Field{
						{Name: "name", Type: "String"},
						{Name: "email", Type: "String"},
						{Name: "active", Type: "boolean"},
					},
				},
				{
					Name: "TestResource",
					Annotations: []parser.Annotation{
						{Name: "Path", Params: map[string]string{"value": "/test"}},
					},
					Methods: []parser.Method{
						{
							Name:        "create",
							Annotations: []parser.Annotation{{Name: "POST"}},
							Parameters: []parser.Parameter{
								{Name: "user", Type: "User", Annotations: []parser.Annotation{}},
							},
							ReturnType: "User",
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

	if ep.RequestBodySchema == nil {
		t.Fatal("Expected RequestBodySchema")
	}
	if !ep.RequestBodySchema.IsObject() {
		t.Fatalf("Expected object model, got kind=%s", ep.RequestBodySchema.Kind)
	}
	for _, field := range []string{"name", "email", "active"} {
		if _, ok := ep.RequestBodySchema.Fields[field]; !ok {
			t.Errorf("Expected field '%s' in request body schema", field)
		}
	}

	if ep.ResponseSchema == nil {
		t.Fatal("Expected ResponseSchema")
	}
	if !ep.ResponseSchema.IsObject() {
		t.Fatalf("Expected object model, got kind=%s", ep.ResponseSchema.Kind)
	}
}

func TestParser_InheritedEndpoints(t *testing.T) {
	results := []parser.ParseResult{
		{
			Classes: []parser.Class{
				{
					Name: "BaseResource",
					Annotations: []parser.Annotation{
						{Name: "Produces", Params: map[string]string{"value": "application/json"}},
					},
					TypeParameters: []string{"T"},
					Methods: []parser.Method{
						{
							Name:        "getById",
							Annotations: []parser.Annotation{{Name: "GET"}, {Name: "Path", Params: map[string]string{"value": "/{id}"}}},
							Parameters: []parser.Parameter{
								{Name: "id", Type: "Long", Annotations: []parser.Annotation{{Name: "PathParam"}}},
							},
							ReturnType: "T",
						},
					},
				},
				{
					Name: "Item",
					Fields: []parser.Field{
						{Name: "id", Type: "Long"},
						{Name: "title", Type: "String"},
					},
				},
				{
					Name: "ItemResource",
					Annotations: []parser.Annotation{
						{Name: "Path", Params: map[string]string{"value": "/items"}},
					},
					SuperClass:         "BaseResource",
					SuperClassTypeArgs: []string{"Item"},
					Methods:            []parser.Method{},
				},
			},
		},
	}

	p := NewParser()
	resources := p.ExtractResources(results)

	var itemResource *Resource
	for i := range resources {
		if resources[i].Name == "ItemResource" {
			itemResource = &resources[i]
			break
		}
	}
	if itemResource == nil {
		t.Fatal("Expected ItemResource")
	}

	if len(itemResource.Endpoints) != 1 {
		t.Fatalf("Expected 1 inherited endpoint, got %d", len(itemResource.Endpoints))
	}

	ep := itemResource.Endpoints[0]
	if ep.MethodName != "getById" {
		t.Errorf("Expected inherited method 'getById', got '%s'", ep.MethodName)
	}
	if ep.Path != "/items/{id}" {
		t.Errorf("Expected path '/items/{id}', got '%s'", ep.Path)
	}

	if ep.ResponseSchema == nil {
		t.Fatal("Expected ResponseSchema for inherited endpoint")
	}
	if !ep.ResponseSchema.IsObject() {
		t.Fatalf("Expected object model for Item, got kind=%s", ep.ResponseSchema.Kind)
	}
	if _, ok := ep.ResponseSchema.Fields["title"]; !ok {
		t.Error("Expected 'title' field in resolved Item schema")
	}
}

func TestParser_ContextParamIgnored(t *testing.T) {
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
							Name:        "get",
							Annotations: []parser.Annotation{{Name: "GET"}},
							Parameters: []parser.Parameter{
								{Name: "uriInfo", Type: "UriInfo", Annotations: []parser.Annotation{{Name: "Context"}}},
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
	if len(resources[0].Endpoints[0].Parameters) != 0 {
		t.Errorf("Expected 0 parameters (@Context should be ignored), got %d", len(resources[0].Endpoints[0].Parameters))
	}
}
