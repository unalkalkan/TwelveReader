# Twelve Reader

## Product Overview
Twelve Reader is a split application: a Golang server orchestrates book ingestion, LLM-driven segmentation, voice assignment, and TTS synthesis, while a Web client (React + TypeScript) provides the interface for testing and using all server functionalities. The Web client streams or downloads synchronized audio-text experiences and will be followed by native clients (Android/Kotlin) in future milestones. Both server and client are designed to swap LLM/TTS/storage providers via configuration so first-party hardware and hosted services can coexist.

## System Architecture
```
User Upload -> Ingestion Service -> LLM Segmentation -> Voice Mapping UI -> TTS Orchestrator
		  |                                                        |
		  v                                                        v
	Storage API ------------------------------------------> Packaging Service -> Client Delivery
```
- **Ingestion Service**: Parses PDF/ePUB/TXT (with optional OCR) into structured chapters/paragraphs and pushes chunks through the pipeline.
- **LLM Segmentation Service**: Calls OpenAI-compatible endpoints to detect speakers, languages, and voice descriptions using configurable context windows.
- **Voice Mapping UI**: Surfaces discovered `Person` entries so uploaders can map them to available TTS personas before synthesis kicks off.
- **TTS Orchestrator**: Streams per-segment requests to pluggable engines (OpenAI-style API contract) with max segment sizes and concurrency determined by provider configs.
- **Storage Abstraction**: Ships with local filesystem support and an S3 backend for audio artifacts plus sidecar JSON metadata.
- **Packaging Service**: Builds distributable ZIP bundles with aligned text/audio/timestamps for offline consumption and future re-voicing.

## Server Responsibilities (Golang)
- Provide REST/gRPC APIs for uploads, processing status, voice mapping, and artifact retrieval.
- Manage workers for ingestion, LLM segmentation, and TTS synthesis with configurable concurrency and rate limiting per provider.
- Persist metadata in JSON (and optionally relational indices) plus binary audio in storage adapters.
- Offer streaming endpoints that push segments as soon as their audio becomes available as well as batch endpoints that deliver final ZIPs.
- Support pluggable OCR and text parsers to cover PDF, ePUB, TXT, and Markdown inputs while preserving table-of-contents structure.

## Client Responsibilities (Web/React, then Android/Kotlin)
- Display synchronized text with highlighting driven by word/sentence timestamps from metadata.
- Handle both streaming playback (progressively fetch segments) and fully downloaded ZIP archives for offline listening.
- Provide interfaces for book upload, voice mapping, and playback control.
- Cache packaged books locally with integrity checks so re-voicing workflows can re-use existing metadata.
- Surface voice mapping summaries to users (read-only in v1) and allow preference toggles (e.g., narrator vs dialogue balance) for future releases.
- Web client (Milestone 5) serves as the initial implementation for testing all functionalities before native mobile clients (Milestone 6+).

## Technology Constraints
- LLM and TTS clients must speak the OpenAI-compatible protocol to simplify provider integration.
- Configuration files define provider-specific limits (max segment size, concurrency, latency tolerance) without code edits.
- Word-level timestamps are preferred; the system gracefully degrades to sentence-level when TTS engines cannot supply finer detail.


