// Package config provides default configuration paths and environment variable overrides.
package config

import (
	"os"
	"path/filepath"
)

const (
	DefaultPluginRegistryFilename = "plugins.json"
	DefaultConfigDir              = ".config/apilot"
)

// DefaultPluginRegistryPath returns the default path to plugins.json,
// respecting the API_MASTER_CONFIG_DIR environment variable if set.
func DefaultPluginRegistryPath() string {
	dir := os.Getenv("API_MASTER_CONFIG_DIR")
	if dir == "" {
		home, _ := os.UserHomeDir()
		dir = filepath.Join(home, DefaultConfigDir)
	}
	return filepath.Join(dir, DefaultPluginRegistryFilename)
}
