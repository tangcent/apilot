package resolver

import (
	"reflect"
	"testing"

	"github.com/tangcent/apilot/api-collector-java/parser"
	model "github.com/tangcent/apilot/api-model"
)

// A type variable is in scope in the class that declares it, so a class that
// declares `<Item>` does not hide a class named Item anywhere else, whether the
// class is a class of the project or one a library declares and the resolver reads
// later, and whatever was read before.
func TestResolve_ATypeVariableOfAnotherClassDoesNotHideAClass(t *testing.T) {
	library := func() *libraryClasses {
		return &libraryClasses{classes: map[string]parser.Class{
			"lib.b.Item": marked("lib.b", "Item", "libB"),
			"lib.b.Node": {Name: "Node", Package: "lib.b", TypeParameters: []string{"Item"}, Fields: []parser.Field{{Name: "first", Type: "Item"}}},
		}}
	}
	holder := parser.Class{Name: "Holder", Package: "app.web", FilePath: "Holder.java", Imports: onDemand("lib.b"), Fields: []parser.Field{{Name: "item", Type: "Item"}}}
	node := parser.Class{Name: "Node", Package: "app.web", FilePath: "Node.java", TypeParameters: []string{"Item"}, Fields: []parser.Field{{Name: "first", Type: "Item"}}}

	for name, tc := range map[string]struct {
		project []parser.Class
		warm    []string
	}{
		"nothing declares a type parameter Item":      {project: []parser.Class{holder}},
		"a class of the project declares <Item>":      {project: []parser.Class{holder, node}},
		"a library class declares <Item>, read first": {project: []parser.Class{holder}, warm: []string{"lib.b.Node"}},
	} {
		t.Run(name, func(t *testing.T) {
			r := NewTypeResolver(tc.project)
			r.SetDependencyResolver(library())
			for _, w := range tc.warm {
				r.Resolve(w, nil)
			}
			if got := fieldsOf(r.ResolveFrom(holder, "app.web.Holder", nil).Fields["item"].Model); !reflect.DeepEqual(got, []string{"libB"}) {
				t.Errorf("Holder.item has %v, want the class lib.b.Item names", got)
			}
		})
	}

	t.Run("a type variable in scope is not asked for", func(t *testing.T) {
		lib := library()
		r := NewTypeResolver([]parser.Class{holder})
		r.SetDependencyResolver(lib)
		boxed := parser.Class{Name: "Box", Package: "app.web", FilePath: "Box.java", TypeParameters: []string{"Item"}, Imports: onDemand("lib.b"), Fields: []parser.Field{{Name: "value", Type: "Item"}}}

		got := r.ResolveFrom(boxed, "String", nil)
		if got.Kind != model.KindSingle {
			t.Fatalf("String = %+v", got)
		}
		field := r.resolveField(boxed, boxed.Fields[0], map[string]string{"Item": "String"}, map[string]bool{"Item": true})
		if field.Model.Kind != model.KindSingle {
			t.Errorf("Box<String>.value = %+v, want a string", field.Model)
		}
		for _, name := range lib.asked {
			if name == "lib.b.Item" || name == "app.web.Item" {
				t.Errorf("%s was asked for, though Item is a type variable of Box", name)
			}
		}
	})
}

// The names of JDK value types are answered for after the scope, so a class that an
// import on demand of a package of the project's libraries brings in is the class
// they denote: it is asked for there, and the JDK's scalar is what is left when no
// class answers.
func TestResolve_AClassNamedLikeAJdkValueTypeIsTheOneAnImportOnDemandNames(t *testing.T) {
	library := &libraryClasses{classes: map[string]parser.Class{
		"lib.m.Period":               marked("lib.m", "Period", "from"),
		"lib.m.Object":               marked("lib.m", "Object", "payload"),
		"javax.money.MonetaryAmount": marked("javax.money", "MonetaryAmount", "amount"),
		"javax.money.Date":           marked("javax.money", "Date", "fromJavax"),
		"lib.n.UUID":                 marked("lib.n", "UUID", "fromN"),
	}}
	source := parser.Class{
		Name: "Holder", Package: "app.web", FilePath: "Holder.java",
		Imports: append(onDemand("lib.m", "javax.money", "java.util"), parser.Import{Path: "java.time.Year"}),
	}
	r := NewTypeResolver([]parser.Class{source})
	r.SetDependencyResolver(library)

	for name, want := range map[string]string{"Period": "from", "Object": "payload", "MonetaryAmount": "amount", "Date": "fromJavax"} {
		if got := fieldsOf(r.ResolveFrom(source, name, nil)); !reflect.DeepEqual(got, []string{want}) {
			t.Errorf("%s has %v, want the class an import on demand names [%s]", name, got, want)
		}
	}
	// A single-type import of the JDK's class wins, and a name no class answers for
	// is the JDK's value.
	if got := r.ResolveFrom(source, "Year", nil); got.Kind != model.KindSingle {
		t.Errorf("Year = %+v, want the JDK's scalar", got)
	}
	if got := r.ResolveFrom(source, "UUID", nil); got.Kind != model.KindSingle {
		t.Errorf("UUID = %+v, want the JDK's scalar (lib.n is not imported)", got)
	}
	for _, name := range library.asked {
		if name == "java.util.Date" {
			t.Errorf("%s, a package of the JDK, was asked for", name)
		}
	}
}

