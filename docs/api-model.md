# ApiEndpoint JSON Schema Reference

This document describes the canonical JSON representation of `ApiEndpoint` — the shared data model that flows between collectors, the engine, and formatters.

---

## ApiEndpoint

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | yes | Human-readable endpoint name |
| `folder` | string | no | Grouping folder (maps to Postman folder / Markdown section) |
| `description` | string | no | Full description of the endpoint |
| `tags` | string[] | no | Arbitrary tags for filtering |
| `path` | string | yes | URL path (e.g. `/users/{id}`) or gRPC method path |
| `method` | string | no | HTTP method (`GET`, `POST`, etc.); empty for non-HTTP |
| `protocol` | string | yes | Protocol identifier: `"http"`, `"grpc"`, `"websocket"` |
| `parameters` | ApiParameter[] | no | Input parameters |
| `headers` | ApiHeader[] | no | HTTP headers |
| `requestBody` | ApiBody | no | Request body schema |
| `response` | ApiBody | no | Response body schema |
| `metadata` | object | no | Protocol-specific extensions (arbitrary key-value) |

---

## ApiParameter

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | yes | Parameter name |
| `type` | string | yes | `"text"` or `"file"` |
| `required` | boolean | yes | Whether the parameter is required |
| `in` | string | yes | Location: `"query"`, `"path"`, `"header"`, `"cookie"`, `"body"`, `"form"` |
| `default` | string | no | Default value |
| `description` | string | no | Parameter description |
| `example` | string | no | Example value |
| `enum` | string[] | no | Allowed values |

---

## ApiHeader

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | yes | Header name |
| `value` | string | no | Static header value |
| `description` | string | no | Header description |
| `example` | string | no | Example value |
| `required` | boolean | yes | Whether the header is required |

---

## ApiBody

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `mediaType` | string | no | MIME type (e.g. `"application/json"`) |
| `schema` | object | no | JSON Schema object describing the body structure |
| `example` | any | no | Example body value |

---

## Example

```json
{
  "name": "Create Order",
  "folder": "Orders",
  "description": "Creates a new order for the authenticated user.",
  "tags": ["orders", "write"],
  "path": "/orders",
  "method": "POST",
  "protocol": "http",
  "parameters": [],
  "headers": [
    { "name": "Authorization", "value": "Bearer {token}", "required": true }
  ],
  "requestBody": {
    "mediaType": "application/json",
    "schema": {
      "type": "object",
      "properties": {
        "productId": { "type": "string" },
        "quantity": { "type": "integer" }
      }
    },
    "example": { "productId": "abc123", "quantity": 2 }
  },
  "response": {
    "mediaType": "application/json",
    "example": { "orderId": "xyz789", "status": "pending" }
  }
}
```
