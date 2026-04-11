# Java/Kotlin Source Parsing Strategy

> **Status**: ✅ Research completed (2026-04-11)
> **Decision**: Tree-sitter with CGO bindings
> **Full Research**: [docs/java-kotlin-parsing-research.md](../docs/java-kotlin-parsing-research.md)

---

## Executive Summary

After evaluating 5 parsing approaches, **Tree-sitter with go-tree-sitter CGO bindings** is recommended for production use. It provides the best balance of accuracy (full AST), performance (native speed), and ecosystem maturity.

**Quick comparison:**

| Approach | Accuracy | Performance | Complexity | Kotlin Support | Recommendation |
|----------|----------|-------------|------------|----------------|----------------|
| Tree-sitter (CGO) | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐⭐ | Medium | ✅ Excellent | **✅ Recommended** |
| ANTLR4 Go | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐ | Medium-High | ⚠️ Requires grammar | ⚙️ Fallback |
| javac subprocess | ⭐⭐⭐⭐⭐ | ⭐⭐ | Low | ✅ Excellent | ❌ Runtime deps |
| JavaParser subprocess | ⭐⭐⭐⭐⭐ | ⭐⭐⭐ | Medium | ⚠️ Limited | ❌ Deployment overhead |
| Regex/heuristic | ⭐⭐ | ⭐⭐⭐⭐⭐ | Low | ⚠️ Limited | ❌ MVP only |

---

## Evaluated Approaches

### 1. Tree-sitter + go-tree-sitter (CGO) ✅ RECOMMENDED

**How it works:**
```go
import tree_sitter "github.com/tree-sitter/go-tree-sitter"
import java "github.com/tree-sitter/tree-sitter/java"

parser := tree_sitter.NewParser(unsafe.Pointer(java.Language()))
tree := parser.ParseString(sourceCode)
// Extract annotations, methods, types from AST
```

