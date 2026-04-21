# Foundation — Controlled Document Distribution + Copy Ledger (Spec 4)

## Goal

Close the `publish → consumer` control loop mandated by ISO 9001 §7.5.3. Guarantee that every user with legitimate operational need for a controlled document:

1. Receives an explicit distribution obligation for the currently-effective revision
2. Attests to having read it (via view-ack or signature-ack, per-doc criticality)
3. Has that attestation automatically invalidated when the document is superseded
4. Can be surfaced at audit time with defensible recipient-resolution evidence

Plus: track uncontrolled copies (PDF exports) via watermark + ledger so auditors can distinguish controlled-in-app from paper/external artifacts.

This spec sits atop Spec 1 (Taxonomy + RBAC + Controlled Document Registry), Spec 2 (Approval State Machine), and Spec 3 (Placeholder Fill-In + Eigenpal Fanout). It consumes Spec 2's `published` transition and Spec 3's Gotenberg render pipeline.

## Architecture

Four new subsystems, one new status transition, and a small set of hook points into existing flows.

### Subsystem 1 — Publish hook (extends Spec 2)

Spec 2's state machine gains one behavior: on every transition into `published` (both direct `approved → published` and `scheduled → published`), write a single `distribution_outbox` row inside the same transaction as the status change. If the doc has `distribution_mode='passive'`, skip the outbox write (reference docs carry no attestation obligation).

### Subsystem 2 — Fan-out worker (new)

Long-running Bun process. Polls `distribution_outbox` with `SELECT ... FOR UPDATE SKIP LOCKED ORDER BY enqueued_at ASC`. Acquires advisory lock keyed on `controlled_document_id` to serialize rev-N / rev-N+1 fan-outs for the same document. Resolves recipients via `user_process_areas` JOIN at worker time. Revokes prior-version obligations (preserving `acked_at` history, setting `revoked_at + revoke_reason`). Inserts new obligations. Retries failed batches with exponential backoff 1m → 5m → 15m → 1h → 4h. After 5 attempts, moves outbox row to `distribution_outbox_dlq` and alerts.

Supports two event types:

- `publish` — full recipient fan-out. If `prior_version_id` is non-null, worker also revokes all prior-version obligations (supersession path, see Flow 6). If null, first-publish path (no prior to revoke).
- `membership_added` — scoped single-user fan-out (insert one obligation for a newly-added area member, for every currently-`published` doc in the area)

### Subsystem 3 — Cutover job (new)

Cron runs every 15 minutes per tenant. Promotes `status='scheduled'` documents to `status='published'` once `effective_date <= NOW() AT TIME ZONE tenants.timezone`. Per-tenant isolation (one tenant failure does not block others). Idempotent: rerun picks up anything missed.

### Subsystem 4 — Reconciliation job (new, safety net)

Nightly per-tenant cron at 02:00 tenant-local time. For each `status='published'` doc:

1. Computes `expected` recipient set via current `user_process_areas`
2. Diffs against `actual` non-revoked obligations
3. Inserts missing obligations (`reason='reconciliation_gap'`) — catches membership triggers that failed or were bypassed via raw SQL
4. Revokes orphan unacked obligations (`revoke_reason='orphan_cleanup'`) where the user is no longer in the area
5. Preserves acked rows regardless (acked + no longer in area = historical evidence)

Emits a `reconciliation_run_summary` metric row per tenant per run. Alerts if `gaps_found` exceeds a configurable threshold (drift smoke alarm).

### Subsystem 5 — Ack service (new)

Issues a single-use nonce (32 random bytes, 15-minute TTL) on document view when an open obligation exists for the viewing user. Validates nonce + records attestation on POST `/ack`.

- **View-ack path:** nonce + session-auth → record `acked_at`, clear `ack_nonce`
- **Signature-ack path:** nonce + password re-auth (reuses Spec 2 re-auth machinery) → record `acked_at` plus `ack_signature = hmac_sha256(server_secret, content_hash || values_hash || schema_hash || user_id || nonce || acked_at_timestamp)`

Rate-limits failed password attempts to 5 per 15-minute window per (user, obligation); breach → temporary lock, notify (email when infra arrives, in-app banner now).

### Subsystem 6 — Export service (extends Spec 3)

Wraps the Spec 3 Gotenberg PDF render pipeline. Every export:

