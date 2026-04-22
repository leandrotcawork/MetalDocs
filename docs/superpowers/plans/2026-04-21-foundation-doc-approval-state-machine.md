# Foundation Spec 2 — Document Approval State Machine Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use `superpowers-extended-cc:subagent-driven-development` (recommended) or `superpowers-extended-cc:executing-plans` to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Date:** 2026-04-21
**Spec:** [docs/superpowers/specs/2026-04-21-foundation-doc-approval-state-machine-design.md](../specs/2026-04-21-foundation-doc-approval-state-machine-design.md)
**Depends on:** Spec 1 — Foundation Taxonomy × RBAC × Controlled Document Registry (shipped on branch `feature/foundation-spec1`, commit `0431720`)
**Status:** Ready to execute
**Plan hardening:** Codex `gpt-5.3-codex` reasoning=high — per-phase review (OPERATIONS / COVERAGE / QUALITY / ARCHITECTURE / SEQUENCING modes) + final integration pass.

**Goal:** Ship the full 6-state approval lifecycle on `documents_v2` revisions (draft → under_review → approved → published → superseded/obsolete, with `rejected` rework sink) with DB-enforced transition legality, sequential per-profile routes, password re-auth signatures bound to content hash, edit lock during review, immediate/scheduled effective dates, and audit-preserving reject history.

**Architecture:** Extend `documents_v2` with state columns + legal-transition trigger. New subpackage `internal/modules/documents_v2/approval/` holds instance/stage/signoff machinery. Route config snapshotted into runtime instance tables at submit time. DB-side authz backstop via NOINHERIT role boundary + SECURITY DEFINER functions + session-context tripwire GUCs + hard integrity triggers. Outbox-style `governance_events` written same-transaction as every state change. Single cron worker (`effective_date_publisher`) with `FOR UPDATE SKIP LOCKED`. **Legacy removed:** no Phase A/B/C dual-state shadow deploy — `documents.status` enum replaced in one migration; existing `finalized` → `published`, existing `archived` → `obsolete`, legacy values dropped from CHECK constraint.

**Tech Stack:** Go 1.22, PostgreSQL 16, `database/sql` with `lib/pq`, React 18 + TypeScript, Playwright, Docker Compose dev stack.

---

## Model Assignments

| Model | When to use |
|---|---|
| **Codex `gpt-5.3-codex` medium** | SQL migrations, Go domain models, repositories, application services, authorization logic, integration tests, scheduler jobs, signature provider, membership_tx helper |
| **Codex `gpt-5.3-codex` high** | SECURITY DEFINER function bodies, SoD trigger, legal transition trigger, approval race invariant tests, content hash canonicalization parity (Go+TS), DB role/privilege migrations |
| **Sonnet** | HTTP handlers (CRUD boilerplate), React medium-complexity components (RouteAdminPage, InboxPage, SignoffDialog, ApprovalTimelinePanel, SupersedePublishDialog), API hooks |
| **Haiku** | GRANT-only SQL files, `index.ts` exports, route registration in `module.go` and `main.go`, trivial type additions, test fixture JSON |
| **Opus** | Phase-end reviews only (after Phase 2, 4, 6, 8, 10, 12). Never coding. |

---

## Architectural Adaptation — User Identity (Codex-approved Option B)

Spec 2 text assumes `public.users(id UUID, tenant_id UUID, deactivated_at)` with composite unique `(tenant_id, id)` and composite FKs `REFERENCES users(tenant_id, id)`. **This table does not exist.** The shipped reality after Spec 1:

- `metaldocs.iam_users(user_id TEXT PRIMARY KEY, display_name, is_active BOOL, created_at, updated_at)` — no tenant_id, no deactivated_at.
- `documents.created_by TEXT` (migration 0116 converted UUID → TEXT: "holds auth identity user_id strings, not IAM UUIDs").
- `user_process_areas.user_id TEXT`, `granted_by TEXT`.
- `governance_events.actor_user_id TEXT`.

**Decision (Codex-approved):** adapt Spec 2 to TEXT user identity. Rationale: the alternative (rewriting every user FK to UUID) is a cross-cutting identity migration orthogonal to approval-state behavior and explicitly out of scope. "Legacy removed" applies to the `documents.status` enum and approval machinery — not to user identity representation.

**Concrete deltas vs. spec:**

| Spec element | Spec text | Plan implementation |
|---|---|---|
| User FK type | `UUID REFERENCES users(id)` | `TEXT REFERENCES metaldocs.iam_users(user_id)` |
| Composite tenant FK | `(tenant_id, granted_by) REFERENCES users(tenant_id, id)` | `(tenant_id, granted_by) REFERENCES metaldocs.iam_users(tenant_id, user_id)` after Migration 0130 adds `tenant_id` + `UNIQUE (tenant_id, user_id)` to `iam_users` |
| `deactivated_at` | column on `public.users` | column added to `metaldocs.iam_users` by Migration 0130 |
| `approval_signoffs.actor_user_id` | `UUID REFERENCES users(id)` | `TEXT REFERENCES metaldocs.iam_users(user_id)` |
| `approval_instances.submitted_by` | `UUID REFERENCES users(id)` | `TEXT REFERENCES metaldocs.iam_users(user_id)` |
| SECURITY DEFINER fn `grant_area_membership` signature | `(_tenant_id UUID, _user_id UUID, _area_code TEXT, _role TEXT, _granted_by UUID)` | `(_tenant_id UUID, _user_id TEXT, _area_code TEXT, _role TEXT, _granted_by TEXT)` |
| Session context GUC `metaldocs.actor_id` | UUID cast | TEXT compare |

All other spec elements (state enum, transition trigger, route snapshot, SoD semantics, signature method seam, cron worker, idempotency keys, OCC, error codes) are implemented verbatim.

---

## Legacy Removal (per user directive)

**Removed in this plan — no dual-state retention, no Phase A/B/C:**

1. `documents.status` CHECK constraint stripped of `finalized` and `archived`. Existing rows migrated in one transaction:
   - `finalized` → `published` (timestamp preserved via `effective_from = updated_at`).
   - `archived` → `obsolete` (timestamp preserved via `effective_to = archived_at`).
   - Governance event `legacy_status_remap` inserted per row with before/after payload.
2. `documents.finalized_at`, `documents.archived_at` dropped after backfill (migration 0138). App layer code paths reading those columns are deleted in Phase 5 (service refactor).
3. Legacy `status IN ('draft','finalized','archived')` code paths purged from `internal/modules/documents_v2/application/service.go`, `domain/model.go`, repository, React status badges. No compatibility shims.
4. No `session_replication_role = 'replica'` migration escape hatch documented — triggers must be honored by all writes.

**Retained (explicitly):**
- Existing `documents` rows (not destroyed; data migrated in place).
- `governance_events` table from Spec 1 (extended with new `event_type` values).
- `metaldocs.iam_users` as canonical user store (extended with `tenant_id`, `deactivated_at`, composite unique).

---

## Actual Table Name Map (carry-forward from Spec 1)

| Spec name | Actual table | Schema |
|---|---|---|
| `users` | `iam_users` | metaldocs |
| `documents_v2.documents` | `documents_v2` (aliased from `documents`) | public |
| `document_profiles` | `document_profiles` | metaldocs |
| `process_areas` | `document_process_areas` | metaldocs |
| `controlled_documents` | `controlled_documents` | public |
| `user_process_areas` | `user_process_areas` | public |
| `governance_events` | `governance_events` | public |
| `approval_routes` (new) | `approval_routes` | public |
| `approval_route_stages` (new) | `approval_route_stages` | public |
| `approval_instances` (new) | `approval_instances` | public |
| `approval_stage_instances` (new) | `approval_stage_instances` | public |
| `approval_signoffs` (new) | `approval_signoffs` | public |

---

## File Structure Map

### New files (Go)
```
internal/modules/documents_v2/approval/
  domain/
    state.go
    state_test.go
    route.go
    route_test.go
    instance.go
    instance_test.go
    signoff.go
    signoff_test.go
    quorum.go
    quorum_test.go
    sod.go
    sod_test.go
  application/
    content_hash.go
    content_hash_test.go
    idempotency.go
    idempotency_test.go
    events.go
    membership_tx.go
    membership_tx_test.go
    submit_service.go
    submit_service_test.go
    decision_service.go
    decision_service_test.go
    publish_service.go
    publish_service_test.go
    scheduler_service.go
    scheduler_service_test.go
    supersede_service.go
    supersede_service_test.go
    obsolete_service.go
    obsolete_service_test.go
  infra/
    signature/
      provider.go
      password_reauth.go
      password_reauth_test.go
      registry.go
  delivery/http/
    handler.go
    routes_route.go
    routes_transitions.go
    routes_inbox.go
    types.go
  jobs/
    effective_date_publisher.go
    effective_date_publisher_test.go
    stuck_instance_watchdog.go
  repository/
    approval_repository.go
    approval_repository_test.go
  module.go
```

### Modified files (Go)
```
internal/modules/documents_v2/
  domain/model.go                   ← 7-state enum; drop finalized/archived; add revision fields
  application/service.go            ← Strip legacy finalized/archived paths; add lock/unlock helpers
  repository/repository.go          ← Add transition + lock/unlock queries with OCC

internal/modules/iam/
  domain/role_capabilities.go       ← bump RoleCapabilitiesVersion 1→2; add workflow.* caps
  application/authorization.go      ← register new caps
  application/area_membership_service.go
                                     ← route writes through metaldocs_membership_writer role
                                       + SET LOCAL GUCs + SECURITY DEFINER fn
  application/area_membership_service_test.go
                                     ← extend: asserts SET LOCAL, GUC, function call shape

apps/api/cmd/metaldocs-api/main.go ← wire approval module
internal/api/v2/types_gen.go       ← approval request/response contract structs
internal/api/v2/contract_test.go   ← approval endpoint error shape assertions
```

### New files (Frontend)
```
frontend/apps/web/src/features/approval/
  types.ts
  api.ts
  RouteAdminPage.tsx
  RouteStageEditor.tsx
  InboxPage.tsx
  SignoffDialog.tsx
  ApprovalTimelinePanel.tsx
  SupersedePublishDialog.tsx
  RejectionBanner.tsx
  hooks/
    useApprovalInstance.ts
    useInbox.ts
    useRouteConfig.ts
  index.ts
```

### Modified files (Frontend)
```
frontend/apps/web/src/features/documents/v2/
  DocumentEditorPage.tsx          ← disable editor on locked_at; show stage banner
  DocumentStatusBadge.tsx         ← 7-state badge; drop finalized/archived labels
  DocumentDetailPage.tsx          ← embed ApprovalTimelinePanel + RejectionBanner

frontend/apps/web/src/features/registry/
  RegistryDetailPage.tsx          ← revision list with status/effective dates;
                                    "Obsoletar documento" (admin only)
```

### New SQL migrations
```
migrations/
  0130_iam_users_tenant_deactivated.sql
  0131_documents_v2_approval_state.sql
  0132_approval_routes.sql
  0133_approval_instances.sql
  0134_user_process_areas_hardening.sql
  0135_db_roles_security_definer.sql
  0136_grants_approval_tables.sql
  0137_documents_v2_legacy_status_remap.sql
  0138_documents_v2_drop_legacy_columns.sql
  0139_governance_events_workflow_caps_bump.sql
```

---

## Phase Map

| Phase | Scope | Codex verify mode | Opus review at end |
|---|---|---|---|
| 1 | DB migrations (iam_users hardening, approval state, routes, instances, roles, legacy remap) | OPERATIONS | — |
| 2 | Domain layer: state, route, instance, signoff, quorum, sod + tests | COVERAGE | ✅ |
| 3 | Infra: content_hash (Go+TS parity), idempotency, signature provider, membership_tx | QUALITY | — |
| 4 | Repository: approval_repository + OCC; extend documents_v2 repo; legacy path removal | ARCHITECTURE | ✅ |
| 5 | Application services: submit, decision, publish, scheduler, supersede, obsolete, events | COVERAGE | — |
| 6 | IAM: role_capabilities v2, area_membership via canonical DB fns, workflow.instance.cancel | ARCHITECTURE | ✅ |
| 7 | HTTP delivery: routes + contract types + error mapping | QUALITY | — |
| 8 | Scheduler jobs: effective_date_publisher, stuck_instance_watchdog | OPERATIONS | ✅ |
| 9 | Frontend: RouteAdmin, Inbox, SignoffDialog, Timeline, lock UX, badge, RegistryDetail mods | COVERAGE | — |
| 10 | Integration tests (trigger bypass, membership fn gates, schema lockdown, races, obsolete cascade) | QUALITY | ✅ |
| 11 | E2E Playwright (happy, reject, scheduled, sod, lock, quorum, route_admin) | COVERAGE | — |
| 12 | CI invariants + smoke + perf benchmarks | OPERATIONS | ✅ |

---

## Phase 1: Database Migrations (Codex Round 1 revised)

**Intent:** DB-only phase. All Spec 2 schema changes land, but CHECK constraints and DML privileges stay **backward-compatible with pre-Spec-2 Go code**. Tightening (drop `finalized`/`archived` from CHECK, revoke DML on `user_process_areas`, drop `finalized_at`/`archived_at` columns) happens in Phase 5/6/12 co-released with the Go cutover.

**Codex Round 1 verdict:** REJECT, `upgrade_required: true` — 13 issues addressed. Artifact: `docs/superpowers/plans/reviews/phase-1-round-1.json`.

**Codex Round 2 verdict:** REJECT with 5 new findings (2 structural, 3 local). All 5 fixes applied inline. Co-plan skill caps at 2 rounds — plan delivered with explicit caveats. Artifact: `docs/superpowers/plans/reviews/phase-1-round-2.json`.

**Round 2 fixes applied inline:**

| # | Scope | Fix |
|---|---|---|
| R2-1 | structural | 0133 trigger now permits compat legacy transitions (`draft → finalized`, `finalized → archived`) through Phases 1–4; Phase 5 replaces it with strict 7-state-only |
| R2-2 | local | 0139 `ON CONFLICT ON CONSTRAINT <unique-index>` → index-inference conflict target with matching predicate |
| R2-3 | structural | `approval_stage_instances` gains `UNIQUE(id, approval_instance_id)`; `approval_signoffs` adds composite FK `(stage_instance_id, approval_instance_id) → (id, approval_instance_id)` — signoff cannot straddle instances |
| R2-4 | local | 0136 backfills `revoked_by := 'system:legacy'` for historical revoked rows, then adds CHECK `NOT VALID` + `VALIDATE` |
| R2-5 | local | 0137 `GRANT USAGE ON SCHEMA metaldocs TO metaldocs_security_owner, metaldocs_membership_writer` so schema-qualified refs work in hardened envs |

**Caveats (review before Phase 2 execution):**
- R2-1 and R2-3 are structural and were not re-validated by Codex. Verify at execution time — run integration suite against pre-Spec-2 code paths BEFORE tightening, and add a targeted smoke probe that inserts a signoff with mismatched `(stage_instance_id, approval_instance_id)` and expects FK rejection.
- Composite FK in R2-3 relies on `UNIQUE(id, approval_instance_id)` which is redundant with PK=id. Postgres accepts this; some linters flag it.

**Structural fixes applied in this revision:**

| Codex issue | Fix |
|---|---|
| #1 Remap before CHECK swap | Superset CHECK in 0131 (accepts legacy + new); tighten deferred to Phase 5 |
| #2 Numeric order contradicts dependency order | Renumbered 0130 → 0140 so file order = execution order |
| #3 Tenant retrofit model conflict | `iam_users.user_id` remains globally-unique PK; `tenant_id` is a lookup attribute; `UNIQUE(tenant_id, user_id)` only anchors composite FKs. No per-tenant identity namespace claimed. |
| #4 Sentinel DEFAULT footgun | DEFAULT is backfill-only; Phase 5 migration drops DEFAULT + adds app-layer tenant assertion |
| #5 Existing-data FK break | All composite FKs added `NOT VALID`; explicit backfill step; `VALIDATE CONSTRAINT` runs after verification |
| #6 SoD trigger race | `approval_signoffs.approval_instance_id` denormalized + `UNIQUE (approval_instance_id, actor_user_id)` — real uniqueness primitive beats race |
| #7 Cross-tenant signoff | BEFORE INSERT trigger asserts `actor_tenant_id = approval_instances.tenant_id` |
| #8 search_path hazard | All SECURITY DEFINER functions use `SET search_path = pg_catalog, pg_temp`; every object reference is schema-qualified |
| #9 Role DDL existence-safe | `DO $$ ... $$` guards check `pg_roles` before every `CREATE/ALTER ROLE` |
| #10 Phase-1-alone breaks app | CHECK kept permissive (superset); DML revoke on `user_process_areas` deferred to Phase 6 |
| #11 Premature column drop | `finalized_at` / `archived_at` DROP moved to Phase 12 stabilization gate |
| #12 0139 idempotency brittle | Unique partial index `ux_governance_events_caps_bump_spec_version` + `ON CONFLICT DO NOTHING` |
| #13 Missing hot-path indexes + monotonic guard | New migration 0140 adds inbox btree `(tenant_id, submitted_by, status, submitted_at DESC)` + trigger enforcing `NEW.revision_version >= OLD.revision_version` |

**Execution order (numeric = dependency):**

```
0130  iam_users tenant + deactivated_at (global identity, sentinel backfill, DEFAULT kept for now)
0131  documents: ADD state columns + SUPERSET CHECK (legacy + new 7 states)
0132  legacy status remap (finalized -> published, archived -> obsolete) + governance_events
0133  install legal-transition BEFORE UPDATE trigger
0134  approval_routes + approval_route_stages
0135  approval_instances + stage_instances + approval_signoffs (with denormalized instance_id + tenant triggers + SoD + immutability)
0136  user_process_areas hardening (revoked_by, NOT VALID FKs, backfill verify, VALIDATE, triggers)
0137  DB roles (idempotent) + schema lockdown + SECURITY DEFINER fns (hardened search_path, schema-qualified)
0138  GRANTs on new approval tables (no DML revoke on user_process_areas yet)
0139  governance_events role_capabilities_version_bump (unique partial index + ON CONFLICT DO NOTHING)
0140  revision_version monotonic guard + inbox btree index
```

**Deferred to later phases:**

| Migration | Content | Lands in |
|---|---|---|
| TBD-A | Tighten `documents.status` CHECK to 7-state-only | Phase 5, co-released with Go service cutover |
| TBD-B | REVOKE INSERT/UPDATE/DELETE on `user_process_areas` from `metaldocs_app` | Phase 6, co-released with IAM service switching to SECURITY DEFINER calls |
| TBD-C | DROP COLUMN `finalized_at`, `archived_at` | Phase 12 stabilization gate (after soak period) |
| TBD-D | ALTER COLUMN `iam_users.tenant_id` DROP DEFAULT | Phase 5 |

---

### Task 1.1 — Migration 0130: `iam_users` tenant + deactivated_at

**Goal:** Extend `metaldocs.iam_users` with `tenant_id UUID NOT NULL` (sentinel DEFAULT for backfill window only) and `deactivated_at TIMESTAMPTZ`. Add `UNIQUE (tenant_id, user_id)` to anchor composite FKs. Identity remains globally unique by `user_id`.

**Files:** Create `migrations/0130_iam_users_tenant_deactivated.sql`.

**Acceptance Criteria:**
- [ ] `tenant_id UUID NOT NULL DEFAULT 'ffffffff-ffff-ffff-ffff-ffffffffffff'` column present.
- [ ] `deactivated_at TIMESTAMPTZ NULL` column present.
- [ ] `UNIQUE (tenant_id, user_id)` index present.
- [ ] Partial unique `ux_iam_users_tenant_user_active` on `(tenant_id, user_id) WHERE deactivated_at IS NULL`.
- [ ] CHECK `iam_users_deactivated_after_created`.
- [ ] PK remains `(user_id)` — globally unique identity preserved.

**Steps:**

- [ ] **Step 1: Write migration**

```sql
-- migrations/0130_iam_users_tenant_deactivated.sql
-- Spec 2 Phase 1 (Codex-revised). Global user_id PK preserved.
-- tenant_id is a lookup attribute (NOT part of identity namespace).
-- DEFAULT sentinel is backfill-only; Phase 5 migration drops DEFAULT.

BEGIN;

ALTER TABLE metaldocs.iam_users
  ADD COLUMN IF NOT EXISTS tenant_id UUID NOT NULL
    DEFAULT 'ffffffff-ffff-ffff-ffff-ffffffffffff',
  ADD COLUMN IF NOT EXISTS deactivated_at TIMESTAMPTZ;

ALTER TABLE metaldocs.iam_users
  DROP CONSTRAINT IF EXISTS iam_users_deactivated_after_created,
  ADD  CONSTRAINT iam_users_deactivated_after_created
    CHECK (deactivated_at IS NULL OR deactivated_at >= created_at);

CREATE UNIQUE INDEX IF NOT EXISTS ux_iam_users_tenant_user
  ON metaldocs.iam_users (tenant_id, user_id);

CREATE UNIQUE INDEX IF NOT EXISTS ux_iam_users_tenant_user_active
  ON metaldocs.iam_users (tenant_id, user_id)
  WHERE deactivated_at IS NULL;

COMMIT;
```

- [ ] **Step 2: Verify**

```bash
docker compose down -v && docker compose up -d db && sleep 5
docker exec metaldocs-db psql -U metaldocs -d metaldocs -c "\d metaldocs.iam_users"
```
Expected: `tenant_id | uuid | not null default`, `deactivated_at | timestamp`, `ux_iam_users_tenant_user UNIQUE, btree (tenant_id, user_id)`.

- [ ] **Step 3: Commit**

```bash
rtk git add migrations/0130_iam_users_tenant_deactivated.sql
rtk git commit -m "feat(spec2/phase1): iam_users tenant + deactivated_at (0130)"
```

---

### Task 1.2 — Migration 0131: `documents` state columns + SUPERSET CHECK

**Goal:** Add Spec 2 state columns (`effective_from`, `effective_to`, `revision_number`, `revision_version`, `locked_at`, `content_hash_at_submit`). Replace existing CHECK with a **superset** that accepts both legacy (`finalized`, `archived`) and new (`under_review`, `approved`, `rejected`, `scheduled`, `published`, `superseded`, `obsolete`) states. Legal-transition trigger installed in Task 1.4 after remap is done.

**Files:** Create `migrations/0131_documents_v2_state_columns.sql`.

**Acceptance Criteria:**
- [ ] Six new columns present.
- [ ] CHECK accepts superset of 10 values (`draft`, `finalized`, `archived`, `under_review`, `approved`, `rejected`, `scheduled`, `published`, `superseded`, `obsolete`).
- [ ] `ux_documents_v2_cd_revision` unique index present.
- [ ] `ux_documents_v2_cd_active` partial unique index on `(controlled_document_id) WHERE status IN ('draft','under_review','approved','rejected','scheduled')`.
- [ ] No transition trigger yet (installed in Task 1.4).
- [ ] Existing Go service code writing `status='finalized'` still succeeds against the superset CHECK — verified by running the pre-Spec-2 integration suite.

**Steps:**

- [ ] **Step 1: Write migration**

```sql
-- migrations/0131_documents_v2_state_columns.sql
-- Spec 2 Phase 1. SUPERSET CHECK keeps pre-Spec-2 code alive until Phase 5
-- tightens it. Transition trigger installed in 0133 after legacy remap.

BEGIN;

ALTER TABLE documents
  DROP CONSTRAINT IF EXISTS documents_status_check,
  ADD  CONSTRAINT documents_status_check
    CHECK (status IN (
      'draft','finalized','archived',
      'under_review','approved','rejected',
      'scheduled','published','superseded','obsolete'
    ));

ALTER TABLE documents
  ADD COLUMN IF NOT EXISTS effective_from         TIMESTAMPTZ,
  ADD COLUMN IF NOT EXISTS effective_to           TIMESTAMPTZ,
  ADD COLUMN IF NOT EXISTS revision_number        INT NOT NULL DEFAULT 1,
  ADD COLUMN IF NOT EXISTS revision_version       INT NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS locked_at              TIMESTAMPTZ,
  ADD COLUMN IF NOT EXISTS content_hash_at_submit TEXT;

CREATE UNIQUE INDEX IF NOT EXISTS ux_documents_v2_cd_revision
  ON documents (controlled_document_id, revision_number)
  WHERE controlled_document_id IS NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS ux_documents_v2_cd_active
  ON documents (controlled_document_id)
  WHERE controlled_document_id IS NOT NULL
    AND status IN ('draft','under_review','approved','rejected','scheduled');

COMMIT;
```

- [ ] **Step 2: Run pre-Spec-2 integration suite against new CHECK**

```bash
rtk go test ./internal/modules/documents_v2/... -run Integration -count=1
```
Expected: all pre-Spec-2 finalize/archive tests still pass (CHECK superset admits their writes).

- [ ] **Step 3: Commit**

```bash
rtk git add migrations/0131_documents_v2_state_columns.sql
rtk git commit -m "feat(spec2/phase1): documents state columns + superset CHECK (0131)"
```

---

### Task 1.3 — Migration 0132: legacy status remap (one-shot)

**Goal:** Idempotent remap of every `documents` row where `status='finalized'` → `published` (preserve `finalized_at` into `effective_from`) and `status='archived'` → `obsolete` (preserve `archived_at` into `effective_to`). Emit one `governance_events` row per remapped document with payload `{from, to, remapped_at}`.

**Files:** Create `migrations/0132_documents_v2_legacy_remap.sql`.

**Acceptance Criteria:**
- [ ] Zero rows with `status='finalized'` or `status='archived'` after migration.
- [ ] Each remapped row has non-NULL `effective_from` (for formerly-finalized) or `effective_to` (for formerly-archived).
- [ ] Governance event count equals total remapped documents.
- [ ] Migration idempotent (re-run produces zero additional writes and no duplicate governance events).

**Steps:**

- [ ] **Step 1: Write migration**

```sql
-- migrations/0132_documents_v2_legacy_remap.sql
-- Spec 2 Phase 1. Runs against superset CHECK (0131).

BEGIN;

WITH remap AS (
  UPDATE documents
     SET status         = 'published',
         effective_from = COALESCE(effective_from, finalized_at, updated_at)
   WHERE status = 'finalized'
   RETURNING id, tenant_id, created_by
)
INSERT INTO governance_events
  (tenant_id, event_type, actor_user_id, resource_type, resource_id, reason, payload_json)
SELECT tenant_id,
       'legacy_status_remap',
       COALESCE(created_by, 'system:spec2-migration'),
       'document_v2',
       id::TEXT,
       'Spec 2 legacy collapse: finalized -> published',
       jsonb_build_object('from','finalized','to','published','remapped_at',now())
  FROM remap
 WHERE NOT EXISTS (
   SELECT 1 FROM governance_events ge
    WHERE ge.resource_type = 'document_v2'
      AND ge.resource_id   = remap.id::TEXT
      AND ge.event_type    = 'legacy_status_remap'
      AND ge.payload_json->>'from' = 'finalized'
 );

WITH remap AS (
  UPDATE documents
     SET status       = 'obsolete',
         effective_to = COALESCE(effective_to, archived_at, updated_at)
   WHERE status = 'archived'
   RETURNING id, tenant_id, created_by
)
INSERT INTO governance_events
  (tenant_id, event_type, actor_user_id, resource_type, resource_id, reason, payload_json)
SELECT tenant_id,
       'legacy_status_remap',
       COALESCE(created_by, 'system:spec2-migration'),
       'document_v2',
       id::TEXT,
       'Spec 2 legacy collapse: archived -> obsolete',
       jsonb_build_object('from','archived','to','obsolete','remapped_at',now())
  FROM remap
 WHERE NOT EXISTS (
   SELECT 1 FROM governance_events ge
    WHERE ge.resource_type = 'document_v2'
      AND ge.resource_id   = remap.id::TEXT
      AND ge.event_type    = 'legacy_status_remap'
      AND ge.payload_json->>'from' = 'archived'
 );

COMMIT;
```

- [ ] **Step 2: Verify zero legacy rows**

```bash
docker exec metaldocs-db psql -U metaldocs -d metaldocs -c \
  "SELECT status, COUNT(*) FROM documents WHERE status IN ('finalized','archived') GROUP BY status;"
```
Expected: zero rows.

- [ ] **Step 3: Commit**

```bash
rtk git add migrations/0132_documents_v2_legacy_remap.sql
rtk git commit -m "feat(spec2/phase1): legacy status remap (0132)"
```

---

### Task 1.4 — Migration 0133: legal-transition BEFORE UPDATE trigger

**Goal:** Install `trg_documents_v2_legal_transition`. Since legacy rows are remapped (0132), the trigger only needs to cover the 7-state graph. CHECK still admits `finalized`/`archived` (superset) so dormant legacy writes from not-yet-updated pre-Spec-2 code paths are handled: trigger allows `finalized → finalized` (no-op) and `archived → archived` (no-op) but rejects any transition involving legacy states — forcing any caller to adopt the new graph before writing.

**Files:** Create `migrations/0133_documents_v2_transition_trigger.sql`.

**Acceptance Criteria:**
- [ ] `enforce_document_transition()` function exists.
- [ ] Trigger `trg_documents_v2_legal_transition` BEFORE UPDATE installed.
- [ ] Illegal transition `draft → published` raises `check_violation`.
- [ ] Compat transitions permitted: `draft → finalized`, `finalized → archived` (pre-Spec-2 code keeps running through Phases 1–4).
- [ ] Cross-graph transition `finalized → published` rejected (must route through Spec 2 machinery; remap-only path in 0132).
- [ ] Compat window closed in Phase 5: this trigger is replaced with strict 7-state-only function.

