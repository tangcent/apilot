package resolver

import (
	"sort"
	"strings"
	"testing"

	collector "github.com/tangcent/apilot/api-collector"
	"github.com/tangcent/apilot/api-collector-java/parser"
)

// A name that denotes a class of the JDK in the scope of the caller is that class
// wherever it travels. A type argument is bound to a type parameter and resolved
// again inside the generic class, whose package may declare a class of the same
// simple name; the argument is therefore spelled once, in the caller's scope, as
// the identity of the class it names.
//
// The tests below give every JDK name the generic classes' package declares a
// class of its own, and ask which one the callers get, by how each caller
// reaches the name: implicitly (java.lang), by a single-type import, by an
// import on demand, or through a class of the project that hides it.

const (
	jdkLibPackage = "com.example.jdklib"
	jdkAppPackage = "com.example.jdkapp"
	jdkDtoPackage = "com.example.jdkapp.dto"
	jdkOwnPackage = "com.example.jdkown"
)

// jdkNames are the classes of the JDK the resolver answers for: the shape each
// exports, and the package of those outside java.lang.
var jdkNames = map[string]struct{ shape, pkg string }{
	"String": {"string", "java.lang"}, "Integer": {"int", "java.lang"}, "Short": {"int", "java.lang"},
	"Long": {"long", "java.lang"}, "Float": {"float", "java.lang"}, "Double": {"double", "java.lang"},
	"Boolean": {"boolean", "java.lang"}, "Void": {"null", "java.lang"}, "Number": {"double", "java.lang"},
	"Byte": {"int", "java.lang"}, "Character": {"string", "java.lang"}, "Object": {"object()", "java.lang"},
	"Date": {"string", "java.util"}, "UUID": {"string", "java.util"},
	"LocalDateTime": {"string", "java.time"}, "BigDecimal": {"double", "java.math"},
	"AtomicInteger": {"int", "java.util.concurrent.atomic"},
}

func jdkNameList(pkgs ...string) []string {
	var names []string
	for name, info := range jdkNames {
		for _, pkg := range pkgs {
			if info.pkg == pkg {
				names = append(names, name)
			}
		}
	}
	sort.Strings(names)
	return names
}

// jdkProjectClass is a class of the project that bears the name of a JDK class.
func jdkProjectClass(pkg, name, field string) parser.Class {
	return parser.Class{
		Name: name, Package: pkg, FilePath: strings.ReplaceAll(pkg, ".", "/") + "/" + name + ".java",
		Fields: []parser.Field{{Name: field, Type: "int"}},
	}
}

func jdkLibraryClass(name string, params []string, superClass string, superArgs []string, fields ...parser.Field) parser.Class {
	return parser.Class{
		Name: name, Package: jdkLibPackage, FilePath: "com/example/jdklib/" + name + ".java",
		TypeParameters: params, SuperClass: superClass, SuperClassTypeArgs: superArgs, Fields: fields,
	}
}

// jdkLibrary declares the generic classes, in a package that declares a class
// for every JDK name under test.
func jdkLibrary() []parser.Class {
	f := typeArgField
	classes := []parser.Class{
		jdkLibraryClass("Box", []string{"T"}, "", nil,
			f("data", "T"), f("list", "List<T>"), f("array", "T[]"),
			f("byKey", "Map<String, T>"), f("bounded", "List<? extends T>")),
		jdkLibraryClass("Wrapper", []string{"U"}, "", nil, f("data", "U")),
		jdkLibraryClass("Outer", []string{"T"}, "", nil,
			f("wrapped", "Wrapper<T>"), f("wrappedList", "Wrapper<List<T>>"),
			f("wrappedArray", "Wrapper<T[]>"), f("wrappedMap", "Wrapper<Map<String, T>>")),
		jdkLibraryClass("Base", []string{"B"}, "", nil,
			f("value", "B"), f("list", "List<B>"), f("array", "B[]"), f("wrapped", "Wrapper<B>")),
		jdkLibraryClass("Mid", []string{"M"}, "Base", []string{"M"}, f("midValue", "M")),
		// The library itself writes the JDK class in full.
		jdkLibraryClass("Holder", nil, "", nil,
			f("text", "java.lang.String"), f("when", "java.util.Date"), f("own", "String"),
			f("boxed", "Box<java.lang.String>"), f("boxedDate", "Box<java.util.Date>"),
			f("listed", "List<java.lang.Integer>"), f("array", "java.lang.Long[]")),
	}
	for name := range jdkNames {
		classes = append(classes, jdkProjectClass(jdkLibPackage, name, "shadow"))
	}
	return classes
}

