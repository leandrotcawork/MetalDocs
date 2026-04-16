#!/usr/bin/env bash
set -e
cd "$(dirname "$0")/.."

if [ ! -f ".env" ]; then
  echo ".env not found" >&2
  exit 1
fi

set -o allexport
source .env
set +o allexport

exec go run ./apps/api/cmd/metaldocs-api
