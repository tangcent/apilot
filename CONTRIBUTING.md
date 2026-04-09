# Contributing to APilot

Thank you for your interest in contributing to APilot! This document provides guidelines and instructions for contributing.

## Code of Conduct

Please be respectful and constructive in all interactions with the community.

## How to Contribute

### Reporting Bugs

Before creating a bug report, please check existing issues to avoid duplicates. When creating a bug report, include:

- A clear and descriptive title
- Steps to reproduce the issue
- Expected behavior and actual behavior
- Your environment (OS, Go version)
- Any relevant output or error messages

### Suggesting Enhancements

Enhancement suggestions are tracked as GitHub issues. Include:

- A clear and descriptive title
- A detailed description of the proposed functionality
- Why this enhancement would be useful
- Examples of how it would be used

### Pull Requests

1. Fork the repository and create your branch from `master`
2. Follow the project's architecture principles (see `docs/architecture.md`)
3. Write clear, concise commit messages using conventional commit format
4. Include tests for new functionality
5. Ensure all tests pass
6. Update documentation as needed

## Development Setup

### Prerequisites

- Go 1.22 or higher

### Building

```bash
# Build the apilot-cli binary
go build -o bin/apilot ./apilot-cli

# Cross-platform builds
GOOS=linux   GOARCH=amd64 go build -o bin/apilot-linux-amd64   ./apilot-cli
GOOS=darwin  GOARCH=arm64 go build -o bin/apilot-darwin-arm64  ./apilot-cli
GOOS=windows GOARCH=amd64 go build -o bin/apilot-windows-amd64.exe ./apilot-cli
```

### Running Tests

```bash
go test ./...
```

## Architecture Guidelines

See [docs/architecture.md](docs/architecture.md) for the full breakdown.

### Key Principles

1. `api-collector` and `api-formater` are the only shared contracts — no module may import above its layer
2. Collectors return `nil, nil` (not an error) when no endpoints are found
3. Formatters must return valid empty output for an empty endpoints slice
4. External plugins use the stdin/stdout JSON protocol — see [docs/plugin-protocol.md](docs/plugin-protocol.md)

## Project Structure

```
apilot/
├── api-collector/                  # Collector interface + ApiEndpoint model
├── api-formater/                   # Formatter interface + FormatOptions
├── api-master/                     # Core engine: registry, plugin loader, orchestration
├── apilot-cli/                     # Bundled CLI: statically links all collectors + formatters
├── api-collector-support-{lang}/   # Language-specific collectors
├── api-formater-{name}/            # Output formatters
├── vscode-plugin/                  # VSCode extension (TypeScript)
└── docs/                           # Architecture and protocol documentation
```

## Commit Message Guidelines

Use conventional commit format:

- `feat:` New feature
- `fix:` Bug fix
- `refactor:` Code refactoring
- `docs:` Documentation changes
- `test:` Test changes
- `chore:` Build/tooling changes

Example: `feat: add Ruby on Rails collector`

## Questions?

Feel free to open an issue for any questions about contributing.
