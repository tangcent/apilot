package kotlincollector

import (
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	collector "github.com/tangcent/apilot/api-collector"
	javaparser "github.com/tangcent/apilot/api-collector-java/parser"
	ktparser "github.com/tangcent/apilot/api-collector-kotlin/parser"
	"github.com/tangcent/apilot/api-collector-kotlin/resolver"
	"github.com/tangcent/apilot/api-collector-kotlin/springmvc"
	model "github.com/tangcent/apilot/api-model"
)

// typeArgumentExpectation is the export of testdata/type-argument: the kind of
// what sits at a path of an endpoint's request body ("request.") or response
// ("response."). The application's Item, `object{name}`, and the library's,
// `object{code}`, share a simple name; the library's generic classes and its
// controllers sit in the package of the second. Every argument is the class the
// application names where it writes it, whatever the generic class's own
// package declares.
//
// The Java collector asserts the same table for the same declarations written
// in Java (api-collector-java/type_argument_test.go), where a wildcard
// (`/type-args/box-bounded` and the `bounded` field) is also declared: keep the
// two in sync.
var typeArgumentExpectation = map[string]map[string]string{
	"/type-args/box": {
		"request.data": "object{name}", "request.list": "array<object{name}>", "request.array": "array<object{name}>",
		"request.byKey": "map<string,object{name}>",
		"response.data": "object{name}", "response.list": "array<object{name}>", "response.array": "array<object{name}>",
		"response.byKey": "map<string,object{name}>",
	},
	"/type-args/outer": {
		"response.wrapped.data":      "object{name}",
		"response.wrappedList.data":  "array<object{name}>",
		"response.wrappedArray.data": "array<object{name}>",
	},
	"/type-args/outer-int": {
		"response.wrapped.data":      "int",
		"response.wrappedList.data":  "array<int>",
		"response.wrappedArray.data": "array<int>",
	},
	"/type-args/box-list": {
		"response.data": "array<object{name}>", "response.list": "array<array<object{name}>>",
	},
	"/type-args/box-array": {
		"response.data": "array<object{name}>", "response.array": "array<array<object{name}>>",
	},
	"/type-args/box-map": {
		"response.data": "map<string,object{name}>", "response.list": "array<map<string,object{name}>>",
	},
	// A type of the JDK the caller imports while the library declares a class of
	// the same name, and a scalar the resolver answers for by its spelling.
	"/type-args/box-period":    {"response.data": "string", "response.list": "array<string>"},
	"/type-args/box-date":      {"response.data": "string", "response.array": "array<string>"},
	"/type-args/box-date-list": {"response.data": "array<string>"},
	"/type-args/box-string":    {"response.data": "string", "response.byKey": "map<string,string>"},
	// A generic class as the argument of itself is expanded; a type that
	// refers to itself ends at a reference.
	"/type-args/box-box": {
		"response.data.data": "object{name}", "response.data.list": "array<object{name}>",
		"response.list": "array<object{array,byKey,data,list}>",
	},
	"/type-args/outer-outer": {"response.wrapped.data.wrapped.data": "object{name}"},
	"/type-args/node":        {"response.value": "object{name}", "response.next": "Node"},
	// A class that extends a map is a map of its superclass argument, written
	// in its own scope.
	"/type-args/bag":   {"response": "map<string,array<object{name}>>"},
	"/type-args/named": {"response": "map<string,object{name}>"},
	"/type-args/pairs": {"response": "map<string,object{name}>"},
	// The classes the library's Holder writes in full are the ones it names,
	// and the one it writes simply is its own.
	"/type-args/holder": {
		"response.fixedList":     "array<object{name}>",
		"response.fixedBox.data": "object{name}",
		"response.fixedBox.list": "array<object{name}>",
		"response.own":           "object{code}",
	},
	// Endpoints inherited from the library's generic controller.
	"/items/one":  {"response": "object{name}"},
	"/items/all":  {"response": "array<object{name}>"},
	"/items/save": {"request": "object{name}", "response": "object{name}"},
	// MidController<T> : BaseController<List<T>>(): the two Ts are not one.
	"/lists/one":  {"response": "array<object{name}>"},
	"/lists/all":  {"response": "array<array<object{name}>>"},
	"/lists/save": {"request": "array<object{name}>", "response": "array<object{name}>"},
	"/lists/mid":  {"response": "object{name}"},
}

