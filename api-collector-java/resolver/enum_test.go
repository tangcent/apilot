package resolver

import (
	"reflect"
	"testing"

	collector "github.com/tangcent/apilot/api-collector"
	"github.com/tangcent/apilot/api-collector-java/parser"
	model "github.com/tangcent/apilot/api-model"
)

const orderWithEnum = `package com.example;

import java.util.List;

public class Order {
    private Status status;
    private List<Status> history;

    public enum Status {
        OPEN("o"),
        CLOSED("c") {
            @Override public String toString() { return "closed"; }
        };

        private final String code;
        Status(String code) { this.code = code; }
    }
}
`

func parseOrderWithEnum(t *testing.T) []parser.Class {
	t.Helper()
	jp, err := parser.NewParser(parser.ParserOptions{})
	if err != nil {
		t.Fatal(err)
	}
	defer jp.Close()
	classes, err := jp.ParseSource([]byte(orderWithEnum))
	if err != nil {
		t.Fatal(err)
	}
	return classes
}

func resolveJSON(classes []parser.Class, typeName string) (*model.ObjectModel, map[string]int) {
	r := NewTypeResolver(classes)
	r.SetUnresolved(collector.NewUnresolvedSet())
	return r.Resolve(typeName, nil), r.Unresolved()
}

func TestParse_EnumConstants(t *testing.T) {
	classes := parseOrderWithEnum(t)
	var enum *parser.Class
	for i := range classes {
		if classes[i].IsEnum {
			enum = &classes[i]
		}
	}
	if enum == nil || enum.Name != "Status" || !reflect.DeepEqual(enum.EnumConstants, []string{"OPEN", "CLOSED"}) {
		t.Fatalf("enum = %+v", enum)
	}
}

// The JSON view reads an enum that Jackson writes as the name of a constant: a
// field typed with it is a string whose options are the constants, and so is a
// collection of it, which lists them on the field.
func TestResolve_EnumFieldIsAStringWithItsConstants(t *testing.T) {
	order, unresolved := resolveJSON(parseOrderWithEnum(t), "com.example.Order")
	want := []any{"OPEN", "CLOSED"}

	status := order.Fields["status"]
	if status.Model.Kind != model.KindSingle || status.Model.TypeName != model.JsonTypeString {
		t.Errorf("status model = %+v", status.Model)
	}
	if got := optionValues(status.Options); !reflect.DeepEqual(got, want) {
		t.Errorf("status options = %v, want %v", got, want)
	}

	history := order.Fields["history"]
	if history.Model.Kind != model.KindArray || history.Model.Items.TypeName != model.JsonTypeString {
		t.Errorf("history model = %+v", history.Model)
	}
	if got := optionValues(history.Options); !reflect.DeepEqual(got, want) {
		t.Errorf("history options = %v, want %v", got, want)
	}
	if len(unresolved) != 0 {
		t.Errorf("unresolved = %v, want none", unresolved)
	}
}

