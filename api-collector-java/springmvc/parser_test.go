package springmvc

import (
	"testing"

	"github.com/tangcent/apilot/api-collector-java/parser"
	model "github.com/tangcent/apilot/api-model"
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

// TestParser_RecordRequestBody verifies the acceptance case for record DTOs:
// a record bound as @RequestBody exports its components as body fields.
func TestParser_RecordRequestBody(t *testing.T) {
	p := NewParser()

	jp, err := parser.NewParser(parser.ParserOptions{})
	if err != nil {
		t.Fatalf("Failed to create java parser: %v", err)
	}
	defer jp.Close()

	source := []byte(`package com.example.api;

import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RestController;

@RestController
public class UserRecordController {

    @PostMapping("/api/users")
    public CreateUserRequest createUser(@RequestBody CreateUserRequest request) {
        return request;
    }
}

record CreateUserRequest(String name, String email, int age) {}
`)

	classes, err := jp.ParseSource(source)
	if err != nil {
		t.Fatalf("Failed to parse source: %v", err)
	}

	controllers := p.ExtractControllers([]parser.ParseResult{
		{FilePath: "UserRecordController.java", Classes: classes},
	})

	if len(controllers) != 1 {
		t.Fatalf("Expected 1 controller, got %d", len(controllers))
	}
	if len(controllers[0].Endpoints) != 1 {
		t.Fatalf("Expected 1 endpoint, got %d", len(controllers[0].Endpoints))
	}

	schema := controllers[0].Endpoints[0].RequestBodySchema
	if schema == nil {
		t.Fatal("Expected request body schema")
	}
	if schema.Kind != model.KindObject {
		t.Fatalf("Expected KindObject, got %s", schema.Kind)
	}
	if schema.TypeName != "CreateUserRequest" {
		t.Errorf("Expected typeName 'CreateUserRequest', got '%s'", schema.TypeName)
	}
	for _, fieldName := range []string{"name", "email", "age"} {
		if schema.Fields[fieldName] == nil {
			t.Errorf("Expected body field '%s'", fieldName)
		}
	}
	if len(schema.Fields) != 3 {
		t.Errorf("Expected 3 body fields, got %d", len(schema.Fields))
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

func TestParser_DocumentationMetadata(t *testing.T) {
	results := []parser.ParseResult{
		{
			Classes: []parser.Class{
				{
					Name:    "DocumentedController",
					Package: "com.example.api",
					Annotations: []parser.Annotation{
						{Name: "RestController"},
					},
					Methods: []parser.Method{
						{
							Name:    "getUser",
							JavaDoc: "JavaDoc fallback",
							JavaDocParams: map[string]string{
								"id": "fallback id",
							},
							Annotations: []parser.Annotation{
								{Name: "Operation", Params: map[string]string{
									"summary":     "Fetch user",
									"description": "Fetch user by id",
								}},
								{Name: "GetMapping", Params: map[string]string{"value": "/users/{id}"}},
							},
							Parameters: []parser.Parameter{
								{
									Name:    "id",
									Type:    "Long",
									JavaDoc: "parameter fallback",
									Annotations: []parser.Annotation{
										{Name: "Parameter", Params: map[string]string{
											"description": "User id",
											"example":     "42",
											"required":    "true",
										}},
										{Name: "PathVariable"},
									},
								},
							},
							ReturnType: "DocumentedResponse",
						},
					},
				},
			},
		},
	}

	p := NewParser()
	controllers := p.ExtractControllers(results)
	if len(controllers) != 1 || len(controllers[0].Endpoints) != 1 {
		t.Fatalf("Expected one documented endpoint, got %#v", controllers)
	}

	endpoint := controllers[0].Endpoints[0]
	if endpoint.Description != "Fetch user\n\nFetch user by id" {
		t.Fatalf("Expected annotation endpoint description, got %q", endpoint.Description)
	}
	if len(endpoint.Parameters) != 1 {
		t.Fatalf("Expected one parameter, got %d", len(endpoint.Parameters))
	}
	param := endpoint.Parameters[0]
	if param.Description != "User id" {
		t.Errorf("Expected parameter description, got %q", param.Description)
	}
	if param.Example != "42" {
		t.Errorf("Expected parameter example, got %q", param.Example)
	}
	if !param.Required {
		t.Error("Expected parameter required from annotation")
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

// TestParser_WildcardReturnType covers issue #167: a handler declared with
// `Result<?>` must never export "?" as the payload type, and the body's
// construction fills the open argument when it names one.
func TestParser_WildcardReturnType(t *testing.T) {
	p := NewParser()

	jp, err := parser.NewParser(parser.ParserOptions{})
	if err != nil {
		t.Fatalf("Failed to create java parser: %v", err)
	}
	defer jp.Close()

	source := []byte(`package com.example.api;

import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RequestParam;
import org.springframework.web.bind.annotation.RestController;
import java.util.List;

@RestController
@RequestMapping("/api/orders")
public class OrderPageController {

    @GetMapping("/page")
    public Result<?> page(@RequestParam(defaultValue = "1") int page,
                          @RequestParam(defaultValue = "10") int size) {
        return Result.success(new PageResult<>());
    }

    @GetMapping("/open")
    public Result<?> open() {
        return Result.success(service.pageOrders(new PageRequest(1, 10)));
    }
}

class Result<T> {
    private int code;
    private String message;
    private T data;
}

class PageResult<T> {
    private long total;
    private int page;
    private int size;
    private List<T> records;
}

class PageRequest {
    private int page;
    private int size;
}
`)

	classes, err := jp.ParseSource(source)
	if err != nil {
		t.Fatalf("Failed to parse source: %v", err)
	}

	controllers := p.ExtractControllers([]parser.ParseResult{
		{FilePath: "OrderPageController.java", Classes: classes},
	})
	if len(controllers) != 1 || len(controllers[0].Endpoints) != 2 {
		t.Fatalf("Expected one controller with 2 endpoints, got %#v", controllers)
	}

	// The body builds `new PageResult<>()`, so the open argument is PageResult.
	paged := controllers[0].Endpoints[0].ResponseSchema
	if paged == nil || !paged.IsObject() {
		t.Fatalf("Expected an object response, got %+v", paged)
	}
	data := paged.Fields["data"]
	if data == nil || data.Model == nil || !data.Model.IsObject() {
		t.Fatalf("Expected data to expand PageResult, got %+v", data)
	}
	if data.Model.TypeName != "PageResult" {
		t.Errorf("Expected data typeName 'PageResult', got %q", data.Model.TypeName)
	}
	for _, fieldName := range []string{"total", "page", "size"} {
		if data.Model.Fields[fieldName] == nil {
			t.Errorf("Expected PageResult field %q under data", fieldName)
		}
	}

	// The body names no payload, so the argument stays open. An open argument
	// is an empty object, never the literal "?".
	open := controllers[0].Endpoints[1].ResponseSchema
	if open == nil || !open.IsObject() {
		t.Fatalf("Expected an object response, got %+v", open)
	}
	openData := open.Fields["data"]
	if openData == nil || openData.Model == nil || !openData.Model.IsObject() {
		t.Fatalf("Expected data to be an open object, got %+v", openData)
	}
	if openData.Model.TypeName != "object" {
		t.Errorf("Expected data typeName 'object', got %q", openData.Model.TypeName)
	}
}

// TestParser_MediaTypesFromMapping covers the consumes/produces attributes of
// a mapping annotation: the declared media types reach the endpoint as wire
// media types, and a mapping that declares none leaves them empty.
func TestParser_MediaTypesFromMapping(t *testing.T) {
	results := []parser.ParseResult{
		{
			Classes: []parser.Class{
				{
					Name: "UploadController",
					Annotations: []parser.Annotation{
						{Name: "RestController"},
					},
					Methods: []parser.Method{
						{
							Name: "upload",
							Annotations: []parser.Annotation{
								{Name: "PostMapping", Params: map[string]string{
									"value":    "/upload",
									"consumes": "MediaType.MULTIPART_FORM_DATA",
									"produces": "MediaType.APPLICATION_JSON",
								}},
							},
							ReturnType: "String",
						},
						{
							Name: "plain",
							Annotations: []parser.Annotation{
								{Name: "PostMapping", Params: map[string]string{"value": "/plain"}},
							},
							ReturnType: "String",
						},
					},
				},
			},
		},
	}

	controllers := NewParser().ExtractControllers(results)
	if len(controllers) != 1 || len(controllers[0].Endpoints) != 2 {
		t.Fatalf("Expected 1 controller with 2 endpoints, got %+v", controllers)
	}

	upload := controllers[0].Endpoints[0]
	if len(upload.Consumes) != 1 || upload.Consumes[0] != "multipart/form-data" {
		t.Errorf("upload consumes = %v, want [multipart/form-data]", upload.Consumes)
	}
	if len(upload.Produces) != 1 || upload.Produces[0] != "application/json" {
		t.Errorf("upload produces = %v, want [application/json]", upload.Produces)
	}

	plain := controllers[0].Endpoints[1]
	if len(plain.Consumes) != 0 || len(plain.Produces) != 0 {
		t.Errorf("plain media types = %v / %v, want none", plain.Consumes, plain.Produces)
	}
}

// TestParser_RequestPartIsFormParam covers the multipart binding @RequestPart:
// a part of a multipart request is a form field, not a query argument.
func TestParser_RequestPartIsFormParam(t *testing.T) {
	results := []parser.ParseResult{
		{
			Classes: []parser.Class{
				{
					Name: "UploadController",
					Annotations: []parser.Annotation{
						{Name: "RestController"},
					},
					Methods: []parser.Method{
						{
							Name: "uploadParts",
							Annotations: []parser.Annotation{
								{Name: "PostMapping", Params: map[string]string{"value": "/parts"}},
							},
							Parameters: []parser.Parameter{
								{
									Name: "avatar",
									Type: "MultipartFile",
									Annotations: []parser.Annotation{
										{Name: "RequestPart", Params: map[string]string{"value": "avatar"}},
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

	controllers := NewParser().ExtractControllers(results)
	if len(controllers) != 1 || len(controllers[0].Endpoints) != 1 {
		t.Fatalf("Expected 1 controller with 1 endpoint, got %+v", controllers)
	}

	params := controllers[0].Endpoints[0].Parameters
	if len(params) != 1 {
		t.Fatalf("Expected 1 parameter, got %+v", params)
	}
	if params[0].ParamType != "form" {
		t.Errorf("avatar paramType = %q, want form", params[0].ParamType)
	}
}