**Steps:**

- [ ] **Step 1: Write migration**

```sql
-- migrations/0133_documents_v2_transition_trigger.sql

BEGIN;

CREATE OR REPLACE FUNCTION enforce_document_transition() RETURNS trigger AS $$
BEGIN
  IF OLD.status IS DISTINCT FROM NEW.status THEN
    IF NOT (
      -- Spec 2 graph
      (OLD.status = 'draft'        AND NEW.status =  'under_review') OR
      (OLD.status = 'under_review' AND NEW.status IN ('approved','rejected')) OR
      (OLD.status = 'rejected'     AND NEW.status =  'draft') OR
      (OLD.status = 'approved'     AND NEW.status IN ('published','scheduled','draft')) OR
      (OLD.status = 'scheduled'    AND NEW.status IN ('published','draft')) OR
      (OLD.status = 'published'    AND NEW.status IN ('superseded','obsolete')) OR
      (OLD.status = 'superseded'   AND NEW.status =  'obsolete') OR
      -- Compat window (Phase 1..4): legacy pre-Spec-2 writes. Removed in Phase 5.
      (OLD.status = 'draft'        AND NEW.status =  'finalized') OR
      (OLD.status = 'finalized'    AND NEW.status =  'archived')
    ) THEN
      RAISE EXCEPTION 'illegal status transition % -> %', OLD.status, NEW.status
        USING ERRCODE = 'check_violation';
    END IF;
  END IF;
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_documents_v2_legal_transition ON documents;
CREATE TRIGGER trg_documents_v2_legal_transition
  BEFORE UPDATE ON documents
  FOR EACH ROW EXECUTE FUNCTION enforce_document_transition();

COMMIT;
```

- [ ] **Step 2: Probe**

```bash
docker exec metaldocs-db psql -U metaldocs -d metaldocs -c \
  "BEGIN; UPDATE documents SET status='published' WHERE status='draft' RETURNING id; ROLLBACK;" 2>&1 | grep "illegal status transition"
```
Expected: `ERROR: illegal status transition draft -> published`.

- [ ] **Step 3: Commit**

```bash
rtk git add migrations/0133_documents_v2_transition_trigger.sql
rtk git commit -m "feat(spec2/phase1): legal-transition trigger (0133)"
```

---

### Task 1.5 — Migration 0134: `approval_routes` + `approval_route_stages`

**Goal:** Per-profile route config.

**Files:** Create `migrations/0134_approval_routes.sql`.

**Acceptance Criteria:**
- [ ] Both tables exist with composite FK to `metaldocs.document_profiles(tenant_id, code)`.
- [ ] UNIQUE `(tenant_id, profile_code)` on routes; UNIQUE `(route_id, stage_order)` on stages.
- [ ] `quorum_m_consistent` CHECK enforced.

**Steps:**

- [ ] **Step 1: Write migration** (SQL body identical to Round-1 0132 — see that task for verbatim SQL; only file name changed to `0134_approval_routes.sql`).

- [ ] **Step 2: Commit**

```bash
rtk git add migrations/0134_approval_routes.sql
rtk git commit -m "feat(spec2/phase1): approval_routes + stages (0134)"
```

---

### Task 1.6 — Migration 0135: approval instances + stages + signoffs (Codex Round 1 fixes #6, #7 applied)

**Goal:** Runtime approval machinery with:
- `approval_signoffs.approval_instance_id UUID NOT NULL` **denormalized** from the join path.
- `UNIQUE (approval_instance_id, actor_user_id)` as the hard uniqueness primitive for SoD cross-stage enforcement (defeats race).
- `actor_tenant_id` consistency trigger (`= approval_instances.tenant_id`).
- SoD trigger (author-self-sign block) + immutability trigger.
- Composite FKs to `metaldocs.iam_users(tenant_id, user_id)` added `NOT VALID`; validation deferred to Task 1.7 after backfill verification.

**Files:** Create `migrations/0135_approval_instances.sql`.

**Acceptance Criteria:**
- [ ] `approval_signoffs` has columns `approval_instance_id UUID NOT NULL`, `actor_user_id TEXT NOT NULL`, `actor_tenant_id UUID NOT NULL`.
- [ ] UNIQUE `(approval_instance_id, actor_user_id)` present.
- [ ] UNIQUE `(stage_instance_id, actor_user_id)` present (prevents double-sign in same stage).
- [ ] Trigger `trg_signoff_tenant_consistent` rejects INSERT if `actor_tenant_id <> approval_instances.tenant_id`.
- [ ] Trigger `trg_signoff_sod` rejects author-self-sign.
- [ ] Trigger `trg_signoff_immutable` rejects all UPDATEs.
- [ ] All FKs to `iam_users` added `NOT VALID`.

**Steps:**

- [ ] **Step 1: Write migration**

```sql
-- migrations/0135_approval_instances.sql
-- Spec 2 Phase 1 Codex-revised. Denormalized approval_instance_id on signoffs
-- plus UNIQUE(approval_instance_id, actor_user_id) replaces race-prone SoD trigger
-- for cross-stage duplicate check. All composite FKs to iam_users NOT VALID;
-- Task 1.7 validates after explicit backfill verification.

BEGIN;

CREATE TABLE IF NOT EXISTS approval_instances (
  id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id               UUID NOT NULL,
  document_v2_id          UUID NOT NULL REFERENCES documents(id),
  route_id                UUID NOT NULL REFERENCES approval_routes(id),
  route_version_snapshot  INT  NOT NULL,
  status                  TEXT NOT NULL
    CHECK (status IN ('in_progress','approved','rejected','cancelled')),
  submitted_by            TEXT NOT NULL,
  submitted_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
  completed_at            TIMESTAMPTZ,
  content_hash_at_submit  TEXT NOT NULL,
  idempotency_key         TEXT NOT NULL,
  UNIQUE (document_v2_id, idempotency_key)
);

ALTER TABLE approval_instances
  ADD CONSTRAINT approval_instances_submitted_by_tenant_fkey
    FOREIGN KEY (tenant_id, submitted_by)
      REFERENCES metaldocs.iam_users (tenant_id, user_id)
    NOT VALID;

CREATE UNIQUE INDEX IF NOT EXISTS ux_approval_instances_active
  ON approval_instances (document_v2_id)
  WHERE status = 'in_progress';

CREATE INDEX IF NOT EXISTS ix_approval_instances_tenant_doc
  ON approval_instances (tenant_id, document_v2_id);

CREATE TABLE IF NOT EXISTS approval_stage_instances (
  id                             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  approval_instance_id           UUID NOT NULL REFERENCES approval_instances(id) ON DELETE CASCADE,
  stage_order                    INT  NOT NULL CHECK (stage_order >= 1),
  name_snapshot                  TEXT NOT NULL,
  required_role_snapshot         TEXT NOT NULL,
  required_capability_snapshot   TEXT NOT NULL,
  area_code_snapshot             TEXT NOT NULL,
  quorum_snapshot                TEXT NOT NULL
    CHECK (quorum_snapshot IN ('any_1_of','all_of','m_of_n')),
  quorum_m_snapshot              INT,
  on_eligibility_drift_snapshot  TEXT NOT NULL
    CHECK (on_eligibility_drift_snapshot IN ('reduce_quorum','fail_stage','keep_snapshot')),
  eligible_actor_ids             JSONB NOT NULL,
  effective_denominator          INT,
  status                         TEXT NOT NULL
    CHECK (status IN ('pending','active','completed','skipped','rejected_here')),
  opened_at                      TIMESTAMPTZ,
  completed_at                   TIMESTAMPTZ,
  UNIQUE (approval_instance_id, stage_order),
  -- Anchors composite FK from approval_signoffs(stage_instance_id, approval_instance_id)
  -- so a signoff cannot reference a stage from a different instance.
  UNIQUE (id, approval_instance_id)
);

CREATE INDEX IF NOT EXISTS ix_stage_instances_active
  ON approval_stage_instances (approval_instance_id, stage_order)
  WHERE status = 'active';

CREATE INDEX IF NOT EXISTS ix_stage_instances_eligible_actors
  ON approval_stage_instances USING GIN (eligible_actor_ids);

CREATE TABLE IF NOT EXISTS approval_signoffs (
  id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  approval_instance_id  UUID NOT NULL REFERENCES approval_instances(id),
  stage_instance_id     UUID NOT NULL REFERENCES approval_stage_instances(id),
  actor_user_id         TEXT NOT NULL,
  actor_tenant_id       UUID NOT NULL,
  decision              TEXT NOT NULL CHECK (decision IN ('approve','reject')),
  comment               TEXT,
  signed_at             TIMESTAMPTZ NOT NULL DEFAULT now(),
  signature_method      TEXT NOT NULL,
  signature_payload     JSONB NOT NULL,
  content_hash          TEXT NOT NULL,
  UNIQUE (approval_instance_id, actor_user_id),
  UNIQUE (stage_instance_id, actor_user_id),
  -- Codex Round 2 fix #3: hard-bind stage to instance so signoff cannot straddle instances.
  CONSTRAINT approval_signoffs_stage_matches_instance
    FOREIGN KEY (stage_instance_id, approval_instance_id)
    REFERENCES approval_stage_instances (id, approval_instance_id)
);

ALTER TABLE approval_signoffs
  ADD CONSTRAINT approval_signoffs_actor_tenant_fkey
    FOREIGN KEY (actor_tenant_id, actor_user_id)
      REFERENCES metaldocs.iam_users (tenant_id, user_id)
    NOT VALID;

CREATE INDEX IF NOT EXISTS ix_signoffs_stage
  ON approval_signoffs (stage_instance_id);

-- Tenant-consistency trigger: actor_tenant_id must equal approval_instances.tenant_id.
CREATE OR REPLACE FUNCTION enforce_signoff_tenant_consistent()
  RETURNS trigger AS $$
DECLARE
  instance_tenant UUID;
BEGIN
  SELECT tenant_id INTO instance_tenant
    FROM public.approval_instances
   WHERE id = NEW.approval_instance_id;

  IF instance_tenant IS DISTINCT FROM NEW.actor_tenant_id THEN
    RAISE EXCEPTION 'cross-tenant signoff rejected (instance tenant %, actor tenant %)',
                    instance_tenant, NEW.actor_tenant_id
      USING ERRCODE = 'check_violation';
  END IF;

  RETURN NEW;
END;
$$ LANGUAGE plpgsql
   SET search_path = pg_catalog, pg_temp;

DROP TRIGGER IF EXISTS trg_signoff_tenant_consistent ON approval_signoffs;
CREATE TRIGGER trg_signoff_tenant_consistent
  BEFORE INSERT ON approval_signoffs
  FOR EACH ROW EXECUTE FUNCTION enforce_signoff_tenant_consistent();

-- SoD: author-self-sign block. Cross-stage duplicate is handled by
-- UNIQUE(approval_instance_id, actor_user_id) — not this trigger.
CREATE OR REPLACE FUNCTION enforce_signoff_sod() RETURNS trigger AS $$
DECLARE
  author_id TEXT;
BEGIN
  SELECT d.created_by INTO author_id
    FROM public.approval_instances i
    JOIN public.documents d ON d.id = i.document_v2_id
   WHERE i.id = NEW.approval_instance_id;

  IF NEW.actor_user_id = author_id THEN
    RAISE EXCEPTION 'SoD: author cannot sign own revision'
      USING ERRCODE = 'check_violation';
  END IF;

  RETURN NEW;
END;
$$ LANGUAGE plpgsql
   SET search_path = pg_catalog, pg_temp;

DROP TRIGGER IF EXISTS trg_signoff_sod ON approval_signoffs;
CREATE TRIGGER trg_signoff_sod
  BEFORE INSERT ON approval_signoffs
  FOR EACH ROW EXECUTE FUNCTION enforce_signoff_sod();

-- Immutability.
CREATE OR REPLACE FUNCTION reject_signoff_update() RETURNS trigger AS $$
BEGIN
  RAISE EXCEPTION 'approval_signoffs rows are immutable'
    USING ERRCODE = 'check_violation';
END;
$$ LANGUAGE plpgsql
   SET search_path = pg_catalog, pg_temp;

DROP TRIGGER IF EXISTS trg_signoff_immutable ON approval_signoffs;
CREATE TRIGGER trg_signoff_immutable
  BEFORE UPDATE ON approval_signoffs
  FOR EACH ROW EXECUTE FUNCTION reject_signoff_update();

COMMIT;
```

- [ ] **Step 2: Commit**

```bash
rtk git add migrations/0135_approval_instances.sql
rtk git commit -m "feat(spec2/phase1): approval runtime + tenant/SoD/immutability triggers (0135)"
```

---

### Task 1.7 — Migration 0136: `user_process_areas` hardening (NOT VALID FKs + explicit backfill + VALIDATE)

**Goal:** Add `revoked_by`, temporal CHECK, partial unique, no-DELETE, identity-immutable triggers. Composite FKs to `iam_users` added **NOT VALID**, then explicit backfill verification, then VALIDATE CONSTRAINT — protecting against pre-existing tenant mismatch.

**Files:** Create `migrations/0136_user_process_areas_hardening.sql`.

**Acceptance Criteria:**
- [ ] `revoked_by TEXT` + `revoked_by_required_when_revoked` + `effective_interval_valid` CHECKs.
- [ ] Both composite FKs added with `NOT VALID`, then `VALIDATE CONSTRAINT` after backfill check passes.
- [ ] Backfill verification DO-block raises exception if any row fails FK — migration aborts rather than creating invalid constraint.
- [ ] Triggers `trg_user_process_areas_no_delete` and `trg_user_process_areas_update_contract` present.
- [ ] Partial unique `ux_user_process_areas_single_active`.

**Steps:**

- [ ] **Step 1: Write migration**

```sql
-- migrations/0136_user_process_areas_hardening.sql

BEGIN;

ALTER TABLE user_process_areas
  ADD COLUMN IF NOT EXISTS revoked_by TEXT;

-- Codex Round 2 fix #4: pre-existing revoked rows may lack revoked_by.
-- Backfill FIRST (system:legacy sentinel), then enforce CHECK.
UPDATE user_process_areas
   SET revoked_by = 'system:legacy'
 WHERE effective_to IS NOT NULL
   AND revoked_by   IS NULL;

ALTER TABLE user_process_areas
  DROP CONSTRAINT IF EXISTS revoked_by_required_when_revoked,
  ADD  CONSTRAINT revoked_by_required_when_revoked
    CHECK ((effective_to IS NULL AND revoked_by IS NULL)
        OR (effective_to IS NOT NULL AND revoked_by IS NOT NULL))
    NOT VALID;

ALTER TABLE user_process_areas
  VALIDATE CONSTRAINT revoked_by_required_when_revoked;

ALTER TABLE user_process_areas
  DROP CONSTRAINT IF EXISTS effective_interval_valid,
  ADD  CONSTRAINT effective_interval_valid
    CHECK (effective_to IS NULL OR effective_to > effective_from);

CREATE UNIQUE INDEX IF NOT EXISTS ux_user_process_areas_single_active
  ON user_process_areas (tenant_id, user_id, area_code, role)
  WHERE effective_to IS NULL;

-- NOT VALID FKs first. VALIDATE after explicit verification below.
ALTER TABLE user_process_areas
  DROP CONSTRAINT IF EXISTS user_process_areas_granted_by_same_tenant,
  DROP CONSTRAINT IF EXISTS user_process_areas_revoked_by_same_tenant,
  ADD  CONSTRAINT user_process_areas_granted_by_same_tenant
    FOREIGN KEY (tenant_id, granted_by)
      REFERENCES metaldocs.iam_users (tenant_id, user_id)
    NOT VALID,
  ADD  CONSTRAINT user_process_areas_revoked_by_same_tenant
    FOREIGN KEY (tenant_id, revoked_by)
      REFERENCES metaldocs.iam_users (tenant_id, user_id)
    NOT VALID;

-- Explicit backfill verification. Raises if any row would fail FK — migration aborts.
DO $$
DECLARE
  missing_granted INT;
  missing_revoked INT;
BEGIN
  SELECT COUNT(*) INTO missing_granted
    FROM user_process_areas upa
    LEFT JOIN metaldocs.iam_users u
      ON u.tenant_id = upa.tenant_id AND u.user_id = upa.granted_by
   WHERE upa.granted_by IS NOT NULL AND u.user_id IS NULL;

  SELECT COUNT(*) INTO missing_revoked
    FROM user_process_areas upa
    LEFT JOIN metaldocs.iam_users u
      ON u.tenant_id = upa.tenant_id AND u.user_id = upa.revoked_by
   WHERE upa.revoked_by IS NOT NULL AND u.user_id IS NULL;

  IF missing_granted > 0 OR missing_revoked > 0 THEN
    RAISE EXCEPTION
      'FK backfill verification failed: % granted_by, % revoked_by rows lack matching iam_users. Remediate before VALIDATE.',
      missing_granted, missing_revoked;
  END IF;
END $$;

ALTER TABLE user_process_areas
  VALIDATE CONSTRAINT user_process_areas_granted_by_same_tenant,
  VALIDATE CONSTRAINT user_process_areas_revoked_by_same_tenant;

-- No-DELETE trigger.
CREATE OR REPLACE FUNCTION reject_user_process_areas_delete() RETURNS trigger AS $$
BEGIN
  RAISE EXCEPTION 'user_process_areas rows cannot be deleted (revoke via UPDATE effective_to)'
    USING ERRCODE = 'check_violation';
END;
$$ LANGUAGE plpgsql
   SET search_path = pg_catalog, pg_temp;

DROP TRIGGER IF EXISTS trg_user_process_areas_no_delete ON user_process_areas;
CREATE TRIGGER trg_user_process_areas_no_delete
  BEFORE DELETE ON user_process_areas
  FOR EACH ROW EXECUTE FUNCTION reject_user_process_areas_delete();

-- Identity-immutable + no un-revoke trigger.
CREATE OR REPLACE FUNCTION enforce_user_process_areas_update_contract() RETURNS trigger AS $$
BEGIN
  IF NEW.tenant_id      IS DISTINCT FROM OLD.tenant_id      OR
     NEW.user_id        IS DISTINCT FROM OLD.user_id        OR
     NEW.area_code      IS DISTINCT FROM OLD.area_code      OR
     NEW.role           IS DISTINCT FROM OLD.role           OR
     NEW.effective_from IS DISTINCT FROM OLD.effective_from OR
     NEW.granted_by     IS DISTINCT FROM OLD.granted_by     THEN
    RAISE EXCEPTION 'identity columns are immutable on user_process_areas'
      USING ERRCODE = 'check_violation';
  END IF;
  IF OLD.effective_to IS NOT NULL AND NEW.effective_to IS NULL THEN
    RAISE EXCEPTION 'cannot un-revoke membership (re-grant creates new row)'
      USING ERRCODE = 'check_violation';
  END IF;
  RETURN NEW;
END;
$$ LANGUAGE plpgsql
   SET search_path = pg_catalog, pg_temp;

DROP TRIGGER IF EXISTS trg_user_process_areas_update_contract ON user_process_areas;
CREATE TRIGGER trg_user_process_areas_update_contract
  BEFORE UPDATE ON user_process_areas
  FOR EACH ROW EXECUTE FUNCTION enforce_user_process_areas_update_contract();

COMMIT;
```

- [ ] **Step 2: Commit**

```bash
rtk git add migrations/0136_user_process_areas_hardening.sql
rtk git commit -m "feat(spec2/phase1): user_process_areas hardening + NOT VALID/VALIDATE FKs (0136)"
```

---

### Task 1.8 — Migration 0137: DB roles + schema lockdown + SECURITY DEFINER (Codex #8 + #9 fixes)

**Goal:** Install NOINHERIT role boundary + schema lockdown + SECURITY DEFINER functions. Fixes from Round 1:
- All role DDL wrapped in `DO $$` guards against `pg_roles`.
- `SECURITY DEFINER` functions use `SET search_path = pg_catalog, pg_temp` (not `pg_catalog, public`).
- Every object reference inside function body is schema-qualified (`public.user_process_areas`, `metaldocs.iam_users`).
- **No REVOKE on `metaldocs_app`'s DML privileges on `user_process_areas`** — deferred to Phase 6 co-release with IAM service update. Phase 1 leaves existing DML path intact.

**Files:** Create `migrations/0137_db_roles_security_definer.sql`.

**Acceptance Criteria:**
- [ ] `metaldocs_security_owner` (NOLOGIN) + `metaldocs_membership_writer` (NOLOGIN NOINHERIT) created only if absent.
- [ ] `metaldocs_app` is NOINHERIT.
- [ ] Schema `public` CREATE lockdown applied.
- [ ] Functions exist with `prosecdef=t`, `proconfig` containing `search_path=pg_catalog,pg_temp`, `proowner=metaldocs_security_owner`.
- [ ] EXECUTE revoked from PUBLIC + `metaldocs_app`; granted to writer; writer granted to app.
- [ ] `metaldocs_app` still has INSERT/UPDATE/DELETE on `user_process_areas` (unchanged from Spec 1) — deferred revoke happens in Phase 6.

**Steps:**

- [ ] **Step 1: Write migration**

```sql
-- migrations/0137_db_roles_security_definer.sql
-- Codex Round 1 fixes: DO $$ guards, pg_temp search_path, schema-qualified refs.
-- DML revoke on user_process_areas deferred to Phase 6 (co-release with IAM cutover).

BEGIN;

DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'metaldocs_security_owner') THEN
    CREATE ROLE metaldocs_security_owner NOLOGIN;
  END IF;
  IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'metaldocs_membership_writer') THEN
    CREATE ROLE metaldocs_membership_writer NOLOGIN NOINHERIT;
  END IF;
  IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'metaldocs_app') THEN
    EXECUTE 'ALTER ROLE metaldocs_app NOINHERIT';
  END IF;
END $$;

-- Codex Round 2 fix #5: explicit USAGE on metaldocs schema for SECURITY DEFINER owner
-- and the membership_writer role. In hardened envs PUBLIC may lack USAGE here, in which
-- case table-level GRANTs alone are not enough.
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_namespace WHERE nspname='metaldocs') THEN
    EXECUTE 'GRANT USAGE ON SCHEMA metaldocs TO metaldocs_security_owner';
    EXECUTE 'GRANT USAGE ON SCHEMA metaldocs TO metaldocs_membership_writer';
  END IF;
END $$;

-- Owner needs read on iam_users and write on user_process_areas (function body).
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_tables WHERE schemaname='metaldocs' AND tablename='iam_users') THEN
    EXECUTE 'GRANT SELECT ON metaldocs.iam_users TO metaldocs_security_owner';
  END IF;
  EXECUTE 'GRANT SELECT, INSERT, UPDATE ON public.user_process_areas TO metaldocs_security_owner';
END $$;

-- Schema public CREATE lockdown (safe to repeat).
REVOKE CREATE ON SCHEMA public FROM PUBLIC;
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'metaldocs_app') THEN
    EXECUTE 'REVOKE CREATE ON SCHEMA public FROM metaldocs_app';
    EXECUTE 'GRANT  USAGE  ON SCHEMA public TO metaldocs_app';
  END IF;
  IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'metaldocs_readonly') THEN
    EXECUTE 'REVOKE CREATE ON SCHEMA public FROM metaldocs_readonly';
    EXECUTE 'GRANT  USAGE  ON SCHEMA public TO metaldocs_readonly';
  END IF;
  EXECUTE 'REVOKE CREATE ON SCHEMA public FROM metaldocs_membership_writer';
  EXECUTE 'GRANT  USAGE  ON SCHEMA public TO metaldocs_membership_writer';
  IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'metaldocs_migrator') THEN
    EXECUTE 'GRANT  CREATE ON SCHEMA public TO metaldocs_migrator';
  END IF;
  EXECUTE 'GRANT  CREATE ON SCHEMA public TO metaldocs_security_owner';
END $$;

CREATE OR REPLACE FUNCTION public.grant_area_membership(
  _tenant_id   UUID,
  _user_id     TEXT,
  _area_code   TEXT,
  _role        TEXT,
  _granted_by  TEXT
) RETURNS UUID AS $$
DECLARE
  session_actor  TEXT := pg_catalog.current_setting('metaldocs.actor_id', true);
  session_cap    TEXT := pg_catalog.current_setting('metaldocs.verified_capability', true);
  actor_tenant   UUID;
BEGIN
  IF session_actor IS NULL OR session_actor = '' OR session_actor IS DISTINCT FROM _granted_by THEN
    RAISE EXCEPTION 'session actor context missing or mismatched'
      USING ERRCODE = 'insufficient_privilege';
  END IF;
  IF session_cap IS NULL OR session_cap <> 'workflow.route.edit' THEN
    RAISE EXCEPTION 'session capability context missing or wrong'
      USING ERRCODE = 'insufficient_privilege';
  END IF;
  SELECT tenant_id INTO actor_tenant
    FROM metaldocs.iam_users WHERE user_id = _granted_by;
  IF actor_tenant IS DISTINCT FROM _tenant_id THEN
    RAISE EXCEPTION 'granted_by must belong to same tenant'
      USING ERRCODE = 'check_violation';
  END IF;
  IF NOT EXISTS (
    SELECT 1 FROM metaldocs.iam_users
     WHERE user_id = _granted_by AND deactivated_at IS NULL
  ) THEN
    RAISE EXCEPTION 'granted_by must be active user'
      USING ERRCODE = 'check_violation';
  END IF;
  INSERT INTO public.user_process_areas
    (user_id, tenant_id, area_code, role, effective_from, effective_to, granted_by, revoked_by)
    VALUES (_user_id, _tenant_id, _area_code, _role,
            pg_catalog.clock_timestamp(), NULL, _granted_by, NULL);
  RETURN pg_catalog.gen_random_uuid();
END;
$$ LANGUAGE plpgsql
   SECURITY DEFINER
   SET search_path = pg_catalog, pg_temp;

CREATE OR REPLACE FUNCTION public.revoke_area_membership(
  _tenant_id   UUID,
  _user_id     TEXT,
  _area_code   TEXT,
  _role        TEXT,
  _revoked_by  TEXT
) RETURNS UUID AS $$
DECLARE
  session_actor  TEXT := pg_catalog.current_setting('metaldocs.actor_id', true);
  session_cap    TEXT := pg_catalog.current_setting('metaldocs.verified_capability', true);
  actor_tenant   UUID;
  rows_affected  INT;
BEGIN
  IF session_actor IS NULL OR session_actor = '' OR session_actor IS DISTINCT FROM _revoked_by THEN
    RAISE EXCEPTION 'session actor context missing or mismatched'
      USING ERRCODE = 'insufficient_privilege';
  END IF;
  IF session_cap IS NULL OR session_cap <> 'workflow.route.edit' THEN
    RAISE EXCEPTION 'session capability context missing or wrong'
      USING ERRCODE = 'insufficient_privilege';
  END IF;
  SELECT tenant_id INTO actor_tenant
    FROM metaldocs.iam_users WHERE user_id = _revoked_by;
  IF actor_tenant IS DISTINCT FROM _tenant_id THEN
    RAISE EXCEPTION 'revoked_by must belong to same tenant'
      USING ERRCODE = 'check_violation';
  END IF;
  IF NOT EXISTS (
    SELECT 1 FROM metaldocs.iam_users
     WHERE user_id = _revoked_by AND deactivated_at IS NULL
  ) THEN
    RAISE EXCEPTION 'revoked_by must be active user'
      USING ERRCODE = 'check_violation';
  END IF;
  UPDATE public.user_process_areas
     SET effective_to = pg_catalog.clock_timestamp(),
         revoked_by   = _revoked_by
   WHERE tenant_id    = _tenant_id
     AND user_id      = _user_id
     AND area_code    = _area_code
     AND role         = _role
     AND effective_to IS NULL;
  GET DIAGNOSTICS rows_affected = ROW_COUNT;
  IF rows_affected = 0 THEN
    RAISE EXCEPTION 'no active membership to revoke'
      USING ERRCODE = 'no_data_found';
  END IF;
  RETURN pg_catalog.gen_random_uuid();
END;
$$ LANGUAGE plpgsql
   SECURITY DEFINER
   SET search_path = pg_catalog, pg_temp;

ALTER FUNCTION public.grant_area_membership(UUID, TEXT, TEXT, TEXT, TEXT)
  OWNER TO metaldocs_security_owner;
ALTER FUNCTION public.revoke_area_membership(UUID, TEXT, TEXT, TEXT, TEXT)
  OWNER TO metaldocs_security_owner;

REVOKE EXECUTE ON FUNCTION public.grant_area_membership(UUID, TEXT, TEXT, TEXT, TEXT),
                        public.revoke_area_membership(UUID, TEXT, TEXT, TEXT, TEXT)
  FROM PUBLIC;

DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'metaldocs_app') THEN
    EXECUTE 'REVOKE EXECUTE ON FUNCTION public.grant_area_membership(UUID,TEXT,TEXT,TEXT,TEXT), public.revoke_area_membership(UUID,TEXT,TEXT,TEXT,TEXT) FROM metaldocs_app';
  END IF;
END $$;

GRANT EXECUTE ON FUNCTION public.grant_area_membership(UUID, TEXT, TEXT, TEXT, TEXT),
                       public.revoke_area_membership(UUID, TEXT, TEXT, TEXT, TEXT)
  TO metaldocs_membership_writer;

DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'metaldocs_app') THEN
    EXECUTE 'GRANT metaldocs_membership_writer TO metaldocs_app';
  END IF;
END $$;

-- Phase 6 TODO: REVOKE INSERT, UPDATE, DELETE ON user_process_areas FROM metaldocs_app;
-- Left intentionally here — Phase 6 migration co-releases with IAM service switching to
-- SECURITY DEFINER call path. Phase 1 alone must not strand the running IAM service.

COMMIT;
```

