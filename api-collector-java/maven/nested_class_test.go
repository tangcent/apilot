package maven

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/tangcent/apilot/api-collector-java/parser"
)

// nestedSource declares Page, the classes Sort and Footer nested in it, and Order
// nested in Sort, as a sources JAR holds them. Decompiled, the same classes
// refer to each other as Page.Sort or Page$Sort.
const nestedSource = `package com.example.lib;

public class Page<T> {
    private Sort sort;
    private Page.Footer footer;

    public static class Sort {
        private Order order;

        public static class Order {
            private boolean asc;
        }
    }

    public static class Footer {
        private String note;
    }
}`

// nestedIndex answers `explore` as the indexer does for a source that declares
// nested classes: only the class that declares them is a name of the index, and
// any other name is answered with no class. It records the name of each call.
func nestedIndex(t *testing.T, held map[string]string, asked *[]string) *MavenDependencyResolver {
	t.Helper()
	r, err := NewMavenDependencyResolver()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(r.Close)
	r.SetRunner(func(args ...string) ([]byte, error) {
		name := args[2]
		*asked = append(*asked, name)
		classes := []map[string]any{}
		if source, ok := held[name]; ok {
			classes = append(classes, map[string]any{"className": name, "artifact": "com.example:lib:1.0.0", "language": "java", "source": source})
		}
		return json.Marshal(map[string]any{"classes": classes})
	})
	return r
}

func TestResolveClass_ANestedClassIsReadFromTheSourceOfItsOuterClass(t *testing.T) {
	var asked []string
	r := nestedIndex(t, map[string]string{"com.example.lib.Page": nestedSource}, &asked)

	for _, tc := range []struct {
		name  string
		want  string
		outer string
	}{
		{"com.example.lib.Page.Sort", "Sort", "Page"},
		{"com.example.lib.Page$Sort", "Sort", "Page"},
		{"com.example.lib.Page.Sort.Order", "Order", "Page.Sort"},
		{"com.example.lib.Page$Sort$Order", "Order", "Page.Sort"},
		{"com.example.lib.Page.Footer", "Footer", "Page"},
		{"com.example.lib.Page", "Page", ""},
	} {
		class := r.ResolveClass(tc.name)
		if class == nil || class.Name != tc.want || class.Outer != tc.outer {
			t.Errorf("ResolveClass(%q) = %+v, want %s nested in %q", tc.name, class, tc.want, tc.outer)
		}
	}
	// The outer class is asked for once, and no dotted nested name is: the index
	// holds none, and the answer for a name it holds is cached. A binary name is
	// asked for as written first, for it may be the name of a top-level class, and
	// that is a miss of the index.
	if want := []string{"com.example.lib.Page", "com.example.lib.Page$Sort", "com.example.lib.Page$Sort$Order"}; !reflect.DeepEqual(asked, want) {
		t.Errorf("asked = %v, want %v", asked, want)
	}

	// A class the source does not declare is no class, and is not asked for
	// again.
	for i := 0; i < 2; i++ {
		if class := r.ResolveClass("com.example.lib.Page.Missing"); class != nil {
			t.Errorf("Page.Missing = %+v, want none", class)
		}
	}
	if len(asked) != 3 {
		t.Errorf("asked = %v, want no call for a nested class of a class that was read", asked)
	}
	// Only the classes read count as resolved: Page, and the nested classes
	// asked for by their names, a binary name counted with the dotted one.
	if got := r.ResolvedCount(); got != 4 {
		t.Errorf("ResolvedCount = %d, want 4", got)
	}
}

// A `$` may be part of the name of a top-level class. A name that reads as the
// binary name of a nested class is read as one first, and as written when that
// finds no class.
func TestResolveClass_ADollarInTheNameOfATopLevelClassIsPartOfItsName(t *testing.T) {
	var asked []string
	r := nestedIndex(t, map[string]string{
		"com.example.lib.Price$Value": "package com.example.lib;\n\npublic class Price$Value {\n    private long price;\n    public static class Part {\n        private String note;\n    }\n}",
	}, &asked)

	class := r.ResolveClass("com.example.lib.Price$Value")
	if class == nil || class.Name != "Price$Value" || class.Outer != "" || fieldNames(class)[0] != "price" {
		t.Fatalf("ResolveClass(Price$Value) = %+v, want the class named so", class)
	}
	count := len(asked)
	if again := r.ResolveClass("com.example.lib.Price$Value"); again != class {
		t.Errorf("a second lookup = %+v, want the class that was read", again)
	}
	if len(asked) != count {
		t.Errorf("asked = %v, want no call for a class that was read", asked)
	}
	// The classes nested in it come with it, as they do for any class.
	var siblings []string
	for _, c := range r.Siblings("com.example.lib.Price$Value") {
		siblings = append(siblings, c.QualifiedName())
	}
	if want := []string{"com.example.lib.Price$Value.Part"}; !reflect.DeepEqual(siblings, want) {
		t.Errorf("Siblings(Price$Value) = %v, want %v", siblings, want)
	}
	// A name that no class carries as written stays a miss.
	if missing := r.ResolveClass("com.example.lib.Price$Gone"); missing != nil {
		t.Errorf("ResolveClass(Price$Gone) = %+v, want none", missing)
	}
}

