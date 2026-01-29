# Voices API Implementation Summary

## Overview
Implemented a `/v1/voices` interface in the TTS provider system to fetch available voices from TTS providers. This enables users to see available voices and map them to book characters/persons. The model used to filter voices comes from each provider's configuration, not from the API request.

## Changes Made

### 1. Core Interface Changes

#### [internal/provider/interfaces.go](internal/provider/interfaces.go)
- Added `Voice` type to represent voice metadata with fields:
  - `ID`: Provider-specific voice identifier
  - `Name`: Human-readable name
  - `Languages`: Supported language codes
  - `Gender`: Voice gender (male/female/neutral)
  - `Accent`: Regional accent
  - `Description`: Additional description
- Added `ListVoices(ctx context.Context) ([]Voice, error)` method to `TTSProvider` interface
  - No model parameter needed - uses provider's configured model internally

#### [pkg/types/book.go](pkg/types/book.go)
- Added `Voice` type in the types package for consistent API responses

### 2. Provider Implementations

#### [internal/provider/openai_tts.go](internal/provider/openai_tts.go)
- Implemented `ListVoices()` method for OpenAI TTS provider
- Calls `GET /models/voices` endpoint on the TTS provider
- Uses the model from provider configuration (`o.model`) as query parameter to the API
- Parses voice data and converts to standard `Voice` format
- Handles both `languages` array and single `language` field for compatibility
- Added supporting types:
  - `voicesAPIResponse`: Response structure
  - `voiceData`: Individual voice data from API

#### [internal/provider/stubs.go](internal/provider/stubs.go)
- Implemented `ListVoices(ctx context.Context, model string)` method for stub TTS provider
- Returns 2 test voices for development/testing purposes
- Accepts but doesn't filter by model parameter (stub behavior)

### 3. API Handler

#### [internal/api/voices_handler.go](internal/api/voices_handler.go) (New)
- Created `VoicesHandler` to handle voice-related endpoints
- Implemented `ListVoices` HTTP handler for `GET /api/v1/voices`
- Features:
  - Optional `provider` query parameter to filter by specific provider
  - Aggregates voices from all providers when no provider specified
  - Graceful error handling (continues if one provider fails)
  - Returns JSON response with voices array and count
  - Model filtering is handled internally by each provider based on configuration

### 4. Server Integration

#### [cmd/server/main.go](cmd/server/main.go)
- Registered `/api/v1/voices` endpoint
- Wired up `VoicesHandler` with provider registry

### 5. Documentation

#### [API.md](API.md)
- Added comprehensive documentation for `/api/v1/voices` endpoint
- Includes:
  - Query parameters (`provider`)
  - Response format with example
  - Field descriptions
  - Status codes
  - Usage examples with curl
  - Note about model-based filtering being handled by provider configuration

### 6. Tests

#### [internal/provider/voices_test.go](internal/provider/voices_test.go)
- `TestOpenAITTSProvider_ListVoices`: Tests OpenAI provider voice listing
- `TestOpenAITTSProvider_ListVoicesError`: Tests error handling
- `TestOpenAITTSProvider_ListVoicesWithConfigModel`: Tests that model from config is passed to API
- `TestStubTTSProvider_ListVoices`: Tests stub provider

#### [internal/api/voices_handler_test.go](internal/api/voices_handler_test.go)
- `TestVoicesHandler_ListVoices`: Tests basic functionality
- `TestVoicesHandler_ListVoicesWithProvider`: Tests provider filtering
- `TestVoicesHandler_ListVoicesProviderNotFound`: Tests 404 error
- `TestVoicesHandler_ListVoicesNoProviders`: Tests 503 error
- `TestVoicesHandler_MethodNotAllowed`: Tests HTTP method validation
- `TestVoicesHandler_ListVoicesPartialFailure`: Tests graceful degradation

## API Usage

### Get all voices from all providers
```bash
curl http://localhost:8080/api/v1/voices
```

### Get voices from specific provider
```bash
curl http://localhost:8080/api/v1/voices?provider=openai-tts
```

### Response Format
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
    }
  ],
  "count": 1
}
```

## Benefits

1. **Voice Discovery**: Users can see all available voices from configured TTS providers
2. **Provider Flexibility**: Works with any TTS provider that implements the interface
3. **Configuration-Based Filtering**: Voices are automatically filtered by the model specified in each provider's configuration
4. **Character Mapping**: Enables mapping specific voices to book characters/persons
5. **Graceful Degradation**: If one provider fails, others still return results
6. **Simple API**: No need to specify model in API requests - handled internally per provider

## Testing

All tests pass successfully:
```bash
go test ./...
# All packages: PASS
```

## Next Steps

To use this in the voice mapping workflow:
1. User uploads a book
2. System segments the book and identifies persons/speakers
3. User calls `/api/v1/voices` to see available voices
4. User sets voice mappings via `/api/v1/books/:id/voice-map` endpoint (already implemented)
5. System uses the mapped voices during TTS synthesis
