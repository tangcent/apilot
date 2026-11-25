// Package parser reads Kotlin source with the tree-sitter Kotlin grammar and
// exposes the declarations a Spring MVC collector needs.
package parser

import (
	"fmt"
	"strings"
)

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
// for wildcard imports. Alias is the name a renaming import (`import a.B as C`)
// gives the class: only that name denotes it in the file.
type Import struct {
	Path     string
	OnDemand bool
	Alias    string
}

// Name returns the simple name the import binds in the file: the alias of a
// renaming import, the last segment of the path otherwise. A wildcard import
// binds no name of its own.
func (i Import) Name() string {
	if i.Alias != "" {
		return i.Alias
	}
	if idx := strings.LastIndex(i.Path, "."); idx >= 0 {
		return i.Path[idx+1:]
	}
	return i.Path
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
	// Vararg marks `vararg name: T`. Type is then `Array<T>`, the array the
	// function receives, and Nullable is false, as the array is never null. A
	// vararg of a primitive type is not an Array of it but the primitive array
	// (`vararg name: Byte` is a ByteArray), which only the scope the type is
	// written in tells (see resolver.ParameterType).
	Vararg bool
	// ElementNullable marks a vararg of `T?`, which is an Array of boxed
	// values whatever T is.
	ElementNullable bool
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
	// Outer is the dotted path of the classes this declaration is nested in,
	// outermost first: "Outer" for Outer.Inner, "Outer.Mid" for Deep in
	// Outer.Mid.Deep. Empty for a top-level declaration. A nested class, inner
	// or not, is a declaration of its own, and this is all that says where it
	// belongs. A companion object is not one: its constants are read onto the
	// class it belongs to.
	Outer string
	// IsInner marks a class written with the modifier inner. Only an inner class
	// sees the type parameters of the classes around it: a nested class without
	// the modifier has no access to them.
	IsInner bool
	// IsPrivate marks a declaration written private. At the top level that makes
	// it visible in its file only.
	IsPrivate bool
	// IsFileFacade marks the class the JVM holds a file's top-level
	// declarations in, named after the file (Headers.kt declares HeadersKt).
	// Only top-level const vals are read into it, as its fields: Kotlin code
	// names them as members of their package.
	IsFileFacade bool
}

// TypeAlias is a top-level typealias declaration. It names a type of its own
// rather than declaring one, so a name that reaches it stands for Target.
type TypeAlias struct {
	Name     string
	Package  string
	FilePath string
	// Target is the aliased type as written in the file that declares the alias:
	// it is read in that file's scope, not in the scope of the code that uses the
	// alias.
	Target string
	// TargetNullable marks a target that is nullable as a whole (`typealias
	// MaybeByte = Byte?`). Target drops the `?`.
	TargetNullable bool
	TypeParameters []string
	Imports        []Import
	// IsPrivate marks an alias written private: visible in its file only.
	IsPrivate bool
}

// ParseResult is the outcome of parsing one file.
type ParseResult struct {
	FilePath    string
	Classes     []Class
	TypeAliases []TypeAlias
	Warnings    []string
	Error       error
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
