// Package javacollector implements the Collector interface for Java projects.
// Supported frameworks: Spring MVC, JAX-RS, Feign.
package javacollector

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"sync"

	collector "github.com/tangcent/apilot/api-collector"
	"github.com/tangcent/apilot/api-collector-java/feign"
	"github.com/tangcent/apilot/api-collector-java/jaxrs"
	"github.com/tangcent/apilot/api-collector-java/maven"
	"github.com/tangcent/apilot/api-collector-java/parser"
	"github.com/tangcent/apilot/api-collector-java/springmvc"
)

// JavaCollector parses Java source trees for API endpoints.
type JavaCollector struct {
	mu            sync.Mutex
	unresolved    map[string]int
	warnings      []string
	frameworks    []string
	depResolution collector.DependencyResolution
	// shared, when set, is the dependency resolution of a Collect call that
	// spans another language as well; its owner closes it.
	shared *Dependencies
}

// mavenAvailable, mavenVersion and loadProjectContext are variables so tests
// can force each branch without installing or uninstalling maven-indexer-cli.
var (
	mavenAvailable     = maven.IsAvailable
	mavenVersion       = maven.CLIVersion
	loadProjectContext = maven.LoadProjectContext
)

// New returns a new JavaCollector.
func New() collector.Collector { return &JavaCollector{} }

func (c *JavaCollector) Name() string { return "java" }

// SupportedLanguages reports only java: Kotlin is not parsed. Claiming it
// made Kotlin-only projects look like successful collections with zero
// endpoints (issue #142).
func (c *JavaCollector) SupportedLanguages() []string { return []string{"java"} }

// Unresolved reports type names the Java type resolver could not expand during
// the last Collect call, mapped to occurrence counts.
func (c *JavaCollector) Unresolved() map[string]int {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make(map[string]int, len(c.unresolved))
	for name, n := range c.unresolved {
		out[name] = n
	}
	return out
}

// Warnings reports the diagnostics raised during the last Collect call: gaps
// the collector worked around so the export could still be produced. A JAX-RS
// response whose payload is only built in the method body is the first of them.
func (c *JavaCollector) Warnings() []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]string(nil), c.warnings...)
}

// DependencyResolution reports whether dependency-based type resolution ran
// during the last Collect call, and why not when it did not.
func (c *JavaCollector) DependencyResolution() collector.DependencyResolution {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.depResolution
}

// Frameworks reports the frameworks the last Collect call found endpoints for.
// The parsers run in a fixed order, so the report is stable across runs.
func (c *JavaCollector) Frameworks() []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]string(nil), c.frameworks...)
}

// Collect walks the source directory and extracts endpoints from Spring MVC, JAX-RS, and Feign sources.
// When maven-indexer-cli is available and a build file (pom.xml/build.gradle)
// declares dependencies, it resolves types that live in dependency sources via
// the indexer CLI; when that is not possible the outcome is reported through
// DependencyResolution.
func (c *JavaCollector) Collect(ctx collector.CollectContext) ([]collector.ApiEndpoint, error) {
	endpoints, _, err := c.collect(ctx, nil)
	return endpoints, err
}

// CollectWithTypes collects the Java endpoints of a tree that also holds
// sources of another JVM language, such as a Kotlin service with a few Java
// classes left in it.
//
// convert is given the Java declarations parsed from the tree and returns the
// other language's declarations converted to the Java model. They join the type
// table of every framework parser, so a Java controller can resolve a request or
// response class written in that language; they never contribute endpoints of
// their own. The conversion takes the Java declarations because the names the
// other language writes bind to them: a class of the package that is called
// Number hides the standard Number there, as it does in Java. The Java
// declarations are returned alongside the endpoints too, so the caller can
// resolve its own endpoints against them without parsing the tree a second time.
func (c *JavaCollector) CollectWithTypes(ctx collector.CollectContext, convert func(javaTypes []parser.Class) []parser.Class) ([]collector.ApiEndpoint, []parser.Class, error) {
	return c.collect(ctx, convert)
}

// ShareDependencies makes the next Collect call resolve dependency types
// through deps instead of setting up its own, so the two halves of a mixed
// tree read the project's dependency tree once and fetch each class once.
// The caller keeps ownership of deps and closes it.
func (c *JavaCollector) ShareDependencies(deps *Dependencies) {
	c.shared = deps
}

