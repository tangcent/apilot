package resolver

import (
	"reflect"
	"sort"
	"strings"
	"testing"

	collector "github.com/tangcent/apilot/api-collector"
	"github.com/tangcent/apilot/api-collector-java/parser"
	model "github.com/tangcent/apilot/api-model"
)

const nestedPkg = "com.example.app"

// nestedClasses declares, in com.example.app: Outer, with Inner, Mid (which
// holds Deep) and Badge nested in it; a top-level Badge; Other, which nests an
// Inner of its own; Sub, which extends Outer; and Plain, which declares nothing
// of its own.
func nestedClasses() []parser.Class {
	return []parser.Class{
		{Name: "Outer", Package: nestedPkg, FilePath: "Outer.java"},
		{Name: "Inner", Package: nestedPkg, Outer: "Outer", FilePath: "Outer.java", Fields: []parser.Field{{Name: "ofOuter", Type: "String"}}},
		{Name: "Mid", Package: nestedPkg, Outer: "Outer", FilePath: "Outer.java"},
		{Name: "Deep", Package: nestedPkg, Outer: "Outer.Mid", FilePath: "Outer.java"},
		{Name: "Badge", Package: nestedPkg, Outer: "Outer", FilePath: "Outer.java", Fields: []parser.Field{{Name: "nestedBadge", Type: "String"}}},
		{Name: "Badge", Package: nestedPkg, FilePath: "Badge.java", Fields: []parser.Field{{Name: "topBadge", Type: "String"}}},
		{Name: "Other", Package: nestedPkg, FilePath: "Other.java"},
		{Name: "Inner", Package: nestedPkg, Outer: "Other", FilePath: "Other.java", Fields: []parser.Field{{Name: "ofOther", Type: "String"}}},
		{Name: "Sub", Package: nestedPkg, FilePath: "Sub.java", SuperClass: "Outer"},
		{Name: "Plain", Package: nestedPkg, FilePath: "Plain.java"},
	}
}

func nestedSource(r *TypeResolver, key string) parser.Class {
	return r.classRegistry[key]
}

func TestResolveSimpleName_ANestedClassIsFoundFromInsideAndThroughItsOuterClass(t *testing.T) {
	r := NewTypeResolver(nestedClasses())
	imported := parser.Class{
		Name: "Imported", Package: "com.example.web", FilePath: "Imported.java",
		Imports: []parser.Import{{Path: "com.example.app.Outer.Inner"}},
	}
	starred := parser.Class{
		Name: "Starred", Package: "com.example.web", FilePath: "Starred.java",
		Imports: []parser.Import{{Path: "com.example.app.Outer", OnDemand: true}},
	}

	outer := nestedSource(r, "com.example.app.Outer")
	mid := nestedSource(r, "com.example.app.Outer.Mid")
	deep := nestedSource(r, "com.example.app.Outer.Mid.Deep")
	other := nestedSource(r, "com.example.app.Other")
	sub := nestedSource(r, "com.example.app.Sub")
	plain := nestedSource(r, "com.example.app.Plain")

	for _, tc := range []struct {
		name   string
		source parser.Class
		ref    string
		want   string
	}{
		{"a member type inside its outer class", outer, "Inner", "com.example.app.Outer.Inner"},
		{"a member type of the outer class, from a nested class", mid, "Inner", "com.example.app.Outer.Inner"},
		{"a member type of the outer class, from a class nested twice", deep, "Inner", "com.example.app.Outer.Inner"},
		{"a member type of a nested class, from inside it", mid, "Deep", "com.example.app.Outer.Mid.Deep"},
		{"a class nested twice, by its path", outer, "Mid.Deep", "com.example.app.Outer.Mid.Deep"},
		{"a member type shadows the top-level class of the package", outer, "Badge", "com.example.app.Outer.Badge"},
		{"the top-level class where no member type shadows it", plain, "Badge", "com.example.app.Badge"},
		{"a member type inherited from the superclass", sub, "Inner", "com.example.app.Outer.Inner"},
		{"the class of another outer class, by its path", outer, "Other.Inner", "com.example.app.Other.Inner"},
		{"a nested class by its path, from the package", plain, "Outer.Inner", "com.example.app.Outer.Inner"},
		{"a nested class by its qualified name", plain, "com.example.app.Outer.Inner", "com.example.app.Outer.Inner"},
		{"a nested class by a single-type import", imported, "Inner", "com.example.app.Outer.Inner"},
		{"a nested class by an import of its outer class", starred, "Inner", "com.example.app.Outer.Inner"},
		{"a nested class through a class imported on demand", starred, "Mid.Deep", "com.example.app.Outer.Mid.Deep"},
		{"a binary name", plain, "Outer$Inner", "com.example.app.Outer.Inner"},
		{"a binary name, qualified", plain, "com.example.app.Outer$Mid$Deep", "com.example.app.Outer.Mid.Deep"},
		// Two classes of that name are nested in the package and nothing in scope
		// says which is meant: the name stays as written, which is unresolved.
		{"a bare name that two nested classes carry", plain, "Inner", "Inner"},
		// A bare name that no scope declares falls back to the only class that
		// carries it, as it always has (see pickByName).
		{"the only nested class of a name, from a class it is not in scope of", other, "Mid", "com.example.app.Outer.Mid"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := r.resolveName(tc.source, tc.ref); got != tc.want {
				t.Errorf("resolveName(%s, %q) = %q, want %q", tc.source.Name, tc.ref, got, tc.want)
			}
		})
	}
}

