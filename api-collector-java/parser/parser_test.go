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

func TestParser_ExtractsDocumentationMetadata(t *testing.T) {
	p, err := NewParser(ParserOptions{LogLevel: LogLevelError})
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}
	defer p.Close()

	source := []byte(`package com.example;

import io.swagger.v3.oas.annotations.Operation;
import io.swagger.v3.oas.annotations.Parameter;
import io.swagger.v3.oas.annotations.media.Schema;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PathVariable;

/**
 * Documented API
 */
public class DocumentedController {
    /**
     * Fetch by id.
     *
     * @param id fallback id
     */
    @Operation(summary = "Fetch user", description = "Fetch user by id")
    @GetMapping("/{id}")
    public DocumentedResponse getUser(
            /**
             * user id
             */
            @Parameter(description = "User id", example = "42", required = true)
            @PathVariable Long id) {
        return null;
    }
}

class DocumentedResponse {
    /**
     * Display name fallback
     */
    @Schema(description = "Display name", example = "Ada", required = true, allowableValues = {"Ada", "Grace"})
    private String name;
}`)

	classes, err := p.ParseSource(source)
	if err != nil {
		t.Fatalf("Failed to parse source: %v", err)
	}
	if len(classes) != 2 {
		t.Fatalf("Expected 2 classes, got %d", len(classes))
	}

	controller := classes[0]
	if controller.JavaDoc != "Documented API" {
		t.Fatalf("Expected class JavaDoc, got %q", controller.JavaDoc)
	}
	if len(controller.Methods) != 1 {
		t.Fatalf("Expected 1 method, got %d", len(controller.Methods))
	}

	method := controller.Methods[0]
	if method.JavaDoc != "Fetch by id." {
		t.Errorf("Expected method JavaDoc, got %q", method.JavaDoc)
	}
	if method.JavaDocParams["id"] != "fallback id" {
		t.Errorf("Expected @param fallback, got %q", method.JavaDocParams["id"])
	}

	operation := findAnnotation(method.Annotations, "Operation")
	if operation == nil {
		t.Fatal("Expected Operation annotation")
	}
	if operation.Params["summary"] != "Fetch user" {
		t.Errorf("Expected Operation summary, got %q", operation.Params["summary"])
	}
	if operation.Params["description"] != "Fetch user by id" {
		t.Errorf("Expected Operation description, got %q", operation.Params["description"])
	}

	if len(method.Parameters) != 1 {
		t.Fatalf("Expected 1 parameter, got %d", len(method.Parameters))
	}
	param := method.Parameters[0]
	if param.JavaDoc != "user id" {
		t.Errorf("Expected parameter JavaDoc, got %q", param.JavaDoc)
	}
	parameterAnn := findAnnotation(param.Annotations, "Parameter")
	if parameterAnn == nil {
		t.Fatal("Expected Parameter annotation")
	}
	if parameterAnn.Params["description"] != "User id" {
		t.Errorf("Expected Parameter description, got %q", parameterAnn.Params["description"])
	}
	if parameterAnn.Params["example"] != "42" {
		t.Errorf("Expected Parameter example, got %q", parameterAnn.Params["example"])
	}
	if parameterAnn.Params["required"] != "true" {
		t.Errorf("Expected Parameter required, got %q", parameterAnn.Params["required"])
	}

	response := classes[1]
	if len(response.Fields) != 1 {
		t.Fatalf("Expected 1 response field, got %d", len(response.Fields))
	}
	field := response.Fields[0]
	if field.JavaDoc != "Display name fallback" {
		t.Errorf("Expected field JavaDoc, got %q", field.JavaDoc)
	}
	schema := findAnnotation(field.Annotations, "Schema")
	if schema == nil {
		t.Fatal("Expected Schema annotation")
	}
	if schema.Params["description"] != "Display name" {
		t.Errorf("Expected Schema description, got %q", schema.Params["description"])
	}
	if schema.Params["example"] != "Ada" {
		t.Errorf("Expected Schema example, got %q", schema.Params["example"])
	}
	if schema.Params["required"] != "true" {
		t.Errorf("Expected Schema required, got %q", schema.Params["required"])
	}
	if schema.Params["allowableValues"] != "Ada,Grace" {
		t.Errorf("Expected Schema allowableValues, got %q", schema.Params["allowableValues"])
	}
}

