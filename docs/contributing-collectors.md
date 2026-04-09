# Contributing a New Collector

This guide walks you through adding a new language/framework collector to APilot. A Collector parses source code for a specific language and produces `[]ApiEndpoint`.

---

## Prerequisites

- Go 1.22 or higher
- Familiarity with the [architecture guide](architecture.md)
- Read the [main contributing guide](../CONTRIBUTING.md)

---

## Step 1 — Create the module directory

Create a new directory at the repo root following the naming convention `api-collector-{lang}/`:

```bash
mkdir api-collector-rust
```

Use a short, lowercase language name (e.g. `rust`, `go`, `java`, `python`).

---

## Step 2 — Add `go.mod`

Create `api-collector-rust/go.mod`:

```
module github.com/tangcent/apilot/api-collector-rust

go 1.22

require github.com/tangcent/apilot/api-collector v0.0.0
```

Every collector depends on `api-collector` for the `Collector` interface, `CollectContext`, and the `ApiEndpoint` model.

---

## Step 3 — Implement the `Collector` interface

Create `api-collector-rust/collector.go` and implement the three methods defined in [`collector.Collector`](../api-collector/collector.go):

```go
package rustcollector

import "github.com/tangcent/apilot/api-collector/collector"

type RustCollector struct{}

func New() collector.Collector { return &RustCollector{} }

func (c *RustCollector) Name() string { return "rust" }

func (c *RustCollector) SupportedLanguages() []string { return []string{"rust"} }

func (c *RustCollector) Collect(ctx collector.CollectContext) ([]collector.ApiEndpoint, error) {
	// TODO: walk source tree and extract endpoints
	return nil, nil
}
```

### Interface contract

| Method | Requirement |
|--------|-------------|
| `Name()` | Return a unique lowercase identifier (e.g. `"rust"`). Must not collide with existing collectors. |
| `SupportedLanguages()` | Return the list of language identifiers this collector handles (e.g. `["rust"]`). |
| `Collect(ctx)` | Parse the source directory described by `ctx` and return discovered endpoints. **Return `nil, nil` (not an error) when no endpoints are found.** |

### Key rules for `Collect`

1. **No endpoints found** — Return `nil, nil`. Do not return an error just because the source tree has no API endpoints.
2. **Unparseable files** — Skip them with a log warning. Do not fail the whole collection.
3. **Protocol** — Always populate `ApiEndpoint.Protocol`. Use `"http"` for REST endpoints.
4. **Folder** — Use `ApiEndpoint.Folder` to group related endpoints (maps to Postman folders / Markdown sections).
5. **Frameworks** — Use `ctx.Frameworks` as a hint to limit which framework parsers run. If empty, run all.

---

## Step 4 — Add framework sub-packages

Most languages have multiple web frameworks. Create a sub-package for each framework under `{framework}/parser.go`:

```
api-collector-rust/
├── collector.go          # Top-level collector: dispatches to framework parsers
├── go.mod
├── actix/
│   └── parser.go         # Actix-web route extraction
└── rocket/
    └── parser.go         # Rocket route extraction
```

Each framework parser exports a `Parse` function:

```go
// actix/parser.go
package actix

import "github.com/tangcent/apilot/api-collector/collector"

func Parse(sourceDir string) ([]collector.ApiEndpoint, error) {
	// TODO: parse Actix-web route macros
	return nil, nil
}
```

The top-level `Collect` method dispatches to the appropriate framework parser:

```go
func (c *RustCollector) Collect(ctx collector.CollectContext) ([]collector.ApiEndpoint, error) {
	var all []collector.ApiEndpoint

	frameworks := ctx.Frameworks
	if len(frameworks) == 0 {
		frameworks = []string{"actix", "rocket"}
	}

	for _, fw := range frameworks {
		var eps []collector.ApiEndpoint
		var err error

		switch fw {
		case "actix":
			eps, err = actix.Parse(ctx.SourceDir)
		case "rocket":
			eps, err = rocket.Parse(ctx.SourceDir)
		default:
			continue
		}

		if err != nil {
			return nil, err
		}
		all = append(all, eps...)
	}

	return all, nil
}
```

---

## Step 5 — Export the `New()` constructor

Your package must export a `New() collector.Collector` function. This is the single entry point used during registration:

```go
func New() collector.Collector { return &RustCollector{} }
```

---

## Step 6 — Register in `apilot-cli/main.go`

Add an import and a registration call in [`apilot-cli/main.go`](../apilot-cli/main.go):

```go
import (
	// ... existing imports ...
	rustcollector "github.com/tangcent/apilot/api-collector-rust"
)

func init() {
	// ... existing registrations ...
	engine.RegisterCollector(rustcollector.New())
}
```

Also add a detection entry in the `detectCollector` function in [`api-master/engine/engine.go`](../api-master/engine/engine.go) if the language has a well-known indicator file:

```go
{"Cargo.toml", "rust"},
```

---

## Step 7 — Add fixture test data and unit tests

Create a test fixture directory and write unit tests:

```
api-collector-rust/
├── collector.go
├── collector_test.go
├── go.mod
├── actix/
│   ├── parser.go
│   └── parser_test.go
├── rocket/
│   ├── parser.go
│   └── parser_test.go
└── testdata/
    └── actix/
        └── main.rs
```

