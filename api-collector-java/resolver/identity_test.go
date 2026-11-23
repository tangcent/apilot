package resolver

import (
	"sort"
	"strings"
	"testing"

	collector "github.com/tangcent/apilot/api-collector"
	"github.com/tangcent/apilot/api-collector-java/parser"
)

// What a name is, is what the scope of the file that writes it says: a class of
// the project that bears the name of a class of the JDK or of a library is
// another class, wherever it sits, and the class of the library is the class
// however a file reaches it. The tests below ask the resolver for the identity
// of each name of its table of library classes in every scope, and read the
// wrappers (Mono, Flux, ResponseEntity), the holders of a value (Optional) and
// the collections by that identity.

func libraryNameList() []string {
	names := make([]string, 0, len(libraryPackages))
	for name := range libraryPackages {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// The identity of a name is the class its scope binds it to, by the order of the
// ladder, and needs no dependency to be known.
func TestIdentity_IsTheClassTheScopeBinds(t *testing.T) {
	const own, other = "com.example.own", "com.example.other"
	for _, name := range libraryNameList() {
		project := parser.Class{Name: name, Package: own, FilePath: "own/" + name + ".java"}
		enum := parser.Class{Name: name, Package: own, FilePath: "own/" + name + ".java", IsEnum: true, EnumConstants: []string{"A"}}
		for _, pkg := range libraryPackages[name] {
			library := pkg + "." + name
			for scope, tc := range map[string]struct {
				source  parser.Class
				classes []parser.Class
				want    string
			}{
				"library, single-type import":  {importing("app", library), nil, library},
				"library, import on demand":    {importing("app", pkg+".*"), nil, library},
				"library, written in full":     {importing("app"), nil, library},
				"project class, same package":  {importing(own, pkg+".*"), []parser.Class{project}, own + "." + name},
				"project class, imported":      {importing("app", own+"."+name, pkg+".*"), []parser.Class{project}, own + "." + name},
				"project class, on demand":     {importing("app", own+".*"), []parser.Class{project}, own + "." + name},
				"project enum, same package":   {importing(own, pkg+".*"), []parser.Class{enum}, own + "." + name},
				"project enum, imported":       {importing("app", own+"."+name), []parser.Class{enum}, own + "." + name},
				"library import beats a class": {importing("app", library), []parser.Class{project}, library},
				"library on demand, class of an unreachable package": {importing("app", pkg+".*"),
					[]parser.Class{{Name: name, Package: other, FilePath: "other/" + name + ".java"}}, library},
				"another library's class": {importing("app", "org.example.legacy."+name, pkg+".*"), nil, "org.example.legacy." + name},
			} {
				typeName := name
				if strings.HasPrefix(scope, "library, written in full") {
					typeName = library
				}
				r := NewTypeResolver(append(append([]parser.Class(nil), tc.classes...), tc.source))
				if got := r.Identity(tc.source, typeName); got != tc.want {
					t.Errorf("%s: Identity(%s) = %s, want %s", scope, typeName, got, tc.want)
				}
				if tc.want == library && !IsLibraryClass(tc.want) {
					t.Errorf("%s is not a library class", tc.want)
				}
				if tc.want != library && IsLibraryClass(tc.want) {
					t.Errorf("%s is a library class, though the name denotes another", tc.want)
				}
			}
		}
	}
}

// A source that is no compilation unit binds nothing: a name the table knows is
// the class it is known for, and a name it does not know stays as written.
func TestIdentity_AnUnscopedSourceReadsAKnownNameAsTheLibrarysClass(t *testing.T) {
	r := NewTypeResolver(nil)
	for name, want := range map[string]string{
		"Mono": "reactor.core.publisher.Mono", "ResponseEntity": "org.springframework.http.ResponseEntity",
		"MultipartFile": MultipartFileClass, "Order": "Order", "int": "int", "String": "java.lang.String",
	} {
		if got := r.Identity(parser.Class{}, name); got != want {
			t.Errorf("Identity(%s) of an unscoped source = %s, want %s", name, got, want)
		}
	}
	// A compilation unit that imports nothing binds nothing for these names.
	scoped := importing("com.example")
	for _, name := range []string{"Mono", "ResponseEntity", "MultipartFile", "Optional"} {
		if got := r.Identity(scoped, name); IsLibraryClass(got) {
			t.Errorf("Identity(%s) without an import = %s, want a name nothing binds", name, got)
		}
	}
}

// The wrappers carry the payload as their argument when they are the library's
// classes, and are classes of their own when the project declares them.
func TestResolve_WrappersAreWrappersByIdentity(t *testing.T) {
	const own = "com.example.own"
	wrappers := map[string]string{
		"Mono":           "reactor.core.publisher",
		"Flux":           "reactor.core.publisher",
		"ResponseEntity": "org.springframework.http",
	}
	order := parser.Class{Name: "Order", Package: own, FilePath: "own/Order.java", Fields: []parser.Field{{Name: "id", Type: "long"}}}
	for name, pkg := range wrappers {
		spelling := name + "<Order>"
		want := "object:Order"
		if name == "Flux" {
			want = "array<object:Order>"
		}
		project := parser.Class{
			Name: name, Package: own, FilePath: "own/" + name + ".java", TypeParameters: []string{"T"},
			Fields: []parser.Field{{Name: "own", Type: "T"}},
		}
		for scope, tc := range map[string]struct {
			source  parser.Class
			classes []parser.Class
			spell   string
			want    string
		}{
			"single-type import":  {importing(own, pkg+"."+name), []parser.Class{order}, spelling, want},
			"import on demand":    {importing(own, pkg+".*"), []parser.Class{order}, spelling, want},
			"written in full":     {importing(own), []parser.Class{order}, pkg + "." + name + "<Order>", want},
			"nested in a wrapper": {importing(own, pkg+"."+name, "java.util.List"), []parser.Class{order}, "List<" + spelling + ">", "array<" + want + ">"},
			"project class":       {importing(own), []parser.Class{order, project}, spelling, "object:" + name},
			"project class, a wrapper's package imported": {importing(own, pkg+".*"), []parser.Class{order, project}, spelling, "object:" + name},
		} {
			r := NewTypeResolver(append(append([]parser.Class(nil), tc.classes...), tc.source))
			r.SetUnresolved(collector.NewUnresolvedSet())
			if got := arrayShape(r.ResolveFrom(tc.source, tc.spell, nil)); got != tc.want {
				t.Errorf("%s, %s: %s = %s, want %s", name, scope, tc.spell, got, tc.want)
			}
		}

		// As the argument of a generic class that declares a wrapper of the same
		// name in its own package: the argument is the caller's.
		libWrapper := parser.Class{
			Name: name, Package: "com.example.lib", FilePath: "lib/" + name + ".java", TypeParameters: []string{"T"},
			Fields: []parser.Field{{Name: "shadow", Type: "T"}},
		}
		box := parser.Class{
			Name: "Box", Package: "com.example.lib", FilePath: "lib/Box.java", TypeParameters: []string{"T"},
			Fields: []parser.Field{{Name: "data", Type: "T"}},
		}
		caller := importing(own, pkg+"."+name, "com.example.lib.Box")
		r := NewTypeResolver([]parser.Class{order, libWrapper, box, caller})
		got := typeArgShape(typeArgAt(r.ResolveFrom(caller, "Box<"+spelling+">", nil), "data"))
		wantShape := "Order(id)"
		if name == "Flux" {
			wantShape = "[Order(id)]"
		}
		if got != wantShape {
			t.Errorf("%s as an argument beside the library's %s class: %s, want %s", name, name, got, wantShape)
		}
	}
}

// The helpers a handler's parser asks: an Optional, a file, a class among
// several, each by the class its name denotes.
func TestIsOptionalTypeAndIsMultipartFile_ReadTheClassTheNameDenotes(t *testing.T) {
	const own = "com.example.own"
	bean := func(name string) parser.Class {
		return parser.Class{Name: name, Package: own, FilePath: "own/" + name + ".java", TypeParameters: []string{"T"}}
	}
	for scope, tc := range map[string]struct {
		source   parser.Class
		classes  []parser.Class
		optional bool
		file     bool
	}{
		"the JDK's and Spring's": {importing("app", "java.util.Optional", "java.util.List", "org.springframework.web.multipart.MultipartFile"), nil, true, true},
		"on demand":              {importing("app", "java.util.*", "org.springframework.web.multipart.*"), nil, true, true},
		"the project's":          {importing(own), []parser.Class{bean("Optional"), bean("MultipartFile")}, false, false},
		"the project's, imported": {importing("app", own+".Optional", own+".MultipartFile", "java.util.*", "org.springframework.web.multipart.*"),
			[]parser.Class{bean("Optional"), bean("MultipartFile")}, false, false},
	} {
		r := NewTypeResolver(append(append([]parser.Class(nil), tc.classes...), tc.source))
		if got := r.IsOptionalType(tc.source, "Optional<String>"); got != tc.optional {
			t.Errorf("%s: IsOptionalType(Optional<String>) = %v, want %v", scope, got, tc.optional)
		}
		for _, typeName := range []string{"MultipartFile", "MultipartFile[]", "List<MultipartFile>", "Optional<MultipartFile>", "Optional<List<MultipartFile>>"} {
			if got := r.IsMultipartFile(tc.source, typeName); got != tc.file {
				t.Errorf("%s: IsMultipartFile(%s) = %v, want %v", scope, typeName, got, tc.file)
			}
		}
	}
	// Written in full, the name needs no import.
	r := NewTypeResolver(nil)
	source := importing("app")
	for typeName, want := range map[string]bool{
		"org.springframework.web.multipart.MultipartFile":                       true,
		"java.util.List<org.springframework.web.multipart.MultipartFile>":       true,
		"java.util.Optional<org.springframework.web.multipart.MultipartFile[]>": true,
		"MultipartFile":                                     false,
		"List<MultipartFile>":                               false,
		"com.example.web.MultipartFile":                     false,
		"java.util.Optional<com.example.web.MultipartFile>": false,
	} {
		if got := r.IsMultipartFile(source, typeName); got != want {
			t.Errorf("IsMultipartFile(%s) = %v, want %v", typeName, got, want)
		}
	}
}