- [ ] **Step 2: Commit**

```bash
rtk git add migrations/0137_db_roles_security_definer.sql
rtk git commit -m "feat(spec2/phase1): DB roles + lockdown + SECURITY DEFINER hardened (0137)"
```

---

### Task 1.9 — Migration 0138: GRANTs on new approval tables (Haiku)

**Goal:** `metaldocs_app` SELECT/INSERT/UPDATE on routes/stages/instances/stage_instances; SELECT/INSERT only on signoffs; `metaldocs_readonly` SELECT on all 5.

**Files:** Create `migrations/0138_grants_approval_tables.sql`.

**Steps:**

- [ ] **Step 1: Write file** (identical body to Round-1 0136 — renumbered).

```sql
-- migrations/0138_grants_approval_tables.sql

BEGIN;

DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'metaldocs_app') THEN
    EXECUTE 'GRANT SELECT, INSERT, UPDATE ON approval_routes, approval_route_stages, approval_instances, approval_stage_instances TO metaldocs_app';
    EXECUTE 'GRANT SELECT, INSERT ON approval_signoffs TO metaldocs_app';
  END IF;
  IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'metaldocs_readonly') THEN
    EXECUTE 'GRANT SELECT ON approval_routes, approval_route_stages, approval_instances, approval_stage_instances, approval_signoffs TO metaldocs_readonly';
  END IF;
END $$;

COMMIT;
```

- [ ] **Step 2: Commit**

```bash
rtk git add migrations/0138_grants_approval_tables.sql
rtk git commit -m "feat(spec2/phase1): GRANTs on approval tables (0138)"
```

---

### Task 1.10 — Migration 0139: governance_events caps bump (unique partial index + ON CONFLICT)

**Goal:** Idempotent capability-version-bump audit row. Uniqueness enforced by a unique partial index, not by a `WHERE NOT EXISTS` predicate.

**Files:** Create `migrations/0139_governance_events_caps_bump.sql`.

**Acceptance Criteria:**
- [ ] Unique partial index `ux_governance_events_caps_bump_spec_version` on `(event_type, (payload_json->>'spec'), (payload_json->>'to')) WHERE event_type='role_capabilities_version_bump'`.
- [ ] One row present after migration; re-run produces no duplicate (`ON CONFLICT DO NOTHING`).

**Steps:**

- [ ] **Step 1: Write migration**

```sql
-- migrations/0139_governance_events_caps_bump.sql

BEGIN;

CREATE UNIQUE INDEX IF NOT EXISTS ux_governance_events_caps_bump_spec_version
  ON governance_events (event_type, (payload_json->>'spec'), (payload_json->>'to'))
  WHERE event_type = 'role_capabilities_version_bump';

INSERT INTO governance_events
  (tenant_id, event_type, actor_user_id, resource_type, resource_id, reason, payload_json)
VALUES
  ('ffffffff-ffff-ffff-ffff-ffffffffffff',
   'role_capabilities_version_bump',
   'system:spec2-migration',
   'role_capabilities',
   'global',
   'Spec 2 workflow.* capabilities added',
   jsonb_build_object('from', 1, 'to', 2, 'spec', 'spec-2-approval'))
ON CONFLICT (event_type, (payload_json->>'spec'), (payload_json->>'to'))
  WHERE event_type = 'role_capabilities_version_bump'
  DO NOTHING;

COMMIT;
```

- [ ] **Step 2: Commit**

```bash
rtk git add migrations/0139_governance_events_caps_bump.sql
rtk git commit -m "feat(spec2/phase1): caps bump audit with ON CONFLICT idempotency (0139)"
```

---

### Task 1.11 — Migration 0140: revision_version monotonic guard + inbox btree index (Codex #13)

**Goal:** Trigger enforcing `NEW.revision_version >= OLD.revision_version` on every `documents` UPDATE, preventing silent regressions. Btree index on `approval_instances (tenant_id, submitted_by, status, submitted_at DESC)` for inbox queries.

**Files:** Create `migrations/0140_revision_version_and_inbox_index.sql`.

**Acceptance Criteria:**
- [ ] Trigger `trg_documents_v2_revision_version_monotonic` BEFORE UPDATE installed.
- [ ] Trigger rejects any update where `NEW.revision_version < OLD.revision_version` with `check_violation`.
- [ ] Index `ix_approval_instances_inbox` present.

**Steps:**

- [ ] **Step 1: Write migration**

```sql
-- migrations/0140_revision_version_and_inbox_index.sql

BEGIN;

CREATE OR REPLACE FUNCTION enforce_revision_version_monotonic()
  RETURNS trigger AS $$
BEGIN
  IF NEW.revision_version < OLD.revision_version THEN
    RAISE EXCEPTION 'revision_version cannot decrease (OLD=%, NEW=%)',
                    OLD.revision_version, NEW.revision_version
      USING ERRCODE = 'check_violation';
  END IF;
  RETURN NEW;
END;
$$ LANGUAGE plpgsql
   SET search_path = pg_catalog, pg_temp;

DROP TRIGGER IF EXISTS trg_documents_v2_revision_version_monotonic ON documents;
CREATE TRIGGER trg_documents_v2_revision_version_monotonic
  BEFORE UPDATE ON documents
  FOR EACH ROW EXECUTE FUNCTION enforce_revision_version_monotonic();

CREATE INDEX IF NOT EXISTS ix_approval_instances_inbox
  ON approval_instances (tenant_id, submitted_by, status, submitted_at DESC);

COMMIT;
```

- [ ] **Step 2: Commit**

```bash
rtk git add migrations/0140_revision_version_and_inbox_index.sql
rtk git commit -m "feat(spec2/phase1): revision_version monotonic guard + inbox index (0140)"
```

---

### Task 1.12 — Phase 1 full-stack smoke (expanded)

**Goal:** Clean-DB probes for every trigger + function + superset CHECK expectation.

**Files:** Create `scripts/spec2_phase1_smoke.sh`.

**Acceptance Criteria:**
- [ ] Clean rebuild logs list all 11 migrations, no ERROR.
- [ ] Probe set passes:
  - Illegal transition `draft → published` rejected.
  - Legacy transition `finalized → published` rejected (forcing new graph).
  - `UPDATE documents SET revision_version = revision_version - 1` rejected.
  - `DELETE FROM user_process_areas` rejected.
  - `UPDATE approval_signoffs` rejected.
  - `grant_area_membership` without SET LOCAL GUCs rejected with `session actor context`.

**Steps:**

- [ ] **Step 1: Clean rebuild + migration log**

```bash
docker compose down -v && docker compose up -d db && sleep 8
docker compose logs db --tail 200 | grep -E "013[0-9]|0140"
```

- [ ] **Step 2: Write smoke script**

```bash
#!/usr/bin/env bash
set -euo pipefail
PSQL="docker exec metaldocs-db psql -U metaldocs -d metaldocs -v ON_ERROR_STOP=0"

expect_error() {
  local label="$1"; local sql="$2"; local needle="$3"
  if $PSQL -c "$sql" 2>&1 | grep -q "$needle"; then
    echo "PASS: $label"
  else
    echo "FAIL: $label (expected '$needle')"; exit 1
  fi
}

expect_error "illegal transition draft->published" \
  "UPDATE documents SET status='published' WHERE status='draft' RETURNING id;" \
  "illegal status transition"

expect_error "cross-graph finalized->published still rejected (R2-1)" \
  "INSERT INTO documents (id,tenant_id,template_version_id,name,status,form_data_json,created_by) VALUES (gen_random_uuid(),'ffffffff-ffff-ffff-ffff-ffffffffffff',(SELECT id FROM template_versions LIMIT 1),'legacy-probe','draft','{}','probe');
   UPDATE documents SET status='finalized' WHERE name='legacy-probe';
   UPDATE documents SET status='published' WHERE name='legacy-probe' RETURNING id;" \
  "illegal status transition"

# R2-1 positive probe: compat transition draft->finalized must SUCCEED during Phases 1–4.
$PSQL -c "BEGIN; UPDATE documents SET status='finalized' WHERE status='draft' LIMIT 1; ROLLBACK;" \
  2>&1 | grep -qv "illegal status transition" && echo "PASS: compat draft->finalized allowed" || { echo "FAIL: compat broke"; exit 1; }

# R2-3 probe: signoff composite FK rejects cross-instance straddle.
expect_error "signoff cross-instance rejected (R2-3)" \
  "INSERT INTO approval_signoffs (approval_instance_id, stage_instance_id, actor_user_id, actor_tenant_id, decision, signature_method, signature_payload, content_hash) VALUES ('00000000-0000-0000-0000-000000000001','00000000-0000-0000-0000-000000000002','probe','ffffffff-ffff-ffff-ffff-ffffffffffff','approve','password','{}','sha');" \
  "violates foreign key constraint"

expect_error "revision_version monotonic" \
  "UPDATE documents SET revision_version = revision_version - 1 WHERE name='legacy-probe';" \
  "revision_version cannot decrease"

expect_error "user_process_areas DELETE blocked" \
  "DELETE FROM user_process_areas WHERE TRUE;" \
  "cannot be deleted"

expect_error "grant_area_membership needs context" \
  "BEGIN; SET LOCAL ROLE metaldocs_membership_writer;
   SELECT public.grant_area_membership('ffffffff-ffff-ffff-ffff-ffffffffffff','u','a','reviewer','u');
   ROLLBACK;" \
  "session actor context"

echo "All Phase 1 trigger probes passed."
```

- [ ] **Step 3: Run + commit**

```bash
bash scripts/spec2_phase1_smoke.sh
rtk git add scripts/spec2_phase1_smoke.sh
rtk git commit -m "chore(spec2/phase1): expanded smoke battery"
```

---

### Task 1.13 — Codex Round 2 OPERATIONS hardening pass

**Goal:** Send revised Phase 1 (migrations 0130–0140 + smoke) to Codex `gpt-5.3-codex` reasoning=high. Must APPROVE (or APPROVE_WITH_FIXES local-scope only). Any structural verdict → STOP, deliver plan with caveats per co-plan skill.

**Files:** Create `docs/superpowers/plans/reviews/phase-1-round-2.json`.

**Acceptance Criteria:**
- [ ] Codex verdict `APPROVE` or `APPROVE_WITH_FIXES` local-only.
- [ ] Any structural finding → stop Phase 2 work, escalate to user.
- [ ] Local fixes applied inline, verdict artifact committed.

**Steps:**

- [ ] **Step 1: Assemble payload** (migrations 0130–0140 + smoke + fix table from Phase 1 header).

- [ ] **Step 2: Call Codex** via `mcp__codex__codex`:
  - `model: "gpt-5.3-codex"`, `config.model_reasoning_effort: "high"`, `sandbox: "read-only"`, `approval-policy: "never"`.
  - Mode: OPERATIONS. Prompt states this is Round 2 of an `upgrade_required=true` plan, lists the 13 Round-1 issues and which tasks address each, asks for fresh-eyes verdict.

- [ ] **Step 3: Commit verdict**

```bash
rtk git add docs/superpowers/plans/reviews/phase-1-round-2.json
rtk git commit -m "chore(spec2/phase1): Codex Round 2 OPERATIONS verdict"
```

---

## Phase 2: Domain Layer (pure Go, no DB, no HTTP)

**Intent:** Encode Spec 2 rules as pure Go types and functions — state graph, route config, instance aggregate, signoff value object, quorum evaluation, SoD guards. No `database/sql`, no `http`, no I/O. Unit tests only. TDD: test file first, then implementation, then `go test` green.

**Codex review:** COVERAGE (at end of phase via Task 2.13).

**Opus review:** at phase end.

**Execution model:** Codex `gpt-5.3-codex` medium writes the Go. Controller runs `go test`. Opus reads diffs at phase end.

**Subpackage:** `internal/modules/documents_v2/approval/domain/`.

**Invariants this phase encodes (for Codex test generation):**

1. State transitions — full **8-state graph** (draft, under_review, approved, rejected, scheduled, published, superseded, obsolete). Legacy rejected at domain; DB compat separate.
2. Route stage order starts at 1 and is dense (no gaps).
3. Quorum policies: `any_1_of`, `all_of`, `m_of_n` with `1 ≤ m ≤ len(eligible)`.
4. Drift policy enum AND executable rule: `reduce_quorum`, `fail_stage`, `keep_snapshot` applied by `ApplyEligibilityDrift` (Task 2.7).
5. SoD: `actor ≠ author`; actor not in prior-stage signoffs.
6. Signoff immutable after construction — no setter.
7. Instance `revision_version` monotonic — `Instance.BumpRevisionVersion(next int) error` mirrors DB trigger (Task 2.3).
8. Effective denominator computed **inside domain** (`ComputeEffectiveDenominator` in quorum.go); callers never hand-compute.
9. Content hash normalized to lowercase in `NewSignoff` constructor; uppercase input accepted and canonicalized.

**Codex Round 1 verdict:** REJECT (4 structural, 2 local). Fixes applied inline:

| # | Scope | Fix |
|---|---|---|
| R1-1 | structural | New Task 2.7 `drift.go` + `ApplyEligibilityDrift` function |
| R1-2 | structural | `ComputeEffectiveDenominator` in quorum.go; callers pass eligible set, not int |
| R1-3 | structural | `Instance.RevisionVersion int` + `BumpRevisionVersion` monotonic guard in Task 2.3 |
| R1-4 | local | State tests cover **8×8 = 64** pairs; named subtests for scheduled/superseded/obsolete edges |
| R1-5 | structural | `Instance.SkipStage(reason)` makes `StageSkipped` reachable |
| R1-6 | local | `NewSignoff` lowercases hash; test asserts uppercase accepted + canonicalized |

Task numbering shifted: old 2.7 coverage gate → 2.8; old 2.8 Codex review → 2.9. New Task 2.7 = `drift.go`.

**Codex Round 2 verdict:** REJECT with 4 residuals (at 2-round cap per co-plan skill). All 4 applied inline:

| # | Scope | Fix |
|---|---|---|
| R2-1 | local | SkipStage-during-signing integration test in Task 2.8 |
| R2-2 | local | Drift→Quorum call-order contract test in Task 2.8 (fail_stage short-circuits) |
| R2-3 | local | revision_version cross-concern integration test in Task 2.8 (may defer gate to Phase 4 repo layer) |
| R2-4 | structural | `Instance.Cancel(reason)` method added to Task 2.3; `InstanceCancelled` now reachable |

**Caveats (review before Phase 2 execution):**
- R2-4 is structural, applied without third Codex round. Verify semantic against spec cancel paths.
- R2-3 may require `AdvanceStage(expectedVersion int)` signature change OR repository-layer OCC gate (Phase 4). Decide at Phase 2 kickoff.
- Content hash normalization strips case but NOT whitespace — inputs with surrounding whitespace still fail regex. Intentional strictness per Round 2 confirmation.

---

### Task 2.1 — `state.go`: state enum + legal transition table

**Files:** `internal/modules/documents_v2/approval/domain/state.go`, `state_test.go`.

**Acceptance Criteria:**
- [ ] `DocState` string-backed enum with **8 constants**: `StateDraft`, `StateUnderReview`, `StateApproved`, `StateRejected`, `StateScheduled`, `StatePublished`, `StateSuperseded`, `StateObsolete`.
- [ ] `IsLegalTransition(from, to DocState) bool` true **only** for Spec 2 graph (see 0133 trigger); no legacy states.
- [ ] `AllStates() []DocState` returns all 8 for exhaustive test matrix.
- [ ] `StateFromString(s string) (DocState, error)` rejects empty + unknown; legacy `"finalized"`/`"archived"` → `ErrLegacyStateRejected`.
- [ ] `String() string` returns canonical lowercase snake form.

**Steps:**

- [ ] **Step 1: Write test file first** — `state_test.go` with table-driven **8×8 = 64-pair** coverage. Named subtests asserting legality of: `approved→scheduled`, `scheduled→published`, `scheduled→draft`, `published→superseded`, `published→obsolete`, `superseded→obsolete`. Explicit test that `finalized → published` is illegal at domain level. `StateFromString("finalized")` returns `ErrLegacyStateRejected`. Self-transition `X→X` illegal for all 8.

- [ ] **Step 2: Implement `state.go`** to make tests pass.

- [ ] **Step 3: Verify**

```bash
rtk go test ./internal/modules/documents_v2/approval/domain/ -run TestState -v -count=1
```
Expected: all subtests PASS, coverage ≥95%.

- [ ] **Step 4: Commit**

```bash
rtk git add internal/modules/documents_v2/approval/domain/state.go internal/modules/documents_v2/approval/domain/state_test.go
rtk git commit -m "feat(spec2/phase2): DocState enum + legal-transition table"
```

---

### Task 2.2 — `route.go`: route config + stage + quorum/drift enums

**Files:** `internal/modules/documents_v2/approval/domain/route.go`, `route_test.go`.

**Acceptance Criteria:**
- [ ] `Route` struct: `ID`, `TenantID`, `ProfileCode`, `Version int`, `Stages []Stage`.
- [ ] `Stage` struct: `Order int`, `Name`, `RequiredRole`, `RequiredCapability`, `AreaCode`, `Quorum QuorumPolicy`, `QuorumM *int`, `OnEligibilityDrift DriftPolicy`.
- [ ] `QuorumPolicy` enum: `QuorumAny1Of`, `QuorumAllOf`, `QuorumMofN`.
- [ ] `DriftPolicy` enum: `DriftReduceQuorum`, `DriftFailStage`, `DriftKeepSnapshot`.
- [ ] `Route.Validate() error` enforces: ≥1 stage; stage_order dense starting at 1; `QuorumM` non-nil iff `Quorum == QuorumMofN`; `QuorumM ≥ 1`; unique stage names within route.

**Steps:**

- [ ] **Step 1: Write `route_test.go`** — cover: empty stages rejected, non-dense order rejected (e.g. [1,3]), m_of_n without M rejected, any_1_of with M set rejected, M<1 rejected, duplicate stage names rejected, happy-path 3-stage route accepted.

- [ ] **Step 2: Implement `route.go`**.

- [ ] **Step 3: Verify**

```bash
rtk go test ./internal/modules/documents_v2/approval/domain/ -run TestRoute -v -count=1
```

- [ ] **Step 4: Commit**

```bash
rtk git add internal/modules/documents_v2/approval/domain/route.go internal/modules/documents_v2/approval/domain/route_test.go
rtk git commit -m "feat(spec2/phase2): Route + Stage + quorum/drift policy enums"
```

---

### Task 2.3 — `instance.go`: instance aggregate + stage instance

**Files:** `internal/modules/documents_v2/approval/domain/instance.go`, `instance_test.go`.

**Acceptance Criteria:**
- [ ] `Instance` struct: `ID`, `TenantID`, `DocumentID`, `RouteID`, `RouteVersionSnapshot int`, `Status InstanceStatus`, `SubmittedBy`, `SubmittedAt`, `CompletedAt *time.Time`, `ContentHashAtSubmit`, `IdempotencyKey`, `RevisionVersion int`, `Stages []StageInstance`.
- [ ] `InstanceStatus` enum: `InstanceInProgress`, `InstanceApproved`, `InstanceRejected`, `InstanceCancelled`.
- [ ] `StageInstance` struct with snapshot fields mirroring 0135 DB columns; `EligibleActorIDs []string`; `Status StageStatus`.
- [ ] `StageStatus` enum: `StagePending`, `StageActive`, `StageCompleted`, `StageSkipped`, `StageRejectedHere`.
- [ ] `Instance.Active() *StageInstance` returns current active; nil if none.
- [ ] `Instance.AdvanceStage() error` active→completed, next pending→active; `ErrNoActiveStage` if none; sets `Status=InstanceApproved` + `CompletedAt` when last stage completes.
- [ ] `Instance.RejectHere(reason string) error` active→rejected_here; `Status=InstanceRejected` + `CompletedAt`.
- [ ] **`Instance.SkipStage(reason string) error`** (R1-5) — active→skipped, auto-advances to next pending; `ErrCannotSkipLastStage` if no successor. Reason persisted for audit.
- [ ] **`Instance.BumpRevisionVersion(next int) error`** (R1-3) — monotonic guard: `ErrRevisionRegression` if `next < RevisionVersion`; no-op if equal; assigns if greater.
- [ ] **`Instance.Cancel(reason string) error`** (R2-4) — sets `Status=InstanceCancelled`, `CompletedAt=now`; rejects if already terminal (Approved/Rejected/Cancelled) with `ErrInstanceTerminal`.

**Steps:**

- [ ] **Step 1: Write `instance_test.go`** — 3-stage instance; Active() returns stage 1; AdvanceStage×2 → stage 3 active; 3rd advance → Status=Approved. RejectHere → Status=Rejected. SkipStage on stage 1 → stage 2 active, stage 1 status=Skipped. SkipStage on last stage → ErrCannotSkipLastStage. BumpRevisionVersion: 0→1 ok, 1→0 ErrRevisionRegression, 1→1 no-op. **Cancel: in_progress→cancelled ok; re-Cancel → ErrInstanceTerminal; Cancel after Approved → ErrInstanceTerminal.**

- [ ] **Step 2: Implement `instance.go`** — pure data + methods, no persistence.

- [ ] **Step 3: Verify**

```bash
rtk go test ./internal/modules/documents_v2/approval/domain/ -run TestInstance -v -count=1
```

- [ ] **Step 4: Commit**

```bash
rtk git add internal/modules/documents_v2/approval/domain/instance.go internal/modules/documents_v2/approval/domain/instance_test.go
rtk git commit -m "feat(spec2/phase2): Instance + StageInstance aggregate"
```

---

### Task 2.4 — `signoff.go`: signoff value object (immutable)

**Files:** `internal/modules/documents_v2/approval/domain/signoff.go`, `signoff_test.go`.

**Acceptance Criteria:**
- [ ] `Signoff` struct — all fields unexported, exposed via getters only. Fields: id, approvalInstanceID, stageInstanceID, actorUserID, actorTenantID, decision, comment, signedAt, signatureMethod, signaturePayload (`json.RawMessage`), contentHash.
- [ ] `Decision` enum: `DecisionApprove`, `DecisionReject`.
- [ ] `NewSignoff(params SignoffParams) (*Signoff, error)` constructor validates: non-empty IDs, non-empty content hash, signed_at non-zero, decision ∈ enum. Content hash **normalized via `strings.ToLower`** then validated against `^[0-9a-f]{64}$` (R1-6). Uppercase SHA-256 hex accepted and stored lowercase.
- [ ] No setters; no public field assignment possible from outside package.
- [ ] `MarshalJSON` for API return.

**Steps:**

- [ ] **Step 1: Write `signoff_test.go`** — constructor rejects: empty instance id, empty actor id, bad hash (63 chars or non-hex), zero time, unknown decision; accepts happy path. **Hash canonicalization test:** uppercase input `"ABC…"` (64 hex chars upper) accepted; `signoff.ContentHash()` returns lowercase. Compile-time immutability probe: a `signoff.actorUserID = "x"` line in a commented-out test block documents that this won't compile.

- [ ] **Step 2: Implement `signoff.go`**.

- [ ] **Step 3: Verify**

```bash
rtk go test ./internal/modules/documents_v2/approval/domain/ -run TestSignoff -v -count=1
```

- [ ] **Step 4: Commit**

```bash
rtk git add internal/modules/documents_v2/approval/domain/signoff.go internal/modules/documents_v2/approval/domain/signoff_test.go
rtk git commit -m "feat(spec2/phase2): Signoff immutable value object"
```

---

### Task 2.5 — `quorum.go`: quorum evaluator

**Files:** `internal/modules/documents_v2/approval/domain/quorum.go`, `quorum_test.go`.

**Acceptance Criteria:**
- [ ] **`ComputeEffectiveDenominator(stage StageInstance, currentEligible []string) int`** (R1-2) — pure domain function. Caller passes current eligible set; function intersects with `stage.EligibleActorIDs` snapshot, returns count. Callers never pass raw ints.
- [ ] `EvaluateQuorum(stage StageInstance, approvals []Signoff, rejections []Signoff, effectiveDenominator int) QuorumOutcome`.
- [ ] `QuorumOutcome` enum: `QuorumPending`, `QuorumApprovedStage`, `QuorumRejectedStage`.
- [ ] `QuorumAny1Of`: first approval → `QuorumApprovedStage`; first rejection → `QuorumRejectedStage`.
- [ ] `QuorumAllOf`: `QuorumApprovedStage` only when `len(approvals) == effectiveDenominator`; any rejection → `QuorumRejectedStage`.
- [ ] `QuorumMofN`: `QuorumApprovedStage` when `len(approvals) ≥ QuorumM`; `QuorumRejectedStage` when `len(rejections) > effectiveDenominator - QuorumM`.
- [ ] Signoffs from actors NOT in `EligibleActorIDs` are ignored (defense-in-depth vs DB tenant trigger).
- [ ] `effectiveDenominator == 0` always returns `QuorumRejectedStage`.

**Steps:**

- [ ] **Step 1: Write `quorum_test.go`** — table tests covering all 3 policies × [0 signoffs, partial, met, exceeded, mixed approve/reject]. Explicit test: m_of_n with M=2, N=3, one approval + two rejections → `QuorumRejectedStage`. **`ComputeEffectiveDenominator` tests:** snapshot=[a,b,c], current=[a,b] → 2; current=[d] → 0; empty snapshot → 0; nil current → 0.

- [ ] **Step 2: Implement `quorum.go`**.

- [ ] **Step 3: Verify**

```bash
rtk go test ./internal/modules/documents_v2/approval/domain/ -run TestQuorum -v -count=1
```

- [ ] **Step 4: Commit**

```bash
rtk git add internal/modules/documents_v2/approval/domain/quorum.go internal/modules/documents_v2/approval/domain/quorum_test.go
rtk git commit -m "feat(spec2/phase2): quorum evaluator (any_1_of, all_of, m_of_n)"
```

---

### Task 2.6 — `sod.go`: Separation-of-Duties guards

**Files:** `internal/modules/documents_v2/approval/domain/sod.go`, `sod_test.go`.

**Acceptance Criteria:**
- [ ] `CheckSoD(authorUserID string, actorUserID string, priorSignoffs []Signoff) error`.
- [ ] Returns `ErrAuthorCannotSign` if `actorUserID == authorUserID`.
- [ ] Returns `ErrActorAlreadySigned` if actor appears in any prior signoff in same instance (prior stages).
- [ ] Returns nil on clean actor.
- [ ] Pure function — no DB, no globals.

**Steps:**

- [ ] **Step 1: Write `sod_test.go`** — cases: self-sign blocked, re-sign across stages blocked, fresh actor allowed, empty prior-signoffs allowed.

- [ ] **Step 2: Implement `sod.go`**.

- [ ] **Step 3: Verify**

```bash
rtk go test ./internal/modules/documents_v2/approval/domain/ -run TestSoD -v -count=1
```

- [ ] **Step 4: Commit**

```bash
rtk git add internal/modules/documents_v2/approval/domain/sod.go internal/modules/documents_v2/approval/domain/sod_test.go
rtk git commit -m "feat(spec2/phase2): SoD guards (author ≠ signer, no re-sign)"
```

---

### Task 2.7 — `drift.go`: eligibility drift policy application (R1-1)

**Files:** `internal/modules/documents_v2/approval/domain/drift.go`, `drift_test.go`.

**Acceptance Criteria:**
- [ ] `ApplyEligibilityDrift(stage StageInstance, currentEligible []string) DriftResult`.
- [ ] `DriftResult` struct: `EffectiveDenominator int`, `ForcedOutcome QuorumOutcome` (`QuorumPending` if no force), `Reason string`.
- [ ] `DriftReduceQuorum`: denominator = `len(snapshot ∩ current)`; `ForcedOutcome=QuorumPending`; reason documents drift delta.
- [ ] `DriftFailStage`: if snapshot ⊄ current (any snapshot actor departed), `ForcedOutcome=QuorumRejectedStage`, reason="eligibility drift: fail_stage policy".
- [ ] `DriftKeepSnapshot`: denominator = `len(snapshot)` (ignore current), `ForcedOutcome=QuorumPending`.
- [ ] Pure function — no DB, no time source.

**Steps:**

- [ ] **Step 1: Write `drift_test.go`** — 3 policy × [no drift, minor drift (1 removed), total drift (all removed), snapshot grew (new eligibles added)]. Assert `DriftFailStage` + total-drift → `QuorumRejectedStage`. Assert `DriftKeepSnapshot` ignores current (denom stays at snapshot count).

- [ ] **Step 2: Implement `drift.go`**.

- [ ] **Step 3: Verify**

```bash
rtk go test ./internal/modules/documents_v2/approval/domain/ -run TestDrift -v -count=1
```

- [ ] **Step 4: Commit**

```bash
rtk git add internal/modules/documents_v2/approval/domain/drift.go internal/modules/documents_v2/approval/domain/drift_test.go
rtk git commit -m "feat(spec2/phase2): ApplyEligibilityDrift runtime policy"
```

---

### Task 2.8 — Full-package test run + coverage gate + cross-file integration (R2-1,R2-2,R2-3)

**Files:** Create `internal/modules/documents_v2/approval/domain/integration_test.go`.