// jdkCaller is a compilation unit that writes the type arguments, with the names
// it writes and the shape each of them has there.
type jdkCaller struct {
	name  string
	class parser.Class
	// shapes maps a name the caller writes to the shape it exports as.
	shapes map[string]string
}

func jdkCallers() []jdkCaller {
	generic := []parser.Import{
		{Path: jdkLibPackage + ".Box"}, {Path: jdkLibPackage + ".Outer"}, {Path: jdkLibPackage + ".Wrapper"},
		{Path: jdkLibPackage + ".Base"}, {Path: jdkLibPackage + ".Mid"}, {Path: jdkLibPackage + ".Holder"},
		{Path: "java.util.List"}, {Path: "java.util.Map"},
	}
	unit := func(pkg string, imports ...parser.Import) parser.Class {
		return parser.Class{
			Name: "Caller", Package: pkg, FilePath: strings.ReplaceAll(pkg, ".", "/") + "/Caller.java",
			Imports: append(append([]parser.Import(nil), generic...), imports...),
		}
	}
	jdkShapes := func(names []string) map[string]string {
		out := map[string]string{}
		for _, name := range names {
			out[name] = jdkNames[name].shape
		}
		return out
	}
	projectShapes := func(names []string) map[string]string {
		out := map[string]string{}
		for _, name := range names {
			out[name] = name + "(own)"
		}
		return out
	}
	var single, demand, ownImports, dtoDemand []parser.Import
	for _, name := range jdkNameList("java.util", "java.time", "java.math", "java.util.concurrent.atomic") {
		single = append(single, parser.Import{Path: jdkNames[name].pkg + "." + name})
		ownImports = append(ownImports, parser.Import{Path: jdkDtoPackage + "." + name})
	}
	for _, pkg := range []string{"java.util", "java.time", "java.math", "java.util.concurrent.atomic"} {
		demand = append(demand, parser.Import{Path: pkg, OnDemand: true})
	}
	dtoDemand = append(dtoDemand, parser.Import{Path: jdkDtoPackage, OnDemand: true})
	javaLang := jdkNameList("java.lang")
	beyond := jdkNameList("java.util", "java.time", "java.math", "java.util.concurrent.atomic")
	all := jdkNameList("java.lang", "java.util", "java.time", "java.math", "java.util.concurrent.atomic")

	var langImports []parser.Import
	for _, name := range javaLang {
		langImports = append(langImports, parser.Import{Path: "java.lang." + name})
	}

	return []jdkCaller{
		{"java.lang by default", unit(jdkAppPackage), jdkShapes(javaLang)},
		{"java.lang imported by name", unit(jdkAppPackage, langImports...), jdkShapes(javaLang)},
		{"java.lang imported on demand", unit(jdkAppPackage, parser.Import{Path: "java.lang", OnDemand: true}), jdkShapes(javaLang)},
		{"JDK imported by name", unit(jdkAppPackage, single...), jdkShapes(beyond)},
		{"JDK imported on demand", unit(jdkAppPackage, demand...), jdkShapes(beyond)},
		{"project class in the package", unit(jdkOwnPackage), projectShapes(all)},
		{"project class imported by name", unit(jdkAppPackage, ownImports...), projectShapes(beyond)},
		{"project class imported on demand", unit(jdkAppPackage, dtoDemand...), projectShapes(all)},
	}
}

// jdkWorld classes: the library, the project's own classes that bear JDK names,
// and an unrelated class of each name that no caller reaches.
func jdkWorldClasses() []parser.Class {
	classes := jdkLibrary()
	for name := range jdkNames {
		classes = append(classes,
			jdkProjectClass(jdkOwnPackage, name, "own"),
			jdkProjectClass(jdkDtoPackage, name, "own"),
			jdkProjectClass("com.example.unrelated", name, "unrelated"))
	}
	return append(classes, jdkSubclasses()...)
}

