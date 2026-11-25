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
// without descriptions, and so does Region, a Kotlin enum decompiled from one:
// CFR prints that as a class rather than an enum. A member reference, a
// library class that is not an enum, a class the library does not have, a name
// the source neither qualifies nor imports and a JDK class add nothing.
//
// The Kotlin collector asserts the same table for the same form written in
// Kotlin (api-collector-kotlin/see_test.go): keep the two in sync.
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
// parameters. A field typed with the library's enum accepts its constants.
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

// seeLibrarySources are the classes of the library testdata/see-dependency
// refers to, as the indexer returns them: OrderState from a sources JAR, and
// PayChannel and Region decompiled from a JAR without one, so with no doc
// comments. The decompiler prints a Java enum as an enum, and a Kotlin enum as
// a class that extends Enum.
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
	"com.example.lib.Region": `/*
 * Decompiled with CFR 0.152.
 */
package com.example.lib;

import kotlin.Metadata;
import kotlin.enums.EnumEntries;
import kotlin.enums.EnumEntriesKt;
import org.jetbrains.annotations.NotNull;

@Metadata(mv={1, 9, 0}, k=1, xi=48)
public final class Region
extends Enum<Region> {
    public static final /* enum */ Region NORTH = new Region("n");
    public static final /* enum */ Region SOUTH = new Region("s");
    private static final /* synthetic */ Region[] $VALUES;
    private static final /* synthetic */ EnumEntries $ENTRIES;
    @NotNull
    private final String code;

    private Region(String code) {
        this.code = code;
    }

    @NotNull
    public final String getCode() {
        return this.code;
    }

    public static Region[] values() {
        return (Region[])$VALUES.clone();
    }

    public static Region valueOf(String value) {
        return Enum.valueOf(Region.class, value);
    }

    @NotNull
    public static EnumEntries<Region> getEntries() {
        return $ENTRIES;
    }

    static {
        $VALUES = regionArray = new Region[]{Region.NORTH, Region.SOUTH};
        $ENTRIES = EnumEntriesKt.enumEntries((Enum[])$VALUES);
    }
}
`,
	"com.example.lib.PageQuery": `package com.example.lib;

public class PageQuery {
    private int page;
    private int size;
}
`,
	// Currency and Tier declare @JsonValue, so a field typed with either takes
	// what the annotated member returns, not the name of a constant (see
	// enum_test.go). Tier is how the decompiler prints a Kotlin enum.
	"com.example.lib.Currency": `package com.example.lib;

import com.fasterxml.jackson.annotation.JsonValue;

public enum Currency {
    USD("usd"),
    EUR("eur");

    private final String symbol;

    Currency(String symbol) {
        this.symbol = symbol;
    }

    @JsonValue
    public String getSymbol() {
        return symbol;
    }
}
`,
	"com.example.lib.Tier": `/*
 * Decompiled with CFR 0.152.
 */
package com.example.lib;

import com.fasterxml.jackson.annotation.JsonValue;
import kotlin.Metadata;
import org.jetbrains.annotations.NotNull;

@Metadata(mv={1, 9, 0}, k=1, xi=48)
public final class Tier
extends Enum<Tier> {
    public static final /* enum */ Tier GOLD = new Tier("gold");
    public static final /* enum */ Tier SILVER = new Tier("silver");
    private static final /* synthetic */ Tier[] $VALUES;
    @NotNull
    private final String code;

    private Tier(String code) {
        this.code = code;
    }

    @JsonValue
    @NotNull
    public final String getCode() {
        return this.code;
    }

    static {
        $VALUES = tierArray = new Tier[]{Tier.GOLD, Tier.SILVER};
    }
}
`,
	// Stage and Mode give a constant a name with @JsonProperty, so a field typed
	// with either takes that name, not the name of the constant (see
	// enum_test.go). Stage is how the decompiler prints a Java enum and Mode how it
	// prints a Kotlin one: the annotation is among the modifiers of the constant's
	// field.
	"com.example.lib.Stage": `/*
 * Decompiled with CFR 0.152.
 */
package com.example.lib;

import com.fasterxml.jackson.annotation.JsonProperty;

public enum Stage {
    @JsonProperty(value="first_stage")
    FIRST,
    SECOND;
}
`,
	"com.example.lib.Mode": `/*
 * Decompiled with CFR 0.152.
 */
package com.example.lib;

import com.fasterxml.jackson.annotation.JsonProperty;
import kotlin.Metadata;
import kotlin.enums.EnumEntries;
import kotlin.enums.EnumEntriesKt;
import org.jetbrains.annotations.NotNull;

@Metadata(mv={1, 9, 0}, k=1, xi=48)
public final class Mode
extends Enum<Mode> {
    @JsonProperty(value="fast_mode")
    public static final /* enum */ Mode FAST = new Mode();
    public static final /* enum */ Mode SLOW = new Mode();
    private static final /* synthetic */ Mode[] $VALUES;
    private static final /* synthetic */ EnumEntries $ENTRIES;

    private Mode() {
    }

    public static Mode[] values() {
        return (Mode[])$VALUES.clone();
    }

    public static Mode valueOf(String value) {
        return Enum.valueOf(Mode.class, value);
    }

    @NotNull
    public static EnumEntries<Mode> getEntries() {
        return $ENTRIES;
    }

    static {
        $VALUES = modeArray = new Mode[]{Mode.FAST, Mode.SLOW};
        $ENTRIES = EnumEntriesKt.enumEntries((Enum[])$VALUES);
    }
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
	// The JSON view exports a field typed with an enum as a string of its
	// constants, whether the project or a library declares it (see
	// enum_test.go).
	if typed := body["typed"]; typed == nil || typed.Model.TypeName != model.JsonTypeString || !reflect.DeepEqual(describeOptions(typed.Options), seeExpectation["state"]) {
		t.Errorf("body field typed = %+v, want a string of the enum's constants", typed)
	}
	// A Kotlin enum decompiled to a class is an enum all the same: not an
	// object of its properties.
	if typed := body["kotlinEnum"]; typed == nil || typed.Model.TypeName != model.JsonTypeString || typed.Model.Kind != model.KindSingle || !reflect.DeepEqual(describeOptions(typed.Options), []string{"NORTH: ", "SOUTH: "}) {
		t.Errorf("body field kotlinEnum = %+v, want a string of the enum's constants", typed)
	}

	// The library is asked for each name once, however many fields refer to
	// it and in whichever view, a miss included. Only qualified names reach it.
	asked := map[string]int{}
	for _, name := range library.asked {
		asked[name]++
	}
	wantAsked := []string{"com.example.lib.OrderState", "com.example.lib.PayChannel", "com.example.lib.Region", "com.example.lib.PageQuery", "com.example.lib.Missing"}
	for _, name := range wantAsked {
		if asked[name] != 1 {
			t.Errorf("%s was asked for %d times, want once", name, asked[name])
		}
	}
	if len(asked) != len(wantAsked) {
		t.Errorf("asked = %v, want only %v", library.asked, wantAsked)
	}
	// The `@see` lookups report nothing, and the enum-typed fields are strings.
	if counts := unresolved.Counts(); len(counts) != 0 {
		t.Errorf("unresolved = %v, want none", counts)
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
