---
name: apilot
version: 0.4.0
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
# Export directly to Postman (recommended)
apilot export ./my-service --formatter postman --params '{"mode":"api"}'

# Export to Postman collection JSON file
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
- `--params <json>` — Formatter-specific params as JSON
- `--output <path>` — Output file path (default: stdout)

**Examples:**

```bash
# Auto-detect language and export to Postman
apilot export ./backend --formatter postman --output collection.json

# Push directly to Postman cloud
apilot export ./backend --formatter postman --params '{"mode":"api"}'

# Generate detailed Markdown docs
apilot export ./backend --formatter markdown --format detailed --output API.md

# Quick cURL reference
apilot export ./backend --formatter curl
```

### `apilot set <key> <value>`

Persist a configuration value.

```bash
apilot set postman.api.key PMAK-xxxx
apilot set postman.export.mode UPDATE_EXISTING
```

### `apilot get <key>`

Read a configuration value.

```bash
apilot get postman.api.key
apilot get postman.export.mode
```

### `apilot settings`

List all settings required by registered formatters.

### `apilot collections [list|remove]`

Manage project-to-collection bindings (auto-saved when using UPDATE_EXISTING mode).

```bash
# List all project collection bindings
apilot collections

# Remove a binding for a specific project
apilot collections remove my-project
```

### `apilot export --list-collectors`

List all available collectors (language/framework parsers).

### `apilot export --list-formatters`

List all available output formatters.

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
| `postman` | Postman Collection v2.1 JSON or direct API push |

## Postman Settings

| Setting | Description |
|---------|-------------|
| `postman.api.key` | Postman API key (required for API push mode) |
| `postman.export.mode` | Export mode: `CREATE_NEW` (always create new collection) or `UPDATE_EXISTING` (update previously exported collection per project, default: `CREATE_NEW`) |

## Smart Mode Detection

The Postman formatter automatically detects the export mode based on context:

| Condition | Mode | Behavior |
|-----------|------|----------|
| API key configured, no `--output` | **api** | Push directly to Postman cloud |
| `--output` specified | **file** | Write Postman Collection JSON to file |
| No API key configured | **file** | Output Postman Collection JSON to stdout |
| `--params '{"mode":"api"}'` | **api** | Explicit push to Postman (overrides auto-detection) |
| `--params '{"mode":"file"}'` | **file** | Explicit file output (overrides auto-detection) |

This means the simplest command just works:

```bash
# If postman.api.key is set → pushes to Postman automatically
apilot export ./backend --formatter postman

# If you want a JSON file instead → use --output
apilot export ./backend --formatter postman --output collection.json
```

## Project Name Inference

The project name is automatically inferred from the source directory path using the basename:

- `/Users/tangcent/code/github/java-spring-demo` → `java-spring-demo`
- `/home/user/repos/api-service` → `api-service`
- `./my-project` → `my-project`

This project name is used as the key for per-project collection bindings. You can override it via the `projectName` param:

```bash
apilot export ./backend --formatter postman --params '{"mode":"api","projectName":"my-custom-name"}'
```

## Postman Export Modes

### CREATE_NEW (default)

Always creates a new collection in Postman. Each export produces a new collection with a fresh UID.

### UPDATE_EXISTING

Remembers which collection and workspace each project maps to. On first export, creates a new collection and saves the binding (including workspace). On subsequent exports, updates the same collection in-place — no duplicates.

The project name is derived from the source directory basename (e.g. `/code/my-service` → `my-service`).

Bindings are stored in `~/.config/apilot/postman_collections.json` and can be managed with:

```bash
apilot collections          # list all bindings
apilot collections remove my-project  # remove a binding
```

## Per-Project Workspace & Collection

Workspace and collection bindings are stored **per-project**, not globally. This means:

- Each project can export to a different workspace and collection
- When using `UPDATE_EXISTING` mode, the workspace is remembered alongside the collection
- No global workspace configuration is needed — the first export determines the workspace

To specify a workspace for the first export of a project:

```bash
# Create collection in a specific workspace (workspace is remembered for future exports)
apilot export ./backend --formatter postman --params '{"mode":"api","workspaceId":"ws-xxxx"}'
```

## Smart Postman Export Workflow

When the user asks to export APIs to Postman, follow this workflow:

### Step 1: Check for Postman API key

Run `apilot get postman.api.key` to check if a key is already configured.

- **If key exists**: Proceed to Step 2.
- **If key is missing**: Ask the user:
  > "No Postman API key found. Would you like to provide one, or export as a JSON file instead?"
  >
  > Options:
  > 1. **Provide API key** — I'll save it with `apilot set postman.api.key <key>` so you can push directly to Postman. Get your key at: https://go.postman.co/settings/api-keys
  > 2. **Export as JSON file** — I'll generate a `.postman_collection.json` file you can import manually.

