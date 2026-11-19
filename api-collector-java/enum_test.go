package javacollector

import (
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	collector "github.com/tangcent/apilot/api-collector"
	"github.com/tangcent/apilot/api-collector-java/parser"
	"github.com/tangcent/apilot/api-collector-java/springmvc"
	model "github.com/tangcent/apilot/api-model"
)

// enumField is how the JSON view exports a field: its model, spelled by
// describeModel, and the options it takes, as "value: description".
type enumField struct {
	model   string
	options []string
}

var orderStateOptions = []string{"PENDING: Waiting for payment.", "PAID: Paid.\n\nShips within a day.", "CLOSED: "}

// enumExpectation is the export of testdata/enum: each field of the request
// body. A field typed with an enum that Jackson writes as the name of a
// constant is a string whose options are the constants, described by their doc
// comments, whatever else the enum carries. So is a collection of them: the
// items are strings, and the field lists the constants. The values of a map
// are strings of the constants too, but the map lists none, as EasyYapi does
// not: the key of a map is a string whatever it is declared as, and says
// nothing of the enum. Values the field declares itself win. An enum with another JSON form, which declares
// @JsonValue or gives itself the shape of an object or a number, keeps its
// type name: what the field takes cannot be told from the constants. So does a
// type the sources do not declare.
//
// The Kotlin collector asserts the same table for the same declarations
// written in Kotlin (api-collector-kotlin/enum_test.go): keep the two in sync.
var enumExpectation = map[string]enumField{
	"state":    {"string", orderStateOptions},
	"history":  {"array<string>", orderStateOptions},
	"allowed":  {"array<string>", orderStateOptions},
	"byRegion": {"map<string,string>", nil},
	"byState":  {"map<string,string>", nil},
	"level":    {"string", []string{"LOW: ", "HIGH: "}},
	"flag":     {"string", []string{"ON: ", "OFF: "}},
	"toggle":   {"string", []string{"YES: ", "NO: "}},
	"declared": {"string", []string{"OPEN: ", "SHUT: "}},
	"channel":  {"ChannelCode", nil},
	"channels": {"array<ChannelCode>", nil},
	"grade":    {"Grade", nil},
	"shape":    {"Shape", nil},
	"rank":     {"Rank", nil},
	"missing":  {"Missing", nil},
}

// enumResponseExpectation is the model of the response of each handler of
// testdata/enum. The response of create is an envelope whose data is the
// enum: that field is a string of the constants.
var enumResponseExpectation = map[string]string{
	"state":   "string",
	"states":  "array<string>",
	"channel": "ChannelCode",
}

// enumQueryExpectation is the same form bound from request parameters, which
// the JSON view does not change: Spring converts a parameter to an enum by the
// name of a constant whatever the enum's JSON form, and a collection takes no
// options.
var enumQueryExpectation = map[string][]string{
	"state":    {"PENDING", "PAID", "CLOSED"},
	"level":    {"LOW", "HIGH"},
	"channel":  {"ALIPAY", "WECHAT"},
	"grade":    {"A", "B"},
	"shape":    {"CIRCLE", "SQUARE"},
	"rank":     {"FIRST", "SECOND"},
	"flag":     {"ON", "OFF"},
	"toggle":   {"YES", "NO"},
	"declared": {"OPEN", "SHUT"},
	"history":  nil,
	"channels": nil,
}

// enumUnresolved are the types of testdata/enum the export reports as
// unresolved: the enums it exports by their names, and the type that is not
// declared.
var enumUnresolved = []string{"ChannelCode", "Grade", "Missing", "Rank", "Shape"}

func TestJSONViewExportsEnumsAsStrings(t *testing.T) {
	abs, err := filepath.Abs("testdata/enum")
	if err != nil {
		t.Fatal(err)
	}
	c := New().(*JavaCollector)
	endpoints, err := c.Collect(collector.CollectContext{SourceDir: abs, NoDeps: true})
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}
	checkEnumExport(t, endpoints, c.Unresolved())
}

// checkEnumExport asserts enumExpectation on the request body of the create
// endpoint, enumResponseExpectation and the data of the envelope it returns,
// and enumQueryExpectation on the page endpoint.
func checkEnumExport(t *testing.T, endpoints []collector.ApiEndpoint, unresolved map[string]int) {
	t.Helper()
	byName := map[string]collector.ApiEndpoint{}
	for _, ep := range endpoints {
		byName[ep.Name] = ep
	}
	create, page := byName["create"], byName["page"]
	if create.RequestBody == nil || create.RequestBody.Body == nil || create.Response == nil || page.Name == "" {
		t.Fatalf("endpoints = %+v, want create with a body and a response, and page", endpoints)
	}

	checkEnumFields(t, "request body", create.RequestBody.Body.Fields, enumExpectation)
	checkEnumFields(t, "response of create", create.Response.Body.Fields, map[string]enumField{"data": {"string", orderStateOptions}})
	for name, want := range enumResponseExpectation {
		ep := byName[name]
		if ep.Response == nil || ep.Response.Body == nil {
			t.Errorf("endpoint %s has no response", name)
			continue
		}
		if got := describeModel(ep.Response.Body); got != want {
			t.Errorf("response of %s = %s, want %s", name, got, want)
		}
	}

	params := map[string]collector.ApiParameter{}
	for _, p := range page.Parameters {
		params[p.Name] = p
	}
	for name, want := range enumQueryExpectation {
		if got := params[name].Enum; !reflect.DeepEqual(got, want) {
			t.Errorf("query parameter %s enum = %q, want %q", name, got, want)
		}
	}

	var names []string
	for name := range unresolved {
		names = append(names, name)
	}
	sort.Strings(names)
	if !reflect.DeepEqual(names, enumUnresolved) {
		t.Errorf("unresolved = %v, want %v", names, enumUnresolved)
	}
}

