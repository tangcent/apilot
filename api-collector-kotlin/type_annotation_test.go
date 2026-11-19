package kotlincollector

import (
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	collector "github.com/tangcent/apilot/api-collector"
	model "github.com/tangcent/apilot/api-model"
)

// typeAnnotationExpectation is the export of testdata/type-annotation: the
// fields of the request body and of the response, each as its kind. A type
// annotation inside a type argument, `List<@NotBlank(message = X) String>`, is
// no part of the type: the property is an array of strings, as it is without
// the annotation.
//
// The Java collector asserts the same tables for the same declarations written
// in Java (api-collector-java/type_annotation_test.go): keep the two in sync.
var typeAnnotationExpectation = map[string]string{
	"names": "array<string>",
	"urls":  "map<string,array<string>>",
	"ids":   "array<long>",
	"tags":  "array<string>",
	"item":  "object{labels}",
}

// typeAnnotationRequired is whether each field is required when only Bean
// Validation constraints make a field so. A constraint on the elements is not a
// constraint on the property: only `tags`, which carries @NotNull itself, is
// required.
var typeAnnotationRequired = map[string]bool{
	"names": false,
	"urls":  false,
	"ids":   false,
	"tags":  true,
	"item":  false,
}

func TestTypeAnnotationsInTypeArgumentsAreLeftOutOfTheExport(t *testing.T) {
	abs, err := filepath.Abs("testdata/type-annotation")
	if err != nil {
		t.Fatal(err)
	}
	c := New().(*KotlinCollector)
	endpoints, err := c.Collect(collector.CollectContext{
		SourceDir: abs,
		NoDeps:    true,
		Config:    map[string]string{collector.ConfigRequiredFields: collector.RequiredFieldsConstraints},
	})
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}
	if len(endpoints) != 1 || endpoints[0].RequestBody == nil || endpoints[0].RequestBody.Body == nil || endpoints[0].Response == nil {
		t.Fatalf("endpoints = %+v, want the save endpoint with a request body and a response", endpoints)
	}

	for what, body := range map[string]*model.ObjectModel{
		"request body": endpoints[0].RequestBody.Body,
		"response":     endpoints[0].Response.Body,
	} {
		if got := typeUseFieldKinds(body); !reflect.DeepEqual(got, typeAnnotationExpectation) {
			t.Errorf("%s:\n got  %v\n want %v", what, got, typeAnnotationExpectation)
		}
		required := map[string]bool{}
		for name, field := range body.Fields {
			required[name] = field.Required
		}
		if !reflect.DeepEqual(required, typeAnnotationRequired) {
			t.Errorf("%s required:\n got  %v\n want %v", what, required, typeAnnotationRequired)
		}
		if labels := body.Fields["item"].Model.Fields["labels"]; labels == nil || typeUseModelKind(labels.Model) != "array<string>" {
			t.Errorf("%s item.labels = %+v, want an array of strings", what, labels)
		}
	}
	if unresolved := c.Unresolved(); len(unresolved) != 0 {
		t.Errorf("unresolved = %v, want none", unresolved)
	}
}

// typeUseFieldKinds spells the fields of an object model as their kinds.
func typeUseFieldKinds(m *model.ObjectModel) map[string]string {
	out := map[string]string{}
	for name, field := range m.Fields {
		out[name] = typeUseModelKind(field.Model)
	}
	return out
}

func typeUseModelKind(m *model.ObjectModel) string {
	switch {
	case m == nil:
		return "nil"
	case m.IsArray():
		return "array<" + typeUseModelKind(m.Items) + ">"
	case m.IsMap():
		return "map<" + typeUseModelKind(m.KeyModel) + "," + typeUseModelKind(m.ValueModel) + ">"
	case m.IsObject():
		names := make([]string, 0, len(m.Fields))
		for name := range m.Fields {
			names = append(names, name)
		}
		sort.Strings(names)
		return "object{" + strings.Join(names, ",") + "}"
	}
	return m.TypeName
}