// An enum whose JSON form is not the name of its constant is exported by its
// type name, as a type whose declaration was not read: what the field takes
// cannot be told from the constants.
func TestResolve_EnumWithAnotherJSONFormKeepsItsTypeName(t *testing.T) {
	tests := []struct {
		name       string
		annotation string // before the enum declaration
		body       string // in the enum body
		constant   string // inside the class body of LOW; empty is a method with no annotation
		asString   bool
	}{
		{name: "no annotation", asString: true},
		{name: "@JsonValue on a method", body: "@JsonValue public String code() { return null; }"},
		{name: "@JsonValue on a field", body: "@JsonValue private String code;"},
		{name: "@JsonValue by its qualified name", body: "@com.fasterxml.jackson.annotation.JsonValue public String code() { return null; }"},
		{name: "@JsonValue switched off", body: "@JsonValue(false) public String code() { return null; }", asString: true},
		{name: "@JsonValue only inside the body of a constant", constant: "@JsonValue String other() { return null; }"},
		{name: "@JsonFormat shape object", annotation: "@JsonFormat(shape = JsonFormat.Shape.OBJECT)"},
		{name: "@JsonFormat shape object, imported", annotation: "@JsonFormat(shape = Shape.OBJECT)"},
		{name: "@JsonFormat shape number", annotation: "@JsonFormat(shape = JsonFormat.Shape.NUMBER)"},
		{name: "@JsonFormat shape number int", annotation: "@JsonFormat(shape = JsonFormat.Shape.NUMBER_INT)"},
		{name: "@JsonFormat shape number float", annotation: "@JsonFormat(shape = JsonFormat.Shape.NUMBER_FLOAT)"},
		{name: "@JsonFormat shape array", annotation: "@JsonFormat(shape = JsonFormat.Shape.ARRAY)"},
		{name: "@JsonFormat shape string", annotation: "@JsonFormat(shape = JsonFormat.Shape.STRING)", asString: true},
		{name: "@JsonFormat without a shape", annotation: `@JsonFormat(pattern = "x")`, asString: true},
		{name: "@JsonSerialize with a serializer", annotation: "@JsonSerialize(using = Writer.class)"},
		{name: "@JsonSerialize with a converter", annotation: "@JsonSerialize(converter = Convert.class)"},
		{name: "@JsonSerialize without one", annotation: "@JsonSerialize(include = Include.ALWAYS)", asString: true},
		{name: "@JsonSerialize with the serializer that stands for none", annotation: "@JsonSerialize(using = JsonSerializer.None.class)", asString: true},
		{name: "@JsonSerialize with the qualified serializer that stands for none", annotation: "@JsonSerialize(using = com.fasterxml.jackson.databind.JsonSerializer.None.class)", asString: true},
		{name: "@JsonSerialize with a serializer whose name ends in None", annotation: "@JsonSerialize(using = FallbackNone.class)"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			constant := tt.constant
			if constant == "" {
				constant = "String other() { return null; }"
			}
			source := `package a;

import com.fasterxml.jackson.annotation.JsonFormat;
import com.fasterxml.jackson.annotation.JsonValue;

` + tt.annotation + `
enum Level {
    LOW {
        ` + constant + `
    },
    HIGH;

    ` + tt.body + `
}

class Holder {
    Level level;
}
`
			jp, err := parser.NewParser(parser.ParserOptions{})
			if err != nil {
				t.Fatal(err)
			}
			defer jp.Close()
			classes, err := jp.ParseSource([]byte(source))
			if err != nil {
				t.Fatal(err)
			}
			holder, unresolved := resolveJSON(classes, "a.Holder")
			level := holder.Fields["level"]
			if tt.asString {
				if level.Model.TypeName != model.JsonTypeString || !reflect.DeepEqual(optionValues(level.Options), []any{"LOW", "HIGH"}) || len(unresolved) != 0 {
					t.Errorf("level = %+v, options %v, unresolved %v; want a string of LOW and HIGH", level.Model, level.Options, unresolved)
				}
				return
			}
			if level.Model.TypeName != "Level" || len(level.Options) != 0 || unresolved["Level"] != 1 {
				t.Errorf("level = %+v, options %v, unresolved %v; want the type name, reported unresolved", level.Model, level.Options, unresolved)
			}
		})
	}
}

// The binding view reads every enum, whatever its JSON form, and a view ends
// with the call that entered it.
func TestResolveForBinding_ReadsEnumsOfAnotherJSONForm(t *testing.T) {
	r := NewTypeResolver([]parser.Class{
		{Name: "Code", Package: "a", IsEnum: true, EnumConstants: []string{"X", "Y"}, EnumJsonValue: true},
		{Name: "Holder", Package: "a", Fields: []parser.Field{{Name: "code", Type: "Code"}}},
	})
	bound := r.ResolveForBinding(parser.Class{}, "a.Holder", nil).Fields["code"]
	if bound.Model.TypeName != model.JsonTypeString || !reflect.DeepEqual(optionValues(bound.Options), []any{"X", "Y"}) {
		t.Errorf("bound = %+v, options %v, want a string of X and Y", bound.Model, bound.Options)
	}
	if got := r.Resolve("a.Holder", nil).Fields["code"]; got.Model.TypeName != "Code" || len(got.Options) != 0 {
		t.Errorf("the JSON view must be unaffected after a binding resolution: %+v, options %v", got.Model, got.Options)
	}
}

