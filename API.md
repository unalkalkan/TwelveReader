# TwelveReader Server API Documentation

## Version
Current Version: `0.1.0-milestone2`

## Base URL
Default: `http://localhost:8080`

## Health Endpoints

### GET /health/live
Liveness check endpoint. Returns `200 OK` if the server process is running.

**Response:**
```json
{
  "status": "healthy",
  "timestamp": "2026-01-25T10:00:00Z",
  "version": "0.1.0-milestone2"
}
```

**Status Codes:**
- `200 OK` - Server is running

---

### GET /health/ready
Readiness check endpoint. Returns `200 OK` if the server is ready to serve traffic.
Performs checks on storage adapter and provider registry.

**Response:**
```json
{
  "status": "healthy",
  "timestamp": "2026-01-25T10:00:00Z",
  "checks": {
    "storage": {
      "status": "healthy"
    },
    "providers": {
      "status": "healthy"
    }
  },
  "version": "0.1.0-milestone2"
}
```

**Status Codes:**
- `200 OK` - Server is ready
- `503 Service Unavailable` - Server is not ready (unhealthy)

**Health Status Values:**
- `healthy` - Component is functioning normally
- `degraded` - Component is functioning but with reduced capability
- `unhealthy` - Component is not functioning

---

### GET /health
Comprehensive health check endpoint. Returns detailed health information for all components.

**Response:**
Same structure as `/health/ready` but always returns `200 OK` regardless of health status.

---

## API Endpoints

### GET /api/v1/info
Returns basic server information.

**Response:**
```json
{
  "version": "0.1.0-milestone2",
  "storage_adapter": "local"
}
```

**Status Codes:**
- `200 OK` - Success

---

### GET /api/v1/providers
Returns information about registered providers.

**Response:**
```json
{
  "llm": ["openai", "local-llm"],
  "tts": ["qwen3", "openai-tts"],
  "ocr": ["tesseract"]
}
```

**Status Codes:**
- `200 OK` - Success

---

## Configuration

The server is configured via a YAML configuration file. See `config/dev.example.yaml` for a complete example.

### Environment Variables

All configuration values can be overridden with environment variables using the `TR_` prefix:

- `TR_SERVER_HOST` - Server host address
- `TR_SERVER_PORT` - Server port
- `TR_STORAGE_ADAPTER` - Storage adapter type (`local` or `s3`)
- `TR_STORAGE_LOCAL_BASE_PATH` - Local storage base path
- `TR_STORAGE_S3_BUCKET` - S3 bucket name
- `TR_STORAGE_S3_REGION` - S3 region
- `TR_STORAGE_S3_ENDPOINT` - S3 endpoint (for MinIO/custom)
- `TR_STORAGE_S3_ACCESS_KEY_ID` - S3 access key
- `TR_STORAGE_S3_SECRET_ACCESS_KEY` - S3 secret key

Provider-specific environment variables follow the pattern:
- `TR_LLM_<NAME>_API_KEY` - API key for LLM provider
- `TR_LLM_<NAME>_ENDPOINT` - Endpoint for LLM provider
- `TR_TTS_<NAME>_API_KEY` - API key for TTS provider
- `TR_TTS_<NAME>_ENDPOINT` - Endpoint for TTS provider
- `TR_OCR_<NAME>_API_KEY` - API key for OCR provider
- `TR_OCR_<NAME>_ENDPOINT` - Endpoint for OCR provider

Where `<NAME>` is the uppercase version of the provider name (e.g., `OPENAI`, `QWEN3`).

---

## Building and Running

### Build
```bash
make build
```

### Test
```bash
make test
```

### Run
```bash
make run
# or
./bin/twelvereader -config config/dev.example.yaml
```

### Development Mode
```bash
make dev
```

---

## Error Responses

All error responses follow this format:

```json
{
  "error": "Error message describing what went wrong",
  "code": "ERROR_CODE"
}
```

Common HTTP status codes:
- `200 OK` - Success
- `400 Bad Request` - Invalid request
- `404 Not Found` - Resource not found
- `500 Internal Server Error` - Server error
- `503 Service Unavailable` - Service temporarily unavailable

---

## Future Endpoints (Milestone 3+)

The following endpoints are planned for future milestones:

- `POST /api/v1/books` - Upload a book for processing
- `GET /api/v1/books/:id` - Get book metadata
- `GET /api/v1/books/:id/status` - Get processing status
- `GET /api/v1/books/:id/segments` - List segments
- `POST /api/v1/books/:id/voice-map` - Set voice mapping
- `GET /api/v1/books/:id/download` - Download packaged book
- `GET /api/v1/books/:id/stream` - Stream segments (NDJSON)
