package plugin

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/tangcent/apilot/api-collector"
	"github.com/tangcent/apilot/api-formatter"
)

// LoadRegistry reads the plugin registry file at path and registers all valid entries.
// Invalid or missing plugin paths are logged as warnings and skipped.
func LoadRegistry(path string, registerCollector func(collector.Collector), registerFormatter func(formatter.Formatter)) error {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // no registry file is fine
		}
		return fmt.Errorf("reading plugin registry: %w", err)
	}

	var reg PluginRegistry
	if err := json.Unmarshal(data, &reg); err != nil {
		return fmt.Errorf("parsing plugin registry: %w", err)
	}

	for _, m := range reg.Plugins {
		if err := registerManifest(m, registerCollector, registerFormatter); err != nil {
			log.Printf("warning: skipping plugin %q: %v", m.Name, err)
		}
	}
	return nil
}

func registerManifest(m PluginManifest, registerCollector func(collector.Collector), registerFormatter func(formatter.Formatter)) error {
	switch m.Type {
	case "collector":
		c, err := newSubprocessCollector(m)
		if err != nil {
			return err
		}
		registerCollector(c)
	case "formatter":
		f, err := newSubprocessFormatter(m)
		if err != nil {
			return err
		}
		registerFormatter(f)
	default:
		return fmt.Errorf("unknown plugin type %q", m.Type)
	}
	return nil
}
