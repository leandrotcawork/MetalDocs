# Foundation Spec 2 — Document Approval State Machine

**Status:** Ready for implementation planning
**Date:** 2026-04-21
**Author:** Leandro Theodoro (MetalDocs) + Claude (Opus) co-designed with Codex validation (10 iterations)
**Depends on:** Spec 1 — Foundation Taxonomy × RBAC × Controlled Document Registry (2026-04-21-foundation-taxonomy-rbac-design.md)
**Scope:** Full approval lifecycle on `documents_v2` revisions — draft → under_review → approved → published → superseded/obsolete, with `rejected` rework sink. Sequential per-profile routes with role-pool quorum, password-reauth signatures with content_hash binding, edit lock during review, immediate/scheduled effective dates, audit-preserving reject history.

> Market target: SaaS professional QMS at parity with Qualio / Greenlight Guru / MasterControl for Brazilian metalworking ISO 9001 segment. Explicitly NOT pharma-grade 21 CFR Part 11 crypto-signed audit.

---

## Goal

Give `documents_v2` revisions a full approval lifecycle that:

1. Walks 6 states with DB-enforced transition legality: `draft → under_review → (approved | rejected→draft) → published → (superseded | obsolete)`.
2. Routes through sequential stages configured per `document_profile`, each stage = `{ role, area-from-doc, quorum }` resolving eligible actors from `user_process_areas`.
3. Binds signatures to content via SHA-256 hash + password re-auth; method pluggable (future ICP-Brasil seam).
4. Supports immediate OR scheduled future effective dates; prior revision auto-supersedes when new rev becomes `published`.
5. Locks revision content during in-flight review; unlocks on reject.
6. Emits `governance_events` in-transaction on every transition (no delivery — downstream spec).
7. Preserves rejected approval instances + all signoffs for audit; resubmit creates new instance.
8. Enforces SoD at DB level: same user cannot sign two stages on same revision; author ≠ any stage signer.

## Architecture

- **State lives on `documents_v2.status`** (extended enum). Child tables hold the in-flight machinery.
- **Route config is per-profile**, snapshotted into instance tables at submit time. In-flight approvals never re-read mutable profile config.
- **Hard invariants in DB** (triggers + partial unique indexes + immutable columns). Service layer is the happy path; DB is the backstop.
- **OCC via `revision_version` column + idempotency keys** on all transition commands.
- **Outbox-style**: `governance_events` written in same transaction as state change.
- **Scheduler**: single cron worker (`effective_date_publisher`) — UTC storage, idempotent via `SELECT FOR UPDATE SKIP LOCKED`, missed-run catchup window alerts rather than blind-processes.
- **Capability gates from Spec 1** fully consumed: `workflow.submit`, `workflow.review`, `workflow.approve`, `workflow.publish`, `workflow.reject`, `workflow.obsolete`, `workflow.supersede`. Mapped onto roles already defined.
- **Signature abstraction**: `signature_method` column + Go interface seam. Only `password_reauth` implementation ships in Spec 2.
- **DB-side authz defense-in-depth**: NOINHERIT role boundary + SECURITY DEFINER functions owned by dedicated role + session context tripwire + trigger-enforced integrity. Application-layer `iam.Check` remains the authoritative authorization boundary.

No new modules — extends `documents_v2` (state, transitions), `taxonomy/registry` (route config on profile), `iam` (capability additions). New subpackage `documents_v2/approval/` for instance/stage/signoff machinery.

---

## Data Model

### Extended `documents_v2.documents` columns

```sql
-- Replace 3-state enum with 7-state (+ legacy 'archived' retained for pre-migration compat)
ALTER TABLE documents_v2.documents
  DROP CONSTRAINT documents_status_check,
  ADD CONSTRAINT documents_status_check
    CHECK (status IN ('draft','under_review','approved','rejected',
                      'published','superseded','obsolete','archived'));

ALTER TABLE documents_v2.documents
  ADD COLUMN revision_number INT NOT NULL DEFAULT 1,
  ADD COLUMN revision_version INT NOT NULL DEFAULT 0,
  ADD COLUMN effective_from TIMESTAMPTZ,
  ADD COLUMN effective_to TIMESTAMPTZ,
  ADD COLUMN locked_at TIMESTAMPTZ,
  ADD COLUMN content_hash_at_submit TEXT;

-- Revision numbering unique per controlled_document
CREATE UNIQUE INDEX ux_documents_v2_cd_revision
  ON documents_v2.documents (controlled_document_id, revision_number);

-- At most one active (non-terminal) revision per controlled_document
CREATE UNIQUE INDEX ux_documents_v2_cd_active
  ON documents_v2.documents (controlled_document_id)
  WHERE status IN ('draft','under_review','approved','rejected');
```

### Legal transition trigger

```sql
CREATE FUNCTION enforce_document_transition() RETURNS trigger AS $$
BEGIN
  IF OLD.status IS DISTINCT FROM NEW.status THEN
    IF NOT (
      (OLD.status = 'draft'        AND NEW.status IN ('under_review','archived')) OR
      (OLD.status = 'under_review' AND NEW.status IN ('approved','rejected')) OR
      (OLD.status = 'rejected'     AND NEW.status = 'draft') OR
      (OLD.status = 'approved'     AND NEW.status IN ('published','draft')) OR
      (OLD.status = 'published'    AND NEW.status IN ('superseded','obsolete')) OR
      (OLD.status = 'superseded'   AND NEW.status = 'obsolete')
    ) THEN
      RAISE EXCEPTION 'illegal status transition % → %', OLD.status, NEW.status
        USING ERRCODE = 'check_violation';
    END IF;
  END IF;
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_documents_v2_legal_transition
  BEFORE UPDATE ON documents_v2.documents
  FOR EACH ROW EXECUTE FUNCTION enforce_document_transition();
```

### Route config (per-profile)

