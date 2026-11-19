package resolver

import (
	"reflect"
	"testing"

	"github.com/tangcent/apilot/api-collector-java/parser"
	model "github.com/tangcent/apilot/api-model"
)

// libraryClasses answers dependency lookups by qualified name only, as the
// indexer does: a simple name finds nothing.
type libraryClasses struct {
	classes map[string]parser.Class
	asked   []string
}

func (l *libraryClasses) ResolveClass(name string) *parser.Class {
	l.asked = append(l.asked, name)
	if c, ok := l.classes[name]; ok {
		return &c
	}
	return nil
}

func TestResolve_TypeArgumentsAreLookedUpByTheNameTheCallerImports(t *testing.T) {
	library := &libraryClasses{classes: map[string]parser.Class{
		"com.example.lib.Wrapper": {
			Name: "Wrapper", Package: "com.example.lib", TypeParameters: []string{"T"},
			Fields: []parser.Field{{Name: "code", Type: "int"}, {Name: "data", Type: "T"}},
		},
		"com.example.lib.PageData": {
			Name: "PageData", Package: "com.example.lib", TypeParameters: []string{"T"},
			Fields: []parser.Field{{Name: "total", Type: "long"}, {Name: "rows", Type: "List<T>"}},
		},
	}}
	// The controller whose return types are resolved, as an endpoint's are.
	controller := parser.Class{
		Name: "OrderController", Package: "com.example.app", FilePath: "OrderController.java",
		Imports: []parser.Import{
			{Path: "com.example.lib.Wrapper"}, {Path: "com.example.lib.PageData"}, {Path: "java.util.List"},
		},
	}
	order := parser.Class{
		Name: "Order", Package: "com.example.app", FilePath: "Order.java",
		Fields: []parser.Field{{Name: "id", Type: "long"}},
	}
	r := NewTypeResolver([]parser.Class{controller, order})
	r.SetDependencyResolver(library)

	page := r.ResolveFrom(controller, "Wrapper<PageData<Order>>", nil)
	data := page.Fields["data"].Model
	if data == nil || data.Kind != model.KindObject {
		t.Fatalf("data = %+v, want PageData expanded", data)
	}
	rows := data.Fields["rows"].Model
	if rows.Kind != model.KindArray || rows.Items.Fields["id"] == nil {
		t.Fatalf("data.rows = %+v, want an array of Order", rows)
	}
	list := r.ResolveFrom(controller, "Wrapper<List<Order>>", nil).Fields["data"].Model
	if list.Kind != model.KindArray || list.Items.Fields["id"] == nil {
		t.Fatalf("data = %+v, want an array of Order: java.util.List stays a collection", list)
	}
	for _, name := range library.asked {
		if name == "PageData" {
			t.Errorf("PageData was asked for by its simple name; asked %v", library.asked)
		}
	}
}

func TestQualifyTypeArgs(t *testing.T) {
	source := parser.Class{
		Name: "Holder", Package: "com.example.app",
		Imports: []parser.Import{{Path: "com.example.lib.PageData"}, {Path: "java.util.List"}, {Path: "kotlin.collections.Map"}},
	}
	r := NewTypeResolver(nil)
	got := r.qualifyTypeArgs(source, []string{"PageData<Unknown>", "List<PageData>", "Map<String, PageData[]>", "?", "T", "? extends PageData"})
	want := []string{
		"com.example.lib.PageData<Unknown>",
		"List<com.example.lib.PageData>",
		"Map<String, com.example.lib.PageData[]>",
		"?",
		"T",
		"? extends com.example.lib.PageData",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("qualifyTypeArgs:\n got  %q\n want %q", got, want)
	}
}

// An enum of the project is a declaration of the project, whether or not the
// JSON view reads it: asking the dependency index for it would read some
// installed release of the project itself and report it as a dependency type.
func TestResolve_AProjectEnumIsNotAskedOfTheDependencies(t *testing.T) {
	tests := []struct {
		name      string
		jsonValue bool
		want      string
	}{
		{name: "an enum written as the name of a constant", want: model.JsonTypeString},
		{name: "an enum the JSON view does not read", jsonValue: true, want: "com.example.app.Status"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			library := &libraryClasses{classes: map[string]parser.Class{
				"com.example.app.Status": {Name: "Status", Package: "com.example.app", IsEnum: true, EnumConstants: []string{"OPEN"}},
			}}
			r := NewTypeResolver([]parser.Class{
				{Name: "Status", Package: "com.example.app", IsEnum: true, EnumConstants: []string{"OPEN"}, EnumJsonValue: tt.jsonValue},
				{Name: "Order", Package: "com.example.app", Fields: []parser.Field{{Name: "status", Type: "com.example.app.Status"}}},
			})
			r.SetDependencyResolver(library)

			order := r.Resolve("com.example.app.Order", nil)
			if status := order.Fields["status"].Model; status.TypeName != tt.want {
				t.Errorf("status = %+v, want %s", status, tt.want)
			}
			if len(library.asked) != 0 {
				t.Errorf("the dependencies were asked for %v", library.asked)
			}
		})
	}
}
