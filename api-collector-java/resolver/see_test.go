package resolver

import (
	"reflect"
	"testing"

	"github.com/tangcent/apilot/api-collector-java/parser"
)

// A class read from the dependencies for a `@see` tag stays out of the type
// table. Registered, it would answer for its simple name in a compilation unit
// that neither declares nor imports that name, in place of the project's enum
// of the same name.
func TestSeeOptions_ALibraryClassDoesNotShadowAProjectEnum(t *testing.T) {
	library := &libraryClasses{classes: map[string]parser.Class{
		// Not an enum, and sorts before the project's Level.
		"com.example.lib.Level": {Name: "Level", Package: "com.example.lib", Fields: []parser.Field{{Name: "rank", Type: "int"}}},
	}}
	r := NewTypeResolver([]parser.Class{
		{Name: "Level", Package: "com.example.web.enums", IsEnum: true, EnumConstants: []string{"LOW", "HIGH"}},
		// Imports the library's Level.
		{
			Name: "Imported", Package: "com.example.web", FilePath: "Imported.java",
			Imports: []parser.Import{{Path: "com.example.lib.Level"}},
			Fields:  []parser.Field{{Name: "level", Type: "String", RawJavaDoc: "/** @see Level */"}},
		},
		// Imports no Level: the name falls back to the declaration carrying it.
		{
			Name: "Bare", Package: "com.example.web", FilePath: "Bare.java",
			Fields: []parser.Field{{Name: "level", Type: "String", RawJavaDoc: "/** @see Level */"}},
		},
	})
	r.SetDependencyResolver(library)

	if imported := r.Resolve("com.example.web.Imported", nil).Fields["level"]; len(imported.Options) != 0 {
		t.Errorf("Imported.level options = %v, want none: the library's Level is not an enum", imported.Options)
	}
	var values []any
	for _, option := range r.Resolve("com.example.web.Bare", nil).Fields["level"].Options {
		values = append(values, option.Value)
	}
	if !reflect.DeepEqual(values, []any{"LOW", "HIGH"}) {
		t.Errorf("Bare.level options = %v, want the project enum's constants", values)
	}
	r.Resolve("com.example.web.Imported", nil)
	if !reflect.DeepEqual(library.asked, []string{"com.example.lib.Level"}) {
		t.Errorf("asked = %v, want the library's Level once", library.asked)
	}
}
