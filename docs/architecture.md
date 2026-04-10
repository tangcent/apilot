# APilot — Architecture Guide

> This document is the primary reference for contributors. Read it before implementing any module.

## Overview

APilot is a multi-module monorepo that parses API source code and exports documentation in multiple formats. The architecture has three layers:

```
┌─────────────────────────────────────────────────────────────┐
│                     IDE Integration Layer                    │
│   jetbrains-plugin (Kotlin/Gradle)  │  vscode-plugin (TS)   │
└──────────────────────┬──────────────────────────────────────┘
                       │ subprocess (vscode only)
┌──────────────────────▼──────────────────────────────────────┐
│                      Go Engine Layer                         │
│  apilot-cli  →  api-master  →  api-collector / api-formatter│
└──────────────────────┬──────────────────────────────────────┘
                       │ implements
┌──────────────────────▼──────────────────────────────────────┐
│                   Collector / Formatter Modules              │
│  api-collector-{java,go,node,python}                 │
│  api-formatter-{markdown,curl,postman}                        │
└─────────────────────────────────────────────────────────────┘
```

---

## Module Map

| Module | Language | Role |
|--------|----------|------|
| `api-collector` | Go | Collector interface + ApiEndpoint model |
| `api-formatter` | Go | Formatter interface + FormatOptions |
| `api-master` | Go | Core engine: CLI, registry, plugin loader, orchestration |
| `apilot-cli` | Go | Bundled CLI: statically links all collectors + formatters |
| `api-collector-java` | Go | Java/Kotlin collector (Spring MVC, JAX-RS, Feign) |
| `api-collector-go` | Go | Go collector (Gin, Echo, Fiber) |
| `api-collector-node` | Go | Node.js collector (Express, Fastify, NestJS) |
| `api-collector-python` | Go | Python collector (FastAPI, Django REST, Flask) |
| `api-formatter-markdown` | Go | Markdown formatter (simple + detailed templates) |
| `api-formatter-curl` | Go | cURL command formatter |
| `api-formatter-postman` | Go | Postman Collection v2.1 formatter |
| `vscode-plugin` | TypeScript | VSCode extension: invokes apilot-cli as subprocess |
| `jetbrains-plugin` | Kotlin | IntelliJ plugin: PSI-based, no Go dependency |

---

## Module Dependency Graph

```
apilot-cli
  ├── api-master (engine + plugin runtime)
  ├── api-collector-java
  ├── api-collector-go
  ├── api-collector-node
  ├── api-collector-python
  ├── api-formatter-markdown
  ├── api-formatter-curl
  └── api-formatter-postman

api-master
  ├── api-collector  (interface only)
  └── api-formatter   (interface only)

api-collector-{java,go,node,python}
  └── api-collector

api-formatter-{markdown,curl,postman}
  ├── api-collector  (for ApiEndpoint type)
  └── api-formatter

vscode-plugin
  └── apilot-cli  (bundled binary, no Go import)

jetbrains-plugin
  └── (no dependency on any Go module)
```

**Rule:** No module may import a module above it in the graph. `api-collector` and `api-formatter` are the only shared contracts.

---

## Data Flow

```
[Source Code]
     │
     ▼
[Collector.Collect(CollectContext)]
     │
     │  []ApiEndpoint
     ▼
[api-master engine]
     │
     │  []ApiEndpoint + FormatOptions
     ▼
[Formatter.Format(...)]
     │
     │  []byte
     ▼
[stdout | file | VSCode output channel]
```

For subprocess plugins, `CollectContext` is written as JSON to the subprocess stdin, and `[]ApiEndpoint` JSON is read from stdout. See [plugin-protocol.md](plugin-protocol.md) for the full protocol spec.

---

## Implementing a New Collector

1. Create a new module directory: `api-collector-<lang>/`
2. Add `go.mod` with module path `github.com/tangcent/apilot/api-collector-<lang>`
3. Declare dependency on `github.com/tangcent/apilot/api-collector`
4. Create `collector.go` with a struct implementing `collector.Collector`:
   - `Name() string` — unique lowercase identifier (e.g. `"rust"`)
   - `SupportedLanguages() []string` — language identifiers (e.g. `["rust"]`)
   - `Collect(ctx collector.CollectContext) ([]collector.ApiEndpoint, error)`
