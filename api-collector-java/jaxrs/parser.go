// Package jaxrs parses JAX-RS annotated classes for API endpoints.
package jaxrs

import (
	"path"
	"strings"

	"github.com/tangcent/apilot/api-collector-java/parser"
)

// Parser extracts JAX-RS endpoints from parsed Java classes.
type Parser struct{}

// NewParser creates a new JAX-RS parser.
func NewParser() *Parser {
	return &Parser{}
}

// ExtractResources extracts JAX-RS resources from parse results.
func (p *Parser) ExtractResources(results []parser.ParseResult) []Resource {
	var resources []Resource
	for _, result := range results {
		if result.Error != nil {
			continue
		}
		for _, class := range result.Classes {
			if resource := p.extractResource(class); resource != nil {
				resources = append(resources, *resource)
			}
		}
	}
	return resources
}

func (p *Parser) extractResource(class parser.Class) *Resource {
	basePath, hasPath := p.extractPath(class.Annotations)
	if !hasPath {
		return nil
	}

	classProduces := p.extractMediaTypes(class.Annotations, "Produces")
	classConsumes := p.extractMediaTypes(class.Annotations, "Consumes")

	var endpoints []Endpoint
	for _, method := range class.Methods {
		ep := p.extractEndpoint(method, basePath, class)
		if ep == nil {
			continue
		}
		if len(ep.Produces) == 0 {
			ep.Produces = classProduces
		}
		if len(ep.Consumes) == 0 {
			ep.Consumes = classConsumes
		}
		endpoints = append(endpoints, *ep)
	}

	return &Resource{
		Name:      class.Name,
		Package:   class.Package,
		BasePath:  basePath,
		Endpoints: endpoints,
		Produces:  classProduces,
		Consumes:  classConsumes,
	}
}

func (p *Parser) extractEndpoint(method parser.Method, basePath string, class parser.Class) *Endpoint {
	httpMethod, ok := p.extractHTTPMethod(method.Annotations)
	if !ok {
		return nil
	}

	methodPath := ""
	if mp, hasPath := p.extractPath(method.Annotations); hasPath {
		methodPath = mp
	}

	fullPath := combinePaths(basePath, methodPath)

	var params []EndpointParameter
	for _, param := range method.Parameters {
		if ep := p.extractParameter(param); ep != nil {
			params = append(params, *ep)
		}
	}

	return &Endpoint{
		Path:       fullPath,
		Method:     httpMethod,
		MethodName: method.Name,
		Parameters: params,
		ReturnType: method.ReturnType,
		Produces:   p.extractMediaTypes(method.Annotations, "Produces"),
		Consumes:   p.extractMediaTypes(method.Annotations, "Consumes"),
		ClassName:  class.Name,
		Package:    class.Package,
	}
}

func (p *Parser) extractPath(annotations []parser.Annotation) (string, bool) {
	for _, ann := range annotations {
		if ann.Name == "Path" {
			if value, ok := ann.Params["value"]; ok {
				return normalizePath(value), true
			}
			return "", true
		}
	}
	return "", false
}

func (p *Parser) extractHTTPMethod(annotations []parser.Annotation) (HTTPMethod, bool) {
	for _, ann := range annotations {
		switch ann.Name {
		case "GET":
			return GET, true
		case "POST":
			return POST, true
		case "PUT":
			return PUT, true
		case "DELETE":
			return DELETE, true
		case "PATCH":
			return PATCH, true
		case "HEAD":
			return HEAD, true
		case "OPTIONS":
			return OPTIONS, true
		}
	}
	return "", false
}

func (p *Parser) extractParameter(param parser.Parameter) *EndpointParameter {
	for _, ann := range param.Annotations {
		switch ann.Name {
		case "PathParam":
			return &EndpointParameter{Name: param.Name, Type: param.Type, ParamType: "path", Required: true}
		case "QueryParam":
			return &EndpointParameter{Name: param.Name, Type: param.Type, ParamType: "query", Required: false}
		case "FormParam":
			return &EndpointParameter{Name: param.Name, Type: param.Type, ParamType: "form", Required: false}
		case "HeaderParam":
			return &EndpointParameter{Name: param.Name, Type: param.Type, ParamType: "header", Required: false}
		case "CookieParam":
			return &EndpointParameter{Name: param.Name, Type: param.Type, ParamType: "cookie", Required: false}
		}
	}
	return nil
}

func (p *Parser) extractMediaTypes(annotations []parser.Annotation, annName string) []string {
	for _, ann := range annotations {
		if ann.Name == annName {
			if value, ok := ann.Params["value"]; ok {
				return []string{strings.Trim(value, "\"'")}
			}
		}
	}
	return nil
}

func normalizePath(s string) string {
	s = strings.Trim(s, "\"'")
	if s != "" && !strings.HasPrefix(s, "/") {
		s = "/" + s
	}
	return s
}

func combinePaths(basePath, methodPath string) string {
	if basePath == "" {
		return methodPath
	}
	if methodPath == "" {
		return basePath
	}
	return path.Join(basePath, methodPath)
}
