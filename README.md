# APilot

> Navigate your APIs. Automatically.

APilot scans your source code, extracts API endpoints, and exports them to the format you need — Postman collections, Markdown docs, cURL commands, and more. No annotations required, no runtime needed.

[![CI](https://github.com/tangcent/apilot/actions/workflows/ci.yml/badge.svg)](https://github.com/tangcent/apilot/actions/workflows/ci.yml)
[![codecov](https://codecov.io/gh/tangcent/apilot/branch/master/graph/badge.svg)](https://codecov.io/gh/tangcent/apilot)
[![npm version](https://img.shields.io/npm/v/@tangcent/apilot.svg)](https://www.npmjs.com/package/@tangcent/apilot)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

1 AI Agent [Skill](./skills/)

| Skill | Description |
|-------|-------------|
| `apilot` | Scan source code and export API endpoints to Postman, Markdown, cURL, and more |

---

## What it does

Point APilot at a source directory and it figures out the rest:

```bash
apilot export ./my-service --formatter postman --output collection.json
apilot export ./my-service --formatter markdown --format detailed
apilot export ./my-service --formatter curl
apilot export ./my-service --formatter json --output api.json
```

It detects your language and framework automatically, walks the source tree, and produces clean, structured API output — ready to import into Postman, share as docs, or drop into a CI pipeline.

---

## Supported languages & frameworks

| Language   | Frameworks                              |
|------------|-----------------------------------------|
| Java       | Spring MVC, JAX-RS, Feign               |
| Kotlin     | Spring MVC                              |
| Go         | Gin, Echo, Fiber                        |
| Node.js    | Express, Fastify, NestJS                |
| Python     | FastAPI, Django REST, Flask             |

---

## Output formats

| Format   | Description                              |
|----------|------------------------------------------|
| markdown | Markdown docs (simple or detailed)       |
| curl     | One `curl` command per endpoint          |
| postman  | Postman Collection v2.1 JSON             |
| json     | Machine-readable endpoint JSON for CI    |

---

## Quick Start (AI Agent)

> The following steps are for AI Agents (Claude Code, Cursor, etc.).

**Step 1 — Install**

```bash
# Install CLI
npm install -g @tangcent/apilot

# Install CLI SKILL (required)
npx skills add tangcent/apilot -y -g
```

**Step 2 — Export APIs**

```bash
# Scan source code and export to Postman
apilot export ./my-service --formatter postman --output api.postman_collection.json

# Generate Markdown documentation
apilot export ./my-service --formatter markdown --format detailed --output API.md

# Generate cURL commands
apilot export ./my-service --formatter curl
```

**Step 3 — Verify**

```bash
apilot export --list-collectors
apilot export --list-formatters
```

---

## Installation

### npm (recommended)

```bash
npm install -g @tangcent/apilot
```

### Download binary

Grab the latest release for your platform from the [releases page](https://github.com/tangcent/apilot/releases):

```bash
# macOS (Apple Silicon)
curl -L https://github.com/tangcent/apilot/releases/latest/download/apilot-darwin-arm64 -o apilot
chmod +x apilot && sudo mv apilot /usr/local/bin/
```

### Build from source

```bash
git clone https://github.com/tangcent/apilot.git
cd apilot
go build -o apilot ./apilot-cli
```

### Install from source (local development)

Use the install script to build and install `apilot` to your local PATH — handy for testing changes before release:

```bash
# Clone and enter the repo
git clone https://github.com/tangcent/apilot.git
cd apilot

# Build and install to /usr/local/bin (or ~/.local/bin if no write access)
./scripts/install-local.sh

# Or specify a custom install directory
./scripts/install-local.sh /usr/local/bin

# Verify
apilot --version
```

After making code changes, just re-run `./scripts/install-local.sh` to rebuild and reinstall.

---

## Usage

```
apilot export <source-path> [flags]

  source-path can be a directory or a single source file.
  When a file is given, the project root is auto-detected by walking up
  to find pom.xml, build.gradle, go.mod, package.json, etc.
  For multi-module projects, the topmost directory with an indicator is used.

Flags:
  --collector      string   Collector name (auto-detected if omitted)
  --formatter      string   Output format: markdown, curl, postman, json (default: markdown)
  --format         string   Format variant, e.g. simple, detailed (default: simple)
  --method         string   Filter to a specific method name (used with file-level export)
  --project-root   string   Override auto-detected project root directory
  --no-deps                 Disable resolving request/response types from dependencies
  --allow-empty             Exit 0 when the collection yields no endpoints (default: fail)
  --output         string   Output file path (default: stdout)
  --list-collectors         Print available collectors and exit
  --list-formatters         Print available formatters and exit
```

### Examples

```bash
# Export a Spring Boot project to Postman
apilot export ./backend --formatter postman --output api.postman_collection.json

# Generate detailed Markdown docs
apilot export ./backend --formatter markdown --format detailed --output API.md

# Quick cURL reference to stdout
apilot export ./backend --formatter curl

# Export all APIs from a specific file
apilot export UserController.java --formatter markdown

# Export a single method from a file
apilot export UserController.java --method getUser --formatter curl

# Multi-module project: override project root
apilot export user-service/src/.../UserController.java --project-root ./company-platform

# Machine-readable export for a CI gate
apilot export ./backend --formatter json --output api.json
```

### Dependency type resolution

Beyond the project's own sources, APilot expands request and response types
that are declared in your dependencies — a shared common module, a third-party
library, a framework base class. Each language reaches those types through its
own environment:

| Language | Prerequisite | What happens without it |
|----------|--------------|--------------------------|
| Java     | [`maven-indexer-cli`](https://github.com/tangcent/maven-indexer-cli) on PATH | Resolution is skipped; a warning names the tool and the number of declared dependencies affected |
| Go       | `go` toolchain on PATH (reads the module cache) | Resolution is skipped; a warning names the missing toolchain |
| Node.js  | `npm install` run (reads `node_modules`) | Resolution is skipped; a warning says to install first |
| Python   | A discoverable environment: project `.venv`/`venv`/`env`, `CONDA_PREFIX`, or `python` on PATH | Resolution is skipped; a warning says which environment to activate |

When resolution runs, APilot reports how many types were expanded from
dependencies (`info: resolved N types from dependencies`). Use `--no-deps` to
turn the capability off explicitly — useful in CI where the prerequisite is
known to be absent.

```bash
apilot export ./backend --formatter markdown --no-deps
```

### Empty exports

A run that yields no endpoints fails instead of writing a document that holds
only its heading. The failure names the collector and the path it looked at, so
an unsupported framework or a filter that matched nothing is visible rather than
silent:

```
endpoints: 0
error: collector "java" extracted 0 endpoints from "./backend" (pass --allow-empty to export an empty document)
```

Every run reports the exported endpoint count on stderr, so a pipeline can assert
the number from logs alone:

```bash
apilot export ./backend --output API.md 2>&1 | grep -qx 'endpoints: 244'
```

When an empty document is the intended result — a service that legitimately has
no endpoints yet — pass `--allow-empty` to keep the run at exit 0.

### Three levels of granularity

| Command | Scope |
|---------|-------|
| `apilot export ./project` | All APIs in project |
| `apilot export UserController.java` | All APIs in that file |
| `apilot export UserController.java --method getUser` | One specific API |

### Machine-readable JSON

`--formatter json` writes one document a pipeline can diff and gate on, instead
of counting headings in markdown:

```bash
apilot export ./backend --formatter json --output api.json
apilot export ./backend --formatter json | jq '.endpoints | length'
```

The output depends only on the collected endpoints — no timestamps, no absolute
paths, no host-specific values — so the same input produces byte-identical
output on every machine, and endpoints are sorted rather than left in whatever
order the source tree was walked.

```json
{
  "endpoints": [
    {
      "name": "getUser",
      "method": "GET",
      "path": "/users/:id",
      "protocol": "http",
      "description": "Returns a single user.",
      "folder": "UserController",
      "request": {
        "parameters": {
          "path": [
            { "name": "id", "type": "text", "required": true }
          ],
          "query": [
            { "name": "verbose", "type": "text", "required": false, "default": "false" }
          ]
        },
        "body": {
          "mediaType": "application/json",
          "schema": {
            "kind": "object",
            "fields": [
              { "name": "name", "kind": "single", "type": "string", "required": true, "description": "display name" }
            ]
          }
        }
      },
      "response": {
        "mediaType": "application/json",
        "schema": {
          "kind": "object",
          "fields": [
            { "name": "id", "kind": "single", "type": "long", "required": true },
            { "name": "roles", "kind": "array", "items": { "kind": "single", "type": "string" } }
          ]
        }
      }
    }
  ]
}
```

**Endpoint order.** `endpoints` is sorted by `folder`, then `name`, `method` and
`path`; endpoints that belong to no folder come last.

**Parameters** are grouped by location under `request.parameters`:

| Group | Location | Notes |
|-------|----------|-------|
| `path` | path variable | |
| `query` | query string | |
| `header` | request header | headers declared outside the parameter list are merged here |
| `cookie` | cookie | |
| `form` | form field | `type` is `file` for uploads |
| `body` | raw body parameter | separate from the `request.body` schema |

A group with no parameter is omitted. Order within a group is the order the
collector reported. Each parameter carries `name`, `type`, `required`, and —
when the source declares them — `default`, `value`, `description`, `example`
and `enum`.

**Schema** is a tree of nodes, one node per value, shared by request and
response bodies:

| `kind` | Meaning | Keys besides `kind` |
|--------|---------|---------------------|
| `single` | primitive | `type`: `string`, `int`, `long`, `float`, `double`, `boolean`, `null` |
| `ref` | named type left unexpanded | `type`: the type name |
| `object` | object | `fields`: member nodes |
| `array` | array | `items`: element node |
| `map` | map | `key`, `value`: nodes |

A member of an object is the same node plus `name`, and the documentation that
belongs to that member: `required`, `description`, `default`, `demo`, `generic`,
`options`. Optional keys are omitted when empty, so a field is required only
when `required` is `true`.

Object members are sorted by name. The collected model keeps fields in a map, so
declaration order is already lost before a formatter runs; sorting is the only
order that is stable across runs and machines. A type that refers back to itself
terminates as a `ref` node instead of expanding forever.

---

## IDE integrations

- VSCode — via the [APilot VSCode extension](https://marketplace.visualstudio.com/items?itemName=tangcent.apilot) (right-click any folder to export)
- JetBrains — see [easy-api](https://github.com/tangcent/easy-api) for the IntelliJ plugin

---

## Extending APilot

APilot has a plugin system. Any binary that speaks the stdin/stdout JSON protocol can be registered as an external collector or formatter — no recompilation needed.

```json
// ~/.config/apilot/plugins.json
{
  "plugins": [
    {
      "name": "rust",
      "type": "collector",
      "command": "apilot-collector-rust"
    }
  ]
}
```

See [docs/plugin-protocol.md](docs/plugin-protocol.md) for the full protocol spec.

---

## Architecture

APilot is a multi-module Go monorepo with a clean three-layer design:

```
apilot-cli  (bundled binary)
  └── api-master  (engine + plugin runtime)
        ├── api-collector-{java,go,node,python}
        └── api-formatter-{markdown,curl,json,postman}
```

The `api-collector` and `api-formatter` packages define the stable interfaces. Everything else is an implementation. See [docs/architecture.md](docs/architecture.md) for the full breakdown.

---

## Contributing

PRs and issues are welcome. See [CONTRIBUTING.md](CONTRIBUTING.md) to get started.

---

## License

[MIT](LICENSE)
