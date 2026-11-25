package resolver

import (
	"testing"

	ktparser "github.com/tangcent/apilot/api-collector-kotlin/parser"
)

// An enum of the project is a string holding one of its constants, wherever a
// type is named like it. A type variable that nothing binds is not that enum,
// whatever it is called: `Tagged<Status>` raw holds a value of its parameter, and
// only an argument that names the enum makes the value one of its constants.

func TestResolve_AnOpenTypeVariableIsNotTheEnumThatCarriesItsName(t *testing.T) {
	p := scopeProp
	enum := ktparser.Class{
		Name: "Status", Package: paramOtherPkg, FilePath: "com/example/kcvarother/Status.kt",
		IsEnum: true, EnumValues: []string{"OPEN", "SHUT"},
	}
	tagged := paramClass("Tagged", []string{paramStatusName},
		p("tag", paramStatusName), p("tags", "List<"+paramStatusName+">"), p("byKey", "Map<kotlin.String, "+paramStatusName+">"))
	holder := paramClass("Holder", []string{paramStatusName}, p("box", "Tagged<"+paramStatusName+">"))
	sub := paramClass("SubTagged", []string{paramStatusName})
	sub.SuperClass, sub.SuperClassTypeArgs = "Base", []string{paramStatusName}
	base := paramClass("Base", []string{"B"}, p("value", "B"), p("list", "List<B>"))
	caller := paramCaller()

	for view, resolve := range scopeViews {
		t.Run(view, func(t *testing.T) {
			r := New([]ktparser.Class{enum, tagged, holder, sub, base})
			for _, tc := range []struct{ typeName, path, want string }{
				{"Tagged", "tag", paramStatusName},
				{"Tagged", "tags", "[" + paramStatusName + "]"},
				{"Tagged", "byKey", "{" + paramStatusName + "}"},
				{"Holder", "box.tag", paramStatusName},
				{"Holder", "box.tags", "[" + paramStatusName + "]"},
				{"SubTagged", "value", paramStatusName},
				{"SubTagged", "list", "[" + paramStatusName + "]"},
				// Bound to the enum, the value is one of its constants.
				{"Tagged<" + paramOtherPkg + "." + paramStatusName + ">", "tag", "string"},
				{"Tagged<" + paramOtherPkg + "." + paramStatusName + ">", "tags", "[string]"},
			} {
				got := scopeModelShape(scopeAt(resolve(r, &caller, paramPackage+"."+tc.typeName), tc.path))
				if got != tc.want {
					t.Errorf("%s.%s = %s, want %s", tc.typeName, tc.path, got, tc.want)
				}
			}
			raw := resolve(r, &caller, paramPackage+".Tagged")
			if field := raw.Fields["tag"]; field == nil || !field.Generic || len(field.Options) != 0 {
				t.Errorf("Tagged.tag = %+v, want a generic field without the options of the enum", field)
			}
			// The JSON view lists the constants of an enum an argument names; the
			// binding view does not read them through a type variable.
			bound := resolve(r, &caller, paramPackage+".Tagged<"+paramOtherPkg+"."+paramStatusName+">")
			if field := bound.Fields["tag"]; field == nil || field.Generic || (view == "json" && len(field.Options) != 2) {
				t.Errorf("Tagged<Status>.tag = %+v, want a field that is not generic, with the two constants of the enum in the JSON view", field)
			}
		})
	}
}