// With nothing in scope to say where a bare name comes from, the name falls
// back to a declaration that carries it: a top-level class first, and a nested
// one only when it is the only candidate.
func TestResolveSimpleName_TheFallbackDoesNotGuessBetweenNestedClasses(t *testing.T) {
	r := NewTypeResolver(nestedClasses())

	for _, tc := range []struct{ ref, want string }{
		{"Badge", "com.example.app.Badge"},
		{"Mid", "com.example.app.Outer.Mid"},
		{"Inner", "Inner"},
	} {
		if got := r.resolveName(parser.Class{}, tc.ref); got != tc.want {
			t.Errorf("resolveName(no scope, %q) = %q, want %q", tc.ref, got, tc.want)
		}
	}

	unresolved := collector.NewUnresolvedSet()
	r.SetUnresolved(unresolved)
	if got := r.Resolve("Inner", nil); got.Kind != model.KindSingle || got.TypeName != "Inner" {
		t.Errorf("Resolve(Inner) = %+v, want an opaque value: no one class is meant", got)
	}
	if got, want := unresolved.Counts(), map[string]int{"Inner": 1}; !reflect.DeepEqual(got, want) {
		t.Errorf("unresolved = %v, want %v", got, want)
	}
}

func TestResolve_TwoNestedClassesOfOneNameAreTwoClasses(t *testing.T) {
	classes := []parser.Class{
		{Name: "A", Package: nestedPkg, FilePath: "A.java", Fields: []parser.Field{
			{Name: "info", Type: "Info"},
			{Name: "wrapped", Type: "Wrapper<A.Info>"},
			{Name: "peer", Type: "Wrapper<B.Info>"},
			{Name: "all", Type: "List<B.Info>"},
		}},
		{Name: "Info", Package: nestedPkg, Outer: "A", FilePath: "A.java", Fields: []parser.Field{{Name: "ofA", Type: "String"}}},
		{Name: "B", Package: nestedPkg, FilePath: "B.java", Fields: []parser.Field{{Name: "info", Type: "Info"}}},
		{Name: "Info", Package: nestedPkg, Outer: "B", FilePath: "B.java", Fields: []parser.Field{{Name: "ofB", Type: "String"}}},
		{Name: "Wrapper", Package: nestedPkg, FilePath: "Wrapper.java", TypeParameters: []string{"T"}, Fields: []parser.Field{{Name: "data", Type: "T"}}},
		{Name: "Controller", Package: nestedPkg, FilePath: "Controller.java"},
	}
	controller := classes[len(classes)-1]

	for _, withDependencies := range []bool{false, true} {
		name := "without a dependency resolver"
		if withDependencies {
			name = "with a dependency resolver"
		}
		t.Run(name, func(t *testing.T) {
			unresolved := collector.NewUnresolvedSet()
			r := NewTypeResolver(classes)
			r.SetUnresolved(unresolved)
			if withDependencies {
				r.SetDependencyResolver(&libraryClasses{})
			}

			// Each class reaches the Info nested in itself by the bare name.
			a := r.ResolveFrom(controller, "A", nil)
			b := r.ResolveFrom(controller, "B", nil)
			assertShape(t, "A.info", a.Fields["info"].Model, "{ofA:string}")
			assertShape(t, "B.info", b.Fields["info"].Model, "{ofB:string}")

			// A type argument is read inside Wrapper, where neither Info is in
			// scope: it is qualified where it is written.
			assertShape(t, "A.wrapped", a.Fields["wrapped"].Model, "{data:{ofA:string}}")
			assertShape(t, "A.peer", a.Fields["peer"].Model, "{data:{ofB:string}}")
			assertShape(t, "A.all", a.Fields["all"].Model, "[{ofB:string}]")

			// And so it is from a class that nests neither.
			assertShape(t, "Wrapper<A.Info>", r.ResolveFrom(controller, "Wrapper<A.Info>", nil), "{data:{ofA:string}}")
			assertShape(t, "Wrapper<B.Info>", r.ResolveFrom(controller, "Wrapper<B.Info>", nil), "{data:{ofB:string}}")
			assertShape(t, "List<B.Info>", r.ResolveFrom(controller, "List<B.Info>", nil), "[{ofB:string}]")
			if counts := unresolved.Counts(); len(counts) != 0 {
				t.Errorf("unresolved = %v, want none", counts)
			}
		})
	}
}

