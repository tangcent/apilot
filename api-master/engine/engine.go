// Package engine orchestrates the collect → format → output pipeline.
package engine

import (
	"flag"
	"fmt"
	"os"

	"github.com/tangcent/apilot/api-collector"
	"github.com/tangcent/apilot/api-formatter"
	"github.com/tangcent/apilot/api-master/config"
	"github.com/tangcent/apilot/api-master/plugin"
)

// Config holds the runtime configuration for a single engine run.
type Config struct {
	SourceDir      string
	CollectorName  string // empty = auto-detect
	FormatterName  string // default: "markdown"
	FormatParams   string // raw JSON passed as FormatOptions.Params (e.g. `{"variant":"detailed"}`)
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
	opts := formatter.FormatOptions{}
	if cfg.FormatParams != "" {
		opts.Params = []byte(cfg.FormatParams)
	}
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
	var (
		collectorName  string
		formatterName  string
		formatParams   string
		outputPath     string
		pluginRegistry string
		listCollectors bool
		listFormatters bool
		showHelp       bool
	)

	flag.StringVar(&collectorName, "collector", "", "collector name (auto-detect if omitted)")
	flag.StringVar(&formatterName, "formatter", "markdown", "formatter name (default: markdown)")
	flag.StringVar(&formatParams, "params", "", "formatter params as JSON (e.g. '{\"variant\":\"detailed\"}')")
	flag.StringVar(&outputPath, "output", "", "output file path (default: stdout)")
	flag.StringVar(&pluginRegistry, "plugin-registry", "", "path to plugins.json")
	flag.BoolVar(&listCollectors, "list-collectors", false, "print registered collectors and exit")
	flag.BoolVar(&listFormatters, "list-formatters", false, "print registered formatters and exit")
	flag.BoolVar(&showHelp, "help", false, "print help and exit")

	flag.Usage = func() {
		printHelp()
	}

	flag.Parse()

	if pluginRegistry == "" {
		pluginRegistry = config.DefaultPluginRegistryPath()
	}

	if err := plugin.LoadRegistry(pluginRegistry, RegisterCollector, RegisterFormatter); err != nil {
		fmt.Fprintf(os.Stderr, "error loading plugin registry: %v\n", err)
		os.Exit(1)
	}

	if showHelp {
		printHelp()
		os.Exit(0)
	}

	if listCollectors {
		printCollectors()
		return
	}

	if listFormatters {
		printFormatters()
		return
	}

	args := flag.Args()
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "error: source path required\n\n")
		printHelp()
		os.Exit(1)
	}

	sourceDir := args[0]

	cfg := Config{
		SourceDir:      sourceDir,
		CollectorName:  collectorName,
		FormatterName:  formatterName,
		FormatParams:   formatParams,
		OutputPath:     outputPath,
		PluginRegistry: pluginRegistry,
	}

	if err := Run(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func printHelp() {
	fmt.Fprintln(os.Stderr, "Usage: apilot <source-path> [flags]")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Flags:")
	fmt.Fprintln(os.Stderr, "  --collector string")
	fmt.Fprintln(os.Stderr, "        collector name (auto-detect if omitted)")
	fmt.Fprintln(os.Stderr, "  --formatter string")
	fmt.Fprintln(os.Stderr, "        formatter name (default: markdown)")
	fmt.Fprintln(os.Stderr, "  --params string")
	fmt.Fprintln(os.Stderr, "        formatter params as JSON (e.g. '{\"variant\":\"detailed\"}')")
	fmt.Fprintln(os.Stderr, "  --output string")
	fmt.Fprintln(os.Stderr, "        output file path (default: stdout)")
	fmt.Fprintln(os.Stderr, "  --plugin-registry string")
	fmt.Fprintln(os.Stderr, "        path to plugins.json")
	fmt.Fprintln(os.Stderr, "  --list-collectors")
	fmt.Fprintln(os.Stderr, "        print registered collectors and exit")
	fmt.Fprintln(os.Stderr, "  --list-formatters")
	fmt.Fprintln(os.Stderr, "        print registered formatters and exit")
	fmt.Fprintln(os.Stderr, "  --help")
	fmt.Fprintln(os.Stderr, "        print help and exit")
	fmt.Fprintln(os.Stderr, "")

	printCollectors()
	fmt.Fprintln(os.Stderr, "")

	printFormatters()
}

func printCollectors() {
	collectors := ListCollectors()
	if len(collectors) == 0 {
		fmt.Println("No collectors registered.")
		return
	}
	fmt.Println("Registered collectors:")
	for name, langs := range collectors {
		fmt.Printf("  %s: %v\n", name, langs)
	}
}

func printFormatters() {
	formatters := ListFormatters()
	if len(formatters) == 0 {
		fmt.Println("No formatters registered.")
		return
	}
	fmt.Println("Registered formatters:")
	for _, name := range formatters {
		fmt.Printf("  %s\n", name)
	}
}
