package kotlincollector

import (
	"testing"
)

func TestImplicitBindingMatchesSpring(t *testing.T) {
	endpoints, _ := collectDir(t, "testdata/implicit")
	locations := func(name string) map[string]string {
		out := map[string]string{}
		for _, p := range findEndpoint(t, endpoints, name).Parameters {
			out[p.Name] = p.In
		}
		return out
	}

	page := locations("page")
	for name, in := range map[string]string{
		"orderNo": "query", "regionList": "query", "address.city": "query",
		"page": "query", "size": "query", "sort": "query",
		"keyword": "query", "ids": "query", "status": "query",
	} {
		if page[name] != in {
			t.Errorf("page: parameter %q in %q, want %q; got %v", name, page[name], in, page)
		}
	}
	if len(page) != 9 {
		t.Errorf("page: expected 9 parameters, got %v", page)
	}
	pageEndpoint := findEndpoint(t, endpoints, "page")
	if !findParameter(t, pageEndpoint, "keyword", "query").Required {
		t.Errorf("keyword carries @NotNull and should be required")
	}
	if findParameter(t, pageEndpoint, "ids", "query").Required {
		t.Errorf("an implicit simple parameter is optional")
	}
	if status := findParameter(t, pageEndpoint, "status", "query"); len(status.Enum) != 2 {
		t.Errorf("status enum = %v", status.Enum)
	}

	submit := locations("submit")
	if submit["orderNo"] != "form" || submit["address.city"] != "form" || submit["attachment"] != "form" {
		t.Errorf("submit parameters = %v", submit)
	}

	json := findEndpoint(t, endpoints, "json")
	if json.RequestBody == nil || len(json.Parameters) != 1 || json.Parameters[0].Name != "trace" || json.Parameters[0].In != "query" {
		t.Errorf("json parameters = %#v", json.Parameters)
	}

	me := locations("me")
	if _, ok := me["operator"]; ok || me["id"] != "query" || len(me) != 1 {
		t.Errorf("me parameters = %v", me)
	}
}
