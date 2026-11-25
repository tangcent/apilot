package resolver

import (
	"testing"

	"github.com/tangcent/apilot/api-collector-java/parser"
	model "github.com/tangcent/apilot/api-model"
)

// A type variable that nothing binds stays open wherever it goes, whatever class
// of the project carries its name. The caller knows the name is its own variable
// when it passes it on as the argument of another generic class or of its
// superclass, and says so with the binding, so the class that receives it does
// not look the name up again among the classes: `Outer<Status>` that holds a
// `Box<Status>` holds a box of its own open variable, also when another package
// declares an enum, or a class, called Status.

func openVariableClasses(unrelated parser.Class) []parser.Class {
	f := typeArgField
	lib := func(name string, params []string, superClass string, superArgs []string, fields ...parser.Field) parser.Class {
		return parser.Class{
			Name: name, Package: typeArgLibPackage, FilePath: "com/example/lib/" + name + ".java",
			TypeParameters: params, SuperClass: superClass, SuperClassTypeArgs: superArgs, Fields: fields,
		}
	}
	outer := lib("Wrapping", []string{"Status"}, "", nil, f("shared", "Status"))
	inner := parser.Class{
		Name: "Inner", Outer: "Wrapping", Inner: true, Package: typeArgLibPackage, FilePath: outer.FilePath,
		Fields: []parser.Field{f("variable", "Status")},
	}
	fixed := parser.Class{
		Name: "Fixed", Outer: "Wrapping", Package: typeArgLibPackage, FilePath: outer.FilePath,
		Fields: []parser.Field{f("named", "Status")},
	}
	return []parser.Class{
		typeArgItem(typeArgDtoPackage, "name"),
		unrelated,
		lib("Box", []string{"T"}, "", nil, f("data", "T"), f("list", "List<T>"), f("array", "T[]")),
		lib("Outer", []string{"Status"}, "", nil,
			f("own", "Status"), f("box", "Box<Status>"), f("nested", "Box<List<Status>>"), f("boxes", "Box<Status[]>")),
		lib("Base", []string{"T"}, "", nil, f("value", "T"), f("list", "List<T>")),
		lib("Sub", []string{"Status"}, "Base", []string{"Status"}),
		lib("Forward", []string{"Status"}, "Sub", []string{"Status"}),
		outer, inner, fixed,
	}
}

// The classes of another package that the names of the variables above could be
// taken for: an enum and a class with a field.
func unrelatedStatuses() map[string]parser.Class {
	other := func() parser.Class {
		return parser.Class{Name: "Status", Package: "com.example.other", FilePath: "com/example/other/Status.java"}
	}
	enum := other()
	enum.IsEnum, enum.EnumConstants = true, []string{"OPEN", "SHUT"}
	class := other()
	class.Fields = []parser.Field{typeArgField("unrelated", "String")}
	return map[string]parser.Class{"an enum": enum, "a class": class}
}

func TestResolve_ATypeVariableOfTheCallerStaysOpenBesideAClassOfTheSameName(t *testing.T) {
	isOpen := func(m *model.ObjectModel) bool { return m != nil && m.IsSingle() && m.TypeName == "Status" }
	generic := func(m *model.ObjectModel, path string) bool {
		parent, name := m, path
		if i := lastDot(path); i >= 0 {
			parent, name = typeArgAt(m, path[:i]), path[i+1:]
		}
		field := parent.Fields[name]
		return field != nil && field.Generic
	}
	for kind, unrelated := range unrelatedStatuses() {
		for view, resolveIn := range typeArgViews {
			t.Run(kind+"/"+view, func(t *testing.T) {
				r := NewTypeResolver(openVariableClasses(unrelated))
				resolve := func(typeName string) *model.ObjectModel { return resolveIn(r, recursionCaller(), typeName) }

				// A field type argument: the variable of Outer, passed to Box.
				outer := resolve("com.example.lib.Outer")
				for path, want := range map[string]bool{
					"own":         true,
					"box.data":    true,
					"nested.list": false, // an array, and no variable itself
					"nested.data": false,
					"boxes.data":  false,
				} {
					if got := generic(outer, path); got != want {
						t.Errorf("Outer.%s generic = %v, want %v", path, got, want)
					}
				}
				for _, path := range []string{"own", "box.data", "box.list.[]", "box.array.[]", "nested.data.[]", "nested.list.[].[]", "boxes.data.[]"} {
					got := typeArgAtItems(outer, path)
					if !isOpen(got) {
						t.Errorf("Outer.%s = %s, want the open variable Status", path, typeArgShape(got))
					}
				}

				// A superclass argument: Sub passes its variable to Base, and Forward
				// passes its own on through Sub.
				for _, name := range []string{"Sub", "Forward"} {
					sub := resolve("com.example.lib." + name)
					if !generic(sub, "value") {
						t.Errorf("%s.value generic = false, want true", name)
					}
					for _, path := range []string{"value", "list.[]"} {
						if got := typeArgAtItems(sub, path); !isOpen(got) {
							t.Errorf("%s.%s = %s, want the open variable Status", name, path, typeArgShape(got))
						}
					}
				}
			})
		}
	}
}

