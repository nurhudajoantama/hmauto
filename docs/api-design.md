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

### Regular API keys (protected routes)

- Header: `Authorization: Bearer {key}`
- Validated via Redis: `GET apikey:{key}` — non-nil = valid
- On success: `last_used` updated asynchronously
- On failure: `401 {"success":false,"error":"Unauthorized"}`
- Applied to: `/hmstt/*`, `/hmalert/*`

### Admin key (admin routes)

- Same header format: `Authorization: Bearer {admin_key}`
- Validated against `config.Security.AdminKey` only — no Redis lookup
- Applied to: `/admin/*`
- Admin key must not be stored in or validated via Redis

## Public endpoints

```
GET /healthz   → 200 "OK"
GET /hello     → 200 "Hello, World!"
GET /health    → 200/503 JSON:
                 {"status":"healthy","timestamp":"...","dependencies":{"redis":"healthy","rabbitmq":"healthy"}}
GET /ready     → 200 "ready" | 503 "not ready"
GET /live      → 200 "alive"
GET /metrics   → Prometheus text format
```

## Protected — hmalert

```
POST /hmalert/publish
  Body: {"level":"info|warning|error","message":"...","tipe":"..."}
  → 202 {"success":true,"data":null}

POST /hmalert/publishbatch
  Body: [{"level":"...","message":"...","tipe":"..."}, ...]
  → 202 {"success":true,"data":null}
```

Alert levels: `info`, `warning`, `error`

## Protected — hmstt

```
GET /hmstt/states
  → 200 {"success":true,"data":[{"type":"switch","key":"modem_switch","value":"on","updated_at":"..."},...]}

GET /hmstt/states/{type}
  → 200 {"success":true,"data":[...]}
  → 404 {"success":false,"error":"no states found for type"} if type has no entries

GET /hmstt/state/{type}/{key}
  → 200 {"success":true,"data":{"type":"switch","key":"modem_switch","value":"on","updated_at":"..."}}
  → 404 {"success":false,"error":"state not found"}

POST /hmstt/state/{type}/{key}
  Body: {"value":"on"}
  → 200 {"success":true,"data":{"type":"switch","key":"modem_switch","value":"on","updated_at":"..."}}
  → 400 {"success":false,"error":"INVALID TYPE OR KEY"} — invalid type/key/value combination
  → 400 {"success":false,"error":"value is required"} — empty value
```

Currently valid type+value combinations (enforced in `canTypeChangedWithKey`):
- type `switch`, values: `on` | `off`

## Admin — API key management

```
GET /admin/apikeys
  → 200 {"success":true,"data":[{"key_hint":"abcd...efgh","label":"iot-1","created_at":"...","last_used":"..."},...]}
  Note: key_hint = first 4 chars + "..." + last 4 chars; full key never returned in list

POST /admin/apikeys
  Body: {"label":"my-device"}
  → 201 {"success":true,"data":{"key":"<full 64-char hex key, returned only once>","label":"my-device","created_at":"..."}}
  → 400 if label is empty

DELETE /admin/apikeys/{key}
  → 200 {"success":true,"data":null}
  → 404 {"success":false,"error":"api key not found"} if key doesn't exist
```

Key generation: `crypto/rand` 32 bytes → hex (64-char string). Implemented in `internal/apikey.RedisStore.CreateKey`.
