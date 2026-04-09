// Package plugin handles loading external collector/formatter plugins from a registry file.
package plugin

// PluginManifest describes a single external plugin entry in the registry.
type PluginManifest struct {
	// Name is the unique identifier used to look up this plugin (e.g. "java", "postman").
	Name string `json:"name"`

	// Type is either "collector" or "formatter".
	Type string `json:"type"`

	// Command is the executable command to invoke for subprocess-based plugins.
	Command string `json:"command,omitempty"`

	// Path is the shared library path for dynlib-based plugins.
	Path string `json:"path,omitempty"`

	// Args are additional arguments passed when invoking a subprocess plugin.
	Args []string `json:"args,omitempty"`
}

// PluginRegistry is the top-level structure of plugins.json.
type PluginRegistry struct {
	Plugins []PluginManifest `json:"plugins"`
}
