package jaxrs

import (
	"sort"
	"strings"
	"testing"

	"github.com/tangcent/apilot/api-collector-java/parser"
	model "github.com/tangcent/apilot/api-model"
)

// kindOf spells a model as its kind, an object by its field names.
func kindOf(m *model.ObjectModel) string {
	switch {
	case m == nil:
		return "nil"
	case m.IsArray():
		return "array<" + kindOf(m.Items) + ">"
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

// The arguments of an extends clause are written in the scope of the subclass:
// the generic classes it extends sit in a package of their own, which declares
// an Item of its own too. Each level binds the parameters of its own
// superclass: the two levels name theirs alike, and a parameter may be handed
// on as it is to the level above.
func TestParser_InheritedEndpointsBindTheClassesTheSubclassNames(t *testing.T) {
	get := func(name, returnType string) parser.Method {
		return parser.Method{
			Name:        name,
			Annotations: []parser.Annotation{{Name: "GET"}, {Name: "Path", Params: map[string]string{"value": "/" + name}}},
			ReturnType:  returnType,
		}
	}
	results := []parser.ParseResult{{
		Classes: []parser.Class{
			{Name: "Item", Package: "com.example.lib", FilePath: "lib/Item.java", Fields: []parser.Field{{Name: "code", Type: "String"}}},
			{Name: "Item", Package: "com.example.app.dto", FilePath: "app/dto/Item.java", Fields: []parser.Field{{Name: "name", Type: "String"}}},
			{
				Name: "BaseResource", Package: "com.example.lib", FilePath: "lib/BaseResource.java",
				TypeParameters: []string{"T"},
				Imports:        []parser.Import{{Path: "java.util.List"}},
				Methods:        []parser.Method{get("one", "T"), get("all", "List<T>")},
			},
			{
				Name: "MidResource", Package: "com.example.lib", FilePath: "lib/MidResource.java",
				TypeParameters: []string{"T"},
				SuperClass:     "BaseResource", SuperClassTypeArgs: []string{"List<T>"},
				Methods: []parser.Method{get("mid", "T")},
			},
			// A parameter handed on as it is: Relay<M> extends Base<M>.
			{
				Name: "RelayResource", Package: "com.example.lib", FilePath: "lib/RelayResource.java",
				TypeParameters: []string{"M"},
				SuperClass:     "BaseResource", SuperClassTypeArgs: []string{"M"},
			},
			{
				Name: "RelayItemResource", Package: "com.example.app", FilePath: "app/RelayItemResource.java",
				Annotations: []parser.Annotation{{Name: "Path", Params: map[string]string{"value": "/relay"}}},
				Imports: []parser.Import{
					{Path: "com.example.app.dto.Item"}, {Path: "com.example.lib.RelayResource"},
				},
				SuperClass: "RelayResource", SuperClassTypeArgs: []string{"Item"},
			},
			{
				Name: "ItemResource", Package: "com.example.app", FilePath: "app/ItemResource.java",
				Annotations: []parser.Annotation{{Name: "Path", Params: map[string]string{"value": "/items"}}},
				Imports: []parser.Import{
					{Path: "com.example.app.dto.Item"}, {Path: "com.example.lib.MidResource"},
				},
				SuperClass: "MidResource", SuperClassTypeArgs: []string{"Item"},
			},
		},
	}}

	resources := NewParser().ExtractResources(results)

	got := map[string]map[string]string{}
	for _, resource := range resources {
		got[resource.Name] = map[string]string{}
		for _, ep := range resource.Endpoints {
			got[resource.Name][ep.MethodName] = kindOf(ep.ResponseSchema)
		}
	}
	want := map[string]map[string]string{
		"ItemResource": {
			"mid": "object{name}",
			// BaseResource<List<T>> under T=Item.
			"one": "array<object{name}>",
			"all": "array<array<object{name}>>",
		},
		"RelayItemResource": {
			"one": "object{name}",
			"all": "array<object{name}>",
		},
	}
	for name, kinds := range want {
		for method, kind := range kinds {
			if got[name][method] != kind {
				t.Errorf("%s.%s = %s, want %s (endpoints: %v)", name, method, got[name][method], kind, got[name])
			}
		}
	}
}
