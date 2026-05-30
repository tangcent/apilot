# Comet Design Handoff

- Change: java-comment-doc-export
- Phase: design
- Mode: compact
- Context hash: a434b04795ac43c817cf9e31045b07a08cbca71f07909358910dcb3d6113c9f6

Generated-by: comet-handoff.sh

OpenSpec remains the canonical capability spec. This handoff is a deterministic, source-traceable context pack, not an agent-authored summary.

## openspec/changes/java-comment-doc-export/proposal.md

- Source: openspec/changes/java-comment-doc-export/proposal.md
- Lines: 1-24
- SHA256: 2da11dee33fc183c10ae29892849f184d65fa225555a4d9bf5bb219a246e5b71

```md
## Why

Java Spring MVC APIs often carry their public documentation in JavaDoc and Swagger/OpenAPI annotations, but the Java collector currently exports endpoint and schema structure without those descriptions. Markdown and other formatters already have model fields for descriptions, so filling them from source annotations makes generated API docs significantly more useful without changing formatter contracts.

## What Changes

- Add Java Spring MVC documentation extraction from JavaDoc comments and common Swagger/OpenAPI annotations.
- Populate existing model description fields for endpoints, parameters, request bodies, and response bodies.
- Apply precedence consistently: Swagger/OpenAPI annotations first, JavaDoc second, code names as fallback.
- Keep the first version limited to Spring MVC; JAX-RS and Feign behavior remains unchanged.
- Do not introduce runtime reflection or new Java dependency scanning requirements.

## Capabilities

### New Capabilities
- `java-comment-doc-export`: Java Spring MVC collector exports API documentation text from JavaDoc and Swagger/OpenAPI annotations.

### Modified Capabilities

## Impact

- Affected modules: `api-collector-java`, `api-model` tests as needed, and formatter tests only if existing Markdown rendering needs coverage for newly populated fields.
- Public collector output gains richer values in existing JSON fields; no breaking schema change is intended.
- CLI and plugin protocol remain compatible because the data is carried in existing `ApiEndpoint`, `ApiParameter`, and `FieldModel` fields.
```

## openspec/changes/java-comment-doc-export/design.md

- Source: openspec/changes/java-comment-doc-export/design.md
- Lines: 1-45
- SHA256: 0b0028c84fe69da2659b2548a1b0ef3342658405cf31c5d65c8e22f3574a92f9

```md
## Context

The project uses a three-layer pipeline: collectors parse source code, `api-master` orchestrates, and formatters render existing `api-model` structures. The model already includes documentation-oriented fields: `ApiEndpoint.Description`, `ApiParameter.Description`, `ApiParameter.Example`, `FieldModel.Comment`, `FieldModel.Demo`, `FieldModel.Options`, and related required/default metadata. Markdown rendering already consumes these fields.

The Java collector parses source with tree-sitter, then framework-specific packages transform parsed classes into endpoints. Spring MVC currently extracts paths, methods, parameters, request schemas, and response schemas, but parsed classes, methods, parameters, and fields do not retain JavaDoc comments. Annotation extraction also captures only simple string values today, which is enough for many mapping annotations but not enough for richer Swagger/OpenAPI metadata unless normalized deliberately.

## Goals / Non-Goals

**Goals:**
- Export Spring MVC endpoint descriptions from `@Operation`, `@ApiOperation`, and method JavaDoc.
- Export parameter descriptions, examples, required flags, defaults, and enum options from `@Parameter`, `@Schema`, JavaDoc `@param`, and existing Spring annotations.
- Export request/response field descriptions and examples from `@Schema`, `@ApiModelProperty`, and field JavaDoc.
- Reuse existing model fields so formatters receive richer data without new public contracts.
- Preserve existing collection behavior when comments or documentation annotations are absent.

**Non-Goals:**
- Add JAX-RS or Feign documentation extraction in the first version.
- Resolve documentation from compiled dependency JARs.
- Implement full JavaDoc HTML rendering or every Swagger/OpenAPI attribute.
- Add new formatter-specific configuration.

## Decisions

1. Documentation is normalized inside `api-collector-java`, not in formatters.
   - Rationale: comments are language/framework-specific source facts. Formatters should continue rendering canonical model fields.
   - Alternative considered: parse annotations in Markdown formatter metadata. This would duplicate Java semantics and leave other formatters without documentation.

2. Precedence is `Swagger/OpenAPI annotation > JavaDoc > code fallback`.
   - Rationale: Swagger/OpenAPI annotations intentionally define the public API contract, while JavaDoc often includes implementation-facing prose.
   - Alternative considered: concatenate both sources. This can produce noisy duplicated docs and unstable output.

3. JavaDoc becomes parser metadata on classes, methods, parameters, and fields.
   - Rationale: framework parsers and type resolvers need structured access to comments at different source locations.
   - Alternative considered: scan raw source again in Spring MVC only. This would duplicate AST traversal and make body schema field comments harder to attach consistently.

4. Field documentation is attached during Java type resolution.
   - Rationale: request and response body tables are generated from `ObjectModel`/`FieldModel`, so the resolver is the natural place to enrich fields.
   - Alternative considered: post-process object models in Spring MVC. That creates extra lookup paths and risks diverging from generic type resolution.

## Risks / Trade-offs

- [Risk] Tree-sitter JavaDoc association can attach the wrong comment if implemented with broad sibling scanning. -> Mitigation: only accept a block comment immediately preceding the declaration, allowing modifiers/annotations between the comment and declaration.
- [Risk] Swagger/OpenAPI annotations support many attributes and nested forms. -> Mitigation: cover common scalar attributes first and ignore unsupported shapes rather than failing collection.
- [Risk] Existing annotation parser only extracts string literals for named values. -> Mitigation: extend it narrowly for booleans, enum-like identifiers, and simple arrays needed by documentation annotations.
- [Risk] Required semantics differ between Spring, Swagger, and validation annotations. -> Mitigation: preserve existing Spring behavior and let explicit documentation annotations override only when present.
```

