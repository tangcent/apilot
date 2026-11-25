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

// starBodyExpectation and starResponseExpectation are the export of
// testdata/star-import, a controller that names every class of its library
// through an import on demand (`import com.example.lib.model.*`): the fields
// of the request body and of the response, each as its kind and, for an object,
// its field names. StarOrder lives in one package of the library and
// StarEnvelope in another; StarOrder refers to StarAddress and StarItem without
// an import, as classes of its own package, and the response carries StarItem
// as a type argument. StarMissing is declared nowhere.
//
// The Java collector asserts the same tables for the same declarations written
// in Java (api-collector-java/star_import_test.go): keep the two in sync.
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

// starLibrarySources are the classes of the library, as the indexer returns a
// class of a Kotlin sources JAR.
var starLibrarySources = map[string]string{
	"com.example.lib.model.StarOrder": `package com.example.lib.model

class StarOrder {
    var no: String? = null
    var address: StarAddress? = null
    var items: List<StarItem> = emptyList()
}
`,
	"com.example.lib.model.StarAddress": `package com.example.lib.model

class StarAddress {
    var city: String? = null
}
`,
	"com.example.lib.model.StarItem": `package com.example.lib.model

class StarItem {
    var sku: String = ""
    var qty: Int = 0
}
`,
	"com.example.lib.response.StarEnvelope": `package com.example.lib.response

class StarEnvelope<T> {
    var code: Int = 0
    var data: T? = null
}
`,
}

// starLibrary answers dependency lookups from starLibrarySources by qualified
// name, as the indexer does, and records each lookup.
type starLibrary struct {
	t     *testing.T
	asked []string
	// sources replaces starLibrarySources when set.
	sources map[string]string
}

