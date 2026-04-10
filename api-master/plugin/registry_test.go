package plugin

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/tangcent/apilot/api-collector"
	"github.com/tangcent/apilot/api-formatter"
)

type mockCollector struct {
	name              string
	supportedLangs    []string
	collectCalled     bool
	collectContext    collector.CollectContext
	collectEndpoints  []collector.ApiEndpoint
	collectError      error
}

func (m *mockCollector) Name() string {
	return m.name
}

func (m *mockCollector) SupportedLanguages() []string {
	return m.supportedLangs
}

func (m *mockCollector) Collect(ctx collector.CollectContext) ([]collector.ApiEndpoint, error) {
	m.collectCalled = true
	m.collectContext = ctx
	return m.collectEndpoints, m.collectError
}

type mockFormatter struct {
	name             string
	supportedFormats []string
	formatCalled     bool
	formatEndpoints  []collector.ApiEndpoint
	formatOpts       formatter.FormatOptions
	formatOutput     []byte
	formatError      error
}

func (m *mockFormatter) Name() string {
	return m.name
}

func (m *mockFormatter) SupportedFormats() []string {
	return m.supportedFormats
}

func (m *mockFormatter) Format(endpoints []collector.ApiEndpoint, opts formatter.FormatOptions) ([]byte, error) {
	m.formatCalled = true
	m.formatEndpoints = endpoints
	m.formatOpts = opts
	return m.formatOutput, m.formatError
}

func TestLoadRegistry_NonExistentFile(t *testing.T) {
	registeredCollectors := make(map[string]collector.Collector)
	registeredFormatters := make(map[string]formatter.Formatter)

	registerCollector := func(c collector.Collector) {
		registeredCollectors[c.Name()] = c
	}
	registerFormatter := func(f formatter.Formatter) {
		registeredFormatters[f.Name()] = f
	}

	err := LoadRegistry("/nonexistent/plugins.json", registerCollector, registerFormatter)
	if err != nil {
		t.Errorf("Expected no error for non-existent file, got: %v", err)
	}

	if len(registeredCollectors) != 0 {
		t.Errorf("Expected 0 collectors, got: %d", len(registeredCollectors))
	}

	if len(registeredFormatters) != 0 {
		t.Errorf("Expected 0 formatters, got: %d", len(registeredFormatters))
	}
}

func TestLoadRegistry_ValidFile(t *testing.T) {
	tmpDir := t.TempDir()
	pluginFile := filepath.Join(tmpDir, "plugins.json")

	content := `{
  "plugins": [
    {
      "name": "test-collector",
      "type": "collector",
      "command": "echo"
    },
    {
      "name": "test-formatter",
      "type": "formatter",
      "command": "echo"
    }
  ]
}`

	err := os.WriteFile(pluginFile, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to write plugin file: %v", err)
	}

	registeredCollectors := make(map[string]collector.Collector)
	registeredFormatters := make(map[string]formatter.Formatter)

	registerCollector := func(c collector.Collector) {
		registeredCollectors[c.Name()] = c
	}
	registerFormatter := func(f formatter.Formatter) {
		registeredFormatters[f.Name()] = f
	}

	err = LoadRegistry(pluginFile, registerCollector, registerFormatter)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if _, ok := registeredCollectors["test-collector"]; !ok {
		t.Error("Expected test-collector to be registered")
	}

	if _, ok := registeredFormatters["test-formatter"]; !ok {
		t.Error("Expected test-formatter to be registered")
	}
}

func TestLoadRegistry_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	pluginFile := filepath.Join(tmpDir, "plugins.json")

	content := `invalid json`

	err := os.WriteFile(pluginFile, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to write plugin file: %v", err)
	}

	registerCollector := func(c collector.Collector) {}
	registerFormatter := func(f formatter.Formatter) {}

	err = LoadRegistry(pluginFile, registerCollector, registerFormatter)
	if err == nil {
		t.Error("Expected error for invalid JSON, got nil")
	}
}

func TestLoadRegistry_UnknownPluginType(t *testing.T) {
	tmpDir := t.TempDir()
	pluginFile := filepath.Join(tmpDir, "plugins.json")

	content := `{
  "plugins": [
    {
      "name": "test-plugin",
      "type": "unknown"
    }
  ]
}`

	err := os.WriteFile(pluginFile, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to write plugin file: %v", err)
	}

	registerCollector := func(c collector.Collector) {}
	registerFormatter := func(f formatter.Formatter) {}

	err = LoadRegistry(pluginFile, registerCollector, registerFormatter)
	if err != nil {
		t.Errorf("Expected no error (unknown plugins should be skipped), got: %v", err)
	}
}

func TestLoadRegistry_MissingCommand(t *testing.T) {
	tmpDir := t.TempDir()
	pluginFile := filepath.Join(tmpDir, "plugins.json")

	content := `{
  "plugins": [
    {
      "name": "test-collector",
      "type": "collector"
    }
  ]
}`

	err := os.WriteFile(pluginFile, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to write plugin file: %v", err)
	}

	registerCollector := func(c collector.Collector) {}
	registerFormatter := func(f formatter.Formatter) {}

	err = LoadRegistry(pluginFile, registerCollector, registerFormatter)
	if err != nil {
		t.Errorf("Expected no error (invalid plugins should be skipped), got: %v", err)
	}
}

