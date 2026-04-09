# Contributing a New Formatter

This guide walks you through adding a new output formatter to APilot. A Formatter converts `[]ApiEndpoint` into a specific output format (Markdown, cURL, Postman, OpenAPI, etc.).

---

## Prerequisites

- Go 1.22 or higher
- Familiarity with the [architecture guide](architecture.md)
- Read the [main contributing guide](../CONTRIBUTING.md)

---

## Step 1 — Create the module directory

Create a new directory at the repo root following the naming convention `api-formatter-{name}/`:

```bash
mkdir api-formatter-openapi
```

Use a short, lowercase name that describes the output format (e.g. `openapi`, `curl`, `postman`).

---

## Step 2 — Add `go.mod`

Create `api-formatter-openapi/go.mod`:

```
module github.com/tangcent/apilot/api-formatter-openapi

go 1.22

require (
	github.com/tangcent/apilot/api-collector v0.0.0
	github.com/tangcent/apilot/api-formatter v0.0.0
)
```

Every formatter depends on both `api-collector` (for the `ApiEndpoint` type) and `api-formatter` (for the `Formatter` interface and `FormatOptions`).

---

## Step 3 — Implement the `Formatter` interface

Create `api-formatter-openapi/formatter.go` and implement the three methods defined in [`formatter.Formatter`](../api-formatter/formatter.go):

```go
package openapi

import (
	"github.com/tangcent/apilot/api-collector/collector"
	"github.com/tangcent/apilot/api-formatter/formatter"
)

type OpenAPIFormatter struct{}

func New() formatter.Formatter { return &OpenAPIFormatter{} }

func (f *OpenAPIFormatter) Name() string { return "openapi" }

func (f *OpenAPIFormatter) SupportedFormats() []string { return []string{"openapi"} }

func (f *OpenAPIFormatter) Format(endpoints []collector.ApiEndpoint, opts formatter.FormatOptions) ([]byte, error) {
	// TODO: convert endpoints to OpenAPI 3.0 YAML/JSON
	return nil, nil
}
```

### Interface contract

| Method | Requirement |
|--------|-------------|
| `Name()` | Return a unique lowercase identifier (e.g. `"openapi"`). Must not collide with existing formatters. |
| `SupportedFormats()` | Return the list of format variant names this formatter handles. For a single-variant formatter, return a one-element slice. |
| `Format(endpoints, opts)` | Convert `[]ApiEndpoint` to `[]byte`. **An empty `endpoints` slice MUST return valid empty output, not an error.** |

### Key rules for `Format`

1. **Empty input** — When `len(endpoints) == 0`, return valid empty output for your format (e.g. an empty JSON array `[]`, an empty Markdown document, a valid but empty Postman collection). Never return an error.
2. **Format variant** — Use `opts.Format` to select the output variant. If empty, default to the first entry in `SupportedFormats()`.
3. **Config** — Use `opts.Config` for formatter-specific key-value options (e.g. collection name, template path).

---

## Step 4 — Export the `New()` constructor

Your package must export a `New() formatter.Formatter` function. This is the single entry point used during registration:

```go
func New() formatter.Formatter { return &OpenAPIFormatter{} }
```

---

## Step 5 — Register in `apilot-cli/main.go`

Add an import and a registration call in [`apilot-cli/main.go`](../apilot-cli/main.go):

```go
import (
	// ... existing imports ...
	openapifmt "github.com/tangcent/apilot/api-formatter-openapi"
)

func init() {
	// ... existing registrations ...
	engine.RegisterFormatter(openapifmt.New())
}
```

---

## Step 6 — Write unit tests

Create `api-formatter-openapi/formatter_test.go`:

