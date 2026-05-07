You are an OpenCode worker in /workspace/TwelveReader. Use model opencode-go/deepseek-v4-pro.

TASK: Implement Step 2 frontend only: single-user default voice UX in Voices tab.

STRICT SCOPE:
- You may edit only: web-client/src/types/api.ts, web-client/src/api/client.ts, web-client/src/api/hooks.ts, web-client/app/(tabs)/voices.tsx, and your result file under .hermes-orchestrator/worker_runs/wr_20260506_step2_default_voice_frontend/.
- Do NOT edit backend, orchestrator state, qwen3-tts, or unrelated screens.
- Do NOT implement Step 3 auto-mapping or Step 4 stale regeneration.

BACKEND CONTRACT ALREADY IMPLEMENTED:
- GET /api/v1/voices/default returns { provider, voice_id, language?, voice_description?, updated_at }
- PUT /api/v1/voices/default with same payload saves selection.

REQUIREMENTS:
1. Add DefaultVoice schema/type in web-client/src/types/api.ts.
2. Add getDefaultVoice and setDefaultVoice client functions.
3. Add useDefaultVoice and useSetDefaultVoice hooks that invalidate/refetch default voice on success.
4. Update Voices tab UX:
   - Show a compact default voice card/status near the top.
   - Mark the currently default voice in rows.
   - Add a clear action per voice to "Set default"; disable or show selected state when already default.
   - Keep existing preview/favorite behavior.
   - Use existing dark/slate/blue style conventions.
5. Preserve TypeScript correctness.

VALIDATION:
- Run: cd web-client && npx tsc --noEmit

OUTPUT:
- Write .hermes-orchestrator/worker_runs/wr_20260506_step2_default_voice_frontend/result.json with status, summary, files_changed, tests_run, blockers, risks, review_request.
- Do not commit. Supervisor owns commits.
