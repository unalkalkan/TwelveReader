# Milestones

| # | Name | Description | Key Deliverables | Status |
| --- | --- | --- | --- | --- |
| 1 | Design Foundations | Consolidate product vision, architecture, data formats, and decision logs. | Updated Manifest/TwelveReader docs, `SystemDesign.md`, `DataFormats.md`, refreshed Q&A record. | Completed |
| 2 | Server Skeleton | Scaffold Golang service with config loading, health checks, provider registry stubs, and storage adapters (local + S3). | Go module, config files, basic REST endpoints, storage abstraction tests. | Completed |
| 3 | Segmentation Pipeline | Implement ingestion, PDF/ePUB/TXT parsing hooks, LLM segmentation worker, and voice mapping API. | Parsers, queue orchestration, persona discovery endpoints, persisted metadata. | Completed |
| 4 | TTS + Packaging | Add TTS orchestrator, timestamp ingestion, streaming feed, and ZIP packaging service with voice-map enforcement. | Provider clients, concurrency controls, NDJSON stream, ZIP builder. | Planned |
| 5 | Android Client MVP | Kotlin app capable of downloading/streaming books, highlighting text via timestamps, and caching offline archives. | Playback UI, sync engine, cache manager, settings for storage/offline mode. | Planned |

Each new agent run should pick the next "Planned" milestone, implement it end-to-end, then update this table to reflect completion status before moving on.