func TestResolveClass_AClassWhoseOuterClassIsNotInTheIndexIsAskedForAsWritten(t *testing.T) {
	var asked []string
	// A package that starts with a capital letter reads as a class.
	r := nestedIndex(t, map[string]string{"com.Acme.Order": "package com.Acme;\n\npublic class Order {\n    private String no;\n}"}, &asked)

	class := r.ResolveClass("com.Acme.Order")
	if class == nil || class.Name != "Order" {
		t.Fatalf("class = %+v, want Order", class)
	}
	if want := []string{"com.Acme", "com.Acme.Order"}; !reflect.DeepEqual(asked, want) {
		t.Errorf("asked = %v, want %v", asked, want)
	}
}

func TestSiblings(t *testing.T) {
	var asked []string
	r := nestedIndex(t, map[string]string{"com.example.lib.Page": nestedSource}, &asked)

	names := func(classes []parser.Class) []string {
		var out []string
		for _, c := range classes {
			out = append(out, c.QualifiedName())
		}
		return out
	}

	if got := r.Siblings("com.example.lib.Page"); got != nil {
		t.Errorf("Siblings before the class is read = %v, want none", names(got))
	}
	r.ResolveClass("com.example.lib.Page")
	want := []string{"com.example.lib.Page.Sort", "com.example.lib.Page.Sort.Order", "com.example.lib.Page.Footer"}
	if got := names(r.Siblings("com.example.lib.Page")); !reflect.DeepEqual(got, want) {
		t.Errorf("Siblings(Page) = %v, want %v", got, want)
	}
	// For a nested class they are the classes of its source, and the class that
	// declares it comes with them.
	want = append([]string{"com.example.lib.Page"}, want...)
	if got := names(r.Siblings("com.example.lib.Page$Sort")); !reflect.DeepEqual(got, want) {
		t.Errorf("Siblings(Page$Sort) = %v, want %v", got, want)
	}
	if got := r.Siblings("com.example.lib.Other"); got != nil {
		t.Errorf("Siblings(Other) = %v, want none", names(got))
	}
}

// A source that declares two top-level classes serves each of them to its own
// lookup: only the classes nested in the one asked for are kept, so a class is
// not read from the source of another, which the index may hold in another
// version.
func TestResolveClass_OnlyTheClassesNestedInTheClassAskedForAreKept(t *testing.T) {
	const source = `package com.example.lib;

public class First {
    public static class Part {
        private String ofFirst;
    }
}

class Second {
    public static class Part {
        private String ofSecond;
    }
}`
	var asked []string
	r := nestedIndex(t, map[string]string{"com.example.lib.First": source, "com.example.lib.Second": source}, &asked)

	if first := r.ResolveClass("com.example.lib.First.Part"); first == nil || fieldNames(first)[0] != "ofFirst" {
		t.Errorf("First.Part = %+v, want its own", first)
	}
	var kept []string
	for _, c := range r.Siblings("com.example.lib.First") {
		kept = append(kept, c.QualifiedName())
	}
	if want := []string{"com.example.lib.First.Part"}; !reflect.DeepEqual(kept, want) {
		t.Errorf("Siblings(First) = %v, want %v", kept, want)
	}
	if second := r.ResolveClass("com.example.lib.Second.Part"); second == nil || fieldNames(second)[0] != "ofSecond" {
		t.Errorf("Second.Part = %+v, want its own", second)
	}
	if want := []string{"com.example.lib.First", "com.example.lib.Second"}; !reflect.DeepEqual(asked, want) {
		t.Errorf("asked = %v, want %v: Second is asked for on its own", asked, want)
	}
}

func TestPickClass_TheQualifiedNameIsTheOneAskedFor(t *testing.T) {
	classes := []parser.Class{
		{Name: "Info", Package: "p", Outer: "A"},
		{Name: "Info", Package: "p"},
		{Name: "Info", Package: "p", Outer: "B"},
	}
	for _, tc := range []struct {
		name string
		want int
	}{
		{"p.A.Info", 0},
		{"p.A$Info", 0},
		{"p.Info", 1},
		{"p.B.Info", 2},
		// Not a name of the source: a top-level class of the simple name, as
		// before nested classes were told apart.
		{"q.Info", 1},
	} {
		if got := pickClass(classes, tc.name); got != &classes[tc.want] {
			t.Errorf("pickClass(%q) = %+v, want %+v", tc.name, got, classes[tc.want])
		}
	}
	if got := pickClass(classes[:1], "q.Info"); got != &classes[0] {
		t.Errorf("pickClass without a top-level declaration = %+v, want the one that carries the name", got)
	}
}

