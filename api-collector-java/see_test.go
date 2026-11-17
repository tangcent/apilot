package javacollector

import (
	"fmt"
	"path/filepath"
	"reflect"
	"testing"

	collector "github.com/tangcent/apilot/api-collector"
	"github.com/tangcent/apilot/api-collector-java/parser"
	"github.com/tangcent/apilot/api-collector-java/springmvc"
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
	checkSeeTables(t, endpoints, want, seeQueryExpectation)
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
// the options of each String field of the request body. The library's
// OrderState gives the options the project's own gives in testdata/see.
// PayChannel, decompiled from a JAR without sources, gives its constants
// without descriptions. A member reference, a library class that is not an
// enum, a class the library does not have, a name the source neither
// qualifies nor imports and a JDK class add nothing.
//
// The Kotlin collector asserts the same table for the same form written in
// Kotlin (api-collector-kotlin/see_test.go): keep the two in sync.
var seeDependencyExpectation = map[string][]string{
	"state":      seeExpectation["state"],
	"qualified":  seeExpectation["state"],
	"channel":    {"ALIPAY: ", "WECHAT: "},
	"member":     nil,
	"notAnEnum":  nil,
	"missing":    nil,
	"unimported": nil,
	"platform":   nil,
}

// seeDependencyQueryExpectation is the same form bound from request
// parameters. A field typed with the library's enum accepts its constants.
var seeDependencyQueryExpectation = map[string][]string{
	"state":      {"PENDING", "PAID", "CLOSED"},
	"qualified":  {"PENDING", "PAID", "CLOSED"},
	"channel":    {"ALIPAY", "WECHAT"},
	"member":     nil,
	"notAnEnum":  nil,
	"missing":    nil,
	"unimported": nil,
	"platform":   nil,
	"typed":      {"PENDING", "PAID", "CLOSED"},
}

// seeLibrarySources are the classes of the library testdata/see-dependency
// refers to, as the indexer returns them: OrderState from a sources JAR, and
// PayChannel decompiled from a JAR without one, so with no doc comments.
var seeLibrarySources = map[string]string{
	"com.example.lib.OrderState": `package com.example.lib;

/**
 * State of an order.
 */
public enum OrderState {
    /** Waiting for payment. */
    PENDING("P"),
    /**
     * Paid.
     *
     * Ships within a day.
     */
    PAID("D"),
    CLOSED("C");

    private final String code;

    OrderState(String code) {
        this.code = code;
    }

    public String getCode() {
        return code;
    }
}
`,
	"com.example.lib.PayChannel": `/*
 * Decompiled with CFR 0.152.
 */
package com.example.lib;

public enum PayChannel {
    ALIPAY("alipay"),
    WECHAT("wechat");

    private final String code;

    private PayChannel(String code) {
        this.code = code;
    }

    public String getCode() {
        return this.code;
    }
}
`,
	"com.example.lib.PageQuery": `package com.example.lib;

public class PageQuery {
    private int page;
    private int size;
}
`,
}

// seeLibrary answers dependency lookups from seeLibrarySources by qualified
// name, as the indexer does, and records each lookup.
type seeLibrary struct {
	t      *testing.T
	parser *parser.Parser
	asked  []string
}

func (l *seeLibrary) ResolveClass(name string) *parser.Class {
	l.asked = append(l.asked, name)
	source, ok := seeLibrarySources[name]
	if !ok {
		return nil
	}
	classes, err := l.parser.ParseSource([]byte(source))
	if err != nil {
		l.t.Fatalf("parsing %s: %v", name, err)
	}
	for i := range classes {
		if classes[i].Package+"."+classes[i].Name == name {
			return &classes[i]
		}
	}
	return nil
}

func TestSeeTagReadsAnEnumOfADependency(t *testing.T) {
	abs, err := filepath.Abs("testdata/see-dependency")
	if err != nil {
		t.Fatal(err)
	}
	p, err := parser.NewParser(parser.ParserOptions{})
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()
	results, err := p.ParseDirectory(abs)
	if err != nil {
		t.Fatal(err)
	}

	library := &seeLibrary{t: t, parser: p}
	unresolved := collector.NewUnresolvedSet()
	sm := springmvc.NewParser()
	sm.SetDependencyResolver(library)
	sm.SetUnresolved(unresolved)
	var endpoints []collector.ApiEndpoint
	for _, ctrl := range sm.ExtractControllers(results) {
		for _, ep := range ctrl.Endpoints {
			endpoints = append(endpoints, springmvcEndpointToAPI(ep, ctrl.Name))
		}
	}

	body := checkSeeTables(t, endpoints, seeDependencyExpectation, seeDependencyQueryExpectation)
	// The JSON view exports a field typed with an enum by the enum's name,
	// whether the project or a library declares it.
	if typed := body["typed"]; typed == nil || typed.Model.TypeName != "OrderState" || len(typed.Options) != 0 {
		t.Errorf("body field typed = %+v, want the enum's name without options", typed)
	}

	// The library is asked for each name once, however many fields refer to
	// it and in whichever view, a miss included. Only qualified names reach it.
	asked := map[string]int{}
	for _, name := range library.asked {
		asked[name]++
	}
	wantAsked := []string{"com.example.lib.OrderState", "com.example.lib.PayChannel", "com.example.lib.PageQuery", "com.example.lib.Missing"}
	for _, name := range wantAsked {
		if asked[name] != 1 {
			t.Errorf("%s was asked for %d times, want once", name, asked[name])
		}
	}
	if len(asked) != len(wantAsked) {
		t.Errorf("asked = %v, want only %v", library.asked, wantAsked)
	}
	// The `@see` lookups report nothing; the enum-typed field is reported by
	// the JSON view as before.
	for name := range unresolved.Counts() {
		if name != "OrderState" {
			t.Errorf("unresolved %s; unresolved = %v", name, unresolved.Counts())
		}
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
