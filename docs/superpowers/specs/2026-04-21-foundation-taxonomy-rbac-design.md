# Foundation Spec 1 — Taxonomy × RBAC × Controlled Document Registry

**Status:** Ready for implementation planning
**Date:** 2026-04-21
**Author:** Leandro Theodoro (MetalDocs) + Claude (Opus) co-designed with Codex validation
**Scope:** Bridge v1 taxonomy (DocumentProfile / ProcessArea / AccessPolicy / DocumentDepartment / DocumentFamily) to v2 editor modules (`documents_v2`, `templates_v2`). Foundation for Specs 2 (doc approval), 3 (placeholder fill), 4 (audit coverage).

> Examples in this document (`po`, `PO-03`, "Quality" area) are illustrative. All profiles, areas, codes are tenant-customizable data rows — nothing hardcoded.

---

## Goal

Unify MetalDocs on a single taxonomy foundation so that:

1. Every editable document is linked to a **controlled document registry entry** (QMS master list, ISO 9001 §7.5.3).
2. Every controlled document entry belongs to a **document profile** (doctype) with a mandatory default template version; entries may override with a profile-matched published template.
3. Every user has **role-per-area memberships** governing `document.*` / `template.*` / `workflow.*` capabilities, with AccessPolicy reserved for exception overrides.
4. Taxonomy codes (`profile_code`, `process_area_code`) are **immutable, non-reusable, tenant-scoped** — providing stable audit anchors for the life of the system.
5. `documents_v2` rows carry **write-once snapshots** of the codes and template version in effect at creation, so historical traceability survives later renames or archives.

## Architecture

- **Extend v1 taxonomy tables in-place.** Do not build a parallel v2 registry. v1's `document_profiles`, `process_areas`, `document_departments`, `document_families`, `subjects`, and `access_policies` are reused as authoritative.
- **Bridge v2 modules via FKs + snapshots.** `documents_v2` and `templates_v2` remain in their own modules but gain FK references and snapshot columns into the v1 taxonomy.
- **Introduce `controlled_documents`** as a new first-class entity (the master list). One entry (`PO-03`) maps to many `documents_v2` revisions over time.
- **Introduce `user_process_areas`** as the primary RBAC mechanism; keep `access_policies` as the exception layer.
- **Code immutability enforced at DB level** via triggers + composite tenant-scoped FKs.

## Components

### Backend (Go)

```
internal/modules/
  taxonomy/                              ← NEW
    domain/
      profile.go                         ← DocumentProfile + default_template_version_id
      area.go                            ← ProcessArea + parent + owner + default_approver_role
    application/
      profile_service.go                 ← CRUD (no code mutation), set_default_template, archive
      area_service.go                    ← CRUD, set_parent, set_owner, archive
      governance_logger.go               ← writes governance_events
    delivery/http/
      routes_profiles.go                 ← /api/v2/taxonomy/profiles
      routes_areas.go                    ← /api/v2/taxonomy/areas
      handler.go

  registry/                              ← NEW
    domain/
      controlled_document.go             ← entity + status machine (active | obsolete | superseded)
      sequence.go                        ← profile_sequence_counters with FOR UPDATE lock
      resolution.go                      ← template resolution: override_template_version_id
                                           ?? profile.default_template_version_id; fail loud
    application/
      service.go                         ← create (auto/manual code), update (non-code fields only),
                                           obsolete, supersede, list (filtered by user's areas)
      migration.go                       ← idempotent backfill of legacy documents_v2 rows
    delivery/http/
      routes.go                          ← /api/v2/controlled-documents
      handler.go

  iam/                                   ← MODIFY (extend existing)
    domain/
      user_area.go                       ← NEW: user_process_areas (effective_from/to, granted_by)
      role_capabilities.go               ← NEW: RoleCapabilities map + RoleCapabilitiesVersion int
    application/
      area_membership_service.go         ← NEW: grant/revoke with governance_events write.
                                           Spec 4 wires a MembershipHook here at app layer:
                                           Grant() → enqueue distribution_outbox for all
                                             published docs in the new area (scoped fan-out).
                                           Revoke() → synchronously revoke pending obligations.
                                           Hook runs after Spec 2's SECURITY DEFINER
                                           grant_area_membership() commits (not a DB trigger).
      authorization.go                   ← NEW: Check(user, capability, resourceCtx):
                                           (1) role caps via user_process_areas lookup
                                           (2) AccessPolicy allow/deny overrides
                                           (3) SoD guards
    delivery/http/
      middleware.go                      ← EXTEND: wire Check() into handler chain

  documents_v2/                          ← MODIFY
    domain/model.go                      ← + ControlledDocumentID, ProfileCodeSnapshot,
                                           ProcessAreaCodeSnapshot, TemplateVersionIDSnapshot
    application/service.go               ← on create: require controlled_document_id,
                                           resolve template via registry.Resolution,
                                           snapshot codes, enforce authz
```