func TestCanonicalClassName(t *testing.T) {
	for _, tc := range []struct{ name, want string }{
		{"com.example.Page", "com.example.Page"},
		{"com.example.Page$Sort", "com.example.Page.Sort"},
		{"com.example.Page$Sort$Order", "com.example.Page.Sort.Order"},
		{"com.example.internal.$Gson$Types", "com.example.internal.$Gson$Types"},
		{"com.example.Page$1", "com.example.Page$1"},
		{"com.example.Page$", "com.example.Page$"},
	} {
		if got := canonicalClassName(tc.name); got != tc.want {
			t.Errorf("canonicalClassName(%q) = %q, want %q", tc.name, got, tc.want)
		}
	}
}

func TestOutermostClass(t *testing.T) {
	for _, tc := range []struct {
		name   string
		want   string
		nested bool
	}{
		{"com.example.Page", "com.example.Page", false},
		{"com.example.Page.Sort", "com.example.Page", true},
		{"com.example.Page.Sort.Order", "com.example.Page", true},
		{"Page.Sort", "Page", true},
		{"Page", "Page", false},
		{"com.example.model", "com.example.model", false},
		{"com..Page", "com..Page", false},
	} {
		if got, nested := outermostClass(tc.name); got != tc.want || nested != tc.nested {
			t.Errorf("outermostClass(%q) = %q, %v; want %q, %v", tc.name, got, nested, tc.want, tc.nested)
		}
	}
}

// A source the indexer decompiled from bytecode says so above its package, and
// every class of it is marked: its names are those of bytecode.
func TestResolveClass_MarksTheClassesOfADecompiledSource(t *testing.T) {
	const decompiled = "/*\n * Decompiled with CFR 0.152.\n */\npackage com.example.lib;\n\npublic class Page {\n    public static class Sort {\n        private Page$Footer footer;\n    }\n}"
	const written = "package com.example.lib;\n\n/** Page, as the sources JAR holds it; it is not decompiled. */\npublic class Plain {\n    public static class Sort {\n    }\n}"
	var asked []string
	r := nestedIndex(t, map[string]string{"com.example.lib.Page": decompiled, "com.example.lib.Plain": written}, &asked)

	for _, name := range []string{"com.example.lib.Page", "com.example.lib.Page.Sort"} {
		if class := r.ResolveClass(name); class == nil || !class.Decompiled {
			t.Errorf("ResolveClass(%q) = %+v, want a class marked as decompiled", name, class)
		}
	}
	for _, name := range []string{"com.example.lib.Plain", "com.example.lib.Plain.Sort"} {
		if class := r.ResolveClass(name); class == nil || class.Decompiled {
			t.Errorf("ResolveClass(%q) = %+v, want a class read from the source a developer wrote", name, class)
		}
	}
}

func TestIsDecompiled(t *testing.T) {
	for _, tc := range []struct {
		source string
		want   bool
	}{
		{"/*\n * Decompiled with CFR 0.152.\n */\npackage p;\nclass A {}", true},
		{"// Decompiled by Procyon v0.6.0\npackage p;\nclass A {}", true},
		{"// Source code recreated from a .class file by IntelliJ IDEA\n// (powered by FernFlower decompiler)\npackage p;\nclass A {}", true},
		{"package p;\n\nclass A {}", false},
		// A word in the class's own documentation, after the package, is no mark.
		{"package p;\n\n/** Reads what a decompiler printed. */\nclass A {}", false},
	} {
		if got := isDecompiled(tc.source); got != tc.want {
			t.Errorf("isDecompiled(%q) = %v, want %v", tc.source, got, tc.want)
		}
	}
}

// A class declared in a method is in scope in that method only, so the classes
// kept with a class do not include it.
func TestResolveClass_ALocalClassIsNotKeptWithTheClassThatDeclaresIt(t *testing.T) {
	const source = "package com.example.lib;\n\npublic class Page {\n    public static class Sort {\n    }\n\n    void run() {\n        class Row {\n        }\n    }\n}"
	var asked []string
	r := nestedIndex(t, map[string]string{"com.example.lib.Page": source}, &asked)

	r.ResolveClass("com.example.lib.Page")
	var kept []string
	for _, c := range r.Siblings("com.example.lib.Page") {
		kept = append(kept, c.QualifiedName())
	}
	if want := []string{"com.example.lib.Page.Sort"}; !reflect.DeepEqual(kept, want) {
		t.Errorf("Siblings(Page) = %v, want %v", kept, want)
	}
	if class := r.ResolveClass("com.example.lib.Page.Row"); class != nil {
		t.Errorf("Page.Row = %+v, want none: a local class has no name outside its method", class)
	}
}
