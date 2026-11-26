package springmvc

import (
	"testing"

	ktparser "github.com/tangcent/apilot/api-collector-kotlin/parser"
	"github.com/tangcent/apilot/api-collector-kotlin/resolver"
)

func testAnnotation(name string, params map[string]string) ktparser.Annotation {
	return ktparser.Annotation{Name: name, Params: params}
}

func testParameter(name, typ string, annotations ...ktparser.Annotation) ktparser.Parameter {
	return ktparser.Parameter{Name: name, Type: typ, Annotations: annotations}
}

func testFunction(name string, annotations ...ktparser.Annotation) ktparser.Function {
	return ktparser.Function{Name: name, Annotations: annotations}
}

func TestExtractMethodInfo(t *testing.T) {
	tests := []struct {
		name         string
		annotations  []ktparser.Annotation
		wantMethod   HTTPMethod
		wantPath     string
		wantConsumes []string
		wantProduces []string
	}{
		{
			name:        "GetMapping with value",
			annotations: []ktparser.Annotation{testAnnotation("GetMapping", map[string]string{"value": "/users"})},
			wantMethod:  GET,
			wantPath:    "/users",
		},
		{
			name: "PostMapping with path and media types",
			annotations: []ktparser.Annotation{testAnnotation("PostMapping", map[string]string{
				"path":     "upload",
				"consumes": "MediaType.MULTIPART_FORM_DATA",
				"produces": "APPLICATION_JSON",
			})},
			wantMethod:   POST,
			wantPath:     "/upload",
			wantConsumes: []string{"multipart/form-data"},
			wantProduces: []string{"application/json"},
		},
		{
			name: "RequestMapping with a single method",
			annotations: []ktparser.Annotation{testAnnotation("RequestMapping", map[string]string{
				"value":  "/items",
				"method": "RequestMethod.PUT",
			})},
			wantMethod: PUT,
			wantPath:   "/items",
		},
		{
			name: "RequestMapping with several methods keeps the first",
			annotations: []ktparser.Annotation{testAnnotation("RequestMapping", map[string]string{
				"method": "RequestMethod.GET, RequestMethod.POST",
			})},
			wantMethod: GET,
		},
		{
			name: "ApiOperation supplies the verb when RequestMapping omits it",
			annotations: []ktparser.Annotation{
				testAnnotation("RequestMapping", nil),
				testAnnotation("ApiOperation", map[string]string{"httpMethod": "delete"}),
			},
			wantMethod: DELETE,
		},
		{
			name:        "no mapping annotation",
			annotations: []ktparser.Annotation{testAnnotation("Deprecated", nil)},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			method, path, consumes, produces := extractMethodInfo(tt.annotations)
			if method != tt.wantMethod {
				t.Errorf("method = %q, want %q", method, tt.wantMethod)
			}
			if path != tt.wantPath {
				t.Errorf("path = %q, want %q", path, tt.wantPath)
			}
			if !equalStrings(consumes, tt.wantConsumes) {
				t.Errorf("consumes = %v, want %v", consumes, tt.wantConsumes)
			}
			if !equalStrings(produces, tt.wantProduces) {
				t.Errorf("produces = %v, want %v", produces, tt.wantProduces)
			}
		})
	}
}

