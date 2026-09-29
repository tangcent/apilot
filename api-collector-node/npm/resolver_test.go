package npm

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// writeDependencyFixture builds a project directory declaring one dependency
// whose types live only in node_modules/@types.
func writeDependencyFixture(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	pkgJSON, err := json.Marshal(map[string]any{
		"name":         "demo",
		"dependencies": map[string]string{"external-models": "^1.0.0"},
	})
	if err != nil {
		t.Fatalf("failed to marshal package.json: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "package.json"), pkgJSON, 0o644); err != nil {
		t.Fatalf("failed to write package.json: %v", err)
	}

	atTypes := filepath.Join(dir, "node_modules", "@types", "external-models")
	if err := os.MkdirAll(atTypes, 0o755); err != nil {
		t.Fatalf("failed to create @types dir: %v", err)
	}
	dts := "export interface ExternalUser {\n  id: number;\n  email: string;\n}\n"
	if err := os.WriteFile(filepath.Join(atTypes, "index.d.ts"), []byte(dts), 0o644); err != nil {
		t.Fatalf("failed to write index.d.ts: %v", err)
	}

	return dir
}

func TestNpmTypeResolver_ResolvedTypeFromNodeModules(t *testing.T) {
	dir := writeDependencyFixture(t)

	r := NewNpmTypeResolver(dir)
	rt := r.ResolveType("ExternalUser")
	if rt == nil {
		t.Fatal("expected ExternalUser to resolve from node_modules")
	}
	if len(rt.Fields) != 2 {
		t.Errorf("expected 2 fields, got %d", len(rt.Fields))
	}
	if got := r.ResolvedCount(); got != 1 {
		t.Errorf("expected ResolvedCount 1, got %d", got)
	}

	// A repeated request must not inflate the count.
	_ = r.ResolveType("ExternalUser")
	if got := r.ResolvedCount(); got != 1 {
		t.Errorf("expected ResolvedCount to stay 1, got %d", got)
	}

	// A type the dependencies do not declare stays unresolved and uncounted.
	if rt := r.ResolveType("NoSuchType"); rt != nil {
		t.Errorf("expected NoSuchType to be unresolved, got %+v", rt)
	}
	if got := r.ResolvedCount(); got != 1 {
		t.Errorf("expected ResolvedCount to stay 1 after a miss, got %d", got)
	}
}
