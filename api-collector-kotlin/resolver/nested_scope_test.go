package resolver

import (
	"testing"

	ktparser "github.com/tangcent/apilot/api-collector-kotlin/parser"
)

// Only the top-level declarations of a file are in the scope of its package. A
// class nested in another is in scope in the body of the class that holds it, of
// the classes nested in that one, and of their subclasses: nowhere else, so it
// neither stands for the name in another class nor hides the standard type that
// has the name there.

const nestedPackage = "com.example.kcnested"

func nestedClass(outer, name, field string) ktparser.Class {
	class := scopeShadow(nestedPackage, name, field)
	class.Outer = outer
	class.FilePath = "nested/Holder.kt"
	return class
}

func nestedForm(file string, fields ...ktparser.Property) ktparser.Class {
	return ktparser.Class{Name: "Form", Package: nestedPackage, FilePath: file, Fields: fields}
}

func checkNestedShapes(t *testing.T, classes []ktparser.Class, source ktparser.Class, want map[string]string) {
	t.Helper()
	for view := range scopeViews {
		t.Run(view, func(t *testing.T) {
			r := New(append(append([]ktparser.Class(nil), classes...), source))
			got := scopeViews[view](r, &source, source.Name)
			for path, shape := range want {
				if actual := scopeModelShape(scopeAt(got, path)); actual != shape {
					t.Errorf("%s.%s = %s, want %s", source.Name, path, actual, shape)
				}
			}
		})
	}
}

// A class nested in another one, in the same file or another, is no class of the
// package: a class of another file that writes Int or Number gets Kotlin's.
func TestResolve_ANestedClassDoesNotHideAStandardType(t *testing.T) {
	classes := []ktparser.Class{
		{Name: "Holder", Package: nestedPackage, FilePath: "nested/Holder.kt"},
		nestedClass("Holder", "Int", "own"),
		{Name: "Other", Package: nestedPackage, FilePath: "nested/Other.kt"},
		nestedClass("Other", "Number", "own"),
	}
	classes[3].FilePath = "nested/Other.kt"
	form := nestedForm("nested/Form.kt", scopeProp("count", "Int"), scopeProp("weight", "Number"), scopeProp("counts", "List<Int>"))
	checkNestedShapes(t, classes, form, map[string]string{"count": "int", "weight": "double", "counts": "[int]"})

	// A class of the same file does not see them either.
	sameFile := nestedForm("nested/Holder.kt", scopeProp("count", "Int"))
	checkNestedShapes(t, classes, sameFile, map[string]string{"count": "int"})

	r := New(append(classes, form))
	if got := r.NormalizeType(&form, "Int"); got != "java.lang.Integer" {
		t.Errorf("Int = %s, want the standard class", got)
	}
}

// The scope of a class body holds the classes nested in it, in the classes that
// enclose it, and in its supertypes.
func TestResolve_ANestedClassIsInScopeAroundItsOuter(t *testing.T) {
	classes := []ktparser.Class{
		{Name: "Holder", Package: nestedPackage, FilePath: "nested/Holder.kt"},
		nestedClass("Holder", "Int", "own"),
		// Nested in the class that holds Int, and one level deeper.
		func() ktparser.Class {
			c := nestedForm("nested/Holder.kt", scopeProp("count", "Int"), scopeProp("total", "Long"))
			c.Name, c.Outer = "Inner", "Holder"
			return c
		}(),
		func() ktparser.Class {
			c := nestedForm("nested/Holder.kt", scopeProp("count", "Int"))
			c.Name, c.Outer = "Deep", "Holder.Inner"
			return c
		}(),
		// A class that inherits the nested classes of its superclass.
		{Name: "Base", Package: nestedPackage, FilePath: "nested/Base.kt"},
		nestedClassIn("nested/Base.kt", "Base", "Number", "digits"),
		{Name: "Sub", Package: nestedPackage, FilePath: "nested/Sub.kt", SuperClass: "Base",
			Fields: []ktparser.Property{scopeProp("weight", "Number"), scopeProp("count", "Int")}},
		{Name: "Plain", Package: nestedPackage, FilePath: "nested/Plain.kt",
			Fields: []ktparser.Property{scopeProp("weight", "Number")}},
	}
	byName := map[string]ktparser.Class{}
	for _, c := range classes {
		byName[c.Outer+"/"+c.Name] = c
	}
	for _, tc := range []struct {
		source string
		want   map[string]string
	}{
		{"Holder/Inner", map[string]string{"count": "Int(own)", "total": "long"}},
		{"Holder.Inner/Deep", map[string]string{"count": "Int(own)"}},
		{"/Sub", map[string]string{"weight": "Number(digits)", "count": "int"}},
		{"/Plain", map[string]string{"weight": "double"}},
	} {
		source := byName[tc.source]
		t.Run(tc.source, func(t *testing.T) {
			for view := range scopeViews {
				r := New(classes)
				got := scopeViews[view](r, &source, nestedKey(source))
				for path, shape := range tc.want {
					if actual := scopeModelShape(scopeAt(got, path)); actual != shape {
						t.Errorf("%s: %s.%s = %s, want %s", view, source.Name, path, actual, shape)
					}
				}
			}
		})
	}
}

