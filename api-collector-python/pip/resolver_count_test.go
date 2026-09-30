package pip

import (
	"os"
	"path/filepath"
	"testing"
)

// writeVenvFixture builds a project with a requirements.txt and a fake venv
// whose site-packages hold one pydantic model package.
func writeVenvFixture(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	requirements := "pydantic==2.5.0\nexternal-models==1.0.0\n"
	if err := os.WriteFile(filepath.Join(dir, "requirements.txt"), []byte(requirements), 0o644); err != nil {
		t.Fatalf("failed to write requirements.txt: %v", err)
	}

	pkgDir := filepath.Join(dir, ".venv", "lib", "python3.11", "site-packages", "external_models")
	if err := os.MkdirAll(pkgDir, 0o755); err != nil {
		t.Fatalf("failed to create package dir: %v", err)
	}
	modelSrc := "from pydantic import BaseModel\n\n" +
		"class ExternalItem(BaseModel):\n" +
		"    name: str\n" +
		"    quantity: int = 1\n"
	if err := os.WriteFile(filepath.Join(pkgDir, "models.py"), []byte(modelSrc), 0o644); err != nil {
		t.Fatalf("failed to write models.py: %v", err)
	}

	return dir
}

func TestPipTypeResolver_ResolvesTypeFromSitePackages(t *testing.T) {
	dir := writeVenvFixture(t)

	r := NewPipTypeResolver(dir)
	rt := r.ResolveType("ExternalItem")
	if rt == nil {
		t.Fatal("expected ExternalItem to resolve from site-packages")
	}
	if len(rt.Fields) != 2 {
		t.Errorf("expected 2 fields, got %d", len(rt.Fields))
	}
	if got := r.ResolvedCount(); got != 1 {
		t.Errorf("expected ResolvedCount 1, got %d", got)
	}

	// A repeated request must not inflate the count.
	_ = r.ResolveType("ExternalItem")
	if got := r.ResolvedCount(); got != 1 {
		t.Errorf("expected ResolvedCount to stay 1, got %d", got)
	}

	// A type the dependencies do not declare stays unresolved and uncounted.
	if rt := r.ResolveType("NoSuchModel"); rt != nil {
		t.Errorf("expected NoSuchModel to be unresolved, got %+v", rt)
	}
	if got := r.ResolvedCount(); got != 1 {
		t.Errorf("expected ResolvedCount to stay 1 after a miss, got %d", got)
	}
}

func TestHasDependencyManifest(t *testing.T) {
	dir := t.TempDir()
	if HasDependencyManifest(dir) {
		t.Error("empty project should have no dependency manifest")
	}

	if err := os.WriteFile(filepath.Join(dir, "pyproject.toml"), []byte("[project]\n"), 0o644); err != nil {
		t.Fatalf("failed to write pyproject.toml: %v", err)
	}
	if !HasDependencyManifest(dir) {
		t.Error("pyproject.toml should count as a dependency manifest")
	}
}
