# TwelveReader

## Product Overview

TwelveReader is an open-source AI audiobook platform. It turns uploaded books into segmented, voice-attributed, synchronized audio/text experiences and is evolving from a single-user generation app into a proper SaaS plus self-hostable server ecosystem.

The product direction is:

- Hosted TwelveReader SaaS with accounts, quotas, billing, admin/support tooling, and managed TTS generation.
- Self-hosted TwelveReader servers that users can choose from the client before login.
- Mobile-first client built with the current web-native stack.
- Private user libraries by default.
- Explore as a collection of public audiobook repositories.
- Official TwelveReader repository limited to public-domain books.
- User public repositories for completed/exportable books.

## Current SaaS Roadmap

The canonical SaaS manifest and roadmap lives in [docs/SAAS_MANIFEST.md](docs/SAAS_MANIFEST.md).

Current milestone sequence:

0. SaaS Readiness Baseline
1. Identity, Sessions, and Ownership Foundation
2. Account-Aware Client and Private Library
3. Usage Metering and Quota Foundation
4. Lazy Generation and Job Management
5. Admin Dashboard
6. Internal Plans, Credits, and Voice Catalog
7. Paid Hosted SaaS
8. Explore and Public Repository
9. Mobile Auth, Private Repos, and Production Hardening

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
- Web client and Admin Dashboard remain the initial implementation surfaces for testing SaaS functionality before native mobile clients.

## Technology Constraints
- LLM and TTS clients must speak the OpenAI-compatible protocol to simplify provider integration.
- Configuration files define provider-specific limits (max segment size, concurrency, latency tolerance) without code edits.
- Word-level timestamps are preferred; the system gracefully degrades to sentence-level when TTS engines cannot supply finer detail.

## Getting Started

### Server Setup
See [SERVER.md](SERVER.md) for detailed server setup instructions.

### Web Client Setup
The Web Client MVP is located in the `web-client/` directory.

**Prerequisites:**
- Node.js 18+ and npm
- Running TwelveReader server on port 8080

**Quick Start:**
```bash
cd web-client
npm install
npm run dev
```

The web client will be available at `http://localhost:3000`.

See [web-client/README.md](web-client/README.md) for detailed documentation.

**Features:**
- Book upload with metadata (title, author, language)
- Real-time processing status monitoring
- Audio playback with synchronized text display
- Built with React + TypeScript + Tamagui + TanStack Query + Zod
- Future-ready for desktop (Electron/Tauri) and mobile (React Native/Expo) ports


