// Package config provides default configuration paths and environment variable overrides.
package config

import (
	"os"
	"path/filepath"
)

const (
	DefaultPluginRegistryFilename = "plugins.json"
	DefaultSettingsFilename       = "settings.json"
	DefaultConfigDir              = ".config/apilot"
)

// ConfigDir returns the base configuration directory path.
//
// The path is determined in the following order of precedence:
//  1. APILOT_CONFIG_DIR environment variable (if set)
//  2. ~/.config/apilot (default)
//
// Returns an empty string if the home directory cannot be determined
// and APILOT_CONFIG_DIR is not set.
func ConfigDir() string {
	dir := os.Getenv("APILOT_CONFIG_DIR")
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return ""
		}
		dir = filepath.Join(home, DefaultConfigDir)
	}
	return dir
}

// DefaultPluginRegistryPath returns the default path to plugins.json.
func DefaultPluginRegistryPath() string {
	dir := ConfigDir()
	if dir == "" {
		return ""
	}
	return filepath.Join(dir, DefaultPluginRegistryFilename)
}
