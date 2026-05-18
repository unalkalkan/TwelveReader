# /api/v1 Compatibility Boundary

## Status: Milestone 0 — Endpoints Implemented (2026-05-18)

This document defines the current `/api/v1` API surface and establishes the compatibility boundary for TwelveReader's versioned API. All future SaaS-facing endpoints live under `/api/v1`.

---

## Current API Surface Inventory

### Health Endpoints (outside /api/v1 — legacy placement)

| Method | Path | Handler | Description |
|--------|------|---------|-------------|
| GET | `/health/live` | `healthHandler.LivenessHandler()` | Liveness probe. Always 200 if running. |
| GET | `/health/ready` | `healthHandler.ReadinessHandler()` | Readiness probe. Checks storage + providers. |
| GET | `/health` | `healthHandler.HealthHandler()` | Detailed health. Always 200. |

**Note:** These are not under `/api/v1`. The SAAS_MANIFEST specifies a `GET /api/v1/health` endpoint for Milestone 0, which is now implemented as a structured, versioned health endpoint with component-level status checks.

---

### System Information Endpoints

| Method | Path | Handler | Description |
|--------|------|---------|-------------|
| GET | `/api/v1/info` | `infoHandler(version, cfg)` | Basic server info: version, storage_adapter. |
| GET | `/api/v1/providers` | `providersHandler(registry)` | Registered providers by type: llm, tts, ocr. |

**Note:** The SAAS_MANIFEST specified a `GET /api/v1/server-info` endpoint for Milestone 0, which is now implemented. It provides detailed server info including version, environment, uptime, providers, pipeline workers, and feature flags — extending the minimal `/api/v1/info`.

---

### Voice Endpoints

| Method | Path | Handler | Description |
|--------|------|---------|-------------|
| GET | `/api/v1/voices` | `voicesHandler.ListVoices` | List available TTS voices. Optional `?provider=` filter. |
| GET | `/api/v1/voices/default` | `voicesHandler.DefaultVoice` | Get/set default narration voice (GET returns, PUT sets). |
| PUT | `/api/v1/voices/default` | `voicesHandler.DefaultVoice` | Set default narration voice. |
| POST | `/api/v1/voices/preview` | `voicesHandler.PreviewVoice` | Generate short TTS preview audio (base64). |

---

### Book Endpoints

All book routes are multiplexed through two `HandleFunc` registrations in main.go:
- `/api/v1/books` — collection-level operations
- `/api/v1/books/` — per-book operations (path suffix determines sub-route)

| Method | Path | Handler | Description |
|--------|------|---------|-------------|
| POST | `/api/v1/books` | `bookHandler.UploadBook` | Upload book file (multipart). Returns created book. |
| GET | `/api/v1/books` | `bookHandler.ListBooks` | List all books. |
| DELETE | `/api/v1/books/{id}` | `bookHandler.DeleteBook` | Delete a book by ID. |
| GET | `/api/v1/books/{id}` | `bookHandler.GetBook` | Get book metadata. |
| GET | `/api/v1/books/{id}/status` | `bookHandler.GetBookStatus` | Get processing status. |
| GET | `/api/v1/books/{id}/segments` | `bookHandler.ListSegments` | List all segments for a book. |
| POST | `/api/v1/books/{id}/voice-map` | `bookHandler.SetVoiceMap` | Set voice persona mapping. |
| GET | `/api/v1/books/{id}/voice-map` | `bookHandler.GetVoiceMap` | Get voice persona mapping. |
| GET | `/api/v1/books/{id}/personas` | `bookHandler.GetPersonas` | Get discovered personas for a book. |
| GET | `/api/v1/books/{id}/stream` | `bookHandler.StreamSegments` | NDJSON stream of segments with audio URLs. Optional `?after=`. |
| GET | `/api/v1/books/{id}/download` | `bookHandler.DownloadBook` | Download packaged book as ZIP archive. |
| GET | `/api/v1/books/{id}/audio/{segmentId}` | `bookHandler.GetAudio` | Stream audio for a specific segment. |
| GET | `/api/v1/books/{id}/pipeline/status` | `bookHandler.GetPipelineStatus` | Get pipeline processing status. |

---

### Debug Endpoints

| Method | Path | Handler | Description |
|--------|------|---------|-------------|
| GET | `/api/v1/debug/events` | `debugHandler.Events` | Debug event log. |
| GET | `/api/v1/debug/stream` | `debugHandler.EventStream` | SSE stream of debug events. |
| GET | `/api/v1/debug/books/{id}/synth-jobs` | `debugHandler.ListSynthJobs` | List synthesis jobs for a book. |
| GET | `/api/v1/debug/books/{id}/audio-validation` | `debugHandler.AudioValidation` | Audio validation results for a book. |
| GET | `/api/v1/debug/books/{id}/playback-events` | `debugHandler.PlaybackEvents` | Playback event log for a book. |
| GET | `/api/v1/debug/books/{id}/user-progress` | `debugHandler.UserProgress` | User progress tracking for a book. |
| GET | `/api/v1/debug/books/{id}/events` | `debugHandler.Events` | Debug events scoped to a book. |
| GET | `/api/v1/debug/books/{id}/stream` | `debugHandler.EventStream` | SSE stream scoped to a book. |

