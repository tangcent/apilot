## 1. Parser Metadata

- [x] 1.1 Add JavaDoc fields to parsed class, method, parameter, and field models.
- [x] 1.2 Extract immediately preceding JavaDoc comments during tree-sitter traversal.
- [x] 1.3 Extend annotation value parsing for common documentation scalar and array attributes.

## 2. Documentation Normalization

- [x] 2.1 Add Java documentation helpers for JavaDoc cleanup, tag parsing, and supported Swagger/OpenAPI annotation precedence.
- [x] 2.2 Populate Spring MVC endpoint and parameter documentation from annotations and JavaDoc.
- [x] 2.3 Populate resolved object field comments, examples, required flags, defaults, and options from field annotations and JavaDoc.

## 3. Verification

- [x] 3.1 Add Spring MVC collector fixtures covering endpoint, parameter, request body, and response body documentation.
- [x] 3.2 Add focused parser tests for JavaDoc and documentation annotation extraction.
- [x] 3.3 Run `go test ./...` for `api-collector-java` and relevant formatter/model modules.