1. Writes `document_exports` ledger row before streaming bytes
2. Injects watermark text: `UNCONTROLLED COPY — {user.email} — {timestamp} — rev {n} — verify at {tenant.domain}/doc/{code}`
3. Computes `watermark_hash = sha256(watermark_text || file_bytes_prefix)` and back-fills on the ledger row
4. Streams PDF to user

If Gotenberg fails → return 503, no ledger row (no uncontrolled copy created). If watermark injection fails → return 500, alert, never ship an unwatermarked PDF.

### Membership-change hook (new)

App-level hook on `user_process_areas` INSERT/UPDATE:

- **User added to area** → enqueue one `distribution_outbox` row of type `membership_added` per `status='published'` doc in that area, targeting the single new user. Worker fans out scoped obligation rows.
- **User removed from area** → synchronously UPDATE all non-revoked, unacked obligations for that user in docs owned by the area: `revoked_at = NOW(), revoke_reason = 'area_removed'`. Acked obligations preserved as historical evidence.

Convergence bound from membership change to obligation visibility: p99 ≤ 5 minutes (shares the fan-out SLO).

### Status lifecycle extension (to Spec 2)

Extends `doc_status_enum` with one new state: `scheduled` (approved + waiting for `effective_date` to elapse). Full lifecycle becomes:

```
draft → in_review → approved → scheduled → published → superseded → archived
                             ↘                       ↗
                              (effective_date NULL or past)
```

`approved → scheduled` if the approver set `effective_date > NOW()` at approval time; otherwise `approved → published` directly (existing Spec 2 path).

## Components

### New tables

```sql
CREATE TYPE ack_type_enum AS ENUM ('view', 'signature');

CREATE TYPE revoke_reason_enum AS ENUM (
  'superseded',
  'area_removed',
  'doc_archived',
  'user_deactivated',
  'orphan_cleanup'
);

CREATE TYPE criticality_tier_enum AS ENUM (
  'standard',     -- informational, reference
  'operational',  -- day-to-day procedures
  'safety',       -- safety-critical; signature-ack enforced
  'regulatory'    -- ISO / legal; signature-ack enforced
);

CREATE TABLE document_distributions (
  id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id             UUID NOT NULL REFERENCES tenants(id),
  doc_version_id        UUID NOT NULL REFERENCES document_versions(id),
  user_id               UUID NOT NULL REFERENCES users(id),
  resolved_via_area_id  UUID NOT NULL REFERENCES process_areas(id),
  resolved_at           TIMESTAMPTZ NOT NULL,
  ack_type              ack_type_enum NOT NULL,
  ack_nonce             TEXT,
  delivered_at          TIMESTAMPTZ,
  acked_at              TIMESTAMPTZ,
  ack_signature         TEXT,
  revoked_at            TIMESTAMPTZ,
  revoke_reason         revoke_reason_enum,
  created_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (doc_version_id, user_id)
);

CREATE INDEX idx_dist_pending_per_doc
  ON document_distributions (tenant_id, doc_version_id)
  WHERE acked_at IS NULL AND revoked_at IS NULL;

CREATE INDEX idx_dist_pending_per_user
  ON document_distributions (user_id)
  WHERE acked_at IS NULL AND revoked_at IS NULL;

CREATE TABLE distribution_outbox (
  id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id         UUID NOT NULL REFERENCES tenants(id),
  doc_version_id    UUID NOT NULL REFERENCES document_versions(id),
  event_type        TEXT NOT NULL,        -- publish | membership_added
  prior_version_id  UUID REFERENCES document_versions(id),
  target_user_id    UUID REFERENCES users(id),  -- only for membership_added
  enqueued_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  processed_at      TIMESTAMPTZ,
  attempt_count     INT NOT NULL DEFAULT 0,
  last_error        TEXT
);

CREATE INDEX idx_outbox_pending
  ON distribution_outbox (enqueued_at)
  WHERE processed_at IS NULL;

CREATE TABLE distribution_outbox_dlq (
  LIKE distribution_outbox INCLUDING ALL,
  moved_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  final_error   TEXT NOT NULL
);

CREATE TABLE document_exports (
  id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id       UUID NOT NULL REFERENCES tenants(id),
  doc_version_id  UUID NOT NULL REFERENCES document_versions(id),
  user_id         UUID NOT NULL REFERENCES users(id),
  exported_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  format          TEXT NOT NULL,         -- pdf | docx
  purpose         TEXT,                   -- optional user-provided reason
  watermark_hash  TEXT NOT NULL
);

CREATE TABLE reconciliation_run_summary (
  id                     UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id              UUID NOT NULL REFERENCES tenants(id),
  run_at                 TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  docs_checked           INT NOT NULL,
  gaps_found             INT NOT NULL,
  obligations_inserted   INT NOT NULL,
  orphans_cleaned        INT NOT NULL,
  duration_ms            INT NOT NULL
);
```

