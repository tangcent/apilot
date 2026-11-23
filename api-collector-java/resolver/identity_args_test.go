package resolver

import (
	"testing"

	collector "github.com/tangcent/apilot/api-collector"
	"github.com/tangcent/apilot/api-collector-java/parser"
)

// A type argument names its class by identity wherever it is read, and what is
// shown of the type, or reported when it cannot be resolved, is the name the
// source wrote.

const (
	identityApp = "com.example.idapp"
	identityLib = "com.example.idlib"
)

// identityBox is a generic class of a library whose compilation unit imports the
// classes of the names under test from another package.
func identityBox(imports ...string) parser.Class {
	box := parser.Class{
		Name: "Box", Package: identityLib, FilePath: "idlib/Box.java", TypeParameters: []string{"T"},
		Fields: []parser.Field{{Name: "data", Type: "T"}, {Name: "list", Type: "List<T>"}},
	}
	for _, path := range imports {
		box.Imports = append(box.Imports, parser.Import{Path: path})
	}
	return box
}

func identityCaller(imports ...string) parser.Class {
	caller := parser.Class{Name: "Caller", Package: identityApp, FilePath: "idapp/Caller.java"}
	caller.Imports = append(caller.Imports, parser.Import{Path: identityLib + ".Box"})
	for _, path := range imports {
		caller.Imports = append(caller.Imports, parser.Import{Path: path})
	}
	return caller
}

// A class of the JDK that the caller imports stays that class in a generic
// class whose compilation unit imports another class of the same name, and the
// other way round.
func TestResolve_ATypeArgumentIsNotReboundByTheGenericClassesImports(t *testing.T) {
	for view, resolveIn := range typeArgViews {
		t.Run(view, func(t *testing.T) {
			// The caller means java.time.LocalDate, the library's Box another one.
			jdkInCaller := NewTypeResolver([]parser.Class{identityBox("com.other.LocalDate")})
			jdkInCaller.SetUnresolved(collector.NewUnresolvedSet())
			box := resolveIn(jdkInCaller, identityCaller("java.time.LocalDate"), "Box<LocalDate>")
			for _, path := range []string{"data", "list"} {
				if shape := typeArgShape(typeArgAt(box, path)); shape != map[string]string{"data": "string", "list": "[string]"}[path] {
					t.Errorf("Box<LocalDate>.%s = %s, want the JDK's LocalDate, a string", path, shape)
				}
			}
			if unresolved := jdkInCaller.Unresolved(); len(unresolved) != 0 {
				t.Errorf("unresolved = %v, want none", unresolved)
			}

			// The caller means a class of another library, Box java.time.LocalDate.
			otherInCaller := NewTypeResolver([]parser.Class{identityBox("java.time.LocalDate")})
			otherInCaller.SetUnresolved(collector.NewUnresolvedSet())
			box = resolveIn(otherInCaller, identityCaller("com.other.LocalDate"), "Box<LocalDate>")
			if shape := typeArgShape(typeArgAt(box, "data")); shape != "LocalDate" {
				t.Errorf("Box<LocalDate>.data = %s, want the other library's class, shown as written", shape)
			}
			if unresolved := otherInCaller.Unresolved(); unresolved["LocalDate"] == 0 {
				t.Errorf("unresolved = %v, want LocalDate: no class of that name is known", unresolved)
			}
		})
	}
}

// A type that cannot be resolved is shown, and reported, as the source wrote it
// in the arguments of a generic class too: the classes it names are passed on in
// full, and that is no part of the name.
func TestResolve_AnUnresolvedTypeIsShownAsWrittenWhateverItsArgumentsAre(t *testing.T) {
	item := parser.Class{Name: "Item", Package: identityLib + ".model", FilePath: "idlib/model/Item.java",
		Fields: []parser.Field{{Name: "id", Type: "int"}}}
	for view, resolveIn := range typeArgViews {
		t.Run(view, func(t *testing.T) {
			r := NewTypeResolver([]parser.Class{identityBox(), item})
			r.SetUnresolved(collector.NewUnresolvedSet())
			caller := identityCaller("com.example.idlib.model.Item", "com.other.Pair", "com.other.Optional")
			for spelling, want := range map[string]string{
				"Optional<Item>":        "Optional<Item>",
				"Optional<String>":      "Optional<String>",
				"Pair<Integer,String>":  "Pair<Integer,String>",
				"Pair<Integer, String>": "Pair<Integer, String>",
				"Pair<Item,List<Item>>": "Pair<Item,List<Item>>",
			} {
				box := resolveIn(r, caller, "Box<"+spelling+">")
				if got := typeArgShape(typeArgAt(box, "data")); got != want {
					t.Errorf("Box<%s>.data = %s, want %s", spelling, got, want)
				}
			}
			for name := range r.Unresolved() {
				if len(name) > 0 && containsDot(name) {
					t.Errorf("unresolved %q names a package: it is reported as written", name)
				}
			}
		})
	}
}

func containsDot(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] == '.' {
			return true
		}
	}
	return false
}

// The scalar that a JDK value type serialises to belongs to the class of the
// JDK: a class of another library that has the same simple name is not one.
func TestResolve_AClassOfAnotherLibraryIsNotAJdkValueType(t *testing.T) {
	names := []string{"LocalDate", "Duration", "UUID", "Date", "BigDecimal", "AtomicInteger", "Instant"}
	jdk := map[string]string{
		"LocalDate": "java.time", "Duration": "java.time", "UUID": "java.util", "Date": "java.util",
		"BigDecimal": "java.math", "AtomicInteger": "java.util.concurrent.atomic", "Instant": "java.time",
	}
	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			r := NewTypeResolver(nil)
			r.SetUnresolved(collector.NewUnresolvedSet())
			field := func(imports ...string) parser.Class {
				c := parser.Class{Name: "Holder", Package: identityApp, FilePath: "idapp/Holder.java"}
				for _, path := range imports {
					c.Imports = append(c.Imports, parser.Import{Path: path})
				}
				return c
			}
			// Imported from the JDK, by name or on demand, or written in full.
			for how, source := range map[string]parser.Class{
				"single import": field(jdk[name] + "." + name),
				"on demand":     {Name: "Holder", Package: identityApp, FilePath: "idapp/Holder.java", Imports: []parser.Import{{Path: jdk[name], OnDemand: true}}},
			} {
				if got := r.ResolveFrom(source, name, nil); !got.IsSingle() || got.TypeName == name {
					t.Errorf("%s of the JDK: %s = %+v, want a scalar", how, name, got)
				}
			}
			if got := r.ResolveFrom(field(), jdk[name]+"."+name, nil); got.TypeName == jdk[name]+"."+name {
				t.Errorf("%s.%s written in full = %+v, want a scalar", jdk[name], name, got)
			}
			// Without any scope to tell the name apart.
			if got := r.Resolve(name, nil); got.TypeName == name {
				t.Errorf("%s without a scope = %+v, want a scalar", name, got)
			}
			// Imported from another library, or declared by the package.
			before := r.Unresolved()[name]
			got := r.ResolveFrom(field("com.other."+name), name, nil)
			if got.TypeName != name {
				t.Errorf("com.other.%s = %+v, want the class of the other library, which is not read", name, got)
			}
			if r.Unresolved()[name] != before+1 {
				t.Errorf("unresolved = %v, want %s reported", r.Unresolved(), name)
			}
		})
	}
}
