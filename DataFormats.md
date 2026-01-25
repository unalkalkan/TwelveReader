# Data Formats

## Segment Metadata (JSON)
Each audio file lives beside a JSON document. Example:
```json
{
  "id": "seg_00042",
  "bookId": "book_abc",
  "chapter": "03",
  "tocPath": ["Part I", "Chapter 3"],
  "text": "\"Where are you going?\" she asked.",
  "language": "en",
  "person": "alice",
  "voiceDescription": "Urgent, whispered curiosity",
  "timestamps": {
    "precision": "word",
    "items": [
      {"word": "Where", "start": 0.00, "end": 0.21},
      {"word": "are", "start": 0.21, "end": 0.32}
    ]
  },
  "sourceContext": {
    "prevParagraphId": "para_00041",
    "nextParagraphId": "para_00043"
  },
  "processing": {
    "segmenterVersion": "v1",
    "ttsProvider": "qwen3",
    "generatedAt": "2026-01-25T10:00:00Z"
  }
}
```

### Required Fields
| Field | Description |
| --- | --- |
| `id` | Stable segment identifier referenced by clients and packaging.
| `text` | Exact textual content sent to the TTS engine.
| `language` | ISO-639-1 code determined per segment.
| `person` | Logical speaker identifier discovered by the LLM.
| `voiceDescription` | Tone/style guidance for the provider.
| `timestamps` | Word or sentence timestamps; precision flag guides the client.
| `processing` | Audit metadata (versions, provider, timestamps).

### Optional Fields
- `tocPath`: Hierarchical breadcrumbs for UI navigation.
- `sourceContext`: Enables re-voicing or debugging with neighboring paragraphs.
- `voiceId`: Filled after the uploader maps personas to provider voices.

## Audio File Naming
`{bookId}/{chapter}/{segmentId}.wav` (or provider-specific extension). The adjacent metadata file uses the same base name with `.json`.

## Offline Package Layout (ZIP)
```
book-{bookId}.zip
├── manifest.json
├── toc.json
├── segments/
│   ├── 000/
│   │   ├── seg_00000.wav
│   │   └── seg_00000.json
│   └── 001/
│       └── ...
└── assets/
    └── cover.jpg
```
- **`manifest.json`**: High-level metadata (title, author, language, total duration, checksum list).
- **`toc.json`**: Ordered chapter/section tree pointing to segment ranges.
- **`segments/`**: Sharded directories (e.g., 100 segments per folder) holding audio+JSON pairs.
- **`assets/`**: Optional artwork or supplemental text.

## Streaming Manifest
For streaming mode, the server exposes a newline-delimited JSON (NDJSON) feed where each line mirrors the segment metadata plus a signed URL to the audio file. Clients can resume by storing the last processed segment ID.

## Voice Mapping Snapshot
After the uploader completes persona assignments, a `voice-map.json` file is stored:
```json
{
  "bookId": "book_abc",
  "persons": [
    {"id": "narrator", "providerVoice": "warm_narrator_v2"},
    {"id": "alice", "providerVoice": "qwen_female_01"}
  ]
}
```
This file is bundled into offline ZIPs and referenced during re-voicing operations.