### Frontend (React / TS)

```
frontend/apps/web/src/features/
  taxonomy/                              ← NEW
    TaxonomyAdminPage.tsx                ← "Tipos documentais" nav target
    ProfileList.tsx                      ← table with default template column
    ProfileEditDialog.tsx                ← name, alias, family, default_template picker
                                           (filtered to published versions of THIS profile),
                                           review interval, approval required, owner
    AreaList.tsx                         ← areas + parent + owner + default approver
    AreaEditDialog.tsx

  registry/                              ← NEW
    RegistryListPage.tsx                 ← master list; filter by profile/area/owner/status
    RegistryCreateDialog.tsx             ← profile picker → auto code preview;
                                           admin toggle for manual code + required reason field;
                                           override template toggle (admin-only)
    RegistryDetailPage.tsx               ← controlled doc view + revisions list + obsoletar action

  iam/
    AreaMembershipAdminPage.tsx          ← NEW: user × area × role matrix
    MembershipGrantDialog.tsx            ← role select + effective_from + reason (→ governance_events)

  documents/v2/
    DocumentCreatePage.tsx               ← MODIFY: pick controlled_document first,
                                           template auto-resolves,
                                           admin-only "override template" toggle
```

### Role → capability map (code constant, versioned)

```go
// internal/modules/iam/domain/role_capabilities.go
const RoleCapabilitiesVersion = 1  // incremented on any change; logged to governance_events on boot

var RoleCapabilities = map[string][]string{
    "viewer":   {"document.view", "template.view"},
    "editor":   {"document.view", "document.create", "document.edit", "template.view"},
    "reviewer": {"document.view", "document.edit", "workflow.review", "template.view"},
    "approver": {"document.view", "workflow.approve", "template.view", "template.publish"},
}
```

Any change to this map requires a `RoleCapabilitiesVersion` bump and emits `governance_events { event_type: "role.capability_map.version_bump" }` at process startup if the version differs from the last recorded row.

> **Spec 2 extension:** Spec 2 bumps `RoleCapabilitiesVersion` to 2 and adds `workflow.*` capabilities: `workflow.submit` (editor, reviewer), `workflow.review` (reviewer), `workflow.approve` / `workflow.publish` / `workflow.supersede` / `workflow.reject` (approver). Admin-only: `workflow.obsolete`, `workflow.route.edit`, `workflow.instance.cancel`. See Spec 2 for the full extended map.

## Data Model

### Extended v1 tables

