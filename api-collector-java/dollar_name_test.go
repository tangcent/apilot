package javacollector

import (
	"reflect"
	"testing"

	"github.com/tangcent/apilot/api-collector-java/maven"
	"github.com/tangcent/apilot/api-collector-java/parser"
	"github.com/tangcent/apilot/api-collector-java/resolver"
)

// dollarLibrary are the classes of a library whose top-level class is named
// Price$Value, a `$` being a character of an identifier. The index holds it under
// that name, and holds no class Price.
var dollarLibrary = map[string]string{
	"lib.Price$Value": "package lib;\n\npublic class Price$Value {\n    private long price;\n}\n",
	"lib.Box":         "package lib;\n\npublic class Box<T> {\n    private T value;\n}\n",
	"lib.Holder":      "package lib;\n\npublic class Holder {\n    private Price$Value value;\n    private Box<Price$Value> box;\n}\n",
}

// A name that reads as the binary name of a nested class is read as one first,
// and as written when that finds no class. The spelling the source wrote travels
// to the index untouched, so the class is found however the name reaches the
// resolver: written in full, through an import on demand, as the type argument of
// a generic class, or inside a class of the library, which names it in its own
// package.
func TestADollarNameOfALibraryClassKeepsItsSpellingThroughTheIndex(t *testing.T) {
	imported := parser.Class{Name: "Imported", Package: "app", FilePath: "Imported.java", Imports: []parser.Import{{Path: "lib.Price$Value"}}}
	starred := parser.Class{Name: "Starred", Package: "app", FilePath: "Starred.java", Imports: []parser.Import{{Path: "lib", OnDemand: true}}}
	bare := parser.Class{Name: "Bare", Package: "app", FilePath: "Bare.java"}

	for _, tc := range []struct {
		name    string
		source  parser.Class
		rawType string
		want    string
	}{
		{"written in full", bare, "lib.Price$Value", "{price:long}"},
		{"written in full, an array of it", bare, "lib.Price$Value[]", "[{price:long}]"},
		{"written in full, in a list", bare, "List<lib.Price$Value>", "[{price:long}]"},
		{"written in full, in a generic class", bare, "lib.Box<lib.Price$Value>", "{value:{price:long}}"},
		{"written in full, in a wildcard bound", bare, "lib.Box<? extends lib.Price$Value>", "{value:{price:long}}"},
		{"through an import on demand", starred, "Price$Value", "{price:long}"},
		{"through an import on demand, in a list", starred, "List<Price$Value>", "[{price:long}]"},
		{"through an import on demand, in a generic class", starred, "Box<Price$Value>", "{value:{price:long}}"},
		{"through a single-type import", imported, "Price$Value", "{price:long}"},
		{"inside a class of the library", bare, "lib.Holder", "{box:{value:{price:long}},value:{price:long}}"},
	} {
		for _, warm := range []bool{false, true} {
			name := tc.name
			if warm {
				name += ", the class read before"
			}
			t.Run(name, func(t *testing.T) {
				var asked []string
				index, err := maven.NewMavenDependencyResolver()
				if err != nil {
					t.Fatal(err)
				}
				defer index.Close()
				index.SetRunner(scriptedIndex(dollarLibrary, &asked))
				r := resolver.NewTypeResolver([]parser.Class{imported, starred, bare})
				r.SetDependencyResolver(index)
				if warm {
					r.Resolve("lib.Price$Value", nil)
				}

				if got := nestedShape(r.ResolveFrom(tc.source, tc.rawType, nil)); got != tc.want {
					t.Errorf("%s:\n got  %s\n want %s\n asked %v", tc.rawType, got, tc.want, asked)
				}
			})
		}
	}
}

// The index is asked for a class once, whichever spelling reaches it.
func TestADollarNameIsAskedOfTheIndexOnce(t *testing.T) {
	var asked []string
	index, err := maven.NewMavenDependencyResolver()
	if err != nil {
		t.Fatal(err)
	}
	defer index.Close()
	index.SetRunner(scriptedIndex(dollarLibrary, &asked))
	bare := parser.Class{Name: "Bare", Package: "app", FilePath: "Bare.java"}
	r := resolver.NewTypeResolver([]parser.Class{bare})
	r.SetDependencyResolver(index)

	for i := 0; i < 3; i++ {
		r.ResolveFrom(bare, "lib.Price$Value", nil)
		r.ResolveFrom(bare, "List<lib.Price$Value>", nil)
	}
	count := map[string]int{}
	for _, name := range asked {
		count[name]++
	}
	for name, n := range count {
		if n != 1 {
			t.Errorf("%s was asked of the index %d times: %v", name, n, asked)
		}
	}
	if !reflect.DeepEqual(count["lib.Price$Value"], 1) {
		t.Errorf("lib.Price$Value was asked %d times, want once: %v", count["lib.Price$Value"], asked)
	}
}
