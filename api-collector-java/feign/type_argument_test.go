package feign

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
			Annotations: []parser.Annotation{{Name: "GetMapping", Params: map[string]string{"value": "/" + name}}},
			ReturnType:  returnType,
		}
	}
	results := []parser.ParseResult{{
		Classes: []parser.Class{
			{Name: "Item", Package: "com.example.lib", FilePath: "lib/Item.java", Fields: []parser.Field{{Name: "code", Type: "String"}}},
			{Name: "Item", Package: "com.example.app.dto", FilePath: "app/dto/Item.java", Fields: []parser.Field{{Name: "name", Type: "String"}}},
			{
				Name: "BaseClient", Package: "com.example.lib", FilePath: "lib/BaseClient.java",
				IsInterface: true, TypeParameters: []string{"T"},
				Imports: []parser.Import{{Path: "java.util.List"}},
				Methods: []parser.Method{get("one", "T"), get("all", "List<T>")},
			},
			{
				Name: "MidClient", Package: "com.example.lib", FilePath: "lib/MidClient.java",
				IsInterface: true, TypeParameters: []string{"T"},
				SuperClass: "BaseClient", SuperClassTypeArgs: []string{"List<T>"},
				Methods: []parser.Method{get("mid", "T")},
			},
			// A parameter handed on as it is: Relay<M> extends Base<M>.
			{
				Name: "RelayClient", Package: "com.example.lib", FilePath: "lib/RelayClient.java",
				IsInterface: true, TypeParameters: []string{"M"},
				SuperClass: "BaseClient", SuperClassTypeArgs: []string{"M"},
			},
			{
				Name: "RelayItemClient", Package: "com.example.app", FilePath: "app/RelayItemClient.java",
				IsInterface: true,
				Annotations: []parser.Annotation{{Name: "FeignClient", Params: map[string]string{"name": "relay"}}},
				Imports: []parser.Import{
					{Path: "com.example.app.dto.Item"}, {Path: "com.example.lib.RelayClient"},
				},
				SuperClass: "RelayClient", SuperClassTypeArgs: []string{"Item"},
			},
			{
				Name: "ItemClient", Package: "com.example.app", FilePath: "app/ItemClient.java",
				IsInterface: true,
				Annotations: []parser.Annotation{{Name: "FeignClient", Params: map[string]string{"name": "items"}}},
				Imports: []parser.Import{
					{Path: "com.example.app.dto.Item"}, {Path: "com.example.lib.MidClient"},
				},
				SuperClass: "MidClient", SuperClassTypeArgs: []string{"Item"},
			},
		},
	}}

	clients := NewParser().ExtractClients(results)

	got := map[string]map[string]string{}
	for _, client := range clients {
		got[client.Name] = map[string]string{}
		for _, ep := range client.Endpoints {
			got[client.Name][ep.MethodName] = kindOf(ep.ResponseSchema)
		}
	}
	want := map[string]map[string]string{
		"ItemClient": {
			"mid": "object{name}",
			// BaseClient<List<T>> under T=Item.
			"one": "array<object{name}>",
			"all": "array<array<object{name}>>",
		},
		"RelayItemClient": {
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