// jdkSubclassName names the class of caller number c that extends super with the
// argument spelled the way spelling number i spells name.
func jdkSubclassName(c int, super, name string, i int) string {
	return "Sub" + super + "_" + string(rune('A'+c)) + "_" + name + "_" + string(rune('a'+i))
}

// jdkSubclasses declares, in the compilation unit of every caller, the classes
// that extend the library's Base and Mid with each argument the caller writes.
func jdkSubclasses() []parser.Class {
	var classes []parser.Class
	for c, caller := range jdkCallers() {
		for name := range caller.shapes {
			for i, arg := range jdkSpellings(name) {
				if strings.HasPrefix(arg.spelling, "? ") {
					continue
				}
				for _, super := range []string{"Base", "Mid"} {
					sub := caller.class
					sub.Name = jdkSubclassName(c, super, name, i)
					sub.FilePath = strings.TrimSuffix(caller.class.FilePath, "Caller.java") + sub.Name + ".java"
					sub.SuperClass, sub.SuperClassTypeArgs = super, []string{arg.spelling}
					classes = append(classes, sub)
				}
			}
		}
	}
	return classes
}

type jdkResolverWorld struct {
	name  string
	build func(t *testing.T, classes []parser.Class) *TypeResolver
}

func jdkResolverWorlds() []jdkResolverWorld {
	return []jdkResolverWorld{
		{"fresh resolver", func(t *testing.T, classes []parser.Class) *TypeResolver {
			return NewTypeResolver(classes)
		}},
		{"dependency resolver set", func(t *testing.T, classes []parser.Class) *TypeResolver {
			r := NewTypeResolver(classes)
			r.SetDependencyResolver(&libraryClasses{})
			return r
		}},
		// The generic classes are read from a dependency, which has no compilation
		// unit of the project for them; the classes that bear JDK names are the
		// project's.
		{"generic classes of a dependency", func(t *testing.T, classes []parser.Class) *TypeResolver {
			dependency := map[string]parser.Class{}
			var scanned []parser.Class
			for _, c := range classes {
				if c.Package == jdkLibPackage && jdkNames[c.Name].pkg == "" {
					c.FilePath = ""
					dependency[jdkLibPackage+"."+c.Name] = c
					continue
				}
				scanned = append(scanned, c)
			}
			r := NewTypeResolver(scanned)
			r.SetDependencyResolver(&libraryClasses{classes: dependency})
			// The resolver reads the superclass of a class from the classes it
			// already holds, so the library's classes are read first.
			for name := range dependency {
				r.Resolve(name, nil)
			}
			return r
		}},
		{"resolver that read the other candidates", func(t *testing.T, classes []parser.Class) *TypeResolver {
			r := NewTypeResolver(classes)
			for name := range jdkNames {
				for _, pkg := range []string{jdkLibPackage, jdkOwnPackage, jdkDtoPackage, "com.example.unrelated"} {
					r.Resolve(pkg+"."+name, nil)
				}
			}
			return r
		}},
	}
}

func jdkSpellings(name string) []struct{ spelling, shape string } {
	return []struct{ spelling, shape string }{
		{name, "%s"},
		{name + "[]", "[%s]"},
		{name + "[][]", "[[%s]]"},
		{"List<" + name + ">", "[%s]"},
		{"Map<String, " + name + ">", "{%s}"},
		{"? extends " + name, "%s"},
	}
}

func jdkShape(pattern, shape string) string { return strings.ReplaceAll(pattern, "%s", shape) }

func forEachJDKCase(t *testing.T, run func(t *testing.T, r *TypeResolver, view string, caller jdkCaller)) {
	t.Helper()
	for _, w := range jdkResolverWorlds() {
		for view := range typeArgViews {
			for _, caller := range jdkCallers() {
				t.Run(w.name+"/"+view+"/"+caller.name, func(t *testing.T) {
					r := w.build(t, jdkWorldClasses())
					r.SetUnresolved(collector.NewUnresolvedSet())
					run(t, r, view, caller)
				})
			}
		}
	}
}

