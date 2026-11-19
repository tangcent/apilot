package parser

import (
	"reflect"
	"testing"
)

// fieldSummary is what a test compares of a field: its name, its type and
// whether it is a String constant, as `name type [constant]`.
type fieldSummary struct {
	Name     string
	Type     string
	Constant string
}

func summarizeFields(fields []Field) []fieldSummary {
	var out []fieldSummary
	for _, f := range fields {
		out = append(out, fieldSummary{f.Name, f.Type, f.Constant})
	}
	return out
}

// `private String a, b;` declares two fields. Only the last one used to be
// read, so `a` was left out of its class without a word.
func TestSeveralDeclaratorsAreSeveralFields(t *testing.T) {
	p, err := NewParser(ParserOptions{})
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()
	classes, err := p.ParseSource([]byte(`package com.example;

import java.util.List;

public class Holder {
    private String a, b;
    private int x = 1, y[], z[][] = null;
    private List<String> left = null, right;
    private int plain;
    public static final String ID = "id", NAME = "name", ALIAS = ID, JOINED = "a" + "b";
    private static final String KEY = "key", KEYS[] = {"a"}, OTHER;
}
`))
	if err != nil {
		t.Fatal(err)
	}

	want := []fieldSummary{
		{Name: "a", Type: "String"},
		{Name: "b", Type: "String"},
		{Name: "x", Type: "int"},
		{Name: "y", Type: "int[]"},
		{Name: "z", Type: "int[][]"},
		{Name: "left", Type: "List<String>"},
		{Name: "right", Type: "List<String>"},
		{Name: "plain", Type: "int"},
		{Name: "ID", Type: "String", Constant: `"id"`},
		{Name: "NAME", Type: "String", Constant: `"name"`},
		{Name: "ALIAS", Type: "String", Constant: "ID"},
		{Name: "JOINED", Type: "String"},
		{Name: "KEY", Type: "String", Constant: `"key"`},
		{Name: "KEYS", Type: "String[]"},
		{Name: "OTHER", Type: "String"},
	}
	if got := summarizeFields(classes[0].Fields); !reflect.DeepEqual(got, want) {
		t.Errorf("fields:\n got  %v\n want %v", got, want)
	}

	// An initializer belongs to the declarator it follows.
	initialized := map[string]bool{}
	for _, f := range classes[0].Fields {
		initialized[f.Name] = f.HasInitializer
	}
	for name, want := range map[string]bool{
		"a": false, "b": false, "x": true, "y": false, "z": true, "left": true, "right": false,
		"ID": true, "NAME": true, "KEYS": true, "OTHER": false,
	} {
		if initialized[name] != want {
			t.Errorf("%s: HasInitializer = %v, want %v", name, initialized[name], want)
		}
	}
}

// The modifiers, the annotations and the doc comment stand before the whole
// declaration, so every field it declares has them.
func TestDeclaratorsShareTheDeclaration(t *testing.T) {
	p, err := NewParser(ParserOptions{})
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()
	classes, err := p.ParseSource([]byte(`package com.example;

public class Holder {
    /** The name of a person. */
    @NotBlank
    @Size(max = 20)
    private final String firstName, lastName;

    public static int count, limit = 3;

    private String loner;
}
`))
	if err != nil {
		t.Fatal(err)
	}
	byName := map[string]Field{}
	for _, f := range classes[0].Fields {
		byName[f.Name] = f
	}

	for _, name := range []string{"firstName", "lastName"} {
		f, ok := byName[name]
		if !ok {
			t.Fatalf("%s is not a field of Holder: %v", name, classes[0].Fields)
		}
		if f.JavaDoc != "The name of a person." {
			t.Errorf("%s: doc = %q", name, f.JavaDoc)
		}
		var annotations []string
		for _, a := range f.Annotations {
			annotations = append(annotations, a.Name)
		}
		if want := []string{"NotBlank", "Size"}; !reflect.DeepEqual(annotations, want) {
			t.Errorf("%s: annotations = %v, want %v", name, annotations, want)
		}
		if len(f.Annotations) == 2 && f.Annotations[1].Params["max"] != "20" {
			t.Errorf("%s: @Size params = %v", name, f.Annotations[1].Params)
		}
		if !f.IsFinal || f.IsStatic {
			t.Errorf("%s: final = %v, static = %v, want final and not static", name, f.IsFinal, f.IsStatic)
		}
	}
	for _, name := range []string{"count", "limit"} {
		if f := byName[name]; !f.IsStatic || f.IsFinal {
			t.Errorf("%s: static = %v, final = %v, want static and not final", name, f.IsStatic, f.IsFinal)
		}
	}
	// What stands before another declaration is not carried over.
	if f := byName["loner"]; f.JavaDoc != "" || len(f.Annotations) != 0 {
		t.Errorf("loner: doc = %q, annotations = %v, want none", f.JavaDoc, f.Annotations)
	}
}

