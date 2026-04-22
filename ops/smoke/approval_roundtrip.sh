#!/usr/bin/env bash
# approval_roundtrip.sh — Synthetic approval smoke test
# Creates disposable tenant, submits + auto-signs + publishes + cleans up.
# Each step must complete within STEP_TIMEOUT seconds.
set -euo pipefail

BASE_URL="${BASE_URL:-https://staging.metaldocs.app}"
STEP_TIMEOUT=2
TENANT_PREFIX="synthetic_smoke_"
ADMIN_TOKEN="${SMOKE_ADMIN_TOKEN:-}"

fail() { echo "❌ $*" >&2; cleanup; exit 1; }
ok()   { echo "✅ $*"; }

TENANT_ID="${TENANT_PREFIX}$(date +%s)"
DOC_ID=""
INSTANCE_ID=""

cleanup() {
  if [ -n "$TENANT_ID" ] && [ -n "$ADMIN_TOKEN" ]; then
    curl -sf --max-time 5 -X POST \
      -H "Authorization: Bearer $ADMIN_TOKEN" \
      -H "Content-Type: application/json" \
      -d "{\"tenantId\":\"$TENANT_ID\"}" \
      "$BASE_URL/internal/admin/cleanup-tenant" >/dev/null 2>&1 || true
    echo "🧹 Cleaned up tenant $TENANT_ID"
  fi
}
trap cleanup EXIT

timed_curl() {
  local label="$1"; shift
  local start end elapsed status
  start=$(date +%s%N)
  status=$(curl -sf --max-time "$STEP_TIMEOUT" -o /tmp/smoke_resp.json -w "%{http_code}" "$@" 2>/dev/null || echo "000")
  end=$(date +%s%N)
  elapsed=$(( (end - start) / 1000000 ))

  if [[ "$status" -ge 200 && "$status" -lt 300 ]]; then
    ok "$label ${elapsed}ms"
  else
    fail "$label failed: HTTP $status after ${elapsed}ms"
  fi
  cat /tmp/smoke_resp.json
}

echo "🚀 Approval roundtrip smoke: $BASE_URL (tenant: $TENANT_ID)"

# 1. Seed synthetic tenant
SEED=$(timed_curl "seed tenant" -X POST \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -d "{\"tenantId\":\"$TENANT_ID\",\"docId\":\"$(uuidgen | tr '[:upper:]' '[:lower:]')\",\"roles\":[\"author\",\"reviewer\"]}" \
  "$BASE_URL/internal/test/seed")

DOC_ID=$(echo "$SEED" | python3 -c "import sys,json; print(json.load(sys.stdin)['docId'])")
AUTHOR_COOKIE=$(echo "$SEED" | python3 -c "import sys,json; print(json.load(sys.stdin)['cookies']['author'])")
REVIEWER_COOKIE=$(echo "$SEED" | python3 -c "import sys,json; print(json.load(sys.stdin)['cookies']['reviewer'])")

# 2. Submit
timed_curl "submit doc" -X POST \
  -H "Content-Type: application/json" \
  -H "Idempotency-Key: $(uuidgen | tr '[:upper:]' '[:lower:]')" \
  -H "Cookie: metaldocs_session=$AUTHOR_COOKIE" \
  -d '{}' \
  "$BASE_URL/api/v2/documents/$DOC_ID/submit" >/dev/null

# 3. Get instance ID
INSTANCE_RESP=$(curl -sf --max-time "$STEP_TIMEOUT" \
  -H "Cookie: metaldocs_session=$AUTHOR_COOKIE" \
  "$BASE_URL/api/v2/documents/$DOC_ID/instance")
INSTANCE_ID=$(echo "$INSTANCE_RESP" | python3 -c "import sys,json; print(json.load(sys.stdin)['id'])")
ok "instance_id=$INSTANCE_ID"

# 4. Reviewer signs
timed_curl "reviewer signoff" -X POST \
  -H "Content-Type: application/json" \
  -H "Idempotency-Key: $(uuidgen | tr '[:upper:]' '[:lower:]')" \
  -H "Cookie: metaldocs_session=$REVIEWER_COOKIE" \
  -d '{"decision":"approve","password":"test1234"}' \
  "$BASE_URL/api/v2/instances/$INSTANCE_ID/signoff" >/dev/null

# 5. Assert published state
STATE_RESP=$(curl -sf --max-time "$STEP_TIMEOUT" \
  -H "Cookie: metaldocs_session=$AUTHOR_COOKIE" \
  "$BASE_URL/api/v2/documents/$DOC_ID")
STATE=$(echo "$STATE_RESP" | python3 -c "import sys,json; print(json.load(sys.stdin)['status'])")
if [[ "$STATE" == "published" || "$STATE" == "approved" ]]; then
  ok "final state=$STATE"
else
  fail "expected published, got $STATE"
fi

echo "✅ Approval roundtrip complete"