func nestedClassIn(file, outer, name, field string) ktparser.Class {
	class := nestedClass(outer, name, field)
	class.FilePath = file
	return class
}

func nestedKey(class ktparser.Class) string {
	name := class.Name
	if class.Outer != "" {
		name = class.Outer + "." + name
	}
	return nestedPackage + "." + name
}

// A nested class that a file imports, by name or with its outer class on
// demand, is in scope there: the import names it by its path through the outer
// class.
func TestResolve_AnImportedNestedClassIsInScope(t *testing.T) {
	classes := []ktparser.Class{
		{Name: "Holder", Package: nestedPackage, FilePath: "nested/Holder.kt"},
		nestedClass("Holder", "Int", "own"),
	}
	byName := ktparser.Class{
		Name: "Form", Package: "com.example.kcimport", FilePath: "import/Form.kt",
		Imports: []ktparser.Import{{Path: nestedPackage + ".Holder.Int"}},
		Fields:  []ktparser.Property{scopeProp("count", "Int"), scopeProp("total", "Long")},
	}
	checkNestedShapes(t, classes, byName, map[string]string{"count": "Int(own)", "total": "long"})

	onDemand := byName
	onDemand.Imports = []ktparser.Import{{Path: nestedPackage + ".Holder", OnDemand: true}}
	checkNestedShapes(t, classes, onDemand, map[string]string{"count": "Int(own)", "total": "long"})
}

// A private top-level class is visible in its file only: a class of another file
// of the package that writes the name gets the standard type.
func TestResolve_APrivateClassOfAnotherFileDoesNotHideAStandardType(t *testing.T) {
	hidden := scopeShadow(nestedPackage, "Int", "own")
	hidden.IsPrivate = true
	hidden.FilePath = "nested/Private.kt"
	classes := []ktparser.Class{hidden}

	other := nestedForm("nested/Form.kt", scopeProp("count", "Int"))
	checkNestedShapes(t, classes, other, map[string]string{"count": "int"})

	sameFile := nestedForm("nested/Private.kt", scopeProp("count", "Int"))
	checkNestedShapes(t, classes, sameFile, map[string]string{"count": "Int(own)"})

	// A class that is not private is the package's.
	hidden.IsPrivate = false
	checkNestedShapes(t, []ktparser.Class{hidden}, other, map[string]string{"count": "Int(own)"})
}

func aliasOf(pkg, file, name, target string, params []string, imports ...ktparser.Import) ktparser.TypeAlias {
	return ktparser.TypeAlias{Name: name, Package: pkg, FilePath: file, Target: target, TypeParameters: params, Imports: imports}
}

func checkAliasShapes(t *testing.T, classes []ktparser.Class, aliases []ktparser.TypeAlias, source ktparser.Class, want map[string]string) {
	t.Helper()
	for view := range scopeViews {
		t.Run(view, func(t *testing.T) {
			r := NewWithAliases(append(append([]ktparser.Class(nil), classes...), source), nil, aliases)
			got := scopeViews[view](r, &source, source.Name)
			for path, shape := range want {
				if actual := scopeModelShape(scopeAt(got, path)); actual != shape {
					t.Errorf("%s.%s = %s, want %s", source.Name, path, actual, shape)
				}
			}
		})
	}
}

