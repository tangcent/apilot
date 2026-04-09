# APilot

> Navigate your APIs. Automatically.

APilot scans your source code, extracts API endpoints, and exports them to the format you need — Postman collections, Markdown docs, cURL commands, and more. No annotations required, no runtime needed.

[![CI](https://github.com/tangcent/apilot/actions/workflows/ci.yml/badge.svg)](https://github.com/tangcent/apilot/actions/workflows/ci.yml)
[![codecov](https://codecov.io/gh/tangcent/apilot/branch/master/graph/badge.svg)](https://codecov.io/gh/tangcent/apilot)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

---

## What it does

Point APilot at a source directory and it figures out the rest:

```bash
apilot export ./my-service --formatter postman --output collection.json
apilot export ./my-service --formatter markdown --format detailed
apilot export ./my-service --formatter curl
```

It detects your language and framework automatically, walks the source tree, and produces clean, structured API output — ready to import into Postman, share as docs, or drop into a CI pipeline.

---

## Supported languages & frameworks

| Language   | Frameworks                              |
|------------|-----------------------------------------|
| Java       | Spring MVC, JAX-RS, Feign               |
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

---

## Installation

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

---

## Usage

```
apilot export <source-path> [flags]

Flags:
  --collector   string   Collector name (auto-detected if omitted)
  --formatter   string   Output format: markdown, curl, postman (default: markdown)
  --format      string   Format variant, e.g. simple, detailed (default: simple)
  --output      string   Output file path (default: stdout)
  --list-collectors      Print available collectors and exit
  --list-formatters      Print available formatters and exit
```

### Examples

```bash
# Export a Spring Boot project to Postman
apilot export ./backend --formatter postman --output api.postman_collection.json

# Generate detailed Markdown docs
apilot export ./backend --formatter markdown --format detailed --output API.md

# Quick cURL reference to stdout
apilot export ./backend --formatter curl
```

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
        └── api-formatter-{markdown,curl,postman}
```

The `api-collector` and `api-formatter` packages define the stable interfaces. Everything else is an implementation. See [docs/architecture.md](docs/architecture.md) for the full breakdown.

---

## Contributing

PRs and issues are welcome. See [CONTRIBUTING.md](CONTRIBUTING.md) to get started.

---

## License

[MIT](LICENSE)