// A name binds to an enum through the scope: nothing else gives one for a simple
// name. A reference no scope declares falls back to a class that carries the name,
// as it always has, and never to an enum, which another package, an import on
// demand or the JDK may name as well.
func TestResolve_AnEnumAnswersThroughItsScopeOnly(t *testing.T) {
	classes := []parser.Class{
		{Name: "Level", Package: "a", IsEnum: true, EnumConstants: []string{"LOW"}},
		{Name: "Level", Package: "b", Fields: []parser.Field{{Name: "rank", Type: "int"}}},
		{Name: "Mode", Package: "a", IsEnum: true, EnumConstants: []string{"ON", "OFF"}},
		{Name: "Holder", Package: "c", Fields: []parser.Field{{Name: "level", Type: "Level"}, {Name: "mode", Type: "Mode"}}},
		{Name: "Neighbour", Package: "a", Fields: []parser.Field{{Name: "mode", Type: "Mode"}}},
	}
	holder, unresolved := resolveJSON(classes, "c.Holder")

	level := holder.Fields["level"]
	if _, ok := level.Model.Fields["rank"]; !ok || len(level.Options) != 0 {
		t.Errorf("level = %+v, options %v, want the class b.Level", level.Model, level.Options)
	}
	mode := holder.Fields["mode"]
	if mode.Model.TypeName != "Mode" || len(mode.Options) != 0 || unresolved["Mode"] != 1 {
		t.Errorf("mode = %+v, options %v, unresolved %v; want the name as written, reported unresolved", mode.Model, mode.Options, unresolved)
	}

	neighbour, _ := resolveJSON(classes, "a.Neighbour")
	if mode := neighbour.Fields["mode"]; mode.Model.TypeName != model.JsonTypeString || !reflect.DeepEqual(optionValues(mode.Options), []any{"ON", "OFF"}) {
		t.Errorf("mode of the package of the enum = %+v, options %v, want a string of ON and OFF", mode.Model, mode.Options)
	}
}

// An enum that implements an interface is one value of it, not the shape every
// value has: the JSON view looks for a class to stand for the interface, and
// never took an enum for one.
func TestResolve_AnEnumIsNotTheImplementationOfAnInterface(t *testing.T) {
	enum := parser.Class{Name: "ErrorCode", Package: "a", IsEnum: true, EnumConstants: []string{"E1"}, Interfaces: []string{"Code"}}
	iface := parser.Class{Name: "Code", Package: "a", IsInterface: true}
	holder := parser.Class{Name: "Holder", Package: "a", Fields: []parser.Field{{Name: "code", Type: "Code"}}}

	got, _ := resolveJSON([]parser.Class{iface, enum, holder}, "a.Holder")
	if code := got.Fields["code"].Model; !code.IsRef() || code.TypeName != "Code" {
		t.Errorf("code = %+v, want a reference to the interface", code)
	}

	plain := parser.Class{Name: "PlainCode", Package: "a", Interfaces: []string{"Code"}, Fields: []parser.Field{{Name: "number", Type: "int"}}}
	got, _ = resolveJSON([]parser.Class{iface, enum, plain, holder}, "a.Holder")
	if code := got.Fields["code"].Model; code.TypeName != "PlainCode" {
		t.Errorf("code = %+v, want the class that implements the interface", code)
	}
}

