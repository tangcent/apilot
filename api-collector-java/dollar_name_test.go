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

// decompiledHeader is what CFR writes above a class it decompiled.
const decompiledHeader = "/*\n * Decompiled with CFR 0.152.\n */\n"

// dollarBothLibrary holds a top-level class Price$Value and a class Value nested in
// Price: the same text names either, and which one it is depends on where the text
// comes from.
var dollarBothLibrary = map[string]string{
	"lib.Price$Value": "package lib;\n\npublic class Price$Value {\n    private String top;\n}\n",
	"lib.Price":       "package lib;\n\npublic class Price {\n    public static class Value {\n        private String nested;\n    }\n}\n",
	// Written by hand, `Price$Value` is a name like any other: the top-level class.
	"lib.H": "package lib;\n\npublic class H {\n    private Price$Value pv;\n    private Price.Value nv;\n    private lib.Price$Value fq;\n}\n",
	// Decompiled, it is the binary name of a class that bytecode names so: the class
	// nested in Price.
	"lib.D": decompiledHeader + "package lib;\n\npublic class D {\n    private Price$Value pv;\n    private Price.Value nv;\n}\n",
}

// A `$` written in source is a character of the name: `Price$Value` is the class of
// that name, whichever class `Price.Value` is and whatever was read before. Only the
// names that bytecode gives a nested class, `Outer$Inner`, are binary names, and
// those come from a decompiled source.
func TestADollarWrittenInSourceIsPartOfTheNameButADecompiledOneIsABinaryName(t *testing.T) {
	bare := parser.Class{Name: "Bare", Package: "app", FilePath: "Bare.java"}
	starred := parser.Class{Name: "Starred", Package: "app", FilePath: "Starred.java", Imports: []parser.Import{{Path: "lib", OnDemand: true}}}

	for _, tc := range []struct {
		name    string
		source  parser.Class
		rawType string
		want    string
		warm    []string
	}{
		{"project, in full", bare, "lib.Price$Value", "{top:string}", nil},
		{"project, in full, the nested class read first", bare, "lib.Price$Value", "{top:string}", []string{"lib.Price"}},
		{"project, through an import on demand", starred, "Price$Value", "{top:string}", nil},
		{"project, through an import on demand, the nested class read first", starred, "Price$Value", "{top:string}", []string{"lib.Price.Value"}},
		{"project, the nested class by its path", starred, "Price.Value", "{nested:string}", []string{"lib.Price$Value"}},
		{"library source, by hand", bare, "lib.H", "{fq:{top:string},nv:{nested:string},pv:{top:string}}", nil},
		{"library source, by hand, the nested class read first", bare, "lib.H", "{fq:{top:string},nv:{nested:string},pv:{top:string}}", []string{"lib.Price"}},
		{"library source, decompiled", bare, "lib.D", "{nv:{nested:string},pv:{nested:string}}", nil},
		{"library source, decompiled, the top-level class read first", bare, "lib.D", "{nv:{nested:string},pv:{nested:string}}", []string{"lib.Price$Value"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var asked []string
			index, err := maven.NewMavenDependencyResolver()
			if err != nil {
				t.Fatal(err)
			}
			defer index.Close()
			index.SetRunner(scriptedIndex(dollarBothLibrary, &asked))
			r := resolver.NewTypeResolver([]parser.Class{bare, starred})
			r.SetDependencyResolver(index)
			for _, w := range tc.warm {
				r.ResolveFrom(bare, w, nil)
			}

			if got := nestedShape(r.ResolveFrom(tc.source, tc.rawType, nil)); got != tc.want {
				t.Errorf("%s:\n got  %s\n want %s\n asked %v", tc.rawType, got, tc.want, asked)
			}
		})
	}
}

// A decompiled source reads the name as a binary name first, so a library of
// decompiled classes costs no call for a name that nothing is named so, and a name
// written by hand is asked for as written first.
func TestADollarNameIsAskedForInTheOrderItsSourceReadsIt(t *testing.T) {
	nestedOnly := map[string]string{
		"lib.Price": dollarBothLibrary["lib.Price"],
		"lib.D":     dollarBothLibrary["lib.D"],
		"lib.H":     dollarBothLibrary["lib.H"],
	}
	for _, tc := range []struct {
		name  string
		class string
		want  []string
		shape string
	}{
		{"decompiled: the binary name only", "lib.D", []string{"lib.D", "lib.Price"}, "{nv:{nested:string},pv:{nested:string}}"},
		// By hand: the class of that name is looked for, and only the nested class
		// is there, which the binary reading finds in the end.
		{"by hand: the binary name is the fall back", "lib.H", []string{"lib.H", "lib.Price$Value", "lib.Price"}, "{fq:{nested:string},nv:{nested:string},pv:{nested:string}}"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var asked []string
			index, err := maven.NewMavenDependencyResolver()
			if err != nil {
				t.Fatal(err)
			}
			defer index.Close()
			index.SetRunner(scriptedIndex(nestedOnly, &asked))
			bare := parser.Class{Name: "Bare", Package: "app", FilePath: "Bare.java"}
			r := resolver.NewTypeResolver([]parser.Class{bare})
			r.SetDependencyResolver(index)

			if got := nestedShape(r.ResolveFrom(bare, tc.class, nil)); got != tc.shape {
				t.Errorf("%s = %s, want %s", tc.class, got, tc.shape)
			}
			if !reflect.DeepEqual(asked, tc.want) {
				t.Errorf("asked = %v, want %v", asked, tc.want)
			}
		})
	}
}
