// Package kotlincollector parses Kotlin source trees for Spring MVC endpoints.
package kotlincollector

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	collector "github.com/tangcent/apilot/api-collector"
	javacollector "github.com/tangcent/apilot/api-collector-java"
	javaparser "github.com/tangcent/apilot/api-collector-java/parser"
	javaspringmvc "github.com/tangcent/apilot/api-collector-java/springmvc"
	ktparser "github.com/tangcent/apilot/api-collector-kotlin/parser"
	"github.com/tangcent/apilot/api-collector-kotlin/resolver"
	"github.com/tangcent/apilot/api-collector-kotlin/springmvc"
)

// KotlinCollector parses Kotlin source trees for API endpoints.
type KotlinCollector struct {
	mu            sync.Mutex
	unresolved    map[string]int
	warnings      []string
	frameworks    []string
	depResolution collector.DependencyResolution
	// shared, when set, is the dependency resolution of a Collect call that
	// spans another collector as well; its owner closes it.
	shared *javacollector.Dependencies
}

// New returns a new KotlinCollector.
func New() collector.Collector { return &KotlinCollector{} }

func (c *KotlinCollector) Name() string { return "kotlin" }

// SelectsFiles reports that Collect limits its result to the files a context
// selects: it extracts and resolves endpoints only for the declarations of
// those files, so the engine takes the result as it is.
func (c *KotlinCollector) SelectsFiles() bool { return true }

// ShareDependencies makes the next Collect call resolve dependency types
// through deps instead of setting up its own. The caller keeps ownership of
// deps and closes it.
func (c *KotlinCollector) ShareDependencies(deps *javacollector.Dependencies) {
	c.shared = deps
}

// SupportedLanguages reports kotlin: this collector is the collection path for
// .kt sources.
func (c *KotlinCollector) SupportedLanguages() []string { return []string{"kotlin"} }

// Unresolved reports type names the resolver could not expand during the last
// Collect call, mapped to occurrence counts.
func (c *KotlinCollector) Unresolved() map[string]int {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make(map[string]int, len(c.unresolved))
	for name, n := range c.unresolved {
		out[name] = n
	}
	return out
}

// Warnings reports non-fatal diagnostics from the last Collect call.
func (c *KotlinCollector) Warnings() []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]string(nil), c.warnings...)
}

// Frameworks reports the frameworks the last Collect call found endpoints for.
func (c *KotlinCollector) Frameworks() []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]string(nil), c.frameworks...)
}

// DependencyResolution reports the dependency-based type resolution of the
// last Collect call, shared by the Kotlin and Java halves of a mixed tree.
func (c *KotlinCollector) DependencyResolution() collector.DependencyResolution {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.depResolution
}

