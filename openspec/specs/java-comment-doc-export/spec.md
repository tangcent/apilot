## Purpose

Java Spring MVC APIs often carry public-facing documentation in JavaDoc and Swagger/OpenAPI annotations. This capability ensures the Java collector exports that documentation through existing canonical model fields so downstream formatters can render richer API documentation without contract changes.

## Requirements

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
