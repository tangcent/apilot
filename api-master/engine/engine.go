// Package engine orchestrates the collect → format → output pipeline.
package engine

import (
	"flag"
	"fmt"
	"os"
	"strings"

	collector "github.com/tangcent/apilot/api-collector"
	formatter "github.com/tangcent/apilot/api-formatter"
	"github.com/tangcent/apilot/api-master/config"
	"github.com/tangcent/apilot/api-master/plugin"
)

type Config struct {
	SourceDir      string
	CollectorName  string
	FormatterName  string
	FormatParams   string
	OutputPath     string
	PluginRegistry string
}

func Run(cfg Config) error {
	c, err := resolveCollector(cfg)
	if err != nil {
		return fmt.Errorf("collector: %w", err)
	}

	ctx := collector.CollectContext{SourceDir: cfg.SourceDir}
	endpoints, err := c.Collect(ctx)
	if err != nil {
		return fmt.Errorf("collection failed: %w", err)
	}

	formatterName := cfg.FormatterName
	if formatterName == "" {
		formatterName = "markdown"
	}
	f, err := LookupFormatter(formatterName)
	if err != nil {
		return fmt.Errorf("formatter: %w", err)
	}

	settings := config.NewLazySettings()

	if checkErr := checkRequiredSettings(f, settings); checkErr != nil {
		return checkErr
	}

	opts := formatter.FormatOptions{
		Settings: settings,
	}
	if cfg.FormatParams != "" {
		opts.Params = []byte(cfg.FormatParams)
	}
	output, err := f.Format(endpoints, opts)
	if err != nil {
		return fmt.Errorf("formatting failed: %w", err)
	}

	return writeOutput(cfg.OutputPath, output)
}

func checkRequiredSettings(f formatter.Formatter, settings formatter.Settings) error {
	sp, ok := f.(formatter.SettingsProvider)
	if !ok {
		return nil
	}
	for _, def := range sp.RequiredSettings() {
		if def.Required {
			if settings.Get(def.Key) == "" {
				return fmt.Errorf("setting %q is required for formatter %q but not configured. Set it with: apilot set %s <value>", def.Key, f.Name(), def.Key)
			}
		}
	}
	return nil
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

func RunCLI() {
	args := os.Args[1:]

	if len(args) == 0 {
		printHelp()
		os.Exit(1)
	}

	subcommand := args[0]
	switch subcommand {
	case "settings":
		handleSettings()
	case "set":
		handleSet(args[1:])
	case "get":
		handleGet(args[1:])
	case "export":
		handleExport(args[1:])
	case "--help", "-h", "help":
		printHelp()
	default:
		if strings.HasPrefix(subcommand, "-") {
			handleExport(args)
		} else {
			handleExport(args)
		}
	}
}

func handleSettings() {
	loadPlugins()

	settingDefs := ListFormatterSettings()
	if len(settingDefs) == 0 {
		fmt.Println("No settings required by registered formatters.")
		return
	}

	settings, err := config.LoadSettings()
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: could not load settings: %v\n", err)
		settings = map[string]string{}
	}
	settingsReader := config.NewMapSettings(settings)

	fmt.Println("Settings:")
	for _, def := range settingDefs {
		value := settingsReader.Get(def.Key)
		if value != "" {
			masked := maskValue(value)
			fmt.Printf("  %-30s %s (current: %s)\n", def.Key, def.Description, masked)
		} else {
			required := ""
			if def.Required {
				required = " [required]"
			}
			fmt.Printf("  %-30s %s%s\n", def.Key, def.Description, required)
		}
	}
}

func handleSet(args []string) {
	if len(args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: apilot set <key> <value>\n")
		os.Exit(1)
	}
	key := args[0]
	value := args[1]
	if err := config.SetSetting(key, value); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Set %s\n", key)
}

func handleGet(args []string) {
	if len(args) < 1 {
		fmt.Fprintf(os.Stderr, "Usage: apilot get <key>\n")
		os.Exit(1)
	}
	key := args[0]
	value, err := config.GetSetting(key)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	if value == "" {
		fmt.Fprintf(os.Stderr, "Setting %q not found\n", key)
		os.Exit(1)
	}
	fmt.Println(value)
}

func handleExport(args []string) {
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

	fs := flag.NewFlagSet("export", flag.ContinueOnError)
	fs.StringVar(&collectorName, "collector", "", "collector name (auto-detect if omitted)")
	fs.StringVar(&formatterName, "formatter", "markdown", "formatter name (default: markdown)")
	fs.StringVar(&formatParams, "params", "", "formatter params as JSON (e.g. '{\"variant\":\"detailed\"}')")
	fs.StringVar(&outputPath, "output", "", "output file path (default: stdout)")
	fs.StringVar(&pluginRegistry, "plugin-registry", "", "path to plugins.json")
	fs.BoolVar(&listCollectors, "list-collectors", false, "print registered collectors and exit")
	fs.BoolVar(&listFormatters, "list-formatters", false, "print registered formatters and exit")
	fs.BoolVar(&showHelp, "help", false, "print help and exit")

	fs.Usage = func() {
		printExportHelp()
	}

	fs.Parse(args)

	if pluginRegistry == "" {
		pluginRegistry = config.DefaultPluginRegistryPath()
	}

	if err := plugin.LoadRegistry(pluginRegistry, RegisterCollector, RegisterFormatter); err != nil {
		fmt.Fprintf(os.Stderr, "error loading plugin registry: %v\n", err)
		os.Exit(1)
	}

	if showHelp {
		printExportHelp()
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

	remaining := fs.Args()
	if len(remaining) == 0 {
		fmt.Fprintf(os.Stderr, "error: source path required\n\n")
		printExportHelp()
		os.Exit(1)
	}

	sourceDir := remaining[0]

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

func loadPlugins() {
	pluginRegistry := config.DefaultPluginRegistryPath()
	if err := plugin.LoadRegistry(pluginRegistry, RegisterCollector, RegisterFormatter); err != nil {
		fmt.Fprintf(os.Stderr, "warning: could not load plugin registry: %v\n", err)
	}
}

func maskValue(value string) string {
	if len(value) <= 8 {
		return "****"
	}
	return value[:4] + "****" + value[len(value)-4:]
}

func printHelp() {
	fmt.Fprintln(os.Stderr, "Usage: apilot <command> [arguments]")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Commands:")
	fmt.Fprintln(os.Stderr, "  export <source-path> [flags]  Export API endpoints from source code")
	fmt.Fprintln(os.Stderr, "  settings                      List settings required by formatters")
	fmt.Fprintln(os.Stderr, "  set <key> <value>             Set a configuration value")
	fmt.Fprintln(os.Stderr, "  get <key>                     Get a configuration value")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Run 'apilot export --help' for export flags.")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Shorthand: 'apilot <source-path> [flags]' is equivalent to 'apilot export <source-path> [flags]'.")
}

func printExportHelp() {
	fmt.Fprintln(os.Stderr, "Usage: apilot export <source-path> [flags]")
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
