package resolver

import (
	"testing"

	ktparser "github.com/tangcent/apilot/api-collector-kotlin/parser"
)

// A type parameter is the first thing a name denotes where it is in scope, before
// the classes nested in the class, the imports, the package and the standard
// library: `class Box<String>(val value: String)` holds a value of its parameter,
// whatever `Box<kotlin.Int>` is, and `kotlin.String` is a different thing. A
// nested class that is not inner has no access to the parameters of the class
// around it.

const (
	paramPackage    = "com.example.kcvar"
	paramOtherPkg   = "com.example.kcvarother"
	paramCallerPkg  = "com.example.kcvarapp"
	paramStatusName = "Status"
)

func paramClass(name string, params []string, fields ...ktparser.Property) ktparser.Class {
	return ktparser.Class{
		Name: name, Package: paramPackage, FilePath: "com/example/kcvar/" + name + ".kt",
		TypeParameters: params, Fields: fields,
	}
}

func paramClasses() []ktparser.Class {
	p := scopeProp
	outer := paramClass("Outer", []string{"String"}, p("own", "String"))
	inner := paramClass("Inner", nil, p("x", "String"), p("xs", "List<String>"))
	inner.Outer, inner.IsInner = "Outer", true
	nested := paramClass("Nested", nil, p("y", "String"))
	nested.Outer = "Outer"
	sub := paramClass("Sub", []string{"String"})
	sub.SuperClass, sub.SuperClassTypeArgs = "Base", []string{"String"}
	subStatus := paramClass("SubStatus", []string{paramStatusName})
	subStatus.SuperClass, subStatus.SuperClassTypeArgs = "Base", []string{paramStatusName}
	return []ktparser.Class{
		paramClass("Box", []string{"String"}, p("value", "String"), p("list", "List<String>"), p("byKey", "Map<kotlin.String, String>")),
		paramClass("Pair2", []string{"Int", "Long"}, p("a", "Int"), p("b", "Long")),
		paramClass("Tagged", []string{paramStatusName}, p("tag", paramStatusName), p("tags", "List<"+paramStatusName+">")),
		paramClass("Base", []string{"B"}, p("value", "B"), p("list", "List<B>")),
		// The classes of the project that carry the names of the parameters: one of
		// the package of the generic classes and one of another package.
		paramClass(paramStatusName, nil, p("same", "kotlin.Int")),
		{
			Name: paramStatusName, Package: paramOtherPkg, FilePath: "com/example/kcvarother/Status.kt",
			Fields: []ktparser.Property{p("other", "kotlin.Int")},
		},
		// Classes of another package called like the types of the standard library.
		{Name: "String", Package: paramOtherPkg, FilePath: "com/example/kcvarother/String.kt", Fields: []ktparser.Property{p("shadow", "kotlin.Int")}},
		{Name: "Int", Package: paramOtherPkg, FilePath: "com/example/kcvarother/Int.kt", Fields: []ktparser.Property{p("shadow", "kotlin.Int")}},
		outer, inner, nested, sub, subStatus,
	}
}

func paramCaller() ktparser.Class {
	return ktparser.Class{Name: "Caller", Package: paramCallerPkg, FilePath: "com/example/kcvarapp/Caller.kt"}
}

