package pycollector

import (
	"os"
	"path/filepath"
	"strings"
	"errors"
	"testing"

	collector "github.com/tangcent/apilot/api-collector"
)

func writeRequirementsTxt(t *testing.T, dir string, lines ...string) {
	t.Helper()
	content := strings.Join(lines, "\n") + "\n"
	if err := os.WriteFile(filepath.Join(dir, "requirements.txt"), []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write requirements.txt: %v", err)
	}
}

func TestDependencyResolution_Disabled(t *testing.T) {
	c := New().(*PythonCollector)
	_, err := c.Collect(collector.CollectContext{SourceDir: "testdata/fastapionly", NoDeps: true})
	if err != nil {
		t.Fatalf("Collect failed: %v", err)
	}

	dr := c.DependencyResolution()
	if dr.State != collector.DependencyResolutionDisabled {
		t.Errorf("expected state disabled, got %q", dr.State)
	}
}

func TestDependencyResolution_NoManifestIsNoDeps(t *testing.T) {
	c := New().(*PythonCollector)

	// The framework testdata dirs hold only sources, no dependency manifest.
	_, err := c.Collect(collector.CollectContext{SourceDir: "testdata/fastapionly"})
	if err != nil {
		t.Fatalf("Collect failed: %v", err)
	}

	dr := c.DependencyResolution()
	if dr.State != collector.DependencyResolutionNoDeps {
		t.Errorf("expected state no-deps, got %q", dr.State)
	}
}

func TestDependencyResolution_NoEnvironmentIsToolMissing(t *testing.T) {
	dir := t.TempDir()
	writeRequirementsTxt(t, dir, "pydantic==2.5.0")

	orig := findSitePackages
	findSitePackages = func(sourceDir string) (string, error) {
		return "", errors.New("no Python environment found")
	}
	defer func() { findSitePackages = orig }()

	c := New().(*PythonCollector)
	_, err := c.Collect(collector.CollectContext{SourceDir: dir})
	if err != nil {
		t.Fatalf("Collect failed: %v", err)
	}

	dr := c.DependencyResolution()
	if dr.State != collector.DependencyResolutionToolMissing {
		t.Fatalf("expected state tool-missing, got %q", dr.State)
	}
	if !strings.Contains(dr.Detail, "no Python environment found") {
		t.Errorf("detail should explain the missing environment, got %q", dr.Detail)
	}
}

func TestDependencyResolution_VenvPresentIsActive(t *testing.T) {
	dir := t.TempDir()
	writeRequirementsTxt(t, dir, "pydantic==2.5.0")

	// A fake project virtualenv is enough for FindSitePackages to succeed
	// without depending on a Python installation on the machine.
	sitePackages := filepath.Join(dir, ".venv", "lib", "python3.11", "site-packages")
	if err := os.MkdirAll(sitePackages, 0o755); err != nil {
		t.Fatalf("failed to create fake venv: %v", err)
	}

	c := New().(*PythonCollector)
	_, err := c.Collect(collector.CollectContext{SourceDir: dir})
	if err != nil {
		t.Fatalf("Collect failed: %v", err)
	}

	dr := c.DependencyResolution()
	if dr.State != collector.DependencyResolutionActive {
		t.Errorf("expected state active, got %q", dr.State)
	}
	if dr.ResolvedTypes != 0 {
		t.Errorf("expected 0 resolved types for a project without sources, got %d", dr.ResolvedTypes)
	}
}