```sql
-- document_profiles
ALTER TABLE document_profiles
  ADD COLUMN default_template_version_id UUID REFERENCES templates_v2.template_versions(id),
  ADD COLUMN owner_user_id UUID REFERENCES users(id),
  ADD COLUMN editable_by_role TEXT NOT NULL DEFAULT 'admin',
  ADD COLUMN archived_at TIMESTAMPTZ,
  ADD CONSTRAINT profile_code_format CHECK (code ~ '^[a-z][a-z0-9_-]{1,63}$');

-- tenant-scoped uniqueness INCLUDING archived rows (codes non-reusable)
CREATE UNIQUE INDEX ux_document_profiles_tenant_code
  ON document_profiles (tenant_id, code);

-- process_areas
ALTER TABLE process_areas
  ADD COLUMN parent_code TEXT REFERENCES process_areas(code),
  ADD COLUMN owner_user_id UUID REFERENCES users(id),
  ADD COLUMN default_approver_role TEXT,
  ADD COLUMN archived_at TIMESTAMPTZ,
  ADD CONSTRAINT area_code_format CHECK (code ~ '^[a-z][a-z0-9_-]{1,63}$');

CREATE UNIQUE INDEX ux_process_areas_tenant_code
  ON process_areas (tenant_id, code);
```

### New tables

```sql
-- Controlled document registry (QMS master list)
CREATE TABLE controlled_documents (
  id UUID PRIMARY KEY,
  tenant_id UUID NOT NULL,
  profile_code TEXT NOT NULL,
  process_area_code TEXT NOT NULL,
  department_code TEXT,
  code TEXT NOT NULL,                              -- e.g. "PO-03"
  sequence_num INT,                                -- NULL only when manual override used
  title TEXT NOT NULL,
  owner_user_id UUID NOT NULL REFERENCES users(id),
  override_template_version_id UUID REFERENCES templates_v2.template_versions(id),
  status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active','obsolete','superseded')),
  created_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL,

  -- Composite tenant-scoped FKs (DB enforces no cross-tenant references)
  FOREIGN KEY (tenant_id, profile_code)
    REFERENCES document_profiles (tenant_id, code),
  FOREIGN KEY (tenant_id, process_area_code)
    REFERENCES process_areas (tenant_id, code),
  UNIQUE (tenant_id, profile_code, code)
);

-- Per-profile sequence counter (concurrency-safe via SELECT FOR UPDATE)
CREATE TABLE profile_sequence_counters (
  tenant_id UUID NOT NULL,
  profile_code TEXT NOT NULL,
  next_seq INT NOT NULL,
  PRIMARY KEY (tenant_id, profile_code),
  FOREIGN KEY (tenant_id, profile_code)
    REFERENCES document_profiles (tenant_id, code)
);

-- User area membership (primary RBAC)
CREATE TABLE user_process_areas (
  user_id UUID NOT NULL REFERENCES users(id),
  tenant_id UUID NOT NULL,
  area_code TEXT NOT NULL,
  role TEXT NOT NULL CHECK (role IN ('viewer','editor','reviewer','approver')),
  effective_from TIMESTAMPTZ NOT NULL,
  effective_to TIMESTAMPTZ,                        -- NULL = currently active
  granted_by UUID REFERENCES users(id),
  PRIMARY KEY (user_id, area_code, effective_from),
  FOREIGN KEY (tenant_id, area_code)
    REFERENCES process_areas (tenant_id, code)
);

CREATE INDEX ix_user_process_areas_active
  ON user_process_areas (user_id, area_code)
  WHERE effective_to IS NULL;

-- Immutable governance audit log (complements existing audit)
-- All specs write to this single table. event_type is TEXT (no enum constraint) to allow
-- extension without schema migrations. Known event_types by spec:
--   Spec 1: numbering.override, template.override, role.grant, role.revoke,
--           profile.default_template_change, profile.rename_mapping,
--           role.capability_map.version_bump
--   Spec 2: workflow.submit, workflow.stage.complete, workflow.approved, workflow.reject,
--           workflow.publish, workflow.publish.scheduled, workflow.supersede,
--           workflow.obsolete, workflow.instance.cancel, legacy_published_no_signoffs
--   Spec 4: distribution.obligation_created, distribution.obligation_delivered,
--           distribution.obligation_acked, distribution.obligation_revoked,
--           distribution.reconciliation_gap
CREATE TABLE governance_events (
  id UUID PRIMARY KEY,
  tenant_id UUID NOT NULL,
  event_type TEXT NOT NULL,
  actor_user_id UUID NOT NULL,
  resource_type TEXT NOT NULL,
  resource_id TEXT NOT NULL,
  reason TEXT,                                     -- REQUIRED for override event_types
  payload_json JSONB NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

### Bridge columns on `documents_v2`

```sql
ALTER TABLE documents_v2.documents
  ADD COLUMN controlled_document_id UUID REFERENCES controlled_documents(id),
  ADD COLUMN profile_code_snapshot TEXT,           -- WRITE-ONCE at create time
  ADD COLUMN process_area_code_snapshot TEXT;