// An enum of a dependency follows the rule of the project's own, and is not
// registered by its simple name: a reference that no scope declares must not
// turn into an enum because some other file imported one of that name.
func TestResolve_ADependencyEnumDoesNotAnswerForASimpleName(t *testing.T) {
	library := &libraryClasses{classes: map[string]parser.Class{
		"com.dep.Status": {Name: "Status", Package: "com.dep", IsEnum: true, EnumConstants: []string{"ON", "OFF"}},
		"com.dep.Code":   {Name: "Code", Package: "com.dep", IsEnum: true, EnumConstants: []string{"X"}, EnumJsonValue: true},
	}}
	r := NewTypeResolver([]parser.Class{
		{
			Name: "Holder", Package: "app", FilePath: "Holder.java",
			Imports: []parser.Import{{Path: "com.dep.Status"}, {Path: "com.dep.Code"}},
			Fields:  []parser.Field{{Name: "status", Type: "Status"}, {Name: "code", Type: "Code"}},
		},
		// Names a Status it does not import.
		{Name: "Other", Package: "other", FilePath: "Other.java", Fields: []parser.Field{{Name: "status", Type: "Status"}}},
	})
	r.SetUnresolved(collector.NewUnresolvedSet())
	r.SetDependencyResolver(library)

	holder := r.Resolve("app.Holder", nil)
	status := holder.Fields["status"]
	if status.Model.TypeName != model.JsonTypeString || !reflect.DeepEqual(optionValues(status.Options), []any{"ON", "OFF"}) {
		t.Errorf("status = %+v, options %v, want a string of ON and OFF", status.Model, status.Options)
	}
	if code := holder.Fields["code"]; code.Model.TypeName != "Code" || len(code.Options) != 0 {
		t.Errorf("code = %+v, options %v, want the type name of an enum that declares @JsonValue", code.Model, code.Options)
	}

	other := r.Resolve("other.Other", nil)
	if got := other.Fields["status"].Model.TypeName; got != "Status" {
		t.Errorf("a Status that is not in scope = %q, want it left unresolved", got)
	}
	if got := r.Unresolved(); !reflect.DeepEqual(got, map[string]int{"Code": 1, "Status": 1}) {
		t.Errorf("unresolved = %v", got)
	}
}

// Two enums nested in different classes of one package are two enums: each
// class reads the one it nests, and a name written through the class reads that
// one.
func TestResolve_EnumsNestedInDifferentClassesAreTold(t *testing.T) {
	classes := []parser.Class{
		{Name: "Order", Package: "p", FilePath: "Order.java"},
		{Name: "Type", Package: "p", Outer: "Order", FilePath: "Order.java", IsEnum: true, EnumConstants: []string{"RETAIL", "WHOLESALE"}},
		{Name: "Product", Package: "p", FilePath: "Product.java"},
		{Name: "Type", Package: "p", Outer: "Product", FilePath: "Product.java", IsEnum: true, EnumConstants: []string{"PHYSICAL", "VIRTUAL"}},
		{Name: "OrderForm", Package: "p", FilePath: "Form.java", Fields: []parser.Field{{Name: "type", Type: "Order.Type"}, {Name: "kind", Type: "Product.Type"}}},
		// A bare name that no scope declares is none of them.
		{Name: "Stray", Package: "q", FilePath: "Stray.java", Fields: []parser.Field{{Name: "type", Type: "Type"}}},
	}
	got, unresolved := resolveJSON(classes, "p.OrderForm")
	if field := got.Fields["type"]; field.Model.TypeName != model.JsonTypeString || !reflect.DeepEqual(optionValues(field.Options), []any{"RETAIL", "WHOLESALE"}) {
		t.Errorf("type = %+v, options %v, want the constants of Order.Type", field.Model, field.Options)
	}
	if field := got.Fields["kind"]; field.Model.TypeName != model.JsonTypeString || !reflect.DeepEqual(optionValues(field.Options), []any{"PHYSICAL", "VIRTUAL"}) {
		t.Errorf("kind = %+v, options %v, want the constants of Product.Type", field.Model, field.Options)
	}
	stray, unresolved := resolveJSON(classes, "q.Stray")
	if field := stray.Fields["type"]; field.Model.TypeName != "Type" || len(field.Options) != 0 || unresolved["Type"] != 1 {
		t.Errorf("type = %+v, options %v, unresolved %v; want the name as written, reported unresolved", field.Model, field.Options, unresolved)
	}
}

