# Chaos Drill Scenarios

Run monthly in staging. Results appended to `ops/chaos/LOG.md`.

---

## Scenario 1: kill_scheduler mid-tick

**Script:** `ops/chaos/kill_scheduler.sh`  
**Validates:** Fencing epoch prevents double-publish after leader replacement.

**Expected outcome:**
- New leader acquires lease with epoch > previous epoch
- Any in-flight job sees context cancellation (heartbeat fails)
- No governance event published twice (checked via unique constraint on idempotency key)
- New leader resumes within lease TTL (≤ 60s)

---

## Scenario 2: network_partition author↔API

**Script:** `ops/chaos/network_partition.sh`  
**Validates:** Offline queue drains on reconnect; no duplicate governance events.

**Expected outcome:**
- Mutations queued client-side during partition (mutationClient retry)
- On reconnect, same Idempotency-Key replays → `Idempotent-Replay: true`
- Zero duplicate governance_events rows (idempotency key deduplication)
- UI shows "reconnected" state

---

## Scenario 3: DB pause (pg_sleep simulation)

**Script:** `ops/chaos/db_pause.sh`  
**Validates:** Backpressure hysteresis enters; non-critical jobs skip; safety jobs degrade.

**Expected outcome:**
- `pg_stat_activity` shows high active connections during pause
- Scheduler enters `inPressure=true` after 3 consecutive high probes
- `SkipOnPressure` jobs: skip count increases
- `DegradeOnPressure` jobs: continue with reduced batch size
- After DB recovers: hysteresis exits after 3 consecutive normal probes

---

## LOG Template

```
---
Date: YYYY-MM-DD HH:MM UTC
Drill: <scenario name>
Operator: <name>
Result: PASS | FAIL
Notes: <details>
```

## LOG

<!-- Results appended by chaos scripts -->
