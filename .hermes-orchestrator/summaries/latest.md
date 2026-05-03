# Latest Summary

**Project:** TwelveReader
**Branch:** `ui`
**Updated At:** 2026-05-03T18:03:39Z

## Current state
- The affected short sample book no longer needs manual segmentation; it needed voice/persona mapping.
- `book_1777829925814889525` is now `synthesized` with 1 segment.
- Persona `narrator` is mapped to voice `serena`.
- Live audio endpoint returns `audio/wav` and the player route serves HTML on port 3002.

## Fixes completed
- Backend hybrid pipeline now requests initial voice mapping even when a short book finishes segmentation before `MinSegmentsBeforeTTS`.
- Backend no longer overwrites `voice_mapping` back to `synthesizing` while waiting for the initial mapping.
- Web client status polling now continues during `voice_mapping`.
- OpenAI-compatible TTS provider now detects actual audio bytes (`wav`, `mp3`, `ogg`, `flac`) instead of always storing responses as `.mp3`.
- Qwen3-TTS submodule now lists custom voices from cached config without loading the 1.7B model just to serve `/v1/voices`, reducing 4GB GPU OOM risk.

## Validation passed
- `docker run --rm -v "$PWD":/src -w /src golang:1.24-alpine /usr/local/go/bin/go test ./internal/pipeline ./internal/provider`
- `cd web-client && npm run build`
- `git diff --check`
- Live backend health check passed.
- Live status/personas/audio endpoints passed.
- Live frontend `/player?bookId=book_1777829925814889525` served HTML.

## Deployment
- Rebuilt `twelvereader-backend` and `twelvereader-frontend` images.
- Recreated `twelvereader-backend-prod` on port 8085 and `twelvereader-frontend-prod` on port 3002.

## Push blocker
- Local code is not yet committed/pushed in the root repo.
- Existing environment still lacks GitHub push authentication.