func (l *starLibrary) ResolveClass(name string) *javaparser.Class {
	l.asked = append(l.asked, name)
	sources := l.sources
	if sources == nil {
		sources = starLibrarySources
	}
	source, ok := sources[name]
	if !ok {
		return nil
	}
	classes, err := parseDependencySource(name, []byte(source))
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
	p, err := ktparser.NewParser()
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()
	dir, _ := filepath.Abs("testdata/star-import")
	results, err := p.ParseDirectory(dir)
	if err != nil {
		t.Fatal(err)
	}

	typeResolver := resolver.New(flattenClasses(results))
	library := &starLibrary{t: t}
	typeResolver.SetDependencyResolver(library)
	endpoints := map[string]collector.ApiEndpoint{}
	for _, ctrl := range springmvc.NewParser(typeResolver).ExtractControllers(results) {
		for _, ep := range ctrl.Endpoints {
			endpoints[ep.MethodName] = springMVCEndpointToAPI(ep, ctrl.Name)
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
	if got := typeResolver.Unresolved(); !reflect.DeepEqual(got, map[string]int{"StarMissing": 1}) {
		t.Errorf("unresolved = %v, want only StarMissing", got)
	}

	// Every qualified name is asked for once, a miss included, and no platform
	// package is tried: the library is a process of the indexer per name.
	asked := map[string]int{}
	for _, name := range library.asked {
		asked[name]++
		if strings.HasPrefix(name, "kotlin.") || strings.HasPrefix(name, "java.") {
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
	// The package of the controller belongs to the scanned sources and is not
	// asked: the first lookup is the first library package the controller
	// imports.
	if want := "com.example.lib.model.StarOrder"; library.asked[0] != want {
		t.Errorf("first lookup = %v, want %v", library.asked[0], want)
	}
	for _, name := range library.asked {
		if strings.HasPrefix(name, "com.example.ondemand.") {
			t.Errorf("%s, a package of the scanned sources, was asked for: %v", name, library.asked)
		}
	}
}

// A class of a library finds its neighbour in its own package before the one an
// import on demand names, whether or not the class of the import was read first.
func TestAPackageOfALibraryClassComesBeforeItsImportOnDemand(t *testing.T) {
	sources := map[string]string{
		"lib.a.Holder": `package lib.a

import lib.b.*

class Holder {
    var shared: Shared? = null
}
`,
		"lib.a.Shared": `package lib.a

class Shared {
    var fromSamePackage: String = ""
}
`,
		"lib.b.Shared": `package lib.b

class Shared {
    var fromOnDemand: String = ""
}
`,
	}
	caller := &ktparser.Class{Name: "OrderController", Package: "app", FilePath: "OrderController.kt"}

	for _, preload := range []string{"", "lib.b.Shared"} {
		t.Run("preloaded "+preload, func(t *testing.T) {
			typeResolver := resolver.New(nil)
			typeResolver.SetDependencyResolver(&starLibrary{t: t, sources: sources})
			if preload != "" {
				typeResolver.ResolveFrom(caller, preload, nil)
			}

			holder := typeResolver.ResolveFrom(caller, "lib.a.Holder", nil)
			field := holder.Fields["shared"]
			if field == nil {
				t.Fatalf("Holder = %+v, want a shared field", holder)
			}
			if got, want := starModelKind(field.Model), "object{fromSamePackage}"; got != want {
				t.Errorf("shared = %s, want %s", got, want)
			}
		})
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

// A class of a library names its neighbours without an import, and a class of
// the project that carries the same simple name does not take their place,
// whether or not the project's class was read first.
func TestAClassOfALibraryNamesItsOwnPackageBeforeAClassOfTheProject(t *testing.T) {
	sources := map[string]string{
		"lib.req.RateConfigRequest": `package lib.req

class RateConfigRequest {
    var tiers: List<TierItem> = emptyList()
    var files: List<FileItem>? = null
}
`,
		"lib.req.TierItem": `package lib.req

class TierItem {
    var libraryTier: String = ""
}
`,
		"lib.req.FileItem": `package lib.req

class FileItem {
    var libraryFile: String = ""
}
`,
	}
	caller := ktparser.Class{
		Name: "RateController", Package: "app.web", FilePath: "RateController.kt",
		Imports: []ktparser.Import{{Path: "app.dto.TierItem"}, {Path: "lib.req.RateConfigRequest"}},
	}
	project := []ktparser.Class{
		{Name: "TierItem", Package: "app.dto", FilePath: "TierItem.kt", Fields: []ktparser.Property{{Name: "projectTier", Type: "String"}}},
		{Name: "FileItem", Package: "app.files", FilePath: "FileItem.kt", Fields: []ktparser.Property{{Name: "projectFile", Type: "String"}}},
		caller,
	}

	for _, readFirst := range []string{"", "TierItem"} {
		t.Run("project class read first "+readFirst, func(t *testing.T) {
			typeResolver := resolver.New(project)
			typeResolver.SetDependencyResolver(&starLibrary{t: t, sources: sources})
			if readFirst != "" {
				if got, want := starModelKind(typeResolver.ResolveFrom(&caller, readFirst, nil)), "object{projectTier}"; got != want {
					t.Fatalf("%s = %s, want %s", readFirst, got, want)
				}
			}

			request := typeResolver.ResolveFrom(&caller, "RateConfigRequest", nil)
			want := map[string]string{"tiers": "array<object{libraryTier}>", "files": "array<object{libraryFile}>"}
			if got := starFieldKinds(request); !reflect.DeepEqual(got, want) {
				t.Errorf("RateConfigRequest:\n got  %v\n want %v", got, want)
			}
		})
	}
}

// A class of a library written in Kotlin that names a class by its qualified name
// keeps that class when a class of the same simple name sits in the library's own
// package, whether or not the library's own Item was read first.
func TestAClassOfALibraryKeepsTheQualifiedClassItNames(t *testing.T) {
	sources := map[string]string{
		"lib.Holder": `package lib

class Holder {
    var items: List<app.dto.Item> = emptyList()
    var box: Box<app.dto.Item>? = null
    var own: Item? = null
}
`,
		"lib.Box": `package lib

class Box<T> {
    var value: T? = null
}
`,
		"lib.Item": `package lib

class Item {
    var libraryItem: String = ""
}
`,
	}
	caller := ktparser.Class{Name: "HolderController", Package: "app.web", FilePath: "HolderController.kt"}
	project := []ktparser.Class{
		{Name: "Item", Package: "app.dto", FilePath: "Item.kt", Fields: []ktparser.Property{{Name: "projectItem", Type: "String"}}},
		caller,
	}

	for _, readFirst := range []string{"", "lib.Item", "app.dto.Item"} {
		t.Run("read first "+readFirst, func(t *testing.T) {
			typeResolver := resolver.New(project)
			typeResolver.SetDependencyResolver(&starLibrary{t: t, sources: sources})
			if readFirst != "" {
				typeResolver.ResolveFrom(&caller, readFirst, nil)
			}

			holder := typeResolver.ResolveFrom(&caller, "lib.Holder", nil)
			want := map[string]string{"items": "array<object{projectItem}>", "box": "object{value}", "own": "object{libraryItem}"}
			if got := starFieldKinds(holder); !reflect.DeepEqual(got, want) {
				t.Errorf("Holder:\n got  %v\n want %v", got, want)
			}
			if got, want := starModelKind(holder.Fields["box"].Model.Fields["value"].Model), "object{projectItem}"; got != want {
				t.Errorf("box.value = %s, want %s", got, want)
			}
		})
	}
}

// The scope of a name is walked tier by tier whatever the declarations come from:
// the package of a library class comes before an import on demand when the
// wildcard's class is one of the project's, and the other way round for a source
// of the project that imports a library on demand while another class of the
// project carries the name elsewhere.
func TestATierOfTheScopeAnswersWhateverKindOfDeclarationItHolds(t *testing.T) {
	sources := map[string]string{
		"lib.a.Holder": `package lib.a

import lib.b.*

class Holder {
    var shared: Shared? = null
}
`,
		"lib.a.Shared": `package lib.a

class Shared {
    var fromSamePackage: String = ""
}
`,
		"lib.Status": `package lib

class Status {
    var fromLibrary: String = ""
}
`,
	}
	caller := ktparser.Class{
		Name: "StatusController", Package: "app.web", FilePath: "StatusController.kt",
		Imports: []ktparser.Import{{Path: "lib", OnDemand: true}},
	}
	project := []ktparser.Class{
		// Declared in the project, in the package a library class imports on demand.
		{Name: "Shared", Package: "lib.b", FilePath: "b/Shared.kt", Fields: []ktparser.Property{{Name: "fromOnDemand", Type: "String"}}},
		// Declared in the project, where the controller cannot reach it.
		{Name: "Status", Package: "app.model", FilePath: "Status.kt", Fields: []ktparser.Property{{Name: "fromProject", Type: "String"}}},
		caller,
	}

	for _, readFirst := range []string{"", "lib.a.Shared", "lib.Status"} {
		t.Run("read first "+readFirst, func(t *testing.T) {
			typeResolver := resolver.New(project)
			typeResolver.SetDependencyResolver(&starLibrary{t: t, sources: sources})
			if readFirst != "" {
				typeResolver.ResolveFrom(&caller, readFirst, nil)
			}

			holder := typeResolver.ResolveFrom(&caller, "lib.a.Holder", nil)
			if got, want := starModelKind(holder.Fields["shared"].Model), "object{fromSamePackage}"; got != want {
				t.Errorf("shared = %s, want %s", got, want)
			}
			if got, want := starModelKind(typeResolver.ResolveFrom(&caller, "Status", nil)), "object{fromLibrary}"; got != want {
				t.Errorf("Status from the controller = %s, want %s", got, want)
			}
		})
	}
}
