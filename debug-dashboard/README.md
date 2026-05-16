# TwelveReader Debug Dashboard

Standalone Tabler-based live state inspector for the TwelveReader user journey.

## Purpose

This project is separate from the TwelveReader reader UI. It exists only to inspect live system state:

- Book upload/processing lifecycle
- Segmentation state
- Synth/audio readiness
- User read/listen progress perspective
- Segment blockers and playback failures
- Live event feed and health indicators

## Run

```bash
cd debug-dashboard
npm install
npm run dev
```

Default backend origin:

```bash
http://localhost:8080
```

Override it with:

```bash
VITE_TWELVEREADER_API_URL=http://localhost:8080 npm run dev
```

## Build

```bash
npm run build
npm run preview
```

## Live behavior

The dashboard polls the TwelveReader API every 2.5 seconds and updates the page without refresh. If the backend is unavailable or has no books, it falls back to reactive demo telemetry so UI behavior remains inspectable.

## Current data sources

- `GET /health`
- `GET /api/v1/providers`
- `GET /api/v1/books`
- `GET /api/v1/books/:id/status`
- `GET /api/v1/books/:id/segments`
- `GET /api/v1/books/:id/stream`
- `GET /api/v1/books/:id/pipeline/status`
- `GET /api/v1/books/:id/personas`

## Known limitation

The current backend does not expose durable user playback telemetry or explicit synth job records. The dashboard derives user-perspective state from available book, segment, stream, pipeline, and audio fields. Once backend telemetry endpoints exist, plug them into `src/api.ts` and `src/state.ts`.