---

## Total Endpoint Count: 33

- Health (non-versioned): 3
- System info: 2
- Voices: 4
- Books: 14
- Debug: 8 (5 unique handlers, 3 are book-scoped variants)
- **Under `/api/v1`:** 30 endpoints (added 3 Milestone 0 endpoints)

---

## /api/v1 Compatibility Rules

### What is covered by the compatibility guarantee

All endpoints under `/api/v1` are subject to these rules:

1. **URL stability.** Path segments and query parameters defined in this document will not change within the v1 lifecycle.
2. **HTTP method stability.** The method (GET/POST/PUT/DELETE) for each path is fixed.
3. **Response envelope stability.** Once structured error responses are implemented (Milestone 0), all endpoints must conform to it.
4. **Status code semantics.** HTTP status codes with their documented meanings are stable.

### What is NOT covered

1. **Field additions** to response objects may be added without version bump.
2. **Internal fields** prefixed with `_` or `internal_` may change freely.
3. **Debug endpoints** under `/api/v1/debug/` are exempt from stability guarantees unless explicitly promoted.
4. **Response timing**, rate limits, and pagination defaults may change.

### Endpoint naming conventions

- Collection: plural nouns (`/books`, `/voices`)
- Single resource: `/{collection}/{id}` (`/books/{id}`)
- Sub-resources: `/{collection}/{id}/{subresource}` (`/books/{id}/segments`)
- Actions on resources: HTTP method discrimination (POST to collection = create, POST to sub-resource = action)
- Health/system endpoints: lowercase kebab-case nouns

### Error response format

Current error responses use a structured format (implemented in Milestone 0):
```json
{
  "error": {
    "code": "string",       // machine-readable error code
    "message": "string",    // human-readable description
    "request_id": "string"  // correlation ID for debugging (from X-Request-ID)
  }
}
```

**Defined error codes:** `NOT_FOUND`, `METHOD_NOT_ALLOWED`, `BAD_REQUEST`, `UNAUTHORIZED`, `FORBIDDEN`, `CONFLICT`, `INTERNAL_SERVER_ERROR`, `SERVICE_UNAVAILABLE`, `TOO_MANY_REQUESTS`.

**Helper functions** (in `internal/api/structured_error.go`):
- `WriteStructuredError(w, r, code, message, statusCode)` — write a structured error
- `WriteMethodNotAllowedError(w, r)` — 405 response
- `WriteNotFoundError(w, r, resource)` — 404 response with resource name
- `HTTPStatusCodeForCode(code)` — map error code to HTTP status

### Request ID (implemented)

Every `/api/v1` response includes an `X-Request-ID` header. The middleware is applied globally via a sub-mux mounted at `/api/v1`. Clients may provide their own `X-Request-ID` in the request to enable request tracing across services. Error responses also embed the request ID in the error body.

---

## Milestone 0 Additions (per SAAS_MANIFEST)

These endpoints have been added under `/api/v1`:

| Method | Path | Description | Status |
|--------|------|-------------|--------|
| GET | `/api/v1/health` | Structured health check with component status. Returns 200 when healthy, 503 when unhealthy. | Implemented |
| GET | `/api/v1/server-info` | Detailed server info: version, environment, uptime, providers, pipeline workers, feature flags. | Implemented |
| GET | `/api/v1/features` | Feature flag status. Returns all configured features with enabled/disabled state. | Implemented |

### Response formats

**`GET /api/v1/health`** — Health check response:
```json
{
  "status": "healthy",            // "healthy" | "degraded" | "unhealthy"
  "timestamp": "2026-05-18T...",
  "checks": {
    "storage":  {"status": "healthy"},
    "providers": {"status": "healthy"}
  },
  "version": "0.1.0-milestone4"
}
```

**`GET /api/v1/server-info`** — Server information:
```json
{
  "version":           "0.1.0-milestone4",
  "environment":       "local",            // "local" | "dev" | "staging" | "production"
  "uptime_seconds":    123.45,
  "storage_adapter":   "local",
  "llm_providers":     ["openai"],
  "tts_providers":     ["qwen3"],
  "ocr_providers":     ["openai-ocr"],
  "pipeline_workers":  4,
  "feature_flags": {
    "saas_auth":      false,
    "usage_metering": false,
    "quota_engine":   false,
    "repository_pub": false,
    "user_accounts":  false,
    "billing":        false
  }
}
```

