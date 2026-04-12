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
