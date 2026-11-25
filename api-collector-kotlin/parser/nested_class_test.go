package parser

import (
	"reflect"
	"testing"
)

// A class nested in another, inner or not, is a class of its own that says where
// it is nested. A companion object is not one: its constants belong to the class.
func TestParser_RecordsTheClassesADeclarationIsNestedIn(t *testing.T) {
	result := parseSource(t, "src/Outer.kt", `package com.example

class Outer {
    class Info {
        var ofOuter: String = ""
    }

    inner class Cell(var value: Int)

    interface Handler {
        class Impl
    }

    enum class State { OPEN, SHUT }

    object Registry

    data class Pair(val left: String) {
        class Info {
            var ofPair: String = ""
        }
    }

    companion object {
        const val VERSION = "v1"

        class Hidden
    }
}

class Other {
    class Info {
        var ofOther: String = ""
    }
}
`)
	got := map[string]string{}
	for _, c := range result.Classes {
		key := c.Name
		if c.Outer != "" {
			key = c.Outer + "." + c.Name
		}
		if _, dup := got[key]; dup {
			t.Fatalf("%s is declared twice: %+v", key, result.Classes)
		}
		got[key] = c.Outer
	}
	want := map[string]string{
		"Outer":              "",
		"Outer.Info":         "Outer",
		"Outer.Cell":         "Outer",
		"Outer.Handler":      "Outer",
		"Outer.Handler.Impl": "Outer.Handler",
		"Outer.State":        "Outer",
		"Outer.Registry":     "Outer",
		"Outer.Pair":         "Outer",
		"Outer.Pair.Info":    "Outer.Pair",
		"Other":              "",
		"Other.Info":         "Other",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("classes:\n got  %v\n want %v", got, want)
	}
	for _, c := range result.Classes {
		if c.Package != "com.example" {
			t.Errorf("%s has package %q", c.Name, c.Package)
		}
		if c.Name == "Outer" {
			// The companion's constant is a static field of the class.
			if len(c.Fields) != 1 || c.Fields[0].Name != "VERSION" || !c.Fields[0].IsConst {
				t.Errorf("fields of Outer = %+v, want the constant of the companion", c.Fields)
			}
		}
		if c.Name == "Info" && c.Outer == "Other" && (len(c.Fields) != 1 || c.Fields[0].Name != "ofOther") {
			t.Errorf("Other.Info fields = %+v, want its own", c.Fields)
		}
	}
}