// Collect walks the Kotlin source and extracts Spring MVC endpoints.
//
// A tree that also holds Java sources is collected as one project: the Java
// endpoints come from the Java collector, Kotlin declarations are visible to
// Java endpoints and Java declarations to Kotlin endpoints, and the
// diagnostics of both halves are reported together.
//
// When ctx selects files, the whole tree is still parsed, because the selected
// files refer to types declared anywhere in it, but endpoints are extracted and
// resolved only for the controllers the selected files declare, Kotlin and
// Java alike. The result is what a directory export holds for those files.
func (c *KotlinCollector) Collect(ctx collector.CollectContext) ([]collector.ApiEndpoint, error) {
	requiredFields, err := ctx.RequiredFields()
	if err != nil {
		return nil, err
	}
	p, err := ktparser.NewParser()
	if err != nil {
		return nil, fmt.Errorf("failed to create kotlin parser: %w", err)
	}
	defer p.Close()

	mixed := hasJavaSources(ctx.SourceDir)

	// A file-level export parses the whole project all the same, because the
	// selected files refer to types declared anywhere in it, but extracts and
	// resolves endpoints only for the declarations of the selected files.
	selection := ctx.FileSelection()
	var kotlinSelected, javaSelected []string
	for _, file := range ctx.SelectedFiles() {
		switch ext := strings.ToLower(filepath.Ext(file)); {
		case ext == ".kt":
			kotlinSelected = append(kotlinSelected, file)
		case ext == ".java" && mixed:
			// A Java file of a mixed tree: the Java collector reads the file;
			// the Kotlin sources are only needed as types.
			javaSelected = append(javaSelected, file)
		default:
			return nil, fmt.Errorf("%s is not a Kotlin source file", file)
		}
	}

	var warnings []string
	results, err := p.ParseDirectory(ctx.SourceDir)
	if err != nil {
		if selection == nil {
			return nil, fmt.Errorf("failed to parse directory %s: %w", ctx.SourceDir, err)
		}
		// The project cannot be read as a whole: the selected files stand alone.
		warnings = append(warnings, fmt.Sprintf(
			"could not read the project under %s (%v): types declared outside the selected files are not available",
			ctx.SourceDir, err))
		results = nil
	}

	// A selected file the walk did not reach, such as one in a build directory
	// or outside the project, is parsed on its own and added.
	paths := make([]string, 0, len(results))
	for _, result := range results {
		paths = append(paths, result.FilePath)
	}
	for _, file := range collector.NewFileSelection(kotlinSelected).Unmatched(paths) {
		fileResult, parseErr := p.ParseFile(file)
		if parseErr != nil {
			return nil, fmt.Errorf("failed to parse file %s: %w", file, parseErr)
		}
		results = append(results, *fileResult)
	}

	for _, result := range results {
		if result.Error == nil {
			continue
		}
		// A file that declares a controller cannot be read without dropping
		// endpoints, so it stops a run that exports it. A file outside the
		// selection exports nothing: the run goes on, the file is named, and a
		// type declared in it shows up as unresolved where an endpoint uses it.
		if selection.Contains(result.FilePath) {
			return nil, result.Error
		}
		warnings = append(warnings, result.Error.Error()+
			"; the file is not selected, so it is skipped and the types declared in it are not available")
	}

	classes := flattenClasses(results)
	warnings = append(warnings, collectWarnings(results, classes, selection)...)

	// One dependency resolution serves both halves of a mixed tree.
	deps := c.shared
	if deps == nil {
		deps = javacollector.SetupDependencies(ctx.SourceDir, ctx.NoDeps)
		defer deps.Close()
	}

	var endpoints []collector.ApiEndpoint
	var matched []string
	var unresolved map[string]int

	// The Java half runs first: its declarations join the Kotlin type table.
	var javaEndpoints []collector.ApiEndpoint
	var javaTypes []javaparser.Class
	var java *javacollector.JavaCollector
	switch {
	case mixed && (selection == nil || len(javaSelected) > 0):
		java = javacollector.New().(*javacollector.JavaCollector)
		java.ShareDependencies(deps)
		javaCtx := ctx
		javaCtx.SourceFile, javaCtx.SourceFiles = "", nil
		if selection != nil {
			javaCtx.SourceFiles = javaSelected
		}
		javaEndpoints, javaTypes, err = java.CollectWithTypes(javaCtx, resolver.ToJavaClasses(classes))
		if err != nil {
			return nil, fmt.Errorf("collecting the Java sources of %s: %w", ctx.SourceDir, err)
		}
	case mixed:
		// Only Kotlin files are selected: the Java sources are types, nothing
		// of them is extracted or resolved.
		javaTypes, err = javacollector.ParseTypes(ctx.SourceDir)
		if err != nil {
			return nil, fmt.Errorf("reading the Java sources of %s: %w", ctx.SourceDir, err)
		}
	}

	if springMVCDisabled(ctx.Frameworks) || (selection != nil && len(kotlinSelected) == 0) {
		endpoints = nil
		unresolved = map[string]int{}
	} else {
		typeResolver := resolver.NewWithJavaTypes(classes, javaTypes)
		if dr := deps.Resolver(); dr != nil {
			typeResolver.SetDependencyResolver(dr)
		}
		typeResolver.SetRequiredFields(requiredFields)
		sm := springmvc.NewParser(typeResolver)
		sm.SetSelection(selection)
		before := len(endpoints)
		for _, ctrl := range sm.ExtractControllers(results) {
			for _, ep := range ctrl.Endpoints {
				api := springMVCEndpointToAPI(ep, ctrl.Name)
				api.FolderDescription = ctrl.Description
				endpoints = append(endpoints, api)
			}
		}
		if len(endpoints) > before {
			matched = append(matched, "spring-mvc")
		}
		warnings = append(warnings, sm.Warnings()...)
		unresolved = typeResolver.Unresolved()
	}

	if java != nil {
		endpoints = append(endpoints, javaEndpoints...)
		for name, n := range java.Unresolved() {
			unresolved[name] += n
		}
		warnings = append(warnings, java.Warnings()...)
		for _, framework := range java.Frameworks() {
			if !containsString(matched, framework) {
				matched = append(matched, framework)
			}
		}
	}

	c.mu.Lock()
	c.unresolved = unresolved
	c.warnings = warnings
	c.frameworks = matched
	c.depResolution = deps.Resolution()
	c.mu.Unlock()

	seen := make(map[string]bool, len(endpoints))
	deduped := make([]collector.ApiEndpoint, 0, len(endpoints))
	for _, ep := range endpoints {
		key := ep.Folder + "|" + ep.Path + "|" + ep.Method + "|" + ep.Name
		if !seen[key] {
			seen[key] = true
			deduped = append(deduped, ep)
		}
	}
	return deduped, nil
}

