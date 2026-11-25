package resolver

import (
	"fmt"
	"sort"
	"strings"
	"testing"

	collector "github.com/tangcent/apilot/api-collector"
	"github.com/tangcent/apilot/api-collector-java/parser"
	model "github.com/tangcent/apilot/api-model"
)

// The tests below read what the export says of a field, as a short spelling:
//
//	str[A,B]  a string whose options are A and B
//	str       a string without options
//	int[0,1]  an integer whose options are 0 and 1
//	unres(X)  a type the resolver could not expand, named X
//	obj(X:a|b) an object of class X with the fields a and b
//	arr<..>, map<k,v>
//
// Every field is read twice, by a resolver of its own and by one that has read the
// fields before it, because what a name denotes must not depend on what the
// resolver has read.

type scopeRead struct {
	holder, field, want string
}

func describeType(m *model.ObjectModel) string {
	switch {
	case m == nil:
		return "<nil>"
	case m.Kind == model.KindArray:
		return "arr<" + describeType(m.Items) + ">"
	case m.Kind == model.KindMap:
		return "map<" + describeType(m.KeyModel) + "," + describeType(m.ValueModel) + ">"
	case m.Kind == model.KindObject:
		names := make([]string, 0, len(m.Fields))
		for name := range m.Fields {
			names = append(names, name)
		}
		sort.Strings(names)
		return "obj(" + m.TypeName + ":" + strings.Join(names, "|") + ")"
	case m.Kind == model.KindRef:
		return "ref(" + m.TypeName + ")"
	case m.TypeName == model.JsonTypeString:
		return "str"
	case m.TypeName == model.JsonTypeInt:
		return "int"
	}
	return "unres(" + m.TypeName + ")"
}

func describeField(f *model.FieldModel) string {
	if f == nil {
		return "<nofield>"
	}
	described := describeType(f.Model)
	if len(f.Options) > 0 {
		values := make([]string, len(f.Options))
		for i, option := range f.Options {
			values[i] = fmt.Sprint(option.Value)
		}
		if strings.HasPrefix(described, "arr<") {
			// The options belong to the items, as they are listed on the field.
			return described + "[" + strings.Join(values, ",") + "]"
		}
		described += "[" + strings.Join(values, ",") + "]"
	}
	return described
}

// libraryOf serves the declarations of the sources as the classes of a
// dependency: without a file path, and by qualified name only.
func libraryOf(t *testing.T, files map[string]string) *libraryClasses {
	t.Helper()
	library := &libraryClasses{classes: map[string]parser.Class{}}
	for _, c := range parseFiles(t, files) {
		c.FilePath = ""
		library.classes[c.QualifiedName()] = c
	}
	return library
}

// checkReads reads the fields in order, in the JSON view or in the binding view.
// library is nil for a run without a dependency resolver.
func checkReads(t *testing.T, label string, classes []parser.Class, library func() DependencyResolver, binding bool, reads ...scopeRead) {
	t.Helper()
	newResolver := func() *TypeResolver {
		r := NewTypeResolver(classes)
		r.SetUnresolved(collector.NewUnresolvedSet())
		if library != nil {
			r.SetDependencyResolver(library())
		}
		return r
	}
	read := func(r *TypeResolver, rd scopeRead) string {
		var resolved *model.ObjectModel
		if binding {
			resolved = r.ResolveForBinding(parser.Class{}, rd.holder, nil)
		} else {
			resolved = r.Resolve(rd.holder, nil)
		}
		if resolved == nil || resolved.Kind != model.KindObject {
			return "<holder: " + describeType(resolved) + ">"
		}
		return describeField(resolved.Fields[rd.field])
	}
	view := "JSON view"
	if binding {
		view = "binding view"
	}
	warm := newResolver()
	for _, rd := range reads {
		fresh := read(newResolver(), rd)
		if fresh != rd.want {
			t.Errorf("%s, %s: %s.%s = %s, want %s", label, view, rd.holder, rd.field, fresh, rd.want)
		}
		if got := read(warm, rd); got != fresh {
			t.Errorf("%s, %s: %s.%s = %s after reading the fields before it, and %s on a resolver of its own", label, view, rd.holder, rd.field, got, fresh)
		}
	}
}

func noLibrary() DependencyResolver { return &libraryClasses{} }

