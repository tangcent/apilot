package springmvc

import (
	"path"
	"strings"

	"github.com/tangcent/apilot/api-collector-java/parser"
)

// Parser extracts Spring MVC endpoints from parsed Java classes
type Parser struct{}

// NewParser creates a new Spring MVC parser
func NewParser() *Parser {
	return &Parser{}
}

// ExtractControllers extracts Spring MVC controllers from parse results
func (p *Parser) ExtractControllers(results []parser.ParseResult) []Controller {
	var controllers []Controller

	for _, result := range results {
		if result.Error != nil {
			continue
		}

		for _, class := range result.Classes {
			if controller := p.extractController(class); controller != nil {
				controllers = append(controllers, *controller)
			}
		}
	}

	return controllers
}

// extractController extracts a controller if the class has @RestController or @Controller
func (p *Parser) extractController(class parser.Class) *Controller {
	if !p.isController(class) {
		return nil
	}

	basePath := p.extractBasePath(class.Annotations)

	var endpoints []Endpoint
	for _, method := range class.Methods {
		if endpoint := p.extractEndpoint(method, basePath, class); endpoint != nil {
			endpoints = append(endpoints, *endpoint)
		}
	}

	return &Controller{
		Name:      class.Name,
		Package:   class.Package,
		BasePath:  basePath,
		Endpoints: endpoints,
	}
}

// isController checks if a class is a Spring MVC controller
func (p *Parser) isController(class parser.Class) bool {
	for _, ann := range class.Annotations {
		if ann.Name == "RestController" || ann.Name == "Controller" {
			return true
		}
	}
	return false
}

// extractBasePath extracts the base path from @RequestMapping on the class
func (p *Parser) extractBasePath(annotations []parser.Annotation) string {
	for _, ann := range annotations {
		if ann.Name == "RequestMapping" {
			if value, ok := ann.Params["value"]; ok {
				return p.normalizePath(value)
			}
			if path, ok := ann.Params["path"]; ok {
				return p.normalizePath(path)
			}
		}
	}
	return ""
}

// extractEndpoint extracts an endpoint from a method
func (p *Parser) extractEndpoint(method parser.Method, basePath string, class parser.Class) *Endpoint {
	httpMethod, methodPath := p.extractMethodInfo(method.Annotations)
	if httpMethod == "" {
		return nil
	}

	fullPath := p.combinePaths(basePath, methodPath)

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
		ClassName:  class.Name,
		Package:    class.Package,
	}
}

// extractMethodInfo extracts HTTP method and path from method annotations
func (p *Parser) extractMethodInfo(annotations []parser.Annotation) (HTTPMethod, string) {
	for _, ann := range annotations {
		switch ann.Name {
		case "GetMapping":
			return GET, p.extractPathFromMapping(ann)
		case "PostMapping":
			return POST, p.extractPathFromMapping(ann)
		case "PutMapping":
			return PUT, p.extractPathFromMapping(ann)
		case "DeleteMapping":
			return DELETE, p.extractPathFromMapping(ann)
		case "PatchMapping":
			return PATCH, p.extractPathFromMapping(ann)
		case "RequestMapping":
			method := p.extractHTTPMethodFromRequestMapping(ann)
			path := p.extractPathFromMapping(ann)
			return method, path
		}
	}
	return "", ""
}

// extractPathFromMapping extracts path from mapping annotation
func (p *Parser) extractPathFromMapping(ann parser.Annotation) string {
	if value, ok := ann.Params["value"]; ok {
		return p.normalizePath(value)
	}
	if path, ok := ann.Params["path"]; ok {
		return p.normalizePath(path)
	}
	return ""
}

// extractHTTPMethodFromRequestMapping extracts HTTP method from @RequestMapping
func (p *Parser) extractHTTPMethodFromRequestMapping(ann parser.Annotation) HTTPMethod {
	if method, ok := ann.Params["method"]; ok {
		method = strings.TrimPrefix(method, "RequestMethod.")
		switch method {
		case "GET":
			return GET
		case "POST":
			return POST
		case "PUT":
			return PUT
		case "DELETE":
			return DELETE
		case "PATCH":
			return PATCH
		}
	}
	return GET // Default to GET if not specified
}

// extractParameter extracts parameter information
func (p *Parser) extractParameter(param parser.Parameter) *EndpointParameter {
	paramType := p.detectParameterType(param.Annotations)
	if paramType == "" {
		return nil
	}

	ep := &EndpointParameter{
		Name:      param.Name,
		Type:      param.Type,
		ParamType: paramType,
		Required:  true, // Default to required
	}

	// Extract additional info from annotations
	for _, ann := range param.Annotations {
		switch ann.Name {
		case "RequestParam":
			if required, ok := ann.Params["required"]; ok {
				ep.Required = required != "false"
			}
			if defaultValue, ok := ann.Params["defaultValue"]; ok {
				ep.DefaultValue = defaultValue
				ep.Required = false
			}
		}
	}

	return ep
}

// detectParameterType detects parameter type from annotations
func (p *Parser) detectParameterType(annotations []parser.Annotation) string {
	for _, ann := range annotations {
		switch ann.Name {
		case "PathVariable":
			return "path"
		case "RequestParam":
			return "query"
		case "RequestBody":
			return "body"
		case "RequestHeader":
			return "header"
		}
	}
	return ""
}

// normalizePath normalizes a path string
func (p *Parser) normalizePath(pathStr string) string {
	// Remove quotes
	pathStr = strings.Trim(pathStr, "\"")
	pathStr = strings.Trim(pathStr, "'")

	// Ensure leading slash
	if pathStr != "" && !strings.HasPrefix(pathStr, "/") {
		pathStr = "/" + pathStr
	}

	return pathStr
}

// combinePaths combines base path and method path
func (p *Parser) combinePaths(basePath, methodPath string) string {
	if basePath == "" {
		return methodPath
	}
	if methodPath == "" {
		return basePath
	}
	return path.Join(basePath, methodPath)
}