-- The existing `template_version_id` column on documents_v2.documents is retained as-is
-- and becomes the canonical "template_version_id_snapshot": its semantics change to
-- WRITE-ONCE (enforced at service layer + documented in repo). No rename required.
```

### DB-level immutability enforcement

```sql
-- Reject UPDATE of code column (belt + suspenders beyond repo layer)
CREATE OR REPLACE FUNCTION reject_code_update() RETURNS trigger AS $$
BEGIN
  IF NEW.code IS DISTINCT FROM OLD.code THEN
    RAISE EXCEPTION 'code column is immutable (table=%, old=%, new=%)',
      TG_TABLE_NAME, OLD.code, NEW.code
      USING ERRCODE = 'check_violation';
  END IF;
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_document_profiles_code_immutable
  BEFORE UPDATE ON document_profiles
  FOR EACH ROW EXECUTE FUNCTION reject_code_update();

CREATE TRIGGER trg_process_areas_code_immutable
  BEFORE UPDATE ON process_areas
  FOR EACH ROW EXECUTE FUNCTION reject_code_update();

CREATE TRIGGER trg_controlled_documents_code_immutable
  BEFORE UPDATE ON controlled_documents
  FOR EACH ROW EXECUTE FUNCTION reject_code_update();

-- Tenant consistency between documents_v2 and controlled_documents
CREATE OR REPLACE FUNCTION check_document_tenant_consistency() RETURNS trigger AS $$
DECLARE
  cd_tenant UUID;
BEGIN
  IF NEW.controlled_document_id IS NOT NULL THEN
    SELECT tenant_id INTO cd_tenant
      FROM controlled_documents WHERE id = NEW.controlled_document_id;
    IF cd_tenant IS DISTINCT FROM NEW.tenant_id THEN
      RAISE EXCEPTION 'tenant mismatch between document (%) and controlled_document (%)',
        NEW.tenant_id, cd_tenant;
    END IF;
  END IF;
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_documents_v2_tenant_consistency
  BEFORE INSERT OR UPDATE ON documents_v2.documents
  FOR EACH ROW EXECUTE FUNCTION check_document_tenant_consistency();
```

## Data Flow

### Create controlled document

```
Admin → POST /api/v2/controlled-documents
  body: {
    profile_code: "po",
    process_area_code: "quality",
    title: "Soldagem TIG",
    owner_user_id: "<uuid>",
    override_template_version_id?: "<uuid>",
    manual_code?: "PO-LEG-47",
    manual_code_reason?: "Legacy code migration from spreadsheet"
  }

registry.service.Create():
  1. iam.authz.Check(actor, "registry.create", { area_code: process_area_code })
     → requires editor+ role in area OR global admin capability.
  2. Validate profile + area not archived.
  3. If manual_code provided:
       - require manual_code_reason (non-empty, ≥10 chars).
       - check unique (tenant, profile, code).
       - write governance_events { event_type: "numbering.override", reason, payload }.
     Else:
       - BEGIN TX; SELECT ... FOR UPDATE on profile_sequence_counters (tenant, profile).
       - seq = counter.next_seq; counter.next_seq += 1.
       - code = fmt("%s-%02d", profile_code.upper(), seq).
  4. If override_template_version_id provided:
       - validate version.status = 'published' AND version.profile_code = this profile.
       - require reason field in payload → governance_events { event_type: "template.override" }.
  5. INSERT controlled_documents row; COMMIT.
  6. Return DTO.