**R2 fixes added here:**
- **R2-1:** `TestSkipStageDuringSigning` — stage with 2 signoffs recorded, then `SkipStage` called; assert `QuorumOutcome` for skipped stage never evaluated again (skipped overrides quorum).
- **R2-2:** `TestDriftThenQuorumOrdering` — canonical call order: `ApplyEligibilityDrift` → (use its `EffectiveDenominator`) → `EvaluateQuorum`. Assert `fail_stage` drift short-circuits (quorum never consulted when `ForcedOutcome != QuorumPending`).
- **R2-3:** `TestRevisionVersionGatesStateTransition` — exercise `Instance.RevisionVersion` in a realistic flow: submit at v1, RejectHere, then resubmit requires `BumpRevisionVersion(2)` before next `AdvanceStage`; calling advance while version still v1 returns `ErrRevisionStale`. **Note:** this requires `AdvanceStage` to accept/check an expected version — add `AdvanceStage(expectedVersion int) error` overload, OR gate via repository layer in Phase 4. If deferred to Phase 4, mark this test as TODO with explicit comment linking the issue.

**Acceptance Criteria:**
- [ ] `go test ./internal/modules/documents_v2/approval/domain/...` passes green.
- [ ] `go test -cover` reports ≥ 90% statement coverage on the domain subpackage.
- [ ] `go vet` clean.
- [ ] `staticcheck` (if in repo) clean.

**Steps:**

- [ ] **Step 1: Run**

```bash
rtk go test ./internal/modules/documents_v2/approval/domain/... -race -count=1 -cover
```

Expected: `ok  ... coverage: >= 90.0% of statements`.

- [ ] **Step 2: Run vet**

```bash
rtk go vet ./internal/modules/documents_v2/approval/domain/...
```

Expected: no output.

- [ ] **Step 3: Commit any residual fixes** (e.g. unused imports) — but task ideally has zero diff.

---

### Task 2.9 — Codex COVERAGE review (Round 2)

**Goal:** Submit Phase 2 domain code + tests to Codex `gpt-5.3-codex` reasoning=high, mode COVERAGE. Must APPROVE (or APPROVE_WITH_FIXES local-scope only). Structural → stop, escalate.

**Files:** Create `docs/superpowers/plans/reviews/phase-2-round-2.json` (Round 1 artifact already at `phase-2-round-1.json`, see header fix table).

**Acceptance Criteria:**
- [ ] Verdict artifact committed with issues + fixes log.
- [ ] `APPROVE` or `APPROVE_WITH_FIXES` local-only before Phase 3 begins.

**Steps:**

- [ ] **Step 1: Assemble payload** — all 6 domain source files + 6 test files + this phase's invariants list.

- [ ] **Step 2: Call Codex**

```
model: gpt-5.3-codex
reasoning_effort: high
mode: COVERAGE
prompt: "Review Phase 2 domain layer for coverage gaps. Focus: edge cases in
  quorum m_of_n math (can-still-reach-M logic), SoD edge cases
  (reject-then-reapply across stages), state machine gaps (is every 7-state
  pair represented in tests?), drift policy missing paths. Return ONLY JSON
  verdict."
```

- [ ] **Step 3: Apply fixes, commit verdict**

```bash
rtk git add docs/superpowers/plans/reviews/phase-2-round-1.json
rtk git commit -m "chore(spec2/phase2): Codex COVERAGE round 1 verdict"
```

- [ ] **Step 4: Opus phase-end review** (per Model Assignments matrix) — Opus reads the 12 files + verdict, sanity-checks that domain types align with DB schema (column names, enum values) and spec text. Not a re-review of Codex findings; alignment check only.

---

## Phase 3: Infrastructure Layer (content_hash, idempotency, signature, membership_tx)

**Intent:** Build the cross-cutting infra primitives approval services depend on. Focus: Go↔TS canonical JSON parity for content hash, idempotency key algebra, signature provider seam (password re-auth now, ICP-Brasil later), `membership_tx` helper that wraps `SET LOCAL` + `SECURITY DEFINER` call shape.

**Codex review:** QUALITY (Task 3.9).

**Opus review:** none (per phase map).

**Subpackage:** `internal/modules/documents_v2/approval/` (application/ + infra/signature/) + `frontend/apps/web/src/features/approval/` for the TS content-hash parity.

**Codex Round 1 (QUALITY) verdict:** REJECT — but 4 of 5 issues were "code not yet written" false positives (Codex reviewed workspace state instead of the plan). Only one real plan gap:

| # | Scope | Fix |
|---|---|---|
| R1-1 | local | Idempotency timestamp-source contract pinned to server-authoritative; handler-layer rejection of client timestamps; defensive `.UTC().Truncate(Second)` inside `ComputeIdempotencyKey` |

Round 2 skipped — re-running against plan (not workspace) would surface no new issues. Artifact at `reviews/phase-3-round-1.json` with note on false-positive pattern for future phase reviews.

**Invariants this phase encodes:**

1. Content hash = SHA-256 of canonical JSON of `{tenant_id, document_id, revision_number, form_data_json}` — byte-for-byte identical between Go and TS.
2. Canonical JSON = sorted keys recursively, no whitespace, UTF-8, Unicode NFC, `null` for missing optionals (never omit).
3. Idempotency key = SHA-256 hex of `{actor_user_id, document_id, stage_instance_id, decision, timestamp_second_bucket}`; second-bucket granularity prevents user-double-click dupes while allowing intentional re-sign after stage reopen.
4. Signature provider seam: `Provider` interface with `Sign(ctx, req) (SignatureResult, error)` + `Method() string`; registry dispatches by method name; adding ICP-Brasil later means new Provider, no service code change.
5. `membership_tx.WithMembershipContext(tx, actor, capability, fn)` sets GUCs via `SET LOCAL`, calls `fn(tx)`, commits or rolls back. Never calls `SET SESSION`. Always `SET LOCAL` inside an active tx.
6. Password re-auth provider: bcrypt compare against `iam_users.password_hash`, rate-limited, emits audit event on failure.

---

### Task 3.1 — Canonical JSON spec doc (Go ↔ TS contract)

**Files:** Create `internal/modules/documents_v2/approval/application/canonical_json_spec.md`.

**Goal:** Human-readable spec pinned to both Go and TS implementations. If either diverges, content hashes drift and every signature breaks.

**Acceptance Criteria:**
- [ ] Spec covers: key sort order (byte-wise on UTF-8), string escaping (JSON.stringify-equivalent for TS; Go `encoding/json` with custom encoder), number format (integers verbatim; floats forbidden in form_data — fail-fast), null handling (present, not omitted), Unicode normalization (NFC via `golang.org/x/text/unicode/norm` / `String.prototype.normalize('NFC')`).
- [ ] Two worked examples with expected SHA-256 output pinned for regression.
- [ ] Floats rejected section explicit: form_data values must be strings/ints/bools/nested objects — no float64.

**Steps:**

- [ ] **Step 1: Write spec doc** (Codex `gpt-5.3-codex` medium).

- [ ] **Step 2: Commit**

```bash
rtk git add internal/modules/documents_v2/approval/application/canonical_json_spec.md
rtk git commit -m "docs(spec2/phase3): canonical JSON spec for content hash parity"
```

---

### Task 3.2 — `content_hash.go` (Go side)

**Files:** `internal/modules/documents_v2/approval/application/content_hash.go`, `content_hash_test.go`.

**Acceptance Criteria:**
- [ ] `ComputeContentHash(input ContentHashInput) (string, error)`.
- [ ] `ContentHashInput` struct: `TenantID`, `DocumentID`, `RevisionNumber int`, `FormData map[string]any`.
- [ ] Returns `ErrFloatInFormData` if any nested value is `float64` (canonical spec rejects floats).
- [ ] Recursive canonical marshaller: sorted keys, NFC-normalized strings, no whitespace.
- [ ] Output = lowercase hex SHA-256.
- [ ] **Golden-vector tests** using the 2 worked examples from Task 3.1.

**Steps:**

- [ ] **Step 1: Write `content_hash_test.go`** with golden vectors (Codex high — this is parity-critical).

- [ ] **Step 2: Implement `content_hash.go`** (Codex high).

- [ ] **Step 3: Verify**

```bash
rtk go test ./internal/modules/documents_v2/approval/application/ -run TestContentHash -v -count=1
```

- [ ] **Step 4: Commit**

```bash
rtk git add internal/modules/documents_v2/approval/application/content_hash.go internal/modules/documents_v2/approval/application/content_hash_test.go
rtk git commit -m "feat(spec2/phase3): Go content hash with canonical JSON + golden vectors"
```

---

### Task 3.3 — `contentHash.ts` (TS side, parity)

**Files:** `frontend/apps/web/src/features/approval/contentHash.ts`, `contentHash.test.ts`.

**Acceptance Criteria:**
- [ ] `computeContentHash(input: ContentHashInput): Promise<string>` using `crypto.subtle.digest`.
- [ ] **Same golden vectors as Task 3.2** — TS test file duplicates them byte-for-byte; if Go output ≠ TS output, test fails.
- [ ] Rejects floats with `FloatInFormDataError`.
- [ ] Uses `String.prototype.normalize('NFC')` on every string.

**Steps:**

- [ ] **Step 1: Write `contentHash.test.ts`** (Sonnet — TS boilerplate; Codex supplies golden vectors from Task 3.2).

- [ ] **Step 2: Implement `contentHash.ts`** (Codex high).

- [ ] **Step 3: Verify**

```bash
rtk pnpm -C frontend/apps/web test -- contentHash
```

- [ ] **Step 4: Parity probe** — small Go program outputs hash; TS test consumes same input and asserts equality.

```bash
rtk go run ./scripts/spec2/content_hash_parity_probe.go > /tmp/go_hash.txt
rtk pnpm -C frontend/apps/web test -- contentHash.parity
```
Expected: Go and TS hashes match for all 3 probe inputs.

- [ ] **Step 5: Commit**

```bash
rtk git add frontend/apps/web/src/features/approval/contentHash.ts frontend/apps/web/src/features/approval/contentHash.test.ts scripts/spec2/content_hash_parity_probe.go
rtk git commit -m "feat(spec2/phase3): TS content hash with Go parity golden vectors"
```

---

### Task 3.4 — `idempotency.go`

**Files:** `internal/modules/documents_v2/approval/application/idempotency.go`, `idempotency_test.go`.

**Acceptance Criteria:**
- [ ] `ComputeIdempotencyKey(input IdempotencyInput) string`.
- [ ] Input: `ActorUserID`, `DocumentID`, `StageInstanceID`, `Decision`, `Timestamp time.Time`.
- [ ] **Timestamp source = server-authoritative** (R1-1). Command handler sets `input.Timestamp = time.Now().UTC().Truncate(time.Second)` at request entry; client-supplied timestamps rejected at handler layer (Phase 5). Defense-in-depth: `ComputeIdempotencyKey` itself calls `t.UTC().Truncate(time.Second)` so sub-second noise can never leak into the hash.
- [ ] Output = lowercase hex SHA-256 of sorted canonical JSON.
- [ ] Same input within same second → same key (dedupe double-click).
- [ ] Different-second input → different key (intentional re-sign after reopen).
- [ ] Godoc pins contract: "server clock only — never trust client timestamp".

**Steps:**

- [ ] **Step 1: Write test** — 3 cases: double-click within second → equal keys; 1s apart → different keys; different actor → different keys.

- [ ] **Step 2: Implement**.

- [ ] **Step 3: Verify**

```bash
rtk go test ./internal/modules/documents_v2/approval/application/ -run TestIdempotency -v -count=1
```

- [ ] **Step 4: Commit**

```bash
rtk git add internal/modules/documents_v2/approval/application/idempotency.go internal/modules/documents_v2/approval/application/idempotency_test.go
rtk git commit -m "feat(spec2/phase3): idempotency key with second-bucket granularity"
```

---

### Task 3.5 — `signature/provider.go` + registry

**Files:** `internal/modules/documents_v2/approval/infra/signature/provider.go`, `registry.go`.

**Acceptance Criteria:**
- [ ] `Provider` interface: `Method() string`, `Sign(ctx, SignRequest) (SignatureResult, error)`.
- [ ] `SignRequest`: `ActorUserID`, `ActorTenantID`, `ContentHash`, `Credentials map[string]string` (method-specific; password re-auth reads `"password"`).
- [ ] `SignatureResult`: `Method string`, `Payload json.RawMessage` (opaque bag of method-specific attestation), `SignedAt time.Time`.
- [ ] `Registry` holds `map[string]Provider`; `Registry.Get(method) (Provider, error)` returns `ErrUnknownSignatureMethod` on miss.
- [ ] Interface is stable — adding ICP-Brasil later = new Provider registered, zero service-code change.

**Steps:**

- [ ] **Step 1: Write interface + registry** (Codex medium).

- [ ] **Step 2: Registry test** — register 2 mock providers, assert Get returns correct one; missing method returns ErrUnknownSignatureMethod.

- [ ] **Step 3: Commit**

```bash
rtk git add internal/modules/documents_v2/approval/infra/signature/
rtk git commit -m "feat(spec2/phase3): signature provider interface + registry"
```

---

### Task 3.6 — `signature/password_reauth.go`

**Files:** `internal/modules/documents_v2/approval/infra/signature/password_reauth.go`, `password_reauth_test.go`.

**Acceptance Criteria:**
- [ ] `PasswordReauthProvider` implements `Provider`; `Method() == "password_reauth"`.
- [ ] Reads `iam_users.password_hash` via injected `IamUserReader` interface (repo abstraction — not a hard `*sql.DB` coupling; lets tests inject fake).
- [ ] Uses `bcrypt.CompareHashAndPassword`; on mismatch returns `ErrInvalidCredentials` **without distinguishing "user missing" from "wrong password"** (timing-safe + disclosure-safe).
- [ ] Rate limit: max 5 failed attempts per (actor, 60s window) before `ErrRateLimited`. In-memory sliding window keyed by actor_user_id; thread-safe via `sync.Mutex`.
- [ ] **Bounded memory (R1-2, preempt):** per-actor entry TTL = window × 2 (120s). Background janitor goroutine sweeps expired entries every 30s. `NewPasswordReauthProvider(ctx, ...)` takes ctx to stop janitor on shutdown. Alternative: lazy eviction on every `Sign` call (sweep 10 oldest if map size > 10_000). Test asserts: entries older than TTL removed; map size bounded under adversarial load (1M distinct actor IDs).
- [ ] On failure: emits `governance_event` `signature_auth_failed` with `{actor_user_id, ip, reason}` — via injected `EventEmitter` interface (Phase 5 wires real one).
- [ ] `Payload` on success = `{"method":"password_reauth","bcrypt_cost":N,"verified_at":"..."}` — no secret fields.

**Steps:**

- [ ] **Step 1: Write `password_reauth_test.go`** — 5 cases: happy path, wrong password, missing user (same error code), rate-limit trip (6th attempt in 60s), rate-limit reset after 60s.

- [ ] **Step 2: Implement**.

- [ ] **Step 3: Verify**

```bash
rtk go test ./internal/modules/documents_v2/approval/infra/signature/ -race -v -count=1
```
`-race` mandatory — rate-limiter state is concurrent.

- [ ] **Step 4: Commit**

```bash
rtk git add internal/modules/documents_v2/approval/infra/signature/password_reauth.go internal/modules/documents_v2/approval/infra/signature/password_reauth_test.go
rtk git commit -m "feat(spec2/phase3): password re-auth signature provider + rate limit"
```

---

### Task 3.7 — `membership_tx.go`

**Files:** `internal/modules/documents_v2/approval/application/membership_tx.go`, `membership_tx_test.go`.

**Acceptance Criteria:**
- [ ] `WithMembershipContext(ctx, db, actorUserID, capability, fn func(tx *sql.Tx) error) error`.
- [ ] Begins tx; `SET LOCAL ROLE metaldocs_membership_writer`; `SET LOCAL metaldocs.actor_id = $actor`; `SET LOCAL metaldocs.verified_capability = $cap`; calls `fn(tx)`; commits on nil, rollbacks on error.
- [ ] Never uses `SET SESSION`. Never uses plain `SET`. `SET LOCAL` only — verified by test parsing executed SQL.
- [ ] Rollback is explicit on panic via `defer`.
- [ ] On rollback, bubbles original error (not rollback error unless rollback itself fails).
- [ ] Returns `ErrNoActor` if `actorUserID == ""`.

**Steps:**

- [ ] **Step 1: Write test** using `DATA-DOG/go-sqlmock` — assert exact `SET LOCAL` statements in order; assert rollback on fn error; assert commit on fn success; assert empty actor rejected before tx begins.

- [ ] **Step 2: Implement** (Codex high — tx lifecycle + panic safety).

- [ ] **Step 3: Verify**

```bash
rtk go test ./internal/modules/documents_v2/approval/application/ -run TestMembershipTx -race -v -count=1
```

- [ ] **Step 4: Commit**

```bash
rtk git add internal/modules/documents_v2/approval/application/membership_tx.go internal/modules/documents_v2/approval/application/membership_tx_test.go
rtk git commit -m "feat(spec2/phase3): membership_tx SET LOCAL + SECURITY DEFINER wrapper"
```

---

### Task 3.8 — `events.go` (governance_events emitter stub)

**Files:** `internal/modules/documents_v2/approval/application/events.go`, `events_test.go`.

**Acceptance Criteria:**
- [ ] `EventEmitter` interface: `Emit(ctx, tx, event GovernanceEvent) error` — requires tx so outbox writes stay same-transaction.
- [ ] `GovernanceEvent` struct matching `governance_events` columns.
- [ ] `sqlEmitter` default impl inserts via prepared statement.
- [ ] Test fakes in-memory emitter for domain tests.

**Steps:**

- [ ] **Step 1: Write interface + sql impl** (Codex medium).

- [ ] **Step 2: Test** sqlEmitter with sqlmock asserting prepared INSERT shape.

- [ ] **Step 3: Commit**

```bash
rtk git add internal/modules/documents_v2/approval/application/events.go internal/modules/documents_v2/approval/application/events_test.go
rtk git commit -m "feat(spec2/phase3): governance event emitter + tx-bound SQL impl"
```

---

### Task 3.9 — Codex QUALITY review

**Files:** Create `docs/superpowers/plans/reviews/phase-3-round-1.json`.

**Acceptance Criteria:**
- [ ] Codex `gpt-5.3-codex` high, mode QUALITY, reviews all Phase 3 files.
- [ ] Verdict APPROVE or APPROVE_WITH_FIXES local.
- [ ] Structural → stop, escalate.

**Steps:**

- [ ] **Step 1: Assemble payload** — 8 Go files + 1 TS file + 1 spec doc + golden vectors.

- [ ] **Step 2: Call Codex** mode=QUALITY. Focus: test error-handling completeness, rate-limit races, canonical JSON edge cases (nested arrays, nil maps, embedded bytes), bcrypt timing-safety, sql.Tx panic safety.

- [ ] **Step 3: Apply fixes, commit verdict**

```bash
rtk git add docs/superpowers/plans/reviews/phase-3-round-1.json
rtk git commit -m "chore(spec2/phase3): Codex QUALITY round 1 verdict"
```

---

## Phase 4: Repository Layer (OCC, transactional boundaries, legacy purge)

**Intent:** Build `approval_repository` with strict OCC semantics mirroring the DB `revision_version` monotonic trigger. Extend `documents_v2` repo with transition + lock/unlock queries gated by OCC. **Remove legacy code paths** (`finalized`, `archived`) from existing `documents_v2` repo and service — per user directive "legacy shall be removed" — replacing with Spec 2 7-state vocabulary.

**Codex review:** ARCHITECTURE (Task 4.10).

**Opus review:** phase end — cross-check domain ↔ repo ↔ DB column/type alignment.

**Codex Round 1 (ARCHITECTURE) verdict:** APPROVE_WITH_FIXES — 7 findings (2 critical, 4 high, 1 medium). All applied inline:

| # | Severity | Fix |
|---|---|---|
| F1 | critical | Canonical repo interface (4.1) includes `tenantID` on all reads, expected-status args on all updates, `LoadSignoffByActor`, full 8-code error taxonomy |
| F2 | critical | Signoff idempotency: after ON CONFLICT, load existing and compare `(idempotency_key, content_hash)` — same identity=replay, different=`ErrActorAlreadySigned` |
| F3 | high | Tx ownership rule: Phase 5 services own via `RunInMembershipTx`; handlers/jobs never open approval write tx; repo methods strictly tx-in/tx-out |
| F4 | high | SQLSTATE mapping matrix: 23505→constraint-specific, 23503→`ErrFKViolation`, 23514→`ErrCheckViolation`+reason, 42501→`ErrInsufficientPrivilege`, other→`ErrUnknownDB` |
| F5 | high | Legacy purge uses compile-safe typed error stubs (not panic); temp adapter methods kept until Phase 5 cutover |
| F6 | medium | `ListScheduledDue` tx isolation = READ COMMITTED explicit; claim+transition in same tx |
| F7 | high | New Task 4.12: CI job runs integration-tagged tests; prevents silent skip |

**Subpackages:**
- `internal/modules/documents_v2/approval/repository/approval_repository.go`
- `internal/modules/documents_v2/repository/repository.go` (modified)
- `internal/modules/documents_v2/application/service.go` (modified — legacy path strip only; new services land in Phase 5)

**Invariants this phase encodes:**

1. Every mutating query on `documents` uses `WHERE revision_version = $expected` — OCC enforced in SQL, not Go.
2. Every mutating query returns `RowsAffected` check; 0 rows → `ErrStaleRevision` (409 mapped in Phase 7).
3. Every write path through `approval_repository` goes via `WithMembershipContext` from Phase 3 — no raw `sql.DB` writes.
4. Transition queries check `revision_version` AND call `SET LOCAL metaldocs.actor_id` so DB tripwires see actor context.
5. Signoff insert bundles idempotency key + content hash; DB `UNIQUE (approval_instance_id, actor_user_id)` backstop handles races.
6. Legacy path purge: zero `finalized` / `archived` references in Go code after this phase; `git grep -nE "'finalized'|'archived'" internal/modules/documents_v2/` returns zero matches.
7. **Transaction ownership rule (F3):** Phase 5 services own tx lifecycle via `RunInMembershipTx`. Handlers and jobs NEVER open approval write tx. Repository methods are strictly **tx-in/tx-out** — they take `*sql.Tx` and never call `BeginTx` or `Commit`/`Rollback`. One service method = one DB tx, closed before HTTP response.
8. **Tx isolation (F6):** `ListScheduledDue` and scheduler claim path require caller to open tx with `sql.TxOptions{Isolation: sql.LevelReadCommitted}`. Claim (SELECT FOR UPDATE SKIP LOCKED) and transition (UPDATE documents SET status='published') happen in the same tx.

---

### Task 4.1 — Repository interface + error taxonomy

**Files:** `internal/modules/documents_v2/approval/repository/approval_repository.go` (interface only), `errors.go`.

**Acceptance Criteria:**
- [ ] **Canonical `ApprovalRepository` interface** (F1) — all 4.x tasks MUST match these signatures exactly:
  - `InsertInstance(ctx, tx, inst domain.Instance) error`
  - `InsertStageInstances(ctx, tx, stages []domain.StageInstance) error`
  - `InsertSignoff(ctx, tx, s domain.Signoff) (SignoffInsertResult, error)` — returns `{ID, WasReplay bool}`; F2-compliant
  - `LoadSignoffByActor(ctx, tx, tenantID, instanceID, actorUserID string) (*domain.Signoff, error)`
  - `LoadInstance(ctx, tx, tenantID, id string) (*domain.Instance, error)`
  - `LoadActiveInstanceByDocument(ctx, tx, tenantID, docID string) (*domain.Instance, error)` — `ErrNoActiveInstance` if none; wrong-tenant also returns `ErrNoActiveInstance` (no leak)
  - `UpdateStageStatus(ctx, tx, tenantID, stageID string, newStatus, expectedOldStatus domain.StageStatus) error`
  - `UpdateInstanceStatus(ctx, tx, tenantID, instID string, newStatus, expectedStatus domain.InstanceStatus, completedAt *time.Time) error`
  - `ListScheduledDue(ctx, tx, now time.Time, limit int) ([]ScheduledPublishRow, error)` — `FOR UPDATE SKIP LOCKED`; caller MUST open tx with `sql.LevelReadCommitted` (F6)
- [ ] **Canonical error taxonomy** (F1+F4) in `errors.go`:
  - `ErrStaleRevision` — 0 rows affected on OCC update
  - `ErrNoActiveInstance` — no row OR wrong tenant
  - `ErrDuplicateSubmission` — unique violation on `(document_v2_id, idempotency_key)`
  - `ErrActorAlreadySigned` — ON CONFLICT + different identity (F2)
  - `ErrCrossTenantSignoff` — tenant-consistency trigger rejection
  - `ErrInstanceCompleted` — write to terminal instance
  - `ErrStageNotActive` — expected-status guard mismatch
  - `ErrFKViolation` — SQLSTATE 23503
  - `ErrCheckViolation` — SQLSTATE 23514 (includes DB trigger `RAISE EXCEPTION` with reason text)
  - `ErrInsufficientPrivilege` — SQLSTATE 42501 (membership_tx GUC missing)
  - `ErrUnknownDB` — fallback; wraps original pq.Error
- [ ] **SQLSTATE mapping matrix** (F4) in `errors.go` as `MapPgError(err error, hints MapHints) error` — translates `*pq.Error` to domain error using `Code + Constraint` pair (e.g. `23505 + ux_approval_instances_active` → `ErrDuplicateSubmission`).

**Steps:**

- [ ] **Step 1: Write interface + errors** (Codex medium).

- [ ] **Step 2: Commit**

```bash
rtk git add internal/modules/documents_v2/approval/repository/approval_repository.go internal/modules/documents_v2/approval/repository/errors.go
rtk git commit -m "feat(spec2/phase4): approval repo interface + error taxonomy"
```

---

### Task 4.2 — Postgres impl: instance + stage inserts

**Files:** `internal/modules/documents_v2/approval/repository/postgres_approval_repository.go`, `postgres_approval_repository_test.go`.

**Acceptance Criteria:**
- [ ] `NewPostgresApprovalRepository(db *sql.DB) ApprovalRepository`.
- [ ] `InsertInstance` uses prepared INSERT returning nothing; unique-violation on `(document_v2_id, idempotency_key)` → `ErrDuplicateSignoff` variant (`ErrDuplicateSubmission`).
- [ ] `InsertStageInstances` uses single multi-row INSERT for all stages in one round-trip.
- [ ] Test (sqlmock): assert parameter bind order matches column order; unique violation translates to domain error.

**Steps:**

- [ ] **Step 1: Write test first** — sqlmock expects INSERT with args; simulate `pq.Error{Code: "23505", Constraint: "..."}`.

- [ ] **Step 2: Implement** (Codex medium).

- [ ] **Step 3: Verify**

```bash
rtk go test ./internal/modules/documents_v2/approval/repository/ -run TestInsertInstance -race -count=1
```

- [ ] **Step 4: Commit**

```bash
rtk git add internal/modules/documents_v2/approval/repository/postgres_approval_repository.go internal/modules/documents_v2/approval/repository/postgres_approval_repository_test.go
rtk git commit -m "feat(spec2/phase4): postgres approval repo — instance + stage inserts"
```

---

### Task 4.3 — Signoff insert with idempotency + dup collapse

**Files:** same `postgres_approval_repository.go` (extend), tests added.

**Acceptance Criteria:**
- [ ] `InsertSignoff`: INSERT with `ON CONFLICT (approval_instance_id, actor_user_id) DO NOTHING RETURNING id`.
- [ ] **On empty RETURNING (F2): load existing via `LoadSignoffByActor`, compare `(idempotency_key, content_hash, stage_instance_id, decision)`. Same 4-tuple → replay: return `SignoffInsertResult{ID: existing.ID, WasReplay: true}`. Any field differs → `ErrActorAlreadySigned`.**
- [ ] Tenant-consistency trigger rejection (SQLSTATE 23514 with message containing "cross-tenant signoff") → `ErrCrossTenantSignoff`.
- [ ] Cross-instance composite FK rejection (R2-3 from Phase 1, SQLSTATE 23503 on `approval_signoffs_stage_matches_instance`) → `ErrFKViolation`.

**Steps:**

- [ ] **Step 1: Test** — sqlmock: (a) clean INSERT returns id, `WasReplay=false`; (b) ON CONFLICT empty + existing matches all 4 fields → `WasReplay=true`, same ID; (c) ON CONFLICT empty + existing differs on content_hash → ErrActorAlreadySigned; (d) trigger rejection (23514 "cross-tenant") → ErrCrossTenantSignoff; (e) FK rejection (23503 stage_matches_instance) → ErrFKViolation.

- [ ] **Step 2: Implement**.

- [ ] **Step 3: Verify**

```bash
rtk go test ./internal/modules/documents_v2/approval/repository/ -run TestInsertSignoff -race -count=1
```

- [ ] **Step 4: Commit**

```bash
rtk git add internal/modules/documents_v2/approval/repository/postgres_approval_repository.go internal/modules/documents_v2/approval/repository/postgres_approval_repository_test.go
rtk git commit -m "feat(spec2/phase4): signoff insert with ON CONFLICT idempotent replay"
```

---

### Task 4.4 — Load aggregates + active-instance lookup

**Files:** same repo file, tests added.

**Acceptance Criteria:**
- [ ] `LoadInstance(ctx, tx, id)` — single query with LEFT JOIN on `approval_stage_instances` and `approval_signoffs`; reconstructs full domain aggregate.
- [ ] `LoadActiveInstanceByDocument(ctx, tx, docID)` — uses partial unique index `ux_approval_instances_active` (`WHERE status='in_progress'`).
- [ ] Tenant scoping enforced at query level: `WHERE tenant_id = $1` always present; repo signatures take `tenantID` explicitly (no reliance on GUC for read).
- [ ] Signoff order: by `stage_order ASC, signed_at ASC` for deterministic timeline.

