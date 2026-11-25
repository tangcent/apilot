package resolver

import (
	"testing"

	"github.com/tangcent/apilot/api-collector-java/parser"
	model "github.com/tangcent/apilot/api-model"
)

// A class that is being expanded is not expanded again, so that a type that
// refers to itself comes to an end: the inner occurrence is a reference. What
// makes an occurrence the same is the class together with its arguments: in
// `Box<Box<Item>>` the inner Box is another type than the outer one, and is
// expanded, while `Node<T>` inside `Node<Item>` is the same type, and is not.

func recursionClasses() []parser.Class {
	f := typeArgField
	class := func(name string, params []string, fields ...parser.Field) parser.Class {
		return parser.Class{
			Name: name, Package: typeArgLibPackage, FilePath: "com/example/lib/" + name + ".java",
			TypeParameters: params, Fields: fields,
		}
	}
	return []parser.Class{
		typeArgItem(typeArgDtoPackage, "name"),
		class("Box", []string{"T"}, f("data", "T"), f("list", "List<T>")),
		class("Wrapper", []string{"U"}, f("data", "U")),
		class("Outer", []string{"T"}, f("wrapped", "Wrapper<T>"), f("deep", "Wrapper<Wrapper<T>>")),

		// Shapes that really refer to themselves.
		class("Node", []string{"T"}, f("value", "T"), f("next", "Node<T>")),
		// The argument grows with each level, so that no type is ever the same.
		class("Growing", []string{"T"}, f("value", "T"), f("next", "Growing<List<T>>")),
		class("Nesting", []string{"T"}, f("value", "T"), f("next", "Nesting<Nesting<T>>")),
		class("Dimensions", []string{"T"}, f("value", "T"), f("next", "Dimensions<T[]>")),
		class("Bounds", []string{"T"}, f("value", "T"), f("next", "Bounds<? extends T>")),
		class("Tree", nil, f("children", "List<Tree>"), f("parent", "Tree")),
		class("Left", []string{"T"}, f("value", "T"), f("right", "Right<T>")),
		class("Right", []string{"T"}, f("left", "Left<T>")),
		class("Cycle", nil, f("other", "Other")),
		class("Other", nil, f("cycle", "Cycle")),
		class("Alternating", []string{"T"}, f("next", "Relay<List<T>>")),
		class("Relay", []string{"U"}, f("back", "Alternating<U>")),
	}
}

func recursionCaller() parser.Class {
	return parser.Class{
		Name: "Controller", Package: typeArgAppPackage, FilePath: "com/example/app/Controller.java",
		Imports: []parser.Import{{Path: typeArgDtoPackage + ".Item"}, {Path: "java.util.List"}},
	}
}

func isReference(m *model.ObjectModel) bool { return m != nil && m.IsRef() }

func forEachRecursionCase(t *testing.T, run func(t *testing.T, resolve func(typeName string) *model.ObjectModel)) {
	t.Helper()
	for view, resolveIn := range typeArgViews {
		t.Run(view, func(t *testing.T) {
			r := NewTypeResolver(recursionClasses())
			run(t, func(typeName string) *model.ObjectModel { return resolveIn(r, recursionCaller(), typeName) })
		})
	}
}

