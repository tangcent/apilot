package resolver

import (
	"sort"
	"strings"
	"testing"

	javaparser "github.com/tangcent/apilot/api-collector-java/parser"
	ktparser "github.com/tangcent/apilot/api-collector-kotlin/parser"
	model "github.com/tangcent/apilot/api-model"
)

// Kotlin binds a name by its own order: explicit imports, the package, star
// imports, and last the packages every file imports (kotlin.*,
// kotlin.collections.*, java.lang.* ...). `Int` is kotlin.Int unless one of the
// first three scopes declares a class of that name, and it is kotlin.Int even
// where the package declares `Integer`, the name the shared resolver would give
// it. The tests below give the package of the generic classes, and the packages
// of the callers, classes that bear the names of Kotlin's types, and ask what
// each caller gets.

const (
	scopeLibPackage     = "com.example.kclib"
	scopeAppPackage     = "com.example.kcapp"
	scopeOwnPackage     = "com.example.kcown"
	scopeShadowPackage  = "com.example.kcshadow"
	scopeUnrelatedPaths = "com.example.kcunrelated"
)

// scopeBuiltins are the types of Kotlin the adapter maps, each with the shape it
// exports.
var scopeBuiltins = map[string]string{
	"Int": "int", "Long": "long", "Short": "int", "Byte": "int", "Double": "double", "Float": "float",
	"Boolean": "boolean", "Char": "string", "String": "string", "Number": "double", "Any": "object()",
	"Unit": "null", "ByteArray": "[int]", "IntArray": "[int]", "LongArray": "[long]",
	"Integer": "int", // java.lang.Integer, which every Kotlin file imports last
}

// scopeShadow declares a class of the project with the name of a Kotlin type.
func scopeShadow(pkg, name, field string) ktparser.Class {
	return ktparser.Class{
		Name: name, Package: pkg, FilePath: strings.ReplaceAll(pkg, ".", "/") + "/" + name + ".kt",
		Fields: []ktparser.Property{{Name: field, Type: "kotlin.Int", InConstructor: true}},
	}
}

func scopeLibraryClass(name string, params []string, superClass string, superArgs []string, fields ...ktparser.Property) ktparser.Class {
	return ktparser.Class{
		Name: name, Package: scopeLibPackage, FilePath: "com/example/kclib/" + name + ".kt",
		TypeParameters: params, SuperClass: superClass, SuperClassTypeArgs: superArgs, Fields: fields,
		Imports: []ktparser.Import{
			{Path: scopeLibPackage + ".Wrapper"}, {Path: scopeLibPackage + ".Base"},
		},
	}
}

func scopeProp(name, typeName string) ktparser.Property {
	return ktparser.Property{Name: name, Type: typeName, InConstructor: true}
}

// scopeLibrary declares the generic classes, in a package that declares a class
// for every name under test.
func scopeLibrary() []ktparser.Class {
	p := scopeProp
	classes := []ktparser.Class{
		scopeLibraryClass("Box", []string{"T"}, "", nil,
			p("data", "T"), p("list", "List<T>"), p("array", "Array<T>"), p("byKey", "Map<kotlin.String, T>")),
		scopeLibraryClass("Wrapper", []string{"U"}, "", nil, p("data", "U")),
		scopeLibraryClass("Outer", []string{"T"}, "", nil,
			p("wrapped", "Wrapper<T>"), p("wrappedList", "Wrapper<List<T>>"), p("wrappedMap", "Wrapper<Map<kotlin.String, T>>")),
		scopeLibraryClass("Base", []string{"B"}, "", nil,
			p("value", "B"), p("list", "List<B>"), p("array", "Array<B>"), p("wrapped", "Wrapper<B>")),
		scopeLibraryClass("Mid", []string{"M"}, "Base", []string{"M"}, p("midValue", "M")),
	}
	for name := range scopeBuiltins {
		classes = append(classes, scopeShadow(scopeLibPackage, name, "shadow"))
	}
	return classes
}

