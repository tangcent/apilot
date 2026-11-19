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

func optionValues(options []model.FieldOption) []any {
	var values []any
	for _, option := range options {
		values = append(values, option.Value)
	}
	return values
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
		asString   bool
	}{
		{name: "no annotation", asString: true},
		{name: "@JsonValue on a method", body: "@JsonValue public String code() { return null; }"},
		{name: "@JsonValue on a field", body: "@JsonValue private String code;"},
		{name: "@JsonValue by its qualified name", body: "@com.fasterxml.jackson.annotation.JsonValue public String code() { return null; }"},
		{name: "@JsonValue switched off", body: "@JsonValue(false) public String code() { return null; }", asString: true},
		{name: "@JsonValue only inside the body of a constant", asString: true},
		{name: "@JsonFormat shape object", annotation: "@JsonFormat(shape = JsonFormat.Shape.OBJECT)"},
		{name: "@JsonFormat shape object, imported", annotation: "@JsonFormat(shape = Shape.OBJECT)"},
		{name: "@JsonFormat shape number", annotation: "@JsonFormat(shape = JsonFormat.Shape.NUMBER)"},
		{name: "@JsonFormat shape number int", annotation: "@JsonFormat(shape = JsonFormat.Shape.NUMBER_INT)"},
		{name: "@JsonFormat shape string", annotation: "@JsonFormat(shape = JsonFormat.Shape.STRING)", asString: true},
		{name: "@JsonFormat without a shape", annotation: `@JsonFormat(pattern = "x")`, asString: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			source := `package a;

import com.fasterxml.jackson.annotation.JsonFormat;
import com.fasterxml.jackson.annotation.JsonValue;

` + tt.annotation + `
enum Level {
    LOW {
        @JsonValue String other() { return null; }
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

// A reference no scope declares falls back to the declarations that carry its
// simple name, and the JSON view did not read enums when that rule was made: a
// class goes before an enum of the same name, and an enum answers only where
// no class does.
func TestResolve_AnEnumDoesNotTakeTheNameOfAClass(t *testing.T) {
	classes := []parser.Class{
		{Name: "Level", Package: "a", IsEnum: true, EnumConstants: []string{"LOW"}},
		{Name: "Level", Package: "b", Fields: []parser.Field{{Name: "rank", Type: "int"}}},
		{Name: "Mode", Package: "a", IsEnum: true, EnumConstants: []string{"ON", "OFF"}},
		{Name: "Holder", Package: "c", Fields: []parser.Field{{Name: "level", Type: "Level"}, {Name: "mode", Type: "Mode"}}},
	}
	holder, _ := resolveJSON(classes, "c.Holder")

	level := holder.Fields["level"]
	if _, ok := level.Model.Fields["rank"]; !ok || len(level.Options) != 0 {
		t.Errorf("level = %+v, options %v, want the class b.Level", level.Model, level.Options)
	}
	mode := holder.Fields["mode"]
	if mode.Model.TypeName != model.JsonTypeString || !reflect.DeepEqual(optionValues(mode.Options), []any{"ON", "OFF"}) {
		t.Errorf("mode = %+v, options %v, want a string of ON and OFF", mode.Model, mode.Options)
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

// Two enums nested in different classes of one package share a registry key,
// and the registry keeps the last. The JSON view does not guess between them:
// it exports both by their type name, as it did before it read enums, while the
// binding view keeps reading the one the registry holds. A file that is read
// twice is no such sharing.
func TestResolve_EnumsThatShareAKeyAreExportedByName(t *testing.T) {
	order := parser.Class{Name: "Order", Package: "p", FilePath: "Order.java"}
	orderType := parser.Class{Name: "Type", Package: "p", FilePath: "Order.java", IsEnum: true, EnumConstants: []string{"RETAIL", "WHOLESALE"}}
	product := parser.Class{Name: "Product", Package: "p", FilePath: "Product.java"}
	productType := parser.Class{Name: "Type", Package: "p", FilePath: "Product.java", IsEnum: true, EnumConstants: []string{"PHYSICAL", "VIRTUAL"}}
	orderForm := parser.Class{Name: "OrderForm", Package: "p", FilePath: "Order.java", Fields: []parser.Field{{Name: "type", Type: "Type"}}}

	got, unresolved := resolveJSON([]parser.Class{order, orderType, product, productType, orderForm}, "p.OrderForm")
	if field := got.Fields["type"]; field.Model.TypeName != "Type" || len(field.Options) != 0 || unresolved["Type"] != 1 {
		t.Errorf("type = %+v, options %v, unresolved %v; want the type name, reported unresolved", field.Model, field.Options, unresolved)
	}

	r := NewTypeResolver([]parser.Class{order, orderType, product, productType, orderForm})
	if bound := r.ResolveForBinding(parser.Class{}, "p.OrderForm", nil).Fields["type"]; bound.Model.TypeName != model.JsonTypeString {
		t.Errorf("the binding view = %+v, want it to go on reading the enum", bound.Model)
	}

	// The same declaration registered twice shares nothing.
	twice, unresolved := resolveJSON([]parser.Class{order, orderType, orderType, orderForm}, "p.OrderForm")
	if field := twice.Fields["type"]; field.Model.TypeName != model.JsonTypeString || len(unresolved) != 0 {
		t.Errorf("type = %+v, unresolved %v; want a string of the constants", field.Model, unresolved)
	}
}

// An enum nested in a class is registered by its package and simple name, but
// is also named through the class: Order.Status, or the import
// com.example.Order.Status. The enum is the one carrying the last segment whose
// enclosing classes are declared in its own file.
func TestResolve_ANestedEnumIsReachedThroughItsOuterClass(t *testing.T) {
	classes := []parser.Class{
		{Name: "Order", Package: "com.example", FilePath: "Order.java"},
		{Name: "Inner", Package: "com.example", FilePath: "Order.java"},
		{Name: "Status", Package: "com.example", FilePath: "Order.java", IsEnum: true, EnumConstants: []string{"OPEN", "CLOSED"}},
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
	for _, name := range []string{"imported", "qualified", "deeper"} {
		field := holder.Fields[name]
		if field.Model.TypeName != model.JsonTypeString || !reflect.DeepEqual(optionValues(field.Options), want) {
			t.Errorf("%s = %+v, options %v, want a string of OPEN and CLOSED", name, field.Model, field.Options)
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
	// A model attribute takes the options of its own type only: a collection
	// of enums is a list of strings, as before.
	history := order.Fields["history"]
	if history.Model.Kind != model.KindArray || history.Model.Items.TypeName != model.JsonTypeString || len(history.Options) != 0 {
		t.Fatalf("history = %+v, options %v", history.Model, history.Options)
	}

	constants, ok := r.EnumConstants(parser.Class{Package: "com.example"}, "Status")
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
