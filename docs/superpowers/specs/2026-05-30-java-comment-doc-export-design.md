---
comet_change: java-comment-doc-export
role: technical-design
canonical_spec: openspec
archived-with: 2026-05-30-java-comment-doc-export
status: final
---

# Java Comment Documentation Export Design

## Summary

Spring MVC collection will export API documentation from JavaDoc and common Swagger/OpenAPI annotations into the existing canonical model fields. The first version is intentionally scoped to Spring MVC and avoids public schema changes: formatters receive richer `Description`, `Comment`, `Example`, `Demo`, `Required`, `Default`, and `Options` values through the fields they already understand.

## Scope

Supported endpoint sources:
- `@Operation(summary, description)`
- `@ApiOperation(value, notes)`
- Method JavaDoc summary

Supported parameter sources:
- `@Parameter(description, required, example, schema)`
- `@Schema(description, required, example, allowableValues, defaultValue)`
- JavaDoc `@param`
- Existing Spring annotation defaults such as `@RequestParam(defaultValue = "...")`

Supported field sources:
- `@Schema(description, required, example, allowableValues, defaultValue)`
- `@ApiModelProperty(value, required, example, allowableValues)`
- Field JavaDoc summary

Out of scope for this change:
- JAX-RS and Feign documentation extraction.
- Documentation lookup from compiled dependencies.
- Full JavaDoc HTML rendering.
- New formatter parameters or output schema changes.

## Architecture

The implementation follows the existing collector pipeline.

```
tree-sitter parser
  -> parser.Class / Method / Parameter / Field with JavaDoc + annotations
  -> Spring MVC parser endpoint extraction
  -> resolver.TypeResolver object models
  -> api-model fields
  -> existing formatters
```

Parser-level JavaDoc fields keep comments close to source declarations. Spring MVC then handles endpoint and method parameter documentation because it understands handler semantics. The type resolver handles DTO field documentation because it already converts Java fields into `FieldModel` values for request and response schemas.

## Precedence

When multiple sources describe the same item, the exported value uses this order:

1. Swagger/OpenAPI annotation value.
2. JavaDoc value.
3. Existing code-derived fallback, such as method or field names.

Values are not concatenated. This keeps generated documentation deterministic and avoids duplicate prose.

## Data Mapping

Endpoint documentation:
- Summary and description are normalized into `ApiEndpoint.Description`.
- If both summary and description are present and distinct, they are joined with a blank line.

Parameter documentation:
- Description maps to `ApiParameter.Description`.
- Example maps to `ApiParameter.Example`.
- Required maps to `ApiParameter.Required` when explicitly documented.
- Default maps to `ApiParameter.Default`.
- Enum values map to `ApiParameter.Enum`.

Body field documentation:
- Description maps to `FieldModel.Comment`.
- Example maps to `FieldModel.Demo`.
- Required maps to `FieldModel.Required`.
- Default maps to `FieldModel.DefaultValue`.
- Enum values map to `FieldModel.Options`.

## Error Handling

Unsupported annotation shapes are ignored rather than causing collection failure. JavaDoc parsing is best effort: malformed tags or empty comments simply produce no documentation value. Existing parser behavior around unparseable files remains unchanged.

## Testing

Tests will focus on collector-level behavior because that is the user-visible contract. Parser tests will cover the tricky AST details: JavaDoc association and annotation value parsing. Spring MVC collector fixtures will verify endpoint descriptions, parameter descriptions, request DTO field comments, and response DTO field comments are present in collected output.