func TestExtractParameter(t *testing.T) {
	p := NewParser(nil)

	tests := []struct {
		name          string
		param         ktparser.Parameter
		wantNil       bool
		wantName      string
		wantParamType string
		wantRequired  bool
		wantDefault   string
	}{
		{
			name:          "PathVariable keeps the identifier",
			param:         testParameter("id", "Long", testAnnotation("PathVariable", nil)),
			wantName:      "id",
			wantParamType: "path",
			wantRequired:  true,
		},
		{
			name:          "PathVariable annotation name wins",
			param:         testParameter("categoryId", "Long", testAnnotation("PathVariable", map[string]string{"value": "id"})),
			wantName:      "id",
			wantParamType: "path",
			wantRequired:  true,
		},
		{
			name:          "RequestParam required=false",
			param:         testParameter("status", "Status?", testAnnotation("RequestParam", map[string]string{"required": "false"})),
			wantName:      "status",
			wantParamType: "query",
			wantRequired:  false,
		},
		{
			name: "RequestParam defaultValue implies optional",
			param: testParameter("pageNum", "Int", testAnnotation("RequestParam", map[string]string{
				"defaultValue": "1",
			})),
			wantName:      "pageNum",
			wantParamType: "query",
			wantRequired:  false,
			wantDefault:   "1",
		},
		{
			name:          "RequestHeader explicit name",
			param:         testParameter("tenant", "String", testAnnotation("RequestHeader", map[string]string{"value": "X-Tenant"})),
			wantName:      "X-Tenant",
			wantParamType: "header",
			wantRequired:  true,
		},
		{
			name:          "RequestHeader required=false",
			param:         testParameter("trace", "String", testAnnotation("RequestHeader", map[string]string{"required": "false"})),
			wantName:      "trace",
			wantParamType: "header",
			wantRequired:  false,
		},
		{
			name:          "RequestBody keeps the Kotlin identifier",
			param:         testParameter("req", "CreateCategoryReq", testAnnotation("RequestBody", nil)),
			wantName:      "req",
			wantParamType: "body",
			wantRequired:  true,
		},
		{
			name:          "RequestPart is a form field",
			param:         testParameter("file", "MultipartFile", testAnnotation("RequestPart", nil)),
			wantName:      "file",
			wantParamType: "form",
			wantRequired:  true,
		},
		{
			name:          "CookieValue",
			param:         testParameter("sessionId", "String", testAnnotation("CookieValue", nil)),
			wantName:      "sessionId",
			wantParamType: "cookie",
			wantRequired:  true,
		},
		{
			name:          "ApiParam is a query argument",
			param:         testParameter("keyword", "String", testAnnotation("ApiParam", nil)),
			wantName:      "keyword",
			wantParamType: "query",
			wantRequired:  true,
		},
		{
			name: "explicit binding wins over ApiParam",
			param: testParameter("tenant", "String",
				testAnnotation("ApiParam", nil),
				testAnnotation("RequestHeader", nil)),
			wantName:      "tenant",
			wantParamType: "header",
			wantRequired:  true,
		},
		{
			name:    "unannotated parameter is not bound",
			param:   testParameter("ignored", "String"),
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := p.extractParameter(&tt.param, nil, "handler", nil)
			if tt.wantNil {
				if got != nil {
					t.Fatalf("expected no parameter, got %#v", got)
				}
				return
			}
			if got == nil {
				t.Fatal("expected a parameter, got nil")
			}
			if got.Name != tt.wantName {
				t.Errorf("name = %q, want %q", got.Name, tt.wantName)
			}
			if got.ParamType != tt.wantParamType {
				t.Errorf("in = %q, want %q", got.ParamType, tt.wantParamType)
			}
			if got.Required != tt.wantRequired {
				t.Errorf("required = %v, want %v", got.Required, tt.wantRequired)
			}
			if got.DefaultValue != tt.wantDefault {
				t.Errorf("default = %q, want %q", got.DefaultValue, tt.wantDefault)
			}
		})
	}
}

func TestExtractParameterNamedByAConstantThatCannotBeResolved(t *testing.T) {
	p := NewParser(nil)
	ann := testAnnotation("RequestHeader", map[string]string{"value": "Headers.TENANT"})
	ann.Refs = map[string]bool{"value": true}
	param := testParameter("tenant", "String", ann)

	got := p.extractParameter(&param, nil, "handle", &ktparser.Class{Name: "Handler"})
	if got == nil || got.Name != "tenant" || got.ParamType != "header" {
		t.Fatalf("parameter = %+v, want the tenant header under the parameter's own name", got)
	}
	want := "Handler.handle: parameter tenant is bound by the name in Headers.TENANT, a constant whose value could not be resolved; it is exported under the parameter's own name"
	if warnings := p.Warnings(); len(warnings) != 1 || warnings[0] != want {
		t.Errorf("warnings = %q, want %q", warnings, want)
	}
}

func TestExtractParameterDocumentation(t *testing.T) {
	p := NewParser(nil)
	param := testParameter("status", "String", testAnnotation("ApiParam", map[string]string{
		"description":     "lifecycle state",
		"example":         "ACTIVE",
		"allowableValues": "ACTIVE,DISABLED",
	}))

	got := p.extractParameter(&param, map[string]string{"status": "the status filter"}, "handler", nil)
	if got == nil {
		t.Fatal("expected a parameter, got nil")
	}
	if got.Description != "lifecycle state" {
		t.Errorf("description = %q", got.Description)
	}
	if got.Example != "ACTIVE" {
		t.Errorf("example = %q", got.Example)
	}
	if !equalStrings(got.Enum, []string{"ACTIVE", "DISABLED"}) {
		t.Errorf("enum = %v", got.Enum)
	}
}

