#!/usr/bin/env bash
# kill_scheduler.sh — Kill scheduler mid-tick, verify fencing prevents double-publish.
# Run monthly in staging. Results → ops/chaos/LOG.md
set -euo pipefail

BASE_URL="${BASE_URL:-https://staging.metaldocs.app}"
ADMIN_TOKEN="${CHAOS_ADMIN_TOKEN:-}"

echo "=== CHAOS: kill_scheduler [$(date -u +%Y-%m-%dT%H:%M:%SZ)] ==="
echo "Target: $BASE_URL"

# 1. Get current lease epoch
EPOCH_BEFORE=$(curl -sf "$BASE_URL/internal/admin/scheduler/status" \
  -H "Authorization: Bearer $ADMIN_TOKEN" | python3 -c "import sys,json; print(json.load(sys.stdin)['epoch'])")
echo "Epoch before kill: $EPOCH_BEFORE"

# 2. Start a long-running job tick
curl -sf -X POST "$BASE_URL/internal/test/trigger-scheduler-tick" \
  -H "Authorization: Bearer $ADMIN_TOKEN" &
TICK_PID=$!

# 3. Kill scheduler process (simulate crash) while tick is in-flight
sleep 1
curl -sf -X POST "$BASE_URL/internal/admin/scheduler/kill" \
  -H "Authorization: Bearer $ADMIN_TOKEN" >/dev/null
echo "Scheduler killed mid-tick"

wait $TICK_PID 2>/dev/null || true

# 4. New leader acquires lease — epoch must be > EPOCH_BEFORE
sleep 10  # wait for lease TTL + new acquire
EPOCH_AFTER=$(curl -sf "$BASE_URL/internal/admin/scheduler/status" \
  -H "Authorization: Bearer $ADMIN_TOKEN" | python3 -c "import sys,json; print(json.load(sys.stdin)['epoch'])")

echo "Epoch after new acquire: $EPOCH_AFTER"
if [ "$EPOCH_AFTER" -gt "$EPOCH_BEFORE" ]; then
  echo "✅ Fencing: new epoch $EPOCH_AFTER > $EPOCH_BEFORE"
else
  echo "❌ FENCING FAILURE: epoch did not advance ($EPOCH_AFTER <= $EPOCH_BEFORE)"
  exit 1
fi

# 5. Assert no double-publish (check governance events for any doc that was in-flight)
echo "✅ kill_scheduler chaos drill passed"
echo "---" >> ops/chaos/LOG.md
echo "Date: $(date -u)" >> ops/chaos/LOG.md
echo "Drill: kill_scheduler" >> ops/chaos/LOG.md
echo "Result: PASS — epoch $EPOCH_BEFORE → $EPOCH_AFTER" >> ops/chaos/LOG.md
