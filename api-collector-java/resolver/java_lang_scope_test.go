package resolver

import (
	"testing"

	collector "github.com/tangcent/apilot/api-collector"
	"github.com/tangcent/apilot/api-collector-java/parser"
)

// javaLangNames are the java.lang classes the resolver answers for without a
// declaration, each with the shape the JDK class exports.
var javaLangNames = map[string]string{
	"String":    "string",
	"Integer":   "int",
	"Short":     "int",
	"Long":      "long",
	"Float":     "float",
	"Double":    "double",
	"Boolean":   "boolean",
	"Void":      "null",
	"Number":    "double",
	"Byte":      "int",
	"Character": "string",
	"Object":    "object:object",
}

// projectType declares a class of the scanned sources that carries a java.lang
// name and has a field of its own, so an export of the project class is told
// from one of the JDK type by the field set.
func projectType(pkg, name string) parser.Class {
	return parser.Class{
		Name:     name,
		Package:  pkg,
		FilePath: "src/" + pkg + "/" + name + ".java",
		Fields:   []parser.Field{{Name: "digits", Type: "String"}},
	}
}

// Java looks a simple name up in the declarations of the compilation unit, then
// in a single-type import, then in the package, then in the imports on demand,
// and java.lang is the last of those (JLS 6.4.1). A class of the project that
// bears a java.lang name hides the JDK class in every file that sees it, so the
// name is the project class there and not the scalar the JDK class would be.
func TestResolve_ProjectClassHidesJavaLangName(t *testing.T) {
	for name := range javaLangNames {
		t.Run(name, func(t *testing.T) {
			hidden := projectType("domain", name)

			cases := map[string]parser.Class{
				"single-type import": {
					Name: "Form", Package: "app", FilePath: "src/app/Form.java",
					Imports: []parser.Import{{Path: "domain." + name}},
				},
				"import on demand": {
					Name: "Form", Package: "app", FilePath: "src/app/Form.java",
					Imports: []parser.Import{{Path: "domain", OnDemand: true}},
				},
				"same package": {
					Name: "Form", Package: "domain", FilePath: "src/domain/Form.java",
				},
			}
			for scope, form := range cases {
				r := NewTypeResolver([]parser.Class{hidden, form})
				for _, spelling := range []string{name, name + "[]", "List<" + name + ">"} {
					resolved := r.ResolveFrom(form, spelling, nil)
					want := "object:" + name
					if spelling != name {
						want = "array<object:" + name + ">"
					}
					if got := arrayShape(resolved); got != want {
						t.Errorf("%s, %s = %s, want %s", scope, spelling, got, want)
					}
				}
			}
		})
	}
}

// A class declared in the same file hides the JDK class, nested or not.
func TestResolve_ClassOfTheSameFileHidesJavaLangName(t *testing.T) {
	form := parser.Class{Name: "Form", Package: "app", FilePath: "src/app/Form.java"}
	nested := parser.Class{
		Name: "Short", Package: "app", FilePath: "src/app/Form.java",
		Fields: []parser.Field{{Name: "digits", Type: "String"}},
	}
	r := NewTypeResolver([]parser.Class{form, nested})

	if got := arrayShape(r.ResolveFrom(form, "Short", nil)); got != "object:Short" {
		t.Errorf("Short = %s, want object:Short", got)
	}
}

// A class of the default package is in scope for the files of that package.
func TestResolve_DefaultPackageClassHidesJavaLangName(t *testing.T) {
	r := NewTypeResolver([]parser.Class{
		{Name: "Short", FilePath: "src/Short.java", Fields: []parser.Field{{Name: "digits", Type: "String"}}},
		{Name: "Form", FilePath: "src/Form.java"},
	})

	if got := arrayShape(r.ResolveFrom(parser.Class{Name: "Form", FilePath: "src/Form.java"}, "Short", nil)); got != "object:Short" {
		t.Errorf("Short = %s, want object:Short", got)
	}
}

// A project class of the same name that the file cannot see does not take the
// JDK class's place: it is neither imported, nor in the package, nor in a
// package imported on demand.
func TestResolve_UnrelatedProjectClassDoesNotHideJavaLangName(t *testing.T) {
	form := parser.Class{
		Name: "Form", Package: "app", FilePath: "src/app/Form.java",
		Imports: []parser.Import{{Path: "other", OnDemand: true}},
	}
	classes := []parser.Class{form}
	for name := range javaLangNames {
		classes = append(classes, projectType("domain", name))
	}
	r := NewTypeResolver(classes)
	r.SetUnresolved(collector.NewUnresolvedSet())

	for name, want := range javaLangNames {
		if got := arrayShape(r.ResolveFrom(form, name, nil)); got != want {
			t.Errorf("%s = %s, want %s", name, got, want)
		}
	}
	if unresolved := r.Unresolved(); len(unresolved) != 0 {
		t.Errorf("unresolved = %v, want none", unresolved)
	}
}

// Importing the JDK class by name, as some projects do, names the JDK class.
func TestResolve_ImportOfTheJavaLangClassNamesTheJdkClass(t *testing.T) {
	form := parser.Class{
		Name: "Form", Package: "domain", FilePath: "src/domain/Form.java",
		Imports: []parser.Import{{Path: "java.lang.Short"}, {Path: "java.lang", OnDemand: true}},
	}
	r := NewTypeResolver([]parser.Class{projectType("domain", "Short"), form})
	r.SetUnresolved(collector.NewUnresolvedSet())

	if got := arrayShape(r.ResolveFrom(form, "Short", nil)); got != "int" {
		t.Errorf("Short = %s, want int", got)
	}
	if unresolved := r.Unresolved(); len(unresolved) != 0 {
		t.Errorf("unresolved = %v, want none", unresolved)
	}
}

// A qualified name needs no scope: java.lang.Short is the JDK class even where
// a class named Short is in the package, and domain.Short is the project class
// wherever it is written.
func TestResolve_QualifiedNameIgnoresTheScope(t *testing.T) {
	form := parser.Class{Name: "Form", Package: "domain", FilePath: "src/domain/Form.java"}
	r := NewTypeResolver([]parser.Class{projectType("domain", "Short"), form})

	if got := arrayShape(r.ResolveFrom(form, "java.lang.Short", nil)); got != "int" {
		t.Errorf("java.lang.Short = %s, want int", got)
	}
	if got := arrayShape(r.ResolveFrom(form, "domain.Short", nil)); got != "object:Short" {
		t.Errorf("domain.Short = %s, want object:Short", got)
	}
}

// The project class of a generic type argument hides the JDK class in the file
// that writes the argument.
func TestResolve_ProjectClassHidesJavaLangNameInTypeArguments(t *testing.T) {
	form := parser.Class{
		Name: "Form", Package: "app", FilePath: "src/app/Form.java",
		Imports: []parser.Import{{Path: "domain.Short"}},
	}
	r := NewTypeResolver([]parser.Class{projectType("domain", "Short"), form})

	for spelling, want := range map[string]string{
		"Map<String, Short>":      "map<object:Short>",
		"ResponseEntity<Short[]>": "array<object:Short>",
		"Mono<Short>":             "object:Short",
		"? extends Short":         "object:Short",
	} {
		if got := arrayShape(r.ResolveFrom(form, spelling, nil)); got != want {
			t.Errorf("%s = %s, want %s", spelling, got, want)
		}
	}
}
