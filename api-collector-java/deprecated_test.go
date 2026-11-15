package javacollector

import (
	"testing"

	collector "github.com/tangcent/apilot/api-collector"
)

// Each framework reports a deprecated handler, by annotation or doc tag, on
// the method or on its class.
func TestDeprecatedEndpoints(t *testing.T) {
	c := New().(*JavaCollector)
	endpoints, err := c.Collect(collector.CollectContext{SourceDir: "testdata/deprecated", NoDeps: true})
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]collector.ApiEndpoint{}
	for _, ep := range endpoints {
		got[ep.Path] = ep
	}
	for path, want := range map[string]struct {
		deprecated bool
		note       string
	}{
		"/orders/current":       {false, ""},
		"/orders/old":           {true, ""},
		"/orders/legacy":        {true, "use /orders/current instead"},
		"/legacy-resource/ping": {true, "replaced by OrderController"},
		"/level":                {true, ""},
	} {
		ep, ok := got[path]
		if !ok {
			t.Errorf("%s not collected; got %v", path, keys(got))
			continue
		}
		if ep.Deprecated != want.deprecated || ep.DeprecationNote != want.note {
			t.Errorf("%s: deprecated = %v, note %q; want %v, %q", path, ep.Deprecated, ep.DeprecationNote, want.deprecated, want.note)
		}
	}
	if legacy := got["/orders/legacy"]; legacy.Description != "Legacy lookup." {
		t.Errorf("the tag stays out of the description: %q", legacy.Description)
	}
}

func keys(m map[string]collector.ApiEndpoint) []string {
	var out []string
	for k := range m {
		out = append(out, k)
	}
	return out
}