### Schema additions to existing tables

```sql
-- Spec 2
ALTER TYPE doc_status_enum ADD VALUE 'scheduled';

ALTER TABLE document_versions
  ADD COLUMN ack_type           ack_type_enum,             -- NULL → inherit from policy chain
  ADD COLUMN effective_date     TIMESTAMPTZ,               -- NULL → effective on publish
  ADD COLUMN distribution_mode  TEXT NOT NULL DEFAULT 'active'
    CHECK (distribution_mode IN ('active', 'passive'));

-- Spec 1
ALTER TABLE controlled_documents
  ADD COLUMN criticality_tier criticality_tier_enum NOT NULL DEFAULT 'standard';

ALTER TABLE process_areas
  ADD COLUMN default_ack_type ack_type_enum;  -- NULL → inherit tenant default

ALTER TABLE tenants
  ADD COLUMN timezone                  TEXT NOT NULL DEFAULT 'UTC',
  ADD COLUMN default_ack_type          ack_type_enum NOT NULL DEFAULT 'view',
  ADD COLUMN dst_ambiguity_resolution  TEXT NOT NULL DEFAULT 'earlier'
    CHECK (dst_ambiguity_resolution IN ('earlier', 'later'));

-- IANA validation enforced at app layer (pg_timezone_names lookup at save).
```

### Ack-type resolution rule (applied at publish time)

First non-null wins, in this priority order:

1. `document_versions.ack_type` — explicit author override
2. **Forced `signature`** if `controlled_documents.criticality_tier IN ('safety', 'regulatory')` — author override of `view` is rejected with HTTP 400 at publish time
3. `process_areas.default_ack_type`
4. `tenants.default_ack_type` (non-null fallback — enum NOT NULL)

Resolved value stamped on every obligation row (`document_distributions.ack_type`) so the attestation strength is frozen per rev.

### Point-of-use access rule

- GET `/documents/:version_id` allowed if user has area membership OR is tenant admin; otherwise 403
- Read access does not require an obligation (read ≠ attest)
- If an open obligation exists for the (user, version) pair, server issues a nonce and the client renders an ack prompt
- If no obligation, document is read-only with no ack prompt

This keeps casual reference reads ungated while preserving the ISO control boundary: attestation is explicit and recorded, not inferred from reads.

### Services (Bun / TypeScript modules)

- `DistributionService` — publish hook, revocation, recipient resolution helpers
- `FanoutWorker` — outbox consumer, supports `publish | supersede_cutover | membership_added`
- `CutoverJob` — scheduled → published promoter
- `ReconciliationJob` — nightly drift safety net
- `AckService` — nonce issue + ack verification
- `ExportService` — watermark + ledger; extends Spec 3
- `MembershipHook` — enqueues scoped outbox rows on `user_process_areas` INSERT; synchronous revocation on removal
- `DistributionDashboardQueries` — pending, overdue, history read models

### UI surfaces (new)

- **User inbox** — "Documents awaiting your acknowledgment" (queries `idx_dist_pending_per_user`)
- **Ack modal** — view-ack (single click) or signature-ack (password re-auth) per resolved `ack_type`
- **Manager dashboard** — per-doc `acked / pending / overdue` counts, pending-user drill-down, tenant-level reconciliation gap indicator
- **Doc header banner** — `SUPERSEDED — current rev is N` when viewing a non-current rev; click-through to current
- **Author publish UI** — ack-type picker (greyed-out if forced by criticality), optional `effective_date` input, `distribution_mode` toggle
- **Tenant admin settings** — timezone (IANA picker), `default_ack_type`, `dst_ambiguity_resolution`

## Data Flow

### Flow 1 — Publish active-mode doc (happy path)