type scopeCaller struct {
	name string
	unit ktparser.Class
	// shapes maps a name the caller writes to the shape it exports as there.
	shapes map[string]string
}

func scopeCallers() []scopeCaller {
	generic := []ktparser.Import{
		{Path: scopeLibPackage + ".Box"}, {Path: scopeLibPackage + ".Outer"}, {Path: scopeLibPackage + ".Wrapper"},
		{Path: scopeLibPackage + ".Base"}, {Path: scopeLibPackage + ".Mid"},
	}
	unit := func(pkg string, imports ...ktparser.Import) ktparser.Class {
		return ktparser.Class{
			Name: "Caller", Package: pkg, FilePath: strings.ReplaceAll(pkg, ".", "/") + "/Caller.kt",
			Imports: append(append([]ktparser.Import(nil), generic...), imports...),
		}
	}
	builtin := map[string]string{}
	project := map[string]string{}
	var named, starred []ktparser.Import
	for name, shape := range scopeBuiltins {
		builtin[name] = shape
		project[name] = name + "(own)"
		named = append(named, ktparser.Import{Path: scopeShadowPackage + "." + name})
	}
	starred = append(starred, ktparser.Import{Path: scopeShadowPackage, OnDemand: true})

	only := func(shapes map[string]string, names ...string) map[string]string {
		out := map[string]string{}
		for _, name := range names {
			out[name] = shapes[name]
		}
		return out
	}
	explicitBuiltin := []ktparser.Import{
		{Path: "kotlin.Int"}, {Path: "kotlin.String"}, {Path: "kotlin.Number"}, {Path: "kotlin.Unit"},
	}
	return []scopeCaller{
		{"default imports", unit(scopeAppPackage), builtin},
		// The package declares Integer, and Int stays kotlin.Int.
		{"package with other names", unit(scopeOwnPackage), mergeShapes(builtin, map[string]string{"Integer": "Integer(own)"})},
		{"package declares the names", unit(scopeShadowPackage), project},
		{"explicit imports of the names", unit(scopeAppPackage, named...), project},
		{"star import of the names", unit(scopeAppPackage, starred...), project},
		// An explicit import is the first scope: it wins over the package.
		{"explicit import beats the package", unit(scopeShadowPackage, explicitBuiltin...),
			mergeShapes(project, only(builtin, "Int", "String", "Number", "Unit"))},
		// The package wins over a star import.
		{"package beats a star import", unit(scopeOwnPackage, starred...), mergeShapes(project, map[string]string{"Integer": "Integer(own)"})},
	}
}

func mergeShapes(base, over map[string]string) map[string]string {
	out := map[string]string{}
	for k, v := range base {
		out[k] = v
	}
	for k, v := range over {
		out[k] = v
	}
	return out
}

// scopeWorldClasses: the library, the classes of the project that bear the names
// of Kotlin's types, and the classes that extend the library's.
func scopeWorldClasses() []ktparser.Class {
	classes := scopeLibrary()
	// The caller's own package declares Integer, and the shadow package every
	// name.
	classes = append(classes, scopeShadow(scopeOwnPackage, "Integer", "own"))
	for name := range scopeBuiltins {
		classes = append(classes, scopeShadow(scopeShadowPackage, name, "own"))
		classes = append(classes, scopeShadow(scopeUnrelatedPaths, name, "unrelated"))
	}
	return append(classes, scopeSubclasses()...)
}

func scopeSubclassName(c int, super, name string, i int) string {
	return "Sub" + super + "_" + string(rune('A'+c)) + "_" + name + "_" + string(rune('a'+i))
}

func scopeSpellings(name string) []struct{ spelling, shape string } {
	return []struct{ spelling, shape string }{
		{name, "%s"},
		{"List<" + name + ">", "[%s]"},
		{"Array<" + name + ">", "[%s]"},
		{"Map<kotlin.String, " + name + ">", "{%s}"},
		{"List<List<" + name + ">>", "[[%s]]"},
	}
}