### Step 2: Check export mode

Run `apilot get postman.export.mode` to check the current export mode.

- **If UPDATE_EXISTING**: The export will automatically update the previously created collection for this project (if one exists). No further configuration needed.
- **If CREATE_NEW or not set**: Each export creates a new collection. Ask the user:
  > "Would you like to use UPDATE_EXISTING mode so the same collection is updated on each export?"
  >
  > Options:
  > 1. **Yes, set UPDATE_EXISTING** — I'll save it with `apilot set postman.export.mode UPDATE_EXISTING`. Future exports will update the same collection.
  > 2. **No, keep CREATE_NEW** — Each export creates a new collection (useful for versioned snapshots).

### Step 3: Check for existing project binding

Run `apilot collections` to see if this project already has a binding.

- **If binding exists**: The export will update the existing collection in the remembered workspace. No further configuration needed.
- **If no binding**: Ask the user:
  > "Would you like to create the collection in a specific Postman workspace?"
  >
  > Options:
  > 1. **Specify workspace** — Provide the workspace ID and it will be remembered for this project: `apilot export <path> --formatter postman --params '{"workspaceId":"ws-xxxx"}'`
  > 2. **Use personal workspace** — The collection will be created in your default personal workspace

### Step 4: Run the export

```bash
apilot export <source-path> --formatter postman
```

The formatter automatically pushes to Postman API when the API key is configured. If you need a JSON file instead, use `--output`.

If the user wants to override defaults for this run:

```bash
# Push to a specific workspace (workspace is remembered for this project)
apilot export <source-path> --formatter postman --params '{"workspaceId":"ws-xxxx"}'

# Update a specific collection (overrides UPDATE_EXISTING binding)
apilot export <source-path> --formatter postman --params '{"collectionUid":"12345-xxxx"}'

# Custom collection name
apilot export <source-path> --formatter postman --params '{"collectionName":"My API"}'

# Force file output (Postman Collection JSON) instead of API push
apilot export <source-path> --formatter postman --output collection.json
```

### Step 5: Report the result

On success, the output will be JSON like:

```json
{
  "collectionId": "...",
  "collectionUid": "...",
  "collectionUrl": "https://go.postman.co/collection/...",
  "action": "created"
}
```

or

```json
{
  "collectionId": "...",
  "collectionUid": "...",
  "collectionUrl": "https://go.postman.co/collection/...",
  "action": "updated"
}
```

Tell the user:
- Whether the collection was **created** or **updated**
- The URL to open it in Postman
- If using UPDATE_EXISTING mode, the binding (including workspace) is automatically saved — no manual configuration needed

## Common Workflows

### Generate API documentation for a Spring Boot project

```bash
apilot export ./spring-boot-app --formatter markdown --format detailed --output API.md
```

### Push APIs directly to Postman (one-time setup)

```bash
apilot set postman.api.key PMAK-xxxx
apilot set postman.export.mode UPDATE_EXISTING
apilot export ./backend --formatter postman
```

### Push to a specific workspace (workspace is remembered per-project)

```bash
# First export: specify workspace, it's remembered for this project
apilot export ./backend --formatter postman --params '{"workspaceId":"ws-xxxx"}'

# Subsequent exports: workspace is automatically used from the saved binding
apilot export ./backend --formatter postman
```

### Update an existing Postman collection automatically

```bash
# First export creates the collection and remembers the binding
apilot export ./backend --formatter postman

# Subsequent exports update the same collection (with UPDATE_EXISTING mode)
apilot export ./backend --formatter postman
```

### Export to Postman JSON file (manual import)

```bash
apilot export ./express-app --formatter postman --output api.postman_collection.json
```

### Quick API reference with cURL

```bash
apilot export ./my-api --formatter curl > api-reference.sh
```

### Export specific collector

```bash
apilot export ./mixed-project --collector java --formatter postman --params '{"mode":"api"}'
```

### Manage collection bindings

```bash
# See which projects are bound to which collections
apilot collections

# Remove a stale binding
apilot collections remove old-project
```

## Tips

- APilot auto-detects the language and framework — you rarely need `--collector`
- Use `--format detailed` with markdown formatter for comprehensive docs including request/response examples
- The tool works on source code, not running services — no server startup required
- Set `postman.export.mode` to `UPDATE_EXISTING` to avoid creating duplicate collections on every export
- Project-to-collection bindings (including workspace) are stored automatically per-project — no need to manually configure collection UIDs or workspace IDs
- The project name is auto-inferred from the source directory basename
- Output to stdout by default — redirect or use `--output` to save to file

## Plugin System

APilot supports external collectors and formatters via a plugin protocol. Any binary that speaks the stdin/stdout JSON protocol can be registered.

See the [plugin protocol documentation](https://github.com/tangcent/apilot/blob/main/docs/plugin-protocol.md) for details.
