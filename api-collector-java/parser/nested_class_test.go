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

// An interface inherits the member types of the interfaces it extends, so the
// parser keeps them as it keeps the interfaces a class implements.
func TestParser_ReadsTheInterfacesAnInterfaceExtends(t *testing.T) {
	classes := parseNestedSource(t, `package com.example;

public interface Wide extends Base.Marker, java.io.Serializable, Comparable<Wide>, Mid {
}

interface Plain {
}
`)
	if want := []string{"Base.Marker", "java.io.Serializable", "Comparable", "Mid"}; !reflect.DeepEqual(classes["com.example.Wide"].Interfaces, want) {
		t.Errorf("Wide extends %v, want %v", classes["com.example.Wide"].Interfaces, want)
	}
	if got := classes["com.example.Plain"].Interfaces; len(got) != 0 {
		t.Errorf("Plain extends %v, want nothing", got)
	}
}

// The type arguments of a superclass are read whether a name is written simply
// or in full, and so is the superclass itself.
func TestParser_ReadsTheTypeArgumentsOfASuperclassWrittenInFull(t *testing.T) {
	classes := parseNestedSource(t, `package com.example;

class Dated extends Base<java.util.Date> {}
class Itemised extends Base<com.example.Item> {}
class Both extends com.example.Base<java.lang.String, com.example.Item[]> {}
class Mapped extends java.util.HashMap<String, Item> {}
class Nested extends Base<java.util.List<com.example.Item>> {}
`)
	for _, tc := range []struct {
		key, super string
		args       []string
	}{
		{"com.example.Dated", "Base", []string{"java.util.Date"}},
		{"com.example.Itemised", "Base", []string{"com.example.Item"}},
		{"com.example.Both", "com.example.Base", []string{"java.lang.String", "com.example.Item[]"}},
		{"com.example.Mapped", "java.util.HashMap", []string{"String", "Item"}},
		{"com.example.Nested", "Base", []string{"java.util.List<com.example.Item>"}},
	} {
		c := classes[tc.key]
		if c.SuperClass != tc.super || !reflect.DeepEqual(c.SuperClassTypeArgs, tc.args) {
			t.Errorf("%s extends %q%v, want %q%v", tc.key, c.SuperClass, c.SuperClassTypeArgs, tc.super, tc.args)
		}
	}
}

// A class declared in a block is local: its name is in scope in that block only.
// A member of a local class is local too, and a member of a member is not.
func TestParser_MarksTheClassesDeclaredInABlock(t *testing.T) {
	classes := parseNestedSource(t, `package com.example;

import java.util.function.Supplier;

public class Outer {
    public static class Member {
        public static class Deeper {}
    }

    private final Runnable anonymous = new Runnable() {
        class InAnonymous {}
        public void run() {}
    };

    static {
        class InInitializer {}
    }

    Outer() {
        record InConstructor(String a) {}
    }

    void run(int x) {
        class InMethod {
            class InLocal {}
        }
        interface LocalInterface {}
        enum LocalEnum { A }
        Supplier<Object> lambda = () -> {
            class InLambda {}
            return null;
        };
        switch (x) {
            case 1:
                class InSwitch {}
        }
    }

    enum Kind {
        ONE {
            class InConstant {}
        };
        static class InEnum {}
    }

    interface Contract {
        class InInterface {}
    }
}
`)
	local := map[string]bool{}
	for key, c := range classes {
		local[key[len("com.example."):]] = c.Local
	}
	want := map[string]bool{
		"Outer": false, "Outer.Member": false, "Outer.Member.Deeper": false,
		"Outer.Kind": false, "Outer.Kind.InEnum": false, "Outer.Contract": false, "Outer.Contract.InInterface": false,
		"Outer.InAnonymous": true, "Outer.InInitializer": true, "Outer.InConstructor": true,
		"Outer.InMethod": true, "Outer.InMethod.InLocal": true, "Outer.LocalInterface": true, "Outer.LocalEnum": true,
		"Outer.InLambda": true, "Outer.InSwitch": true, "Outer.Kind.InConstant": true,
	}
	if !reflect.DeepEqual(local, want) {
		t.Errorf("local classes:\n got  %v\n want %v", local, want)
	}
}

// An inner class is a member class that is not static, and it is the only kind
// that sees the type parameters of the classes around it. A member of an
// interface is static without saying so; an interface, an enum and a record
// are never inner, and a class declared in a block is not a member.
func TestParser_MarksTheInnerClasses(t *testing.T) {
	classes := parseNestedSource(t, `package com.example;

public class Outer<T> {
    class Inner {}
    static class Nested {}
    public final class FinalInner {}
    class Mid {
        class Deep {}
        static class Fixed {}
    }
    interface Contract {
        class InInterface {}
    }
    enum Kind {
        ONE;
        class InEnum {}
    }
    record Pair(String a) {
        class InRecord {}
    }
    private final Runnable anonymous = new Runnable() {
        class InAnonymous {}
        public void run() {}
    };
    void run() {
        class InMethod {}
    }
}
`)
	inner := map[string]bool{}
	for key, c := range classes {
		inner[key[len("com.example."):]] = c.Inner
	}
	want := map[string]bool{
		"Outer": false, "Outer.Inner": true, "Outer.Nested": false, "Outer.FinalInner": true,
		"Outer.Mid": true, "Outer.Mid.Deep": true, "Outer.Mid.Fixed": false,
		"Outer.Contract": false, "Outer.Contract.InInterface": false,
		"Outer.Kind": false, "Outer.Kind.InEnum": true,
		"Outer.Pair": false, "Outer.Pair.InRecord": true,
		"Outer.InAnonymous": false, "Outer.InMethod": false,
	}
	if !reflect.DeepEqual(inner, want) {
		t.Errorf("inner classes:\n got  %v\n want %v", inner, want)
	}
}
