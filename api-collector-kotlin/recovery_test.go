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
	for _, tc := range []struct{ dir, where string }{
		{"testdata/syntax-controller", "BrokenController.kt:6"},
		// A fully qualified @RestController is a controller too.
		{"testdata/syntax-controller-qualified", "BrokenController.kt:5"},
	} {
		t.Run(tc.dir, func(t *testing.T) {
			c := New().(*KotlinCollector)
			abs, err := filepath.Abs(tc.dir)
			if err != nil {
				t.Fatal(err)
			}
			_, err = c.Collect(collector.CollectContext{SourceDir: abs, NoDeps: true})
			if err == nil {
				t.Fatal("expected an error: skipping part of a controller could drop endpoints silently")
			}
			if !strings.Contains(err.Error(), tc.where) {
				t.Fatalf("error does not name the file and line %s: %v", tc.where, err)
			}
		})
	}
}

func TestSyntaxErrorMentioningAControllerIsNotOne(t *testing.T) {
	// "@RestController" in a string or a comment does not make a file a
	// controller, so the damaged file is recovered, not fatal.
	_, c := collectDir(t, "testdata/syntax-string-mention")
	if !strings.Contains(strings.Join(c.Warnings(), "\n"), "Marker is skipped") {
		t.Fatalf("warnings = %v", c.Warnings())
	}
}

func TestSyntaxErrorSkipsOnlyTheDamagedDeclaration(t *testing.T) {
	endpoints, c := collectDir(t, "testdata/syntax-partial")

	// The declarations around the damaged one are read in full, including the
	// one the grammar folded into the error when it parsed the whole file.
	objectField(t, findEndpoint(t, endpoints, "first").Response.Body, "a")
	objectField(t, findEndpoint(t, endpoints, "third").Response.Body, "c")

	// The damaged declaration is skipped whole: inside it the grammar no
	// longer tells where one member ends. It is named with its line, and is an
	// unresolved type where it is referenced.
	if body := findEndpoint(t, endpoints, "damaged").Response.Body; body.Kind == "object" {
		t.Fatalf("Damaged should not be read in part: %v", fieldNames(body))
	}
	warnings := strings.Join(c.Warnings(), "\n")
	if !strings.Contains(warnings, "Models.kt:10: Damaged is skipped") {
		t.Fatalf("expected Damaged to be named with its line, got:\n%s", warnings)
	}
	if strings.Contains(warnings, "Third is skipped") || strings.Contains(warnings, "First is skipped") {
		t.Fatalf("intact declarations must not be skipped:\n%s", warnings)
	}
	if _, ok := c.Unresolved()["Damaged"]; !ok {
		t.Fatalf("the skipped class should be reported as unresolved: %v", c.Unresolved())
	}
}
