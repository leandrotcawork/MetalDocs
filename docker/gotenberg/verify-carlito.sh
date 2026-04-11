#!/usr/bin/env bash
# Phase 0 gating check: verify the running Gotenberg container has Carlito
# installed. Run against a live container ID or name.
#
# Usage:
#   ./docker/gotenberg/verify-carlito.sh metaldocs-gotenberg
#
# Exit codes:
#   0  - Carlito present
#   1  - container not found
#   2  - Carlito missing

set -euo pipefail

CONTAINER="${1:-metaldocs-gotenberg}"

if ! docker ps --format '{{.Names}}' | grep -q "^${CONTAINER}$"; then
  echo "ERROR: container '${CONTAINER}' is not running" >&2
  exit 1
fi

if docker exec "${CONTAINER}" fc-list 2>/dev/null | grep -qi "carlito"; then
  echo "OK: Carlito is installed in container '${CONTAINER}'"
  docker exec "${CONTAINER}" fc-list | grep -i "carlito"
  exit 0
fi

echo "FAIL: Carlito font is missing from container '${CONTAINER}'" >&2
echo "Fix: rebuild the Gotenberg image from docker/gotenberg/Dockerfile" >&2
exit 2
