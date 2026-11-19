package parser

import (
	"reflect"
	"testing"
)

// An annotation inside a type argument belongs to the element type, not to the
// declaration that holds the type, so the type is spelled without it. The
// annotation's arguments may be anything an annotation takes: a constant, a
// string with commas and brackets, an array.
//
// The Kotlin parser reads the same declarations the same way
// (api-collector-kotlin/parser/type_annotation_test.go): keep the two in sync.
func TestParser_LeavesTypeAnnotationsOutOfTypes(t *testing.T) {
	p, err := NewParser(ParserOptions{})
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()
	classes, err := p.ParseSource([]byte(`package com.example;

import java.util.List;
import java.util.Map;

public class Form extends Base<@NotBlank String> {
    @NotNull
    private List<@NotBlank(message = Msg.CANNOT_BE_EMPTY) String> names;
    private Map<String, List<@Patterns(regexps = {Re.HTTPS_URL, "a, b"}) String>> byKind;
    private java.util.List<@NotNull Long> ids;
    private List<@Size(max = 3, message = "a (b), [c]") String> quoted;
    private List<? extends @NotNull Number> bounded;
    private Map<@NotBlank String, @NotNull @Valid Item> items;
    private List<String> plain;

    public List<@NotBlank String> lookup(List<@NotNull Long> filter, @RequestParam("q") String q, String @NotEmpty [] names) {
        return null;
    }
}
`))
	if err != nil {
		t.Fatal(err)
	}
	form := classes[0]

	wantFields := map[string]string{
		"names":   "List<String>",
		"byKind":  "Map<String, List<String>>",
		"ids":     "java.util.List<Long>",
		"quoted":  "List<String>",
		"bounded": "List<? extends Number>",
		"items":   "Map<String, Item>",
		"plain":   "List<String>",
	}
	if len(form.Fields) != len(wantFields) {
		t.Fatalf("fields = %+v", form.Fields)
	}
	for _, field := range form.Fields {
		if field.Type != wantFields[field.Name] {
			t.Errorf("field %s: type = %q, want %q", field.Name, field.Type, wantFields[field.Name])
		}
	}

	// The field keeps the annotations written on the field. The ones on the
	// elements are not the field's: @NotBlank on each name does not make the
	// list required.
	for _, field := range form.Fields {
		var names []string
		for _, ann := range field.Annotations {
			names = append(names, ann.Name)
		}
		var want []string
		if field.Name == "names" {
			want = []string{"NotNull"}
		}
		if !reflect.DeepEqual(names, want) {
			t.Errorf("field %s: annotations = %v, want %v", field.Name, names, want)
		}
	}

	method := form.Methods[0]
	if method.ReturnType != "List<String>" {
		t.Errorf("return type = %q, want List<String>", method.ReturnType)
	}
	wantParams := map[string]string{"filter": "List<Long>", "q": "String", "names": "String[]"}
	for _, param := range method.Parameters {
		if param.Type != wantParams[param.Name] {
			t.Errorf("parameter %s: type = %q, want %q", param.Name, param.Type, wantParams[param.Name])
		}
	}
	if form.SuperClass != "Base" || !reflect.DeepEqual(form.SuperClassTypeArgs, []string{"String"}) {
		t.Errorf("superclass = %q%v, want Base<String>", form.SuperClass, form.SuperClassTypeArgs)
	}
}