## openspec/changes/java-comment-doc-export/tasks.md

- Source: openspec/changes/java-comment-doc-export/tasks.md
- Lines: 1-17
- SHA256: d5aa36f3448e5ecb7b024aab528f0c67e1745a6dede2e4ea45cfe0e755b0fd3d

```md
## 1. Parser Metadata

- [ ] 1.1 Add JavaDoc fields to parsed class, method, parameter, and field models.
- [ ] 1.2 Extract immediately preceding JavaDoc comments during tree-sitter traversal.
- [ ] 1.3 Extend annotation value parsing for common documentation scalar and array attributes.

## 2. Documentation Normalization

- [ ] 2.1 Add Java documentation helpers for JavaDoc cleanup, tag parsing, and supported Swagger/OpenAPI annotation precedence.
- [ ] 2.2 Populate Spring MVC endpoint and parameter documentation from annotations and JavaDoc.
- [ ] 2.3 Populate resolved object field comments, examples, required flags, defaults, and options from field annotations and JavaDoc.

## 3. Verification

- [ ] 3.1 Add Spring MVC collector fixtures covering endpoint, parameter, request body, and response body documentation.
- [ ] 3.2 Add focused parser tests for JavaDoc and documentation annotation extraction.
- [ ] 3.3 Run `go test ./...` for `api-collector-java` and relevant formatter/model modules.
```

## openspec/changes/java-comment-doc-export/specs/java-comment-doc-export/spec.md

- Source: openspec/changes/java-comment-doc-export/specs/java-comment-doc-export/spec.md
- Lines: 1-45
- SHA256: 18387b3072a607e72ec3a40e08adf25eac17c7684ad411738059a5f0ab01a3ce

```md
## ADDED Requirements

### Requirement: Spring MVC endpoint descriptions
The Java collector SHALL populate Spring MVC endpoint descriptions from documentation annotations or JavaDoc.

#### Scenario: Operation annotation takes precedence
- **WHEN** a Spring MVC handler method has both JavaDoc and `@Operation(summary = "...", description = "...")`
- **THEN** the collected endpoint description uses the OpenAPI annotation text before JavaDoc text

#### Scenario: JavaDoc method fallback
- **WHEN** a Spring MVC handler method has JavaDoc but no supported documentation annotation
- **THEN** the collected endpoint description uses the JavaDoc summary text

### Requirement: Spring MVC parameter documentation
The Java collector SHALL populate Spring MVC parameter descriptions, examples, required flags, defaults, and enum values from supported parameter annotations and JavaDoc.

#### Scenario: Parameter annotation metadata
- **WHEN** a Spring MVC handler parameter has `@Parameter(description = "...", example = "...", required = true)`
- **THEN** the collected API parameter includes the description, example, and required values from the annotation

#### Scenario: JavaDoc param fallback
- **WHEN** a Spring MVC handler parameter has no supported documentation annotation and the method JavaDoc contains `@param <name> ...`
- **THEN** the collected API parameter description uses the matching JavaDoc param text

### Requirement: Spring MVC body field documentation
The Java collector SHALL populate request and response body field documentation from supported field annotations and JavaDoc.

#### Scenario: Schema field annotation
- **WHEN** a request or response DTO field has `@Schema(description = "...", example = "...", required = true)`
- **THEN** the collected field model includes the description, demo value, and required value from the annotation

#### Scenario: ApiModelProperty fallback
- **WHEN** a request or response DTO field has `@ApiModelProperty(value = "...")`
- **THEN** the collected field model comment uses the annotation value

#### Scenario: JavaDoc field fallback
- **WHEN** a request or response DTO field has JavaDoc but no supported documentation annotation
- **THEN** the collected field model comment uses the JavaDoc summary text

### Requirement: Non-disruptive behavior
The Java collector SHALL preserve existing Spring MVC endpoint collection when documentation metadata is absent or unsupported.

#### Scenario: No documentation metadata
- **WHEN** a Spring MVC source file has no JavaDoc and no supported documentation annotation
- **THEN** the collector still returns the same endpoint paths, methods, parameters, and schemas as before
```

