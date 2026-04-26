# Foundation Spec 4 — Controlled Document Distribution + Copy Ledger Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use `superpowers-extended-cc:subagent-driven-development` (recommended) or `superpowers-extended-cc:executing-plans` to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Date:** 2026-04-24
**Spec:** [docs/superpowers/specs/2026-04-21-foundation-distribution-controlled-copy-ledger-design.md](../specs/2026-04-21-foundation-distribution-controlled-copy-ledger-design.md)
**Depends on:** Spec 1 (Taxonomy+RBAC — shipped), Spec 2 (Approval State Machine — shipped), Spec 3 (Placeholder Fill-In + Gotenberg render — shipped)
**Status:** Ready to execute (Codex-hardened via `writing-coplan`)
**Plan hardening:** Codex `gpt-5.4` (fallback `gpt-5.3-codex`) reasoning=high — single COVERAGE pass with structural revisions if needed, max 2 rounds.

**Goal:** Close the ISO §7.5.3 publish→consumer loop. Every user with legitimate need gets an explicit distribution obligation per effective revision; attests via view-ack or signature-ack (per criticality); attestations auto-revoke on supersession; watermarked PDF exports logged to a copy ledger.

**Architecture:** Six cooperating subsystems — (1) publish hook that writes `distribution_outbox` rows inside Spec 2's publish txn, (2) long-running `FanoutWorker` that resolves recipients via `user_process_areas` JOIN and inserts obligations under per-document advisory lock, (3) `CutoverJob` promoting `scheduled → published` on effective_date, (4) `MembershipHook` that enqueues scoped fan-outs on membership add and synchronously revokes on removal, (5) nonce-based `AckService` with view / signature (HMAC) paths, (6) `ExportService` extending Spec 3 Gotenberg to write watermark + ledger. Nightly `ReconciliationJob` catches drift. All state changes emit `governance_events` rows for audit.

**Tech Stack:** Go 1.22, PostgreSQL 16 (`FOR UPDATE SKIP LOCKED`, advisory locks, partial indexes), `database/sql` + `lib/pq`, React 18 + TypeScript, Vitest, Playwright, Docker Compose dev stack, existing docgen-v2 Node PDF pipeline (extended with watermark injection).

---

## Model Assignments

This plan orchestrates execution across four models. The controller stays Opus; coding is delegated to Codex (`gpt-5.3-codex`) at reasoning-effort calibrated to complexity; mechanical work goes to Haiku; medium boilerplate goes to Sonnet.

| Model | Effort | When to use | Task types |
|---|---|---|---|
| **Codex `gpt-5.3-codex` high** | high | Concurrency-critical, authz-sensitive, crypto, SQL DDL with triggers, race invariants | Advisory-lock serialization, HMAC signature computation, nonce TTL+rate limit, outbox FOR UPDATE SKIP LOCKED + backoff, reconciliation drift detection, DST resolution, race-window tests |
| **Codex `gpt-5.3-codex` medium** | medium | Domain models, services, repositories, integration tests, migrations without complex triggers | Policy chain, obligation aggregate, repository methods, publish hook extension, membership hook, cutover job, export service watermark wrapper, scheduler wiring |
| **Codex `gpt-5.3-codex` low** | low | Plain CRUD migrations, simple SQL inserts, GRANT files | Column additions, enum CREATEs, GRANT statements, index adds |
| **Sonnet** | — | HTTP CRUD boilerplate, medium React components, API hooks, form wiring | Ack handler routes, InboxPage, AckModal, ManagerDashboard, PublishDialog ack-type picker, Tenant settings page, API client hooks |
| **Haiku** | — | Trivial mechanical edits | `index.ts` exports, route registration in `module.go`/`main.go`, type additions, test fixture JSON, GRANT-only SQL files |
| **Opus** | — | Phase-end reviews only. Never coding. | Review at end of Phase 2, 4, 6, 8, 10, 12. Uses `nexus:code-reviewer` subagent with explicit `model="opus"`. |

**Effort selection rule inside Codex tier:** if the task introduces or depends on DB-level concurrency primitives (advisory locks, SKIP LOCKED, triggers), race invariants, cryptographic material, or RBAC trigger logic → **high**. If it is a service method, repository query, or integration test → **medium**. If it is a migration adding columns/indexes/GRANTs only → **low**.

---

## Codex Watch Protocol (user directive)

**Every Codex dispatch is launched in the background and watched to completion before the next task starts.** No fire-and-forget.

```
# Dispatch
node scripts/codex-companion.mjs task --background --write --model gpt-5.3-codex --effort <eff> "<prompt>"
  → returns {job_id}

# Watch loop (poll every 20–30 s)
until [[ "$(node scripts/codex-companion.mjs status <job_id> --json | jq -r '.state')" =~ ^(completed|failed|canceled)$ ]]; do
  node scripts/codex-companion.mjs status <job_id> --json | jq -r '.state,.progress'
  sleep 25
done

# Fetch
node scripts/codex-companion.mjs result <job_id> --json
```

- If `state=failed` → read `.error`, fix prompt ambiguity OR reduce task scope, re-dispatch. Never "re-run identically and hope."
- If watch loop exceeds **20 min** without state change → cancel (`cancel <job_id>`), split task in half.
- Opus never proceeds to the next task while a Codex job is `running` or `queued`.

Controller (Opus) responsibilities per task: (a) compose Codex prompt using `codex:gpt-5-4-prompting` skill, (b) dispatch with correct `--model`/`--effort`, (c) watch to completion, (d) run acceptance-criteria verify command, (e) commit, (f) advance.

---

## Architectural Adaptation — Carry-Forward from Spec 2

Spec 4 prose refers to `documents_v2.documents` and `users(id UUID)`. The shipped codebase after Spec 2 uses:

- `documents` (public schema, no `documents_v2.` prefix — that's a spec-level alias used only in documentation).
- `metaldocs.iam_users(user_id TEXT PRIMARY KEY, tenant_id UUID, deactivated_at)` — user identity is **TEXT**, not UUID.
- `documents.created_by TEXT`, `user_process_areas.user_id TEXT`, `governance_events.actor_user_id TEXT`.

**Adaptation rule:** every spec-level `user_id UUID REFERENCES users(id)` becomes `user_id TEXT REFERENCES metaldocs.iam_users(user_id)`. Every spec-level `documents_v2.documents` becomes `documents`. Every other spec element (column semantics, constraint shapes, index predicates) is implemented verbatim.

**Spec says "effective_date" — reality is "effective_from".** Spec 2 migration 0131 already added `effective_from TIMESTAMPTZ`. Spec 4 reuses that column; Flow 3 SQL in the spec is wrong (`effective_date`) and must be implemented as `effective_from`.

**Tenants table:** no dedicated `tenants` table exists. `tenant_id UUID` is a lookup attribute carried on domain rows. Spec 4 creates `metaldocs.tenants(id UUID PRIMARY KEY, timezone TEXT, default_ack_type ack_type_enum, dst_ambiguity_resolution TEXT)` — this is new, task 1.3.

**`scheduled` status:** Spec 2's superset CHECK in migration 0131 already includes `'scheduled'` in the accepted set (plan 2026-04-21 Task 1.2, line 421). Spec 4 adds no new ALTER TABLE for the CHECK; it only enforces the `approved → scheduled → published` legal-transition trigger in Task 1.11 (extending Spec 2's trigger installed by migration 0133).

---

## Actual Table Name Map

| Spec name | Actual table | Schema |
|---|---|---|
| `users` | `iam_users` | metaldocs |
| `documents_v2.documents` | `documents` | public |
| `controlled_documents` | `controlled_documents` | public |
| `process_areas` | `document_process_areas` | metaldocs |
| `user_process_areas` | `user_process_areas` | public |
| `governance_events` | `governance_events` | public |
| `tenants` (NEW — Spec 4) | `tenants` | metaldocs |
| `document_distributions` (NEW) | `document_distributions` | public |
| `distribution_outbox` (NEW) | `distribution_outbox` | public |
| `distribution_outbox_dlq` (NEW) | `distribution_outbox_dlq` | public |
| `document_exports` (NEW) | `document_exports` | public |
| `reconciliation_run_summary` (NEW) | `reconciliation_run_summary` | public |

---

## File Structure Map

### New Go files

```
internal/modules/distribution/
  domain/
    ack_type.go
    ack_type_test.go
    criticality.go
    criticality_test.go
    revoke_reason.go
    policy_chain.go
    policy_chain_test.go
    obligation.go
    obligation_test.go
    nonce.go
    nonce_test.go
    signature.go
    signature_test.go
  application/
    fanout_worker.go
    fanout_worker_test.go
    ack_service.go
    ack_service_test.go
    rate_limiter.go
    rate_limiter_test.go
    publish_hook.go
    publish_hook_test.go
    dashboard_queries.go
    dashboard_queries_test.go
  repository/
    outbox_repository.go
    outbox_repository_test.go
    distribution_repository.go
    distribution_repository_test.go
    export_repository.go
    export_repository_test.go
    tenant_repository.go
    tenant_repository_test.go
  delivery/http/
    ack_handler.go
    ack_handler_test.go
    dashboard_handler.go
    dashboard_handler_test.go
    inbox_handler.go
    inbox_handler_test.go
    tenant_settings_handler.go
    export_handler.go                    ← wraps existing render export with ledger
    types.go
  jobs/
    cutover_job.go
    cutover_job_test.go
    reconciliation_job.go
    reconciliation_job_test.go
    dst_resolver.go
    dst_resolver_test.go
  module.go
```

### Modified Go files

```
internal/modules/documents_v2/approval/application/publish_service.go
    ← write distribution_outbox row inside publish txn (event_type='publish', prior_version_id threaded)
internal/modules/documents_v2/application/export_service.go
    ← extend to call distribution.ExportService.Export (ledger + watermark)
internal/modules/iam/area_membership/area_membership.go
    ← Grant(): after txn commit, enqueue membership_added outbox rows
    ← Revoke(): synchronous UPDATE of non-acked obligations with revoke_reason='area_removed'
internal/modules/render/fanout/pdf_dispatcher.go
    ← accept watermark_text render option; pass through to docgen-v2
apps/api/cmd/metaldocs-api/main.go
    ← wire distribution module (router mount, service init, scheduler registration)
apps/worker/cmd/metaldocs-worker/main.go
    ← register FanoutWorker + CutoverJob + ReconciliationJob in scheduler runner
internal/api/v2/types_gen.go
    ← request/response structs for /ack, /inbox, /distributions/*, /tenants/:id/settings
```

### New migrations

```
migrations/
  0157_distribution_enums.sql                 ← ack_type_enum, revoke_reason_enum, criticality_tier_enum
  0158_tenants_table.sql                      ← metaldocs.tenants (id, timezone, default_ack_type, dst_ambiguity_resolution)
  0159_controlled_documents_criticality.sql   ← ADD criticality_tier
  0160_process_areas_default_ack.sql          ← ADD default_ack_type
  0161_documents_ack_type_distmode.sql        ← ADD ack_type, distribution_mode
  0162_document_distributions.sql             ← table + 2 partial indexes
  0163_distribution_outbox.sql                ← outbox + DLQ + partial index
  0164_document_exports.sql                   ← ledger
  0165_reconciliation_run_summary.sql         ← metrics row per tenant per run
  0166_scheduled_transition_trigger.sql       ← extend Spec 2 legal-transition trigger with scheduled arm
  0167_distribution_grants.sql                ← GRANTs per role (metaldocs_app, metaldocs_worker, metaldocs_security_owner)
  0168_governance_events_distribution_event_types.sql
                                              ← register event_type values: distribution.obligation_created/delivered/acked/revoked/reconciliation_gap
```

### New frontend files

```
frontend/apps/web/src/features/distribution/
  types.ts
  api.ts
  InboxPage.tsx
  AckModal.tsx
  SignatureAckModal.tsx
  ManagerDashboard.tsx
  PendingUsersPanel.tsx
  ReconciliationGapIndicator.tsx
  SupersededBanner.tsx
  hooks/
    useInbox.ts
    useObligation.ts
    useDistributionDashboard.ts
    useAck.ts
  index.ts

frontend/apps/web/src/features/tenants/
  TenantSettingsPage.tsx
  TimezonePicker.tsx
  DstResolutionToggle.tsx
  types.ts
  api.ts
  index.ts
```

### Modified frontend files

```
frontend/apps/web/src/features/documents/v2/DocumentEditorPage.tsx
    ← render SupersededBanner when viewing non-current rev; fire AckModal when obligation open
frontend/apps/web/src/features/approval/SupersedePublishDialog.tsx
    ← ack-type picker (greyed when criticality forces signature), distribution_mode toggle
frontend/apps/web/src/app/routes.tsx
    ← mount /inbox, /distributions, /tenants/:id/settings
```

---

## Phase Map

| Phase | Scope | Codex review mode | Opus review at end |
|---|---|---|---|
| 1 | DB migrations (enums, tables, extensions, legal-transition extension, grants, event types) | OPERATIONS | — |
| 2 | Distribution domain layer: ack_type, criticality, policy_chain, obligation, revoke_reason + tests | COVERAGE | ✅ |
| 3 | Outbox + FanoutWorker (FOR UPDATE SKIP LOCKED, advisory lock, backoff, DLQ, supersession revoke) | SEQUENCING | — |
| 4 | Publish hook (extends Spec 2 publish_service); passive-mode skip; prior_version threading | ARCHITECTURE | ✅ |
| 5 | Membership hook: Grant enqueues scoped membership_added; Revoke synchronously revokes unacked | COVERAGE | — |
| 6 | AckService + nonce + HMAC signature + rate limiter + HTTP handlers | QUALITY | ✅ |
| 7 | ExportService watermark + ledger (extends Spec 3 Gotenberg pipeline via docgen-v2) | ARCHITECTURE | — |
| 8 | CutoverJob (scheduled→published) + DST resolution + per-tenant isolation | OPERATIONS | ✅ |
| 9 | ReconciliationJob + metrics + governance events + alerts | COVERAGE | — |
| 10 | Frontend: Inbox, AckModal, ManagerDashboard, PublishDialog, TenantSettings | COVERAGE | ✅ |
| 11 | Integration + load + E2E Playwright (race, SLO, RBAC drift, supersession) | QUALITY | — |
| 12 | Hardening + cutover: tighten CHECK, revoke DML, rollback runbook, CI invariants, SLO alerts | OPERATIONS | ✅ |

---

## Phase Contracts (cross-phase sequencing — read before executing)

**Contract P1-extends-Spec-2 (SEQ-1):**
Migration 0166 amends Spec 2's legal-transition trigger (installed by migration 0133) to permit `approved → scheduled`, `scheduled → published`, and to reject `scheduled → approved` / `scheduled → rejected`. This is **not** a new trigger; it is `CREATE OR REPLACE FUNCTION` on the Spec 2 function body. If Spec 2 trigger is renamed or relocated, P1 migration must follow suit.

**Contract P3-outbox-before-P4 (SEQ-2):**
Phase 3 ships `outbox_repository.Enqueue` and the FanoutWorker consuming it. Phase 4 extends Spec 2 publish_service to call `outbox_repository.Enqueue` inside the publish txn. Phase 4 MUST NOT compile until Phase 3's repository interface exists. P4 acceptance test stubs the repository in unit scope and integration-tests end-to-end against the real P3 impl.

**Contract P6-nonce-opaque (SEQ-3):**
P6 AckService treats the nonce as opaque 32-byte random. The `DistributionRepository.IssueNonce(obligationID) (nonce, expires_at)` signature is stable; internal representation (column `ack_nonce TEXT`, TTL computed as `NOW() + 15min`) is repository-private. P10 UI consumes nonce as opaque base64url string via JSON.

**Contract P7-render-watermark (SEQ-4):**
P7 adds a `watermark_text` option to `render.fanout.PDFRenderOptions`. docgen-v2 must learn to accept and inject this option. If the docgen-v2 pipeline cannot inject at the PDF layer (it renders via DOCX→PDF through LibreOffice), the injection happens at the HTML-to-PDF bridge instead. P7 validates injection via text-layer grep on the rendered PDF bytes before computing `watermark_hash`.

**Contract P10-ack-type-resolution-parity (SEQ-5):**
P10 PublishDialog must display the same ack_type the server will stamp. It calls GET `/distributions/resolve-ack-type?version_id=X` which runs the exact policy chain used by the publish hook. No duplicated resolution logic in TS — single Go source of truth.

**Contract P12-DML-revoke (SEQ-6):**
P12 migrations REVOKE INSERT/UPDATE/DELETE on `distribution_outbox`, `document_distributions`, `distribution_outbox_dlq` from `metaldocs_app`. Only `metaldocs_worker` keeps DML. P12 MUST NOT land until P3–P9 have been end-to-end validated against the broadened P1 grants; otherwise the worker-only grant breaks app-layer enqueue paths that shouldn't exist but might sneak in during development.

---

## Phase 1: Database Migrations

**Intent:** DB-only phase. All schema needed by later phases lands. Extends Spec 2 legal-transition trigger (0133) with the `scheduled` arm. GRANTs stay broad (app + worker) in this phase; P12 narrows `metaldocs_app` down.

---

### Task 1.1 — Migration 0157: distribution enum types

**Model:** Codex `gpt-5.3-codex` **low** (pure DDL, no logic).

**Goal:** Create three PostgreSQL ENUM types that back ack strength, revocation reason, and criticality tier.

**Files:** Create `migrations/0157_distribution_enums.sql`.

**Acceptance Criteria:**
- [ ] `ack_type_enum` exists with values `('view','signature')`.
- [ ] `revoke_reason_enum` exists with values `('superseded','area_removed','doc_archived','user_deactivated','orphan_cleanup','reconciliation_manual')`.
- [ ] `criticality_tier_enum` exists with values `('standard','operational','safety','regulatory')`.
- [ ] Migration idempotent: rerun is a no-op.

**Steps:**

- [ ] **Step 1: Write migration**

```sql
-- migrations/0157_distribution_enums.sql
-- Spec 4 Phase 1. Three enum types backing distribution state.

BEGIN;

DO $$ BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'ack_type_enum') THEN
    CREATE TYPE ack_type_enum AS ENUM ('view','signature');
  END IF;
END $$;

DO $$ BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'revoke_reason_enum') THEN
    CREATE TYPE revoke_reason_enum AS ENUM (
      'superseded',
      'area_removed',
      'doc_archived',
      'user_deactivated',
      'orphan_cleanup',
      'reconciliation_manual'
    );
  END IF;
END $$;

DO $$ BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'criticality_tier_enum') THEN
    CREATE TYPE criticality_tier_enum AS ENUM ('standard','operational','safety','regulatory');
  END IF;
END $$;

COMMIT;
```

- [ ] **Step 2: Verify**

```bash
docker exec metaldocs-db psql -U metaldocs -d metaldocs -c "\dT+ ack_type_enum revoke_reason_enum criticality_tier_enum"
```
Expected: three types listed, each with their values.

- [ ] **Step 3: Commit**

```bash
rtk git add migrations/0157_distribution_enums.sql
rtk git commit -m "feat(spec4/phase1): distribution enum types (0157)"
```

---

### Task 1.2 — Migration 0158: `metaldocs.tenants` table

**Model:** Codex `gpt-5.3-codex` **low**.

**Goal:** Create the tenants lookup table needed by timezone + ack-type defaults. One row per tenant. `tenant_id UUID` already used as a lookup attribute across existing tables — this makes it a real FK target.

**Files:** Create `migrations/0158_tenants_table.sql`.

**Acceptance Criteria:**
- [ ] `metaldocs.tenants(id UUID PK, timezone TEXT NOT NULL DEFAULT 'UTC', default_ack_type ack_type_enum NOT NULL DEFAULT 'view', dst_ambiguity_resolution TEXT NOT NULL DEFAULT 'earlier' CHECK IN ('earlier','later'), reconciliation_alert_threshold INT NOT NULL DEFAULT 50, created_at TIMESTAMPTZ NOT NULL DEFAULT now())`.
- [ ] Backfill row for every distinct `tenant_id` currently present in `documents`, `controlled_documents`, `iam_users`, `user_process_areas`, `governance_events` — union'd. Defaults applied.
- [ ] Migration idempotent.

**Steps:**

- [ ] **Step 1: Write migration**

```sql
-- migrations/0158_tenants_table.sql
-- Spec 4 Phase 1. Promotes tenant_id from lookup attribute to real FK target.

BEGIN;

CREATE TABLE IF NOT EXISTS metaldocs.tenants (
  id                              UUID PRIMARY KEY,
  timezone                        TEXT NOT NULL DEFAULT 'UTC',
  default_ack_type                ack_type_enum NOT NULL DEFAULT 'view',
  dst_ambiguity_resolution        TEXT NOT NULL DEFAULT 'earlier'
    CHECK (dst_ambiguity_resolution IN ('earlier','later')),
  reconciliation_alert_threshold  INT NOT NULL DEFAULT 50,
  created_at                      TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Backfill from existing tenant_id columns. Use UNION to dedupe.
INSERT INTO metaldocs.tenants (id)
SELECT DISTINCT tenant_id FROM (
  SELECT tenant_id FROM documents WHERE tenant_id IS NOT NULL
  UNION
  SELECT tenant_id FROM controlled_documents WHERE tenant_id IS NOT NULL
  UNION
  SELECT tenant_id FROM metaldocs.iam_users WHERE tenant_id IS NOT NULL
  UNION
  SELECT tenant_id FROM user_process_areas WHERE tenant_id IS NOT NULL
  UNION
  SELECT tenant_id FROM governance_events WHERE tenant_id IS NOT NULL
) t
ON CONFLICT (id) DO NOTHING;

COMMIT;
```

- [ ] **Step 2: Verify**

```bash
docker exec metaldocs-db psql -U metaldocs -d metaldocs -c "SELECT COUNT(*) FROM metaldocs.tenants"
```
Expected: count equals `SELECT COUNT(DISTINCT tenant_id)` across source tables.

- [ ] **Step 3: Commit**

```bash
rtk git add migrations/0158_tenants_table.sql
rtk git commit -m "feat(spec4/phase1): metaldocs.tenants table with backfill (0158)"
```

---

### Task 1.3 — Migration 0159: `controlled_documents.criticality_tier`

**Model:** Codex `gpt-5.3-codex` **low**.

**Goal:** Every controlled document carries a criticality tier. Defaults to `'standard'`. Tiers `'safety'` and `'regulatory'` will later force signature-ack at publish time.

**Files:** Create `migrations/0159_controlled_documents_criticality.sql`.

**Acceptance Criteria:**
- [ ] `controlled_documents.criticality_tier criticality_tier_enum NOT NULL DEFAULT 'standard'` added.
- [ ] Idempotent.

**Steps:**

- [ ] **Step 1: Write migration**

```sql
-- migrations/0159_controlled_documents_criticality.sql
BEGIN;
ALTER TABLE controlled_documents
  ADD COLUMN IF NOT EXISTS criticality_tier criticality_tier_enum NOT NULL DEFAULT 'standard';
COMMIT;
```

- [ ] **Step 2: Verify**

```bash
docker exec metaldocs-db psql -U metaldocs -d metaldocs -c "\d controlled_documents" | grep criticality_tier
```
Expected: `criticality_tier | criticality_tier_enum | not null default 'standard'`.

- [ ] **Step 3: Commit**

```bash
rtk git add migrations/0159_controlled_documents_criticality.sql
rtk git commit -m "feat(spec4/phase1): controlled_documents.criticality_tier (0159)"
```

---

### Task 1.4 — Migration 0160: `process_areas.default_ack_type`

**Model:** Haiku (column-add only).

**Goal:** Area-level optional override for ack strength. NULL means inherit from tenant default.

**Files:** Create `migrations/0160_process_areas_default_ack.sql`.

**Acceptance Criteria:**
- [ ] `metaldocs.document_process_areas.default_ack_type ack_type_enum` (nullable) added.

**Steps:**

- [ ] **Step 1: Write migration**

```sql
-- migrations/0160_process_areas_default_ack.sql
BEGIN;
ALTER TABLE metaldocs.document_process_areas
  ADD COLUMN IF NOT EXISTS default_ack_type ack_type_enum;
COMMIT;
```

- [ ] **Step 2: Verify**

```bash
docker exec metaldocs-db psql -U metaldocs -d metaldocs -c "\d metaldocs.document_process_areas" | grep default_ack_type
```
Expected: column present, nullable.

- [ ] **Step 3: Commit**

```bash
rtk git add migrations/0160_process_areas_default_ack.sql
rtk git commit -m "feat(spec4/phase1): process_areas.default_ack_type (0160)"
```

---

### Task 1.5 — Migration 0161: `documents` ack_type + distribution_mode

**Model:** Codex `gpt-5.3-codex` **low**.

**Goal:** Per-revision explicit ack_type override (nullable → fall through policy chain) and `distribution_mode` (`active` triggers fan-out, `passive` skips outbox — reference docs).

**Files:** Create `migrations/0161_documents_ack_type_distmode.sql`.

**Acceptance Criteria:**
- [ ] `documents.ack_type ack_type_enum` (nullable) added.
- [ ] `documents.distribution_mode TEXT NOT NULL DEFAULT 'active' CHECK IN ('active','passive')` added.

**Steps:**

- [ ] **Step 1: Write migration**

```sql
-- migrations/0161_documents_ack_type_distmode.sql
BEGIN;

ALTER TABLE documents
  ADD COLUMN IF NOT EXISTS ack_type ack_type_enum,
  ADD COLUMN IF NOT EXISTS distribution_mode TEXT NOT NULL DEFAULT 'active';

ALTER TABLE documents
  DROP CONSTRAINT IF EXISTS documents_distribution_mode_check,
  ADD  CONSTRAINT documents_distribution_mode_check
    CHECK (distribution_mode IN ('active','passive'));

COMMIT;
```

- [ ] **Step 2: Verify**

```bash
docker exec metaldocs-db psql -U metaldocs -d metaldocs -c "\d documents" | grep -E "ack_type|distribution_mode"
```
Expected: both columns with correct types.

- [ ] **Step 3: Commit**

```bash
rtk git add migrations/0161_documents_ack_type_distmode.sql
rtk git commit -m "feat(spec4/phase1): documents.ack_type + distribution_mode (0161)"
```

---

### Task 1.6 — Migration 0162: `document_distributions` table

**Model:** Codex `gpt-5.3-codex` **medium**.

**Goal:** Obligation table. One row per (doc_version, user). Holds ack state + nonce + signature + revocation.