// A typealias is a declaration of its package like a class: one named like a
// standard type hides it, and the name stands for the type the alias names.
func TestResolve_ATypeAliasHidesAStandardTypeAndStandsForItsTarget(t *testing.T) {
	aliases := []ktparser.TypeAlias{
		aliasOf(nestedPackage, "nested/Aliases.kt", "Number", "Long", nil),
		aliasOf(nestedPackage, "nested/Aliases.kt", "Ids", "List<T>", []string{"T"}),
		aliasOf(nestedPackage, "nested/Aliases.kt", "Dict", "Map<String, V>", []string{"V"}),
	}
	form := nestedForm("nested/Form.kt",
		scopeProp("weight", "Number"), scopeProp("ids", "Ids<Int>"), scopeProp("dict", "Dict<Boolean>"), scopeProp("count", "Int"))
	checkAliasShapes(t, nil, aliases, form, map[string]string{
		"weight": "long", "ids": "[int]", "dict": "{boolean}", "count": "int",
	})
}

// The target is read where the alias is declared: a class it names by a simple
// name is that file's class, whichever file the alias is used in.
func TestResolve_ATypeAliasTargetIsReadWhereTheAliasIsDeclared(t *testing.T) {
	item := scopeShadow("com.example.kcmodel", "Item", "code")
	local := scopeShadow(nestedPackage, "Item", "name")
	local.FilePath = "nested/Item.kt"
	aliases := []ktparser.TypeAlias{
		aliasOf("com.example.kcalias", "alias/Aliases.kt", "Dto", "Item", nil, ktparser.Import{Path: "com.example.kcmodel.Item"}),
		aliasOf("com.example.kcalias", "alias/Aliases.kt", "Dtos", "List<Item>", nil, ktparser.Import{Path: "com.example.kcmodel.Item"}),
	}
	form := nestedForm("nested/Form.kt", scopeProp("one", "Dto"), scopeProp("many", "Dtos"), scopeProp("own", "Item"))
	form.Imports = []ktparser.Import{{Path: "com.example.kcalias.Dto"}, {Path: "com.example.kcalias.Dtos"}}
	checkAliasShapes(t, []ktparser.Class{item, local}, aliases, form, map[string]string{
		"one": "Item(code)", "many": "[Item(code)]", "own": "Item(name)",
	})
}

// A private alias is visible in its file only; one that names a function type,
// which no class stands for, stays unresolved under its own name; two that name
// each other end.
func TestResolve_ATypeAliasThatIsPrivateFunctionalOrCircular(t *testing.T) {
	aliases := []ktparser.TypeAlias{
		{Name: "Number", Package: nestedPackage, FilePath: "nested/Private.kt", Target: "Long", IsPrivate: true},
		aliasOf(nestedPackage, "nested/Aliases.kt", "Handler", "(String)->Unit", nil),
		aliasOf(nestedPackage, "nested/Aliases.kt", "First", "Second", nil),
		aliasOf(nestedPackage, "nested/Aliases.kt", "Second", "First", nil),
	}
	form := nestedForm("nested/Form.kt", scopeProp("weight", "Number"), scopeProp("handler", "Handler"), scopeProp("loop", "First"))
	for view := range scopeViews {
		t.Run(view, func(t *testing.T) {
			r := NewWithAliases([]ktparser.Class{form}, nil, aliases)
			got := scopeViews[view](r, &form, "Form")
			if shape := scopeModelShape(scopeAt(got, "weight")); shape != "double" {
				t.Errorf("Form.weight = %s, want the standard Number: the alias is private to another file", shape)
			}
			if shape := scopeModelShape(scopeAt(got, "handler")); shape != nestedPackage+".Handler" {
				t.Errorf("Form.handler = %s, want the alias under its own name", shape)
			}
			if _, reported := r.Unresolved()[nestedPackage+".Handler"]; !reported {
				t.Errorf("unresolved = %v, want the function-type alias", r.Unresolved())
			}
			if shape := scopeModelShape(scopeAt(got, "loop")); shape == "" {
				t.Errorf("Form.loop = %q", shape)
			}
		})
	}
}