// A type argument is qualified for the class that reads it, a nested class
// through its path (`Outer.Inner`) included, whether or not the resolver has
// dependencies; a name that nothing declares stays as it was written.
func TestQualifyTypeArgsSpellsNestedClassesInFull(t *testing.T) {
	r := NewTypeResolver(nestedClasses())
	plain := nestedSource(r, "com.example.app.Plain")

	got := r.qualifyTypeArgs(plain, nil, []string{"Plain", "Outer.Inner", "List<Outer.Inner>", "Map<String, Plain>", "Map<String, Outer.Mid.Deep>", "? extends Outer.Inner", "Unknown"})
	want := []string{
		"com.example.app.Plain", "com.example.app.Outer.Inner", "List<com.example.app.Outer.Inner>", "Map<java.lang.String, com.example.app.Plain>",
		"Map<java.lang.String, com.example.app.Outer.Mid.Deep>", "? extends com.example.app.Outer.Inner", "Unknown",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("qualifyTypeArgs:\n got  %q\n want %q", got, want)
	}
}

func TestCanonicalName(t *testing.T) {
	for _, tc := range []struct{ name, want string }{
		{"Outer", "Outer"},
		{"Outer$Inner", "Outer.Inner"},
		{"com.example.Outer$Mid$Deep", "com.example.Outer.Mid.Deep"},
		{"$Gson$Types", "$Gson$Types"},
		{"com.example.$Gson$Types", "com.example.$Gson$Types"},
		{"Outer$1", "Outer$1"},
		{"Outer$", "Outer$"},
	} {
		if got := canonicalName(tc.name); got != tc.want {
			t.Errorf("canonicalName(%q) = %q, want %q", tc.name, got, tc.want)
		}
	}
}

