# Design Document — APilot Refactor

## Overview

This document describes the technical design for restructuring APilot into a multi-module, multi-language monorepo. The architecture separates concerns into three layers:

1. **Go engine layer** — `api-collector`, `api-formatter`, their implementations, `api-master`, and `apilot-cli`
2. **IDE integration layer** — `jetbrains-plugin` (unchanged behavior) and `vscode-plugin` (new)
3. **Shared model layer** — `ApiEndpoint` JSON contract that bridges collectors, formatters, and IDE extensions

---

## Module Designs

### 1. `api-collector` — Collector Interface Package

**Type:** Go module (interface-only, no executable)

**Purpose:** Defines the stable contract all collector implementations must satisfy.

**Key types:**

```go
// collector.go
package collector

type Collector interface {
    Name() string
    SupportedLanguages() []string
    Collect(ctx CollectContext) ([]ApiEndpoint, error)
}

type CollectContext struct {
    SourceDir  string            `json:"sourceDir"`
    Frameworks []string          `json:"frameworks,omitempty"` // hints
    Config     map[string]string `json:"config,omitempty"`
}
```

```go
// endpoint.go
package collector

type ApiEndpoint struct {
    Name        string            `json:"name"`
    Folder      string            `json:"folder,omitempty"`
    Description string            `json:"description,omitempty"`
    Tags        []string          `json:"tags,omitempty"`
    Path        string            `json:"path"`
    Method      string            `json:"method,omitempty"`   // empty for non-HTTP
    Protocol    string            `json:"protocol"`           // "http", "grpc", etc.
    Parameters  []ApiParameter    `json:"parameters,omitempty"`
    Headers     []ApiHeader       `json:"headers,omitempty"`
    RequestBody *ApiBody          `json:"requestBody,omitempty"`
    Response    *ApiBody          `json:"response,omitempty"`
    Metadata    map[string]any    `json:"metadata,omitempty"` // protocol-specific extensions
}

type ApiParameter struct {
    Name        string   `json:"name"`
    Type        string   `json:"type"`           // "text", "file"
    Required    bool     `json:"required"`
    In          string   `json:"in"`             // "query","path","header","cookie","body","form"
    Default     string   `json:"default,omitempty"`
    Description string   `json:"description,omitempty"`
    Example     string   `json:"example,omitempty"`
    Enum        []string `json:"enum,omitempty"`
}

type ApiHeader struct {
    Name        string `json:"name"`
    Value       string `json:"value,omitempty"`
    Description string `json:"description,omitempty"`
    Example     string `json:"example,omitempty"`
    Required    bool   `json:"required"`
}

type ApiBody struct {
    MediaType string `json:"mediaType,omitempty"` // "application/json", etc.
    Schema    any    `json:"schema,omitempty"`    // JSON Schema object
    Example   any    `json:"example,omitempty"`
}
```

```go
// json.go
package collector

import "encoding/json"

func MarshalEndpoints(endpoints []ApiEndpoint) ([]byte, error) { ... }
func UnmarshalEndpoints(data []byte) ([]ApiEndpoint, error) { ... }
```

**Design decisions:**
- `Protocol` field enables non-HTTP protocols (gRPC, WebSocket) without breaking the HTTP-centric fields.
- `Metadata map[string]any` absorbs protocol-specific data without polluting the core struct.
- JSON tags use `omitempty` on optional fields to keep wire format compact.

---

### 2. `api-formatter` — Formatter Interface Package

**Type:** Go module (interface-only, no executable)

**Key types:**

```go
// formatter.go
package formatter

import "github.com/tangcent/apilot/api-collector/collector"

type Formatter interface {
    Name() string
    SupportedFormats() []string
    Format(endpoints []collector.ApiEndpoint, opts FormatOptions) ([]byte, error)
}

type FormatOptions struct {
    Format string            `json:"format"`
    Config map[string]string `json:"config,omitempty"`
}
```

**Design decisions:**
- `Formatter` imports `api-collector` for the `ApiEndpoint` type — this is the only cross-module dependency at the interface level.
- Empty `endpoints` slice returns valid empty output (e.g., empty Markdown, empty Postman collection), never an error.

---

### 3. `api-master` — Core Engine

**Type:** Go module + executable

**Responsibilities:**
- CLI entry point with flag parsing
- In-process collector/formatter registry
- Plugin Registry loader (external binaries and shared libraries)
- Orchestration: collect → format → output