```sql
CREATE TABLE approval_routes (
  id UUID PRIMARY KEY,
  tenant_id UUID NOT NULL,
  profile_code TEXT NOT NULL,
  version INT NOT NULL DEFAULT 1,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (tenant_id, profile_code),
  FOREIGN KEY (tenant_id, profile_code)
    REFERENCES document_profiles (tenant_id, code)
);

CREATE TABLE approval_route_stages (
  id UUID PRIMARY KEY,
  route_id UUID NOT NULL REFERENCES approval_routes(id) ON DELETE CASCADE,
  stage_order INT NOT NULL,
  name TEXT NOT NULL,
  required_role TEXT NOT NULL,
  required_capability TEXT NOT NULL,
  quorum TEXT NOT NULL CHECK (quorum IN ('any_1_of','all_of','m_of_n')),
  quorum_m INT,
  on_eligibility_drift TEXT NOT NULL DEFAULT 'reduce_quorum'
    CHECK (on_eligibility_drift IN ('reduce_quorum','fail_stage','keep_snapshot')),
  UNIQUE (route_id, stage_order)
);
```

### Approval instance (runtime per revision)

```sql
CREATE TABLE approval_instances (
  id UUID PRIMARY KEY,
  tenant_id UUID NOT NULL,
  document_v2_id UUID NOT NULL REFERENCES documents_v2.documents(id),
  route_id UUID NOT NULL REFERENCES approval_routes(id),
  route_version_snapshot INT NOT NULL,
  status TEXT NOT NULL CHECK (status IN ('in_progress','approved','rejected','cancelled')),
  submitted_by UUID NOT NULL REFERENCES users(id),
  submitted_at TIMESTAMPTZ NOT NULL,
  completed_at TIMESTAMPTZ,
  content_hash_at_submit TEXT NOT NULL,
  idempotency_key TEXT NOT NULL,
  UNIQUE (document_v2_id, idempotency_key)
);

-- At most one in-progress instance per revision
CREATE UNIQUE INDEX ux_approval_instances_active
  ON approval_instances (document_v2_id)
  WHERE status = 'in_progress';

CREATE TABLE approval_stage_instances (
  id UUID PRIMARY KEY,
  approval_instance_id UUID NOT NULL REFERENCES approval_instances(id) ON DELETE CASCADE,
  stage_order INT NOT NULL,
  name_snapshot TEXT NOT NULL,
  required_role_snapshot TEXT NOT NULL,
  required_capability_snapshot TEXT NOT NULL,
  area_code_snapshot TEXT NOT NULL,
  quorum_snapshot TEXT NOT NULL,
  quorum_m_snapshot INT,
  on_eligibility_drift_snapshot TEXT NOT NULL,
  eligible_actor_ids JSONB NOT NULL,
  effective_denominator INT,
  status TEXT NOT NULL CHECK (status IN ('pending','active','completed','skipped','rejected_here')),
  opened_at TIMESTAMPTZ,
  completed_at TIMESTAMPTZ,
  UNIQUE (approval_instance_id, stage_order)
);

CREATE TABLE approval_signoffs (
  id UUID PRIMARY KEY,
  stage_instance_id UUID NOT NULL REFERENCES approval_stage_instances(id),
  actor_user_id UUID NOT NULL REFERENCES users(id),
  decision TEXT NOT NULL CHECK (decision IN ('approve','reject')),
  comment TEXT,
  signed_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  signature_method TEXT NOT NULL,
  signature_payload JSONB NOT NULL,
  content_hash TEXT NOT NULL,
  UNIQUE (stage_instance_id, actor_user_id)
);
```

### Signoff invariants

```sql
-- SoD: actor cannot be document author; actor cannot have signed another stage in same instance
CREATE FUNCTION enforce_signoff_sod() RETURNS trigger AS $$
DECLARE
  author_id UUID;
  already_signed INT;
BEGIN
  SELECT d.created_by INTO author_id
    FROM approval_stage_instances s
    JOIN approval_instances i ON i.id = s.approval_instance_id
    JOIN documents_v2.documents d ON d.id = i.document_v2_id
   WHERE s.id = NEW.stage_instance_id;

  IF NEW.actor_user_id = author_id THEN
    RAISE EXCEPTION 'SoD: author cannot sign own revision'
      USING ERRCODE = 'check_violation';
  END IF;

  SELECT COUNT(*) INTO already_signed
    FROM approval_signoffs so
    JOIN approval_stage_instances s ON s.id = so.stage_instance_id
    JOIN approval_stage_instances target ON target.id = NEW.stage_instance_id
   WHERE s.approval_instance_id = target.approval_instance_id
     AND so.actor_user_id = NEW.actor_user_id
     AND so.stage_instance_id <> NEW.stage_instance_id;

  IF already_signed > 0 THEN
    RAISE EXCEPTION 'SoD: actor already signed another stage in this instance'
      USING ERRCODE = 'check_violation';
  END IF;

  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_signoff_sod
  BEFORE INSERT ON approval_signoffs
  FOR EACH ROW EXECUTE FUNCTION enforce_signoff_sod();

-- Signoff rows immutable
CREATE FUNCTION reject_signoff_update() RETURNS trigger AS $$
BEGIN
  RAISE EXCEPTION 'approval_signoffs rows are immutable'
    USING ERRCODE = 'check_violation';
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_signoff_immutable
  BEFORE UPDATE ON approval_signoffs
  FOR EACH ROW EXECUTE FUNCTION reject_signoff_update();
```

### `user_process_areas` hardening (Spec 2 additions)

