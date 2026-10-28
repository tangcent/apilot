// Package kotlincollector parses Kotlin source trees for Spring MVC endpoints.
package kotlincollector

import (
	"fmt"
	"path/filepath"
	"strings"
	"sync"

	collector "github.com/tangcent/apilot/api-collector"
	ktparser "github.com/tangcent/apilot/api-collector-kotlin/parser"
	"github.com/tangcent/apilot/api-collector-kotlin/resolver"
	"github.com/tangcent/apilot/api-collector-kotlin/springmvc"
)

// KotlinCollector parses Kotlin source trees for API endpoints.
type KotlinCollector struct {
	mu         sync.Mutex
	unresolved map[string]int
	warnings   []string
	frameworks []string
}

// New returns a new KotlinCollector.
func New() collector.Collector { return &KotlinCollector{} }

func (c *KotlinCollector) Name() string { return "kotlin" }

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

// Collect walks the Kotlin source and extracts Spring MVC endpoints.
func (c *KotlinCollector) Collect(ctx collector.CollectContext) ([]collector.ApiEndpoint, error) {
	p, err := ktparser.NewParser()
	if err != nil {
		return nil, fmt.Errorf("failed to create kotlin parser: %w", err)
	}
	defer p.Close()

	var results []ktparser.ParseResult
	if ctx.SourceFile != "" {
		if strings.ToLower(filepath.Ext(ctx.SourceFile)) != ".kt" {
			return nil, fmt.Errorf("%s is not a Kotlin source file", ctx.SourceFile)
		}
		fileResult, parseErr := p.ParseFile(ctx.SourceFile)
		if parseErr != nil {
			return nil, fmt.Errorf("failed to parse file %s: %w", ctx.SourceFile, parseErr)
		}
		results = append(results, *fileResult)
		// Parse the rest of the project so cross-file DTO references resolve.
		dirResults, dirErr := p.ParseDirectory(ctx.SourceDir)
		if dirErr == nil {
			results = append(results, dirResults...)
		}
	} else {
		results, err = p.ParseDirectory(ctx.SourceDir)
		if err != nil {
			return nil, fmt.Errorf("failed to parse directory %s: %w", ctx.SourceDir, err)
		}
	}

	for _, result := range results {
		if result.Error != nil {
			return nil, result.Error
		}
	}

	classes := flattenClasses(results)
	warnings := collectWarnings(results, classes)

	var endpoints []collector.ApiEndpoint
	var matched []string
	var unresolved map[string]int

	if springMVCDisabled(ctx.Frameworks) {
		endpoints = nil
		unresolved = map[string]int{}
	} else {
		typeResolver := resolver.New(classes)
		sm := springmvc.NewParser(typeResolver)
		before := len(endpoints)
		for _, ctrl := range sm.ExtractControllers(results) {
			for _, ep := range ctrl.Endpoints {
				endpoints = append(endpoints, springMVCEndpointToAPI(ep, ctrl.Name))
			}
		}
		if len(endpoints) > before {
			matched = append(matched, "spring-mvc")
		}
		warnings = append(warnings, sm.Warnings()...)
		unresolved = typeResolver.Unresolved()
	}

	c.mu.Lock()
	c.unresolved = unresolved
	c.warnings = warnings
	c.frameworks = matched
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

func collectWarnings(results []ktparser.ParseResult, classes []ktparser.Class) []string {
	var warnings []string
	for _, result := range results {
		warnings = append(warnings, result.Warnings...)
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
	for _, p := range ep.Parameters {
		switch {
		case p.ParamType == "body":
			out.RequestBody = &collector.ApiBody{Name: p.Name, MediaType: "application/json"}
		case isMultipartFile(p.Type):
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

func isMultipartFile(kotlinType string) bool {
	return kotlinType == "MultipartFile" || strings.HasSuffix(kotlinType, ".MultipartFile")
}
