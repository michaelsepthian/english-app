#!/usr/bin/env sh
set -eu
cd "$(dirname "$0")/.."
docker run --rm --env-file backend/.env -e DB_HOST=host.docker.internal -e RUN_DB_TESTS=1 --add-host host.docker.internal:host-gateway -v "$PWD/backend:/src" -v english-app-go-mod:/go/pkg/mod -v english-app-go-cache:/root/.cache/go-build -w /src golang:1.26-alpine sh -c 'gofmt -w . && go test -count=1 ./... && go vet ./...'
npm --prefix frontend run check
npm --prefix frontend run build