// checkEnumFields asserts the model and the options of the fields in want.
func checkEnumFields(t *testing.T, where string, fields map[string]*model.FieldModel, want map[string]enumField) {
	t.Helper()
	for name, expected := range want {
		field := fields[name]
		if field == nil {
			t.Errorf("%s: field %s missing", where, name)
			continue
		}
		if got := describeModel(field.Model); got != expected.model {
			t.Errorf("%s: field %s = %s, want %s", where, name, got, expected.model)
		}
		if got := describeOptions(field.Options); !reflect.DeepEqual(got, expected.options) {
			t.Errorf("%s: field %s options:\n got  %q\n want %q", where, name, got, expected.options)
		}
	}
}

// describeModel spells a model: a scalar or a type by its name, a collection as
// array<item>, a map as map<key,value>.
func describeModel(m *model.ObjectModel) string {
	switch {
	case m == nil:
		return "nil"
	case m.IsArray():
		return "array<" + describeModel(m.Items) + ">"
	case m.IsMap():
		return "map<" + describeModel(m.KeyModel) + "," + describeModel(m.ValueModel) + ">"
	case m.Kind == model.KindObject:
		return "object " + m.TypeName
	}
	return m.TypeName
}

// enumDependencyExpectation is the export of testdata/enum-dependency: a form
// typed with the enums of a library rather than of the project. They follow the
// rule of the project's own. OrderState comes from a sources JAR, with doc
// comments; PayChannel, decompiled from a JAR without sources, gives its
// constants without descriptions, and so does Region, a Kotlin enum decompiled
// from one: the decompiler prints that as a class rather than an enum. Currency
// declares @JsonValue on a method, and Tier, decompiled the same way as Region,
// on its getter: both keep their type names, as does a class the library does
// not have.
//
// The Kotlin collector asserts the same table for the same form written in
// Kotlin (api-collector-kotlin/enum_test.go): keep the two in sync.
var enumDependencyExpectation = map[string]enumField{
	"state":    {"string", orderStateOptions},
	"states":   {"array<string>", orderStateOptions},
	"channel":  {"string", []string{"ALIPAY: ", "WECHAT: "}},
	"region":   {"string", []string{"NORTH: ", "SOUTH: "}},
	"currency": {"Currency", nil},
	"tier":     {"Tier", nil},
	"missing":  {"Missing", nil},
}

// enumDependencyUnresolved are the types of testdata/enum-dependency the
// export reports as unresolved.
var enumDependencyUnresolved = []string{"Currency", "Missing", "Tier"}

func TestJSONViewReadsEnumsOfADependency(t *testing.T) {
	abs, err := filepath.Abs("testdata/enum-dependency")
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
	var create *collector.ApiEndpoint
	for _, ctrl := range sm.ExtractControllers(results) {
		for _, ep := range ctrl.Endpoints {
			if api := springmvcEndpointToAPI(ep, ctrl.Name); api.Name == "create" {
				create = &api
			}
		}
	}
	if create == nil || create.RequestBody == nil || create.RequestBody.Body == nil {
		t.Fatalf("create = %+v, want an endpoint with a request body", create)
	}
	checkEnumFields(t, "request body", create.RequestBody.Body.Fields, enumDependencyExpectation)

	var names []string
	for name := range unresolved.Counts() {
		names = append(names, name)
	}
	sort.Strings(names)
	if !reflect.DeepEqual(names, enumDependencyUnresolved) {
		t.Errorf("unresolved = %v, want %v", names, enumDependencyUnresolved)
	}

	// The library is asked for each name once, however many fields refer to it.
	asked := map[string]int{}
	for _, name := range library.asked {
		asked[name]++
	}
	for _, name := range []string{"OrderState", "PayChannel", "Region", "Currency", "Tier", "Missing"} {
		if n := asked["com.example.lib."+name]; n != 1 {
			t.Errorf("%s was asked for %d times, want once; asked = %v", name, n, library.asked)
		}
	}
	for name := range asked {
		if !strings.HasPrefix(name, "com.example.lib.") {
			t.Errorf("the library was asked for %s, which the source does not qualify", name)
		}
	}
}