// A name written in the caller's scope is the class that scope gives it, as a
// direct type, as an array, and as the argument of a generic class at one level,
// at two levels and inside another argument.
func TestResolve_AJdkNameAsATypeArgumentIsTheClassTheCallerNames(t *testing.T) {
	forEachJDKCase(t, func(t *testing.T, r *TypeResolver, view string, caller jdkCaller) {
		for name, shape := range caller.shapes {
			for _, arg := range jdkSpellings(name) {
				want := jdkShape(arg.shape, shape)
				if strings.HasPrefix(arg.spelling, "? ") && name == "Void" {
					continue
				}

				// Directly, with the argument spelled as a type of its own.
				if !strings.Contains(arg.spelling, "?") {
					direct := typeArgViews[view](r, caller.class, arg.spelling)
					if got := typeArgShape(direct); got != want {
						t.Errorf("%s = %s, want %s", arg.spelling, got, want)
					}
				}

				box := typeArgViews[view](r, caller.class, "Box<"+arg.spelling+">")
				for path, wrap := range map[string]string{
					"data": "%s", "list": "[%s]", "array": "[%s]", "byKey": "{%s}", "bounded": "[%s]",
				} {
					if got := typeArgShape(typeArgAt(box, path)); got != jdkShape(wrap, want) {
						t.Errorf("Box<%s>.%s = %s, want %s", arg.spelling, path, got, jdkShape(wrap, want))
					}
				}

				outer := typeArgViews[view](r, caller.class, "Outer<"+arg.spelling+">")
				for path, wrap := range map[string]string{
					"wrapped.data": "%s", "wrappedList.data": "[%s]", "wrappedArray.data": "[%s]", "wrappedMap.data": "{%s}",
				} {
					if got := typeArgShape(typeArgAt(outer, path)); got != jdkShape(wrap, want) {
						t.Errorf("Outer<%s>.%s = %s, want %s", arg.spelling, path, got, jdkShape(wrap, want))
					}
				}

				nested := typeArgViews[view](r, caller.class, "Box<Outer<"+arg.spelling+">>")
				if got := typeArgShape(typeArgAt(nested, "data.wrapped.data")); got != want {
					t.Errorf("Box<Outer<%s>>.data.wrapped.data = %s, want %s", arg.spelling, got, want)
				}
			}
		}
		if unresolved := r.Unresolved(); len(unresolved) != 0 {
			t.Errorf("unresolved = %v, want none", unresolved)
		}
	})
}

// A superclass argument is written in the subclass's scope as well.
func TestResolve_AJdkNameAsASuperclassArgumentIsTheClassTheSubclassNames(t *testing.T) {
	callers := jdkCallers()
	forEachJDKCase(t, func(t *testing.T, r *TypeResolver, view string, caller jdkCaller) {
		c := 0
		for index, other := range callers {
			if other.name == caller.name {
				c = index
			}
		}
		for name, shape := range caller.shapes {
			for i, arg := range jdkSpellings(name) {
				if strings.HasPrefix(arg.spelling, "? ") {
					continue
				}
				want := jdkShape(arg.shape, shape)
				sub := typeArgViews[view](r, caller.class, jdkSubclassName(c, "Base", name, i))
				for path, wrap := range map[string]string{
					"value": "%s", "list": "[%s]", "array": "[%s]", "wrapped.data": "%s",
				} {
					wantPath := jdkShape(wrap, want)
					if got := typeArgShape(typeArgAt(sub, path)); got != wantPath {
						t.Errorf("Sub extends Base<%s>.%s = %s, want %s", arg.spelling, path, got, wantPath)
					}
				}

				// Through a second superclass of the library, which binds its own
				// parameter to the first one's.
				leaf := typeArgViews[view](r, caller.class, jdkSubclassName(c, "Mid", name, i))
				for _, path := range []string{"midValue", "value"} {
					if got := typeArgShape(typeArgAt(leaf, path)); got != want {
						t.Errorf("Sub extends Mid<%s>.%s = %s, want %s", arg.spelling, path, got, want)
					}
				}
			}
		}
	})
}