func TestPathHelpers(t *testing.T) {
	tests := []struct {
		raw  string
		want string
	}{
		{`"/users"`, "/users"},
		{`'users'`, "/users"},
		{"users", "/users"},
		{"/users", "/users"},
		{"", ""},
	}
	for _, tt := range tests {
		if got := normalizePath(tt.raw); got != tt.want {
			t.Errorf("normalizePath(%q) = %q, want %q", tt.raw, got, tt.want)
		}
	}

	combined := []struct {
		base, method, want string
	}{
		{"/api", "/users", "/api/users"},
		{"/api/", "/users", "/api/users"},
		{"", "/users", "/users"},
		{"/api", "", "/api"},
		{"", "", ""},
	}
	for _, tt := range combined {
		if got := combinePaths(tt.base, tt.method); got != tt.want {
			t.Errorf("combinePaths(%q, %q) = %q, want %q", tt.base, tt.method, got, tt.want)
		}
	}
}

func TestIsVoidReturn(t *testing.T) {
	source := &ktparser.Class{Name: "Controller", Package: "com.example.app"}
	p := NewParser(resolver.New(nil))
	voidReturns := []string{"", "Unit", "kotlin.Unit", "Void", "java.lang.Void", "Nothing", "kotlin.Nothing", " Unit "}
	for _, typ := range voidReturns {
		if !p.isVoidReturn(source, typ) {
			t.Errorf("isVoidReturn(%q) = false, want true", typ)
		}
	}

	nonVoid := []string{"Result<Void>", "String", "List<Unit>", "Result<Unit>"}
	for _, typ := range nonVoid {
		if p.isVoidReturn(source, typ) {
			t.Errorf("isVoidReturn(%q) = true, want false", typ)
		}
	}
}

// A class of the project that is called Unit, Void or Nothing is a class, in the
// package that declares it and where it is imported; elsewhere the name is the
// type that returns nothing.
func TestIsVoidReturn_AProjectClassOfThatNameIsNotVoid(t *testing.T) {
	unit := ktparser.Class{Name: "Unit", Package: "com.example.model", IsEnum: true, EnumValues: []string{"PIECE"}}
	void := ktparser.Class{Name: "Void", Package: "com.example.model"}
	p := NewParser(resolver.New([]ktparser.Class{unit, void}))

	same := &ktparser.Class{Name: "Controller", Package: "com.example.model"}
	imported := &ktparser.Class{
		Name: "Controller", Package: "com.example.web",
		Imports: []ktparser.Import{{Path: "com.example.model.Unit"}, {Path: "com.example.model.Void"}},
	}
	starred := &ktparser.Class{
		Name: "Controller", Package: "com.example.web",
		Imports: []ktparser.Import{{Path: "com.example.model", OnDemand: true}},
	}
	other := &ktparser.Class{Name: "Controller", Package: "com.example.web"}
	for name, source := range map[string]*ktparser.Class{"same package": same, "imported": imported, "star-imported": starred} {
		for _, typ := range []string{"Unit", "Void"} {
			if p.isVoidReturn(source, typ) {
				t.Errorf("%s: isVoidReturn(%q) = true, want false: the project declares it", name, typ)
			}
		}
		// The names of the standard library written in full stay what they are.
		for _, typ := range []string{"kotlin.Unit", "java.lang.Void"} {
			if !p.isVoidReturn(source, typ) {
				t.Errorf("%s: isVoidReturn(%q) = false, want true", name, typ)
			}
		}
	}
	for _, typ := range []string{"Unit", "Void"} {
		if !p.isVoidReturn(other, typ) {
			t.Errorf("isVoidReturn(%q) = false, want true where nothing declares it", typ)
		}
	}
}