// An enum nested in a class is named through the class: Order.Status, or the
// import com.example.Order.Status. The enum is the one that is nested in exactly
// those classes.
func TestResolve_ANestedEnumIsReachedThroughItsOuterClass(t *testing.T) {
	classes := []parser.Class{
		{Name: "Order", Package: "com.example", FilePath: "Order.java"},
		{Name: "Inner", Package: "com.example", Outer: "Order", FilePath: "Order.java"},
		{Name: "Status", Package: "com.example", Outer: "Order", FilePath: "Order.java", IsEnum: true, EnumConstants: []string{"OPEN", "CLOSED"}},
		{Name: "Status", Package: "com.example", Outer: "Order.Inner", FilePath: "Order.java", IsEnum: true, EnumConstants: []string{"DEEP"}},
		// Declared in a file of its own, so nested in nothing.
		{Name: "Stray", Package: "com.example", FilePath: "Stray.java", IsEnum: true, EnumConstants: []string{"X"}},
		{
			Name: "Holder", Package: "com.other", FilePath: "Holder.java",
			Imports: []parser.Import{{Path: "com.example.Order.Status"}},
			Fields: []parser.Field{
				{Name: "imported", Type: "Status"},
				{Name: "qualified", Type: "com.example.Order.Status"},
				{Name: "deeper", Type: "com.example.Order.Inner.Status"},
				{Name: "strayed", Type: "com.example.Order.Stray"},
				{Name: "foreign", Type: "com.dep.Order.Status"},
			},
		},
		{
			Name: "Local", Package: "com.example", FilePath: "Local.java",
			Fields: []parser.Field{{Name: "relative", Type: "Order.Status"}, {Name: "missing", Type: "Order.Gone"}},
		},
	}
	want := []any{"OPEN", "CLOSED"}

	holder, _ := resolveJSON(classes, "com.other.Holder")
	for name, constants := range map[string][]any{"imported": want, "qualified": want, "deeper": {"DEEP"}} {
		field := holder.Fields[name]
		if field.Model.TypeName != model.JsonTypeString || !reflect.DeepEqual(optionValues(field.Options), constants) {
			t.Errorf("%s = %+v, options %v, want a string of %v", name, field.Model, optionValues(field.Options), constants)
		}
	}
	for name, typeName := range map[string]string{"strayed": "com.example.Order.Stray", "foreign": "com.dep.Order.Status"} {
		if field := holder.Fields[name]; field.Model.TypeName != typeName || len(field.Options) != 0 {
			t.Errorf("%s = %+v, options %v, want the type left unresolved as %s", name, field.Model, field.Options, typeName)
		}
	}

	local, _ := resolveJSON(classes, "com.example.Local")
	if field := local.Fields["relative"]; field.Model.TypeName != model.JsonTypeString || !reflect.DeepEqual(optionValues(field.Options), want) {
		t.Errorf("relative = %+v, options %v, want a string of OPEN and CLOSED", field.Model, field.Options)
	}
	if field := local.Fields["missing"]; field.Model.TypeName != "Order.Gone" {
		t.Errorf("missing = %+v, want the type left unresolved", field.Model)
	}
}

// A type parameter bound to an enum, and a collection of one, give the field the
// constants, as the field's own type does. The values of a map are strings of
// them, and the map lists none.
func TestResolve_EnumsBehindTypeArgumentsAndCollections(t *testing.T) {
	classes := []parser.Class{
		{Name: "Status", Package: "p", IsEnum: true, EnumConstants: []string{"ON", "OFF"}},
		{Name: "Envelope", Package: "p", TypeParameters: []string{"T"}, Fields: []parser.Field{{Name: "data", Type: "T"}}},
		{Name: "Board", Package: "p", Fields: []parser.Field{
			{Name: "set", Type: "Set<Status>"},
			{Name: "byName", Type: "Map<String, Status>"},
			{Name: "wild", Type: "List<? extends Status>"},
			{Name: "nested", Type: "List<List<Status>>"},
			{Name: "numbers", Type: "List<Integer>"},
		}},
	}
	r := NewTypeResolver(classes)
	source := parser.Class{Package: "p"}
	want := []any{"ON", "OFF"}

	for _, typeName := range []string{"Envelope<Status>", "Envelope<List<Status>>"} {
		got := r.ResolveFrom(source, typeName, nil).Fields["data"]
		if !reflect.DeepEqual(optionValues(got.Options), want) {
			t.Errorf("%s: data options = %v, want %v", typeName, optionValues(got.Options), want)
		}
	}

	board := r.ResolveFrom(source, "Board", nil)
	for _, name := range []string{"set", "wild", "nested"} {
		if got := optionValues(board.Fields[name].Options); !reflect.DeepEqual(got, want) {
			t.Errorf("%s options = %v, want %v", name, got, want)
		}
	}
	byName := board.Fields["byName"]
	if !byName.Model.IsMap() || byName.Model.ValueModel.TypeName != model.JsonTypeString || len(byName.Options) != 0 {
		t.Errorf("byName = %+v, options %v, want a map of strings that lists no options", byName.Model, byName.Options)
	}
	for _, name := range []string{"numbers"} {
		if got := board.Fields[name].Options; len(got) != 0 {
			t.Errorf("%s has options %v", name, got)
		}
	}
}

