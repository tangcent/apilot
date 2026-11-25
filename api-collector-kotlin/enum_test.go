package kotlincollector

import (
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	collector "github.com/tangcent/apilot/api-collector"
	ktparser "github.com/tangcent/apilot/api-collector-kotlin/parser"
	"github.com/tangcent/apilot/api-collector-kotlin/resolver"
	"github.com/tangcent/apilot/api-collector-kotlin/springmvc"
	model "github.com/tangcent/apilot/api-model"
)

// enumField is how the JSON view exports a field: its model, spelled by
// describeModel, and the options it takes, as "value: description".
type enumField struct {
	model   string
	options []string
}

var orderStateOptions = []string{"PENDING: Waiting for payment.", "PAID: Paid.\n\nShips within a day.", "CLOSED: "}

// enumExpectation is the export of testdata/enum: each property of the request
// body. A property typed with an enum that Jackson writes as the name of an
// entry is a string whose options are the entries, described by their KDoc,
// whatever else the enum carries. So is a collection of them: the items are
// strings, and the property lists the entries. The values of a map are strings
// of the entries too, but the map lists none, as EasyYapi does not: the key of
// a map is a string whatever it is declared as, and says nothing of the enum.
// Values the property declares itself win. An enum with another JSON form, which declares
// @JsonValue or gives itself the shape of an object or a number, keeps its
// type name: what the property takes cannot be told from the entries. So does
// a type the sources do not declare.
//
// The Java collector asserts the same table for the same declarations written
// in Java (api-collector-java/enum_test.go): keep the two in sync.
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
// enum: that property is a string of the entries.
var enumResponseExpectation = map[string]string{
	"state":   "string",
	"states":  "array<string>",
	"channel": "ChannelCode",
}

// enumQueryExpectation is the same form bound from request parameters, which
// the JSON view does not change: Spring converts a parameter to an enum by the
// name of an entry whatever the enum's JSON form, and a collection of them takes
// the entries of the enum it holds.
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
	"history":  {"PENDING", "PAID", "CLOSED"},
	"channels": {"ALIPAY", "WECHAT"},
}

// enumUnresolved are the types of testdata/enum the export reports as
// unresolved: the enums it exports by their names, and the type that is not
// declared.
var enumUnresolved = []string{"ChannelCode", "Grade", "Missing", "Rank", "Shape"}

func TestJSONViewExportsEnumsAsStrings(t *testing.T) {
	endpoints, c := collectDir(t, "testdata/enum")
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
// rule of the project's own. OrderState comes from a sources JAR, with KDoc;
// PayChannel, decompiled from a JAR without sources, gives its entries without
// descriptions, and so does Region, a Kotlin enum decompiled from one: the
// decompiler prints that as a class rather than an enum. Currency declares
// @JsonValue on its getter, and Tier, decompiled the same way as Region, does
// too: both keep their type names, as does a class the library does not have.
//
// The Java collector asserts the same table for the same form written in Java
// (api-collector-java/enum_test.go): keep the two in sync.
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
	p, err := ktparser.NewParser()
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()
	dir, _ := filepath.Abs("testdata/enum-dependency")
	results, err := p.ParseDirectory(dir)
	if err != nil {
		t.Fatal(err)
	}

	typeResolver := resolver.New(flattenClasses(results))
	library := &libraryResolver{t: t}
	typeResolver.SetDependencyResolver(library)
	sm := springmvc.NewParser(typeResolver)
	var create *collector.ApiEndpoint
	for _, ctrl := range sm.ExtractControllers(results) {
		for _, ep := range ctrl.Endpoints {
			if api := springMVCEndpointToAPI(ep, ctrl.Name); api.Name == "create" {
				create = &api
			}
		}
	}
	if create == nil || create.RequestBody == nil || create.RequestBody.Body == nil {
		t.Fatalf("create = %+v, want an endpoint with a request body", create)
	}
	checkEnumFields(t, "request body", create.RequestBody.Body.Fields, enumDependencyExpectation)

	var names []string
	for name := range typeResolver.Unresolved() {
		names = append(names, name)
	}
	sort.Strings(names)
	if !reflect.DeepEqual(names, enumDependencyUnresolved) {
		t.Errorf("unresolved = %v, want %v", names, enumDependencyUnresolved)
	}

	// The library is asked for each name once, however many properties refer to
	// it.
	asked := map[string]int{}
	for _, name := range library.queried {
		asked[name]++
	}
	for _, name := range []string{"OrderState", "PayChannel", "Region", "Currency", "Tier", "Missing"} {
		if n := asked["com.example.lib."+name]; n != 1 {
			t.Errorf("%s was asked for %d times, want once; asked = %v", name, n, library.queried)
		}
	}
	for name := range asked {
		if !strings.HasPrefix(name, "com.example.lib.") {
			t.Errorf("the library was asked for %s, which the source does not qualify", name)
		}
	}
}