func TestLoadRegistry_EmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	pluginFile := filepath.Join(tmpDir, "plugins.json")

	content := `{}`

	err := os.WriteFile(pluginFile, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to write plugin file: %v", err)
	}

	registerCollector := func(c collector.Collector) {}
	registerFormatter := func(f formatter.Formatter) {}

	err = LoadRegistry(pluginFile, registerCollector, registerFormatter)
	if err != nil {
		t.Errorf("Expected no error for empty plugins, got: %v", err)
	}
}

func TestLoadRegistry_MultiplePlugins(t *testing.T) {
	tmpDir := t.TempDir()
	pluginFile := filepath.Join(tmpDir, "plugins.json")

	content := `{
  "plugins": [
    {
      "name": "collector1",
      "type": "collector",
      "command": "echo"
    },
    {
      "name": "collector2",
      "type": "collector",
      "command": "echo"
    },
    {
      "name": "formatter1",
      "type": "formatter",
      "command": "echo"
    },
    {
      "name": "formatter2",
      "type": "formatter",
      "command": "echo"
    }
  ]
}`

	err := os.WriteFile(pluginFile, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to write plugin file: %v", err)
	}

	registeredCollectors := make(map[string]collector.Collector)
	registeredFormatters := make(map[string]formatter.Formatter)

	registerCollector := func(c collector.Collector) {
		registeredCollectors[c.Name()] = c
	}
	registerFormatter := func(f formatter.Formatter) {
		registeredFormatters[f.Name()] = f
	}

	err = LoadRegistry(pluginFile, registerCollector, registerFormatter)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if len(registeredCollectors) != 2 {
		t.Errorf("Expected 2 collectors, got: %d", len(registeredCollectors))
	}

	if len(registeredFormatters) != 2 {
		t.Errorf("Expected 2 formatters, got: %d", len(registeredFormatters))
	}
}

func TestLoadRegistry_NonExistentBinary(t *testing.T) {
	tmpDir := t.TempDir()
	pluginFile := filepath.Join(tmpDir, "plugins.json")

	content := `{
  "plugins": [
    {
      "name": "test-collector",
      "type": "collector",
      "command": "/nonexistent/binary"
    },
    {
      "name": "test-formatter",
      "type": "formatter",
      "command": "/nonexistent/binary"
    }
  ]
}`

	err := os.WriteFile(pluginFile, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to write plugin file: %v", err)
	}

	registeredCollectors := make(map[string]collector.Collector)
	registeredFormatters := make(map[string]formatter.Formatter)

	registerCollector := func(c collector.Collector) {
		registeredCollectors[c.Name()] = c
	}
	registerFormatter := func(f formatter.Formatter) {
		registeredFormatters[f.Name()] = f
	}

	err = LoadRegistry(pluginFile, registerCollector, registerFormatter)
	if err != nil {
		t.Errorf("Expected no error (non-existent binaries should be skipped), got: %v", err)
	}

	if len(registeredCollectors) != 0 {
		t.Errorf("Expected 0 collectors (non-existent binary should be skipped), got: %d", len(registeredCollectors))
	}

	if len(registeredFormatters) != 0 {
		t.Errorf("Expected 0 formatters (non-existent binary should be skipped), got: %d", len(registeredFormatters))
	}
}

func TestLoadRegistry_MixedValidInvalid(t *testing.T) {
	tmpDir := t.TempDir()
	pluginFile := filepath.Join(tmpDir, "plugins.json")

	content := `{
  "plugins": [
    {
      "name": "valid-collector",
      "type": "collector",
      "command": "echo"
    },
    {
      "name": "invalid-collector",
      "type": "collector",
      "command": "/nonexistent/binary"
    },
    {
      "name": "valid-formatter",
      "type": "formatter",
      "command": "echo"
    },
    {
      "name": "unknown-type",
      "type": "unknown"
    }
  ]
}`

	err := os.WriteFile(pluginFile, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to write plugin file: %v", err)
	}

	registeredCollectors := make(map[string]collector.Collector)
	registeredFormatters := make(map[string]formatter.Formatter)

	registerCollector := func(c collector.Collector) {
		registeredCollectors[c.Name()] = c
	}
	registerFormatter := func(f formatter.Formatter) {
		registeredFormatters[f.Name()] = f
	}

	err = LoadRegistry(pluginFile, registerCollector, registerFormatter)
	if err != nil {
		t.Errorf("Expected no error (invalid plugins should be skipped), got: %v", err)
	}

	if len(registeredCollectors) != 1 {
		t.Errorf("Expected 1 collector (invalid should be skipped), got: %d", len(registeredCollectors))
	}

	if _, ok := registeredCollectors["valid-collector"]; !ok {
		t.Error("Expected valid-collector to be registered")
	}

	if len(registeredFormatters) != 1 {
		t.Errorf("Expected 1 formatter (invalid should be skipped), got: %d", len(registeredFormatters))
	}

	if _, ok := registeredFormatters["valid-formatter"]; !ok {
		t.Error("Expected valid-formatter to be registered")
	}
}
