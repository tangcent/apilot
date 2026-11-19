package kotlincollector

import (
	"fmt"
	"path/filepath"
	"reflect"
	"testing"

	collector "github.com/tangcent/apilot/api-collector"
	ktparser "github.com/tangcent/apilot/api-collector-kotlin/parser"
	"github.com/tangcent/apilot/api-collector-kotlin/resolver"
	"github.com/tangcent/apilot/api-collector-kotlin/springmvc"
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
	checkSeeTables(t, endpoints, seeExpectation, seeQueryExpectation)
}

// checkSeeTables asserts the options of the string fields in want on the
// request body of the create endpoint, and the values of the query parameters
// in wantQuery on the page endpoint. It returns the fields of the body.
func checkSeeTables(t *testing.T, endpoints []collector.ApiEndpoint, want, wantQuery map[string][]string) map[string]*model.FieldModel {
	t.Helper()
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
	for name, wantEnum := range wantQuery {
		if got := params[name].Enum; !reflect.DeepEqual(got, wantEnum) {
			t.Errorf("query parameter %s enum = %q, want %q", name, got, wantEnum)
		}
	}
	return fields
}

// seeDependencyExpectation is the export of testdata/see-dependency, a form
// whose `@see` tags point at classes of a library rather than of the project:
// the options of each String property of the request body. The library's
// OrderState, a Kotlin source, gives the options the project's own gives in
// testdata/see. PayChannel, decompiled from a JAR without sources, gives its
// constants without descriptions, and so does Region, a Kotlin enum decompiled
// from one: the decompiler prints that as a class rather than an enum. A
// member reference, a library class that is not an enum, a class the library
// does not have, a name the source neither qualifies nor imports and a JDK
// class add nothing.
//
// The Java collector asserts the same table for the same form written in Java
// (api-collector-java/see_test.go): keep the two in sync.
var seeDependencyExpectation = map[string][]string{
	"state":      seeExpectation["state"],
	"qualified":  seeExpectation["state"],
	"channel":    {"ALIPAY: ", "WECHAT: "},
	"region":     {"NORTH: ", "SOUTH: "},
	"member":     nil,
	"notAnEnum":  nil,
	"missing":    nil,
	"unimported": nil,
	"platform":   nil,
}

// seeDependencyQueryExpectation is the same form bound from request
// parameters. A property typed with the library's enum accepts its constants.
var seeDependencyQueryExpectation = map[string][]string{
	"state":      {"PENDING", "PAID", "CLOSED"},
	"qualified":  {"PENDING", "PAID", "CLOSED"},
	"channel":    {"ALIPAY", "WECHAT"},
	"region":     {"NORTH", "SOUTH"},
	"member":     nil,
	"notAnEnum":  nil,
	"missing":    nil,
	"unimported": nil,
	"platform":   nil,
	"typed":      {"PENDING", "PAID", "CLOSED"},
	"kotlinEnum": {"NORTH", "SOUTH"},
}

func TestSeeTagReadsAnEnumOfADependency(t *testing.T) {
	p, err := ktparser.NewParser()
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()
	dir, _ := filepath.Abs("testdata/see-dependency")
	results, err := p.ParseDirectory(dir)
	if err != nil {
		t.Fatal(err)
	}

	typeResolver := resolver.New(flattenClasses(results))
	library := &libraryResolver{t: t}
	typeResolver.SetDependencyResolver(library)
	sm := springmvc.NewParser(typeResolver)
	var endpoints []collector.ApiEndpoint
	for _, ctrl := range sm.ExtractControllers(results) {
		for _, ep := range ctrl.Endpoints {
			endpoints = append(endpoints, springMVCEndpointToAPI(ep, ctrl.Name))
		}
	}

	body := checkSeeTables(t, endpoints, seeDependencyExpectation, seeDependencyQueryExpectation)
	// The JSON view exports a property typed with an enum as a string of its
	// entries, whether the project or a library declares it (see
	// enum_test.go).
	if typed := body["typed"]; typed == nil || typed.Model.TypeName != model.JsonTypeString || !reflect.DeepEqual(describeOptions(typed.Options), seeExpectation["state"]) {
		t.Errorf("body field typed = %+v, want a string of the enum's entries", typed)
	}
	// A Kotlin enum decompiled to a class is an enum all the same: not an
	// object of its properties.
	if typed := body["kotlinEnum"]; typed == nil || typed.Model.TypeName != model.JsonTypeString || typed.Model.Kind != model.KindSingle || !reflect.DeepEqual(describeOptions(typed.Options), []string{"NORTH: ", "SOUTH: "}) {
		t.Errorf("body field kotlinEnum = %+v, want a string of the enum's entries", typed)
	}

	// The library is asked for each name once by each view, however many
	// fields refer to it, a miss included: the JSON view and the binding view
	// keep a type table each. Only qualified names reach it.
	asked := map[string]int{}
	for _, name := range library.queried {
		asked[name]++
	}
	wantAsked := []string{"com.example.lib.OrderState", "com.example.lib.PayChannel", "com.example.lib.Region", "com.example.lib.PageQuery", "com.example.lib.Missing"}
	for _, name := range wantAsked {
		if asked[name] != 2 {
			t.Errorf("%s was asked for %d times, want once by each view", name, asked[name])
		}
	}
	if len(asked) != len(wantAsked) {
		t.Errorf("asked = %v, want only %v", library.queried, wantAsked)
	}
	// The `@see` lookups report nothing, and the enum-typed properties are
	// strings.
	if unresolved := typeResolver.Unresolved(); len(unresolved) != 0 {
		t.Errorf("unresolved = %v, want none", unresolved)
	}
	if warnings := sm.Warnings(); len(warnings) != 0 {
		t.Errorf("warnings = %q", warnings)
	}
}

func describeOptions(options []model.FieldOption) []string {
	var out []string
	for _, o := range options {
		out = append(out, fmt.Sprintf("%v: %s", o.Value, o.Desc))
	}
	return out
}
