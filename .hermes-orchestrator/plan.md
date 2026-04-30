# Plan

**Project:** TwelveReader
**Updated At:** 2026-04-30T20:43:00Z

## Phase 1 — Orchestrator and scope baseline
- Initialize `.hermes-orchestrator/` state.
- Record owner feedback without secrets.
- Replace template mission/design/acceptance/plan with project-specific documents.
- Convert WIP gaps into machine-readable backlog.

## Phase 2 — Compile and API-contract stabilization
- Fix current TypeScript compile error in Explore voice count.
- Harden API parsing/error handling for book/persona/voice map states.
- Ensure upload progress and polling stay coherent.

## Phase 3 — Missing core UX flows
- Add voice-mapping UI for books in `voice_mapping` state.
- Add download/export/share affordances on player.
- Improve player processing/error/empty states.
- Improve library progress summaries.

## Phase 4 — Quality gates
- Run `npx tsc --noEmit`.
- Run `npm run build` if feasible in this environment.
- Run Go tests when Go exists; currently blocked by missing `go` binary.
- Review diff for secrets, path traversal, and broken API contracts.

## Phase 5 — Commit and continuation
- Commit accepted work on `ui`.
- Keep summaries/latest.md up to date.
- Use a cron continuation if the chat/tool session stops before the deadline and work remains.

## Current focus
Phase 2 then Phase 3: compile fix, voice-mapping UI, and player/library polish.
