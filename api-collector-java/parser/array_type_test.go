package parser

import (
	"reflect"
	"testing"
)

// A field, a parameter or a return type declared as an array, or by a name
// qualified with its package or its enclosing class, is read like any other:
// it used to be skipped, which dropped the field from the export without a
// word.
func TestArrayAndQualifiedTypes(t *testing.T) {
	p, err := NewParser(ParserOptions{})
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()
	classes, err := p.ParseSource([]byte(`package com.example;

import java.util.List;

public class Holder {
    private String[] tags;
    private int[] ids;
    private Status[] states;
    private List<String>[] groups;
    private int[][] grid;
    private String [] spaced;
    private String legacy[];
    private int[] mixed[];
    private Order.Status nested;
    private Order.Status[] nestedStates;
    private java.time.LocalDate day;
    private java.time.LocalDate[] days;
    private java.util.List<String> names;
    @NotNull
    private String[] annotated;
    public static final String KEYS[] = {"a", "b"};
    private int total = limit;

    public String[] tags(String[] in, String legacy[], Order.Status status, java.time.LocalDate day, int grid[][]) {
        return in;
    }

    public Order.Status status() { return null; }

    public List<String>[] groups() { return null; }

    public int[] ids() { return null; }

    public java.util.Map<String, String> map() { return null; }
}
`))
	if err != nil {
		t.Fatal(err)
	}
	holder := classes[0]

	wantFields := map[string]string{
		"tags":         "String[]",
		"ids":          "int[]",
		"states":       "Status[]",
		"groups":       "List<String>[]",
		"grid":         "int[][]",
		"spaced":       "String[]",
		"legacy":       "String[]",
		"mixed":        "int[][]",
		"nested":       "Order.Status",
		"nestedStates": "Order.Status[]",
		"day":          "java.time.LocalDate",
		"days":         "java.time.LocalDate[]",
		"names":        "java.util.List<String>",
		"annotated":    "String[]",
		"KEYS":         "String[]",
		"total":        "int",
	}
	gotFields := map[string]string{}
	for _, field := range holder.Fields {
		gotFields[field.Name] = field.Type
	}
	if !reflect.DeepEqual(gotFields, wantFields) {
		t.Errorf("fields:\n got  %v\n want %v", gotFields, wantFields)
	}

	for _, field := range holder.Fields {
		// An array of strings is not a String constant an annotation could name.
		if field.Name == "KEYS" && field.Constant != "" {
			t.Errorf("KEYS constant = %q, want none", field.Constant)
		}
	}

	wantReturns := map[string]string{
		"tags":   "String[]",
		"status": "Order.Status",
		"groups": "List<String>[]",
		"ids":    "int[]",
		"map":    "java.util.Map<String, String>",
	}
	gotReturns := map[string]string{}
	for _, method := range holder.Methods {
		gotReturns[method.Name] = method.ReturnType
	}
	if !reflect.DeepEqual(gotReturns, wantReturns) {
		t.Errorf("return types:\n got  %v\n want %v", gotReturns, wantReturns)
	}

	wantParams := map[string]string{
		"in":     "String[]",
		"legacy": "String[]",
		"status": "Order.Status",
		"day":    "java.time.LocalDate",
		"grid":   "int[][]",
	}
	gotParams := map[string]string{}
	for _, param := range holder.Methods[0].Parameters {
		gotParams[param.Name] = param.Type
	}
	if !reflect.DeepEqual(gotParams, wantParams) {
		t.Errorf("parameters:\n got  %v\n want %v", gotParams, wantParams)
	}
}

// The components of a record are read as parameters, so the same spellings
// reach the fields of a record.
func TestArrayAndQualifiedRecordComponents(t *testing.T) {
	p, err := NewParser(ParserOptions{})
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()
	classes, err := p.ParseSource([]byte(`package com.example;

public record Form(String[] tags, java.time.LocalDate day, Order.Status[] states) {}
`))
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]string{}
	for _, field := range classes[0].Fields {
		got[field.Name] = field.Type
	}
	want := map[string]string{"tags": "String[]", "day": "java.time.LocalDate", "states": "Order.Status[]"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("components:\n got  %v\n want %v", got, want)
	}
}

