package parser

import (
	"fmt"
	"testing"
)

func TestParseSpringMVCController(t *testing.T) {
	parser, err := NewParser()
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}
	defer parser.Close()

	// Parse the test file
	tree, source, err := parser.ParseFile("../testdata/UserController.java")
	if err != nil {
		t.Fatalf("Failed to parse file: %v", err)
	}
	defer tree.Close()

	// Extract classes
	classes, err := parser.ExtractClasses(tree, source)
	if err != nil {
		t.Fatalf("Failed to extract classes: %v", err)
	}

	// Verify we found the UserController class
	if len(classes) != 1 {
		t.Fatalf("Expected 1 class, got %d", len(classes))
	}

	class := classes[0]

	// Verify class name
	if class.Name != "UserController" {
		t.Errorf("Expected class name 'UserController', got '%s'", class.Name)
	}

	// Verify package name
	if class.Package != "com.example.demo.controller" {
		t.Errorf("Expected package 'com.example.demo.controller', got '%s'", class.Package)
	}

	// Verify class annotations
	if len(class.Annotations) < 2 {
		t.Errorf("Expected at least 2 class annotations, got %d", len(class.Annotations))
	}

	// Check for @RestController
	hasRestController := false
	for _, ann := range class.Annotations {
		if ann.Name == "RestController" {
			hasRestController = true
			break
		}
	}
	if !hasRestController {
		t.Error("Expected @RestController annotation on class")
	}

	// Check for @RequestMapping
	hasRequestMapping := false
	var requestMappingValue string
	for _, ann := range class.Annotations {
		if ann.Name == "RequestMapping" {
			hasRequestMapping = true
			requestMappingValue = ann.Params["value"]
			break
		}
	}
	if !hasRequestMapping {
		t.Error("Expected @RequestMapping annotation on class")
	}
	if requestMappingValue != "/api/users" {
		t.Errorf("Expected @RequestMapping value '/api/users', got '%s'", requestMappingValue)
	}

	// Verify methods
	if len(class.Methods) != 5 {
		t.Errorf("Expected 5 methods, got %d", len(class.Methods))
	}

	// Check getUser method
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

	// Verify @GetMapping annotation
	hasGetMapping := false
	var getMappingValue string
	for _, ann := range getUserMethod.Annotations {
		if ann.Name == "GetMapping" {
			hasGetMapping = true
			getMappingValue = ann.Params["value"]
			break
		}
	}
	if !hasGetMapping {
		t.Error("Expected @GetMapping annotation on getUser method")
	}
	if getMappingValue != "/{id}" {
		t.Errorf("Expected @GetMapping value '/{id}', got '%s'", getMappingValue)
	}

	// Verify method parameters
	if len(getUserMethod.Parameters) != 1 {
		t.Errorf("Expected 1 parameter, got %d", len(getUserMethod.Parameters))
	}

	if len(getUserMethod.Parameters) > 0 {
		param := getUserMethod.Parameters[0]
		if param.Name != "id" {
			t.Errorf("Expected parameter name 'id', got '%s'", param.Name)
		}
		if param.Type != "Long" {
			t.Errorf("Expected parameter type 'Long', got '%s'", param.Type)
		}

		// Check for @PathVariable annotation
		hasPathVariable := false
		for _, ann := range param.Annotations {
			if ann.Name == "PathVariable" {
				hasPathVariable = true
				break
			}
		}
		if !hasPathVariable {
			t.Error("Expected @PathVariable annotation on id parameter")
		}
	}

	// Print summary for manual verification
	fmt.Println("\n=== Parsing Results ===")
	fmt.Printf("Class: %s\n", class.Name)
	fmt.Printf("Package: %s\n", class.Package)
	fmt.Printf("Class Annotations: %d\n", len(class.Annotations))
	for _, ann := range class.Annotations {
		fmt.Printf("  - @%s", ann.Name)
		if len(ann.Params) > 0 {
			fmt.Printf("(")
			first := true
			for k, v := range ann.Params {
				if !first {
					fmt.Printf(", ")
				}
				fmt.Printf("%s=\"%s\"", k, v)
				first = false
			}
			fmt.Printf(")")
		}
		fmt.Println()
	}

	fmt.Printf("\nMethods: %d\n", len(class.Methods))
	for _, method := range class.Methods {
		fmt.Printf("\n  Method: %s\n", method.Name)
		fmt.Printf("  Return Type: %s\n", method.ReturnType)
		fmt.Printf("  Annotations: %d\n", len(method.Annotations))
		for _, ann := range method.Annotations {
			fmt.Printf("    - @%s", ann.Name)
			if len(ann.Params) > 0 {
				fmt.Printf("(")
				first := true
				for k, v := range ann.Params {
					if !first {
						fmt.Printf(", ")
					}
					fmt.Printf("%s=\"%s\"", k, v)
					first = false
				}
				fmt.Printf(")")
			}
			fmt.Println()
		}
		fmt.Printf("  Parameters: %d\n", len(method.Parameters))
		for _, param := range method.Parameters {
			fmt.Printf("    - %s %s", param.Type, param.Name)
			if len(param.Annotations) > 0 {
				fmt.Printf(" [")
				for i, ann := range param.Annotations {
					if i > 0 {
						fmt.Printf(", ")
					}
					fmt.Printf("@%s", ann.Name)
				}
				fmt.Printf("]")
			}
			fmt.Println()
		}
	}
	fmt.Println("======================")
}
