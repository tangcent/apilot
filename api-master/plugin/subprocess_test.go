package plugin

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	fmtr "github.com/tangcent/apilot/api-formatter"
)

// binaries holds paths to compiled test helper binaries, built once in TestMain.
var binaries = map[string]string{}

// TestMain builds all fake subprocess binaries before running tests.
// This is the cross-platform alternative to writing shell scripts.
func TestMain(m *testing.M) {
	tmpDir, err := os.MkdirTemp("", "apilot-plugin-test-*")
	if err != nil {
		panic("failed to create temp dir: " + err.Error())
	}
	defer os.RemoveAll(tmpDir)

	helpers := []string{
		"fake-collector",
		"fake-formatter",
		"fake-plugin-invalid-json",
		"fake-plugin-empty-array",
		"fake-plugin-with-args",
	}

	// Resolve the package directory so paths work regardless of working directory.
	_, thisFile, _, _ := runtime.Caller(0)
	pkgDir := filepath.Dir(thisFile)

	for _, name := range helpers {
		binPath := filepath.Join(tmpDir, name)
		if runtime.GOOS == "windows" {
			binPath += ".exe"
		}
		srcDir := filepath.Join(pkgDir, "testdata", name)
		cmd := exec.Command("go", "build", "-o", binPath, srcDir)
		cmd.Stdout = os.Stderr
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			panic("failed to build " + name + ": " + err.Error())
		}
		binaries[name] = binPath
	}

	os.Exit(m.Run())
}

func TestSubprocessCollector_SupportedLanguages(t *testing.T) {
	manifest := PluginManifest{
		Name:    "test-collector",
		Type:    "collector",
		Command: binaries["fake-collector"],
	}

	collector, err := newSubprocessCollector(manifest)
	if err != nil {
		t.Fatalf("Failed to create collector: %v", err)
	}

	langs := collector.SupportedLanguages()
	if len(langs) != 2 {
		t.Errorf("Expected 2 languages, got: %d", len(langs))
	}

	expected := []string{"java", "kotlin"}
	for i, lang := range langs {
		if lang != expected[i] {
			t.Errorf("Expected language %q, got: %q", expected[i], lang)
		}
	}
}

func TestSubprocessCollector_SupportedLanguages_Error(t *testing.T) {
	manifest := PluginManifest{
		Name:    "test-collector",
		Type:    "collector",
		Command: "/nonexistent/binary",
	}

	collector := &subprocessCollector{manifest: manifest}
	langs := collector.SupportedLanguages()
	if langs != nil {
		t.Errorf("Expected nil for non-existent binary, got: %v", langs)
	}
}

func TestSubprocessFormatter_Format_PassesParams(t *testing.T) {
	manifest := PluginManifest{
		Name:    "test-formatter",
		Type:    "formatter",
		Command: binaries["fake-formatter"],
	}

	formatter, err := newSubprocessFormatter(manifest)
	if err != nil {
		t.Fatalf("Failed to create formatter: %v", err)
	}

	out, err := formatter.Format(nil, fmtr.FormatOptions{})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if string(out) != "ok" {
		t.Errorf("Expected 'ok', got: %q", string(out))
	}
}

func TestSubprocessFormatter_SatisfiesInterface(t *testing.T) {
	// SupportedFormats was removed from the Formatter interface.
	// Verify the subprocess formatter still satisfies it.
	f := &subprocessFormatter{manifest: PluginManifest{Name: "test-formatter"}}
	if f.Name() != "test-formatter" {
		t.Errorf("Expected name 'test-formatter', got: %q", f.Name())
	}
}

func TestQuerySubprocessFlag_InvalidJSON(t *testing.T) {
	manifest := PluginManifest{
		Name:    "test-plugin",
		Command: binaries["fake-plugin-invalid-json"],
	}

	result, err := querySubprocessFlag(manifest, "--test-flag")
	if err == nil {
		t.Errorf("Expected error for invalid JSON, got result: %v", result)
	}
}

func TestQuerySubprocessFlag_EmptyArray(t *testing.T) {
	manifest := PluginManifest{
		Name:    "test-plugin",
		Command: binaries["fake-plugin-empty-array"],
	}

	result, err := querySubprocessFlag(manifest, "--test-flag")
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("Expected empty array, got: %v", result)
	}
}

func TestQuerySubprocessFlag_WithArgs(t *testing.T) {
	manifest := PluginManifest{
		Name:    "test-plugin",
		Command: binaries["fake-plugin-with-args"],
		Args:    []string{"--arg1"},
	}

	result, err := querySubprocessFlag(manifest, "--supported-languages")
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
	if len(result) != 1 || result[0] != "test" {
		t.Errorf("Expected [\"test\"], got: %v", result)
	}
}
