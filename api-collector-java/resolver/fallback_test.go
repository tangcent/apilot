package resolver

import (
	"reflect"
	"testing"

	"github.com/tangcent/apilot/api-collector-java/parser"
)

// A name that no tier of the scope declares falls back to the class of the project
// that carries it, for sources parsed without scope information. A source that
// imports a package on demand which the project does not hold has scope
// information, and that package may well be where the name comes from: the class
// of another package that shares the name is a guess that it is not.
func TestResolve_ANameThatAnImportOnDemandOfAnUnscannedPackageMayNameIsNotGuessedAtInTheProject(t *testing.T) {
	sort := parser.Class{Name: "Sort", Package: "com.pr.lib", FilePath: "Sort.java", Fields: []parser.Field{{Name: "own", Type: "String"}}}
	source := func(imports ...parser.Import) parser.Class {
		return parser.Class{Name: "Ctl", Package: "com.pr.web", FilePath: "Ctl.java", Imports: imports}
	}

	for _, tc := range []struct {
		name    string
		imports []parser.Import
		want    string
	}{
		{"no import tells where the name is from", nil, "com.pr.lib.Sort"},
		{"single-type imports of other classes", []parser.Import{{Path: "com.other.Page"}}, "com.pr.lib.Sort"},
		{"a package of the project, imported on demand", onDemand("com.pr.lib"), "com.pr.lib.Sort"},
		{"a package that the project does not hold, imported on demand", onDemand("com.other"), "Sort"},
		{"that, among packages of the project", onDemand("com.pr.lib", "com.other"), "com.pr.lib.Sort"},
		{"a package of the JDK, imported on demand", onDemand("java.util"), "Sort"},
		{"a static import on demand names members", []parser.Import{{Path: "com.other.Statics", OnDemand: true, Static: true}}, "com.pr.lib.Sort"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := NewTypeResolver([]parser.Class{sort})
			if got := r.resolveName(source(tc.imports...), "Sort"); got != tc.want {
				t.Errorf("resolveName(Sort) = %q, want %q", got, tc.want)
			}
		})
	}

	t.Run("the dependencies hold the class the package names", func(t *testing.T) {
		library := &libraryClasses{classes: map[string]parser.Class{"com.other.Sort": marked("com.other", "Sort", "fromOther")}}
		r := NewTypeResolver([]parser.Class{sort})
		r.SetDependencyResolver(library)
		ctl := source(onDemand("com.other")...)

		if got := fieldsOf(r.ResolveFrom(ctl, "Sort", nil)); !reflect.DeepEqual(got, []string{"fromOther"}) {
			t.Errorf("Sort has %v, want the class of com.other", got)
		}
	})

	t.Run("a class of the project is a declaration of the scanned sources", func(t *testing.T) {
		// The class that the scope names is found as before: this guards the tiers,
		// which are tried before the fallback.
		neighbour := parser.Class{Name: "Page", Package: "com.pr.web", FilePath: "Page.java"}
		r := NewTypeResolver([]parser.Class{sort, neighbour})
		if got := r.resolveName(source(onDemand("com.other")...), "Page"); got != "com.pr.web.Page" {
			t.Errorf("resolveName(Page) = %q, want the class of the package", got)
		}
	})
}
