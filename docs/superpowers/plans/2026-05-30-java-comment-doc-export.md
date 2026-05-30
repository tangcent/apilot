---
change: java-comment-doc-export
design-doc: docs/superpowers/specs/2026-05-30-java-comment-doc-export-design.md
base-ref: bfcd220372c6f1b9e2d2ffefe83c54f19f8a3dc5
archived-with: 2026-05-30-java-comment-doc-export
---

# Java Comment Doc Export Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [x]`) syntax for tracking.

**Goal:** Export Spring MVC API documentation from JavaDoc and Swagger/OpenAPI annotations through existing model fields.

**Architecture:** Extend the Java parser to retain JavaDoc and richer annotation values, add Java documentation normalization helpers, enrich Spring MVC endpoints and resolved field models, then verify via parser and collector tests. No formatter contract changes are planned because Markdown already renders existing description fields.

**Tech Stack:** Go, tree-sitter Java bindings, existing `api-collector-java` parser/resolver/Spring MVC packages, Go test.

## File Structure

- Modify `api-collector-java/parser/types.go`: add JavaDoc fields to `Class`, `Method`, `Parameter`, and `Field`.
- Modify `api-collector-java/parser/extractor.go`: attach immediately preceding JavaDoc comments and parse scalar/array annotation values used by documentation annotations.
- Create `api-collector-java/doc/doc.go`: normalize JavaDoc text, extract `@param` tags, and choose Swagger/OpenAPI values with precedence over JavaDoc.
- Modify `api-collector-java/springmvc/types.go`: add description/example/enum/default fields to intermediate endpoint and parameter types.
- Modify `api-collector-java/springmvc/parser.go`: populate endpoint and parameter documentation.
- Modify `api-collector-java/collector.go`: map Spring MVC intermediate documentation fields into `api-model`.
- Modify `api-collector-java/resolver/resolver.go`: enrich `FieldModel` values from parsed field annotations and JavaDoc.
- Add or modify tests in `api-collector-java/parser/parser_test.go`, `api-collector-java/springmvc/parser_test.go`, `api-collector-java/resolver/resolver_test.go`, and `api-collector-java/collector_test.go`.
- Add `api-collector-java/testdata/DocumentedController.java`, `DocumentedRequest.java`, and `DocumentedResponse.java`.

### Task 1: Parser JavaDoc And Annotation Metadata

**Files:**
- Modify: `api-collector-java/parser/types.go`
- Modify: `api-collector-java/parser/extractor.go`
- Test: `api-collector-java/parser/parser_test.go`

- [x] **Step 1: Write failing parser test**

Add `TestParser_ExtractsDocumentationMetadata` to `api-collector-java/parser/parser_test.go`. The test should write a temporary Java file containing class JavaDoc, method JavaDoc with `@param id user id`, `@Operation(summary = "Fetch user", description = "Fetch user by id")`, `@Parameter(description = "User id", example = "42", required = true)`, and a DTO field with `@Schema(description = "Display name", example = "Ada", required = true, allowableValues = {"Ada", "Grace"})`.

Assert:
- class `JavaDoc == "Documented API"`
- method `JavaDoc == "Fetch by id."`
- method `JavaDocParams["id"] == "fallback id"`
- operation annotation params contain `summary` and `description`
- parameter `JavaDoc == "user id"`
- parameter annotation params contain `description`, `example`, and `required`
- field `JavaDoc == "Display name fallback"`
- schema annotation params contain `description`, `example`, `required`, and `allowableValues`

- [x] **Step 2: Run parser test and verify RED**

Run:

```bash
cd api-collector-java
go test ./parser -run TestParser_ExtractsDocumentationMetadata -count=1
```

Expected: FAIL because parser types do not expose JavaDoc fields and annotation parsing does not preserve all metadata.

- [x] **Step 3: Implement parser metadata**

Add JavaDoc fields:
- `Class.JavaDoc string`
- `Method.JavaDoc string`
- `Method.JavaDocParams map[string]string`
- `Parameter.JavaDoc string`
- `Field.JavaDoc string`

In `extractClass`, `extractInterface`, `extractMethod`, `extractParameter`, and `extractField`, attach the immediately preceding block comment when it starts with `/**`.

Extend annotation parameter parsing to support:
- default positional string literals as `value`
- named string literals
- named booleans as `true` / `false`
- named enum/scoped identifiers as their source text
- named arrays as comma-separated values in source order with quotes stripped

Keep unsupported annotation values as best-effort source text and never fail parsing.

- [x] **Step 4: Run parser tests and verify GREEN**

Run:

```bash
cd api-collector-java
go test ./parser -run TestParser_ExtractsDocumentationMetadata -count=1
go test ./parser -count=1
```

Expected: PASS.

### Task 2: Documentation Normalization Helpers

**Files:**
- Create: `api-collector-java/doc/doc.go`
- Test: `api-collector-java/doc/doc_test.go`

- [x] **Step 1: Write failing helper tests**

Add tests for:
- JavaDoc cleanup removes `/**`, `*/`, leading `*`, blank lines, and tag lines from the summary.
- `ParseJavaDocParams` returns `map[string]string{"id": "user id"}` for `@param id user id`.
- `EndpointDescription` prefers `@Operation(summary, description)` over JavaDoc.
- `ParameterDoc` prefers `@Parameter(description, example, required)` over JavaDoc.
- `FieldDoc` prefers `@Schema(description, example, required, allowableValues, defaultValue)` over `@ApiModelProperty(value)`, then JavaDoc.

- [x] **Step 2: Run helper tests and verify RED**

Run:

```bash
cd api-collector-java
go test ./doc -count=1
```

Expected: FAIL because package `doc` does not exist.