```sql
-- Add revoke attribution column
ALTER TABLE user_process_areas
  ADD COLUMN revoked_by UUID,
  ADD CONSTRAINT revoked_by_required_when_revoked
    CHECK ((effective_to IS NULL AND revoked_by IS NULL)
        OR (effective_to IS NOT NULL AND revoked_by IS NOT NULL)),
  ADD CONSTRAINT effective_interval_valid
    CHECK (effective_to IS NULL OR effective_to > effective_from);

-- Tenant-scoped composite FKs for audit attribution (requires users UNIQUE (tenant_id, id))
ALTER TABLE user_process_areas
  DROP CONSTRAINT IF EXISTS user_process_areas_granted_by_fkey,
  ADD CONSTRAINT user_process_areas_granted_by_same_tenant
    FOREIGN KEY (tenant_id, granted_by) REFERENCES users(tenant_id, id),
  ADD CONSTRAINT user_process_areas_revoked_by_same_tenant
    FOREIGN KEY (tenant_id, revoked_by) REFERENCES users(tenant_id, id);

-- At most one active row per (tenant, user, area, role)
CREATE UNIQUE INDEX ux_user_process_areas_single_active
  ON user_process_areas (tenant_id, user_id, area_code, role)
  WHERE effective_to IS NULL;

-- Forbid DELETE
CREATE FUNCTION reject_user_process_areas_delete() RETURNS trigger AS $$
BEGIN
  RAISE EXCEPTION 'user_process_areas rows cannot be deleted (revoke via UPDATE effective_to)'
    USING ERRCODE = 'check_violation';
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_user_process_areas_no_delete
  BEFORE DELETE ON user_process_areas
  FOR EACH ROW EXECUTE FUNCTION reject_user_process_areas_delete();

-- Enforce identity immutability + no un-revoke
CREATE FUNCTION enforce_user_process_areas_update_contract() RETURNS trigger AS $$
BEGIN
  IF NEW.tenant_id     IS DISTINCT FROM OLD.tenant_id     OR
     NEW.user_id       IS DISTINCT FROM OLD.user_id       OR
     NEW.area_code     IS DISTINCT FROM OLD.area_code     OR
     NEW.role          IS DISTINCT FROM OLD.role          OR
     NEW.effective_from IS DISTINCT FROM OLD.effective_from OR
     NEW.granted_by    IS DISTINCT FROM OLD.granted_by    THEN
    RAISE EXCEPTION 'identity columns are immutable on user_process_areas'
      USING ERRCODE = 'check_violation';
  END IF;
  IF OLD.effective_to IS NOT NULL AND NEW.effective_to IS NULL THEN
    RAISE EXCEPTION 'cannot un-revoke membership (re-grant creates new row)'
      USING ERRCODE = 'check_violation';
  END IF;
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_user_process_areas_update_contract
  BEFORE UPDATE ON user_process_areas
  FOR EACH ROW EXECUTE FUNCTION enforce_user_process_areas_update_contract();
```

### DB-side authz boundary (roles + canonical functions)

```sql
-- Role hierarchy
CREATE ROLE metaldocs_security_owner NOLOGIN;
CREATE ROLE metaldocs_membership_writer NOLOGIN NOINHERIT;
ALTER ROLE metaldocs_app NOINHERIT;

-- Owner role has minimum privileges needed to execute function body
GRANT SELECT ON users TO metaldocs_security_owner;
GRANT SELECT, INSERT, UPDATE ON user_process_areas TO metaldocs_security_owner;

-- Schema public lockdown (defense-in-depth)
REVOKE CREATE ON SCHEMA public FROM PUBLIC;
REVOKE CREATE ON SCHEMA public FROM metaldocs_app;
REVOKE CREATE ON SCHEMA public FROM metaldocs_membership_writer;
REVOKE CREATE ON SCHEMA public FROM metaldocs_readonly;
GRANT  USAGE  ON SCHEMA public TO metaldocs_app, metaldocs_membership_writer, metaldocs_readonly;
GRANT  CREATE ON SCHEMA public TO metaldocs_migrator, metaldocs_security_owner;

-- Canonical mutation functions (SECURITY DEFINER, hardened search_path, schema-qualified)
CREATE OR REPLACE FUNCTION public.grant_area_membership(
  _tenant_id UUID, _user_id UUID, _area_code TEXT, _role TEXT, _granted_by UUID
) RETURNS UUID AS $$
DECLARE
  session_actor TEXT := current_setting('metaldocs.actor_id', true);
  session_cap   TEXT := current_setting('metaldocs.verified_capability', true);
  actor_tenant  UUID;
BEGIN
  IF session_actor IS NULL OR session_actor::UUID IS DISTINCT FROM _granted_by THEN
    RAISE EXCEPTION 'session actor context missing or mismatched'
      USING ERRCODE = 'insufficient_privilege';
  END IF;
  IF session_cap IS NULL OR session_cap <> 'workflow.route.edit' THEN
    RAISE EXCEPTION 'session capability context missing'
      USING ERRCODE = 'insufficient_privilege';
  END IF;
  SELECT tenant_id INTO actor_tenant FROM public.users WHERE id = _granted_by;
  IF actor_tenant IS DISTINCT FROM _tenant_id THEN
    RAISE EXCEPTION 'granted_by must belong to same tenant' USING ERRCODE='check_violation';
  END IF;
  IF NOT EXISTS (SELECT 1 FROM public.users
                  WHERE id = _granted_by AND deactivated_at IS NULL) THEN
    RAISE EXCEPTION 'granted_by must be active user' USING ERRCODE='check_violation';
  END IF;
  INSERT INTO public.user_process_areas
    (user_id, tenant_id, area_code, role, effective_from, effective_to, granted_by, revoked_by)
    VALUES (_user_id, _tenant_id, _area_code, _role,
            pg_catalog.clock_timestamp(), NULL, _granted_by, NULL);
  RETURN pg_catalog.gen_random_uuid();
END;
$$ LANGUAGE plpgsql
   SECURITY DEFINER
   SET search_path = pg_catalog, public;

ALTER FUNCTION public.grant_area_membership(UUID, UUID, TEXT, TEXT, UUID)
  OWNER TO metaldocs_security_owner;

-- revoke_area_membership follows same template: session context assert,
-- tenant consistency, active-user check, UPDATE of active row, RAISE on not-found.

-- Privilege binding
REVOKE EXECUTE ON FUNCTION public.grant_area_membership(UUID, UUID, TEXT, TEXT, UUID),
                        public.revoke_area_membership(UUID, UUID, TEXT, TEXT, UUID)
  FROM PUBLIC, metaldocs_app;

GRANT EXECUTE ON FUNCTION public.grant_area_membership(UUID, UUID, TEXT, TEXT, UUID),
                       public.revoke_area_membership(UUID, UUID, TEXT, TEXT, UUID)
  TO metaldocs_membership_writer;

GRANT metaldocs_membership_writer TO metaldocs_app;
-- PG 16+: GRANT metaldocs_membership_writer TO metaldocs_app WITH INHERIT FALSE, SET TRUE;

REVOKE INSERT, UPDATE, DELETE ON user_process_areas FROM metaldocs_app;
GRANT  SELECT ON user_process_areas TO metaldocs_app;
```