```go
package openapi

import (
	"testing"

	"github.com/tangcent/apilot/api-collector/collector"
	"github.com/tangcent/apilot/api-formatter/formatter"
)

func TestFormat_EmptyInput(t *testing.T) {
	f := &OpenAPIFormatter{}
	out, err := f.Format(nil, formatter.FormatOptions{})
	if err != nil {
		t.Fatalf("empty input should not error: %v", err)
	}
	if len(out) == 0 {
		t.Fatal("empty input should produce valid empty output, not nil/empty bytes")
	}
}

func TestFormat_SingleEndpoint(t *testing.T) {
	endpoints := []collector.ApiEndpoint{
		{
			Name:     "ListUsers",
			Path:     "/users",
			Method:   "GET",
			Protocol: "http",
		},
	}
	f := &OpenAPIFormatter{}
	out, err := f.Format(endpoints, formatter.FormatOptions{Format: "openapi"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out) == 0 {
		t.Fatal("expected non-empty output")
	}
}

func TestName(t *testing.T) {
	f := &OpenAPIFormatter{}
	if f.Name() != "openapi" {
		t.Fatalf("expected name 'openapi', got %q", f.Name())
	}
}

func TestSupportedFormats(t *testing.T) {
	f := &OpenAPIFormatter{}
	formats := f.SupportedFormats()
	if len(formats) == 0 {
		t.Fatal("expected at least one supported format")
	}
}
```

### Required test cases

| Test case | What it verifies |
|-----------|-----------------|
| Empty input | `Format(nil, ...)` returns valid output, not an error |
| Single endpoint | Basic formatting works for one endpoint |
| Multiple endpoints | Output is correct for a slice with several entries |
| Round-trip | (If applicable) Parse the output back and verify it matches the input |
| Name / SupportedFormats | Interface methods return correct values |

---

## Step 7 — Update `docs/architecture.md`

Add your formatter to the **Module Map** table and the **Module Dependency Graph** in [`docs/architecture.md`](architecture.md):

**Module Map** — add a row:

```
| `api-formatter-openapi` | Go | OpenAPI 3.0 YAML/JSON formatter |
```

**Module Dependency Graph** — add under `apilot-cli`:

```
├── api-formatter-openapi
```

And add the dependency block:

```
api-formatter-openapi
  ├── api-collector  (for ApiEndpoint type)
  └── api-formatter
```

---

## Complete working example — cURL formatter

Here is a real, minimal formatter from the codebase ([`api-formatter-curl/formatter.go`](../api-formatter-curl/formatter.go)):

```go
package curl

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/tangcent/apilot/api-collector/collector"
	"github.com/tangcent/apilot/api-formatter/formatter"
)

type CurlFormatter struct{}

func New() formatter.Formatter { return &CurlFormatter{} }

func (f *CurlFormatter) Name() string { return "curl" }

func (f *CurlFormatter) SupportedFormats() []string { return []string{"curl"} }

func (f *CurlFormatter) Format(endpoints []collector.ApiEndpoint, _ formatter.FormatOptions) ([]byte, error) {
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
	method := ep.Method
	if method == "" {
		method = "GET"
	}
	sb.WriteString(fmt.Sprintf("curl -X %s", method))
	for _, h := range ep.Headers {
		sb.WriteString(fmt.Sprintf(" \\\n  -H '%s: %s'", h.Name, h.Value))
	}
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
	if ep.RequestBody != nil {
		sb.WriteString(" \\\n  -H 'Content-Type: application/json'")
		sb.WriteString(" \\\n  -d '{}'")
	}
	return sb.String()
}
```

Note how `Format` handles the empty case naturally — the loop simply doesn't execute, and `buf.Bytes()` returns an empty (but valid) byte slice.

---

## Checklist

Before submitting your PR, verify:

- [ ] Module directory is named `api-formatter-{name}/`
- [ ] `go.mod` declares dependencies on `api-collector` and `api-formatter`
- [ ] `Formatter` interface is fully implemented (`Name`, `SupportedFormats`, `Format`)
- [ ] Empty `endpoints` slice returns valid output, not an error
- [ ] `New() formatter.Formatter` constructor is exported
- [ ] Formatter is registered in `apilot-cli/main.go`
- [ ] Unit tests cover empty input, single endpoint, and multiple endpoints
- [ ] `docs/architecture.md` module map and dependency graph are updated
- [ ] `go test ./api-formatter-{name}/...` passes
- [ ] `go vet ./api-formatter-{name}/...` passes