**Package layout:**

```
api-master/
├── main.go              # flag parsing, delegates to engine
├── engine/
│   ├── engine.go        # Run(cfg Config) error
│   └── registry.go      # Register/Lookup for collectors and formatters
├── plugin/
│   ├── registry.go      # loads plugins.json, populates engine registry
│   ├── manifest.go      # PluginManifest struct
│   ├── subprocess.go    # wraps external binary as Collector/Formatter
│   └── dynlib.go        # dlopen-based shared library loader
└── config/
    └── defaults.go      # default paths, env var overrides
```

**CLI interface:**

```
api-master [flags] <source-path>

Flags:
  --collector   string   Collector name (auto-detect if omitted)
  --formatter   string   Formatter name (default: markdown)
  --output      string   Output file path (default: stdout)
  --format      string   Format variant passed to formatter (e.g. simple, detailed)
  --plugin-registry string  Path to plugins.json (default: ~/.config/api-master/plugins.json)
  --list-collectors        Print registered collectors and exit
  --list-formatters        Print registered formatters and exit
```

**Engine flow:**

```
main.go
  └── engine.Run(cfg)
        ├── plugin.LoadRegistry(cfg.PluginRegistry)   // populates registry
        ├── registry.LookupCollector(cfg.Collector)
        │     └── if not found → subprocess.NewCollector(manifest)
        ├── collector.Collect(ctx)
        ├── registry.LookupFormatter(cfg.Formatter)
        │     └── if not found → subprocess.NewFormatter(manifest)
        ├── formatter.Format(endpoints, opts)
        └── write output to file or stdout
```

**Subprocess protocol:**

When `api-master` invokes an external collector binary:
1. Writes `CollectContext` JSON to the subprocess's stdin.
2. Reads `[]ApiEndpoint` JSON from the subprocess's stdout.
3. Any stderr output from the subprocess is forwarded to `api-master`'s stderr.

When `api-master` invokes an external formatter binary:
1. Writes a JSON envelope `{"endpoints": [...], "options": {...}}` to stdin.
2. Reads raw formatted bytes from stdout.

Exit code non-zero from subprocess is treated as an error.

**Plugin Registry (`plugins.json`) struct:**

```go
// plugin/manifest.go
type PluginManifest struct {
    Name    string   `json:"name"`
    Type    string   `json:"type"`    // "collector" | "formatter"
    Command string   `json:"command,omitempty"`
    Path    string   `json:"path,omitempty"`   // shared library path
    Args    []string `json:"args,omitempty"`
}

type PluginRegistry struct {
    Plugins []PluginManifest `json:"plugins"`
}
```

**Error handling:**
- Missing/unexecutable plugin → log warning, skip (do not fail startup).
- Unknown collector/formatter name → exit 1 with descriptive stderr message.
- Collection error → exit 1 with error details on stderr.

---

### 4. `apilot-cli` — Bundled CLI

**Type:** Go module + executable

**Purpose:** Statically links all collector and formatter modules so users get a single binary with no external plugin dependencies.

```go
// main.go
package main

import (
    "github.com/tangcent/apilot/api-master/engine"
    javacollector  "github.com/tangcent/apilot/api-collector-java"
    gocollector    "github.com/tangcent/apilot/api-collector-go"
    nodecollector  "github.com/tangcent/apilot/api-collector-node"
    pycollector    "github.com/tangcent/apilot/api-collector-python"
    mdfmt          "github.com/tangcent/apilot/api-formatter-markdown"
    curlfmt        "github.com/tangcent/apilot/api-formatter-curl"
    postmanfmt     "github.com/tangcent/apilot/api-formatter-postman"
)

func init() {
    engine.RegisterCollector(javacollector.New())
    engine.RegisterCollector(gocollector.New())
    engine.RegisterCollector(nodecollector.New())
    engine.RegisterCollector(pycollector.New())
    engine.RegisterFormatter(mdfmt.New())
    engine.RegisterFormatter(curlfmt.New())
    engine.RegisterFormatter(postmanfmt.New())
}

func main() {
    engine.RunCLI()
}
```

