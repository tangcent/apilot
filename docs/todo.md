# APilot — Implementation TODO

> Tasks are grouped by module. Each task is self-contained and can be assigned independently.
> See [architecture.md](architecture.md) and [refactor-design.md](refactor-design.md) for context before picking up a task.

---

## Core Engine

### api-master

- [X] Implement `RunCLI()` in `engine/engine.go` — flag parsing for `--collector`, `--formatter`, `--format`, `--output`, `--plugin-registry`, `--list-collectors`, `--list-formatters`
- [X] Implement auto-detection logic in `engine/engine.go` — scan source dir for indicator files (`pom.xml`, `go.mod`, `package.json`, etc.) and select the matching collector
- [X] Implement `plugin/registry.go` — load `plugins.json` from default path (`~/.config/apilot/plugins.json`) or `--plugin-registry` flag, register valid entries, log and skip invalid ones
- [X] Implement `plugin/subprocess.go` — `SupportedLanguages()` / `SupportedFormats()` via subprocess `--supported-languages` / `--supported-formats` flags
- [X] Implement `plugin/dynlib.go` — shared library (`.so`/`.dylib`/`.dll`) loader via CGO `dlopen` (v2, deferred)
- [X] Write unit tests for `engine/engine.go` — happy path, missing collector, missing formatter, auto-detect
- [X] Write unit tests for `plugin/registry.go` — valid registry, missing binary, unknown type, empty file

### apilot-cli

- [ ] Implement `--help` output listing all registered collectors and formatters
- [ ] Write integration test — invoke `apilot-cli` on a fixture source directory and assert output

---

## Collectors

### api-collector (interface module)

- [ ] Write round-trip JSON tests for `ApiEndpoint`, `ApiParameter`, `ApiHeader`, `ApiBody` — `serialize → deserialize` must produce equivalent value

### api-collector-go

- [ ] Implement `gin/parser.go` — walk Go AST for `gin.RouterGroup` / `gin.Engine` `.GET/.POST/.PUT/.DELETE/.PATCH` call expressions, extract path string literals and handler doc comments
- [ ] Implement `echo/parser.go` — same pattern for `echo.Echo` / `echo.Group`
- [ ] Implement `fiber/parser.go` — same pattern for `fiber.App` / `fiber.Router`
- [ ] Wire parsers into `collector.go` `Collect()` — walk `.go` files, delegate to framework parsers, merge results
- [ ] Write unit tests for each parser with fixture Go source files

### api-collector-java

- [ ] Spike: evaluate Java parsing strategy — pure Go regex/heuristic vs. `tree-sitter-java` CGO bindings vs. subprocess `javac` (document decision in `api-collector-java/NOTES.md`)
- [ ] Implement `springmvc/parser.go` — extract `@RestController`, `@RequestMapping`, `@GetMapping`, `@PostMapping`, `@PutMapping`, `@DeleteMapping`, `@PatchMapping`, `@PathVariable`, `@RequestParam`, `@RequestBody`, Javadoc
- [ ] Implement `jaxrs/parser.go` — extract `@Path`, `@GET`, `@POST`, `@PUT`, `@DELETE`, `@Produces`, `@Consumes`
- [ ] Implement `feign/parser.go` — extract `@FeignClient` interfaces and their method annotations
- [ ] Implement `maven/resolver.go` — invoke `maven-indexer-cli` subprocess to resolve dependency JARs from `pom.xml` / `build.gradle`
- [ ] Wire parsers into `collector.go` `Collect()` — discover `.java`/`.kt` files, optionally resolve deps, delegate to framework parsers
- [ ] Write unit tests for each parser with fixture Java source files

#### Code Quality Improvements (MEDIUM Priority)

- [ ] **Fix error handling in ParseDirectory** — `parser_v2.go:135` ignores ParseFile errors, should collect and report failed files
- [ ] **Fix error handling in ParseDirectoryParallel** — `parser_v2.go:188` ignores ParseFile errors, should collect and report failed files
- [ ] **Add input validation** — `parser_v2.go:51,110,143` should validate empty paths, relative paths, and non-.java files
- [ ] **Fix encapsulation in ParseDirectoryParallel** — `parser_v2.go:179` directly accesses `cache.cacheDir` and `logger.level` private fields, use getter methods or pass ParserOptions
- [ ] **Validate worker count** — `parser_v2.go:143` ParseDirectoryParallel should validate `workers > 0` to prevent deadlock

### api-collector-node

- [ ] Spike: evaluate Node.js parsing strategy — `tree-sitter-typescript` CGO bindings vs. pure-Go JS/TS AST (e.g. `goja`) (document decision in `api-collector-node/NOTES.md`)
- [ ] Implement `express/parser.go` — extract `app.get`, `app.post`, `router.use`, etc. route registrations
- [ ] Implement `fastify/parser.go` — extract `fastify.get`, `fastify.post`, etc. route registrations
- [ ] Implement `nestjs/parser.go` — extract `@Controller`, `@Get`, `@Post`, `@Put`, `@Delete`, `@Patch` decorated classes and methods
- [ ] Wire parsers into `collector.go` `Collect()` — discover `.ts`/`.js` files, delegate to framework parsers
- [ ] Write unit tests for each parser with fixture TypeScript/JavaScript source files

