# API Design

All endpoints return JSON. No HTML. No HTMX.

## Response envelope

Use `internal/response` for consistent shape:

```json
// success
{"success": true, "data": {...}}

// error
{"success": false, "error": "message"}
```

## Authentication

### Regular API keys

- Header: `Authorization: Bearer {key}`
- Middleware validates: `GET apikey:{key}` from Redis → non-null = authorized
- On successful auth: update `last_used` in background (non-blocking)
- On failure: `401 Unauthorized`

### Admin key

- Same header format: `Authorization: Bearer {admin_key}`
- Validated against `config.Security.AdminKey` (not Redis) — config only
- Applied only to `/admin/*` subrouter
- Admin key must NOT be stored in Redis (separate code path)

## Endpoints

### Public (no auth)

```
GET  /healthz         → 200 "OK"
GET  /hello           → 200 "Hello, World!"
GET  /health          → 200 JSON {redis: ok/fail, rabbitmq: ok/fail}
GET  /ready           → 200/503 based on dependency status
GET  /live            → 200 always (liveness)
GET  /metrics         → Prometheus text format
```

### Protected — hmalert

```
POST /hmalert/publish
  Body: {"level":"info|warning|error", "message":"...", "tipe":"..."}
  → 202 {"success":true, "data":{"queued":true}}

POST /hmalert/publishbatch
  Body: [{"level":"...","message":"...","tipe":"..."}, ...]
  → 202 {"success":true, "data":{"queued":N}}
```

### Protected — hmstt

```
GET  /hmstt/states
  → 200 {"success":true, "data":[{"type":"switch","key":"modem_switch","value":"on","updated_at":"..."},...]}

GET  /hmstt/states/{type}
  → 200 {"success":true, "data":[...]}
  → 404 if type has no states

GET  /hmstt/state/{type}/{key}
  → 200 {"success":true, "data":{"type":"switch","key":"modem_switch","value":"on","updated_at":"..."}}
  → 404 if not found

POST /hmstt/state/{type}/{key}
  Body: {"value":"on"}
  → 200 {"success":true, "data":{"type":"switch","key":"modem_switch","value":"on","updated_at":"..."}}
  → 400 if invalid type/key/value
```

### Admin — API key management

```
GET  /admin/apikeys
  → 200 {"success":true, "data":[{"key_hint":"abc...xyz","label":"iot-1","created_at":"...","last_used":"..."},...]}
  Note: never return full key value in list

POST /admin/apikeys
  Body: {"label":"my-device"}
  → 201 {"success":true, "data":{"key":"<full key — only returned once>","label":"my-device","created_at":"..."}}

DELETE /admin/apikeys/{key}
  → 200 {"success":true}
  → 404 if key not found
```

Key generation: `crypto/rand` → 32 bytes → hex string (64 chars). Never use `math/rand`.

## Removed endpoints

These are deleted as part of HTMX removal:
- `GET  /hmstt/` (HTML index)
- `GET  /hmstt/statehtml/{type}/{key}` (HTMX fragment)
- `POST /hmstt/setstatehtml/{type}/{key}` (HTMX form handler)
- `GET  /hmstt/getstatevalue/{type}/{key}` → replaced by `GET /hmstt/state/{type}/{key}`

## Route registration pattern

```go
// in RegisterHandlers:
protected := srv.GetRouter().PathPrefix("/hmstt").Subrouter()
srv.ApplyAuthMiddleware(protected)  // attaches Redis-backed auth

protected.HandleFunc("/states", h.listAllStates).Methods("GET")
protected.HandleFunc("/states/{type}", h.listStatesByType).Methods("GET")
protected.HandleFunc("/state/{type}/{key}", h.getState).Methods("GET")
protected.HandleFunc("/state/{type}/{key}", h.setState).Methods("POST")
```

Admin routes — new `server.ApplyAdminMiddleware(subrouter)` method:
```go
admin := srv.GetRouter().PathPrefix("/admin").Subrouter()
srv.ApplyAdminMiddleware(admin)  // uses config AdminKey
admin.HandleFunc("/apikeys", h.listKeys).Methods("GET")
admin.HandleFunc("/apikeys", h.createKey).Methods("POST")
admin.HandleFunc("/apikeys/{key}", h.revokeKey).Methods("DELETE")
```