func scopeShape(pattern, shape string) string { return strings.ReplaceAll(pattern, "%s", shape) }

func scopeSubclasses() []ktparser.Class {
	var classes []ktparser.Class
	for c, caller := range scopeCallers() {
		for name := range caller.shapes {
			for i, arg := range scopeSpellings(name) {
				for _, super := range []string{"Base", "Mid"} {
					sub := caller.unit
					sub.Name = scopeSubclassName(c, super, name, i)
					sub.FilePath = strings.TrimSuffix(caller.unit.FilePath, "Caller.kt") + sub.Name + ".kt"
					sub.SuperClass, sub.SuperClassTypeArgs = super, []string{arg.spelling}
					classes = append(classes, sub)
				}
			}
		}
	}
	return classes
}

func scopeModelShape(m *model.ObjectModel) string {
	switch {
	case m == nil:
		return "nil"
	case m.IsArray():
		return "[" + scopeModelShape(m.Items) + "]"
	case m.IsMap():
		return "{" + scopeModelShape(m.ValueModel) + "}"
	case m.IsObject():
		names := make([]string, 0, len(m.Fields))
		for name := range m.Fields {
			names = append(names, name)
		}
		sort.Strings(names)
		return m.TypeName + "(" + strings.Join(names, ",") + ")"
	}
	return m.TypeName
}

func scopeAt(m *model.ObjectModel, path string) *model.ObjectModel {
	for _, name := range strings.Split(path, ".") {
		if m == nil || m.Fields[name] == nil {
			return nil
		}
		m = m.Fields[name].Model
	}
	return m
}

var scopeViews = map[string]func(r *Resolver, source *ktparser.Class, typeName string) *model.ObjectModel{
	"json": func(r *Resolver, source *ktparser.Class, typeName string) *model.ObjectModel {
		return r.ResolveFrom(source, typeName, nil)
	},
	"binding": func(r *Resolver, source *ktparser.Class, typeName string) *model.ObjectModel {
		return r.ResolveForBinding(source, typeName, nil)
	},
}

func forEachScopeCase(t *testing.T, run func(t *testing.T, r *Resolver, view string, caller scopeCaller)) {
	t.Helper()
	for view := range scopeViews {
		for _, caller := range scopeCallers() {
			t.Run(view+"/"+caller.name, func(t *testing.T) {
				run(t, New(scopeWorldClasses()), view, caller)
			})
		}
	}
}

// A name written in the caller's scope is what Kotlin binds it to, as a type of
// its own, as an element of an array or a collection, as a map value, and as the
// argument of a generic class at one level and at two levels.
func TestResolve_AKotlinNameIsWhatKotlinBindsItTo(t *testing.T) {
	forEachScopeCase(t, func(t *testing.T, r *Resolver, view string, caller scopeCaller) {
		source := caller.unit
		for name, shape := range caller.shapes {
			for _, arg := range scopeSpellings(name) {
				want := scopeShape(arg.shape, shape)
				if got := scopeModelShape(scopeViews[view](r, &source, arg.spelling)); got != want {
					t.Errorf("%s = %s, want %s", arg.spelling, got, want)
				}

				box := scopeViews[view](r, &source, "Box<"+arg.spelling+">")
				for path, wrap := range map[string]string{"data": "%s", "list": "[%s]", "array": "[%s]", "byKey": "{%s}"} {
					if got := scopeModelShape(scopeAt(box, path)); got != scopeShape(wrap, want) {
						t.Errorf("Box<%s>.%s = %s, want %s", arg.spelling, path, got, scopeShape(wrap, want))
					}
				}
				outer := scopeViews[view](r, &source, "Outer<"+arg.spelling+">")
				for path, wrap := range map[string]string{"wrapped.data": "%s", "wrappedList.data": "[%s]", "wrappedMap.data": "{%s}"} {
					if got := scopeModelShape(scopeAt(outer, path)); got != scopeShape(wrap, want) {
						t.Errorf("Outer<%s>.%s = %s, want %s", arg.spelling, path, got, scopeShape(wrap, want))
					}
				}
				nested := scopeViews[view](r, &source, "Box<Outer<"+arg.spelling+">>")
				if got := scopeModelShape(scopeAt(nested, "data.wrapped.data")); got != want {
					t.Errorf("Box<Outer<%s>>.data.wrapped.data = %s, want %s", arg.spelling, got, want)
				}
			}
		}
		if unresolved := r.Unresolved(); len(unresolved) != 0 {
			t.Errorf("unresolved = %v, want none", unresolved)
		}
	})
}