```

### Create editable document from registry

```
User → GET /api/v2/controlled-documents?area=<accessible to user>
User picks controlled doc → POST /api/v2/documents
  body: { controlled_document_id, name? }

documents_v2.service.Create():
  1. Load controlled_doc. If status != 'active' → 409.
  2. authz.Check(actor, "document.create", { area_code: cd.process_area_code })
     → requires editor+ in area.
  3. Resolve template:
       resolved = cd.override_template_version_id
                  ?? profile.default_template_version_id (with profile loaded).
       If resolved IS NULL → 409 "profile_has_no_default_template".
       If resolved version.status != 'published' → 409 "template_unavailable".
  4. Snapshot:
       profile_code_snapshot = cd.profile_code
       process_area_code_snapshot = cd.process_area_code
       template_version_id_snapshot = resolved
  5. Render DOCX from template → S3.
  6. INSERT documents_v2 row with snapshots.
  7. audit.log("document.created", { controlled_document_id, template_version_id }).
```

### Authorization check (every v2 endpoint)

```
middleware.AuthzCheck(capability, resourceCtx):
  1. SELECT area_code, role FROM user_process_areas
     WHERE user_id = $1 AND tenant_id = $2
       AND effective_from <= now()
       AND (effective_to IS NULL OR effective_to > now())
  2. For each row: caps ∪= RoleCapabilities[role]
  3. Apply access_policies overrides scoped to resource (deny subtracts, allow adds).
  4. If capability ∈ caps → pass. Else → 403.
  5. SoD guard:
     - For `template.publish`: look up prior-stage actors in templates_v2 version history
       (author, reviewer) — active in Spec 1.
     - For `workflow.review` / `workflow.approve` on documents_v2: SoD is NOT implemented
       in Spec 1 middleware. Spec 2 implements SoD at the DB trigger level via
       `enforce_signoff_sod()`. In Spec 1 these capabilities are granted but not yet
       enforced at any workflow endpoint — enforcement begins when Spec 2 ships.
  6. Cache result for the remainder of the current HTTP request only.