// A `$` may be part of an identifier: `Price$Value` is a top-level class of its
// own, not the binary name of a class Value nested in Price. A name is read as a
// binary name only when nothing carries it as written.
func TestResolve_ADollarInAClassNameIsNotABinaryName(t *testing.T) {
	classes := []parser.Class{
		{Name: "Order", Package: "p", FilePath: "Order.java", Fields: []parser.Field{
			{Name: "value", Type: "Price$Value"},
			{Name: "values", Type: "List<Price$Value>"},
			{Name: "qualified", Type: "p.Price$Value"},
			// The binary name of a nested class is still read as such.
			{Name: "binary", Type: "Outer$Inner"},
		}},
		{Name: "Price$Value", Package: "p", FilePath: "Price$Value.java", Fields: []parser.Field{{Name: "price", Type: "long"}}},
		{Name: "Outer", Package: "p", FilePath: "Outer.java"},
		{Name: "Inner", Package: "p", Outer: "Outer", FilePath: "Outer.java", Fields: []parser.Field{{Name: "ofInner", Type: "String"}}},
	}

	t.Run("a class of the scanned sources", func(t *testing.T) {
		r := NewTypeResolver(classes)
		unresolved := collector.NewUnresolvedSet()
		r.SetUnresolved(unresolved)

		assertShape(t, "Order", r.ResolveFrom(classes[0], "p.Order", nil),
			"{binary:{ofInner:string},qualified:{price:long},value:{price:long},values:[{price:long}]}")
		if got := r.resolveName(classes[0], "Price$Value"); got != "p.Price$Value" {
			t.Errorf("resolveName(Price$Value) = %q, want p.Price$Value", got)
		}
		if got := r.Resolve("Price$Value", nil); !got.IsObject() {
			t.Errorf("Resolve(Price$Value) = %+v, want the class", got)
		}
		if counts := unresolved.Counts(); len(counts) != 0 {
			t.Errorf("unresolved = %v, want none", counts)
		}
	})

	t.Run("a class of a library that the source imports by that name", func(t *testing.T) {
		library := &nestedLibrary{classes: map[string]parser.Class{
			"lib.Price$Value": {Name: "Price$Value", Package: "lib", Fields: []parser.Field{{Name: "price", Type: "long"}}},
		}}
		controller := parser.Class{
			Name: "PriceController", Package: "app", FilePath: "PriceController.java",
			Imports: []parser.Import{{Path: "lib.Price$Value"}},
		}
		r := NewTypeResolver([]parser.Class{controller})
		r.SetDependencyResolver(library)

		assertShape(t, "Price$Value", r.ResolveFrom(controller, "Price$Value", nil), "{price:long}")
		if want := []string{"lib.Price$Value"}; !reflect.DeepEqual(library.asked, want) {
			t.Errorf("asked = %v, want %v", library.asked, want)
		}
	})
}

// Two nested enums of one name hold their own constants: the key of each has the
// class it is nested in, so the one registered last does not stand for both.
func TestEnumConstants_TwoNestedEnumsOfOneNameKeepTheirOwnConstants(t *testing.T) {
	classes := []parser.Class{
		{Name: "A", Package: nestedPkg, FilePath: "A.java"},
		{Name: "Status", Package: nestedPkg, Outer: "A", FilePath: "A.java", IsEnum: true, EnumConstants: []string{"OPEN", "SHUT"}},
		{Name: "B", Package: nestedPkg, FilePath: "B.java"},
		{Name: "Status", Package: nestedPkg, Outer: "B", FilePath: "B.java", IsEnum: true, EnumConstants: []string{"NEW", "DONE"}},
		{Name: "Plain", Package: nestedPkg, FilePath: "Plain.java"},
	}
	r := NewTypeResolver(classes)

	for _, tc := range []struct {
		source int
		ref    string
		want   []string
	}{
		{0, "Status", []string{"OPEN", "SHUT"}},
		{2, "Status", []string{"NEW", "DONE"}},
		{4, "A.Status", []string{"OPEN", "SHUT"}},
		{4, "B.Status", []string{"NEW", "DONE"}},
		{4, "com.example.app.B.Status", []string{"NEW", "DONE"}},
	} {
		got, ok := r.EnumConstants(classes[tc.source], tc.ref)
		if !ok || !reflect.DeepEqual(got, tc.want) {
			t.Errorf("EnumConstants(%s, %q) = %v, %v; want %v", classes[tc.source].Name, tc.ref, got, ok, tc.want)
		}
	}
	// A bare name that no scope declares is not guessed between the two.
	if got, ok := r.EnumConstants(classes[4], "Status"); ok {
		t.Errorf("EnumConstants(Plain, Status) = %v, want none: two enums carry the name", got)
	}
}

