package markdown

import (
	"encoding/json"
	"strings"
	"testing"

	formatter "github.com/tangcent/apilot/api-formatter"
	model "github.com/tangcent/apilot/api-model"
)

// TestFormat_DocumentMetadata covers the document heading: it names the
// project, the collector and the framework the endpoints were read from, and
// how many there are. Without it an export outside its directory — or one
// written to stdout — says nothing about where it came from (#170).
func TestFormat_DocumentMetadata(t *testing.T) {
	f := New()
	endpoints := []model.ApiEndpoint{
		{Name: "Get Order", Path: "/orders/{id}", Method: "GET", Protocol: "http"},
		{Name: "Create Order", Path: "/orders", Method: "POST", Protocol: "http"},
	}

	ordersDoc := render(t, f, endpoints, Params{
		Variant: "detailed", ProjectName: "orders-service", Collector: "java", Frameworks: []string{"spring-mvc"},
	})
	want := "# orders-service API — java / spring-mvc, 2 endpoints"
	if !strings.Contains(ordersDoc, want) {
		t.Errorf("output should contain %q, got heading %q", want, firstLine(ordersDoc))
	}

	// Two projects must be distinguishable by that first heading alone.
	paymentsDoc := render(t, f, endpoints, Params{
		Variant: "detailed", ProjectName: "payments-service", Collector: "go", Frameworks: []string{"gin"},
	})
	if !strings.Contains(paymentsDoc, "# payments-service API — go / gin, 2 endpoints") {
		t.Errorf("output should contain the payments heading, got %q", firstLine(paymentsDoc))
	}
	if firstLine(ordersDoc) == firstLine(paymentsDoc) {
		t.Error("two projects should not produce the same first heading")
	}

	// One endpoint reads "1 endpoint", not "1 endpoints".
	singleDoc := render(t, f, endpoints[:1], Params{Variant: "detailed", Collector: "java"})
	if !strings.Contains(singleDoc, "# API — java, 1 endpoint\n") {
		t.Errorf("output should carry a singular endpoint count, got %q", firstLine(singleDoc))
	}

	// A caller that supplies no metadata gets a heading it recognises: an
	// export is never left with a bare "# API" claiming nothing.
	if got := firstLine(render(t, f, endpoints, Params{Variant: "detailed"})); got != "# API — 2 endpoints" {
		t.Errorf("heading without metadata = %q, want %q", got, "# API — 2 endpoints")
	}
}

// TestFormat_FieldDeclarationOrder covers the order fields are rendered in: a
// wrapper declared `code, message, data` is exported that way round, which is
// also the order the Query and Path tables already use (#170).
func TestFormat_FieldDeclarationOrder(t *testing.T) {
	f := New()

	body := model.NewObjectModelBuilder().
		IntField("code", model.WithComment("status code")).
		StringField("message").
		StringField("data").
		Build()
	endpoints := []model.ApiEndpoint{{
		Name:     "Get Overview",
		Path:     "/overview",
		Method:   "GET",
		Protocol: "http",
		Response: &model.ApiBody{MediaType: "application/json", Body: body},
	}}

	output := render(t, f, endpoints, Params{Variant: "detailed", OutputDemo: boolPtr(false)})
	assertRowOrder(t, "declaration", output,
		"| code | int | status code |",
		"| message | string |  |",
		"| data | string |  |",
	)

	// Alphabetical stays available, opt-in, for a consumer that prefers an
	// order stable under refactors over one faithful to the source.
	sorted := render(t, f, endpoints, Params{Variant: "detailed", OutputDemo: boolPtr(false), SortFields: true})
	assertRowOrder(t, "alphabetical", sorted,
		"| code | int | status code |",
		"| data | string |  |",
		"| message | string |  |",
	)
}

