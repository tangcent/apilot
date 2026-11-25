package resolver

import (
	"testing"

	collector "github.com/tangcent/apilot/api-collector"
	"github.com/tangcent/apilot/api-collector-java/parser"
)

// libraryName is a JDK or library type that the resolver answers for by name:
// how it is spelled when it holds a value, the package a source imports on
// demand to get it, and what it exports as.
type libraryName struct {
	spelling string
	pkg      string
	want     string
}

var libraryNames = map[string]libraryName{
	"Optional":       {"Optional<String>", "java.util", "string"},
	"OptionalInt":    {"OptionalInt", "java.util", "int"},
	"OptionalLong":   {"OptionalLong", "java.util", "long"},
	"OptionalDouble": {"OptionalDouble", "java.util", "double"},
	"Locale":         {"Locale", "java.util", "string"},
	"Currency":       {"Currency", "java.util", "string"},
	"TimeZone":       {"TimeZone", "java.util", "string"},
	"DayOfWeek":      {"DayOfWeek", "java.time", "string"},
	"Month":          {"Month", "java.time", "string"},
	"Timestamp":      {"Timestamp", "java.sql", "string"},
	"URL":            {"URL", "java.net", "string"},
	"JsonNode":       {"JsonNode", "com.fasterxml.jackson.databind", "object:object"},
	"ObjectNode":     {"ObjectNode", "com.fasterxml.jackson.databind.node", "object:object"},
	"ArrayNode":      {"ArrayNode", "com.fasterxml.jackson.databind.node", "array<object:object>"},
	"MultipartFile":  {"MultipartFile", "org.springframework.web.multipart", "MultipartFile"},
	"List":           {"List<String>", "java.util", "array<string>"},
	"Set":            {"Set<String>", "java.util", "array<string>"},
	"Collection":     {"Collection<String>", "java.util", "array<string>"},
	"Iterable":       {"Iterable<String>", "java.lang", "array<string>"},
	"LinkedHashSet":  {"LinkedHashSet<String>", "java.util", "array<string>"},
	"SortedSet":      {"SortedSet<String>", "java.util", "array<string>"},
	"TreeSet":        {"TreeSet<String>", "java.util", "array<string>"},
	"Map":            {"Map<String, String>", "java.util", "map<string>"},
	"SortedMap":      {"SortedMap<String, String>", "java.util", "map<string>"},
}

// projectGeneric declares a class of the project that bears the name of a
// library type, generic when the library type is, and has a property of its own.
func projectGeneric(pkg, name, spelling string) parser.Class {
	class := projectType(pkg, name)
	if _, args := parseGenericType(spelling); len(args) > 0 {
		class.TypeParameters = []string{"T"}
	}
	return class
}

// A class of the project that bears the name of a JDK or library type hides it
// in every file that sees the class: the name is the class in a file of its
// package, in one that imports it by name or its package on demand, and not the
// type the table answers for. Before, Optional was unwrapped and a List was an
// array whatever class the name denoted.
func TestResolve_ProjectClassHidesLibraryName(t *testing.T) {
	for name, library := range libraryNames {
		t.Run(name, func(t *testing.T) {
			hidden := projectGeneric("domain", name, library.spelling)

			cases := map[string]parser.Class{
				"same package":       importing("domain", library.pkg+".*"),
				"single-type import": importing("app", "domain."+name, library.pkg+".*"),
				"import on demand":   importing("app", "domain.*", library.pkg+".*"),
			}
			for scope, source := range cases {
				r := NewTypeResolver([]parser.Class{hidden, source})
				if got := arrayShape(r.ResolveFrom(source, library.spelling, nil)); got != "object:"+name {
					t.Errorf("%s: %s = %s, want object:%s", scope, library.spelling, got, name)
				}
			}
		})
	}
}

// A class of the project that the file does not see is not the type its name
// denotes there: the file imports the package of the library type on demand, and
// that is where the name comes from. The class neither takes its place nor turns
// the name into an unknown one.
func TestResolve_UnrelatedProjectClassDoesNotHideLibraryName(t *testing.T) {
	for name, library := range libraryNames {
		t.Run(name, func(t *testing.T) {
			unrelated := projectGeneric("elsewhere", name, library.spelling)
			source := importing("app", library.pkg+".*")

			r := NewTypeResolver([]parser.Class{unrelated, source})
			r.SetUnresolved(collector.NewUnresolvedSet())
			resolved := r.ResolveFrom(source, library.spelling, nil)
			if got := arrayShape(resolved); got != library.want {
				t.Errorf("%s = %s, want %s", library.spelling, got, library.want)
			}
			if upload := r.IsUpload(resolved); upload != (name == "MultipartFile") {
				t.Errorf("%s: IsUpload = %v", library.spelling, upload)
			}
			if unresolved := r.Unresolved(); len(unresolved) != 0 {
				t.Errorf("unresolved = %v, want none", unresolved)
			}
		})
	}
}

