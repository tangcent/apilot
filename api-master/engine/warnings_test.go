package engine

import (
	"bytes"
	"os"
	"strings"
	"testing"

	collector "github.com/tangcent/apilot/api-collector"
)

// warningCollector is a collector that also reports gaps it worked around.
type warningCollector struct {
	name     string
	warnings []string
}

func (m *warningCollector) Name() string { return m.name }

func (m *warningCollector) SupportedLanguages() []string { return []string{"test"} }

func (m *warningCollector) Collect(ctx collector.CollectContext) ([]collector.ApiEndpoint, error) {
	return []collector.ApiEndpoint{{Name: "test-endpoint"}}, nil
}

func (m *warningCollector) Warnings() []string { return m.warnings }

func TestReportWarnings_Format(t *testing.T) {
	var buf bytes.Buffer

	reportWarnings(&buf, []string{
		"ResponseResource.update: the response entity built in the method body does not name a type",
		"ResponseResource.patch: the response entity built in the method body does not name a type",
	})

	want := "warning: ResponseResource.update: the response entity built in the method body does not name a type\n" +
		"warning: ResponseResource.patch: the response entity built in the method body does not name a type\n"
	if buf.String() != want {
		t.Errorf("reportWarnings output:\n%q\nwant:\n%q", buf.String(), want)
	}
}

func TestReportWarnings_EmptyWritesNothing(t *testing.T) {
	var buf bytes.Buffer

	reportWarnings(&buf, nil)
	reportWarnings(&buf, []string{})

	if buf.Len() != 0 {
		t.Errorf("no warnings must produce no output, got %q", buf.String())
	}
}

func TestRun_ReportsCollectorWarningsOnStderr(t *testing.T) {
	tmpDir := t.TempDir()

	RegisterCollector(&warningCollector{
		name:     "warning-collector",
		warnings: []string{"ReportResource.get: the response entity built in the method body does not name a type"},
	})
	RegisterFormatter(&mockFormatter{
		name:         "warning-formatter",
		formatOutput: []byte("formatted output"),
	})

	cfg := Config{
		SourceDir:      tmpDir,
		CollectorName:  "warning-collector",
		FormatterName:  "warning-formatter",
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

	// A marked section is a gap to fill, never a failure.
	if err != nil {
		t.Fatalf("Run must succeed even when the collector reports warnings, got: %v", err)
	}
	if !strings.Contains(stderr.String(), "warning: ReportResource.get") {
		t.Errorf("expected the collector warning on stderr, got: %q", stderr.String())
	}
	if stdout.String() != "formatted output" {
		t.Errorf("stdout must stay byte-identical, got: %q", stdout.String())
	}
}
