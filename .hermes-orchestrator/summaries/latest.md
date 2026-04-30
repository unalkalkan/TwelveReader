# Latest Hermes Orchestrator Summary

Generated: 2026-04-30T22:16:00Z

## Current state
- Branch: `ui`.
- Latest bounded run: `wr_20260430_006` for `blg_backend_requeue_failed_segments`.
- Accepted implementation commit: `674e7a6` (`fix(pipeline): requeue failed hybrid TTS segments`).
- OpenCode was attempted with exactly `opencode-go/glm-5.1`; the non-PTY implementation attempt timed out after partial edits.
- Hermes completed/reconciled the implementation directly and used a PTY OpenCode GLM-5.1 run for read-only review.

## Implemented this cycle
- Added per-segment retry and permanent-failure tracking to `internal/pipeline/queue.go`.
- Updated hybrid pipeline TTS workers to requeue failed mapped segments up to a bounded retry budget.
- Added `activeSynthesis` visibility around dequeue/synthesis to avoid premature worker completion while another worker owns a segment that may be requeued.
- Updated hybrid pipeline completion to mark books as `error` when segment synthesis exhausts retries or remains incomplete, instead of silently marking `synthesized`.
- Added focused pipeline tests for queue retry lifecycle, retry-then-success, and retry-budget exhaustion.

## Validation
- `git diff --check`: passed.
- `cd web-client && npx tsc --noEmit`: passed.
- `cd web-client && npm run build`: passed; Expo web export completed with the existing `expo-av` deprecation warning.
- `go test ./internal/pipeline`: blocked because `go` is not installed in this Hermes environment.

## Review
- Review `rv_20260430_006`: approved.
- No secrets or token-bearing remotes observed.
- Origin remote remains `https://github.com/unalkalkan/TwelveReader.git`.

## Remaining backlog
- Backend parser/OCR implementation.
- Persistent character voice knowledge base.
- Live-backend/browser validation for upload, voice mapping, and active pipeline behavior.

## Next action
- Commit accepted retry/requeue implementation and orchestrator records, then continue the next bounded backlog item only if time remains.
