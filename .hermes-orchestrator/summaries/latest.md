# Latest Hermes Orchestrator Summary

Generated: 2026-04-30T21:49:00Z

## Current state
- Branch: `ui`.
- Accepted implementation commit: `ffc5c46` (`fix(provider): add bounded retries for OpenAI-compatible calls`).
- Orchestrator checkpoint was committed after the implementation checkpoint.
- UI MVP remains validated.
- Latest bounded run: `wr_20260430_005` for backend provider timeout/retry hardening.
- OpenCode was attempted with exactly `opencode-go/glm-5.1`; it timed out after partial edits, so Hermes completed the bounded task directly and recorded the fallback.

## Implemented this cycle
- Added `internal/provider/retry.go` with shared retry option parsing, capped exponential backoff, context-aware retry sleep, and reusable JSON POST request construction.
- Updated OpenAI-compatible LLM provider calls to parse `max_retries` and `retry_backoff_ms`, rebuild request bodies per attempt, retry transport/429/5xx failures, and preserve parsed non-retryable API errors.
- Updated OpenAI-compatible TTS provider speech calls with the same bounded retry behavior.
- Added provider tests covering retry option parsing, transient status retry success, non-retryable 4xx no-retry behavior, and context-stopped retry backoff for LLM and TTS.
- Split the remaining persistent segment requeue/resume work into `blg_backend_requeue_failed_segments`.

## Validation
- `git diff --check`: passed.
- `cd web-client && npx tsc --noEmit`: passed.
- `cd web-client && npm run build`: passed; Expo web export completed with the existing `expo-av` deprecation warning.
- `go test ./internal/provider ./internal/tts ./internal/pipeline`: blocked because `go` is not installed in this Hermes environment.

## Review
- Review `rv_20260430_005`: approved.
- No secrets or token-bearing remotes observed.
- Origin remote remains `https://github.com/unalkalkan/TwelveReader.git`.

## Remaining backlog
- Requeue failed segmentation and synthesis work.
- Backend parser/OCR implementation.
- Persistent character voice knowledge base.
- Live-backend/browser validation for upload, voice mapping, and active pipeline behavior.

## Next action
- Continue `blg_backend_requeue_failed_segments` only after selecting it as a separate bounded task; Go validation should run first on a host with Go installed.
