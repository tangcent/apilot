// Package parser provides Tree-sitter based Java source parsing.
package parser

// Annotation represents a Java annotation with its name and parameters.
type Annotation struct {
	Name   string            // e.g., "RestController", "GetMapping"
	Params map[string]string // e.g., {"value": "/api/users"}
	// Refs marks the params written as a name rather than a literal: a
	// constant (@RequestHeader(Headers.VERSION)) or an enum constant. Params
	// holds the name as written, and a string literal without its quotes, so
	// the two cannot be told apart from Params alone.
	Refs map[string]bool
}

// Method represents a Java method with annotations.
type Method struct {
	Name          string
	Annotations   []Annotation
	Parameters    []Parameter
	ReturnType    string
	JavaDoc       string
	JavaDocParams map[string]string
	// RawJavaDoc is the doc comment as written, markers and block tags
	// included, for readers of a tag other than @param (@deprecated).
	RawJavaDoc string
	// Body is the raw text of the method body, braces included, or empty for
	// a declaration without one (interface and abstract methods). Frameworks
	// whose response payload is built inside the body rather than declared in
	// the return type need it to recover the payload.
	Body string
	// IsStatic reports a `static` modifier. A static method is not instance
	// state, so it never contributes a serialised property.
	IsStatic bool
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
	// RawJavaDoc is the doc comment as written, block tags included, so a
	// tag such as @see can be read.
	RawJavaDoc string
	// Constant is the initializer of a static final String field, as
	// written, when an annotation can use the field in place of a string:
	// a string literal ("\"X-Version\"") or the name of another constant
	// ("Base.VERSION"). Empty for any other field or initializer.
	Constant string
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
	// Outer is the dotted path of the classes this declaration is nested in,
	// outermost first: "Outer" for Outer.Inner, "Outer.Mid" for Deep in
	// Outer.Mid.Deep. Empty for a top-level declaration. A nested class is a
	// declaration of its own, and this is all that says where it belongs.
	Outer string
	// Local marks a class declared in a method, a constructor, an initializer or a
	// lambda, or inside an anonymous class or the body of an enum constant. Its
	// name is in scope in the block that declares it and nowhere else, so it is
	// neither a member type of the class around it nor a class of its package.
	// Outer still names the types around it.
	Local bool
	// Inner marks a member class that is not static. Only an inner class sees the
	// type parameters of the classes around it: `class Outer<T> { class Inner { T
	// value; } }` reads the T of Outer, and a static member class has no such T. A
	// member of an interface is static whether or not it says so, and an
	// interface, an enum and a record are never inner.
	Inner bool
	// Decompiled marks a class read from a source that a decompiler printed from
	// bytecode, as against one a developer wrote. Bytecode spells the class Inner
	// of Outer as `Outer$Inner`, its binary name, and a decompiler may keep that
	// spelling; written by hand, a `$` is a character of an identifier.
	Decompiled bool
	// FilePath is the compilation unit that declared this class. Simple-name
	// resolution needs it to tell a declaration in the same file apart from
	// one that is merely imported (JLS 6.5.5.1). Empty for sources parsed
	// without a path, such as an in-memory dependency source.
	FilePath string
	// Imports carries the import declarations of the same compilation unit.
	Imports []Import
	// RawJavaDoc is the doc comment as written, block tags included.
	RawJavaDoc string
	// IsEnum marks an enum declaration, or a class that extends java.lang.Enum
	// (how CFR prints a Kotlin enum); EnumConstants lists its constants in
	// declaration order.
	IsEnum        bool
	EnumConstants []string
	// EnumConstantDocs holds the doc comment of each constant, cleaned, in
	// the order of EnumConstants: "" for a constant without one.
	EnumConstantDocs []string
}

// QualifiedName returns the name a declaration is known by: its package, the
// classes it is nested in and its own name, dot-separated, as in
// com.example.Outer.Inner. Two declarations that share a simple name still
// differ here, whether they sit in different packages or in different classes.
func (c Class) QualifiedName() string {
	name := c.Name
	if c.Outer != "" {
		name = c.Outer + "." + name
	}
	if c.Package == "" {
		return name
	}
	return c.Package + "." + name
}

// ParseResult contains the parsing result for a single file.
type ParseResult struct {
	FilePath string
	Classes  []Class
	Error    error
}