**Files:** Create `migrations/0162_document_distributions.sql`.

**Acceptance Criteria:**
- [ ] Table created with all columns per spec (translated UUID→TEXT for user_id).
- [ ] `UNIQUE (doc_version_id, user_id)` enforced.
- [ ] Two partial indexes: `idx_dist_pending_per_doc` and `idx_dist_pending_per_user` both `WHERE acked_at IS NULL AND revoked_at IS NULL`.
- [ ] FK `doc_version_id → documents(id)` ON DELETE RESTRICT.
- [ ] FK `user_id → metaldocs.iam_users(user_id)` ON DELETE RESTRICT.
- [ ] FK `resolved_via_area_id → metaldocs.document_process_areas(id)` ON DELETE RESTRICT.
- [ ] FK `tenant_id → metaldocs.tenants(id)` ON DELETE RESTRICT.

**Steps:**

- [ ] **Step 1: Write migration**

```sql
-- migrations/0162_document_distributions.sql
-- Spec 4 Phase 1. One obligation row per (doc_version, user).
-- user_id is TEXT per Spec 2 carry-forward.

BEGIN;

CREATE TABLE IF NOT EXISTS document_distributions (
  id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id             UUID NOT NULL REFERENCES metaldocs.tenants(id),
  doc_version_id        UUID NOT NULL REFERENCES documents(id),
  user_id               TEXT NOT NULL REFERENCES metaldocs.iam_users(user_id),
  resolved_via_area_id  UUID NOT NULL REFERENCES metaldocs.document_process_areas(id),
  resolved_at           TIMESTAMPTZ NOT NULL,
  ack_type              ack_type_enum NOT NULL,
  ack_nonce             TEXT,
  ack_nonce_expires_at  TIMESTAMPTZ,
  delivered_at          TIMESTAMPTZ,
  acked_at              TIMESTAMPTZ,
  ack_signature         TEXT,
  revoked_at            TIMESTAMPTZ,
  revoke_reason         revoke_reason_enum,
  created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (doc_version_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_dist_pending_per_doc
  ON document_distributions (tenant_id, doc_version_id)
  WHERE acked_at IS NULL AND revoked_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_dist_pending_per_user
  ON document_distributions (user_id)
  WHERE acked_at IS NULL AND revoked_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_dist_version_lookup
  ON document_distributions (doc_version_id);

COMMIT;
```

- [ ] **Step 2: Verify**

```bash
docker exec metaldocs-db psql -U metaldocs -d metaldocs -c "\d document_distributions"
```
Expected: all columns, UNIQUE constraint, 3 indexes present.

- [ ] **Step 3: Commit**

```bash
rtk git add migrations/0162_document_distributions.sql
rtk git commit -m "feat(spec4/phase1): document_distributions table (0162)"
```

---

### Task 1.7 — Migration 0163: `distribution_outbox` + DLQ

**Model:** Codex `gpt-5.3-codex` **medium**.

**Goal:** Outbox consumed by FanoutWorker. Supports `publish` and `membership_added` events. Separate DLQ table for poison messages.

**Files:** Create `migrations/0163_distribution_outbox.sql`.

**Acceptance Criteria:**
- [ ] `distribution_outbox` table with required columns.
- [ ] CHECK on `event_type IN ('publish','membership_added')`.
- [ ] `target_user_id` NULL unless `event_type='membership_added'` (CHECK).
- [ ] Partial index `idx_outbox_pending (enqueued_at) WHERE processed_at IS NULL`.
- [ ] `distribution_outbox_dlq` with same schema + `moved_at`, `final_error`.

**Steps:**

- [ ] **Step 1: Write migration**

```sql
-- migrations/0163_distribution_outbox.sql
BEGIN;

CREATE TABLE IF NOT EXISTS distribution_outbox (
  id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id         UUID NOT NULL REFERENCES metaldocs.tenants(id),
  doc_version_id    UUID NOT NULL REFERENCES documents(id),
  event_type        TEXT NOT NULL,
  prior_version_id  UUID REFERENCES documents(id),
  target_user_id    TEXT REFERENCES metaldocs.iam_users(user_id),
  enqueued_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
  processed_at      TIMESTAMPTZ,
  attempt_count     INT NOT NULL DEFAULT 0,
  last_error        TEXT,
  next_attempt_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT distribution_outbox_event_type_ck
    CHECK (event_type IN ('publish','membership_added')),
  CONSTRAINT distribution_outbox_target_user_ck
    CHECK (
      (event_type = 'membership_added' AND target_user_id IS NOT NULL)
      OR (event_type = 'publish' AND target_user_id IS NULL)
    )
);

CREATE INDEX IF NOT EXISTS idx_outbox_pending
  ON distribution_outbox (next_attempt_at)
  WHERE processed_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_outbox_controlled_doc
  ON distribution_outbox (doc_version_id);

CREATE TABLE IF NOT EXISTS distribution_outbox_dlq (
  id                UUID PRIMARY KEY,
  tenant_id         UUID NOT NULL,
  doc_version_id    UUID NOT NULL,
  event_type        TEXT NOT NULL,
  prior_version_id  UUID,
  target_user_id    TEXT,
  enqueued_at       TIMESTAMPTZ NOT NULL,
  attempt_count     INT NOT NULL,
  last_error        TEXT,
  moved_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
  final_error       TEXT NOT NULL
);

COMMIT;
```

- [ ] **Step 2: Verify**

```bash
docker exec metaldocs-db psql -U metaldocs -d metaldocs -c "\d distribution_outbox" && docker exec metaldocs-db psql -U metaldocs -d metaldocs -c "\d distribution_outbox_dlq"
```
Expected: both tables present with CHECK constraints.

- [ ] **Step 3: Commit**

```bash
rtk git add migrations/0163_distribution_outbox.sql
rtk git commit -m "feat(spec4/phase1): distribution_outbox + DLQ (0163)"
```

---

### Task 1.8 — Migration 0164: `document_exports` ledger

**Model:** Codex `gpt-5.3-codex` **low**.

**Goal:** Append-only log of every PDF/DOCX export. Watermark_hash deterministic per (text, file prefix).

**Files:** Create `migrations/0164_document_exports.sql`.

**Acceptance Criteria:**
- [ ] Table per spec with UUID PK, FKs enforced.
- [ ] `format CHECK IN ('pdf','docx')`.
- [ ] Index on `(tenant_id, exported_at DESC)`.

**Steps:**

- [ ] **Step 1: Write migration**

```sql
-- migrations/0164_document_exports.sql
BEGIN;

CREATE TABLE IF NOT EXISTS document_exports (
  id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id       UUID NOT NULL REFERENCES metaldocs.tenants(id),
  doc_version_id  UUID NOT NULL REFERENCES documents(id),
  user_id         TEXT NOT NULL REFERENCES metaldocs.iam_users(user_id),
  exported_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
  format          TEXT NOT NULL CHECK (format IN ('pdf','docx')),
  purpose         TEXT,
  watermark_hash  TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_exports_tenant_recent
  ON document_exports (tenant_id, exported_at DESC);

CREATE INDEX IF NOT EXISTS idx_exports_version
  ON document_exports (doc_version_id);

COMMIT;
```

- [ ] **Step 2: Verify**

```bash
docker exec metaldocs-db psql -U metaldocs -d metaldocs -c "\d document_exports"
```
Expected: table + 2 indexes.

- [ ] **Step 3: Commit**

```bash
rtk git add migrations/0164_document_exports.sql
rtk git commit -m "feat(spec4/phase1): document_exports ledger (0164)"
```

---

### Task 1.9 — Migration 0165: `reconciliation_run_summary`

**Model:** Haiku.

**Goal:** Metrics row per tenant per nightly reconciliation run.

**Files:** Create `migrations/0165_reconciliation_run_summary.sql`.

**Acceptance Criteria:**
- [ ] Table created with all columns per spec.
- [ ] Index on `(tenant_id, run_at DESC)`.

**Steps:**

- [ ] **Step 1: Write migration**

```sql
-- migrations/0165_reconciliation_run_summary.sql
BEGIN;

CREATE TABLE IF NOT EXISTS reconciliation_run_summary (
  id                     UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id              UUID NOT NULL REFERENCES metaldocs.tenants(id),
  run_at                 TIMESTAMPTZ NOT NULL DEFAULT now(),
  docs_checked           INT NOT NULL,
  gaps_found             INT NOT NULL,
  obligations_inserted   INT NOT NULL,
  orphans_cleaned        INT NOT NULL,
  duration_ms            INT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_recon_tenant_recent
  ON reconciliation_run_summary (tenant_id, run_at DESC);

COMMIT;
```

- [ ] **Step 2: Verify**

```bash
docker exec metaldocs-db psql -U metaldocs -d metaldocs -c "\d reconciliation_run_summary"
```
Expected: table + index present.

- [ ] **Step 3: Commit**

```bash
rtk git add migrations/0165_reconciliation_run_summary.sql
rtk git commit -m "feat(spec4/phase1): reconciliation_run_summary (0165)"
```

---

### Task 1.10 — Migration 0166: extend Spec 2 legal-transition trigger with `scheduled` arm

**Model:** Codex `gpt-5.3-codex` **high** (trigger logic, regressions risk).

**Goal:** Spec 2's trigger (installed by migration 0133) currently permits `approved → published` directly. Spec 4 needs `approved → scheduled → published` as an optional branch. The trigger must allow both paths and reject `scheduled → approved`, `scheduled → rejected`, `scheduled → draft`.

**Files:** Create `migrations/0166_scheduled_transition_trigger.sql`. Modify (CREATE OR REPLACE) the Spec 2 legal-transition function.

**Acceptance Criteria:**
- [ ] `approved → scheduled` permitted.
- [ ] `scheduled → published` permitted.
- [ ] `scheduled → approved`, `scheduled → rejected`, `scheduled → draft` rejected with SQLSTATE `22023` and message `ILLEGAL_TRANSITION`.
- [ ] All existing Spec 2 transitions still pass (regression suite).
- [ ] Spec 2 integration tests `./internal/modules/documents_v2/approval/...` stay green.

**Steps:**

- [ ] **Step 1: Locate Spec 2 trigger function**

```bash
rtk grep -n "CREATE OR REPLACE FUNCTION.*documents_legal_transition" migrations/0133*.sql
```
Expected: function name identified. Record exact name for Step 2.

- [ ] **Step 2: Write migration**

```sql
-- migrations/0166_scheduled_transition_trigger.sql
-- Spec 4 Phase 1. Extends Spec 2 legal-transition trigger with scheduled arm.

BEGIN;

CREATE OR REPLACE FUNCTION documents_legal_transition()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
DECLARE
  legal BOOLEAN := FALSE;
BEGIN
  IF OLD.status = NEW.status THEN
    RETURN NEW;
  END IF;

  legal := CASE
    WHEN OLD.status = 'draft'        AND NEW.status IN ('under_review')                         THEN TRUE
    WHEN OLD.status = 'under_review' AND NEW.status IN ('approved','rejected','draft')          THEN TRUE
    WHEN OLD.status = 'approved'     AND NEW.status IN ('published','scheduled','obsolete')     THEN TRUE
    WHEN OLD.status = 'scheduled'    AND NEW.status IN ('published','obsolete')                 THEN TRUE
    WHEN OLD.status = 'rejected'     AND NEW.status IN ('draft')                                THEN TRUE
    WHEN OLD.status = 'published'    AND NEW.status IN ('superseded','obsolete')                THEN TRUE
    WHEN OLD.status = 'superseded'   AND NEW.status IN ('obsolete')                             THEN TRUE
    ELSE FALSE
  END;

  IF NOT legal THEN
    RAISE EXCEPTION 'ILLEGAL_TRANSITION: % -> %', OLD.status, NEW.status
      USING ERRCODE = '22023';
  END IF;

  RETURN NEW;
END;
$$;

COMMIT;
```

- [ ] **Step 3: Run Spec 2 regression suite**

```bash
rtk go test ./internal/modules/documents_v2/approval/... -count=1
```
Expected: all Spec 2 tests green.

- [ ] **Step 4: Add targeted probe — scheduled→approved must raise**

```sql
-- Add to integration test file internal/modules/documents_v2/approval/integration/scheduled_transition_test.go (Task 1.10b, next task)
```

- [ ] **Step 5: Commit**

```bash
rtk git add migrations/0166_scheduled_transition_trigger.sql
rtk git commit -m "feat(spec4/phase1): extend legal-transition trigger with scheduled arm (0166)"
```

---

### Task 1.11 — Integration probe: scheduled transition legality

**Model:** Codex `gpt-5.3-codex` **medium**.

**Goal:** Regression test asserting every legal + illegal scheduled transition.

**Files:** Create `internal/modules/documents_v2/approval/integration/scheduled_transition_test.go`.

**Acceptance Criteria:**
- [ ] Test inserts doc in `approved`, transitions to `scheduled` → success.
- [ ] Test transitions `scheduled → published` → success.
- [ ] Test transitions `scheduled → approved` → fails with `22023`.
- [ ] Test transitions `scheduled → draft` → fails with `22023`.
- [ ] Test transitions `scheduled → obsolete` → success.

**Steps:**

- [ ] **Step 1: Write test**

```go
// internal/modules/documents_v2/approval/integration/scheduled_transition_test.go
package integration_test

import (
    "context"
    "testing"

    "github.com/lib/pq"
)

func TestScheduledTransitions(t *testing.T) {
    ctx := context.Background()
    db := openTestDB(t)
    docID := seedApprovedDoc(t, db)

    cases := []struct {
        name      string
        from, to  string
        wantErr   string // empty = no err
    }{
        {"approved→scheduled",  "approved",  "scheduled",  ""},
        {"scheduled→published", "scheduled", "published",  ""},
        {"scheduled→approved",  "scheduled", "approved",   "22023"},
        {"scheduled→draft",     "scheduled", "draft",      "22023"},
        {"scheduled→obsolete",  "scheduled", "obsolete",   ""},
    }
    for _, tc := range cases {
        t.Run(tc.name, func(t *testing.T) {
            setStatus(t, db, docID, tc.from)
            _, err := db.ExecContext(ctx, "UPDATE documents SET status=$1 WHERE id=$2", tc.to, docID)
            if tc.wantErr == "" && err != nil {
                t.Fatalf("expected success, got %v", err)
            }
            if tc.wantErr != "" {
                if err == nil {
                    t.Fatalf("expected SQLSTATE %s, got nil", tc.wantErr)
                }
                pqErr, ok := err.(*pq.Error)
                if !ok || string(pqErr.Code) != tc.wantErr {
                    t.Fatalf("expected SQLSTATE %s, got %v", tc.wantErr, err)
                }
            }
        })
    }
}
```

- [ ] **Step 2: Run test**

```bash
rtk go test ./internal/modules/documents_v2/approval/integration/ -run TestScheduledTransitions -v -count=1
```
Expected: 5/5 cases pass.

- [ ] **Step 3: Commit**

```bash
rtk git add internal/modules/documents_v2/approval/integration/scheduled_transition_test.go
rtk git commit -m "test(spec4/phase1): scheduled transition legality (0166 probe)"
```

---

### Task 1.12 — Migration 0167: GRANTs per role

**Model:** Haiku.

**Goal:** Grant table-level privileges to existing roles (`metaldocs_app`, `metaldocs_worker`, `metaldocs_security_owner`). Phase 1 stays broad; P12 narrows.

**Files:** Create `migrations/0167_distribution_grants.sql`.

**Acceptance Criteria:**
- [ ] `metaldocs_app`: SELECT/INSERT/UPDATE on `document_distributions`, `distribution_outbox`, `document_exports`; SELECT on `reconciliation_run_summary`, `metaldocs.tenants`, `distribution_outbox_dlq`.
- [ ] `metaldocs_worker`: full DML on all Spec-4 tables.
- [ ] `metaldocs_security_owner`: full privileges + ownership of enums.
- [ ] Idempotent — re-run succeeds.

**Steps:**

- [ ] **Step 1: Write migration**

```sql
-- migrations/0167_distribution_grants.sql
BEGIN;

GRANT SELECT, INSERT, UPDATE ON
  document_distributions,
  distribution_outbox,
  document_exports
TO metaldocs_app;

GRANT SELECT ON
  reconciliation_run_summary,
  metaldocs.tenants,
  distribution_outbox_dlq
TO metaldocs_app;

GRANT SELECT, INSERT, UPDATE, DELETE ON
  document_distributions,
  distribution_outbox,
  distribution_outbox_dlq,
  document_exports,
  reconciliation_run_summary
TO metaldocs_worker;

GRANT SELECT, INSERT, UPDATE ON metaldocs.tenants TO metaldocs_worker;

GRANT USAGE ON TYPE ack_type_enum, revoke_reason_enum, criticality_tier_enum
  TO metaldocs_app, metaldocs_worker;

COMMIT;
```

- [ ] **Step 2: Verify**

```bash
docker exec metaldocs-db psql -U metaldocs -d metaldocs -c "\dp document_distributions distribution_outbox document_exports"
```
Expected: grants shown for metaldocs_app + metaldocs_worker.

- [ ] **Step 3: Commit**

```bash
rtk git add migrations/0167_distribution_grants.sql
rtk git commit -m "feat(spec4/phase1): distribution table grants (0167)"
```

---

### Task 1.13 — Migration 0168: register distribution event types

**Model:** Haiku.

