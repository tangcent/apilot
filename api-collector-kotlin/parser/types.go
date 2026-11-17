// Package parser reads Kotlin source with the tree-sitter Kotlin grammar and
// exposes the declarations a Spring MVC collector needs.
package parser

import "fmt"

// Annotation is a Kotlin annotation as written on a declaration.
type Annotation struct {
	Name          string
	Params        map[string]string
	UseSiteTarget string
	// Refs marks the params written as a name rather than a literal, such as
	// a constant (@RequestHeader(Headers.VERSION)); Params holds the name as
	// written.
	Refs map[string]bool
}

// Import is a Kotlin import. Path is the fully qualified name; OnDemand is true
// for wildcard imports.
type Import struct {
	Path     string
	OnDemand bool
}

// Parameter is a function/constructor parameter.
type Parameter struct {
	Name         string
	Type         string
	Annotations  []Annotation
	HasDefault   bool
	DefaultValue string
	// Nullable marks a nullable type (`String?`). Type drops the `?`.
	Nullable bool
}

// Property is a constructor or body property (val/var).
type Property struct {
	Name         string
	Type         string
	Annotations  []Annotation
	KDoc         string
	HasDefault   bool
	DefaultValue string
	IsMutable    bool
	// InConstructor marks a property declared in the primary constructor.
	InConstructor bool
	// IsConst marks a const val. On the JVM it is a static field of its class,
	// not instance state; a const val of a companion object is read onto the
	// class the companion belongs to, where Java and annotations see it.
	IsConst bool
}

// Function is a Kotlin function declaration.
type Function struct {
	Name              string
	Annotations       []Annotation
	Parameters        []Parameter
	ReturnType        string
	HasDeclaredReturn bool
	Body              string
	KDoc              string
	IsOverride        bool
}

// Class is a Kotlin class, interface, or enum declaration.
type Class struct {
	Name               string
	Package            string
	FilePath           string
	KDoc               string
	Annotations        []Annotation
	TypeParameters     []string
	SuperClass         string
	SuperClassTypeArgs []string
	Interfaces         []string
	Fields             []Property
	Methods            []Function
	Imports            []Import
	IsData             bool
	IsEnum             bool
	IsSealed           bool
	IsInterface        bool
	EnumValues         []string
	// EnumValueKDocs holds the KDoc of each enum entry as written, in the
	// order of EnumValues: "" for an entry without one.
	EnumValueKDocs []string
	// IsFileFacade marks the class the JVM holds a file's top-level
	// declarations in, named after the file (Headers.kt declares HeadersKt).
	// Only top-level const vals are read into it, as its fields: Kotlin code
	// names them as members of their package.
	IsFileFacade bool
}

// ParseResult is the outcome of parsing one file.
type ParseResult struct {
	FilePath string
	Classes  []Class
	Warnings []string
	Error    error
}

// UnsupportedConstructError reports a Kotlin construct whose meaning the
// collector cannot model. Failing here is deliberate: exporting a wrong field
// name is worse than refusing the file.
type UnsupportedConstructError struct {
	FilePath  string
	Construct string
	// Reason says why the construct cannot be modelled.
	Reason string
}

func (e *UnsupportedConstructError) Error() string {
	return fmt.Sprintf("%s: unsupported Kotlin construct %q: %s", e.FilePath, e.Construct, e.Reason)
}

// NoKotlinSourcesError reports a project that has no .kt files to read.
type NoKotlinSourcesError struct {
	Dir string
}

func (e *NoKotlinSourcesError) Error() string {
	return fmt.Sprintf("no Kotlin sources found under %q", e.Dir)
}
