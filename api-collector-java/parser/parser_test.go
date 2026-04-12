package parser

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParser_ParseFile(t *testing.T) {
	p, err := NewParser(ParserOptions{
		CacheDir: filepath.Join(t.TempDir(), "cache"),
		LogLevel: LogLevelError,
	})
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}
	defer p.Close()

	result, err := p.ParseFile("../testdata/UserController.java")
	if err != nil {
		t.Fatalf("Failed to parse file: %v", err)
	}
	if result.Error != nil {
		t.Fatalf("Parse result error: %v", result.Error)
	}
	if len(result.Classes) != 1 {
		t.Fatalf("Expected 1 class, got %d", len(result.Classes))
	}

	class := result.Classes[0]
	if class.Name != "UserController" {
		t.Errorf("Expected class name 'UserController', got '%s'", class.Name)
	}
	if class.Package != "com.example.demo.controller" {
		t.Errorf("Expected package 'com.example.demo.controller', got '%s'", class.Package)
	}
	if len(class.Annotations) != 2 {
		t.Errorf("Expected 2 class annotations, got %d", len(class.Annotations))
	}
	if len(class.Methods) != 5 {
		t.Errorf("Expected 5 methods, got %d", len(class.Methods))
	}

	// Verify @RestController annotation
	hasRestController := false
	for _, ann := range class.Annotations {
		if ann.Name == "RestController" {
			hasRestController = true
		}
	}
	if !hasRestController {
		t.Error("Expected @RestController annotation")
	}

	// Verify getUser method and its @PathVariable parameter
	var getUserMethod *Method
	for i := range class.Methods {
		if class.Methods[i].Name == "getUser" {
			getUserMethod = &class.Methods[i]
			break
		}
	}
	if getUserMethod == nil {
		t.Fatal("Expected to find getUser method")
	}
	if len(getUserMethod.Parameters) != 1 {
		t.Fatalf("Expected 1 parameter, got %d", len(getUserMethod.Parameters))
	}
	param := getUserMethod.Parameters[0]
	if param.Name != "id" || param.Type != "Long" {
		t.Errorf("Expected param 'Long id', got '%s %s'", param.Type, param.Name)
	}
	hasPathVariable := false
	for _, ann := range param.Annotations {
		if ann.Name == "PathVariable" {
			hasPathVariable = true
		}
	}
	if !hasPathVariable {
		t.Error("Expected @PathVariable on id parameter")
	}
}

func TestParser_ParseInterface(t *testing.T) {
	p, err := NewParser(ParserOptions{LogLevel: LogLevelError})
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}
	defer p.Close()

	result, err := p.ParseFile("../testdata/UserClient.java")
	if err != nil {
		t.Fatalf("Failed to parse file: %v", err)
	}
	if result.Error != nil {
		t.Fatalf("Parse result error: %v", result.Error)
	}
	if len(result.Classes) != 1 {
		t.Fatalf("Expected 1 class/interface, got %d", len(result.Classes))
	}

	iface := result.Classes[0]
	if !iface.IsInterface {
		t.Error("Expected IsInterface=true")
	}
	if iface.Name != "UserClient" {
		t.Errorf("Expected 'UserClient', got '%s'", iface.Name)
	}
}

func TestParser_Cache(t *testing.T) {
	p, err := NewParser(ParserOptions{
		CacheDir: filepath.Join(t.TempDir(), "cache"),
		LogLevel: LogLevelError,
	})
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}
	defer p.Close()

	testFile := "../testdata/UserController.java"

	result1, err := p.ParseFile(testFile)
	if err != nil {
		t.Fatalf("First parse failed: %v", err)
	}
	result2, err := p.ParseFile(testFile)
	if err != nil {
		t.Fatalf("Second parse (cache hit) failed: %v", err)
	}

	if len(result1.Classes) != len(result2.Classes) {
		t.Error("Cache returned different number of classes")
	}
	if result1.Classes[0].Name != result2.Classes[0].Name {
		t.Error("Cache returned different class name")
	}

	if _, err := os.Stat(filepath.Join(p.cache.cacheDir)); os.IsNotExist(err) {
		t.Error("Cache directory was not created")
	}
}

func TestParser_NoCacheMode(t *testing.T) {
	p, err := NewParser(ParserOptions{LogLevel: LogLevelError})
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}
	defer p.Close()

	result, err := p.ParseFile("../testdata/UserController.java")
	if err != nil {
		t.Fatalf("Failed to parse file: %v", err)
	}
	if result.Error != nil {
		t.Fatalf("Parse result error: %v", result.Error)
	}
	if len(result.Classes) != 1 {
		t.Fatalf("Expected 1 class, got %d", len(result.Classes))
	}
}

func TestParser_ParseDirectory(t *testing.T) {
	p, err := NewParser(ParserOptions{
		CacheDir: filepath.Join(t.TempDir(), "cache"),
		LogLevel: LogLevelError,
	})
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}
	defer p.Close()

	results, err := p.ParseDirectory("../testdata")
	if err != nil {
		t.Fatalf("Failed to parse directory: %v", err)
	}
	if len(results) == 0 {
		t.Error("Expected at least one parse result")
	}

	found := false
	for _, result := range results {
		for _, class := range result.Classes {
			if class.Name == "UserController" {
				found = true
			}
		}
	}
	if !found {
		t.Error("UserController not found in parse results")
	}
}

func TestParser_ParseDirectoryParallel(t *testing.T) {
	p, err := NewParser(ParserOptions{
		CacheDir: filepath.Join(t.TempDir(), "cache"),
		LogLevel: LogLevelError,
	})
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}
	defer p.Close()

	parallel, err := p.ParseDirectoryParallel("../testdata", 2)
	if err != nil {
		t.Fatalf("Failed to parse directory in parallel: %v", err)
	}
	sequential, err := p.ParseDirectory("../testdata")
	if err != nil {
		t.Fatalf("Failed to parse directory sequentially: %v", err)
	}
	if len(parallel) != len(sequential) {
		t.Errorf("Parallel returned %d results, sequential returned %d",
			len(parallel), len(sequential))
	}
}

func TestParser_LogLevels(t *testing.T) {
	for _, level := range []LogLevel{LogLevelDebug, LogLevelInfo, LogLevelWarn, LogLevelError} {
		p, err := NewParser(ParserOptions{
			CacheDir: filepath.Join(t.TempDir(), "cache"),
			LogLevel: level,
		})
		if err != nil {
			t.Fatalf("Failed to create parser with log level %d: %v", level, err)
		}
		if _, err := p.ParseFile("../testdata/UserController.java"); err != nil {
			t.Errorf("Parse failed with log level %d: %v", level, err)
		}
		p.Close()
	}
}

func TestParser_ConcurrentAccess(t *testing.T) {
	p, err := NewParser(ParserOptions{
		CacheDir: filepath.Join(t.TempDir(), "cache"),
		LogLevel: LogLevelError,
	})
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}
	defer p.Close()

	const n = 50
	errChan := make(chan error, n)
	for i := 0; i < n; i++ {
		go func() {
			_, err := p.ParseFile("../testdata/UserController.java")
			errChan <- err
		}()
	}
	for i := 0; i < n; i++ {
		if err := <-errChan; err != nil {
			t.Errorf("Concurrent parse failed: %v", err)
		}
	}
}