**Goal:** Seed `governance_events` allowed `event_type` registry (if such a registry exists per Spec 1; if it's a CHECK constraint, extend it). The spec emits these event types: `distribution.obligation_created`, `distribution.obligation_delivered`, `distribution.obligation_acked`, `distribution.obligation_revoked`, `distribution.reconciliation_gap`.

**Files:** Create `migrations/0168_governance_events_distribution_event_types.sql`.

**Acceptance Criteria:**
- [ ] If event_type is free-form TEXT (no CHECK): migration is a no-op with a comment. Verify via `\d governance_events`.
- [ ] If event_type has a CHECK constraint: extend to include the 5 new values.
- [ ] If event_type has a registry table: insert 5 new rows idempotently.

**Steps:**

- [ ] **Step 1: Inspect governance_events**

```bash
docker exec metaldocs-db psql -U metaldocs -d metaldocs -c "\d governance_events"
```
Record whether `event_type` has a CHECK constraint or registry FK.

- [ ] **Step 2: Write migration (branch on Step 1)**

Case A (free-form TEXT):
```sql
-- migrations/0168_governance_events_distribution_event_types.sql
-- governance_events.event_type is free-form TEXT (per Spec 1).
-- Spec 4 event_types:
--   distribution.obligation_created
--   distribution.obligation_delivered
--   distribution.obligation_acked
--   distribution.obligation_revoked
--   distribution.reconciliation_gap
-- No DDL needed. Documented here for audit trail.
BEGIN; COMMIT;
```

Case B (CHECK constraint):
```sql
BEGIN;
ALTER TABLE governance_events
  DROP CONSTRAINT IF EXISTS governance_events_event_type_check,
  ADD  CONSTRAINT governance_events_event_type_check
    CHECK (event_type IN (
      -- existing types (copy from current constraint)
      'distribution.obligation_created',
      'distribution.obligation_delivered',
      'distribution.obligation_acked',
      'distribution.obligation_revoked',
      'distribution.reconciliation_gap'
    ));
COMMIT;
```

- [ ] **Step 3: Commit**

```bash
rtk git add migrations/0168_governance_events_distribution_event_types.sql
rtk git commit -m "feat(spec4/phase1): register distribution event types (0168)"
```

---

### Phase 1 closeout

- [ ] Run full migration stack fresh: `rtk docker compose down -v && rtk docker compose up -d db && rtk make migrate`. Expect 0157–0168 apply cleanly in order.
- [ ] Run Spec 2 regression: `rtk go test ./internal/modules/documents_v2/approval/... -count=1`. Expect green.
- [ ] Commit any remaining integration fixtures.

---

## Phase 2: Distribution Domain Layer

**Intent:** Pure Go domain objects — no DB, no HTTP. The policy chain, the obligation aggregate, the nonce + HMAC signature primitives. Every file has a `_test.go` next to it. TDD enforced.

---

### Task 2.1 — `distribution/domain/ack_type.go`

**Model:** Codex `gpt-5.3-codex` **low**.

**Goal:** `AckType` enum + parse/string conversion matching DB enum.

**Files:** Create `internal/modules/distribution/domain/ack_type.go`.

**Acceptance Criteria:**
- [ ] `type AckType string` with constants `AckTypeView = "view"`, `AckTypeSignature = "signature"`.
- [ ] `ParseAckType(s string) (AckType, error)` returns error on unknown.
- [ ] `(AckType) String() string`, `(AckType) Valid() bool`.

**Steps:**

- [ ] **Step 1: Write failing test**

```go
// internal/modules/distribution/domain/ack_type_test.go
package domain_test

import (
    "testing"
    "github.com/metaldocs/internal/modules/distribution/domain"
)

func TestParseAckType(t *testing.T) {
    cases := []struct {
        in      string
        want    domain.AckType
        wantErr bool
    }{
        {"view", domain.AckTypeView, false},
        {"signature", domain.AckTypeSignature, false},
        {"", "", true},
        {"other", "", true},
    }
    for _, tc := range cases {
        got, err := domain.ParseAckType(tc.in)
        if tc.wantErr && err == nil { t.Fatalf("%q: want err, got nil", tc.in) }
        if !tc.wantErr && got != tc.want { t.Fatalf("%q: want %v, got %v", tc.in, tc.want, got) }
    }
}
```

- [ ] **Step 2: Run → FAIL (undefined)**

```bash
rtk go test ./internal/modules/distribution/domain/... -run TestParseAckType
```

- [ ] **Step 3: Implement**

```go
// internal/modules/distribution/domain/ack_type.go
package domain

import "fmt"

type AckType string

const (
    AckTypeView      AckType = "view"
    AckTypeSignature AckType = "signature"
)

func (a AckType) String() string { return string(a) }

func (a AckType) Valid() bool {
    return a == AckTypeView || a == AckTypeSignature
}

func ParseAckType(s string) (AckType, error) {
    a := AckType(s)
    if !a.Valid() {
        return "", fmt.Errorf("invalid ack_type: %q", s)
    }
    return a, nil
}
```

- [ ] **Step 4: Run → PASS**

```bash
rtk go test ./internal/modules/distribution/domain/... -run TestParseAckType
```

- [ ] **Step 5: Commit**

```bash
rtk git add internal/modules/distribution/domain/ack_type.go internal/modules/distribution/domain/ack_type_test.go
rtk git commit -m "feat(spec4/phase2): AckType enum + parse"
```

---

### Task 2.2 — `distribution/domain/criticality.go`

**Model:** Codex `gpt-5.3-codex` **low**.

**Goal:** `CriticalityTier` enum + the `ForcesSignature()` method. Safety and regulatory force signature-ack.

**Files:** Create `internal/modules/distribution/domain/criticality.go` + test.

**Acceptance Criteria:**
- [ ] Enum values `standard`, `operational`, `safety`, `regulatory`.
- [ ] `ForcesSignature() bool` returns true for `safety` + `regulatory`, false otherwise.
- [ ] `Parse` + `Valid` mirror Task 2.1.

**Steps:**

- [ ] **Step 1: Write failing test**

```go
// internal/modules/distribution/domain/criticality_test.go
package domain_test

import (
    "testing"
    "github.com/metaldocs/internal/modules/distribution/domain"
)

func TestCriticalityForcesSignature(t *testing.T) {
    cases := map[domain.CriticalityTier]bool{
        domain.CritStandard:    false,
        domain.CritOperational: false,
        domain.CritSafety:      true,
        domain.CritRegulatory:  true,
    }
    for k, want := range cases {
        if got := k.ForcesSignature(); got != want {
            t.Fatalf("%s: want %v, got %v", k, want, got)
        }
    }
}
```

- [ ] **Step 2: Run → FAIL**

- [ ] **Step 3: Implement**

```go
// internal/modules/distribution/domain/criticality.go
package domain

import "fmt"

type CriticalityTier string

const (
    CritStandard    CriticalityTier = "standard"
    CritOperational CriticalityTier = "operational"
    CritSafety      CriticalityTier = "safety"
    CritRegulatory  CriticalityTier = "regulatory"
)

func (c CriticalityTier) String() string { return string(c) }

func (c CriticalityTier) Valid() bool {
    switch c {
    case CritStandard, CritOperational, CritSafety, CritRegulatory:
        return true
    }
    return false
}

func (c CriticalityTier) ForcesSignature() bool {
    return c == CritSafety || c == CritRegulatory
}

func ParseCriticalityTier(s string) (CriticalityTier, error) {
    c := CriticalityTier(s)
    if !c.Valid() {
        return "", fmt.Errorf("invalid criticality_tier: %q", s)
    }
    return c, nil
}
```

- [ ] **Step 4: Run → PASS**

- [ ] **Step 5: Commit**

```bash
rtk git add internal/modules/distribution/domain/criticality.go internal/modules/distribution/domain/criticality_test.go
rtk git commit -m "feat(spec4/phase2): CriticalityTier + ForcesSignature"
```

---

### Task 2.3 — `distribution/domain/revoke_reason.go`

**Model:** Haiku.

**Goal:** Enum mirror.

**Files:** Create `internal/modules/distribution/domain/revoke_reason.go`.

**Acceptance Criteria:**
- [ ] Constants for all 6 DB enum values.
- [ ] `ParseRevokeReason` + `Valid`.

**Steps:**

- [ ] **Step 1: Write**

```go
// internal/modules/distribution/domain/revoke_reason.go
package domain

import "fmt"

type RevokeReason string

const (
    RevokeSuperseded           RevokeReason = "superseded"
    RevokeAreaRemoved          RevokeReason = "area_removed"
    RevokeDocArchived          RevokeReason = "doc_archived"
    RevokeUserDeactivated      RevokeReason = "user_deactivated"
    RevokeOrphanCleanup        RevokeReason = "orphan_cleanup"
    RevokeReconciliationManual RevokeReason = "reconciliation_manual"
)

func (r RevokeReason) Valid() bool {
    switch r {
    case RevokeSuperseded, RevokeAreaRemoved, RevokeDocArchived,
        RevokeUserDeactivated, RevokeOrphanCleanup, RevokeReconciliationManual:
        return true
    }
    return false
}

func ParseRevokeReason(s string) (RevokeReason, error) {
    r := RevokeReason(s)
    if !r.Valid() {
        return "", fmt.Errorf("invalid revoke_reason: %q", s)
    }
    return r, nil
}
```

- [ ] **Step 2: Compile-check + commit**

```bash
rtk go build ./internal/modules/distribution/...
rtk git add internal/modules/distribution/domain/revoke_reason.go
rtk git commit -m "feat(spec4/phase2): RevokeReason enum"
```

---

### Task 2.4 — `distribution/domain/policy_chain.go`

**Model:** Codex `gpt-5.3-codex` **medium** (matrix semantics).

**Goal:** Implement the ack_type resolution rule from the spec. First non-null wins in this priority:

1. `version.AckType` explicit override.
2. **Forced** `signature` if `tier.ForcesSignature()` — if `version.AckType == view` in that case, reject with `ErrAckTypeLocked`.
3. `area.DefaultAckType`.
4. `tenant.DefaultAckType` (non-null fallback).

**Files:** Create `internal/modules/distribution/domain/policy_chain.go` + test.

**Acceptance Criteria:**
- [ ] `ResolveAckType(version *AckType, tier CriticalityTier, areaDefault *AckType, tenantDefault AckType) (AckType, error)`.
- [ ] Error `ErrAckTypeLocked` returned when criticality forces signature but version sets view.
- [ ] All 6 matrix rows from the spec pass.

**Steps:**

- [ ] **Step 1: Write test (matrix from spec)**

```go
// internal/modules/distribution/domain/policy_chain_test.go
package domain_test

import (
    "errors"
    "testing"
    "github.com/metaldocs/internal/modules/distribution/domain"
)

func ptrAck(a domain.AckType) *domain.AckType { return &a }

func TestResolveAckType(t *testing.T) {
    cases := []struct {
        name          string
        version       *domain.AckType
        tier          domain.CriticalityTier
        areaDefault   *domain.AckType
        tenantDefault domain.AckType
        want          domain.AckType
        wantErr       error
    }{
        {
            name: "safety forces signature; no version override",
            tier: domain.CritSafety, tenantDefault: domain.AckTypeView,
            want: domain.AckTypeSignature,
        },
        {
            name: "regulatory + area=signature + tenant=view → signature",
            tier: domain.CritRegulatory, areaDefault: ptrAck(domain.AckTypeSignature),
            tenantDefault: domain.AckTypeView, want: domain.AckTypeSignature,
        },
        {
            name: "safety + author tries view → ErrAckTypeLocked",
            version: ptrAck(domain.AckTypeView), tier: domain.CritSafety,
            tenantDefault: domain.AckTypeView,
            wantErr: domain.ErrAckTypeLocked,
        },
        {
            name: "standard + author=view + area=signature → view (author wins)",
            version: ptrAck(domain.AckTypeView), tier: domain.CritStandard,
            areaDefault: ptrAck(domain.AckTypeSignature),
            tenantDefault: domain.AckTypeSignature, want: domain.AckTypeView,
        },
        {
            name: "standard + area=view + tenant=signature → view",
            tier: domain.CritStandard, areaDefault: ptrAck(domain.AckTypeView),
            tenantDefault: domain.AckTypeSignature, want: domain.AckTypeView,
        },
        {
            name: "standard + no version, no area, tenant=signature → signature",
            tier: domain.CritStandard, tenantDefault: domain.AckTypeSignature,
            want: domain.AckTypeSignature,
        },
        {
            name: "operational + tenant=view → view",
            tier: domain.CritOperational, tenantDefault: domain.AckTypeView,
            want: domain.AckTypeView,
        },
    }
    for _, tc := range cases {
        t.Run(tc.name, func(t *testing.T) {
            got, err := domain.ResolveAckType(tc.version, tc.tier, tc.areaDefault, tc.tenantDefault)
            if tc.wantErr != nil {
                if !errors.Is(err, tc.wantErr) {
                    t.Fatalf("want %v, got %v", tc.wantErr, err)
                }
                return
            }
            if err != nil { t.Fatalf("unexpected err: %v", err) }
            if got != tc.want { t.Fatalf("want %v, got %v", tc.want, got) }
        })
    }
}
```

- [ ] **Step 2: Run → FAIL**

- [ ] **Step 3: Implement**

```go
// internal/modules/distribution/domain/policy_chain.go
package domain

import "errors"

var ErrAckTypeLocked = errors.New("ack_type locked by criticality tier")

func ResolveAckType(
    version *AckType,
    tier CriticalityTier,
    areaDefault *AckType,
    tenantDefault AckType,
) (AckType, error) {
    // Step 1: version override (unless criticality forces signature and override is view)
    if tier.ForcesSignature() {
        if version != nil && *version == AckTypeView {
            return "", ErrAckTypeLocked
        }
        // Criticality wins regardless of what other layers say.
        return AckTypeSignature, nil
    }

    // Step 2: explicit author version override
    if version != nil && version.Valid() {
        return *version, nil
    }

    // Step 3: area default
    if areaDefault != nil && areaDefault.Valid() {
        return *areaDefault, nil
    }

    // Step 4: tenant default (enum NOT NULL)
    if !tenantDefault.Valid() {
        return "", errors.New("tenant default ack_type missing or invalid")
    }
    return tenantDefault, nil
}
```

- [ ] **Step 4: Run → PASS (7 cases)**

- [ ] **Step 5: Commit**

```bash
rtk git add internal/modules/distribution/domain/policy_chain.go internal/modules/distribution/domain/policy_chain_test.go
rtk git commit -m "feat(spec4/phase2): ack_type policy chain"
```

---

### Task 2.5 — `distribution/domain/obligation.go`

**Model:** Codex `gpt-5.3-codex` **medium**.

**Goal:** `Obligation` struct + state accessors (`IsPending`, `IsAcked`, `IsRevoked`). Immutable-style: mutations return new instances.

**Files:** Create `internal/modules/distribution/domain/obligation.go` + test.

**Acceptance Criteria:**
- [ ] Struct holds all columns from `document_distributions`.
- [ ] `IsPending() bool` = `acked_at == nil && revoked_at == nil`.
- [ ] `IsAcked()` / `IsRevoked()`.
- [ ] `MarkDelivered()`, `MarkAcked(signature *string)`, `Revoke(reason RevokeReason)` return new structs.

**Steps:**

- [ ] **Step 1: Write failing test**

```go
// internal/modules/distribution/domain/obligation_test.go
package domain_test

import (
    "testing"
    "time"
    "github.com/metaldocs/internal/modules/distribution/domain"
)

func TestObligationTransitions(t *testing.T) {
    o := domain.Obligation{AckType: domain.AckTypeView}
    if !o.IsPending() { t.Fatal("fresh should be pending") }

    o2 := o.MarkDelivered(time.Now())
    if o2.DeliveredAt == nil { t.Fatal("delivered expected") }
    if !o2.IsPending() { t.Fatal("delivered still pending") }

    o3 := o2.MarkAcked(time.Now(), nil)
    if !o3.IsAcked() { t.Fatal("acked expected") }
    if o3.IsPending() { t.Fatal("acked not pending") }

    o4 := o3.Revoke(time.Now(), domain.RevokeSuperseded)
    if !o4.IsRevoked() { t.Fatal("revoked expected") }
}
```

- [ ] **Step 2: Run → FAIL**

- [ ] **Step 3: Implement**

```go
// internal/modules/distribution/domain/obligation.go
package domain

import "time"

type Obligation struct {
    ID                 string
    TenantID           string
    DocVersionID       string
    UserID             string
    ResolvedViaAreaID  string
    ResolvedAt         time.Time
    AckType            AckType
    AckNonce           *string
    AckNonceExpiresAt  *time.Time
    DeliveredAt        *time.Time
    AckedAt            *time.Time
    AckSignature       *string
    RevokedAt          *time.Time
    RevokeReason       *RevokeReason
    CreatedAt          time.Time
}

func (o Obligation) IsPending() bool  { return o.AckedAt == nil && o.RevokedAt == nil }
func (o Obligation) IsAcked() bool    { return o.AckedAt != nil && o.RevokedAt == nil }
func (o Obligation) IsRevoked() bool  { return o.RevokedAt != nil }

func (o Obligation) MarkDelivered(at time.Time) Obligation {
    if o.DeliveredAt != nil {
        return o
    }
    o.DeliveredAt = &at
    return o
}

func (o Obligation) MarkAcked(at time.Time, sig *string) Obligation {
    o.AckedAt = &at
    o.AckSignature = sig
    o.AckNonce = nil
    o.AckNonceExpiresAt = nil
    return o
}

func (o Obligation) Revoke(at time.Time, reason RevokeReason) Obligation {
    o.RevokedAt = &at
    o.RevokeReason = &reason
    return o
}
```

- [ ] **Step 4: Run → PASS**

- [ ] **Step 5: Commit**

```bash
rtk git add internal/modules/distribution/domain/obligation.go internal/modules/distribution/domain/obligation_test.go
rtk git commit -m "feat(spec4/phase2): Obligation aggregate"
```

---

### Task 2.6 — `distribution/domain/nonce.go`

**Model:** Codex `gpt-5.3-codex` **high** (crypto primitive).

**Goal:** `NewNonce()` generates 32 cryptographically-random bytes, base64url-encoded (no padding). `VerifyNonce` compares constant-time. TTL = 15 min.

**Files:** Create `internal/modules/distribution/domain/nonce.go` + test.

**Acceptance Criteria:**
- [ ] `NewNonce() (token string, expiresAt time.Time, err error)` — 32 bytes, `now+15min`.
- [ ] `VerifyNonce(stored, submitted string) bool` — constant-time compare.
- [ ] Entropy test: 1000 nonces produce 1000 distinct strings.

**Steps:**

- [ ] **Step 1: Write failing test**

```go
// internal/modules/distribution/domain/nonce_test.go
package domain_test

import (
    "testing"
    "time"
    "github.com/metaldocs/internal/modules/distribution/domain"
)

func TestNonceEntropy(t *testing.T) {
    seen := make(map[string]struct{}, 1000)
    for i := 0; i < 1000; i++ {
        tok, exp, err := domain.NewNonce()
        if err != nil { t.Fatalf("err: %v", err) }
        if _, dup := seen[tok]; dup { t.Fatalf("collision at %d", i) }
        seen[tok] = struct{}{}
        if time.Until(exp) < 14*time.Minute { t.Fatalf("expiry too short") }
        if time.Until(exp) > 16*time.Minute { t.Fatalf("expiry too long") }
    }
}

func TestNonceVerify(t *testing.T) {
    tok, _, _ := domain.NewNonce()
    if !domain.VerifyNonce(tok, tok) { t.Fatal("equal tokens should verify") }
    if domain.VerifyNonce(tok, tok+"x") { t.Fatal("different tokens should not verify") }
}
```

- [ ] **Step 2: Run → FAIL**

- [ ] **Step 3: Implement**

```go
// internal/modules/distribution/domain/nonce.go
package domain

import (
    "crypto/rand"
    "crypto/subtle"
    "encoding/base64"
    "time"
)

const NonceTTL = 15 * time.Minute

func NewNonce() (string, time.Time, error) {
    buf := make([]byte, 32)
    if _, err := rand.Read(buf); err != nil {
        return "", time.Time{}, err
    }
    return base64.RawURLEncoding.EncodeToString(buf), time.Now().Add(NonceTTL), nil
}

func VerifyNonce(stored, submitted string) bool {
    if stored == "" || submitted == "" { return false }
    return subtle.ConstantTimeCompare([]byte(stored), []byte(submitted)) == 1
}
```

- [ ] **Step 4: Run → PASS**

- [ ] **Step 5: Commit**

```bash
rtk git add internal/modules/distribution/domain/nonce.go internal/modules/distribution/domain/nonce_test.go
rtk git commit -m "feat(spec4/phase2): nonce primitive + entropy test"
```

---

### Task 2.7 — `distribution/domain/signature.go`

**Model:** Codex `gpt-5.3-codex` **high** (HMAC binding).

**Goal:** Compute `ack_signature = hmac_sha256(secret, content_hash || values_hash || schema_hash || user_id || nonce || acked_at_iso8601)`. Deterministic given identical inputs. Output hex-encoded.

**Files:** Create `internal/modules/distribution/domain/signature.go` + test.

**Acceptance Criteria:**
- [ ] `ComputeAckSignature(secret []byte, inputs SignatureInputs) string`.
- [ ] Identical inputs → identical output (property test with 100 random input sets).
- [ ] Changing any single field changes the output (property test).
- [ ] Output is 64-char lowercase hex.

**Steps:**

- [ ] **Step 1: Write test**

```go
// internal/modules/distribution/domain/signature_test.go
package domain_test

import (
    "testing"
    "time"
    "github.com/metaldocs/internal/modules/distribution/domain"
)

func TestSignatureDeterministic(t *testing.T) {
    secret := []byte("test-secret")
    inputs := domain.SignatureInputs{
        ContentHash: "c", ValuesHash: "v", SchemaHash: "s",
        UserID: "u", Nonce: "n", AckedAt: time.Unix(1700000000, 0).UTC(),
    }
    a := domain.ComputeAckSignature(secret, inputs)
    b := domain.ComputeAckSignature(secret, inputs)
    if a != b { t.Fatalf("non-deterministic: %s vs %s", a, b) }
    if len(a) != 64 { t.Fatalf("want 64-char hex, got %d", len(a)) }
}

func TestSignatureFieldSensitivity(t *testing.T) {
    secret := []byte("s")
    base := domain.SignatureInputs{
        ContentHash: "c", ValuesHash: "v", SchemaHash: "s",
        UserID: "u", Nonce: "n", AckedAt: time.Unix(1, 0).UTC(),
    }
    orig := domain.ComputeAckSignature(secret, base)
    mutate := []domain.SignatureInputs{
        func() domain.SignatureInputs { x := base; x.ContentHash = "x"; return x }(),
        func() domain.SignatureInputs { x := base; x.ValuesHash = "x"; return x }(),
        func() domain.SignatureInputs { x := base; x.SchemaHash = "x"; return x }(),
        func() domain.SignatureInputs { x := base; x.UserID = "x"; return x }(),
        func() domain.SignatureInputs { x := base; x.Nonce = "x"; return x }(),
        func() domain.SignatureInputs { x := base; x.AckedAt = time.Unix(2, 0).UTC(); return x }(),
    }
    for i, m := range mutate {
        if domain.ComputeAckSignature(secret, m) == orig {
            t.Fatalf("field %d: mutation produced same signature", i)
        }
    }
}
```

- [ ] **Step 2: Run → FAIL**

- [ ] **Step 3: Implement**

```go
// internal/modules/distribution/domain/signature.go
package domain

import (
    "crypto/hmac"
    "crypto/sha256"
    "encoding/hex"
    "time"
)

type SignatureInputs struct {
    ContentHash string
    ValuesHash  string
    SchemaHash  string
    UserID      string
    Nonce       string
    AckedAt     time.Time
}

func ComputeAckSignature(secret []byte, in SignatureInputs) string {
    mac := hmac.New(sha256.New, secret)
    // Null-byte separator avoids concatenation ambiguity.
    sep := []byte{0x00}
    mac.Write([]byte(in.ContentHash)); mac.Write(sep)
    mac.Write([]byte(in.ValuesHash));  mac.Write(sep)
    mac.Write([]byte(in.SchemaHash));  mac.Write(sep)
    mac.Write([]byte(in.UserID));      mac.Write(sep)
    mac.Write([]byte(in.Nonce));       mac.Write(sep)
    mac.Write([]byte(in.AckedAt.UTC().Format(time.RFC3339Nano)))
    return hex.EncodeToString(mac.Sum(nil))
}
```

- [ ] **Step 4: Run → PASS**

- [ ] **Step 5: Commit**

```bash
rtk git add internal/modules/distribution/domain/signature.go internal/modules/distribution/domain/signature_test.go
rtk git commit -m "feat(spec4/phase2): ack signature HMAC"
```

---

### Phase 2 closeout — Opus review

Dispatch Opus review via `nexus:code-reviewer` subagent with `model="opus"`. Review scope: `internal/modules/distribution/domain/**`. Focus: type safety, policy chain matrix correctness, HMAC binding, crypto primitive correctness. Require APPROVE before Phase 3.

---

## Phase 3: Outbox + FanoutWorker

**Intent:** The scalability heart of Spec 4. Outbox is polled with `FOR UPDATE SKIP LOCKED`; worker acquires `pg_advisory_xact_lock` keyed on `controlled_document_id` to serialize rev-N / rev-N+1 fan-outs. Retries with exponential backoff; 5 strikes → DLQ. TDD + integration tests against real Postgres.

---

### Task 3.1 — `distribution/repository/outbox_repository.go` (skeleton + Enqueue)

**Model:** Codex `gpt-5.3-codex` **medium**.

**Goal:** Repository interface + `Enqueue(ctx, tx, OutboxRow) error`. Used by publish hook (Phase 4) and membership hook (Phase 5). Transaction-aware — caller passes tx for atomic enqueue with business write.

**Files:** Create `internal/modules/distribution/repository/outbox_repository.go` + test.

**Acceptance Criteria:**
- [ ] Interface `OutboxRepository` with `Enqueue(ctx, tx, OutboxRow) error` (minimum).
- [ ] Concrete `PgOutboxRepository` struct.
- [ ] Enqueue inserts into `distribution_outbox` with event_type + optional prior_version_id + optional target_user_id.
- [ ] Integration test against real Postgres: enqueue then `SELECT COUNT(*)` == 1.

**Steps:**

- [ ] **Step 1: Write failing test**

```go
// internal/modules/distribution/repository/outbox_repository_test.go
package repository_test

import (
    "context"
    "database/sql"
    "testing"

    "github.com/metaldocs/internal/modules/distribution/repository"
)

func TestEnqueuePublishEvent(t *testing.T) {
    ctx := context.Background()
    db := openTestDB(t)
    tenantID, docVerID, _ := seedTenantAndDoc(t, db)

    repo := repository.NewPgOutboxRepository(db)

    tx, err := db.BeginTx(ctx, nil)
    if err != nil { t.Fatal(err) }
    defer tx.Rollback()

    row := repository.OutboxRow{
        TenantID:     tenantID,
        DocVersionID: docVerID,
        EventType:    "publish",
    }
    if err := repo.Enqueue(ctx, tx, row); err != nil { t.Fatalf("enqueue: %v", err) }
    if err := tx.Commit(); err != nil { t.Fatal(err) }

    var n int
    if err := db.QueryRowContext(ctx,
        "SELECT COUNT(*) FROM distribution_outbox WHERE doc_version_id=$1",
        docVerID).Scan(&n); err != nil {
        t.Fatal(err)
    }
    if n != 1 { t.Fatalf("want 1 outbox row, got %d", n) }
}

var _ = sql.ErrNoRows // import guard
```

- [ ] **Step 2: Run → FAIL**

- [ ] **Step 3: Implement**

```go
// internal/modules/distribution/repository/outbox_repository.go
package repository

import (
    "context"
    "database/sql"
)

type OutboxRow struct {
    TenantID       string
    DocVersionID   string
    EventType      string  // "publish" | "membership_added"
    PriorVersionID *string
    TargetUserID   *string
}

type OutboxRepository interface {
    Enqueue(ctx context.Context, tx *sql.Tx, row OutboxRow) error
}

type PgOutboxRepository struct {
    db *sql.DB
}

func NewPgOutboxRepository(db *sql.DB) *PgOutboxRepository {
    return &PgOutboxRepository{db: db}
}

func (r *PgOutboxRepository) Enqueue(ctx context.Context, tx *sql.Tx, row OutboxRow) error {
    const q = `
        INSERT INTO distribution_outbox
            (tenant_id, doc_version_id, event_type, prior_version_id, target_user_id)
        VALUES ($1, $2, $3, $4, $5)
    `
    _, err := tx.ExecContext(ctx, q,
        row.TenantID, row.DocVersionID, row.EventType,
        row.PriorVersionID, row.TargetUserID)
    return err
}
```

- [ ] **Step 4: Run → PASS**

- [ ] **Step 5: Commit**

```bash
rtk git add internal/modules/distribution/repository/outbox_repository.go internal/modules/distribution/repository/outbox_repository_test.go
rtk git commit -m "feat(spec4/phase3): outbox repository + Enqueue"
```

---

### Task 3.2 — Outbox Claim (FOR UPDATE SKIP LOCKED)

**Model:** Codex `gpt-5.3-codex` **high** (concurrency primitive).

**Goal:** `Claim(ctx, limit int) ([]OutboxRow, tx *sql.Tx, error)` — opens a txn, selects up to `limit` rows with `next_attempt_at <= NOW() AND processed_at IS NULL ORDER BY enqueued_at ASC FOR UPDATE SKIP LOCKED`. Caller commits after processing. Also expose `MarkProcessed(tx, id)`, `FailAttempt(tx, id, error)`, `MoveToDLQ(tx, id, finalError)`.

**Files:** Extend `outbox_repository.go` + test.

**Acceptance Criteria:**
- [ ] `Claim` returns at most `limit` rows.
- [ ] Two concurrent `Claim`s skip each other's locked rows (integration test).
- [ ] `next_attempt_at` gates visibility — rows with future `next_attempt_at` are invisible.
- [ ] `MarkProcessed` sets `processed_at = NOW()`.
- [ ] `FailAttempt` increments `attempt_count`, stores `last_error`, computes `next_attempt_at` via backoff table.
- [ ] `MoveToDLQ` copies row to `distribution_outbox_dlq` and deletes from outbox.

**Steps:**

- [ ] **Step 1: Write test for SKIP LOCKED**

```go
// internal/modules/distribution/repository/outbox_repository_test.go (extend)

func TestClaimSkipLocked(t *testing.T) {
    ctx := context.Background()
    db := openTestDB(t)
    repo := repository.NewPgOutboxRepository(db)

    tenantID, v1, v2 := seedTenantAndDoc(t, db)
    _ = v2
    enqueueN(t, db, repo, tenantID, v1, 2)  // 2 rows

    // Two workers claim simultaneously
    rowsA, txA, err := repo.Claim(ctx, 10)
    if err != nil { t.Fatal(err) }
    defer txA.Rollback()

    rowsB, txB, err := repo.Claim(ctx, 10)
    if err != nil { t.Fatal(err) }
    defer txB.Rollback()

    // Each sees 1 row, never both
    if len(rowsA)+len(rowsB) != 2 { t.Fatalf("combined = %d, want 2", len(rowsA)+len(rowsB)) }
    if len(rowsA) == 2 || len(rowsB) == 2 { t.Fatal("one worker saw both rows — SKIP LOCKED broken") }
}
```

- [ ] **Step 2: Run → FAIL**

- [ ] **Step 3: Implement Claim + MarkProcessed + FailAttempt + MoveToDLQ**

```go
// Extend outbox_repository.go

var BackoffSchedule = []time.Duration{
    1 * time.Minute, 5 * time.Minute, 15 * time.Minute, 1 * time.Hour, 4 * time.Hour,
}

const MaxAttempts = 5

func (r *PgOutboxRepository) Claim(ctx context.Context, limit int) ([]OutboxRow, *sql.Tx, error) {
    tx, err := r.db.BeginTx(ctx, nil)
    if err != nil { return nil, nil, err }

    const q = `
        SELECT id, tenant_id, doc_version_id, event_type, prior_version_id, target_user_id,
               attempt_count
          FROM distribution_outbox
         WHERE processed_at IS NULL
           AND next_attempt_at <= NOW()
         ORDER BY enqueued_at ASC
         FOR UPDATE SKIP LOCKED
         LIMIT $1
    `
    rows, err := tx.QueryContext(ctx, q, limit)
    if err != nil { tx.Rollback(); return nil, nil, err }
    defer rows.Close()

    var out []OutboxRow
    for rows.Next() {
        var r OutboxRow
        if err := rows.Scan(&r.ID, &r.TenantID, &r.DocVersionID, &r.EventType,
            &r.PriorVersionID, &r.TargetUserID, &r.AttemptCount); err != nil {
            tx.Rollback(); return nil, nil, err
        }
        out = append(out, r)
    }
    return out, tx, rows.Err()
}

func (r *PgOutboxRepository) MarkProcessed(ctx context.Context, tx *sql.Tx, id string) error {
    _, err := tx.ExecContext(ctx,
        `UPDATE distribution_outbox SET processed_at=NOW() WHERE id=$1`, id)
    return err
}

func (r *PgOutboxRepository) FailAttempt(ctx context.Context, tx *sql.Tx, id string, attempts int, errMsg string) error {
    if attempts >= MaxAttempts {
        return r.moveToDLQ(ctx, tx, id, errMsg)
    }
    backoff := BackoffSchedule[attempts-1]  // attempts starts at 1 after first failure
    _, err := tx.ExecContext(ctx, `
        UPDATE distribution_outbox
           SET attempt_count = attempt_count + 1,
               last_error = $2,
               next_attempt_at = NOW() + $3::interval
         WHERE id = $1`, id, errMsg, backoff.String())
    return err
}

func (r *PgOutboxRepository) moveToDLQ(ctx context.Context, tx *sql.Tx, id, finalError string) error {
    const q = `
        WITH moved AS (
          DELETE FROM distribution_outbox WHERE id = $1
          RETURNING *
        )
        INSERT INTO distribution_outbox_dlq
          (id, tenant_id, doc_version_id, event_type, prior_version_id, target_user_id,
           enqueued_at, attempt_count, last_error, final_error)
        SELECT id, tenant_id, doc_version_id, event_type, prior_version_id, target_user_id,
               enqueued_at, attempt_count, last_error, $2
          FROM moved
    `
    _, err := tx.ExecContext(ctx, q, id, finalError)
    return err
}
```

Also extend `OutboxRow` with `ID string` and `AttemptCount int`.

- [ ] **Step 4: Run → PASS + add backoff matrix test + DLQ test**

- [ ] **Step 5: Commit**

```bash
rtk git add internal/modules/distribution/repository/outbox_repository.go internal/modules/distribution/repository/outbox_repository_test.go
rtk git commit -m "feat(spec4/phase3): outbox Claim/MarkProcessed/FailAttempt/DLQ"
```

---

### Task 3.3 — `distribution/repository/distribution_repository.go`

**Model:** Codex `gpt-5.3-codex` **medium**.

**Goal:** CRUD for `document_distributions`. Methods: `InsertObligations(ctx, tx, []Obligation)`, `RevokePriorVersion(ctx, tx, priorVersionID, reason)`, `ResolveByUserAndVersion(ctx, userID, versionID)`, `SetNonce(ctx, tx, obligationID, token, expires)`, `MarkAcked(ctx, tx, obligationID, ackedAt, signature *string)`, `ResolveRecipientSet(ctx, controlledDocID, tenantID)`.

**Files:** Create `internal/modules/distribution/repository/distribution_repository.go` + test.

**Acceptance Criteria:**
- [ ] `InsertObligations` uses batch insert; honors `ON CONFLICT (doc_version_id, user_id) DO NOTHING`.
- [ ] `RevokePriorVersion` UPDATEs `revoked_at=NOW(), revoke_reason=$reason WHERE doc_version_id=$ver AND revoked_at IS NULL`.
- [ ] `ResolveRecipientSet` JOIN of `user_process_areas` × `controlled_documents` per spec Flow 1 step 5, returns `(user_id, area_id)`.
- [ ] `MarkAcked` clears nonce + sets acked_at/signature atomically.
- [ ] Unit tests per method against real Postgres.

**Steps:**

- [ ] **Step 1: Write failing tests (one per method)**

```go
// internal/modules/distribution/repository/distribution_repository_test.go
// Test cases:
//   TestInsertObligationsIdempotent — same (version,user) → 1 row
//   TestRevokePriorVersion — acked rows revoked + acked_at preserved
//   TestResolveRecipientSet — users only in target area are returned
//   TestSetNonce — nonce + expires_at persisted
//   TestMarkAcked — nonce cleared, acked_at set
// (See implementation patterns from Spec 2 approval_repository_test.go)
```

Test bodies follow the same structure as Task 3.1 — seed, call method, query DB, assert. Keep bodies short; one behavior per test.

- [ ] **Step 2: Run → FAIL**

- [ ] **Step 3: Implement**

```go
// internal/modules/distribution/repository/distribution_repository.go
package repository

import (
    "context"
    "database/sql"
    "time"

    "github.com/metaldocs/internal/modules/distribution/domain"
)

type RecipientRow struct {
    UserID string
    AreaID string
}

type DistributionRepository interface {
    InsertObligations(ctx context.Context, tx *sql.Tx, obs []domain.Obligation) error
    RevokePriorVersion(ctx context.Context, tx *sql.Tx, priorVersionID string, reason domain.RevokeReason) (int, error)
    RevokeByUserInArea(ctx context.Context, tx *sql.Tx, userID, areaCode string, reason domain.RevokeReason) (int, error)
    ResolveRecipientSet(ctx context.Context, controlledDocID, tenantID string) ([]RecipientRow, error)
    ResolveByUserAndVersion(ctx context.Context, userID, versionID string) (*domain.Obligation, error)
    SetNonce(ctx context.Context, tx *sql.Tx, obligationID, token string, expires time.Time) error
    MarkAcked(ctx context.Context, tx *sql.Tx, obligationID string, at time.Time, signature *string) error
}

type PgDistributionRepository struct{ db *sql.DB }

func NewPgDistributionRepository(db *sql.DB) *PgDistributionRepository {
    return &PgDistributionRepository{db: db}
}

func (r *PgDistributionRepository) InsertObligations(ctx context.Context, tx *sql.Tx, obs []domain.Obligation) error {
    if len(obs) == 0 { return nil }
    const q = `
        INSERT INTO document_distributions
          (tenant_id, doc_version_id, user_id, resolved_via_area_id, resolved_at, ack_type)
        VALUES ($1, $2, $3, $4, $5, $6)
        ON CONFLICT (doc_version_id, user_id) DO NOTHING
    `
    for _, o := range obs {
        if _, err := tx.ExecContext(ctx, q,
            o.TenantID, o.DocVersionID, o.UserID,
            o.ResolvedViaAreaID, o.ResolvedAt, o.AckType); err != nil {
            return err
        }
    }
    return nil
}

func (r *PgDistributionRepository) RevokePriorVersion(
    ctx context.Context, tx *sql.Tx, priorVersionID string, reason domain.RevokeReason,
) (int, error) {
    const q = `
        UPDATE document_distributions
           SET revoked_at = NOW(), revoke_reason = $2
         WHERE doc_version_id = $1
           AND revoked_at IS NULL
    `
    res, err := tx.ExecContext(ctx, q, priorVersionID, reason)
    if err != nil { return 0, err }
    n, _ := res.RowsAffected()
    return int(n), nil
}

func (r *PgDistributionRepository) RevokeByUserInArea(
    ctx context.Context, tx *sql.Tx, userID, areaCode string, reason domain.RevokeReason,
) (int, error) {
    const q = `
        UPDATE document_distributions dd
           SET revoked_at = NOW(), revoke_reason = $3
          FROM documents d
          JOIN controlled_documents cd ON cd.id = d.controlled_document_id
         WHERE dd.doc_version_id = d.id
           AND dd.user_id = $1
           AND cd.process_area_code = $2
           AND d.status = 'published'
           AND dd.acked_at IS NULL
           AND dd.revoked_at IS NULL
    `
    res, err := tx.ExecContext(ctx, q, userID, areaCode, reason)
    if err != nil { return 0, err }
    n, _ := res.RowsAffected()
    return int(n), nil
}

func (r *PgDistributionRepository) ResolveRecipientSet(
    ctx context.Context, controlledDocID, tenantID string,
) ([]RecipientRow, error) {
    const q = `
        SELECT DISTINCT upa.user_id, pa.id
          FROM user_process_areas upa
          JOIN controlled_documents cd ON cd.process_area_code = upa.area_code
          JOIN metaldocs.document_process_areas pa ON pa.code = upa.area_code
         WHERE cd.id = $1
           AND upa.tenant_id = $2
           AND upa.effective_to IS NULL
    `
    rows, err := r.db.QueryContext(ctx, q, controlledDocID, tenantID)
    if err != nil { return nil, err }
    defer rows.Close()
    var out []RecipientRow
    for rows.Next() {
        var r RecipientRow
        if err := rows.Scan(&r.UserID, &r.AreaID); err != nil { return nil, err }
        out = append(out, r)
    }
    return out, rows.Err()
}

func (r *PgDistributionRepository) ResolveByUserAndVersion(
    ctx context.Context, userID, versionID string,
) (*domain.Obligation, error) {
    const q = `
        SELECT id, tenant_id, doc_version_id, user_id, resolved_via_area_id, resolved_at,
               ack_type, ack_nonce, ack_nonce_expires_at, delivered_at, acked_at,
               ack_signature, revoked_at, revoke_reason, created_at
          FROM document_distributions
         WHERE user_id = $1 AND doc_version_id = $2
    `
    var o domain.Obligation
    err := r.db.QueryRowContext(ctx, q, userID, versionID).Scan(
        &o.ID, &o.TenantID, &o.DocVersionID, &o.UserID, &o.ResolvedViaAreaID, &o.ResolvedAt,
        &o.AckType, &o.AckNonce, &o.AckNonceExpiresAt, &o.DeliveredAt, &o.AckedAt,
        &o.AckSignature, &o.RevokedAt, &o.RevokeReason, &o.CreatedAt)
    if err == sql.ErrNoRows { return nil, nil }
    if err != nil { return nil, err }
    return &o, nil
}

func (r *PgDistributionRepository) SetNonce(
    ctx context.Context, tx *sql.Tx, obligationID, token string, expires time.Time,
) error {
    const q = `
        UPDATE document_distributions
           SET ack_nonce = $2, ack_nonce_expires_at = $3,
               delivered_at = COALESCE(delivered_at, NOW())
         WHERE id = $1 AND acked_at IS NULL AND revoked_at IS NULL
    `
    res, err := tx.ExecContext(ctx, q, obligationID, token, expires)
    if err != nil { return err }
    n, _ := res.RowsAffected()
    if n == 0 { return sql.ErrNoRows }
    return nil
}

func (r *PgDistributionRepository) MarkAcked(
    ctx context.Context, tx *sql.Tx, obligationID string, at time.Time, signature *string,
) error {
    const q = `
        UPDATE document_distributions
           SET acked_at = $2, ack_signature = $3, ack_nonce = NULL, ack_nonce_expires_at = NULL
         WHERE id = $1 AND acked_at IS NULL AND revoked_at IS NULL
    `
    res, err := tx.ExecContext(ctx, q, obligationID, at, signature)
    if err != nil { return err }
    n, _ := res.RowsAffected()
    if n == 0 { return sql.ErrNoRows }
    return nil
}
```

- [ ] **Step 4: Run → PASS**

- [ ] **Step 5: Commit**

```bash
rtk git add internal/modules/distribution/repository/distribution_repository.go internal/modules/distribution/repository/distribution_repository_test.go
rtk git commit -m "feat(spec4/phase3): distribution repository"
```

---

### Task 3.4 — `distribution/application/fanout_worker.go` (publish path)

**Model:** Codex `gpt-5.3-codex` **high** (concurrency + advisory lock + transactional invariants).

**Goal:** Worker tick loop: claim outbox rows → for each, acquire `pg_advisory_xact_lock(hash(controlled_document_id))` → resolve recipients → resolve ack_type via policy chain → INSERT obligations → if `prior_version_id` set, RevokePriorVersion → MarkProcessed. All in one worker txn. On error → FailAttempt (same tx fresh).

**Files:** Create `internal/modules/distribution/application/fanout_worker.go` + test.

**Acceptance Criteria:**
- [ ] `FanoutWorker.Tick(ctx) (processed int, err error)`.
- [ ] `publish` event with `prior_version_id = nil` → inserts obligations only (no revoke).
- [ ] `publish` event with `prior_version_id` set → revokes prior AND inserts new, same txn.
- [ ] Two concurrent publishes for same controlled_document_id are serialized (advisory lock).
- [ ] Acquires advisory lock keyed on `controlled_document_id` via `pg_advisory_xact_lock(hashtextextended($cdID::text, 0))`.
- [ ] Criticality forces signature for tier-constrained docs (integration-test path).

**Steps:**

- [ ] **Step 1: Write failing test — first-publish path**

```go
// internal/modules/distribution/application/fanout_worker_test.go
package application_test

import (
    "context"
    "testing"
    "github.com/metaldocs/internal/modules/distribution/application"
    "github.com/metaldocs/internal/modules/distribution/repository"
)

func TestFanoutPublishFirstTime(t *testing.T) {
    ctx := context.Background()
    db := openTestDB(t)
    tenantID, cdID, docVerID := seedControlledDocAndVersion(t, db, "area-welding", "operational")
    _ = seedUserInArea(t, db, tenantID, "u1", "area-welding")
    _ = seedUserInArea(t, db, tenantID, "u2", "area-welding")

    outboxRepo := repository.NewPgOutboxRepository(db)
    distRepo := repository.NewPgDistributionRepository(db)

    tx, _ := db.BeginTx(ctx, nil)
    outboxRepo.Enqueue(ctx, tx, repository.OutboxRow{
        TenantID: tenantID, DocVersionID: docVerID, EventType: "publish",
    })
    tx.Commit()

    w := application.NewFanoutWorker(db, outboxRepo, distRepo, /* tenantRepo */ nil)
    n, err := w.Tick(ctx)
    if err != nil { t.Fatalf("tick: %v", err) }
    if n != 1 { t.Fatalf("want processed=1, got %d", n) }

    var count int
    db.QueryRow("SELECT COUNT(*) FROM document_distributions WHERE doc_version_id=$1", docVerID).Scan(&count)
    if count != 2 { t.Fatalf("want 2 obligations, got %d", count) }
}
```

- [ ] **Step 2: Run → FAIL**

- [ ] **Step 3: Implement**

```go
// internal/modules/distribution/application/fanout_worker.go
package application

import (
    "context"
    "database/sql"
    "fmt"
    "time"

    "github.com/metaldocs/internal/modules/distribution/domain"
    "github.com/metaldocs/internal/modules/distribution/repository"
)

type TenantDefaults struct {
    DefaultAckType domain.AckType
}

type VersionContext struct {
    ControlledDocID   string
    TenantID          string
    VersionAckType    *domain.AckType
    CriticalityTier   domain.CriticalityTier
    AreaDefault       *domain.AckType
    TenantDefault     domain.AckType
}

type ContextResolver interface {
    ResolveVersion(ctx context.Context, versionID string) (*VersionContext, error)
}

type FanoutWorker struct {
    db       *sql.DB
    outbox   repository.OutboxRepository
    dist     repository.DistributionRepository
    resolver ContextResolver
    batch    int
}

func NewFanoutWorker(
    db *sql.DB,
    outbox repository.OutboxRepository,
    dist repository.DistributionRepository,
    resolver ContextResolver,
) *FanoutWorker {
    return &FanoutWorker{db: db, outbox: outbox, dist: dist, resolver: resolver, batch: 20}
}

func (w *FanoutWorker) Tick(ctx context.Context) (int, error) {
    rows, tx, err := w.outbox.(interface {
        Claim(context.Context, int) ([]repository.OutboxRow, *sql.Tx, error)
    }).Claim(ctx, w.batch)
    if err != nil { return 0, err }

    processed := 0
    for _, row := range rows {
        if err := w.processOne(ctx, tx, row); err != nil {
            // Mark failure in a fresh tx (current tx will roll back on commit failure).
            if failErr := w.markFailure(ctx, row.ID, row.AttemptCount, err.Error()); failErr != nil {
                return processed, fmt.Errorf("failure path: %w (orig: %v)", failErr, err)
            }
            continue
        }
        processed++
    }
    if err := tx.Commit(); err != nil {
        return processed, err
    }
    return processed, nil
}

func (w *FanoutWorker) processOne(ctx context.Context, tx *sql.Tx, row repository.OutboxRow) error {
    vctx, err := w.resolver.ResolveVersion(ctx, row.DocVersionID)
    if err != nil { return fmt.Errorf("resolve version: %w", err) }

    // Advisory lock — serializes fan-outs for the same controlled_document.
    if _, err := tx.ExecContext(ctx,
        `SELECT pg_advisory_xact_lock(hashtextextended($1::text, 0))`,
        vctx.ControlledDocID); err != nil {
        return fmt.Errorf("advisory lock: %w", err)
    }

    ackType, err := domain.ResolveAckType(
        vctx.VersionAckType, vctx.CriticalityTier, vctx.AreaDefault, vctx.TenantDefault)
    if err != nil { return fmt.Errorf("resolve ack_type: %w", err) }

    switch row.EventType {
    case "publish":
        return w.processPublish(ctx, tx, row, vctx, ackType)
    case "membership_added":
        return w.processMembership(ctx, tx, row, vctx, ackType)
    default:
        return fmt.Errorf("unknown event_type: %s", row.EventType)
    }
}

func (w *FanoutWorker) processPublish(
    ctx context.Context, tx *sql.Tx, row repository.OutboxRow,
    vctx *VersionContext, ackType domain.AckType,
) error {
    recipients, err := w.dist.ResolveRecipientSet(ctx, vctx.ControlledDocID, row.TenantID)
    if err != nil { return err }

    now := time.Now()
    obs := make([]domain.Obligation, 0, len(recipients))
    for _, rcpt := range recipients {
        obs = append(obs, domain.Obligation{
            TenantID: row.TenantID, DocVersionID: row.DocVersionID,
            UserID: rcpt.UserID, ResolvedViaAreaID: rcpt.AreaID,
            ResolvedAt: now, AckType: ackType,
        })
    }
    if err := w.dist.InsertObligations(ctx, tx, obs); err != nil { return err }

    if row.PriorVersionID != nil {
        if _, err := w.dist.RevokePriorVersion(ctx, tx, *row.PriorVersionID, domain.RevokeSuperseded); err != nil {
            return err
        }
    }

    return w.outbox.(interface {
        MarkProcessed(context.Context, *sql.Tx, string) error
    }).MarkProcessed(ctx, tx, row.ID)
}

func (w *FanoutWorker) processMembership(
    ctx context.Context, tx *sql.Tx, row repository.OutboxRow,
    vctx *VersionContext, ackType domain.AckType,
) error {
    if row.TargetUserID == nil {
        return fmt.Errorf("membership_added requires target_user_id")
    }
    // Single-user insert; ON CONFLICT DO NOTHING covers the duplicate case
    // where ReconciliationJob inserted the row first.
    // ResolvedViaAreaID is the area that granted this membership — look it up
    // via the controlled_document's process_area_code.
    areaID, err := w.resolveAreaID(ctx, vctx.ControlledDocID)
    if err != nil { return err }

    o := domain.Obligation{
        TenantID: row.TenantID, DocVersionID: row.DocVersionID,
        UserID: *row.TargetUserID, ResolvedViaAreaID: areaID,
        ResolvedAt: time.Now(), AckType: ackType,
    }
    if err := w.dist.InsertObligations(ctx, tx, []domain.Obligation{o}); err != nil {
        return err
    }
    return w.outbox.(interface {
        MarkProcessed(context.Context, *sql.Tx, string) error
    }).MarkProcessed(ctx, tx, row.ID)
}

func (w *FanoutWorker) resolveAreaID(ctx context.Context, cdID string) (string, error) {
    var id string
    err := w.db.QueryRowContext(ctx, `
        SELECT pa.id
          FROM controlled_documents cd
          JOIN metaldocs.document_process_areas pa ON pa.code = cd.process_area_code
         WHERE cd.id = $1`, cdID).Scan(&id)
    return id, err
}

func (w *FanoutWorker) markFailure(ctx context.Context, id string, attempts int, msg string) error {
    tx, err := w.db.BeginTx(ctx, nil)
    if err != nil { return err }
    defer tx.Rollback()
    if err := w.outbox.(interface {
        FailAttempt(context.Context, *sql.Tx, string, int, string) error
    }).FailAttempt(ctx, tx, id, attempts+1, msg); err != nil {
        return err
    }
    return tx.Commit()
}
```

- [ ] **Step 4: Run → PASS**

- [ ] **Step 5: Commit**

```bash
rtk git add internal/modules/distribution/application/fanout_worker.go internal/modules/distribution/application/fanout_worker_test.go
rtk git commit -m "feat(spec4/phase3): fanout worker publish path"
```

---

### Task 3.5 — Fanout Worker: supersession path test

**Model:** Codex `gpt-5.3-codex` **high** (race invariant).

**Goal:** Test that rev-5 publish with `prior_version_id=rev-4-id` revokes all rev-4 obligations (acked + unacked) and inserts rev-5 obligations, in one txn.

**Files:** Add to `fanout_worker_test.go`.

**Acceptance Criteria:**
- [ ] Seed rev 4 published with 14 obligations (some acked, some pending).
- [ ] Enqueue rev 5 publish with `prior_version_id=rev-4-id`.
- [ ] After Tick: every rev-4 obligation has `revoked_at` set + `revoke_reason='superseded'`; acked ones retain `acked_at`.
- [ ] Rev-5 obligations exist per current recipient set.

**Steps:**

- [ ] **Step 1: Write test**

```go
func TestFanoutSupersedes(t *testing.T) {
    ctx := context.Background()
    db := openTestDB(t)
    tenantID, cdID, rev4 := seedControlledDocAndVersion(t, db, "area-welding", "operational")
    users := seedNUsersInArea(t, db, tenantID, 14, "area-welding")

    // Seed rev-4 obligations (mix acked + pending)
    seedObligations(t, db, tenantID, rev4, users)
    ackSome(t, db, rev4, users[:5])

    // Publish rev 5
    rev5 := seedNewVersion(t, db, cdID, tenantID)
    outboxRepo := repository.NewPgOutboxRepository(db)
    distRepo := repository.NewPgDistributionRepository(db)

    priorID := rev4
    tx, _ := db.BeginTx(ctx, nil)
    outboxRepo.Enqueue(ctx, tx, repository.OutboxRow{
        TenantID: tenantID, DocVersionID: rev5, EventType: "publish",
        PriorVersionID: &priorID,
    })
    tx.Commit()

    w := newFakeWorker(db, outboxRepo, distRepo)
    if _, err := w.Tick(ctx); err != nil { t.Fatal(err) }

    // rev-4 fully revoked
    var r4count int
    db.QueryRow(`SELECT COUNT(*) FROM document_distributions WHERE doc_version_id=$1 AND revoked_at IS NULL`, rev4).Scan(&r4count)
    if r4count != 0 { t.Fatalf("rev-4 should have 0 non-revoked, got %d", r4count) }

    // rev-4 acked rows preserve acked_at
    var r4acked int
    db.QueryRow(`SELECT COUNT(*) FROM document_distributions WHERE doc_version_id=$1 AND acked_at IS NOT NULL`, rev4).Scan(&r4acked)
    if r4acked != 5 { t.Fatalf("rev-4 acked should stay 5, got %d", r4acked) }

    // rev-5 recipient set
    var r5count int
    db.QueryRow(`SELECT COUNT(*) FROM document_distributions WHERE doc_version_id=$1`, rev5).Scan(&r5count)
    if r5count != 14 { t.Fatalf("rev-5 should have 14, got %d", r5count) }
}
```

- [ ] **Step 2: Run → PASS (using Task 3.4 implementation)**

- [ ] **Step 3: Commit**

```bash
rtk git add internal/modules/distribution/application/fanout_worker_test.go
rtk git commit -m "test(spec4/phase3): supersession path preserves acked_at"
```

---

### Task 3.6 — Advisory lock serialization test

**Model:** Codex `gpt-5.3-codex` **high** (concurrency test — needs careful setup).

**Goal:** Simultaneously publish rev-N+1 and rev-N+2 for same controlled document. Assert they process in enqueue order (not interleaved).

**Files:** Add to `fanout_worker_test.go`.

**Acceptance Criteria:**
- [ ] Two goroutines run `worker.Tick(ctx)` on same DB.
- [ ] Rev-N+2 fan-out never observes rev-N+1 obligations mid-flight.
- [ ] Final state: rev-N+2 obligations for all recipients; rev-N+1 all revoked.

**Steps:**

- [ ] **Step 1: Write test (scoped)**

```go
func TestAdvisoryLockSerializesOrdering(t *testing.T) {
    ctx := context.Background()
    db := openTestDB(t)
    tenantID, cdID, revN1 := seedControlledDocAndVersion(t, db, "area-welding", "operational")
    users := seedNUsersInArea(t, db, tenantID, 5, "area-welding")
    _ = users

    revN2 := seedNewVersion(t, db, cdID, tenantID)

    outbox := repository.NewPgOutboxRepository(db)
    dist := repository.NewPgDistributionRepository(db)

    // Enqueue both in order N1, N2
    tx, _ := db.BeginTx(ctx, nil)
    outbox.Enqueue(ctx, tx, repository.OutboxRow{TenantID: tenantID, DocVersionID: revN1, EventType: "publish"})
    priorN1 := revN1
    outbox.Enqueue(ctx, tx, repository.OutboxRow{TenantID: tenantID, DocVersionID: revN2, EventType: "publish", PriorVersionID: &priorN1})
    tx.Commit()

    // Two workers fire concurrently
    done := make(chan error, 2)
    for i := 0; i < 2; i++ {
        go func() {
            w := newFakeWorker(db, outbox, dist)
            _, err := w.Tick(ctx)
            done <- err
        }()
    }
    for i := 0; i < 2; i++ { if err := <-done; err != nil { t.Fatal(err) } }

    // Final: revN1 all revoked, revN2 all pending
    var n1rev, n2pend int
    db.QueryRow(`SELECT COUNT(*) FROM document_distributions WHERE doc_version_id=$1 AND revoked_at IS NOT NULL`, revN1).Scan(&n1rev)
    db.QueryRow(`SELECT COUNT(*) FROM document_distributions WHERE doc_version_id=$1 AND revoked_at IS NULL AND acked_at IS NULL`, revN2).Scan(&n2pend)
    if n1rev != 5 { t.Fatalf("revN1 revoked want 5, got %d", n1rev) }
    if n2pend != 5 { t.Fatalf("revN2 pending want 5, got %d", n2pend) }
}
```

- [ ] **Step 2: Run → PASS**

- [ ] **Step 3: Commit**

```bash
rtk git add internal/modules/distribution/application/fanout_worker_test.go
rtk git commit -m "test(spec4/phase3): advisory lock serializes rev-N/rev-N+1"
```

---

### Task 3.7 — Backoff matrix + DLQ tests

**Model:** Codex `gpt-5.3-codex` **medium**.

**Goal:** Simulate RBAC resolution failure 5× → verify backoff intervals (1m/5m/15m/1h/4h) and DLQ transition at attempt 6.

**Files:** Add to `fanout_worker_test.go`.

**Acceptance Criteria:**
- [ ] Inject failing resolver; each Tick increments attempt_count + sets next_attempt_at per schedule.
- [ ] 6th failure → row present in `distribution_outbox_dlq` with `final_error` populated; original outbox row deleted.

**Steps:**

- [ ] **Step 1: Write test**

```go
func TestBackoffAndDLQ(t *testing.T) {
    ctx := context.Background()
    db := openTestDB(t)
    tenantID, _, rev := seedControlledDocAndVersion(t, db, "area-welding", "operational")

    outbox := repository.NewPgOutboxRepository(db)
    dist := repository.NewPgDistributionRepository(db)
    failingResolver := &alwaysFailResolver{}

    tx, _ := db.BeginTx(ctx, nil)
    outbox.Enqueue(ctx, tx, repository.OutboxRow{TenantID: tenantID, DocVersionID: rev, EventType: "publish"})
    tx.Commit()

    w := application.NewFanoutWorker(db, outbox, dist, failingResolver)

    wantIntervals := []time.Duration{1 * time.Minute, 5 * time.Minute, 15 * time.Minute, 1 * time.Hour, 4 * time.Hour}
    for i, want := range wantIntervals {
        // Fast-forward next_attempt_at for test
        db.Exec(`UPDATE distribution_outbox SET next_attempt_at=NOW() WHERE doc_version_id=$1`, rev)

        if _, err := w.Tick(ctx); err != nil { t.Fatalf("attempt %d: %v", i+1, err) }

        var ac int
        var nxt, now time.Time
        db.QueryRow(`SELECT attempt_count, next_attempt_at, NOW() FROM distribution_outbox WHERE doc_version_id=$1`, rev).Scan(&ac, &nxt, &now)
        if ac != i+1 { t.Fatalf("attempt_count want %d got %d", i+1, ac) }
        diff := nxt.Sub(now)
        if abs(diff-want) > 30*time.Second {
            t.Fatalf("attempt %d: backoff want %v, got %v", i+1, want, diff)
        }
    }

    // 6th failure → DLQ
    db.Exec(`UPDATE distribution_outbox SET next_attempt_at=NOW() WHERE doc_version_id=$1`, rev)
    w.Tick(ctx)

    var origCount, dlqCount int
    db.QueryRow(`SELECT COUNT(*) FROM distribution_outbox WHERE doc_version_id=$1`, rev).Scan(&origCount)
    db.QueryRow(`SELECT COUNT(*) FROM distribution_outbox_dlq WHERE doc_version_id=$1`, rev).Scan(&dlqCount)
    if origCount != 0 || dlqCount != 1 {
        t.Fatalf("after 6 fails: orig=%d dlq=%d, want 0/1", origCount, dlqCount)
    }
}

func abs(d time.Duration) time.Duration { if d < 0 { return -d }; return d }
```

- [ ] **Step 2: Run → PASS**

- [ ] **Step 3: Commit**

```bash
rtk git add internal/modules/distribution/application/fanout_worker_test.go
rtk git commit -m "test(spec4/phase3): backoff schedule + DLQ transition"
```

---

### Phase 3 closeout

No Opus review scheduled mid-phase. Full Opus review happens after Phase 4.

- [ ] Run `rtk go test ./internal/modules/distribution/... -count=1` green.
- [ ] Inspect test duration — if `TestBackoffAndDLQ` exceeds 30s, mark with `-short` skip.

---

## Phase 4: Publish Hook (extends Spec 2)

**Goal:** inside the same tx that flips `approved → published` (and `approved → scheduled` / `scheduled → published` via CutoverJob), enqueue `distribution_outbox` rows. Passive `distribution_mode` skips enqueue. Thread `prior_version_id` (previous rev) so FanoutWorker can supersede.

**Reuse rule (/simplify):** do **NOT** introduce a new hook layer. Extend `PublishService` by injecting one dependency (`DistributionEnqueuer`) and calling it before `tx.Commit()`. Do not refactor Spec 2 tests beyond what the new dep requires.

**Codex model:** `gpt-5.3-codex --effort high` — ARCHITECTURE-sensitive (cross-module contract).
**Codex review mode:** ARCHITECTURE.
**Opus review:** YES, after Phase 4.

**Files:**
- Modify: `internal/modules/documents_v2/approval/application/publish_service.go` (+20 lines; inject `Distributor` iface; call `.EnqueueFanout(ctx, tx, …)` before `tx.Commit()` in both `PublishApproved` and `SchedulePublish` paths — scheduled path enqueues too so CutoverJob has no enqueue work later; see contract)
- Create: `internal/modules/distribution/application/publish_hook.go` (`DistributionEnqueuer` implementation that resolves `distribution_mode`, loads `prior_version_id`, calls `OutboxRepo.Enqueue`)
- Create: `internal/modules/distribution/application/publish_hook_test.go`
- Modify: `internal/modules/documents_v2/approval/application/publish_service_test.go` (only to inject stub `Distributor`; existing assertions unchanged)
- Modify: `apps/api/cmd/metaldocs-api/main.go` (wire real `DistributionEnqueuer` into `PublishService`)

**Acceptance Criteria:**
- [ ] `PublishApproved` enqueues one `distribution_outbox` row per published rev when `distribution_mode='active'`
- [ ] `PublishApproved` enqueues zero rows when `distribution_mode='passive'`
- [ ] Enqueue uses SAME `tx` as the document UPDATE (rollback semantics preserved)
- [ ] `prior_version_id` = id of last non-obsolete rev before this publish; `NULL` on first publish
- [ ] `SchedulePublish` enqueues an outbox row with `scheduled_for=effective_date`; CutoverJob then only flips status
- [ ] Existing Spec 2 publish tests still pass unmodified (only stub wiring added)

**Verify:** `rtk go test ./internal/modules/documents_v2/approval/... ./internal/modules/distribution/... -count=1`

---

### Task 4.1: Define Distributor interface + stub in Spec 2

**Files:** modify `publish_service.go`, add stub in `publish_service_test.go`.

- [ ] **Step 1: Add interface at top of publish_service.go**

```go
// Distributor is the upstream contract for distribution fan-out.
// Extended Spec 4 — publish hook enqueues one outbox row per active publish.
type Distributor interface {
    EnqueueFanout(ctx context.Context, tx *sql.Tx, in EnqueueFanoutInput) error
}

// EnqueueFanoutInput is the minimal data the distribution module needs
// from a publish. prior_version_id is NULL on first publish.
type EnqueueFanoutInput struct {
    TenantID            string
    ControlledDocumentID string
    DocVersionID         string  // the rev just published / scheduled
    PriorVersionID       *string // last non-obsolete rev before this one
    ScheduledFor         *time.Time // nil = ASAP; set = future cutover
}
```

- [ ] **Step 2: Add field to `PublishService` struct + constructor.** Insert `distributor Distributor` field. Update `NewPublishService` to take it as a required arg.

- [ ] **Step 3: Add call site in `PublishApproved`** — after step 4 (emit event), BEFORE step 5 (commit):

```go
// Step 4.5 (Spec 4): enqueue distribution outbox.
var prior *string
priorID, err := s.repo.LoadPriorNonObsoleteRev(ctx, tx, req.TenantID, instance.DocumentID, instance.RevisionVersion)
if err != nil && !errors.Is(err, sql.ErrNoRows) {
    _ = tx.Rollback()
    return PublishResult{}, fmt.Errorf("publishApproved: load prior rev: %w", err)
}
if priorID != "" { prior = &priorID }
if err := s.distributor.EnqueueFanout(ctx, tx, EnqueueFanoutInput{
    TenantID: req.TenantID, ControlledDocumentID: instance.DocumentID,
    DocVersionID: instance.ID, PriorVersionID: prior, ScheduledFor: nil,
}); err != nil {
    _ = tx.Rollback()
    return PublishResult{}, fmt.Errorf("publishApproved: enqueue fanout: %w", err)
}
```

- [ ] **Step 4: Same pattern in `SchedulePublish`** — set `ScheduledFor: &req.EffectiveDate`.

- [ ] **Step 5: Add `LoadPriorNonObsoleteRev` to `ApprovalRepository`** in `repository/approval_repository.go`:

```go
func (r *sqlApprovalRepo) LoadPriorNonObsoleteRev(ctx context.Context, tx *sql.Tx, tenantID, docID string, currentRev int) (string, error) {
    var id string
    err := tx.QueryRowContext(ctx, `
        SELECT id FROM documents
         WHERE tenant_id=$1 AND id=$2 AND status <> 'obsolete' AND revision_version < $3
         ORDER BY revision_version DESC LIMIT 1`, tenantID, docID, currentRev).Scan(&id)
    if errors.Is(err, sql.ErrNoRows) { return "", nil }
    return id, err
}
```

- [ ] **Step 6: Update Spec 2 tests** — only add stub `Distributor` into `NewPublishService(...)` calls. No assertion changes.

```go
type stubDistributor struct{ calls int }
func (s *stubDistributor) EnqueueFanout(ctx context.Context, tx *sql.Tx, in EnqueueFanoutInput) error {
    s.calls++; return nil
}
```

- [ ] **Step 7: Run Spec 2 tests → all green.**

```bash
rtk go test ./internal/modules/documents_v2/approval/... -count=1
```

- [ ] **Step 8: Commit**

```bash
rtk git add internal/modules/documents_v2/approval/application/ internal/modules/documents_v2/approval/repository/
rtk git commit -m "feat(spec4/phase4): distributor hook contract on publish_service"
```

---

### Task 4.2: Publish hook — active mode enqueues one row

**Files:** create `distribution/application/publish_hook.go` + `_test.go`.

- [ ] **Step 1: RED — test active mode enqueues exactly one row**

```go
// internal/modules/distribution/application/publish_hook_test.go
package application_test

// TestPublishHook_ActiveModeEnqueuesOne verifies that an active-mode document
// produces exactly one outbox row per publish call.
func TestPublishHook_ActiveModeEnqueuesOne(t *testing.T) {
    db := testhelper.DB(t)
    tenantID := testhelper.SeedTenant(t, db)
    docID, revID := testhelper.SeedControlledDoc(t, db, tenantID, "active")

    hook := application.NewPublishHook(repository.NewOutboxRepo(), repository.NewDocMetadataRepo())

    tx, _ := db.Begin()
    err := hook.EnqueueFanout(ctx, tx, approval.EnqueueFanoutInput{
        TenantID: tenantID, ControlledDocumentID: docID,
        DocVersionID: revID, PriorVersionID: nil, ScheduledFor: nil,
    })
    if err != nil { t.Fatal(err) }
    tx.Commit()

    var n int
    db.QueryRow(`SELECT COUNT(*) FROM distribution_outbox WHERE doc_version_id=$1`, revID).Scan(&n)
    if n != 1 { t.Fatalf("want 1 outbox row, got %d", n) }
}
```

- [ ] **Step 2: Implement `PublishHook`:**

```go
// internal/modules/distribution/application/publish_hook.go
package application

type PublishHook struct {
    outbox repository.OutboxRepository
    docs   repository.DocMetadataRepository
}
func NewPublishHook(o repository.OutboxRepository, d repository.DocMetadataRepository) *PublishHook {
    return &PublishHook{outbox: o, docs: d}
}

func (h *PublishHook) EnqueueFanout(ctx context.Context, tx *sql.Tx, in approval.EnqueueFanoutInput) error {
    meta, err := h.docs.LoadDistributionMetadata(ctx, tx, in.TenantID, in.ControlledDocumentID)
    if err != nil { return fmt.Errorf("publishHook: load dist meta: %w", err) }
    if meta.DistributionMode == "passive" { return nil } // /simplify: single branch, no strategy pattern
    now := time.Now().UTC()
    scheduled := now
    if in.ScheduledFor != nil { scheduled = in.ScheduledFor.UTC() }
    return h.outbox.Enqueue(ctx, tx, repository.OutboxRow{
        TenantID: in.TenantID, ControlledDocumentID: in.ControlledDocumentID,
        DocVersionID: in.DocVersionID, PriorVersionID: in.PriorVersionID,
        EnqueuedAt: now, NextAttemptAt: scheduled, AttemptCount: 0,
    })
}
```

- [ ] **Step 3: Run → PASS.**

- [ ] **Step 4: Commit**

```bash
rtk git add internal/modules/distribution/application/publish_hook.go internal/modules/distribution/application/publish_hook_test.go
rtk git commit -m "feat(spec4/phase4): publish hook enqueues outbox row in active mode"
```

---

### Task 4.3: Publish hook — passive mode skips enqueue

- [ ] **Step 1: RED**

```go
func TestPublishHook_PassiveModeSkips(t *testing.T) {
    db := testhelper.DB(t)
    tenantID := testhelper.SeedTenant(t, db)
    docID, revID := testhelper.SeedControlledDoc(t, db, tenantID, "passive")

    hook := application.NewPublishHook(repository.NewOutboxRepo(), repository.NewDocMetadataRepo())
    tx, _ := db.Begin()
    hook.EnqueueFanout(ctx, tx, approval.EnqueueFanoutInput{
        TenantID: tenantID, ControlledDocumentID: docID, DocVersionID: revID,
    })
    tx.Commit()

    var n int
    db.QueryRow(`SELECT COUNT(*) FROM distribution_outbox WHERE doc_version_id=$1`, revID).Scan(&n)
    if n != 0 { t.Fatalf("passive mode: want 0 outbox rows, got %d", n) }
}
```

- [ ] **Step 2: Run → PASS (already covered by branch in 4.2 impl).**

- [ ] **Step 3: Commit**

```bash
rtk git add internal/modules/distribution/application/publish_hook_test.go
rtk git commit -m "test(spec4/phase4): passive mode skips enqueue"
```

---

### Task 4.4: prior_version_id threading on supersession

- [ ] **Step 1: RED — rev 2 publish produces outbox row with prior_version_id = rev 1 id**

```go
func TestPublishHook_PriorVersionIdSetOnSupersession(t *testing.T) {
    db := testhelper.DB(t)
    tenantID := testhelper.SeedTenant(t, db)
    docID, rev1 := testhelper.SeedControlledDoc(t, db, tenantID, "active")
    testhelper.TransitionToPublished(t, db, rev1)
    rev2 := testhelper.SeedApprovedRev(t, db, tenantID, docID, /*revNo=*/2)

    publishSvc := approval.NewPublishService(repo, emitter, clock, application.NewPublishHook(outboxRepo, docRepo))
    publishSvc.PublishApproved(ctx, db, approval.PublishRequest{
        TenantID: tenantID, InstanceID: rev2InstanceID, PublishedBy: "user1",
    })

    var prior sql.NullString
    db.QueryRow(`SELECT prior_version_id FROM distribution_outbox WHERE doc_version_id=$1`, rev2).Scan(&prior)
    if !prior.Valid || prior.String != rev1 { t.Fatalf("want prior=%s, got %v", rev1, prior) }
}
```

- [ ] **Step 2: Run → PASS** (Task 4.1 already wires `LoadPriorNonObsoleteRev`).

- [ ] **Step 3: Commit**

```bash
rtk git add internal/modules/distribution/application/publish_hook_test.go
rtk git commit -m "test(spec4/phase4): prior_version_id threaded on supersession"
```

---

### Task 4.5: Wire real hook in main.go

- [ ] **Step 1: Modify `apps/api/cmd/metaldocs-api/main.go`** — where `PublishService` is constructed:

```go
publishHook := distapp.NewPublishHook(distrepo.NewOutboxRepo(), distrepo.NewDocMetadataRepo())
publishSvc := approval.NewPublishService(approvalRepo, eventEmitter, realClock, publishHook)
```

- [ ] **Step 2: Build → green.**

```bash
rtk go build ./apps/api/cmd/metaldocs-api
```

- [ ] **Step 3: Commit**

```bash
rtk git add apps/api/cmd/metaldocs-api/main.go
rtk git commit -m "wire(spec4/phase4): inject PublishHook into PublishService"
```

---

### Phase 4 closeout

- [ ] Dispatch Opus review (model=opus): "Review Phase 4 (Publish Hook extending Spec 2). Verify: (a) no changes to Spec 2 test assertions, (b) enqueue inside tx, (c) active/passive branch single-statement (/simplify), (d) scheduled-path enqueue contract preserves CutoverJob idempotency, (e) no new abstractions beyond `Distributor` iface."
- [ ] Codex ARCHITECTURE review via `task --background --model gpt-5.3-codex --effort high` — prompt pastes phase + contracts P4.
- [ ] Apply fixes, re-run Spec 2 + Spec 4 test suites.

---

## Phase 5: Membership Hook

**Goal:** on `Grant()`, enqueue scoped `membership_added` outbox rows for all currently-active published documents in the granted areas (so new members pick up past obligations). On `Revoke()`, synchronously UPDATE all non-acked obligations for the (user, area) pair with `revoke_reason='area_removed'`, `revoked_at=NOW()`.

**/simplify:** Extend existing `area_membership.go` functions with ONE additional DB call each. No new service object. Use existing transaction.

**Codex model:** `gpt-5.3-codex --effort medium`.
**Codex review mode:** COVERAGE.
**Opus review:** NO mid-phase; full review after Phase 6.

**Files:**
- Modify: `internal/modules/iam/area_membership/area_membership.go` (+30 lines)
- Modify: `internal/modules/iam/area_membership/area_membership_test.go`

**Acceptance Criteria:**
- [ ] `Grant(user, area)` enqueues one outbox row per currently-published doc in that area (event_type='membership_added'), scoped to that user only — via `scoped_recipient_user_id` column
- [ ] `Revoke(user, area)` UPDATEs non-acked obligations for (user, any doc in that area) setting `status='revoked'`, `revoke_reason='area_removed'`, `revoked_at=NOW()`
- [ ] `Revoke` is idempotent (re-calling produces zero additional rows changed)
- [ ] Already-acked obligations are not touched on revoke

**Verify:** `rtk go test ./internal/modules/iam/area_membership/... -count=1 -run Spec4`

---

### Task 5.1: Add scoped_recipient_user_id to outbox row + migration

Already covered by migration 0161 (distribution_outbox) in Phase 1. **No new migration.** Confirm column exists with `psql -c "\d distribution_outbox"`.

- [ ] **Step 1: Verify column present** — no file changes.
- [ ] **Step 2:** no commit.

---

### Task 5.2: Grant enqueues scoped outbox rows

- [ ] **Step 1: RED**

```go
// internal/modules/iam/area_membership/area_membership_test.go
func TestGrant_EnqueuesScopedOutbox_Spec4(t *testing.T) {
    db := testhelper.DB(t)
    tenantID := testhelper.SeedTenant(t, db)
    areaA := testhelper.SeedArea(t, db, tenantID, "A")
    // Seed 2 published active-mode docs in area A, 1 in area B.
    docA1 := testhelper.SeedPublishedDoc(t, db, tenantID, areaA, "active")
    docA2 := testhelper.SeedPublishedDoc(t, db, tenantID, areaA, "active")
    _     = testhelper.SeedPublishedDoc(t, db, tenantID, testhelper.SeedArea(t, db, tenantID, "B"), "active")

    if err := area_membership.Grant(ctx, db, tenantID, "user1", []string{areaA}); err != nil { t.Fatal(err) }

    rows, _ := db.Query(`
        SELECT doc_version_id, scoped_recipient_user_id
          FROM distribution_outbox
         WHERE event_type='membership_added' AND scoped_recipient_user_id=$1`, "user1")
    defer rows.Close()
    seen := map[string]bool{}
    for rows.Next() { var d, u string; rows.Scan(&d, &u); seen[d] = true }
    if !seen[docA1] || !seen[docA2] || len(seen) != 2 {
        t.Fatalf("want scoped rows for {%s,%s}, got %v", docA1, docA2, seen)
    }
}
```

- [ ] **Step 2: Extend `Grant`** — after existing INSERT into `area_memberships`, INSIDE same tx:

```go
_, err = tx.ExecContext(ctx, `
    INSERT INTO distribution_outbox
      (tenant_id, controlled_document_id, doc_version_id, event_type,
       scoped_recipient_user_id, enqueued_at, next_attempt_at, attempt_count)
    SELECT d.tenant_id, d.id, d.id, 'membership_added',
           $2, NOW(), NOW(), 0
      FROM documents d
     WHERE d.tenant_id=$1 AND d.process_area_code = ANY($3)
       AND d.status='published' AND d.distribution_mode='active'`,
    tenantID, userID, pq.Array(areaCodes))
if err != nil { return fmt.Errorf("grant: enqueue membership_added: %w", err) }
```

- [ ] **Step 3: Run → PASS.**

- [ ] **Step 4: Commit**

```bash
rtk git add internal/modules/iam/area_membership/
rtk git commit -m "feat(spec4/phase5): grant enqueues scoped membership_added outbox rows"
```

---

### Task 5.3: Revoke updates non-acked obligations

- [ ] **Step 1: RED**

```go
func TestRevoke_MarksNonAckedObligations_Spec4(t *testing.T) {
    db := testhelper.DB(t)
    tenantID := testhelper.SeedTenant(t, db)
    areaA := testhelper.SeedArea(t, db, tenantID, "A")
    doc := testhelper.SeedPublishedDoc(t, db, tenantID, areaA, "active")
    // user1: acked; user2: pending; user3: pending (also in area B — other grant)
    user1Ob := testhelper.SeedObligation(t, db, tenantID, doc, "user1", "view"); testhelper.Ack(t, db, user1Ob)
    user2Ob := testhelper.SeedObligation(t, db, tenantID, doc, "user2", "view")
    user3Ob := testhelper.SeedObligation(t, db, tenantID, doc, "user3", "view")

    area_membership.Grant(ctx, db, tenantID, "user2", []string{areaA})
    area_membership.Grant(ctx, db, tenantID, "user3", []string{areaA})

    if err := area_membership.Revoke(ctx, db, tenantID, "user2", []string{areaA}); err != nil { t.Fatal(err) }

    // user1 untouched (acked), user2 revoked, user3 still pending
    var s1, s2, s3 string
    db.QueryRow(`SELECT status FROM document_distributions WHERE id=$1`, user1Ob).Scan(&s1)
    db.QueryRow(`SELECT status FROM document_distributions WHERE id=$1`, user2Ob).Scan(&s2)
    db.QueryRow(`SELECT status FROM document_distributions WHERE id=$1`, user3Ob).Scan(&s3)
    if s1 != "acked" { t.Fatalf("user1 want acked, got %s", s1) }
    if s2 != "revoked" { t.Fatalf("user2 want revoked, got %s", s2) }
    if s3 != "pending" { t.Fatalf("user3 want pending, got %s", s3) }

    // idempotency: second revoke = 0 rows changed
    tag, _ := db.Exec(`-- nothing, just check second Revoke call is safe`)
    _ = tag
    if err := area_membership.Revoke(ctx, db, tenantID, "user2", []string{areaA}); err != nil { t.Fatal("idempotent revoke:", err) }
}
```

- [ ] **Step 2: Extend `Revoke`** — inside existing tx, after deleting from `area_memberships`:

```go
_, err = tx.ExecContext(ctx, `
    UPDATE document_distributions
       SET status='revoked', revoke_reason='area_removed', revoked_at=NOW()
     WHERE tenant_id=$1 AND recipient_user_id=$2 AND status='pending'
       AND controlled_document_id IN (
           SELECT id FROM documents
            WHERE tenant_id=$1 AND process_area_code = ANY($3)
       )`,
    tenantID, userID, pq.Array(areaCodes))
if err != nil { return fmt.Errorf("revoke: mark obligations: %w", err) }
```

- [ ] **Step 3: Run → PASS.**

- [ ] **Step 4: Commit**

```bash
rtk git add internal/modules/iam/area_membership/
rtk git commit -m "feat(spec4/phase5): revoke marks non-acked obligations area_removed"
```

---

### Phase 5 closeout

- [ ] Run `rtk go test ./internal/modules/iam/area_membership/... -count=1` green.
- [ ] No Opus review. Next Opus review at Phase 6 closeout.

---

## Phase 6: AckService + HTTP handlers

**Goal:** service layer for nonce issuance, view acks, and signature acks — including password re-auth and rate limiting. HTTP layer for `GET /documents/:id` (issues nonce) and `POST /documents/:id/ack`.

**/simplify:** one `AckService` struct, one method per ack variant. Rate limiter is the existing shared `RateLimiter` from `internal/shared/ratelimit/`. Password verify uses existing `auth.PasswordVerifier`. Do not invent a new audit subsystem — reuse `GovernanceEventEmitter` from Spec 2.

**Codex model:** `gpt-5.3-codex --effort high` (crypto + rate limit + HTTP).
**Codex review mode:** QUALITY.
**Opus review:** YES, after Phase 6.

**Files:**
- Create: `internal/modules/distribution/application/ack_service.go`
- Create: `internal/modules/distribution/application/ack_service_test.go`
- Create: `internal/modules/distribution/delivery/http/ack_handler.go`
- Create: `internal/modules/distribution/delivery/http/ack_handler_test.go`
- Create: `internal/modules/distribution/repository/nonce_repo.go` (Redis-free: Postgres `distribution_nonces` scratch table — added in migration 0167 addendum if not present; if present, skip)
- Modify: `apps/api/cmd/metaldocs-api/main.go` (routes)

**Acceptance Criteria:**
- [ ] `IssueNonce(obligationID)` → base64url nonce, 15min TTL, stored with `obligation_id`
- [ ] `RecordViewAck(nonce, userID)` → verifies nonce, builds `SignatureInputs`, computes HMAC, writes `document_distributions.acked_at + ack_signature`, consumes nonce (single-use)
- [ ] `RecordSignatureAck(nonce, userID, password)` → same + `auth.PasswordVerifier.Verify(userID, password)` MUST pass before ack
- [ ] Rate limit: 5 failed attempts / 15min / (user_id, obligation_id) → `ErrTooManyAttempts` (HTTP 429)
- [ ] Nonce reuse → `ErrNonceConsumed` (HTTP 409)
- [ ] Expired nonce → `ErrNonceExpired` (HTTP 410)
- [ ] `ack_signature` is persisted (64-char hex) and roundtrip-verifiable
- [ ] HTTP handlers wired; `GET /documents/:id` returns doc + nonce + ack_type

**Verify:** `rtk go test ./internal/modules/distribution/application/... ./internal/modules/distribution/delivery/... -count=1`

---

### Task 6.1: Nonce repository + issue nonce

- [ ] **Step 1: Confirm migration** — `distribution_nonces` table must exist (added to Phase 1 migration 0167). If missing, add it:

```sql
CREATE TABLE distribution_nonces (
    nonce             TEXT PRIMARY KEY,
    tenant_id         TEXT NOT NULL,
    obligation_id     UUID NOT NULL REFERENCES document_distributions(id) ON DELETE CASCADE,
    user_id           TEXT NOT NULL,
    issued_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at        TIMESTAMPTZ NOT NULL,
    consumed_at       TIMESTAMPTZ
);
CREATE INDEX idx_nonces_expires ON distribution_nonces(expires_at) WHERE consumed_at IS NULL;
```

(This is appended to Phase 1 migration 0167 if not yet present — double-check during Phase 1 self-review.)

- [ ] **Step 2: RED — TestIssueNonce persists + is base64url-32b**

```go
func TestIssueNonce_PersistsWithTTL(t *testing.T) {
    db := testhelper.DB(t)
    tenantID := testhelper.SeedTenant(t, db)
    ob := testhelper.SeedObligation(t, db, tenantID, "doc1", "user1", "view")

    svc := application.NewAckService(repository.NewNonceRepo(), distRepo, nil, secret, testhelper.FixedClock, rl)
    nonce, err := svc.IssueNonce(ctx, tenantID, "user1", ob)
    if err != nil { t.Fatal(err) }

    raw, _ := base64.RawURLEncoding.DecodeString(nonce)
    if len(raw) != 32 { t.Fatalf("want 32 bytes, got %d", len(raw)) }

    var expires time.Time
    db.QueryRow(`SELECT expires_at FROM distribution_nonces WHERE nonce=$1`, nonce).Scan(&expires)
    if expires.Sub(testhelper.FixedClock.Now()) != 15*time.Minute {
        t.Fatalf("want 15min TTL, got %v", expires.Sub(testhelper.FixedClock.Now()))
    }
}
```

- [ ] **Step 3: Implement NonceRepo.Issue / Consume.**

```go
// repository/nonce_repo.go
func (r *sqlNonceRepo) Issue(ctx context.Context, db *sql.DB, row NonceRow) error {
    _, err := db.ExecContext(ctx, `
        INSERT INTO distribution_nonces (nonce, tenant_id, obligation_id, user_id, issued_at, expires_at)
        VALUES ($1,$2,$3,$4,$5,$6)`,
        row.Nonce, row.TenantID, row.ObligationID, row.UserID, row.IssuedAt, row.ExpiresAt)
    return err
}
func (r *sqlNonceRepo) Consume(ctx context.Context, tx *sql.Tx, nonce string, now time.Time) (NonceRow, error) {
    var n NonceRow
    err := tx.QueryRowContext(ctx, `
        UPDATE distribution_nonces
           SET consumed_at = $2
         WHERE nonce = $1 AND consumed_at IS NULL AND expires_at > $2
        RETURNING nonce, tenant_id, obligation_id, user_id, issued_at, expires_at`,
        nonce, now).Scan(&n.Nonce, &n.TenantID, &n.ObligationID, &n.UserID, &n.IssuedAt, &n.ExpiresAt)
    if errors.Is(err, sql.ErrNoRows) {
        // distinguish expired vs consumed via second probe
        var consumed sql.NullTime; var exp time.Time
        r.db.QueryRowContext(ctx, `SELECT consumed_at, expires_at FROM distribution_nonces WHERE nonce=$1`, nonce).Scan(&consumed, &exp)
        if consumed.Valid { return n, ErrNonceConsumed }
        if !exp.IsZero() && exp.Before(now) { return n, ErrNonceExpired }
        return n, ErrNonceNotFound
    }
    return n, err
}
```

- [ ] **Step 4: Implement `AckService.IssueNonce`.**

```go
func (s *AckService) IssueNonce(ctx context.Context, tenantID, userID, obligationID string) (string, error) {
    buf := make([]byte, 32)
    if _, err := rand.Read(buf); err != nil { return "", fmt.Errorf("issueNonce: rand: %w", err) }
    nonce := base64.RawURLEncoding.EncodeToString(buf)
    now := s.clock.Now()
    err := s.nonces.Issue(ctx, s.db, repository.NonceRow{
        Nonce: nonce, TenantID: tenantID, ObligationID: obligationID, UserID: userID,
        IssuedAt: now, ExpiresAt: now.Add(15 * time.Minute),
    })
    if err != nil { return "", fmt.Errorf("issueNonce: persist: %w", err) }
    return nonce, nil
}
```

- [ ] **Step 5: Run → PASS.**

- [ ] **Step 6: Commit**

```bash
rtk git add internal/modules/distribution/repository/nonce_repo.go internal/modules/distribution/application/ack_service.go internal/modules/distribution/application/ack_service_test.go
rtk git commit -m "feat(spec4/phase6): nonce repo + IssueNonce (32b base64url, 15min TTL)"
```

---

### Task 6.2: RecordViewAck — happy path

- [ ] **Step 1: RED**

```go
func TestRecordViewAck_WritesAckedAtAndSignature(t *testing.T) {
    db := testhelper.DB(t)
    tenantID := testhelper.SeedTenant(t, db)
    ob := testhelper.SeedObligation(t, db, tenantID, "docV1", "user1", "view")
    svc := application.NewAckService(nonceRepo, distRepo, nil, []byte("secret"), clock, rl)

    nonce, _ := svc.IssueNonce(ctx, tenantID, "user1", ob)
    if err := svc.RecordViewAck(ctx, tenantID, "user1", nonce); err != nil { t.Fatal(err) }

    var acked sql.NullTime; var sig sql.NullString
    db.QueryRow(`SELECT acked_at, ack_signature FROM document_distributions WHERE id=$1`, ob).Scan(&acked, &sig)
    if !acked.Valid { t.Fatal("acked_at not set") }
    if !sig.Valid || len(sig.String) != 64 { t.Fatalf("bad signature: %v", sig) }
}
```

- [ ] **Step 2: Implement.** (Signature inputs pulled from repo via `LoadSignatureInputs`.)

```go
func (s *AckService) RecordViewAck(ctx context.Context, tenantID, userID, nonce string) error {
    tx, err := s.db.BeginTx(ctx, nil); if err != nil { return err }
    defer tx.Rollback()
    n, err := s.nonces.Consume(ctx, tx, nonce, s.clock.Now())
    if err != nil { return err }
    if n.UserID != userID || n.TenantID != tenantID { return ErrNonceUserMismatch }

    inputs, err := s.dists.LoadSignatureInputs(ctx, tx, n.ObligationID)
    if err != nil { return err }
    ackedAt := s.clock.Now().UTC()
    inputs.UserID = userID; inputs.Nonce = nonce; inputs.AckedAt = ackedAt
    sig := domain.ComputeAckSignature(s.secret, inputs)

    if err := s.dists.MarkAcked(ctx, tx, n.ObligationID, ackedAt, sig); err != nil { return err }
    return tx.Commit()
}
```

- [ ] **Step 3: Run → PASS.**

- [ ] **Step 4: Commit**

```bash
rtk git add internal/modules/distribution/application/
rtk git commit -m "feat(spec4/phase6): RecordViewAck writes acked_at + HMAC signature"
```

---

### Task 6.3: RecordSignatureAck requires password

- [ ] **Step 1: RED**

```go
func TestRecordSignatureAck_WrongPassword_Rejected(t *testing.T) {
    verifier := &stubVerifier{ok: false}
    svc := application.NewAckService(nonceRepo, distRepo, verifier, []byte("s"), clock, rl)
    nonce, _ := svc.IssueNonce(ctx, tid, "u1", obID)

    err := svc.RecordSignatureAck(ctx, tid, "u1", nonce, "wrong-password")
    if !errors.Is(err, application.ErrInvalidPassword) { t.Fatalf("want ErrInvalidPassword, got %v", err) }

    // nonce NOT consumed on password failure
    var consumed sql.NullTime
    db.QueryRow(`SELECT consumed_at FROM distribution_nonces WHERE nonce=$1`, nonce).Scan(&consumed)
    if consumed.Valid { t.Fatal("nonce consumed on failed password — should still be live") }
}

func TestRecordSignatureAck_CorrectPassword_Succeeds(t *testing.T) {
    verifier := &stubVerifier{ok: true}
    svc := application.NewAckService(nonceRepo, distRepo, verifier, []byte("s"), clock, rl)
    nonce, _ := svc.IssueNonce(ctx, tid, "u1", obID)
    if err := svc.RecordSignatureAck(ctx, tid, "u1", nonce, "correct"); err != nil { t.Fatal(err) }
}
```

- [ ] **Step 2: Implement.**

```go
func (s *AckService) RecordSignatureAck(ctx context.Context, tenantID, userID, nonce, password string) error {
    // Password verify BEFORE nonce consume — failed attempts do not burn nonces.
    ok, err := s.pw.Verify(ctx, userID, password)
    if err != nil { return fmt.Errorf("recordSignatureAck: verify: %w", err) }
    if !ok {
        s.rl.Record(userID + ":" + nonce) // count failed attempt
        return ErrInvalidPassword
    }
    return s.recordAckWithNonce(ctx, tenantID, userID, nonce) // shared w/ RecordViewAck
}
```

- [ ] **Step 3: Run → PASS.**

- [ ] **Step 4: Commit**

```bash
rtk git add internal/modules/distribution/application/
rtk git commit -m "feat(spec4/phase6): RecordSignatureAck requires password re-auth"
```

---

### Task 6.4: Rate limit (5 failed / 15min / (user,obligation))

- [ ] **Step 1: RED**

```go
func TestRecordSignatureAck_RateLimit_5FailsIn15Min(t *testing.T) {
    verifier := &stubVerifier{ok: false}
    rl := ratelimit.NewWindow(5, 15*time.Minute)
    svc := application.NewAckService(nonceRepo, distRepo, verifier, []byte("s"), clock, rl)

    for i := 0; i < 5; i++ {
        n, _ := svc.IssueNonce(ctx, tid, "u1", obID)
        svc.RecordSignatureAck(ctx, tid, "u1", n, "wrong") // returns ErrInvalidPassword
    }
    n, _ := svc.IssueNonce(ctx, tid, "u1", obID)
    err := svc.RecordSignatureAck(ctx, tid, "u1", n, "wrong")
    if !errors.Is(err, application.ErrTooManyAttempts) { t.Fatalf("want ErrTooManyAttempts, got %v", err) }
}
```

- [ ] **Step 2: Implement.** Add check BEFORE password verify:

```go
if s.rl.Exceeded(userID + ":" + nonce /* stable key */) {
    return ErrTooManyAttempts
}
```

Key caveat: the rate-limit key binds `(userID, obligationID)` — resolve `obligationID` from the nonce BEFORE password verify, without consuming. Add `NonceRepo.Peek` (read-only, respects expiry but does not update `consumed_at`).

```go
// NonceRepo.Peek — read-only fetch
func (r *sqlNonceRepo) Peek(ctx context.Context, db *sql.DB, nonce string, now time.Time) (NonceRow, error) {
    var n NonceRow
    err := db.QueryRowContext(ctx, `
        SELECT nonce, tenant_id, obligation_id, user_id, issued_at, expires_at
          FROM distribution_nonces WHERE nonce=$1 AND consumed_at IS NULL AND expires_at>$2`,
        nonce, now).Scan(&n.Nonce, &n.TenantID, &n.ObligationID, &n.UserID, &n.IssuedAt, &n.ExpiresAt)
    return n, err
}
```

Service:

```go
peek, err := s.nonces.Peek(ctx, s.db, nonce, s.clock.Now()); if err != nil { return err }
key := userID + ":" + peek.ObligationID
if s.rl.Exceeded(key) { return ErrTooManyAttempts }
// ... then password verify, then consume
```

- [ ] **Step 3: Run → PASS.**

- [ ] **Step 4: Commit**

```bash
rtk git add internal/modules/distribution/repository/nonce_repo.go internal/modules/distribution/application/
rtk git commit -m "feat(spec4/phase6): rate limit 5 failed signature attempts / 15min / (user,obligation)"
```

---

### Task 6.5: HTTP handlers

- [ ] **Step 1: Implement routes.**

```go
// delivery/http/ack_handler.go
func (h *AckHandler) GetDocument(w http.ResponseWriter, r *http.Request) {
    docID := chi.URLParam(r, "id"); userID := authctx.UserID(r.Context()); tenantID := authctx.TenantID(r.Context())
    ob, err := h.svc.LoadObligationForUser(r.Context(), tenantID, userID, docID)
    if err != nil { writeErr(w, err); return }
    nonce, err := h.svc.IssueNonce(r.Context(), tenantID, userID, ob.ID)
    if err != nil { writeErr(w, err); return }
    writeJSON(w, http.StatusOK, map[string]any{
        "document_id": docID, "ack_type": ob.AckType, "nonce": nonce, "expires_at": time.Now().UTC().Add(15*time.Minute),
    })
}

func (h *AckHandler) PostAck(w http.ResponseWriter, r *http.Request) {
    var body struct { Nonce, Password string }
    if err := json.NewDecoder(r.Body).Decode(&body); err != nil { http.Error(w, err.Error(), 400); return }
    userID := authctx.UserID(r.Context()); tenantID := authctx.TenantID(r.Context())
    if body.Password == "" {
        err := h.svc.RecordViewAck(r.Context(), tenantID, userID, body.Nonce); writeAckResult(w, err); return
    }
    err := h.svc.RecordSignatureAck(r.Context(), tenantID, userID, body.Nonce, body.Password); writeAckResult(w, err)
}

func writeAckResult(w http.ResponseWriter, err error) {
    switch {
    case err == nil: w.WriteHeader(http.StatusNoContent)
    case errors.Is(err, application.ErrInvalidPassword): http.Error(w, "invalid password", 401)
    case errors.Is(err, application.ErrTooManyAttempts): http.Error(w, "too many attempts", 429)
    case errors.Is(err, repository.ErrNonceConsumed): http.Error(w, "nonce consumed", 409)
    case errors.Is(err, repository.ErrNonceExpired): http.Error(w, "nonce expired", 410)
    default: http.Error(w, "internal", 500)
    }
}
```

- [ ] **Step 2: Handler test — full happy path + each error code.**

```go
func TestAckHandler_InvalidPassword_Returns401(t *testing.T) { /* ... */ }
func TestAckHandler_TooManyAttempts_Returns429(t *testing.T) { /* ... */ }
func TestAckHandler_NonceConsumed_Returns409(t *testing.T) { /* ... */ }
func TestAckHandler_NonceExpired_Returns410(t *testing.T) { /* ... */ }
func TestAckHandler_GET_ReturnsNonceAndAckType(t *testing.T) { /* ... */ }
func TestAckHandler_POST_ViewAck_ReturnsNoContent(t *testing.T) { /* ... */ }
```

- [ ] **Step 3: Wire routes in main.go.**

```go
r.Get("/documents/{id}", ackHandler.GetDocument)
r.Post("/documents/{id}/ack", ackHandler.PostAck)
```

- [ ] **Step 4: Run → PASS.**

- [ ] **Step 5: Commit**

```bash
rtk git add internal/modules/distribution/delivery/http/ apps/api/cmd/metaldocs-api/main.go
rtk git commit -m "feat(spec4/phase6): ack HTTP handlers — GET /documents/:id + POST /ack"
```

---

### Phase 6 closeout

- [ ] Opus review (model=opus): "Review Phase 4–6. Verify: (a) Spec 2 publish tests unchanged, (b) Grant/Revoke modifications contained to area_membership.go, (c) AckService crypto — rand.Read + base64.RawURLEncoding; HMAC via hmac.Equal; constant-time password check delegated to PasswordVerifier; (d) rate limit keyed correctly; (e) nonce semantics (consume iff password OK on signature path); (f) HTTP error code mapping complete."
- [ ] Codex QUALITY review — same JSON-verdict protocol.
- [ ] Apply fixes.

---

## Phase 7: Export Service watermark (extends Spec 3)

**Goal:** extend existing export path to (a) insert one `document_exports` ledger row per PDF export, (b) inject `UNCONTROLLED COPY — {email} — {iso-ts} — rev {n}` watermark via docgen-v2 pipeline, (c) compute `watermark_hash = sha256(watermark_text || first 4KB of pdf bytes)` and back-fill on the ledger row. **503 on Gotenberg failure with no ledger row.** **500 on watermark injection failure with rolled-back ledger.**

**/simplify:** extend Spec 3 `export_service.go` only where needed — single new method `ExportPdfWithWatermark`. Call existing docgen-v2 HTTP client; add one new docgen endpoint query param `?watermark=<urlenc>`. No new render abstraction.

**Codex model:** `gpt-5.3-codex --effort high` (PDF pipeline + rollback semantics).
**Codex review mode:** ARCHITECTURE.
**Opus review:** YES, after Phase 7.

**Files:**
- Modify: `internal/modules/documents_v2/application/export_service.go`
- Modify: `internal/modules/documents_v2/application/export_service_test.go`
- Modify: `apps/docgen-v2/src/routes/convert-pdf.ts` (read `watermark` query, render footer overlay)
- Create: `internal/modules/distribution/repository/exports_repo.go` (ExportsRepository.Insert, BackfillWatermarkHash)
- Modify: `internal/modules/documents_v2/delivery/http/export_handler.go` (503 on gotenberg fail)

**Acceptance Criteria:**
- [ ] Successful export produces one `document_exports` row with `watermark_text`, `watermark_hash`, `exported_by`, `exported_at`
- [ ] `watermark_text` exactly matches `UNCONTROLLED COPY — {email} — {RFC3339Nano-UTC ts} — rev {n}`
- [ ] `watermark_hash = sha256(watermark_text_bytes || pdf_bytes[:4096])` hex
- [ ] Gotenberg failure → HTTP 503 + no ledger row
- [ ] Watermark inject failure (docgen 500) → HTTP 500 + ledger row rolled back
- [ ] Ledger row includes Spec 3 content_hash (reused from render pipeline)

**Verify:** `rtk go test ./internal/modules/documents_v2/application/... -count=1 -run Export`

---

### Task 7.1: ExportsRepository

- [ ] **Step 1: Red**

```go
func TestExportsRepo_InsertAndBackfill(t *testing.T) {
    db := testhelper.DB(t)
    repo := repository.NewExportsRepo()
    id, err := repo.Insert(ctx, db, repository.ExportRow{
        TenantID: "t1", ControlledDocumentID: "d1", DocVersionID: "v1",
        ExportedBy: "u1", ExportedAt: time.Now(), WatermarkText: "UNCONTROLLED — u1@co — ...",
        ContentHash: "abc123",
    })
    if err != nil || id == "" { t.Fatal(err) }

    if err := repo.BackfillWatermarkHash(ctx, db, id, "sha256-hex"); err != nil { t.Fatal(err) }
    var h string
    db.QueryRow(`SELECT watermark_hash FROM document_exports WHERE id=$1`, id).Scan(&h)
    if h != "sha256-hex" { t.Fatalf("hash not backfilled: %s", h) }
}
```

- [ ] **Step 2: Implement repo.**

```go
func (r *sqlExportsRepo) Insert(ctx context.Context, db *sql.DB, row ExportRow) (string, error) {
    var id string
    err := db.QueryRowContext(ctx, `
        INSERT INTO document_exports
          (tenant_id, controlled_document_id, doc_version_id, exported_by, exported_at,
           watermark_text, content_hash)
        VALUES ($1,$2,$3,$4,$5,$6,$7)
        RETURNING id`,
        row.TenantID, row.ControlledDocumentID, row.DocVersionID, row.ExportedBy,
        row.ExportedAt, row.WatermarkText, row.ContentHash).Scan(&id)
    return id, err
}

func (r *sqlExportsRepo) BackfillWatermarkHash(ctx context.Context, db *sql.DB, id, hash string) error {
    _, err := db.ExecContext(ctx, `UPDATE document_exports SET watermark_hash=$2 WHERE id=$1`, id, hash)
    return err
}

func (r *sqlExportsRepo) Delete(ctx context.Context, db *sql.DB, id string) error {
    _, err := db.ExecContext(ctx, `DELETE FROM document_exports WHERE id=$1`, id)
    return err
}
```

- [ ] **Step 3: Run → PASS. Commit.**

```bash
rtk git add internal/modules/distribution/repository/exports_repo.go internal/modules/distribution/repository/exports_repo_test.go
rtk git commit -m "feat(spec4/phase7): ExportsRepository (Insert, BackfillWatermarkHash, Delete)"
```

---

### Task 7.2: ExportPdfWithWatermark — happy path

- [ ] **Step 1: RED**

```go
func TestExportPdfWithWatermark_InsertsLedgerAndInjects(t *testing.T) {
    db := testhelper.DB(t)
    docgen := testhelper.StubDocgen(t, testhelper.DocgenStubOpts{WatermarkEcho: true})
    defer docgen.Close()

    svc := application.NewExportService(exportRepo, docgen.Client(), clock)
    pdf, meta, err := svc.ExportPdfWithWatermark(ctx, application.ExportRequest{
        TenantID: "t1", DocVersionID: "v1", ControlledDocumentID: "d1",
        UserID: "u1", UserEmail: "alice@co", RevisionVersion: 7,
    })
    if err != nil { t.Fatal(err) }
    if len(pdf) == 0 { t.Fatal("empty pdf") }

    expectedText := fmt.Sprintf("UNCONTROLLED COPY — %s — %s — rev %d", "alice@co", clock.Now().UTC().Format(time.RFC3339Nano), 7)
    if meta.WatermarkText != expectedText {
        t.Fatalf("watermark text mismatch:\nwant: %q\ngot:  %q", expectedText, meta.WatermarkText)
    }

    h := sha256.Sum256(append([]byte(expectedText), pdf[:min(4096, len(pdf))]...))
    if meta.WatermarkHash != hex.EncodeToString(h[:]) {
        t.Fatal("watermark hash mismatch")
    }

    var ledgerHash string
    db.QueryRow(`SELECT watermark_hash FROM document_exports WHERE id=$1`, meta.ExportID).Scan(&ledgerHash)
    if ledgerHash != meta.WatermarkHash { t.Fatal("ledger hash not backfilled") }
}
```

- [ ] **Step 2: Implement.**

```go
func (s *ExportService) ExportPdfWithWatermark(ctx context.Context, req ExportRequest) ([]byte, ExportMeta, error) {
    now := s.clock.Now().UTC()
    watermark := fmt.Sprintf("UNCONTROLLED COPY — %s — %s — rev %d",
        req.UserEmail, now.Format(time.RFC3339Nano), req.RevisionVersion)

    // Step 1: insert ledger row (no hash yet).
    exportID, err := s.exports.Insert(ctx, s.db, repository.ExportRow{
        TenantID: req.TenantID, ControlledDocumentID: req.ControlledDocumentID,
        DocVersionID: req.DocVersionID, ExportedBy: req.UserID, ExportedAt: now,
        WatermarkText: watermark, ContentHash: req.ContentHash,
    })
    if err != nil { return nil, ExportMeta{}, fmt.Errorf("export: insert ledger: %w", err) }

    // Step 2: call docgen with watermark query.
    pdf, err := s.docgen.RenderPdfWithWatermark(ctx, req.DocVersionID, watermark)
    if err != nil {
        // Gotenberg-style outage → rollback ledger, surface 503.
        if errors.Is(err, docgen.ErrServiceUnavailable) {
            _ = s.exports.Delete(ctx, s.db, exportID)
            return nil, ExportMeta{}, ErrGotenbergUnavailable
        }
        // Watermark inject failure (docgen 500) → rollback ledger, surface 500.
        _ = s.exports.Delete(ctx, s.db, exportID)
        return nil, ExportMeta{}, fmt.Errorf("export: watermark inject: %w", err)
    }

    // Step 3: compute + backfill hash.
    cut := pdf
    if len(cut) > 4096 { cut = cut[:4096] }
    h := sha256.Sum256(append([]byte(watermark), cut...))
    hash := hex.EncodeToString(h[:])
    if err := s.exports.BackfillWatermarkHash(ctx, s.db, exportID, hash); err != nil {
        return nil, ExportMeta{}, fmt.Errorf("export: backfill hash: %w", err)
    }
    return pdf, ExportMeta{ExportID: exportID, WatermarkText: watermark, WatermarkHash: hash}, nil
}
```

- [ ] **Step 3: Run → PASS.**

- [ ] **Step 4: Commit**

```bash
rtk git add internal/modules/documents_v2/application/export_service.go internal/modules/documents_v2/application/export_service_test.go
rtk git commit -m "feat(spec4/phase7): ExportPdfWithWatermark — ledger insert + inject + hash backfill"
```

---

### Task 7.3: Docgen endpoint — inject watermark footer

- [ ] **Step 1: Modify `apps/docgen-v2/src/routes/convert-pdf.ts`** to read `watermark` query param and render a footer overlay on every page.

```ts
// convert-pdf.ts — add inside handler
const watermark = req.query.watermark ? String(req.query.watermark) : undefined;
if (watermark) {
  // Inject a footer DIV pre-conversion. Every page gets it via CSS @page footer.
  html = injectWatermarkFooter(html, watermark);
}

function injectWatermarkFooter(html: string, watermark: string): string {
  const css = `
  <style>
    @media print {
      body::after {
        content: ${JSON.stringify(watermark)};
        position: fixed; bottom: 8mm; left: 0; right: 0;
        text-align: center; font-size: 9pt; color: #B00020;
        font-family: Arial, sans-serif;
      }
    }
  </style>`;
  if (html.includes("</head>")) return html.replace("</head>", css + "</head>");
  return css + html;
}
```

- [ ] **Step 2: Add docgen-v2 test.**

```ts
// apps/docgen-v2/src/routes/convert-pdf.test.ts
test("watermark query param injects footer CSS", async () => {
  const res = await request(app).post("/convert-pdf?watermark=UNCONTROLLED").send({ html: "<html><head></head><body>doc</body></html>" });
  expect(res.status).toBe(200);
  // Grep the intermediate HTML used for conversion (stub wires out the processed html).
  expect(stubGotenberg.lastHtml).toContain("UNCONTROLLED");
});
```

- [ ] **Step 3: Run pnpm vitest → PASS. Commit.**

```bash
rtk git add apps/docgen-v2/src/routes/convert-pdf.ts apps/docgen-v2/src/routes/convert-pdf.test.ts
rtk git commit -m "feat(spec4/phase7): docgen-v2 accepts ?watermark= and injects CSS footer"
```

---

### Task 7.4: 503 on Gotenberg failure, 500 on watermark failure

- [ ] **Step 1: RED**

```go
func TestExport_Gotenberg503_NoLedgerRow(t *testing.T) {
    docgen := testhelper.StubDocgen(t, testhelper.DocgenStubOpts{GotenbergDown: true})
    svc := application.NewExportService(exportRepo, docgen.Client(), clock)

    _, _, err := svc.ExportPdfWithWatermark(ctx, req)
    if !errors.Is(err, application.ErrGotenbergUnavailable) { t.Fatal(err) }

    var n int
    db.QueryRow(`SELECT COUNT(*) FROM document_exports WHERE doc_version_id=$1`, req.DocVersionID).Scan(&n)
    if n != 0 { t.Fatalf("ledger should have no row, got %d", n) }
}

func TestExport_WatermarkInjectFail_RollsBackLedger(t *testing.T) {
    docgen := testhelper.StubDocgen(t, testhelper.DocgenStubOpts{WatermarkInjectFail: true})
    svc := application.NewExportService(exportRepo, docgen.Client(), clock)
    _, _, err := svc.ExportPdfWithWatermark(ctx, req)
    if err == nil || errors.Is(err, application.ErrGotenbergUnavailable) { t.Fatalf("want 500-class, got %v", err) }

    var n int
    db.QueryRow(`SELECT COUNT(*) FROM document_exports WHERE doc_version_id=$1`, req.DocVersionID).Scan(&n)
    if n != 0 { t.Fatalf("ledger should have rolled back, got %d", n) }
}
```

- [ ] **Step 2: Implement** — error classification already in Task 7.2; just confirm `docgen.Client` maps HTTP codes to `ErrServiceUnavailable` (503) vs generic error (500).

```go
// internal/modules/documents_v2/docgen/client.go (add)
func (c *Client) RenderPdfWithWatermark(ctx context.Context, docID, watermark string) ([]byte, error) {
    u := fmt.Sprintf("%s/convert-pdf?watermark=%s", c.baseURL, url.QueryEscape(watermark))
    resp, err := c.httpDo(ctx, "POST", u, /*body: docID->html fetched separately*/ nil)
    if err != nil { return nil, err }
    if resp.StatusCode == http.StatusServiceUnavailable { return nil, ErrServiceUnavailable }
    if resp.StatusCode >= 500 { return nil, fmt.Errorf("docgen: %d", resp.StatusCode) }
    return io.ReadAll(resp.Body)
}
```

- [ ] **Step 3: Run → PASS.**

- [ ] **Step 4: HTTP handler mapping** in `delivery/http/export_handler.go`:

```go
switch {
case errors.Is(err, application.ErrGotenbergUnavailable): http.Error(w, "gotenberg unavailable", 503)
case err != nil: http.Error(w, "export failed", 500)
}
```

- [ ] **Step 5: Commit**

```bash
rtk git add internal/modules/documents_v2/
rtk git commit -m "feat(spec4/phase7): 503 on gotenberg down, 500 w/ rollback on watermark fail"
```

---

### Phase 7 closeout

- [ ] Opus review (model=opus): "Review Phase 7. Verify: (a) rollback semantics — ledger deleted on every non-success path, (b) watermark text is deterministic (same inputs → same bytes), (c) hash input = watermark_bytes || pdf[:4096], (d) no Spec 3 render-pipeline refactor beyond adding `?watermark=` query support."
- [ ] Codex ARCHITECTURE review.
- [ ] Apply fixes.

---

## Phase 8: CutoverJob + DST resolver

**Goal:** scheduled-publish worker. Every 60s, pick outbox rows with `scheduled_for <= NOW()` that reference `scheduled` documents, flip them to `published` in a tx (advisory-locked on `controlled_document_id`), and mark the outbox row ready for immediate fan-out. DST ambiguity resolution runs at `ScheduledFor` parse time (earlier on fall-back, later on spring-forward skip).

**/simplify:** one job struct, one Tick method, one per-row flow. Advisory lock reused from Phase 3. No new scheduler abstraction — plug into existing `internal/modules/jobs/scheduler`.

**Codex model:** `gpt-5.3-codex --effort medium`.
**Codex review mode:** QUALITY.
**Opus review:** NO mid-phase; full review after Phase 9.

**Files:**
- Create: `internal/modules/distribution/application/cutover_job.go`
- Create: `internal/modules/distribution/application/cutover_job_test.go`
- Create: `internal/modules/distribution/domain/dst.go` (DST resolver — `ResolveLocalToUTC(local time.Time, loc *time.Location, policy DSTPolicy) (time.Time, error)`)
- Create: `internal/modules/distribution/domain/dst_test.go`
- Modify: `apps/worker/cmd/metaldocs-worker/main.go` (register `CutoverJob` with scheduler)

**Acceptance Criteria:**
- [ ] `CutoverJob.Tick()` finds outbox rows where `scheduled_for <= NOW()` AND doc status = `scheduled` AND `attempt_count = 0`
- [ ] For each row: advisory-lock on `controlled_document_id`, flip `documents.status='scheduled'→'published'` (OCC via `WHERE status='scheduled'`), `UPDATE distribution_outbox SET next_attempt_at = NOW()` so FanoutWorker picks it up next Tick
- [ ] If doc not in scheduled status (concurrent cutover won) → skip, leave row alone
- [ ] DST fall-back ambiguity → policy `earlier` picks first wall-clock instance; `later` picks second
- [ ] DST spring-forward gap → `earlier` picks pre-gap wall time; `later` picks post-gap
- [ ] DST policy default = `earlier` (ISO 9001: earlier effective = safer)

**Verify:** `rtk go test ./internal/modules/distribution/application/... ./internal/modules/distribution/domain/... -count=1 -run Cutover` and `-run DST`

---

### Task 8.1: DST resolver

- [ ] **Step 1: RED — table-driven DST test**

```go
// internal/modules/distribution/domain/dst_test.go
func TestResolveLocalToUTC(t *testing.T) {
    ny, _ := time.LoadLocation("America/New_York")
    // 2025-11-02 01:30 NY = ambiguous (fall back)
    // 2025-03-09 02:30 NY = gap (spring forward)
    cases := []struct{
        name string; localY, localMo, localD, localH, localMi int
        loc *time.Location; policy domain.DSTPolicy
        wantHourUTC int // hour component of UTC result (easier to assert)
    }{
        {"normal time earlier", 2025, 6, 15, 14, 0, ny, domain.DSTEarlier, 18},
        {"fall back ambiguous earlier (EDT)", 2025, 11, 2, 1, 30, ny, domain.DSTEarlier, 5},
        {"fall back ambiguous later (EST)",   2025, 11, 2, 1, 30, ny, domain.DSTLater,   6},
        {"spring gap earlier (pre EST→EDT)",  2025, 3, 9, 2, 30, ny, domain.DSTEarlier,  7}, // 02:30 EST = 07:30 UTC
        {"spring gap later  (post EST→EDT)",  2025, 3, 9, 2, 30, ny, domain.DSTLater,    6}, // 02:30 EDT = 06:30 UTC
    }
    for _, c := range cases {
        t.Run(c.name, func(t *testing.T) {
            local := time.Date(c.localY, time.Month(c.localMo), c.localD, c.localH, c.localMi, 0, 0, c.loc)
            got, err := domain.ResolveLocalToUTC(local, c.policy)
            if err != nil { t.Fatal(err) }
            if got.UTC().Hour() != c.wantHourUTC {
                t.Fatalf("%s: want UTC hour %d, got %d (%v)", c.name, c.wantHourUTC, got.UTC().Hour(), got.UTC())
            }
        })
    }
}
```

- [ ] **Step 2: Implement.**

```go
// internal/modules/distribution/domain/dst.go
package domain

type DSTPolicy string
const ( DSTEarlier DSTPolicy = "earlier"; DSTLater DSTPolicy = "later" )

// ResolveLocalToUTC resolves ambiguous/gap wall times per DSTPolicy.
// Ambiguous (fall-back): two valid UTC instants for same wall time.
// Gap (spring-forward): wall time is skipped; we pick before/after gap.
func ResolveLocalToUTC(local time.Time, policy DSTPolicy) (time.Time, error) {
    loc := local.Location()
    // Strip sec/nsec for deterministic DST probe.
    wall := time.Date(local.Year(), local.Month(), local.Day(), local.Hour(), local.Minute(), local.Second(), local.Nanosecond(), loc)
    before := wall.Add(-2 * time.Hour)
    after  := wall.Add( 2 * time.Hour)

    _, offBefore := before.Zone()
    _, offAfter  := after.Zone()

    if offBefore == offAfter {
        // No transition nearby — unambiguous.
        return wall.UTC(), nil
    }

    // Transition in the 4h window around wall. Construct both candidate UTC instants.
    cand1 := time.Date(wall.Year(), wall.Month(), wall.Day(), wall.Hour(), wall.Minute(), 0, 0, time.FixedZone("c1", offBefore)).UTC()
    cand2 := time.Date(wall.Year(), wall.Month(), wall.Day(), wall.Hour(), wall.Minute(), 0, 0, time.FixedZone("c2", offAfter)).UTC()

    if policy == DSTEarlier {
        if cand1.Before(cand2) { return cand1, nil }
        return cand2, nil
    }
    if cand1.After(cand2) { return cand1, nil }
    return cand2, nil
}
```

- [ ] **Step 3: Run → PASS.**

- [ ] **Step 4: Commit**

```bash
rtk git add internal/modules/distribution/domain/dst.go internal/modules/distribution/domain/dst_test.go
rtk git commit -m "feat(spec4/phase8): DST resolver with earlier/later policy"
```

---

### Task 8.2: CutoverJob.Tick — flips scheduled→published

- [ ] **Step 1: RED**

```go
func TestCutoverJob_FlipsScheduledToPublished(t *testing.T) {
    db := testhelper.DB(t)
    tenantID := testhelper.SeedTenant(t, db)
    docID, revID := testhelper.SeedControlledDoc(t, db, tenantID, "active")
    testhelper.TransitionToScheduled(t, db, revID, time.Now().Add(-1*time.Minute)) // scheduled_for in past

    job := application.NewCutoverJob(db, docRepo, outboxRepo, clock)
    n, err := job.Tick(ctx)
    if err != nil { t.Fatal(err) }
    if n != 1 { t.Fatalf("want 1 flipped, got %d", n) }

    var st string
    db.QueryRow(`SELECT status FROM documents WHERE id=$1`, revID).Scan(&st)
    if st != "published" { t.Fatalf("want published, got %s", st) }

    var nextAttempt time.Time
    db.QueryRow(`SELECT next_attempt_at FROM distribution_outbox WHERE doc_version_id=$1`, revID).Scan(&nextAttempt)
    if nextAttempt.After(time.Now()) { t.Fatalf("next_attempt_at should be ≤ now for immediate fan-out, got %v", nextAttempt) }
}
```

- [ ] **Step 2: Implement.**

```go
func (j *CutoverJob) Tick(ctx context.Context) (int, error) {
    rows, err := j.db.QueryContext(ctx, `
        SELECT o.id, o.tenant_id, o.controlled_document_id, o.doc_version_id
          FROM distribution_outbox o
          JOIN documents d ON d.id = o.doc_version_id
         WHERE o.scheduled_for <= NOW() AND o.attempt_count = 0
           AND d.status = 'scheduled'
         ORDER BY o.scheduled_for ASC
         LIMIT 50`)
    if err != nil { return 0, err }
    defer rows.Close()

    type row struct{ outboxID, tenantID, cdID, revID string }
    var batch []row
    for rows.Next() { var r row; rows.Scan(&r.outboxID, &r.tenantID, &r.cdID, &r.revID); batch = append(batch, r) }

    flipped := 0
    for _, r := range batch {
        tx, err := j.db.BeginTx(ctx, nil); if err != nil { return flipped, err }
        _, err = tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1::text, 0))`, r.cdID)
        if err != nil { tx.Rollback(); continue }

        res, err := tx.ExecContext(ctx, `
            UPDATE documents SET status='published', revision_version = revision_version + 1
             WHERE id=$1 AND tenant_id=$2 AND status='scheduled'`, r.revID, r.tenantID)
        if err != nil { tx.Rollback(); continue }
        n, _ := res.RowsAffected()
        if n == 0 { tx.Commit(); continue } // lost race — another cutover won

        _, err = tx.ExecContext(ctx, `UPDATE distribution_outbox SET next_attempt_at = NOW() WHERE id=$1`, r.outboxID)
        if err != nil { tx.Rollback(); continue }
        if err := tx.Commit(); err != nil { continue }
        flipped++
    }
    return flipped, nil
}
```

- [ ] **Step 3: Run → PASS.**

- [ ] **Step 4: Commit**

```bash
rtk git add internal/modules/distribution/application/cutover_job.go internal/modules/distribution/application/cutover_job_test.go
rtk git commit -m "feat(spec4/phase8): CutoverJob flips scheduled→published w/ advisory lock"
```

---

### Task 8.3: Wire CutoverJob into scheduler

- [ ] **Step 1: Modify `apps/worker/cmd/metaldocs-worker/main.go`:**

```go
cutoverJob := distapp.NewCutoverJob(db, docRepo, outboxRepo, realClock)
scheduler.Every(60*time.Second, "distribution.cutover", cutoverJob.Tick)
```

- [ ] **Step 2: Build → green; commit.**

```bash
rtk go build ./apps/worker/cmd/metaldocs-worker
rtk git add apps/worker/cmd/metaldocs-worker/main.go
rtk git commit -m "wire(spec4/phase8): register CutoverJob on scheduler (60s)"
```

---

### Phase 8 closeout

- [ ] Run `rtk go test ./internal/modules/distribution/... -count=1` green.
- [ ] Codex QUALITY review — prompt: "Verify CutoverJob concurrency: advisory lock granularity, OCC guard catches races, LIMIT 50 per tick is sufficient, scheduled→published triggers legal-transition check from migration 0166."

---

## Phase 9: ReconciliationJob + observability

**Goal:** nightly job reconciles (membership × published-active-docs) vs `document_distributions` — INSERT any missing obligations (backfill), flag any orphaned obligations (recipient no longer in area), write one `reconciliation_run_summary` row. Emit Prometheus metrics for outbox depth, DLQ depth, ack latency p50/p95/p99.

**/simplify:** one SQL `INSERT … SELECT … ON CONFLICT DO NOTHING` covers backfill. Orphan detection is one `UPDATE … WHERE NOT EXISTS`. Metrics: use existing `internal/shared/metrics/` prom registry.

**Codex model:** `gpt-5.3-codex --effort medium`.
**Codex review mode:** OPERATIONS.
**Opus review:** YES, after Phase 9.

**Files:**
- Create: `internal/modules/distribution/application/reconciliation_job.go`
- Create: `internal/modules/distribution/application/reconciliation_job_test.go`
- Create: `internal/modules/distribution/observability/metrics.go`
- Modify: `apps/worker/cmd/metaldocs-worker/main.go` (register nightly + metrics server)

**Acceptance Criteria:**
- [ ] `ReconciliationJob.Tick()` inserts missing obligations where (user ∈ area) ∧ (doc published active) ∧ (no existing row)
- [ ] Inserted rows have `reconciliation_run_id = <this run>` for audit
- [ ] Orphans flagged: `status='revoked'`, `revoke_reason='reconciliation_orphan'`, `revoked_at=NOW()` where user no longer in area AND status='pending'
- [ ] Summary row `reconciliation_run_summary` records: `started_at`, `finished_at`, `backfilled_count`, `orphan_count`, `error_count`, `error_samples` (top 5 sql errors)
- [ ] Prometheus gauges: `distribution_outbox_depth`, `distribution_dlq_depth`
- [ ] Histograms: `distribution_ack_latency_seconds{ack_type="view|signature"}` p50/p95/p99

**Verify:** `rtk go test ./internal/modules/distribution/application/... -count=1 -run Reconciliation`

---

### Task 9.1: Backfill missing obligations

- [ ] **Step 1: RED**

```go
func TestReconciliationJob_BackfillsMissingObligations(t *testing.T) {
    db := testhelper.DB(t)
    tenantID := testhelper.SeedTenant(t, db)
    areaA := testhelper.SeedArea(t, db, tenantID, "A")
    docX := testhelper.SeedPublishedDoc(t, db, tenantID, areaA, "active")
    testhelper.AddMembership(t, db, tenantID, "alice", areaA)
    testhelper.AddMembership(t, db, tenantID, "bob",   areaA)
    // Seed obligation for alice only — bob is missing.
    testhelper.SeedObligation(t, db, tenantID, docX, "alice", "view")

    job := application.NewReconciliationJob(db)
    res, err := job.Tick(ctx)
    if err != nil { t.Fatal(err) }
    if res.Backfilled != 1 { t.Fatalf("want 1 backfilled, got %d", res.Backfilled) }

    var n int
    db.QueryRow(`SELECT COUNT(*) FROM document_distributions WHERE recipient_user_id=$1 AND controlled_document_id=$2`, "bob", docX).Scan(&n)
    if n != 1 { t.Fatalf("bob obligation not backfilled") }
}
```

- [ ] **Step 2: Implement.**

```go
func (j *ReconciliationJob) Tick(ctx context.Context) (RunResult, error) {
    started := time.Now().UTC()

    runID := uuid.NewString()
    tx, err := j.db.BeginTx(ctx, nil); if err != nil { return RunResult{}, err }
    defer tx.Rollback()

    // Backfill.
    res, err := tx.ExecContext(ctx, `
        INSERT INTO document_distributions
          (id, tenant_id, controlled_document_id, recipient_user_id, ack_type, status, obliged_at, reconciliation_run_id)
        SELECT uuid_generate_v4(), d.tenant_id, d.id, m.user_id,
               COALESCE(d.ack_type, pa.default_ack_type, t.default_ack_type, 'view'),
               'pending', NOW(), $1
          FROM documents d
          JOIN area_memberships m ON m.process_area_code = d.process_area_code AND m.tenant_id = d.tenant_id
          JOIN process_areas pa ON pa.code = d.process_area_code AND pa.tenant_id = d.tenant_id
          JOIN metaldocs.tenants t ON t.id = d.tenant_id
         WHERE d.status = 'published' AND d.distribution_mode = 'active'
           AND NOT EXISTS (
               SELECT 1 FROM document_distributions dd
                WHERE dd.controlled_document_id = d.id AND dd.recipient_user_id = m.user_id
           )`, runID)
    if err != nil { return RunResult{}, err }
    backfilled, _ := res.RowsAffected()

    // Orphans.
    res2, err := tx.ExecContext(ctx, `
        UPDATE document_distributions dd
           SET status='revoked', revoke_reason='reconciliation_orphan', revoked_at=NOW()
         WHERE dd.status='pending'
           AND NOT EXISTS (
               SELECT 1 FROM area_memberships m
                JOIN documents d ON d.process_area_code = m.process_area_code AND d.tenant_id = m.tenant_id
                WHERE m.user_id = dd.recipient_user_id AND d.id = dd.controlled_document_id
           )`)
    if err != nil { return RunResult{}, err }
    orphans, _ := res2.RowsAffected()

    _, err = tx.ExecContext(ctx, `
        INSERT INTO reconciliation_run_summary
          (id, started_at, finished_at, backfilled_count, orphan_count, error_count)
        VALUES ($1, $2, NOW(), $3, $4, 0)`, runID, started, backfilled, orphans)
    if err != nil { return RunResult{}, err }

    if err := tx.Commit(); err != nil { return RunResult{}, err }
    return RunResult{RunID: runID, Backfilled: int(backfilled), Orphans: int(orphans)}, nil
}
```

- [ ] **Step 3: Run → PASS.**

- [ ] **Step 4: Commit**

```bash
rtk git add internal/modules/distribution/application/reconciliation_job.go internal/modules/distribution/application/reconciliation_job_test.go
rtk git commit -m "feat(spec4/phase9): reconciliation backfills missing obligations"
```

---

### Task 9.2: Orphan detection

- [ ] **Step 1: RED**

```go
func TestReconciliationJob_FlagsOrphans(t *testing.T) {
    db := testhelper.DB(t)
    tenantID := testhelper.SeedTenant(t, db)
    areaA := testhelper.SeedArea(t, db, tenantID, "A")
    docX := testhelper.SeedPublishedDoc(t, db, tenantID, areaA, "active")
    // user alice HAS obligation but is NOT in area (orphan).
    ob := testhelper.SeedObligation(t, db, tenantID, docX, "alice", "view")

    res, err := application.NewReconciliationJob(db).Tick(ctx)
    if err != nil { t.Fatal(err) }
    if res.Orphans != 1 { t.Fatalf("want 1 orphan, got %d", res.Orphans) }

    var status, reason string
    db.QueryRow(`SELECT status, revoke_reason FROM document_distributions WHERE id=$1`, ob).Scan(&status, &reason)
    if status != "revoked" || reason != "reconciliation_orphan" {
        t.Fatalf("got status=%s reason=%s", status, reason)
    }
}
```

- [ ] **Step 2: Run → PASS** (Task 9.1 impl already covers).
- [ ] **Step 3: Commit**

```bash
rtk git add internal/modules/distribution/application/reconciliation_job_test.go
rtk git commit -m "test(spec4/phase9): orphan detection flags revoked+reconciliation_orphan"
```

---

### Task 9.3: Prometheus metrics

- [ ] **Step 1: Implement**

```go
// internal/modules/distribution/observability/metrics.go
var (
    OutboxDepth = prometheus.NewGauge(prometheus.GaugeOpts{
        Name: "distribution_outbox_depth", Help: "rows in distribution_outbox (non-DLQ)",
    })
    DLQDepth = prometheus.NewGauge(prometheus.GaugeOpts{
        Name: "distribution_dlq_depth", Help: "rows in distribution_outbox_dlq",
    })
    AckLatency = prometheus.NewHistogramVec(prometheus.HistogramOpts{
        Name: "distribution_ack_latency_seconds",
        Help: "time from publish to ack",
        Buckets: []float64{60, 300, 900, 3600, 14400, 86400, 604800}, // 1m..7d
    }, []string{"ack_type"})
)