// Importing another library's class of the same name by name says that the name
// is that class: a java.awt.List is no collection.
func TestResolve_ImportOfAnotherLibraryClassIsNotAJdkCollection(t *testing.T) {
	r := NewTypeResolver(nil)
	r.SetUnresolved(collector.NewUnresolvedSet())

	source := importing("app", "java.awt.List", "java.util.*")
	if got := arrayShape(r.ResolveFrom(source, "List<String>", nil)); got != "List<String>" {
		t.Errorf("java.awt.List = %s, want the opaque name", got)
	}
	if got := arrayShape(r.ResolveFrom(importing("app", "java.util.*"), "List<String>", nil)); got != "array<string>" {
		t.Errorf("java.util.List = %s, want array<string>", got)
	}
}

// An enum of the project that bears a library name hides it as well, although
// the JSON view does not read the enum, which declares @JsonValue: the name is
// not the JDK's.
func TestResolve_ProjectEnumHidesJdkNames(t *testing.T) {
	enum := func(name string) parser.Class {
		return parser.Class{
			Name: name, Package: "domain", FilePath: "src/domain/" + name + ".java",
			IsEnum: true, EnumConstants: []string{"A", "B"}, EnumJsonValue: true,
		}
	}
	source := importing("domain", "java.util.*", "java.time.*")
	for _, name := range []string{"Currency", "DayOfWeek", "Short", "Number", "List"} {
		r := NewTypeResolver([]parser.Class{enum(name), source})
		r.SetUnresolved(collector.NewUnresolvedSet())

		spelling := name
		if name == "List" {
			spelling = "List<String>"
		}
		if got := arrayShape(r.ResolveFrom(source, spelling, nil)); got != spelling {
			t.Errorf("project enum %s = %s, want the opaque name", spelling, got)
		}
	}
}

// What a type argument is follows from the file that writes it, which is where
// the project's class and the library's differ.
func TestResolve_LibraryNamesInTypeArgumentsFollowTheScope(t *testing.T) {
	r := NewTypeResolver([]parser.Class{projectType("domain", "Locale")})
	r.SetUnresolved(collector.NewUnresolvedSet())

	jdk := importing("app", "java.util.*")
	project := importing("app", "domain.Locale", "java.util.*")
	for _, tc := range []struct {
		source   parser.Class
		typeName string
		want     string
	}{
		{jdk, "List<Locale>", "array<string>"},
		{jdk, "Map<String, Locale>", "map<string>"},
		{project, "List<Locale>", "array<object:Locale>"},
		{project, "Map<String, Locale>", "map<object:Locale>"},
	} {
		if got := arrayShape(r.ResolveFrom(tc.source, tc.typeName, nil)); got != tc.want {
			t.Errorf("%s = %s, want %s", tc.typeName, got, tc.want)
		}
	}
}

// UnwrapOptional reads the type an Optional holds, and only java.util's.
func TestUnwrapOptional_OnlyJavaUtilOptional(t *testing.T) {
	own := projectGeneric("domain", "Optional", "Optional<String>")
	r := NewTypeResolver([]parser.Class{own})

	for _, tc := range []struct {
		name     string
		source   parser.Class
		typeName string
		want     string
	}{
		{"imported by name", importing("app", "java.util.Optional"), "Optional<String>", "String"},
		{"imported on demand", importing("app", "java.util.*"), "Optional<List<Long>>", "List<Long>"},
		{"imported on demand, next to a project package", importing("app", "other.*", "java.util.*"), "Optional<Long>", "Long"},
		{"written in full", importing("domain"), "java.util.Optional<Long>", "Long"},
		{"project class in the package", importing("domain", "java.util.*"), "Optional<String>", "Optional<String>"},
		{"project class imported by name", importing("app", "domain.Optional"), "Optional<String>", "Optional<String>"},
		{"project class imported on demand", importing("app", "domain.*"), "Optional<String>", "Optional<String>"},
		{"another library's Optional", importing("app", "com.google.common.base.Optional"), "Optional<String>", "Optional<String>"},
		{"not imported", importing("app"), "Optional<String>", "Optional<String>"},
		{"no type argument", importing("app", "java.util.Optional"), "Optional", "Optional"},
		{"not an Optional", importing("app", "java.util.Optional"), "List<String>", "List<String>"},
	} {
		if got := r.UnwrapOptional(tc.source, tc.typeName); got != tc.want {
			t.Errorf("%s: UnwrapOptional(%q) = %q, want %q", tc.name, tc.typeName, got, tc.want)
		}
	}
}
