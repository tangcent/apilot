// Package engine orchestrates the collect → format → output pipeline.
package engine

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	collector "github.com/tangcent/apilot/api-collector"
	formatter "github.com/tangcent/apilot/api-formatter"
	"github.com/tangcent/apilot/api-master/config"
	"github.com/tangcent/apilot/api-master/plugin"
)

type Config struct {
	SourceDir      string
	SourceFile     string
	MethodFilter   string
	ProjectRoot    string
	CollectorName  string
	FormatterName  string
	FormatParams   string
	OutputPath     string
	PluginRegistry string
}

func Run(cfg Config) error {
	sourceDir := cfg.SourceDir
	var sourceFile string

	info, err := os.Stat(sourceDir)
	if err != nil {
		return fmt.Errorf("source path %q: %w", sourceDir, err)
	}

	if !info.IsDir() {
		sourceFile, err = filepath.Abs(sourceDir)
		if err != nil {
			return fmt.Errorf("resolving source file path: %w", err)
		}
		if cfg.ProjectRoot != "" {
			sourceDir = cfg.ProjectRoot
		} else {
			projectRoot, findErr := findProjectRoot(filepath.Dir(sourceFile))
			if findErr != nil {
				return fmt.Errorf("could not find project root for %q: %w", sourceDir, findErr)
			}
			sourceDir = projectRoot
		}
	} else if cfg.ProjectRoot != "" {
		sourceDir = cfg.ProjectRoot
	}

	c, err := resolveCollectorWithDir(cfg, sourceDir)
	if err != nil {
		return fmt.Errorf("collector: %w", err)
	}

	ctx := collector.CollectContext{
		SourceDir:  sourceDir,
		SourceFile: sourceFile,
	}
	endpoints, err := c.Collect(ctx)
	if err != nil {
		return fmt.Errorf("collection failed: %w", err)
	}

	if sourceFile != "" {
		endpoints = filterEndpointsByFile(endpoints, sourceFile)
	}

	if cfg.MethodFilter != "" {
		endpoints = filterEndpointsByMethod(endpoints, cfg.MethodFilter)
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

	projectName := resolveProjectName(sourceDir)

	opts := formatter.FormatOptions{
		Settings:    settings,
		Collections: config.NewCollectionStore(),
	}
	if cfg.FormatParams != "" {
		opts.Params = []byte(cfg.FormatParams)
	}
	if projectName != "" || cfg.OutputPath != "" {
		var existing map[string]any
		if len(opts.Params) > 0 {
			_ = json.Unmarshal(opts.Params, &existing)
		}
		if existing == nil {
			existing = map[string]any{}
		}
		if projectName != "" {
			existing["projectName"] = projectName
		}
		if cfg.OutputPath != "" {
			existing["outputPath"] = cfg.OutputPath
		}
		merged, _ := json.Marshal(existing)
		opts.Params = merged
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

func resolveCollectorWithDir(cfg Config, sourceDir string) (collector.Collector, error) {
	name := cfg.CollectorName
	if name == "" {
		detected, err := detectCollector(sourceDir)
		if err != nil {
			return nil, fmt.Errorf("auto-detect failed: %w", err)
		}
		name = detected
	}
	return LookupCollector(name)
}

var projectRootIndicators = []string{
	"pom.xml",
	"build.gradle",
	"build.gradle.kts",
	"go.mod",
	"package.json",
	"requirements.txt",
	"pyproject.toml",
}

func findProjectRoot(dir string) (string, error) {
	for {
		for _, indicator := range projectRootIndicators {
			if _, err := os.Stat(filepath.Join(dir, indicator)); err == nil {
				return dir, nil
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", fmt.Errorf("no project root indicator found")
}

func filterEndpointsByFile(endpoints []collector.ApiEndpoint, sourceFile string) []collector.ApiEndpoint {
	absFile, err := filepath.Abs(sourceFile)
	if err != nil {
		absFile = sourceFile
	}
	baseName := filepath.Base(absFile)
	className := strings.TrimSuffix(baseName, filepath.Ext(baseName))

	var filtered []collector.ApiEndpoint
	for _, ep := range endpoints {
		if ep.Folder == className {
			filtered = append(filtered, ep)
		}
	}
	return filtered
}

func filterEndpointsByMethod(endpoints []collector.ApiEndpoint, method string) []collector.ApiEndpoint {
	var filtered []collector.ApiEndpoint
	for _, ep := range endpoints {
		if ep.Name == method {
			filtered = append(filtered, ep)
		}
	}
	return filtered
}

type collectorIndicator struct {
	file      string
	collector string
}

var collectorIndicators = []collectorIndicator{
	{"pom.xml", "java"},
	{"build.gradle", "java"},
	{"build.gradle.kts", "java"},
	{"go.mod", "go"},
	{"package.json", "node"},
	{"requirements.txt", "python"},
	{"pyproject.toml", "python"},
}

func detectCollector(sourceDir string) (string, error) {
	for _, ci := range collectorIndicators {
		if _, err := os.Stat(filepath.Join(sourceDir, ci.file)); err == nil {
			if _, ok := collectors[ci.collector]; ok {
				return ci.collector, nil
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
	case "collections":
		handleCollections(args[1:])
	case "export":
		handleExport(args[1:])
	case "--help", "-h", "help":
		printExportHelp()
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
	if isSensitiveKey(key) {
		fmt.Println(maskValue(value))
	} else {
		fmt.Println(value)
	}
}

func isSensitiveKey(key string) bool {
	lower := strings.ToLower(key)
	return strings.Contains(lower, "key") ||
		strings.Contains(lower, "token") ||
		strings.Contains(lower, "secret") ||
		strings.Contains(lower, "password") ||
		strings.Contains(lower, "apikey")
}

func handleCollections(args []string) {
	if len(args) == 0 {
		listCollections()
		return
	}
	subcmd := args[0]
	switch subcmd {
	case "list", "ls":
		listCollections()
	case "remove", "rm":
		if len(args) < 2 {
			fmt.Fprintf(os.Stderr, "Usage: apilot collections remove <project-name>\n")
			os.Exit(1)
		}
		if err := config.RemoveCollectionBinding(args[1]); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Removed collection binding for %q\n", args[1])
	default:
		fmt.Fprintf(os.Stderr, "Unknown collections subcommand: %s\n", subcmd)
		fmt.Fprintf(os.Stderr, "Usage: apilot collections [list|remove]\n")
		os.Exit(1)
	}
}

func listCollections() {
	bindings, err := config.ListCollectionBindings()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	if len(bindings) == 0 {
		fmt.Println("No collection bindings configured.")
		fmt.Println("Bindings are automatically saved when you export with postman.export.mode=UPDATE_EXISTING")
		return
	}
	fmt.Println("Project collection bindings:")
	for project, binding := range bindings {
		fmt.Printf("  %s: workspace=%s collection=%s\n", project, binding.WorkspaceID, binding.CollectionUID)
	}
}

func handleExport(args []string) {
	var (
		collectorName  string
		formatterName  string
		formatVariant  string
		formatParams   string
		outputPath     string
		pluginRegistry string
		methodFilter   string
		projectRoot    string
		listCollectors bool
		listFormatters bool
		showHelp       bool
	)

	fs := flag.NewFlagSet("export", flag.ContinueOnError)
	fs.StringVar(&collectorName, "collector", "", "collector name (auto-detect if omitted)")
	fs.StringVar(&formatterName, "formatter", "markdown", "formatter name (default: markdown)")
	fs.StringVar(&formatVariant, "format", "", "format variant, e.g. simple, detailed (default: simple)")
	fs.StringVar(&formatParams, "params", "", "formatter params as JSON (e.g. '{\"variant\":\"detailed\"}')")
	fs.StringVar(&outputPath, "output", "", "output file path (default: stdout)")
	fs.StringVar(&pluginRegistry, "plugin-registry", "", "path to plugins.json")
	fs.StringVar(&methodFilter, "method", "", "filter to a specific method name (used with file-level export)")
	fs.StringVar(&projectRoot, "project-root", "", "override auto-detected project root directory")
	fs.BoolVar(&listCollectors, "list-collectors", false, "print registered collectors and exit")
	fs.BoolVar(&listFormatters, "list-formatters", false, "print registered formatters and exit")
	fs.BoolVar(&showHelp, "help", false, "print help and exit")

	fs.Usage = func() {
		printExportHelp()
	}

	reordered := reorderArgs(args)
	if err := fs.Parse(reordered); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		printExportHelp()
		os.Exit(1)
	}

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

	if formatVariant != "" && formatParams == "" {
		formatParams = fmt.Sprintf(`{"variant":"%s"}`, formatVariant)
	}

	cfg := Config{
		SourceDir:      sourceDir,
		CollectorName:  collectorName,
		FormatterName:  formatterName,
		FormatParams:   formatParams,
		OutputPath:     outputPath,
		PluginRegistry: pluginRegistry,
		MethodFilter:   methodFilter,
		ProjectRoot:    projectRoot,
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
	fmt.Println("Usage: apilot <command> [arguments]")
	fmt.Println("")
	fmt.Println("Commands:")
	fmt.Println("  export <source-path> [flags]  Export API endpoints from source code")
	fmt.Println("  settings                      List settings required by formatters")
	fmt.Println("  set <key> <value>             Set a configuration value")
	fmt.Println("  get <key>                     Get a configuration value")
	fmt.Println("  collections [list|remove]     Manage project-to-collection bindings")
	fmt.Println("")
	fmt.Println("Run 'apilot export --help' for export flags.")
	fmt.Println("")
	fmt.Println("Shorthand: 'apilot <source-path> [flags]' is equivalent to 'apilot export <source-path> [flags]'.")
}

func printExportHelp() {
	fmt.Println("Usage: apilot <source-path> [flags]")
	fmt.Println("")
	fmt.Println("  source-path can be a directory or a single source file.")
	fmt.Println("  When a file is given, the project root is auto-detected by walking up")
	fmt.Println("  to find pom.xml, build.gradle, go.mod, package.json, etc.")
	fmt.Println("  For multi-module projects, the nearest directory with an indicator is used.")
	fmt.Println("")
	fmt.Println("Flags:")
	fmt.Println("  --collector string")
	fmt.Println("        collector name (auto-detect if omitted)")
	fmt.Println("  --formatter string")
	fmt.Println("        formatter name (default: markdown)")
	fmt.Println("  --format string")
	fmt.Println("        format variant, e.g. simple, detailed (default: simple)")
	fmt.Println("  --method string")
	fmt.Println("        filter to a specific method name (used with file-level export)")
	fmt.Println("  --project-root string")
	fmt.Println("        override auto-detected project root directory")
	fmt.Println("  --params string")
	fmt.Println("        formatter params as JSON (e.g. '{\"variant\":\"detailed\"}')")
	fmt.Println("  --output string")
	fmt.Println("        output file path (default: stdout)")
	fmt.Println("  --plugin-registry string")
	fmt.Println("        path to plugins.json")
	fmt.Println("  --list-collectors")
	fmt.Println("        print registered collectors and exit")
	fmt.Println("  --list-formatters")
	fmt.Println("        print registered formatters and exit")
	fmt.Println("  --help")
	fmt.Println("        print help and exit")
	fmt.Println("")

	printCollectors()
	fmt.Println("")

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

func reorderArgs(args []string) []string {
	var flags []string
	var positional []string
	for i := 0; i < len(args); i++ {
		if strings.HasPrefix(args[i], "-") {
			flags = append(flags, args[i])
			if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
				if !strings.Contains(args[i], "=") {
					flags = append(flags, args[i+1])
					i++
				}
			}
		} else {
			positional = append(positional, args[i])
		}
	}
	return append(flags, positional...)
}

func resolveProjectName(sourceDir string) string {
	abs, err := filepath.Abs(sourceDir)
	if err != nil {
		abs = sourceDir
	}
	return filepath.Base(abs)
}