func Register(r prometheus.Registerer) {
    r.MustRegister(OutboxDepth, DLQDepth, AckLatency)
}

// Poller — populates gauges from SQL every 15s.
func PollDepths(ctx context.Context, db *sql.DB, every time.Duration) {
    t := time.NewTicker(every); defer t.Stop()
    for { select {
    case <-ctx.Done(): return
    case <-t.C:
        var o, d int
        db.QueryRowContext(ctx, `SELECT COUNT(*) FROM distribution_outbox`).Scan(&o)
        db.QueryRowContext(ctx, `SELECT COUNT(*) FROM distribution_outbox_dlq`).Scan(&d)
        OutboxDepth.Set(float64(o)); DLQDepth.Set(float64(d))
    }}
}
```

- [ ] **Step 2: Hook `AckLatency` into AckService** — after successful ack, compute `time.Since(obliged_at)` and observe with ack_type label.

- [ ] **Step 3: Register + start poller in worker main.go.**

- [ ] **Step 4: Commit**

```bash
rtk git add internal/modules/distribution/observability/ internal/modules/distribution/application/ apps/worker/cmd/metaldocs-worker/main.go
rtk git commit -m "feat(spec4/phase9): prometheus metrics — outbox depth, DLQ depth, ack latency"
```

---

### Phase 9 closeout

- [ ] Opus review (model=opus): "Review Phase 8+9. Verify: (a) cutover job uses same advisory-lock key as FanoutWorker (no deadlock risk), (b) DST policy default=earlier is documented, (c) reconciliation backfill respects ack_type resolution chain (version → area → tenant → 'view'), (d) orphan query is correct SQL, (e) histograms have sensible buckets for ISO 9001 (minutes to days)."
- [ ] Codex OPERATIONS review.

---

## Phase 10: Frontend UI

**Goal:** 5 new screens + 1 extension. Inbox (ack queue for current user), AckModal (read + confirm / password re-auth), ManagerDashboard (per-area coverage), PublishDialog extension (ack_type picker + active/passive toggle), TenantSettings (default ack_type).

**/simplify:** use existing TanStack Query patterns from Spec 1/2. No new state library. No new form library — reuse existing RHF + Zod. One React component per screen, colocated `.test.tsx`.

**Codex model:** `gpt-5.3-codex --effort low` for UI scaffolding + TanStack calls; `haiku` for pure boilerplate CRUD forms. Switch to `sonnet` only if a component exceeds 200 lines.
**Codex review mode:** QUALITY.
**Opus review:** NO mid-phase; full review after Phase 10.

**Files:**
- Create: `frontend/apps/web/src/features/distribution/pages/InboxPage.tsx`
- Create: `frontend/apps/web/src/features/distribution/pages/InboxPage.test.tsx`
- Create: `frontend/apps/web/src/features/distribution/components/AckModal.tsx`
- Create: `frontend/apps/web/src/features/distribution/components/AckModal.test.tsx`
- Create: `frontend/apps/web/src/features/distribution/pages/ManagerDashboardPage.tsx`
- Create: `frontend/apps/web/src/features/distribution/pages/ManagerDashboardPage.test.tsx`
- Modify: `frontend/apps/web/src/features/approval/components/PublishDialog.tsx` (+ack_type picker + mode toggle)
- Create: `frontend/apps/web/src/features/tenant/pages/TenantSettingsPage.tsx`
- Create: `frontend/apps/web/src/features/tenant/pages/TenantSettingsPage.test.tsx`
- Modify: `frontend/apps/web/src/app/routes.tsx` (add `/inbox`, `/distribution/manager`, `/settings/tenant`)

**Acceptance Criteria:**
- [ ] Inbox lists open obligations for current user sorted by `obliged_at DESC`, shows title, area, ack_type badge, status
- [ ] Clicking obligation opens AckModal with document body + "Confirm Read" button
- [ ] If ack_type=signature, AckModal shows password field; submit blocks until filled
- [ ] 429/409/410 responses render distinct messages
- [ ] ManagerDashboard lists process areas the user manages; per-doc row shows acked/pending/revoked counts
- [ ] PublishDialog adds mode=active|passive radio + ack_type select (auto-locked if criticality forces signature)
- [ ] TenantSettings lets tenant admin pick default ack_type
- [ ] All pages rendered under existing app shell with correct route guards (RBAC: doc.read for inbox, doc.manage for dashboard, tenant.admin for settings)

**Verify:** `cd frontend && rtk pnpm -C apps/web vitest run src/features/distribution src/features/tenant` and `rtk pnpm -C apps/web tsc --noEmit`

---

### Task 10.1: InboxPage — list obligations

- [ ] **Step 1: RED — vitest**

```tsx
// InboxPage.test.tsx
it("renders obligations sorted by obliged_at desc", async () => {
  server.use(http.get("/api/distribution/inbox", () => HttpResponse.json({
    items: [
      { id: "o1", document_id: "d1", title: "ISO Manual", area: "Quality", ack_type: "view",      obliged_at: "2026-04-23T10:00:00Z", status: "pending" },
      { id: "o2", document_id: "d2", title: "Safety SOP", area: "Safety",  ack_type: "signature", obliged_at: "2026-04-24T09:00:00Z", status: "pending" },
    ],
  })));
  render(<InboxPage />, { wrapper: QueryWrapper });
  const rows = await screen.findAllByRole("row");
  expect(rows[1]).toHaveTextContent("Safety SOP"); // newest first
  expect(rows[2]).toHaveTextContent("ISO Manual");
});
```

- [ ] **Step 2: Implement** — thin TanStack Query list page (no custom state):

```tsx
// InboxPage.tsx
export function InboxPage() {
  const q = useQuery({ queryKey: ["inbox"], queryFn: () => api.get<InboxList>("/distribution/inbox") });
  const [activeId, setActiveId] = useState<string | null>(null);
  if (q.isLoading) return <Spinner />;
  return (
    <>
      <Table>
        <thead><tr><th>Title</th><th>Area</th><th>Ack</th><th>Obliged</th><th></th></tr></thead>
        <tbody>
          {q.data?.items.map((it) => (
            <tr key={it.id} role="row">
              <td>{it.title}</td><td>{it.area}</td>
              <td><AckTypeBadge type={it.ack_type} /></td>
              <td>{formatRelative(it.obliged_at)}</td>
              <td><Button onClick={() => setActiveId(it.id)}>Open</Button></td>
            </tr>
          ))}
        </tbody>
      </Table>
      {activeId && <AckModal obligationId={activeId} onClose={() => setActiveId(null)} />}
    </>
  );
}
```

- [ ] **Step 3: Run → PASS. Commit.**

```bash
rtk git add frontend/apps/web/src/features/distribution/pages/InboxPage.tsx frontend/apps/web/src/features/distribution/pages/InboxPage.test.tsx
rtk git commit -m "feat(spec4/phase10): InboxPage — list pending obligations"
```

---

### Task 10.2: AckModal — view + signature paths

- [ ] **Step 1: RED**

```tsx
it("signature ack requires password; shows 401 on wrong password", async () => {
  server.use(
    http.get("/api/documents/:id", () => HttpResponse.json({ document_id: "d1", ack_type: "signature", nonce: "N1", body_html: "<p>text</p>" })),
    http.post("/api/documents/:id/ack", async ({ request }) => {
      const b = await request.json(); return b.password === "right" ? new HttpResponse(null, { status: 204 }) : new HttpResponse("invalid password", { status: 401 });
    })
  );
  render(<AckModal obligationId="o1" onClose={() => {}} />, { wrapper: QueryWrapper });
  await userEvent.type(await screen.findByLabelText("Password"), "wrong");
  await userEvent.click(screen.getByRole("button", { name: /confirm/i }));
  expect(await screen.findByText(/invalid password/i)).toBeInTheDocument();

  await userEvent.clear(screen.getByLabelText("Password"));
  await userEvent.type(screen.getByLabelText("Password"), "right");
  await userEvent.click(screen.getByRole("button", { name: /confirm/i }));
  await waitFor(() => expect(screen.queryByRole("dialog")).not.toBeInTheDocument());
});
```

- [ ] **Step 2: Implement**

```tsx
export function AckModal({ obligationId, onClose }: Props) {
  const doc = useQuery({ queryKey: ["ack-doc", obligationId], queryFn: () => api.get<AckDoc>(`/documents/${obligationId}`) });
  const [password, setPassword] = useState("");
  const [error, setError] = useState<string | null>(null);

  const ack = useMutation({
    mutationFn: () => api.post(`/documents/${obligationId}/ack`, { nonce: doc.data!.nonce, password: password || undefined }),
    onSuccess: onClose,
    onError: (e: ApiError) => {
      setError(
        e.status === 401 ? "Invalid password" :
        e.status === 409 ? "This confirmation is no longer valid — refresh and try again" :
        e.status === 410 ? "Confirmation expired — refresh and try again" :
        e.status === 429 ? "Too many attempts. Try again in 15 minutes." :
        "Could not confirm. Try again."
      );
    },
  });

  if (doc.isLoading) return <Modal open><Spinner /></Modal>;
  return (
    <Modal open onClose={onClose} role="dialog">
      <Content dangerouslySetInnerHTML={{ __html: doc.data!.body_html }} />
      {doc.data!.ack_type === "signature" && (
        <label>Password
          <input type="password" value={password} onChange={(e) => setPassword(e.target.value)} />
        </label>
      )}
      {error && <div role="alert">{error}</div>}
      <Button disabled={doc.data!.ack_type === "signature" && !password} onClick={() => ack.mutate()}>
        Confirm Read
      </Button>
    </Modal>
  );
}
```

- [ ] **Step 3: Run → PASS. Commit.**

```bash
rtk git add frontend/apps/web/src/features/distribution/components/
rtk git commit -m "feat(spec4/phase10): AckModal — view + signature paths, full error mapping"
```

---

### Task 10.3: ManagerDashboard

- [ ] **Step 1: RED**

```tsx
it("shows per-doc ack coverage counts", async () => {
  server.use(http.get("/api/distribution/manager/coverage", () => HttpResponse.json({
    rows: [
      { document_id: "d1", title: "ISO Manual", area: "Quality", pending: 3, acked: 12, revoked: 1 },
    ],
  })));
  render(<ManagerDashboardPage />, { wrapper: QueryWrapper });
  const row = await screen.findByText("ISO Manual");
  expect(row.closest("tr")).toHaveTextContent("3");
  expect(row.closest("tr")).toHaveTextContent("12");
});
```

- [ ] **Step 2: Implement** — thin table component (≤80 lines).
- [ ] **Step 3: Run → PASS. Commit.**

```bash
rtk git add frontend/apps/web/src/features/distribution/pages/ManagerDashboardPage.tsx frontend/apps/web/src/features/distribution/pages/ManagerDashboardPage.test.tsx
rtk git commit -m "feat(spec4/phase10): ManagerDashboard — per-doc ack coverage"
```

---

### Task 10.4: PublishDialog extension — ack_type + mode

- [ ] **Step 1: RED — lock signature when criticality=safety/regulatory**

```tsx
it("locks ack_type to signature when criticality=safety", async () => {
  render(<PublishDialog open doc={{ id: "d1", criticality: "safety" }} />, { wrapper: QueryWrapper });
  const select = screen.getByLabelText("Acknowledgement type");
  expect(select).toHaveValue("signature");
  expect(select).toBeDisabled();
});
```

- [ ] **Step 2: Implement** — extend existing dialog with two new fields driven by doc metadata:

```tsx
const lockSignature = doc.criticality === "safety" || doc.criticality === "regulatory";
const [ackType, setAckType] = useState(lockSignature ? "signature" : (doc.ack_type ?? "view"));
const [mode, setMode] = useState(doc.distribution_mode ?? "active");

