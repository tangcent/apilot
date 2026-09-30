package nodecollector

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	collector "github.com/tangcent/apilot/api-collector"
)

func writePackageJSON(t *testing.T, dir string, deps map[string]string) {
	t.Helper()
	content, err := json.Marshal(map[string]any{"name": "demo", "dependencies": deps})
	if err != nil {
		t.Fatalf("failed to marshal package.json: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "package.json"), content, 0o644); err != nil {
		t.Fatalf("failed to write package.json: %v", err)
	}
}

func TestDependencyResolution_Disabled(t *testing.T) {
	c := New().(*NodeCollector)
	_, err := c.Collect(collector.CollectContext{SourceDir: "testdata/mixed", NoDeps: true})
	if err != nil {
		t.Fatalf("Collect failed: %v", err)
	}

	dr := c.DependencyResolution()
	if dr.State != collector.DependencyResolutionDisabled {
		t.Errorf("expected state disabled, got %q", dr.State)
	}
}

func TestDependencyResolution_NoPackageJSONIsNoDeps(t *testing.T) {
	c := New().(*NodeCollector)

	// The framework testdata dirs hold only sources, no package.json.
	_, err := c.Collect(collector.CollectContext{SourceDir: "testdata/mixed"})
	if err != nil {
		t.Fatalf("Collect failed: %v", err)
	}

	dr := c.DependencyResolution()
	if dr.State != collector.DependencyResolutionNoDeps {
		t.Errorf("expected state no-deps, got %q", dr.State)
	}
}

func TestDependencyResolution_NodeModulesMissingIsToolMissing(t *testing.T) {
	dir := t.TempDir()
	writePackageJSON(t, dir, map[string]string{"fastify": "^4.0.0"})

	c := New().(*NodeCollector)
	_, err := c.Collect(collector.CollectContext{SourceDir: dir})
	if err != nil {
		t.Fatalf("Collect failed: %v", err)
	}

	dr := c.DependencyResolution()
	if dr.State != collector.DependencyResolutionToolMissing {
		t.Fatalf("expected state tool-missing, got %q", dr.State)
	}
	if !strings.Contains(dr.Detail, "node_modules not found") {
		t.Errorf("detail should name the missing artifacts, got %q", dr.Detail)
	}
	if !strings.Contains(dr.Detail, "npm install") {
		t.Errorf("detail should say how to restore the capability, got %q", dr.Detail)
	}
}

func TestDependencyResolution_NoDeclaredDepsIsNoDeps(t *testing.T) {
	dir := t.TempDir()
	writePackageJSON(t, dir, nil)
	if err := os.MkdirAll(filepath.Join(dir, "node_modules"), 0o755); err != nil {
		t.Fatalf("failed to create node_modules: %v", err)
	}

	c := New().(*NodeCollector)
	_, err := c.Collect(collector.CollectContext{SourceDir: dir})
	if err != nil {
		t.Fatalf("Collect failed: %v", err)
	}

	dr := c.DependencyResolution()
	if dr.State != collector.DependencyResolutionNoDeps {
		t.Errorf("expected state no-deps, got %q", dr.State)
	}
}

func TestDependencyResolution_NodeModulesPresentIsActive(t *testing.T) {
	dir := t.TempDir()
	writePackageJSON(t, dir, map[string]string{"fastify": "^4.0.0"})
	if err := os.MkdirAll(filepath.Join(dir, "node_modules"), 0o755); err != nil {
		t.Fatalf("failed to create node_modules: %v", err)
	}

	c := New().(*NodeCollector)
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
