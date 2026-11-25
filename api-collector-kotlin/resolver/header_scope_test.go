package resolver

import (
	"testing"

	ktparser "github.com/tangcent/apilot/api-collector-kotlin/parser"
)

// The supertype list of a class is outside its body: the classes nested in the
// class, and the ones it inherits, are not in scope there, and `Int` in `class
// Counts : HashMap<String, Int>()` is kotlin.Int although Counts declares a class
// called Int. The classes around the class are in scope in its header.

const headerPackage = "com.example.kchead"

func headerClasses() []ktparser.Class {
	p := scopeProp
	class := func(name string, params []string, superClass string, superArgs []string, fields ...ktparser.Property) ktparser.Class {
		return ktparser.Class{
			Name: name, Package: headerPackage, FilePath: "com/example/kchead/" + name + ".kt",
			TypeParameters: params, SuperClass: superClass, SuperClassTypeArgs: superArgs, Fields: fields,
		}
	}
	nested := func(outer, name string) ktparser.Class {
		c := class(name, nil, "", nil, p("wrong", "kotlin.String"))
		c.Outer, c.FilePath = outer, "com/example/kchead/"+outer+".kt"
		return c
	}
	return []ktparser.Class{
		class("Base", []string{"B"}, "", nil, p("value", "B"), p("list", "List<B>")),
		// Each has a class nested in it that is called like a type the header writes.
		class("Counts", nil, "HashMap", []string{"String", "Int"}),
		nested("Counts", "Int"),
		nested("Counts", "HashMap"),
		class("Sub", nil, "Base", []string{"Int"}),
		nested("Sub", "Int"),
		class("Wide", nil, "Base", []string{"List<Long>"}),
		nested("Wide", "Long"),
		class("Holder", nil, "", nil, p("count", "Int"), p("counts", "List<Int>")),
		nested("Holder", "Int"),
		// A class around the class is in scope in the header.
		class("Around", nil, "", nil),
		func() ktparser.Class {
			c := class("Item", nil, "", nil, p("right", "kotlin.String"))
			c.Outer, c.FilePath = "Around", "com/example/kchead/Around.kt"
			return c
		}(),
		func() ktparser.Class {
			c := class("Leaf", nil, "Base", []string{"Item"})
			c.Outer, c.FilePath = "Around", "com/example/kchead/Around.kt"
			return c
		}(),
	}
}

func TestResolve_TheHeaderOfAKotlinClassDoesNotSeeItsOwnNestedClasses(t *testing.T) {
	caller := ktparser.Class{Name: "Caller", Package: "com.example.kccaller", FilePath: "com/example/kccaller/Caller.kt"}
	for view, resolve := range scopeViews {
		t.Run(view, func(t *testing.T) {
			r := New(headerClasses())
			for _, tc := range []struct{ typeName, path, want string }{
				{"Counts", "", "{int}"},
				{"Sub", "value", "int"},
				{"Sub", "list", "[int]"},
				{"Wide", "value", "[long]"},
				{"Around.Leaf", "value", "Item(right)"},
				// The body of the class sees its nested classes.
				{"Holder", "count", "Int(wrong)"},
				{"Holder", "counts", "[Int(wrong)]"},
			} {
				got := resolve(r, &caller, headerPackage+"."+tc.typeName)
				if tc.path != "" {
					got = scopeAt(got, tc.path)
				}
				if shape := scopeModelShape(got); shape != tc.want {
					t.Errorf("%s.%s = %s, want %s", tc.typeName, tc.path, shape, tc.want)
				}
			}
			if unresolved := r.Unresolved(); len(unresolved) != 0 {
				t.Errorf("unresolved = %v, want none", unresolved)
			}
		})
	}
}