func TestResolveForBinding_EnumIsAStringWithItsConstants(t *testing.T) {
	r := NewTypeResolver(parseOrderWithEnum(t))
	order := r.ResolveForBinding(parser.Class{}, "com.example.Order", nil)
	status := order.Fields["status"]
	if status.Model.Kind != model.KindSingle || status.Model.TypeName != model.JsonTypeString {
		t.Fatalf("status model = %+v", status.Model)
	}
	if values := optionValues(status.Options); !reflect.DeepEqual(values, []any{"OPEN", "CLOSED"}) {
		t.Fatalf("status options = %v", values)
	}
	// A collection of enums is a list of strings whose field lists the constants,
	// as in the JSON view.
	history := order.Fields["history"]
	if history.Model.Kind != model.KindArray || history.Model.Items.TypeName != model.JsonTypeString || !reflect.DeepEqual(optionValues(history.Options), []any{"OPEN", "CLOSED"}) {
		t.Fatalf("history = %+v, options %v", history.Model, history.Options)
	}

	constants, ok := r.EnumConstants(parser.Class{Name: "Order", Package: "com.example"}, "Status")
	if !ok || !reflect.DeepEqual(constants, []string{"OPEN", "CLOSED"}) {
		t.Fatalf("EnumConstants = %v, %v", constants, ok)
	}
}

func TestResolveForBinding_LeavesOutGetterOnlyProperties(t *testing.T) {
	classes := []parser.Class{{
		Name:    "Query",
		Package: "com.example",
		Fields:  []parser.Field{{Name: "keyword", Type: "String"}},
		Methods: []parser.Method{{Name: "getSummary", ReturnType: "String"}},
	}}
	r := NewTypeResolver(classes)
	if _, ok := r.Resolve("com.example.Query", nil).Fields["summary"]; !ok {
		t.Fatal("Jackson writes a getter-only property, so the JSON view keeps it")
	}
	bound := r.ResolveForBinding(parser.Class{}, "com.example.Query", nil)
	if _, ok := bound.Fields["summary"]; ok {
		t.Fatal("the binding view must leave out a property Spring cannot set")
	}
	if _, ok := bound.Fields["keyword"]; !ok {
		t.Fatal("the binding view keeps settable fields")
	}
	// The view ends with the call.
	if _, ok := r.Resolve("com.example.Query", nil).Fields["summary"]; !ok {
		t.Fatal("the JSON view must be unaffected after a binding resolution")
	}
}

