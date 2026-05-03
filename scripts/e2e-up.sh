#!/usr/bin/env sh
set -eu

cd "$(dirname "$0")/.."

docker compose -f docker-compose.e2e.yaml up -d --build backend frontend
