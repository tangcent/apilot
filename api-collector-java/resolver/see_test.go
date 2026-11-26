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
		// Imports the package of the project's Level on demand.
		{
			Name: "Bare", Package: "com.example.web", FilePath: "Bare.java",
			Imports: []parser.Import{{Path: "com.example.web.enums", OnDemand: true}},
			Fields:  []parser.Field{{Name: "level", Type: "String", RawJavaDoc: "/** @see Level */"}},
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

// A `@see` tag names an enum as the scope of its source gives the name, as a type
// does: the enum of the source's own package, and the enum nested in the outer
// class that scope gives, whatever other enum carries the same simple name.
func TestSeeOptions_NamesAreResolvedInTheScopeOfTheSource(t *testing.T) {
	fields := []parser.Field{
		{Name: "code", Type: "String", RawJavaDoc: "/** @see Code */"},
		{Name: "status", Type: "String", RawJavaDoc: "/** @see Order.Status */"},
	}
	r := NewTypeResolver([]parser.Class{
		{Name: "Code", Package: "a", FilePath: "a/Code.java", IsEnum: true, EnumConstants: []string{"A"}, EnumJsonValue: true},
		{Name: "Code", Package: "b", FilePath: "b/Code.java", IsEnum: true, EnumConstants: []string{"B"}},
		{Name: "Order", Package: "a", FilePath: "a/Order.java"},
		{Name: "Status", Package: "a", Outer: "Order", FilePath: "a/Order.java", IsEnum: true, EnumConstants: []string{"A1"}},
		{Name: "Order", Package: "b", FilePath: "b/Order.java"},
		{Name: "Status", Package: "b", Outer: "Order", FilePath: "b/Order.java", IsEnum: true, EnumConstants: []string{"B1"}},
		{Name: "Holder", Package: "a", FilePath: "a/Holder.java", Fields: fields},
		{Name: "Holder", Package: "b", FilePath: "b/Holder.java", Fields: fields},
	})

	for holder, want := range map[string]map[string][]any{
		"a.Holder": {"code": {"A"}, "status": {"A1"}},
		"b.Holder": {"code": {"B"}, "status": {"B1"}},
	} {
		got := r.Resolve(holder, nil)
		for name, values := range want {
			if options := optionValues(got.Fields[name].Options); !reflect.DeepEqual(options, values) {
				t.Errorf("%s.%s options = %v, want %v", holder, name, options, values)
			}
		}
	}
}
