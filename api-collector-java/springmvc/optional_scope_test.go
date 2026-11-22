package springmvc

import (
	"reflect"
	"testing"

	"github.com/tangcent/apilot/api-collector-java/parser"
)

// optionalHandler is the parameters a GET handler `own(<paramType> filter)`
// exports, as name/location/type, for a controller written in the given scope
// next to the classes of the project.
func optionalHandler(t *testing.T, controller parser.Class, paramType string, project ...parser.Class) []string {
	t.Helper()

	controller.Name = "FilterController"
	controller.FilePath = "src/" + controller.Package + "/FilterController.java"
	controller.Annotations = []parser.Annotation{{Name: "RestController"}}
	controller.Methods = []parser.Method{{
		Name:        "own",
		Annotations: []parser.Annotation{{Name: "GetMapping", Params: map[string]string{"value": "/own"}}},
		Parameters:  []parser.Parameter{{Name: "filter", Type: paramType}},
		ReturnType:  "String",
	}}

	results := []parser.ParseResult{{Classes: append([]parser.Class{controller}, project...)}}
	controllers := NewParser().ExtractControllers(results)
	if len(controllers) != 1 || len(controllers[0].Endpoints) != 1 {
		t.Fatalf("want 1 controller with 1 endpoint, got %+v", controllers)
	}
	var got []string
	for _, p := range controllers[0].Endpoints[0].Parameters {
		got = append(got, p.Name+"/"+p.ParamType+"/"+p.Type)
	}
	return got
}

// A project class named Optional is a bean like any other: Spring binds it
// property by property, and only java.util.Optional is bound as the type it
// holds. The name alone does not say which one a parameter is; the scope of the
// controller does.
func TestParser_ProjectOptionalIsABoundBean(t *testing.T) {
	own := parser.Class{
		Name: "Optional", Package: "com.example.own", FilePath: "src/com.example.own/Optional.java",
		TypeParameters: []string{"T"},
		Fields:         []parser.Field{{Name: "prefix", Type: "String"}, {Name: "value", Type: "T"}},
	}
	want := []string{"prefix/query/string", "value/query/string"}

	for scope, controller := range map[string]parser.Class{
		"same package": {Package: "com.example.own"},
		"single-type import": {
			Package: "com.example.web",
			Imports: []parser.Import{{Path: "com.example.own.Optional"}},
		},
		"import on demand": {
			Package: "com.example.web",
			Imports: []parser.Import{{Path: "com.example.own", OnDemand: true}, {Path: "java.util", OnDemand: true}},
		},
	} {
		if got := optionalHandler(t, controller, "Optional<String>", own); !reflect.DeepEqual(got, want) {
			t.Errorf("%s: parameters = %v, want %v", scope, got, want)
		}
	}
}

// java.util.Optional is bound as the type it holds, however the controller
// reaches it, and a class of the project that the controller does not see does
// not take its place.
func TestParser_JavaUtilOptionalIsBoundAsItsValue(t *testing.T) {
	elsewhere := parser.Class{
		Name: "Optional", Package: "com.example.own", FilePath: "src/com.example.own/Optional.java",
		TypeParameters: []string{"T"},
		Fields:         []parser.Field{{Name: "prefix", Type: "String"}},
	}
	want := []string{"filter/query/Optional<String>"}

	for scope, controller := range map[string]parser.Class{
		"single-type import": {Package: "com.example.web", Imports: []parser.Import{{Path: "java.util.Optional"}}},
		"import on demand":   {Package: "com.example.web", Imports: []parser.Import{{Path: "java.util", OnDemand: true}}},
		"on demand next to the project's package": {
			Package: "com.example.web",
			Imports: []parser.Import{{Path: "com.example.other", OnDemand: true}, {Path: "java.util", OnDemand: true}},
		},
	} {
		if got := optionalHandler(t, controller, "Optional<String>", elsewhere); !reflect.DeepEqual(got, want) {
			t.Errorf("%s: parameters = %v, want %v", scope, got, want)
		}
	}

	// Written in full, the name needs no import.
	wantFull := []string{"filter/query/java.util.Optional<String>"}
	if got := optionalHandler(t, parser.Class{Package: "com.example.own"}, "java.util.Optional<String>", elsewhere); !reflect.DeepEqual(got, wantFull) {
		t.Errorf("written in full: parameters = %v, want %v", got, wantFull)
	}
}