// ... in JSX:
<label>Acknowledgement type
  <select value={ackType} onChange={(e) => setAckType(e.target.value)} disabled={lockSignature}>
    <option value="view">View</option>
    <option value="signature">Signature</option>
  </select>
</label>
<fieldset>
  <legend>Distribution mode</legend>
  <label><input type="radio" checked={mode==="active"}  onChange={()=>setMode("active")} /> Active (push)</label>
  <label><input type="radio" checked={mode==="passive"} onChange={()=>setMode("passive")} /> Passive (pull)</label>
</fieldset>
```

- [ ] **Step 3: Run → PASS. Commit.**

```bash
rtk git add frontend/apps/web/src/features/approval/components/PublishDialog.tsx frontend/apps/web/src/features/approval/components/PublishDialog.test.tsx
rtk git commit -m "feat(spec4/phase10): PublishDialog — ack_type picker + mode toggle"
```

---

### Task 10.5: TenantSettings

- [ ] **Step 1: RED + Implement** — single-form page with RHF + Zod (~60 lines). Dispatch to `haiku` via Codex.

```tsx
const schema = z.object({ default_ack_type: z.enum(["view", "signature"]) });
export function TenantSettingsPage() {
  const { data } = useQuery({ queryKey:["tenant","settings"], queryFn: () => api.get("/tenant/settings") });
  const { register, handleSubmit } = useForm({ resolver: zodResolver(schema), values: data });
  const save = useMutation({ mutationFn: (v) => api.put("/tenant/settings", v) });
  return (
    <form onSubmit={handleSubmit((v) => save.mutate(v))}>
      <label>Default acknowledgement type
        <select {...register("default_ack_type")}>
          <option value="view">View</option>
          <option value="signature">Signature</option>
        </select>
      </label>
      <Button type="submit">Save</Button>
    </form>
  );
}
```

- [ ] **Step 2: Commit.**

```bash
rtk git add frontend/apps/web/src/features/tenant/
rtk git commit -m "feat(spec4/phase10): TenantSettings — default ack_type"
```

---

### Task 10.6: Routes + RBAC

- [ ] **Step 1: Register routes** in `app/routes.tsx` with existing `<RequirePermission>` wrappers:

```tsx
<Route path="/inbox"               element={<RequirePermission perm="doc.read"><InboxPage /></RequirePermission>} />
<Route path="/distribution/manager" element={<RequirePermission perm="doc.manage"><ManagerDashboardPage /></RequirePermission>} />
<Route path="/settings/tenant"     element={<RequirePermission perm="tenant.admin"><TenantSettingsPage /></RequirePermission>} />
```

- [ ] **Step 2: tsc --noEmit → green. Build → green. Commit.**

```bash
rtk pnpm -C frontend/apps/web tsc --noEmit
rtk git add frontend/apps/web/src/app/routes.tsx
rtk git commit -m "feat(spec4/phase10): routes + RBAC guards for inbox/manager/tenant"
```

---

### Phase 10 closeout

- [ ] Opus review (model=opus): "Review Phase 10 UI. Verify: (a) no new state library, (b) error code mapping matches backend (401/409/410/429), (c) lock-signature logic correct for criticality, (d) RBAC guards present on all 3 routes, (e) InboxPage sorts desc by obliged_at."
- [ ] Codex QUALITY review.

---

## Phase 11: Integration / Load / E2E Tests

**Goal:** full-stack coverage. Integration = real Postgres, exercised worker + API. Load = advisory-lock contention under rev-N / rev-N+1 race. E2E = Playwright flow from publish to ack.

**/simplify:** reuse existing integration test harness (`internal/testhelper/integration.go`) and Playwright config. No new test framework.

**Codex model:** `gpt-5.3-codex --effort medium` (integration + load) and `gpt-5.3-codex --effort low` (E2E scripting).
**Codex review mode:** COVERAGE.
**Opus review:** YES, after Phase 11.

**Files:**
- Create: `internal/modules/distribution/integration_test.go` (build tag `//go:build integration`)
- Create: `internal/modules/distribution/load_test.go` (build tag `//go:build load`)
- Create: `frontend/apps/web/e2e/distribution.spec.ts`

