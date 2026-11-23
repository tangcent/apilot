package javacollector

import (
	"encoding/json"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	collector "github.com/tangcent/apilot/api-collector"
	"github.com/tangcent/apilot/api-collector-java/maven"
	"github.com/tangcent/apilot/api-collector-java/parser"
	"github.com/tangcent/apilot/api-collector-java/resolver"
	"github.com/tangcent/apilot/api-collector-java/springmvc"
	model "github.com/tangcent/apilot/api-model"
)

// nestedResponseExpectation is the response of each endpoint of
// testdata/nested-class, spelled by nestedShape: a class nested in another is a
// class of its own. NestHolder and NestOther each nest a Detail, and each
// reaches its own by the simple name; the other is reached as NestOther.Detail,
// and NestHolder.Detail as the return type, as a type argument and by an import.
// NestHolder also nests a NestBadge, which shadows the top-level class of the
// package inside NestHolder and nowhere else, and Mid.Deep, a class nested two
// deep. NestChild reads the Detail of NestHolder through its superclass.
//
// The Kotlin collector asserts the same table for the same declarations written
// in Kotlin (api-collector-kotlin/nested_class_test.go): keep the two in sync.
var nestedResponseExpectation = map[string]string{
	"holder":         `{badge:{text:string},deep:{depth:int},other:{amount:long},others:[{amount:long}],own:{code:string},qualified:{code:string},title:string}`,
	"other":          `{badge:{label:string},own:{amount:long}}`,
	"child":          `{badge:{text:string},deep:{depth:int},inherited:{code:string},other:{amount:long},others:[{amount:long}],own:{code:string},qualified:{code:string},title:string}`,
	"mid":            `{deep:{depth:int},detail:{code:string}}`,
	"detail":         `{code:string}`,
	"create":         `{code:int,data:{amount:long}}`,
	"envelope":       `{code:int,data:{code:string}}`,
	"list":           `[{amount:long}]`,
	"single":         `{code:string}`,
	"star":           `{amount:long}`,
	"fullyQualified": `{code:string}`,
}

// nestedBodyExpectation is the request body of the create endpoint.
var nestedBodyExpectation = `{code:string}`

// nestedLibraryExpectation is the response of each endpoint of
// testdata/nested-class-dependency, whose classes are nested in a class of a
// library: NestPage is read from the index by its own name, and the classes
// nested in it are taken from its source. NestPage.Missing is declared nowhere.
//
// The Kotlin collector asserts the same table for the same library written in
// Kotlin (api-collector-kotlin/nested_class_test.go): keep the two in sync.
var nestedLibraryExpectation = map[string]string{
	"sort":    `{field:string,order:{asc:boolean}}`,
	"footer":  `{note:string}`,
	"page":    `{footer:{note:string},rows:[{name:string}],sort:{field:string,order:{asc:boolean}},total:int}`,
	"missing": `NestPage.Missing`,
}

// nestedLibrarySources are the classes of the library, as the indexer returns
// them: the source of NestPage from a sources JAR, and the same class
// decompiled, which spells a nested class `NestPage.Sort` and, where the
// decompiler lost track of the enclosing class, by its binary name
// `NestPage$Footer`. The indexer holds NestPage only: no nested class is a name
// of its own.
var nestedLibrarySources = map[string]string{
	"sources": `package com.example.nestlib;

import java.util.List;

public class NestPage<T> {
    private int total;
    private List<T> rows;
    private Sort sort;
    private NestPage.Footer footer;

    public static class Sort {
        private String field;
        private Order order;

        public static class Order {
            private boolean asc;
        }
    }

    public static class Footer {
        private String note;
    }
}
`,
	"decompiled": `/*
 * Decompiled with CFR 0.152.
 */
package com.example.nestlib;

import java.util.List;

public class NestPage<T> {
    private int total;
    private List<T> rows;
    private NestPage.Sort sort;
    private NestPage$Footer footer;

    public int getTotal() {
        return this.total;
    }

    public static class Sort {
        private String field;
        private Sort.Order order;

        public static class Order {
            private boolean asc;
        }
    }

    public static class Footer {
        private String note;
    }
}
`,
}

func TestNestedClassesAreClassesOfTheirOwn(t *testing.T) {
	endpoints, unresolved := nestedProjectExport(t, nil)
	checkNestedExport(t, endpoints)
	if len(unresolved) != 0 {
		t.Errorf("unresolved = %v, want none", unresolved)
	}
}

// With a dependency index the names that qualify a type argument are resolved
// by another route; the export is the same, and nothing of the project is asked
// of the index.
func TestNestedClassesAreClassesOfTheirOwnWithADependencyIndex(t *testing.T) {
	library := &emptyIndex{}
	endpoints, unresolved := nestedProjectExport(t, library)
	checkNestedExport(t, endpoints)
	if len(unresolved) != 0 {
		t.Errorf("unresolved = %v, want none", unresolved)
	}
	if len(library.asked) != 0 {
		t.Errorf("the index was asked for %v, though the project declares every class", library.asked)
	}
}