App-layer call path:

```go
func (s *Service) Grant(ctx context.Context, actor User, target User, area, role string) error {
    if err := s.authz.Check(ctx, actor, "workflow.route.edit",
        ResourceCtx{AreaCode: area}); err != nil {
        return err
    }
    return s.db.Tx(ctx, func(tx *sql.Tx) error {
        if _, err := tx.ExecContext(ctx, `SET LOCAL ROLE metaldocs_membership_writer`); err != nil {
            return err
        }
        if _, err := tx.ExecContext(ctx, `SET LOCAL metaldocs.actor_id = $1`, actor.ID); err != nil {
            return err
        }
        if _, err := tx.ExecContext(ctx,
            `SET LOCAL metaldocs.verified_capability = 'workflow.route.edit'`); err != nil {
            return err
        }
        _, err := tx.ExecContext(ctx,
            `SELECT public.grant_area_membership($1,$2,$3,$4,$5)`,
            target.TenantID, target.ID, area, role, actor.ID)
        return err
    })
}
```

### Capability map extension

```go
// RoleCapabilitiesVersion bumps to 2
var RoleCapabilities = map[string][]string{
    "viewer":   {"document.view", "template.view"},
    "editor":   {"document.view", "document.create", "document.edit",
                 "workflow.submit", "template.view"},
    "reviewer": {"document.view", "document.edit",
                 "workflow.submit", "workflow.review", "template.view"},
    "approver": {"document.view", "workflow.approve", "workflow.publish",
                 "workflow.supersede", "workflow.reject",
                 "template.view", "template.publish"},
    // Admin-only (AccessPolicy or is_admin flag): workflow.obsolete,
    //   workflow.route.edit, workflow.instance.cancel
}
```

---

## Components

### Backend (Go)

```
internal/modules/documents_v2/
  approval/
    domain/
      state.go                     ← state enum + CanTransition() matrix
      route.go                     ← Route + Stage value objects
      instance.go                  ← Instance + StageInstance aggregates
      signoff.go                   ← Signoff + content_hash computation
      quorum.go                    ← any_1_of | all_of | m_of_n evaluator
      sod.go                       ← service-layer SoD pre-check
    application/
      submit_service.go            ← draft → under_review
      decision_service.go          ← review/approve/reject at active stage
      publish_service.go           ← approved → published (immediate)
      scheduler_service.go         ← cron-invoked scheduled publish
      supersede_service.go         ← explicit supersede at publish
      obsolete_service.go          ← manual obsolete (admin only)
      content_hash.go              ← canonical JSON SHA-256
      idempotency.go               ← key dedup per transition
      events.go                    ← same-TX governance_events writer
      membership_tx.go             ← helper wrapping SET LOCAL ROLE + context GUCs
    infra/
      signature/
        provider.go                ← interface Signature{ Verify(ctx,userID,password,payload) }
        password_reauth.go         ← bcrypt compare via iam service
        registry.go                ← method → provider lookup
    delivery/http/
      routes_route.go              ← /api/v2/taxonomy/profiles/{code}/route (GET/PUT)
      routes_transitions.go        ← /api/v2/documents/{id}/{submit|decision|publish|supersede}
                                      /api/v2/controlled-documents/{id}/obsolete
      routes_inbox.go              ← /api/v2/workflow/inbox
      handler.go
    jobs/
      effective_date_publisher.go  ← cron every 60s
      stuck_instance_watchdog.go   ← daily; metrics only, no transitions

internal/modules/documents_v2/repository/
  approval_repository.go           ← NEW: instance/stage/signoff queries + OCC
  repository.go                    ← EXTEND: transition helper + lock/unlock

internal/modules/iam/
  domain/role_capabilities.go      ← EXTEND: workflow.* caps, bump version to 2
  application/area_membership_service.go  ← EXTEND: use canonical DB functions
```

### Frontend (React / TS)

```
frontend/apps/web/src/features/
  approval/
    RouteAdminPage.tsx             ← per-profile stage editor
    InboxPage.tsx                  ← "Minha caixa" pending stages
    SignoffDialog.tsx              ← approve/reject + comment + password re-auth
    ApprovalTimelinePanel.tsx      ← instance history on doc detail
    SupersedePublishDialog.tsx     ← immediate vs scheduled date/time

  documents/v2/
    DocumentEditorPage.tsx         ← MODIFY: disable editor on lock; stage banner
    DocumentStatusBadge.tsx        ← 7-state badge
    RejectionBanner.tsx            ← shows last signoff comment + rework action

  registry/
    RegistryDetailPage.tsx         ← MODIFY: revision list with status/effective dates;
                                     "Obsoletar documento" (admin only)
```

### HTTP endpoints

