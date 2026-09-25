package engine

import (
	"bytes"
	"os"
	"strings"
	"testing"

	collector "github.com/tangcent/apilot/api-collector"
)

// unresolvedCollector is a collector that also reports unresolved types.
type unresolvedCollector struct {
	name             string
	collectEndpoints []collector.ApiEndpoint
	unresolved       map[string]int
}

func (m *unresolvedCollector) Name() string { return m.name }

func (m *unresolvedCollector) SupportedLanguages() []string { return []string{"test"} }

func (m *unresolvedCollector) Collect(ctx collector.CollectContext) ([]collector.ApiEndpoint, error) {
	return m.collectEndpoints, nil
}

func (m *unresolvedCollector) Unresolved() map[string]int { return m.unresolved }

func TestReportUnresolved_Format(t *testing.T) {
	var buf bytes.Buffer

	reportUnresolved(&buf, map[string]int{"PageResult": 9, "ApiResponse": 4, "UserPrincipal": 1})

	want := "warning: 3 types could not be resolved\n" +
		"  PageResult           9 occurrences\n" +
		"  ApiResponse          4 occurrences\n" +
		"  UserPrincipal        1 occurrence\n"
	if buf.String() != want {
		t.Errorf("reportUnresolved output:\n%q\nwant:\n%q", buf.String(), want)
	}
}

func TestReportUnresolved_SortsByCountThenName(t *testing.T) {
	var buf bytes.Buffer

	reportUnresolved(&buf, map[string]int{"Bee": 2, "Ant": 2, "Cow": 5})

	lines := strings.Split(strings.TrimRight(buf.String(), "\n"), "\n")
	if len(lines) != 4 {
		t.Fatalf("expected 4 lines, got %d: %q", len(lines), buf.String())
	}
	if !strings.Contains(lines[0], "3 types could not be resolved") {
		t.Errorf("summary line = %q", lines[0])
	}
	if !strings.Contains(lines[1], "Cow") {
		t.Errorf("highest count must come first, got %q", lines[1])
	}
	if !strings.Contains(lines[2], "Ant") || !strings.Contains(lines[3], "Bee") {
		t.Errorf("ties must be ordered by name, got %q and %q", lines[2], lines[3])
	}
}

func TestReportUnresolved_EmptyWritesNothing(t *testing.T) {
	var buf bytes.Buffer

	reportUnresolved(&buf, nil)
	reportUnresolved(&buf, map[string]int{})

	if buf.Len() != 0 {
		t.Errorf("no unresolved types must produce no output, got %q", buf.String())
	}
}

func TestRun_ReportsUnresolvedTypesOnStderr(t *testing.T) {
	tmpDir := t.TempDir()

	RegisterCollector(&unresolvedCollector{
		name:             "unresolved-collector",
		collectEndpoints: []collector.ApiEndpoint{{Name: "test-endpoint"}},
		unresolved:       map[string]int{"PageResult": 9},
	})
	RegisterFormatter(&mockFormatter{
		name:         "test-formatter",
		formatOutput: []byte("formatted output"),
	})

	cfg := Config{
		SourceDir:      tmpDir,
		CollectorName:  "unresolved-collector",
		FormatterName:  "test-formatter",
		PluginRegistry: "",
	}

	var stdout, stderr bytes.Buffer
	origStdout, origStderr := os.Stdout, os.Stderr
	outR, outW, _ := os.Pipe()
	errR, errW, _ := os.Pipe()
	os.Stdout, os.Stderr = outW, errW

	err := Run(cfg)

	outW.Close()
	errW.Close()
	os.Stdout, os.Stderr = origStdout, origStderr
	stdout.ReadFrom(outR)
	stderr.ReadFrom(errR)

	// Unresolved types are a diagnostic, never a failure.
	if err != nil {
		t.Fatalf("Run must succeed even with unresolved types, got: %v", err)
	}
	if !strings.Contains(stderr.String(), "warning: 1 types could not be resolved") {
		t.Errorf("expected unresolved summary on stderr, got: %q", stderr.String())
	}
	if !strings.Contains(stderr.String(), "PageResult") || !strings.Contains(stderr.String(), "9 occurrences") {
		t.Errorf("expected type detail on stderr, got: %q", stderr.String())
	}
	if stdout.String() != "formatted output" {
		t.Errorf("stdout must stay byte-identical, got: %q", stdout.String())
	}
}

func TestRun_SilentWhenCollectorReportsNothing(t *testing.T) {
	tmpDir := t.TempDir()

	// mockCollector deliberately does not implement UnresolvedReporter.
	RegisterCollector(&mockCollector{
		name:             "plain-collector",
		supportedLangs:   []string{"test"},
		collectEndpoints: []collector.ApiEndpoint{{Name: "test-endpoint"}},
	})
	RegisterFormatter(&mockFormatter{
		name:         "test-formatter",
		formatOutput: []byte("formatted output"),
	})

	cfg := Config{
		SourceDir:      tmpDir,
		CollectorName:  "plain-collector",
		FormatterName:  "test-formatter",
		PluginRegistry: "",
	}

	var stdout, stderr bytes.Buffer
	origStdout, origStderr := os.Stdout, os.Stderr
	outR, outW, _ := os.Pipe()
	errR, errW, _ := os.Pipe()
	os.Stdout, os.Stderr = outW, errW

	err := Run(cfg)

	outW.Close()
	errW.Close()
	os.Stdout, os.Stderr = origStdout, origStderr
	stdout.ReadFrom(outR)
	stderr.ReadFrom(errR)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if strings.Contains(stderr.String(), "could not be resolved") {
		t.Errorf("collector without UnresolvedReporter must stay silent, got: %q", stderr.String())
	}
}
