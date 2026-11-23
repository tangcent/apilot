package parser

import (
	"reflect"
	"testing"
)

func parseNestedSource(t *testing.T, source string) map[string]Class {
	t.Helper()
	p, err := NewParser(ParserOptions{})
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()
	classes, err := p.ParseSource([]byte(source))
	if err != nil {
		t.Fatal(err)
	}
	byKey := map[string]Class{}
	for _, c := range classes {
		if _, dup := byKey[c.QualifiedName()]; dup {
			t.Fatalf("%s is declared twice: %+v", c.QualifiedName(), classes)
		}
		byKey[c.QualifiedName()] = c
	}
	return byKey
}

// A class nested in another is a class of its own that says where it is
// nested, and two nested classes of one name differ by it.
func TestParser_RecordsTheClassesADeclarationIsNestedIn(t *testing.T) {
	classes := parseNestedSource(t, `package com.example;

public class Outer {
    public static class Info {
        String ofOuter;
    }

    public interface Handler {
        class Impl {
            String handled;
        }
    }

    public enum State { OPEN, SHUT }

    public record Pair(String left, String right) {
        public static class Info {
            String ofPair;
        }
    }

    void run() {
        class Local {
            String local;
        }
    }
}

class Other {
    static class Info {
        String ofOther;
    }
}
`)
	want := map[string]string{
		"com.example.Outer":              "",
		"com.example.Outer.Info":         "Outer",
		"com.example.Outer.Handler":      "Outer",
		"com.example.Outer.Handler.Impl": "Outer.Handler",
		"com.example.Outer.State":        "Outer",
		"com.example.Outer.Pair":         "Outer",
		"com.example.Outer.Pair.Info":    "Outer.Pair",
		"com.example.Outer.Local":        "Outer",
		"com.example.Other":              "",
		"com.example.Other.Info":         "Other",
	}
	got := map[string]string{}
	for key, c := range classes {
		got[key] = c.Outer
		if c.Package != "com.example" {
			t.Errorf("%s has package %q", key, c.Package)
		}
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("classes:\n got  %v\n want %v", got, want)
	}
	if fields := classes["com.example.Outer.Info"].Fields; len(fields) != 1 || fields[0].Name != "ofOuter" {
		t.Errorf("Outer.Info fields = %+v, want its own", fields)
	}
	if fields := classes["com.example.Other.Info"].Fields; len(fields) != 1 || fields[0].Name != "ofOther" {
		t.Errorf("Other.Info fields = %+v, want its own", fields)
	}
}

func TestQualifiedName(t *testing.T) {
	for _, tc := range []struct {
		class Class
		want  string
	}{
		{Class{Name: "Order", Package: "com.example"}, "com.example.Order"},
		{Class{Name: "Item", Package: "com.example", Outer: "Order"}, "com.example.Order.Item"},
		{Class{Name: "Deep", Package: "com.example", Outer: "Order.Item"}, "com.example.Order.Item.Deep"},
		{Class{Name: "Order"}, "Order"},
		{Class{Name: "Item", Outer: "Order"}, "Order.Item"},
	} {
		if got := tc.class.QualifiedName(); got != tc.want {
			t.Errorf("QualifiedName(%+v) = %q, want %q", tc.class, got, tc.want)
		}
	}
}

// A field, a return type and a supertype written with a qualified name are read
// as written: `Outer.Inner` names a nested class, and `java.util.Date` a class
// of a package. Neither was read before, and the declaration was left out.
func TestParser_ReadsATypeWrittenWithAQualifiedName(t *testing.T) {
	classes := parseNestedSource(t, `package com.example;

import java.util.List;

public class Holder extends Base.Part<Holder.Detail, String> implements Base.Marker, java.io.Serializable, Comparable<Holder> {
    private Holder.Detail detail;
    private java.util.Date when;
    private List<Holder.Detail> details;
    private Base.Part<Holder.Detail, String> part;

    public Holder.Detail first() {
        return null;
    }

    public java.util.Date at(Holder.Detail detail, java.util.Date when) {
        return null;
    }

    public static class Detail {
        String code;
    }
}
`)
	holder := classes["com.example.Holder"]

	fields := map[string]string{}
	for _, f := range holder.Fields {
		fields[f.Name] = f.Type
	}
	wantFields := map[string]string{
		"detail":  "Holder.Detail",
		"when":    "java.util.Date",
		"details": "List<Holder.Detail>",
		"part":    "Base.Part<Holder.Detail, String>",
	}
	if !reflect.DeepEqual(fields, wantFields) {
		t.Errorf("fields:\n got  %v\n want %v", fields, wantFields)
	}

	returns := map[string]string{}
	for _, m := range holder.Methods {
		returns[m.Name] = m.ReturnType
	}
	if want := map[string]string{"first": "Holder.Detail", "at": "java.util.Date"}; !reflect.DeepEqual(returns, want) {
		t.Errorf("return types:\n got  %v\n want %v", returns, want)
	}
	at := holder.Methods[1]
	if len(at.Parameters) != 2 || at.Parameters[0].Type != "Holder.Detail" || at.Parameters[1].Type != "java.util.Date" {
		t.Errorf("parameters of at = %+v", at.Parameters)
	}

	if holder.SuperClass != "Base.Part" || !reflect.DeepEqual(holder.SuperClassTypeArgs, []string{"Holder.Detail", "String"}) {
		t.Errorf("superclass = %q%v, want Base.Part<Holder.Detail, String>", holder.SuperClass, holder.SuperClassTypeArgs)
	}
	if want := []string{"Base.Marker", "java.io.Serializable", "Comparable"}; !reflect.DeepEqual(holder.Interfaces, want) {
		t.Errorf("interfaces = %v, want %v", holder.Interfaces, want)
	}
}
