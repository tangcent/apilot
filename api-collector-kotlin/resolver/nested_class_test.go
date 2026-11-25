package resolver

import (
	"fmt"
	"reflect"
	"testing"

	ktparser "github.com/tangcent/apilot/api-collector-kotlin/parser"
)

func nestedKotlinClasses() []ktparser.Class {
	const pkg = "com.example.app"
	return []ktparser.Class{
		{Name: "Outer", Package: pkg, FilePath: "Outer.kt"},
		{Name: "Inner", Package: pkg, Outer: "Outer", FilePath: "Outer.kt"},
		{Name: "Mid", Package: pkg, Outer: "Outer", FilePath: "Outer.kt"},
		{Name: "Deep", Package: pkg, Outer: "Outer.Mid", FilePath: "Outer.kt"},
		{Name: "Badge", Package: pkg, Outer: "Outer", FilePath: "Outer.kt"},
		{Name: "Badge", Package: pkg, FilePath: "Badge.kt"},
		{Name: "Other", Package: pkg, FilePath: "Other.kt"},
		{Name: "Inner", Package: pkg, Outer: "Other", FilePath: "Other.kt"},
		{Name: "Plain", Package: pkg, FilePath: "Plain.kt"},
		{
			Name: "Imported", Package: "com.example.web", FilePath: "Imported.kt",
			Imports: []ktparser.Import{{Path: pkg + ".Outer.Inner"}},
		},
		{
			Name: "Starred", Package: "com.example.web", FilePath: "Starred.kt",
			Imports: []ktparser.Import{{Path: pkg + ".Outer", OnDemand: true}},
		},
	}
}

// FindClass finds a class nested in another from inside its outer class and
// through it, and keeps the Kotlin precedence for the rest.
func TestFindClass_ANestedClassIsFoundFromInsideAndThroughItsOuterClass(t *testing.T) {
	classes := nestedKotlinClasses()
	r := New(classes)
	byName := func(name, outer string) *ktparser.Class {
		for i := range classes {
			if classes[i].Name == name && classes[i].Outer == outer {
				return &classes[i]
			}
		}
		t.Fatalf("no class %s nested in %q", name, outer)
		return nil
	}
	outer, mid := byName("Outer", ""), byName("Mid", "Outer")
	plain, other := byName("Plain", ""), byName("Other", "")
	imported, starred := byName("Imported", ""), byName("Starred", "")

	for _, tc := range []struct {
		name   string
		source *ktparser.Class
		ref    string
		want   string
	}{
		{"a member inside its outer class", outer, "Inner", "com.example.app.Outer.Inner"},
		{"a member of the outer class, from a nested class", mid, "Inner", "com.example.app.Outer.Inner"},
		{"a member shadows the top-level class", outer, "Badge", "com.example.app.Outer.Badge"},
		{"the top-level class where nothing shadows it", plain, "Badge", "com.example.app.Badge"},
		{"a nested class through its outer class", plain, "Outer.Inner", "com.example.app.Outer.Inner"},
		{"a class nested twice", plain, "Outer.Mid.Deep", "com.example.app.Outer.Mid.Deep"},
		{"the class of another outer class", outer, "Other.Inner", "com.example.app.Other.Inner"},
		{"a qualified name", plain, "com.example.app.Outer.Inner", "com.example.app.Outer.Inner"},
		{"a single import", imported, "Inner", "com.example.app.Outer.Inner"},
		{"an import of the outer class on demand", starred, "Inner", "com.example.app.Outer.Inner"},
		{"the only nested class of a name", other, "Mid", "com.example.app.Outer.Mid"},
	} {
		got, ok := r.FindClass(tc.source, tc.ref)
		if !ok || qualifiedName(*got) != tc.want {
			t.Errorf("%s: FindClass(%s, %q) = %v, %v; want %s", tc.name, tc.source.Name, tc.ref, got, ok, tc.want)
		}
	}
	// Two nested classes carry the name and nothing in scope says which is meant.
	if got, ok := r.FindClass(plain, "Inner"); ok {
		t.Errorf("FindClass(Plain, Inner) = %+v, want none: two nested classes carry the name", got)
	}
	if _, ok := r.FindClass(plain, "Missing.Inner"); ok {
		t.Error("FindClass(Plain, Missing.Inner) found a class")
	}
}

