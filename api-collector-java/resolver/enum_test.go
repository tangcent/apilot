package resolver

import (
	"reflect"
	"testing"

	"github.com/tangcent/apilot/api-collector-java/parser"
	model "github.com/tangcent/apilot/api-model"
)

func TestResolve_EnumIsAStringWithItsConstants(t *testing.T) {
	jp, err := parser.NewParser(parser.ParserOptions{})
	if err != nil {
		t.Fatal(err)
	}
	defer jp.Close()
	classes, err := jp.ParseSource([]byte(`package com.example;

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
`))
	if err != nil {
		t.Fatal(err)
	}

	var enum *parser.Class
	for i := range classes {
		if classes[i].IsEnum {
			enum = &classes[i]
		}
	}
	if enum == nil || enum.Name != "Status" || !reflect.DeepEqual(enum.EnumConstants, []string{"OPEN", "CLOSED"}) {
		t.Fatalf("enum = %+v", enum)
	}

	order := NewTypeResolver(classes).Resolve("com.example.Order", nil)
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