func TestConstantValue_TwoNestedClassesOfOneNameKeepTheirOwnConstants(t *testing.T) {
	classes := []parser.Class{
		{Name: "A", Package: nestedPkg, FilePath: "A.java"},
		{Name: "Names", Package: nestedPkg, Outer: "A", FilePath: "A.java", Fields: []parser.Field{constant("KEY", `"a-key"`)}},
		{Name: "B", Package: nestedPkg, FilePath: "B.java"},
		{Name: "Names", Package: nestedPkg, Outer: "B", FilePath: "B.java", Fields: []parser.Field{constant("KEY", `"b-key"`)}},
		{Name: "Plain", Package: nestedPkg, FilePath: "Plain.java"},
	}
	r := NewTypeResolver(classes)

	for _, tc := range []struct {
		source int
		ref    string
		want   string
		ok     bool
	}{
		{0, "Names.KEY", "a-key", true},
		{2, "Names.KEY", "b-key", true},
		{4, "A.Names.KEY", "a-key", true},
		{4, "B.Names.KEY", "b-key", true},
		{4, "Names.KEY", "", false},
	} {
		if got, ok := r.ConstantValue(classes[tc.source], tc.ref); got != tc.want || ok != tc.ok {
			t.Errorf("ConstantValue(%s, %q) = %q, %v; want %q, %v", classes[tc.source].Name, tc.ref, got, ok, tc.want, tc.ok)
		}
	}
}

// An interface nested in one class is not implemented by a class that
// implements another interface of the same name, nested elsewhere.
func TestResolve_ANestedInterfaceIsNotImplementedByTheSameNameElsewhere(t *testing.T) {
	classes := []parser.Class{
		{Name: "A", Package: nestedPkg, FilePath: "A.java", Fields: []parser.Field{{Name: "handler", Type: "Handler"}}},
		{Name: "Handler", Package: nestedPkg, Outer: "A", FilePath: "A.java", IsInterface: true},
		{Name: "Impl", Package: nestedPkg, Outer: "A", FilePath: "A.java", Interfaces: []string{"Handler"}, Fields: []parser.Field{{Name: "ofA", Type: "String"}}},
		{Name: "B", Package: nestedPkg, FilePath: "B.java", Fields: []parser.Field{{Name: "handler", Type: "Handler"}}},
		{Name: "Handler", Package: nestedPkg, Outer: "B", FilePath: "B.java", IsInterface: true},
		{Name: "Impl", Package: nestedPkg, Outer: "B", FilePath: "B.java", Interfaces: []string{"Handler"}, Fields: []parser.Field{{Name: "ofB", Type: "String"}}},
	}
	r := NewTypeResolver(classes)

	assertShape(t, "A.handler", r.ResolveFrom(classes[0], "A", nil).Fields["handler"].Model, "{ofA:string}")
	assertShape(t, "B.handler", r.ResolveFrom(classes[3], "B", nil).Fields["handler"].Model, "{ofB:string}")
}

// The member types a class inherits are those of its supertypes as the header of
// the class names them: here Base is imported, and Item is nested in it.
func TestResolveSimpleName_AMemberTypeIsInheritedThroughAnImportedSuperclass(t *testing.T) {
	classes := []parser.Class{
		{Name: "Base", Package: "com.example.base", FilePath: "Base.java"},
		{Name: "Item", Package: "com.example.base", Outer: "Base", FilePath: "Base.java"},
		{Name: "Contract", Package: "com.example.base", FilePath: "Contract.java", IsInterface: true},
		{Name: "Rule", Package: "com.example.base", Outer: "Contract", FilePath: "Contract.java"},
		{
			Name: "Child", Package: nestedPkg, FilePath: "Child.java", SuperClass: "Base", Interfaces: []string{"Contract"},
			Imports: []parser.Import{{Path: "com.example.base.Base"}, {Path: "com.example.base.Contract"}},
		},
		{Name: "Item", Package: nestedPkg, FilePath: "Item.java"},
		{Name: "Loop", Package: nestedPkg, FilePath: "Loop.java", SuperClass: "Loop"},
	}
	r := NewTypeResolver(classes)
	child := classes[4]

	for _, tc := range []struct{ ref, want string }{
		{"Item", "com.example.base.Base.Item"},
		{"Rule", "com.example.base.Contract.Rule"},
		{"Child.Item", "com.example.base.Base.Item"},
	} {
		if got := r.resolveName(child, tc.ref); got != tc.want {
			t.Errorf("resolveName(Child, %q) = %q, want %q", tc.ref, got, tc.want)
		}
	}
	// A class that names itself as its superclass ends the search.
	if got := r.resolveName(classes[6], "Item"); got != "com.example.app.Item" {
		t.Errorf("resolveName(Loop, Item) = %q, want the class of the package", got)
	}
}