- [x] **Step 3: Implement helper package**

Create `api-collector-java/doc/doc.go` with exported types/functions:
- `type FieldDocumentation struct { Description string; Example string; Required *bool; Default string; Enum []string }`
- `func CleanJavaDoc(raw string) string`
- `func ParseJavaDocParams(raw string) map[string]string`
- `func EndpointDescription(annotations []parser.Annotation, javaDoc string) string`
- `func ParameterDocumentation(annotations []parser.Annotation, javaDoc string) FieldDocumentation`
- `func FieldDocumentationFor(annotations []parser.Annotation, javaDoc string) FieldDocumentation`

The helper package may import `github.com/tangcent/apilot/api-collector-java/parser`.

- [x] **Step 4: Run helper tests and verify GREEN**

Run:

```bash
cd api-collector-java
go test ./doc -count=1
```

Expected: PASS.

### Task 3: Spring MVC Endpoint And Parameter Documentation

**Files:**
- Modify: `api-collector-java/springmvc/types.go`
- Modify: `api-collector-java/springmvc/parser.go`
- Modify: `api-collector-java/collector.go`
- Test: `api-collector-java/springmvc/parser_test.go`
- Test: `api-collector-java/collector_test.go`

- [x] **Step 1: Write failing Spring MVC tests**

Add a Spring MVC parser unit test where a method has JavaDoc and `@Operation`, plus a `@PathVariable` parameter with `@Parameter`. Assert intermediate endpoint description and parameter documentation fields.

Add a collector-level test using `api-collector-java/testdata/DocumentedController.java` that asserts:
- endpoint `Description` is populated from `@Operation`
- path/query parameter `Description`, `Example`, and `Required` are populated
- `@RequestParam(defaultValue = "active")` still maps to `Default` and optional required behavior unless explicitly overridden by documentation annotation

- [x] **Step 2: Run tests and verify RED**

Run:

```bash
cd api-collector-java
go test ./springmvc -run Documentation -count=1
go test . -run TestCollect_SpringMVCDocumentation -count=1
```

Expected: FAIL because Spring MVC intermediate and collector mapping do not expose documentation.

- [x] **Step 3: Implement endpoint and parameter mapping**

Update intermediate structs:
- `Endpoint.Description string`
- `EndpointParameter.Description string`
- `EndpointParameter.Example string`
- `EndpointParameter.Enum []string`

In `extractEndpoint`, set description using `doc.EndpointDescription(method.Annotations, method.JavaDoc)`.

In `extractParameter`, combine annotation documentation and method `JavaDocParams[param.Name]`; preserve existing Spring `required` and `defaultValue` behavior, then apply explicit documentation values from supported annotations.

In `springmvcEndpointToAPI`, copy intermediate description/example/enum/default values into `collector.ApiEndpoint` and `collector.ApiParameter`.

- [x] **Step 4: Run Spring MVC and collector tests and verify GREEN**

Run:

```bash
cd api-collector-java
go test ./springmvc -run Documentation -count=1
go test . -run TestCollect_SpringMVCDocumentation -count=1
```

Expected: PASS.

### Task 4: Body Field Documentation

**Files:**
- Modify: `api-collector-java/resolver/resolver.go`
- Test: `api-collector-java/resolver/resolver_test.go`
- Test: `api-collector-java/collector_test.go`
- Testdata: `api-collector-java/testdata/DocumentedRequest.java`
- Testdata: `api-collector-java/testdata/DocumentedResponse.java`

- [x] **Step 1: Write failing resolver and collector field tests**

Add a resolver test with a parsed class containing fields documented by `@Schema`, `@ApiModelProperty`, and JavaDoc. Assert `FieldModel.Comment`, `Demo`, `Required`, `DefaultValue`, and `Options`.

Extend the collector-level documentation test to assert request and response body fields include descriptions and examples from DTO annotations/JavaDoc.

- [x] **Step 2: Run field tests and verify RED**

Run:

```bash
cd api-collector-java
go test ./resolver -run Documentation -count=1
go test . -run TestCollect_SpringMVCDocumentation -count=1
```

Expected: FAIL because field documentation is not applied to `FieldModel`.

- [x] **Step 3: Implement field enrichment**

In `resolver.resolveField`, call `doc.FieldDocumentationFor(f.Annotations, f.JavaDoc)` after existing required defaults. Apply:
- `Comment` from description
- `Demo` from example
- `DefaultValue` from default
- `Options` from enum values as `[]model.FieldOption`
- `Required` override only when the helper returns a non-nil required value

Do not change static/final filtering or generic handling.

- [x] **Step 4: Run field tests and verify GREEN**

Run:

```bash
cd api-collector-java
go test ./resolver -run Documentation -count=1
go test . -run TestCollect_SpringMVCDocumentation -count=1
```

Expected: PASS.

### Task 5: Full Verification And Comet Task Sync

**Files:**
- Modify: `openspec/changes/java-comment-doc-export/tasks.md`
- Modify: any tests that need minor expected-output updates.

- [x] **Step 1: Run full Java collector tests**

Run:

```bash
cd api-collector-java
go test ./...
go vet ./...
```

Expected: PASS.

- [x] **Step 2: Run relevant formatter/model tests**

Run:

```bash
cd api-model
go test ./...
cd ../api-formatter-markdown
go test ./...
```

Expected: PASS.

- [x] **Step 3: Update OpenSpec tasks**

Mark all completed items in `openspec/changes/java-comment-doc-export/tasks.md` with `- [x]`.

- [x] **Step 4: Run build guard**

Run:

```bash
bash .agents/skills/comet/scripts/comet-guard.sh java-comment-doc-export build --apply
```

Expected: PASS and `.comet.yaml` phase transitions to `verify`.