// enumScopeExpectation is the export of testdata/enum-scope: the request body of
// each handler. Two packages declare a Code and an Order with an enum Status
// nested in it, and the holder of each package names them in its own scope.
// a.Code declares @JsonValue, so a property typed with it keeps its type name,
// and the b.Code that the holder of a does not see does not stand in for it. The
// holder of b reads b.Code. A nested enum named through Order is the one nested
// in the Order the holder's own package declares.
//
// The Java collector asserts the same table for the same declarations written in
// Java (api-collector-java/enum_test.go): keep the two in sync.
var enumScopeExpectation = map[string]map[string]enumField{
	"createA": {
		"code":     {"Code", nil},
		"codes":    {"array<Code>", nil},
		"statuses": {"array<string>", []string{"A_OPEN: Opened in a.", "A_CLOSED: "}},
	},
	"createB": {
		"code":     {"string", []string{"B: ", "C: "}},
		"codes":    {"array<string>", []string{"B: ", "C: "}},
		"statuses": {"array<string>", []string{"B_OPEN: Opened in b.", "B_CLOSED: "}},
	},
}

// enumScopeUnresolved are the types of testdata/enum-scope the export reports as
// unresolved: the enum of a that Jackson writes in another form.
var enumScopeUnresolved = []string{"Code"}

func TestJSONViewResolvesEnumNamesInTheScopeOfTheSource(t *testing.T) {
	endpoints, c := collectDir(t, "testdata/enum-scope")
	byName := map[string]collector.ApiEndpoint{}
	for _, ep := range endpoints {
		byName[ep.Name] = ep
	}
	for name, want := range enumScopeExpectation {
		ep := byName[name]
		if ep.RequestBody == nil || ep.RequestBody.Body == nil {
			t.Errorf("endpoint %s has no request body; endpoints = %+v", name, endpoints)
			continue
		}
		checkEnumFields(t, name, ep.RequestBody.Body.Fields, want)
	}

	var names []string
	for name := range c.Unresolved() {
		names = append(names, name)
	}
	sort.Strings(names)
	if !reflect.DeepEqual(names, enumScopeUnresolved) {
		t.Errorf("unresolved = %v, want %v", names, enumScopeUnresolved)
	}
}

// enumNestedExpectation is the export of testdata/enum-nested: two classes of
// one file, each nesting an enum named Status that declares the entry A.
// Order.Status declares @JsonValue and Product.Status does not, so a property
// typed with the first keeps its type name, however the type is written around
// it, and a property typed with the second is a string of its own entries, A and C.
//
// The Java collector asserts the same table for the same declarations written in
// Java (api-collector-java/enum_test.go): keep the two in sync.
var enumNestedExpectation = map[string]enumField{
	"order":        {"Order.Status", nil},
	"orders":       {"array<Order.Status>", nil},
	"orderArray":   {"array<Order.Status>", nil},
	"product":      {"string", []string{"A: ", "C: "}},
	"products":     {"array<string>", []string{"A: ", "C: "}},
	"productArray": {"array<string>", []string{"A: ", "C: "}},
	"byName":       {"map<string,string>", nil},
}

// enumNestedBinding is the same declarations bound from request parameters: the
// binding view reads every enum, so the entries are those of the enum each
// parameter names, a collection of them included.
var enumNestedBinding = map[string][]string{
	"order":    {"A", "B"},
	"product":  {"A", "C"},
	"orders":   {"A", "B"},
	"products": {"A", "C"},
	// Parameters of a handler that carries no annotation: an array and a
	// collection of an enum list the entries of the enum they hold.
	"orderStatus":     {"A", "B"},
	"byParam":         {"A", "B"},
	"byPath":          {"A", "C"},
	"byList":          {"A", "C"},
	"productStatus":   {"A", "C"},
	"productStatuses": {"A", "C"},
	"orderStatusList": {"A", "B"},
}

func TestJSONViewTellsEnumsNestedInDifferentClassesApart(t *testing.T) {
	endpoints, c := collectDir(t, "testdata/enum-nested")
	byName := map[string]collector.ApiEndpoint{}
	for _, ep := range endpoints {
		byName[ep.Name] = ep
	}
	create := byName["create"]
	if create.RequestBody == nil || create.RequestBody.Body == nil {
		t.Fatalf("endpoints = %+v, want create with a body", endpoints)
	}
	checkEnumFields(t, "request body", create.RequestBody.Body.Fields, enumNestedExpectation)

	params := map[string]collector.ApiParameter{}
	for _, name := range []string{"page", "values", "explicit"} {
		for _, p := range byName[name].Parameters {
			params[p.Name] = p
		}
	}
	for name, want := range enumNestedBinding {
		if got := params[name].Enum; !reflect.DeepEqual(got, want) {
			t.Errorf("parameter %s enum = %q, want %q", name, got, want)
		}
	}

	var names []string
	for name := range c.Unresolved() {
		names = append(names, name)
	}
	sort.Strings(names)
	if want := []string{"Order.Status"}; !reflect.DeepEqual(names, want) {
		t.Errorf("unresolved = %v, want %v", names, want)
	}
}

