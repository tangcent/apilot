package springmvc

import (
	"testing"

	"github.com/tangcent/apilot/api-collector-java/parser"
)

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