### Test file: `collector_test.go`

```go
package rustcollector

import (
	"testing"

	"github.com/tangcent/apilot/api-collector/collector"
)

func TestName(t *testing.T) {
	c := &RustCollector{}
	if c.Name() != "rust" {
		t.Fatalf("expected name 'rust', got %q", c.Name())
	}
}

func TestSupportedLanguages(t *testing.T) {
	c := &RustCollector{}
	langs := c.SupportedLanguages()
	if len(langs) == 0 {
		t.Fatal("expected at least one supported language")
	}
}

func TestCollect_EmptyDir(t *testing.T) {
	c := &RustCollector{}
	endpoints, err := c.Collect(collector.CollectContext{
		SourceDir:  "testdata/empty",
		Frameworks: []string{"actix"},
	})
	if err != nil {
		t.Fatalf("empty dir should not error: %v", err)
	}
	if endpoints != nil {
		t.Fatalf("expected nil endpoints for empty dir, got %d", len(endpoints))
	}
}

func TestCollect_NoEndpoints(t *testing.T) {
	c := &RustCollector{}
	endpoints, err := c.Collect(collector.CollectContext{
		SourceDir:  "testdata/no_routes",
		Frameworks: []string{"actix"},
	})
	if err != nil {
		t.Fatalf("no endpoints should not error: %v", err)
	}
	if endpoints != nil {
		t.Fatalf("expected nil endpoints, got %d", len(endpoints))
	}
}
```

### Required test cases

| Test case | What it verifies |
|-----------|-----------------|
| Empty source directory | `Collect` returns `nil, nil`, not an error |
| Source with no routes | `Collect` returns `nil, nil` |
| Source with routes | Correct `ApiEndpoint` values are extracted |
| Framework filtering | Only requested framework parsers run |
| Name / SupportedLanguages | Interface methods return correct values |

### Fixture data

Place sample source files under `testdata/`. Use minimal, realistic examples:

```rust
// testdata/actix/main.rs
use actix_web::{web, App, HttpServer};

async fn get_users() -> &'static str {
    "[]"
}

async fn create_user() -> &'static str {
    "{}"
}

#[actix_web::main]
async fn main() -> std::io::Result<()> {
    HttpServer::new(|| {
        App::new()
            .route("/users", web::get().to(get_users))
            .route("/users", web::post().to(create_user))
    })
    .bind("127.0.0.1:8080")?
    .run()
    .await
}
```

---

## Step 8 — Update `docs/architecture.md`

Add your collector to the **Module Map** table and the **Module Dependency Graph** in [`docs/architecture.md`](architecture.md):

**Module Map** — add a row:

```
| `api-collector-rust` | Go | Rust collector (Actix, Rocket) |
```

**Module Dependency Graph** — add under `apilot-cli`:

```
├── api-collector-rust
```

And add the dependency block:

```
api-collector-rust
  └── api-collector
```

---

## Complete working example — Go collector

Here is a real, minimal collector from the codebase ([`api-collector-go/collector.go`](../api-collector-go/collector.go)):

```go
package gocollector

import "github.com/tangcent/apilot/api-collector/collector"

type GoCollector struct{}

func New() collector.Collector { return &GoCollector{} }

func (c *GoCollector) Name() string { return "go" }

func (c *GoCollector) SupportedLanguages() []string { return []string{"go"} }

func (c *GoCollector) Collect(ctx collector.CollectContext) ([]collector.ApiEndpoint, error) {
	// 1. Walk .go files under ctx.SourceDir using go/parser
	// 2. Delegate to gin/, echo/, fiber/ sub-packages
	return nil, nil
}
```

And a framework sub-package ([`api-collector-go/gin/parser.go`](../api-collector-go/gin/parser.go)):

```go
package gin

import "github.com/tangcent/apilot/api-collector/collector"

func Parse(sourceDir string) ([]collector.ApiEndpoint, error) {
	// Walk the AST for gin.RouterGroup.GET/POST/PUT/DELETE/PATCH/HEAD/OPTIONS calls
	return nil, nil
}
```

The Go collector currently supports three frameworks — Gin, Echo, and Fiber — each with its own `parser.go` under the framework sub-directory.

---

## Checklist

Before submitting your PR, verify:

- [ ] Module directory is named `api-collector-{lang}/`
- [ ] `go.mod` declares dependency on `api-collector`
- [ ] `Collector` interface is fully implemented (`Name`, `SupportedLanguages`, `Collect`)
- [ ] `Collect` returns `nil, nil` (not an error) when no endpoints are found
- [ ] Framework sub-packages are under `{framework}/parser.go`
- [ ] `New() collector.Collector` constructor is exported
- [ ] Collector is registered in `apilot-cli/main.go`
- [ ] Detection entry is added in `api-master/engine/engine.go` (if applicable)
- [ ] Fixture test data and unit tests are included
- [ ] `docs/architecture.md` module map and dependency graph are updated
- [ ] `go test ./api-collector-{lang}/...` passes
- [ ] `go vet ./api-collector-{lang}/...` passes
