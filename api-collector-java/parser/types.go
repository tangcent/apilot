// Package parser provides Tree-sitter based Java/Kotlin source parsing.
package parser

// Annotation represents a Java annotation with its name and parameters.
type Annotation struct {
	Name   string            // e.g., "RestController", "GetMapping"
	Params map[string]string // e.g., {"value": "/api/users"}
}

// Method represents a Java method with annotations.
type Method struct {
	Name          string
	Annotations   []Annotation
	Parameters    []Parameter
	ReturnType    string
	JavaDoc       string
	JavaDocParams map[string]string
}

// Parameter represents a method parameter.
type Parameter struct {
	Name        string
	Type        string
	Annotations []Annotation
	JavaDoc     string
}

// Field represents a Java class field declaration.
type Field struct {
	Name        string
	Type        string
	Annotations []Annotation
	JavaDoc     string
	IsStatic    bool
	IsFinal     bool
}

// Class represents a Java class or interface with annotations and methods.
type Class struct {
	Name               string
	Package            string
	JavaDoc            string
	IsInterface        bool
	Annotations        []Annotation
	Methods            []Method
	Fields             []Field
	SuperClass         string
	SuperClassTypeArgs []string
	TypeParameters     []string
	Interfaces         []string
}

// ParseResult contains the parsing result for a single file.
type ParseResult struct {
	FilePath string
	Classes  []Class
	Error    error
}