**Acceptance Criteria:**
- [ ] Integration: publish → outbox → FanoutWorker tick → `document_distributions` row created → AckService → `acked_at` set; all round-trip in ≤3 seconds
- [ ] Load: 50 parallel publishers on same controlled_document_id with rev-N and rev-N+1; final state has exactly 1 non-obsolete rev and obligations point to it; zero duplicate obligations
- [ ] E2E: login as user in area → inbox shows doc → click → confirm read → inbox clears; signature path asks password
- [ ] E2E: 410 expired nonce — force nonce expiry, click confirm, expect red error banner

**Verify:** `rtk go test -tags=integration ./internal/modules/distribution/... -count=1` + `rtk go test -tags=load ./internal/modules/distribution/... -count=1 -timeout 5m` + `rtk pnpm -C frontend/apps/web exec playwright test e2e/distribution.spec.ts`

---

### Task 11.1: Integration — full publish → ack round-trip

- [ ] **Step 1: Test (single flow).**

```go
//go:build integration
func TestIntegration_PublishThroughAck(t *testing.T) {
    env := testhelper.StartFullStack(t) // boots Postgres + starts worker + API
    defer env.Stop()

    tid := env.SeedTenant(t)
    area := env.SeedArea(t, tid, "A")
    env.AddMembership(t, tid, "alice", area)
    cd := env.SeedControlledDoc(t, tid, area, "active", "view")

    // Publish rev 1.
    revID := env.PublishApproved(t, tid, cd, "alice")

    // Wait up to 3s for FanoutWorker to create obligation.
    deadline := time.Now().Add(3 * time.Second)
    var obID string
    for time.Now().Before(deadline) {
        obID = env.FindObligation(t, tid, revID, "alice")
        if obID != "" { break }
        time.Sleep(200 * time.Millisecond)
    }
    if obID == "" { t.Fatal("obligation not created within 3s") }

    // Alice acks.
    env.Ack(t, tid, "alice", obID, "view")

    // Ledger + signature persisted.
    var sig sql.NullString; var ackedAt sql.NullTime
    env.DB().QueryRow(`SELECT ack_signature, acked_at FROM document_distributions WHERE id=$1`, obID).Scan(&sig, &ackedAt)
    if !sig.Valid || len(sig.String) != 64 { t.Fatal("missing sig") }
    if !ackedAt.Valid { t.Fatal("missing acked_at") }
}
```