func TestParser_ParseRecord(t *testing.T) {
	p, err := NewParser(ParserOptions{LogLevel: LogLevelError})
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}
	defer p.Close()

	t.Run("plain record", func(t *testing.T) {
		result, err := p.ParseFile("../testdata/CreateUserRequest.java")
		if err != nil {
			t.Fatalf("Failed to parse file: %v", err)
		}
		if result.Error != nil {
			t.Fatalf("Parse result error: %v", result.Error)
		}
		if len(result.Classes) != 1 {
			t.Fatalf("Expected 1 class, got %d", len(result.Classes))
		}

		rec := result.Classes[0]
		if rec.Name != "CreateUserRequest" {
			t.Errorf("Expected name 'CreateUserRequest', got '%s'", rec.Name)
		}
		if rec.Package != "com.example.demo.model" {
			t.Errorf("Expected package 'com.example.demo.model', got '%s'", rec.Package)
		}
		if len(rec.Fields) != 3 {
			t.Fatalf("Expected 3 fields, got %d", len(rec.Fields))
		}
		expectedFields := []struct{ name, typ string }{
			{"name", "String"},
			{"email", "String"},
			{"age", "int"},
		}
		for i, want := range expectedFields {
			if rec.Fields[i].Name != want.name || rec.Fields[i].Type != want.typ {
				t.Errorf("Expected field %d '%s %s', got '%s %s'",
					i, want.typ, want.name, rec.Fields[i].Type, rec.Fields[i].Name)
			}
			if !rec.Fields[i].IsFinal {
				t.Errorf("Expected record component '%s' to be marked final", want.name)
			}
		}
	})

	t.Run("generic record", func(t *testing.T) {
		result, err := p.ParseFile("../testdata/PageResponseRecord.java")
		if err != nil {
			t.Fatalf("Failed to parse file: %v", err)
		}
		if result.Error != nil {
			t.Fatalf("Parse result error: %v", result.Error)
		}
		if len(result.Classes) != 1 {
			t.Fatalf("Expected 1 class, got %d", len(result.Classes))
		}

		rec := result.Classes[0]
		if rec.Name != "PageResponseRecord" {
			t.Errorf("Expected name 'PageResponseRecord', got '%s'", rec.Name)
		}
		if len(rec.TypeParameters) != 1 || rec.TypeParameters[0] != "T" {
			t.Errorf("Expected type parameter 'T', got %v", rec.TypeParameters)
		}
		if len(rec.Fields) != 3 {
			t.Fatalf("Expected 3 fields, got %d", len(rec.Fields))
		}
		if rec.Fields[0].Name != "items" || rec.Fields[0].Type != "List<T>" {
			t.Errorf("Expected field 'List<T> items', got '%s %s'",
				rec.Fields[0].Type, rec.Fields[0].Name)
		}
	})

	t.Run("record with validation annotations and body", func(t *testing.T) {
		result, err := p.ParseFile("../testdata/ValidatedUserRecord.java")
		if err != nil {
			t.Fatalf("Failed to parse file: %v", err)
		}
		if result.Error != nil {
			t.Fatalf("Parse result error: %v", result.Error)
		}
		if len(result.Classes) != 1 {
			t.Fatalf("Expected 1 class, got %d", len(result.Classes))
		}

		rec := result.Classes[0]
		if rec.Name != "ValidatedUserRecord" {
			t.Errorf("Expected name 'ValidatedUserRecord', got '%s'", rec.Name)
		}
		if rec.JavaDoc != "A user payload validated through record component annotations." {
			t.Errorf("Expected record JavaDoc, got %q", rec.JavaDoc)
		}
		if len(rec.Fields) != 3 {
			t.Fatalf("Expected 3 fields, got %d", len(rec.Fields))
		}

		name := rec.Fields[0]
		if name.Name != "name" {
			t.Fatalf("Expected first field 'name', got '%s'", name.Name)
		}
		if len(name.Annotations) != 2 {
			t.Fatalf("Expected 2 annotations on 'name', got %d", len(name.Annotations))
		}
		if findAnnotation(name.Annotations, "NotNull") == nil {
			t.Error("Expected @NotNull on 'name'")
		}
		size := findAnnotation(name.Annotations, "Size")
		if size == nil {
			t.Fatal("Expected @Size on 'name'")
		}
		if size.Params["min"] != "2" || size.Params["max"] != "50" {
			t.Errorf("Expected Size min=2 max=50, got %v", size.Params)
		}

		// The compact constructor and body method must not become fields.
		for _, f := range rec.Fields {
			if f.Name != "name" && f.Name != "email" && f.Name != "age" {
				t.Errorf("Unexpected field from record body: '%s'", f.Name)
			}
		}
	})
}

