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

// A blank line of a KDoc comment separates paragraphs, so a renderer can tell
// them from wrapped lines.
func TestKDocKeepsParagraphs(t *testing.T) {
	endpoints, _ := collectDir(t, "testdata/kdoc")
	ep := findEndpoint(t, endpoints, "draft")
	if want := "Save a draft.\n\nA draft is not submitted\nuntil it is confirmed."; ep.Description != want {
		t.Errorf("endpoint description = %q, want %q", ep.Description, want)
	}
	if got, want := objectField(t, ep.RequestBody.Body, "note").Comment, "Free-text note,\nshown to the buyer.\n\nKept for a year."; got != want {
		t.Errorf("note comment = %q, want %q", got, want)
	}
}