```
1. Approver triggers Spec 2 approved → published
2. Inside same txn:
     - document_versions.status = 'published'
     - INSERT distribution_outbox (event_type='publish', doc_version_id)
     - audit_log entry
   Commit.
3. FanoutWorker picks up outbox row (SKIP LOCKED, ORDER BY enqueued_at)
4. Worker acquires advisory lock on controlled_document_id
5. Worker resolves recipients:
     SELECT DISTINCT upa.user_id, upa.area_id
     FROM user_process_areas upa
     JOIN controlled_documents cd ON cd.area_id = upa.area_id
     WHERE cd.id = :controlled_doc_id
       AND upa.tenant_id = :tenant_id
       AND upa.revoked_at IS NULL
6. Worker resolves ack_type via policy chain (version > criticality force > area > tenant)
7. INSERT document_distributions rows (one per recipient) with resolved_via_area_id + resolved_at
8. UPDATE distribution_outbox SET processed_at = NOW()
9. Release advisory lock, commit.
```

### Flow 2 — User views + acks

```
1. GET /documents/:version_id
     - Authz: area membership OR admin
     - If obligation row exists (version, user) with acked_at IS NULL and revoked_at IS NULL:
         UPDATE document_distributions
           SET delivered_at = COALESCE(delivered_at, NOW()),
               ack_nonce = :new_nonce   -- crypto-random 32 bytes, expires at NOW() + 15min
     - Return doc + (nonce, ack_type) when obligation exists, else doc alone
2. POST /ack { obligation_id, nonce [, password] }
     - Verify nonce matches stored ack_nonce AND not expired
     - view-ack: UPDATE acked_at = NOW(), ack_nonce = NULL
     - signature-ack: verify password (Spec 2 re-auth) → compute signature →
         UPDATE acked_at = NOW(), ack_signature = hmac_sha256(...), ack_nonce = NULL
3. Response: 200 OK + user's updated pending count
```

### Flow 3 — Scheduled publish with effective_date

```
1. Approver sets effective_date = 2026-05-01 08:00 (interpreted in tenant TZ)
2. Spec 2 transitions: approved → scheduled (NO outbox row)
3. Prior rev remains current and visible; new rev is invisible to recipients
4. Every 15 min, CutoverJob runs per tenant:
     SELECT id FROM document_versions
     WHERE tenant_id = :t
       AND status = 'scheduled'
       AND effective_date <= NOW() AT TIME ZONE t.timezone
5. For each matching row:
     BEGIN;
       UPDATE status = 'published';
       INSERT distribution_outbox (event_type='publish',
                                    prior_version_id = <prior published rev or NULL>);
     COMMIT;
6. Worker processes per Flow 1 + prior-version revocation (see Flow 6).
```

### Flow 4 — Grace-window delta at cutover

```
Scenario: rev 4 scheduled for 2026-05-01. Maria joins Welding area on 2026-04-28.

- 2026-04-20 (publish/approval): Maria NOT in area → no obligation
- 2026-04-28: Maria added to user_process_areas → MembershipHook fires

  For Maria's new area, for each currently-'published' doc (rev 4 is NOT yet
  published — it's 'scheduled'):
  - Rev 3 is published → enqueue membership_added outbox → Maria gets rev 3 obligation
  - Rev 4 is scheduled → NOT eligible → no obligation yet

- 2026-05-01 cutover:
  - status: scheduled → published
  - supersede_cutover outbox enqueued
  - Worker re-resolves recipients at fan-out time (Flow 1 step 5)
  - Maria now in area → gets rev 4 obligation via normal fan-out
  - Maria's rev 3 obligation (acked or not) → revoked via supersede path

Recipient set for any published rev = RBAC state at the moment of that rev's
fan-out. resolved_at on each obligation row records this timestamp.
```

### Flow 5 — Export PDF

```
1. User clicks "Export PDF" on doc viewer (any rev user is authorized to read)
2. Server:
     a. INSERT document_exports (tenant, version, user, format='pdf', purpose, watermark_hash='pending')
     b. Call Spec 3 Gotenberg with watermark injection:
          watermark_text = "UNCONTROLLED COPY — {user.email} — {ISO8601 ts}
                            — rev {n} — verify at {tenant.domain}/doc/{code}"
     c. Compute watermark_hash = sha256(watermark_text || first 4KB of output)
     d. UPDATE document_exports SET watermark_hash = <hash>
     e. Stream PDF to response
3. On Gotenberg failure: abort BEFORE step (a) commits → no ledger row, return 503
4. On watermark injection failure: return 500, alert, no PDF shipped
```