// A variable that an argument binds is what it is bound to, and a variable of an
// inner class that belongs to the class around it is open there. A static member
// class has no variable of the class around it, so the name is a class.
func TestResolve_ATypeVariableIsDecidedByTheScopeOfTheClassThatWritesIt(t *testing.T) {
	for kind, unrelated := range unrelatedStatuses() {
		for view, resolveIn := range typeArgViews {
			t.Run(kind+"/"+view, func(t *testing.T) {
				r := NewTypeResolver(openVariableClasses(unrelated))
				caller := recursionCaller()

				// Bound to a class, the variable is that class and the field is not generic.
				for _, typeName := range []string{"com.example.lib.Outer<Item>", "com.example.lib.Wrapping<Item>"} {
					bound := resolveIn(r, caller, typeName)
					for _, path := range []string{"own", "shared"} {
						field, ok := bound.Fields[path]
						if !ok {
							continue
						}
						if field.Generic || typeArgShape(field.Model) != "Item(name)" {
							t.Errorf("%s.%s = %s (generic %v), want Item(name) that is not generic", typeName, path, typeArgShape(field.Model), field.Generic)
						}
					}
				}
				if box := resolveIn(r, caller, "com.example.lib.Outer<Item>"); box.Fields["box"].Model.Fields["data"].Generic ||
					typeArgShape(typeArgAt(box, "box.data")) != "Item(name)" {
					t.Errorf("Outer<Item>.box.data = %s, want Item(name) that is not generic", typeArgShape(typeArgAt(box, "box.data")))
				}

				// An inner class reads the variable of the class around it.
				inner := resolveIn(r, parser.Class{}, "com.example.lib.Wrapping.Inner")
				if field := inner.Fields["variable"]; field == nil || !field.Generic || !field.Model.IsSingle() || field.Model.TypeName != "Status" {
					t.Errorf("Wrapping.Inner.variable = %+v, want the open variable Status", field)
				}
				// A static member class does not: its Status is the class of that name.
				fixed := resolveIn(r, parser.Class{}, "com.example.lib.Wrapping.Fixed")
				if field := fixed.Fields["named"]; field == nil || field.Generic {
					t.Errorf("Wrapping.Fixed.named = %+v, want a field that is not generic", field)
				}
				if kind == "a class" {
					if got := typeArgShape(typeArgAt(fixed, "named")); got != "Status(unrelated)" {
						t.Errorf("Wrapping.Fixed.named = %s, want the class Status(unrelated)", got)
					}
				}
			})
		}
	}
}

// The model of an open variable is told from the model of a class or an enum that
// carries the name by the model itself: both are a single value called Status.
func TestIsOpenVariable_TellsAVariableFromAClassOfTheSameName(t *testing.T) {
	for kind, unrelated := range unrelatedStatuses() {
		t.Run(kind, func(t *testing.T) {
			r := NewTypeResolver(openVariableClasses(unrelated))
			caller := recursionCaller()

			outer := r.ResolveFrom(caller, "com.example.lib.Outer", nil)
			for _, path := range []string{"own", "box.data", "box.list.[]"} {
				if got := typeArgAtItems(outer, path); !r.IsOpenVariable(got) {
					t.Errorf("Outer.%s = %s, want the model of an open variable", path, typeArgShape(got))
				}
			}
			// Bound to the other package's Status, the value is that class or enum.
			bound := r.ResolveFrom(caller, "com.example.lib.Outer<com.example.other.Status>", nil)
			if got := typeArgAt(bound, "own"); r.IsOpenVariable(got) {
				t.Errorf("Outer<Status>.own = %s, want what Status is bound to", typeArgShape(got))
			}
			if r.IsOpenVariable(nil) || r.IsOpenVariable(model.SingleModel("Status")) {
				t.Error("a model that the resolver did not answer with is no open variable")
			}
		})
	}
}

// typeArgAtItems follows a dotted path of field names, and a segment `[]` steps
// into the element of an array.
func typeArgAtItems(m *model.ObjectModel, path string) *model.ObjectModel {
	for _, segment := range splitPath(path) {
		if m == nil {
			return nil
		}
		if segment == "[]" {
			m = m.Items
			continue
		}
		if m.Fields[segment] == nil {
			return nil
		}
		m = m.Fields[segment].Model
	}
	return m
}

func splitPath(path string) []string {
	var segments []string
	start := 0
	for i := 0; i <= len(path); i++ {
		if i == len(path) || path[i] == '.' {
			segments = append(segments, path[start:i])
			start = i + 1
		}
	}
	return segments
}