// The supertype list of a class is outside its body, so the classes nested in
// the class are not in scope there: the Base of `class Child : Base()` is the
// class the file imports. The classes nested in the class around it are.
func TestFindSupertype_TheClassesNestedInTheSubclassAreNotInScope(t *testing.T) {
	classes := []ktparser.Class{
		{Name: "Child", Package: "p", FilePath: "Child.kt", SuperClass: "Base", Imports: []ktparser.Import{{Path: "q.Base"}}},
		{Name: "Base", Package: "p", Outer: "Child", FilePath: "Child.kt"},
		{Name: "Base", Package: "q", FilePath: "Base.kt"},
		{Name: "Outer", Package: "p", FilePath: "Outer.kt"},
		{Name: "Sibling", Package: "p", Outer: "Outer", FilePath: "Outer.kt"},
		{Name: "Nested", Package: "p", Outer: "Outer", FilePath: "Outer.kt", SuperClass: "Sibling"},
	}
	r := New(classes)

	if got, ok := r.FindSupertype(&classes[0], "Base"); !ok || qualifiedName(*got) != "q.Base" {
		t.Errorf("FindSupertype(Child, Base) = %v, %v; want q.Base", got, ok)
	}
	// In the body of Child the name is Child's own class.
	if got, ok := r.FindClass(&classes[0], "Base"); !ok || qualifiedName(*got) != "p.Child.Base" {
		t.Errorf("FindClass(Child, Base) = %v, %v; want p.Child.Base", got, ok)
	}
	if got, ok := r.FindSupertype(&classes[5], "Sibling"); !ok || qualifiedName(*got) != "p.Outer.Sibling" {
		t.Errorf("FindSupertype(Nested, Sibling) = %v, %v; want p.Outer.Sibling", got, ok)
	}
	// Written through its outer class, a nested class is named as it always was.
	if got, ok := r.FindSupertype(&classes[0], "Child.Base"); !ok || qualifiedName(*got) != "p.Child.Base" {
		t.Errorf("FindSupertype(Child, Child.Base) = %v, %v; want p.Child.Base", got, ok)
	}
}

// ToJavaClasses carries where each class is nested, enums and companions of
// the binding view included, so the Java resolver keys them by it.
func TestToJavaClasses_CarryTheClassesADeclarationIsNestedIn(t *testing.T) {
	classes := []ktparser.Class{
		{Name: "Outer", Package: "com.example"},
		{Name: "Info", Package: "com.example", Outer: "Outer"},
		{Name: "State", Package: "com.example", Outer: "Outer", IsEnum: true, EnumValues: []string{"OPEN"}},
		{Name: "Deep", Package: "com.example", Outer: "Outer.Info"},
	}
	var got []string
	for _, c := range ToJavaClasses(classes, nil, nil) {
		got = append(got, c.QualifiedName())
	}
	want := []string{"com.example.Outer", "com.example.Outer.Info", "com.example.Outer.State", "com.example.Outer.Info.Deep"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("keys = %v, want %v", got, want)
	}
	for _, c := range []ktparser.Class{classes[1], classes[2]} {
		if javaClass := (*typeScope)(nil).toBindingClass(c); javaClass.Outer != "Outer" {
			t.Errorf("toBindingClass(%s).Outer = %q, want Outer", c.Name, javaClass.Outer)
		}
	}
}

// Two nested enums of one name keep their own entries as the Java resolver
// reads them in the binding view.
func TestResolveForBinding_TwoNestedEnumsOfOneNameKeepTheirOwnEntries(t *testing.T) {
	classes := []ktparser.Class{
		{Name: "A", Package: "com.example", FilePath: "A.kt", Fields: []ktparser.Property{{Name: "status", Type: "Status", IsMutable: true}}},
		{Name: "Status", Package: "com.example", Outer: "A", FilePath: "A.kt", IsEnum: true, EnumValues: []string{"OPEN", "SHUT"}},
		{Name: "B", Package: "com.example", FilePath: "B.kt", Fields: []ktparser.Property{{Name: "status", Type: "Status", IsMutable: true}}},
		{Name: "Status", Package: "com.example", Outer: "B", FilePath: "B.kt", IsEnum: true, EnumValues: []string{"NEW", "DONE"}},
	}
	r := New(classes)

	for i, want := range [][]string{{"OPEN", "SHUT"}, {"NEW", "DONE"}} {
		holder := &classes[i*2]
		got := r.ResolveForBinding(holder, holder.Name, nil)
		var values []string
		for _, option := range got.Fields["status"].Options {
			values = append(values, fmt.Sprint(option.Value))
		}
		if !reflect.DeepEqual(values, want) {
			t.Errorf("%s.status options = %v, want %v", holder.Name, values, want)
		}
	}
}