// The constants of an interface are declared the same way.
func TestSeveralInterfaceConstantsInOneDeclaration(t *testing.T) {
	p, err := NewParser(ParserOptions{})
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()
	classes, err := p.ParseSource([]byte(`package com.example;

public interface Keys {
    String PAGE = "page_no", SIZE = "page_size";
}
`))
	if err != nil {
		t.Fatal(err)
	}
	want := []fieldSummary{
		{Name: "PAGE", Type: "String", Constant: `"page_no"`},
		{Name: "SIZE", Type: "String", Constant: `"page_size"`},
	}
	if got := summarizeFields(classes[0].Fields); !reflect.DeepEqual(got, want) {
		t.Errorf("constants:\n got  %v\n want %v", got, want)
	}
}

// A varargs parameter is an array of the type it is declared with. It used
// to be left out of the parameters of its method, together with its
// annotations.
func TestVarargsParametersAreArrays(t *testing.T) {
	p, err := NewParser(ParserOptions{})
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()
	classes, err := p.ParseSource([]byte(`package com.example;

import java.util.List;

public class Holder {
    public void names(String... ids) {}
    public void mixed(int page, @RequestParam("q") final String... terms) {}
    public void nested(int[]... grid) {}
    public void generic(List<String>... lists) {}
    public void qualified(java.time.LocalDate... days) {}
    public void spaced(String ... spaced) {}
    public void annotated(@RequestParam(required = false) @Deprecated Long... ids) {}
}
`))
	if err != nil {
		t.Fatal(err)
	}
	type param struct{ Name, Type string }
	got := map[string][]param{}
	for _, m := range classes[0].Methods {
		for _, pm := range m.Parameters {
			got[m.Name] = append(got[m.Name], param{pm.Name, pm.Type})
		}
	}
	want := map[string][]param{
		"names":     {{"ids", "String[]"}},
		"mixed":     {{"page", "int"}, {"terms", "String[]"}},
		"nested":    {{"grid", "int[][]"}},
		"generic":   {{"lists", "List<String>[]"}},
		"qualified": {{"days", "java.time.LocalDate[]"}},
		"spaced":    {{"spaced", "String[]"}},
		"annotated": {{"ids", "Long[]"}},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("parameters:\n got  %v\n want %v", got, want)
	}

	// The annotations of the parameter are kept.
	for _, m := range classes[0].Methods {
		switch m.Name {
		case "mixed":
			if len(m.Parameters) != 2 {
				continue
			}
			terms := m.Parameters[1]
			if len(terms.Annotations) != 1 || terms.Annotations[0].Name != "RequestParam" || terms.Annotations[0].Params["value"] != "q" {
				t.Errorf("terms: annotations = %v, want @RequestParam(\"q\")", terms.Annotations)
			}
		case "annotated":
			if len(m.Parameters) != 1 {
				continue
			}
			var names []string
			for _, a := range m.Parameters[0].Annotations {
				names = append(names, a.Name)
			}
			if want := []string{"RequestParam", "Deprecated"}; !reflect.DeepEqual(names, want) {
				t.Errorf("ids: annotations = %v, want %v", names, want)
			}
		}
	}
}

// A record component can be varargs too; it is an array field.
func TestVarargsRecordComponent(t *testing.T) {
	p, err := NewParser(ParserOptions{})
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()
	classes, err := p.ParseSource([]byte(`package com.example;

public record Batch(String name, @NotNull int... counts) {}
`))
	if err != nil {
		t.Fatal(err)
	}
	want := []fieldSummary{
		{Name: "name", Type: "String"},
		{Name: "counts", Type: "int[]"},
	}
	if got := summarizeFields(classes[0].Fields); !reflect.DeepEqual(got, want) {
		t.Errorf("components:\n got  %v\n want %v", got, want)
	}
	if fields := classes[0].Fields; len(fields) == 2 && (len(fields[1].Annotations) != 1 || fields[1].Annotations[0].Name != "NotNull") {
		t.Errorf("counts: annotations = %v, want @NotNull", fields[1].Annotations)
	}
}
