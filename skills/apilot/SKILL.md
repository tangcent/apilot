---
name: apilot
version: 0.1.1
description: "Navigate your APIs. Automatically scans source code (Java/Spring, Go/Gin, Node.js/Express, Python/FastAPI) and exports API endpoints to Postman collections, Markdown docs, or cURL commands. Use when the user needs to extract APIs from code, generate API documentation, create Postman collections, or export API endpoints from a codebase."
metadata:
  requires:
    bins: ["apilot"]
  cliHelp: "apilot --help"
---

# apilot

APilot automatically discovers API endpoints in your source code and exports them to various formats — no annotations or runtime required.

## Quick Start

```bash
# Export to Postman collection
apilot export ./my-service --formatter postman --output api.postman_collection.json

# Generate Markdown documentation
apilot export ./my-service --formatter markdown --format detailed --output API.md

# Generate cURL commands
apilot export ./my-service --formatter curl
```

## Core Commands

### `apilot export <source-path>`

Scan source code and export API endpoints.

**Flags:**
- `--collector <name>` — Collector to use (auto-detected if omitted)
- `--formatter <name>` — Output format: `markdown`, `curl`, `postman` (default: `markdown`)
- `--format <variant>` — Format variant: `simple`, `detailed` (default: `simple`)
- `--output <path>` — Output file path (default: stdout)

**Examples:**

```bash
# Auto-detect language and export to Postman
apilot export ./backend --formatter postman --output collection.json

# Generate detailed Markdown docs
apilot export ./backend --formatter markdown --format detailed --output API.md

# Quick cURL reference
apilot export ./backend --formatter curl
```

### `apilot export --list-collectors`

List all available collectors (language/framework parsers).

```bash
apilot export --list-collectors
```

### `apilot export --list-formatters`

List all available output formatters.

```bash
apilot export --list-formatters
```

## Supported Languages & Frameworks

| Language | Frameworks |
|----------|-----------|
| Java | Spring MVC, JAX-RS, Feign |
| Go | Gin, Echo, Fiber |
| Node.js | Express, Fastify, NestJS |
| Python | FastAPI, Django REST, Flask |

## Output Formats

| Format | Description |
|--------|-------------|
| `markdown` | Markdown documentation (simple or detailed) |
| `curl` | One cURL command per endpoint |
| `postman` | Postman Collection v2.1 JSON |

## Common Workflows

### Generate API documentation for a Spring Boot project

```bash
apilot export ./spring-boot-app --formatter markdown --format detailed --output API.md
```

### Create Postman collection from Express.js app

```bash
apilot export ./express-app --formatter postman --output api.postman_collection.json
```

### Quick API reference with cURL

```bash
apilot export ./my-api --formatter curl > api-reference.sh
```

### Export specific collector

```bash
# Force Java collector even if auto-detection might choose differently
apilot export ./mixed-project --collector java --formatter postman --output java-apis.json
```

## Tips

- APilot auto-detects the language and framework — you rarely need `--collector`
- Use `--format detailed` with markdown formatter for comprehensive docs including request/response examples
- The tool works on source code, not running services — no server startup required
- Output to stdout by default — redirect or use `--output` to save to file

## Plugin System

APilot supports external collectors and formatters via a plugin protocol. Any binary that speaks the stdin/stdout JSON protocol can be registered.

See the [plugin protocol documentation](https://github.com/tangcent/apilot/blob/main/docs/plugin-protocol.md) for details.