```

### Migration (existing 25 documents_v2 rows)

Three-phase cutover, fully reversible:

**Phase A — shadow (deploy 1):**
- Run `ALTER TABLE documents_v2.documents ADD COLUMN ... NULLABLE`.
- Run idempotent backfill job:
  ```
  FOR each document WHERE controlled_document_id IS NULL:
    Try match legacy metadata (document_profile, process_area) to existing profiles/areas.
    If no match: synthesize profile "unassigned" (reserved code) + area "unassigned".
    Auto-create controlled_document with synthesized code ("MIG-<uuid-prefix>") or legacy code.
    UPDATE document SET controlled_document_id, profile_code_snapshot,
                        process_area_code_snapshot, template_version_id_snapshot.
  ```
- Re-runnable: first step filters `WHERE controlled_document_id IS NULL`, so repeat execution is a no-op.

**Phase B — dual-write (deploy 2):**
- New document creates REQUIRE controlled_document_id (validated at service layer).
- Monitor: count of rows with NULL new cols must hit 0 before Phase C.

**Phase C — enforce (deploy 3):**
- `ALTER TABLE ... SET NOT NULL` on all snapshot cols.
- Remove any NULL-tolerant fallback in code paths.

**Rollback:** Each phase has a matching DOWN migration (drop NOT NULL → stop dual-write → drop columns). Phase A backfill is forward-only but causes no data loss.

### Template resolution edge cases

| Scenario | Behavior |
|---|---|
| Profile has no default + doc has no override | **409** `profile_has_no_default_template` (UI: prompt admin to set default) |
| Override template deleted | **409** `override_template_deleted` |
| Default template deleted, override exists | Proceed with override |
| Default template obsoleted (status=obsolete) | **409** for new docs; existing docs unaffected (snapshot) |
| Override points to different profile's template | **422** `template_profile_mismatch` |
| Profile archived | New controlled_documents entries rejected (409); existing entries read-only |
| Area archived | Same as profile archived |

No silent fallbacks. Every ambiguous state returns an explicit error with `suggested_action` payload for the UI.

### Governed rename path

Codes are immutable. To "rename" a profile or area:

1. Create a new profile/area row with the new code.
2. Write `governance_events { event_type: "profile.rename_mapping", payload: { old_code, new_code, reason } }`.
3. Optionally archive the old row.
4. Existing documents keep their `profile_code_snapshot` / `process_area_code_snapshot` → audit preserved.
5. New documents use the new code.

No in-place `UPDATE document_profiles SET code = ...`. Blocked by DB trigger.

## Error Handling

### HTTP mapping

| Category | HTTP | Shape | Example |
|---|---|---|---|
| Validation | 400 | `{ code, field, message }` | missing `manual_code_reason` on override |
| Authn | 401 | `{ code: "unauthorized" }` | missing/expired token |
| Authz | 403 | `{ code, required_capability, area_code }` | user not in area |
| Not found | 404 | `{ code, resource }` | profile/area/controlled_doc not found |
| Conflict | 409 | `{ code, detail, suggested_action? }` | duplicate code, template unavailable, profile has no default |
| SoD violation | 422 | `{ code: "sod_violation", stage, actor, prior_stage_actor }` | self-approve attempt |
| Server | 500 | `{ code: "internal", trace_id }` | DB / S3 failure |

### Specific error conditions

**Create controlled_document**
- 409 `profile_has_no_default_template` — UI prompts admin to configure profile first.
- 422 `override_template_not_published`.
- 422 `template_profile_mismatch`.
- 409 `code_taken` / `code_in_use_archived` (code was used and tombstoned).
- 400 `reason_required_for_override`.

**Create document**
- 404 / 409 on missing or archived controlled_document.
- 409 `template_unavailable` with `suggested_action: { type: "admin_update_default", profile_code }`.
- 403 `missing_capability` with `required_capability: "document.create"` and `area_code`.
- 500 with `trace_id`; no DB row committed on S3 failure (transaction rollback).

**Area membership**
- Grant overlapping current membership → upsert-merge (extend `effective_to` or split row). No error.
- Revoke non-existent membership → 404.
- Grant with role not in `RoleCapabilities` → 400 `unknown_role`.

**Code update attempt (should never occur via API, only via direct SQL)**
- DB trigger raises `check_violation` → surfaces as 500 with log entry. Alerting wired.

### Concurrency

- `profile_sequence_counters` uses `SELECT ... FOR UPDATE` inside transaction. Serializes sequence allocation per `(tenant, profile)`.
- Crash between counter increment and `controlled_documents` insert → **sequence gap is acceptable**. ISO 9001 allows code gaps if counter bumps are auditable (they are, via governance_events on override-only paths; non-override auto-bumps are tracked by `profile_sequence_counters.next_seq` history).
- Optimistic concurrency (version column) on `document_profiles` UPDATE → prevents concurrent default-template changes clobbering each other.

### Migration failure modes

- Backfill is idempotent (filter `WHERE controlled_document_id IS NULL`).
- Partial failures retryable; Phase C NOT-NULL enforcement gated on 24h monitoring of zero NULL rows.

### Per-request cache

- Membership lookup cached for the lifetime of a single HTTP request (request-scoped context). No cross-request cache in Spec 1.

## Testing Approach

### Unit (Go)

```
taxonomy/
  profile_service_test.go       — set_default_template: happy, not-published, wrong-profile
  area_service_test.go          — parent cycle prevention, owner validation
