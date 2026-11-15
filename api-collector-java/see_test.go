package javacollector

import (
	"fmt"
	"path/filepath"
	"reflect"
	"testing"

	collector "github.com/tangcent/apilot/api-collector"
	model "github.com/tangcent/apilot/api-model"
)

// seeExpectation is the export of testdata/see: the options of each field of
// the request body, as "value: description". A String field whose doc says
// `@see OrderState` takes the constants of that enum, described by their doc
// comments. A member reference, a quoted string and a class that is not an
// enum add nothing, and values the field declares itself win.
//
// The Kotlin collector asserts the same table for the same declarations
// written in Kotlin (api-collector-kotlin/see_test.go): keep the two in sync.
var seeExpectation = map[string][]string{
	"state":     {"PENDING: Waiting for payment.", "PAID: Paid.\n\nShips within a day.", "CLOSED: "},
	"qualified": {"PENDING: Waiting for payment.", "PAID: Paid.\n\nShips within a day.", "CLOSED: "},
	"member":    nil,
	"quoted":    nil,
	"notAnEnum": nil,
	"declared":  {"OPEN: ", "SHUT: "},
}

// seeQueryExpectation is the same form bound from request parameters: each
// option is a value the parameter accepts.
var seeQueryExpectation = map[string][]string{
	"state":     {"PENDING", "PAID", "CLOSED"},
	"qualified": {"PENDING", "PAID", "CLOSED"},
	"member":    nil,
	"quoted":    nil,
	"notAnEnum": nil,
	"declared":  {"OPEN", "SHUT"},
}

func TestSeeTagGivesAFieldTheConstantsOfAnEnum(t *testing.T) {
	abs, err := filepath.Abs("testdata/see")
	if err != nil {
		t.Fatal(err)
	}
	endpoints, err := New().Collect(collector.CollectContext{SourceDir: abs, NoDeps: true})
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}
	checkSeeExport(t, endpoints, nil)
}

// checkSeeExport asserts seeExpectation on the create endpoint and
// seeQueryExpectation on the page endpoint, plus the fields of extra.
func checkSeeExport(t *testing.T, endpoints []collector.ApiEndpoint, extra map[string][]string) {
	t.Helper()
	want := map[string][]string{}
	for name, options := range seeExpectation {
		want[name] = options
	}
	for name, options := range extra {
		want[name] = options
	}

	var create, page *collector.ApiEndpoint
	for i := range endpoints {
		switch endpoints[i].Name {
		case "create":
			create = &endpoints[i]
		case "page":
			page = &endpoints[i]
		}
	}
	if create == nil || page == nil {
		t.Fatalf("endpoints = %+v, want create and page", endpoints)
	}

	if create.RequestBody == nil || create.RequestBody.Body == nil {
		t.Fatalf("create has no request body")
	}
	fields := create.RequestBody.Body.Fields
	for name, wantOptions := range want {
		field := fields[name]
		if field == nil {
			t.Errorf("body field %s missing", name)
			continue
		}
		if got := describeOptions(field.Options); !reflect.DeepEqual(got, wantOptions) {
			t.Errorf("body field %s options:\n got  %q\n want %q", name, got, wantOptions)
		}
		if field.Model == nil || field.Model.TypeName != model.JsonTypeString {
			t.Errorf("body field %s = %+v, want a string", name, field.Model)
		}
	}

	params := map[string]collector.ApiParameter{}
	for _, p := range page.Parameters {
		params[p.Name] = p
	}
	for name, wantEnum := range seeQueryExpectation {
		if got := params[name].Enum; !reflect.DeepEqual(got, wantEnum) {
			t.Errorf("query parameter %s enum = %q, want %q", name, got, wantEnum)
		}
	}
}

func describeOptions(options []model.FieldOption) []string {
	var out []string
	for _, o := range options {
		out = append(out, fmt.Sprintf("%v: %s", o.Value, o.Desc))
	}
	return out
}
