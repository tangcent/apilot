package kotlincollector

import (
	"path/filepath"
	"strings"
	"testing"

	collector "github.com/tangcent/apilot/api-collector"
)

func TestSyntaxErrorsDoNotAbortCollection(t *testing.T) {
	endpoints, c := collectDir(t, "testdata/syntax-recovery")
	if len(endpoints) != 2 {
		t.Fatalf("expected 2 endpoints, got %d: %v", len(endpoints), endpointNames(endpoints))
	}

	// The grammar rejects the annotation argument list after a property
	// initialised to null; blanking the arguments of @Range, which the export
	// never reads, makes the class readable.
	query := findEndpoint(t, endpoints, "query").RequestBody.Body
	for _, name := range []string{"recordNo", "generatedState", "operatorId"} {
		objectField(t, query, name)
	}

	// A stray ";" after the last enum entry leaves the entries readable.
	group := objectField(t, findEndpoint(t, endpoints, "sort").RequestBody.Body, "group")
	if len(group.Options) != 2 {
		t.Fatalf("group options = %#v", group.Options)
	}

	warnings := strings.Join(c.Warnings(), "\n")
	if !strings.Contains(warnings, "SettlementQuery.kt:") {
		t.Fatalf("expected a warning naming SettlementQuery.kt, got:\n%s", warnings)
	}
	// The annotation class with "= []" defaults is unparseable even after
	// recovery: it is skipped with a warning, not fatal.
	if !strings.Contains(warnings, "TypeSize.kt:") {
		t.Fatalf("expected a warning naming TypeSize.kt, got:\n%s", warnings)
	}
}

func TestSyntaxErrorInControllerStillFails(t *testing.T) {
	c := New().(*KotlinCollector)
	abs, err := filepath.Abs("testdata/syntax-controller")
	if err != nil {
		t.Fatal(err)
	}
	_, err = c.Collect(collector.CollectContext{SourceDir: abs, NoDeps: true})
	if err == nil {
		t.Fatal("expected an error: skipping part of a controller could drop endpoints silently")
	}
	if !strings.Contains(err.Error(), "BrokenController.kt:6") {
		t.Fatalf("error does not name the file and line: %v", err)
	}
}