// A JDK class the library writes in full is the JDK class, though the library's
// own package declares a class of that simple name; a name it writes simply is
// its own.
func TestResolve_AQualifiedJdkNameNeedsNoScope(t *testing.T) {
	for _, w := range jdkResolverWorlds() {
		for view := range typeArgViews {
			t.Run(w.name+"/"+view, func(t *testing.T) {
				r := w.build(t, jdkWorldClasses())
				caller := jdkCallers()[0].class
				holder := typeArgViews[view](r, caller, "Holder")
				for path, want := range map[string]string{
					"text": "string", "when": "string", "own": "String(shadow)",
					"boxed.data": "string", "boxedDate.data": "string", "boxed.list": "[string]",
					"listed": "[int]", "array": "[long]",
				} {
					if got := typeArgShape(typeArgAt(holder, path)); got != want {
						t.Errorf("Holder.%s = %s, want %s", path, got, want)
					}
				}
				// A qualified name resolves the same wherever it is written.
				for spelling, want := range map[string]string{
					"java.lang.String": "string", "java.lang.Integer": "int", "java.lang.Object": "object()",
					"java.util.Date": "string", "java.time.LocalDateTime": "string", "java.math.BigDecimal": "double",
					"java.util.UUID": "string", "java.util.concurrent.atomic.AtomicInteger": "int",
					"java.lang.String[]": "[string]", "java.util.Date[]": "[string]",
				} {
					for _, source := range []parser.Class{caller, jdkCallers()[5].class, {Name: "Plain"}, r.classRegistry[jdkLibPackage+".Box"]} {
						if got := typeArgShape(typeArgViews[view](r, source, spelling)); got != want {
							t.Errorf("%s in %s = %s, want %s", spelling, source.Package, got, want)
						}
					}
				}
			})
		}
	}
}

// A JDK class that an import on demand supplies is that class, though a class of
// the project that no scope of the caller reaches has the same simple name.
func TestResolve_AnImportOnDemandSuppliesTheJdkClassBeforeAnUnrelatedClass(t *testing.T) {
	unrelated := jdkProjectClass("com.example.unrelated", "Date", "unrelated")
	caller := parser.Class{
		Name: "Caller", Package: jdkAppPackage, FilePath: "app/Caller.java",
		Imports: []parser.Import{{Path: "java.util", OnDemand: true}, {Path: jdkLibPackage + ".Box"}},
		Fields:  []parser.Field{{Name: "date", Type: "Date"}},
	}
	box := jdkLibraryClass("Box", []string{"T"}, "", nil, typeArgField("data", "T"))
	boxShadow := jdkProjectClass(jdkLibPackage, "Date", "shadow")
	r := NewTypeResolver([]parser.Class{unrelated, caller, box, boxShadow})
	for spelling, want := range map[string]string{"Date": "string", "Box<Date>": "Box(data)"} {
		if got := typeArgShape(r.ResolveFrom(caller, spelling, nil)); got != want {
			t.Errorf("%s = %s, want %s", spelling, got, want)
		}
	}
	if got := typeArgShape(typeArgAt(r.ResolveFrom(caller, "Box<Date>", nil), "data")); got != "string" {
		t.Errorf("Box<Date>.data = %s, want the JDK date", got)
	}
}

// An import on demand supplies the JDK value types the resolver answers for, so
// the tables that answer by simple name and the one that names the packages
// must list the same classes.
func TestResolve_EveryJdkValueTypeIsSuppliedByItsPackage(t *testing.T) {
	named := map[string]bool{}
	for qualified := range jdkQualifiedScalars {
		named[strings.ToLower(lastSegment(qualified))] = true
	}
	for qualified := range jdkValueTypes {
		simple := strings.ToLower(lastSegment(qualified))
		named[simple] = true
		if _, ok := jdkScalarTypes[simple]; !ok {
			t.Errorf("%s is supplied by its package but has no scalar", qualified)
		}
	}
	for simple := range jdkScalarTypes {
		if !named[simple] {
			t.Errorf("%s has a scalar but no import of its package supplies it", simple)
		}
	}
	for qualified := range jdkValueTypes {
		if !isPlatformType(qualified) {
			t.Errorf("%s is not a platform type", qualified)
		}
	}
}