// The extends and implements clauses of a class are outside its body, and the
// member types of a class are in scope inside the body only. `Base` in
// `class Child extends Base` is the class the file imports, not the `Child.Base`
// that Child declares.
func TestResolve_TheHeaderOfAClassDoesNotSeeItsOwnMemberTypes(t *testing.T) {
	child := parser.Class{
		Name: "Child", Package: "p", FilePath: "Child.java", SuperClass: "Base", Interfaces: []string{"Consts"},
		Imports: []parser.Import{{Path: "q.Base"}, {Path: "q.Consts"}},
		Fields:  []parser.Field{{Name: "own", Type: "String"}},
	}
	// The member types of Child that the imported ones must not be mistaken for.
	nestedBase := parser.Class{Name: "Base", Package: "p", Outer: "Child", FilePath: "Child.java", Fields: []parser.Field{{Name: "wrong", Type: "String"}}}
	nestedConsts := parser.Class{Name: "Consts", Package: "p", Outer: "Child", FilePath: "Child.java", IsInterface: true, Fields: []parser.Field{constant("KEY", `"nested-key"`)}}
	base := parser.Class{Name: "Base", Package: "q", FilePath: "Base.java", Fields: []parser.Field{{Name: "inherited", Type: "String"}}}
	consts := parser.Class{Name: "Consts", Package: "q", FilePath: "Consts.java", IsInterface: true, Fields: []parser.Field{constant("KEY", `"imported-key"`)}}
	caller := parser.Class{Name: "Caller", Package: "app", FilePath: "Caller.java"}
	scanned := []parser.Class{caller, child, nestedBase, nestedConsts, base, consts}

	t.Run("a class of the scanned sources", func(t *testing.T) {
		r := NewTypeResolver(scanned)

		assertShape(t, "Child", r.ResolveFrom(caller, "p.Child", nil), "{inherited:string,own:string}")
		if got, ok := r.Supertype(child, "Base"); !ok || got.Package != "q" {
			t.Errorf("Supertype(Child, Base) = %+v, %v, want q.Base", got, ok)
		}
		if got, ok := r.ConstantValue(child, "KEY"); !ok || got != "imported-key" {
			t.Errorf("ConstantValue(Child, KEY) = %q, %v, want the imported interface's", got, ok)
		}
	})

	t.Run("a class read from the dependencies", func(t *testing.T) {
		library := &nestedLibrary{
			classes:  map[string]parser.Class{"p.Child": child, "p.Child.Base": nestedBase, "p.Child.Consts": nestedConsts, "q.Base": base, "q.Consts": consts},
			siblings: map[string][]parser.Class{"p.Child": {nestedBase, nestedConsts}},
		}
		r := NewTypeResolver([]parser.Class{caller})
		r.SetDependencyResolver(library)
		// The superclass and the interface were read earlier.
		r.ResolveFrom(caller, "q.Base", nil)
		r.ResolveFrom(caller, "q.Consts", nil)

		assertShape(t, "Child", r.ResolveFrom(caller, "p.Child", nil), "{inherited:string,own:string}")
		if got, ok := r.ConstantValue(child, "KEY"); !ok || got != "imported-key" {
			t.Errorf("ConstantValue(Child, KEY) = %q, %v, want the imported interface's", got, ok)
		}
	})

	t.Run("a type argument of the superclass", func(t *testing.T) {
		holder := parser.Class{
			Name: "Holder", Package: "p", FilePath: "Holder.java", SuperClass: "Wrapper", SuperClassTypeArgs: []string{"Item"},
			Imports: []parser.Import{{Path: "q.Wrapper"}},
		}
		nestedItem := parser.Class{Name: "Item", Package: "p", Outer: "Holder", FilePath: "Holder.java", Fields: []parser.Field{{Name: "wrong", Type: "String"}}}
		wrapper := parser.Class{Name: "Wrapper", Package: "q", FilePath: "Wrapper.java", TypeParameters: []string{"T"}, Fields: []parser.Field{{Name: "data", Type: "T"}}}
		item := parser.Class{Name: "Item", Package: "q", FilePath: "Item.java", Fields: []parser.Field{{Name: "right", Type: "String"}}}
		holder.Imports = append(holder.Imports, parser.Import{Path: "q.Item"})
		r := NewTypeResolver([]parser.Class{caller, holder, nestedItem, wrapper, item})

		assertShape(t, "Holder", r.ResolveFrom(caller, "p.Holder", nil), "{data:{right:string}}")
	})

	t.Run("the body of the class still sees them", func(t *testing.T) {
		withField := child
		withField.Fields = []parser.Field{{Name: "base", Type: "Base"}}
		r := NewTypeResolver([]parser.Class{caller, withField, nestedBase, nestedConsts, base, consts})

		assertShape(t, "Child", r.ResolveFrom(caller, "p.Child", nil), "{base:{wrong:string},inherited:string}")
	})
}