// An array among the type arguments of a type is spelled as it is anywhere
// else: the brackets follow the element without a space, whatever the source
// puts between them or in front of them. The resolver reads the brackets off
// the end of the spelling, so `String []` was the array of an element called
// `String ` and `int [ ]` was the name of a type that no declaration has.
func TestArraySpellingInsideTypeArguments(t *testing.T) {
	p, err := NewParser(ParserOptions{})
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()
	classes, err := p.ParseSource([]byte(`package com.example;

import java.util.List;
import java.util.Map;

public class Holder<T> extends Base<Order [ ], List<String []>> {
    private List<String []> names;
    private List<int [ ]> ids;
    private List<String @NotNull []> annotated;
    private List<@NotNull String [ ][ ]> grid;
    private Map<String, List<Order [ ] [ ]>> byKind;
    private Map<String, T [ ]> byName;
    private List<? extends Order [ ]> bounded;
    private List<String []> [ ] outer;
    private List<java.lang.String [ ]> qualified;
    private List<String[]> plain;
    private String @NotNull [] @Valid [] marked;

    public List<String []> names(List<int [ ]> in, Map<String, T [ ]> byName, List<String @NotNull []> annotated, List<String []> legacy[], String @NotNull [] @Valid [] marked) {
        return null;
    }

    public Map<String, int [ ]> sizes() { return null; }

    public String @NotNull [] tags() { return null; }
}
`))
	if err != nil {
		t.Fatal(err)
	}
	holder := classes[0]

	wantFields := map[string]string{
		"names":     "List<String[]>",
		"ids":       "List<int[]>",
		"annotated": "List<String[]>",
		"grid":      "List<String[][]>",
		"byKind":    "Map<String, List<Order[][]>>",
		"byName":    "Map<String, T[]>",
		"bounded":   "List<? extends Order[]>",
		"outer":     "List<String[]>[]",
		"qualified": "List<java.lang.String[]>",
		"plain":     "List<String[]>",
		"marked":    "String[][]",
	}
	gotFields := map[string]string{}
	for _, field := range holder.Fields {
		gotFields[field.Name] = field.Type
	}
	if !reflect.DeepEqual(gotFields, wantFields) {
		t.Errorf("fields:\n got  %v\n want %v", gotFields, wantFields)
	}

	wantReturns := map[string]string{"names": "List<String[]>", "sizes": "Map<String, int[]>", "tags": "String[]"}
	gotReturns := map[string]string{}
	for _, method := range holder.Methods {
		gotReturns[method.Name] = method.ReturnType
	}
	if !reflect.DeepEqual(gotReturns, wantReturns) {
		t.Errorf("return types:\n got  %v\n want %v", gotReturns, wantReturns)
	}

	wantParams := map[string]string{
		"in":        "List<int[]>",
		"byName":    "Map<String, T[]>",
		"annotated": "List<String[]>",
		"legacy":    "List<String[]>[]",
		"marked":    "String[][]",
	}
	gotParams := map[string]string{}
	for _, param := range holder.Methods[0].Parameters {
		gotParams[param.Name] = param.Type
	}
	if !reflect.DeepEqual(gotParams, wantParams) {
		t.Errorf("parameters:\n got  %v\n want %v", gotParams, wantParams)
	}

	// An array is a type argument of the superclass like any other: it binds
	// the superclass's parameter at its position.
	if wantArgs := []string{"Order[]", "List<String[]>"}; holder.SuperClass != "Base" || !reflect.DeepEqual(holder.SuperClassTypeArgs, wantArgs) {
		t.Errorf("superclass = %q%v, want Base%v", holder.SuperClass, holder.SuperClassTypeArgs, wantArgs)
	}
}

// The components of a record are read as parameters, so the same spellings
// reach the fields of a record.
func TestArraySpellingInsideTypeArgumentsOfRecordComponents(t *testing.T) {
	p, err := NewParser(ParserOptions{})
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()
	classes, err := p.ParseSource([]byte(`package com.example;

public record Form(List<String []> names, Map<String, int [ ]> sizes) {}
`))
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]string{}
	for _, field := range classes[0].Fields {
		got[field.Name] = field.Type
	}
	want := map[string]string{"names": "List<String[]>", "sizes": "Map<String, int[]>"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("components:\n got  %v\n want %v", got, want)
	}
}
