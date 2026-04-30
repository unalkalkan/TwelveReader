# Acceptance

**Project:** TwelveReader
**Updated At:** 2026-04-30T20:43:00Z

## Global requirements
- Web client compiles with `npx tsc --noEmit`.
- No committed secrets or token-bearing remotes.
- UI uses real API contracts where endpoints exist; unsupported flows must be clearly marked and not pretend to work.
- Upload, library, voices, voice preview, player, processing status, voice mapping, and download flows are reachable.
- Loading, empty, processing, error, and blocked/waiting-for-mapping states are visible to users.
- Accepted implementation is committed to git on `ui`.

## Milestone: Orchestrator initialized
- `.hermes-orchestrator/mission.md`, `design.md`, `plan.md`, `acceptance.md`, `decisions.md`, `feedback.json`, `backlog.json`, `state.json`, `lock.json`, and summaries exist.
- The initial human feedback is recorded with secrets redacted.
- Backlog captures UI and backend follow-up tasks.

## Milestone: UI MVP working
- `web-client/app/(tabs)/explore.tsx` compiles against `useVoices()` response shape.
- Voice mapping can fetch personas/voices, choose provider voices per persona, submit via `POST /voice-map`, and refresh book state.
- Player can show processing, mapping-required, no-audio, and error states without crashing.
- Library shows useful progress based on available status fields rather than placeholder math.
- Download action opens the backend ZIP endpoint on web.

## Review checklist
- Security: no tokens, credentials, or private provider keys in tracked files.
- Type safety: TypeScript compile passes.
- API compatibility: request/response fields match `API.md` and Go handlers.
- UX: every disabled or not-yet-supported action explains why.
- Resumability: summary and backlog state reflect the next safe action.