// The argument of a superclass is written in the subclass's scope.
func TestResolve_AKotlinNameAsASuperclassArgumentIsWhatKotlinBindsItTo(t *testing.T) {
	callers := scopeCallers()
	forEachScopeCase(t, func(t *testing.T, r *Resolver, view string, caller scopeCaller) {
		c := 0
		for index, other := range callers {
			if other.name == caller.name {
				c = index
			}
		}
		source := caller.unit
		for name, shape := range caller.shapes {
			for i, arg := range scopeSpellings(name) {
				want := scopeShape(arg.shape, shape)
				sub := scopeViews[view](r, &source, scopeSubclassName(c, "Base", name, i))
				for path, wrap := range map[string]string{"value": "%s", "list": "[%s]", "array": "[%s]", "wrapped.data": "%s"} {
					if got := scopeModelShape(scopeAt(sub, path)); got != scopeShape(wrap, want) {
						t.Errorf(": Base<%s>.%s = %s, want %s", arg.spelling, path, got, scopeShape(wrap, want))
					}
				}
				leaf := scopeViews[view](r, &source, scopeSubclassName(c, "Mid", name, i))
				for _, path := range []string{"midValue", "value"} {
					if got := scopeModelShape(scopeAt(leaf, path)); got != want {
						t.Errorf(": Mid<%s>.%s = %s, want %s", arg.spelling, path, got, want)
					}
				}
			}
		}
	})
}

// A property of a class, and the type of a function's value, are written in the
// scope of the file that declares them; the adapter hands the resolver the
// spelling of that scope, not the caller's.
func TestResolve_APropertyIsReadInTheScopeOfItsOwnFile(t *testing.T) {
	holder := ktparser.Class{
		Name: "Holder", Package: scopeOwnPackage, FilePath: "own/Holder.kt",
		Fields: []ktparser.Property{
			scopeProp("count", "Int"), scopeProp("boxed", "Integer"), scopeProp("counts", "List<Int>"),
			scopeProp("number", "Number"), scopeProp("bytes", "ByteArray"),
		},
	}
	classes := append(scopeWorldClasses(), holder, ktparser.Class{
		Name: "Number", Package: scopeOwnPackage, FilePath: "own/Number.kt",
		Fields: []ktparser.Property{scopeProp("digits", "String")},
	})
	for view := range scopeViews {
		t.Run(view, func(t *testing.T) {
			r := New(classes)
			got := scopeViews[view](r, &holder, "Holder")
			for path, want := range map[string]string{
				"count": "int", "boxed": "Integer(own)", "counts": "[int]", "number": "Number(digits)", "bytes": "[int]",
			} {
				if shape := scopeModelShape(scopeAt(got, path)); shape != want {
					t.Errorf("Holder.%s = %s, want %s", path, shape, want)
				}
			}
		})
	}
}

