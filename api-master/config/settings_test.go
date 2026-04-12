package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSettingsFilePath_Default(t *testing.T) {
	os.Unsetenv("APILOT_CONFIG_DIR")
	path := SettingsFilePath()
	if path == "" {
		t.Fatal("SettingsFilePath() returned empty string")
	}
	if filepath.Base(path) != "settings.json" {
		t.Errorf("Expected base name 'settings.json', got %q", filepath.Base(path))
	}
}

func TestSettingsFilePath_EnvOverride(t *testing.T) {
	customDir := "/tmp/apilot-test-config"
	os.Setenv("APILOT_CONFIG_DIR", customDir)
	defer os.Unsetenv("APILOT_CONFIG_DIR")

	path := SettingsFilePath()
	expected := filepath.Join(customDir, "settings.json")
	if path != expected {
		t.Errorf("Expected %q, got %q", expected, path)
	}
}

func TestLoadSettings_NoFile(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("APILOT_CONFIG_DIR", tmpDir)
	defer os.Unsetenv("APILOT_CONFIG_DIR")

	settings, err := LoadSettings()
	if err != nil {
		t.Fatalf("LoadSettings() error = %v", err)
	}
	if len(settings) != 0 {
		t.Errorf("Expected empty settings, got %v", settings)
	}
}

func TestSaveAndLoadSettings(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("APILOT_CONFIG_DIR", tmpDir)
	defer os.Unsetenv("APILOT_CONFIG_DIR")

	original := map[string]string{
		"postman.api.key": "PMAK-test-key",
		"other.setting":   "value",
	}

	if err := SaveSettings(original); err != nil {
		t.Fatalf("SaveSettings() error = %v", err)
	}

	loaded, err := LoadSettings()
	if err != nil {
		t.Fatalf("LoadSettings() error = %v", err)
	}

	if len(loaded) != len(original) {
		t.Errorf("Expected %d settings, got %d", len(original), len(loaded))
	}
	for k, v := range original {
		if loaded[k] != v {
			t.Errorf("settings[%q] = %q, want %q", k, loaded[k], v)
		}
	}
}

func TestSaveSettings_CreatesDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	nestedDir := filepath.Join(tmpDir, "nested", "dir")
	os.Setenv("APILOT_CONFIG_DIR", nestedDir)
	defer os.Unsetenv("APILOT_CONFIG_DIR")

	settings := map[string]string{"key": "value"}
	if err := SaveSettings(settings); err != nil {
		t.Fatalf("SaveSettings() error = %v", err)
	}

	if _, err := os.Stat(filepath.Join(nestedDir, "settings.json")); os.IsNotExist(err) {
		t.Error("settings.json was not created in nested directory")
	}
}

func TestLoadSettings_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("APILOT_CONFIG_DIR", tmpDir)
	defer os.Unsetenv("APILOT_CONFIG_DIR")

	path := filepath.Join(tmpDir, "settings.json")
	if err := os.WriteFile(path, []byte("not valid json"), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	settings, err := LoadSettings()
	if err != nil {
		t.Fatalf("LoadSettings() with invalid JSON should not error, got: %v", err)
	}
	if len(settings) != 0 {
		t.Errorf("Expected empty settings for invalid JSON, got %v", settings)
	}
}

func TestGetSetting(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("APILOT_CONFIG_DIR", tmpDir)
	defer os.Unsetenv("APILOT_CONFIG_DIR")

	settings := map[string]string{"test.key": "test-value"}
	if err := SaveSettings(settings); err != nil {
		t.Fatalf("SaveSettings() error = %v", err)
	}

	value, err := GetSetting("test.key")
	if err != nil {
		t.Fatalf("GetSetting() error = %v", err)
	}
	if value != "test-value" {
		t.Errorf("GetSetting() = %q, want %q", value, "test-value")
	}
}

func TestGetSetting_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("APILOT_CONFIG_DIR", tmpDir)
	defer os.Unsetenv("APILOT_CONFIG_DIR")

	value, err := GetSetting("nonexistent")
	if err != nil {
		t.Fatalf("GetSetting() error = %v", err)
	}
	if value != "" {
		t.Errorf("GetSetting() for nonexistent key should return empty string, got %q", value)
	}
}

func TestSetSetting(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("APILOT_CONFIG_DIR", tmpDir)
	defer os.Unsetenv("APILOT_CONFIG_DIR")

	if err := SetSetting("new.key", "new-value"); err != nil {
		t.Fatalf("SetSetting() error = %v", err)
	}

	value, err := GetSetting("new.key")
	if err != nil {
		t.Fatalf("GetSetting() error = %v", err)
	}
	if value != "new-value" {
		t.Errorf("GetSetting() = %q, want %q", value, "new-value")
	}
}

func TestSetSetting_Overwrite(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("APILOT_CONFIG_DIR", tmpDir)
	defer os.Unsetenv("APILOT_CONFIG_DIR")

	if err := SetSetting("key", "value1"); err != nil {
		t.Fatalf("SetSetting() error = %v", err)
	}
	if err := SetSetting("key", "value2"); err != nil {
		t.Fatalf("SetSetting() error = %v", err)
	}

	value, err := GetSetting("key")
	if err != nil {
		t.Fatalf("GetSetting() error = %v", err)
	}
	if value != "value2" {
		t.Errorf("GetSetting() = %q, want %q", value, "value2")
	}
}

