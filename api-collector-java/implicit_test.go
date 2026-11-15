package javacollector

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	collector "github.com/tangcent/apilot/api-collector"
)

// implicitExpectation is the export of testdata/implicit. The Kotlin collector
// asserts the same table for the same handlers written in Kotlin
// (api-collector-kotlin/implicit_test.go): keep the two in sync.
var implicitExpectation = map[string][]string{
	"page": {
		"orderNo/query", "regionList/query", "address.city/query", "displayName/query", "secret/query",
		"page/query", "size/query", "sort/query",
		"keyword/query", "ids/query", "status/query", "from/query",
	},
	"submit": {
		"orderNo/form", "regionList/form", "address.city/form", "displayName/form", "secret/form",
		"attachments/form",
	},
	"json":   {"trace/query"},
	"me":     {"id/query"},
	"search": {"name/query", "sort/query"},
}

func TestImplicitBindingMatchesSpring(t *testing.T) {
	abs, err := filepath.Abs("testdata/implicit")
	if err != nil {
		t.Fatal(err)
	}
	c := New().(*JavaCollector)
	endpoints, err := c.Collect(collector.CollectContext{SourceDir: abs, NoDeps: true})
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}
	checkImplicitExport(t, endpoints, c.Warnings())
}

func checkImplicitExport(t *testing.T, endpoints []collector.ApiEndpoint, warnings []string) {
	t.Helper()
	params := map[string]map[string]collector.ApiParameter{}
	for _, ep := range endpoints {
		var got []string
		params[ep.Name] = map[string]collector.ApiParameter{}
		for _, p := range ep.Parameters {
			got = append(got, p.Name+"/"+p.In)
			params[ep.Name][p.Name] = p
		}
		if want, ok := implicitExpectation[ep.Name]; ok && !reflect.DeepEqual(got, want) {
			t.Errorf("%s parameters:\n got  %v\n want %v", ep.Name, got, want)
		}
	}
	for name := range implicitExpectation {
		if _, ok := params[name]; !ok {
			t.Errorf("endpoint %s not exported", name)
		}
	}

	page := params["page"]
	if !page["keyword"].Required || page["ids"].Required {
		t.Errorf("an implicit simple parameter is optional unless @NotNull: keyword=%v ids=%v", page["keyword"].Required, page["ids"].Required)
	}
	if status := page["status"]; !reflect.DeepEqual(status.Enum, []string{"ACTIVE", "CLOSED"}) || status.Default != "ACTIVE" {
		t.Errorf("status = enum %v default %q", status.Enum, status.Default)
	}
	if page["orderNo"].Description != "Order number." {
		t.Errorf("orderNo description = %q", page["orderNo"].Description)
	}
	if params["submit"]["attachments"].Type != "file" {
		t.Errorf("a list of files is a file part: %+v", params["submit"]["attachments"])
	}

	joined := strings.Join(warnings, "\n")
	if !strings.Contains(joined, "OrderController.me: parameter operator carries @CurrentUser") {
		t.Errorf("the dropped @CurrentUser parameter is not reported:\n%s", joined)
	}
}
