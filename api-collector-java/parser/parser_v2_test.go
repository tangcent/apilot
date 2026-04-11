package parser

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParserV2_ParseFile(t *testing.T) {
	tmpDir := t.TempDir()

	opts := ParserOptions{
		CacheDir: filepath.Join(tmpDir, "cache"),
		LogLevel: LogLevelError,
	}

	p, err := NewParserV2(opts)
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}
	defer p.Close()

	testFile := "../testdata/UserController.java"
	result, err := p.ParseFile(testFile)
	if err != nil {
		t.Fatalf("Failed to parse file: %v", err)
	}

	if result.Error != nil {
		t.Fatalf("Parse result contains error: %v", result.Error)
	}

	if len(result.Classes) != 1 {
		t.Fatalf("Expected 1 class, got %d", len(result.Classes))
	}

	class := result.Classes[0]
	if class.Name != "UserController" {
		t.Errorf("Expected class name 'UserController', got '%s'", class.Name)
	}

	if len(class.Annotations) != 2 {
		t.Errorf("Expected 2 class annotations, got %d", len(class.Annotations))
	}

	if len(class.Methods) != 5 {
		t.Errorf("Expected 5 methods, got %d", len(class.Methods))
	}
}

func TestParserV2_Cache(t *testing.T) {
	tmpDir := t.TempDir()

	opts := ParserOptions{
		CacheDir: filepath.Join(tmpDir, "cache"),
		LogLevel: LogLevelError,
	}

	p, err := NewParserV2(opts)
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}
	defer p.Close()

	testFile := "../testdata/UserController.java"

	// First parse - cache miss
	result1, err := p.ParseFile(testFile)
	if err != nil {
		t.Fatalf("First parse failed: %v", err)
	}

	// Second parse - cache hit
	result2, err := p.ParseFile(testFile)
	if err != nil {
		t.Fatalf("Second parse failed: %v", err)
	}

	// Results should be identical
	if len(result1.Classes) != len(result2.Classes) {
		t.Errorf("Cache returned different number of classes")
	}

	if result1.Classes[0].Name != result2.Classes[0].Name {
		t.Errorf("Cache returned different class name")
	}

	// Verify cache directory exists
	cacheDir := filepath.Join(tmpDir, "cache")
	if _, err := os.Stat(cacheDir); os.IsNotExist(err) {
		t.Error("Cache directory was not created")
	}
}

func TestParserV2_ParseDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	opts := ParserOptions{
		CacheDir: filepath.Join(tmpDir, "cache"),
		LogLevel: LogLevelError,
	}

	p, err := NewParserV2(opts)
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

	// Verify at least one result is UserController
	found := false
	for _, result := range results {
		if result.Error == nil && len(result.Classes) > 0 {
			if result.Classes[0].Name == "UserController" {
				found = true
				break
			}
		}
	}

	if !found {
		t.Error("UserController not found in parse results")
	}
}

func TestParserV2_ParseDirectoryParallel(t *testing.T) {
	tmpDir := t.TempDir()

	opts := ParserOptions{
		CacheDir: filepath.Join(tmpDir, "cache"),
		LogLevel: LogLevelError,
	}

	p, err := NewParserV2(opts)
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}
	defer p.Close()

	// Test with 2 workers
	results, err := p.ParseDirectoryParallel("../testdata", 2)
	if err != nil {
		t.Fatalf("Failed to parse directory in parallel: %v", err)
	}

	if len(results) == 0 {
		t.Error("Expected at least one parse result")
	}

	// Verify results are the same as sequential parse
	seqResults, err := p.ParseDirectory("../testdata")
	if err != nil {
		t.Fatalf("Failed to parse directory sequentially: %v", err)
	}

	if len(results) != len(seqResults) {
		t.Errorf("Parallel parse returned %d results, sequential returned %d",
			len(results), len(seqResults))
	}
}

func TestParserV2_NoCacheMode(t *testing.T) {
	opts := ParserOptions{
		CacheDir: "", // Disable cache
		LogLevel: LogLevelError,
	}

	p, err := NewParserV2(opts)
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}
	defer p.Close()

	testFile := "../testdata/UserController.java"
	result, err := p.ParseFile(testFile)
	if err != nil {
		t.Fatalf("Failed to parse file: %v", err)
	}

	if result.Error != nil {
		t.Fatalf("Parse result contains error: %v", result.Error)
	}

	if len(result.Classes) != 1 {
		t.Fatalf("Expected 1 class, got %d", len(result.Classes))
	}
}

func TestParserV2_LogLevels(t *testing.T) {
	tmpDir := t.TempDir()

	levels := []LogLevel{LogLevelDebug, LogLevelInfo, LogLevelWarn, LogLevelError}

	for _, level := range levels {
		opts := ParserOptions{
			CacheDir: filepath.Join(tmpDir, "cache"),
			LogLevel: level,
		}

		p, err := NewParserV2(opts)
		if err != nil {
			t.Fatalf("Failed to create parser with log level %d: %v", level, err)
		}

		testFile := "../testdata/UserController.java"
		_, err = p.ParseFile(testFile)
		if err != nil {
			t.Errorf("Parse failed with log level %d: %v", level, err)
		}

		p.Close()
	}
}