```
POST   /api/v2/documents/:id/submit                    { idempotency_key }
POST   /api/v2/documents/:id/decision                  { decision, comment, password, idempotency_key }
POST   /api/v2/documents/:id/publish                   { effective_from?, idempotency_key }
POST   /api/v2/documents/:id/supersede                 { new_revision_id }
POST   /api/v2/controlled-documents/:id/obsolete       { reason }
GET    /api/v2/workflow/inbox                          ?area=&stage_role=
GET    /api/v2/documents/:id/approval                  → instance + stages + signoffs
GET    /api/v2/taxonomy/profiles/:code/route           → current route + stages
PUT    /api/v2/taxonomy/profiles/:code/route           { stages: [...] }
```

---

## Data Flow

### 1. Submit for review (`draft → under_review`)

```
Author → POST /api/v2/documents/{id}/submit { idempotency_key }

1. BEGIN TX.
2. Load document FOR UPDATE. Assert status='draft', locked_at IS NULL.
3. Load controlled_document → derive area_code.
4. Load approval_routes by (tenant, profile_code). If missing → 409 route_not_configured.
5. Load approval_route_stages. If empty → 409 route_has_no_stages.
6. Compute content_hash = SHA-256(canonical JSON of revision body).
7. INSERT approval_instances.
     ON CONFLICT (document_v2_id, idempotency_key) → return existing (idempotent).
8. For each route stage:
     - Snapshot all stage fields.
     - area_code_snapshot = controlled_document.area_code.
     - eligible_actor_ids = user_process_areas active members matching role + area.
     - Pre-flight: if eligible_count < required
         (any_1_of:1, all_of:1, m_of_n:quorum_m) → ROLLBACK, 409 stage_unsatisfiable
         with suggested_action { type:"grant_area_role", area_code, role }.
     - status = (stage_order=1) ? 'active' : 'pending'.
9. UPDATE documents_v2.documents
     SET status='under_review', locked_at=now(),
         content_hash_at_submit=hash, revision_version=revision_version+1
     WHERE id=$1 AND revision_version=$expected.
     If 0 rows → 409 version_conflict.
10. INSERT governance_events { event_type:'workflow.submit', ... }.
11. COMMIT.
```

### 2. Stage decision (review/approve/reject)

```
Signer → POST /api/v2/documents/{id}/decision
  body: { decision, comment, password, idempotency_key }

1. Verify password via signature provider.
   Fail → 401 reauth_failed. Rate-limit: 5/15min → 423.
2. BEGIN TX ISOLATION LEVEL READ COMMITTED; SET LOCAL timezone='UTC';
3. now_tx := CURRENT_TIMESTAMP.
4. SELECT document FOR UPDATE. Assert status='under_review'.
5. SELECT instance + active stage FOR UPDATE.
6. Authz service-layer:
   - actor.id ∈ stage.eligible_actor_ids.
   - actor.id != document.created_by (DB trigger backstop).
   - actor hasn't signed another stage (DB trigger backstop).
7. SELECT user_process_areas snapshot rows FOR SHARE
     WHERE user_id = ANY(eligible_actor_ids) AND area+role match
       AND effective_from <= now_tx AND (effective_to IS NULL OR effective_to > now_tx).
   → current_active set.
8. Apply on_eligibility_drift_snapshot policy:
     policy=keep_snapshot OR reduce_quorum:
       signable_pool = eligible_actor_ids ∩ current_active
     policy=fail_stage:
       if |snapshot| != |current_active| → stage auto-fails (same path as reject).
     required_for_advance:
       any_1_of: 1; all_of: |signable_pool| (reduce) or |snapshot| (keep); m_of_n: quorum_m.
     if |signable_pool| < required_for_advance → stage auto-fails.
9. Verify content_hash(doc) == instance.content_hash_at_submit.
   Mismatch → 500 content_hash_drift + alert; no stage advance.
10. INSERT approval_signoffs.
     DB triggers re-verify SoD + immutability + uniqueness.
11a. decision=reject:
      - stage → 'rejected_here', instance → 'rejected',
        document → 'rejected', locked_at=NULL, revision_version++.
      - governance_event 'workflow.reject'.
11b. decision=approve + quorum met:
      - stage → 'completed'.
      - next stage exists → activate.
      - no next stage → instance → 'approved', document → 'approved',
        revision_version++, locked_at retained until publish.
      - governance_event 'workflow.stage.complete' / 'workflow.approved'.
11c. decision=approve + quorum not yet met: stage stays 'active', COMMIT.
12. COMMIT.
```

### 3. Publish

```
Approver → POST /api/v2/documents/{id}/publish { effective_from?, idempotency_key }

1. BEGIN TX.
2. Load document FOR UPDATE. Assert status='approved'.
3. authz.Check(actor, 'workflow.publish', {area: snapshot}).
4. effective_from = body.effective_from ?? now().
5. If effective_from > now():
     UPDATE document SET effective_from=$1, revision_version++.
     INSERT governance_events 'workflow.publish.scheduled'.
     COMMIT.
   Else (immediate):
     Find prior published rev of same controlled_document.
     UPDATE prior SET status='superseded', effective_to=effective_from, revision_version++.
     UPDATE new SET status='published', effective_from=$1, revision_version++.
     INSERT governance_events 'workflow.publish' + 'workflow.supersede'.
     COMMIT.
```

### 4. Scheduled publish (cron)

```
effective_date_publisher runs every 60s:
1. BEGIN TX per candidate.
2. SELECT id FROM documents_v2.documents
     WHERE status='approved' AND effective_from <= now()
     FOR UPDATE SKIP LOCKED LIMIT 100.
3. For each: publish_service internal path, actor='system:scheduler',
   idempotency_key=fmt("auto-pub-%s-%s", doc.id, effective_from).
4. Rows with effective_from < now() - 1h → process + metric workflow.publish.delayed.
5. Rows with effective_from < now() - 24h → skip + alert.
6. COMMIT per doc.
```

### 5. Reject → resubmit

```
After reject, document.status='rejected', locked_at=NULL.
Author explicit transition 'rejected' → 'draft' (governance_event).
Edit allowed in draft.
Fresh submit creates NEW approval_instance (prior stays for audit timeline).
```

### 6. Obsolete

