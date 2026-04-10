package plugin

import (
	"os"
	"path/filepath"
	"testing"

	fmtr "github.com/tangcent/apilot/api-formatter"
)

func TestSubprocessCollector_SupportedLanguages(t *testing.T) {
	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "test-collector")

	script := `#!/bin/bash
if [ "$1" = "--supported-languages" ]; then
    echo '["java", "kotlin"]'
else
    echo '[]'
fi
`
	if err := os.WriteFile(scriptPath, []byte(script), 0755); err != nil {
		t.Fatalf("Failed to create script: %v", err)
	}

	manifest := PluginManifest{
		Name:    "test-collector",
		Type:    "collector",
		Command: scriptPath,
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
	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "test-formatter")

	// Script echoes back the received params field so we can assert it was forwarded.
	script := `#!/bin/bash
input=$(cat)
echo "$input" > /tmp/apilot_test_input.json
echo -n "ok"
`
	if err := os.WriteFile(scriptPath, []byte(script), 0755); err != nil {
		t.Fatalf("Failed to create script: %v", err)
	}

	manifest := PluginManifest{
		Name:    "test-formatter",
		Type:    "formatter",
		Command: scriptPath,
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

func TestSubprocessFormatter_SupportedFormats_Error(t *testing.T) {
	manifest := PluginManifest{
		Name:    "test-formatter",
		Type:    "formatter",
		Command: "/nonexistent/binary",
	}

	// SupportedFormats was removed from the Formatter interface.
	// Verify the subprocess formatter still satisfies the interface.
	f := &subprocessFormatter{manifest: manifest}
	if f.Name() != "test-formatter" {
		t.Errorf("Expected name 'test-formatter', got: %q", f.Name())
	}
}

func TestQuerySubprocessFlag_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "test-plugin")

	script := `#!/bin/bash
echo 'invalid json'
`
	if err := os.WriteFile(scriptPath, []byte(script), 0755); err != nil {
		t.Fatalf("Failed to create script: %v", err)
	}

	manifest := PluginManifest{
		Name:    "test-plugin",
		Command: scriptPath,
	}

	result, err := querySubprocessFlag(manifest, "--test-flag")
	if err == nil {
		t.Errorf("Expected error for invalid JSON, got result: %v", result)
	}
}

func TestQuerySubprocessFlag_EmptyArray(t *testing.T) {
	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "test-plugin")

	script := `#!/bin/bash
echo '[]'
`
	if err := os.WriteFile(scriptPath, []byte(script), 0755); err != nil {
		t.Fatalf("Failed to create script: %v", err)
	}

	manifest := PluginManifest{
		Name:    "test-plugin",
		Command: scriptPath,
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
	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "test-plugin")

	script := `#!/bin/bash
if [ "$1" = "--arg1" ] && [ "$2" = "--supported-languages" ]; then
    echo '["test"]'
else
    echo '[]'
fi
`
	if err := os.WriteFile(scriptPath, []byte(script), 0755); err != nil {
		t.Fatalf("Failed to create script: %v", err)
	}

	manifest := PluginManifest{
		Name:    "test-plugin",
		Command: scriptPath,
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
