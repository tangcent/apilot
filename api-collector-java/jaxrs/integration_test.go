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
