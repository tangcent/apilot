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
	// sources replaces starLibrarySources when set.
	sources map[string]string
}

func (l *starLibrary) ResolveClass(name string) *parser.Class {
	l.asked = append(l.asked, name)
	sources := l.sources
	if sources == nil {
		sources = starLibrarySources
	}
	source, ok := sources[name]
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
	dir := t.TempDir()
	writeTree(t, dir, map[string]string{
		"src/main/java/app/dto/TierItem.java": `package app.dto;

public class TierItem {
    private String projectTier;
}
`,
		"src/main/java/app/files/FileItem.java": `package app.files;

public class FileItem {
    private String projectFile;
}
`,
		// The project's TierItem is read for the first endpoint, before the
		// library class that names a TierItem of its own.
		"src/main/java/app/web/RateController.java": `package app.web;

import app.dto.TierItem;
import java.util.List;
import lib.req.RateConfigRequest;
import org.springframework.web.bind.annotation.*;

@RestController
@RequestMapping("/rate")
public class RateController {

    @GetMapping("/tiers")
    public List<TierItem> tiers() {
        return null;
    }

    @PostMapping("/save")
    public String save(@RequestBody RateConfigRequest request) {
        return null;
    }
}
`,
	})
	sources := map[string]string{
		"lib.req.RateConfigRequest": `package lib.req;

import java.util.List;

public class RateConfigRequest {
    private List<TierItem> tiers;
    private List<FileItem> files;
}
`,
		"lib.req.TierItem": `package lib.req;

public class TierItem {
    private String libraryTier;
}
`,
		"lib.req.FileItem": `package lib.req;

public class FileItem {
    private String libraryFile;
}
`,
	}

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
	sm.SetDependencyResolver(&starLibrary{t: t, parser: p, sources: sources})
	endpoints := map[string]collector.ApiEndpoint{}
	for _, ctrl := range sm.ExtractControllers(results) {
		for _, ep := range ctrl.Endpoints {
			endpoints[ep.MethodName] = springmvcEndpointToAPI(ep, ctrl.Name)
		}
	}

	tiers, save := endpoints["tiers"], endpoints["save"]
	if tiers.Response == nil || save.RequestBody == nil || save.RequestBody.Body == nil {
		t.Fatalf("tiers = %+v, save = %+v, want a response and a request body", tiers, save)
	}
	if got, want := starModelKind(tiers.Response.Body), "array<object{projectTier}>"; got != want {
		t.Errorf("response of tiers = %s, want %s", got, want)
	}
	want := map[string]string{"tiers": "array<object{libraryTier}>", "files": "array<object{libraryFile}>"}
	if got := starFieldKinds(save.RequestBody.Body); !reflect.DeepEqual(got, want) {
		t.Errorf("request body:\n got  %v\n want %v", got, want)
	}
}

// A class of a library that writes a class by its qualified name keeps that
// class when a class of the same simple name sits in the library's own package:
// `List<app.dto.Item>` and `Box<app.dto.Item>` name the project's Item, and
// `Item` written simply is the library's own, whether or not the library's Item
// was read first.
func TestAClassOfALibraryKeepsTheQualifiedClassItNames(t *testing.T) {
	sources := map[string]string{
		"lib.Holder": `package lib;

import java.util.List;

public class Holder {
    private List<app.dto.Item> items;
    private Box<app.dto.Item> box;
    private Item own;
}
`,
		"lib.Box": `package lib;

public class Box<T> {
    private T value;
}
`,
		"lib.Item": `package lib;

public class Item {
    private String libraryItem;
}
`,
	}
	for name, controller := range map[string]string{
		// The library's Item is read for the first endpoint, before Holder's.
		"library Item read first": `
    @GetMapping("/item")
    public Item item() {
        return null;
    }

    @GetMapping("/holder")
    public Holder holder() {
        return null;
    }
`,
		"nothing read first": `
    @GetMapping("/holder")
    public Holder holder() {
        return null;
    }
`,
	} {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			writeTree(t, dir, map[string]string{
				"src/main/java/app/dto/Item.java": `package app.dto;

public class Item {
    private String projectItem;
}
`,
				"src/main/java/app/web/HolderController.java": `package app.web;

import lib.Holder;
import lib.Item;
import org.springframework.web.bind.annotation.*;

@RestController
public class HolderController {
` + controller + `}
`,
			})
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
			sm.SetDependencyResolver(&starLibrary{t: t, parser: p, sources: sources})
			endpoints := map[string]collector.ApiEndpoint{}
			for _, ctrl := range sm.ExtractControllers(results) {
				for _, ep := range ctrl.Endpoints {
					endpoints[ep.MethodName] = springmvcEndpointToAPI(ep, ctrl.Name)
				}
			}

			holder := endpoints["holder"]
			if holder.Response == nil {
				t.Fatalf("holder = %+v, want a response", holder)
			}
			want := map[string]string{"items": "array<object{projectItem}>", "box": "object{value}", "own": "object{libraryItem}"}
			if got := starFieldKinds(holder.Response.Body); !reflect.DeepEqual(got, want) {
				t.Errorf("response:\n got  %v\n want %v", got, want)
			}
			// The type argument of Box is the project's class too.
			for _, field := range holder.Response.Body.Fields["box"].Model.Fields {
				if got, want := starModelKind(field.Model), "object{projectItem}"; got != want {
					t.Errorf("box.value = %s, want %s", got, want)
				}
			}
		})
	}
}
