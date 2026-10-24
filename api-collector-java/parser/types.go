// Package parser provides Tree-sitter based Java source parsing.
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
	// Body is the raw text of the method body, braces included, or empty for
	// a declaration without one (interface and abstract methods). Frameworks
	// whose response payload is built inside the body rather than declared in
	// the return type need it to recover the payload.
	Body string
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
	// HasInitializer reports whether the declaration carries an `= value`
	// part. A static final field with one is a compile-time constant — the
	// distinction between a constant and an instance-final piece of data.
	HasInitializer bool
}

// Import is one import declaration of a compilation unit.
type Import struct {
	// Path is the imported name: a type for a single-type import
	// ("com.example.dto.Result"), a package for an on-demand import
	// ("com.example.dto").
	Path string
	// OnDemand reports `import com.example.dto.*`.
	OnDemand bool
	// Static reports `import static ...`. Those name members rather than
	// types, so type resolution ignores them.
	Static bool
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
	// FilePath is the compilation unit that declared this class. Simple-name
	// resolution needs it to tell a declaration in the same file apart from
	// one that is merely imported (JLS 6.5.5.1). Empty for sources parsed
	// without a path, such as an in-memory dependency source.
	FilePath string
	// Imports carries the import declarations of the same compilation unit.
	Imports []Import
}

// ParseResult contains the parsing result for a single file.
type ParseResult struct {
	FilePath string
	Classes  []Class
	Error    error
}