func (c *JavaCollector) collect(ctx collector.CollectContext, convert func(javaTypes []parser.Class) []parser.Class) ([]collector.ApiEndpoint, []parser.Class, error) {
	requiredFields, err := ctx.RequiredFields()
	if err != nil {
		return nil, nil, err
	}
	deps := c.shared
	if deps == nil {
		deps = SetupDependencies(ctx.SourceDir, ctx.NoDeps)
		defer deps.Close()
	}
	depResolver := deps.resolver
	c.mu.Lock()
	c.depResolution = deps.Resolution()
	c.mu.Unlock()

	p, err := parser.NewParser(parser.ParserOptions{})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create java parser: %w", err)
	}
	defer p.Close()

	// A file-level export parses the whole project all the same, because the
	// selected files refer to types declared anywhere in it, but the framework
	// parsers below extract endpoints only for the declarations of the
	// selected files.
	selection := ctx.FileSelection()
	for _, file := range ctx.SelectedFiles() {
		if filepath.Ext(file) == ".kt" {
			return nil, nil, fmt.Errorf("%s is a Kotlin source file but the java collector only parses Java", file)
		}
	}

	var warnings []string
	var results []parser.ParseResult
	if selection == nil {
		results, err = p.ParseDirectory(ctx.SourceDir)
		if err != nil {
			// A Kotlin-only project must fail loudly: the Java grammar
			// cannot parse it, and an empty success would hide the gap.
			var ko *parser.KotlinOnlyError
			if errors.As(err, &ko) {
				return nil, nil, err
			}
			return nil, nil, fmt.Errorf("failed to parse directory %s: %w", ctx.SourceDir, err)
		}
	} else {
		results, warnings, err = parseSelectedProject(p, ctx.SourceDir, selection)
		if err != nil {
			return nil, nil, err
		}
	}

	var parsed []parser.Class
	for _, r := range results {
		if r.Error == nil {
			parsed = append(parsed, r.Classes...)
		}
	}
	if convert != nil {
		if types := convert(parsed); len(types) > 0 {
			results = append(results, parser.ParseResult{Classes: types})
		}
	}

	frameworks := resolveFrameworks(ctx)
	var endpoints []collector.ApiEndpoint
	// Only a framework that produced endpoints was actually in use: a project
	// written for Spring MVC still has its JAX-RS parser run over it.
	var matched []string

	// Shared across framework parsers: a type may be referenced by endpoints
	// from more than one framework.
	unresolved := collector.NewUnresolvedSet()

	if frameworks["spring-mvc"] {
		sm := springmvc.NewParser()
		if depResolver != nil {
			sm.SetDependencyResolver(depResolver)
		}
		sm.SetUnresolved(unresolved)
		sm.SetRequiredFields(requiredFields)
		sm.SetSelection(selection)
		before := len(endpoints)
		controllers := sm.ExtractControllers(results)
		warnings = append(warnings, sm.Warnings()...)
		for _, ctrl := range controllers {
			for _, ep := range ctrl.Endpoints {
				api := springmvcEndpointToAPI(ep, ctrl.Name)
				api.FolderDescription = ctrl.Description
				endpoints = append(endpoints, api)
			}
		}
		if len(endpoints) > before {
			matched = append(matched, "spring-mvc")
		}
	}

	if frameworks["jaxrs"] {
		jr := jaxrs.NewParser()
		if depResolver != nil {
			jr.SetDependencyResolver(depResolver)
		}
		jr.SetUnresolved(unresolved)
		jr.SetRequiredFields(requiredFields)
		jr.SetSelection(selection)
		before := len(endpoints)
		for _, res := range jr.ExtractResources(results) {
			for _, ep := range res.Endpoints {
				api := jaxrsEndpointToAPI(ep, res.Name)
				api.FolderDescription = res.Description
				endpoints = append(endpoints, api)
			}
		}
		if len(endpoints) > before {
			matched = append(matched, "jaxrs")
		}
		warnings = append(warnings, jr.Warnings()...)
	}

	if frameworks["feign"] {
		fg := feign.NewParser()
		if depResolver != nil {
			fg.SetDependencyResolver(depResolver)
		}
		fg.SetUnresolved(unresolved)
		fg.SetRequiredFields(requiredFields)
		fg.SetSelection(selection)
		before := len(endpoints)
		for _, client := range fg.ExtractClients(results) {
			for _, ep := range client.Endpoints {
				endpoints = append(endpoints, feignEndpointToAPI(ep, client.Name))
			}
		}
		if len(endpoints) > before {
			matched = append(matched, "feign")
		}
	}

	c.mu.Lock()
	c.unresolved = unresolved.Counts()
	c.warnings = warnings
	c.depResolution = deps.Resolution()
	c.frameworks = matched
	c.mu.Unlock()

	// Deduplicate endpoints by (Folder, Path, Method, Name).
	seen := make(map[string]bool, len(endpoints))
	deduped := make([]collector.ApiEndpoint, 0, len(endpoints))
	for _, ep := range endpoints {
		key := ep.Folder + "|" + ep.Path + "|" + ep.Method + "|" + ep.Name
		if !seen[key] {
			seen[key] = true
			deduped = append(deduped, ep)
		}
	}
	return deduped, parsed, nil
}