- [ ] **Step 2: Run → PASS. Commit.**

```bash
rtk go test -tags=integration ./internal/modules/distribution/... -count=1
rtk git add internal/modules/distribution/integration_test.go
rtk git commit -m "test(spec4/phase11): integration — publish through ack round-trip"
```

---

### Task 11.2: Load — rev-N / rev-N+1 advisory-lock contention

- [ ] **Step 1: Load test**

```go
//go:build load
func TestLoad_Rev_NPlus1_Contention(t *testing.T) {
    env := testhelper.StartFullStack(t); defer env.Stop()
    tid := env.SeedTenant(t)
    area := env.SeedArea(t, tid, "A")
    for i := 0; i < 100; i++ { env.AddMembership(t, tid, fmt.Sprintf("u%d", i), area) }
    cd := env.SeedControlledDoc(t, tid, area, "active", "view")

    var wg sync.WaitGroup
    // 50 parallel publishers producing rev 1..50.
    for i := 1; i <= 50; i++ {
        wg.Add(1)
        go func(rev int) { defer wg.Done(); env.ForcePublishRev(t, tid, cd, rev) }(i)
    }
    wg.Wait()
    env.WaitUntilOutboxDrained(t, 30*time.Second)

    // Exactly 1 non-obsolete rev remains.
    var n int
    env.DB().QueryRow(`SELECT COUNT(*) FROM documents WHERE controlled_document_id=$1 AND status<>'obsolete'`, cd).Scan(&n)
    if n != 1 { t.Fatalf("want 1 live rev, got %d", n) }

    // No duplicate obligations per (user, doc).
    var dup int
    env.DB().QueryRow(`
        SELECT COUNT(*) FROM (
          SELECT recipient_user_id, controlled_document_id, COUNT(*) c
            FROM document_distributions WHERE controlled_document_id=$1 GROUP BY 1,2 HAVING COUNT(*) > 1
        ) x`, cd).Scan(&dup)
    if dup != 0 { t.Fatalf("got %d duplicate obligations", dup) }
}
```

