package kotlincollector

import (
	"testing"
)

func TestPropertyKDocIsCleaned(t *testing.T) {
	endpoints, _ := collectDir(t, "testdata/kdoc")
	ep := findEndpoint(t, endpoints, "create")
	if ep.Description != "Create an order." {
		t.Fatalf("endpoint description = %q", ep.Description)
	}
	req := ep.RequestBody.Body
	for name, want := range map[string]string{
		"orderNo": "Purchase order number.",
		// Block tags such as @see are not part of the description, as on Java.
		"kind":   "Kind of the order.",
		"remark": "Single-line comment.",
	} {
		if got := objectField(t, req, name).Comment; got != want {
			t.Errorf("%s comment = %q, want %q", name, got, want)
		}
	}
}