// SelectsFiles reports that Collect limits its result to the files a context
// selects: it extracts and resolves endpoints only for the declarations of
// those files, so the engine takes the result as it is.
func (c *JavaCollector) SelectsFiles() bool { return true }

// ParseTypes parses the Java sources under dir and returns their declarations
// without extracting any endpoint. A collector of another JVM language whose
// selected files hold no Java needs them only as types its endpoints may refer
// to.
func ParseTypes(dir string) ([]parser.Class, error) {
	p, err := parser.NewParser(parser.ParserOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to create java parser: %w", err)
	}
	defer p.Close()

	results, err := p.ParseDirectory(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to parse directory %s: %w", dir, err)
	}
	var classes []parser.Class
	for _, r := range results {
		if r.Error == nil {
			classes = append(classes, r.Classes...)
		}
	}
	return classes, nil
}

// parseSelectedProject parses the project under dir for a file-level export.
//
// The project is parsed as a directory export parses it, and each selected file
// is read as part of that parse, once. A selected file the directory walk did
// not reach, such as one outside dir, is parsed on its own and added. A
// selected file that cannot be parsed fails the export; a project that cannot
// be read as a whole leaves the selected files alone, and says so in a
// warning, because the types declared elsewhere are then unavailable.
func parseSelectedProject(p *parser.Parser, dir string, selection *collector.FileSelection) ([]parser.ParseResult, []string, error) {
	var warnings []string
	results, err := p.ParseDirectory(dir)
	if err != nil {
		warnings = append(warnings, fmt.Sprintf(
			"could not read the project under %s (%v): types declared outside the selected files are not available", dir, err))
		results = nil
	}

	paths := make([]string, 0, len(results))
	for _, r := range results {
		paths = append(paths, r.FilePath)
	}
	for _, file := range selection.Unmatched(paths) {
		r, parseErr := p.ParseFile(file)
		if parseErr != nil {
			return nil, nil, fmt.Errorf("failed to parse file %s: %w", file, parseErr)
		}
		results = append(results, *r)
	}

	for _, r := range results {
		if r.Error != nil && selection.Contains(r.FilePath) {
			return nil, nil, fmt.Errorf("failed to parse file %s: %w", r.FilePath, r.Error)
		}
	}
	return results, warnings, nil
}

// installGuidance tells how to get the indexer CLI.
const installGuidance = "Install it with `npm install -g maven-indexer-cli` (https://github.com/tangcent/maven-indexer-mcp) and make sure it is on PATH"

// Dependencies is the dependency-based type resolution of one Collect call.
type Dependencies struct {
	resolution collector.DependencyResolution
	resolver   *maven.MavenDependencyResolver
}

// NewDependencies returns the active dependency resolution that reads classes
// through dr. SetupDependencies builds one when the project's build file and
// the indexer allow it; a caller that brings a resolver of its own, such as a
// test that scripts the indexer's answers, shares it between collectors the
// same way. The caller closes the result.
func NewDependencies(dr *maven.MavenDependencyResolver) *Dependencies {
	return &Dependencies{
		resolution: collector.DependencyResolution{State: collector.DependencyResolutionActive},
		resolver:   dr,
	}
}

