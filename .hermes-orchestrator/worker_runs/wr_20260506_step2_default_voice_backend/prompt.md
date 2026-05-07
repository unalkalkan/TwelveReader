You are an OpenCode worker in /workspace/TwelveReader. Use model opencode-go/deepseek-v4-pro.

TASK: Implement Step 2 backend only: single-user persisted default voice setting.

STRICT SCOPE:
- You may edit: pkg/types/*, internal/book/*, internal/api/voices_handler.go, internal/api/voices_handler_test.go, cmd/server/main.go, API.md, and your worker result file under .hermes-orchestrator/worker_runs/wr_20260506_step2_default_voice_backend/.
- Do NOT edit web-client, pipeline, TTS orchestrator, qwen3-tts, docker compose, or unrelated orchestrator state.
- Do NOT implement Step 3 auto-mapping or Step 4 stale regeneration.

REQUIREMENTS:
1. Add a single-user default voice model. Recommended shape:
   provider, voice_id, language,omitempty, voice_description,omitempty, updated_at.
2. Persist it in storage at a stable global single-user path, e.g. settings/default-voice.json.
3. Extend book.Repository minimally with SaveDefaultVoice/GetDefaultVoice (or an equivalent small repository abstraction if simpler). Missing setting should be non-fatal and allow bootstrapping.
4. Add API endpoint under /api/v1/voices/default:
   - GET returns current default; if missing, assign first available TTS provider voice and persist it.
   - PUT validates provider exists and voice_id appears in provider.ListVoices, then persists and returns it.
5. Wire route in cmd/server/main.go without breaking existing voices preview sample storage.
6. Update API.md with endpoint contract.
7. Preserve existing behavior and tests.

TDD REQUIREMENTS:
- Add failing tests first, then implementation.
- Repository tests in internal/book/repository_test.go for save/get, missing default, persistence across repository instances if feasible.
- API tests in internal/api/voices_handler_test.go for GET bootstrap, PUT valid, reject unknown provider, reject unknown voice.
- Use existing stub providers if possible; inspect internal/provider/stubs.go/tests.

VALIDATION COMMANDS:
- If host go is unavailable, run: ./scripts/container-go-test.sh
- If docker is unavailable, still run any feasible static checks and document blocker.

OUTPUT:
- Write .hermes-orchestrator/worker_runs/wr_20260506_step2_default_voice_backend/result.json with status, summary, files_changed, tests run, blockers, risks, and review_request.
- Do not commit. Supervisor owns commits.
