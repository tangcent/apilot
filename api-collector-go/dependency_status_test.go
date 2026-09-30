package gocollector

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	collector "github.com/tangcent/apilot/api-collector"
)

func TestDependencyResolution_Disabled(t *testing.T) {
	c := New().(*GoCollector)
	_, err := c.Collect(collector.CollectContext{SourceDir: "testdata/ginonly", NoDeps: true})
	if err != nil {
		t.Fatalf("Collect failed: %v", err)
	}

	dr := c.DependencyResolution()
	if dr.State != collector.DependencyResolutionDisabled {
		t.Errorf("expected state disabled, got %q", dr.State)
	}
}

func TestDependencyResolution_NoGoModIsNoDeps(t *testing.T) {
	c := New().(*GoCollector)

	// The framework testdata dirs hold only Go sources, no go.mod.
	_, err := c.Collect(collector.CollectContext{SourceDir: "testdata/ginonly"})
	if err != nil {
		t.Fatalf("Collect failed: %v", err)
	}

	dr := c.DependencyResolution()
	if dr.State != collector.DependencyResolutionNoDeps {
		t.Errorf("expected state no-deps, got %q", dr.State)
	}
}

func TestDependencyResolution_GoMissingIsToolMissing(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/demo\n\ngo 1.22\n"), 0o644); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}

	orig := goOnPath
	goOnPath = func() bool { return false }
	defer func() { goOnPath = orig }()

	c := New().(*GoCollector)
	_, err := c.Collect(collector.CollectContext{SourceDir: dir})
	if err != nil {
		t.Fatalf("Collect failed: %v", err)
	}

	dr := c.DependencyResolution()
	if dr.State != collector.DependencyResolutionToolMissing {
		t.Fatalf("expected state tool-missing, got %q", dr.State)
	}
	if !strings.Contains(dr.Detail, "go toolchain not found on PATH") {
		t.Errorf("detail should name the missing toolchain, got %q", dr.Detail)
	}
	if !strings.Contains(dr.Detail, "https://go.dev/dl/") {
		t.Errorf("detail should carry install guidance, got %q", dr.Detail)
	}
}

func TestDependencyResolution_GoPresentIsActive(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/demo\n\ngo 1.22\n"), 0o644); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}

	orig := goOnPath
	goOnPath = func() bool { return true }
	defer func() { goOnPath = orig }()

	c := New().(*GoCollector)
	_, err := c.Collect(collector.CollectContext{SourceDir: dir})
	if err != nil {
		t.Fatalf("Collect failed: %v", err)
	}

	dr := c.DependencyResolution()
	if dr.State != collector.DependencyResolutionActive {
		t.Errorf("expected state active, got %q", dr.State)
	}
}
