// Package parser provides Tree-sitter based Java/Kotlin source parsing.
package parser

// Annotation represents a Java annotation with its name and parameters.
type Annotation struct {
	Name   string            // e.g., "RestController", "GetMapping"
	Params map[string]string // e.g., {"value": "/api/users"}
}

// Method represents a Java method with annotations.
type Method struct {
	Name        string
	Annotations []Annotation
	Parameters  []Parameter
	ReturnType  string
}

// Parameter represents a method parameter.
type Parameter struct {
	Name        string
	Type        string
	Annotations []Annotation
}

// Class represents a Java class with annotations and methods.
type Class struct {
	Name        string
	Package     string
	Annotations []Annotation
	Methods     []Method
}

// ParseResult contains the parsing result for a single file.
type ParseResult struct {
	FilePath string
	Classes  []Class
	Error    error
}