// The same generic class as the argument of itself is two types, and both are
// expanded, whatever the argument is wrapped in.
func TestResolve_AGenericClassAsTheArgumentOfItselfIsExpanded(t *testing.T) {
	const item = "Item(name)"
	forEachRecursionCase(t, func(t *testing.T, resolve func(string) *model.ObjectModel) {
		for typeName, wants := range map[string]map[string]string{
			"Box<Box<Item>>": {
				"data.data": item, "data.list": "[" + item + "]", "list": "[Box(data,list)]",
			},
			"Box<Box<Box<Item>>>": {"data.data.data": item, "data.data.list": "[" + item + "]"},
			"Box<Box<Item>[]>":    {"data": "[Box(data,list)]"},
			"Box<List<Box<Item>>>": {
				"data": "[Box(data,list)]",
			},
			"Box<Map<String, Box<Item>>>": {"data": "{Box(data,list)}"},
			"Outer<Outer<Item>>":          {"wrapped.data.wrapped.data": item, "wrapped.data.deep.data.data": item},
			"Outer<Item>":                 {"wrapped.data": item, "deep.data.data": item},
			"Outer<Box<Item>>":            {"wrapped.data.data": item, "deep.data.data.data": item},
			"Wrapper<Wrapper<Item>>":      {"data.data": item},
			"Box<Wrapper<Box<Item>>>":     {"data.data.data": item},
		} {
			got := resolve(typeName)
			for path, want := range wants {
				if shape := typeArgShape(typeArgAt(got, path)); shape != want {
					t.Errorf("%s.%s = %s, want %s", typeName, path, shape, want)
				}
			}
		}
		// The element of an array of Boxes is a Box.
		if items := resolve("Box<Box<Item>[]>").Fields["data"].Model.Items; items == nil || typeArgShape(typeArgAt(items, "data")) != item {
			t.Errorf("Box<Box<Item>[]>.data[].data = %v, want %s", items, item)
		}
	})
}

// A type that contains itself ends where it comes back to the same type, and a
// type whose arguments grow without end ends at the first level that is not
// one already being expanded.
func TestResolve_ATypeThatContainsItselfEnds(t *testing.T) {
	const item = "Item(name)"
	forEachRecursionCase(t, func(t *testing.T, resolve func(string) *model.ObjectModel) {
		node := resolve("Node<Item>")
		if got := typeArgShape(typeArgAt(node, "value")); got != item {
			t.Errorf("Node<Item>.value = %s, want %s", got, item)
		}
		if next := typeArgAt(node, "next"); !isReference(next) {
			t.Errorf("Node<Item>.next = %+v, want a reference", next)
		}

		for _, typeName := range []string{"Growing<Item>", "Nesting<Item>", "Dimensions<Item>", "Bounds<Item>"} {
			got := resolve(typeName)
			if value := typeArgShape(typeArgAt(got, "value")); value != item {
				t.Errorf("%s.value = %s, want %s", typeName, value, item)
			}
			if next := typeArgAt(got, "next"); !isReference(next) {
				t.Errorf("%s.next = %+v, want a reference: its argument is not one being expanded", typeName, next)
			}
		}

		tree := resolve("Tree")
		if children := typeArgAt(tree, "children"); children == nil || !children.IsArray() || !isReference(children.Items) {
			t.Errorf("Tree.children = %+v, want an array of references", children)
		}
		if parent := typeArgAt(tree, "parent"); !isReference(parent) {
			t.Errorf("Tree.parent = %+v, want a reference", parent)
		}

		// Two classes that refer to each other, with and without arguments.
		left := resolve("Left<Item>")
		if got := typeArgShape(typeArgAt(left, "value")); got != item {
			t.Errorf("Left<Item>.value = %s, want %s", got, item)
		}
		if right := typeArgAt(left, "right"); right == nil || !right.IsObject() {
			t.Errorf("Left<Item>.right = %+v, want Right<Item> expanded", right)
		} else if back := typeArgAt(right, "left"); !isReference(back) {
			t.Errorf("Left<Item>.right.left = %+v, want a reference", back)
		}
		cycle := resolve("Cycle")
		if other := typeArgAt(cycle, "other"); other == nil || !other.IsObject() || !isReference(typeArgAt(other, "cycle")) {
			t.Errorf("Cycle.other = %+v, want Other with a reference to Cycle", other)
		}
		// The argument changes on the way back: Alternating<Item> to
		// Relay<List<Item>> to Alternating<List<Item>>.
		alternating := resolve("Alternating<Item>")
		if next := typeArgAt(alternating, "next"); next == nil || !next.IsObject() || !isReference(typeArgAt(next, "back")) {
			t.Errorf("Alternating<Item>.next = %+v, want Relay expanded, its back a reference", next)
		}
	})
}