```
Admin → POST /api/v2/controlled-documents/{id}/obsolete { reason }

1. authz.Check(actor, 'workflow.obsolete').
2. reason non-empty, ≥10 chars.
3. BEGIN TX.
4. UPDATE controlled_documents SET status='obsolete'.
5. UPDATE documents_v2.documents SET status='obsolete', effective_to=now()
     WHERE controlled_document_id=$1 AND status IN ('published','superseded','approved').
6. Cancel in-flight instance (status='in_progress' → 'cancelled').
7. governance_event 'workflow.obsolete'.
8. COMMIT.
```

### Content hash canonicalization

```
canonical_json = json.Marshal with:
  - sorted object keys (recursive)
  - numbers as shortest decimal
  - no whitespace
SHA-256 over UTF-8 bytes → lowercase hex.
```

Same implementation in Go + TS (shared test fixture ensures cross-layer parity).

### Quorum deadlock resolution

Admin-only `workflow.instance.cancel` capability: `UPDATE approval_instances SET status='cancelled'`, `UPDATE document SET status='draft', locked_at=NULL`. Governance event with reason. User resubmits → fresh snapshot, fresh pool.

---

## Error Handling

### HTTP mapping

| Category | HTTP | Shape |
|---|---|---|
| Validation | 400 | `{ code, field, message }` |
| Reauth failed | 401 | `{ code:"reauth_failed" }` |
| Authz | 403 | `{ code, required_capability, area_code }` |
| Not found | 404 | `{ code, resource }` |
| Conflict | 409 | `{ code, detail, suggested_action? }` |
| SoD / semantic | 422 | `{ code, stage, actor, detail }` |
| Locked / rate-limit | 423 | `{ code, retry_after }` |
| Server | 500 | `{ code, trace_id }` |

### Specific conditions

**Submit:**
- `403 workflow.submit` — actor lacks cap in doc's area.
- `409 document_locked` — `locked_at != NULL`.
- `409 route_not_configured` — `suggested_action: admin_configure_route`.
- `409 route_has_no_stages`.
- `409 stage_unsatisfiable` — pre-flight quorum short. Payload includes `stage_order, quorum, required, eligible_count, suggested_action`.
- `409 version_conflict` — OCC mismatch.

**Decision:**
- `401 reauth_failed`. Rate limit 5/15min → `423 account_locked_reauth`.
- `403 not_eligible`.
- `422 sod_author_cannot_sign`.
- `422 sod_cross_stage_duplicate`.
- `409 stage_not_active`.
- `409 document_not_under_review`.
- `409 already_signed_this_stage`.
- `409 stage_auto_failed_unsatisfiable` — eligibility drift triggered auto-fail.
- `500 content_hash_drift` — alert fires.

**Publish:**
- `403 workflow.publish`.
- `409 not_approved`.
- `400 effective_from_in_past` (>1 min past).
- `409 effective_from_conflicts_prior`.

**Scheduler:**
- Transient DB error per row → next tick retries. Metric `workflow.publish.scheduler.retry`.
- >1h delay → warning metric `workflow.publish.delayed`.
- >24h delay → page oncall, manual intervention.

**Supersede:**
- `409 new_revision_not_published`.
- `409 prior_already_superseded` — idempotent 200.

**Obsolete:**
- `403 not_admin`.
- `400 reason_required` (<10 chars).
- `409 already_obsolete` — idempotent 200.

**Route config (PUT):**
- `403 not_admin`.
- `400 stages_empty`.
- `400 stage_order_not_contiguous`.
- `400 invalid_on_eligibility_drift`.
- `200 with warning` if in-flight instances exist (snapshot protects them).

**Membership function errors (should not occur via API):**
- `400 granted_by_tenant_mismatch` / `400 revoked_by_tenant_mismatch`.
- `400 actor_deactivated`.
- `409 no_active_membership_to_revoke`.
- `500 membership_contract_violation` — trigger fire (integrity breach).

### Concurrency

- **OCC on every transition** via `revision_version` bump.
- **Idempotency keys** on submit/decision/publish/obsolete.
- **`FOR UPDATE`** on doc row at start of every transition.
- **`FOR SHARE`** on `user_process_areas` snapshot rows during decision — blocks concurrent revoke.
- **Scheduler uses `SELECT FOR UPDATE SKIP LOCKED`** — multi-worker safe.

### Race resolutions

| Race | Outcome |
|---|---|
| Two signers approve any_1_of stage simultaneously | Both signoffs insert (UNIQUE on stage+actor). First to commit completes stage; second sees stage 'completed' → 409 stage_not_active. |
| Submit + concurrent edit | `FOR UPDATE` + OCC → second → 409 version_conflict. |
| Scheduler vs manual publish | `FOR UPDATE` serializes; second sees status≠approved → 409. |
| Route edit mid-review | Zero interference (instance uses snapshot). Editor bumps route.version. |
| Reject + concurrent approve | `FOR UPDATE` + trigger → second sees illegal transition → 409. |
| Revoke + decision race | `FOR SHARE` holds decision TX until commit; revoke UPDATE blocks. |

---

## Testing Approach

### Unit (Go)

```
approval/domain/
  state_test.go                     — transition matrix + property test (no path reaches
                                      published without passing under_review+approved)
  quorum_test.go                    — every quorum type happy/sad; unsatisfiability auto-fail
  sod_test.go                       — author signs blocked; cross-stage duplicate blocked
  content_hash_test.go              — canonicalization parity Go + TS via shared fixture

approval/application/
  submit_service_test.go            — happy, route missing, pre-flight fail, idempotency, OCC
  decision_service_test.go          — quorum paths, reject, reauth_failed,
                                      content_hash_drift, concurrent race,
                                      drift policies × deactivation scenarios
  publish_service_test.go           — immediate, scheduled, supersede prior,
                                      effective_from past, first revision
  scheduler_service_test.go         — FOR UPDATE SKIP LOCKED, catchup, idempotent retry
  obsolete_service_test.go          — cancels in-flight, reason required, idempotent

iam/role_capabilities_test.go       — version bump logged on boot
```

