# Incident Runbook — MetalDocs Approval System

**Tabletop drill:** Quarterly. Sign-off logged below.  
**On-call rotation:** PagerDuty service `metaldocs-approval`.

---

## Escalation Levels

| Level | Trigger | Response Time | Escalate To |
|-------|---------|--------------|-------------|
| P1 | Prod smoke fail / 0 lease holders | 15 min | CTO + SRE lead |
| P2 | 2 consecutive staging smoke fails | 30 min | SRE on-call |
| P3 | Alert rule firing (412 rate, skip streak) | 2h | Dev team |

---

## Scenario 1: Stuck Scheduler (no lease holder)

**Detection:** `metaldocs_scheduler_last_tick_timestamp` > 5 min ago. Alert: `Lease Steal Detected`.

**Diagnostic queries:**
```sql
-- Check job_leases state
SELECT name, holder, epoch, expires_at, expires_at < now() AS expired
FROM job_leases;

-- Check recent scheduler errors in logs
-- (Grafana: filter metaldocs_scheduler_errors_total)
```

**Mitigation:**
1. Check if all API pods are running: `kubectl get pods -l app=metaldocs-api`
2. If pod crashed: `kubectl rollout restart deployment/metaldocs-api`
3. Expired lease auto-clears in ≤60s; new pod acquires on next tick
4. If epoch stuck: run `psql -c "UPDATE job_leases SET expires_at = now() - interval '1s' WHERE name = 'main';"` to force release
5. Monitor: `metaldocs_scheduler_last_tick_timestamp` should update within 2 min

**Escalate if:** No recovery in 5 min → P1

---

## Scenario 2: Lease Storm (high epoch churn)

**Detection:** `lease_steal_count > 5/min` alert fires.

**Diagnostic:**
```sql
-- Check epoch history (high epoch = many steals)
SELECT epoch, holder, expires_at FROM job_leases ORDER BY epoch DESC LIMIT 10;

-- Check heartbeat failures in logs
```

**Mitigation:**
1. Check for network partition between pods and DB: `kubectl exec -it <pod> -- pg_isready -h $DB_HOST`
2. Reduce scheduler instances if too many competing: scale down to 1 replica temporarily
3. Check DB connection pool exhaustion (backpressure may have throttled heartbeats)
4. If flapping: force freeze to stop all scheduler activity → `bash ops/incident/freeze.sh`

---

## Scenario 3: Tripwire Floods

**Detection:** `metaldocs_tripwire_firings_total` increasing rapidly.

**Diagnostic:**
```sql
-- Check recent tripwire events
SELECT event_type, count(*), max(created_at)
FROM governance_events
WHERE event_type LIKE '%tripwire%'
GROUP BY event_type
ORDER BY max(created_at) DESC;
```

**Mitigation:**
1. Identify event type causing flood
2. Check if it's a bug in state machine (illegal transition retried in loop)
3. If approval instance stuck in loop: cancel instance via admin API
   ```bash
   curl -X POST /api/v2/instances/{id}/cancel \
     -H "Authorization: Bearer $ADMIN_TOKEN" \
     -d '{"reason":"stuck-tripwire-flood"}'
   ```
4. Freeze API to stop new submissions if flood continues: `bash ops/incident/freeze.sh`

---

## Scenario 4: Cascade Bug

**Detection:** Multiple documents transitioning to wrong states unexpectedly.

**Diagnostic:**
```sql
-- Find unexpected state transitions in last hour
SELECT document_v2_id, event_type, actor_user_id, created_at
FROM governance_events
WHERE created_at > now() - interval '1h'
  AND event_type IN ('doc.published', 'doc.cancelled', 'doc.obsoleted')
ORDER BY created_at DESC;
```

**Mitigation:**
1. Freeze API immediately: `bash ops/incident/freeze.sh`
2. Identify affected documents and tenants
3. Roll back state via direct SQL (requires SRE + dev sign-off):
   ```sql
   -- Example: revert erroneously published docs
   UPDATE documents_v2 SET status = 'approved'
   WHERE id IN (<affected_ids>) AND tenant_id = '<tenant>';
   ```
4. Deploy hotfix with bug fix, then unfreeze: `bash ops/incident/freeze.sh --unfreeze`

---

## Scenario 5: Idempotency Collision Spike

**Detection:** `409 idempotency.key_conflict` rate > 10/min.

**Diagnostic:**
```sql
-- Check recent idempotency key conflicts
SELECT key, method, url, count(*) as attempts
FROM idempotency_keys
WHERE created_at > now() - interval '1h'
  AND status = 'conflict'
GROUP BY key, method, url
ORDER BY attempts DESC
LIMIT 20;
```

**Mitigation:**
1. Usually client bug: client reusing same key with different body
2. Identify tenant/user causing spike from logs
3. If intentional attack: rate-limit that tenant's API key
4. If library bug: notify client developers + temporarily increase 409 TTL tolerance

---

## Tabletop Drill Sign-Off

| Date | Scenario | Operator | Outcome | Notes |
|------|---------|---------|---------|-------|
| 2026-Q2 | All | TBD | TBD | Initial drill |