func TestSaveSettings_AtomicWrite(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("APILOT_CONFIG_DIR", tmpDir)
	defer os.Unsetenv("APILOT_CONFIG_DIR")

	settings := map[string]string{"key": "value"}
	if err := SaveSettings(settings); err != nil {
		t.Fatalf("SaveSettings() error = %v", err)
	}

	data, err := os.ReadFile(filepath.Join(tmpDir, "settings.json"))
	if err != nil {
		t.Fatalf("Failed to read settings file: %v", err)
	}

	var parsed map[string]string
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Settings file is not valid JSON: %v", err)
	}

	if parsed["key"] != "value" {
		t.Errorf("Expected key=value, got %v", parsed)
	}

	if _, err := os.Stat(filepath.Join(tmpDir, "settings.json.tmp")); !os.IsNotExist(err) {
		t.Error("Temp file should not exist after save")
	}
}

func TestLazySettings_Get(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("APILOT_CONFIG_DIR", tmpDir)
	defer os.Unsetenv("APILOT_CONFIG_DIR")

	settings := map[string]string{"lazy.key": "lazy-value"}
	if err := SaveSettings(settings); err != nil {
		t.Fatalf("SaveSettings() error = %v", err)
	}

	ls := NewLazySettings()
	value := ls.Get("lazy.key")
	if value != "lazy-value" {
		t.Errorf("LazySettings.Get() = %q, want %q", value, "lazy-value")
	}
}

func TestLazySettings_Get_MissingKey(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("APILOT_CONFIG_DIR", tmpDir)
	defer os.Unsetenv("APILOT_CONFIG_DIR")

	ls := NewLazySettings()
	value := ls.Get("nonexistent")
	if value != "" {
		t.Errorf("LazySettings.Get() for missing key should return empty string, got %q", value)
	}
}

func TestLazySettings_LazyLoad(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("APILOT_CONFIG_DIR", tmpDir)
	defer os.Unsetenv("APILOT_CONFIG_DIR")

	ls := NewLazySettings()

	if err := SaveSettings(map[string]string{"key": "value"}); err != nil {
		t.Fatalf("SaveSettings() error = %v", err)
	}

	value := ls.Get("key")
	if value != "value" {
		t.Errorf("LazySettings.Get() = %q, want %q", value, "value")
	}
}

func TestMapSettings_Get(t *testing.T) {
	ms := NewMapSettings(map[string]string{"map.key": "map-value"})

	value := ms.Get("map.key")
	if value != "map-value" {
		t.Errorf("MapSettings.Get() = %q, want %q", value, "map-value")
	}
}

func TestMapSettings_Get_MissingKey(t *testing.T) {
	ms := NewMapSettings(map[string]string{})

	value := ms.Get("nonexistent")
	if value != "" {
		t.Errorf("MapSettings.Get() for missing key should return empty string, got %q", value)
	}
}

func TestMapSettings_NilMap(t *testing.T) {
	ms := NewMapSettings(nil)

	value := ms.Get("any.key")
	if value != "" {
		t.Errorf("MapSettings.Get() with nil map should return empty string, got %q", value)
	}
}

func TestSettingsFilePath_EnvOverrideEmpty(t *testing.T) {
	os.Setenv("APILOT_CONFIG_DIR", "")
	defer os.Unsetenv("APILOT_CONFIG_DIR")

	path := SettingsFilePath()
	if path == "" {
		t.Fatal("SettingsFilePath() should not return empty when env is empty string (should use default)")
	}
}

func TestLoadSettings_EmptyPath(t *testing.T) {
	os.Unsetenv("APILOT_CONFIG_DIR")

	path := SettingsFilePath()
	if path == "" {
		settings, err := LoadSettings()
		if err != nil {
			t.Errorf("LoadSettings() with empty path should not error, got: %v", err)
		}
		if len(settings) != 0 {
			t.Errorf("LoadSettings() with empty path should return empty map, got: %v", settings)
		}
	} else {
		t.Skip("Home directory available, cannot test empty path case")
	}
}

func TestSaveSettings_EmptyPath(t *testing.T) {
	os.Unsetenv("APILOT_CONFIG_DIR")

	path := SettingsFilePath()
	if path == "" {
		err := SaveSettings(map[string]string{"key": "value"})
		if err == nil {
			t.Error("SaveSettings() with empty path should return error")
		}
	} else {
		t.Skip("Home directory available, cannot test empty path case")
	}
}

func TestConfigDir_Default(t *testing.T) {
	os.Unsetenv("APILOT_CONFIG_DIR")

	dir := ConfigDir()
	if dir == "" {
		t.Fatal("ConfigDir() returned empty string")
	}
	if !strings.HasSuffix(dir, ".config/apilot") {
		t.Errorf("Expected dir to end with '.config/apilot', got %q", dir)
	}
}

func TestConfigDir_EnvOverride(t *testing.T) {
	customDir := "/tmp/custom-config"
	os.Setenv("APILOT_CONFIG_DIR", customDir)
	defer os.Unsetenv("APILOT_CONFIG_DIR")

	dir := ConfigDir()
	if dir != customDir {
		t.Errorf("Expected %q, got %q", customDir, dir)
	}
}

func TestDefaultPluginRegistryPath(t *testing.T) {
	os.Unsetenv("APILOT_CONFIG_DIR")

	path := DefaultPluginRegistryPath()
	if path == "" {
		t.Fatal("DefaultPluginRegistryPath() returned empty string")
	}
	if filepath.Base(path) != "plugins.json" {
		t.Errorf("Expected base name 'plugins.json', got %q", filepath.Base(path))
	}
}

func TestDefaultPluginRegistryPath_EnvOverride(t *testing.T) {
	customDir := "/tmp/custom-apilot"
	os.Setenv("APILOT_CONFIG_DIR", customDir)
	defer os.Unsetenv("APILOT_CONFIG_DIR")

	path := DefaultPluginRegistryPath()
	expected := filepath.Join(customDir, "plugins.json")
	if path != expected {
		t.Errorf("Expected %q, got %q", expected, path)
	}
}