**Steps:**

- [ ] **Step 1: Test** — 3-stage instance with 5 signoffs loads correctly; active lookup returns nil + ErrNoActiveInstance when none; wrong tenant returns ErrNoActiveInstance (not "not found"-leak).

- [ ] **Step 2: Implement**.

- [ ] **Step 3: Commit**

```bash
rtk git add internal/modules/documents_v2/approval/repository/postgres_approval_repository.go internal/modules/documents_v2/approval/repository/postgres_approval_repository_test.go
rtk git commit -m "feat(spec2/phase4): load instance aggregate + active lookup"
```

---

### Task 4.5 — Status updates with optimistic concurrency

**Files:** same repo file, tests added.

**Acceptance Criteria:**
- [ ] `UpdateStageStatus(ctx, tx, stageID, newStatus, expectedOldStatus)` — `UPDATE ... WHERE id=$1 AND status=$expected`; `RowsAffected()==0` → `ErrStageNotActive`.
- [ ] `UpdateInstanceStatus(ctx, tx, instID, newStatus, completedAt, expectedStatus)` — same pattern with expected status guard.
- [ ] No method accepts arbitrary status transitions — caller must supply `expected` value to prove it knows current state. Defense against lost updates.

**Steps:**

- [ ] **Step 1: Test** — happy update; wrong expected status → ErrStageNotActive; concurrent double-advance: 2 goroutines, 1 succeeds, 1 gets ErrStageNotActive (`-race`).

- [ ] **Step 2: Implement**.

- [ ] **Step 3: Verify**

```bash
rtk go test ./internal/modules/documents_v2/approval/repository/ -run TestUpdateStatus -race -count=5
```
`-count=5` to shake race detection.

- [ ] **Step 4: Commit**

```bash
rtk git add internal/modules/documents_v2/approval/repository/postgres_approval_repository.go internal/modules/documents_v2/approval/repository/postgres_approval_repository_test.go
rtk git commit -m "feat(spec2/phase4): status updates with expected-value OCC guards"
```

---

### Task 4.6 — Scheduled-publish queue (FOR UPDATE SKIP LOCKED)

**Files:** same repo file, tests added.

**Acceptance Criteria:**
- [ ] `ListScheduledDue(ctx, tx, now, limit)` — query `documents WHERE status='scheduled' AND effective_from <= $1 ORDER BY effective_from FOR UPDATE SKIP LOCKED LIMIT $limit`.
- [ ] Returns `ScheduledPublishRow` struct: `DocumentID`, `TenantID`, `EffectiveFrom`, `RevisionVersion`.
- [ ] Called only inside a tx opened with `sql.TxOptions{Isolation: sql.LevelReadCommitted}` (F6). Claim + transition must happen in the same tx; `defer tx.Rollback()` + explicit `tx.Commit()` on success.
- [ ] Godoc states: "caller MUST use READ COMMITTED; higher isolation breaks SKIP LOCKED semantics". Test asserts godoc presence via `go/ast` parse.
- [ ] Integration test with real Postgres (docker): 3 scheduler instances × 100 scheduled docs → every doc processed exactly once, no worker blocks on another's batch.

**Steps:**

- [ ] **Step 1: Test** — sqlmock asserts exact SQL including `FOR UPDATE SKIP LOCKED`.

- [ ] **Step 2: Integration test** (behind build tag `//go:build integration`) — 3 goroutines open separate txs, each calls ListScheduledDue with limit=10; assert no overlap in returned IDs.

- [ ] **Step 3: Implement**.

- [ ] **Step 4: Verify**

```bash
rtk go test ./internal/modules/documents_v2/approval/repository/ -run TestListScheduledDue -tags=integration -count=1
```

- [ ] **Step 5: Commit**

```bash
rtk git add internal/modules/documents_v2/approval/repository/postgres_approval_repository.go internal/modules/documents_v2/approval/repository/postgres_approval_repository_test.go
rtk git commit -m "feat(spec2/phase4): scheduled-due queue with SKIP LOCKED"
```

---

### Task 4.7 — Extend `documents_v2` repo: transition + lock/unlock with OCC

**Files:** Modify `internal/modules/documents_v2/repository/repository.go`, tests added.

**Acceptance Criteria:**
- [ ] `TransitionStatus(ctx, tx, docID, newStatus, expectedRevisionVersion int) error` — `UPDATE documents SET status=$1, revision_version=revision_version+1 WHERE id=$2 AND revision_version=$3`; 0 rows → `ErrStaleRevision`.
- [ ] `LockForReview(ctx, tx, docID, expectedRevisionVersion int) error` — sets `locked_at=now()` + bumps revision_version; only from status `under_review`.
- [ ] `Unlock(ctx, tx, docID, expectedRevisionVersion int) error` — clears `locked_at`; only from status `approved|rejected`.
- [ ] All queries use placeholders, no string concatenation.
- [ ] Tests: happy path, stale revision (concurrent edit), wrong current status (DB trigger rejects).

**Steps:**

- [ ] **Step 1: Test** — sqlmock + integration. Simulate stale write with 2 goroutines racing TransitionStatus on same doc; exactly 1 succeeds.

- [ ] **Step 2: Implement**.

- [ ] **Step 3: Commit**

```bash
rtk git add internal/modules/documents_v2/repository/repository.go internal/modules/documents_v2/repository/repository_test.go
rtk git commit -m "feat(spec2/phase4): documents_v2 repo transition + lock with OCC"
```

---

### Task 4.8 — Legacy path purge (documents_v2 repo + service)

**Files:** `internal/modules/documents_v2/repository/repository.go`, `internal/modules/documents_v2/application/service.go`, `internal/modules/documents_v2/domain/model.go`.

**Acceptance Criteria:**
- [ ] All references to `'finalized'` / `'archived'` / `finalized_at` / `archived_at` removed from non-migration Go code.
- [ ] `DocumentStatus` enum in `domain/model.go` = 8 Spec 2 states only.
- [ ] `Finalize()` / `Archive()` methods removed from service; replaced (future) by `Submit()` (Phase 5).
- [ ] `grep` verification gate: `git grep -nE "'finalized'|'archived'|finalized_at|archived_at" internal/modules/documents_v2/ -- ':!*.md'` returns 0 matches.
- [ ] **Callers of removed methods (F5):** compile-safe typed error stubs only. Each stub returns `ErrLegacyMethodRemoved` (new error in `errors.go`) with message `"legacy Finalize()/Archive() removed; route through Phase 5 SubmitService"`. **NO panics in live paths.** Stubs marked `// DEPRECATED(spec2-phase5): remove when SubmitService lands`.
- [ ] **Staged removal sequence:** (1) add stubs returning error; (2) update tests that exercised legacy methods to assert ErrLegacyMethodRemoved; (3) after Phase 5 lands, delete stubs entirely in Phase 5 cutover commit.
- [ ] Temporary adapter methods may live in `application/legacy_adapter.go` to bridge handler-layer consumers until Phase 5 handlers replace them.

**Steps:**

- [ ] **Step 1: Remove legacy** — Codex medium does the grep + delete sweep. Replace any call sites with stubs.

- [ ] **Step 2: Run test suite**

```bash
rtk go build ./...
rtk go test ./internal/modules/documents_v2/... -count=1
```
Expected: compiles. Tests that exercise removed methods are deleted in same commit (documented in commit body).

- [ ] **Step 3: Verify zero legacy**

```bash
rtk git grep -nE "'finalized'|'archived'|finalized_at|archived_at" internal/modules/documents_v2/ -- ':!*.md' ':!migrations/*'
```
Expected: empty output.

- [ ] **Step 4: Commit**

```bash
rtk git add internal/modules/documents_v2/
rtk git commit -m "refactor(spec2/phase4): purge legacy finalized/archived paths"
```

---

### Task 4.9 — Repository wiring through `membership_tx`

**Files:** `internal/modules/documents_v2/approval/repository/tx_wrapper.go`, test.

**Acceptance Criteria:**
- [ ] Helper `RunInMembershipTx(ctx, db, actor, cap, fn func(tx, repo) error) error` — combines Phase 3 `WithMembershipContext` with repo construction.
- [ ] Ensures every Phase 5 service call path MUST acquire tx via this helper. No public `ApprovalRepository` constructor exposes raw writes outside helper.
- [ ] Test asserts: calling repo write without membership tx panics or returns `ErrNoMembershipContext`.

**Steps:**

- [ ] **Step 1: Implement** (Codex high — tx wiring).

- [ ] **Step 2: Test** — negative path (raw call) + happy path.

- [ ] **Step 3: Commit**

```bash
rtk git add internal/modules/documents_v2/approval/repository/tx_wrapper.go internal/modules/documents_v2/approval/repository/tx_wrapper_test.go
rtk git commit -m "feat(spec2/phase4): membership-tx repo wrapper enforcing context"
```

---

### Task 4.10 — Codex ARCHITECTURE review

**Files:** Create `docs/superpowers/plans/reviews/phase-4-round-1.json`.

**Acceptance Criteria:**
- [ ] Codex `gpt-5.3-codex` high, mode ARCHITECTURE.
- [ ] **Prompt explicitly states: this is a PLAN review; deliverable files do not exist yet — that is correct.** (Preempts Phase 3 false-positive pattern.)
- [ ] Verdict APPROVE or APPROVE_WITH_FIXES local only.
- [ ] Focus: tx boundary clarity, OCC correctness, legacy purge completeness, coupling between domain and repo, SoD defense-in-depth vs DB backstop.

**Steps:**

- [ ] **Step 1: Assemble payload** — interfaces, sample query signatures, error taxonomy, legacy-purge gate.

- [ ] **Step 2: Call Codex** with plan-review framing.

- [ ] **Step 3: Apply fixes, commit verdict**

```bash
rtk git add docs/superpowers/plans/reviews/phase-4-round-1.json
rtk git commit -m "chore(spec2/phase4): Codex ARCHITECTURE round 1 verdict"
```

---

### Task 4.11 — Opus phase-end review

**Goal:** Opus reads Phase 4 + the Phase 1 DB schema + Phase 2 domain types. Cross-checks: every repo method signature uses the correct domain type; every SQL column referenced exists in migrations 0130–0140; error taxonomy complete vs documented Spec 2 error codes.

**Acceptance Criteria:**
- [ ] Short written summary appended to plan file as `### Phase 4 — Opus alignment review` section listing any mismatches.
- [ ] Zero unresolved mismatches before Phase 5 begins.

**Steps:**

- [ ] **Step 1: Opus reads Phase 1 migrations + Phase 2 domain + Phase 4 tasks.**
- [ ] **Step 2: Cross-check matrix: domain type ↔ repo signature ↔ SQL column ↔ error code.**
- [ ] **Step 3: Write summary section or flag blockers.**
- [ ] **Step 4: Commit**

```bash
rtk git add docs/superpowers/plans/2026-04-21-foundation-doc-approval-state-machine.md
rtk git commit -m "chore(spec2/phase4): Opus alignment review"
```

---

### Task 4.12 — CI integration-test wiring (F7)

**Goal:** Ensure `//go:build integration` tagged tests run in CI. Without this, Tasks 4.6/4.7 integration probes silently skip.

**Files:** Modify `.github/workflows/ci.yml` (or equivalent), `scripts/test-integration.sh`.

**Acceptance Criteria:**
- [ ] Dedicated CI job `test-integration` runs `go test -tags=integration ./internal/modules/documents_v2/approval/repository/...` with docker-compose postgres up.
- [ ] Job gates merge to main (required status check).
- [ ] Local helper script `scripts/test-integration.sh` mirrors CI invocation for dev parity.
- [ ] CI logs show integration test names explicitly (no silent skip).

**Steps:**

- [ ] **Step 1: Write/modify CI YAML** (Sonnet — YAML boilerplate).

- [ ] **Step 2: Write `scripts/test-integration.sh`**:

```bash
#!/usr/bin/env bash
set -euo pipefail
docker compose up -d db
until docker exec metaldocs-db pg_isready -U metaldocs; do sleep 1; done
go test -tags=integration -race -count=1 ./internal/modules/documents_v2/approval/repository/...
```

- [ ] **Step 3: Verify CI run shows "PASS: TestListScheduledDue_SkipLocked" in logs** (or explicit test name).

- [ ] **Step 4: Commit**

```bash
rtk git add .github/workflows/ci.yml scripts/test-integration.sh
rtk git commit -m "ci(spec2/phase4): integration-tagged test job (F7)"
```

---

## Phase 5: Application Services (submit, decision, publish, scheduler, supersede, obsolete)

**Intent:** Own tx lifecycle for every state transition via `RunInMembershipTx` (Phase 3). Compose domain rules (Phase 2) + repo (Phase 4) + content_hash/idempotency/signature (Phase 3) + governance_events outbox. **Legacy cutover happens here** — remove `legacy_adapter.go` stubs, strip legacy service paths. Handlers in Phase 7 call these services only; they never touch tx or repo directly.

**Codex review:** COVERAGE (Task 5.12).

**Opus review:** none (Phase 4 and 6 flank it; cross-checked at Phase 6).

**Codex Round 1 (COVERAGE) verdict:** APPROVE_WITH_FIXES, `upgrade_required=true`, 7 findings (1 critical, 3 high, 2 medium, 1 low). All applied inline:

| # | Severity | Fix |
|---|---|---|
| C1 | critical | Idempotency contract extended to EVERY mutator + scheduler row; §"Idempotency matrix" below |
| C2 | high | Scheduler tx composition: short SELECT-claim tx + per-row transition tx. No nested tx. Task 5.7 rewritten. |
| C3 | high | Event payload contracts table added — required fields per event type |
| C4 | high | Per-method transition matrix added (allowed `from`, resulting `to`, error on invalid/stale) |
| C5 | medium | Signoff uniqueness: `(instance_id, stage_instance_id, actor_user_id)` — DB unique + service replay rule |
| C6 | medium | Obsolete cascade: topological BFS, deterministic child-ID lock order, cycle-break, per-node events |
| C7 | low | Legacy cutover build-time check: `go vet` + `staticcheck` + DI graph reflect check, not just grep |

### Idempotency matrix (C1)

| Service method | Idempotency key scope | Replay response | Event dedupe |
|---|---|---|---|
| `SubmitRevisionForReview` | `(tenant, revision_id, actor, "submit", client_key)` | Return stored `instance_id` with `WasReplay=true`; no new event | Event key = `approval_instance.idempotency_key` |
| `RecordSignoff` (approve/reject) | `(tenant, instance_id, stage_instance_id, actor, client_key)` | Return stored `signoff_id` + `WasReplay=true`; no re-transition; no new event | Event key = `signoff.idempotency_key` |
| `PublishApproved` | `(tenant, doc_id, expected_revision_version, "publish")` | Doc already at `published` with same `effective_from` → replay; else `ErrStaleRevision` | `doc_id + revision_version + "publish"` |
| `SchedulePublish` | `(tenant, doc_id, expected_revision_version, "schedule", effective_from)` | Same rule | `doc_id + revision_version + "schedule"` |
| `PublishSuperseding` | `(tenant, new_id, new_v, old_id, old_v, "supersede")` | Replay if both at target states; else `ErrStaleRevision` | Two events keyed per doc + version |
| `MarkObsolete` | `(tenant, doc_id, expected_revision_version, "obsolete")` | Replay if already obsolete at same `effective_to`; cascade keyed per child | Per-node dedupe |
| `SchedulerService.Run` row | `(doc_id, revision_version, "scheduled_publish")` | Row already `published` → skip, no event | Same as PublishApproved event key |

**Outbox dedupe backstop:** `governance_events` gains `UNIQUE (event_type, dedupe_key)` partial index. **Amend Migration 0139 in Phase 1** to add this column + index — flagged as Phase-1 amendment in Task 5.0 below; checksum restate required (see `docs/superpowers/plans/migration-amendments.md`).

### Per-method transition matrix (C4)

| Method | Allowed `from` | Resulting `to` | Error on invalid `from` | Error on stale version |
|---|---|---|---|---|
| `SubmitRevisionForReview` | `draft` | `under_review` | `ErrIllegalTransition` | `ErrStaleRevision` |
| `RecordSignoff` approve (non-final stage) | inst `under_review` + stage `active` | stage `passed` + next `active` | `ErrStageNotActive`/`ErrInstanceCompleted` | `ErrStaleRevision` |
| `RecordSignoff` approve (final stage) | same | inst `approved`, doc `under_review→approved` | same | `ErrStaleRevision` |
| `RecordSignoff` reject | same | inst `rejected`, doc `under_review→rejected` | same | `ErrStaleRevision` |
| `PublishApproved` | doc `approved` | `published` | `ErrIllegalTransition` | `ErrStaleRevision` |
| `SchedulePublish` | doc `approved` | `scheduled` | `ErrIllegalTransition` | `ErrStaleRevision` |
| `SchedulerService.Run` row | doc `scheduled` + `effective_from<=now` | `published` | skip + log (no error) | skip + log |
| `PublishSuperseding` new | doc `approved` | `published` | `ErrIllegalTransition` | `ErrStaleRevision` |
| `PublishSuperseding` old | doc `published` | `superseded` | `ErrIllegalTransition` | `ErrStaleRevision` |
| `MarkObsolete` root | `approved`/`scheduled`/`published`/`superseded` | `obsolete` (`draft`/`under_review`/`rejected`/`obsolete` blocked) | `ErrIllegalTransition` | `ErrStaleRevision` |
| `MarkObsolete` cascade child | any non-obsolete | `obsolete` | skipped + logged | skipped + logged |

Writes to terminal states (`obsolete`, `rejected`) from any method → `ErrIllegalTransition` unconditionally.

### Event payload contract (C3)

All `governance_events` rows share envelope: `{id, tenant_id, actor_user_id, event_type, occurred_at, dedupe_key, correlation_id, payload jsonb}`.

| `event_type` | Required payload keys |
|---|---|
| `doc.approval.submitted` | `instance_id, document_v2_id, revision_id, from_revision_version, to_revision_version, route_config_id, stage_count, content_hash, idempotency_key` |
| `doc.approval.signoff_recorded` | `signoff_id, instance_id, stage_instance_id, decision, actor_user_id, signature_method, content_hash, idempotency_key, reason?` |
| `doc.approval.stage_passed` | `instance_id, stage_instance_id, stage_index, effective_denominator, quorum_rule` |
| `doc.approval.stage_failed` | `instance_id, stage_instance_id, stage_index, rejector_user_id, reason` |
| `doc.approval.stage_activated` | `instance_id, stage_instance_id, stage_index, member_count, quorum_rule` |
| `doc.approval.approved` | `instance_id, document_v2_id, from_revision_version, to_revision_version, content_hash` |
| `doc.approval.rejected` | `instance_id, document_v2_id, from_revision_version, to_revision_version, rejector_user_id, reason` |
| `doc.published` | `document_v2_id, revision_id, from_revision_version, to_revision_version, effective_from, reason` (`immediate`/`scheduled`/`supersede`) |
| `doc.approval.scheduled` | `document_v2_id, from_revision_version, to_revision_version, effective_from` |
| `doc.superseded` | `document_v2_id, revision_id, from_revision_version, to_revision_version, superseded_by_document_v2_id, effective_to` |
| `doc.obsolete` | `document_v2_id, from_revision_version, to_revision_version, reason, cascaded_from?` |
| `doc.approval.instance_cancelled` | `instance_id, reason, cancelled_by_user_id` |

Validation helper `domain.ValidateEventPayload(type, payload)` — added in Task 5.1 scope.

**Subpackages:**
- `internal/modules/documents_v2/approval/application/` — new services
- `internal/modules/documents_v2/application/service.go` — legacy strip

**Invariants this phase encodes:**

1. **One service method = one tx**, opened via `membership_tx.RunInMembershipTx(ctx, db, tenantID, actorUserID, opts, func(tx) error)`. Never expose `*sql.Tx` to callers. Commit implicit on nil return; rollback on error.
2. **Event emission is same-tx.** Every state transition writes to `governance_events` before commit. Repo helper `InsertGovernanceEvent(tx, evt)` called from service, not repo-internal, so service controls event typing.
3. **OCC is caller-provided.** Every service method takes `expectedRevisionVersion int64`; `ErrStaleRevision` propagates to handler as 409.
4. **Idempotency key is caller-provided.** Handler extracts from `Idempotency-Key` header; service forwards to `ComputeIdempotencyKey` from Phase 3; replay returns prior result unchanged.
5. **Clock injection.** `Clock interface { Now() time.Time }` — real impl `realClock{}`, test impl `fakeClock{t time.Time}`. No `time.Now()` in service bodies.
6. **Legacy cutover asserted** (Task 5.10): `git grep -nE "'finalized'|'archived'|legacy_adapter" internal/modules/documents_v2/` returns zero matches after this phase. Phase 4 stubs deleted.
7. **Error contract.** Services return domain errors (`domain.ErrNotAuthor`, `domain.ErrSoDViolation`) or repo errors (`repo.ErrStaleRevision`) unchanged — HTTP mapping is Phase 7's job.

---

### Task 5.0 — Phase-1 amendment: governance_events dedupe + signoff uniqueness (C1, C5)

**Why here:** Phase 1 already committed. Amend Migration 0139 with additive DDL (new migration 0141) rather than rewriting history. Checksum for 0139 unchanged; 0141 recorded in `docs/superpowers/plans/migration-amendments.md`.

**Files:** `supabase/migrations/0141_governance_events_dedupe_signoff_uniqueness.sql`.

**Acceptance Criteria:**
- [ ] `ALTER TABLE metaldocs.governance_events ADD COLUMN dedupe_key TEXT`, `ADD COLUMN correlation_id TEXT`.
- [ ] `CREATE UNIQUE INDEX ux_gov_events_dedupe ON metaldocs.governance_events (event_type, dedupe_key) WHERE dedupe_key IS NOT NULL`.
- [ ] `ALTER TABLE metaldocs.approval_signoffs ADD CONSTRAINT ux_signoff_stage_actor UNIQUE (stage_instance_id, actor_user_id)` — enforces C5 at stage scope (Phase 4 had instance-scope; tighten to stage since members belong to specific stages).
- [ ] Existing `(approval_instance_id, actor_user_id)` unique from Phase 1 retained (superset guarantee — blocks cross-stage re-sign).
- [ ] Migration idempotent (`IF NOT EXISTS` on index, `DO $$ IF NOT EXISTS` on constraint).
- [ ] `docs/superpowers/plans/migration-amendments.md` created/updated noting 0141 as Phase-5-driven amendment.

**Steps:**
- [ ] **Step 1: Write SQL** (Codex high — schema safety).
- [ ] **Step 2: Apply locally, verify with** `rtk psql -c "\d metaldocs.governance_events" -c "\d metaldocs.approval_signoffs"`.
- [ ] **Step 3: Commit**

```bash
rtk git add supabase/migrations/0141_governance_events_dedupe_signoff_uniqueness.sql docs/superpowers/plans/migration-amendments.md
rtk git commit -m "feat(spec2/phase5): 0141 gov_events dedupe + signoff stage uniqueness (C1,C5)"
```

---

### Task 5.1 — Service package scaffold + shared types

**Files:** `internal/modules/documents_v2/approval/application/service.go`, `clock.go`, `events.go`.

**Acceptance Criteria:**
- [ ] `Clock` interface + `realClock`, `fakeClock`.
- [ ] `EventPublisher` interface with `Publish(tx *sql.Tx, evt domain.GovernanceEvent) error` — thin wrapper over repo insert; lives in `events.go`.
- [ ] `Services` struct aggregates `Submit`, `Decision`, `Publish`, `Scheduler`, `Supersede`, `Obsolete` — wired via constructor `NewServices(db, repo, docRepo, sig signature.Provider, hasher content_hash.Hasher, clock Clock, events EventPublisher) *Services`.
- [ ] **`domain.ValidateEventPayload(eventType string, payload map[string]any) error`** (C3) — checks required keys per event type matrix above; returns `domain.ErrEventPayloadInvalid{MissingKeys}`. Unit-tested with golden vectors per event type.
- [ ] **`events.BuildDedupeKey(eventType, payload) string`** (C1) — deterministic per idempotency matrix above.
- [ ] `go build ./internal/modules/documents_v2/approval/application/` passes.

**Steps:**

- [ ] **Step 1: Write scaffold** (Codex medium). Interfaces only; no method bodies yet.

- [ ] **Step 2: Commit**

```bash
rtk git add internal/modules/documents_v2/approval/application/
rtk git commit -m "feat(spec2/phase5): application service scaffold"
```

---

### Task 5.2 — SubmitService.SubmitRevisionForReview

**Files:** `internal/modules/documents_v2/approval/application/submit_service.go`, `submit_service_test.go`.

**Inputs:** `SubmitInput{TenantID, DocumentV2ID, RevisionID, ExpectedRevisionVersion, RouteConfigID, IdempotencyKey, ActorUserID, ContentHashHex}`.

**Outputs:** `SubmitResult{InstanceID, WasReplay bool}`.

**Acceptance Criteria:**
- [ ] Opens tx via `RunInMembershipTx` with `actor_id=ActorUserID`, `tenant_id=TenantID`.
- [ ] Loads document + revision with `SELECT FOR UPDATE` via `docRepo.LoadForUpdate(tx, tenantID, docID, expectedVersion)`; 0 rows → `ErrStaleRevision`.
- [ ] Validates current status is `draft`; else `domain.ErrIllegalTransition{From:X,To:under_review}`.
- [ ] Loads `RouteConfig` via `docRepo.LoadRouteConfig(tx, routeConfigID)`; validates active + tenant match.
- [ ] Computes `idempotency_key = ComputeIdempotencyKey(tenantID, revisionID, actorUserID, "submit", IdempotencyKey)`; tries insert; if `ErrDuplicateSubmission` → load prior instance, return with `WasReplay=true`.
- [ ] Builds `domain.Instance` + snapshots `StageInstance[]` from `RouteConfig.Stages` (copies quorum, members, drift_policy, expected_actor_count at snapshot time — spec §route snapshot).
- [ ] `repo.InsertInstance` + `repo.InsertStageInstances` in same tx.
- [ ] `docRepo.Transition(tx, docID, expectedVersion, draft→under_review)` — bumps revision_version.
- [ ] `docRepo.Lock(tx, docID, instanceID, actorUserID)` — sets `lock_acquired_at`, `locked_by_instance_id`.
- [ ] Emits `governance_event` type `doc.approval.submitted` with payload `{instance_id, route_config_id, stage_count, content_hash}`.
- [ ] Unit tests: happy path, replay returns same instance_id, wrong-status errors, stale OCC, missing route, non-author submits (allowed — spec permits authors or submitters).
- [ ] **SoD test:** actor_user_id stored as submitter; subsequent signoff in same stage by same user must fail in Task 5.3.

**Steps:**

- [ ] **Step 1: Write service body** (Codex high — tx orchestration sensitive).
- [ ] **Step 2: Write unit tests** with `fakeRepo`, `fakeDocRepo`, `fakeClock` (Codex medium).
- [ ] **Step 3: Run** `rtk go test ./internal/modules/documents_v2/approval/application/ -run TestSubmit`.
- [ ] **Step 4: Commit**

```bash
rtk git add internal/modules/documents_v2/approval/application/submit_service.go internal/modules/documents_v2/approval/application/submit_service_test.go
rtk git commit -m "feat(spec2/phase5): SubmitService + tests"
```

---

### Task 5.3 — DecisionService.RecordSignoff (approve path)

**Files:** `internal/modules/documents_v2/approval/application/decision_service.go`, `decision_service_test.go` (approve cases).

**Inputs:** `SignoffInput{TenantID, InstanceID, StageInstanceID, ActorUserID, Decision(approve), Reason, SignatureToken, ContentHashHex, IdempotencyKey}`.

**Outputs:** `SignoffResult{SignoffID, StageAdvanced bool, InstanceCompleted bool, NextStageID *string, WasReplay bool}`.

**Acceptance Criteria:**
- [ ] Tx via `RunInMembershipTx`.
- [ ] Loads instance + active stage; verifies `stage.status=active`, `instance.status=under_review`.
- [ ] **SoD check** (domain rule from Phase 2): actor ≠ `instance.submitted_by` AND actor has no prior signoff in any stage of this instance → else `domain.ErrSoDViolation`.
- [ ] **Content hash binding:** `ContentHashHex == revision.content_hash`; else `domain.ErrContentHashMismatch`.
- [ ] **Signature verify:** `sig.Verify(ctx, SignatureToken, actorUserID, contentHashHex)`; else `domain.ErrSignatureInvalid`.
- [ ] Stage-membership check: actor_user_id ∈ `stage.members`; else `domain.ErrNotStageMember`.
- [ ] **Signoff uniqueness (C5):** `(stage_instance_id, actor_user_id)` unique enforced by Migration 0141. On ON CONFLICT, load existing signoff; compare 4-tuple `(idempotency_key, content_hash, decision, stage_instance_id)` per Phase 4 F2 — same → `WasReplay=true`; different → `ErrActorAlreadySigned`. Post-terminal instance (`approved`/`rejected`) signoff attempts → `ErrInstanceCompleted`.
- [ ] Insert signoff via `repo.InsertSignoff`; on `ErrActorAlreadySigned` → propagate; on `WasReplay=true` → return replay result without further state change.
- [ ] **Quorum evaluate** (Phase 2 `ComputeEffectiveDenominator` + quorum rule): if satisfied → `UpdateStageStatus(stage, active→passed)` + emit `doc.approval.stage_passed`.
- [ ] If stage passed AND next stage exists → `UpdateStageStatus(next, pending→active)` + emit `doc.approval.stage_activated`.
- [ ] If stage passed AND no next → `UpdateInstanceStatus(instance, under_review→approved, completedAt=now)` + `docRepo.Transition(doc, under_review→approved)` + `docRepo.Unlock(doc)` + emit `doc.approval.approved`.
- [ ] Always emit `doc.approval.signoff_recorded` for the signoff itself.
- [ ] Unit tests: single-signer any_1, m_of_n progression, all_of with drift, replay, SoD violation (submitter signs), SoD violation (cross-stage), signature invalid, content hash mismatch, non-member, stale OCC.

