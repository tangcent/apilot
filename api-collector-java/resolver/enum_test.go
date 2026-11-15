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

// withoutEnums drops the enum declarations, as the parser did before it read
// them.
func withoutEnums(classes []parser.Class) []parser.Class {
	var out []parser.Class
	for _, c := range classes {
		if !c.IsEnum {
			out = append(out, c)
		}
	}
	return out
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

// The JSON view does not see enums: it resolves every type exactly as it does
// when the enum declarations are not read at all.
func TestResolve_JSONViewDoesNotSeeEnums(t *testing.T) {
	classes := append(parseOrderWithEnum(t),
		// An enum shares its simple name with a class of another package. A
		// reference without imports falls back to the simple name, which must
		// still find the class.
		parser.Class{Name: "Level", Package: "a", IsEnum: true, EnumConstants: []string{"LOW"}},
		parser.Class{Name: "Level", Package: "b", Fields: []parser.Field{{Name: "rank", Type: "int"}}},
		parser.Class{Name: "Holder", Package: "c", Fields: []parser.Field{{Name: "level", Type: "Level"}}},
	)
	for _, typeName := range []string{"com.example.Order", "c.Holder"} {
		got, gotUnresolved := resolveJSON(classes, typeName)
		want, wantUnresolved := resolveJSON(withoutEnums(classes), typeName)
		if !reflect.DeepEqual(got, want) {
			t.Errorf("%s:\n got  %+v\n want %+v", typeName, got, want)
		}
		if !reflect.DeepEqual(gotUnresolved, wantUnresolved) {
			t.Errorf("%s unresolved = %v, want %v", typeName, gotUnresolved, wantUnresolved)
		}
	}

	order, _ := resolveJSON(classes, "com.example.Order")
	if status := order.Fields["status"]; status.Model.TypeName != "Status" || len(status.Options) != 0 {
		t.Errorf("status = %+v, options %v", status.Model, status.Options)
	}
	holder, _ := resolveJSON(classes, "c.Holder")
	if _, ok := holder.Fields["level"].Model.Fields["rank"]; !ok {
		t.Errorf("level = %+v, want the class b.Level", holder.Fields["level"].Model)
	}
}

func TestResolveForBinding_EnumIsAStringWithItsConstants(t *testing.T) {
	r := NewTypeResolver(parseOrderWithEnum(t))
	order := r.ResolveForBinding(parser.Class{}, "com.example.Order", nil)
	status := order.Fields["status"]
	if status.Model.Kind != model.KindSingle || status.Model.TypeName != model.JsonTypeString {
		t.Fatalf("status model = %+v", status.Model)
	}
	var values []any
	for _, option := range status.Options {
		values = append(values, option.Value)
	}
	if !reflect.DeepEqual(values, []any{"OPEN", "CLOSED"}) {
		t.Fatalf("status options = %v", values)
	}
	history := order.Fields["history"].Model
	if history.Kind != model.KindArray || history.Items.TypeName != model.JsonTypeString {
		t.Fatalf("history = %+v", history)
	}

	constants, ok := r.EnumConstants(parser.Class{Package: "com.example"}, "Status")
	if !ok || !reflect.DeepEqual(constants, []string{"OPEN", "CLOSED"}) {
		t.Fatalf("EnumConstants = %v, %v", constants, ok)
	}
	// The view ends with the call.
	if got := r.Resolve("com.example.Order", nil).Fields["status"]; got.Model.TypeName != "Status" || len(got.Options) != 0 {
		t.Fatalf("the JSON view must be unaffected after a binding resolution: %+v", got)
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
