# W5 Rollback Runbook

**When to invoke:** A P0 incident during the Task 2 post-flip soak, a failure
during Task 3 migration apply, or a post-Task-6 regression that cannot be
hot-fixed within a single 24h window.

## Decision tree

```
Incident detected
       │
       ▼
Has destructive migration 0113 applied on prod?
       │
       ├── NO  → Flag-only rollback: redeploy API/worker/frontend replicas
       │        with METALDOCS_DOCX_V2_ENABLED=false (the flag is a
       │        process-level env var, NOT a DB row — see Plan A Task 9 +
       │        Plan E Task 1). Verify via
       │        `curl /api/v1/feature-flags` per replica. CK5 code path
       │        wakes back up. No DB restore needed.
       │
       └── YES → Full rollback required:
                 1. Bring all API + worker replicas offline (503 page).
                 2. Run scripts/w5-rollback.sh with the w5-preflight dump.
                 3. Redeploy with METALDOCS_DOCX_V2_ENABLED=false.
                 4. `git checkout w5-preflight` on all deploy hosts.
                 5. Re-deploy.
                 6. Bring back online.
```

## Prerequisites verified before any cutover commit

- [ ] Tag `w5-preflight` exists on origin.
- [ ] pg_dump at `$DUMP_PATH` has sha256 matching evidence file.
- [ ] Blob store (S3/MinIO) is append-only; no rollback action needed.
- [ ] Staging rehearsal of `w5-rollback.sh` passed (see Task 10 Step 4).

## Full-rollback procedure (step-by-step)

1. Open incident ticket; notify @admin + @sre + @pm in #incident channel.
2. `make maintenance-mode ENABLE=true` — 503 all /api/v2 routes.
3. On deploy host:
   ```bash
   export PGHOST=... PGDATABASE=... PGUSER=... PGPASSWORD=... PGPORT=...
   ./scripts/w5-rollback.sh \
     --dump /secure/backup/metaldocs-w5-preflight-*.dump \
     --tag  w5-preflight \
     --env  production \
     --force-production
   # Interactive: type YES at the preflight gate to proceed.
   ```
4. Flip env var back: redeploy every API/worker/frontend replica with
   `METALDOCS_DOCX_V2_ENABLED=false`. Verify each replica via
   `curl http://<replica>:8080/api/v1/feature-flags`.
5. `git fetch --tags && git checkout w5-preflight`
6. Build + deploy rolled-back image.
7. Smoke test:
   - `curl /api/v1/templates` → 200
   - `curl /api/v2/templates` → 404 or unreachable
8. `make maintenance-mode ENABLE=false`
9. Incident retro within 48h.

## What is NOT rolled back

- Audit log entries created during soak remain (immutable).
- Export PDFs generated during soak remain in S3 (content-addressed; orphans).
- User form-data changes made during soak are LOST if they were made through
  the W5 UI after flip — the dump is from before flip. This is accepted risk;
  the W5 soak window must reject full-rollback once any tenant has used
  /api/v2 for real production data.
