# APilot — VSCode Extension

> Navigate your APIs. Automatically.

APilot scans your source code, extracts API endpoints, and exports them to the format you need — Postman collections, Markdown docs, or cURL commands. No annotations required, no runtime needed.

## Features

- **Right-click to export** — right-click in the Explorer or the editor and choose **APilot: Export**
- **Auto-detection** — automatically detects your language and framework
- **Multiple output formats** — Markdown, cURL, or Postman Collection v2.1

## Usage

There are three ways to trigger APilot:

| Method | Scope |
|--------|-------|
| Right-click a file/folder in the **Explorer** → **APilot: Export** | Exports the selected file or folder |
| Right-click in the **editor** → **APilot: Export** | Exports the current file |
| Command Palette (`Ctrl+Shift+P` / `Cmd+Shift+P`) → **APilot: Export** | Exports the current file |

The output appears in the **APilot** output channel (or a file, depending on your settings).

## Supported Languages & Frameworks

| Language | Frameworks |
|----------|------------|
| Java | Spring MVC, JAX-RS, Feign |
| Go | Gin, Echo, Fiber |
| Node.js | Express, Fastify, NestJS |
| Python | FastAPI, Django REST, Flask |

## Output Formats

| Format | Description |
|--------|-------------|
| `markdown` | Markdown docs (simple or detailed) |
| `curl` | One `curl` command per endpoint |
| `postman` | Postman Collection v2.1 JSON |

## Configuration

Open VSCode settings and search for **APilot**, or edit `settings.json` directly:

| Setting | Values | Default | Description |
|---------|--------|---------|-------------|
| `apilot.formatter` | `markdown`, `curl`, `postman` | `markdown` | Output format |
| `apilot.format` | `simple`, `detailed` | `simple` | Format variant (e.g. simple or detailed markdown) |
| `apilot.outputDestination` | `channel`, `file` | `channel` | Where to write the output |
| `apilot.outputFile` | file path | *(empty)* | Output file path (used when `outputDestination` is `file`) |
| `apilot.binaryPath` | file path | *(empty)* | Custom path to the apilot-cli binary (overrides the bundled binary) |

### Example `settings.json`

```json
{
  "apilot.formatter": "postman",
  "apilot.outputDestination": "file",
  "apilot.outputFile": "api.postman_collection.json"
}
```

## Requirements

The extension ships with a bundled `apilot` binary for macOS (Apple Silicon). On other platforms, or if you prefer to use your own build, set the `apilot.binaryPath` setting to point to the binary.

To install the CLI separately:

```bash
npm install -g @tangcent/apilot
```

## License

[MIT](https://github.com/tangcent/apilot/blob/master/LICENSE)
