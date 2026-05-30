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
