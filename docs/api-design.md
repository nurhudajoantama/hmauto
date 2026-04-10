# API Design

All endpoints return JSON. No HTML.

## Response envelope

```json
// success
{"success": true, "data": {...}}

// error
{"success": false, "error": "message"}
```

Use `internal/response.SuccessResponse` and `internal/response.ErrorResponse`.

## Authentication

### Bearer token (protected routes)

- Header: `Authorization: Bearer {token}`
- Validated against `config.Security.BearerToken`
- On failure: `401 {"success":false,"error":"Unauthorized"}`
- Applied to: `/v1/states/*`

### MCP query token

- Query parameter: `?token={token}`
- Validated against `config.Security.MCPToken`
- On failure: `401 {"success":false,"error":"Unauthorized"}`
- Applied to: `/mcp`

## Public endpoints

```
GET /healthz   → 200 "OK"
GET /health    → 200/503 JSON:
                 {"status":"healthy","timestamp":"...","dependencies":{"redis":"healthy","rabbitmq":"healthy"}}
GET /ready     → 200 "ready" | 503 "not ready"
GET /live      → 200 "alive"
GET /metrics   → Prometheus text format
```

## Protected — hmstt

```
GET /v1/states
  → 200 {"success":true,"data":[{"type":"switch","key":"modem_switch","value":"on","updated_at":"..."},...]}

GET /v1/states/{type}
  → 200 {"success":true,"data":[...]}
  → 404 {"success":false,"error":"no states found for type"} if type has no entries

GET /v1/states/{type}/{key}
  → 200 {"success":true,"data":{"type":"switch","key":"modem_switch","value":"on","updated_at":"..."}}
  → 404 {"success":false,"error":"state not found"}

PUT /v1/states/{type}/{key}
  Body: {"value":"on"}
  → 200 {"success":true,"data":{"type":"switch","key":"modem_switch","value":"on","updated_at":"..."}}
  → 400 {"success":false,"error":"INVALID TYPE OR KEY"} — invalid type/key/value combination
  → 400 {"success":false,"error":"value is required"} — empty value
```

Currently valid type+value combinations (enforced in `canTypeChangedWithKey`):
- type `switch`, values: `on` | `off`

## MCP endpoint

```
POST /mcp
  → MCP streamable HTTP endpoint
  → requires `?token={config.Security.MCPToken}`
```