// SetupDependencies decides whether dependency-based type resolution runs for
// the project in sourceDir, and why not when it does not.
//
// The build file is read before the tool check so that a project with no
// dependencies reports no-deps instead of blaming a missing tool. A root that
// aggregates modules usually declares no dependency itself, so it is asked
// about before it is judged to have none.
//
// When resolution runs with maven-indexer-cli 1.2.0 or later, the project's
// dependency tree is read first: every class is then pinned to the version
// the tree names. An older CLI reads classes with get-class, as before, from
// the version the index holds.
func SetupDependencies(sourceDir string, noDeps bool) *Dependencies {
	state := func(s collector.DependencyResolutionState, detail string) *Dependencies {
		return &Dependencies{resolution: collector.DependencyResolution{State: s, Detail: detail}}
	}
	if noDeps {
		return state(collector.DependencyResolutionDisabled, "")
	}

	if !maven.HasBuildFile(sourceDir) {
		return state(collector.DependencyResolutionNoDeps, "")
	}

	declared, _ := maven.DetectDependencies(sourceDir)
	modules := maven.DeclaresModules(sourceDir)
	if len(declared) == 0 && !modules {
		return state(collector.DependencyResolutionNoDeps, "")
	}

	if !mavenAvailable() {
		affected := fmt.Sprintf("the %d declared dependencies", len(declared))
		if len(declared) == 0 {
			affected = "the dependencies of its modules"
		}
		return state(collector.DependencyResolutionToolMissing,
			fmt.Sprintf("%s not found on PATH — request and response types from %s cannot be resolved. %s",
				maven.ToolName, affected, installGuidance))
	}

	active := func(detail string) *Dependencies {
		dr, err := maven.NewMavenDependencyResolver()
		if err != nil {
			return state(collector.DependencyResolutionToolMissing,
				fmt.Sprintf("%s found but the dependency class resolver failed to start: %v", maven.ToolName, err))
		}
		deps := state(collector.DependencyResolutionActive, detail)
		deps.resolver = dr
		return deps
	}

	// A CLI older than 1.2.0 has neither project-context nor explore.
	if version, err := mavenVersion(); err == nil && !maven.PinsProjectVersions(version) {
		deps := active(fmt.Sprintf("%s %s reads each type from the version the index holds; 1.2.0 or later pins types to the versions the project uses",
			maven.ToolName, version))
		if deps.resolver != nil {
			deps.resolver.UseGetClass()
		}
		return deps
	}

	project, err := loadProjectContext(sourceDir)
	if err != nil {
		// Classes can still be read; only the versions are in doubt.
		deps := active(fmt.Sprintf("%s could not read the project's dependency tree (%v): types may come from a version other than the one the project uses",
			maven.ToolName, err))
		if deps.resolver != nil {
			deps.resolver.SetProjectPath(sourceDir)
		}
		return deps
	}
	if !project.HasDependencies() && len(declared) == 0 {
		return state(collector.DependencyResolutionNoDeps, "")
	}

	detail := ""
	if project.Tree != "resolved" {
		detail = unpinnedTreeDetail(project)
	}
	deps := active(detail)
	if deps.resolver != nil {
		deps.resolver.SetProjectPath(sourceDir)
	}
	return deps
}

// unpinnedTreeDetail explains why versions may not follow the project when its
// full dependency tree is unknown.
func unpinnedTreeDetail(project maven.ProjectContext) string {
	reason := "it has not been resolved"
	if project.ResolveError != "" {
		reason = project.ResolveError
	}
	if project.Declared > 0 {
		return fmt.Sprintf("the project's full dependency tree is unavailable (%s): only the %d dependencies its build file declares are pinned to their versions, other types come from the version the index holds",
			reason, project.Declared)
	}
	return fmt.Sprintf("the project's dependency tree is unavailable (%s): types come from the version the index holds, not the one the project uses", reason)
}

// Resolution reports the outcome, counting what the resolver did so far.
func (d *Dependencies) Resolution() collector.DependencyResolution {
	res := d.resolution
	if d.resolver != nil {
		res.ResolvedTypes = d.resolver.ResolvedCount()
		res.Unpinned = d.resolver.Unpinned()
		res.Failed = d.resolver.Failed()
	}
	return res
}