registry/
  sequence_test.go              — FOR UPDATE lock, concurrent goroutines assert strict monotonic,
                                  gap-after-crash scenario
  resolution_test.go            — override > default, both-null 409, override inactive 409,
                                  default obsolete 409, profile archived 409
  controlled_document_service_test.go
                                — create auto, create manual + reason required, duplicate code,
                                  archived-code reuse rejected, template profile mismatch
iam/
  authorization_test.go         — role cap union, access_policy deny override,
                                  access_policy allow override, SoD guards
                                  (review=author, approve=author, approve=reviewer)
  area_membership_test.go       — grant new, grant overlap merge, revoke,
                                  temporal query (effective_from/to)
  role_capabilities_test.go     — version bump logged to governance_events
taxonomy/
  immutability_test.go          — UPDATE of code column returns DB error via trigger
```

### Integration (Go + Postgres)

```
registry_integration_test.go
  - Full flow: create profile → set default template → create controlled_doc
    → create documents_v2 row → assert all snapshot fields populated
  - Backfill: seed 25 legacy docs → run backfill → assert all linked;
    run again → no-op (idempotent)
  - Cross-profile override rejected
  - Rename flow: create new profile, archive old, assert old documents keep old snapshot
authz_integration_test.go
  - Editor in Quality: can create in Quality, cannot in Production
  - Reviewer cannot approve own review-submitted doc (SoD)
  - access_policies deny row overrides role grant
  - effective_to < now → capability revoked
tenant_isolation_test.go
  - Composite FK prevents cross-tenant controlled_documents → profile reference
  - Trigger blocks documents_v2 pointing at controlled_document from different tenant
```

### E2E (Playwright)

```
e2e/taxonomy.spec.ts              — admin creates profile, sets default template,
                                    appears in "Tipos documentais" nav
e2e/registry.spec.ts              — auto-code increments per profile;
                                    manual code + reason works; duplicate code shows error
e2e/area_membership.spec.ts       — admin grants user area/editor;
                                    user sees only assigned-area docs;
                                    revoke → 403 on create
e2e/document_from_registry.spec.ts
                                  — pick controlled doc → template auto-resolves →
                                    editable document created
e2e/sod.spec.ts                   — author submits for review → cannot review own
e2e/rename_via_new_code.spec.ts   — admin creates new profile, archives old;
                                    old documents render with historical code
