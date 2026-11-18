// Package engine orchestrates the collect → format → output pipeline.
package engine

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	collector "github.com/tangcent/apilot/api-collector"
	formatter "github.com/tangcent/apilot/api-formatter"
	"github.com/tangcent/apilot/api-master/config"
	"github.com/tangcent/apilot/api-master/plugin"
)

type Config struct {
	// SourceDir is the path to export: a directory, or a single source file.
	SourceDir  string
	SourceFile string
	// SourcePaths, when set, is the path list to export in place of SourceDir:
	// one directory, or one or more source files of the same project.
	SourcePaths    []string
	MethodFilter   string
	ProjectRoot    string
	CollectorName  string
	FormatterName  string
	FormatParams   string
	OutputPath     string
	PluginRegistry string
	NoDeps         bool
	// AllowEmpty turns a zero-endpoint result into a success. Without it such a
	// run is an error, because a document with no endpoints is far more often a
	// collector gap than an intentional export.
	AllowEmpty bool
	// RequiredFields selects how collectors mark fields required: empty or
	// collector.RequiredFieldsDefault, or collector.RequiredFieldsConstraints.
	RequiredFields string
}

// sourceTarget is what one run exports.
type sourceTarget struct {
	// dir is the project directory the collector reads.
	dir string
	// files holds the absolute paths of the selected source files; empty for an
	// export of the whole of dir.
	files []string
	// requested is the path list as given by the caller. Error messages report
	// it: for a file-level export dir is rewritten to the project root, which is
	// not what the user asked to export.
	requested string
}

// resolveSources decides what a run exports from the paths it was given: one
// directory, or one or more source files. Files are exported as part of the
// project they belong to, found by walking up from each of them unless
// projectRoot names it, and all of them must belong to the same one.
func resolveSources(paths []string, projectRoot string) (sourceTarget, error) {
	target := sourceTarget{requested: strings.Join(paths, ", ")}

	isDir := make([]bool, len(paths))
	for i, path := range paths {
		info, err := os.Stat(path)
		if err != nil {
			return target, fmt.Errorf("source path %q: %w", path, err)
		}
		isDir[i] = info.IsDir()
	}

	if len(paths) == 1 && isDir[0] {
		target.dir = paths[0]
		if projectRoot != "" {
			target.dir = projectRoot
		}
		return target, nil
	}
	for i, path := range paths {
		if isDir[i] {
			return target, fmt.Errorf("source path %q is a directory: export either one directory or one or more source files, not both", path)
		}
	}

	var firstPath string
	for _, path := range paths {
		file, err := filepath.Abs(path)
		if err != nil {
			return target, fmt.Errorf("resolving source file path: %w", err)
		}
		if slices.Contains(target.files, file) {
			continue
		}
		root := projectRoot
		if root == "" {
			found, findErr := findProjectRoot(filepath.Dir(file))
			if findErr != nil {
				return target, fmt.Errorf("could not find project root for %q: %w", path, findErr)
			}
			root = found
		}
		if len(target.files) == 0 {
			target.dir, firstPath = root, path
		} else if root != target.dir {
			return target, fmt.Errorf("source files belong to different projects: %q is in %q but %q is in %q; export them separately, or pass --project-root to export them together",
				firstPath, target.dir, path, root)
		}
		target.files = append(target.files, file)
	}
	return target, nil
}

