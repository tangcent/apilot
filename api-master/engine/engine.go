// Package engine orchestrates the collect → format → output pipeline.
package engine

import (
	"fmt"
	"os"

	"github.com/tangcent/apilot/api-collector/collector"
	"github.com/tangcent/apilot/api-formater/formater"
)

// Config holds the runtime configuration for a single engine run.
type Config struct {
	SourceDir      string
	CollectorName  string // empty = auto-detect
	FormatterName  string // default: "markdown"
	FormatVariant  string // passed to FormatOptions.Format
	OutputPath     string // empty = stdout
	PluginRegistry string // path to plugins.json
}

// Run executes the full collect → format → output pipeline.
func Run(cfg Config) error {
	// 1. Resolve collector
	c, err := resolveCollector(cfg)
	if err != nil {
		return fmt.Errorf("collector: %w", err)
	}

	// 2. Collect endpoints
	ctx := collector.CollectContext{SourceDir: cfg.SourceDir}
	endpoints, err := c.Collect(ctx)
	if err != nil {
		return fmt.Errorf("collection failed: %w", err)
	}

	// 3. Resolve formatter
	formatterName := cfg.FormatterName
	if formatterName == "" {
		formatterName = "markdown"
	}
	f, err := LookupFormatter(formatterName)
	if err != nil {
		return fmt.Errorf("formatter: %w", err)
	}

	// 4. Format
	opts := formater.FormatOptions{Format: cfg.FormatVariant}
	output, err := f.Format(endpoints, opts)
	if err != nil {
		return fmt.Errorf("formatting failed: %w", err)
	}

	// 5. Write output
	return writeOutput(cfg.OutputPath, output)
}

func resolveCollector(cfg Config) (collector.Collector, error) {
	name := cfg.CollectorName
	if name == "" {
		detected, err := detectCollector(cfg.SourceDir)
		if err != nil {
			return nil, fmt.Errorf("auto-detect failed: %w", err)
		}
		name = detected
	}
	return LookupCollector(name)
}

// detectCollector inspects the source directory for well-known indicator files
// and returns the name of the most appropriate registered collector.
func detectCollector(sourceDir string) (string, error) {
	indicators := []struct {
		file      string
		collector string
	}{
		{"pom.xml", "java"},
		{"build.gradle", "java"},
		{"build.gradle.kts", "java"},
		{"go.mod", "go"},
		{"package.json", "node"},
		{"requirements.txt", "python"},
		{"pyproject.toml", "python"},
	}
	for _, ind := range indicators {
		if _, err := os.Stat(sourceDir + "/" + ind.file); err == nil {
			if _, ok := collectors[ind.collector]; ok {
				return ind.collector, nil
			}
		}
	}
	return "", fmt.Errorf("could not auto-detect collector for %q", sourceDir)
}

func writeOutput(path string, data []byte) error {
	if path == "" {
		_, err := os.Stdout.Write(data)
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// RunCLI parses os.Args and calls Run. Used by apilot-cli and api-master main.
func RunCLI() {
	// TODO: implement flag parsing (--collector, --formatter, --output, --format,
	//       --plugin-registry, --list-collectors, --list-formatters)
	panic("RunCLI: not yet implemented")
}
