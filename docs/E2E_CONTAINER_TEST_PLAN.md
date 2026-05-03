# Container-First E2E Test Plan

> **For Hermes:** Use subagent-driven-development skill to implement this plan task-by-task.

**Goal:** Make TwelveReader runnable and testable entirely through containers, including Go build/test, frontend build, API smoke tests, and browser E2E checks.

**Architecture:** Use Docker/Compose as the canonical runtime boundary. Keep the host free of Go requirements by running Go commands in the official Golang container and by building the backend through the existing multi-stage `Dockerfile`. Add a lightweight E2E profile that can run with stub providers first, then optionally run provider-backed tests when real LLM/TTS/OCR endpoints and secrets are available.

**Tech Stack:** Docker Compose, Go 1.24 container, Expo web export, nginx, Playwright or containerized browser runner, shell/Python smoke scripts.

---

## Current findings

- Host Go is not installed, but Docker is available.
- Containerized Go validation works:
  - `docker run --rm -v "$PWD":/src -w /src golang:1.24-alpine sh -c 'go version && go test ./...'`
  - Result on 2026-05-02: all Go packages passed.
- Frontend validation works on host Node:
  - `cd web-client && npx tsc --noEmit && npm run build`
- Existing production image builds backend and web export in one multi-stage `Dockerfile`.
- Existing `docker-compose.yaml` references `./qwen3-tts`, but that folder is absent in this branch, so full compose startup currently needs a container profile split or compose override before it can be a reliable default.
- Existing `config/config.yaml` points at real provider endpoints. E2E should start with deterministic stub providers, then separately test real providers.
- Web client API resolution is container-friendly: if no `EXPO_PUBLIC_API_URL` is set, it uses `window.location.origin`, and `web-client/nginx.conf` proxies `/api/` to `backend:8080`.

## Strategy

### Test pyramid for this repo

1. **Containerized unit/build checks**
   - Go: run `go test ./...` inside `golang:1.24-alpine`.
   - Backend image: run `docker build -t twelvereader:local .`.
   - Frontend: run TypeScript/build either on host Node or in the existing Node build stage.

2. **Backend API smoke tests with stub providers**
   - Start backend container with a test config that uses stub LLM/TTS/OCR providers.
   - Assert `/health/live`, `/health/ready`, `/api/v1/info`, `/api/v1/providers`, and `/api/v1/voices`.
   - Upload a tiny `.txt` book via `POST /api/v1/books`.
   - Poll status until one of the expected terminal/blocked states: `voice_mapping`, `ready`, `synthesized`, or `error`.
   - If `voice_mapping`, submit a narrator voice map and keep polling.
   - Assert `/segments`, `/personas`, `/stream`, audio URL behavior, and `/download` where artifacts exist.

3. **Browser E2E through the frontend container**
   - Serve frontend with nginx on `localhost:3000`.
   - Use Playwright against `http://localhost:3000`.
   - Cover: landing/library visibility, add/upload flow, voices page, voice mapping modal/state, player state, and download link.

4. **Provider-backed E2E profile**
   - Use only when secrets/endpoints are configured.
   - Run the same smoke flow against real LLM/TTS/OCR providers.
   - Mark slow/flaky provider tests separately from deterministic stub E2E.

## Proposed files

Create these files in order:

- `config/e2e.stub.yaml`
  - local storage path `/app/data`
  - one enabled LLM provider with no endpoint/model to force `StubLLMProvider`
  - one enabled TTS provider with no endpoint/model to force `StubTTSProvider`
  - one enabled OCR provider with no endpoint to force `StubOCRProvider`

- `docker-compose.e2e.yaml`
  - `backend` service built from root `Dockerfile`
  - mount `./config/e2e.stub.yaml:/app/config/config.yaml:ro`
  - named volume for `/app/data`
  - expose `8080:8080`
  - `frontend` service built from `web-client/Dockerfile`
  - expose `3000:80`
  - no `qwen3-tts` service by default
  - healthchecks for backend and frontend

- `scripts/container-go-test.sh`
  - runs Go tests inside `golang:1.24-alpine`
  - preserves host-independent validation

- `scripts/e2e-smoke.sh` or `scripts/e2e-smoke.py`
  - starts compose e2e stack
  - waits for health
  - runs API smoke flow
  - exits non-zero on failure
  - prints concise diagnostics and `docker compose logs` on failure

- `web-client/e2e/*.spec.ts`
  - Playwright browser tests after the API smoke layer is stable

- `.github/workflows/e2e.yml` later
  - run containerized Go tests, frontend build, e2e compose smoke on GitHub Actions

## Bite-sized implementation tasks

### Task 1: Add a deterministic stub E2E config

**Objective:** Make backend startup independent of external providers.

**Files:**
- Create: `config/e2e.stub.yaml`