func TestParser_ParseImmutableProduct(t *testing.T) {
	p, err := NewParser(ParserOptions{LogLevel: LogLevelError})
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}
	defer p.Close()

	result, err := p.ParseFile("../testdata/ImmutableProduct.java")
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
	if class.Name != "ImmutableProduct" {
		t.Errorf("Expected name 'ImmutableProduct', got '%s'", class.Name)
	}
	if len(class.Fields) != 3 {
		t.Fatalf("Expected 3 fields, got %d", len(class.Fields))
	}

	fields := make(map[string]Field, len(class.Fields))
	for _, f := range class.Fields {
		fields[f.Name] = f
	}

	constant := fields["TYPE"]
	if constant.Name == "" {
		t.Fatal("Expected static final constant 'TYPE' to be captured")
	}
	if !constant.IsStatic || !constant.IsFinal || !constant.HasInitializer {
		t.Errorf("Expected 'TYPE' static final with initializer, got static=%v final=%v initializer=%v",
			constant.IsStatic, constant.IsFinal, constant.HasInitializer)
	}

	for _, want := range []struct{ name, typ string }{{"sku", "String"}, {"price", "BigDecimal"}} {
		f := fields[want.name]
		if f.Name == "" {
			t.Fatalf("Expected field '%s' to be captured", want.name)
		}
		if f.Type != want.typ {
			t.Errorf("Expected '%s' type '%s', got '%s'", want.name, want.typ, f.Type)
		}
		if !f.IsFinal || f.IsStatic {
			t.Errorf("Expected '%s' final instance field, got final=%v static=%v",
				want.name, f.IsFinal, f.IsStatic)
		}
		if f.HasInitializer {
			t.Errorf("Expected '%s' to have no initializer", want.name)
		}
	}
}

func findAnnotation(annotations []Annotation, name string) *Annotation {
	for i := range annotations {
		if annotations[i].Name == name {
			return &annotations[i]
		}
	}
	return nil
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

func TestParser_ParseSource(t *testing.T) {
	p, err := NewParser(ParserOptions{LogLevel: LogLevelError})
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}
	defer p.Close()

	t.Run("parses simple class", func(t *testing.T) {
		source := []byte(`package com.example;

public class User {
    private Long id;
    private String name;
}`)
		classes, err := p.ParseSource(source)
		if err != nil {
			t.Fatalf("Failed to parse source: %v", err)
		}
		if len(classes) != 1 {
			t.Fatalf("Expected 1 class, got %d", len(classes))
		}
		if classes[0].Name != "User" {
			t.Errorf("Expected class name 'User', got '%s'", classes[0].Name)
		}
		if classes[0].Package != "com.example" {
			t.Errorf("Expected package 'com.example', got '%s'", classes[0].Package)
		}
		if len(classes[0].Fields) != 2 {
			t.Fatalf("Expected 2 fields, got %d", len(classes[0].Fields))
		}
	})

	t.Run("parses generic class", func(t *testing.T) {
		source := []byte(`public class Result<T> {
    private int code;
    private T data;
}`)
		classes, err := p.ParseSource(source)
		if err != nil {
			t.Fatalf("Failed to parse source: %v", err)
		}
		if len(classes) != 1 {
			t.Fatalf("Expected 1 class, got %d", len(classes))
		}
		if classes[0].Name != "Result" {
			t.Errorf("Expected class name 'Result', got '%s'", classes[0].Name)
		}
		if len(classes[0].TypeParameters) != 1 || classes[0].TypeParameters[0] != "T" {
			t.Errorf("Expected type parameter 'T', got %v", classes[0].TypeParameters)
		}
	})

	t.Run("returns empty classes for empty source", func(t *testing.T) {
		source := []byte("")
		classes, err := p.ParseSource(source)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if len(classes) != 0 {
			t.Errorf("Expected 0 classes for empty source, got %d", len(classes))
		}
	})
}