**Auto-detection logic** (in `engine.go`):
- Scan source directory for indicator files: `pom.xml`/`build.gradle` → java, `go.mod` → go, `package.json` → node, `requirements.txt`/`pyproject.toml` → python.
- First match wins; if ambiguous, prefer the collector whose `SupportedLanguages()` matches detected file extensions.

---

### 5. Collector Implementations

Each collector follows the same structural pattern:

```
api-collector-<lang>/
├── go.mod
├── collector.go      # New() constructor, implements Collector interface
└── <framework>/
    └── parser.go     # framework-specific AST/regex parsing logic
```

**Java collector (`api-collector-java`):**
- Uses `go/ast` is not applicable for Java; instead uses a Java source parser library (e.g., `github.com/nicholasgasior/gsfmt` or a custom regex+heuristic parser, or invokes `javap`/`javac` via subprocess).
- Preferred approach: invoke a bundled `maven-indexer-cli` subprocess for dependency resolution; parse source files with a lightweight Java grammar (e.g., `antlr4` generated parser or `tree-sitter-java` bindings).
- Extracts: `@RestController`, `@RequestMapping`, `@GetMapping`, `@PostMapping`, `@PathVariable`, `@RequestParam`, `@RequestBody`, Javadoc.

**Go collector (`api-collector-go`):**
- Uses Go's standard `go/ast` and `go/parser` packages — no external dependency needed.
- Walks AST for `gin.RouterGroup.GET/POST/...`, `echo.Echo.GET/POST/...`, `fiber.App.Get/Post/...` call expressions.
- Extracts route path string literals and associated handler function doc comments.

**Node.js collector (`api-collector-node`):**
- Invokes `node` or `ts-node` subprocess, or uses a Go-based TypeScript/JS parser (e.g., `tree-sitter` with `tree-sitter-typescript`).
- Preferred: `tree-sitter-typescript` Go bindings for zero external runtime dependency.
- Extracts Express/Fastify route registrations and NestJS decorators.

**Python collector (`api-collector-python`):**
- Uses `tree-sitter-python` Go bindings.
- Extracts FastAPI/Flask decorators and Django REST `urlpatterns`.

---

### 6. Formatter Implementations

**Markdown (`api-formatter-markdown`):**
- Uses Go `text/template` with two embedded templates: `simple.md.tmpl` and `detailed.md.tmpl`.
- `simple`: one section per endpoint, method + path headline, parameter list.
- `detailed`: full schema expansion, request/response body as JSON code block.
- Template selection driven by `FormatOptions.Config["format"]` (`"simple"` default).

**cURL (`api-formatter-curl`):**
- Pure string building, no template needed.
- Output: one `curl` command per endpoint, separated by blank lines.
- Query params appended to URL; body as `-d '{...}'`; headers as `-H 'Name: Value'`.

**Postman (`api-formatter-postman`):**
- Builds Postman Collection v2.1 JSON using typed Go structs in `model/` sub-package.
- Groups endpoints by `ApiEndpoint.Folder` into Postman `ItemGroup` folders.
- Uses `encoding/json` for final serialization.

```
api-formatter-postman/model/
├── collection.go   # Collection, Info, Item, ItemGroup
├── request.go      # Request, URL, Header, Body
└── response.go     # Response (example responses)
```

---

### 7. `vscode-plugin` — VSCode Extension

**Type:** TypeScript, VSCode Extension API

**Package layout:**

```
vscode-plugin/
├── package.json          # contributes commands, settings, activationEvents
├── tsconfig.json
├── src/
│   ├── extension.ts      # activate(), registers commands
│   ├── runner.ts         # ExportRunner: builds args, spawns process, streams output
│   ├── binaryResolver.ts # resolves platform binary path
│   └── settings.ts       # typed wrapper around vscode.workspace.getConfiguration
└── bin/                  # bundled apilot-cli binaries (gitignored, added at release)
```

**Command:** `apilot.export`
- Triggered via right-click context menu on files/folders.
- Reads settings: `apilot.formatter`, `apilot.outputDestination` (`"channel"` | `"file"`), `apilot.binaryPath`.
- Resolves binary via `binaryResolver.ts` (custom path → bundled platform binary).
- Spawns `apilot-cli` as child process, pipes stdout/stderr.
- On success: writes to output channel or file.
- On failure: shows `vscode.window.showErrorMessage` with stderr content.

**Binary resolution logic (`binaryResolver.ts`):**

