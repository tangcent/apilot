// Package curl implements the Formatter interface producing cURL command output.
package curl

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/tangcent/apilot/api-formatter"
	"github.com/tangcent/apilot/api-model"
)

// Params holds curl-specific formatting options.
type Params struct {
	// BaseURL is prepended to each endpoint path. Defaults to "http://localhost".
	BaseURL string `json:"baseURL"`
}

// CurlFormatter formats endpoints as cURL commands.
type CurlFormatter struct{}

// New returns a new CurlFormatter.
func New() formatter.Formatter { return &CurlFormatter{} }

func (f *CurlFormatter) Name() string { return "curl" }

// Format produces one cURL command per endpoint, separated by blank lines.
// An empty endpoints slice returns an empty byte slice.
func (f *CurlFormatter) Format(endpoints []model.ApiEndpoint, opts formatter.FormatOptions) ([]byte, error) {
	var p Params
	if err := opts.DecodeParams(&p); err != nil {
		return nil, err
	}
	if p.BaseURL == "" {
		p.BaseURL = "http://localhost"
	}

	var buf bytes.Buffer
	for i, ep := range endpoints {
		if i > 0 {
			buf.WriteString("\n\n")
		}
		buf.WriteString(buildCurl(ep, p))
	}
	return buf.Bytes(), nil
}

func buildCurl(ep model.ApiEndpoint, p Params) string {
	var sb strings.Builder

	method := ep.Method
	if method == "" {
		method = "GET"
	}
	sb.WriteString(fmt.Sprintf("curl -X %s", method))

	for _, h := range ep.Headers {
		sb.WriteString(fmt.Sprintf(" \\\n  -H '%s: %s'", h.Name, h.Value))
	}

	var queryParts []string
	for _, param := range ep.Parameters {
		if param.In == "query" {
			queryParts = append(queryParts, fmt.Sprintf("%s=", param.Name))
		}
	}

	path := ep.Path
	if len(queryParts) > 0 {
		path += "?" + strings.Join(queryParts, "&")
	}
	sb.WriteString(fmt.Sprintf(" \\\n  '%s%s'", p.BaseURL, path))

	if ep.RequestBody != nil {
		sb.WriteString(" \\\n  -H 'Content-Type: application/json'")
		sb.WriteString(" \\\n  -d '{}'")
	}

	return sb.String()
}
