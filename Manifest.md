# Manifest

## Vision
Twelve Reader transforms static books into fully voiced, time-aligned experiences. A Golang backend parses any PDF/ePUB/TXT input, delegates segmentation and annotation to an LLM, and hands rich segments to pluggable TTS providers (starting with Qwen3-TTS Voice Design Mode). An Android client consumes synchronized audio+text bundles either via streaming or fully packaged downloads.

## Segment Requirements
- Every synthesized chunk carries `Text to Synthesize`, `Language`, `Person`, and `Voice Description` so that the TTS runtime knows exactly what to render.
- Segments are created per paragraph by default, but dialogue exchanges split into per-speaker turns to keep characters distinct.
- Speaker attribution, tone suggestions, and contextual hints come from the LLM using sliding windows of nearby paragraphs.
- Segment metadata remains configurable so future engines with different knobs can reuse the same pipeline.

## Operational Goals
- Support both streaming (segment-by-segment) and batch (whole book) pipelines without code changes.
- Persist synchronized audio+text artifacts alongside JSON metadata so the reader UI can highlight text at word-level accuracy.
- Keep people/voice mapping decoupled: the pipeline annotates `Person`, while uploaders later bind those people to available TTS voices.
- Handle mixed-language works by letting each segment declare its detected language.

## Reference Documents
- High-level architecture: [SystemDesign.md](SystemDesign.md)
- Data and packaging formats: [DataFormats.md](DataFormats.md)
- Detailed Q&A outcomes: [Session01_Answers.md](Session01_Answers.md)
- Delivery plan: [Milestones.md](Milestones.md)