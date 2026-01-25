# Session 01 Answers

## Segment Size and Level
Segment size stays dynamic. Each TTS provider configuration exposes `max_segment_size`, but the pipeline favors shorter excerpts when possible so characters feel responsive and latency stays low.

## Speaker Detection
Automatic, LLM-only detection. No manual annotation round is planned; heuristics (quotes, tags) guide prompts, but the final decision is delegated to the LLM.

## Context Window
Segmentation prompts see N surrounding paragraphs to keep tone, language, and speaker guesses consistent.

## Segment Granularity
Paragraph-level by default. Dialogue-heavy sections split into per-turn segments so `Person` changes cleanly align with speech.

## Audio Storage
Storage drivers are pluggable. Local filesystem is the default, with an S3-compatible adapter ready for deployment. DRM/licensing scopes are intentionally deferred.

## Metadata Structure
Every audio file is paired with a JSON sidecar describing segment parameters, timestamps, and provenance.

## Voice Mapping
All segments are prepared upfront without synthesis. Uploaders receive the list of discovered `Person` entities and map them to whatever personas the active TTS provider exposes. Only after mapping does TTS generation begin.

## TTS Quality vs Speed
Handled by each provider. The orchestrator simply respects provider hints rather than attempting additional optimizations.

## Language Detection
Mixed-language books are auto-detected per segment; the `Language` field reflects local detection rather than a single document-level value.

## Sync Precision
Aim for word-level timestamps; fall back to sentence-level if a provider cannot deliver finer granularity.

## Scaling and Performance
Concurrency and rate limiting are server configuration values so deployments can tune throughput per hardware profile or provider quota.

## Cost
No caching layer is necessary because the assumption is self-hosted TTS capacity.

## Offline Support
Clients cache/download complete audiobooks. A standardized ZIP layout packages text, metadata, and audio for easy publishing and later re-voicing.

## Re-voice Capability
Out of scope for milestone one, but the offline ZIP structure keeps all data needed to re-run selected segments with new voice assignments.

## Format Support
Support every major format (PDF/ePUB/TXT/Markdown) plus optional OCR so scans can join the same flow.

## Table of Contents
The ingestion pipeline preserves TOC hierarchy so clients can present chapters/sections alongside synchronized playback.