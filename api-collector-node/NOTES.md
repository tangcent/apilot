# Node.js/TypeScript Source Parsing Strategy

> **Status**: Research completed (2026-04-18)
> **Decision**: Continue with tree-sitter (CGO); evaluate gotreesitter (pure Go) as migration path
> **Issue**: [#21](https://github.com/tangcent/apilot/issues/21)

---

## Executive Summary

After evaluating 5 parsing approaches, **tree-sitter with go-tree-sitter CGO bindings** remains the recommended approach for production use. The current implementation already uses it successfully for Express, Fastify, and NestJS parsing. A promising new option — **gotreesitter** (pure Go tree-sitter runtime) — may eliminate the CGO requirement in the near future without changing the extraction logic.

**Quick comparison:**

| Approach | Accuracy | Performance | CGO Required | TypeScript Support | Recommendation |
|----------|----------|-------------|--------------|-------------------|----------------|
| tree-sitter (CGO) | Excellent | Excellent | Yes | Full | **Current (keep)** |
| gotreesitter (pure Go) | Excellent | Good | No | Full (205 grammars) | **Future migration** |
| goja parser (pure Go) | Limited | Good | No | None (ES5.1 only) | Rejected |
| Subprocess node/ts-node | Excellent | Poor | No | Full | Rejected |
| typescript-go parser | Excellent | Excellent | No | Full | Watch (API not ready) |

---

## Current Implementation

The `api-collector-node` module already uses tree-sitter with CGO bindings:

- **Express/Fastify**: `tree-sitter-javascript` via `github.com/tree-sitter/tree-sitter-javascript/bindings/go`
- **NestJS**: `tree-sitter-typescript` via `github.com/tree-sitter/tree-sitter-typescript/bindings/go`
- **Core**: `github.com/tree-sitter/go-tree-sitter v0.25.0`

This works well. All three framework parsers (Express, Fastify, NestJS) produce accurate results by walking the tree-sitter AST to extract route definitions, HTTP methods, path parameters, and JSDoc comments.

**Current go.mod dependencies:**
```
github.com/tree-sitter/go-tree-sitter v0.25.0
github.com/tree-sitter/tree-sitter-javascript v0.25.0
github.com/tree-sitter/tree-sitter-typescript v0.23.2
github.com/mattn/go-pointer v0.0.1  // indirect (CGO helper)
```

---

## Evaluated Approaches

### 1. Tree-sitter + go-tree-sitter (CGO) — CURRENT

**How it works:**
```go
import tree_sitter "github.com/tree-sitter/go-tree-sitter"
import javascript "github.com/tree-sitter/tree-sitter-javascript/bindings/go"

p := tree_sitter.NewParser()
defer p.Close()
p.SetLanguage(tree_sitter.NewLanguage(javascript.Language()))
tree := p.Parse(source, nil)
defer tree.Close()
root := tree.RootNode()
// Walk AST nodes to extract route definitions
```

**Pros:**
- Full AST with comments, decorators, type annotations
- Battle-tested (Neovim, GitHub Code Search, Zed editor)
- Native performance (C runtime, static linking)
- Unified JavaScript + TypeScript support
- No runtime dependencies (no Node.js required)
- Incremental parsing capability
- Already implemented and working in this project

**Cons:**
- CGO required (complicates cross-compilation)
- Larger binary size (~2-5 MB added per grammar)
- Manual memory management (`defer tree.Close()`, `defer p.Close()`)
- Cross-compilation needs platform-specific C toolchain
- Go race detector and coverage tools cannot see across CGO boundary
- `go install` fails for downstream users without a C compiler

**Cross-compilation strategy:**
```bash
# Native build (macOS/ARM64)
GOOS=darwin GOARCH=arm64 CGO_ENABLED=1 go build

# Container build (Linux AMD64)
docker run --rm -v $PWD:/app -w /app golang:1.23 \
  bash -c "apt-get update && apt-get install -y build-essential && \
           go build -o bin/apilot-linux-amd64"

# Windows (mingw)
GOOS=windows GOARCH=amd64 CGO_ENABLED=1 \
CC=x86_64-w64-mingw32-gcc go build
```

---

### 2. gotreesitter (Pure Go Tree-sitter) — RECOMMENDED MIGRATION PATH

**How it works:**
```go
import gts "github.com/odvcencio/gotreesitter"
import "github.com/odvcencio/gotreesitter/grammars"

entry := grammars.DetectLanguage("example.ts")
lang := entry.Language()
parser := gts.NewParser(lang)
tree, _ := parser.Parse(src)
root := tree.RootNode()
// Same AST walking logic as CGO version
```

**What it is:** A ground-up reimplementation of the tree-sitter runtime in pure Go. No C code, no CGO, no shared libraries. Ships 205 embedded grammars as compressed blobs that lazy-load on first use.

**Pros:**
- Zero CGO — cross-compiles to any Go target (including WASM)
- Same parse-table format as tree-sitter — existing grammars work without recompilation
- 205 embedded grammars (JavaScript, TypeScript, and more)
- Incremental parsing supported
- S-expression query engine with predicates and cursor-based streaming
- Arena allocator minimizes GC pressure
- Claims 90x faster incremental edits vs C implementation
- CI/CD simplification: no C toolchain needed in build pipeline

**Cons:**
- Newer project (less battle-tested than C tree-sitter)
- API differs from go-tree-sitter (requires code changes)
- Full-parse performance may be slightly slower than C runtime
- Smaller community and fewer production deployments
- Not yet widely adopted in the Go ecosystem

**Migration effort:**
The AST node types and structure are the same (same grammar = same tree shape), so the extraction logic in `express/parser.go`, `fastify/parser.go`, and `nestjs/parser.go` would need API adaptation but the tree-walking logic would remain conceptually identical. The main changes would be:
1. Replace `tree_sitter.NewParser()` with `gts.NewParser(lang)`
2. Replace `node.Child(i)` / `node.Kind()` / `node.Utf8Text(source)` with gotreesitter equivalents
3. Remove CGO-related build configuration

**When to migrate:** Once gotreesitter reaches v1.0 or after validating it against the existing test suite.

---

### 3. Pure-Go JS/TS AST via goja — REJECTED

**How it works:**
```go
import "github.com/dop251/goja/parser"

ast, err := parser.ParseFile(nil, "", source, 0)
// Walk goja's AST to extract route definitions
```

**What it is:** goja is a pure Go ECMAScript 5.1 engine. It includes a lexer and parser that produce an AST. It is used in production by Grafana k6 and Nakama.

**Pros:**
- Pure Go — no CGO, cross-compiles everywhere
- Well-maintained and production-proven
- Can execute JavaScript (not just parse)
- ES6+ features partially implemented (promises, classes, arrow functions, destructuring)

**Cons:**
- **No TypeScript support** — goja only parses JavaScript (ES5.1 + partial ES6)
- Cannot parse `.ts` files with decorators (NestJS), type annotations, or TS-specific syntax
- Would require a separate TypeScript transpilation step before parsing
- AST node types differ from tree-sitter — would require rewriting all extraction logic
- ES6 support is incomplete and in-progress (modules/import still experimental)
- Not goroutine-safe (one runtime per goroutine)
- For NestJS specifically: decorators are a TypeScript feature that goja cannot parse

**Why rejected:** The NestJS parser relies entirely on TypeScript decorators (`@Controller`, `@Get`, `@Post`, `@Param`, etc.). goja cannot parse TypeScript, making it unsuitable as the primary parser. Even for JavaScript-only frameworks (Express, Fastify), the AST structure differs from tree-sitter, requiring a complete rewrite of extraction logic for no accuracy gain.

---

### 4. Subprocess node/ts-node — REJECTED

**How it works:**
```go
// Option A: Use Node.js to run a custom extraction script
exec.Command("node", "--input-type=module", "-e", extractionScript).Output()

// Option B: Use ts-node for TypeScript files
exec.Command("npx", "ts-node", extractionScript).Output()

// Option C: Use TypeScript compiler API
exec.Command("npx", "tsgo", "--emitDeclarationOnly", file).Output()
```

**Pros:**
- 100% accurate (uses the actual JS/TS parser)
- No CGO required
- Full TypeScript support including decorators, generics, and type annotations
- Can leverage the TypeScript Compiler API for rich AST extraction
- tsgo (TypeScript native) offers 10x speed improvement over tsc

**Cons:**
- Requires Node.js installed at runtime
- Process spawn overhead per file (100ms+ per invocation)
- User environment complexity (Node version, npm availability)
- Violates the "zero-dependency CLI" principle
- Distribution complexity (must bundle extraction scripts)
- Cannot parse files offline or in air-gapped environments
- Security concerns (executing arbitrary code in subprocess)

**Why rejected:** Runtime dependency on Node.js is a deal-breaker for a CLI tool that should work with zero external dependencies. The project's architecture guide explicitly states that `apilot-cli` should be a self-contained binary.

---

### 5. typescript-go Parser (Microsoft) — WATCH

**How it works:**
```go
// Future API (not yet available as standalone module)
import "github.com/microsoft/typescript-go/internal/ast"
// Parse TypeScript source into AST
```

**What it is:** Microsoft's official native Go port of the TypeScript compiler (Project Corsa / TypeScript 7). Includes a complete TypeScript parser written in Go.

**Pros:**
- Official Microsoft implementation — highest accuracy guarantee
- Pure Go — no CGO
- Full TypeScript + JSX + JavaScript support
- 10x faster than the JavaScript-based tsc
- Will eventually become the official TypeScript compiler

**Cons:**
- API is marked "not ready" for external use
- Parser is in `internal/` directory — cannot be imported as a library
- No standalone parser module published yet
- Community has requested extraction as independent module ([Discussion #2442](https://github.com/microsoft/typescript-go/discussions/2442)) but no commitment from Microsoft
- May require forking or copying internal packages

**When to reconsider:** If Microsoft publishes the parser as a standalone Go module, this would become the best option for TypeScript parsing. Monitor [microsoft/typescript-go](https://github.com/microsoft/typescript-go) for API availability.

---

## Decision Matrix

Scored on 8 dimensions (5 = best, 1 = worst):

| Dimension | tree-sitter (CGO) | gotreesitter | goja | subprocess | typescript-go |
|-----------|-------------------|--------------|------|------------|---------------|
| Accuracy | 5 | 5 | 3 | 5 | 5 |
| Go Purity | 2 | 5 | 5 | 3 | 5 |
| Performance | 5 | 4 | 4 | 2 | 5 |
| TypeScript Support | 5 | 5 | 1 | 5 | 5 |
| Dependency Simplicity | 3 | 5 | 4 | 1 | 3 |
| Cross-compilation | 2 | 5 | 5 | 4 | 5 |
| Maintainability | 4 | 4 | 3 | 2 | 2 |
| Maturity | 5 | 3 | 4 | 4 | 2 |
| **TOTAL** | **31/40** | **36/40** | **29/40** | **26/40** | **32/40** |

---

## Recommendation

### Short-term (now): Keep tree-sitter with CGO

The current implementation works correctly. All tests pass. The CGO requirement is a build-time concern, not a runtime concern — users of `apilot-cli` get a statically linked binary and never need a C compiler.

### Medium-term (next quarter): Evaluate gotreesitter migration

Once gotreesitter stabilizes (v1.0 release or sufficient community adoption), migrate from `go-tree-sitter` (CGO) to `gotreesitter` (pure Go). This would:
- Eliminate CGO from the entire project
- Simplify CI/CD (no C toolchain in build matrix)
- Enable WASM builds
- Reduce binary size
- Keep the same tree-sitter grammar semantics (minimal extraction logic changes)

**Validation plan:**
1. Add gotreesitter as an alternative parser behind a build tag
2. Run existing test suite against gotreesitter
3. Benchmark parse performance on real-world Express/Fastify/NestJS projects
4. If all tests pass and performance is acceptable, switch default

### Long-term: Monitor typescript-go

If Microsoft publishes the TypeScript parser as a standalone Go module, evaluate it as a replacement for tree-sitter-typescript specifically. This would provide the most accurate TypeScript parsing available.

---

## Prototype Code

### gotreesitter Proof of Concept

```go
package nodecollector

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	gts "github.com/odvcencio/gotreesitter"
	"github.com/odvcencio/gotreesitter/grammars"

	collector "github.com/tangcent/apilot/api-collector"
)

func discoverJSFiles(sourceDir string) ([]string, error) {
	var files []string
	err := filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() && (strings.HasSuffix(path, ".js") || strings.HasSuffix(path, ".mjs")) {
			files = append(files, path)
		}
		return nil
	})
	return files, err
}

func parseExpressFile(filePath string) ([]collector.ApiEndpoint, error) {
	source, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	entry := grammars.DetectLanguage(filePath)
	if entry == nil {
		return nil, nil
	}

	lang := entry.Language()
	parser := gts.NewParser(lang)
	defer parser.Close()

	tree, err := parser.Parse(source)
	if err != nil {
		return nil, fmt.Errorf("parse error: %w", err)
	}
	defer tree.Close()

	root := tree.RootNode()
	// Walk AST using same logic as current express/parser.go
	// Node types are identical because the grammar is the same
	_ = root

	return nil, nil
}
```

### goja AST Extraction (JavaScript only)

```go
package nodecollector

import (
	"fmt"

	"github.com/dop251/goja/parser"
	"github.com/dop251/goja/ast"
)

func extractExpressRoutesJS(source string) error {
	prog, err := parser.ParseFile(nil, "", source, 0)
	if err != nil {
		return fmt.Errorf("parse error: %w", err)
	}

	for _, stmt := range prog.Body {
		exprStmt, ok := stmt.(*ast.ExpressionStatement)
		if !ok {
			continue
		}

		call, ok := exprStmt.Expression.(*ast.CallExpression)
		if !ok {
			continue
		}

		member, ok := callee.MemberExpression
		if !ok {
			continue
		}

		// Check if member.Property is an HTTP method
		// Extract path from first argument
		// ...
		_ = call
	}

	return nil
}
```

---

## References

### Tree-sitter
- [go-tree-sitter](https://github.com/tree-sitter/go-tree-sitter) — Official Go bindings (CGO)
- [tree-sitter-javascript](https://github.com/tree-sitter/tree-sitter-javascript) — JavaScript grammar
- [tree-sitter-typescript](https://github.com/tree-sitter/tree-sitter-typescript) — TypeScript grammar

### Pure-Go Alternatives
- [gotreesitter](https://github.com/odvcencio/gotreesitter) — Pure Go tree-sitter runtime (no CGO, 205 grammars)
- [goja](https://github.com/dop251/goja) — Pure Go ECMAScript 5.1 engine (parser only, no TypeScript)
- [typescript-ast-go](https://github.com/armsnyder/typescript-ast-go) — Limited TypeScript AST parser (not feature-complete)

### Microsoft TypeScript Native
- [typescript-go](https://github.com/microsoft/typescript-go) — Official native Go port of TypeScript
- [Parser extraction proposal](https://github.com/microsoft/typescript-go/discussions/2442) — Community request for standalone parser module

### APilot Project
- [architecture.md](../docs/architecture.md) — Overall architecture (Open Questions §2)
- [api-collector-java/NOTES.md](../api-collector-java/NOTES.md) — Java parsing strategy (same tree-sitter decision)

---

*Last updated: 2026-04-18*
*Issue: [#21](https://github.com/tangcent/apilot/issues/21)*
