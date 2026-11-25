package resolver

import (
	"testing"

	collector "github.com/tangcent/apilot/api-collector"
	"github.com/tangcent/apilot/api-collector-java/parser"
)

// A library or JDK class passed as the argument of a generic class is the class
// the caller's scope gives, whatever the generic class's own compilation unit
// imports: the wrappers are looked through, the value types and enums are
// scalars, and the file is an upload.

// libraryArguments are the classes under test: the package that declares each,
// and the shape `Box<...>.data` has when the caller means that class.
var libraryArguments = map[string]struct{ spelling, pkg, want string }{
	"Mono":           {"Mono<Item>", "reactor.core.publisher", "object:Item"},
	"Flux":           {"Flux<Item>", "reactor.core.publisher", "array<object:Item>"},
	"ResponseEntity": {"ResponseEntity<Item>", "org.springframework.http", "object:Item"},
	"Optional":       {"Optional<Item>", "java.util", "object:Item"},
	"OptionalInt":    {"OptionalInt", "java.util", "int"},
	"Locale":         {"Locale", "java.util", "string"},
	"Currency":       {"Currency", "java.util", "string"},
	"TimeZone":       {"TimeZone", "java.util", "string"},
	"Month":          {"Month", "java.time", "string"},
	"DayOfWeek":      {"DayOfWeek", "java.time", "string"},
	"JsonNode":       {"JsonNode", "com.fasterxml.jackson.databind", "object:object"},
	"MultipartFile":  {"MultipartFile", "org.springframework.web.multipart", "MultipartFile"},
}

func libraryArgumentItem() parser.Class {
	return parser.Class{Name: "Item", Package: identityLib + ".model", FilePath: "idlib/model/Item.java",
		Fields: []parser.Field{{Name: "id", Type: "int"}}}
}

// The caller imports the library class; the generic class's compilation unit
// imports a class of another library that has the same simple name.
func TestResolve_ALibraryTypeArgumentIsNotReboundByTheGenericClassesImports(t *testing.T) {
	for name, library := range libraryArguments {
		t.Run(name, func(t *testing.T) {
			for view, resolveIn := range typeArgViews {
				box := identityBox("com.other." + name)
				r := NewTypeResolver([]parser.Class{box, libraryArgumentItem()})
				r.SetUnresolved(collector.NewUnresolvedSet())
				caller := identityCaller(library.pkg+"."+name, identityLib+".model.Item")
				resolved := resolveIn(r, caller, "Box<"+library.spelling+">")
				if got := arrayShape(typeArgAt(resolved, "data")); got != library.want {
					t.Errorf("%s: Box<%s>.data = %s, want %s", view, library.spelling, got, library.want)
				}
				if upload := r.IsUpload(typeArgAt(resolved, "data")); upload != (name == "MultipartFile") {
					t.Errorf("%s: Box<%s>.data IsUpload = %v", view, library.spelling, upload)
				}
				if unresolved := r.Unresolved(); len(unresolved) != 0 {
					t.Errorf("%s: Box<%s> unresolved = %v, want none", view, library.spelling, unresolved)
				}
			}
		})
	}
}

// The reverse: the caller imports another library's class of that name, and the
// generic class's compilation unit imports the library one. The wrapper is not
// looked through and the value is no scalar: the name is a class that no body is
// known for, shown as written.
func TestResolve_AnotherLibrarysClassIsNotTakenForTheLibraryOneInAnArgument(t *testing.T) {
	for name, library := range libraryArguments {
		t.Run(name, func(t *testing.T) {
			for view, resolveIn := range typeArgViews {
				box := identityBox(library.pkg + "." + name)
				r := NewTypeResolver([]parser.Class{box, libraryArgumentItem()})
				r.SetUnresolved(collector.NewUnresolvedSet())
				caller := identityCaller("com.other."+name, identityLib+".model.Item")
				resolved := resolveIn(r, caller, "Box<"+library.spelling+">")
				data := typeArgAt(resolved, "data")
				if got := arrayShape(data); got != library.spelling {
					t.Errorf("%s: Box<%s>.data = %s, want the other library's class, as written", view, library.spelling, got)
				}
				if r.IsUpload(data) {
					t.Errorf("%s: Box<%s>.data is an upload, but the caller's class is not Spring's", view, library.spelling)
				}
			}
		})
	}
}
