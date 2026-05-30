package doc

import (
	"reflect"
	"testing"

	"github.com/tangcent/apilot/api-collector-java/parser"
)

func TestCleanJavaDoc(t *testing.T) {
	raw := `/**
 * Fetch user.
 *
 * More details.
 * @param id user id
 * @return user
 */`

	got := CleanJavaDoc(raw)
	want := "Fetch user.\nMore details."
	if got != want {
		t.Fatalf("Expected cleaned JavaDoc %q, got %q", want, got)
	}
}

func TestParseJavaDocParams(t *testing.T) {
	raw := `/**
 * Fetch user.
 * @param id user id
 * @param status current status
 */`

	got := ParseJavaDocParams(raw)
	want := map[string]string{
		"id":     "user id",
		"status": "current status",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Expected params %#v, got %#v", want, got)
	}
}

func TestEndpointDescriptionPrefersOperation(t *testing.T) {
	annotations := []parser.Annotation{
		{Name: "Operation", Params: map[string]string{
			"summary":     "Fetch user",
			"description": "Fetch user by id",
		}},
	}

	got := EndpointDescription(annotations, "JavaDoc fallback")
	want := "Fetch user\n\nFetch user by id"
	if got != want {
		t.Fatalf("Expected endpoint description %q, got %q", want, got)
	}
}

func TestParameterDocumentationPrefersAnnotation(t *testing.T) {
	required := true
	got := ParameterDocumentation([]parser.Annotation{
		{Name: "Parameter", Params: map[string]string{
			"description": "User id",
			"example":     "42",
			"required":    "true",
		}},
	}, "fallback id")

	want := FieldDocumentation{
		Description: "User id",
		Example:     "42",
		Required:    &required,
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Expected parameter documentation %#v, got %#v", want, got)
	}
}

func TestFieldDocumentationPrecedence(t *testing.T) {
	required := true
	got := FieldDocumentationFor([]parser.Annotation{
		{Name: "ApiModelProperty", Params: map[string]string{
			"value":    "Legacy name",
			"required": "false",
		}},
		{Name: "Schema", Params: map[string]string{
			"description":     "Display name",
			"example":         "Ada",
			"required":        "true",
			"allowableValues": "Ada,Grace",
			"defaultValue":    "Ada",
		}},
	}, "fallback name")

	want := FieldDocumentation{
		Description: "Display name",
		Example:     "Ada",
		Required:    &required,
		Default:     "Ada",
		Enum:        []string{"Ada", "Grace"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Expected field documentation %#v, got %#v", want, got)
	}
}

func TestFieldDocumentationFallsBackToJavaDoc(t *testing.T) {
	got := FieldDocumentationFor(nil, "fallback field")
	want := FieldDocumentation{Description: "fallback field"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Expected JavaDoc fallback %#v, got %#v", want, got)
	}
}