// typeArgumentParameters are the parameters each endpoint binds from the
// request, by the binding view: `search(query: T)` binds the fields of the
// argument of its generic controller.
var typeArgumentParameters = map[string][]string{
	"/items/search": {"name/query"},
}

func TestTypeArgumentsAreTheClassesTheApplicationNames(t *testing.T) {
	abs, err := filepath.Abs("testdata/type-argument")
	if err != nil {
		t.Fatal(err)
	}
	c := New().(*KotlinCollector)
	endpoints, err := c.Collect(collector.CollectContext{SourceDir: abs, NoDeps: true})
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}
	checkTypeArgumentExport(t, endpoints, typeArgumentExpectation, typeArgumentParameters)
	if unresolved := c.Unresolved(); len(unresolved) != 0 {
		t.Errorf("unresolved = %v, want none", unresolved)
	}
}

// The library's classes are Kotlin sources read through a dependency resolver,
// as the indexer returns them from a sources JAR. Each imports its neighbours,
// so that reading one does not depend on the resolver reading its package.
var typeArgumentLibrarySources = map[string]string{
	"com.example.typearg.lib.Item": `package com.example.typearg.lib

class Item(
    val code: String?,
)
`,
	"com.example.typearg.lib.Box": `package com.example.typearg.lib

class Box<T>(
    val data: T?,
    val list: List<T>?,
    val array: Array<T>?,
    val byKey: Map<String, T>?,
)
`,
	"com.example.typearg.lib.Wrapper": `package com.example.typearg.lib

class Wrapper<U>(
    val data: U?,
)
`,
	"com.example.typearg.lib.Outer": `package com.example.typearg.lib

import com.example.typearg.lib.Wrapper

class Outer<T>(
    val wrapped: Wrapper<T>?,
    val wrappedList: Wrapper<List<T>>?,
    val wrappedArray: Wrapper<Array<T>>?,
)
`,
	"com.example.typearg.lib.Holder": `package com.example.typearg.lib

import com.example.typearg.lib.Box

class Holder(
    val fixedList: List<com.example.typearg.dto.Item>?,
    val fixedBox: Box<com.example.typearg.dto.Item>?,
)
`,
}

// typeArgumentLibrary answers dependency lookups from
// typeArgumentLibrarySources by qualified name, as the indexer does.
type typeArgumentLibrary struct {
	t     *testing.T
	asked []string
}

func (l *typeArgumentLibrary) ResolveClass(className string) *javaparser.Class {
	l.asked = append(l.asked, className)
	source, ok := typeArgumentLibrarySources[className]
	if !ok {
		return nil
	}
	classes, err := parseDependencySource(className, []byte(source))
	if err != nil {
		l.t.Fatalf("parsing %s: %v", className, err)
	}
	simple := className[strings.LastIndex(className, ".")+1:]
	for i := range classes {
		if classes[i].Name == simple {
			return &classes[i]
		}
	}
	return nil
}