### Flow 6 — Supersession recall

```
1. Rev 5 approved + published (prior = rev 4; 14 users acked rev 4)
2. Outbox row: event_type='publish', prior_version_id=<rev-4-id>
3. Worker:
     BEGIN;
       Acquire advisory lock on controlled_document_id
       Resolve rev-5 recipient set (may differ from rev 4)
       UPDATE document_distributions
         SET revoked_at = NOW(), revoke_reason = 'superseded'
         WHERE doc_version_id = <rev-4-id>
           AND revoked_at IS NULL
       -- both acked and unacked rev-4 obligations revoked
       INSERT new rev-5 obligations
       UPDATE distribution_outbox SET processed_at = NOW()
     COMMIT;
4. Rev 4 now shows SUPERSEDED banner on view
5. Rev 5 obligations pending; all 14 (+ any new area members) must re-ack
6. Dashboard: rev 4 = "fully revoked 14/14"; rev 5 = "0/N acked"
```

### Flow 7 — User removed from area

```
1. Admin removes Maria from Welding area (user_process_areas UPDATE revoked_at)
2. MembershipHook fires synchronously:
     UPDATE document_distributions
       SET revoked_at = NOW(), revoke_reason = 'area_removed'
       WHERE user_id = :maria
         AND doc_version_id IN (
               SELECT dv.id FROM document_versions dv
               JOIN controlled_documents cd ON cd.id = dv.controlled_document_id
               WHERE cd.area_id = :welding_area
                 AND dv.status = 'published')
         AND acked_at IS NULL
         AND revoked_at IS NULL
3. Acked obligations preserved (acked_at intact, revoked_at NULL) →
   historical audit answers "Maria acked rev 3 on date X, then left area"
```

### Flow 8 — Nightly reconciliation (safety net)

```
1. ReconciliationJob fires per tenant at 02:00 tenant-local time
2. For each status='published' doc_version in tenant:
     expected = SELECT DISTINCT upa.user_id
                FROM user_process_areas upa
                JOIN controlled_documents cd ON cd.area_id = upa.area_id
                WHERE cd.id = :controlled_doc_id
                  AND upa.revoked_at IS NULL
     actual   = SELECT user_id FROM document_distributions
                WHERE doc_version_id = :v AND revoked_at IS NULL
3. missing = expected - actual → INSERT obligation rows
             (ack_type resolved via same policy chain; resolved_via_area_id set
              from current RBAC; resolved_at = NOW(); audit_log reason='reconciliation_gap')
4. orphans = (actual - expected) WHERE acked_at IS NULL
             → UPDATE revoked_at = NOW(), revoke_reason = 'orphan_cleanup'
             (acked rows preserved regardless)
5. INSERT reconciliation_run_summary row
6. If gaps_found > tenant.reconciliation_alert_threshold → alert
```

Reconciliation is a **safety net**, not the primary convergence path. The `MembershipHook` handles real-time convergence (≤ 5 min p99). Reconciliation catches:

- Raw SQL mutations of `user_process_areas` that bypassed the hook
- Hook failures that silently dropped enqueues
- Edge cases during migrations or data fixes

### Flow 9 — Membership change (real-time convergence)

```
On user_process_areas INSERT (user added to area):
  For each status='published' doc_version dv
  WHERE dv.controlled_document.area_id = :new_area:
    INSERT distribution_outbox (
      event_type='membership_added',
      doc_version_id = dv.id,
      target_user_id = :user_id
    )

FanoutWorker handling of membership_added:
  1. Acquire advisory lock on controlled_document_id
  2. Resolve ack_type via policy chain
  3. INSERT ONE document_distributions row for (doc_version_id, target_user_id)
     ON CONFLICT DO NOTHING (idempotent — UNIQUE(doc_version_id, user_id))
  4. Mark outbox processed

Convergence SLO: p99 ≤ 5 min from INSERT on user_process_areas to obligation visible.
```

### Flow 10 — DST-ambiguous effective_date

