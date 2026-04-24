package jaxrs

import (
	"path/filepath"
	"testing"

	"github.com/tangcent/apilot/api-collector-java/parser"
)

func TestIntegration_ParseRealResource(t *testing.T) {
	p, err := parser.NewParser(parser.ParserOptions{
		LogLevel: parser.LogLevelError,
	})
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}
	defer p.Close()

	testFile := filepath.Join("..", "testdata", "UserResource.java")
	result, err := p.ParseFile(testFile)
	if err != nil {
		t.Fatalf("Failed to parse file: %v", err)
	}
	if result.Error != nil {
		t.Fatalf("Parse result contains error: %v", result.Error)
	}

	jParser := NewParser()
	resources := jParser.ExtractResources([]parser.ParseResult{*result})

	if len(resources) != 1 {
		t.Fatalf("Expected 1 resource, got %d", len(resources))
	}

	res := resources[0]
	if res.Name != "UserResource" {
		t.Errorf("Expected 'UserResource', got '%s'", res.Name)
	}
	if res.BasePath != "/api/users" {
		t.Errorf("Expected '/api/users', got '%s'", res.BasePath)
	}
	if len(res.Endpoints) != 5 {
		t.Fatalf("Expected 5 endpoints, got %d", len(res.Endpoints))
	}

	// GET /{id}
	if res.Endpoints[0].Method != GET {
		t.Errorf("Expected GET, got %s", res.Endpoints[0].Method)
	}
	if res.Endpoints[0].Path != "/api/users/{id}" {
		t.Errorf("Expected '/api/users/{id}', got '%s'", res.Endpoints[0].Path)
	}

	// GET (list)
	if res.Endpoints[1].Method != GET {
		t.Errorf("Expected GET, got %s", res.Endpoints[1].Method)
	}
	if res.Endpoints[1].Path != "/api/users" {
		t.Errorf("Expected '/api/users', got '%s'", res.Endpoints[1].Path)
	}
	if len(res.Endpoints[1].Parameters) != 2 {
		t.Errorf("Expected 2 query params, got %d", len(res.Endpoints[1].Parameters))
	}

	// POST
	if res.Endpoints[2].Method != POST {
		t.Errorf("Expected POST, got %s", res.Endpoints[2].Method)
	}

	// PUT /{id}
	if res.Endpoints[3].Method != PUT {
		t.Errorf("Expected PUT, got %s", res.Endpoints[3].Method)
	}

	// DELETE /{id}
	if res.Endpoints[4].Method != DELETE {
		t.Errorf("Expected DELETE, got %s", res.Endpoints[4].Method)
	}
}

