package resolver

import (
	"testing"

	"github.com/tangcent/apilot/api-collector-java/parser"
)

// A type variable is what its name denotes in the class that declares it, before
// the class of java.lang or of a framework that the name is known for: the rules
// that read a type by the name it is written with (a void return, an upload, an
// Optional, a wrapper) do not take a variable for that class.

func TestResolve_ATypeVariableIsNotAClassThatTheResolverKnowsByName(t *testing.T) {
	imports := []parser.Import{
		{Path: "org.springframework.web.multipart.MultipartFile"}, {Path: "java.util.Optional"},
		{Path: "reactor.core.publisher.Mono"}, {Path: "org.springframework.http.ResponseEntity"},
	}
	declares := func(params ...string) parser.Class {
		return parser.Class{Name: "Ctl", Package: "p", FilePath: "p/Ctl.java", TypeParameters: params, Imports: imports}
	}
	plain := declares()
	r := NewTypeResolver([]parser.Class{plain})

	// Without the variable the names are the classes they are known for.
	if got := r.Identity(plain, "MultipartFile"); got != "org.springframework.web.multipart.MultipartFile" {
		t.Fatalf("Identity(MultipartFile) = %s, want the class the file imports", got)
	}
	if !r.IsMultipartFile(plain, "MultipartFile") || !r.IsVoid(plain, "Void") || !r.IsOptionalType(plain, "Optional") {
		t.Fatal("the names are expected to be read as the classes they are known for where no variable has them")
	}

	for _, name := range []string{"MultipartFile", "Optional", "Mono", "ResponseEntity", "Void", "Object", "String"} {
		source := declares(name)
		if got := r.Identity(source, name); got != name {
			t.Errorf("Identity(%s) = %s, want the variable", name, got)
		}
		if r.IsMultipartFile(source, name) {
			t.Errorf("IsMultipartFile(%s) = true, want false: it is a variable", name)
		}
		if r.IsVoid(source, name) {
			t.Errorf("IsVoid(%s) = true, want false: it is a variable", name)
		}
		if r.IsOptionalType(source, name) {
			t.Errorf("IsOptionalType(%s) = true, want false: it is a variable", name)
		}
		if got := r.UnwrapOptional(source, name); got != name {
			t.Errorf("UnwrapOptional(%s) = %s, want the variable", name, got)
		}
	}
}

// An inner class reads the variables of the class around it, a static member class
// does not.
func TestResolve_ATypeVariableOfTheClassAroundAnInnerClassIsNotAJavaLangClass(t *testing.T) {
	outer := parser.Class{Name: "Outer", Package: "p", FilePath: "p/Outer.java", TypeParameters: []string{"Void"}}
	inner := parser.Class{Name: "Inner", Package: "p", Outer: "Outer", Inner: true, FilePath: "p/Outer.java"}
	fixed := parser.Class{Name: "Fixed", Package: "p", Outer: "Outer", FilePath: "p/Outer.java"}
	r := NewTypeResolver([]parser.Class{outer, inner, fixed})

	if r.IsVoid(inner, "Void") {
		t.Error("IsVoid(Void) in an inner class = true, want false: the class around it declares the variable Void")
	}
	if !r.IsVoid(fixed, "Void") {
		t.Error("IsVoid(Void) in a static member class = false, want true: it has no access to the variable")
	}
}
