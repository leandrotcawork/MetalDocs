#!/usr/bin/env bash
# healthz.sh — Check API health + DB + scheduler heartbeat
set -euo pipefail

BASE_URL="${BASE_URL:-https://staging.metaldocs.app}"
MAX_WAIT="${MAX_WAIT:-10}"
TIMEOUT=5

fail() { echo "❌ $*" >&2; exit 1; }
ok()   { echo "✅ $*"; }

check_endpoint() {
  local url="$1" label="$2"
  local status
  status=$(curl -sf --max-time "$TIMEOUT" -o /dev/null -w "%{http_code}" "$url" 2>/dev/null || echo "000")
  if [ "$status" = "200" ]; then
    ok "$label ($status)"
  else
    fail "$label returned $status (expected 200)"
  fi
}

echo "🔍 Smoke: $BASE_URL"

check_endpoint "$BASE_URL/healthz" "healthz"
check_endpoint "$BASE_URL/readyz"  "readyz"

# DB check via healthz JSON
HEALTH=$(curl -sf --max-time "$TIMEOUT" "$BASE_URL/healthz" 2>/dev/null || echo '{}')
DB_STATUS=$(echo "$HEALTH" | python3 -c "import sys,json; d=json.load(sys.stdin); print(d.get('db','unknown'))" 2>/dev/null || echo "unknown")
if [ "$DB_STATUS" = "ok" ]; then
  ok "db=$DB_STATUS"
else
  fail "db=$DB_STATUS (expected ok)"
fi

# Scheduler heartbeat
SCHED_STATUS=$(echo "$HEALTH" | python3 -c "import sys,json; d=json.load(sys.stdin); print(d.get('scheduler','unknown'))" 2>/dev/null || echo "unknown")
if [ "$SCHED_STATUS" = "ok" ] || [ "$SCHED_STATUS" = "standby" ]; then
  ok "scheduler=$SCHED_STATUS"
else
  fail "scheduler=$SCHED_STATUS (expected ok or standby)"
fi

echo "✅ All health checks passed"
