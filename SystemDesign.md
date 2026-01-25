# System Design

## Summary
Twelve Reader ingests arbitrary books, asks an LLM to enrich the text with speaker-aware metadata, and hands structured segments to pluggable TTS providers. The resulting audio and metadata are stored side-by-side so Android clients can stream or download fully synchronized experiences.

## Component Overview
- **Ingestion Gateway**: Accepts uploads (PDF/ePUB/TXT/Markdown) and normalizes them into chapters/paragraphs. Optional OCR adapters convert scanned PDFs into text.
- **Content Repository**: Tracks book manifests, chapter trees, and processed segment metadata. Can be a relational DB plus object storage.
- **LLM Segmentation Service**: Calls OpenAI-compatible models to decide segment boundaries, detect speakers, language, and recommend `Voice Description` cues with sliding context windows.
- **Voice Mapping Service**: Aggregates discovered `Person` identifiers and exposes them via an admin UI/API so uploaders can bind personas to provider voices.
- **TTS Orchestrator**: Streams requests to configured providers (Qwen3-TTS first) with respect to each provider's `max_segment_size`, concurrency, and rate limits.
- **Storage Adapters**: Local filesystem by default, S3-compatible implementation ready. Responsible for audio binaries and JSON sidecars.
- **Packaging Service**: Builds offline-ready ZIP bundles with deterministic folder layout for audio, metadata, and table-of-contents data.
- **Client Delivery API**: Serves streaming endpoints (segment feeds) and download endpoints (ZIP bundles, metadata manifests). Initially consumed by the Web client (React + TypeScript), with native clients following in later milestones.

## Processing Pipeline
```
Upload -> Parse -> Segment -> Voice Map -> Synthesize -> Package -> Deliver
```
1. **Upload**: User submits book; ingestion stores raw asset and extracts structure.
2. **Parse**: Format-specific parsers (ePUB, PDF+OCR, TXT, Markdown) convert content to normalized paragraphs with TOC references.
3. **Segment**: LLM receives paragraphs with configurable context to decide segment boundaries, `Person`, `Language`, and `Voice Description`.
4. **Voice Map**: Pipeline pauses while uploader maps `Person` list to available TTS personas; mapping saved per book.
5. **Synthesize**: Orchestrator pushes segments to provider queues, respecting concurrency/rate settings, and records timestamp info.
6. **Package**: Once segments exist, packaging service combines audio + metadata into streams or ZIP bundles.
7. **Deliver**: Client fetches streaming feed or downloads offline package. Re-voicing can revisit steps 4-6 later.

## Streaming vs Batch
- **Streaming Mode**: Segments are published to a message/topic as soon as their audio files arrive. Clients subscribe or poll for incremental playback.
- **Batch Mode**: Pipeline runs to completion, packaging everything into an archive before exposing it.
Both modes reuse the same storage format; the difference lies in delivery timing.

## Configuration Surface
| Setting | Purpose |
| --- | --- |
| `max_segment_size` (per TTS provider) | Keeps requests under provider limits while encouraging shorter utterances.
| `context_window` | Controls how many neighboring paragraphs the LLM sees for speaker/tone consistency.
| `concurrency_limit` | Worker pool size per provider to balance throughput vs. resource usage.
| `rate_limit.qps` | Hard throttle for provider-specific quotas.
| `storage.adapter` | Chooses between `local` and `s3` (more adapters later).
| `ocr.provider` | Selects OCR backend for scanned PDFs.
| `timestamp_precision` | Declares expected granularity (word vs sentence) so clients know how to render highlights.

## Open Questions (Tracked)
- Should the Voice Mapping UI live inside the Golang server or a separate admin portal?
- Do we need a lightweight DB schema draft now or postpone until server scaffolding?
- What observability stack (metrics/logging) best fits the streaming pipeline?