// Every path that turns a name into a class goes through the one lookup, so the
// class an import on demand or a single-type import names is read from the
// dependencies whether or not another path read it first: the enum's constants, the
// class itself, and the constant of a class.
func TestResolve_EveryPathReadsAClassOfTheDependenciesWhateverCameBefore(t *testing.T) {
	mkLibrary := func() *libraryClasses {
		return &libraryClasses{classes: map[string]parser.Class{
			"lib.model.Status": {Name: "Status", Package: "lib.model", IsEnum: true, EnumConstants: []string{"OPEN", "CLOSED"}},
			"lib.consts.Paths": {Name: "Paths", Package: "lib.consts", Fields: []parser.Field{{Name: "TOKEN", Type: "String", IsStatic: true, IsFinal: true, Constant: `"X-Token"`}}},
		}}
	}
	for _, imports := range [][]parser.Import{
		onDemand("lib.model", "lib.consts"),
		{{Path: "lib.model.Status"}, {Path: "lib.consts.Paths"}},
	} {
		source := parser.Class{Name: "Controller", Package: "app.web", FilePath: "Controller.java", Imports: imports}
		for _, warm := range []string{"", "lib.model.Status", "lib.consts.Paths"} {
			name := "imports " + imports[0].Path + ", read first " + warm
			t.Run(name, func(t *testing.T) {
				r := NewTypeResolver([]parser.Class{source})
				r.SetDependencyResolver(mkLibrary())
				if warm != "" {
					r.ResolveForBinding(parser.Class{}, warm, nil)
				}

				if constants, ok := r.EnumConstants(source, "Status"); !ok || !reflect.DeepEqual(constants, []string{"OPEN", "CLOSED"}) {
					t.Errorf("EnumConstants(Status) = %v, %v, want the constants of lib.model.Status", constants, ok)
				}
				if class, ok := r.Class(source, "Status"); !ok || class.Package != "lib.model" || !class.IsEnum {
					// The JSON view does not see an enum.
					if ok {
						t.Errorf("Class(Status) = %+v", class)
					}
				}
				if class, ok := r.Class(source, "Paths"); !ok || class.Package != "lib.consts" {
					t.Errorf("Class(Paths) = %+v, %v, want lib.consts.Paths", class, ok)
				}
				if value, ok := r.ConstantValue(source, "Paths.TOKEN"); !ok || value != "X-Token" {
					t.Errorf("ConstantValue(Paths.TOKEN) = %q, %v, want X-Token", value, ok)
				}
			})
		}
	}
}

// The one cache: a class is asked of the dependencies once, whichever path asks.
func TestResolve_AClassOfTheDependenciesIsAskedOnceWhicheverPathAsks(t *testing.T) {
	library := &libraryClasses{classes: map[string]parser.Class{
		"lib.consts.Paths": {Name: "Paths", Package: "lib.consts", Fields: []parser.Field{{Name: "TOKEN", Type: "String", IsStatic: true, IsFinal: true, Constant: `"X-Token"`}}},
	}}
	source := parser.Class{Name: "Controller", Package: "app.web", FilePath: "Controller.java", Imports: onDemand("lib.consts")}
	r := NewTypeResolver([]parser.Class{source})
	r.SetDependencyResolver(library)

	r.ConstantValue(source, "Paths.TOKEN")
	r.Class(source, "Paths")
	r.ResolveFrom(source, "Paths", nil)
	r.ConstantValue(source, "Paths.TOKEN")
	count := 0
	for _, name := range library.asked {
		if name == "lib.consts.Paths" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("lib.consts.Paths was asked %d times: %v", count, library.asked)
	}
}