```
Tenant TZ = America/Sao_Paulo. Spring-forward day has ambiguous 00:30.

1. Author saves effective_date = 2026-10-18 00:30 (local)
2. App-level validation:
     a. Check if local time is DST-ambiguous (two valid UTC instants)
        → apply tenant.dst_ambiguity_resolution ('earlier' → pre-DST UTC;
          'later' → post-DST UTC)
     b. Check if local time is DST-gap (non-existent, e.g. spring-forward skip)
        → reject with HTTP 400 "Time does not exist on that date"
3. Store resolved TIMESTAMPTZ; cutover uses stored absolute instant (no ambiguity)
4. If tenant.dst_ambiguity_resolution is somehow unset (legacy) → reject save with 400
```

## Error Handling

### Publish-time

| Condition | Behavior |
| --- | --- |
| Outbox INSERT fails (DB down, constraint) | Spec 2 publish txn rolls back. Doc stays `approved`. UI shows retry. No partial state. |
| Outbox row orphaned (worker never picks up) | Monitoring alert: `SELECT COUNT(*) FROM distribution_outbox WHERE processed_at IS NULL AND enqueued_at < NOW() - interval '5 min'`. Worker restart recovers via SKIP LOCKED. |
| Criticality-forced signature downgrade attempt | Publish endpoint returns HTTP 400 `{ code: 'ACK_TYPE_LOCKED', required: 'signature' }`. |

### Worker-time

| Condition | Behavior |
| --- | --- |
| RBAC resolution query fails | Increment `attempt_count`, store `last_error`, leave `processed_at = NULL`. Backoff 1m/5m/15m/1h/4h. After 5 failures → DLQ + alert. |
| Partial INSERT failure (unique violation, FK) | Full worker txn rollback + retry. If a `user_id` vanished mid-resolve (user deactivated) → skip that row with warn, continue others. |
| Prior-version revocation failure | Same worker txn as new inserts. All-or-nothing. |
| Two outbox rows for same controlled_document | Advisory lock serializes them; processed in `enqueued_at` order. |

### Ack-time

| Condition | Behavior |
| --- | --- |
| Nonce mismatch | 409 Conflict `{ code: 'NONCE_INVALID' }`, instruct client to reopen doc. |
| Nonce expired (>15 min) | 409 `{ code: 'NONCE_EXPIRED' }`. |
| Double-POST with same nonce | First UPDATE clears `ack_nonce`; second UPDATE matches 0 rows. If `acked_at` already set for same (version, user) → 200 idempotent; else 409. |
| Wrong password (signature-ack) | 401. Rate-limit 5 attempts / 15 min / (user, obligation). On breach → temporary lock + notify. |
| Ack attempted on revoked obligation | 410 Gone `{ code: 'OBLIGATION_REVOKED', current_version_id: ... }`. |

### Cutover-job

| Condition | Behavior |
| --- | --- |
| Crash mid-run | Idempotent re-select picks up remaining scheduled docs on next tick. Per-doc transition atomic. |
| Tenant timezone NULL | Fall back to UTC, log warning. (Prevented by NOT NULL DEFAULT 'UTC'.) |
| effective_date in the past at publish | Treat as NULL → immediate `approved → published` path (not `scheduled`). |
| Tenant in error state (e.g. DB unavailable) | Per-tenant isolation. Other tenants continue. Failed tenant retried next tick. |

### Export

| Condition | Behavior |
| --- | --- |
| Gotenberg unreachable | 503. No ledger row (commit occurs only after bytes produced). |
| Watermark injection failure | 500, alert. Never ship unwatermarked PDF — rollback ledger row. |

### Supersession race

```
Rev-N ack at 09:15:22.100. Rev-N+1 publish at 09:15:22.150.

  - Rev-N ack commits first (its txn started earlier).
  - Rev-N+1 worker acquires advisory lock on controlled_document_id.
  - Worker's UPDATE rev-N obligations → revoked_at = NOW() (sees Rev-N ack
    already present → revoked_at set on acked row, acked_at preserved).
  - Worker INSERTs rev-N+1 obligations.
  - User sees rev-N+1 pending in inbox on next load.

Ordering enforced by: (a) advisory lock per controlled_document in worker,
(b) outbox ORDER BY enqueued_at, (c) row-level locks on UPDATE.
```

### Reconciliation

| Condition | Behavior |
| --- | --- |
| One tenant's reconciliation fails | Tenant isolated. Others proceed. Failure logged + alert. |
| Reconciliation discovers drift > threshold | Alert with tenant + gap count. Manual investigation. |
| Running while publish/cutover active | Reads through existing obligations; advisory locks on fan-out paths. Reconciliation uses separate advisory keys to avoid deadlock. |