func TestIntegration_TypeResolution(t *testing.T) {
	p, err := parser.NewParser(parser.ParserOptions{
		LogLevel: parser.LogLevelError,
	})
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}
	defer p.Close()

	results, err := p.ParseDirectory(filepath.Join("..", "testdata"))
	if err != nil {
		t.Fatalf("Failed to parse directory: %v", err)
	}

	jParser := NewParser()
	resources := jParser.ExtractResources(results)

	var userResource *Resource
	for i := range resources {
		if resources[i].Name == "UserResource" {
			userResource = &resources[i]
			break
		}
	}
	if userResource == nil {
		t.Fatal("Expected UserResource in parsed resources")
	}

	// createUser has implicit body param (User) and returns Response
	var createEp *Endpoint
	for i := range userResource.Endpoints {
		if userResource.Endpoints[i].MethodName == "createUser" {
			createEp = &userResource.Endpoints[i]
			break
		}
	}
	if createEp == nil {
		t.Fatal("Expected 'createUser' endpoint")
	}

	if createEp.RequestBodySchema == nil {
		t.Fatal("Expected RequestBodySchema for createUser")
	}
	if !createEp.RequestBodySchema.IsObject() {
		t.Fatalf("Expected object model for User, got kind=%s", createEp.RequestBodySchema.Kind)
	}

	expectedFields := []string{"name", "email", "active"}
	for _, name := range expectedFields {
		if _, ok := createEp.RequestBodySchema.Fields[name]; !ok {
			t.Errorf("Expected field '%s' in User request body", name)
		}
	}

	// User extends BaseEntity, so inherited fields should be present
	inheritedFields := []string{"id", "createdAt", "updatedAt"}
	for _, name := range inheritedFields {
		if _, ok := createEp.RequestBodySchema.Fields[name]; !ok {
			t.Errorf("Expected inherited field '%s' from BaseEntity in User request body", name)
		}
	}

	// Response return type should not produce a schema
	if createEp.ResponseSchema != nil {
		t.Error("Plain Response return should not produce a ResponseSchema")
	}

	// getUser returns User directly
	var getEp *Endpoint
	for i := range userResource.Endpoints {
		if userResource.Endpoints[i].MethodName == "getUser" {
			getEp = &userResource.Endpoints[i]
			break
		}
	}
	if getEp == nil {
		t.Fatal("Expected 'getUser' endpoint")
	}

	if getEp.ResponseSchema == nil {
		t.Fatal("Expected ResponseSchema for getUser")
	}
	if !getEp.ResponseSchema.IsObject() {
		t.Fatalf("Expected object model for User response, got kind=%s", getEp.ResponseSchema.Kind)
	}

	// listUsers returns List<User>
	var listEp *Endpoint
	for i := range userResource.Endpoints {
		if userResource.Endpoints[i].MethodName == "listUsers" {
			listEp = &userResource.Endpoints[i]
			break
		}
	}
	if listEp == nil {
		t.Fatal("Expected 'listUsers' endpoint")
	}

	if listEp.ResponseSchema == nil {
		t.Fatal("Expected ResponseSchema for listUsers")
	}
	if !listEp.ResponseSchema.IsArray() {
		t.Fatalf("Expected array model for List<User>, got kind=%s", listEp.ResponseSchema.Kind)
	}
	if listEp.ResponseSchema.Items == nil || !listEp.ResponseSchema.Items.IsObject() {
		t.Error("Expected array items to be User object")
	}
}

func TestIntegration_InheritedResourceEndpoints(t *testing.T) {
	p, err := parser.NewParser(parser.ParserOptions{
		LogLevel: parser.LogLevelError,
	})
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}
	defer p.Close()

	results, err := p.ParseDirectory(filepath.Join("..", "testdata"))
	if err != nil {
		t.Fatalf("Failed to parse directory: %v", err)
	}

	jParser := NewParser()
	resources := jParser.ExtractResources(results)

	var orderResource *Resource
	for i := range resources {
		if resources[i].Name == "OrderResource" {
			orderResource = &resources[i]
			break
		}
	}
	if orderResource == nil {
		t.Fatal("Expected OrderResource in parsed resources")
	}

	// OrderResource extends BaseCrudResource<CreateOrderReq, OrderVO>
	// It should have its own methods + inherited CRUD methods
	methodNames := make(map[string]bool)
	for _, ep := range orderResource.Endpoints {
		methodNames[ep.MethodName] = true
	}

	// Own methods
	if !methodNames["search"] {
		t.Error("Expected 'search' method from OrderResource")
	}
	if !methodNames["batchCreate"] {
		t.Error("Expected 'batchCreate' method from OrderResource")
	}

	// Inherited methods from BaseCrudResource
	for _, name := range []string{"create", "getById", "list", "update", "delete"} {
		if !methodNames[name] {
			t.Errorf("Expected inherited method '%s' from BaseCrudResource", name)
		}
	}

	// Verify type resolution on inherited 'create' method
	var createEp *Endpoint
	for i := range orderResource.Endpoints {
		if orderResource.Endpoints[i].MethodName == "create" {
			createEp = &orderResource.Endpoints[i]
			break
		}
	}
	if createEp == nil {
		t.Fatal("Expected 'create' endpoint")
	}

	// create(Req request) with Req=CreateOrderReq
	if createEp.RequestBodySchema == nil {
		t.Fatal("Expected RequestBodySchema for inherited create endpoint")
	}
	if !createEp.RequestBodySchema.IsObject() {
		t.Fatalf("Expected object model for CreateOrderReq, got kind=%s", createEp.RequestBodySchema.Kind)
	}

	// create returns Res with Res=OrderVO
	if createEp.ResponseSchema == nil {
		t.Fatal("Expected ResponseSchema for inherited create endpoint")
	}
	if !createEp.ResponseSchema.IsObject() {
		t.Fatalf("Expected object model for OrderVO, got kind=%s", createEp.ResponseSchema.Kind)
	}
}
