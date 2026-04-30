# Latest Hermes Orchestrator Summary

Generated: 2026-04-30T22:58:40Z

## Current state
- Branch: `ui`.
- Latest bounded run: `wr_20260430_008` for `blg_backend_parser_ocr`.
- Worker: OpenCode was invoked with exactly `opencode-go/glm-5.1` for implementation and read-only review.
- Review: `rv_20260430_008` approved with follow-up.
- Commit: pending at summary-write time; expected message `feat(parser): add bounded pdf and epub extraction`.

## Implemented this cycle
- Replaced PDF parser placeholder output with bounded extraction for simple uncompressed PDF content streams.
- Added literal-string parsing with escaped parentheses, newline/tab/backslash escapes, octal escape handling, and safer `BT`/`ET` operator boundary detection.
- Replaced EPUB parser placeholder output with ZIP-based extraction using OPF spine order when available.
- Added EPUB fallback to sorted HTML/XHTML files when OPF/container data is missing or unusable.
- Added script/style stripping, HTML tag removal, entity decoding, stable chapter IDs/TOC paths, and decompressed entry/total size limits to reduce zip-bomb risk.
- Added parser tests for EPUB spine order, fallback behavior, sanitization, entity decoding, nested OPF href paths, invalid input, stable IDs, simple PDF text extraction, escaped literal strings, placeholder removal, and invalid PDFs.
- Split remaining OCR work to `blg_backend_ocr_provider`.

## Validation
- `git diff --check`: passed.
- `cd web-client && npx tsc --noEmit`: passed.
- `cd web-client && npm run build`: passed; existing `expo-av` deprecation warning remains.
- `go test ./internal/parser`: blocked because `go` is not installed on PATH in this Hermes environment.
- Supplemental only: `/root/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.24.12.linux-amd64/bin/go test ./internal/parser -v` passed.

## Review
- OpenCode GLM-5.1 read-only review returned `REVISE` for zip-bomb risk, PDF octal/operator handling, missing escaped literal/nested OPF href tests, and regex recompilation.
- Hermes fixed the required findings directly and reran validation.
- No secrets or token-bearing remotes observed. Origin remote remains `https://github.com/unalkalkan/TwelveReader.git`.

## Remaining backlog
- `blg_backend_ocr_provider`: real OCR provider integration for scanned PDFs/images.
- Add compressed/encoded real-world PDF handling, e.g. FlateDecode support or an audited parser dependency if dependency policy allows.
- Live-backend/browser validation for upload, voice mapping, and active pipeline behavior.

## Next action
- Commit accepted parser and orchestration state changes, then continue OCR provider implementation or live validation if time remains.
