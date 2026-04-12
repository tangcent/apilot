package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	formatter "github.com/tangcent/apilot/api-formatter"
)

// SettingsFilePath returns the absolute path to the settings file.
// Returns an empty string if the config directory cannot be determined.
func SettingsFilePath() string {
	dir := ConfigDir()
	if dir == "" {
		return ""
	}
	return filepath.Join(dir, DefaultSettingsFilename)
}

func LoadSettings() (map[string]string, error) {
	path := SettingsFilePath()
	if path == "" {
		return map[string]string{}, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]string{}, nil
		}
		return nil, err
	}
	var settings map[string]string
	if err := json.Unmarshal(data, &settings); err != nil {
		return map[string]string{}, nil
	}
	if settings == nil {
		settings = map[string]string{}
	}
	return settings, nil
}

func SaveSettings(settings map[string]string) error {
	path := SettingsFilePath()
	if path == "" {
		return fmt.Errorf("cannot determine settings path: home directory unavailable and APILOT_CONFIG_DIR not set")
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func GetSetting(key string) (string, error) {
	settings, err := LoadSettings()
	if err != nil {
		return "", err
	}
	return settings[key], nil
}

func SetSetting(key, value string) error {
	settings, err := LoadSettings()
	if err != nil {
		return err
	}
	settings[key] = value
	return SaveSettings(settings)
}

// LazySettings implements formatter.Settings with on-demand loading.
// The settings file is read only when Get() is first called.
type LazySettings struct {
	once     sync.Once
	settings map[string]string
}

func NewLazySettings() *LazySettings {
	return &LazySettings{}
}

func (s *LazySettings) load() {
	s.settings, _ = LoadSettings()
	if s.settings == nil {
		s.settings = map[string]string{}
	}
}

func (s *LazySettings) Get(key string) string {
	s.once.Do(s.load)
	return s.settings[key]
}

// MapSettings implements formatter.Settings backed by a simple map.
// Useful for testing and for CLI commands that already have settings loaded.
type MapSettings struct {
	Values map[string]string
}

func NewMapSettings(values map[string]string) *MapSettings {
	if values == nil {
		values = map[string]string{}
	}
	return &MapSettings{Values: values}
}

func (s *MapSettings) Get(key string) string {
	return s.Values[key]
}

var _ formatter.Settings = (*LazySettings)(nil)
var _ formatter.Settings = (*MapSettings)(nil)
