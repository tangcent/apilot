package springmvc

import (
	"reflect"
	"testing"

	"github.com/tangcent/apilot/api-collector-java/parser"
	model "github.com/tangcent/apilot/api-model"
)

func TestClassifyImplicit(t *testing.T) {
	isEnum := func(t string) bool { return t == "Status" }
	for _, tc := range []struct {
		typeName    string
		annotations []string
		want        ImplicitKind
		blocker     string
	}{
		{"String", nil, ImplicitParam, ""},
		{"Status", nil, ImplicitParam, ""},
		{"List<Status>", nil, ImplicitParam, ""},
		{"Long[]", nil, ImplicitParam, ""},
		{"OrderQuery", nil, ImplicitModel, ""},
		{"HttpServletRequest", nil, ImplicitNone, ""},
		{"Map<String,String>", nil, ImplicitNone, ""},
		{"Pageable", nil, ImplicitPageable, ""},
		{"Sort", nil, ImplicitSort, ""},
		// Files: one, a collection, an array.
		{"MultipartFile", nil, ImplicitFile, ""},
		{"List<MultipartFile>", nil, ImplicitFile, ""},
		{"MultipartFile[]", nil, ImplicitFile, ""},
		// The type is read as written. An Optional is a bean until the scope of
		// its name says it is java.util.Optional, which the caller looks
		// through before it asks (see TestParser_JavaUtilOptionalIsBoundAsItsValue).
		{"Optional<String>", nil, ImplicitModel, ""},
		{"java.util.Optional<Long>", nil, ImplicitModel, ""},
		// What an Optional holds is classified like the type on its own.
		{"String", nil, ImplicitParam, ""},
		{"List<Long>", nil, ImplicitParam, ""},
		{"OrderQuery", nil, ImplicitModel, ""},
		{"List<MultipartFile>", nil, ImplicitFile, ""},
		{"HttpServletRequest", nil, ImplicitNone, ""},
		// Annotations that format or default a value do not change the binding.
		{"LocalDate", []string{"DateTimeFormat"}, ImplicitParam, ""},
		{"BigDecimal", []string{"NumberFormat"}, ImplicitParam, ""},
		{"Pageable", []string{"PageableDefault", "Qualifier"}, ImplicitPageable, ""},
		{"Sort", []string{"SortDefault"}, ImplicitSort, ""},
		// An annotation no binding covers leaves the parameter to a custom
		// resolver, and is named so the caller can say so.
		{"Operator", []string{"CurrentUser"}, ImplicitNone, "CurrentUser"},
		{"Operator", []string{"Valid", "CurrentUser"}, ImplicitNone, "CurrentUser"},
		// An explicit @ModelAttribute wins wherever it is written.
		{"OrderFilter", []string{"Audit", "ModelAttribute"}, ImplicitModel, ""},
		{"OrderFilter", []string{"ModelAttribute", "Audit"}, ImplicitModel, ""},
	} {
		kind, blocker := ClassifyImplicit(tc.typeName, tc.annotations, isEnum)
		if kind != tc.want || blocker != tc.blocker {
			t.Errorf("ClassifyImplicit(%q, %v) = %v, %q; want %v, %q", tc.typeName, tc.annotations, kind, blocker, tc.want, tc.blocker)
		}
	}
}