// A class of Java in the same tree hides a Kotlin type of that name as well.
func TestResolve_AJavaClassOfTheSamePackageHidesAKotlinName(t *testing.T) {
	source := ktparser.Class{
		Name: "Form", Package: scopeOwnPackage, FilePath: "own/Form.kt",
		Fields: []ktparser.Property{scopeProp("weight", "Number"), scopeProp("count", "Int"), scopeProp("weights", "List<Number>")},
	}
	javaNumber := javaparser.Class{
		Name: "Number", Package: scopeOwnPackage, FilePath: "own/Number.java",
		Fields: []javaparser.Field{{Name: "digits", Type: "String"}},
	}
	if got := New(nil).NormalizeType(&source, "Number"); got != "java.lang.Number" {
		t.Errorf("Number = %s, want the standard class where nothing declares it", got)
	}
	r := NewWithJavaTypes([]ktparser.Class{source}, []javaparser.Class{javaNumber})
	if got := r.NormalizeType(&source, "Number"); got != "Number" {
		t.Errorf("Number = %s, want the class of the package", got)
	}
	form := r.ResolveFrom(&source, "Form", nil)
	for path, want := range map[string]string{"weight": "Number(digits)", "count": "int", "weights": "[Number(digits)]"} {
		if got := scopeModelShape(scopeAt(form, path)); got != want {
			t.Errorf("Form.%s = %s, want %s", path, got, want)
		}
	}
}

// An enum of the project that bears the name of a Kotlin type is that enum where
// the package declares it, not the type of the standard library.
func TestResolve_AProjectEnumHidesAKotlinName(t *testing.T) {
	enumPackage := "com.example.kcenum"
	enumUnit := ktparser.Class{
		Name: "Unit", Package: enumPackage, FilePath: "enum/Unit.kt", IsEnum: true,
		EnumValues: []string{"PIECE", "BOX"},
	}
	enumNumber := ktparser.Class{
		Name: "Number", Package: enumPackage, FilePath: "enum/Number.kt", IsEnum: true,
		EnumValues: []string{"ONE", "TWO"},
	}
	form := ktparser.Class{
		Name: "Form", Package: enumPackage, FilePath: "enum/Form.kt",
		Fields: []ktparser.Property{
			scopeProp("unit", "Unit"), scopeProp("number", "Number"), scopeProp("units", "List<Unit>"),
			scopeProp("count", "Int"), scopeProp("total", "Long"),
		},
	}
	for view := range scopeViews {
		t.Run(view, func(t *testing.T) {
			r := New([]ktparser.Class{enumUnit, enumNumber, form})
			got := scopeViews[view](r, &form, "Form")
			for path, want := range map[string]string{
				"unit": "string", "number": "string", "units": "[string]", "count": "int", "total": "long",
			} {
				if shape := scopeModelShape(scopeAt(got, path)); shape != want {
					t.Errorf("Form.%s = %s, want %s", path, shape, want)
				}
			}
			if options := got.Fields["unit"].Options; view == "binding" && len(options) != 2 {
				t.Errorf("Form.unit options = %v, want the constants of the enum", options)
			}
		})
	}
}

// A renaming import binds only the new name, to the class it imports: `Int` is
// that class where `import a.Number as Int` says so, and the class's own name
// is not bound by it.
func TestResolve_ARenamingImportBindsOnlyTheNewName(t *testing.T) {
	classes := append(scopeWorldClasses(), ktparser.Class{
		Name: "Form", Package: scopeAppPackage, FilePath: "app/Form.kt",
		Imports: []ktparser.Import{
			{Path: scopeShadowPackage + ".Number", Alias: "Int"},
			{Path: "kotlin.collections.List", Alias: "Items"},
		},
		Fields: []ktparser.Property{
			scopeProp("count", "Int"), scopeProp("counts", "Items<Long>"), scopeProp("weight", "Number"), scopeProp("total", "Long"),
		},
	})
	form := classes[len(classes)-1]
	for view := range scopeViews {
		t.Run(view, func(t *testing.T) {
			r := New(classes)
			got := scopeViews[view](r, &form, "Form")
			for path, want := range map[string]string{
				"count": "Number(own)", "counts": "[long]", "weight": "double", "total": "long",
			} {
				if shape := scopeModelShape(scopeAt(got, path)); shape != want {
					t.Errorf("Form.%s = %s, want %s", path, shape, want)
				}
			}
			if name := r.NormalizeType(&form, "Int"); name != scopeShadowPackage+".Number" {
				t.Errorf("Int = %s, want the class the alias names, in full", name)
			}
		})
	}
}

