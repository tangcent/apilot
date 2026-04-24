package feign

import (
	"path/filepath"
	"testing"

	"github.com/tangcent/apilot/api-collector-java/parser"
)

func TestIntegration_ParseRealFeignClient(t *testing.T) {
	p, err := parser.NewParser(parser.ParserOptions{
		LogLevel: parser.LogLevelError,
	})
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}
	defer p.Close()

	testFile := filepath.Join("..", "testdata", "UserClient.java")
	result, err := p.ParseFile(testFile)
	if err != nil {
		t.Fatalf("Failed to parse file: %v", err)
	}
	if result.Error != nil {
		t.Fatalf("Parse result contains error: %v", result.Error)
	}

	fParser := NewParser()
	clients := fParser.ExtractClients([]parser.ParseResult{*result})

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
	if len(client.Endpoints) != 4 {
		t.Fatalf("Expected 4 endpoints, got %d", len(client.Endpoints))
	}

	// GET /api/users/{id}
	ep0 := client.Endpoints[0]
	if ep0.Method != GET {
		t.Errorf("Expected GET, got %s", ep0.Method)
	}
	if ep0.Path != "/api/users/{id}" {
		t.Errorf("Expected '/api/users/{id}', got '%s'", ep0.Path)
	}
	if len(ep0.Parameters) != 1 || ep0.Parameters[0].ParamType != "path" {
		t.Errorf("Expected path param, got %v", ep0.Parameters)
	}

	// GET /api/users (list)
	ep1 := client.Endpoints[1]
	if ep1.Method != GET {
		t.Errorf("Expected GET, got %s", ep1.Method)
	}
	if len(ep1.Parameters) != 2 {
		t.Errorf("Expected 2 query params, got %d", len(ep1.Parameters))
	}

	// POST /api/users
	ep2 := client.Endpoints[2]
	if ep2.Method != POST {
		t.Errorf("Expected POST, got %s", ep2.Method)
	}

	// DELETE /api/users/{id}
	ep3 := client.Endpoints[3]
	if ep3.Method != DELETE {
		t.Errorf("Expected DELETE, got %s", ep3.Method)
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

	fParser := NewParser()
	clients := fParser.ExtractClients(results)

	var userClient *FeignClient
	for i := range clients {
		if clients[i].Name == "UserClient" {
			userClient = &clients[i]
			break
		}
	}
	if userClient == nil {
		t.Fatal("Expected UserClient in parsed clients")
	}

	// getUser returns User — should resolve to object with inherited fields
	var getEp *Endpoint
	for i := range userClient.Endpoints {
		if userClient.Endpoints[i].MethodName == "getUser" {
			getEp = &userClient.Endpoints[i]
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
		t.Fatalf("Expected object model for User, got kind=%s", getEp.ResponseSchema.Kind)
	}

	// User extends BaseEntity — check inherited fields
	for _, name := range []string{"name", "email", "active", "id", "createdAt", "updatedAt"} {
		if _, ok := getEp.ResponseSchema.Fields[name]; !ok {
			t.Errorf("Expected field '%s' in User response schema", name)
		}
	}

	// listUsers returns List<User>
	var listEp *Endpoint
	for i := range userClient.Endpoints {
		if userClient.Endpoints[i].MethodName == "listUsers" {
			listEp = &userClient.Endpoints[i]
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

	// createUser has @RequestBody User and returns User
	var createEp *Endpoint
	for i := range userClient.Endpoints {
		if userClient.Endpoints[i].MethodName == "createUser" {
			createEp = &userClient.Endpoints[i]
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
		t.Fatalf("Expected object model for User request body, got kind=%s", createEp.RequestBodySchema.Kind)
	}
	if createEp.ResponseSchema == nil {
		t.Fatal("Expected ResponseSchema for createUser")
	}

	// deleteUser returns void
	var deleteEp *Endpoint
	for i := range userClient.Endpoints {
		if userClient.Endpoints[i].MethodName == "deleteUser" {
			deleteEp = &userClient.Endpoints[i]
			break
		}
	}
	if deleteEp == nil {
		t.Fatal("Expected 'deleteUser' endpoint")
	}
	if deleteEp.ResponseSchema != nil {
		t.Error("void return should not produce a ResponseSchema")
	}
}
