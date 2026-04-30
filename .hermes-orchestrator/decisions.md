# Decisions

## DEC-001: Use project-local Hermes Orchestrator state
- Date: 2026-04-30T20:43:00Z
- Status: accepted
- Context: The user requested long-running, resumable work on the WIP `ui` branch.
- Decision: Store orchestration state under `.hermes-orchestrator/` in the repo.
- Consequences: Mission, design, backlog, feedback, reviews, summaries, and lock state travel with the project and can survive chat/session limits.

## DEC-002: Implement directly in Hermes Agent for this phase
- Date: 2026-04-30T20:43:00Z
- Status: accepted
- Context: The user explicitly said not to spawn OpenCode for this request.
- Decision: Use Hermes tools and direct edits rather than OpenCode worker runs.
- Consequences: Worker run records describe direct Hermes implementation cycles instead of external OpenCode runs.

## DEC-003: Prioritize UI MVP blockers before backend deep work
- Date: 2026-04-30T20:43:00Z
- Status: accepted
- Context: The `ui` branch is WIP and the first validation failure is in the web client TypeScript compile.
- Decision: Fix compile/API-contract problems and missing core user flows before parser/OCR/cache backend tasks.
- Consequences: PDF/ePUB/OCR/LLM-cache improvements remain backlog items but are not first-phase blockers for UI MVP.

## DEC-004: Treat voice mapping as a core UI flow
- Date: 2026-04-30T20:43:00Z
- Status: accepted
- Context: Backend can set books to `voice_mapping`, which blocks synthesis until the user maps discovered personas.
- Decision: Add an in-player voice mapping panel/dialog connected to personas, voices, and voice-map endpoints.
- Consequences: UI MVP is not complete until a user can resolve mapping waits from the app.