// nestedLibrary answers dependency lookups as the dependency resolver does: by
// qualified name, a nested class by its path, and the other classes of a source
// with the class that was read.
type nestedLibrary struct {
	classes  map[string]parser.Class
	siblings map[string][]parser.Class
	asked    []string
}

func (l *nestedLibrary) ResolveClass(name string) *parser.Class {
	l.asked = append(l.asked, name)
	if c, ok := l.classes[name]; ok {
		return &c
	}
	return nil
}

// Siblings returns the other classes of the source a class was read from: for a
// class nested in another, that includes the class that declares it.
func (l *nestedLibrary) Siblings(name string) []parser.Class {
	for outer, siblings := range l.siblings {
		switch {
		case name == outer:
			return siblings
		case strings.HasPrefix(name, outer+"."):
			return append([]parser.Class{l.classes[outer]}, siblings...)
		}
	}
	return nil
}

func newNestedLibrary() *nestedLibrary {
	const pkg = "com.example.lib"
	// Page is a decompiled class: it spells a nested class by its binary name.
	page := parser.Class{Name: "Page", Package: pkg, Decompiled: true, TypeParameters: []string{"T"}, Fields: []parser.Field{
		{Name: "rows", Type: "List<T>"},
		{Name: "sort", Type: "Sort"},
		{Name: "footer", Type: "Page.Footer"},
		{Name: "legacy", Type: "Page$Legacy"},
	}}
	sort := parser.Class{Name: "Sort", Package: pkg, Outer: "Page", Fields: []parser.Field{{Name: "field", Type: "String"}, {Name: "order", Type: "Sort.Order"}}}
	order := parser.Class{Name: "Order", Package: pkg, Outer: "Page.Sort", Fields: []parser.Field{{Name: "asc", Type: "boolean"}}}
	footer := parser.Class{Name: "Footer", Package: pkg, Outer: "Page", Fields: []parser.Field{{Name: "note", Type: "String"}}}
	legacy := parser.Class{Name: "Legacy", Package: pkg, Outer: "Page", Fields: []parser.Field{{Name: "old", Type: "String"}}}
	return &nestedLibrary{
		classes: map[string]parser.Class{
			pkg + ".Page": page, pkg + ".Page.Sort": sort, pkg + ".Page.Sort.Order": order,
			pkg + ".Page.Footer": footer, pkg + ".Page.Legacy": legacy,
		},
		siblings: map[string][]parser.Class{pkg + ".Page": {sort, order, footer, legacy}},
	}
}