// Resolver returns the class resolver, or nil when resolution does not run.
func (d *Dependencies) Resolver() *maven.MavenDependencyResolver {
	return d.resolver
}

// Close releases the resolver.
func (d *Dependencies) Close() {
	if d.resolver != nil {
		d.resolver.Close()
	}
}

// resolveFrameworks returns the set of frameworks to parse.
// If no hints are provided, all supported frameworks are enabled.
func resolveFrameworks(ctx collector.CollectContext) map[string]bool {
	if len(ctx.Frameworks) == 0 {
		return map[string]bool{
			"spring-mvc": true,
			"jaxrs":      true,
			"feign":      true,
		}
	}

	frameworks := make(map[string]bool, len(ctx.Frameworks))
	for _, f := range ctx.Frameworks {
		switch f {
		case "spring", "spring-mvc", "springmvc":
			frameworks["spring-mvc"] = true
		case "jaxrs", "jax-rs":
			frameworks["jaxrs"] = true
		case "feign":
			frameworks["feign"] = true
		}
	}
	return frameworks
}

func springmvcEndpointToAPI(ep springmvc.Endpoint, folder string) collector.ApiEndpoint {
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
		if p.ParamType == "body" {
			out.RequestBody = &collector.ApiBody{Name: p.Name, MediaType: "application/json"}
		} else if isMultipartFile(p.HeldType()) {
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
		} else if p.ParamType == "header" {
			out.Headers = append(out.Headers, collector.ApiHeader{
				Name:        p.Name,
				Value:       p.DefaultValue,
				Description: p.Description,
				Example:     p.Example,
				Required:    p.Required,
			})
		} else {
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

func jaxrsEndpointToAPI(ep jaxrs.Endpoint, folder string) collector.ApiEndpoint {
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
		if p.ParamType == "body" {
			out.RequestBody = &collector.ApiBody{Name: p.Name, MediaType: firstMediaType(ep.Consumes)}
		} else if isMultipartFile(p.Type) {
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
		} else if p.ParamType == "header" {
			out.Headers = append(out.Headers, collector.ApiHeader{
				Name:        p.Name,
				Value:       p.DefaultValue,
				Description: p.Description,
				Example:     p.Example,
				Required:    p.Required,
			})
		} else {
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
			MediaType: firstMediaType(ep.Produces, "application/json"),
			Body:      ep.ResponseSchema,
		}
	}
	return out
}

func feignEndpointToAPI(ep feign.Endpoint, folder string) collector.ApiEndpoint {
	out := collector.ApiEndpoint{
		Name:        ep.MethodName,
		Folder:      folder,
		Description: ep.Description,
		Tags:        []string{folder},
		Path:        ep.Path,
		Method:      string(ep.Method),
		Protocol:    "http",
	}
	out.Deprecated, out.DeprecationNote = ep.Deprecated, ep.DeprecationNote
	for _, p := range ep.Parameters {
		if p.ParamType == "body" {
			out.RequestBody = &collector.ApiBody{Name: p.Name, MediaType: "application/json"}
		} else if isMultipartFile(p.Type) {
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
		} else if p.ParamType == "header" {
			out.Headers = append(out.Headers, collector.ApiHeader{
				Name:        p.Name,
				Value:       p.DefaultValue,
				Description: p.Description,
				Example:     p.Example,
				Required:    p.Required,
			})
		} else {
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

// isMultipartFile reports whether a declared Java type is a multipart file
// part. The parameter travels in a multipart form body, so it is a form
// parameter: documenting it under Query would tell a reader to send ?file=.
// isMultipartFile reports whether a parameter of this type is uploaded as
// file parts: a MultipartFile, or a collection or array of them.
func isMultipartFile(javaType string) bool {
	element := springmvc.ElementType(javaType)
	return element == "MultipartFile" || strings.HasSuffix(element, ".MultipartFile")
}

// firstMediaType returns the first declared media type, or fallback when the
// source declares none.
func firstMediaType(mediaTypes []string, fallback ...string) string {
	if len(mediaTypes) > 0 {
		return mediaTypes[0]
	}
	if len(fallback) > 0 {
		return fallback[0]
	}
	return ""
}