// A library class read from a dependency gets the same arguments as one of the
// scanned sources: the classes the application names, not the library's Item.
func TestTypeArgumentsOfAKotlinLibraryAreTheClassesTheApplicationNames(t *testing.T) {
	p, err := ktparser.NewParser()
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()
	dir, _ := filepath.Abs("testdata/type-argument-dependency")
	results, err := p.ParseDirectory(dir)
	if err != nil {
		t.Fatal(err)
	}

	for name, warmFirst := range map[string]bool{"fresh": false, "library Item read first": true} {
		t.Run(name, func(t *testing.T) {
			typeResolver := resolver.New(flattenClasses(results))
			library := &typeArgumentLibrary{t: t}
			typeResolver.SetDependencyResolver(library)
			if warmFirst {
				if got := typeResolver.ResolveFrom(&ktparser.Class{}, "com.example.typearg.lib.Item", nil); got.Kind != model.KindObject {
					t.Fatalf("the library's Item = %+v", got)
				}
			}

			var endpoints []collector.ApiEndpoint
			for _, ctrl := range springmvc.NewParser(typeResolver).ExtractControllers(results) {
				for _, ep := range ctrl.Endpoints {
					endpoints = append(endpoints, springMVCEndpointToAPI(ep, ctrl.Name))
				}
			}
			checkTypeArgumentExport(t, endpoints, map[string]map[string]string{
				"/type-args/box": {
					"request.data": "object{name}", "request.list": "array<object{name}>",
					"response.data": "object{name}", "response.list": "array<object{name}>",
					"response.array": "array<object{name}>", "response.byKey": "map<string,object{name}>",
				},
				"/type-args/outer": {
					"response.wrapped.data":      "object{name}",
					"response.wrappedList.data":  "array<object{name}>",
					"response.wrappedArray.data": "array<object{name}>",
				},
				"/type-args/outer-int": {
					"response.wrapped.data":      "int",
					"response.wrappedList.data":  "array<int>",
					"response.wrappedArray.data": "array<int>",
				},
				"/type-args/holder": {
					"response.fixedList":     "array<object{name}>",
					"response.fixedBox.data": "object{name}",
					"response.fixedBox.list": "array<object{name}>",
				},
			}, nil)
			if unresolved := typeResolver.Unresolved(); len(unresolved) != 0 {
				t.Errorf("unresolved = %v, want none", unresolved)
			}
		})
	}
}

// checkTypeArgumentExport asserts the kinds of expectation and the parameters
// of parameters against what a collector exported.
func checkTypeArgumentExport(t *testing.T, endpoints []collector.ApiEndpoint, expectation map[string]map[string]string, parameters map[string][]string) {
	t.Helper()
	byPath := map[string]*collector.ApiEndpoint{}
	var exported []string
	for i := range endpoints {
		byPath[endpoints[i].Path] = &endpoints[i]
		exported = append(exported, endpoints[i].Path)
	}
	sort.Strings(exported)
	paths := make([]string, 0, len(expectation))
	for path := range expectation {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	for _, path := range paths {
		ep := byPath[path]
		if ep == nil {
			t.Errorf("endpoint %s was not exported; exported %v", path, exported)
			continue
		}
		wants := expectation[path]
		keys := make([]string, 0, len(wants))
		for key := range wants {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			if got := typeArgumentKindAt(ep, key); got != wants[key] {
				t.Errorf("%s %s = %s, want %s", path, key, got, wants[key])
			}
		}
	}
	for path, want := range parameters {
		ep := byPath[path]
		if ep == nil {
			t.Errorf("endpoint %s was not exported; exported %v", path, exported)
			continue
		}
		var got []string
		for _, p := range ep.Parameters {
			got = append(got, p.Name+"/"+p.In)
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("%s parameters = %v, want %v", path, got, want)
		}
	}
}

// typeArgumentKindAt follows "request.a.b" or "response.a.b" to a kind.
func typeArgumentKindAt(ep *collector.ApiEndpoint, key string) string {
	parts := strings.Split(key, ".")
	var m *model.ObjectModel
	switch parts[0] {
	case "request":
		if ep.RequestBody != nil {
			m = ep.RequestBody.Body
		}
	case "response":
		if ep.Response != nil {
			m = ep.Response.Body
		}
	}
	for _, name := range parts[1:] {
		if m == nil || m.Fields[name] == nil {
			return "nil"
		}
		m = m.Fields[name].Model
	}
	return arrayFieldKind(m)
}
