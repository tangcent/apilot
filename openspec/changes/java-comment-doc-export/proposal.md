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