func TestMediaTypeNormalization(t *testing.T) {
	tests := []struct {
		raw  string
		want []string
	}{
		{"", nil},
		{"APPLICATION_JSON", []string{"application/json"}},
		{"MediaType.MULTIPART_FORM_DATA", []string{"multipart/form-data"}},
		{"application/json, text/plain", []string{"application/json", "text/plain"}},
		{`"application/xml"`, []string{"application/xml"}},
		{"UNKNOWN_CONSTANT", []string{"UNKNOWN_CONSTANT"}},
	}
	for _, tt := range tests {
		got := mediaTypes(tt.raw)
		if !equalStrings(got, tt.want) {
			t.Errorf("mediaTypes(%q) = %v, want %v", tt.raw, got, tt.want)
		}
	}
}

func TestExtractControllers(t *testing.T) {
	classes := []ktparser.Class{
		{
			Name:        "UserController",
			Package:     "com.example",
			Annotations: []ktparser.Annotation{testAnnotation("RestController", nil), testAnnotation("RequestMapping", map[string]string{"value": "/users"})},
			Methods: []ktparser.Function{
				testFunction("list", testAnnotation("GetMapping", nil)),
				testFunction("helper"),
			},
		},
		{
			Name:        "LegacyController",
			Package:     "com.example",
			Annotations: []ktparser.Annotation{testAnnotation("Controller", nil)},
			Methods: []ktparser.Function{
				testFunction("show", testAnnotation("GetMapping", map[string]string{"value": "/show"})),
			},
		},
		{Name: "UserService", Package: "com.example", Methods: []ktparser.Function{testFunction("list", testAnnotation("GetMapping", nil))}},
	}

	p := NewParser(resolver.New(classes))
	controllers := p.ExtractControllers([]ktparser.ParseResult{{Classes: classes}})
	if len(controllers) != 2 {
		t.Fatalf("expected 2 controllers, got %d: %#v", len(controllers), controllers)
	}

	byName := map[string]Controller{}
	for _, c := range controllers {
		byName[c.Name] = c
	}
	user, ok := byName["UserController"]
	if !ok {
		t.Fatal("UserController was not extracted")
	}
	if user.BasePath != "/users" {
		t.Errorf("base path = %q", user.BasePath)
	}
	if len(user.Endpoints) != 1 || user.Endpoints[0].Path != "/users" || user.Endpoints[0].Method != GET {
		t.Errorf("endpoints = %#v", user.Endpoints)
	}

	legacy, ok := byName["LegacyController"]
	if !ok {
		t.Fatal("@Controller class was not extracted")
	}
	if len(legacy.Endpoints) != 1 || legacy.Endpoints[0].Path != "/show" {
		t.Errorf("legacy endpoints = %#v", legacy.Endpoints)
	}
	if _, ok := byName["UserService"]; ok {
		t.Error("plain class was treated as a controller")
	}
}

func TestInheritedEndpoints(t *testing.T) {
	base := ktparser.Class{
		Name:        "BaseController",
		Package:     "com.example",
		FilePath:    "BaseController.kt",
		Annotations: []ktparser.Annotation{testAnnotation("RequestMapping", map[string]string{"value": "/base"})},
		Methods: []ktparser.Function{
			testFunction("list", testAnnotation("GetMapping", nil)),
			testFunction("overridden", testAnnotation("GetMapping", nil)),
			testFunction("helper"),
		},
	}
	child := ktparser.Class{
		Name:        "UserController",
		Package:     "com.example",
		FilePath:    "UserController.kt",
		SuperClass:  "BaseController",
		Annotations: []ktparser.Annotation{testAnnotation("RestController", nil)},
		Methods: []ktparser.Function{
			testFunction("overridden", testAnnotation("GetMapping", nil)),
			testFunction("create", testAnnotation("PostMapping", map[string]string{"value": "/create"})),
		},
	}

	classes := []ktparser.Class{base, child}
	p := NewParser(resolver.New(classes))
	controllers := p.ExtractControllers([]ktparser.ParseResult{{Classes: classes}})
	if len(controllers) != 1 {
		t.Fatalf("expected 1 controller, got %d", len(controllers))
	}

	got := map[string]Endpoint{}
	for _, ep := range controllers[0].Endpoints {
		got[ep.MethodName] = ep
	}
	if len(got) != 3 {
		t.Fatalf("expected 3 endpoints, got %d: %#v", len(got), got)
	}

	list, ok := got["list"]
	if !ok {
		t.Fatal("inherited endpoint was not extracted")
	}
	if list.Path != "/base" {
		t.Errorf("inherited path = %q, want /base", list.Path)
	}
	if _, ok := got["helper"]; ok {
		t.Error("non-mapping method was exported")
	}
	if overridden := got["overridden"]; overridden.ClassName != "UserController" {
		t.Errorf("override came from %q, want UserController", overridden.ClassName)
	}
	if create := got["create"]; create.Path != "/create" || create.Method != POST {
		t.Errorf("child endpoint = %#v", create)
	}
}