```typescript
export function resolveBinary(config: Settings): string {
    if (config.binaryPath) return config.binaryPath;
    const platform = process.platform;   // "linux" | "darwin" | "win32"
    const arch = process.arch;           // "x64" | "arm64"
    const suffix = platform === "win32" ? ".exe" : "";
    const name = `apilot-cli-${platform}-${arch}${suffix}`;
    return path.join(__dirname, "..", "bin", name);
}
```

**Settings (`package.json` contributions):**

| Setting | Type | Default | Description |
|---|---|---|---|
| `apilot.formatter` | enum | `"markdown"` | Output format |
| `apilot.outputDestination` | enum | `"channel"` | Where to write output |
| `apilot.outputFile` | string | `""` | File path when destination is "file" |
| `apilot.binaryPath` | string | `""` | Custom binary path (overrides bundled) |

---

### 8. `jetbrains-plugin` — JetBrains Plugin (Preserved)

**No architectural changes.** The existing source tree moves into `jetbrains-plugin/` with the following adjustments:

- `build.gradle.kts`, `gradle.properties`, `settings.gradle.kts`, `gradlew`/`gradlew.bat` move into `jetbrains-plugin/`.
- `src/` moves into `jetbrains-plugin/src/` unchanged.
- All existing service classes, action classes, PSI parsers, exporters, and settings remain intact.
- The module has zero runtime dependency on any Go module.

**Build artifact:** `jetbrains-plugin/build/distributions/apilot-<version>.zip`

---

## Cross-Cutting Concerns

### Build System

| Module | Build Tool | Artifact |
|---|---|---|
| `api-collector` | `go build` | Go package (no binary) |
| `api-formatter` | `go build` | Go package (no binary) |
| `api-master` | `go build` | Binary per platform |
| `apilot-cli` | `go build` | Binary per platform (5 targets) |
| `api-collector-*` | `go build` | Go package (no binary) |
| `api-formatter-*` | `go build` | Go package (no binary) |
| `vscode-plugin` | `npm run compile` | VSIX package |
| `jetbrains-plugin` | `./gradlew buildPlugin` | `.zip` plugin artifact |

Cross-platform Go builds use `GOOS`/`GOARCH` env vars. A top-level `Makefile` or GitHub Actions matrix handles the release matrix.

### CI Pipelines

- `ci-go.yml`: matrix over all Go modules, runs `go test ./...` and `go vet ./...`.
- `ci-jetbrains.yml`: runs `./gradlew test buildPlugin`.
- `ci-vscode.yml`: runs `npm ci && npm run compile && npm test`.

### Versioning

- Go modules use semantic versioning via Git tags (`api-collector/v1.0.0`, etc.).
- `apilot-cli` version is the canonical release version embedded in all binaries via `ldflags`.
- `vscode-plugin` `package.json` version matches `apilot-cli` version.
- `jetbrains-plugin` version is managed independently in `gradle.properties`.

### Error Propagation

```
Collector error  →  engine logs to stderr, exits 1
Formatter error  →  engine logs to stderr, exits 1
Plugin load warn →  engine logs warning, continues
VSCode error     →  showErrorMessage with stderr text
JetBrains error  →  existing IdeaConsole + NotificationUtils (unchanged)
```

---

## Data Flow Diagram

```
[Source Code]
     │
     ▼
[Collector] ──── CollectContext ────► Collect()
     │
     │  []ApiEndpoint (JSON over stdin/stdout for subprocess plugins)
     ▼
[api-master engine]
     │
     │  []ApiEndpoint + FormatOptions
     ▼
[Formatter] ──────────────────────► Format()
     │
     │  []byte (Markdown / cURL / Postman JSON)
     ▼
[Output: stdout | file | VSCode channel]
```

---

## Open Questions

1. Java source parsing strategy: pure Go parser vs. subprocess `javac`/`tree-sitter-java` — needs a spike to evaluate accuracy vs. complexity trade-off.
2. Node.js collector: `tree-sitter` Go bindings require CGO; evaluate whether a pure-Go JS/TS parser (e.g., `goja` AST) is sufficient for route extraction.
3. Shared library (`dynlib.go`) support: CGO required for `dlopen`; consider deferring to v2 and shipping subprocess-only plugin support in v1.
4. `apilot-cli` binary size: static linking all collectors may produce a large binary; evaluate `upx` compression for the release artifacts.