func TestResolve_ATypeParameterIsWhatItsNameDenotesInTheClass(t *testing.T) {
	caller := paramCaller()
	for view, resolve := range scopeViews {
		t.Run(view, func(t *testing.T) {
			r := New(paramClasses())
			shape := func(typeName, path string) string {
				return scopeModelShape(scopeAt(resolve(r, &caller, paramPackage+"."+typeName), path))
			}
			for _, tc := range []struct{ typeName, path, want string }{
				// Bound to a type of the standard library.
				{"Box<Int>", "value", "int"},
				{"Box<Int>", "list", "[int]"},
				{"Box<Int>", "byKey", "{int}"},
				{"Box<kotlin.Int>", "value", "int"},
				{"Box<Boolean>", "value", "boolean"},
				{"Pair2<String, Boolean>", "a", "string"},
				{"Pair2<String, Boolean>", "b", "boolean"},
				{paramStatusName + "<Long>", "same", "int"}, // the class of the package is a class
				{"Tagged<Int>", "tag", "int"},
				{"Tagged<Int>", "tags", "[int]"},
				// Bound to a class that carries the name of the parameter.
				{"Tagged<" + paramOtherPkg + "." + paramStatusName + ">", "tag", paramStatusName + "(other)"},
				// Raw: the parameter stays open, and is not the class of the name.
				{"Box", "value", "String"},
				{"Box", "list", "[String]"},
				{"Pair2", "a", "Int"},
				{"Tagged", "tag", paramStatusName},
				{"Tagged", "tags", "[" + paramStatusName + "]"},
			} {
				if got := shape(tc.typeName, tc.path); got != tc.want {
					t.Errorf("%s.%s = %s, want %s", tc.typeName, tc.path, got, tc.want)
				}
			}
			for _, tc := range []struct {
				typeName, path string
				generic        bool
			}{
				{"Box", "value", true},
				{"Tagged", "tag", true},
				{"Box<Int>", "value", false},
				{"Tagged<Int>", "tag", false},
			} {
				field := resolve(r, &caller, paramPackage+"."+tc.typeName).Fields[tc.path]
				if field == nil || field.Generic != tc.generic {
					t.Errorf("%s.%s generic = %+v, want %v", tc.typeName, tc.path, field, tc.generic)
				}
			}
			if unresolved := r.Unresolved(); len(unresolved) != 0 {
				t.Errorf("unresolved = %v, want none", unresolved)
			}
		})
	}
}

// The argument of a supertype list is written in the class's own scope, where its
// type parameter is in scope as well.
func TestResolve_ATypeParameterAsASuperclassArgumentIsBoundByTheArgument(t *testing.T) {
	caller := paramCaller()
	for view, resolve := range scopeViews {
		t.Run(view, func(t *testing.T) {
			r := New(paramClasses())
			for _, tc := range []struct{ typeName, path, want string }{
				{"Sub<Int>", "value", "int"},
				{"Sub<Int>", "list", "[int]"},
				{"Sub<Boolean>", "value", "boolean"},
				{"SubStatus<Int>", "value", "int"},
				{"Sub", "value", "String"},
				{"SubStatus", "value", paramStatusName},
				{"SubStatus", "list", "[" + paramStatusName + "]"},
			} {
				got := scopeModelShape(scopeAt(resolve(r, &caller, paramPackage+"."+tc.typeName), tc.path))
				if got != tc.want {
					t.Errorf("%s.%s = %s, want %s", tc.typeName, tc.path, got, tc.want)
				}
			}
			for _, name := range []string{"Sub", "SubStatus"} {
				if field := resolve(r, &caller, paramPackage+"."+name).Fields["value"]; field == nil || !field.Generic {
					t.Errorf("%s.value = %+v, want a generic field", name, field)
				}
			}
		})
	}
}

// An inner class sees the type parameters of the class around it, and a nested
// class that is not inner does not: its String is the standard one.
func TestResolve_AnInnerClassSeesTheTypeParametersAroundIt(t *testing.T) {
	caller := paramCaller()
	for view, resolve := range scopeViews {
		t.Run(view, func(t *testing.T) {
			r := New(paramClasses())
			inner := resolve(r, &caller, paramPackage+".Outer.Inner")
			for path, want := range map[string]string{"x": "String", "xs": "[String]"} {
				if got := scopeModelShape(scopeAt(inner, path)); got != want {
					t.Errorf("Outer.Inner.%s = %s, want the type parameter %s", path, got, want)
				}
			}
			if field := inner.Fields["x"]; field == nil || !field.Generic {
				t.Errorf("Outer.Inner.x = %+v, want a generic field", field)
			}

			nested := resolve(r, &caller, paramPackage+".Outer.Nested")
			if got := scopeModelShape(scopeAt(nested, "y")); got != "string" {
				t.Errorf("Outer.Nested.y = %s, want string", got)
			}
			if field := nested.Fields["y"]; field == nil || field.Generic {
				t.Errorf("Outer.Nested.y = %+v, want a field that is not generic", field)
			}
		})
	}
}