```

### Test data

- No hardcoded `po` / `it` / `rg` in tests. Fixture helper `makeProfile(code, name)` generates unique codes per test.
- Golden path runs on empty tenant (enforces the "no required seeds" contract).
- Separate suite runs with `DefaultDocumentProfiles()` applied (backward compat check).

### Coverage targets

| Module | Target |
|---|---|
| `registry/*` | 85% |
| `iam/authorization.go` | 90% |
| `taxonomy/*` | 75% |
| `documents_v2` bridge code | 80% |

### Manual smoke checklist (pre-merge)

- [ ] Create profile via "Tipos documentais" → nav badge count updates.
- [ ] Set profile default template → picker shows only published versions of this profile.
- [ ] Create controlled doc (auto-code) → code reflects `{PROFILE.upper()}-{NN}`.
- [ ] Create controlled doc (manual code) → reason required, duplicate rejected.
- [ ] Assign user to area → user sees area-filtered registry list only.
- [ ] Attempt SoD violation (author reviews own) → blocked with 422.
- [ ] Migrate 25 existing docs → all linked, no broken list entries.
- [ ] Attempt direct SQL `UPDATE document_profiles SET code = ...` → DB trigger rejects.

### Perf sanity

- Authz middleware: < 5 ms overhead per request (single indexed query on `user_process_areas`).
- Benchmark: 100 concurrent creates of controlled docs under same profile → no sequence collisions, p99 < 50 ms.

## Cross-Spec Glossary (canonical terms)

Authoritative terminology for all four foundational specs. When in doubt, defer to these names.

| Concept | Canonical term | DB column / table | Notes |
|---|---|---|---|
| A single controlled document revision | **document revision** | `documents_v2.documents` row | "Version" is avoided — use "revision". One row = one revision. |
| The revision's unique counter | **revision_number** | `documents_v2.documents.revision_number` | OCC counter is `revision_version` (separate). |
| Effective date of a revision | **effective_from** | `documents_v2.documents.effective_from` | Spec 4 prose uses "effective_date" — the actual DB column is `effective_from`. |
| A doctype / document class | **document profile** | `document_profiles` table | Code is `profile_code` (TEXT, immutable). |
| An organizational scope unit | **process area** | `process_areas` table | Code is `process_area_code` (TEXT, immutable). Short form "area" is acceptable in prose but joins use `area_code`. |
| Area membership for a user | **user_process_areas** row | `user_process_areas` table | Revocation tracked via `effective_to` (not `revoked_at`). `revoked_by` added by Spec 2. |
| A duty to acknowledge a doc | **obligation** | `document_distributions` row | Not a synonym for "distribution" (the process). |
| The process of issuing obligations | **distribution** | `distribution_outbox` → `FanoutWorker` | |
| A user's proof of acknowledgment | **attestation** | `acked_at` + `ack_signature` | |
| The immutable audit trail table | **governance_events** | `governance_events` table (Spec 1) | All specs write here. No separate `audit_log` table exists. |
| DOCX content binding hash | **content_hash** | `approval_signoffs.content_hash` | DOCX-only hash. Spec 3 adds `schema_hash` + `values_hash` for triple-hash audit. |

---

## Out of Scope

**Deferred to later Foundation specs:**

| Item | Spec |
|---|---|
| Document approval state machine on `documents_v2` (submit → review → approve → publish) | Spec 2 |
| Approver inbox / reviewer queue UI | Spec 2 |
| Transition notifications (email / in-app) | Spec 2 |
| Placeholder fill-in form (schema-driven UI at doc creation) | Spec 3 |
| Eigenpal variable fanout wiring | Spec 3 |
| Full audit log coverage (session, autosave, reads, every transition) | Spec 4 |
| Audit viewer UI | Spec 4 |

**Explicit non-goals:**

- **v1 `documents` module deprecation** — v1 stays running untouched.
- **DB-level custom role definition** — role→capability map stays in code + version-bumped.
- **Cross-tenant sharing** — every table scoped by `tenant_id`; multi-tenant federation not addressed.
- **Deep area hierarchy** — only single-level `parent_code`. Arbitrary trees rejected.
- **Per-profile custom code format** — fixed `{CODE}-{NN}` format.
- **Profile import/export between tenants** — no migration tooling in this spec.
- **Scheduled obsoletion by `ReviewIntervalDays`** — field stored, cron not implemented.
- **Retention auto-archive (`RetentionDays`)** — stored, not enforced. When implemented in a future spec, it must coordinate with Spec 4's `document_distributions` table to mark obligations as `revoked_at` with `revoke_reason='doc_archived'` before archiving.
- **Validity period enforcement (`ValidityDays`)** — stored, not enforced.
- **Cross-request membership cache** — request-scoped cache only. Revisit if load test flags.
- **AccessPolicy engine rewrite** — kept as-is, only consumed by new `authz.Check` function.

**Assumptions:**

- Single tenant per deploy for MVP; multi-tenant enforcement correct but not stress-tested.
- All writes go through REST; no background jobs in Spec 1 except the migration backfill.
- `users` table + JWT auth already works (not modified).
- `templates_v2` publish flow already works (Spec 1 only consumes published versions).