// A name only binds to an enum through the scope: member types, the file, a
// single-type import, the package, an import on demand. An import on demand of a
// package the sources do not hold, or java.lang, may carry the name as well, and
// the enum of some other package that shares it is not the type written. What a
// name nothing declares denotes is what it denotes without the enum: unresolved,
// or the scalar of the JDK type of that name (java.util.Currency, imported on
// demand, is a string, and has none of the constants of the enum of that name).
func TestResolve_ANameNoScopeDeclaresIsNotTakenForAnEnumOfThatName(t *testing.T) {
	classes := parseFiles(t, map[string]string{
		"other/Period.java":   "package other; public enum Period { DAY, WEEK }",
		"other/Currency.java": "package other; public enum Currency { CNY, USD }",
		"other/Date.java":     "package other; public enum Date { TODAY }",
		"other/Thread.java":   "package other; public enum Thread { MAIN }",
		"other/Status.java":   "package other; public enum Status { OTHER }",
		"app/Jdk.java": `package app;
import java.time.*;
import java.util.*;
class Jdk { Period period; Currency currency; Date date; Thread thread; LocalDate day; }
`,
		"app/Single.java": `package app;
import java.time.Period;
class Single { Period period; }
`,
		"app/Unscanned.java": `package app;
import thirdparty.model.*;
class Unscanned { Status status; java.util.List<Status> statuses; Status[] array; }
`,
	})
	for _, library := range []func() DependencyResolver{nil, noLibrary} {
		for _, binding := range []bool{false, true} {
			checkReads(t, "an import on demand", classes, library, binding,
				scopeRead{"app.Jdk", "currency", "str"},
				scopeRead{"app.Jdk", "thread", "unres(Thread)"},
				scopeRead{"app.Unscanned", "status", "unres(Status)"},
				scopeRead{"app.Unscanned", "array", "arr<unres(Status)>"},
			)
		}
		// The JDK value types keep their scalar, in the JSON view.
		checkReads(t, "an import on demand", classes, library, false,
			scopeRead{"app.Jdk", "period", "str"},
			scopeRead{"app.Jdk", "date", "str"},
			scopeRead{"app.Jdk", "day", "str"},
			scopeRead{"app.Single", "period", "str"},
		)
	}
}

// An import of a package of the dependencies, on demand or of the class by name,
// names the enum of the library, not a project enum that carries the simple name.
// Which of them the resolver has read does not change that.
func TestResolve_AnEnumOfTheDependenciesIsNotReplacedByAProjectEnumOfItsName(t *testing.T) {
	classes := parseFiles(t, map[string]string{
		"other/Status.java": "package other; public enum Status { OTHER }",
		"app/Star.java":     "package app; import lib.*; class Star { Status s; }",
		"app/Single.java":   "package app; import lib.Status; class Single { Status s; }",
	})
	library := func() DependencyResolver {
		return libraryOf(t, map[string]string{"lib/Status.java": "package lib; public enum Status { LIB }"})
	}
	checkReads(t, "an import of lib", classes, library, false,
		scopeRead{"app.Star", "s", "str[LIB]"},
		scopeRead{"app.Single", "s", "str[LIB]"},
		scopeRead{"app.Star", "s", "str[LIB]"},
	)
}

// A class read from a library is a compilation unit of its own: the Status its
// field names is the Status of its package, not the Status of the project, and the
// enum nested in it is not the enum of the project that carries its name. Whether
// the library's enum has been read does not change that.
func TestResolve_AClassOfALibraryDoesNotTakeTheEnumsOfTheProject(t *testing.T) {
	classes := parseFiles(t, map[string]string{
		"app/Status.java": "package app; public enum Status { APP }",
		"app/Kind.java":   "package app; public enum Kind { KAPP }",
		"app/H.java":      "package app; import lib.Order; import lib.Ticket; class H { Order o; Ticket t; }",
	})
	library := func() DependencyResolver {
		return libraryOf(t, map[string]string{
			"lib/Order.java":  "package lib; import java.util.List; public class Order { public Status status; public List<Status> list; }",
			"lib/Status.java": "package lib; public enum Status { LIB }",
			"lib/Ticket.java": "package lib; public class Ticket { public enum Kind { K1 } public Kind kind; }",
		})
	}
	for _, warm := range []bool{false, true} {
		r := NewTypeResolver(classes)
		r.SetUnresolved(collector.NewUnresolvedSet())
		r.SetDependencyResolver(library())
		if warm {
			// The library's enum has been read already.
			r.Resolve("lib.Status", nil)
		}
		h := r.Resolve("app.H", nil)
		if got := describeField(h.Fields["o"].Model.Fields["status"]); got != "str[LIB]" {
			t.Errorf("warm=%v: lib.Order.status = %s, want str[LIB], not the enum of the project", warm, got)
		}
		if got := describeField(h.Fields["o"].Model.Fields["list"]); got != "arr<str>[LIB]" {
			t.Errorf("warm=%v: lib.Order.list = %s, want arr<str>[LIB]", warm, got)
		}
		if got := describeField(h.Fields["t"].Model.Fields["kind"]); got != "unres(Kind)" {
			t.Errorf("warm=%v: lib.Ticket.kind = %s, want unres(Kind), not the app.Kind of the project", warm, got)
		}
	}
}