// TestFormat_HeadingsAreUnique covers resources that share method names: every
// `###` heading in the document must be different, so an anchor can address one
// endpoint and tooling keying on headings sees no duplicates (#170).
func TestFormat_HeadingsAreUnique(t *testing.T) {
	f := New()

	var endpoints []model.ApiEndpoint
	for _, folder := range []string{"OrderController", "ProductController"} {
		for _, name := range []string{"create", "update", "delete"} {
			endpoints = append(endpoints, model.ApiEndpoint{
				Name:     name,
				Folder:   folder,
				Path:     "/" + strings.ToLower(folder) + "/" + name,
				Method:   "POST",
				Protocol: "http",
			})
		}
	}

	params := Params{Variant: "detailed", OutputDemo: boolPtr(false)}
	output := render(t, f, endpoints, params)

	var headings []string
	for _, line := range strings.Split(output, "\n") {
		if strings.HasPrefix(line, "### ") {
			headings = append(headings, line)
		}
	}
	if len(headings) != len(endpoints) {
		t.Fatalf("got %d headings, want %d", len(headings), len(endpoints))
	}
	seen := make(map[string]bool, len(headings))
	for _, h := range headings {
		if seen[h] {
			t.Errorf("heading %q is not unique in the document", h)
		}
		seen[h] = true
	}

	// The resource qualifies the name, which is what a reader looks for.
	for _, want := range []string{"### OrderController / create", "### ProductController / create"} {
		if !strings.Contains(output, want) {
			t.Errorf("output should contain %q", want)
		}
	}

	// A name that occurs once stays bare: qualifying every heading would bury
	// the ones that need no qualification.
	unique := []model.ApiEndpoint{{Name: "getOverview", Path: "/overview", Method: "GET", Protocol: "http"}}
	if out := render(t, f, unique, params); !strings.Contains(out, "### getOverview\n") {
		t.Errorf("a unique name should stay unqualified:\n%s", out)
	}
}

// TestFormat_NoBasicTag covers the removal of the `> BASIC` marker: it was
// emitted once per endpoint and carried no information at all (#170).
func TestFormat_NoBasicTag(t *testing.T) {
	f := New()
	endpoints := []model.ApiEndpoint{
		{
			Name: "Get User", Path: "/users/{id}", Method: "GET", Protocol: "http",
			Parameters: []model.ApiParameter{{Name: "id", In: "path", Type: "text"}},
		},
		{Name: "Create User", Path: "/users", Method: "POST", Protocol: "http"},
	}

	output := render(t, f, endpoints, Params{Variant: "detailed"})
	if strings.Contains(output, "> BASIC") {
		t.Error("output should not contain the inert '> BASIC' marker")
	}
	// The markers that do carry meaning survive.
	if !strings.Contains(output, "> REQUEST") {
		t.Error("output should still contain '> REQUEST'")
	}
}

// render renders endpoints with the given params, failing the test on error.
func render(t *testing.T, f formatter.Formatter, endpoints []model.ApiEndpoint, p Params) string {
	t.Helper()
	paramsJSON, err := json.Marshal(p)
	if err != nil {
		t.Fatalf("marshalling params: %v", err)
	}
	out, err := f.Format(endpoints, formatter.FormatOptions{Params: paramsJSON})
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}
	// The templates are checked out with CRLF on Windows; the line endings are
	// not what these tests are about.
	return strings.ReplaceAll(string(out), "\r\n", "\n")
}

// assertRowOrder checks that the rows appear in the document in this order.
func assertRowOrder(t *testing.T, order string, output string, rows ...string) {
	t.Helper()
	pos := -1
	for _, row := range rows {
		at := strings.Index(output, row+"\n")
		if at < 0 {
			t.Fatalf("%s order: output should contain %q:\n%s", order, row, output)
		}
		if at < pos {
			t.Errorf("%s order: %q appears before the row that should precede it", order, row)
		}
		pos = at
	}
}

func firstLine(s string) string {
	if i := strings.Index(s, "\n"); i >= 0 {
		return s[:i]
	}
	return s
}

func boolPtr(b bool) *bool { return &b }