### DST

| Condition | Behavior |
| --- | --- |
| Ambiguous local time at save | Apply `dst_ambiguity_resolution`. Store resolved UTC. |
| Gap (non-existent) local time at save | Reject 400 `{ code: 'DST_GAP', message: 'Time does not exist on that date' }`. |
| Tenant `dst_ambiguity_resolution` unset | 400 on save "Tenant DST preference required". |

### SLO breach

| Condition | Behavior |
| --- | --- |
| p99 fan-out latency > 10 min | Page on-call. Dashboard flags tenant amber/red. No auto-degraded mode; manual intervention only. |
| Outbox DLQ non-empty | Alert. Operator investigates, re-enqueues or manually fixes. |
| Reconciliation gap > threshold for consecutive nights | Escalate to product team — indicates hook is dropping events. |

### Observability

- **Metrics:** `fan_out_latency_p50/p99`, `outbox_lag`, `fan_out_failure_rate`, `ack_rate`, `overdue_count_per_tenant`, `reconciliation_gaps_per_tenant`
- **Alerts:** outbox lag > 5 min, DLQ non-empty, cutover job failure, watermark failure, reconciliation gap spike
- **Audit log:** every obligation state change (created, delivered, acked, revoked) written to Spec 2's `audit_log` with `entity_type = 'document_distribution'` and diff payload

## Testing Approach

### Unit

**DistributionService**
- Publish hook writes exactly one outbox row inside Spec 2 publish txn
- `distribution_mode='passive'` → no outbox write
- Spec 2 rollback → no orphan outbox row

**FanoutWorker**
- `publish` event: RBAC resolution produces expected recipient set
- Multi-area user gets one obligation (UNIQUE constraint enforced)
- `supersede_cutover`: all prior obligations revoked (acked + unacked), acked_at preserved
- `membership_added`: single-user INSERT with ON CONFLICT DO NOTHING (idempotent)
- Outbox processed strictly in `enqueued_at` order per controlled_document (advisory lock)
- Retry backoff schedule matches spec
- After 5 failures → DLQ

**AckService**
- View-ack: valid nonce → success; stores `acked_at`
- Signature-ack: valid nonce + password → success; `ack_signature = hmac(...)` matches expected shape
- Reused nonce → 409
- Expired nonce → 409
- Revoked obligation → 410
- Wrong password × 6 in 15 min → lock
- Ack type resolution on read matches publish-time stamped value

**CutoverJob**
- effective_date in past → promote + enqueue
- effective_date in future → no-op
- Tenant TZ respected (Sao Paulo, UTC, Tokyo)
- Missing tenant TZ → UTC + warn
- Crash mid-run → rerun completes
- Per-tenant isolation

**ReconciliationJob**
- Manually inserted drift (area membership bypassing hook) → next run inserts obligation
- Acked user removed from area → orphan_cleanup skips acked rows
- Unacked user removed from area → orphan_cleanup revokes
- Summary row emitted

**ExportService**
- Ledger row written before bytes streamed
- Watermark text present in PDF output (text-layer string match)
- `watermark_hash` deterministic for identical input
- Gotenberg down → no ledger row, 503
- Watermark failure → ledger rollback, 500, alert

