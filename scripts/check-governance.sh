#!/usr/bin/env bash
set -euo pipefail

BASE_REF="${BASE_REF:-origin/main}"
CHANGED="$(git diff --name-only "$BASE_REF"...HEAD || true)"

echo "Changed files:"
echo "$CHANGED"

fail() {
  echo "[governance-check] $1" >&2
  exit 1
}

if echo "$CHANGED" | grep -E '^apps/api/' >/dev/null 2>&1; then
  if ! echo "$CHANGED" | grep -E '^api/openapi/v1/openapi.yaml$' >/dev/null 2>&1; then
    fail "API handler change detected without OpenAPI update."
  fi
fi

if echo "$CHANGED" | grep -E '^internal/modules/' >/dev/null 2>&1; then
  if ! echo "$CHANGED" | grep -E '^tests/' >/dev/null 2>&1; then
    fail "Domain change detected without test updates under tests/."
  fi
fi

if echo "$CHANGED" | grep -E '^deploy/|^scripts/' >/dev/null 2>&1; then
  if ! echo "$CHANGED" | grep -E '^docs/runbooks/' >/dev/null 2>&1; then
    fail "Infra/ops change detected without runbook update."
  fi
fi

echo "[governance-check] OK"