// enumFormExpectation is the export of testdata/enum-forms: an enum that Jackson
// writes as the name of its entry is a string of the names. Every other form keeps
// the type name of the enum: an entry renamed with @JsonProperty, the value of a
// member of an interface the enum implements, the shape ARRAY or a number of
// @JsonFormat on the enum or on the property that holds it (a collection of them
// too), and a serializer of the enum or of the property.
//
// The Java collector asserts the same table for the same declarations written in
// Java (api-collector-java/enum_test.go): keep the two in sync. Arrayed and
// Serialized have two entries here and one constant there: the Kotlin grammar
// does not read an annotated enum of a single entry that has no trailing comma.
var enumFormExpectation = map[string]enumField{
	"plain":           {"string", []string{"A: ", "B: "}},
	"renamed":         {"Renamed", nil},
	"renamedList":     {"array<Renamed>", nil},
	"viaCoded":        {"ViaCoded", nil},
	"indexed":         {"Indexed", nil},
	"arrayed":         {"Arrayed", nil},
	"serialized":      {"Serialized", nil},
	"fieldIndex":      {"Plain", nil},
	"listIndex":       {"array<Plain>", nil},
	"fieldSerialized": {"Plain", nil},
}

// enumFormBinding is the same form bound from request parameters: Spring converts a
// parameter by the name of the entry in the source, whatever Jackson writes.
var enumFormBinding = map[string][]string{
	"plain":      {"A", "B"},
	"renamed":    {"A", "B"},
	"viaCoded":   {"A"},
	"indexed":    {"X", "Y"},
	"arrayed":    {"A", "B"},
	"serialized": {"A", "B"},
	"fieldIndex": {"A", "B"},
}

func TestJSONViewKeepsAnEnumOfACustomJacksonFormUnresolved(t *testing.T) {
	endpoints, c := collectDir(t, "testdata/enum-forms")
	byName := map[string]collector.ApiEndpoint{}
	for _, ep := range endpoints {
		byName[ep.Name] = ep
	}
	create := byName["create"]
	if create.RequestBody == nil || create.RequestBody.Body == nil {
		t.Fatalf("endpoints = %+v, want create with a body", endpoints)
	}
	checkEnumFields(t, "request body", create.RequestBody.Body.Fields, enumFormExpectation)

	params := map[string]collector.ApiParameter{}
	for _, p := range byName["page"].Parameters {
		params[p.Name] = p
	}
	for name, want := range enumFormBinding {
		if got := params[name].Enum; !reflect.DeepEqual(got, want) {
			t.Errorf("parameter %s enum = %q, want %q", name, got, want)
		}
	}

	var names []string
	for name := range c.Unresolved() {
		names = append(names, name)
	}
	sort.Strings(names)
	if want := []string{"Arrayed", "Indexed", "Plain", "Renamed", "Serialized", "ViaCoded"}; !reflect.DeepEqual(names, want) {
		t.Errorf("unresolved = %v, want %v", names, want)
	}
}

// enumPackagesExpectation is the export of testdata/enum-packages: two enums named
// Status in different packages, and a class of that name in a third that nothing
// imports. Each name takes the enum its scope gives it, and the class is taken for
// neither.
//
// The Java collector asserts the same table for the same declarations written in
// Java (api-collector-java/enum_test.go): keep the two in sync.
var enumPackagesExpectation = map[string]enumField{
	"own":    {"string", []string{"OPEN: ", "CLOSED: "}},
	"ownLib": {"string", []string{"ON: ", "OFF: "}},
	"many":   {"array<string>", []string{"OPEN: ", "CLOSED: "}},
}

func TestJSONViewTellsEnumsOfOneNameInPackagesApart(t *testing.T) {
	endpoints, c := collectDir(t, "testdata/enum-packages")
	byName := map[string]collector.ApiEndpoint{}
	for _, ep := range endpoints {
		byName[ep.Name] = ep
	}
	create := byName["create"]
	if create.RequestBody == nil || create.RequestBody.Body == nil {
		t.Fatalf("endpoints = %+v, want create with a body", endpoints)
	}
	checkEnumFields(t, "request body", create.RequestBody.Body.Fields, enumPackagesExpectation)
	for name, want := range map[string]string{"direct": "string", "list": "array<string>"} {
		ep := byName[name]
		if ep.Response == nil || ep.Response.Body == nil {
			t.Errorf("endpoint %s has no response", name)
			continue
		}
		if got := describeModel(ep.Response.Body); got != want {
			t.Errorf("response of %s = %s, want %s", name, got, want)
		}
	}
	if got := c.Unresolved(); len(got) != 0 {
		t.Errorf("unresolved = %v, want none", got)
	}
}
