package springmvc

import (
	"path/filepath"
	"testing"

	"github.com/tangcent/apilot/api-collector-java/parser"
)

func TestIntegration_ParseRealController(t *testing.T) {
	// Create parser
	p, err := parser.NewParserV2(parser.ParserOptions{
		LogLevel: parser.LogLevelError,
	})
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}
	defer p.Close()

	// Parse UserController.java
	testFile := filepath.Join("..", "testdata", "UserController.java")
	result, err := p.ParseFile(testFile)
	if err != nil {
		t.Fatalf("Failed to parse file: %v", err)
	}

	if result.Error != nil {
		t.Fatalf("Parse result contains error: %v", result.Error)
	}

	// Extract Spring MVC endpoints
	springParser := NewParser()
	controllers := springParser.ExtractControllers([]parser.ParseResult{*result})

	if len(controllers) != 1 {
		t.Fatalf("Expected 1 controller, got %d", len(controllers))
	}

	controller := controllers[0]

	// Verify controller
	if controller.Name != "UserController" {
		t.Errorf("Expected controller name 'UserController', got '%s'", controller.Name)
	}

	if controller.BasePath != "/api/users" {
		t.Errorf("Expected base path '/api/users', got '%s'", controller.BasePath)
	}

	if len(controller.Endpoints) != 5 {
		t.Fatalf("Expected 5 endpoints, got %d", len(controller.Endpoints))
	}

	// Verify each endpoint
	endpoints := controller.Endpoints

	// GET /{id}
	if endpoints[0].Method != GET {
		t.Errorf("Expected GET method for getUser, got %s", endpoints[0].Method)
	}
	if endpoints[0].Path != "/api/users/{id}" {
		t.Errorf("Expected path '/api/users/{id}', got '%s'", endpoints[0].Path)
	}
	if len(endpoints[0].Parameters) != 1 {
		t.Errorf("Expected 1 parameter for getUser, got %d", len(endpoints[0].Parameters))
	}
	if endpoints[0].Parameters[0].ParamType != "path" {
		t.Errorf("Expected path parameter, got '%s'", endpoints[0].Parameters[0].ParamType)
	}

	// GET (list)
	if endpoints[1].Method != GET {
		t.Errorf("Expected GET method for listUsers, got %s", endpoints[1].Method)
	}
	if endpoints[1].Path != "/api/users" {
		t.Errorf("Expected path '/api/users', got '%s'", endpoints[1].Path)
	}
	if len(endpoints[1].Parameters) != 2 {
		t.Errorf("Expected 2 parameters for listUsers, got %d", len(endpoints[1].Parameters))
	}

	// POST
	if endpoints[2].Method != POST {
		t.Errorf("Expected POST method for createUser, got %s", endpoints[2].Method)
	}
	if len(endpoints[2].Parameters) != 1 {
		t.Errorf("Expected 1 parameter for createUser, got %d", len(endpoints[2].Parameters))
	}
	if endpoints[2].Parameters[0].ParamType != "body" {
		t.Errorf("Expected body parameter, got '%s'", endpoints[2].Parameters[0].ParamType)
	}

	// PUT /{id}
	if endpoints[3].Method != PUT {
		t.Errorf("Expected PUT method for updateUser, got %s", endpoints[3].Method)
	}
	if endpoints[3].Path != "/api/users/{id}" {
		t.Errorf("Expected path '/api/users/{id}', got '%s'", endpoints[3].Path)
	}

	// DELETE /{id}
	if endpoints[4].Method != DELETE {
		t.Errorf("Expected DELETE method for deleteUser, got %s", endpoints[4].Method)
	}
	if endpoints[4].Path != "/api/users/{id}" {
		t.Errorf("Expected path '/api/users/{id}', got '%s'", endpoints[4].Path)
	}
}

func TestIntegration_ParseDirectory(t *testing.T) {
	// Create parser
	p, err := parser.NewParserV2(parser.ParserOptions{
		LogLevel: parser.LogLevelError,
	})
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}
	defer p.Close()

	// Parse entire testdata directory
	results, err := p.ParseDirectory(filepath.Join("..", "testdata"))
	if err != nil {
		t.Fatalf("Failed to parse directory: %v", err)
	}

	// Extract Spring MVC endpoints
	springParser := NewParser()
	controllers := springParser.ExtractControllers(results)

	if len(controllers) == 0 {
		t.Error("Expected at least one controller")
	}

	// Verify we found UserController
	found := false
	for _, controller := range controllers {
		if controller.Name == "UserController" {
			found = true
			if len(controller.Endpoints) != 5 {
				t.Errorf("Expected 5 endpoints in UserController, got %d", len(controller.Endpoints))
			}
		}
	}

	if !found {
		t.Error("UserController not found in parsed controllers")
	}
}