**`GET /api/v1/features`** — Feature flags:
```json
{
  "features": {
    "saas_auth":      {"name": "saas_auth",      "enabled": false},
    "usage_metering": {"name": "usage_metering", "enabled": false},
    "quota_engine":   {"name": "quota_engine",   "enabled": false},
    "repository_pub": {"name": "repository_pub", "enabled": false},
    "user_accounts":  {"name": "user_accounts",  "enabled": false},
    "billing":        {"name": "billing",        "enabled": false}
  }
}
```

### Request ID support

All `/api/v1` endpoints pass through the request ID middleware (applied globally via sub-mux). Every response includes an `X-Request-ID` header. Clients may provide their own `X-Request-ID` in the request to enable request tracing. The request ID is also embedded in the structured error response body.

Additionally, Milestone 0 introduces cross-cutting concerns applied to ALL `/api/v1` endpoints:
- Request ID middleware (`X-Request-ID` header on all responses) — applied via a single `/api/v1` sub-mux mount point
- Structured error format (as defined below)
- Environment mode labels (`local`, `dev`, `staging`, `production`) exposed in server-info and logs
- Feature flag mechanism

---

## Endpoint groups (for future SaaS expansion)

Per SAAS_MANIFEST, the planned API groupings for later milestones:

| Group | Prefix | Status |
|-------|--------|--------|
| System | `/api/v1/health`, `/api/v1/server-info`, `/api/v1/features` | Implemented (Milestone 0) |
| Voices | `/api/v1/voices/*` | Implemented |
| Books/Library | `/api/v1/books/*` | Implemented (current state; account scoping planned in Milestone 2) |
| Auth | `/api/v1/auth/*` | Not started (Milestone 1) |
| User | `/api/v1/users/*` | Not started (Milestone 1) |
| Upload/Import | `/api/v1/uploads/*` | Not started (Milestone 2) |
| TTS Jobs | `/api/v1/jobs/*` | Not started (Milestone 4) |
| Progress Sync | `/api/v1/progress/*` | Not started (Milestone 2) |
| Usage/Quota | `/api/v1/usage/*`, `/api/v1/quota/*` | Not started (Milestone 3) |
| Billing | `/api/v1/billing/*` | Not started (Milestones 6-7) |
| Repository | `/api/v1/repositories/*` | Not started (Milestones 8-9) |
| Admin | `/api/v1/admin/*` | Not started (Milestone 5) |
| Debug | `/api/v1/debug/*` | Implemented (current state; moves under Admin -> Debug in Milestone 5) |

---

## Verification

### Smoke test commands (against a running server at localhost:8080)

```bash
# Milestone 0: Versioned system endpoints
curl -s http://localhost:8080/api/v1/health
curl -s http://localhost:8080/api/v1/server-info
curl -s http://localhost:8080/api/v1/features

# Health endpoints (existing, non-versioned)
curl -s http://localhost:8080/health/live
curl -s http://localhost:8080/health/ready
curl -s http://localhost:8080/health

# System info
curl -s http://localhost:8080/api/v1/info
curl -s http://localhost:8080/api/v1/providers

# Voices
curl -s http://localhost:8080/api/v1/voices
curl -s http://localhost:8080/api/v1/voices/default

# Books (requires existing books)
curl -s http://localhost:8080/api/v1/books

# Debug
curl -s http://localhost:8080/api/v1/debug/events
```

### Existing automated smoke tests

The file `scripts/e2e-api-smoke.py` contains automated API smoke tests. Run with:
```bash
python scripts/e2e-api-smoke.py
```

---

## Files Modified

### Work 0.1 (API inventory and compatibility boundary)
- Created: `docs/API_V1_BOUNDARY.md` (this document)

### Work 0.2 (Health, server-info, features endpoints)
- Updated: `docs/API_V1_BOUNDARY.md` — documented response formats, updated status
- Code files involved (created before this task):
  - `internal/api/v1_system.go` — V1SystemHandler with HealthHandler, ServerInfoHandler, FeaturesHandler
  - `internal/api/v1_system_test.go` — unit tests for all three endpoints
  - `internal/api/middleware.go` — request ID middleware (RequestContext)
  - `internal/features/flags.go` — feature flag store with HTTPHandler
  - `internal/health/handler.go` — health check handler with RunChecks, status types
  - `cmd/server/main.go` — route registration for all three endpoints
- No code changes in this task — the implementation was already present; this task verified it end-to-end.

### Work 0.3 (Request ID propagation and structured error responses)
- Created: `internal/api/structured_error.go` — StructuredError type, ErrorBody, error codes, helper functions
- Created: `internal/api/structured_error_test.go` — unit tests for all structured error functions
- Updated: `cmd/server/main.go` — applied request ID middleware globally to all /api/v1 routes via sub-mux; replaced plain-text method-not-found with structured error; added respondDebugNotFoundStructured helper
- Updated: `docs/API_V1_BOUNDARY.md` — documented implemented error format, error codes, and helper functions
