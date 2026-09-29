package engine

import (
	"bytes"
	"testing"

	collector "github.com/tangcent/apilot/api-collector"
)

func TestReportDependencyResolution_Active(t *testing.T) {
	var buf bytes.Buffer

	reportDependencyResolution(&buf, collector.DependencyResolution{
		State:         collector.DependencyResolutionActive,
		ResolvedTypes: 7,
	})

	want := "info: resolved 7 types from dependencies\n"
	if buf.String() != want {
		t.Errorf("reportDependencyResolution output:\n%q\nwant:\n%q", buf.String(), want)
	}
}

func TestReportDependencyResolution_ActiveSingular(t *testing.T) {
	var buf bytes.Buffer

	reportDependencyResolution(&buf, collector.DependencyResolution{
		State:         collector.DependencyResolutionActive,
		ResolvedTypes: 1,
	})

	want := "info: resolved 1 type from dependencies\n"
	if buf.String() != want {
		t.Errorf("reportDependencyResolution output:\n%q\nwant:\n%q", buf.String(), want)
	}
}

func TestReportDependencyResolution_ActiveZeroIsSilent(t *testing.T) {
	var buf bytes.Buffer

	// Active with nothing resolved means the endpoints never referenced a
	// dependency type; there is nothing to act on.
	reportDependencyResolution(&buf, collector.DependencyResolution{
		State: collector.DependencyResolutionActive,
	})

	if buf.String() != "" {
		t.Errorf("expected no output for active with zero resolved types, got %q", buf.String())
	}
}

func TestReportDependencyResolution_ToolMissing(t *testing.T) {
	var buf bytes.Buffer

	reportDependencyResolution(&buf, collector.DependencyResolution{
		State:  collector.DependencyResolutionToolMissing,
		Detail: "maven-indexer-cli not found on PATH",
	})

	want := "warning: dependency type resolution skipped: maven-indexer-cli not found on PATH\n"
	if buf.String() != want {
		t.Errorf("reportDependencyResolution output:\n%q\nwant:\n%q", buf.String(), want)
	}
}

func TestReportDependencyResolution_Disabled(t *testing.T) {
	var buf bytes.Buffer

	reportDependencyResolution(&buf, collector.DependencyResolution{
		State: collector.DependencyResolutionDisabled,
	})

	want := "info: dependency type resolution disabled (--no-deps)\n"
	if buf.String() != want {
		t.Errorf("reportDependencyResolution output:\n%q\nwant:\n%q", buf.String(), want)
	}
}

func TestReportDependencyResolution_NoDepsIsSilent(t *testing.T) {
	var buf bytes.Buffer

	reportDependencyResolution(&buf, collector.DependencyResolution{
		State: collector.DependencyResolutionNoDeps,
	})

	if buf.String() != "" {
		t.Errorf("expected no output for no-deps, got %q", buf.String())
	}
}