// A class of the project that bears the name of a type of Kotlin's standard
// library keeps its name when it is passed as an argument: to a generic class,
// and to a superclass, whose bindings are the shared resolver's own spelling and
// are not read as Kotlin names again.
func TestResolve_AProjectClassPassedAsAnArgumentKeepsItsName(t *testing.T) {
	// List, Set, Iterable and Map are read by the class their name denotes too.
	names := []string{"Sequence", "Array", "Number", "MutableList", "IntArray", "Unit", "List", "Set", "Iterable", "Map"}
	classes := scopeLibrary()
	var subs []ktparser.Class
	for _, name := range names {
		classes = append(classes, scopeShadow(scopeShadowPackage, name, "own"))
		subs = append(subs, ktparser.Class{
			Name: "Sub" + name, Package: scopeShadowPackage, FilePath: "kcshadow/Sub" + name + ".kt",
			Imports:    []ktparser.Import{{Path: scopeLibPackage + ".Base"}, {Path: scopeLibPackage + ".Box"}},
			SuperClass: "Base", SuperClassTypeArgs: []string{name},
		})
	}
	source := ktparser.Class{
		Name: "Caller", Package: scopeShadowPackage, FilePath: "kcshadow/Caller.kt",
		Imports: []ktparser.Import{{Path: scopeLibPackage + ".Box"}, {Path: scopeLibPackage + ".Outer"}},
	}
	classes = append(classes, subs...)
	for view := range scopeViews {
		t.Run(view, func(t *testing.T) {
			r := New(classes)
			for _, name := range names {
				want := name + "(own)"
				if got := scopeModelShape(scopeAt(scopeViews[view](r, &source, "Box<"+name+">"), "data")); got != want {
					t.Errorf("Box<%s>.data = %s, want %s", name, got, want)
				}
				if got := scopeModelShape(scopeAt(scopeViews[view](r, &source, "Outer<"+name+">"), "wrapped.data")); got != want {
					t.Errorf("Outer<%s>.wrapped.data = %s, want %s", name, got, want)
				}
				sub := scopeViews[view](r, &source, "Sub"+name)
				for _, path := range []string{"value", "wrapped.data"} {
					if got := scopeModelShape(scopeAt(sub, path)); got != want {
						t.Errorf("Sub%s %s = %s, want %s", name, path, got, want)
					}
				}
				if got := scopeModelShape(scopeAt(sub, "list")); got != "["+want+"]" {
					t.Errorf("Sub%s list = %s, want [%s]", name, got, want)
				}

				// The bindings a controller hands down its superclass chain.
				leaf := ktparser.Class{
					Name: "Leaf" + name, Package: scopeShadowPackage, FilePath: "kcshadow/Leaf" + name + ".kt",
					Imports:    []ktparser.Import{{Path: scopeLibPackage + ".Mid"}},
					SuperClass: "Mid", SuperClassTypeArgs: []string{name},
				}
				var mid, base ktparser.Class
				for _, class := range classes {
					switch {
					case class.Package == scopeLibPackage && class.Name == "Mid":
						mid = class
					case class.Package == scopeLibPackage && class.Name == "Base":
						base = class
					}
				}
				midBindings := r.SuperClassBindings(&leaf, &mid, nil)
				baseBindings := r.SuperClassBindings(&mid, &base, midBindings)
				qualified := scopeShadowPackage + "." + name
				if midBindings["M"] != qualified || baseBindings["B"] != qualified {
					t.Errorf("Leaf%s bindings = %v, then %v; want %s", name, midBindings, baseBindings, qualified)
				}
			}
		})
	}
}
