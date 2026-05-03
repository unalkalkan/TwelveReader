#!/usr/bin/env sh
set -eu

cd "$(dirname "$0")/.."

GO_IMAGE="${GO_IMAGE:-golang:1.24-alpine}"

docker run --rm \
  -v "$PWD":/src \
  -w /src \
  -e GOCACHE=/tmp/go-cache \
  -e GOMODCACHE=/tmp/go-mod-cache \
  "$GO_IMAGE" \
  sh -c 'go version && go test ./...'
