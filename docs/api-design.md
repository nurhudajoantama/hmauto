# API Design

All endpoints return JSON. No HTML.

## Response envelope

```json
// success
{"message": "success", "data": {...}}

// error
{"message": "message", "error": "details"}
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

GET /v1/states/{type}/batch?key=server_1&key=server_2
  → 200 {"message":"success","data":[{"type":"switch","key":"server_1","value":"on","description":"...","updated_at":"..."}]}
  → 400 {"message":"at least one key query parameter is required"}

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

State changes are published to RabbitMQ by this repo. Any MQTT topic consumed by microcontrollers is expected to come from an external bridge or separate service.