// hasJavaSources reports whether a tree holds a .java file outside build and
// tool directories.
func hasJavaSources(root string) bool {
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
		if strings.EqualFold(filepath.Ext(path), ".java") {
			found = true
			return filepath.SkipAll
		}
		return nil
	})
	return found
}

func containsString(values []string, value string) bool {
	for _, v := range values {
		if v == value {
			return true
		}
	}
	return false
}

func flattenClasses(results []ktparser.ParseResult) []ktparser.Class {
	var classes []ktparser.Class
	for _, result := range results {
		if result.Error != nil {
			continue
		}
		classes = append(classes, result.Classes...)
	}
	return classes
}

// collectWarnings gathers the diagnostics of the parse. What the parser says of
// a file it had to read in part concerns the export only when the file is one
// the export reads endpoints from, so a file outside the selection of a
// file-level export stays quiet. A sealed class is reported wherever it is
// declared: whether an endpoint reaches it is known only to the resolver.
func collectWarnings(results []ktparser.ParseResult, classes []ktparser.Class, selection *collector.FileSelection) []string {
	var warnings []string
	for _, result := range results {
		if selection.Contains(result.FilePath) {
			warnings = append(warnings, result.Warnings...)
		}
	}
	for _, class := range classes {
		if class.IsSealed {
			warnings = append(warnings, class.FilePath+": sealed class "+class.Name+
				" is exported as an unexpanded reference; its subtypes are not flattened")
		}
	}
	return warnings
}

func springMVCDisabled(frameworks []string) bool {
	if len(frameworks) == 0 {
		return false
	}
	for _, framework := range frameworks {
		switch framework {
		case "spring", "spring-mvc", "springmvc":
			return false
		}
	}
	return true
}

func springMVCEndpointToAPI(ep springmvc.Endpoint, folder string) collector.ApiEndpoint {
	out := collector.ApiEndpoint{
		Name:        ep.MethodName,
		Folder:      folder,
		Description: ep.Description,
		Tags:        []string{folder},
		Path:        ep.Path,
		Method:      string(ep.Method),
		Protocol:    "http",
		Consumes:    ep.Consumes,
		Produces:    ep.Produces,
	}
	out.Deprecated, out.DeprecationNote = ep.Deprecated, ep.DeprecationNote
	for _, p := range ep.Parameters {
		switch {
		case p.ParamType == "body":
			out.RequestBody = &collector.ApiBody{Name: p.Name, MediaType: "application/json"}
		case isMultipartFile(p.HeldType()):
			out.Parameters = append(out.Parameters, collector.ApiParameter{
				Name:        p.Name,
				Type:        "file",
				TypeName:    p.Type,
				In:          "form",
				Required:    p.Required,
				Default:     p.DefaultValue,
				Description: p.Description,
				Example:     p.Example,
				Enum:        p.Enum,
			})
		case p.ParamType == "header":
			out.Headers = append(out.Headers, collector.ApiHeader{
				Name:        p.Name,
				Value:       p.DefaultValue,
				Description: p.Description,
				Example:     p.Example,
				Required:    p.Required,
			})
		default:
			out.Parameters = append(out.Parameters, collector.ApiParameter{
				Name:        p.Name,
				Type:        "text",
				TypeName:    p.Type,
				In:          p.ParamType,
				Required:    p.Required,
				Default:     p.DefaultValue,
				Description: p.Description,
				Example:     p.Example,
				Enum:        p.Enum,
			})
		}
	}
	if ep.RequestBodySchema != nil && out.RequestBody != nil {
		out.RequestBody.Body = ep.RequestBodySchema
	}
	if ep.ResponseSchema != nil {
		out.Response = &collector.ApiBody{
			MediaType: "application/json",
			Body:      ep.ResponseSchema,
		}
	}
	return out
}

// isMultipartFile reports whether a parameter of this type is uploaded as
// file parts: a MultipartFile, or a collection or array of them.
func isMultipartFile(kotlinType string) bool {
	element := javaspringmvc.ElementType(resolver.NormalizeModelType(kotlinType))
	return element == "MultipartFile" || strings.HasSuffix(element, ".MultipartFile")
}
