package kotlincollector

import (
	"path/filepath"
	"reflect"
	"sort"
	"testing"

	collector "github.com/tangcent/apilot/api-collector"
	ktparser "github.com/tangcent/apilot/api-collector-kotlin/parser"
	"github.com/tangcent/apilot/api-collector-kotlin/resolver"
	"github.com/tangcent/apilot/api-collector-kotlin/springmvc"
)

// enumInterfaceScenarios are the ways the supertype list of an enum names an
// interface. Each row has two enums that implement the same kind of interface:
// json implements one that carries @JsonValue on a member, so Jackson writes the
// value that member returns, and the property keeps the type name of the enum;
// plain implements one that declares the same member without the annotation, so
// the property is a string of its entries. Both enums are written in Kotlin,
// and every row is read from a holder written in Kotlin and from one written in
// Java: the Java resolver reads the interfaces of the enum through the names the
// Kotlin file wrote, so they have to mean in its scope what they meant there.
var enumInterfaceScenarios = []struct {
	field   string
	json    string
	plain   string
	entries []string
}{
	// The interface is imported under an alias, and the supertype list writes the alias.
	{"renamed", "RenamedJson", "RenamedPlain", []string{"RENAMED_ONE", "RENAMED_TWO"}},
	// ... which takes a type argument.
	{"generic", "GenericJson", "GenericPlain", []string{"GENERIC_ONE", "GENERIC_TWO"}},
	// ... which is nested in a class, and imported under an alias itself.
	{"nestedRenamed", "NestedRenamedJson", "NestedRenamedPlain", []string{"NESTED_ONE", "NESTED_TWO"}},
	// ... which extends, under an alias of its own file, the interface that carries @JsonValue.
	{"chain", "ChainJson", "ChainPlain", []string{"CHAIN_ONE", "CHAIN_TWO"}},
	// ... which is written in Java.
	{"javaInterface", "JavaJson", "JavaPlain", []string{"JAVA_ONE", "JAVA_TWO"}},
	// The interface is named by a typealias.
	{"typealias", "TypealiasJson", "TypealiasPlain", []string{"TYPEALIAS_ONE", "TYPEALIAS_TWO"}},
	// The package of the enum declares an interface of the simple name of the one
	// the file imports under an alias of that name, and the import wins.
	{"decoy", "DecoyJson", "DecoyPlain", []string{"DECOY_ONE", "DECOY_TWO"}},
	// The enum is nested in a class, and its interface is imported under an alias.
	{"boxed", "Box.Kind", "Box.Calm", []string{"BOX_ONE", "BOX_TWO"}},
	// The interface is named without an alias: they were read before and stay so.
	{"imported", "ImportedJson", "ImportedPlain", []string{"IMPORTED_ONE", "IMPORTED_TWO"}},
	{"samePackage", "SameJson", "SamePlain", []string{"SAME_ONE", "SAME_TWO"}},
	{"onDemand", "StarJson", "StarPlain", []string{"STAR_ONE", "STAR_TWO"}},
	{"qualified", "QualifiedJson", "QualifiedPlain", []string{"QUALIFIED_ONE", "QUALIFIED_TWO"}},
	{"outer", "OuterJson", "OuterPlain", []string{"OUTER_ONE", "OUTER_TWO"}},
}

func TestJSONViewReadsTheInterfacesOfAnEnumAsItsFileNamesThem(t *testing.T) {
	endpoints, c := collectDir(t, "testdata/enum-alias")
	byEndpoint := map[string]collector.ApiEndpoint{}
	for _, ep := range endpoints {
		byEndpoint[ep.Folder+"."+ep.Name] = ep
	}

	for _, holder := range []string{"KotlinController", "JavaController"} {
		create, page := byEndpoint[holder+".create"], byEndpoint[holder+".page"]
		if create.RequestBody == nil || create.RequestBody.Body == nil || page.Name == "" {
			t.Fatalf("endpoints = %+v, want %s with create and page", endpoints, holder)
		}
		params := map[string]collector.ApiParameter{}
		for _, p := range page.Parameters {
			params[p.Name] = p
		}
		for _, s := range enumInterfaceScenarios {
			t.Run(holder+"/"+s.field, func(t *testing.T) {
				var plain []string
				for _, entry := range s.entries {
					plain = append(plain, entry+": ")
				}
				checkEnumFields(t, "request body", create.RequestBody.Body.Fields, map[string]enumField{
					s.field:           {s.json, nil},
					s.field + "Plain": {"string", plain},
				})

				// Spring binds a parameter to an enum by the name of an entry, whatever
				// Jackson writes, so the form of the enum does not change the options.
				if got, want := params[s.field].Enum, []string{"A", "B"}; !reflect.DeepEqual(got, want) {
					t.Errorf("parameter %s enum = %q, want %q", s.field, got, want)
				}
				if got := params[s.field+"Plain"].Enum; !reflect.DeepEqual(got, s.entries) {
					t.Errorf("parameter %sPlain enum = %q, want %q", s.field, got, s.entries)
				}
			})
		}
	}

	// The enums that implement an interface with @JsonValue are reported by their
	// type names: no more, and none of the enums that do not.
	var want []string
	for _, s := range enumInterfaceScenarios {
		want = append(want, s.json)
	}
	sort.Strings(want)
	var names []string
	for name := range c.Unresolved() {
		names = append(names, name)
	}
	sort.Strings(names)
	if !reflect.DeepEqual(names, want) {
		t.Errorf("unresolved = %v, want %v", names, want)
	}
}

// An interface the enum implements may come from a dependency: it is imported
// under an alias, or by its own name, and the library is asked for it by its
// qualified name either way. The library's Valued carries @JsonValue and its
// Coded does not.
func TestJSONViewReadsTheInterfaceOfAnEnumFromADependencyImportedUnderAnAlias(t *testing.T) {
	p, err := ktparser.NewParser()
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()
	dir, _ := filepath.Abs("testdata/enum-alias-dependency")
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
	checkEnumFields(t, "request body", create.RequestBody.Body.Fields, map[string]enumField{
		"byAlias":      {"ByAliasJson", nil},
		"byAliasPlain": {"string", []string{"ALIAS_ONE: ", "ALIAS_TWO: "}},
		"byName":       {"ByNameJson", nil},
		"byNamePlain":  {"string", []string{"NAME_ONE: ", "NAME_TWO: "}},
	})

	var names []string
	for name := range typeResolver.Unresolved() {
		names = append(names, name)
	}
	sort.Strings(names)
	if want := []string{"ByAliasJson", "ByNameJson"}; !reflect.DeepEqual(names, want) {
		t.Errorf("unresolved = %v, want %v", names, want)
	}

	// The library is asked for each interface once, by its qualified name, however
	// many enums implement it and however the file spells it.
	asked := map[string]int{}
	for _, name := range library.queried {
		asked[name]++
	}
	if want := map[string]int{"com.example.lib.Valued": 1, "com.example.lib.Coded": 1}; !reflect.DeepEqual(asked, want) {
		t.Errorf("the library was asked for %v, want %v", asked, want)
	}
}