// A simple name denotes the declaration the scope of the source gives it, whether
// or not the current view reads that declaration. An enum of the source's own
// package whose JSON form is not the name of its constant stays unresolved in the
// JSON view, and an enum of another package that carries the same simple name
// does not stand in for it.
func TestResolve_AnEnumInScopeIsNotReplacedByAnotherOfTheSameName(t *testing.T) {
	excluded := parser.Class{Name: "Code", Package: "a", FilePath: "a/Code.java", IsEnum: true, EnumConstants: []string{"A"}, EnumJsonValue: true}
	plain := parser.Class{Name: "Code", Package: "b", FilePath: "b/Code.java", IsEnum: true, EnumConstants: []string{"B"}}
	fields := []parser.Field{{Name: "code", Type: "Code"}, {Name: "codes", Type: "List<Code>"}}
	classes := []parser.Class{
		excluded,
		plain,
		{Name: "Holder", Package: "a", FilePath: "a/Holder.java", Fields: fields},
		{Name: "Holder", Package: "b", FilePath: "b/Holder.java", Fields: fields},
		{Name: "OnDemand", Package: "c", FilePath: "c/OnDemand.java", Imports: []parser.Import{{Path: "a", OnDemand: true}}, Fields: fields},
		{Name: "Single", Package: "c", FilePath: "c/Single.java", Imports: []parser.Import{{Path: "a.Code"}}, Fields: fields},
		// Declares its own Code, nested in it, next to the two above.
		{Name: "Own", Package: "d", FilePath: "d/Own.java", Fields: fields},
		{Name: "Code", Package: "d", FilePath: "d/Own.java", IsEnum: true, EnumConstants: []string{"D"}, EnumJsonValue: true},
		// Neither declares nor imports a Code: no scope binds the name, and the
		// declarations that carry it are not taken for it, in either view.
		{Name: "Stranger", Package: "e", FilePath: "e/Stranger.java", Fields: fields},
	}

	for _, holder := range []string{"a.Holder", "c.OnDemand", "c.Single", "d.Own", "e.Stranger"} {
		got, _ := resolveJSON(classes, holder)
		for _, name := range []string{"code", "codes"} {
			field := got.Fields[name]
			typeName := field.Model.TypeName
			if name == "codes" {
				typeName = field.Model.Items.TypeName
			}
			if typeName != "Code" || len(field.Options) != 0 {
				t.Errorf("%s.%s = %+v, options %v; want the type name of the enum it names, left unresolved", holder, name, field.Model, optionValues(field.Options))
			}
		}
	}

	got, _ := resolveJSON(classes, "b.Holder")
	if code := got.Fields["code"]; code.Model.TypeName != model.JsonTypeString || !reflect.DeepEqual(optionValues(code.Options), []any{"B"}) {
		t.Errorf("b.Holder.code = %+v, options %v, want a string of B", code.Model, optionValues(code.Options))
	}

	// The binding view reads every enum, and the one in scope is the one it reads.
	r := NewTypeResolver(classes)
	if bound := r.ResolveForBinding(parser.Class{}, "a.Holder", nil).Fields["code"]; !reflect.DeepEqual(optionValues(bound.Options), []any{"A"}) {
		t.Errorf("binding a.Holder.code options = %v, want A", optionValues(bound.Options))
	}
}

// The same holds for a class of the same simple name elsewhere: the name binds to
// the enum in scope, and the JSON view does not expand the class in its place.
func TestResolve_AnEnumInScopeIsNotReplacedByAClassOfTheSameName(t *testing.T) {
	classes := []parser.Class{
		{Name: "Code", Package: "a", FilePath: "a/Code.java", IsEnum: true, EnumConstants: []string{"A"}, EnumJsonValue: true},
		{Name: "Code", Package: "b", FilePath: "b/Code.java", Fields: []parser.Field{{Name: "number", Type: "int"}}},
		{Name: "Holder", Package: "a", FilePath: "a/Holder.java", Fields: []parser.Field{{Name: "code", Type: "Code"}}},
	}
	got, _ := resolveJSON(classes, "a.Holder")
	if code := got.Fields["code"]; code.Model.TypeName != "Code" || len(code.Model.Fields) != 0 {
		t.Errorf("code = %+v, want the type name of the enum in scope, not the fields of b.Code", code.Model)
	}
}