### Integration (Go + Postgres)

```
approval_integration_test.go
  — Full happy path end-to-end
  — Reject + resubmit (new instance, prior preserved)
  — Scheduled publish flip via scheduler
  — Route version drift: instance uses v1 after v2 edited mid-review
  — DB trigger bypass attempts (direct SQL):
    * UPDATE status illegal → trigger reject
    * INSERT signoff author as actor → trigger reject
    * INSERT double-stage signoff same actor → trigger reject
    * UPDATE approval_signoffs → trigger reject
    * DELETE user_process_areas → trigger reject
    * UPDATE user_process_areas identity columns → trigger reject
    * UPDATE user_process_areas un-revoke → trigger reject
  — Concurrent decision any_1_of → one 200, one 409
  — Cross-tenant signoff → composite FK reject
  — Obsolete cascade (in-flight cancel + all revs flip)
  — Quorum deadlock resolution via admin cancel
  — Membership function gates:
    * Call function as metaldocs_app without SET ROLE → permission denied
    * With SET LOCAL ROLE metaldocs_membership_writer but no context GUCs →
      insufficient_privilege
    * With SET LOCAL ROLE + context GUCs + iam.Check passed → success
    * Tenant-mismatch attribution → function reject
  — Schema public lockdown:
    * metaldocs_app CREATE TABLE public.X → permission denied
    * Search-path hijack: attacker schema shadow → function still uses public.X
  — Role inheritance:
    * pg_roles.rolinherit=FALSE for app + writer
    * App without SET ROLE cannot execute writer functions
```

### E2E (Playwright)

```
e2e/approval_happy.spec.ts         — configure route, submit, review approve, approver approve,
                                     publish immediate, timeline renders
e2e/approval_reject.spec.ts        — reject with comment, rework, resubmit, prior instance visible
e2e/approval_scheduled.spec.ts     — scheduled effective_from, cron flip
e2e/approval_sod.spec.ts           — author cannot self-sign
e2e/approval_lock.spec.ts          — editor locks on submit, unlocks on reject
e2e/approval_quorum.spec.ts        — all_of requires both approvals
e2e/route_admin.spec.ts            — edit route; in-flight unaffected
```

### Property / invariant tests

Generated command sequences assert:

1. `documents_v2.status` never leaves state graph.
2. Zero rows with two `in_progress` instances per revision.
3. For approved instances, signoff actors ∩ {document author} = ∅.
4. For any approval_instance, no actor appears in two stage_instances.
5. `published` rev has exactly one prior `superseded` predecessor (unless first rev).
6. `content_hash_at_submit` on instance equals `content_hash` on every signoff in that instance.

### Coverage targets

| Module | Target |
|---|---|
| `approval/domain/*` | 90% |
| `approval/application/*` | 85% |
| `approval/infra/signature/*` | 80% |
| `approval_repository.go` | 80% |
| DB triggers (integration) | 100% of invariant cases |

### Manual smoke

- [ ] Configure route for `po` (2 stages) → visible in route admin.
- [ ] Submit with no route → 409 with `suggested_action`.
- [ ] Submit with no eligible signers → 409 with actionable error.
- [ ] Approve with wrong password → 401, no signoff.
- [ ] Reject at stage 2 → doc → rejected; rework; old instance in timeline.
- [ ] Schedule publish 1 min out → cron flips.
- [ ] Admin obsoletes mid-review → instance cancelled, revs flipped.
- [ ] Direct `UPDATE documents SET status='published' FROM 'draft'` → trigger error.
- [ ] Delete user mid-review sole `all_of` signer → deadlock; admin cancel recovers.

### Perf

- Transition endpoints p99 < 150 ms at 50 concurrent users.
- Scheduler: 1000 scheduled docs processed within 2 min single worker.
- Inbox query p99 < 100 ms at 10k open stage_instances (GIN on `eligible_actor_ids`).

### CI invariants

- `pg_proc.proowner = metaldocs_security_owner` for every `SECURITY DEFINER` function.
- `proconfig` contains `search_path=pg_catalog, public` for every `SECURITY DEFINER` function.
- `pg_roles.rolinherit=FALSE` for `metaldocs_app`, `metaldocs_membership_writer`.
- `has_schema_privilege(role, 'public', 'CREATE') = FALSE` for all non-privileged roles.
- Pre-merge linter: SECURITY DEFINER bodies contain no unqualified object references.
- Fails deploy on any drift.

---

## Out of Scope

### Deferred to later specs

| Item | Spec |
|---|---|
| Notification delivery (email, in-app, digest, unsubscribe) | Post-Spec 4 |
| Placeholder fill-in form + eigenpal variable fanout | Spec 3 |
| Full audit log coverage (session, autosave, read events, field-level changes) | Spec 4 |
| Audit viewer UI | Spec 4 |
| Scheduled re-review cron (`review_interval_days` enforcement) | Later |
| Retention auto-archive cron (`retention_days` enforcement) | Later |
| Validity-period enforcement (`validity_days`) | Later |
| Per-controlled-document route override | Later |
| Stage-level SLA timers (auto-escalate if stage active > N days) | Later |
| Delegation ("user X delegates role to Y for 2 weeks") | Later |
| Reviewer comment threads / inline annotations | Later |
| Parallel stage execution | Later (sequential only) |

### Explicit non-goals

