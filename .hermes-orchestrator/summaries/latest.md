# TwelveReader Hermes Orchestrator Summary

Updated: 2026-05-07T06:22:10Z

## Current focus
Short-term core Qwen3-TTS UX hardening. Steps 3 and 4 are now implemented and verified in the working tree.

## Completed in this supervisor cycle
- Step 3: discovered personas auto-map to the persisted single-user default voice.
  - Initial persona discovery no longer waits for manual mapping when a default voice exists.
  - Newly discovered personas also receive the default voice unless already explicitly mapped.
  - Auto-mappings are persisted to the per-book voice map for API/UI visibility.
- Step 4: persona remaps handle stale audio safely.
  - Existing old-voice audio is marked `audio_stale` with `stale_voice_id`.
  - Fresh/current segments are synthesized before stale regeneration.
  - Stale retry work stays in the stale queue.
  - In-flight old-voice synthesis is marked stale and queued for regeneration if a remap lands during synthesis.
- API/types/docs updated for `voice_id`, `audio_stale`, and `stale_voice_id` segment fields.

## Verification
- `git diff --check` passed.
- `docker run --rm -v "$PWD":/app -w /app golang:1.24-alpine go test ./...` passed.
- `cd web-client && npx tsc --noEmit --skipLibCheck` passed.
- Independent review attempt timed out, so supervisor performed focused must-fix review and corrected stale ordering, stale retry, in-flight remap, and voice-map persistence issues before final validation.

## Next safe work
- Commit accepted Steps 3-4 implementation.
- Step 1 remains listed as ready/planned in existing backlog despite later steps now being implemented; decide whether to explicitly reconcile or leave it as an umbrella/provider-hardening item.
