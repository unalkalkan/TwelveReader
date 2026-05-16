# Project Design

**Project Name:** TwelveReader
**Project ID:** `twelvereader`
**Updated At:** 2026-05-06T18:49:55Z

This is the canonical design state for TwelveReader on the `ui` branch.

## Purpose
TwelveReader is an AI audiobook generation and playback system. It turns uploaded source material into segmented, voice-attributed, synchronized text/audio output. The current focus is the web/Expo UI that exercises backend features end-to-end.

## Scope
Current scope includes:
- Book upload from files and typed text.
- Book metadata collection: title, author, language.
- Library and exploration views for uploaded/processed books.
- Voice discovery, search, favorites, recents, and cached preview playback through `/api/v1/voices` and `/api/v1/voices/preview`.
- Player view for synchronized text/audio stream playback through `/api/v1/books/:id/stream` and `/audio/:segmentId`.
- Processing state visibility through `/status` and `/pipeline/status`.
- Voice mapping for discovered personas through `/personas`, `/voice-map`, and `POST /voice-map`.
- Download/export entry points through `/download`.

## Non-goals
- Native Android/Kotlin implementation.
- Full OCR quality (scanned PDF image-to-text) beyond existing parser contracts.
- EPUB enhancement beyond current paragraph-level extraction.
- Provider-specific credential management UI.
- Social/community discovery features unless later requested.
- A separate orchestration service inside the app; project orchestration remains in `.hermes-orchestrator/`.

## Current Direction
Prioritize short-term core Qwen3-TTS UX hardening through Step 4 only. Skip quality/production rollout, medium-term enhancements, and any vLLM-omni/workstation changes for now. The app should feel sound from upload through playback: one default voice exists, books begin processing immediately with that voice, and persona-specific remapping can happen later without blocking new synthesis.

## Architecture
- `cmd/server/main.go` exposes a Go HTTP API under `/api/v1` plus health endpoints.
- `internal/api/book_handler.go` owns book upload, status, segments, streaming, voice map, personas, audio, download, and pipeline status endpoints.
- `internal/pipeline` coordinates segmentation, persona discovery, mapping wait states, and synthesis.
- `internal/provider` abstracts OpenAI-compatible LLM/TTS/OCR providers.
- `internal/storage` abstracts local/S3 artifact storage, including reusable voice preview sample files under `voice-samples/` and single-user preference state until real accounts exist.
- `web-client/app` contains Expo Router screens and tab navigation.
- `web-client/src/api` contains typed API functions and TanStack Query hooks.
- `web-client/src/store` stores playback/favorites state in AsyncStorage.
- `web-client/src/components` contains reusable player/list components.

## Constraints
- API schemas must match Go JSON contracts; Zod should stay strict where the backend is stable and tolerant where API data can be absent during processing.
- UI status polling must continue while books are processing or waiting for mapping.
- Player state should not auto-play after restore.
- Audio preview/player resources must be unloaded on unmount or switch.
- Voice preview samples should be generated after backend startup/TTS voice discovery when missing, then reused from storage for subsequent preview clicks and across restarts.
- Treat the current app as a single-user system until accounts exist; single-user preferences may persist in storage.
- Upload must not block on persona voice picking. Unknown/new personas default to the current default voice and remain remappable.
- Persona remapping should prioritize fresh/newer synthesis and defer stale old-audio regeneration until after current book synthesis completes.
- Theme should be compact dark/slate/blue: slate backgrounds, blue for primary actions, semantic colors only for status.
- Do not store GitHub PATs or provider API keys in committed files.

## Design Decisions
- **DEC-001 accepted:** Use `.hermes-orchestrator/` as the durable project brain for mission, design, backlog, feedback, state, worker runs, reviews, and summaries.
- **DEC-002 accepted:** Continue direct Hermes Agent implementation on this phase; no OpenCode worker spawning.
- **DEC-003 accepted:** UI MVP completion is higher priority than deep backend parser/OCR improvements.
- **DEC-004 accepted:** Voice mapping is a first-class UI flow because backend processing can block at `voice_mapping`.
- **DEC-005 accepted:** TypeScript compilation is the primary available frontend quality gate in this environment; Go tests are recorded as blocked until Go is available.
- **DEC-006 accepted:** For now, harden only short-term Qwen3-TTS core UX through Step 4; no quality/production/medium-term or vLLM-omni/workstation work.
- **DEC-007 accepted:** Model account behavior as exactly one user with a persisted default voice until real accounts exist.
- **DEC-008 accepted:** Book processing should auto-map personas to the default voice so segmentation/synthesis starts immediately; later persona remaps mark old audio stale and prioritize fresh synthesis.

## Human Steering Incorporated
- `fb_20260430_001`: Initialize Hermes Orchestrator, assess WIP `ui` branch, create backlog, implement missing pieces directly, and continue until deadline or working product.
- `fb_20260505_voice_samples`: Voice preview playback must use persisted test samples generated after backend startup/TTS connection instead of synthesizing on every click.
- `fb_20260506_short_step1_core_tts_config` through `fb_20260506_short_step4_remap_stale_resynthesis`: Focus short-term only, add single-user default voice, start upload synthesis immediately with the default voice, and support non-blocking remap/stale regeneration semantics.

## Open Questions
- Which deployment target should be optimized first: static web export behind nginx, Expo dev server, or native mobile preview?
- Should URL import and camera scan remain placeholders until backend support exists, or should client-side fetch/OCR be introduced?
- Should backend Go be installed in this Hermes environment to run Go tests, or should validation run on another host/CI?

## Change Log
- 2026-04-30T20:43:00Z — Initial real design assessment for the `ui` branch after repository inspection.
- 2026-05-05T09:13:01Z — Added persistent cached voice preview sample behavior to backend scope/storage constraints.
- 2026-05-06T18:49:55Z — Added short-term Qwen3-TTS default voice and remap/stale regeneration direction from project-owner feedback.