func Run(cfg Config) error {
	paths := cfg.SourcePaths
	if len(paths) == 0 {
		paths = []string{cfg.SourceDir}
	}
	target, err := resolveSources(paths, cfg.ProjectRoot)
	if err != nil {
		return err
	}
	sourceDir, sourceFiles, requestedPath := target.dir, target.files, target.requested

	c, err := resolveCollectorWithDir(cfg, sourceDir)
	if err != nil {
		return fmt.Errorf("collector: %w", err)
	}

	ctx := collector.CollectContext{
		SourceDir:   sourceDir,
		SourceFiles: sourceFiles,
		NoDeps:      cfg.NoDeps,
	}
	// A collector that reads only SourceFile must never take one file of
	// several for the whole selection, so it is set for a single file only.
	if len(sourceFiles) == 1 {
		ctx.SourceFile = sourceFiles[0]
	}
	if cfg.RequiredFields != "" {
		ctx.Config = map[string]string{collector.ConfigRequiredFields: cfg.RequiredFields}
		// Reject an unknown mode before the tree is parsed, not after.
		if _, err := ctx.RequiredFields(); err != nil {
			return err
		}
	}
	endpoints, err := c.Collect(ctx)
	if err != nil {
		return fmt.Errorf("collection failed: %w", err)
	}

	// Dependency-based type resolution silently does nothing when its
	// prerequisites are missing; surface whether it ran so an export full of
	// opaque types can be triaged.
	if reporter, ok := c.(collector.DependencyResolutionReporter); ok {
		reportDependencyResolution(os.Stderr, reporter.DependencyResolution())
	}

	// A resolver that cannot expand a type returns an opaque single-value model
	// that looks exactly like a resolved primitive. Report those types so a
	// "missing fields" export can be triaged instead of guessed at.
	if reporter, ok := c.(collector.UnresolvedReporter); ok {
		reportUnresolved(os.Stderr, reporter.Unresolved())
	}

	// A collector that had to leave part of a document unexpanded says so
	// here; otherwise the gap is only visible to whoever reads the export.
	if reporter, ok := c.(collector.WarningsReporter); ok {
		reportWarnings(os.Stderr, reporter.Warnings())
	}

	collected := len(endpoints)

	// A collector that honours the selected files has already limited its result
	// to them, by the file each declaration came from. Any other collector
	// returns the endpoints of the whole project, and the file name is all that
	// is left to tell which of them belong to the selected files.
	if fs, ok := c.(collector.FileSelector); !ok || !fs.SelectsFiles() {
		if len(sourceFiles) > 0 {
			endpoints = filterEndpointsByFiles(endpoints, sourceFiles)
		}
	}

	if cfg.MethodFilter != "" {
		endpoints = filterEndpointsByMethod(endpoints, cfg.MethodFilter)
	}

	// Print how many endpoints are about to be exported, every run, including a
	// zero. A pipeline can then assert the count from logs instead of counting
	// headings in the document. stderr keeps stdout byte-stable for the export.
	fmt.Fprintf(os.Stderr, "endpoints: %d\n", len(endpoints))

	// A collector that locks onto a project but extracts nothing used to exit 0
	// with a document holding nothing but its heading, which made an unsupported
	// framework indistinguishable from a successful export (#161).
	if len(endpoints) == 0 && !cfg.AllowEmpty {
		return emptyCollectionError(c.Name(), requestedPath, collected)
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

	// What the export is: the project it came from, the collector that read it,
	// and the framework that collector matched. A document that does not name
	// these is unidentifiable once it leaves the directory it was made in
	// (#170).
	projectName := resolveProjectName(sourceDir)
	var frameworks []string
	if fr, ok := c.(collector.FrameworkReporter); ok {
		frameworks = fr.Frameworks()
	}

	opts := formatter.FormatOptions{
		Settings:    settings,
		Collections: config.NewCollectionStore(),
	}
	if cfg.FormatParams != "" {
		opts.Params = []byte(cfg.FormatParams)
	}
	if projectName != "" || cfg.OutputPath != "" || len(frameworks) > 0 || sourceDir != "" {
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
		// The collector is always known: it is the one that produced the
		// endpoints. It is injected unconditionally so the count that follows
		// it in a formatter's header has something to sit next to.
		existing["collector"] = c.Name()
		if len(frameworks) > 0 {
			existing["frameworks"] = frameworks
		}
		if cfg.OutputPath != "" {
			existing["outputPath"] = cfg.OutputPath
		}
		// Where the sources are, so a formatter can read project-level files
		// that travel with them, such as an .easy.api.config rules file.
		if abs, err := filepath.Abs(sourceDir); err == nil {
			existing["sourceDir"] = abs
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

// emptyCollectionError explains a zero-endpoint run. Naming the collector and
// the path is the whole point: "nothing was exported" only becomes actionable
// once the user knows who looked and where.
func emptyCollectionError(collectorName, path string, collected int) error {
	const hint = "pass --allow-empty to export an empty document"
	if collected == 0 {
		return fmt.Errorf("collector %q extracted 0 endpoints from %q (%s)", collectorName, path, hint)
	}
	return fmt.Errorf("collector %q extracted %d endpoints from %q, none of which matched the file and method filters (%s)",
		collectorName, collected, path, hint)
}

// reportUnresolved writes a summary of types the collector could not resolve.
// Output goes to a side channel (stderr) so exported documents stay byte-stable,
// and it never affects the exit code.
func reportUnresolved(w io.Writer, counts map[string]int) {
	if len(counts) == 0 {
		return
	}

	names := make([]string, 0, len(counts))
	for name := range counts {
		names = append(names, name)
	}
	sort.Slice(names, func(i, j int) bool {
		if counts[names[i]] != counts[names[j]] {
			return counts[names[i]] > counts[names[j]]
		}
		return names[i] < names[j]
	})

	fmt.Fprintf(w, "warning: %d types could not be resolved\n", len(counts))
	for _, name := range names {
		unit := "occurrences"
		if counts[name] == 1 {
			unit = "occurrence"
		}
		fmt.Fprintf(w, "  %-20s %d %s\n", name, counts[name], unit)
	}
}

// reportWarnings writes the diagnostics a collector raised while producing the
// export, one line each. Like the other reports it goes to a side channel
// (stderr) so exported documents stay byte-stable, and it never affects the
// exit code: a marked section is a gap to fill, not a failed run.
func reportWarnings(w io.Writer, warnings []string) {
	for _, warning := range warnings {
		fmt.Fprintf(w, "warning: %s\n", warning)
	}
}

// reportDependencyResolution prints one diagnostic line telling the user
// whether dependency-based type resolution ran, followed by warnings when
// resolved types may come from a version other than the project's or when
// lookups failed. Output goes to a side channel (stderr) so exported documents
// stay byte-stable, and it never affects the exit code. A no-deps outcome
// stays silent: nothing was skipped, so there is nothing to act on.
func reportDependencyResolution(w io.Writer, dr collector.DependencyResolution) {
	switch dr.State {
	case collector.DependencyResolutionActive:
		if dr.ResolvedTypes > 0 {
			unit := "types"
			if dr.ResolvedTypes == 1 {
				unit = "type"
			}
			fmt.Fprintf(w, "info: resolved %d %s from dependencies\n", dr.ResolvedTypes, unit)
		}
		// A type read from another version of its library can differ from
		// the one the project compiles against. Without a type looked up
		// there is nothing to warn about.
		if dr.Detail != "" && (dr.ResolvedTypes > 0 || len(dr.Failed) > 0) {
			fmt.Fprintf(w, "warning: dependency versions: %s\n", dr.Detail)
		}
		if n := len(dr.Failed); n > 0 {
			subject := "lookups"
			if n == 1 {
				subject = "lookup"
			}
			fmt.Fprintf(w, "warning: %d dependency type %s failed: %s\n", n, subject, strings.Join(dr.Failed, "; "))
		}
		if n := len(dr.Unpinned); n > 0 {
			subject := "types were"
			if n == 1 {
				subject = "type was"
			}
			fmt.Fprintf(w, "warning: %d dependency %s not pinned to the project's version and read from the indexed one: %s\n",
				n, subject, strings.Join(dr.Unpinned, ", "))
		}
	case collector.DependencyResolutionToolMissing:
		fmt.Fprintf(w, "warning: dependency type resolution skipped: %s\n", dr.Detail)
	case collector.DependencyResolutionDisabled:
		fmt.Fprintf(w, "info: dependency type resolution disabled (--no-deps)\n")
	}
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

// findProjectRoot returns the project root of a source file that sits in dir:
// the nearest directory at or above dir with a build file, lifted to the
// topmost aggregator when that directory is a module of a multi-module build
// (see topmostAggregator). The nearest directory alone would leave the sibling
// modules a controller depends on outside the project, to be looked up in the
// dependency index one type at a time.
func findProjectRoot(dir string) (string, error) {
	for {
		for _, indicator := range projectRootIndicators {
			if _, err := os.Stat(filepath.Join(dir, indicator)); err == nil {
				return topmostAggregator(dir), nil
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

// filterEndpointsByFiles keeps the endpoints whose folder is named after one of
// the files, the way a controller is named after the file that declares it.
func filterEndpointsByFiles(endpoints []collector.ApiEndpoint, sourceFiles []string) []collector.ApiEndpoint {
	classNames := make(map[string]bool, len(sourceFiles))
	for _, sourceFile := range sourceFiles {
		baseName := filepath.Base(sourceFile)
		classNames[strings.TrimSuffix(baseName, filepath.Ext(baseName))] = true
	}

	var filtered []collector.ApiEndpoint
	for _, ep := range endpoints {
		if classNames[ep.Folder] {
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
			name := ci.collector
			// A Gradle or Maven build file says "JVM project", not "Java
			// source": build.gradle.kts is also the build script of an ordinary
			// Java project. Decide by the sources: a tree with Kotlin in it goes
			// to the Kotlin collector, which hands the Java sources of a mixed
			// tree to the Java collector. The Java collector does not read .kt
			// files, so a mixed tree sent to it would lose every Kotlin endpoint.
			if name == "java" && hasKotlinSources(sourceDir) {
				name = "kotlin"
			}
			if _, ok := collectors[name]; ok {
				return name, nil
			}
		}
	}
	// No build indicator at all: a tree of Kotlin sources is still collectable.
	if hasKotlinSources(sourceDir) {
		if _, ok := collectors["kotlin"]; ok {
			return "kotlin", nil
		}
	}
	return "", fmt.Errorf("could not auto-detect collector for %q", sourceDir)
}

// hasKotlinSources reports whether a tree holds at least one .kt file.
// Generated and build directories are skipped so a stale artifact cannot
// decide the collector.
func hasKotlinSources(root string) bool {
	found := false
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || found {
			return nil
		}
		if d.IsDir() {
			switch d.Name() {
			case ".git", ".gradle", ".idea", ".kotlin", "build", "out", "target", "node_modules":
				return filepath.SkipDir
			}
			return nil
		}
		if strings.EqualFold(filepath.Ext(path), ".kt") {
			found = true
			return filepath.SkipAll
		}
		return nil
	})
	return found
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
		noDeps         bool
		allowEmpty     bool
		requiredFields string
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
	fs.BoolVar(&noDeps, "no-deps", false, "disable resolving request/response types from dependencies")
	fs.BoolVar(&allowEmpty, "allow-empty", false, "exit 0 when the collection yields no endpoints")
	fs.StringVar(&requiredFields, "required-fields", "", `how fields are marked required: "default" (unless @Nullable) or "constraints" (only with @NotNull, @NotBlank or @NotEmpty)`)
	fs.BoolVar(&listCollectors, "list-collectors", false, "print registered collectors and exit")
	fs.BoolVar(&listFormatters, "list-formatters", false, "print registered formatters and exit")
	fs.BoolVar(&showHelp, "help", false, "print help and exit")

	fs.Usage = func() {
		printExportHelp()
	}

	sourcePaths, err := parseInterleaved(fs, args)
	if err != nil {
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

	if len(sourcePaths) == 0 {
		fmt.Fprintf(os.Stderr, "error: source path required\n\n")
		printExportHelp()
		os.Exit(1)
	}

	if formatVariant != "" && formatParams == "" {
		formatParams = fmt.Sprintf(`{"variant":"%s"}`, formatVariant)
	}

	cfg := Config{
		SourceDir:      sourcePaths[0],
		SourcePaths:    sourcePaths,
		CollectorName:  collectorName,
		FormatterName:  formatterName,
		FormatParams:   formatParams,
		OutputPath:     outputPath,
		PluginRegistry: pluginRegistry,
		MethodFilter:   methodFilter,
		ProjectRoot:    projectRoot,
		NoDeps:         noDeps,
		AllowEmpty:     allowEmpty,
		RequiredFields: requiredFields,
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
	fmt.Println("  source-path can be a directory, or one or more source files.")
	fmt.Println("  When files are given, the project root is auto-detected by walking up")
	fmt.Println("  to find pom.xml, build.gradle, go.mod, package.json, etc.")
	fmt.Println("  For multi-module projects, the topmost aggregator is used: the directory")
	fmt.Println("  whose pom.xml lists the module, or that holds the settings.gradle(.kts)")
	fmt.Println("  of its Gradle build. A project that no build above lists keeps its own.")
	fmt.Println("  All files must belong to the same project; use --project-root otherwise.")
	fmt.Println("  Files of one project are exported in one run, sharing one parse of the")
	fmt.Println("  project and one dependency lookup session.")
	fmt.Println("")
	fmt.Println("Flags:")
	fmt.Println("  --collector string")
	fmt.Println("        collector name (auto-detect if omitted)")
	fmt.Println("  --formatter string")
	fmt.Println("        formatter name (default: markdown)")
	fmt.Println("  --format string")
	fmt.Println("        format variant, e.g. simple, detailed (default: simple)")
	fmt.Println("  --method string")
	fmt.Println("        filter to a specific method name (used with file-level export;")
	fmt.Println("        applies to every selected file)")
	fmt.Println("  --project-root string")
	fmt.Println("        override auto-detected project root directory")
	fmt.Println("  --no-deps")
	fmt.Println("        disable resolving request/response types from dependencies")
	fmt.Println("  --allow-empty")
	fmt.Println("        exit 0 when the collection yields no endpoints (default: fail)")
	fmt.Println("  --required-fields string")
	fmt.Println("        how fields are marked required: default (unless @Nullable) or")
	fmt.Println("        constraints (only with @NotNull, @NotBlank or @NotEmpty) (default: default)")
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

// parseInterleaved parses args with fs, accepting flags before, between and
// after the source paths, and returns the paths in the order given. The flag
// set decides which flags take a value, so a boolean flag such as --no-deps
// never swallows the path that follows it.
func parseInterleaved(fs *flag.FlagSet, args []string) ([]string, error) {
	var positional []string
	for {
		if err := fs.Parse(args); err != nil {
			return nil, err
		}
		rest := fs.Args()
		if len(rest) == 0 {
			return positional, nil
		}
		positional = append(positional, rest[0])
		args = rest[1:]
	}
}

func resolveProjectName(sourceDir string) string {
	abs, err := filepath.Abs(sourceDir)
	if err != nil {
		abs = sourceDir
	}
	return filepath.Base(abs)
}
