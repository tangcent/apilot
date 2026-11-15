package kotlincollector

import "testing"

// Kotlin's @Deprecated marks an endpoint deprecated with its message, on the
// function or on its class.
func TestDeprecatedEndpoints(t *testing.T) {
	endpoints, _ := collectDir(t, "testdata/deprecated")
	for name, want := range map[string]struct {
		deprecated bool
		note       string
	}{
		"current": {false, ""},
		"old":     {true, "Use /orders/current"},
		"ping":    {true, "Kept for old clients"},
	} {
		ep := findEndpoint(t, endpoints, name)
		if ep.Deprecated != want.deprecated || ep.DeprecationNote != want.note {
			t.Errorf("%s: deprecated = %v, note %q; want %v, %q", name, ep.Deprecated, ep.DeprecationNote, want.deprecated, want.note)
		}
	}
}