5. Add sub-packages per framework under `<framework>/parser.go`
6. Export a `New() collector.Collector` constructor
7. Register in `apilot-cli/main.go`: `engine.RegisterCollector(rustcollector.New())`

### Collector contract

- Return `nil, nil` (not an error) when no endpoints are found.
- Skip unparseable files with a log warning; do not fail the whole collection.
- Populate `ApiEndpoint.Protocol` — use `"http"` for REST endpoints.
- Use `ApiEndpoint.Folder` to group related endpoints (maps to Postman folders / Markdown sections).

---

## Implementing a New Formatter

1. Create a new module directory: `api-formatter-<name>/`
2. Add `go.mod` with module path `github.com/tangcent/apilot/api-formatter-<name>`
3. Declare dependencies on `api-collector` and `api-formatter`
4. Create `formatter.go` with a struct implementing `formater.Formatter`:
   - `Name() string` — unique lowercase identifier (e.g. `"openapi"`)
   - `SupportedFormats() []string` — format variant names
   - `Format(endpoints []collector.ApiEndpoint, opts formater.FormatOptions) ([]byte, error)`
5. Export a `New() formater.Formatter` constructor
6. Register in `apilot-cli/main.go`: `engine.RegisterFormatter(openapifmt.New())`

### Formatter contract

- An empty `endpoints` slice MUST return valid empty output, never an error.
- Use `opts.Format` to select output variant; default to the first supported format.
- Use `opts.Config` for formatter-specific options.

---

## External Plugin (subprocess)

Any binary that speaks the stdin/stdout JSON protocol can be registered as a plugin without recompiling `apilot-cli`. See [plugin-protocol.md](plugin-protocol.md).

Register in `~/.config/apilot/plugins.json`:

```json
{
  "plugins": [
    {
      "name": "rust",
      "type": "collector",
      "command": "api-collector-rust",
      "args": []
    }
  ]
}
```

---

## Build & Release

### Go modules

```bash
# Build apilot for all platforms
GOOS=linux   GOARCH=amd64 go build -o bin/apilot-linux-amd64   ./apilot-cli
GOOS=linux   GOARCH=arm64 go build -o bin/apilot-linux-arm64   ./apilot-cli
GOOS=darwin  GOARCH=amd64 go build -o bin/apilot-darwin-amd64  ./apilot-cli
GOOS=darwin  GOARCH=arm64 go build -o bin/apilot-darwin-arm64  ./apilot-cli
GOOS=windows GOARCH=amd64 go build -o bin/apilot-windows-amd64.exe ./apilot-cli
```

### VSCode extension

```bash
cd vscode-plugin
npm ci
npm run compile
# Copy platform binaries into vscode-plugin/bin/ before packaging
npx vsce package
```

### JetBrains plugin

```bash
cd jetbrains-plugin
./gradlew buildPlugin
# Artifact: build/distributions/apilot-<version>.zip
```

---

## CI Pipelines

| Workflow | Trigger | What it does |
|----------|---------|--------------|
| `.github/workflows/ci.yml` | push / PR | `go test ./...` + `go vet ./...` for all Go modules |
| `.github/workflows/co.yml` | push / PR | Coverage report upload to Codecov |

---

## Open Questions

1. **Java parsing strategy** — pure Go parser vs. `tree-sitter-java` CGO bindings vs. subprocess `javac`. Needs a spike.
2. **Node.js collector** — `tree-sitter-typescript` requires CGO. Evaluate pure-Go JS/TS AST (e.g. `goja`) for route extraction.
3. **Shared library plugins** (`dynlib.go`) — requires CGO (`dlopen`). Deferred to v2; v1 ships subprocess-only plugin support.
4. **Binary size** — static linking all collectors may produce a large `apilot-cli`. Evaluate `upx` compression for release artifacts.