**Content shape:**
```yaml
server:
  host: "0.0.0.0"
  port: 8080
  read_timeout: 30
  write_timeout: 30

storage:
  adapter: "local"
  local:
    base_path: "/app/data"

providers:
  llm:
    - name: "stub-llm"
      enabled: true
      endpoint: ""
      api_key: ""
      model: ""
      context_window: 4096
      concurrency: 1
      rate_limit_qps: 10.0
  tts:
    - name: "stub-tts"
      enabled: true
      endpoint: ""
      api_key: ""
      max_segment_size: 500
      concurrency: 1
      rate_limit_qps: 10.0
      timestamp_precision: "word"
      options: {}
  ocr:
    - name: "stub-ocr"
      enabled: true
      endpoint: ""
      api_key: ""
      concurrency: 1
      options: {}

pipeline:
  worker_pool_size: 1
  max_retries: 1
  retry_backoff_ms: 100
  temp_dir: "/tmp/twelvereader-e2e"
```

**Verification:**
```bash
docker run --rm -v "$PWD":/src -w /src golang:1.24-alpine sh -c 'go test ./internal/config ./internal/provider ./internal/api'
```
Expected: tests pass.

### Task 2: Add containerized Go test script

**Objective:** Make Go test validation first-class without host Go.

**Files:**
- Create: `scripts/container-go-test.sh`
- Modify: `Makefile`

**Script behavior:**
```bash
#!/usr/bin/env sh
set -eu
cd "$(dirname "$0")/.."
docker run --rm \
  -v "$PWD":/src \
  -w /src \
  -e GOCACHE=/tmp/go-cache \
  -e GOMODCACHE=/tmp/go-mod-cache \
  golang:1.24-alpine \
  sh -c 'go version && go test ./...'
```

**Make target:**
```make
test-container: ## Run Go tests inside Docker, no host Go required
	./scripts/container-go-test.sh
```

**Verification:**
```bash
chmod +x scripts/container-go-test.sh
make test-container
```
Expected: all Go packages pass.

### Task 3: Add E2E compose profile without qwen3 dependency

**Objective:** Start backend and frontend containers without the missing `qwen3-tts` build context.

**Files:**
- Create: `docker-compose.e2e.yaml`

**Acceptance criteria:**
- `docker compose -f docker-compose.e2e.yaml config --quiet` passes.
- `docker compose -f docker-compose.e2e.yaml up -d --build backend` starts backend.
- `curl http://localhost:8080/health/live` returns HTTP 200.
- `curl http://localhost:8080/api/v1/providers` lists stub providers.

### Task 4: Add API smoke script

**Objective:** Verify a minimum upload/status/persona/stream/download flow via HTTP.

**Files:**
- Create: `scripts/e2e-api-smoke.py`
- Create: `scripts/e2e-up.sh`
- Create: `scripts/e2e-down.sh`

**Smoke flow:**
1. wait for `GET /health/live`
2. assert `GET /health/ready`
3. assert `GET /api/v1/info`
4. assert `GET /api/v1/providers`
5. assert `GET /api/v1/voices`
6. create a tiny text file in tempdir
7. upload it with multipart form to `POST /api/v1/books`
8. poll `GET /api/v1/books/:id/status`
9. inspect `GET /api/v1/books/:id/personas`
10. if mapping is required, post narrator mapping to `POST /api/v1/books/:id/voice-map?initial=true`
11. fetch `GET /api/v1/books/:id/segments`
12. fetch `GET /api/v1/books/:id/stream`
13. attempt `GET /api/v1/books/:id/download`; accept either 200 ZIP or a clear 500 if no synthesized artifacts exist in stub mode

**Verification:**
```bash
./scripts/e2e-up.sh
python3 scripts/e2e-api-smoke.py --base-url http://localhost:8080
./scripts/e2e-down.sh
```
Expected: smoke script exits 0 and prints endpoint-by-endpoint results.

### Task 5: Add frontend browser E2E after API smoke is stable

**Objective:** Exercise the web UI against the e2e container stack.

**Files:**
- Modify: `web-client/package.json`
- Create: `web-client/e2e/twelvereader.spec.ts`
- Create: `web-client/playwright.config.ts`

**Recommended tests:**
- app loads at `/`
- tabs render: Library, Add, Explore/Voices as applicable
- Add screen can submit typed text or upload fixture file
- Library shows uploaded book/progress
- Voices screen renders stub voices
- Player route handles no-audio/processing state without crashing

**Verification:**
```bash
cd web-client
npm install
npx playwright install --with-deps chromium
npm run e2e
```
Expected: Chromium E2E tests pass against `http://localhost:3000` with backend proxied through nginx.

### Task 6: Add GitHub Actions E2E workflow

**Objective:** Preserve container-first validation in CI.

**Files:**
- Create: `.github/workflows/container-e2e.yml`

**Workflow stages:**
1. checkout
2. containerized Go tests via `scripts/container-go-test.sh`
3. frontend TypeScript/build
4. compose e2e stack startup
5. API smoke
6. upload logs as artifacts on failure

## Provider-backed E2E policy

Run deterministic stub E2E on every PR. Run provider-backed E2E manually or nightly only when these are available:

- LLM OpenAI-compatible endpoint and API key/model
- TTS OpenAI-compatible endpoint and API key/model
- OCR endpoint/API key/model for scanned PDF/image cases
- GPU-backed qwen3 TTS service, if needed

Provider tests should be tagged as slow and allowed to emit detailed logs without exposing secrets.

## Immediate next recommendation

Implement Tasks 1-4 first. That will give the project a reliable, host-Go-free E2E baseline and prove that the app can run in containers before adding browser automation.
