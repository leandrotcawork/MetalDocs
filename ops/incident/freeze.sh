#!/usr/bin/env bash
# freeze.sh — Emergency read-only mode for MetalDocs API.
# Writes read-only flag to central config. All mutating routes return 503.
# GET requests continue normally.
#
# Usage:
#   bash ops/incident/freeze.sh [--unfreeze]
set -euo pipefail

ADMIN_TOKEN="${INCIDENT_ADMIN_TOKEN:-}"
CONFIG_URL="${CONFIG_URL:-https://api.metaldocs.app/internal/admin/config}"

if [ -z "$ADMIN_TOKEN" ]; then
  echo "❌ INCIDENT_ADMIN_TOKEN not set" >&2
  exit 1
fi

UNFREEZE=0
for arg in "$@"; do
  [ "$arg" = "--unfreeze" ] && UNFREEZE=1
done

if [ "$UNFREEZE" = "1" ]; then
  echo "🔓 Unfreezing API (re-enabling mutations)..."
  curl -sf -X PUT "$CONFIG_URL" \
    -H "Authorization: Bearer $ADMIN_TOKEN" \
    -H "Content-Type: application/json" \
    -d '{"read_only_mode": false}'
  echo "✅ API unfrozen — mutations re-enabled"
else
  echo "🧊 Freezing API (read-only mode)..."
  echo "⚠️  All POST/PUT/PATCH/DELETE will return 503 until unfrozen."
  curl -sf -X PUT "$CONFIG_URL" \
    -H "Authorization: Bearer $ADMIN_TOKEN" \
    -H "Content-Type: application/json" \
    -d '{"read_only_mode": true}'
  echo "✅ API frozen — GET requests still active"
  echo ""
  echo "To unfreeze: bash ops/incident/freeze.sh --unfreeze"
fi
