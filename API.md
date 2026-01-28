# TwelveReader Server API Documentation

## Version
Current Version: `0.1.0-milestone4`

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

### GET /api/v1/voices
Returns available TTS voices from all or a specific provider. Useful for mapping voices to book characters/persons.

**Query Parameters:**
- `provider` (optional): Filter voices by provider name (e.g., `openai-tts`)
- `model` (optional): Filter voices by TTS model (e.g., `tts-1`, `tts-1-hd`)

**Response:**
```json
{
  "voices": [
    {
      "id": "alloy",
      "name": "Alloy",
      "languages": ["en"],
      "gender": "neutral",
      "accent": "",
      "description": "A balanced, clear voice",
      "provider": "openai-tts"
    },
    {
      "id": "echo",
      "name": "Echo",
      "languages": ["en"],
      "gender": "male",
      "accent": "american",
      "description": "A confident, professional voice",
      "provider": "openai-tts"
    }
  ],
  "count": 2
}
```

**Voice Object Fields:**
- `id` (string): Provider-specific voice identifier
- `name` (string): Human-readable voice name
- `languages` (array): Supported language codes (ISO-639-1)
- `gender` (string): Voice gender (`male`, `female`, `neutral`, or empty)
- `accent` (string): Regional accent (e.g., `british`, `american`)
- `description` (string): Additional voice description
- `provider` (string): Name of the TTS provider

**Status Codes:**
- `200 OK` - Success
- `404 Not Found` - Specified provider not found
- `503 Service Unavailable` - No TTS providers configured

**Example Usage:**
```bash
# Get all voices from all providers
curl http://localhost:8080/api/v1/voices

# Get voices from specific provider
curl http://localhost:8080/api/v1/voices?provider=openai-tts

# Get voices for specific model
curl http://localhost:8080/api/v1/voices?model=tts-1-hd

# Get voices from specific provider and model
curl "http://localhost:8080/api/v1/voices?provider=openai-tts&model=tts-1-hd"
```

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

## Book Management Endpoints (Milestone 3)

### POST /api/v1/books
Upload a book for processing. Supports TXT, PDF (stub), and ePUB (stub) formats.

**Request:**
- Content-Type: `multipart/form-data`
- Form fields:
  - `file` (required): Book file
  - `title` (optional): Book title
  - `author` (optional): Book author
  - `language` (optional): ISO-639-1 language code (default: "en")

**Response:**
```json
{
  "id": "book_1234567890",
  "title": "Sample Book",
  "author": "John Doe",
  "language": "en",
  "uploaded_at": "2026-01-25T10:00:00Z",
  "status": "uploaded",
  "orig_format": "txt",
  "total_chapters": 0,
  "total_segments": 0
}
```

**Status Codes:**
- `201 Created` - Book uploaded successfully
- `400 Bad Request` - Invalid request or unsupported format
- `500 Internal Server Error` - Server error

---

### GET /api/v1/books/:id
Get book metadata by ID.

**Response:**
```json
{
  "id": "book_1234567890",
  "title": "Sample Book",
  "author": "John Doe",
  "language": "en",
  "uploaded_at": "2026-01-25T10:00:00Z",
  "status": "ready",
  "orig_format": "txt",
  "total_chapters": 5,
  "total_segments": 120
}
```

**Status Codes:**
- `200 OK` - Success
- `404 Not Found` - Book not found

---

### GET /api/v1/books/:id/status
Get processing status for a book.

**Response:**
```json
{
  "book_id": "book_1234567890",
  "status": "segmenting",
  "stage": "segmenting",
  "progress": 60.0,
  "total_chapters": 5,
  "parsed_chapters": 5,
  "total_segments": 45,
  "updated_at": "2026-01-25T10:05:00Z"
}
```

**Status Values:**
- `uploaded` - Book uploaded, waiting for processing
- `parsing` - Extracting text from book
- `segmenting` - Running LLM segmentation
- `voice_mapping` - Waiting for voice assignments
- `ready` - Book is ready for TTS synthesis
- `error` - Processing failed

**Status Codes:**
- `200 OK` - Success
- `404 Not Found` - Book not found

---

### GET /api/v1/books/:id/segments
List all segments for a book.