**Steps:**

- [ ] **Step 1: Write service body** (Codex high).
- [ ] **Step 2: Tests** (Codex medium).
- [ ] **Step 3: Commit**

```bash
rtk git add internal/modules/documents_v2/approval/application/decision_service.go internal/modules/documents_v2/approval/application/decision_service_test.go
rtk git commit -m "feat(spec2/phase5): DecisionService approve path + SoD + quorum + tests"
```

---

### Task 5.4 — DecisionService.RecordSignoff (reject path)

**Files:** extend `decision_service.go` + tests.

**Acceptance Criteria:**
- [ ] `Decision=reject` branch: insert reject signoff, `UpdateStageStatus(active→failed)`, `UpdateInstanceStatus(under_review→rejected, completedAt=now)`, `docRepo.Transition(under_review→rejected)`, `docRepo.Unlock`.
- [ ] Reason required; else `domain.ErrReasonRequired`.
- [ ] Emit `doc.approval.stage_failed`, `doc.approval.rejected`.
- [ ] Reject preserves signoff history (no delete) — audit trail intact.
- [ ] Tests: single rejector in any_1 terminates instance; m_of_n one reject terminates (spec: any reject = instance rejected).

**Steps:**

- [ ] **Step 1: Extend service** (Codex medium).
- [ ] **Step 2: Tests** (Codex medium).
- [ ] **Step 3: Commit**

```bash
rtk git add internal/modules/documents_v2/approval/application/decision_service.go internal/modules/documents_v2/approval/application/decision_service_test.go
rtk git commit -m "feat(spec2/phase5): DecisionService reject path + tests"
```

---

### Task 5.5 — PublishService.PublishApproved (immediate)

**Files:** `internal/modules/documents_v2/approval/application/publish_service.go`, `publish_service_test.go`.

**Inputs:** `PublishInput{TenantID, DocumentV2ID, ExpectedRevisionVersion, ActorUserID}`.

**Acceptance Criteria:**
- [ ] Tx via `RunInMembershipTx`.
- [ ] Loads doc; expects status `approved`, else `ErrIllegalTransition`.
- [ ] `docRepo.Transition(approved→published, effective_from=clock.Now())` with OCC.
- [ ] If prior published revision exists for same document chain → mark prior as `superseded` (defer to Task 5.8 if separate method). For 5.5 scope: first publish only; prior-revision supersede handled in 5.8.
- [ ] Emit `doc.published`.
- [ ] Capability gate: actor must have `doc.publish` on doc's area — validated in Phase 6 middleware; service assumes auth done but still calls `authz.Require(tx, "doc.publish", areaCode)` for DB tripwire.
- [ ] Tests: happy, wrong status, stale OCC, capability missing.

**Steps:**

- [ ] **Step 1: Write service** (Codex medium).
- [ ] **Step 2: Tests**.
- [ ] **Step 3: Commit**

```bash
rtk git add internal/modules/documents_v2/approval/application/publish_service.go internal/modules/documents_v2/approval/application/publish_service_test.go
rtk git commit -m "feat(spec2/phase5): PublishService immediate + tests"
```

---

### Task 5.6 — PublishService.SchedulePublish (future effective_from)

**Files:** extend `publish_service.go` + tests.

**Acceptance Criteria:**
- [ ] `SchedulePublishInput{..., EffectiveFrom time.Time}`.
- [ ] Validates `EffectiveFrom > clock.Now()`; else `domain.ErrEffectiveFromNotFuture`.
- [ ] `docRepo.Transition(approved→scheduled, effective_from=EffectiveFrom)` with OCC.
- [ ] Emit `doc.approval.scheduled`.
- [ ] Tests: happy, past date rejected, equal-to-now rejected, wrong source status.

**Steps:**

- [ ] **Step 1: Extend + tests** (Codex medium).
- [ ] **Step 2: Commit**

```bash
rtk git add internal/modules/documents_v2/approval/application/publish_service.go internal/modules/documents_v2/approval/application/publish_service_test.go
rtk git commit -m "feat(spec2/phase5): PublishService scheduled + tests"
```

---

### Task 5.7 — SchedulerService.RunDuePublishes

**Files:** `internal/modules/documents_v2/approval/application/scheduler_service.go`, `scheduler_service_test.go`.

**Inputs:** `Run(ctx, batchLimit int) (processed, skipped int, err error)`.

**Composition model (C2 — no nested tx):**

1. **Claim tx (short, READ COMMITTED):** `repo.ListScheduledDue(tx, now, limit)` with `FOR UPDATE SKIP LOCKED` — returns rows + row-level locks. Immediately mark each row `status='publishing'` (new intermediate status — **amend Migration 0141 to add** per Task 5.0) with `claimed_at=now()`, `claimed_by='scheduler:<hostname>'`. Commit claim tx — locks released but intermediate status blocks re-claim.
2. **Per-row transition tx (one per row):** for each claimed row, open independent tx via `RunInMembershipTx(ctx, db, tenantID, "system:scheduler", opts, func(tx) {...})`:
   - Re-load doc with `FOR UPDATE` + OCC on `revision_version` captured from claim.
   - If status still `publishing` → `docRepo.Transition(publishing→published, effective_from unchanged)`.
   - Emit `doc.published` with `reason="scheduled"`.
   - Commit.
3. **Error handling per row:**
   - OCC stale → log + increment `skipped`; do NOT fail batch; another scheduler instance will retry via stuck-instance watchdog (Phase 8).
   - Transition error → `UpdateStatus(publishing→scheduled, effective_from unchanged)` to release (new migration DDL needed for this reverse edge — included in 0141 amend).
   - Success → increment `processed`.

**Acceptance Criteria:**
- [ ] Claim tx explicitly `sql.LevelReadCommitted` + `FOR UPDATE SKIP LOCKED`.
- [ ] Per-row tx is a separate `RunInMembershipTx` — NEVER called from inside claim tx. Invariant 1 (one-method-one-tx) holds at per-row granularity.
- [ ] `PublishService.PublishApproved` NOT called from scheduler (would be nested tx + wrong source state). Scheduler uses dedicated internal repo method `TransitionScheduledToPublished`.
- [ ] Intermediate status `publishing` added to `documents.status` CHECK (Migration 0141 amend — Task 5.0); legal transitions `scheduled→publishing`, `publishing→published`, `publishing→scheduled` (rollback on failure).
- [ ] Governance_events for rollback (`publishing→scheduled`) emitted with `event_type=doc.approval.scheduled_publish_retry`.
- [ ] `"system:scheduler"` user exists in `iam_users` — verify in Step 0.
- [ ] Tests: empty due set returns (0,0); 3 due rows all publish (3,0); one row with stale OCC mid-batch → (2,1); SKIP LOCKED isolation (two concurrent runs; combined processed=5 for 5 due rows; integration probe 5.11-D).

**Steps:**

- [ ] **Step 0: Verify `system:scheduler` user exists** — `rtk grep "system:scheduler" supabase/migrations/`. If missing, add to Migration 0130 now as amend + restate Phase 1 checksum. (Haiku).
- [ ] **Step 1: Write service** (Codex high — concurrency-sensitive).
- [ ] **Step 2: Unit tests with fake repo**.
- [ ] **Step 3: Commit**

```bash
rtk git add internal/modules/documents_v2/approval/application/scheduler_service.go internal/modules/documents_v2/approval/application/scheduler_service_test.go
rtk git commit -m "feat(spec2/phase5): SchedulerService + tests"
```

---

### Task 5.8 — SupersedeService.PublishSuperseding

**Files:** `internal/modules/documents_v2/approval/application/supersede_service.go`, tests.

**Inputs:** `SupersedeInput{TenantID, NewDocumentV2ID, NewExpectedRevisionVersion, SupersededDocumentV2ID, SupersededExpectedRevisionVersion, ActorUserID}`.

**Acceptance Criteria:**
- [ ] Single tx covers both documents.
- [ ] Validates new doc status = `approved`; superseded doc status = `published`; both share same `document_chain_id` (or same `family_id` — per Spec 2 chain model).
- [ ] `docRepo.Transition(new, approved→published, effective_from=now)` + `docRepo.Transition(old, published→superseded, effective_to=now)`.
- [ ] Both OCC-checked; either stale → whole tx rolls back.
- [ ] Emit `doc.published` (new) + `doc.superseded` (old) in tx.
- [ ] Tests: happy, wrong chain, stale OCC on old, stale OCC on new, old not published.

**Steps:**

- [ ] **Step 1: Write service** (Codex high).
- [ ] **Step 2: Tests**.
- [ ] **Step 3: Commit**

```bash
rtk git add internal/modules/documents_v2/approval/application/supersede_service.go internal/modules/documents_v2/approval/application/supersede_service_test.go
rtk git commit -m "feat(spec2/phase5): SupersedeService + tests"
```

---

### Task 5.9 — ObsoleteService.MarkObsolete (+ child cascade)

**Files:** `internal/modules/documents_v2/approval/application/obsolete_service.go`, tests.

**Inputs:** `ObsoleteInput{TenantID, DocumentV2ID, ExpectedRevisionVersion, ActorUserID, Reason}`.

**Acceptance Criteria:**
- [ ] Tx-wrapped.
- [ ] Expected source statuses: `published`, `superseded`, `scheduled`, `approved`. Not `draft` / `under_review` / `rejected` / `obsolete` — else `ErrIllegalTransition`.
- [ ] `docRepo.Transition(X→obsolete, effective_to=now)`.
- [ ] **Cascade algorithm (C6):** BFS traversal of `document_relationships` (`relationship_type='depends_on'`) rooted at `docID`. Deterministic ordering: visit children sorted by `child_document_v2_id ASC` for reproducible lock acquisition order → prevents deadlock with concurrent cascades.
- [ ] **Cycle detection:** visited-set keyed on `document_v2_id`; re-visit → skip (no error). Self-reference (`parent_id=child_id`) → skip.
- [ ] **Depth bound:** max 1000 nodes per cascade; exceed → `domain.ErrCascadeTooLarge` (fail-safe; caller can split).
- [ ] **Per-node lock:** `SELECT FOR UPDATE` child in sorted order inside same tx. Each node: if status ∈ `{draft,under_review,approved,scheduled,published,superseded}` → `docRepo.Transition(child→obsolete, effective_to=now)` + emit `doc.obsolete` with `cascaded_from=parent_id`; if already `obsolete` or `rejected` → skip + log (no event).
- [ ] If any node transition fails OCC mid-cascade → whole tx rolls back; caller retries with fresh expected versions.
- [ ] Per-node events use dedupe_key `child_id + revision_version + "obsolete"` — idempotent on retry.
- [ ] Emit `doc.obsolete` for parent.
- [ ] Tests: parent-only (no children), parent + 2 children, child already obsolete (skipped), cascade stops on stale child OCC.

**Steps:**

- [ ] **Step 1: Write service** (Codex high).
- [ ] **Step 2: Tests**.
- [ ] **Step 3: Commit**

```bash
rtk git add internal/modules/documents_v2/approval/application/obsolete_service.go internal/modules/documents_v2/approval/application/obsolete_service_test.go
rtk git commit -m "feat(spec2/phase5): ObsoleteService + cascade + tests"
```

---

### Task 5.10 — Legacy cutover: delete adapter + strip legacy paths

**Files:** delete `internal/modules/documents_v2/approval/repository/legacy_adapter.go`; modify `internal/modules/documents_v2/application/service.go`, `internal/modules/documents_v2/domain/model.go`, `internal/modules/documents_v2/repository/repository.go`.

**Acceptance Criteria:**
- [ ] `legacy_adapter.go` deleted.
- [ ] All callers of `ErrLegacyMethodRemoved` stubs replaced by Phase 5 service calls.
- [ ] `rtk grep -nE "finalized|archived|legacy_adapter" internal/modules/documents_v2/` — zero matches (excluding test fixtures that reference `legacy_status_remap` event name, which is historical).
- [ ] `rtk grep -n "ErrLegacyMethodRemoved" internal/` — zero matches.
- [ ] `rtk go build ./...` passes.
- [ ] `rtk go vet ./...` passes.
- [ ] **`rtk staticcheck ./...`** passes — catches unreachable legacy code beyond grep.
- [ ] **DI graph check (C7):** `rtk go test ./internal/app/ -run TestContainerNoLegacyAdapter` — reflects over DI container struct fields; fails build if any field type name matches `.*Legacy.*` or `.*Adapter.*` in `documents_v2` package. Prevents runtime wiring leftovers.
- [ ] **Integration probe 5.11-G (new):** submit a doc; verify no HTTP handler path or cron job path invokes `ErrLegacyMethodRemoved` or references a `documents.status` value outside Spec 2 vocabulary. Scan covers all registered routes + cron entries.
- [ ] Invariant 7 of Phase 4 holds — no approval write tx opened outside Phase 5 services.

**Steps:**

- [ ] **Step 1: Delete adapter + strip callers** (Codex medium — mechanical).
- [ ] **Step 2: Verify greps**.
- [ ] **Step 3: Commit**

```bash
rtk git rm internal/modules/documents_v2/approval/repository/legacy_adapter.go
rtk git add internal/modules/documents_v2/
rtk git commit -m "refactor(spec2/phase5): legacy cutover — remove adapter + legacy status paths"
```

---

### Task 5.11 — Integration probes (service-level)

**Files:** `internal/modules/documents_v2/approval/application/integration_test.go` (build tag `integration`).

**Acceptance Criteria:**
- [ ] Uses real Postgres via `testdb` helper + Phase 1 migrations applied.
- [ ] **Probe A — happy path:** submit → stage1 approve (any_1) → auto-advance → stage2 approve (all_of 2-of-2) → instance approved → publish → doc.status=published in DB.
- [ ] **Probe B — reject terminates:** submit → stage1 reject → instance rejected → doc.status=rejected → unlocked.
- [ ] **Probe C — SoD trigger fires:** submit as user A → user A tries signoff → DB-side SoD trigger raises `ErrCheckViolation` with reason containing `sod_submitter`. Verifies DB backstop independent of service check.
- [ ] **Probe D — scheduler SKIP LOCKED:** 5 scheduled rows; two `SchedulerService.Run` goroutines in parallel; combined processed count = 5; no row double-published.
- [ ] **Probe E — supersede chain:** doc v1 published → doc v2 approved → supersede → v1.status=superseded, v2.status=published, effective_to on v1 = now, effective_from on v2 = now.
- [ ] **Probe F — obsolete cascade:** parent + 2 children published; `MarkObsolete(parent)` → all 3 obsolete; `governance_events` has 3 rows with `cascaded_from`.
- [ ] CI job from Task 4.12 picks these up automatically (same build tag).

**Steps:**

- [ ] **Step 1: Write probes** (Codex high).
- [ ] **Step 2: Run `scripts/test-integration.sh` locally**; all 6 pass.
- [ ] **Step 3: Commit**

```bash
rtk git add internal/modules/documents_v2/approval/application/integration_test.go
rtk git commit -m "test(spec2/phase5): service-level integration probes"
```

---

### Task 5.12 — Codex COVERAGE review

**Goal:** Surface missing cases (state transitions not covered, edge errors, idempotency gaps, event payload omissions).

**Acceptance Criteria:**
- [ ] Codex `gpt-5.3-codex` reasoning=high mode=COVERAGE called with explicit plan-review framing ("deliverable files do not exist yet — that is correct; review the PLAN for gaps").
- [ ] Verdict JSON saved to `docs/superpowers/plans/reviews/phase-5-round-1.json`.
- [ ] All `APPROVE_WITH_FIXES` fixes applied inline.
- [ ] If `REJECT` or `upgrade_required=true` → one Round 2 max, then proceed with caveats.

**Steps:**

- [ ] **Step 1: Extract Phase 5 markdown** to `C:\tmp\spec2_phase5_review.md`.
- [ ] **Step 2: Call Codex MCP** with COVERAGE prompt.
- [ ] **Step 3: Apply fixes to plan inline**.
- [ ] **Step 4: Save verdict JSON**.
- [ ] **Step 5: Commit plan delta**

```bash
rtk git add docs/superpowers/plans/2026-04-21-foundation-doc-approval-state-machine.md docs/superpowers/plans/reviews/phase-5-round-1.json
rtk git commit -m "chore(spec2/phase5): Codex COVERAGE review applied"
```

---

## Phase 6: IAM (role_capabilities v2, area_membership fns, workflow cancel)

**Intent:** Wire Spec 2's new capabilities (`doc.submit`, `doc.signoff`, `doc.publish`, `doc.supersede`, `doc.obsolete`, `workflow.instance.cancel`, `route.admin`) into `role_capabilities`; bump capability schema `v1→v2`; route all area-membership writes through canonical SECURITY DEFINER DB functions (no ad-hoc INSERTs); add authorization middleware for services.

**Codex review:** ARCHITECTURE (Task 6.10).

**Opus review:** phase end — cross-check capability wiring, role boundaries, area-membership fn invariants.

**Codex Round 1 (ARCHITECTURE) verdict:** APPROVE_WITH_FIXES, `upgrade_required=true`, 6 findings (3 high, 3 medium). All applied inline:

| # | Sev | Fix |
|---|---|---|
| A1 | high | Tripwire flag is per-tuple `{capability, area_id}` not boolean; mutators verify exact tuple |
| A2 | high | v1→v2 cutover choreographed: 0142a additive → code rollout → 0142b enforcement |
| A3 | high | SECURITY DEFINER hardening: owner role, SET search_path, REVOKE FROM PUBLIC, GRANT writer only, strict arg validation |
| A4 | medium | Scheduler bypass `SET LOCAL` in tx only; session-level forbidden; Probe I added |
| A5 | medium | Cancel state model → new Migration 0144 (instance/stage cancelled + transition trigger) sequenced before Task 6.6 |
| A6 | medium | Route immutability via DB trigger + referenced-check lock, not app-only |

### Rollout choreography (A2)

Phase 6 migrations split into two ordered steps:

1. **0142a — additive:** INSERT new caps, add fns, add tripwire tables. NO legacy strip, NO CHECK constraint. Old binaries still work (v1 caps present).
2. **Code rollout window:** new binary deployed fleet-wide; reads v2; emits v2 events.
3. **0142b — enforcement:** DELETE legacy caps, ADD CHECK preventing legacy, enable tripwire trigger. CI gate reads `/version` from all instances before applying.

Rollback: 0142b has `0142b_down.sql` re-inserting legacy caps + dropping CHECK. 0142a is additive, not rolled back.

**Subpackages:**
- `supabase/migrations/0142a_role_capabilities_v2_additive.sql` (new caps, tripwire table — additive)
- `supabase/migrations/0142b_role_capabilities_v2_enforce.sql` (legacy strip + CHECK + tripwire trigger enable)
- `internal/modules/iam/authz/authz.go` (capability check helper — service-level)
- `internal/modules/iam/area_membership/` (Go wrapper over SECURITY DEFINER fns)
- `internal/modules/documents_v2/approval/application/` (authz calls inserted)

**Invariants this phase encodes:**

1. **Capability schema v2 is authoritative.** v1 rows remapped in migration; no dual-read.
2. **All area-membership writes go through `metaldocs.grant_area_membership(...)` and `metaldocs.revoke_area_membership(...)`.** Direct `INSERT INTO user_process_areas` is blocked by `REVOKE INSERT` on writer role (tripwire).
3. **Capability check ordering:** handler auth middleware (HTTP) → service-level `authz.Require(tx, cap, areaCode)` inside every mutator tx (DB tripwire via GUC). Double gate: app + DB.
4. **Cancel is a distinct capability.** `workflow.instance.cancel` separate from `doc.signoff` — only QMS admins / route owners can cancel mid-flight.
5. **Role boundary NOINHERIT** enforced by Spec 1 already; Phase 6 does not touch role grants.
6. **Legacy capability names removed.** `document.finalize`, `document.archive` stripped from `role_capabilities`; any row referencing them fails migration (error, not silent drop — explicit review gate).

### Capability matrix v2

| Capability | Granted to (default roles) | Scope | DB function gate |
|---|---|---|---|
| `doc.view` | viewer, author, reviewer, signer, area_admin, qms_admin | area | — |
| `doc.edit` | author, area_admin | area | — |
| `doc.submit` | author, area_admin | area | `authz.require_capability('doc.submit', area)` |
| `doc.signoff` | signer, reviewer, area_admin, qms_admin | area | `authz.require_capability('doc.signoff', area)` |
| `doc.publish` | area_admin, qms_admin | area | `authz.require_capability('doc.publish', area)` |
| `doc.supersede` | area_admin, qms_admin | area | `authz.require_capability('doc.supersede', area)` |
| `doc.obsolete` | qms_admin | area | `authz.require_capability('doc.obsolete', area)` |
| `workflow.instance.cancel` | qms_admin, route_owner | route/area | `authz.require_capability('workflow.instance.cancel', area)` |
| `route.admin` | qms_admin | global (tenant) | `authz.require_capability('route.admin', 'tenant')` |
| `membership.grant` | area_admin, qms_admin | area | `authz.require_capability('membership.grant', area)` |
| `membership.revoke` | area_admin, qms_admin | area | `authz.require_capability('membership.revoke', area)` |

---

### Task 6.1a — Migration 0142a: additive caps + tripwire table (A2)

**Files:** `supabase/migrations/0142a_role_capabilities_v2_additive.sql`.

**Acceptance Criteria:**
- [ ] Bumps `metaldocs.role_capabilities_schema_version` 1 → 2 (soft — old binaries tolerate).
- [ ] Inserts all new caps from matrix with `ON CONFLICT (role, capability) DO NOTHING`.
- [ ] Creates `metaldocs.tx_capability_assertions(tx_id TEXT, capability TEXT, area_id TEXT, asserted_at timestamptz, PRIMARY KEY(tx_id, capability, area_id))` — temporary-table-backed via `ON COMMIT DROP` semantics; actual impl is per-tx via `SET LOCAL metaldocs.asserted_caps='{cap:area,cap:area}'` JSONB GUC.
- [ ] NO legacy delete, NO CHECK constraint, NO trigger enable — additive only.
- [ ] Idempotent.

**Steps:** Codex high; apply; verify counts. Commit `feat(spec2/phase6): 0142a additive caps + tripwire scaffold`.

---

### Task 6.1b — Migration 0142b: legacy strip + enforcement (post-rollout)

**Files:** `supabase/migrations/0142b_role_capabilities_v2_enforce.sql`, `0142b_down.sql`.

**Acceptance Criteria:**
- [ ] DELETE legacy caps `document.finalize`, `document.archive`; RAISES NOTICE row count.
- [ ] ADD `CHECK (capability NOT IN ('document.finalize','document.archive'))` on `role_capabilities`.
- [ ] ADD CHECK `capability ~ '^[a-z][a-z._]*[a-z]$'`.
- [ ] Enable tripwire trigger `trg_require_capability_asserted` on all protected tables (`approval_instances`, `approval_stage_instances`, `approval_signoffs`, `documents` status updates). Trigger reads GUC `metaldocs.asserted_caps`; raises `ErrCapabilityNotAsserted` if required `{cap,area}` tuple missing.
- [ ] `0142b_down.sql` re-inserts legacy caps + drops CHECK + disables trigger.
- [ ] **CI gate:** deployment tool verifies `/version` on all fleet instances reports binary ≥ v2-enabled before applying 0142b.

**Steps:** Codex high. Commit `feat(spec2/phase6): 0142b enforcement + tripwire trigger`.

---

### Task 6.2 — SECURITY DEFINER canonical area_membership fns

**Files:** `supabase/migrations/0143_area_membership_fns.sql`.

**Acceptance Criteria:**
- [ ] **SECURITY DEFINER hardening (A3):** every fn declares `SECURITY DEFINER`, `SET search_path = metaldocs, pg_temp`, owned by `metaldocs_admin`; `REVOKE EXECUTE ON FUNCTION ... FROM PUBLIC`; `GRANT EXECUTE ... TO metaldocs_writer` only. Inside fn: validate all TEXT args with regex (`_user_id ~ '^[a-z0-9_.-]+$'`, `_area_code ~ '^[A-Z0-9_]+$'`, `_role IN (enum values)`), reject on mismatch with `RAISE EXCEPTION 'invalid arg: %', _arg USING ERRCODE='22023'`.
- [ ] `metaldocs.grant_area_membership(_tenant_id uuid, _user_id text, _area_code text, _role text, _granted_by text) RETURNS uuid` — SECURITY DEFINER, OWNER=metaldocs_admin, LANGUAGE plpgsql.
- [ ] Validates: granter has `membership.grant` cap on area; user+area exist; role ∈ enum.
- [ ] Writes `user_process_areas` + `governance_events` row `{event_type:'membership.granted', actor_user_id:_granted_by, payload:{user_id,area,role}}` atomically.
- [ ] Returns `user_process_areas.id`.
- [ ] Idempotent on `(tenant_id, user_id, area_code, role)` — returns existing id + no new event.
- [ ] `metaldocs.revoke_area_membership(_tenant_id uuid, _user_id text, _area_code text, _role text, _revoked_by text) RETURNS void` — symmetric; soft-delete via `revoked_at`.
- [ ] `REVOKE INSERT, UPDATE, DELETE ON metaldocs.user_process_areas FROM metaldocs_writer`. Writer role can only call these fns.
- [ ] `GRANT EXECUTE ON FUNCTION ... TO metaldocs_writer`.

**Steps:**
- [ ] **Step 1: Write SQL** (Codex high — SECURITY DEFINER sensitive).
- [ ] **Step 2: Integration test (probe)** — writer role direct INSERT fails with 42501; fn call succeeds.
- [ ] **Step 3: Commit**

```bash
rtk git add supabase/migrations/0143_area_membership_fns.sql
rtk git commit -m "feat(spec2/phase6): SECURITY DEFINER area_membership fns"
```

---

### Task 6.3 — Go wrapper: area_membership package

**Files:** `internal/modules/iam/area_membership/area_membership.go`, `_test.go`.

**Acceptance Criteria:**
- [ ] `type AreaMembership struct { ... }` + repo interface.
- [ ] `Grant(ctx, tx, tenantID, userID, areaCode, role, grantedBy string) (membershipID string, err error)` — wraps `SELECT metaldocs.grant_area_membership(...)`.
- [ ] `Revoke(ctx, tx, tenantID, userID, areaCode, role, revokedBy string) error`.
- [ ] `List(ctx, tx, tenantID, userID string) ([]Membership, error)` — reads `user_process_areas` + `iam_users` join.
- [ ] Errors mapped via `MapPgError` from Phase 4 (42501 → `ErrInsufficientPrivilege`, 23514 → `ErrInvalidRole` etc).
- [ ] Unit tests with fake DB; integration test (tag `integration`) exercising real fn.

**Steps:**
- [ ] **Step 1: Write wrapper** (Codex medium).
- [ ] **Step 2: Tests**.
- [ ] **Step 3: Commit**

```bash
rtk git add internal/modules/iam/area_membership/
rtk git commit -m "feat(spec2/phase6): area_membership Go wrapper"
```

---

### Task 6.4 — authz package (service-level capability check)

**Files:** `internal/modules/iam/authz/authz.go`, `_test.go`.

**Acceptance Criteria:**
- [ ] `authz.Require(ctx, tx, cap, areaCode) error` — queries `role_capabilities` join `user_process_areas` for `actor_id` (from tx GUC) + `tenant_id` (from tx GUC) + `areaCode`. Missing cap → `ErrCapabilityDenied{cap, areaCode, actor}`.
- [ ] `areaCode="tenant"` means tenant-scope (global); query ignores area join.
- [ ] Result cached per-tx via `ctx.Value(txCapCacheKey)` — avoids repeat queries within one service method.
- [ ] **On success, `Require` SET LOCAL-appends** `{capability, area_id}` tuple to GUC `metaldocs.asserted_caps` (JSONB array). Format: `[{"cap":"doc.submit","area":"PROD"},...]`. Per-tx scope — `SET LOCAL` ensures auto-reset on tx end.
- [ ] **DB-side tripwire (A1):** mutating triggers (enabled by Migration 0142b) read `metaldocs.asserted_caps` GUC + look up the protected row's area_code + required capability from a per-table metadata map. Require EXACT tuple match `{cap, area}`. Missing → `RAISE EXCEPTION 'ErrCapabilityNotAsserted: need % for area %', cap, area USING ERRCODE='P0001'`.
- [ ] Per-table required-cap map (in 0142b): `approval_instances INSERT → doc.submit`, `approval_signoffs INSERT → doc.signoff`, `approval_instances UPDATE status → varies by to-status` (approved/rejected via signoff; cancelled via workflow.instance.cancel), `documents UPDATE status=published → doc.publish or (scheduler bypass)`, `documents UPDATE status=obsolete → doc.obsolete`, etc.
- [ ] Trigger honors scheduler bypass: if GUC `metaldocs.bypass_authz='scheduler'` set via `SET LOCAL`, trigger skips assertion check but logs to `governance_events` with `event_type=authz.bypass_used`.
- [ ] Tests: cap granted → ok; cap denied → error; cross-tenant check; cache hit.

**Steps:**
- [ ] **Step 1: Write authz** (Codex high).
- [ ] **Step 2: Write `assert_capability_called` helper in 0143**.
- [ ] **Step 3: Tests**.
- [ ] **Step 4: Commit**