func TestResolve_ANestedClassOfALibraryClassComesWithTheClass(t *testing.T) {
	const pkg = "com.example.lib"
	controller := parser.Class{
		Name: "PageController", Package: nestedPkg, FilePath: "PageController.java",
		Imports: []parser.Import{{Path: pkg + ".Page"}},
	}

	t.Run("the class is read first", func(t *testing.T) {
		library := newNestedLibrary()
		r := NewTypeResolver([]parser.Class{controller})
		r.SetDependencyResolver(library)

		page := r.ResolveFrom(controller, "Page<String>", nil)
		assertShape(t, "Page", page, "{footer:{note:string},legacy:{old:string},rows:[string],sort:{field:string,order:{asc:boolean}}}")
		assertShape(t, "Page.Sort", r.ResolveFrom(controller, "Page.Sort", nil), "{field:string,order:{asc:boolean}}")
		assertShape(t, "Page$Footer", r.ResolveFrom(controller, "Page$Footer", nil), "{note:string}")
		assertShape(t, "qualified", r.ResolveFrom(controller, pkg+".Page.Footer", nil), "{note:string}")
		if want := []string{pkg + ".Page"}; !reflect.DeepEqual(library.asked, want) {
			t.Errorf("asked = %v, want %v: the classes nested in Page come with it", library.asked, want)
		}
	})

	t.Run("a nested class is read first", func(t *testing.T) {
		library := newNestedLibrary()
		r := NewTypeResolver([]parser.Class{controller})
		r.SetDependencyResolver(library)

		assertShape(t, "Page.Sort", r.ResolveFrom(controller, "Page.Sort", nil), "{field:string,order:{asc:boolean}}")
		// The class that declares it, and the ones next to it, are known now.
		assertShape(t, "Page.Footer", r.ResolveFrom(controller, "Page.Footer", nil), "{note:string}")
		assertShape(t, "Page", r.ResolveFrom(controller, "Page<String>", nil), "{footer:{note:string},legacy:{old:string},rows:[string],sort:{field:string,order:{asc:boolean}}}")
		if want := []string{pkg + ".Page.Sort"}; !reflect.DeepEqual(library.asked, want) {
			t.Errorf("asked = %v, want %v", library.asked, want)
		}
	})

	t.Run("a class imported on demand names the classes nested in it", func(t *testing.T) {
		library := newNestedLibrary()
		starred := parser.Class{
			Name: "StarController", Package: nestedPkg, FilePath: "StarController.java",
			Imports: []parser.Import{{Path: pkg + ".Page", OnDemand: true}},
		}
		r := NewTypeResolver([]parser.Class{starred})
		r.SetDependencyResolver(library)

		assertShape(t, "Footer", r.ResolveFrom(starred, "Footer", nil), "{note:string}")
		assertShape(t, "Sort", r.ResolveFrom(starred, "Sort", nil), "{field:string,order:{asc:boolean}}")
		// The package of the source is one of the scanned sources and is not asked;
		// the imported class is, once, and answers for the rest.
		want := []string{pkg + ".Page.Footer"}
		if !reflect.DeepEqual(library.asked, want) {
			t.Errorf("asked = %v, want %v", library.asked, want)
		}
	})
}

// A class nested in a class of the project is a declaration of the project. The
// dependencies are not asked for one that is not there: they would read the
// class of the project again, from some release of it installed elsewhere.
func TestResolve_AMemberOfAClassOfTheProjectIsNotAskedOfTheDependencies(t *testing.T) {
	library := newNestedLibrary()
	library.classes[nestedPkg+".Outer.Inner"] = parser.Class{Name: "Inner", Package: nestedPkg, Outer: "Outer", Fields: []parser.Field{{Name: "installed", Type: "String"}}}
	library.classes[nestedPkg+".Outer.Gone"] = parser.Class{Name: "Gone", Package: nestedPkg, Outer: "Outer"}
	r := NewTypeResolver(nestedClasses())
	r.SetDependencyResolver(library)
	plain := nestedSource(r, "com.example.app.Plain")

	assertShape(t, "Outer.Inner", r.ResolveFrom(plain, "Outer.Inner", nil), "{ofOuter:string}")
	if got := r.ResolveFrom(plain, "Outer.Gone", nil); got.Kind != model.KindSingle {
		t.Errorf("Outer.Gone = %+v, want an opaque value: Outer declares no such class", got)
	}
	if len(library.asked) != 0 {
		t.Errorf("the dependencies were asked for %v", library.asked)
	}
}

func assertShape(t *testing.T, what string, m *model.ObjectModel, want string) {
	t.Helper()
	if got := shapeOf(m); got != want {
		t.Errorf("%s:\n got  %s\n want %s", what, got, want)
	}
}

// shapeOf spells a model as its structure: an object by the shape of each of its
// fields in name order, an array by the shape of its items, any other by its
// type name.
func shapeOf(m *model.ObjectModel) string {
	switch {
	case m == nil:
		return "nil"
	case m.IsArray():
		return "[" + shapeOf(m.Items) + "]"
	case m.IsObject():
		names := make([]string, 0, len(m.Fields))
		for name := range m.Fields {
			names = append(names, name)
		}
		sort.Strings(names)
		parts := make([]string, len(names))
		for i, name := range names {
			parts[i] = name + ":" + shapeOf(m.Fields[name].Model)
		}
		return "{" + strings.Join(parts, ",") + "}"
	}
	return m.TypeName
}
