#!/usr/bin/env sh
set -eu

cd "$(dirname "$0")/.."

python3 scripts/e2e-api-smoke.py --base-url "${TWELVEREADER_E2E_BASE_URL:-http://localhost:8080}"