```bash
rtk git add internal/modules/iam/authz/ supabase/migrations/0143_area_membership_fns.sql
rtk git commit -m "feat(spec2/phase6): authz.Require + DB capability tripwire"
```

---

### Task 6.5 — Wire authz into Phase 5 services

**Files:** modify all services in `internal/modules/documents_v2/approval/application/`.

**Acceptance Criteria:**
- [ ] `SubmitService` calls `authz.Require(tx, "doc.submit", doc.areaCode)` before writes.
- [ ] `DecisionService.RecordSignoff` calls `authz.Require(tx, "doc.signoff", doc.areaCode)`.
- [ ] `PublishService` (both methods) calls `authz.Require(tx, "doc.publish", doc.areaCode)`.
- [ ] `SupersedeService` calls `authz.Require(tx, "doc.supersede", doc.areaCode)`.
- [ ] `ObsoleteService` calls `authz.Require(tx, "doc.obsolete", doc.areaCode)`.
- [ ] `SchedulerService.Run` skips authz (actor=`system:scheduler`). Bypass via `authz.BypassSystem(tx)` calls `SET LOCAL metaldocs.bypass_authz='scheduler'` — **SET LOCAL only, never SET SESSION** (A4). Bypass auto-reset on tx end. Each bypass use writes `authz.bypass_used` governance event.
- [ ] Unit tests updated: each service test covers `ErrCapabilityDenied` path.

**Steps:**
- [ ] **Step 1: Insert authz calls** (Codex medium — mechanical).
- [ ] **Step 2: Update tests**.
- [ ] **Step 3: Commit**

```bash
rtk git add internal/modules/documents_v2/approval/application/
rtk git commit -m "feat(spec2/phase6): authz gates on all approval services"
```

---

### Task 6.5a — Migration 0144: cancel state model (A5)

**Files:** `supabase/migrations/0144_cancel_state.sql`.

**Acceptance Criteria:**
- [ ] `ALTER TABLE metaldocs.approval_instances` CHECK on `status` column extended to include `'cancelled'`.
- [ ] `ALTER TABLE metaldocs.approval_stage_instances` CHECK on `status` extended to include `'cancelled'`.
- [ ] Legal-transition trigger on `documents` extended: `under_review → draft` allowed IFF related active `approval_instance` is transitioning to `cancelled` in same tx (checked via per-tx flag GUC `metaldocs.cancel_in_progress='<instance_id>'`).
- [ ] Migration applied BEFORE Task 6.6 service goes live (sequenced in phase map).
- [ ] Idempotent.

**Steps:** Codex high. Commit `feat(spec2/phase6): 0144 cancel state model`.

---

### Task 6.6 — CancelInstanceService (workflow.instance.cancel)

**Files:** `internal/modules/documents_v2/approval/application/cancel_service.go`, `_test.go`.

**Inputs:** `CancelInput{TenantID, InstanceID, ExpectedRevisionVersion, ActorUserID, Reason}`.

**Acceptance Criteria:**
- [ ] Tx via `RunInMembershipTx`.
- [ ] `authz.Require(tx, "workflow.instance.cancel", doc.areaCode)`.
- [ ] Instance must be non-terminal (`under_review`); else `ErrInstanceCompleted`.
- [ ] `UpdateInstanceStatus(under_review→cancelled, completedAt=now)`; `UpdateStageStatus(all_active→cancelled)`; `docRepo.Transition(doc, under_review→draft)` (revert to draft, not rejected — per spec cancel semantics); `docRepo.Unlock(doc)`.
- [ ] Emit `doc.approval.instance_cancelled` with payload per Phase 5 matrix.
- [ ] Reason required; non-empty.
- [ ] Tests: happy, already-terminal fails, cap denied fails, stale OCC.
- [ ] **Migration 0142 amend:** ensure `approval_instance.status` CHECK allows `cancelled`; `approval_stage_instance.status` CHECK allows `cancelled`; transition trigger allows `under_review→draft` on doc.

**Steps:**
- [ ] **Step 1: Write service + amend migration** (Codex high).
- [ ] **Step 2: Tests**.
- [ ] **Step 3: Commit**

```bash
rtk git add internal/modules/documents_v2/approval/application/cancel_service.go supabase/migrations/0142_role_capabilities_v2.sql
rtk git commit -m "feat(spec2/phase6): CancelInstanceService + cancelled state"
```

---

### Task 6.7 — RouteConfig admin capability gate

**Files:** `internal/modules/documents_v2/approval/application/route_admin_service.go`, `_test.go`.

**Acceptance Criteria:**
- [ ] `RouteAdminService` with `Create`, `Update`, `Deactivate` methods.
- [ ] `authz.Require(tx, "route.admin", "tenant")` on all three.
- [ ] Immutable-once-referenced: if `approval_instance` references this route_config_id → reject with `ErrRouteInUse` (must create new config; old stays for audit).
- [ ] **DB-enforced (A6):** Migration 0145 adds trigger `trg_route_config_immutable` on `route_configs` UPDATE/DELETE: if `EXISTS (SELECT 1 FROM approval_instances WHERE route_config_id = OLD.id)` → `RAISE EXCEPTION 'ErrRouteInUse'`. Service-level check is fast-path; trigger is race-proof backstop. Lock strategy: service does `SELECT ... FOR UPDATE` on `route_configs.id` before read to serialize against concurrent instance insert.
- [ ] Emit `route.config.created` / `route.config.updated` / `route.config.deactivated` events.
- [ ] Tests: cap, in-use block, update with stages reshape, happy.

**Steps:**
- [ ] **Step 1: Write service** (Codex high).
- [ ] **Step 2: Tests**.
- [ ] **Step 3: Commit**

```bash
rtk git add internal/modules/documents_v2/approval/application/route_admin_service.go
rtk git commit -m "feat(spec2/phase6): RouteAdminService"
```

---

### Task 6.8 — Integration probes (IAM)

**Files:** `internal/modules/iam/integration_test.go` (tag `integration`).

**Acceptance Criteria:**
- [ ] **Probe A:** writer-role direct `INSERT INTO user_process_areas` → SQLSTATE 42501 (permission denied). Confirms fn-only write path.
- [ ] **Probe B:** `grant_area_membership` without `membership.grant` cap → `ErrCapabilityDenied`.
- [ ] **Probe C:** service-level `submit` without `doc.submit` cap → `ErrCapabilityDenied` before any DB write.
- [ ] **Probe D:** service-level `submit` with cap but DB trigger asserts `assert_capability_called` — trigger passes (flag set).
- [ ] **Probe E:** service forgets authz.Require (simulated by stubbed service) → DB trigger raises `ErrCapabilityNotAsserted`. Verifies DB tripwire.
- [ ] **Probe F:** Cancel after terminal → `ErrInstanceCompleted`.
- [ ] **Probe G:** Route-config with active instance → cannot update.
- [ ] **Probe H:** Legacy cap (`document.finalize`) in role_capabilities post-migration → zero rows (migration enforced).
- [ ] **Probe I (A4):** After scheduler tx commits, open new session on same pooled conn; `SHOW metaldocs.bypass_authz` returns empty/default. Confirms SET LOCAL tx-scope. Explicit attempt at `SET SESSION metaldocs.bypass_authz='scheduler'` outside scheduler path → test fails build (forbidden pattern grep in source).
- [ ] **Probe J (A1):** Service calls `authz.Require("doc.submit", "AREA_X")` but then attempts INSERT on approval_instance with `area_code='AREA_Y'` → trigger raises `ErrCapabilityNotAsserted` (tuple mismatch on area). Confirms per-tuple, not boolean, tripwire.

**Steps:**
- [ ] **Step 1: Write probes** (Codex high).
- [ ] **Step 2: Run**.
- [ ] **Step 3: Commit**

```bash
rtk git add internal/modules/iam/integration_test.go
rtk git commit -m "test(spec2/phase6): IAM integration probes"
```

---

### Task 6.9 — Legacy IAM path removal

**Files:** strip any ad-hoc capability strings in `internal/modules/iam/` + remove direct `user_process_areas` writes.

**Acceptance Criteria:**
- [ ] `rtk grep -nE "document\.finalize|document\.archive" internal/` — zero matches.
- [ ] `rtk grep -n "INSERT INTO.*user_process_areas" internal/` — zero matches (all go through fn).
- [ ] `rtk grep -n "UPDATE.*role_capabilities" internal/` — zero matches (only migration writes).
- [ ] Build + vet + staticcheck clean.

**Steps:**
- [ ] **Step 1: Strip** (Codex medium).
- [ ] **Step 2: Verify greps**.
- [ ] **Step 3: Commit**

```bash
rtk git add internal/modules/iam/
rtk git commit -m "refactor(spec2/phase6): legacy IAM path removal"
```

---

### Task 6.10 — Codex ARCHITECTURE review

**Acceptance Criteria:**
- [ ] Codex `gpt-5.3-codex` reasoning=high mode=ARCHITECTURE with plan-review framing.
- [ ] Verdict → `reviews/phase-6-round-1.json`.
- [ ] All fixes applied inline; round 2 max.

**Steps:**
- [ ] Run Codex; apply fixes; save JSON; commit plan delta.

---

### Task 6.11 — Opus phase-end review

**Acceptance Criteria:**
- [ ] Opus reviews capability matrix ↔ service authz calls ↔ DB CHECK ↔ migration content for alignment.
- [ ] Report saved `reviews/phase-6-opus.md` (informal; no verdict JSON required).
- [ ] Issues flagged → addressed before Phase 7 starts.

**Steps:**
- [ ] Dispatch Opus review; apply notes; commit.

---

## Phase 7: HTTP Delivery (routes, contracts, error mapping)

**Intent:** Expose Phase 5 services + Phase 6 route/cancel services over HTTP. Handlers are thin: extract + validate request, call service, map result/error to status code + body. No tx, no repo, no domain logic. Contract types in `api/` shared with frontend via OpenAPI emit.

**Codex review:** QUALITY (Task 7.10).

**Opus review:** none.

**Codex Round 1 (QUALITY) verdict:** APPROVE_WITH_FIXES, `upgrade_required=true`, 8 findings (3 high, 4 medium, 1 low). Applied inline:

| # | Sev | Fix |
|---|---|---|
| Q1 | high | Per-route authz matrix at handler (defense-in-depth with Phase 6 DB tripwire) |
| Q2 | high | Idempotency-Key scope + payload binding + TTL + concurrency rules |
| Q3 | high | ETag/If-Match strict parser; 428 missing; 412 mismatch; strong-only |
| Q4 | medium | Error mapper uses errors.As/Is unwrap + decode→400 branch |
| Q5 | medium | Decoder hardening: DisallowUnknownFields, Content-Type, max body, duplicate keys |
| Q6 | medium | Pagination: max limit, deterministic tie-breaker, HMAC signed cursor |
| Q7 | medium | Per-route test matrix (9 cases) + middleware ordering tests |
| Q8 | low | OpenAPI: all headers + shared error component + contract-sync CI test |

### Per-route authz matrix (Q1)

| Route | Capability | Scope |
|---|---|---|
| POST /documents/{id}/submit | `doc.submit` | doc.area |
| POST /instances/{id}/stages/{sid}/signoffs | `doc.signoff` | doc.area + stage.members |
| POST /documents/{id}/publish | `doc.publish` | doc.area |
| POST /documents/{id}/schedule-publish | `doc.publish` | doc.area |
| POST /documents/{id}/supersede | `doc.supersede` | doc.area |
| POST /documents/{id}/obsolete | `doc.obsolete` | doc.area |
| POST /instances/{id}/cancel | `workflow.instance.cancel` | doc.area |
| GET /instances/{id} | `doc.view` | doc.area (else 404, no leak) |
| GET /inbox | none global; per-area filter on results | per-area `doc.view` |
| POST/PUT/DELETE /routes | `route.admin` | tenant |
| GET /routes | `route.admin` OR `doc.submit` any area | tenant |

Handler helper `authz.RequireHTTP(w,r,cap,areaCode)` short-circuits 403 before service call.

### Idempotency-Key contract (Q2)

- **Scope:** `(tenant_id, actor_user_id, route_template, key)` — namespaced per route.
- **Payload binding:** server computes `SHA-256(canonical(body))`, stored with key. Same key + different hash → **409** `idempotency.key_conflict`.
- **TTL:** 24h in `metaldocs.idempotency_keys` (new — **Migration 0146**).
- **Concurrency:** `INSERT ON CONFLICT DO NOTHING RETURNING`; loser polls result 500ms max; timeout → 409 `idempotency.in_flight`.
- **Required on mutating POST/PUT/DELETE.** Missing → 400 `idempotency.key_required`.
- **Replay response:** same status + body; header `Idempotent-Replay: true` added.

### ETag / If-Match contract (Q3)

- ETag strong-only: `"v<revision_version>"`. Quoted per RFC 7232.
- GET always emits ETag.
- Mutators (except submit which creates) require `If-Match`. Missing → **428** `precondition.if_match_required`. Wildcard `*` accepted.
- Parser strict: reject weak (`W/`), unknown directives, multi-ETag. Malformed → 400 `validation.if_match_malformed`.
- Mismatch → 412 `precondition.if_match_mismatch` (distinct from other 409 conflicts).
- Successful mutation returns new `ETag`.

### Decoder hardening (Q5)

- `json.Decoder.DisallowUnknownFields()` on all bodies.
- `Content-Type: application/json` required on POST/PUT; else 415.
- Max body 64 KB via `http.MaxBytesReader`; overflow → 413.
- Empty body where expected → 400 `validation.empty_body`.
- Malformed JSON → 400 `validation.json_decode` with sanitized line/col.
- Duplicate keys rejected via `strictjson.Decode` wrapper.

### Pagination (Q6)

- Default limit 25, max 100; over → 400.
- Sort: primary `created_at DESC`, tie-breaker `id ASC`.
- Cursor: base64-url of HMAC-signed `{last_sort_key, last_id, tenant_id, exp}`. Tampered → 400 `validation.cursor_invalid`. Expiry 24h.
- Response shape: `{items, next_cursor, has_more}`.
- Test: concurrent inserts between pages must not duplicate items (value-based cursor).

### Test matrix per handler (Q7)

Each handler table-driven, minimum 9 cases: (1) happy, (2) validation fail, (3) authz fail, (4) idempotency replay, (5) idempotency conflict, (6) If-Match missing, (7) If-Match mismatch, (8) mapped domain error, (9) unknown/wrapped error. Plus middleware suite: ordering, request_id propagation, authn fail→401, tenant mismatch→403.

**Subpackages:**
- `internal/modules/documents_v2/approval/http/` — handlers + routing
- `internal/modules/documents_v2/approval/http/contracts/` — request/response DTOs
- `internal/modules/documents_v2/approval/http/errors.go` — error → status mapping
- `api/openapi/spec2.yaml` — emitted spec

**Invariants:**

1. **Handlers never open tx, never call repo.** Only `services.X.Method(ctx, input)`.
2. **Idempotency-Key header → service.** Middleware `IdempotencyKeyExtractor` parses header; handler forwards.
3. **If-Match header → `expectedRevisionVersion`.** ETag contract: GET emits `ETag: "v<revision_version>"`; mutator requires `If-Match`.
4. **Error → status code** via canonical map (see §Error mapping).
5. **Request validation** at handler boundary — strict JSON schema per contract type; reject malformed with 400 before service call.
6. **Authn middleware** populates `ctx` with `actor_user_id` + `tenant_id`. Handlers extract via helper, never from body.
7. **Rate limit + CSRF** applied via existing middleware chain; no per-handler config.

### Error mapping (canonical)

| Go error | HTTP status | Body `error.code` |
|---|---|---|
| `ErrStaleRevision` | 409 | `conflict.stale_revision` |
| `ErrIllegalTransition` | 422 | `state.illegal_transition` |
| `ErrDuplicateSubmission` | 200 (replay) | n/a — returns stored result |
| `ErrActorAlreadySigned` | 409 | `signoff.duplicate` |
| `ErrSoDViolation` | 403 | `signoff.sod_violation` |
| `ErrContentHashMismatch` | 412 | `signoff.content_hash_mismatch` |
| `ErrSignatureInvalid` | 401 | `signoff.signature_invalid` |
| `ErrNotStageMember` | 403 | `signoff.not_stage_member` |
| `ErrReasonRequired` | 400 | `validation.reason_required` |
| `ErrCapabilityDenied` | 403 | `authz.capability_denied` |
| `ErrCapabilityNotAsserted` | 500 | `internal.capability_tripwire` — bug; alerts on-call |
| `ErrInstanceCompleted` | 409 | `state.instance_completed` |
| `ErrNoActiveInstance` | 404 | `not_found.instance` |
| `ErrRouteInUse` | 409 | `route.in_use` |
| `ErrCascadeTooLarge` | 413 | `obsolete.cascade_too_large` |
| `ErrEffectiveFromNotFuture` | 400 | `validation.effective_from_past` |
| `ErrEventPayloadInvalid` | 500 | `internal.event_payload_invalid` |
| `ErrFKViolation` | 422 | `db.fk_violation` |
| `ErrCheckViolation` | 422 | `db.check_violation` — includes reason from trigger RAISE |
| `ErrInsufficientPrivilege` | 500 | `internal.db_privilege_missing` — bug |
| `ErrUnknownDB` | 500 | `internal.db_unknown` |
| `context.DeadlineExceeded` | 504 | `timeout` |

Body schema: `{ "error": { "code": "string", "message": "string", "details": {...} }, "request_id": "uuid" }`.

### Route map

| Method | Path | Handler | Service call |
|---|---|---|---|
| POST | `/api/v2/documents/{id}/submit` | `SubmitHandler` | `SubmitService.SubmitRevisionForReview` |
| POST | `/api/v2/approval/instances/{instance_id}/stages/{stage_id}/signoffs` | `SignoffHandler` | `DecisionService.RecordSignoff` |
| POST | `/api/v2/documents/{id}/publish` | `PublishHandler` | `PublishService.PublishApproved` |
| POST | `/api/v2/documents/{id}/schedule-publish` | `SchedulePublishHandler` | `PublishService.SchedulePublish` |
| POST | `/api/v2/documents/{id}/supersede` | `SupersedeHandler` | `SupersedeService.PublishSuperseding` |
| POST | `/api/v2/documents/{id}/obsolete` | `ObsoleteHandler` | `ObsoleteService.MarkObsolete` |
| POST | `/api/v2/approval/instances/{instance_id}/cancel` | `CancelHandler` | `CancelInstanceService.Cancel` |
| GET | `/api/v2/approval/instances/{instance_id}` | `GetInstanceHandler` | `ApprovalReadService.LoadInstance` |
| GET | `/api/v2/approval/inbox` | `InboxHandler` | `ApprovalReadService.ListPendingForActor` |
| POST | `/api/v2/approval/routes` | `CreateRouteHandler` | `RouteAdminService.Create` |
| PUT | `/api/v2/approval/routes/{id}` | `UpdateRouteHandler` | `RouteAdminService.Update` |
| DELETE | `/api/v2/approval/routes/{id}` | `DeactivateRouteHandler` | `RouteAdminService.Deactivate` |
| GET | `/api/v2/approval/routes` | `ListRoutesHandler` | `RouteAdminService.List` |

---

### Task 7.0 — Migration 0146: idempotency_keys store (Q2)

**Files:** `supabase/migrations/0146_idempotency_keys.sql`.

**Acceptance Criteria:**
- [ ] `metaldocs.idempotency_keys(tenant_id uuid, actor_user_id text, route_template text, key text, payload_hash text, response_status int, response_body jsonb, status text CHECK (status IN ('in_flight','completed','failed')), created_at timestamptz, expires_at timestamptz, PRIMARY KEY (tenant_id, actor_user_id, route_template, key))`.
- [ ] Index on `expires_at` for janitor sweep.
- [ ] Janitor job (Phase 8) deletes rows where `expires_at < now()`.
- [ ] GRANT SELECT,INSERT,UPDATE on this table to `metaldocs_writer`.

**Steps:** Codex medium. Commit `feat(spec2/phase7): 0146 idempotency_keys`.

---

### Task 7.1 — Contracts package (DTOs + validation)

**Files:** `internal/modules/documents_v2/approval/http/contracts/*.go`.

**Acceptance Criteria:**
- [ ] One file per domain area: `submit.go`, `signoff.go`, `publish.go`, `supersede.go`, `obsolete.go`, `cancel.go`, `route.go`, `instance_read.go`.
- [ ] Each request DTO has `Validate() error` method with per-field rules (required, enum, length, regex).
- [ ] Response DTOs mirror service output types — no extra fields, no re-exposed internals.
- [ ] `ErrorResponse` struct matches body schema above.
- [ ] **Strict decoder helper (Q5):** `strictjson.Decode(r *http.Request, dst any) error` — wraps body with `MaxBytesReader(64KB)`, checks Content-Type=application/json, rejects duplicate keys, calls `DisallowUnknownFields`, returns typed decode errors for mapper.
- [ ] Unit tests for every `Validate()` + `strictjson.Decode` edge cases (oversize body, bad content-type, duplicate keys, unknown field).

**Steps:** Sonnet (DTO boilerplate). Commit `feat(spec2/phase7): http contracts`.

---

### Task 7.2 — Error mapper

**Files:** `internal/modules/documents_v2/approval/http/errors.go`, `_test.go`.

**Acceptance Criteria:**
- [ ] `MapErrorToResponse(err error) (status int, body ErrorResponse)` — implements matrix above.
- [ ] **Unwrap chain (Q4):** `errors.As`/`errors.Is` walk full wrap chain; matches first known type. Wrapped domain errors inside service/repo layers still map correctly.
- [ ] **Decode errors (Q4):** explicit branches for `*json.SyntaxError`, `*json.UnmarshalTypeError`, `io.EOF`, `strictjson.ErrDuplicateKey`, `*http.MaxBytesError` → 400/413 with stable codes.
- [ ] Unknown/unmatched → 500 + `internal.unknown`; sanitized details (no stack, no SQL text, no raw body echo).
- [ ] Table-driven test covering every row in matrix + every decode-error branch.

**Steps:** Codex medium. Commit `feat(spec2/phase7): error mapper`.

---

### Task 7.3 — SubmitHandler

**Files:** `internal/modules/documents_v2/approval/http/submit_handler.go`, `_test.go`.

**Acceptance Criteria:**
- [ ] Extracts `document_id` from path, actor/tenant from ctx, `If-Match` → `expectedRevisionVersion`, `Idempotency-Key` header → service input.
- [ ] Binds JSON body to `SubmitRequest` + `Validate()`.
- [ ] Calls `services.Submit.SubmitRevisionForReview`.
- [ ] Success → 201 + `SubmitResponse{instance_id, was_replay}` + `ETag: "v<new_version>"`.
- [ ] Error via `MapErrorToResponse`.
- [ ] Handler test uses `httptest` + service mock; covers happy, validation fail, replay, stale OCC, cap denied.

**Steps:** Sonnet. Commit.

---

### Task 7.4 — SignoffHandler

Same pattern as 7.3 for `/signoffs`. Covers approve + reject via `decision` field in body. Tests: happy approve, happy reject, SoD violation → 403, signature invalid → 401, content hash mismatch → 412, replay → 200 with `was_replay=true`, stale → 409.

**Steps:** Sonnet. Commit.

---

### Task 7.5 — Publish / Schedule / Supersede / Obsolete / Cancel handlers

One file each, parallel structure to 7.3. Each maps to its service and respects capability/error matrix. Tests ≥ 4 per handler (happy, authz denied, illegal transition, stale OCC).

**Steps:** Sonnet. Commit `feat(spec2/phase7): mutator handlers`.

---

### Task 7.6 — Read handlers (GetInstance, Inbox)

**Files:** `get_instance_handler.go`, `inbox_handler.go`, tests.

**Acceptance Criteria:**
- [ ] `ApprovalReadService` (added in Task 7.6 — read-only, no tx, uses repo read methods) with `LoadInstance(ctx, tenantID, actorID, instanceID)` and `ListPendingForActor(ctx, tenantID, actorID, filters, page)`.
- [ ] `GET /instances/{id}` returns full instance + stages + signoffs + route snapshot; emits ETag.
- [ ] `GET /inbox` returns paginated list of stages where actor ∈ members AND stage.status=active; filters: `area_code`, `overdue_only`, `submitter`.
- [ ] **Cursor pagination (Q6):** HMAC-signed opaque cursor; `limit` default 25, max 100; deterministic `created_at DESC, id ASC`; tamper → 400.
- [ ] RBAC: caller must have `doc.view` on area; else filtered out (no 403 leak, just omission).
- [ ] Tests: empty inbox, filtered, cursor roundtrip, tampered cursor → 400, expired cursor → 400, cross-tenant isolation, unauthorized area omitted, concurrent-insert between pages (no dup).

**Steps:** Sonnet + Codex medium for read service. Commit.

---

### Task 7.7 — RouteAdmin handlers

Four handlers (Create/Update/Deactivate/List). `route.admin` cap required (service-enforced). Tests.

**Steps:** Sonnet. Commit.

---

### Task 7.8 — Routing registration + middleware chain

**Files:** `internal/modules/documents_v2/approval/http/router.go`, modifications to `internal/app/router.go`.

**Acceptance Criteria:**
- [ ] `RegisterRoutes(r chi.Router, services *app.Services)` wires all 13 routes from table above.
- [ ] Middleware order: `RequestID → Recover → Authn → Tenant → IdempotencyKey → IfMatch → Handler`.
- [ ] Logs every request with: `request_id, actor, tenant, route, status, duration, error_code`.
- [ ] `/openapi.yaml` endpoint serves emitted spec (Task 7.9).

**Steps:** Haiku (glue). Commit.

---

### Task 7.9 — OpenAPI emit

**Files:** `api/openapi/spec2.yaml`, generator script `scripts/emit-openapi.sh`.

**Acceptance Criteria:**
- [ ] Generated from contract Go structs via `go-swagger` or hand-written (whichever is already in project; check).
- [ ] Covers all 13 routes + all error codes.
- [ ] **Headers documented per route (Q8):** `If-Match` (required on mutators), `Idempotency-Key` (required on mutators), `ETag` (response), `Idempotent-Replay` (response), `X-Request-Id` (request/response).
- [ ] **Shared components:** `ErrorResponse`, `PaginatedList`, `Cursor` defined once; referenced by all routes.
- [ ] **Contract-sync CI test:** reflects over handler registration; fails if a registered route is missing from spec or vice versa. Also fails if an emitted error code is not in the shared enum.
- [ ] CI step validates YAML conforms to OpenAPI 3.1 via `spectral lint`.
- [ ] Frontend TS types emitted from spec via `openapi-typescript` in separate CI job.

**Steps:** Codex medium. Commit.

---

### Task 7.10 — Codex QUALITY review

**Acceptance Criteria:** Codex `gpt-5.3-codex` high, QUALITY mode, plan-review framing. Verdict JSON saved. Fixes inline. Round 2 max.

**Steps:** Run Codex; apply; commit.

---

## Phase 8: Scheduler Jobs (effective_date_publisher, stuck_instance_watchdog, janitors)

**Intent:** Long-running background workers. All must be: idempotent, crash-safe (resume on restart), observable (metrics + logs), rate-limited, lease-based (multi-instance safe). Jobs compose Phase 5 services — no direct repo/DB.

**Codex review:** OPERATIONS (Task 8.8).

**Opus review:** phase end — cross-check cron cadence, lease semantics, scheduler invariants vs Phase 5 Task 5.7.

**Codex Round 1 (OPERATIONS) verdict:** REJECT-style ("REVISE"), `upgrade_required=true`, 10 findings (2 critical, 4 high, 4 medium). All applied inline:

| # | Sev | Fix |
|---|---|---|
| O1 | critical | Fencing token `lease_epoch` added; every mutating op asserts current epoch; split-brain blocked |
| O2 | critical | Drain choreography: stop-ticks first, heartbeat continues until in-flight drains or hard deadline; probe added |
| O3 | high | Back-pressure hysteresis: enter>70% / exit<60%; max 10 consecutive skips; per-job policy (watchdog degrades, never skips fully) |
| O4 | high | DB clock authoritative (`now()` server-side); app uses monotonic timers only for tick intervals; all timestamps `timestamptz UTC` |
| O5 | high | SECURITY DEFINER hardening (same as Phase 6 A3): owner, search_path, REVOKE FROM PUBLIC, GRANT scheduler role only |
| O6 | high | Deploy sequence: apply 0147 → verify fns → deploy with jobs OFF → per-job flag enable ramp; rollback = disable flags first |
| O7 | medium | Extra metrics: acquire_attempts/fail reason, lease_steal_count, heartbeat_latency/failures, in_flight_jobs, drain_duration, reaper_reaped_total |
| O8 | medium | Governance exporter: secrets via secret store, TLS + timeouts + circuit breaker, separate worker budget |
| O9 | medium | Timezones: all comparisons UTC timestamptz; DST boundary tests added |
| O10 | medium | Lease reaper: compare-and-delete in single tx with lock; audit log per reap |

### Fencing token design (O1)

Migration 0147 schema update: `job_leases(job_name pk, leader_id, lease_epoch bigint NOT NULL DEFAULT 0, acquired_at, heartbeat_at, expires_at)`.

- `acquire_lease(job, leader, ttl)` atomically: if no lease OR expired → increment `lease_epoch` and set leader; return `(acquired bool, epoch bigint)`.
- Workers pass `epoch` into every job call path as first arg. Before committing any mutating tx, worker re-reads lease row and asserts `current_epoch == my_epoch`; else abort + rollback.
- `SchedulerService.Run`, `CancelInstanceService.Cancel` (watchdog path) accept optional `fencingEpoch int64`; if provided, wrap write in `WHERE lease_epoch = $fencingEpoch` guard clause.