**MembershipHook**
- INSERT on user_process_areas → outbox rows enqueued for all published docs in area
- Scheduled docs skipped (they'll be re-resolved at cutover)
- Removal on user_process_areas → synchronous revocation of pending obligations in that area

### Integration

- End-to-end publish + ack (tenant, area, 3 users): obligations fan out, users ack, dashboard reflects
- Supersession: rev N acked → rev N+1 published → rev N revoked + banner → rev N+1 pending
- Effective-date grace: schedule rev, advance clock, cutover fires, obligations created
- Membership-added real-time: user joins area → obligation appears within SLO window
- RBAC drift: remove user from area → trigger revokes → audit query still shows historical acks
- Outbox failure recovery: inject DB error → retry succeeds → final state idempotent
- Concurrent publish race (rev N ack vs rev N+1 publish within 50ms): correct ordering + preservation

### Property

- For any RBAC state S + doc D: recipient set at publish = `user_process_areas` filtered by area + tenant + not-revoked
- For any ack A: `ack_signature` verifies against `content_hash || values_hash || schema_hash || user_id || nonce || acked_at`
- For any superseded doc: both `acked_at` and `revoked_at` non-null on all acked obligations → audit integrity preserved
- UNIQUE(doc_version_id, user_id) never violated across concurrent publish + membership_added paths

### Load

- 500-user recipient set: p95 fan-out < 10s, p99 < 5 min
- 10 concurrent publishes across tenants: no deadlock, correct per-doc ordering
- 10k-obligation tenant dashboard query: p95 < 200 ms (validates partial-index effectiveness)

### Matrix tests

**Ack-type resolution**

| version | criticality | area default | tenant default | expected | author-override rejected? |
| --- | --- | --- | --- | --- | --- |
| — | safety | — | view | signature | yes if author picks view |
| — | regulatory | signature | view | signature | yes if author picks view |
| view | standard | signature | signature | view | no (author override wins) |
| — | standard | view | signature | view | n/a |
| — | standard | — | signature | signature | n/a |
| — | operational | — | view | view | n/a |

**DST handling**

| local input | tenant | expected behavior |
| --- | --- | --- |
| 2026-10-18 00:30 (ambiguous) | America/Sao_Paulo, earlier | stored UTC = pre-DST |
| 2026-10-18 00:30 (ambiguous) | America/Sao_Paulo, later | stored UTC = post-DST |
| 2026-02-14 02:30 (gap) | America/New_York | reject 400 DST_GAP |
| 2026-03-15 14:00 (normal) | any | stored UTC deterministic |

**Point-of-use access**

| user state | obligation | expected |
| --- | --- | --- |
| in area, no obligation | n/a | 200, read-only, no ack prompt |
| in area, open obligation | pending | 200, nonce issued, ack prompt |
| in area, acked obligation | acked | 200, no ack prompt |
| not in area, not admin | any | 403 |
| admin, any area | any | 200 (admin read always permitted) |

### Manual QA checklist

- Publish UI: ack-type picker greys out when criticality forces signature
- effective_date picker rejects DST gap times
- User inbox displays pending docs with metadata
- Ack modal routes to view vs signature per resolved ack_type
- SUPERSEDED banner visible on old rev, click-through to current
- Exported PDF has legible watermark with email + timestamp + rev
- Manager dashboard: acked / pending / overdue buckets accurate
- Non-UTC tenant cutover fires at correct local time
- Nightly reconciliation summary visible in admin observability panel

## Out of Scope

1. **Email notifications** — in-app banner only. Email deferred until email infra exists.
2. **External recipients** — contractors, customers, auditors outside tenant RBAC. Future spec.
3. **Work-cell / point-of-use physical binding** — no `work_cells` table, no QR codes, no machine-bound views. Digital-only.
4. **Print tracking** — no print-vs-download distinction. Export ledger covers both.
5. **Escalation / auto-suspend** — overdue obligations surface on manager dashboard but do not suspend area access or block other work.
6. **Digest notifications** — no daily/weekly summary emails.
7. **Teams / Slack / webhook channels** — no third-party integrations.
8. **Training records / competency matrix** — ISO §7.2. Attestations are prerequisites; training module is separate.
9. **Audit evidence export bundle** — regulator-facing signed PDF/ZIP. Separate future spec (consumes Spec 4 data).
10. **CAPA cross-linking** — ISO §10.2. Separate module.
11. **Bulk re-ack campaigns** — no admin tool to force re-ack without a new rev.
12. **Delegation of ack** — personal non-repudiation only; no proxies.
13. **Read-time analytics** — no "user spent N seconds" tracking. Binary delivered / acked.
14. **Canonical XML / DOCX reconstruction** — deferred per Spec 3; drift observability via `reconstruction_attempts` JSONB.
15. **Partitioning / archival** — obligations + exports not partitioned in Spec 4. Future ops spec once tenants exceed ~1M rows.
16. **i18n of watermark text** — English only.
17. **Mobile app** — web UI only.
18. **Sub-5-min fan-out SLA** — p99 ≤ 5 min is the commitment; real-time streaming not promised.
19. **DST gap-time forgiveness** — ambiguous times resolved per tenant pref; gap times hard-rejected.
20. **Cross-tenant reconciliation** — each tenant reconciled independently; no global consistency checks.
