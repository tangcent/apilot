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

// starBodyExpectation and starResponseExpectation are the export of
// testdata/star-import, a controller that names every class of its library
// through an import on demand (`import com.example.lib.model.*;`): the fields
// of the request body and of the response, each as its kind and, for an object,
// its field names. StarOrder lives in one package of the library and
// StarEnvelope in another; StarOrder refers to StarAddress and StarItem without
// an import, as classes of its own package, and the response carries StarItem
// as a type argument. StarMissing is declared nowhere.
//
// The Kotlin collector asserts the same tables for the same declarations
// written in Kotlin (api-collector-kotlin/star_import_test.go): keep the two in
// sync.
var starBodyExpectation = map[string]string{
	"no":      "string",
	"address": "object{city}",
	"items":   "array<object{qty,sku}>",
}

var starResponseExpectation = map[string]string{
	"code": "int",
	"data": "array<object{qty,sku}>",
}

// starMissingExpectation is the response of the endpoint whose type argument no
// library declares: the envelope is expanded, the argument stays opaque and is
// reported.
var starMissingExpectation = map[string]string{
	"code": "int",
	"data": "StarMissing",
}

// starLibrarySources are the classes of the library, as the indexer returns
// them.
var starLibrarySources = map[string]string{
	"com.example.lib.model.StarOrder": `package com.example.lib.model;

import java.util.List;

public class StarOrder {
    private String no;
    private StarAddress address;
    private List<StarItem> items;
}
`,
	"com.example.lib.model.StarAddress": `package com.example.lib.model;

public class StarAddress {
    private String city;
}
`,
	"com.example.lib.model.StarItem": `package com.example.lib.model;

public class StarItem {
    private String sku;
    private int qty;
}
`,
	"com.example.lib.response.StarEnvelope": `package com.example.lib.response;

public class StarEnvelope<T> {
    private int code;
    private T data;
}
`,
}

// starLibrary answers dependency lookups from starLibrarySources by qualified
// name, as the indexer does, and records each lookup.
type starLibrary struct {
	t      *testing.T
	parser *parser.Parser
	asked  []string
}

func (l *starLibrary) ResolveClass(name string) *parser.Class {
	l.asked = append(l.asked, name)
	source, ok := starLibrarySources[name]
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

func TestImportOnDemandNamesClassesOfADependency(t *testing.T) {
	abs, err := filepath.Abs("testdata/star-import")
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

	library := &starLibrary{t: t, parser: p}
	unresolved := collector.NewUnresolvedSet()
	sm := springmvc.NewParser()
	sm.SetDependencyResolver(library)
	sm.SetUnresolved(unresolved)
	endpoints := map[string]collector.ApiEndpoint{}
	for _, ctrl := range sm.ExtractControllers(results) {
		for _, ep := range ctrl.Endpoints {
			endpoints[ep.MethodName] = springmvcEndpointToAPI(ep, ctrl.Name)
		}
	}

	create, missing := endpoints["create"], endpoints["missing"]
	if create.RequestBody == nil || create.RequestBody.Body == nil || create.Response == nil {
		t.Fatalf("create = %+v, want a request body and a response", create)
	}
	if got := starFieldKinds(create.RequestBody.Body); !reflect.DeepEqual(got, starBodyExpectation) {
		t.Errorf("request body:\n got  %v\n want %v", got, starBodyExpectation)
	}
	if got := starFieldKinds(create.Response.Body); !reflect.DeepEqual(got, starResponseExpectation) {
		t.Errorf("response:\n got  %v\n want %v", got, starResponseExpectation)
	}
	if missing.Response == nil {
		t.Fatalf("missing = %+v, want a response", missing)
	}
	if got := starFieldKinds(missing.Response.Body); !reflect.DeepEqual(got, starMissingExpectation) {
		t.Errorf("response of missing:\n got  %v\n want %v", got, starMissingExpectation)
	}
	if got := unresolved.Counts(); !reflect.DeepEqual(got, map[string]int{"StarMissing": 1}) {
		t.Errorf("unresolved = %v, want only StarMissing", got)
	}

	// Every qualified name is asked for once, a miss included, and no platform
	// package is tried: the library is a process of the indexer per name.
	asked := map[string]int{}
	for _, name := range library.asked {
		asked[name]++
		if strings.HasPrefix(name, "java.") {
			t.Errorf("the library was asked for %s", name)
		}
	}
	var repeated []string
	for name, n := range asked {
		if n != 1 {
			repeated = append(repeated, name)
		}
	}
	sort.Strings(repeated)
	if len(repeated) != 0 {
		t.Errorf("asked more than once: %v; all asked: %v", repeated, library.asked)
	}
	for name := range starLibrarySources {
		if asked[name] != 1 {
			t.Errorf("%s was asked for %d times, want once; asked %v", name, asked[name], library.asked)
		}
	}
	// The package of the controller comes first, then each import in turn.
	want := []string{"com.example.ondemand.StarOrder", "com.example.lib.model.StarOrder"}
	if got := library.asked[:2]; !reflect.DeepEqual(got, want) {
		t.Errorf("first lookups = %v, want %v", got, want)
	}
}

// starFieldKinds spells the fields of an object model as their kinds.
func starFieldKinds(m *model.ObjectModel) map[string]string {
	out := map[string]string{}
	for name, field := range m.Fields {
		out[name] = starModelKind(field.Model)
	}
	return out
}

func starModelKind(m *model.ObjectModel) string {
	switch {
	case m == nil:
		return "nil"
	case m.IsArray():
		return "array<" + starModelKind(m.Items) + ">"
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