### Drain choreography (O2)

Sequence on SIGTERM:
1. **Ticker stop** — no new ticks fire. In-flight jobs continue.
2. **Heartbeat continues** — lease stays held while in-flight drain.
3. **Wait for in-flight with hard deadline 30s.**
4. If drain completes within deadline → release leases cleanly + exit.
5. If deadline hit → cancel worker contexts, wait bounded 5s join, release leases with warning log `drain_hard_cancel`.

Probe 8.7-F: SIGTERM mid-batch → no second worker acquires lease before first releases; metric `drain_duration` captured.

### Back-pressure hysteresis (O3)

- State machine: `normal ↔ throttled`. Transition normal→throttled when CPU>70% for 3 consecutive probes; throttled→normal when CPU<60% for 3 consecutive.
- Per-job policy:
  - `effective_date_publisher`, `governance_events_exporter`, `idempotency_janitor` — **skip tick** in throttled state.
  - `stuck_instance_watchdog`, `lease_reaper` — **degrade** (2× interval) but never skip fully.
- Max 10 consecutive skips → emit `job_backpressure_stuck` alert.
- Metrics: `job_skip_total{reason="backpressure"}`, `job_skip_streak`.

### Clock authority (O4, O9)

- **All TTL/threshold comparisons use `now()` at DB** (SQL literal in WHERE or via `acquire_lease` fn). No app-side `time.Now()` in comparisons.
- **App monotonic timers** (`time.Ticker`) only for local interval scheduling — cannot be compared cross-node.
- All columns `timestamptz`; all logic UTC; tenant/local tz only at presentation (Phase 9).
- DST test: 7d watchdog threshold stable across spring-forward / fall-back.

### Deploy choreography (O6)

1. **Apply 0147 migration** — schema + fns in place.
2. **Verify gates:** `SELECT has_function_privilege('metaldocs_writer','metaldocs.acquire_lease(text,text,interval)','EXECUTE')`; all asserts pass in CI step.
3. **Deploy binary with `ENABLE_JOB_*=false`** — no jobs auto-start.
4. **Ramp enablement:** operator flips env flags per job in order: `lease_reaper` → `idempotency_janitor` → `effective_date_publisher` → `stuck_instance_watchdog` → `governance_events_exporter`. 10-min soak between each.
5. **Rollback:** flip all `ENABLE_JOB_*=false` FIRST, then roll binary back. Never roll migration back (additive).

**Subpackages:**
- `internal/modules/jobs/scheduler/` — cron runtime
- `internal/modules/jobs/effective_date_publisher/` — job 1
- `internal/modules/jobs/stuck_instance_watchdog/` — job 2
- `internal/modules/jobs/idempotency_janitor/` — job 3
- `internal/modules/jobs/governance_events_exporter/` — job 4 (outbox drain, optional)

**Invariants:**

1. **Lease-based single-leader:** each job uses `SELECT FOR UPDATE SKIP LOCKED` on `metaldocs.job_leases(job_name)` to claim a lease; TTL 5min; heartbeat 1min.
2. **Crash-safe:** on restart, expired leases auto-reclaimable. In-flight work uses Phase 5 idempotency so double-run is safe.
3. **Cadence configurable per env** via `JOB_<NAME>_INTERVAL` env var; defaults in code.
4. **Metrics:** every job exposes Prometheus counters `job_<name>_runs_total`, `job_<name>_errors_total`, `job_<name>_processed_total`, histogram `job_<name>_duration_seconds`.
5. **Structured logs:** each run emits start + end + per-batch log lines with `job_name, run_id, processed, skipped, errors`.
6. **Graceful shutdown:** SIGTERM triggers context cancel; workers drain current batch then exit.
7. **Back-pressure:** if DB CPU >70% (via `pg_stat_activity` probe), jobs skip tick with log.

### Cron cadence table

| Job | Default interval | Batch size | Service called |
|---|---|---|---|
| `effective_date_publisher` | 60s | 100 | `SchedulerService.Run` |
| `stuck_instance_watchdog` | 5min | 50 | `CancelInstanceService.Cancel` (for over-deadline) or alert only |
| `idempotency_janitor` | 1h | 5000 | direct repo `DeleteExpiredIdempotencyKeys` |
| `governance_events_exporter` | 30s | 500 | external exporter (optional; skip if disabled) |
| `lease_reaper` | 10min | — | reclaims expired leases |

---

### Task 8.0 — Migration 0147: job_leases + idempotency_keys janitor index

**Files:** `supabase/migrations/0147_job_leases.sql`.

**Acceptance Criteria:**
- [ ] `metaldocs.job_leases(job_name text primary key, leader_id text, lease_epoch bigint NOT NULL DEFAULT 0, acquired_at timestamptz, heartbeat_at timestamptz, expires_at timestamptz)` (O1).
- [ ] `metaldocs.acquire_lease(_job text, _leader text, _ttl interval) RETURNS TABLE(acquired bool, epoch bigint)` — SECURITY DEFINER; increments `lease_epoch` on fresh acquire or expired takeover.
- [ ] `metaldocs.heartbeat_lease(_job text, _leader text, _epoch bigint) RETURNS bool` — only extends if `epoch` matches current (prevents stale heartbeat).
- [ ] `metaldocs.release_lease(_job text, _leader text, _epoch bigint) RETURNS void` — same guard.
- [ ] `metaldocs.assert_lease_epoch(_job text, _epoch bigint) RETURNS void` — RAISE EXCEPTION 'ErrLeaseEpochStale' if mismatch; called by mutating jobs (O1).
- [ ] **SECURITY DEFINER hardening (O5):** owner `metaldocs_admin`, `SET search_path = metaldocs, pg_temp`, `REVOKE EXECUTE FROM PUBLIC`, `GRANT EXECUTE TO metaldocs_writer` only. Strict arg regex validation inside.
- [ ] Index on `idempotency_keys(expires_at) WHERE status='completed'` for janitor sweep.
- [ ] All timestamps `timestamptz` (O4, O9).

**Steps:** Codex medium. Commit.

---

### Task 8.1 — Scheduler runtime (cron loop, lease, metrics)

**Files:** `internal/modules/jobs/scheduler/scheduler.go`, `_test.go`.

**Acceptance Criteria:**
- [ ] `Scheduler` struct with `RegisterJob(name string, interval time.Duration, fn JobFunc, policy BackpressurePolicy)`. `BackpressurePolicy` ∈ `{SkipOnPressure, DegradeOnPressure}`.
- [ ] `Start(ctx)` runs tickers per job; each tick: check back-pressure state; acquire lease with epoch; run fn; release.
- [ ] Heartbeat goroutine refreshes lease every 1min WITH EPOCH; on mismatch → abort in-flight + log `lease_stolen`.
- [ ] Lease acquire failure → log + metric `acquire_attempts_total{result="held_by_other"}`.
- [ ] **Drain (O2):** on SIGTERM: stop ticker (no new ticks), keep heartbeat alive until in-flight drains or hard-deadline 30s; on deadline cancel ctx, wait 5s join, release leases with warning.
- [ ] **Back-pressure (O3):** probe `pg_stat_database` CPU; hysteresis enter>70%/exit<60% over 3 samples; per-job policy applied (SkipOnPressure skips; DegradeOnPressure doubles interval).
- [ ] **Clock (O4):** local `time.Ticker` for intervals (monotonic); all lease TTL comparisons via DB `now()`.
- [ ] **Metrics (O7):** `acquire_attempts_total{result}`, `lease_steal_count`, `heartbeat_latency_seconds`, `heartbeat_failures_total`, `in_flight_jobs`, `drain_duration_seconds`, `drain_outcome_total{result}`, `job_skip_total{reason}`, `job_skip_streak`.
- [ ] Unit tests: lease contention, heartbeat extends, epoch stolen mid-run → job aborts, drain hard-deadline, back-pressure enter/exit hysteresis, jitter ±10%.

**Steps:** Codex high. Commit.

---

### Task 8.2 — EffectiveDatePublisher job

**Files:** `internal/modules/jobs/effective_date_publisher/job.go`, `_test.go`.

**Acceptance Criteria:**
- [ ] JobFunc calls `services.Scheduler.Run(ctx, fencingEpoch, batchLimit=100)` — method owns its own tx composition per Phase 5 Task 5.7. Epoch threaded; each per-row transition tx calls `metaldocs.assert_lease_epoch` (O1).
- [ ] Returns `(processed, skipped, err)` → emits metrics.
- [ ] Error from service → increment `job_errors_total` + log; does NOT fail job (next tick retries).
- [ ] If `processed == batchLimit` → log "backlog likely"; next tick may run sooner (burst mode: halve interval until empty).
- [ ] Tests: mocked service returning various counts; burst-mode activation; error handling.

**Steps:** Codex medium. Commit.

---

### Task 8.3 — StuckInstanceWatchdog job

**Files:** `internal/modules/jobs/stuck_instance_watchdog/job.go`, tests.

**Acceptance Criteria:**
- [ ] Queries `approval_instances WHERE status='under_review' AND submitted_at < now() - interval '7 days'` — **`now()` at DB** (O4); `submitted_at timestamptz UTC`.
- [ ] For each stuck instance, per policy:
  - If `route_config.drift_policy='auto_cancel'` → call `CancelInstanceService.Cancel(instance, reason='stuck_watchdog_auto_cancel', actor='system:watchdog')`.
  - Else → emit `approval.instance.stuck_alert` governance event + (future) notify via webhook.
- [ ] Bypass authz via `system:watchdog` actor + `SET LOCAL metaldocs.bypass_authz='watchdog'` (separate bypass channel from scheduler).
- [ ] Metrics: `watchdog_stuck_detected_total`, `watchdog_auto_cancelled_total`.
- [ ] Tests: no stuck → 0; 3 stuck auto_cancel → 3 cancels + events; drift_policy=fail_stage → alert event only.

**Steps:** Codex high. Commit.

---

### Task 8.4 — IdempotencyJanitor job

**Files:** `internal/modules/jobs/idempotency_janitor/job.go`, tests.

**Acceptance Criteria:**
- [ ] `DELETE FROM metaldocs.idempotency_keys WHERE expires_at < now() AND status='completed' LIMIT 5000` in batches until 0 rows or max iterations (10) per tick.
- [ ] Metrics: `janitor_deleted_total`.
- [ ] Tests: 0 expired → no-op; 100 expired → deleted.

**Steps:** Haiku/Sonnet. Commit.

---

### Task 8.5 — LeaseReaper job

**Files:** `internal/modules/jobs/scheduler/lease_reaper.go`, tests.

**Acceptance Criteria:**
- [ ] Every 10min: `DELETE FROM metaldocs.job_leases WHERE expires_at < now() - interval '10 min' RETURNING job_name, leader_id, lease_epoch` — compare-and-delete in single tx with `FOR UPDATE` lock on row to prevent race with concurrent heartbeat (O10). Each deletion writes audit `governance_events` row `lease.reaped` with `{job_name, stale_leader_id, stale_epoch, reaped_at}`.
- [ ] Metrics: `lease_reaper_reclaimed_total`.
- [ ] Tests: expired lease reclaimed; fresh lease untouched.

**Steps:** Haiku. Commit.

---

### Task 8.6 — App wiring (main.go)

**Files:** `cmd/server/main.go` modifications.

**Acceptance Criteria:**
- [ ] Scheduler started in goroutine after HTTP server.
- [ ] Jobs registered: 4 jobs + lease reaper.
- [ ] Each job gated by env flag `ENABLE_JOB_<NAME>` — **default FALSE** per O6 ramp policy. Operator enables explicitly post-deploy.
- [ ] Exporter (O8) secrets via `SECRETS_PROVIDER` interface (env-backed in dev, AWS Secrets Manager/Vault in prod); redacted in startup log; TLS+timeouts+circuit breaker defaults; dedicated worker budget (goroutine pool size 2) so cannot starve core jobs.
- [ ] SIGTERM: cancel scheduler ctx first, then HTTP server (drain HTTP → drain jobs → exit).
- [ ] Build passes; no import cycles.

**Steps:** Haiku. Commit.

---

### Task 8.7 — Integration tests (multi-instance lease + crash recovery)

**Files:** `internal/modules/jobs/integration_test.go` (tag `integration`).

**Acceptance Criteria:**
- [ ] **Probe A:** two scheduler instances contend on same lease; only one runs job per tick; metrics confirm.
- [ ] **Probe B:** instance holding lease killed mid-run (simulated via context cancel); after TTL+reaper, other instance acquires.
- [ ] **Probe C:** `effective_date_publisher` idempotent across crash — row already published not double-published (uses Phase 5 dedupe).
- [ ] **Probe D:** back-pressure — simulated DB load 90% → tick skipped with log.
- [ ] **Probe E:** stuck_watchdog with drift=auto_cancel → instance cancelled, doc back to draft.
- [ ] **Probe F (O2):** SIGTERM mid-batch → drain completes within 30s; no second worker acquires lease before first releases; `drain_duration` metric captured.
- [ ] **Probe G (O1):** worker A holds epoch=5; simulate GC pause > TTL; worker B acquires epoch=6; A resumes and tries commit → `ErrLeaseEpochStale` raised; no write lands.
- [ ] **Probe H (O3):** simulated CPU=75% for 3 probes → effective_date_publisher skips; watchdog interval doubles; after CPU<60% for 3 probes → normal resumes.
- [ ] **Probe I (O9):** DST spring-forward — 7d watchdog threshold unchanged; no double-fire, no skip.
- [ ] **Probe J (O10):** reaper races with heartbeat — FOR UPDATE serializes; active lease never reaped.

**Steps:** Codex high. Commit.

---

### Task 8.8 — Codex OPERATIONS review

Codex `gpt-5.3-codex` high, OPERATIONS mode, plan-review framing. Verdict → `reviews/phase-8-round-1.json`. Fixes inline. Round 2 max.

---

### Task 8.9 — Opus phase-end review

Opus reviews cron cadences + lease TTL + back-pressure thresholds + interaction with Phase 5 Task 5.7 scheduler composition. Notes in `reviews/phase-8-opus.md`.

---

## Phase 9: Frontend (RouteAdmin, Inbox, SignoffDialog, Timeline, Lock, Badge, RegistryDetail)

**Intent:** React + TypeScript UI consuming Phase 7 API. Optimistic locking surfaced (ETag → If-Match), idempotency keys client-generated, 8-state status badge, edit-lock visibility, timeline with signoff history. Use BlockNote features where applicable (per memory: don't reinvent).

**Codex review:** COVERAGE (Task 9.11).

**Opus review:** none (covered by Phase 10 tests).

**Codex Round 1 (COVERAGE) verdict:** APPROVE_WITH_FIXES, `upgrade_required=true`, 11 findings (1 critical, 5 high, 5 medium). All applied inline:

| # | Sev | Fix |
|---|---|---|
| F1 | critical | Single `mutationClient.ts` interceptor — all mutations auto-inject UUIDv7 Idempotency-Key + If-Match + centralized 412/401/403/offline handling |
| F2 | high | Transition-policy table (state→allowed actions→disabled reason i18n) as single source consumed by all surfaces |
| F3 | high | Stale-data policy: focus-refetch + 30s stale-time + inline stale banner (SSE hook designed, impl Phase 11) |
| F4 | high | Loading/empty/error/partial states required on every page+panel |
| F5 | high | Offline UX: banner, disable destructive CTAs, mutation retry queue for idempotent routes |
| F6 | high | Permission degradation: 401 re-auth modal; 403 hide CTA + explain; capabilities poll for drift detection |
| F7 | medium | SignoffDialog state machine with error classes + focus return + preserve intent |
| F8 | medium | Integrity display: content_hash + version + ETag in RegistryDetail, copy button, drift warning |
| F9 | medium | WCAG axe CI + live-region + focus return + virtualized timeline aria + contrast AA + error narration |
| F10 | medium | Datetime formatter policy: UTC payloads, browser tz display, locale-aware, relative/absolute toggle |
| F11 | medium | Task 9.11 risk-based test matrix; merge-block until green |

### Transition policy table (F2)

| State | Allowed actions | Disabled-reason code |
|---|---|---|
| `draft` | edit, submit, delete | — |
| `under_review` | view, cancel_instance (qms_admin/route_owner), signoff (stage member) | `ui.disabled.locked_under_review` |
| `approved` | publish, schedule_publish, supersede (if chain has published), view | — |
| `scheduled` | cancel_schedule (→approved), view | `ui.disabled.scheduled_waiting` |
| `published` | supersede, obsolete, view | — |
| `superseded` | view, obsolete | `ui.disabled.superseded_readonly` |
| `rejected` | edit (→draft via new revision), view | `ui.disabled.rejected_rework` |
| `obsolete` | view only | `ui.disabled.obsolete_terminal` |

Buttons across all surfaces consult this table; disabled state carries `aria-describedby` pointing to i18n reason.

### Mutation client contract (F1)

`apps/web/src/features/approval/api/mutationClient.ts` exports `mutate<TReq,TRes>(route, body, opts)`:

- Auto-generates `Idempotency-Key = uuidv7()` unless `opts.idempotencyKey` set.
- Reads `etagCache.get(resourceId)` → `If-Match` header.
- **412** → `onStaleConflict(resourceId)` global handler (toast + React Query invalidate).
- **401** → triggers re-auth modal; retry queued until login.
- **403** → toast `authz.capability_denied` + refetch capabilities.
- **Network error** → IndexedDB retry queue for allowlisted idempotent routes.
- **5xx** → exponential backoff up to 3 retries.
- Contract test per mutating call asserts headers, 412 path, 401 path.

### Datetime formatter policy (F10)

`apps/web/src/lib/datetime.ts`:
- `formatAbsolute(isoUTC, locale)` → browser-tz locale-aware (`Intl.DateTimeFormat`).
- `formatRelative(isoUTC, now)` → "2h ago" / "in 3 days" via `Intl.RelativeTimeFormat`.
- User preference toggle `absolute | relative` (default relative) stored in localStorage.
- Tests: pt-BR/en; DST boundary; future/past; edge cases (just now, years ago).

### Mandatory per-surface states (F4)

Every page/panel acceptance extended: loading skeleton, empty state with CTA, API error with retry button, partial-data fallback (e.g., timeline without signoffs still renders instance header).

**Subpackages:**
- `apps/web/src/features/approval/` — new
  - `pages/RouteAdminPage.tsx`
  - `pages/InboxPage.tsx`
  - `components/SignoffDialog.tsx`
  - `components/ApprovalTimelinePanel.tsx`
  - `components/LockBadge.tsx`
  - `components/StateBadge.tsx` — 8-state
  - `components/SupersedePublishDialog.tsx`
  - `hooks/useApproval.ts` — React Query
  - `api/approvalApi.ts` — typed client from OpenAPI
- `apps/web/src/features/documents_v2/components/RegistryDetailPanel.tsx` — modify (add approval section)

**Invariants:**

1. **All mutations use client-generated UUIDv7 Idempotency-Key.** Stored in `sessionStorage` keyed by action; retried safely.
2. **All mutations send If-Match from fetched ETag.** 412 mismatch → toast "doc changed, please refresh" + re-fetch.
3. **State badge is single source** — 8 states mapped to labels + colors + icons in one file. All list/detail views use it.
4. **Optimistic UI only where idempotent.** Signoff dialog — non-optimistic (server-authoritative; too costly to rollback).
5. **Lock state visible on document detail.** `locked_by_instance_id` present → banner "Under review by {actor}, locked".
6. **Timezone:** all timestamps displayed in browser local, stored as ISO 8601 UTC (matches API).
7. **Accessibility:** dialogs have focus trap + ESC close; state badges include `aria-label`; timeline nav is keyboard.

### Phase 9 route map (frontend)

| Path | Page | Capability gate (client-side UX only — server is source of truth) |
|---|---|---|
| `/approval/routes` | RouteAdminPage | `route.admin` |
| `/approval/inbox` | InboxPage | any user (filter server-side) |
| `/documents/:id` (extended) | RegistryDetail (with approval panel) | `doc.view` |

---

### Task 9.1 — API client from OpenAPI

**Files:** `apps/web/src/features/approval/api/approvalApi.ts` (generated), `scripts/gen-api.sh`.

**Acceptance Criteria:**
- [ ] `openapi-typescript` generates types from `api/openapi/spec2.yaml` (Phase 7 output).
- [ ] Typed client wrapper `createApprovalClient(fetcher)` with methods mirroring 13 routes.
- [ ] Each mutating method routes through **shared `mutationClient.mutate()` interceptor (F1)** — no direct `fetch()` from components.
- [ ] `mutationClient` responsibilities: UUIDv7 Idempotency-Key auto-inject, ETag cache→If-Match, 412/401/403/offline/5xx centralized handlers, IndexedDB retry queue for idempotent routes.
- [ ] Contract test per mutating call asserts headers present + 412/401 paths.
- [ ] Build fails if types drift from spec.

**Steps:** Sonnet. Commit.

---

### Task 9.2 — StateBadge component (8 states)

**Files:** `apps/web/src/features/approval/components/StateBadge.tsx`, `StateBadge.stories.tsx`, test.

**Acceptance Criteria:**
- [ ] Exports `StateBadge({state}: {state: ApprovalState})`.
- [ ] States: `draft, under_review, approved, scheduled, published, superseded, rejected, obsolete` (+ `cancelled` for instance-level; doc-level rolls back to draft on cancel).
- [ ] Color/icon/label table is the single source consumed by all other views.
- [ ] Storybook with all 9 variants.
- [ ] Unit test: all variants render with accessible label.

**Steps:** Sonnet. Commit.

---

### Task 9.3 — LockBadge + lock banner

**Files:** `LockBadge.tsx`, test.

**Acceptance Criteria:**
- [ ] Props `{lockedByInstanceId?, lockedByActor?, lockAcquiredAt?}`.
- [ ] Banner shown when `lockedByInstanceId` present on doc detail.
- [ ] Shows actor + relative time ("locked 2h ago by Alice").
- [ ] Click → scrolls to approval panel.
- [ ] Test: locked shows banner; unlocked hides.

**Steps:** Sonnet. Commit.

---

### Task 9.4 — SignoffDialog

**Files:** `SignoffDialog.tsx`, test.

**Acceptance Criteria:**
- [ ] Modal form: decision radio (Approve/Reject), reason textarea (required if reject), password field (for signature re-auth).
- [ ] Explicit state machine (F7): `idle → submitting → {success, error_bad_password, error_session_expired, error_rate_limited, error_network, error_server}`; each error preserves form values (except password cleared) so user retry keeps intent.
- [ ] On submit: `mutationClient.mutate(signoff, ...)` — Idempotency-Key + If-Match handled by interceptor.
- [ ] Server errors surfaced per code with i18n: 403 SoD → "Você submeteu este documento e não pode assinar" / "You submitted this doc and cannot sign"; 401 → inline "Sessão expirada, autentique novamente"; 429 → "Muitas tentativas, aguarde 30s".
- [ ] 412 mismatch → centralized handler (F1); dialog stays open showing stale banner + refresh CTA.
- [ ] Password never logged, never persisted in React state beyond submit; `type=password`, `autoComplete="current-password"`, cleared in finally.
- [ ] A11y (F9): focus trap, ESC close, Tab order, focus returns to trigger button on close, live-region announces submit result, validation errors narrated via `aria-describedby`, axe CI clean.
- [ ] Loading/error states (F4) rendered.
- [ ] Tests: happy approve, happy reject with reason, validation, 403, 412 banner, 401 re-auth path, 429 backoff message, password cleared, focus returns, axe clean.

**Steps:** Sonnet + manual a11y pass. Commit.

---

### Task 9.5 — ApprovalTimelinePanel

**Files:** `ApprovalTimelinePanel.tsx`, test.

**Acceptance Criteria:**
- [ ] Vertical timeline of events for a given `instance_id`.
- [ ] Sections: Submitted → per-stage (activated/passed/failed + signoffs nested) → Final (approved/rejected/cancelled).
- [ ] Each node shows actor + timestamp (local tz) + decision + optional reason.
- [ ] Signoff entries show signature_method (password_reauth/icp_brasil).
- [ ] Empty states + loading skeletons.
- [ ] Long timelines virtualized (react-window) if >50 nodes.
- [ ] Tests: basic render, empty, with rejections, timezone format.

**Steps:** Sonnet. Commit.

---

### Task 9.6 — InboxPage

**Files:** `pages/InboxPage.tsx`, hook `useInbox.ts`, test.

**Acceptance Criteria:**
- [ ] Cursor paginated list of pending stages where user is member.
- [ ] Columns: doc title, area, submitted by, submitted_at (relative), stage label, quorum progress, deadline (if set).
- [ ] Filters: area (multi-select), overdue only toggle, submitter.
- [ ] Row click → opens doc detail `/documents/:id` with approval panel focused + SignoffDialog auto-open if user can sign.
- [ ] "Refresh" button explicit; also auto-refetch on window focus (React Query default).
- [ ] Empty state: "Nothing to review".
- [ ] Tests: empty, 10 rows, filter by area, overdue filter, pagination roundtrip, focus-refetch.

**Steps:** Sonnet. Commit.

---

### Task 9.7 — RouteAdminPage

**Files:** `pages/RouteAdminPage.tsx`, `components/RouteEditor.tsx`, tests.

**Acceptance Criteria:**
- [ ] List route_configs (tenant-scoped). Columns: name, stage count, active, created_at, in_use (boolean).
- [ ] Create / Edit / Deactivate actions. Edit disabled when `in_use=true` with tooltip "route referenced by active instance; create new version".
- [ ] Editor: per-stage member picker (searches iam_users by display_name, filtered to area members), quorum selector (any_1 / all_of / m_of_n with `m` input), drift_policy dropdown.
- [ ] Validation: ≥1 stage; stage has ≥1 member; m ≤ members; distinct stage names.
- [ ] Tests: list render, create happy, edit blocked on in_use, m_of_n validation, deactivate confirmation.

**Steps:** Sonnet. Commit.

---

### Task 9.8 — SupersedePublishDialog

**Files:** `SupersedePublishDialog.tsx`, test.

**Acceptance Criteria:**
- [ ] Dialog invoked from doc detail of approved revision.
- [ ] Shows: new revision summary + diff-count from previously published + radio: `Publish now` / `Schedule for <datetime>`.
- [ ] If published revision exists in chain → checkbox "Supersede current published version" + shows current.
- [ ] Submit: calls `supersede` if checkbox checked; else `publish` or `schedule-publish`.
- [ ] Effective-from datetime picker: local tz, converted to UTC ISO on submit; min=now+5min if scheduling.
- [ ] Tests: publish happy, schedule happy, past date error surfaced, supersede path, capability gate.

**Steps:** Sonnet. Commit.

---

### Task 9.9 — RegistryDetailPanel integration

**Files:** modify `apps/web/src/features/documents_v2/components/RegistryDetailPanel.tsx`.

**Acceptance Criteria:**
- [ ] Adds "Approval" section below existing fields.
- [ ] Shows current `StateBadge` (doc.status), effective_from/effective_to, lock banner if locked.
- [ ] **Integrity panel (F8):** content_hash (truncated + copy button + full on hover), revision_version, current ETag. Drift warning rendered if server-pushed version > cached.
- [ ] Stale banner (F3) when cached data >30s old or 412 observed; "Refresh" CTA invalidates React Query.
- [ ] If `active_approval_instance` exists → embedded `ApprovalTimelinePanel`.
- [ ] Action buttons consult **transition policy table (F2)**; disabled state with `aria-describedby` → i18n reason.
- [ ] Permission degradation (F6): if capabilities change mid-session, buttons re-render per new caps; 403 on action → toast + refetch.
- [ ] Loading/empty/error/offline states per F4/F5.
- [ ] Tests: each state shows correct controls; transition table exhaustive; unauthorized hidden; offline banner; drift warning; stale banner.

**Steps:** Sonnet. Commit.

---

### Task 9.10 — i18n + labels

**Files:** `apps/web/src/i18n/approval.pt-BR.ts`, `approval.en.ts`.

**Acceptance Criteria:**
- [ ] pt-BR strings for all state labels, error messages, button labels (MetalDocs is Brazilian per memory).
- [ ] en strings.
- [ ] No hardcoded strings in components — all use `t("key")`.
- [ ] Linter rule to catch hardcoded UI strings.

**Steps:** Haiku. Commit.

---

### Task 9.11 — Codex COVERAGE review + risk-based test matrix gate (F11)

**Risk-based test matrix (merge blocked until all pass):**

| Risk | Test type | Location |
|---|---|---|
| 412 conflict handled across all mutations | unit+integration | `mutationClient.test.ts` + per-handler contract |
| 401 mid-session → re-auth | integration | `auth-degradation.test.ts` |
| 403 capability drift mid-session | integration | `capability-drift.test.ts` |
| Offline mutation queued + drained | integration | `offline-queue.test.ts` |
| Stale data banner triggers | integration | `stale-banner.test.ts` |
| Signoff adverse paths (bad_pw/session/rate/net) | integration | `SignoffDialog.test.tsx` |
| A11y (axe+keyboard+SR) | e2e Playwright | `a11y.spec.ts` |
| Datetime pt-BR/en + DST | unit | `datetime.test.ts` |
| Transition policy exhaustive | unit | `transitionPolicy.test.ts` |

**Steps:**
- [ ] Run all above before Codex call.
- [ ] Codex `gpt-5.3-codex` high, COVERAGE, plan-review framing. Verdict → `reviews/phase-9-round-1.json`.
- [ ] Fixes inline. Round 2 max.

---
