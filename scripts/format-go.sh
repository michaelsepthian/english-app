#!/usr/bin/env sh
# MIVA-QUALITY: use the same Go formatter locally and in Docker.
set -eu
cd "$(dirname "$0")/.."
mode=${1:---write}
case "$mode" in --write|--check) ;; *) echo 'Usage: scripts/format-go.sh [--write|--check]' >&2; exit 2 ;; esac
if command -v gofmt >/dev/null 2>&1; then
  if [ "$mode" = '--write' ]; then
    gofmt -w backend
  else
    files=$(gofmt -l backend)
    if [ -n "$files" ]; then printf '%s\n' "$files"; exit 1; fi
  fi
else
  docker run --rm -v "$PWD/backend:/src" -w /src golang:1.26-alpine sh -ec '
    if [ "$1" = "--write" ]; then
      gofmt -w .
    else
      files=$(gofmt -l .)
      if [ -n "$files" ]; then printf "%s\n" "$files"; exit 1; fi
    fi
  ' sh "$mode"
fi
