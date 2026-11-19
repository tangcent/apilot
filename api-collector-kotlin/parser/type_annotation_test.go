package parser

import (
	"reflect"
	"testing"
)

// An annotation inside a type argument belongs to the element type, not to the
// declaration that holds the type, so the type is spelled without it. The
// annotation's arguments may be anything an annotation takes: a constant, a
// string with commas and brackets, an array. A trailing `?` is the type's own
// nullability and stays out of the spelling, as it always has.
//
// The Java parser reads the same declarations the same way
// (api-collector-java/parser/type_annotation_test.go): keep the two in sync.
func TestLeavesTypeAnnotationsOutOfTypes(t *testing.T) {
	result := parseSource(t, "src/Form.kt", `package com.example

import javax.validation.constraints.NotNull

class Form(
    @field:NotNull
    val names: List<@NotBlank(message = Msg.CANNOT_BE_EMPTY) String>,
    val byKind: Map<String, List<@Patterns(regexps = [Re.HTTPS_URL, "a, b"]) String>>?,
    val ids: kotlin.collections.List<@NotNull Long>,
    val quoted: List<@Size(max = 3, message = "a (b), [c]") String?>,
    val items: Map<@NotBlank String, @NotNull @Valid Item>,
    val plain: List<String>,
) : Base<@NotBlank String>() {
    fun lookup(filter: List<@NotNull Long>, q: String): List<@NotBlank String> = TODO()
}
`)
	form := result.Classes[0]

	wantFields := map[string]string{
		"names":  "List<String>",
		"byKind": "Map<String,List<String>>",
		"ids":    "kotlin.collections.List<Long>",
		"quoted": "List<String>",
		"items":  "Map<String,Item>",
		"plain":  "List<String>",
	}
	if len(form.Fields) != len(wantFields) {
		t.Fatalf("fields = %+v", form.Fields)
	}
	for _, field := range form.Fields {
		if field.Type != wantFields[field.Name] {
			t.Errorf("property %s: type = %q, want %q", field.Name, field.Type, wantFields[field.Name])
		}
	}

	// The property keeps the annotations written on the property. The ones on
	// the elements are not the property's: @NotBlank on each name does not make
	// the list required.
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
			t.Errorf("property %s: annotations = %v, want %v", field.Name, names, want)
		}
	}

	method := form.Methods[0]
	if method.ReturnType != "List<String>" {
		t.Errorf("return type = %q, want List<String>", method.ReturnType)
	}
	wantParams := map[string]string{"filter": "List<Long>", "q": "String"}
	for _, param := range method.Parameters {
		if param.Type != wantParams[param.Name] {
			t.Errorf("parameter %s: type = %q, want %q", param.Name, param.Type, wantParams[param.Name])
		}
	}
	if form.SuperClass != "Base" || !reflect.DeepEqual(form.SuperClassTypeArgs, []string{"String"}) {
		t.Errorf("superclass = %q%v, want Base<String>", form.SuperClass, form.SuperClassTypeArgs)
	}
}
