# Plugin Subprocess Protocol

External collector and formatter plugins communicate with `api-master` via stdin/stdout using newline-delimited JSON.

---

## Collector Protocol

### Metadata Queries

Collectors can optionally support the `--supported-languages` flag to report their capabilities:

```bash
./my-collector --supported-languages
```

Expected output (JSON array of strings):

```json
["java", "kotlin"]
```

If the flag is not supported or fails, the method returns `nil`.

### Input (stdin)

`api-master` writes a single JSON object representing `CollectContext`:

```json
{
  "sourceDir": "/path/to/project",
  "frameworks": ["spring-mvc"],
  "config": {
    "key": "value"
  }
}
```

When the user exports source files rather than the whole directory, `sourceDir`
is the root of the project the files belong to, and the files are named in
`sourceFiles`, as absolute paths:

```json
{
  "sourceDir": "/path/to/project",
  "sourceFile": "/path/to/project/src/UserController.java",
  "sourceFiles": ["/path/to/project/src/UserController.java"]
}
```

`sourceFile` is present only when exactly one file is selected, so a plugin that
reads only `sourceFile` never takes one file of several for the whole selection.
Both keys are absent for a directory export.

A plugin may ignore them and return every endpoint of the project: `api-master`
keeps the endpoints whose `folder` is named after a selected file, without its
extension. A plugin that reads the whole project, which supplies the types the
files refer to, but builds endpoints only for the declarations of the selected
files, does less work.

### Output (stdout)

The plugin writes a JSON array of `ApiEndpoint` objects:

```json
[
  {
    "name": "Get User",
    "folder": "Users",
    "path": "/users/{id}",
    "method": "GET",
    "protocol": "http",
    "parameters": [
      { "name": "id", "in": "path", "type": "text", "required": true }
    ]
  }
]
```

An empty result is `[]`.

### Exit codes

- `0` — success
- non-zero — failure; write a human-readable error message to stderr

---

## Formatter Protocol

### Metadata Queries

Formatters can optionally support the `--supported-formats` flag to report their capabilities:

```bash
./my-formatter --supported-formats
```

Expected output (JSON array of strings):

```json
["markdown", "json", "yaml"]
```

If the flag is not supported or fails, the method returns `nil`.

### Input (stdin)

`api-master` writes a single JSON envelope:

```json
{
  "endpoints": [ /* []ApiEndpoint */ ],
  "options": {
    "format": "simple",
    "config": {}
  }
}
```

### Output (stdout)

The plugin writes the raw formatted bytes (Markdown text, JSON, etc.) directly to stdout. No JSON wrapping.

### Exit codes

- `0` — success
- non-zero — failure; write a human-readable error message to stderr

---

## ApiEndpoint JSON Schema

See [api-model.md](api-model.md) for the full field reference.

---

## Testing a Plugin Manually

```bash
# Test a collector plugin
echo '{"sourceDir":"/path/to/project"}' | ./my-collector-binary

# Test a formatter plugin
echo '{"endpoints":[],"options":{"format":"simple"}}' | ./my-formatter-binary
```
