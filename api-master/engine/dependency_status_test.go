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

func TestReportDependencyResolution_ActiveWarnsAboutVersions(t *testing.T) {
	var buf bytes.Buffer

	reportDependencyResolution(&buf, collector.DependencyResolution{
		State:         collector.DependencyResolutionActive,
		ResolvedTypes: 3,
		Detail:        "the project's dependency tree is unavailable (mvn exited with code 1)",
		Unpinned:      []string{"com.example.Page (com.example:lib:1.0.0)"},
	})

	want := "info: resolved 3 types from dependencies\n" +
		"warning: dependency versions: the project's dependency tree is unavailable (mvn exited with code 1)\n" +
		"warning: 1 dependency type was not pinned to the project's version and read from the indexed one: com.example.Page (com.example:lib:1.0.0)\n"
	if buf.String() != want {
		t.Errorf("reportDependencyResolution output:\n%q\nwant:\n%q", buf.String(), want)
	}
}

func TestReportDependencyResolution_ActiveVersionDetailNeedsResolvedTypes(t *testing.T) {
	var buf bytes.Buffer

	// Nothing came from dependencies, so no version can be off.
	reportDependencyResolution(&buf, collector.DependencyResolution{
		State:  collector.DependencyResolutionActive,
		Detail: "the project's dependency tree is unavailable",
	})

	if buf.String() != "" {
		t.Errorf("expected no output, got %q", buf.String())
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

func TestReportDependencyResolution_ActiveReportsFailedLookups(t *testing.T) {
	var buf bytes.Buffer

	// Nothing resolved, but lookups failed: an export of opaque types must
	// not pass for a project that never referenced a dependency type.
	reportDependencyResolution(&buf, collector.DependencyResolution{
		State:  collector.DependencyResolutionActive,
		Detail: "maven-indexer-cli could not read the project's dependency tree (exit status 1)",
		Failed: []string{
			"com.example.Page: explore answered without a classes list; get-class: get-class failed: exit status 1",
			"com.example.Sort: explore failed: exit status 1",
		},
	})

	want := "warning: dependency versions: maven-indexer-cli could not read the project's dependency tree (exit status 1)\n" +
		"warning: 2 dependency type lookups failed: com.example.Page: explore answered without a classes list; get-class: get-class failed: exit status 1; com.example.Sort: explore failed: exit status 1\n"
	if buf.String() != want {
		t.Errorf("reportDependencyResolution output:\n%q\nwant:\n%q", buf.String(), want)
	}
}