### api-collector-python

- [ ] Implement `fastapi/parser.go` — extract `@app.get`, `@router.post`, etc. decorated functions using `tree-sitter-python` or regex
- [ ] Implement `django/parser.go` — extract `@api_view`, `APIView`, `ViewSet` classes and `urlpatterns` definitions
- [ ] Implement `flask/parser.go` — extract `@app.route`, `@blueprint.route` decorated functions
- [ ] Wire parsers into `collector.go` `Collect()` — discover `.py` files, delegate to framework parsers
- [ ] Write unit tests for each parser with fixture Python source files

---

## Formatters

### api-formatter-markdown

- [ ] Implement `simple.md.tmpl` — one section per endpoint: method + path headline, description, parameter table
- [ ] Implement `detailed.md.tmpl` — full schema expansion: request/response body as JSON code block, tags, headers
- [ ] Write unit tests — empty input returns valid empty Markdown, known endpoint produces expected output

### api-formatter-curl

- [ ] Implement path parameter substitution — replace `{param}` with `<param>` placeholder in the URL
- [ ] Implement form parameter support — add `--data-urlencode` flags for `in: form` parameters
- [ ] Write unit tests — empty input returns empty bytes, endpoint with headers/body/query params produces correct `curl` command

### api-formatter-postman

- [ ] Implement path parameter extraction — convert `{param}` to `:param` in Postman URL path segments
- [ ] Implement response example support — populate Postman example responses from `ApiEndpoint.Response`
- [ ] Write unit tests — empty input returns valid empty Postman collection, round-trip: produce → parse → produce produces equivalent JSON

---

## VSCode Extension

- [ ] Implement `vscode-plugin/src/extension.ts` — register `apilot.export` command, resolve source dir from right-click URI or active editor
- [ ] Implement `vscode-plugin/src/runner.ts` — spawn `apilot-cli` subprocess, stream stdout to output channel, show error notification on non-zero exit
- [ ] Implement `vscode-plugin/src/binaryResolver.ts` — resolve bundled platform binary path, throw descriptive error if not found
- [ ] Implement `vscode-plugin/src/settings.ts` — typed wrapper for `apilot.formatter`, `apilot.outputDestination`, `apilot.outputFile`, `apilot.binaryPath`
- [ ] Update `vscode-plugin/package.json` — rename all `easyapi.*` contribution keys to `apilot.*`, update command title to "APilot: Export"
- [ ] Add `vscode-plugin/tsconfig.json` — configure TypeScript compilation
- [ ] Write VSCode extension tests — mock subprocess, assert output channel content and error notification behavior
- [ ] Add CI workflow `ci-vscode.yml` — `npm ci && npm run compile && npm test`

---

## npm Wrapper

- [ ] Test `scripts/install.js` on all platforms (darwin/linux/win32 × amd64/arm64) — verify binary downloads, extracts, and is executable
- [ ] Test `scripts/run.js` — verify args are forwarded correctly, verify error message when binary is missing
- [ ] Add `.npmignore` — exclude source files, keep only `scripts/`, `bin/`, `CHANGELOG.md`, `package.json`

---

## Release & CI

- [ ] Add `build.Version` / `build.Date` ldflags wiring in `apilot-cli` — create `apilot-cli/build/version.go` with `Version` and `Date` vars
- [ ] Verify `.goreleaser.yml` produces correct archive names (`apilot-{version}-{os}-{arch}.tar.gz`) matching what `scripts/install.js` expects
- [ ] Add `ci-vscode.yml` workflow — build and test the VSCode extension on push/PR
- [ ] Add `NPM_TOKEN` secret to GitHub repo for npm publish step in `release.yml`
- [ ] Test full release flow end-to-end with a `v0.1.0` tag — GoReleaser produces archives, npm publish succeeds, `npm install -g @tangcent/apilot` downloads and runs correctly

---

## Documentation

- [ ] Update `docs/architecture.md` — fix CI pipeline table to reflect actual workflow filenames (`ci.yml`, `co.yml`)
- [ ] Update `docs/architecture.md` — fix config dir reference from `~/.config/api-master/` to `~/.config/apilot/`
- [ ] Update `docs/architecture.md` — fix build commands to use `apilot` binary name (not `apilot-cli`)
- [X] Remove `docs/apilot-readme-draft.md` — content has been merged into `README.md`
- [X] Add `docs/contributing-collectors.md` — step-by-step guide for adding a new language collector
- [X] Add `docs/contributing-formatters.md` — step-by-step guide for adding a new output formatter