func TestBuildTypeBindings(t *testing.T) {
	parent := ktparser.Class{
		Name:           "BaseController",
		Package:        "com.example",
		TypeParameters: []string{"T"},
	}
	child := ktparser.Class{
		Name:               "UserController",
		Package:            "com.example",
		SuperClass:         "BaseController",
		SuperClassTypeArgs: []string{"User"},
	}
	p := NewParser(resolver.New([]ktparser.Class{parent, child}))

	bindings := p.buildTypeBindings(&child, nil)
	if bindings["T"] != "User" {
		t.Fatalf("bindings = %#v", bindings)
	}

	if got := p.buildTypeBindings(&parent, nil); got != nil {
		t.Errorf("class without a superclass should have no bindings: %#v", got)
	}
}

// Each level binds the parameters of its own superclass and nothing else: a
// parameter that two levels name alike is bound once for each of them, and an
// argument that mentions a parameter of its class is spelled with its binding.
func TestBuildTypeBindings_BindsEachLevelOnItsOwn(t *testing.T) {
	base := ktparser.Class{Name: "Base", Package: "com.example", TypeParameters: []string{"T"}}
	mid := ktparser.Class{
		Name: "Mid", Package: "com.example", TypeParameters: []string{"T"},
		SuperClass: "Base", SuperClassTypeArgs: []string{"List<T>"},
	}
	leaf := ktparser.Class{
		Name: "Leaf", Package: "com.example",
		SuperClass: "Mid", SuperClassTypeArgs: []string{"User"},
	}
	p := NewParser(resolver.New([]ktparser.Class{base, mid, leaf}))

	midBindings := p.buildTypeBindings(&leaf, nil)
	if midBindings["T"] != "User" {
		t.Fatalf("bindings of Mid = %#v, want T=User", midBindings)
	}
	baseBindings := p.buildTypeBindings(&mid, midBindings)
	if baseBindings["T"] != "java.util.List<User>" {
		t.Errorf("bindings of Base = %#v, want T=java.util.List<User>", baseBindings)
	}
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// A handler parameter without a binding annotation is an enum, and takes its
// entries, by the enum its type name denotes in the scope of the controller: the
// Status of the controller's own package, not another Status that was declared
// later, and not an enum for the sake of a Status elsewhere when the controller's
// own Status is a class.
func TestImplicitParameterNamesAnEnumInTheScopeOfTheController(t *testing.T) {
	classes := []ktparser.Class{
		{Name: "Status", Package: "b", FilePath: "b/Status.kt", IsEnum: true, EnumValues: []string{"ON", "OFF"}},
		{Name: "Status", Package: "a", FilePath: "a/Status.kt", IsEnum: true, EnumValues: []string{"ACTIVE"}},
		{Name: "Status", Package: "c", FilePath: "c/Status.kt", Fields: []ktparser.Property{{Name: "label", Type: "String", IsMutable: true}}},
	}
	p := NewParser(resolver.New(classes))
	param := testParameter("status", "Status")

	inB := p.implicitParameters(&param, "query", "list", nil, &ktparser.Class{Name: "BController", Package: "b", FilePath: "b/BController.kt"}, nil)
	if len(inB) != 1 || inB[0].Name != "status" || !equalStrings(inB[0].Enum, []string{"ON", "OFF"}) {
		t.Errorf("in b = %+v, want the entries of b.Status", inB)
	}

	inC := p.implicitParameters(&param, "query", "list", nil, &ktparser.Class{Name: "CController", Package: "c", FilePath: "c/CController.kt"}, nil)
	if len(inC) != 1 || inC[0].Name != "label" {
		t.Errorf("in c = %+v, want the properties of the class c.Status as model attribute parameters", inC)
	}
}