func TestParser_ImplicitBinding(t *testing.T) {
	jp, err := parser.NewParser(parser.ParserOptions{})
	if err != nil {
		t.Fatalf("Failed to create java parser: %v", err)
	}
	defer jp.Close()

	source := []byte(`package com.example.api;

import jakarta.servlet.http.HttpServletRequest;
import jakarta.validation.Valid;
import jakarta.validation.constraints.NotNull;
import java.util.List;
import org.springframework.data.domain.Pageable;
import org.springframework.web.bind.annotation.*;
import org.springframework.web.multipart.MultipartFile;

@RestController
@RequestMapping("/orders")
public class OrderController {

    @GetMapping
    public String page(@Valid OrderQuery query, Pageable pageable, HttpServletRequest request,
                       @NotNull String keyword, List<Long> ids) {
        return "";
    }

    @PostMapping("/form")
    public String submit(OrderQuery form, MultipartFile attachment) {
        return "";
    }

    @PostMapping("/json")
    public String json(@RequestBody OrderQuery body, String trace) {
        return "";
    }

    @GetMapping("/me")
    public String me(@CurrentUser Operator operator, Long id) {
        return "";
    }
}

class Address {
    private String city;
}

class OrderQuery {
    /** Order number. */
    private String orderNo;
    private List<String> regionList;
    private Address address;
}

class Operator {
    private Long id;
}
`)
	classes, err := jp.ParseSource(source)
	if err != nil {
		t.Fatalf("Failed to parse source: %v", err)
	}
	controllers := NewParser().ExtractControllers([]parser.ParseResult{
		{FilePath: "OrderController.java", Classes: classes},
	})
	if len(controllers) != 1 || len(controllers[0].Endpoints) != 4 {
		t.Fatalf("expected 1 controller with 4 endpoints, got %#v", controllers)
	}
	byName := map[string]Endpoint{}
	for _, ep := range controllers[0].Endpoints {
		byName[ep.MethodName] = ep
	}
	locations := func(ep Endpoint) map[string]string {
		out := map[string]string{}
		for _, p := range ep.Parameters {
			out[p.Name] = p.ParamType
		}
		return out
	}

	// A model attribute's properties are request parameters, a nested object's
	// under a dotted path; framework objects export nothing; Pageable becomes
	// page, size and sort.
	page := locations(byName["page"])
	for name, in := range map[string]string{
		"orderNo": "query", "regionList": "query", "address.city": "query",
		"page": "query", "size": "query", "sort": "query",
		"keyword": "query", "ids": "query",
	} {
		if page[name] != in {
			t.Errorf("page: parameter %q in %q, want %q; got %v", name, page[name], in, page)
		}
	}
	if len(page) != 8 {
		t.Errorf("page: expected 8 parameters, got %v", page)
	}
	for _, p := range byName["page"].Parameters {
		switch p.Name {
		case "keyword":
			if !p.Required {
				t.Errorf("keyword carries @NotNull and should be required")
			}
		case "ids":
			if p.Required {
				t.Errorf("an implicit simple parameter is optional")
			}
		case "orderNo":
			if p.Description != "Order number." {
				t.Errorf("orderNo description = %q", p.Description)
			}
		}
	}

	// Without @RequestBody, a POST binds the model attribute from the form.
	submit := locations(byName["submit"])
	if submit["orderNo"] != "form" || submit["address.city"] != "form" || submit["attachment"] != "form" {
		t.Errorf("submit parameters = %v", submit)
	}

	// With @RequestBody the body is taken, so the rest comes from the query.
	json := locations(byName["json"])
	if json["trace"] != "query" || json["body"] != "body" || len(json) != 2 {
		t.Errorf("json parameters = %v", json)
	}

	// A parameter left to a custom argument resolver is not request data.
	me := locations(byName["me"])
	if _, ok := me["operator"]; ok || me["id"] != "query" || len(me) != 1 {
		t.Errorf("me parameters = %v", me)
	}
}

func TestOptionalArgument(t *testing.T) {
	for _, tc := range []struct {
		name        string
		paramType   string
		typeName    string
		annotations []string
		optional    bool
		want        bool
	}{
		{"plain query parameter", "query", "String", nil, false, false},
		{"Optional", "query", "Optional<String>", nil, false, true},
		{"qualified Optional", "header", "java.util.Optional<Long>", nil, false, true},
		{"a type merely named like it", "query", "OptionalFeature", nil, false, false},
		{"@Nullable", "header", "String", []string{"RequestHeader", "Nullable"}, false, true},
		{"Kotlin nullable or default", "form", "String", nil, true, true},
		{"a constraint keeps it mandatory", "query", "Long", []string{"RequestParam", "NotNull"}, true, false},
		{"path variable", "path", "Optional<String>", nil, true, false},
		{"request body", "body", "String", []string{"Nullable"}, true, false},
	} {
		if got := OptionalArgument(tc.paramType, tc.typeName, tc.annotations, tc.optional); got != tc.want {
			t.Errorf("%s: OptionalArgument = %v, want %v", tc.name, got, tc.want)
		}
	}
}

// The element of a parameter that holds a list of files is the file, which is
// how the collector tells it from a value. The type is read as written: an
// Optional is looked through by the caller, once the scope of its name says it
// is java.util.Optional.
func TestElementType_ReadsTheTypeAsWritten(t *testing.T) {
	for typeName, want := range map[string]string{
		"List<MultipartFile>":     "MultipartFile",
		"MultipartFile[]":         "MultipartFile",
		"MultipartFile":           "MultipartFile",
		"List<Long>":              "Long",
		"Optional<MultipartFile>": "Optional",
	} {
		if got := ElementType(typeName); got != want {
			t.Errorf("ElementType(%q) = %q, want %q", typeName, got, want)
		}
	}
}

// A property that holds files is one of the request's parts. The resolver
// answers a MultipartFile with a single value named after it, and the type
// that reaches the parameter must still say it is a file, for one file and for
// a collection or an array of them.
func TestModelAttributeFields_FilesKeepTheirType(t *testing.T) {
	file := model.SingleModel("MultipartFile")
	upload := model.NewObjectModelBuilder().
		StringField("title").
		Field("cover", file).
		ArrayField("attachments", file).
		ArrayField("tags", model.SingleModel(model.JsonTypeString)).
		Build()

	got := map[string]string{}
	for _, field := range ModelAttributeFields(upload) {
		got[field.Name] = field.TypeName
	}
	want := map[string]string{
		"title":       "string",
		"cover":       "MultipartFile",
		"attachments": "MultipartFile[]",
		"tags":        "array",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("types of the bound properties = %v, want %v", got, want)
	}
}