- [ ] **Step 2: Run → PASS. Commit.**

```bash
rtk go test -tags=load ./internal/modules/distribution/... -count=1 -timeout 5m
rtk git add internal/modules/distribution/load_test.go
rtk git commit -m "test(spec4/phase11): load — 50-way rev contention, 0 duplicate obligations"
```

---

### Task 11.3: E2E Playwright — publish → ack

- [ ] **Step 1:**

```ts
// frontend/apps/web/e2e/distribution.spec.ts
test("user acks published doc and inbox clears", async ({ page, seed }) => {
  await seed.tenant("t1");
  await seed.user("alice", "alice@co", "pw");
  await seed.area("Q");
  await seed.membership("alice", "Q");
  const docID = await seed.publishDoc({ area: "Q", title: "ISO", mode: "active", ackType: "view" });

  await page.goto("/login"); await page.fill("input[name=email]", "alice@co"); await page.fill("input[name=password]", "pw"); await page.click("button[type=submit]");
  await page.goto("/inbox");
  await expect(page.getByText("ISO")).toBeVisible();
  await page.getByRole("button", { name: /open/i }).click();
  await page.getByRole("button", { name: /confirm read/i }).click();
  await expect(page.getByText("ISO")).not.toBeVisible();
});

test("signature path with wrong password shows error", async ({ page, seed }) => { /* … */ });
test("expired nonce shows 410 banner", async ({ page, seed }) => { /* … */ });
```

- [ ] **Step 2: Run Playwright → PASS. Commit.**

```bash
rtk pnpm -C frontend/apps/web exec playwright test e2e/distribution.spec.ts
rtk git add frontend/apps/web/e2e/distribution.spec.ts
rtk git commit -m "test(spec4/phase11): E2E — inbox ack + signature + expired nonce"
```

---

### Phase 11 closeout

- [ ] Opus review (model=opus): "Review Phase 11. Verify: (a) integration harness reuses existing, (b) load test asserts both liveness (1 rev) and correctness (0 dup obligations), (c) E2E covers all 4 HTTP error codes (201/401/409/410/429) from Phase 6."
- [ ] Codex COVERAGE review — final gap hunt.

---

## Phase 12: Hardening + Cutover Migrations

**Goal:** production hardening — (a) backfill obligations for any pre-existing published active-mode docs in prod (DML migration 0168), (b) expose manual replay admin endpoint for DLQ rows, (c) add feature flag `distribution.active_mode_enabled` (default OFF per tenant; on via migration script).

**/simplify:** DML migration uses the same SQL as reconciliation backfill. Replay endpoint is 30 LOC. Feature flag uses existing `internal/shared/featureflags/`.

**Codex model:** `gpt-5.3-codex --effort high` (prod-safety critical).
**Codex review mode:** OPERATIONS.
**Opus review:** YES, after Phase 12 — FINAL.

**Files:**
- Create: `migrations/0169_distribution_backfill_production.sql` (DML — idempotent)
- Create: `internal/modules/distribution/delivery/http/admin_dlq_handler.go`
- Create: `internal/modules/distribution/delivery/http/admin_dlq_handler_test.go`
- Modify: `internal/shared/featureflags/` (register `distribution.active_mode_enabled`)
- Modify: `apps/api/cmd/metaldocs-api/main.go` (route + flag guard)

**Acceptance Criteria:**
- [ ] Migration 0169 idempotent (running twice = zero new rows)
- [ ] DLQ admin endpoint `POST /admin/distribution/dlq/:id/replay` moves row back to outbox with `attempt_count=0`, `next_attempt_at=NOW()`
- [ ] DLQ admin endpoint requires `distribution.admin` permission
- [ ] Feature flag `distribution.active_mode_enabled` gates FanoutWorker.Tick — if OFF, tick is a no-op
- [ ] Flag default OFF; migration 0169 does NOT auto-enable (runbook step)

**Verify:** `rtk go test ./internal/modules/distribution/... -count=1 -run Admin` and `rtk psql -f migrations/0169... -f migrations/0169...` (apply twice, assert idempotent)

---

### Task 12.1: Migration 0169 — DML backfill

- [ ] **Step 1: Write migration (idempotent):**

```sql
-- migrations/0169_distribution_backfill_production.sql
-- DML — inserts obligations for pre-existing published active-mode docs.
-- Idempotent: safe to re-run. Does NOT enable the active_mode flag.
BEGIN;

INSERT INTO document_distributions
  (id, tenant_id, controlled_document_id, recipient_user_id, ack_type, status, obliged_at, reconciliation_run_id)
SELECT gen_random_uuid(), d.tenant_id, d.id, m.user_id,
       COALESCE(d.ack_type, pa.default_ack_type, t.default_ack_type, 'view'),
       'pending', NOW(), 'migration_0169'
  FROM documents d
  JOIN area_memberships m ON m.process_area_code = d.process_area_code AND m.tenant_id = d.tenant_id
  JOIN process_areas pa ON pa.code = d.process_area_code AND pa.tenant_id = d.tenant_id
  JOIN metaldocs.tenants t ON t.id = d.tenant_id
 WHERE d.status = 'published' AND d.distribution_mode = 'active'
   AND NOT EXISTS (
       SELECT 1 FROM document_distributions dd
        WHERE dd.controlled_document_id = d.id AND dd.recipient_user_id = m.user_id
   );

COMMIT;
```

- [ ] **Step 2: Apply twice, assert second apply inserts 0 rows.**

- [ ] **Step 3: Commit.**

```bash
rtk git add migrations/0169_distribution_backfill_production.sql
rtk git commit -m "feat(spec4/phase12): migration 0169 — idempotent DML backfill for prod cutover"
```

---

### Task 12.2: DLQ replay admin endpoint

- [ ] **Step 1: RED**

```go
func TestAdminDLQReplay_MovesRowBack(t *testing.T) {
    db := testhelper.DB(t)
    dlqID := testhelper.SeedDLQRow(t, db, "t1", "d1", "v1")

    h := http.NewDLQAdminHandler(repo, authz)
    req := httptest.NewRequest("POST", fmt.Sprintf("/admin/distribution/dlq/%s/replay", dlqID), nil).WithContext(testhelper.CtxWithPerm("distribution.admin"))
    w := httptest.NewRecorder(); h.Replay(w, req)
    if w.Code != 204 { t.Fatalf("want 204, got %d — body=%s", w.Code, w.Body.String()) }

    var n, d int
    db.QueryRow(`SELECT COUNT(*) FROM distribution_outbox WHERE doc_version_id='v1'`).Scan(&n)
    db.QueryRow(`SELECT COUNT(*) FROM distribution_outbox_dlq WHERE id=$1`, dlqID).Scan(&d)
    if n != 1 || d != 0 { t.Fatalf("outbox=%d dlq=%d (want 1/0)", n, d) }
}
```

- [ ] **Step 2: Implement** (30 LOC total: handler + repo method `ReplayFromDLQ`).

```go
func (r *sqlOutboxRepo) ReplayFromDLQ(ctx context.Context, db *sql.DB, dlqID string) error {
    tx, _ := db.BeginTx(ctx, nil); defer tx.Rollback()
    var row OutboxRow
    err := tx.QueryRowContext(ctx, `
        DELETE FROM distribution_outbox_dlq WHERE id=$1
        RETURNING tenant_id, controlled_document_id, doc_version_id, prior_version_id, event_type, scoped_recipient_user_id`,
        dlqID).Scan(&row.TenantID, &row.ControlledDocumentID, &row.DocVersionID, &row.PriorVersionID, &row.EventType, &row.ScopedRecipientUserID)
    if err != nil { return err }
    _, err = tx.ExecContext(ctx, `
        INSERT INTO distribution_outbox
          (tenant_id, controlled_document_id, doc_version_id, prior_version_id, event_type,
           scoped_recipient_user_id, enqueued_at, next_attempt_at, attempt_count)
        VALUES ($1,$2,$3,$4,$5,$6, NOW(), NOW(), 0)`,
        row.TenantID, row.ControlledDocumentID, row.DocVersionID, row.PriorVersionID, row.EventType, row.ScopedRecipientUserID)
    if err != nil { return err }
    return tx.Commit()
}

func (h *DLQAdminHandler) Replay(w http.ResponseWriter, r *http.Request) {
    if err := h.authz.Require(r.Context(), "distribution.admin"); err != nil { http.Error(w, "forbidden", 403); return }
    id := chi.URLParam(r, "id")
    if err := h.repo.ReplayFromDLQ(r.Context(), h.db, id); err != nil { http.Error(w, err.Error(), 500); return }
    w.WriteHeader(204)
}
```

- [ ] **Step 3: Run → PASS. Commit.**

```bash
rtk git add internal/modules/distribution/delivery/http/admin_dlq_handler.go internal/modules/distribution/delivery/http/admin_dlq_handler_test.go internal/modules/distribution/repository/outbox_repo.go
rtk git commit -m "feat(spec4/phase12): DLQ admin replay endpoint (RBAC: distribution.admin)"
```

---

### Task 12.3: Feature flag gate

- [ ] **Step 1: Register flag** in `internal/shared/featureflags/registry.go`:

```go
const DistributionActiveModeEnabled = "distribution.active_mode_enabled"
// default: false (per-tenant opt-in)
```

- [ ] **Step 2: Guard FanoutWorker.Tick:**

```go
func (w *FanoutWorker) Tick(ctx context.Context) (int, error) {
    if !w.flags.Enabled(ctx, DistributionActiveModeEnabled, currentTenantID(ctx)) {
        return 0, nil
    }
    // ... existing claim loop
}
```

- [ ] **Step 3: Test flag-off = no-op.**

```go
func TestFanoutWorker_FlagOff_Noops(t *testing.T) {
    db := testhelper.DB(t)
    flags := testhelper.StubFlags(map[string]bool{application.DistributionActiveModeEnabled: false})
    w := application.NewFanoutWorker(db, outboxRepo, distRepo, resolver, flags)
    testhelper.SeedOutboxRow(t, db, "v1") // would be picked up if flag ON
    n, err := w.Tick(ctx)
    if err != nil || n != 0 { t.Fatalf("want noop, got n=%d err=%v", n, err) }
}
```

- [ ] **Step 4: Run → PASS. Commit.**

```bash
rtk git add internal/shared/featureflags/ internal/modules/distribution/application/fanout_worker.go internal/modules/distribution/application/fanout_worker_test.go
rtk git commit -m "feat(spec4/phase12): feature-flag gate distribution.active_mode_enabled (default off)"
```

---

### Phase 12 closeout — FINAL

- [ ] Apply migration 0169 to staging; assert 0 rows on second apply.
- [ ] Manually flip flag ON for one tenant via psql; verify FanoutWorker processes outbox.
- [ ] **Opus FINAL review (model=opus):** "Review entire Spec 4 — cross-phase. Verify: (a) every Phase 1 migration has a corresponding feature, (b) all six Phase Contracts upheld, (c) no orphan code (iface without impl, repo method without caller), (d) feature flag defaults OFF and runbook documents per-tenant enablement, (e) migration 0169 is pure DML and idempotent, (f) every error from ISO 9001 audit scenarios has a corresponding test."
- [ ] Codex OPERATIONS review — final cross-phase pass.
- [ ] Update CLAUDE.md memory: `project_spec4_plan.md` — status "implemented 2026-MM-DD".

---

## Testing Strategy Summary

| Layer | Framework | Trigger | Cost-gate |
|-------|-----------|---------|-----------|
| Unit (Go) | stdlib `testing` | every commit | `rtk go test -count=1` |
| Integration (Go+PG) | `//go:build integration` | PR | `rtk go test -tags=integration -count=1` |
| Load (Go+PG concurrent) | `//go:build load` | nightly | `-timeout 5m` |
| Unit (TS) | vitest | every commit | `rtk pnpm -C frontend/apps/web vitest run` |
| E2E | Playwright | PR | `playwright test` |
| Type check | tsc | every commit | `rtk pnpm -C frontend/apps/web tsc --noEmit` |

**Golden rule:** any PR that modifies `internal/modules/distribution/**` or `migrations/016{7,8,9}_*` MUST run integration + E2E in CI.

---

## Open Risks / Non-Blocking Followups (post-cutover)

1. **Nonce storage in Postgres:** adequate at current scale (<1k/day). If nonce volume exceeds 10k/day, migrate to Redis with 15-min TTL keys. Log as spec4-followup-01.
2. **Load test coverage:** only rev-N/N+1 contention. Post-launch, add test for 1k simultaneous acks on one doc. Log as spec4-followup-02.
3. **Watermark CSS `body::after`:** works for Chromium/Gotenberg; verify with actual SOP page samples before GA. Log as spec4-followup-03.
4. **DST handling only covers IANA zones** with ±hour transitions. LMT/historical zones not tested. Log as spec4-followup-04.
5. **DLQ admin UI** intentionally omitted — CLI/psql-only for now. Add if operator reports friction.

---

## End of Plan