**Response:**
```json
[
  {
    "id": "seg_00001",
    "book_id": "book_1234567890",
    "chapter": "chapter_001",
    "toc_path": ["Chapter 1"],
    "text": "Sample segment text.",
    "language": "en",
    "person": "narrator",
    "voice_description": "neutral",
    "processing": {
      "segmenter_version": "v1",
      "generated_at": "2026-01-25T10:03:00Z"
    }
  }
]
```

**Status Codes:**
- `200 OK` - Success
- `404 Not Found` - Book not found

---

### POST /api/v1/books/:id/voice-map
Set voice mapping for discovered personas.

**Request:**
```json
{
  "persons": [
    {"id": "narrator", "provider_voice": "voice_1"},
    {"id": "alice", "provider_voice": "voice_2"}
  ]
}
```

**Response:**
```json
{
  "book_id": "book_1234567890",
  "persons": [
    {"id": "narrator", "provider_voice": "voice_1"},
    {"id": "alice", "provider_voice": "voice_2"}
  ]
}
```

**Status Codes:**
- `200 OK` - Voice map saved successfully
- `400 Bad Request` - Invalid request
- `404 Not Found` - Book not found

---

### GET /api/v1/books/:id/voice-map
Get voice mapping for a book.

**Response:**
```json
{
  "book_id": "book_1234567890",
  "persons": [
    {"id": "narrator", "provider_voice": "voice_1"},
    {"id": "alice", "provider_voice": "voice_2"}
  ]
}
```

**Status Codes:**
- `200 OK` - Success
- `404 Not Found` - Voice map not found

---

## TTS and Packaging Endpoints (Milestone 4)

### GET /api/v1/books/:id/stream
Stream book segments as NDJSON (newline-delimited JSON) for progressive playback.

**Query Parameters:**
- `after` (optional): Resume from segment ID - only return segments after this ID

**Response:**
NDJSON stream where each line is a segment with audio URL:
```json
{"id":"seg_00001","book_id":"book_123","text":"First segment.","language":"en","person":"narrator","voice_description":"neutral","timestamps":{"precision":"word","items":[{"word":"First","start":0.0,"end":0.3}]},"audio_url":"/api/v1/books/book_123/audio/seg_00001"}
{"id":"seg_00002","book_id":"book_123","text":"Second segment.","language":"en","person":"narrator","voice_description":"neutral","timestamps":{"precision":"word","items":[{"word":"Second","start":0.0,"end":0.4}]},"audio_url":"/api/v1/books/book_123/audio/seg_00002"}
```

**Status Codes:**
- `200 OK` - Success
- `404 Not Found` - Book not found
- `500 Internal Server Error` - Server error

---

### GET /api/v1/books/:id/download
Download a packaged book as a ZIP archive containing audio files and metadata.

The ZIP contains:
- `manifest.json` - Book metadata and total duration
- `toc.json` - Table of contents with chapter/segment mapping
- `voice-map.json` - Voice persona to provider voice mapping
- `segments/XXX/` - Sharded directories containing audio files and segment metadata

**Response:**
Binary ZIP file

**Status Codes:**
- `200 OK` - Success (ZIP download)
- `404 Not Found` - Book not found
- `500 Internal Server Error` - Book not synthesized or packaging failed

**Example:**
```bash
curl -O http://localhost:8080/api/v1/books/book_123/download
```

---

### GET /api/v1/books/:id/audio/:segmentId
Stream audio for a specific segment.

**Response:**
Binary audio file (format: wav, mp3, ogg, or flac)

**Status Codes:**
- `200 OK` - Success (audio stream)
- `404 Not Found` - Audio file not found
- `500 Internal Server Error` - Server error

**Example:**
```bash
curl http://localhost:8080/api/v1/books/book_123/audio/seg_00001 -o segment.wav
```

---

## Status Values

The book processing pipeline includes these status values:

- `uploaded` - Book uploaded, waiting for processing
- `parsing` - Extracting text from book
- `segmenting` - Running LLM segmentation
- `voice_mapping` - Waiting for voice assignments
- `ready` - Book is ready for TTS synthesis
- `synthesizing` - TTS synthesis in progress (Milestone 4)
- `synthesized` - TTS synthesis completed, book ready for download (Milestone 4)
- `synthesis_error` - TTS synthesis failed (Milestone 4)
- `error` - Processing failed

---

## Future Endpoints (Milestone 6+)

The following endpoints are planned for future milestones:

- Native client (Android) synchronization endpoints
- Re-voicing workflow endpoints