- **ICP-Brasil / PKI signatures.** Only `password_reauth` ships. Schema + interface seam ready; concrete provider is future work.
- **Cryptographic signing of content bytes.** `content_hash` binds signature to what was signed, but signatures are password-reauth proofs. Sufficient for ISO 9001; not MP 2.200-2 legal weight.
- **Custom state machines per profile.** One fixed 6-state graph. Different profiles have different routes (stages/roles/quorum) but top-level state enum is universal.
- **Per-stage custom forms.** Signoff = decision + comment + re-auth. No checklists, risk ratings, evidence attachments.
- **Reviewer workload balancing.** Inbox shows all eligible docs to all eligible users; first to act wins.
- **Bulk operations.** One doc, one transition per request.
- **Revision comparison / diff UI at signoff.**
- **Rollback-by-approver** (published → draft). Admin-only via explicit cancel/rollback API.
- **External approver (email magic-link).** All signers must be MetalDocs users with area membership.
- **Approval templates / copy-from-another-profile UI.**
- **Historical comment migration.** Legacy comments stay on original doc rows, not transferred across revisions.
- **Multi-language signoff comments.** Free-text, no i18n structure.
- **Parallel approval instances per revision.** `UNIQUE` index enforces one in-flight; prior rejected stays historical.

### Accepted residual risks

**1. DB-side authz relies on application-layer coupling.**

Membership-mutation path layers four defenses:

1. Application `iam.Check` (authoritative business authorization).
2. NOINHERIT role boundary (must explicitly `SET LOCAL ROLE metaldocs_membership_writer`).
3. Session context tripwire (`metaldocs.actor_id` + `metaldocs.verified_capability` via `SET LOCAL`).
4. DB integrity triggers (tenant consistency, temporal bounds, signoff/identity immutability).

A full compromise requires SQL-injection (or equivalent arbitrary SQL execution) in the application layer, allowing the attacker to issue `SET LOCAL ROLE` + `SET LOCAL metaldocs.*` + call function. This is the same attack surface as any data-exfiltration path in the application.

**Session context GUCs are tripwire, not authorization source.** They force developer awareness (one centralized helper) and generate anomaly events on mismatch. They do not replace `iam.Check`. The authoritative authorization boundary is the application service.

**Mitigations accepted as sufficient for MetalDocs MVP SaaS posture:**
- All Go DB calls use parameterized queries (`database/sql` standard); no string-concat SQL.
- Pre-commit static analysis (`golangci-lint` with `sqlclosecheck`, `rowserrcheck`, custom rule forbidding raw SQL concatenation) in CI.
- Row-level security planned for future spec (per-tenant).
- Cryptographic capability tokens validated at DB function boundary are explicitly deferred (requires KMS/HSM key management; justified when SaaS graduates to regulated-industry tier beyond ISO 9001 metalworking).

Industry reference: Qualio, Greenlight Guru, MasterControl operate with application-layer gates + parameterized queries + audit logging as the standard for ISO 9001 QMS SaaS tier. Per-call cryptographic authz tokens are pharma-grade controls (Veeva Vault 21 CFR Part 11), not standard in ISO 9001 metalworking SaaS.

**2. Quorum deadlock requires admin cancel (no auto-escalation).**

When `all_of` / `m_of_n` quorum becomes unreachable via deactivation drift, auto-fail triggers and bounces to draft. When multiple users remain but no consensus forms (slow decisions), no automatic timeout fires — admin must manually cancel via `workflow.instance.cancel`. Stage SLA auto-escalation deferred to later spec.

**3. Privileged migration escape hatch.**

`session_replication_role = 'replica'` can bypass triggers during schema migrations. Documented in runbook, not automated. Ops discipline required.

### Assumptions

- Spec 1 shipped: `controlled_documents`, `user_process_areas`, role capabilities, `governance_events`, tenant-scoped FKs, code immutability triggers.
- `users` table has `UNIQUE (tenant_id, id)` and `deactivated_at TIMESTAMPTZ` columns. Spec 2 migration adds if missing.
- `users` supports password re-auth (bcrypt hash accessible via iam service method).
- Existing `documents_v2.status` enum is the only state source (no parallel `workflow_status` column).
- Single-tenant-per-deploy MVP; multi-tenant enforcement via `tenant_id` columns, not load-tested.
- Scheduler timezone UTC internally; tenant-configured display TZ from existing settings.
- Only one cron worker instance per deploy (FOR UPDATE SKIP LOCKED makes multi-worker safe; no leader election needed).
- REST only; no GraphQL, no websockets. Inbox refresh = poll.
- PostgreSQL 16+ for `WITH INHERIT FALSE` per-grant syntax; 14+ base feature set acceptable with NOINHERIT at role level.
- Infrastructure provisions four DB roles: `metaldocs_app`, `metaldocs_migrator`, `metaldocs_readonly`, `metaldocs_security_owner`, `metaldocs_membership_writer`.
- CI/CD pipeline executes invariant checks (role attributes, SECURITY DEFINER metadata, schema privilege grants, unqualified-reference linter).

---

## Migration

Phased, reversible.

**Phase A — shadow deploy:**
- Add columns NULLABLE.
- Add new tables + triggers + functions + roles + privileges.
- Existing `documents_v2.status` values preserved (`draft`, `finalized`, `archived`).

**Phase B — backfill:**
- Idempotent job:
  ```
  FOR each document:
    revision_number = 1 if not set
    revision_version = 0
    If status='finalized' → set status='published', effective_from=updated_at,
                             governance_events insert 'legacy_published_no_signoffs'
                             with payload { reason: "pre-Spec-2 data, no approval history" }.
    If status='archived' → keep as-is (legacy archive state retained).
  ```
- Dual-write new transitions go through Spec 2 machinery.

**Phase C — enforce:**
- `NOT NULL` on `revision_number`, `revision_version`.
- `content_hash_at_submit` stays nullable (only populated at submit time).
- State transition trigger enforced (no more direct status UPDATE bypass).

**Rollback:**
- Each phase reversible via corresponding DOWN migration.
- Legacy `finalized` state preservable in a rollback by reversing `published`↔`finalized` mapping (recorded in governance_events).

---

## Change log

- **2026-04-21:** Initial draft through 10 Codex-validated iterations. Final verdict: accepted with documented residual risks appropriate for ISO 9001 SaaS market tier.
