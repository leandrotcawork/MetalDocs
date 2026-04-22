# Deploy Runbook — Approval v2 (Spec 2)

**SRE sign-off required before production deploy.**  
Each step links to a runbook section. Complete in order.

---

## Pre-Deploy Checklist

- [ ] All CI gates green on release branch
- [ ] Perf benchmarks run on `main` (check Actions: `Perf Benchmarks / perf-full`)
- [ ] `CAPABILITY_CATALOG.sha256` matches current catalog
- [ ] Database backup confirmed (ops: `pg_dump` snapshot tagged `pre-spec2-deploy`)
- [ ] On-call SRE notified + PagerDuty maintenance window opened

---

## Step 1 — Apply Additive Migrations

Migrations 0141-0148 are additive (new tables, columns, functions). Safe to run against live DB.

```bash
# Run in order:
psql $DATABASE_URL -f migrations/0141_approval_routes.sql
psql $DATABASE_URL -f migrations/0142_approval_instances.sql
psql $DATABASE_URL -f migrations/0143_approval_signoffs.sql
psql $DATABASE_URL -f migrations/0144_governance_events.sql
psql $DATABASE_URL -f migrations/0145_idempotency_keys.sql
psql $DATABASE_URL -f migrations/0146_job_leases.sql
psql $DATABASE_URL -f migrations/0147_approval_rls.sql
psql $DATABASE_URL -f migrations/0148_approval_security_definer.sql
```

**Verify:** `SELECT count(*) FROM approval_routes;` returns 0 (no data yet).

---

## Step 2 — Deploy API (feature flag OFF)

Deploy new API build with `APPROVAL_V2_PCT=0`.

```bash
kubectl set image deployment/metaldocs-api api=metaldocs:${RELEASE_TAG}
kubectl rollout status deployment/metaldocs-api --timeout=5m
```

**Verify:** smoke probe passes with `APPROVAL_V2_PCT=0`.

---

## Step 3 — Run Smoke

```bash
BASE_URL=https://api.metaldocs.app bash ops/smoke/healthz.sh
BASE_URL=https://api.metaldocs.app SMOKE_ADMIN_TOKEN=$TOKEN bash ops/smoke/approval_roundtrip.sh
```

**If smoke fails:** roll back deployment (`kubectl rollout undo deployment/metaldocs-api`).

---

## Step 4 — Canary Ramp

```bash
# Start canary at 1%
export APPROVAL_V2_PCT=1
go run ./ops/canary --step

# Monitor for 30 min, then advance:
go run ./ops/canary --step  # 5%
# ... repeat each 30 min: 25%, 50%, 100%
```

Canary controller validates metrics automatically before each advance.  
**If breach detected:** controller sets `APPROVAL_V2_PCT=0` and exits 2.

---

## Step 5 — Apply Enforcement Migrations

After 100% traffic, apply enforcement (0149 lease epoch monotonicity):

```bash
psql $DATABASE_URL -f migrations/0149_job_leases_epoch_monotonic.sql
```

---

## Step 6 — Enable Jobs

```bash
kubectl set env deployment/metaldocs-api \
  ENABLE_SCHEDULER=true \
  ENABLE_REAPER=true \
  ENABLE_STUCK_WATCHDOG=true
kubectl rollout status deployment/metaldocs-api --timeout=3m
```

---

## Step 7 — Final Smoke

```bash
BASE_URL=https://api.metaldocs.app bash ops/smoke/healthz.sh
BASE_URL=https://api.metaldocs.app SMOKE_ADMIN_TOKEN=$TOKEN bash ops/smoke/approval_roundtrip.sh
```

---

## Rollback Playbook

Reverse order. Run if any step fails:

```bash
# 1. Flag back to 0
export APPROVAL_V2_PCT=0

# 2. Roll back API
kubectl rollout undo deployment/metaldocs-api
kubectl rollout status deployment/metaldocs-api

# 3. Disable jobs
kubectl set env deployment/metaldocs-api ENABLE_SCHEDULER=false ENABLE_REAPER=false

# 4. Revert enforcement migration (if applied)
psql $DATABASE_URL -c "
  -- Revert 0149: restore DELETE behavior on release_lease
  CREATE OR REPLACE FUNCTION release_lease(p_name text, p_holder text)
  RETURNS void LANGUAGE plpgsql AS \$\$
  BEGIN
    DELETE FROM job_leases WHERE name = p_name AND holder = p_holder;
  END;
  \$\$;"

# 5. Run smoke to confirm rollback
bash ops/smoke/healthz.sh
```

**SRE sign-off:** _______________________ Date: _______