**Pros:**
- ✅ Full AST with comments, type information
- ✅ Battle-tested (Neovim, GitHub Code Search)
- ✅ Native performance (static linking)
- ✅ Unified Kotlin support ([tree-sitter-kotlin](https://github.com/tree-sitter/tree-sitter-kotlin))
- ✅ No runtime dependencies (JDK/javac not required)
- ✅ Incremental parsing capability (future optimization)

**Cons:**
- ⚠️ CGO required (complicates cross-compilation)
- ⚠️ Larger binary size
- ⚠️ Manual memory management (`defer tree.Close()`)
- ⚠️ Cross-compilation needs platform-specific C toolchain

**Key Implementation Notes:**
```go
// Memory management pattern
parser := tree_sitter.NewParser(unsafe.Pointer(lang))
defer parser.Close()

tree := parser.ParseString(sourceCode)
defer tree.Close()

cursor := tree.Walk()
defer cursor.Close()
```

**Cross-compilation strategy:**
```bash
# Native build (macOS/ARM64)
GOOS=darwin GOARCH=arm64 go build

# Container build (Linux AMD64)
docker run --rm -v $PWD:/app -w /app golang:1.22 \
  bash -c "apt-get update && apt-get install -y build-essential && \
           go build -o bin/apilot-linux-amd64"

# Windows (mingw)
GOOS=windows GOARCH=amd64 CGO_ENABLED=1 \
CC=x86_64-w64-mingw32-gcc go build
```

**Why this wins:**
- Accuracy + Performance + Ecosystem maturity
- One parser for Java + Kotlin (consistent API)
- No external runtime dependencies
- Proven in production (GitHub, Neovim)

---

### 2. ANTLR4 Go Runtime ⚙️ FALLBACK

**How it works:**
```bash
go get github.com/antlr/antlr4/v4
go get github.com/antlr4-go/antlr  # Pure Go runtime
# Use pre-compiled Java grammar from antlr4-grammars
```

**Pros:**
- ✅ Pure Go runtime (no CGO)
- ✅ Powerful parser generator
- ✅ Pre-compiled grammars available

**Cons:**
- ⚠️ Grammar generation still needs ANTLR toolchain (Java/Python)
- ⚠️ Steeper learning curve
- ⚠️ Kotlin support requires separate grammar
- ⚠️ Code generation adds build complexity

**When to use:** If CGO cross-compilation proves insurmountable after 2+ days of effort

---

### 3. javac/kotlinc Subprocess ❌ NOT RECOMMENDED

**How it works:**
```go
exec.Command("javac", "-proc:only", "-Xprint:ast", file).Output()
```

**Pros:**
- ✅ 100% accurate (compiler's own parser)
- ✅ No CGO

**Cons:**
- ❌ Requires JDK at runtime
- ❌ Platform compatibility issues
- ❌ Process spawn overhead
- ❌ User environment complexity

**Why rejected:** Runtime dependency on JDK violates "zero-dependency CLI" principle

---

### 4. JavaParser Subprocess ❌ NOT RECOMMENDED

**How it works:**
```go
exec.Command("java", "-jar", "javaparser-cli.jar", file).Output()
```

**Pros:**
- ✅ Accurate parsing
- ✅ No CGO

**Cons:**
- ❌ Requires Java runtime
- ❌ Must distribute JAR file
- ❌ Deployment complexity

**Why rejected:** Same runtime dependency issue as javac

---

### 5. Pure Go Regex/Heuristic Parser 🚧 MVP ONLY

**How it works:**
```go
re.MustCompile(`@GetMapping\("([^"]+)"\)`).FindAllStringSubmatch(src, -1)
```

**Pros:**
- ✅ Zero dependencies
- ✅ Fast implementation
- ✅ Cross-platform

**Cons:**
- ❌ Limited accuracy (can't handle complex syntax)
- ❌ Hard to maintain (regex complexity)
- ❌ Misses edge cases (nested annotations, generics)
- ❌ No type information extraction

**When to use:** 3-day MVP prototype only, then upgrade to Tree-sitter

---

## Decision Matrix

Scored on 8 dimensions (5 = best, 1 = worst):

| Dimension | Tree-sitter | ANTLR4 | javac | JavaParser | Regex |
|-----------|-------------|--------|-------|------------|-------|
| Accuracy | 5 | 5 | 5 | 5 | 2 |
| Go Purity | 4 | 5 | 3 | 3 | 5 |
| Performance | 5 | 4 | 2 | 3 | 5 |
| Dependency Simplicity | 4 | 3 | 2 | 3 | 5 |
| Kotlin Support | 5 | 3 | 5 | 3 | 2 |
| Cross-compilation | 3 | 5 | 5 | 5 | 5 |
| Maintainability | 4 | 3 | 2 | 3 | 2 |
| Learning Curve | 3 | 2 | 3 | 3 | 5 |
| **TOTAL** | **38/40** | **35/40** | **32/40** | **31/40** | **28/40** |

---

## Implementation Plan

### Phase 0: PoC Validation (1 day) ⬅️ START HERE

**Goal:** Verify Tree-sitter can extract Spring annotations accurately

```go
// Test case: minimal Spring MVC controller
@RestController
@RequestMapping("/api/users")
public class UserController {
    @GetMapping("/{id}")
    public ResponseEntity<User> getUser(@PathVariable Long id) {
        return ResponseEntity.ok(userService.findById(id));
    }
}
```

**PoC Checklist:**
- [ ] Parse `@RestController`, `@RequestMapping` annotations
- [ ] Extract method-level `@GetMapping` with path
- [ ] Extract parameter annotations (`@PathVariable`)
- [ ] Extract return type `ResponseEntity<User>`
- [ ] Test macOS → Linux cross-compilation

**Success Criteria:**
- All annotations extracted correctly
- Cross-compilation builds without errors
- Parse time < 100ms for 50-line file

**Failure Condition:**
- If PoC fails after 1 day → switch to ANTLR4

---

### Phase 1: Parser Adapter Layer (1-2 days)

```go
// api-collector-java/parser/parser.go
package parser

import tree_sitter "github.com/tree-sitter/go-tree-sitter"

type Parser struct {
    lang   *tree_sitter.Language
    parser *tree_sitter.Parser
}

func NewJavaParser() (*Parser, error) {
    lang := unsafe.Pointer(java.Language())
    parser := tree_sitter.NewParser(lang)
    return &Parser{lang: lang, parser: parser}, nil
}

func (p *Parser) ParseFile(path string) (*AST, error) {
    src, err := os.ReadFile(path)
    if err != nil {
        return nil, err
    }

    tree := p.parser.ParseString(string(src))
    defer tree.Close()

    return &AST{tree: tree, source: src}, nil
}

func (p *Parser) Close() {
    p.parser.Close()
}
```

---

### Phase 2: Spring MVC Parser (2-3 days)

**Target annotations:**
- `@RestController` / `@Controller`
- `@RequestMapping` (class + method level)
- `@GetMapping` / `@PostMapping` / `@PutMapping` / `@DeleteMapping` / `@PatchMapping`
- `@PathVariable` / `@RequestParam` / `@RequestBody`

**Extraction logic:**
1. Walk AST to find classes with `@RestController`
2. Extract class-level `@RequestMapping` path
3. For each method:
   - Extract HTTP method from annotation
   - Combine class path + method path
   - Extract parameters and types
   - Extract return type

---

### Phase 3: JAX-RS Parser (1-2 days)

**Target annotations:**
- `@Path` (class + method level)
- `@GET` / `@POST` / `@PUT` / `@DELETE`
- `@PathParam` / `@QueryParam` / `@Consumes` / `@Produces`

---

### Phase 4: Feign Client Parser (1-2 days)

**Target annotations:**
- `@FeignClient`
- `@RequestLine` (Netflix Feign)
- Spring MVC annotations (Spring Cloud OpenFeign)

---

### Phase 5: Kotlin Support (1 day)

**Kotlin-specific considerations:**
- Data classes: `data class User(val id: Long, val name: String)`
- Companion objects: `companion object { @JvmStatic fun create() }`
- Extension functions: `fun String.toUser(): User`
- DSL routing (Ktor): `routing { get("/users") { ... } }`

**Test projects:**
- Pure Kotlin Spring Boot
- Java + Kotlin mixed project
- Ktor framework project

---

### Phase 6: Maven Integration (1-2 days)

**Dependency resolution strategy:**

```go
// 1. Run maven-indexer-cli to get dependency classes
exec.Command("maven-indexer-cli", "scan", projectDir).Output()
// → Produces classes.json: {"com.example.User": "user-service-1.0.0.jar"}

// 2. During parsing, resolve type imports
import "com.example.User"  // → Look up in classes.json
```

**Integration order:**
- **Phase 6a**: Serial (maven-indexer first, then tree-sitter)
- **Phase 6b** (optimization): Parallel with lazy lookup

---

### Phase 7: CI/CD Cross-compilation (1 day)

**GitHub Actions matrix:**

```yaml
# .github/workflows/build.yml
strategy:
  matrix:
    os: [ubuntu-latest, macos-latest, windows-latest]
    arch: [amd64, arm64]
    exclude:
      - os: windows-latest
        arch: arm64

steps:
  - name: Install CGO deps (Linux)
    if: runner.os == 'Linux'
    run: sudo apt-get install -y build-essential

  - name: Install CGO deps (Windows)
    if: runner.os == 'Windows'
    run: choco install mingw

  - name: Build
    run: go build -o bin/apilot-${{ matrix.os }}-${{ matrix.arch }}
```

---

## Prototype Benchmark (Planned)

**Test Projects:**
1. **Small**: Spring PetClinic (~50 Java files)
2. **Medium**: Spring Cloud Gateway (~500 files)
3. **Large**: Apache Dubbo (~2000 files)

**Performance Targets:**
- Small: < 1 second
- Medium: < 5 seconds
- Large: < 20 seconds

**Comparison Matrix:**
- Tree-sitter vs ANTLR4 vs javac subprocess
- Single-threaded vs parallel parsing (4/8/16 workers)

---

## Error Handling Strategy

### Error Classification

**Fatal Errors** (stop execution):
- go-tree-sitter initialization failure
- Source directory unreadable
- Memory allocation failure

**Warning Errors** (skip file, continue):
- Single file syntax error
- Unrecognized annotation format
- Corrupted source file

**Debug Info** (log only):
- No API annotations found in file
- Unused import statements
- Non-framework classes

### Error Output Format

```
❌ Fatal: Failed to initialize Java parser
   Cause: go-tree-sitter library not found
   Fix: Ensure CGO is enabled and tree-sitter is installed

⚠️  Skipped: src/main/java/com/example/UserController.java
   Cause: Syntax error at line 45 (missing closing brace)
   Impact: API endpoints in this file will not be collected

ℹ️  Debug: src/main/java/com/example/Utils.java
   Info: No API annotations found (normal utility class)
```

---

## Open Questions & Mitigation

### Q1: CGO cross-compilation complexity?

**Risk**: Windows/ARM builds may fail in CI

**Mitigation:**
- Use Docker for Linux builds
- Use mingw-w64 for Windows builds
- Fallback to ANTLR4 if blocked > 2 days

### Q2: Type inference accuracy for generics?

**Risk**: `ResponseEntity<List<UserDTO>>` may not be fully resolved

**Mitigation:**
- PoC test with complex generics
- Compare with javac AST output
- Accept 95%+ accuracy (edge cases documented)

### Q3: Performance on large codebases?

**Risk**: 10,000+ Java files may exceed target time

**Mitigation:**
- Implement parallel file parsing (goroutines)
- Add file-level caching (hash-based skip)
- Profile and optimize hot paths

---

## Decision Timeline

| Date | Milestone |
|------|-----------|
| 2026-04-10 | Research completed |
| 2026-04-11 | **Decision: Tree-sitter recommended** |
| 2026-04-12 | PoC validation (Phase 0) |
| 2026-04-13 | Decision checkpoint: continue or switch to ANTLR4 |
| 2026-04-15 | Parser adapter layer (Phase 1) |
| 2026-04-18 | Spring MVC parser (Phase 2) |
| 2026-04-20 | JAX-RS + Feign parsers (Phase 3-4) |
| 2026-04-22 | Kotlin support (Phase 5) |
| 2026-04-25 | Maven integration (Phase 6) |
| 2026-04-26 | CI/CD configuration (Phase 7) |

**Total estimated time:** 10-12 working days

---

## References

### Tree-sitter
- [go-tree-sitter](https://github.com/tree-sitter/go-tree-sitter) — Official Go bindings
- [tree-sitter-java](https://github.com/tree-sitter/tree-sitter-java) — Java grammar
- [tree-sitter-kotlin](https://github.com/tree-sitter/tree-sitter-kotlin) — Kotlin grammar
- [Tree-sitter Go Tutorial](https://dev.to/shrsv/tinkering-with-tree-sitter-using-go-4d8n)

### ANTLR4
- [antlr4-go/antlr](https://github.com/antlr4-go/antlr) — Go runtime
- [antlr4-grammars](https://github.com/antlr/grammars-v4) — Pre-built grammars

### Java/Kotlin Parsing
- [JavaParser](https://javaparser.org/) — Java library (JVM only)
- [kotlinx/ast](https://github.com/kotlinx/ast) — Kotlin AST (JVM only)

### APilot Project
- [architecture.md](../docs/architecture.md) — Overall architecture
- [contributing-collectors.md](../docs/contributing-collectors.md) — Collector development guide
- [java-kotlin-parsing-research.md](../docs/java-kotlin-parsing-research.md) — Full research details

---

## Conclusion

**Recommended:** Tree-sitter-java + go-tree-sitter (CGO)

**Next Steps:**
1. ✅ Create this NOTES.md
2. ⏭️ Implement PoC (Phase 0)
3. ⏭️ Decision checkpoint: proceed or fallback to ANTLR4

**Estimated delivery:** 10-12 days from PoC start

---

*Last updated: 2026-04-11*
*Author: APilot Team*
*Issue: [#15](https://github.com/tangcent/apilot/issues/15)*
