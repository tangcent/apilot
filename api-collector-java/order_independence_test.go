package javacollector

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/tangcent/apilot/api-collector-java/parser"
	"github.com/tangcent/apilot/api-collector-java/springmvc"
)

// orderExport collects a Spring MVC project whose library is answered by the index
// of topLevelLibrary and returns each endpoint by "Controller.method", as the
// parameters (name, location, values of an enum) and the headers it exports.
func orderExport(t *testing.T, files map[string]string, sources map[string]string) map[string]string {
	t.Helper()
	dir := t.TempDir()
	writeTree(t, dir, files)
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
	sm.SetDependencyResolver(&topLevelLibrary{t: t, parser: p, sources: sources})
	out := map[string]string{}
	for _, ctrl := range sm.ExtractControllers(results) {
		for _, ep := range ctrl.Endpoints {
			api := springmvcEndpointToAPI(ep, ctrl.Name)
			summary := api.Method + " " + api.Path
			for _, param := range api.Parameters {
				summary += fmt.Sprintf(" param(%s in=%s enum=%v)", param.Name, param.In, param.Enum)
			}
			for _, header := range api.Headers {
				summary += " header(" + header.Name + ")"
			}
			if api.Response != nil && api.Response.Body != nil {
				summary += " resp=" + starModelKind(api.Response.Body)
			}
			out[ctrl.Name+"."+ep.MethodName] = summary
		}
	}
	return out
}

const springAnnotations = "import org.springframework.web.bind.annotation.*;\n"

// A library enum that a request parameter names has its constants whichever
// endpoint reads the enum first, whether it is imported on demand or by name.
func TestAnEnumOfALibraryHasItsConstantsWhateverIsReadFirst(t *testing.T) {
	sources := map[string]string{
		"lib.model.Status": "package lib.model;\npublic enum Status { OPEN, CLOSED }\n",
	}
	for _, imp := range []string{"import lib.model.*;", "import lib.model.Status;"} {
		for _, order := range [][]string{{"param", "field"}, {"field", "param"}} {
			t.Run(imp+" "+strings.Join(order, ","), func(t *testing.T) {
				bodies := map[string]string{
					"param": "  @GetMapping(\"/param\") public String param(Status status) { return null; }\n",
					"field": "  @GetMapping(\"/field\") public Holder field() { return null; }\n",
				}
				var endpoints strings.Builder
				for _, o := range order {
					endpoints.WriteString(bodies[o])
				}
				got := orderExport(t, map[string]string{
					"src/main/java/app/web/Holder.java": "package app.web;\n" + imp + "\npublic class Holder { private Status status; }\n",
					"src/main/java/app/web/C.java":      "package app.web;\n" + springAnnotations + imp + "\n@RestController\n@RequestMapping(\"/c\")\npublic class C {\n" + endpoints.String() + "}\n",
				}, sources)

				want := "GET /c/param param(status in=query enum=[OPEN CLOSED]) resp=string"
				if got["C.param"] != want {
					t.Errorf("C.param = %q, want %q", got["C.param"], want)
				}
			})
		}
	}
}

// The constant of a class of a library names a header whether or not the class was
// read as a type before.
func TestTheConstantOfALibraryClassNamesTheHeaderWhateverWasReadFirst(t *testing.T) {
	sources := map[string]string{
		"lib.consts.Paths": "package lib.consts;\npublic class Paths { public static final String TOKEN = \"X-Token\"; }\n",
	}
	for _, imp := range []string{"import lib.consts.Paths;", "import lib.consts.*;"} {
		for _, readAsType := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s read as a type first %v", imp, readAsType), func(t *testing.T) {
				extra := ""
				if readAsType {
					extra = "  @GetMapping(\"/a\") public Paths a() { return null; }\n"
				}
				got := orderExport(t, map[string]string{
					"src/main/java/app/web/C.java": "package app.web;\n" + springAnnotations + imp + "\n@RestController\npublic class C {\n" + extra +
						"  @GetMapping(\"/list\") public String list(@RequestHeader(Paths.TOKEN) String token) { return null; }\n}\n",
				}, sources)

				if want := "GET /list header(X-Token) resp=string"; !reflect.DeepEqual(got["C.list"], want) {
					t.Errorf("C.list = %q, want %q", got["C.list"], want)
				}
			})
		}
	}
}

// A superclass that only a library declares is read by the same lookup, so the
// controller inherits its endpoints and a DTO its fields, imported or on demand.
func TestASuperclassThatOnlyALibraryDeclaresIsReadByTheScope(t *testing.T) {
	sources := map[string]string{
		"lib.BaseVO": "package lib;\npublic class BaseVO { private String id; }\n",
		"lib.BaseCtl": "package lib;\n" + springAnnotations +
			"public abstract class BaseCtl {\n  @GetMapping(\"/base\") public String base() { return null; }\n}\n",
	}
	for _, imp := range []string{"import lib.*;", "import lib.BaseVO;\nimport lib.BaseCtl;"} {
		t.Run(imp, func(t *testing.T) {
			got := orderExport(t, map[string]string{
				"src/main/java/app/web/Vo.java": "package app.web;\n" + imp + "\npublic class Vo extends BaseVO { private String own; }\n",
				"src/main/java/app/web/C.java": "package app.web;\n" + springAnnotations + imp + "\n@RestController\n@RequestMapping(\"/c\")\npublic class C extends BaseCtl {\n" +
					"  @GetMapping(\"/vo\") public Vo vo() { return null; }\n}\n",
			}, sources)
			if _, ok := got["C.base"]; !ok {
				t.Errorf("the endpoint of the library's controller is not inherited: %v", got)
			}
			// The DTO's fields are those of the library's class too.
			if want := "GET /c/vo resp=object{id,own}"; got["C.vo"] != want {
				t.Errorf("C.vo = %q, want %q", got["C.vo"], want)
			}
		})
	}
}
