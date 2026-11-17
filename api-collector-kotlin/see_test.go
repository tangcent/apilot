package kotlincollector

import (
	"fmt"
	"reflect"
	"testing"

	collector "github.com/tangcent/apilot/api-collector"
	model "github.com/tangcent/apilot/api-model"
)

// seeExpectation is the export of testdata/see: the options of each field of
// the request body, as "value: description". A String property whose KDoc
// says `@see OrderState` takes the constants of that enum, described by the
// KDoc of each entry. A member reference, a quoted string and a class that is
// not an enum add nothing, and values the property declares itself win.
//
// The Java collector asserts the same table for the same declarations written
// in Java (api-collector-java/see_test.go): keep the two in sync.
var seeExpectation = map[string][]string{
	"state":     {"PENDING: Waiting for payment.", "PAID: Paid.\n\nShips within a day.", "CLOSED: "},
	"qualified": {"PENDING: Waiting for payment.", "PAID: Paid.\n\nShips within a day.", "CLOSED: "},
	"member":    nil,
	"quoted":    nil,
	"notAnEnum": nil,
	"declared":  {"OPEN: ", "SHUT: "},
	// Kotlin exports a property typed with an enum as a string with the
	// enum's values; the entries' KDoc describes them.
	"typed": {"PENDING: Waiting for payment.", "PAID: Paid.\n\nShips within a day.", "CLOSED: "},
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
	"typed":     {"PENDING", "PAID", "CLOSED"},
}

func TestSeeTagGivesAFieldTheConstantsOfAnEnum(t *testing.T) {
	endpoints, _ := collectDir(t, "testdata/see")

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
	for name, want := range seeExpectation {
		field := fields[name]
		if field == nil {
			t.Errorf("body field %s missing", name)
			continue
		}
		if got := describeOptions(field.Options); !reflect.DeepEqual(got, want) {
			t.Errorf("body field %s options:\n got  %q\n want %q", name, got, want)
		}
		if field.Model == nil || field.Model.TypeName != model.JsonTypeString {
			t.Errorf("body field %s = %+v, want a string", name, field.Model)
		}
	}

	params := map[string]collector.ApiParameter{}
	for _, p := range page.Parameters {
		params[p.Name] = p
	}
	for name, want := range seeQueryExpectation {
		if got := params[name].Enum; !reflect.DeepEqual(got, want) {
			t.Errorf("query parameter %s enum = %q, want %q", name, got, want)
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