// An enum of the project that bears the name of a JDK class hides it as well,
// though the JSON view does not read enums: written as an argument, the name
// stays the enum's.
func TestResolve_AProjectEnumWithAJdkNameIsNotTheJdkClass(t *testing.T) {
	var classes []parser.Class
	for name := range jdkNames {
		classes = append(classes, parser.Class{
			Name: name, Package: jdkOwnPackage, FilePath: "own/" + name + ".java",
			IsEnum: true, EnumConstants: []string{"ONE", "TWO"},
		})
	}
	classes = append(classes, jdkLibrary()...)
	caller := parser.Class{
		Name: "Caller", Package: jdkOwnPackage, FilePath: "own/Caller.java",
		Imports: []parser.Import{{Path: jdkLibPackage + ".Box"}, {Path: jdkLibPackage + ".Outer"}, {Path: "java.util.List"}},
	}
	for name, info := range jdkNames {
		if info.shape == "string" {
			// An enum is a string in the binding view, and these classes are one:
			// nothing tells them apart.
			continue
		}
		for view := range typeArgViews {
			t.Run(name+"/"+view, func(t *testing.T) {
				r := NewTypeResolver(classes)
				r.SetUnresolved(collector.NewUnresolvedSet())
				for _, tc := range []struct{ spelling, path, wrap string }{
					{name, "", "%s"},
					{"Box<" + name + ">", "data", "%s"},
					{"Outer<" + name + ">", "wrapped.data", "%s"},
					{"Box<List<" + name + ">>", "data", "[%s]"},
				} {
					m := typeArgViews[view](r, caller, tc.spelling)
					if tc.path != "" {
						m = typeArgAt(m, tc.path)
					}
					got := typeArgShape(m)
					if got == jdkShape(tc.wrap, info.shape) {
						t.Errorf("%s = %s, the JDK class; the enum of the package hides it", tc.spelling, got)
					}
					if view == "binding" && got != jdkShape(tc.wrap, "string") {
						t.Errorf("%s = %s, want the enum as a string in the binding view", tc.spelling, got)
					}
				}
			})
		}
	}
}

// A method returns nothing when its return type is the keyword or java.lang.Void,
// and a class of the project called Void is a class.
func TestResolve_IsVoid(t *testing.T) {
	project := jdkProjectClass(jdkOwnPackage, "Void", "own")
	r := NewTypeResolver([]parser.Class{project, jdkProjectClass(jdkLibPackage, "Void", "shadow")})
	callers := map[string]parser.Class{
		"default":        {Name: "Caller", Package: jdkAppPackage, FilePath: "app/Caller.java"},
		"imported":       {Name: "Caller", Package: jdkAppPackage, FilePath: "app/Caller.java", Imports: []parser.Import{{Path: jdkOwnPackage + ".Void"}}},
		"on demand":      {Name: "Caller", Package: jdkAppPackage, FilePath: "app/Caller.java", Imports: []parser.Import{{Path: jdkOwnPackage, OnDemand: true}}},
		"same package":   {Name: "Caller", Package: jdkOwnPackage, FilePath: "own/Caller.java"},
		"explicit class": {Name: "Caller", Package: jdkOwnPackage, FilePath: "own/Caller.java", Imports: []parser.Import{{Path: "java.lang.Void"}}},
	}
	for name, want := range map[string][2]bool{
		// void, java.lang.Void by default; a project class where one is in scope.
		"default": {true, true}, "imported": {true, false}, "on demand": {true, false},
		"same package": {true, false}, "explicit class": {true, true},
	} {
		source := callers[name]
		if got := r.IsVoid(source, "void"); got != want[0] {
			t.Errorf("%s: IsVoid(void) = %v, want %v", name, got, want[0])
		}
		if got := r.IsVoid(source, "Void"); got != want[1] {
			t.Errorf("%s: IsVoid(Void) = %v, want %v", name, got, want[1])
		}
		if !r.IsVoid(source, "java.lang.Void") {
			t.Errorf("%s: IsVoid(java.lang.Void) = false, want true", name)
		}
		if r.IsVoid(source, "List<Void>") || r.IsVoid(source, "String") {
			t.Errorf("%s: a type that holds or is something else is not void", name)
		}
	}
}