// An enum named through its outer class, Order.Status, is the one nested in the
// Order the source's scope gives, not the first one of that name: the outer class
// is looked up like any other simple name, and then its nested enum is taken.
func TestResolve_ANestedEnumIsFoundThroughTheOuterClassInScope(t *testing.T) {
	fields := []parser.Field{
		{Name: "status", Type: "Order.Status"},
		{Name: "history", Type: "List<Order.Status>"},
		{Name: "wide", Type: "List<? extends Order.Status>"},
		{Name: "set", Type: "Set<Order.Status>"},
	}
	bHolder := parser.Class{Name: "Holder", Package: "b", FilePath: "b/Holder.java", Fields: fields}
	classes := []parser.Class{
		{Name: "Order", Package: "a", FilePath: "a/Order.java"},
		{Name: "Status", Package: "a", Outer: "Order", FilePath: "a/Order.java", IsEnum: true, EnumConstants: []string{"A1", "A2"}},
		{Name: "Order", Package: "b", FilePath: "b/Order.java"},
		{Name: "Status", Package: "b", Outer: "Order", FilePath: "b/Order.java", IsEnum: true, EnumConstants: []string{"B1"}},
		{Name: "Holder", Package: "a", FilePath: "a/Holder.java", Fields: fields},
		bHolder,
		{Name: "Single", Package: "c", FilePath: "c/Single.java", Imports: []parser.Import{{Path: "b.Order"}}, Fields: fields},
		{Name: "OnDemand", Package: "c", FilePath: "c/OnDemand.java", Imports: []parser.Import{{Path: "b", OnDemand: true}}, Fields: fields},
		{Name: "Qualified", Package: "c", FilePath: "c/Qualified.java", Fields: []parser.Field{{Name: "status", Type: "b.Order.Status"}, {Name: "history", Type: "List<b.Order.Status>"}}},
	}

	for holder, want := range map[string][]any{
		"a.Holder":    {"A1", "A2"},
		"b.Holder":    {"B1"},
		"c.Single":    {"B1"},
		"c.OnDemand":  {"B1"},
		"c.Qualified": {"B1"},
	} {
		got, unresolved := resolveJSON(classes, holder)
		for name, field := range got.Fields {
			if !reflect.DeepEqual(optionValues(field.Options), want) {
				t.Errorf("%s.%s options = %v, want %v", holder, name, optionValues(field.Options), want)
			}
		}
		if len(unresolved) != 0 {
			t.Errorf("%s: unresolved = %v, want none", holder, unresolved)
		}
	}

	r := NewTypeResolver(classes)
	if bound := r.ResolveForBinding(bHolder, "Order.Status", nil); bound.TypeName != model.JsonTypeString {
		t.Errorf("binding Order.Status = %+v, want a string", bound)
	}
	constants, ok := r.EnumConstants(bHolder, "Order.Status")
	if !ok || !reflect.DeepEqual(constants, []string{"B1"}) {
		t.Errorf("EnumConstants(b.Holder, Order.Status) = %v, %v, want B1", constants, ok)
	}
}

// A type argument is written in the scope of the source and read later inside
// the generic class, where that scope no longer applies: a nested enum named
// through its outer class is qualified, as a simple name is, so that it is not
// looked up again among the declarations of the class that receives it.
func TestResolve_ANestedEnumTypeArgumentKeepsTheScopeItIsWrittenIn(t *testing.T) {
	classes := []parser.Class{
		{Name: "Order", Package: "b", FilePath: "b/Order.java"},
		{Name: "Status", Package: "b", Outer: "Order", FilePath: "b/Order.java", IsEnum: true, EnumConstants: []string{"B1"}},
		{Name: "Order", Package: "a", FilePath: "a/Order.java"},
		{Name: "Status", Package: "a", Outer: "Order", FilePath: "a/Order.java", IsEnum: true, EnumConstants: []string{"A1"}},
		{Name: "Envelope", Package: "a", FilePath: "a/Envelope.java", TypeParameters: []string{"T"}, Fields: []parser.Field{{Name: "data", Type: "T"}}},
		{Name: "Holder", Package: "b", FilePath: "b/Holder.java", Imports: []parser.Import{{Path: "a.Envelope"}}, Fields: []parser.Field{
			{Name: "one", Type: "Envelope<Order.Status>"},
			{Name: "many", Type: "Envelope<List<Order.Status>>"},
		}},
	}
	r := NewTypeResolver(classes)
	r.SetDependencyResolver(&libraryClasses{})

	holder := r.Resolve("b.Holder", nil)
	for _, name := range []string{"one", "many"} {
		data := holder.Fields[name].Model.Fields["data"]
		if data == nil || data.Model.Kind == model.KindObject {
			t.Fatalf("%s = %+v, want an envelope with a data field", name, holder.Fields[name].Model)
		}
		if got := optionValues(data.Options); !reflect.DeepEqual(got, []any{"B1"}) {
			t.Errorf("%s.data options = %v, want the constants of b.Order.Status", name, got)
		}
	}
}
