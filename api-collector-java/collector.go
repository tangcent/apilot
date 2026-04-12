// Package javacollector implements the Collector interface for Java/Kotlin projects.
// Supported frameworks: Spring MVC, JAX-RS, Feign.
package javacollector

import (
	"fmt"

	collector "github.com/tangcent/apilot/api-collector"
	"github.com/tangcent/apilot/api-collector-java/feign"
	"github.com/tangcent/apilot/api-collector-java/jaxrs"
	"github.com/tangcent/apilot/api-collector-java/parser"
	"github.com/tangcent/apilot/api-collector-java/springmvc"
)

// JavaCollector parses Java/Kotlin source trees for API endpoints.
type JavaCollector struct{}

// New returns a new JavaCollector.
func New() collector.Collector { return &JavaCollector{} }

func (c *JavaCollector) Name() string { return "java" }

func (c *JavaCollector) SupportedLanguages() []string { return []string{"java", "kotlin"} }

// Collect walks the source directory and extracts endpoints from Spring MVC, JAX-RS, and Feign sources.
func (c *JavaCollector) Collect(ctx collector.CollectContext) ([]collector.ApiEndpoint, error) {
	p, err := parser.NewParser(parser.ParserOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to create java parser: %w", err)
	}
	defer p.Close()

	results, err := p.ParseDirectory(ctx.SourceDir)
	if err != nil {
		return nil, fmt.Errorf("failed to parse directory %s: %w", ctx.SourceDir, err)
	}

	frameworks := resolveFrameworks(ctx)
	var endpoints []collector.ApiEndpoint

	if frameworks["spring-mvc"] {
		sm := springmvc.NewParser()
		for _, ctrl := range sm.ExtractControllers(results) {
			for _, ep := range ctrl.Endpoints {
				endpoints = append(endpoints, springmvcEndpointToAPI(ep, ctrl.Name))
			}
		}
	}

	if frameworks["jaxrs"] {
		jr := jaxrs.NewParser()
		for _, res := range jr.ExtractResources(results) {
			for _, ep := range res.Endpoints {
				endpoints = append(endpoints, jaxrsEndpointToAPI(ep, res.Name))
			}
		}
	}

	if frameworks["feign"] {
		fg := feign.NewParser()
		for _, client := range fg.ExtractClients(results) {
			for _, ep := range client.Endpoints {
				endpoints = append(endpoints, feignEndpointToAPI(ep, client.Name))
			}
		}
	}

	return endpoints, nil
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
		Name:     ep.MethodName,
		Folder:   folder,
		Path:     ep.Path,
		Method:   string(ep.Method),
		Protocol: "http",
	}
	for _, p := range ep.Parameters {
		if p.ParamType == "body" {
			out.RequestBody = &collector.ApiBody{MediaType: "application/json"}
		} else {
			out.Parameters = append(out.Parameters, collector.ApiParameter{
				Name:     p.Name,
				Type:     "text",
				In:       p.ParamType,
				Required: p.Required,
				Default:  p.DefaultValue,
			})
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
		Name:     ep.MethodName,
		Folder:   folder,
		Path:     ep.Path,
		Method:   string(ep.Method),
		Protocol: "http",
	}
	for _, p := range ep.Parameters {
		if p.ParamType == "body" {
			out.RequestBody = &collector.ApiBody{MediaType: mediaType}
		} else {
			out.Parameters = append(out.Parameters, collector.ApiParameter{
				Name:     p.Name,
				Type:     "text",
				In:       p.ParamType,
				Required: p.Required,
			})
		}
	}
	return out
}

func feignEndpointToAPI(ep feign.Endpoint, folder string) collector.ApiEndpoint {
	out := collector.ApiEndpoint{
		Name:     ep.MethodName,
		Folder:   folder,
		Path:     ep.Path,
		Method:   string(ep.Method),
		Protocol: "http",
	}
	for _, p := range ep.Parameters {
		if p.ParamType == "body" {
			out.RequestBody = &collector.ApiBody{MediaType: "application/json"}
		} else {
			out.Parameters = append(out.Parameters, collector.ApiParameter{
				Name:     p.Name,
				Type:     "text",
				In:       p.ParamType,
				Required: p.Required,
			})
		}
	}
	return out
}
