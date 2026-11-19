package yapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestLoadRules(t *testing.T) {
	// A remote rules file, as a team shares one.
	remote := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("# shared mocks\n" +
			"mtimestamp=@now(\"T\")\n" +
			"natural=@natural(0,1000)\n" +
			"mock.[code|integer]=10000\n" +
			"mock.[*Time|string]=${mtimestamp}\n" +
			"mock.[*.name|string]=@cname\n" +
			"mock.[*|integer]=${natural}\n" +
			"mock.rule=*.email|string=@email\n"))
	}))
	defer remote.Close()

	project := t.TempDir()
	// A multi-module project keeps the file in a module.
	writeFile(t, filepath.Join(project, "order-server", RulesFileName),
		"# Long as String\n"+
			"json.rule.convert[java.lang.Long]=java.lang.String\n"+
			"json.rule.convert[java.util.Date]=java.lang.String\n"+
			"properties.additional="+remote.URL+"/rules.config\n")
	// Build output is not searched.
	writeFile(t, filepath.Join(project, "target", RulesFileName), "mock.[*|string]=never\n")
	extra := filepath.Join(t.TempDir(), "extra.config")
	writeFile(t, extra, "properties.additional=more.config\n")
	writeFile(t, filepath.Join(filepath.Dir(extra), "more.config"), "mock.[*|boolean]=@boolean\n")

	rules, err := LoadRules(project, []string{extra}, HTTPFetcher(remote.Client()))
	if err != nil {
		t.Fatal(err)
	}
	if len(rules.Sources) != 4 {
		t.Fatalf("sources = %v", rules.Sources)
	}
	if got := rules.convertType("long"); got != "string" {
		t.Errorf("long converts to %q", got)
	}
	if len(rules.Warnings) != 1 || !strings.Contains(rules.Warnings[0], "java.util.Date") {
		t.Errorf("warnings = %v", rules.Warnings)
	}

	for _, tc := range []struct {
		path, jsonType, want string
	}{
		{"code", "integer", "10000"},                 // exact path, root only
		{"data.code", "integer", "@natural(0,1000)"}, // falls through to [*|integer]
		{"data.createTime", "string", `@now("T")`},   // ${} resolved
		{"data.items[].name", "string", "@cname"},    // "*." matches any parent
		{"name", "string", "@cname"},                 // and no parent
		{"data.nickname", "string", ""},              // "*.name" needs a dot boundary
		{"flag", "boolean", "@boolean"},              // from the included extra file
		{"data.email", "string", ""},                 // 3.x mock.rule lines are not body rules
	} {
		got, _ := rules.mockFor(tc.path, tc.jsonType)
		if got != tc.want {
			t.Errorf("mockFor(%q, %q) = %q, want %q", tc.path, tc.jsonType, got, tc.want)
		}
	}
}

func TestLoadRulesFailsOnUnreadableInclude(t *testing.T) {
	project := t.TempDir()
	writeFile(t, filepath.Join(project, RulesFileName), "properties.additional=missing.config\n")
	if _, err := LoadRules(project, nil, nil); err == nil || !strings.Contains(err.Error(), "missing.config") {
		t.Fatalf("an unreadable include should fail and name the file, got %v", err)
	}
}

// A rule names a boxed type by its simple name as often as by its qualified
// one: `Long` and `Integer` are read, and so are `Short` and `Byte`, which the
// model holds as an int as well.
func TestLoadRulesReadsShortAndByteByTheirSimpleNames(t *testing.T) {
	for _, from := range []string{"Short", "java.lang.Short", "short", "Byte", "java.lang.Byte", "byte"} {
		project := t.TempDir()
		writeFile(t, filepath.Join(project, RulesFileName),
			"json.rule.convert["+from+"]=java.lang.String\n")
		rules, err := LoadRules(project, nil, nil)
		if err != nil {
			t.Fatal(err)
		}
		if len(rules.Warnings) != 0 {
			t.Errorf("convert[%s]: warnings = %v, want none", from, rules.Warnings)
		}
		if got := rules.convertType("int"); got != "string" {
			t.Errorf("convert[%s]: an int converts to %q, want string", from, got)
		}
	}
}

func TestSchemaAppliesRules(t *testing.T) {
	project := t.TempDir()
	writeFile(t, filepath.Join(project, RulesFileName),
		"json.rule.convert[java.lang.Long]=java.lang.String\n"+
			"mock.[code|integer]=10000\n"+
			"mock.[*Time|string]=@now\n"+
			"mock.[*.name|string]=@cname\n"+
			"mock.[*|string]=@string\n")
	rules, err := LoadRules(project, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	ep := orderEndpoint()
	it := buildInterface(ep, &schemaBuilder{rules: rules})

	var res struct {
		Properties map[string]json.RawMessage `json:"properties"`
	}
	if err := json.Unmarshal([]byte(it.ResBody), &res); err != nil {
		t.Fatal(err)
	}
	if got := string(res.Properties["code"]); got != `{"type":"integer","mock":{"mock":"10000"}}` {
		t.Errorf("code = %s", got)
	}
	data := string(res.Properties["data"])
	// A long converted to string takes the string rule for its path.
	if !strings.Contains(data, `"createTime":{"type":"string","mock":{"mock":"@now"}}`) {
		t.Errorf("createTime in %s", data)
	}
	if !strings.Contains(data, `"name":{"type":"string","mock":{"mock":"@cname"}}`) {
		t.Errorf("items[].name in %s", data)
	}

	// An enum's own pick mock is kept over a rule.
	if !strings.Contains(it.ReqBodyOther, `"status":{"type":"string","enum":["OPEN","CLOSED"],"enumDesc":"OPEN: open","mock":{"mock":"@pick(`) {
		t.Errorf("status lost its pick mock: %s", it.ReqBodyOther)
	}
	// The converted amount is a string and takes the catch-all string rule.
	if !strings.Contains(it.ReqBodyOther, `"amount":{"type":"string","mock":{"mock":"@string"}}`) {
		t.Errorf("amount in %s", it.ReqBodyOther)
	}
}

func TestGlobMatcher(t *testing.T) {
	for _, tc := range []struct {
		pattern, value string
		want           bool
	}{
		{"*", "anything", true},
		{"", "anything", true},
		{"code", "code", true},
		{"code", "data.code", false},
		{"*Code", "data.errorCode", true},
		{"*.id", "id", true},
		{"*.id", "user.id", true},
		{"*.id", "userid", false},
		{"*.*time*", "data.updatetime", true},
		{"a.b", "aXb", false},
	} {
		if got := globMatcher(tc.pattern)(tc.value); got != tc.want {
			t.Errorf("globMatcher(%q)(%q) = %v, want %v", tc.pattern, tc.value, got, tc.want)
		}
	}
}
