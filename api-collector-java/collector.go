// Package javacollector implements the Collector interface for Java projects.
// Supported frameworks: Spring MVC, JAX-RS, Feign.
package javacollector

import (
	"errors"
	"fmt"
	"path/filepath"
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
	depResolution collector.DependencyResolution
}

// mavenAvailable is a variable so tests can force the tool-missing branch
// without uninstalling maven-indexer-cli from the machine.
var mavenAvailable = maven.IsAvailable

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

// DependencyResolution reports whether dependency-based type resolution ran
// during the last Collect call, and why not when it did not.
func (c *JavaCollector) DependencyResolution() collector.DependencyResolution {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.depResolution
}

// Collect walks the source directory and extracts endpoints from Spring MVC, JAX-RS, and Feign sources.
// When maven-indexer-cli is available and a build file (pom.xml/build.gradle)
// declares dependencies, it resolves types that live in dependency sources via
// the indexer CLI; when that is not possible the outcome is reported through
// DependencyResolution.
func (c *JavaCollector) Collect(ctx collector.CollectContext) ([]collector.ApiEndpoint, error) {
	depResolution, depResolver := setupDependencyResolution(ctx.SourceDir, ctx.NoDeps)
	c.mu.Lock()
	c.depResolution = depResolution
	c.mu.Unlock()
	if depResolver != nil {
		defer depResolver.Close()
	}

	p, err := parser.NewParser(parser.ParserOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to create java parser: %w", err)
	}
	defer p.Close()

	var results []parser.ParseResult
	if ctx.SourceFile != "" {
		if filepath.Ext(ctx.SourceFile) == ".kt" {
			return nil, fmt.Errorf("%s is a Kotlin source file but the java collector only parses Java", ctx.SourceFile)
		}
		r, parseErr := p.ParseFile(ctx.SourceFile)
		if parseErr != nil {
			return nil, fmt.Errorf("failed to parse file %s: %w", ctx.SourceFile, parseErr)
		}
		results = []parser.ParseResult{*r}

		resolverResults, resolverErr := p.ParseDirectory(ctx.SourceDir)
		if resolverErr == nil {
			results = append(results, resolverResults...)
		}
	} else {
		results, err = p.ParseDirectory(ctx.SourceDir)
		if err != nil {
			// A Kotlin-only project must fail loudly: the Java grammar
			// cannot parse it, and an empty success would hide the gap.
			var ko *parser.KotlinOnlyError
			if errors.As(err, &ko) {
				return nil, err
			}
			return nil, fmt.Errorf("failed to parse directory %s: %w", ctx.SourceDir, err)
		}
	}

	frameworks := resolveFrameworks(ctx)
	var endpoints []collector.ApiEndpoint

	// Shared across framework parsers: a type may be referenced by endpoints
	// from more than one framework.
	unresolved := collector.NewUnresolvedSet()

	if frameworks["spring-mvc"] {
		sm := springmvc.NewParser()
		if depResolver != nil {
			sm.SetDependencyResolver(depResolver)
		}
		sm.SetUnresolved(unresolved)
		for _, ctrl := range sm.ExtractControllers(results) {
			for _, ep := range ctrl.Endpoints {
				endpoints = append(endpoints, springmvcEndpointToAPI(ep, ctrl.Name))
			}
		}
	}

	if frameworks["jaxrs"] {
		jr := jaxrs.NewParser()
		if depResolver != nil {
			jr.SetDependencyResolver(depResolver)
		}
		jr.SetUnresolved(unresolved)
		for _, res := range jr.ExtractResources(results) {
			for _, ep := range res.Endpoints {
				endpoints = append(endpoints, jaxrsEndpointToAPI(ep, res.Name))
			}
		}
	}

	if frameworks["feign"] {
		fg := feign.NewParser()
		if depResolver != nil {
			fg.SetDependencyResolver(depResolver)
		}
		fg.SetUnresolved(unresolved)
		for _, client := range fg.ExtractClients(results) {
			for _, ep := range client.Endpoints {
				endpoints = append(endpoints, feignEndpointToAPI(ep, client.Name))
			}
		}
	}

	c.mu.Lock()
	c.unresolved = unresolved.Counts()
	depResolution.ResolvedTypes = depResolverCount(depResolver)
	c.depResolution = depResolution
	c.mu.Unlock()

	// Deduplicate endpoints by (Folder, Path, Method, Name) to avoid
	// duplicates when a single file is parsed both individually and as
	// part of directory scanning for type resolution.
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

// setupDependencyResolution decides whether dependency-based type resolution
// runs for this Collect call, and why not when it does not.
//
// The build file is read before the tool check so that a project with no
// dependencies reports no-deps instead of blaming a missing tool.
func setupDependencyResolution(sourceDir string, noDeps bool) (collector.DependencyResolution, *maven.MavenDependencyResolver) {
	if noDeps {
		return collector.DependencyResolution{State: collector.DependencyResolutionDisabled}, nil
	}

	if !maven.HasBuildFile(sourceDir) {
		return collector.DependencyResolution{State: collector.DependencyResolutionNoDeps}, nil
	}

	deps, err := maven.DetectDependencies(sourceDir)
	if err != nil || len(deps) == 0 {
		return collector.DependencyResolution{State: collector.DependencyResolutionNoDeps}, nil
	}

	if !mavenAvailable() {
		return collector.DependencyResolution{
			State: collector.DependencyResolutionToolMissing,
			Detail: fmt.Sprintf("%s not found on PATH — request and response types from the %d declared dependencies cannot be resolved. "+
				"Install it (https://github.com/tangcent/maven-indexer-cli) and make sure it is on PATH",
				maven.ToolName, len(deps)),
		}, nil
	}

	dr, err := maven.NewMavenDependencyResolver()
	if err != nil {
		return collector.DependencyResolution{
			State: collector.DependencyResolutionToolMissing,
			Detail: fmt.Sprintf("%s found but the dependency class resolver failed to start: %v",
				maven.ToolName, err),
		}, nil
	}

	return collector.DependencyResolution{State: collector.DependencyResolutionActive}, dr
}

// depResolverCount reports how many distinct types the resolver expanded from
// dependencies, or 0 when no resolver ran.
func depResolverCount(depResolver *maven.MavenDependencyResolver) int {
	if depResolver == nil {
		return 0
	}
	return depResolver.ResolvedCount()
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
	}
	for _, p := range ep.Parameters {
		if p.ParamType == "body" {
			out.RequestBody = &collector.ApiBody{MediaType: "application/json"}
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
	mediaType := ""
	if len(ep.Consumes) > 0 {
		mediaType = ep.Consumes[0]
	}

	out := collector.ApiEndpoint{
		Name:        ep.MethodName,
		Folder:      folder,
		Description: ep.Description,
		Tags:        []string{folder},
		Path:        ep.Path,
		Method:      string(ep.Method),
		Protocol:    "http",
	}
	for _, p := range ep.Parameters {
		if p.ParamType == "body" {
			out.RequestBody = &collector.ApiBody{MediaType: mediaType}
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
		respMediaType := "application/json"
		if len(ep.Produces) > 0 {
			respMediaType = ep.Produces[0]
		}
		out.Response = &collector.ApiBody{
			MediaType: respMediaType,
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
	for _, p := range ep.Parameters {
		if p.ParamType == "body" {
			out.RequestBody = &collector.ApiBody{MediaType: "application/json"}
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