func TestNestedClassesOfADependencyAreReadFromTheSourceOfTheirOuterClass(t *testing.T) {
	for _, shape := range []string{"sources", "decompiled"} {
		t.Run(shape, func(t *testing.T) {
			var asked []string
			index, err := maven.NewMavenDependencyResolver()
			if err != nil {
				t.Fatal(err)
			}
			defer index.Close()
			index.SetRunner(scriptedIndex(map[string]string{"com.example.nestlib.NestPage": nestedLibrarySources[shape]}, &asked))

			abs, err := filepath.Abs("testdata/nested-class-dependency")
			if err != nil {
				t.Fatal(err)
			}
			unresolved := collector.NewUnresolvedSet()
			endpoints := nestedEndpoints(t, abs, index, unresolved)

			for name, want := range nestedLibraryExpectation {
				ep, ok := endpoints[name]
				if !ok || ep.Response == nil {
					t.Errorf("endpoint %s has no response: %+v", name, ep)
					continue
				}
				if got := nestedShape(ep.Response.Body); got != want {
					t.Errorf("%s:\n got  %s\n want %s", name, got, want)
				}
			}
			if got, want := unresolved.Counts(), map[string]int{"NestPage.Missing": 1}; !reflect.DeepEqual(got, want) {
				t.Errorf("unresolved = %v, want %v", got, want)
			}
			// The index holds top-level classes: the outer class is asked for once,
			// and no nested name is.
			if want := []string{"com.example.nestlib.NestPage"}; !reflect.DeepEqual(asked, want) {
				t.Errorf("the index was asked for %v, want %v", asked, want)
			}
		})
	}
}

// emptyIndex is a dependency index that holds nothing.
type emptyIndex struct{ asked []string }

func (e *emptyIndex) ResolveClass(name string) *parser.Class {
	e.asked = append(e.asked, name)
	return nil
}

// scriptedIndex answers `explore` for the classes in sources, by qualified name,
// as the indexer CLI does, and records the name of each lookup. A name it does
// not hold is answered with no class.
func scriptedIndex(sources map[string]string, asked *[]string) func(args ...string) ([]byte, error) {
	return func(args ...string) ([]byte, error) {
		name := ""
		for i, arg := range args {
			if arg == "explore" && i+1 < len(args) {
				name = args[i+1]
			}
		}
		*asked = append(*asked, name)
		classes := []map[string]any{}
		if source, ok := sources[name]; ok {
			classes = append(classes, map[string]any{
				"className": name, "artifact": "com.example:nestlib:1.0.0", "language": "java", "source": source,
			})
		}
		return json.Marshal(map[string]any{"classes": classes})
	}
}

// nestedProjectExport collects testdata/nested-class and returns its endpoints
// by name, and the types it could not resolve.
func nestedProjectExport(t *testing.T, library resolver.DependencyResolver) (map[string]collector.ApiEndpoint, map[string]int) {
	t.Helper()
	abs, err := filepath.Abs("testdata/nested-class")
	if err != nil {
		t.Fatal(err)
	}
	unresolved := collector.NewUnresolvedSet()
	return nestedEndpoints(t, abs, library, unresolved), unresolved.Counts()
}

func nestedEndpoints(t *testing.T, dir string, dependencies resolver.DependencyResolver, unresolved *collector.UnresolvedSet) map[string]collector.ApiEndpoint {
	t.Helper()
	p, err := parser.NewParser(parser.ParserOptions{})
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()
	results, err := p.ParseDirectory(dir)
	if err != nil {
		t.Fatal(err)
	}

	sm := springmvc.NewParser()
	if dependencies != nil {
		sm.SetDependencyResolver(dependencies)
	}
	sm.SetUnresolved(unresolved)
	endpoints := map[string]collector.ApiEndpoint{}
	for _, ctrl := range sm.ExtractControllers(results) {
		for _, ep := range ctrl.Endpoints {
			endpoints[ep.MethodName] = springmvcEndpointToAPI(ep, ctrl.Name)
		}
	}
	return endpoints
}

func checkNestedExport(t *testing.T, endpoints map[string]collector.ApiEndpoint) {
	t.Helper()
	for name, want := range nestedResponseExpectation {
		ep, ok := endpoints[name]
		if !ok || ep.Response == nil {
			t.Errorf("endpoint %s has no response: %+v", name, ep)
			continue
		}
		if got := nestedShape(ep.Response.Body); got != want {
			t.Errorf("%s:\n got  %s\n want %s", name, got, want)
		}
	}
	create := endpoints["create"]
	if create.RequestBody == nil || create.RequestBody.Body == nil {
		t.Fatalf("create has no request body: %+v", create)
	}
	if got := nestedShape(create.RequestBody.Body); got != nestedBodyExpectation {
		t.Errorf("request body of create:\n got  %s\n want %s", got, nestedBodyExpectation)
	}
}

// nestedShape spells a model as its structure: an object by the shape of each of
// its fields in name order, an array by the shape of its items, any other by its
// type name.
func nestedShape(m *model.ObjectModel) string {
	switch {
	case m == nil:
		return "nil"
	case m.IsArray():
		return "[" + nestedShape(m.Items) + "]"
	case m.IsObject():
		names := make([]string, 0, len(m.Fields))
		for name := range m.Fields {
			names = append(names, name)
		}
		sort.Strings(names)
		parts := make([]string, len(names))
		for i, name := range names {
			parts[i] = name + ":" + nestedShape(m.Fields[name].Model)
		}
		return "{" + strings.Join(parts, ",") + "}"
	}
	return m.TypeName
}
