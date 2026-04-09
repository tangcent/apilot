// Package curl implements the Formatter interface producing cURL command output.
package curl

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/tangcent/apilot/api-collector/collector"
	"github.com/tangcent/apilot/api-formater/formater"
)

// CurlFormatter formats endpoints as cURL commands.
type CurlFormatter struct{}

// New returns a new CurlFormatter.
func New() formater.Formatter { return &CurlFormatter{} }

func (f *CurlFormatter) Name() string { return "curl" }

func (f *CurlFormatter) SupportedFormats() []string { return []string{"curl"} }

// Format produces one cURL command per endpoint, separated by blank lines.
// An empty endpoints slice returns an empty byte slice.
func (f *CurlFormatter) Format(endpoints []collector.ApiEndpoint, _ formater.FormatOptions) ([]byte, error) {
	var buf bytes.Buffer
	for i, ep := range endpoints {
		if i > 0 {
			buf.WriteString("\n\n")
		}
		buf.WriteString(buildCurl(ep))
	}
	return buf.Bytes(), nil
}

func buildCurl(ep collector.ApiEndpoint) string {
	var sb strings.Builder

	// Method
	method := ep.Method
	if method == "" {
		method = "GET"
	}
	sb.WriteString(fmt.Sprintf("curl -X %s", method))

	// Headers
	for _, h := range ep.Headers {
		sb.WriteString(fmt.Sprintf(" \\\n  -H '%s: %s'", h.Name, h.Value))
	}

	// Query parameters
	var queryParts []string
	for _, p := range ep.Parameters {
		if p.In == "query" {
			queryParts = append(queryParts, fmt.Sprintf("%s=", p.Name))
		}
	}

	path := ep.Path
	if len(queryParts) > 0 {
		path += "?" + strings.Join(queryParts, "&")
	}
	sb.WriteString(fmt.Sprintf(" \\\n  'http://localhost%s'", path))

	// Request body
	if ep.RequestBody != nil {
		sb.WriteString(" \\\n  -H 'Content-Type: application/json'")
		sb.WriteString(" \\\n  -d '{}'")
	}

	return sb.String()
}
