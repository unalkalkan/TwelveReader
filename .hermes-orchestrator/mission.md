# Mission

**Project Name:** TwelveReader
**Project ID:** `twelvereader`
**Updated At:** 2026-04-30T20:43:00Z

## Objective
TwelveReader converts uploaded books and long-form text into synchronized, character-aware audiobook experiences. A Go backend ingests content, segments it with an OpenAI-compatible LLM, maps discovered personas to TTS voices, synthesizes audio, stores artifacts, and exposes REST/streaming APIs. The Expo/React web client is the primary user interface for upload, monitoring, voice mapping, playback, and library management.

## Why this exists
The product lets a user turn text/PDF/ePUB/TXT content into a listenable multi-voice audiobook workflow using local or hosted LLM/TTS providers. The current `ui` branch is the WIP surface for validating all backend functionality before native/mobile clients are expanded.

## Constraints
- Work on the `ui` branch of `unalkalkan/TwelveReader`.
- Implement directly in Hermes Agent; do not spawn OpenCode workers for this phase.
- Keep the repository git-native and commit accepted work after validation.
- Do not persist secrets from clone URLs or runtime provider credentials.
- Frontend stack: Expo Router, React 19, React Native Web, TypeScript, TanStack Query, Zod.
- Backend stack: Go HTTP server with local/S3 storage and OpenAI-compatible provider abstractions.
- Current Hermes environment lacks `go`; Go tests cannot run here unless Go is installed later.
- UI should follow the user's default dark/slate/blue control-panel design preference where practical.

## Non-goals
- Do not replace the Go backend architecture in this UI milestone.
- Do not build native Android/Kotlin clients in this branch.
- Do not require a specific hosted provider; provider integration must remain configurable.
- Do not add fake secrets or hardcoded private endpoints.

## Success definition
A working MVP has a compiling web client, durable orchestration state, a clear backlog, upload/library/voice browsing/player flows that connect to real API contracts, a usable voice-mapping flow for books waiting on persona assignment, resilient error/empty/loading states, and committed checkpoints that can be resumed by Hermes Orchestrator.
