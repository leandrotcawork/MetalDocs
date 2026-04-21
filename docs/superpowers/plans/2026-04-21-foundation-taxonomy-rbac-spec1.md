# Foundation Spec 1 — Taxonomy × RBAC × Controlled Document Registry

**Date:** 2026-04-21
**Spec:** [docs/superpowers/specs/2026-04-21-foundation-taxonomy-rbac-design.md](../specs/2026-04-21-foundation-taxonomy-rbac-design.md)
**Status:** Ready to execute

## Model Assignments

| Model | When to use |
|---|---|
| **Codex gpt-5.3-codex (medium)** | All Go domain models, services, repositories, authorization logic, SQL migrations, integration tests, E2E tests |
| **Sonnet** | Go HTTP handlers (CRUD boilerplate), React components (medium complexity), API hooks, React pages |
| **Haiku** | GRANT SQL files, `index.ts` exports, route registration in `module.go`, trivial type additions |
| **Opus** | Phase-end reviews only (after Phase 2, 4, 6, 8) |

## Architecture Notes (Actual Table Names)

The spec uses simplified names. Actual tables differ:

| Spec name | Actual table | Schema |
|---|---|---|
| `document_profiles` | `metaldocs.document_profiles` | metaldocs |
| `process_areas` | `metaldocs.document_process_areas` | metaldocs |
| `document_families` | `metaldocs.document_families` | metaldocs |
| `template_versions` | `templates_v2_template_version` | public |
| `documents_v2.documents` | `documents_v2` | public |

**v1 tables lack `tenant_id`** — migrations 0122–0123 add it with `DEFAULT '<sentinel-uuid>'` for single-tenant MVP. New tables (`controlled_documents`, `user_process_areas`, `governance_events`, `profile_sequence_counters`) use public schema following the `documents_v2` pattern.

**Migration mechanism:** plain SQL files in `migrations/` loaded via `docker-entrypoint-initdb.d`. No binary tool. Verify by restarting DB container.

**Three-phase bridge cutover (spec §Migration):**
- Phase A (migration 0126): nullable columns — deploy first
- Phase B: service enforces `controlled_document_id` required on create; backfill job runs once at startup
- Phase C (migration 0129): `ALTER TABLE documents_v2 ALTER COLUMN controlled_document_id SET NOT NULL` — gated on 24h monitoring showing 0 NULL rows

**Tenant propagation rule:** Every repository interface method that touches taxonomy/registry/IAM tables MUST accept `tenantID string` as first argument after `ctx`. HTTP handlers extract tenant from JWT claim and inject. Missing tenant argument is a compile-time error by design.

**SoD policy matrix (Spec 1 scope):**

| Capability | Actor constraint | Enforcing component |
|---|---|---|
| `template.publish` | Actor must NOT be author or reviewer of that version | `AuthorizationService.Check` + `TemplateAuthorChecker` |
| `workflow.review` | NOT enforced in Spec 1 (Spec 2 DB trigger) | — |
| `workflow.approve` | NOT enforced in Spec 1 (Spec 2 DB trigger) | — |

Each SoD rule maps to exactly one test in `authorization_test.go`.

**RoleCapabilitiesVersion startup behavior:**
- Non-dev (`APP_ENV != "development"`): hard-fail if DB write of governance_event fails (wraps in startup error)
- Dev: log warning, continue
- Re-run same version: no-op (idempotent query: `SELECT COUNT(*) WHERE event_type=... AND payload->>'version'=$1`)

---

## File Structure Map

### New files (Go)
```
internal/modules/taxonomy/
  domain/
    profile.go
    profile_test.go
    area.go
    area_test.go
  application/
    profile_service.go
    profile_service_test.go
    area_service.go
    area_service_test.go
    governance_logger.go
    immutability_test.go
  infrastructure/
    repository.go
  delivery/http/
    handler.go
    routes_profiles.go
    routes_areas.go
  module.go

internal/modules/registry/
  domain/
    controlled_document.go
    sequence.go
    sequence_test.go
    resolution.go
    resolution_test.go
  application/
    service.go
    service_test.go
    migration.go
  infrastructure/
    repository.go
  delivery/http/
    handler.go
    routes.go
  module.go
```

### Modified files (Go)
```
internal/modules/iam/
  domain/
    model.go                          ← ADD Capability type + constants
    user_area.go                      ← NEW: UserProcessArea struct
    role_capabilities.go              ← NEW: RoleCapabilities map v1
    role_capabilities_test.go         ← NEW
  application/
    authorization.go                  ← NEW: Check(user, capability, resourceCtx)
    authorization_test.go             ← NEW
    area_membership_service.go        ← NEW: grant/revoke
    area_membership_test.go           ← NEW
  infrastructure/
    postgres/                         ← NEW: user_area_repository.go
  delivery/http/
    middleware.go                     ← EXTEND: wire Check()
    routes_memberships.go             ← NEW

internal/modules/documents_v2/
  domain/model.go                     ← ADD snapshot fields
  application/service.go              ← EXTEND: require CD, resolve template, snapshot
  application/service_cd_test.go      ← NEW

apps/api/cmd/metaldocs-api/main.go   ← EXTEND: wire taxonomy + registry modules
internal/api/v2/types_gen.go         ← NEW: canonical request/response contract structs
internal/api/v2/contract_test.go     ← NEW: validates error shape per endpoint
```

### New files (Frontend)
```
frontend/apps/web/src/features/taxonomy/
  types.ts
  api.ts
  ProfileList.tsx
  ProfileEditDialog.tsx
  AreaList.tsx
  AreaEditDialog.tsx
  TaxonomyAdminPage.tsx
  index.ts

frontend/apps/web/src/features/registry/
  types.ts                            ← NEW (replaces/extends useRegistryExplorer)
  api.ts                              ← NEW
  RegistryListPage.tsx                ← REPLACES RegistryExplorerView.tsx
  RegistryCreateDialog.tsx            ← NEW
  RegistryDetailPage.tsx              ← NEW
  index.ts

frontend/apps/web/src/features/iam/
  types.ts                            ← EXTEND
  api.ts                              ← NEW (area memberships)
  AreaMembershipAdminPage.tsx         ← NEW
  MembershipGrantDialog.tsx           ← NEW

frontend/apps/web/e2e/
  taxonomy.spec.ts
  registry.spec.ts
  area-membership.spec.ts
  document-from-registry.spec.ts
  sod.spec.ts
  rename-via-new-code.spec.ts
```

### New migration files
```
migrations/
  0122_taxonomy_extend_document_profiles.sql
  0123_taxonomy_extend_process_areas.sql
  0124_registry_controlled_documents.sql
  0125_registry_iam_user_process_areas_governance_events.sql
  0126_documents_v2_bridge_columns.sql
  0127_documents_v2_tenant_consistency_trigger.sql
  0128_grants_new_tables.sql
  0129_documents_v2_bridge_not_null.sql   ← Phase C: run after backfill verified complete
```

---

## Phase 1: Database Migrations

> **Model:** Codex (0122–0127) · Haiku (0128)

### Task 1.1 — Migration 0122: extend `metaldocs.document_profiles`

**File:** `migrations/0122_taxonomy_extend_document_profiles.sql`

**Action:** Add `tenant_id`, `default_template_version_id`, `owner_user_id`, `editable_by_role`, `archived_at` to `metaldocs.document_profiles`. Create tenant-scoped unique index. Add `CHECK` constraint on code format. Add immutability trigger.

```sql
-- 0122_taxonomy_extend_document_profiles.sql

-- Step 1: add tenant_id with sentinel default (single-tenant MVP)
ALTER TABLE metaldocs.document_profiles
  ADD COLUMN IF NOT EXISTS tenant_id UUID NOT NULL DEFAULT 'ffffffff-ffff-ffff-ffff-ffffffffffff';

-- Step 2: new governance columns
ALTER TABLE metaldocs.document_profiles
  ADD COLUMN IF NOT EXISTS default_template_version_id UUID
    REFERENCES templates_v2_template_version(id),
  ADD COLUMN IF NOT EXISTS owner_user_id UUID,
  ADD COLUMN IF NOT EXISTS editable_by_role TEXT NOT NULL DEFAULT 'admin',
  ADD COLUMN IF NOT EXISTS archived_at TIMESTAMPTZ;

-- Step 3: code format constraint (skip rows violating format — they predate this constraint)
-- NOTE: existing seeds may have simple codes like 'po','it' — check regex allows 2-char: '^[a-z][a-z0-9_-]{1,63}$'
-- The regex requires min length 2; existing codes like 'po' (len 2) satisfy it.
ALTER TABLE metaldocs.document_profiles
  ADD CONSTRAINT IF NOT EXISTS profile_code_format
    CHECK (code ~ '^[a-z][a-z0-9_-]{1,63}$');

-- Step 4: tenant-scoped unique index (includes archived rows — codes non-reusable)
CREATE UNIQUE INDEX IF NOT EXISTS ux_document_profiles_tenant_code
  ON metaldocs.document_profiles (tenant_id, code);

-- Step 5: immutability trigger
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

DROP TRIGGER IF EXISTS trg_document_profiles_code_immutable ON metaldocs.document_profiles;
CREATE TRIGGER trg_document_profiles_code_immutable
  BEFORE UPDATE ON metaldocs.document_profiles
  FOR EACH ROW EXECUTE FUNCTION reject_code_update();
```

**Verify:** `make down && make up` completes. Then:
```
docker compose -f deploy/compose/docker-compose.yml exec postgres \
  psql -U metaldocs -d metaldocs -c "\d metaldocs.document_profiles"
```
Expected: `tenant_id`, `default_template_version_id`, `archived_at` columns present.

---

### Task 1.2 — Migration 0123: extend `metaldocs.document_process_areas`

**File:** `migrations/0123_taxonomy_extend_process_areas.sql`

**Action:** Add `tenant_id`, `parent_code`, `owner_user_id`, `default_approver_role`, `archived_at`. Create tenant-scoped unique index. Add immutability trigger (reuse function from 0122).

```sql
-- 0123_taxonomy_extend_process_areas.sql

ALTER TABLE metaldocs.document_process_areas
  ADD COLUMN IF NOT EXISTS tenant_id UUID NOT NULL DEFAULT 'ffffffff-ffff-ffff-ffff-ffffffffffff',
  ADD COLUMN IF NOT EXISTS parent_code TEXT
    REFERENCES metaldocs.document_process_areas(code),
  ADD COLUMN IF NOT EXISTS owner_user_id UUID,
  ADD COLUMN IF NOT EXISTS default_approver_role TEXT,
  ADD COLUMN IF NOT EXISTS archived_at TIMESTAMPTZ;

ALTER TABLE metaldocs.document_process_areas
  ADD CONSTRAINT IF NOT EXISTS area_code_format
    CHECK (code ~ '^[a-z][a-z0-9_-]{1,63}$');

CREATE UNIQUE INDEX IF NOT EXISTS ux_process_areas_tenant_code
  ON metaldocs.document_process_areas (tenant_id, code);

-- reject_code_update() already created in 0122
DROP TRIGGER IF EXISTS trg_process_areas_code_immutable ON metaldocs.document_process_areas;
CREATE TRIGGER trg_process_areas_code_immutable
  BEFORE UPDATE ON metaldocs.document_process_areas
  FOR EACH ROW EXECUTE FUNCTION reject_code_update();
```

**Verify:** Same restart pattern. `\d metaldocs.document_process_areas` shows new columns.

---

### Task 1.3 — Migration 0124: create `controlled_documents` + `profile_sequence_counters`

**File:** `migrations/0124_registry_controlled_documents.sql`

```sql
-- 0124_registry_controlled_documents.sql

CREATE TABLE IF NOT EXISTS controlled_documents (
  id                              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id                       UUID NOT NULL,
  profile_code                    TEXT NOT NULL,
  process_area_code               TEXT NOT NULL,
  department_code                 TEXT,
  code                            TEXT NOT NULL,
  sequence_num                    INT,
  title                           TEXT NOT NULL,
  owner_user_id                   UUID NOT NULL,
  override_template_version_id    UUID REFERENCES templates_v2_template_version(id),
  status                          TEXT NOT NULL DEFAULT 'active'
                                    CHECK (status IN ('active','obsolete','superseded')),
  created_at                      TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at                      TIMESTAMPTZ NOT NULL DEFAULT now(),

  FOREIGN KEY (tenant_id, profile_code)
    REFERENCES metaldocs.document_profiles (tenant_id, code),
  FOREIGN KEY (tenant_id, process_area_code)
    REFERENCES metaldocs.document_process_areas (tenant_id, code),

  UNIQUE (tenant_id, profile_code, code)
);

CREATE TABLE IF NOT EXISTS profile_sequence_counters (
  tenant_id     UUID NOT NULL,
  profile_code  TEXT NOT NULL,
  next_seq      INT NOT NULL DEFAULT 1,
  PRIMARY KEY (tenant_id, profile_code),
  FOREIGN KEY (tenant_id, profile_code)
    REFERENCES metaldocs.document_profiles (tenant_id, code)
);

CREATE INDEX IF NOT EXISTS ix_controlled_documents_tenant_area
  ON controlled_documents (tenant_id, process_area_code);

CREATE INDEX IF NOT EXISTS ix_controlled_documents_tenant_profile
  ON controlled_documents (tenant_id, profile_code);

-- code column immutability trigger (reuse function from 0122)
DROP TRIGGER IF EXISTS trg_controlled_documents_code_immutable ON controlled_documents;
CREATE TRIGGER trg_controlled_documents_code_immutable
  BEFORE UPDATE ON controlled_documents
  FOR EACH ROW EXECUTE FUNCTION reject_code_update();
```

**Verify:** `\d controlled_documents` + `\d profile_sequence_counters` show tables with correct columns and constraints.

---

### Task 1.4 — Migration 0125: create `user_process_areas` + `governance_events`

**File:** `migrations/0125_registry_iam_user_process_areas_governance_events.sql`

```sql
-- 0125_registry_iam_user_process_areas_governance_events.sql

CREATE TABLE IF NOT EXISTS user_process_areas (
  user_id         TEXT NOT NULL,
  tenant_id       UUID NOT NULL,
  area_code       TEXT NOT NULL,
  role            TEXT NOT NULL CHECK (role IN ('viewer','editor','reviewer','approver')),
  effective_from  TIMESTAMPTZ NOT NULL,
  effective_to    TIMESTAMPTZ,
  granted_by      TEXT,
  PRIMARY KEY (user_id, area_code, effective_from),
  FOREIGN KEY (tenant_id, area_code)
    REFERENCES metaldocs.document_process_areas (tenant_id, code)
);

CREATE INDEX IF NOT EXISTS ix_user_process_areas_active
  ON user_process_areas (user_id, area_code)
  WHERE effective_to IS NULL;

CREATE TABLE IF NOT EXISTS governance_events (
  id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id      UUID NOT NULL,
  event_type     TEXT NOT NULL,
  actor_user_id  TEXT NOT NULL,
  resource_type  TEXT NOT NULL,
  resource_id    TEXT NOT NULL,
  reason         TEXT,
  payload_json   JSONB NOT NULL,
  created_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS ix_governance_events_tenant_type
  ON governance_events (tenant_id, event_type, created_at DESC);

CREATE INDEX IF NOT EXISTS ix_governance_events_resource
  ON governance_events (resource_type, resource_id);
```

**Note on `user_id` type:** v2 tables use `TEXT` for user IDs (see `documents_v2` migration 0117: `user_ids_to_text`). Match that convention.

**Verify:** `\d user_process_areas` + `\d governance_events` show correct structure.

---

### Task 1.5 — Migration 0126: bridge columns on `documents_v2`

**File:** `migrations/0126_documents_v2_bridge_columns.sql`

```sql
-- 0126_documents_v2_bridge_columns.sql

-- Phase A (shadow): add as NULLABLE — Phase C (enforce) adds NOT NULL after backfill
ALTER TABLE documents_v2
  ADD COLUMN IF NOT EXISTS controlled_document_id UUID
    REFERENCES controlled_documents(id),
  ADD COLUMN IF NOT EXISTS profile_code_snapshot TEXT,
  ADD COLUMN IF NOT EXISTS process_area_code_snapshot TEXT;

-- existing template_version_id (templates_v2_template_version_id) is retained as-is;
-- service layer treats it as write-once after this migration.

CREATE INDEX IF NOT EXISTS ix_documents_v2_controlled_doc
  ON documents_v2 (controlled_document_id)
  WHERE controlled_document_id IS NOT NULL;
```

**Verify:** `\d documents_v2` shows three new nullable columns.

---

### Task 1.6 — Migration 0127: tenant consistency trigger on `documents_v2`

**File:** `migrations/0127_documents_v2_tenant_consistency_trigger.sql`

```sql
-- 0127_documents_v2_tenant_consistency_trigger.sql

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

DROP TRIGGER IF EXISTS trg_documents_v2_tenant_consistency ON documents_v2;
CREATE TRIGGER trg_documents_v2_tenant_consistency
  BEFORE INSERT OR UPDATE ON documents_v2
  FOR EACH ROW EXECUTE FUNCTION check_document_tenant_consistency();
```

**Verify:** `make down && make up` completes without errors.

---

### Task 1.8 — Migration 0129: Phase C — enforce NOT NULL on bridge columns

**File:** `migrations/0129_documents_v2_bridge_not_null.sql`
**Model: Codex**

Gate: run ONLY after backfill verifies 0 NULL rows (see Task 9.5).

```sql
-- 0129_documents_v2_bridge_not_null.sql
-- Phase C: enforce NOT NULL after 24h monitoring confirms 0 NULL rows.
-- Preflight guard: fail if any NULLs remain.

DO $$
DECLARE null_count INT;
BEGIN
  SELECT COUNT(*) INTO null_count FROM documents_v2 WHERE controlled_document_id IS NULL;
  IF null_count > 0 THEN
    RAISE EXCEPTION 'Phase C blocked: % documents_v2 rows still have NULL controlled_document_id',
      null_count;
  END IF;
END $$;

ALTER TABLE documents_v2
  ALTER COLUMN controlled_document_id SET NOT NULL,
  ALTER COLUMN profile_code_snapshot SET NOT NULL,
  ALTER COLUMN process_area_code_snapshot SET NOT NULL;
```

**Verify:** Migration runs successfully only after backfill is complete. If run before: exception with NULL count.

---

### Task 1.7 — Migration 0128: GRANT permissions on new tables

**File:** `migrations/0128_grants_new_tables.sql`

**Model: Haiku** — follow exact same pattern as `migrations/0010_grant_document_types_privileges.sql`.

Open `migrations/0010_grant_document_types_privileges.sql`, copy the GRANT pattern, replace table names with:
- `controlled_documents`
- `profile_sequence_counters`
- `user_process_areas`
- `governance_events`

**Verify:** `make down && make up` — no permission errors in API logs on startup.

---

## Phase 2: Taxonomy Module (Go) — Domain + Service

> **Model:** Codex · TDD enforced: write test → run (fail) → implement → run (pass)
> **Phase-end review: Opus**

### Task 2.1 — Write domain: `DocumentProfile`

**File:** `internal/modules/taxonomy/domain/profile.go`

```go
package domain

import (
    "errors"
    "time"
)

type DocumentProfile struct {
    Code                       string
    TenantID                   string
    FamilyCode                 string
    Name                       string
    Description                string
    ReviewIntervalDays         int
    DefaultTemplateVersionID   *string   // nil = no default set
    OwnerUserID                *string
    EditableByRole             string
    ArchivedAt                 *time.Time
    CreatedAt                  time.Time
}

var (
    ErrProfileNotFound          = errors.New("profile not found")
    ErrProfileCodeImmutable     = errors.New("profile code is immutable")
    ErrProfileArchived          = errors.New("profile is archived")
    ErrTemplateNotPublished     = errors.New("template version is not published")
    ErrTemplateProfileMismatch  = errors.New("template version belongs to different profile")
)

func (p *DocumentProfile) IsActive() bool { return p.ArchivedAt == nil }

func (p *DocumentProfile) Archive(now time.Time) error {
    if !p.IsActive() {
        return ErrProfileArchived
    }
    p.ArchivedAt = &now
    return nil
}
```

**Verify:** `go vet ./internal/modules/taxonomy/domain/...` → exit 0

---

### Task 2.2 — Write domain test: `profile_test.go`

**File:** `internal/modules/taxonomy/domain/profile_test.go`

Write tests for:
- `IsActive()` returns true when `ArchivedAt == nil`
- `Archive()` sets `ArchivedAt` to provided time
- `Archive()` on already-archived profile returns `ErrProfileArchived`

**Verify:** `go test ./internal/modules/taxonomy/domain/... -run TestDocumentProfile -v` → `PASS`

---

### Task 2.3 — Write domain: `ProcessArea`

**File:** `internal/modules/taxonomy/domain/area.go`

```go
package domain

import (
    "errors"
    "time"
)

type ProcessArea struct {
    Code                 string
    TenantID             string
    Name                 string
    Description          string
    ParentCode           *string
    OwnerUserID          *string
    DefaultApproverRole  *string
    ArchivedAt           *time.Time
    CreatedAt            time.Time
}

var (
    ErrAreaNotFound      = errors.New("process area not found")
    ErrAreaArchived      = errors.New("process area is archived")
    ErrAreaParentCycle   = errors.New("area parent assignment creates cycle")
    ErrAreaCodeImmutable = errors.New("area code is immutable")
)

func (a *ProcessArea) IsActive() bool { return a.ArchivedAt == nil }

func (a *ProcessArea) Archive(now time.Time) error {
    if !a.IsActive() {
        return ErrAreaArchived
    }
    a.ArchivedAt = &now
    return nil
}
```

**Verify:** `go vet ./internal/modules/taxonomy/domain/...` → exit 0

---

### Task 2.4 — Write domain test: `area_test.go`

**File:** `internal/modules/taxonomy/domain/area_test.go`

Tests for `IsActive()`, `Archive()`, `Archive()` on archived.

**Verify:** `go test ./internal/modules/taxonomy/domain/... -v` → `PASS`

---

### Task 2.5 — Write taxonomy repository port

**File:** `internal/modules/taxonomy/domain/port.go`

```go
package domain

import "context"

type ProfileRepository interface {
    GetByCode(ctx context.Context, tenantID, code string) (*DocumentProfile, error)
    List(ctx context.Context, tenantID string, includeArchived bool) ([]DocumentProfile, error)
    Create(ctx context.Context, p *DocumentProfile) error
    Update(ctx context.Context, p *DocumentProfile) error   // code field ignored (immutable)
}

type AreaRepository interface {
    GetByCode(ctx context.Context, tenantID, code string) (*ProcessArea, error)
    List(ctx context.Context, tenantID string, includeArchived bool) ([]ProcessArea, error)
    Create(ctx context.Context, a *ProcessArea) error
    Update(ctx context.Context, a *ProcessArea) error
    // ListAncestors returns codes reachable by following ParentCode upward
    ListAncestors(ctx context.Context, tenantID, code string) ([]string, error)
}

type GovernanceLogger interface {
    Log(ctx context.Context, e GovernanceEvent) error
}

type GovernanceEvent struct {
    TenantID     string
    EventType    string
    ActorUserID  string
    ResourceType string
    ResourceID   string
    Reason       string
    PayloadJSON  []byte
}
```

**Verify:** `go build ./internal/modules/taxonomy/...` → exit 0

---

### Task 2.6 — Write taxonomy infrastructure: `repository.go`

**File:** `internal/modules/taxonomy/infrastructure/repository.go`

Implement `ProfileRepository` and `AreaRepository` against `metaldocs.document_profiles` and `metaldocs.document_process_areas` tables. Key queries:

- `GetByCode`: `SELECT ... FROM metaldocs.document_profiles WHERE tenant_id=$1 AND code=$2`
- `Update`: excludes `code` column from SET list
- `ListAncestors` for `AreaRepository`: recursive CTE `WITH RECURSIVE ancestors AS (SELECT code, parent_code FROM metaldocs.document_process_areas WHERE tenant_id=$1 AND code=$2 UNION ALL ...)`

**Verify:** `go build ./internal/modules/taxonomy/...` → exit 0

---

### Task 2.7 — Write `profile_service.go` (test-first)

**File (test first):** `internal/modules/taxonomy/application/profile_service_test.go`

Write stub service interface, then tests:
- `SetDefaultTemplate`: happy path (template published + same profile) → `DefaultTemplateVersionID` set
- `SetDefaultTemplate`: template not published → `ErrTemplateNotPublished`
- `SetDefaultTemplate`: template belongs to different profile → `ErrTemplateProfileMismatch`
- `Archive`: sets `ArchivedAt`
- `Archive`: already archived → `ErrProfileArchived`

Use in-memory fakes for `ProfileRepository`, `GovernanceLogger`, and a `TemplateVersionChecker` port.

**Verify:** `go test ./internal/modules/taxonomy/application/... -run TestProfileService -v` → FAIL (no implementation yet)

**File:** `internal/modules/taxonomy/application/profile_service.go`

```go
package application

import (
    "context"
    "time"

    "metaldocs/internal/modules/taxonomy/domain"
)

type TemplateVersionChecker interface {
    // IsPublished returns (isPublished, profileCode, error)
    IsPublished(ctx context.Context, versionID string) (bool, string, error)
}

type ProfileService struct {
    profiles  domain.ProfileRepository
    tplCheck  TemplateVersionChecker
    govLogger domain.GovernanceLogger
    now       func() time.Time
}

func NewProfileService(
    profiles domain.ProfileRepository,
    tplCheck TemplateVersionChecker,
    govLogger domain.GovernanceLogger,
) *ProfileService {
    return &ProfileService{profiles: profiles, tplCheck: tplCheck, govLogger: govLogger, now: time.Now}
}

func (s *ProfileService) SetDefaultTemplate(ctx context.Context, tenantID, profileCode, templateVersionID, actorID string) error {
    p, err := s.profiles.GetByCode(ctx, tenantID, profileCode)
    if err != nil {
        return err
    }
    if !p.IsActive() {
        return domain.ErrProfileArchived
    }
    published, tplProfileCode, err := s.tplCheck.IsPublished(ctx, templateVersionID)
    if err != nil {
        return err
    }
    if !published {
        return domain.ErrTemplateNotPublished
    }
    if tplProfileCode != profileCode {
        return domain.ErrTemplateProfileMismatch
    }
    p.DefaultTemplateVersionID = &templateVersionID
    if err := s.profiles.Update(ctx, p); err != nil {
        return err
    }
    return s.govLogger.Log(ctx, domain.GovernanceEvent{
        TenantID:     tenantID,
        EventType:    "profile.default_template_change",
        ActorUserID:  actorID,
        ResourceType: "document_profile",
        ResourceID:   profileCode,
        PayloadJSON:  []byte(`{"template_version_id":"` + templateVersionID + `"}`),
    })
}

func (s *ProfileService) Archive(ctx context.Context, tenantID, profileCode, actorID string) error {
    p, err := s.profiles.GetByCode(ctx, tenantID, profileCode)
    if err != nil {
        return err
    }
    if err := p.Archive(s.now()); err != nil {
        return err
    }
    return s.profiles.Update(ctx, p)
}
```

**Verify:** `go test ./internal/modules/taxonomy/application/... -run TestProfileService -v` → `PASS`

---

### Task 2.8 — Write `area_service.go` (test-first)

**File (test first):** `internal/modules/taxonomy/application/area_service_test.go`

Tests:
- `SetParent`: valid parent → saved
- `SetParent`: parent is descendant of area → `ErrAreaParentCycle`
- `Archive`: sets `ArchivedAt`
- `Archive`: already archived → `ErrAreaArchived`

**File:** `internal/modules/taxonomy/application/area_service.go`

Implement `AreaService` with `SetParent` (cycle check via `ListAncestors`), `Archive`.

**Verify:** `go test ./internal/modules/taxonomy/application/... -run TestAreaService -v` → `PASS`

---

### Task 2.9 — Write `governance_logger.go` (DB implementation)

**File:** `internal/modules/taxonomy/application/governance_logger.go`

Implement `GovernanceLogger` backed by a `*sql.DB` — inserts into `governance_events`.

```go
package application

import (
    "context"
    "database/sql"

    "metaldocs/internal/modules/taxonomy/domain"
)

type DBGovernanceLogger struct{ db *sql.DB }

func NewDBGovernanceLogger(db *sql.DB) *DBGovernanceLogger { return &DBGovernanceLogger{db: db} }

func (l *DBGovernanceLogger) Log(ctx context.Context, e domain.GovernanceEvent) error {
    _, err := l.db.ExecContext(ctx,
        `INSERT INTO governance_events
           (tenant_id, event_type, actor_user_id, resource_type, resource_id, reason, payload_json)
         VALUES ($1,$2,$3,$4,$5,$6,$7)`,
        e.TenantID, e.EventType, e.ActorUserID, e.ResourceType, e.ResourceID,
        nullString(e.Reason), e.PayloadJSON,
    )
    return err
}

func nullString(s string) sql.NullString {
    return sql.NullString{String: s, Valid: s != ""}
}
```

**Verify:** `go build ./internal/modules/taxonomy/...` → exit 0

---

### Task 2.10 — Write immutability trigger test

**File:** `internal/modules/taxonomy/application/immutability_test.go`

Integration test (build tag `//go:build integration`):
- Connect to test DB
- `INSERT INTO metaldocs.document_profiles (code, tenant_id, ...)` one row
- `UPDATE metaldocs.document_profiles SET code = 'new_code' WHERE code = 'old_code'`
- Assert error contains `"check_violation"` or `"code column is immutable"`

**Verify:** `go test ./internal/modules/taxonomy/... -tags=integration -run TestCodeImmutability -v` → `PASS`

---

### Task 2.11 — Write taxonomy `module.go`

**File:** `internal/modules/taxonomy/module.go`

```go
package taxonomy

import (
    "database/sql"
    "net/http"

    "metaldocs/internal/modules/taxonomy/application"
    thttp "metaldocs/internal/modules/taxonomy/delivery/http"
    "metaldocs/internal/modules/taxonomy/infrastructure"
)

type Module struct {
    Handler *thttp.Handler
}

type Dependencies struct {
    DB          *sql.DB
    TplChecker  application.TemplateVersionChecker
}

func New(deps Dependencies) *Module {
    profileRepo := infrastructure.NewProfileRepository(deps.DB)
    areaRepo := infrastructure.NewAreaRepository(deps.DB)
    govLogger := application.NewDBGovernanceLogger(deps.DB)

    profileSvc := application.NewProfileService(profileRepo, deps.TplChecker, govLogger)
    areaSvc := application.NewAreaService(areaRepo, govLogger)

    h := thttp.NewHandler(profileSvc, areaSvc)
    return &Module{Handler: h}
}

func (m *Module) RegisterRoutes(mux *http.ServeMux) {
    m.Handler.RegisterRoutes(mux)
}
```

**Verify:** `go build ./internal/modules/taxonomy/...` → exit 0

---

### Task 2.12 — Run full Phase 2 test suite

```bash
go test ./internal/modules/taxonomy/... -v
```

Expected: all tests `PASS`

**→ Phase 2 complete. Trigger Opus review.**

---

## Phase 3: Registry Module (Go) — Domain + Service

> **Model:** Codex
> **Phase-end review: Opus**

### Task 3.1 — Write domain: `ControlledDocument`

**File:** `internal/modules/registry/domain/controlled_document.go`

```go
package domain

import (
    "errors"
    "fmt"
    "strings"
    "time"
)

type CDStatus string

const (
    CDStatusActive     CDStatus = "active"
    CDStatusObsolete   CDStatus = "obsolete"
    CDStatusSuperseded CDStatus = "superseded"
)

type ControlledDocument struct {
    ID                          string
    TenantID                    string
    ProfileCode                 string
    ProcessAreaCode             string
    DepartmentCode              *string
    Code                        string
    SequenceNum                 *int
    Title                       string
    OwnerUserID                 string
    OverrideTemplateVersionID   *string
    Status                      CDStatus
    CreatedAt                   time.Time
    UpdatedAt                   time.Time
}

var (
    ErrCDNotFound               = errors.New("controlled document not found")
    ErrCDCodeTaken              = errors.New("controlled document code already in use")
    ErrCDArchivedCodeReuse      = errors.New("code was previously used — codes are non-reusable")
    ErrCDNotActive              = errors.New("controlled document is not active")
    ErrManualCodeReasonRequired = errors.New("manual_code_reason required (min 10 chars) for code override")
    ErrOverrideReasonRequired   = errors.New("reason required for template override")
)

func (cd *ControlledDocument) IsActive() bool { return cd.Status == CDStatusActive }

// AutoCode generates the standard code from profile and sequence number.
func AutoCode(profileCode string, seq int) string {
    return fmt.Sprintf("%s-%02d", strings.ToUpper(profileCode), seq)
}
```

**Verify:** `go vet ./internal/modules/registry/domain/...` → exit 0

---

### Task 3.2 — Write domain: `sequence.go`

**File:** `internal/modules/registry/domain/sequence.go`

Define the `SequenceAllocator` port:

```go
package domain

import "context"

// SequenceAllocator atomically claims the next sequence number for a (tenant, profile).
// Implementation must use SELECT FOR UPDATE inside a transaction.
type SequenceAllocator interface {
    // NextAndIncrement claims and returns the next seq, incrementing the counter.
    // Must be called inside an existing transaction.
    NextAndIncrement(ctx context.Context, tx interface{}, tenantID, profileCode string) (int, error)
    // EnsureCounter creates the counter row at seq=1 if not exists (idempotent).
    EnsureCounter(ctx context.Context, tenantID, profileCode string) error
}
```

**Verify:** `go build ./internal/modules/registry/...` → exit 0

---

### Task 3.3 — Write sequence test

**File:** `internal/modules/registry/domain/sequence_test.go`

Integration test (build tag `integration`):
- Create `profile_sequence_counters` row for test profile
- Spawn 50 goroutines each calling `NextAndIncrement` concurrently
- Assert: no duplicate sequence numbers, all 50 numbers are sequential 1..50

**Verify:** `go test ./internal/modules/registry/... -tags=integration -run TestSequenceAllocator_Concurrent -v` → `PASS`

---

### Task 3.4 — Write domain: `resolution.go`

**File:** `internal/modules/registry/domain/resolution.go`

```go
package domain

import "errors"

type TemplateResolutionInput struct {
    OverrideTemplateVersionID       *string
    OverrideTemplateVersionStatus   *string  // nil if override not set
    DefaultTemplateVersionID        *string
    DefaultTemplateVersionStatus    *string  // nil if default not set
    ProfileCode                     string
    OverrideTemplateProfileCode     *string  // for mismatch check
}

type TemplateResolutionResult struct {
    ResolvedVersionID string
}

var (
    ErrProfileHasNoDefaultTemplate = errors.New("profile_has_no_default_template")
    ErrOverrideTemplateDeleted     = errors.New("override_template_deleted")
    ErrOverrideNotPublished        = errors.New("override_template_not_published")
    ErrDefaultObsolete             = errors.New("default_template_obsolete")
    ErrTemplateProfileMismatch     = errors.New("template_profile_mismatch")
)

// Resolve implements the spec resolution table (spec §Template resolution edge cases).
// All ambiguous states return explicit errors — no silent fallbacks.
func Resolve(in TemplateResolutionInput) (TemplateResolutionResult, error) {
    if in.OverrideTemplateVersionID != nil {
        if in.OverrideTemplateVersionStatus == nil {
            return TemplateResolutionResult{}, ErrOverrideTemplateDeleted
        }
        if *in.OverrideTemplateVersionStatus != "published" {
            return TemplateResolutionResult{}, ErrOverrideNotPublished
        }
        if in.OverrideTemplateProfileCode != nil && *in.OverrideTemplateProfileCode != in.ProfileCode {
            return TemplateResolutionResult{}, ErrTemplateProfileMismatch
        }
        return TemplateResolutionResult{ResolvedVersionID: *in.OverrideTemplateVersionID}, nil
    }

    if in.DefaultTemplateVersionID == nil {
        return TemplateResolutionResult{}, ErrProfileHasNoDefaultTemplate
    }
    if in.DefaultTemplateVersionStatus == nil {
        // Default template was deleted
        return TemplateResolutionResult{}, ErrProfileHasNoDefaultTemplate
    }
    if *in.DefaultTemplateVersionStatus == "obsolete" {
        return TemplateResolutionResult{}, ErrDefaultObsolete
    }
    if *in.DefaultTemplateVersionStatus != "published" {
        return TemplateResolutionResult{}, ErrProfileHasNoDefaultTemplate
    }
    return TemplateResolutionResult{ResolvedVersionID: *in.DefaultTemplateVersionID}, nil
}
```

**Verify:** `go build ./internal/modules/registry/...` → exit 0

---

### Task 3.5 — Write resolution tests

**File:** `internal/modules/registry/domain/resolution_test.go`

Cover all spec edge cases:

| Test | Input | Expected |
|---|---|---|
| `TestResolve_Override_Wins` | override=published, default=published | override version returned |
| `TestResolve_Override_Deleted` | override ID set but status nil | `ErrOverrideTemplateDeleted` |
| `TestResolve_Override_NotPublished` | override status="draft" | `ErrOverrideNotPublished` |
| `TestResolve_Override_ProfileMismatch` | override profile != doc profile | `ErrTemplateProfileMismatch` |
| `TestResolve_Default_Only` | no override, default=published | default version returned |
| `TestResolve_BothNull` | no override, no default | `ErrProfileHasNoDefaultTemplate` |
| `TestResolve_Default_Obsolete` | no override, default=obsolete | `ErrDefaultObsolete` |
| `TestResolve_Default_Deleted` | no override, default ID set but status nil | `ErrProfileHasNoDefaultTemplate` |
| `TestResolve_DefaultObsolete_Override_Exists` | override=published, default=obsolete | override returned |

**Verify:** `go test ./internal/modules/registry/domain/... -run TestResolve -v` → all `PASS`

---

### Task 3.6 — Write registry repository port

**File:** `internal/modules/registry/domain/port.go`

```go
package domain

import "context"

type ControlledDocumentRepository interface {
    GetByID(ctx context.Context, tenantID, id string) (*ControlledDocument, error)
    GetByCode(ctx context.Context, tenantID, profileCode, code string) (*ControlledDocument, error)
    // CodeExists checks both active AND archived rows (codes non-reusable).
    CodeExists(ctx context.Context, tenantID, profileCode, code string) (bool, error)
    List(ctx context.Context, tenantID string, filter CDFilter) ([]ControlledDocument, error)
    Create(ctx context.Context, cd *ControlledDocument) error
    UpdateStatus(ctx context.Context, tenantID, id string, status CDStatus) error
}

type CDFilter struct {
    ProfileCode     string
    ProcessAreaCode string
    Status          CDStatus  // empty = all
    OwnerUserID     string
    // UserAreaCodes limits results to docs in areas where user has membership
    UserAreaCodes []string
}
```

**Verify:** `go build ./internal/modules/registry/...` → exit 0

---

### Task 3.7 — Write registry `service.go` (test-first)

**File (test first):** `internal/modules/registry/application/service_test.go`

Write fake implementations of all ports, then tests:
- `Create_AutoCode`: happy path — sequence allocated, code = `"PO-01"`, `governance_events` NOT written (auto-code is not an override)
- `Create_ManualCode`: reason provided, code saved, `governance_events { event_type: "numbering.override" }` written
- `Create_ManualCode_MissingReason`: reason empty → `ErrManualCodeReasonRequired`
- `Create_ManualCode_ReasonTooShort`: reason < 10 chars → `ErrManualCodeReasonRequired`
- `Create_DuplicateCode`: `CodeExists` returns true → `ErrCDCodeTaken`
- `Create_OverrideTemplate_WritesGovernanceEvent`: override set with reason → `governance_events { event_type: "template.override" }`
- `Create_OverrideTemplate_MissingReason`: override set but no reason → `ErrOverrideReasonRequired`
- `Create_ProfileArchived`: profile `IsActive()` = false → `ErrProfileArchived`
- `Create_AreaArchived`: area `IsActive()` = false → `ErrAreaArchived`

**Verify (before impl):** `go test ./internal/modules/registry/application/... -run TestRegistryService -v` → FAIL

**File:** `internal/modules/registry/application/service.go`

Implement `RegistryService.Create`, `Update`, `Obsolete`, `Supersede`, `List`. The `Create` method must:
1. `iam.authz.Check` placeholder (wired in Phase 4)
2. Validate profile + area not archived
3. If `ManualCode`: validate reason ≥ 10 chars, check uniqueness, write `numbering.override` event
4. If auto-code: call `SequenceAllocator.NextAndIncrement` inside tx, build `AutoCode()`
5. If `OverrideTemplateVersionID`: validate published + same profile (via `TemplateVersionChecker`), require reason, write `template.override` event
6. `INSERT controlled_documents`

**Verify:** `go test ./internal/modules/registry/application/... -run TestRegistryService -v` → `PASS`

---

### Task 3.8 — Write registry `migration.go` + startup runner

**File:** `internal/modules/registry/application/migration.go`

**Invocation path:** `BackfillLegacyDocuments` is called once at API startup inside `registry.Module.RunStartupMigrations(ctx)`. Protected by advisory lock `pg_try_advisory_lock(hashtext('registry_backfill'))` so only one pod runs it. Emits structured log lines: `{"event":"backfill","processed":N,"skipped":N,"errors":N}`. If errors > 0, logs each error but does NOT fail startup (Phase B: shadow mode).

Idempotent backfill function `BackfillLegacyDocuments(ctx, db)`:

```
FOR each row in documents_v2 WHERE controlled_document_id IS NULL:
  Look up profile from templates_v2_template_version JOIN templates_v2_template
    using existing templates_v2_template_version_id.
  If found: use existing profile_code + process_area_code metadata.
  If not found: use synthetic profile_code = "unassigned", area_code = "unassigned".
  Ensure "unassigned" profile and area rows exist (upsert).
  Auto-create controlled_document with code = "MIG-" + first 8 chars of document UUID.
  UPDATE documents_v2 SET controlled_document_id, profile_code_snapshot,
    process_area_code_snapshot, templates_v2_template_version_id (already set = snapshot).
```

Add `BackfillLegacyDocuments_ReRunIsNoop` test (integration, build tag `integration`).

**Verify:** `go build ./internal/modules/registry/...` → exit 0

---

### Task 3.9 — Write registry `module.go`

**File:** `internal/modules/registry/module.go`

Follow same pattern as `internal/modules/taxonomy/module.go`. Wire `ControlledDocumentRepository`, `SequenceAllocator`, `RegistryService`, `DBGovernanceLogger`, `Handler`.

**Verify:** `go build ./internal/modules/registry/...` → exit 0

---

### Task 3.10 — Run full Phase 3 unit tests

```bash
go test ./internal/modules/registry/... -v
```

Expected: all unit tests `PASS`

**→ Phase 3 complete. Trigger Opus review.**

---

## Phase 4: IAM Extension (Go)

> **Model:** Codex
> **Phase-end review: Opus**

### Task 4.1 — Add `Capability` type to IAM domain

**File:** `internal/modules/iam/domain/model.go` (EXTEND existing)

Append capability constants:

```go
type Capability string

const (
    CapDocumentView     Capability = "document.view"
    CapDocumentCreate   Capability = "document.create"
    CapDocumentEdit     Capability = "document.edit"
    CapTemplateView     Capability = "template.view"
    CapTemplatePublish  Capability = "template.publish"
    CapWorkflowReview   Capability = "workflow.review"
    CapWorkflowApprove  Capability = "workflow.approve"
    CapRegistryCreate   Capability = "registry.create"
)
```

**Note:** Keep existing `Permission` type untouched — `Capability` is the new v2 type; both coexist during migration.

**Verify:** `go build ./internal/modules/iam/...` → exit 0

---

### Task 4.2 — Write `user_area.go`

**File:** `internal/modules/iam/domain/user_area.go`

```go
package domain

import "time"

type UserProcessArea struct {
    UserID        string
    TenantID      string
    AreaCode      string
    Role          Role
    EffectiveFrom time.Time
    EffectiveTo   *time.Time
    GrantedBy     *string
}

func (u *UserProcessArea) IsActive(now time.Time) bool {
    return !u.EffectiveFrom.After(now) &&
        (u.EffectiveTo == nil || u.EffectiveTo.After(now))
}
```

**Verify:** `go build ./internal/modules/iam/...` → exit 0

---

### Task 4.3 — Write `role_capabilities.go` + test

**File:** `internal/modules/iam/domain/role_capabilities.go`

```go
package domain

const RoleCapabilitiesVersion = 1

var RoleCapabilities = map[Role][]Capability{
    RoleViewer:   {CapDocumentView, CapTemplateView},
    RoleEditor:   {CapDocumentView, CapDocumentCreate, CapDocumentEdit, CapTemplateView},
    RoleReviewer: {CapDocumentView, CapDocumentEdit, CapWorkflowReview, CapTemplateView},
    RoleApprover: {CapDocumentView, CapWorkflowApprove, CapTemplateView, CapTemplatePublish},
}
```

**File:** `internal/modules/iam/domain/role_capabilities_test.go`

Test: `TestRoleCapabilities_VersionBumpEmitsGovernanceEvent`
- Create test DB with `governance_events` table
- Call `OnStartup(db, tenantID)` helper (to be written in service layer)
- Assert row inserted with `event_type = "role.capability_map.version_bump"`
- Run again → no duplicate row (idempotent on same version)

**Verify:** `go test ./internal/modules/iam/domain/... -v` → `PASS`

---

### Task 4.4 — Write `authorization.go` (test-first)

**File (test first):** `internal/modules/iam/application/authorization_test.go`

Tests (use fake `UserAreaRepository`):
- `TestAuthz_RoleCapUnion`: user has viewer in Area-A + editor in Area-B → can `document.create` in Area-B, cannot in Area-A
- `TestAuthz_DenyOverride`: access_policy deny row → capability removed even if role grants it
- `TestAuthz_AllowOverride`: access_policy allow row → capability added even if role doesn't grant it
- `TestAuthz_ExpiredMembership`: `effective_to` = yesterday → 403
- `TestAuthz_SoD_TemplateSelfPublish`: user is author of template version → cannot `template.publish` that version
- `TestAuthz_PerRequestCache`: second `Check` call for same user+capability+resource returns cached result without DB query

**Verify (before impl):** `go test ./internal/modules/iam/application/... -run TestAuthz -v` → FAIL

**File:** `internal/modules/iam/application/authorization.go`

```go
package application

import (
    "context"

    "metaldocs/internal/modules/iam/domain"
)

type ResourceCtx struct {
    AreaCode   string
    ResourceID string // template version ID for SoD check
}

type AuthorizationService struct {
    userAreas     UserAreaRepository
    accessPolicies AccessPolicyRepository
    templateAuthors TemplateAuthorChecker // for SoD on template.publish
}

// Check implements the spec §Authorization check flow:
// (1) load active area memberships → union capabilities
// (2) apply access_policy overrides
// (3) SoD guard for template.publish
// (4) cache result for request lifetime via context
func (s *AuthorizationService) Check(
    ctx context.Context,
    userID string,
    tenantID string,
    cap domain.Capability,
    res ResourceCtx,
) error {
    // ... implementation following spec flow
}
```

**Verify:** `go test ./internal/modules/iam/application/... -run TestAuthz -v` → `PASS`

---

### Task 4.5 — Write `area_membership_service.go` (test-first)

**File (test first):** `internal/modules/iam/application/area_membership_test.go`

Tests:
- `Grant_New`: no existing active membership → new `user_process_areas` row, `governance_events { event_type: "role.grant" }`
- `Grant_Overlap_Merge`: active membership exists → `effective_to` set to new `effective_from`, new row inserted (split)
- `Revoke_Active`: sets `effective_to = now()`
- `Revoke_NonExistent`: 404-equivalent error
- `Grant_UnknownRole`: role not in `RoleCapabilities` → error
- `TemporalQuery_EffectiveTo_Past`: membership with `effective_to < now()` → not returned by active query

**Verify (before impl):** FAIL

**File:** `internal/modules/iam/application/area_membership_service.go`

Implement `AreaMembershipService` with `Grant`, `Revoke` methods. `Grant` must:
1. Validate role in `domain.RoleCapabilities`
2. Check for existing active membership (same user+area) → if found, close it (set `effective_to`)
3. Insert new `user_process_areas` row
4. Write `governance_events { event_type: "role.grant" }`

**Verify:** `go test ./internal/modules/iam/application/... -run TestAreaMembership -v` → `PASS`

---

### Task 4.6 — Write IAM infrastructure: `user_area_repository.go`

**File:** `internal/modules/iam/infrastructure/postgres/user_area_repository.go`

Implement `UserAreaRepository` backed by `user_process_areas` table.
Active membership query:
```sql
SELECT user_id, tenant_id, area_code, role, effective_from, effective_to, granted_by
FROM user_process_areas
WHERE user_id = $1 AND tenant_id = $2
  AND effective_from <= now()
  AND (effective_to IS NULL OR effective_to > now())
```

**Verify:** `go build ./internal/modules/iam/...` → exit 0

---

### Task 4.7 — Extend `middleware.go` to wire `AuthorizationService`

**File:** `internal/modules/iam/delivery/http/middleware.go` (EXTEND)

Add a second middleware constructor `NewV2Middleware(authzSvc *application.AuthorizationService, ...)` that:
- Resolves `(capability, areaCode)` from route context (injected by handlers)
- Calls `authzSvc.Check(ctx, userID, tenantID, cap, res)`
- Returns 403 with `{ code, required_capability, area_code }` on failure
- Keeps existing `Middleware` struct + `NewMiddleware` untouched (backward compat)

**Verify:** `go build ./internal/modules/iam/...` → exit 0

---

### Task 4.8 — Write IAM authz integration test

**File:** `internal/modules/iam/application/authz_integration_test.go` (build tag `integration`)

Tests against real DB:
- Editor in Quality: `Check("document.create", {AreaCode:"quality"})` → pass
- Editor in Quality: `Check("document.create", {AreaCode:"production"})` → fail (403)
- Reviewer: `Check("workflow.approve", ...)` → fail
- `access_policies` deny row overrides role grant
- `effective_to < now()` → capability revoked

**Verify:** `go test ./internal/modules/iam/... -tags=integration -run TestAuthz_Integration -v` → `PASS`

---

### Task 4.9 — Startup version bump check

**File:** `internal/modules/iam/application/startup.go`

```go
package application

import (
    "context"
    "database/sql"
    "encoding/json"

    "metaldocs/internal/modules/iam/domain"
)

// CheckRoleCapabilitiesVersion emits a governance_event if the current
// RoleCapabilitiesVersion differs from the last recorded row.
// Called once at process startup.
func CheckRoleCapabilitiesVersion(ctx context.Context, db *sql.DB, tenantID string) error {
    // SELECT MAX(payload_json->>'version') FROM governance_events
    // WHERE event_type = 'role.capability_map.version_bump' AND tenant_id = $1
    // If nil or != current version → INSERT new event row
    // ...
}
```

**Verify:** `go build ./internal/modules/iam/...` → exit 0

---

### Task 4.10 — Run full Phase 4 unit tests

```bash
go test ./internal/modules/iam/... -v
```

Expected: all `PASS`

**→ Phase 4 complete. Trigger Opus review.**

---

## Phase 5: documents_v2 Bridge

> **Model:** Codex (service logic) · Haiku (domain field additions)

### Task 5.1 — Extend `documents_v2` domain model

**File:** `internal/modules/documents_v2/domain/model.go` (EXTEND)

Add to `Document` struct:

```go
// Bridge fields (Spec 1 — added as nullable, enforced NOT NULL after Phase C migration)
ControlledDocumentID     *string
ProfileCodeSnapshot      *string
ProcessAreaCodeSnapshot  *string
// TemplateVersionIDSnapshot is the existing TemplateVersionID field — now semantically write-once
```

**Verify:** `go build ./internal/modules/documents_v2/...` → exit 0

---

### Task 5.2 — Extend `documents_v2` service: require controlled doc on create (test-first)

**File (test first):** `internal/modules/documents_v2/application/service_cd_test.go`

Tests:
- `Create_FromRegistry_Happy`: controlled_doc active + template resolves → Document created with all snapshots set
- `Create_CD_NotActive`: controlled_doc status = "obsolete" → 409 error
- `Create_NoDefaultTemplate`: profile has no default, no override → `ErrProfileHasNoDefaultTemplate`
- `Create_AuthzFail`: user not in area → 403 error
- `Create_NoControlledDocID`: `controlled_document_id` omitted → validation error

**Verify (before impl):** FAIL

**File:** `internal/modules/documents_v2/application/service.go` (EXTEND existing)

Extend `Create` method. Add:
1. If `ControlledDocumentID` provided:
   - Load `ControlledDocument` from registry
   - Check `status == active`, else 409
   - `authz.Check(actor, "document.create", {AreaCode: cd.ProcessAreaCode})`
   - Call `registry.Resolve(input)` for template resolution
   - Set `ProfileCodeSnapshot`, `ProcessAreaCodeSnapshot`, `TemplateVersionID` (snapshot) on insert
2. If `ControlledDocumentID` omitted: return validation error (Phase B dual-write enforcement)

Add `RegistryReader` and `AuthorizationChecker` port interfaces to the service dependencies.

**Verify:** `go test ./internal/modules/documents_v2/application/... -run TestCreateFromRegistry -v` → `PASS`

---

### Task 5.3 — Run full documents_v2 test suite

```bash
go test ./internal/modules/documents_v2/... -v
```

Expected: all `PASS`, no regressions in existing tests

---

## Phase 6: HTTP Delivery Layer

> **Model:** Sonnet (handlers/routes) · Haiku (module.go updates)
> **Phase-end review: Opus**

### Task 6.1 — Write taxonomy HTTP handler + profile routes

**File:** `internal/modules/taxonomy/delivery/http/routes_profiles.go`

Routes:
```
GET    /api/v2/taxonomy/profiles                 → list (query: includeArchived)
POST   /api/v2/taxonomy/profiles                 → create
GET    /api/v2/taxonomy/profiles/{code}          → get
PUT    /api/v2/taxonomy/profiles/{code}          → update (non-code fields)
PUT    /api/v2/taxonomy/profiles/{code}/default-template → set default template
DELETE /api/v2/taxonomy/profiles/{code}          → archive
```

Request/response shapes follow spec §HTTP mapping. All routes require `iam` admin-level permission.

**File:** `internal/modules/taxonomy/delivery/http/handler.go`

Compose routes, inject `ProfileService` and `AreaService`.

**Verify:** `go build ./internal/modules/taxonomy/delivery/...` → exit 0

---

### Task 6.2 — Write taxonomy area routes

**File:** `internal/modules/taxonomy/delivery/http/routes_areas.go`

Routes:
```
GET    /api/v2/taxonomy/areas
POST   /api/v2/taxonomy/areas
GET    /api/v2/taxonomy/areas/{code}
PUT    /api/v2/taxonomy/areas/{code}
DELETE /api/v2/taxonomy/areas/{code}
```

**Verify:** `go build ./internal/modules/taxonomy/delivery/...` → exit 0

---

### Task 6.3 — Write registry HTTP handler + routes

**File:** `internal/modules/registry/delivery/http/routes.go`

Routes:
```
GET    /api/v2/controlled-documents               → list (filtered by user's areas)
POST   /api/v2/controlled-documents               → create
GET    /api/v2/controlled-documents/{id}          → get detail + revisions
PUT    /api/v2/controlled-documents/{id}/obsolete → obsolete action
PUT    /api/v2/controlled-documents/{id}/supersede → supersede action
```

**File:** `internal/modules/registry/delivery/http/handler.go`

**Verify:** `go build ./internal/modules/registry/delivery/...` → exit 0

---

### Task 6.4 — Write IAM membership routes

**File:** `internal/modules/iam/delivery/http/routes_memberships.go`

Routes:
```
GET    /api/v2/iam/area-memberships               → list (query: userID, areaCode)
POST   /api/v2/iam/area-memberships               → grant
DELETE /api/v2/iam/area-memberships/{id}          → revoke (sets effective_to)
```

**Verify:** `go build ./internal/modules/iam/delivery/...` → exit 0

---

### Task 6.5 — Generate and lock API contract types

**Model: Haiku**

Write `internal/api/v2/types_gen.go` containing Go request/response structs for all new endpoints. These structs are the canonical contract — HTTP handlers serialize/deserialize only via these types.

Also write `frontend/apps/web/src/api/v2-types.ts` — TypeScript mirror of the same structs (hand-written, kept in sync until OpenAPI tooling is added).

Lock expected error codes per endpoint (see spec §HTTP mapping). Add a `contract_test.go` that:
- Validates each handler's 400/403/404/409 responses match the `{ code, ... }` shape
- Fails if a handler returns a raw string body instead of JSON

**Verify:** `go test ./internal/api/... -run TestContractShapes -v` → `PASS`

---

### Task 6.7 — Wire new modules into `main.go`

**File:** `apps/api/cmd/metaldocs-api/main.go` (EXTEND)

1. Import `taxonomy` + `registry` modules
2. Construct `TemplateVersionChecker` adaptor (reads `templates_v2_template_version` table)
3. `taxonomyMod := taxonomy.New(taxonomy.Dependencies{DB: db, TplChecker: tplChecker})`
4. `registryMod := registry.New(registry.Dependencies{...})`
5. `taxonomyMod.RegisterRoutes(mux)`
6. `registryMod.RegisterRoutes(mux)`
7. Replace `NewStaticAuthorizer()` with `NewAuthorizationService(userAreaRepo, accessPoliciesRepo, templateAuthorChecker)`
8. Call `iam.CheckRoleCapabilitiesVersion(ctx, db, tenantID)` at startup — hard-fail if `APP_ENV != "development"` and DB write fails
9. Call `registryMod.RunStartupMigrations(ctx)` to trigger backfill (advisory-locked, non-fatal on error)

**Verify:** `go build ./apps/api/cmd/metaldocs-api/...` → exit 0

---

### Task 6.6 — Run full compile + existing test suite

```bash
go build ./...
go test ./... -v -short
```

Expected: clean build, all existing tests still `PASS`

**→ Phase 6 complete. Trigger Opus review.**

---

## Phase 7: Frontend — Taxonomy Admin

> **Model:** Sonnet (components + pages) · Haiku (types.ts, index.ts, route registration)

### Task 7.1 — Write `taxonomy/types.ts`

**File:** `frontend/apps/web/src/features/taxonomy/types.ts`

```typescript
export interface DocumentProfile {
  code: string;
  tenantId: string;
  familyCode: string;
  name: string;
  description: string;
  reviewIntervalDays: number;
  defaultTemplateVersionId: string | null;
  ownerUserId: string | null;
  editableByRole: string;
  archivedAt: string | null;
  createdAt: string;
}

export interface ProcessArea {
  code: string;
  tenantId: string;
  name: string;
  description: string;
  parentCode: string | null;
  ownerUserId: string | null;
  defaultApproverRole: string | null;
  archivedAt: string | null;
  createdAt: string;
}

export interface SetDefaultTemplateRequest {
  templateVersionId: string;
}

export interface CreateProfileRequest {
  code: string;
  familyCode: string;
  name: string;
  description?: string;
  reviewIntervalDays: number;
  editableByRole?: string;
}

export interface CreateAreaRequest {
  code: string;
  name: string;
  description?: string;
  parentCode?: string;
  defaultApproverRole?: string;
}
```

**Verify:** `cd frontend/apps/web && npx tsc --noEmit` → exit 0

---

### Task 7.2 — Write `taxonomy/api.ts`

**File:** `frontend/apps/web/src/features/taxonomy/api.ts`

React Query hooks:
- `useTaxonomyProfiles(tenantId, includeArchived?)` → `GET /api/v2/taxonomy/profiles`
- `useTaxonomyAreas(tenantId, includeArchived?)` → `GET /api/v2/taxonomy/areas`
- `useCreateProfile()` → `POST /api/v2/taxonomy/profiles`
- `useUpdateProfile()` → `PUT /api/v2/taxonomy/profiles/{code}`
- `useSetDefaultTemplate()` → `PUT /api/v2/taxonomy/profiles/{code}/default-template`
- `useArchiveProfile()` → `DELETE /api/v2/taxonomy/profiles/{code}`
- `useCreateArea()` → `POST /api/v2/taxonomy/areas`
- `useUpdateArea()` → `PUT /api/v2/taxonomy/areas/{code}`
- `useArchiveArea()` → `DELETE /api/v2/taxonomy/areas/{code}`

**Verify:** `cd frontend/apps/web && npx tsc --noEmit` → exit 0

---

### Task 7.3 — Write `ProfileList.tsx`

**File:** `frontend/apps/web/src/features/taxonomy/ProfileList.tsx`

Table with columns: Code, Name, Family, Default Template, Owner, Status (active/archived).
"Set default template" button opens `ProfileEditDialog`. "Archive" button with confirm.

**Verify:** `cd frontend/apps/web && npx tsc --noEmit` → exit 0

---

### Task 7.4 — Write `ProfileEditDialog.tsx`

**File:** `frontend/apps/web/src/features/taxonomy/ProfileEditDialog.tsx`

Dialog fields: name, alias (description), family (dropdown), default_template picker (filtered to published versions of THIS profile only — filter on `templates_v2_template.doc_type_code = profile.code`), review interval, editable_by_role, owner.

The default template picker uses a `useTaxonomyPublishedTemplateVersions(profileCode)` hook — `GET /api/v2/taxonomy/profiles/{code}/template-versions?status=published`.

**Verify:** `cd frontend/apps/web && npx tsc --noEmit` → exit 0

---

### Task 7.5 — Write `AreaList.tsx` + `AreaEditDialog.tsx`

**Files:**
- `frontend/apps/web/src/features/taxonomy/AreaList.tsx`
- `frontend/apps/web/src/features/taxonomy/AreaEditDialog.tsx`

`AreaList`: table with Code, Name, Parent, Owner, Default Approver Role, Status.
`AreaEditDialog`: name, description, parent dropdown (areas list — exclude self + descendants), default_approver_role, owner.

**Verify:** `cd frontend/apps/web && npx tsc --noEmit` → exit 0

---

### Task 7.6 — Write `TaxonomyAdminPage.tsx`

**File:** `frontend/apps/web/src/features/taxonomy/TaxonomyAdminPage.tsx`

Composes `ProfileList` and `AreaList` in tabs. Nav target: "Tipos documentais".

**Verify:** `cd frontend/apps/web && npx tsc --noEmit` → exit 0

---

### Task 7.7 — Write `taxonomy/index.ts` + register route

**File:** `frontend/apps/web/src/features/taxonomy/index.ts`

Export `TaxonomyAdminPage`.

Register `/admin/taxonomy` route in the app router (wherever existing admin routes live — find the pattern from `AdminCenterView.tsx`).

**Verify:** `cd frontend/apps/web && npx tsc --noEmit` → exit 0

---

## Phase 8: Frontend — Registry + IAM Membership + Document Create

> **Model:** Sonnet (pages/dialogs) · Haiku (types, api hooks)

### Task 8.1 — Write `registry/types.ts`

**File:** `frontend/apps/web/src/features/registry/types.ts`

```typescript
export interface ControlledDocument {
  id: string;
  tenantId: string;
  profileCode: string;
  processAreaCode: string;
  departmentCode: string | null;
  code: string;
  sequenceNum: number | null;
  title: string;
  ownerUserId: string;
  overrideTemplateVersionId: string | null;
  status: 'active' | 'obsolete' | 'superseded';
  createdAt: string;
  updatedAt: string;
}

export interface CreateControlledDocumentRequest {
  profileCode: string;
  processAreaCode: string;
  title: string;
  ownerUserId: string;
  overrideTemplateVersionId?: string;
  overrideReason?: string;
  manualCode?: string;
  manualCodeReason?: string;
}
```

**Verify:** `cd frontend/apps/web && npx tsc --noEmit` → exit 0

---

### Task 8.2 — Write `registry/api.ts`

**File:** `frontend/apps/web/src/features/registry/api.ts`

Hooks:
- `useControlledDocuments(filter)` → `GET /api/v2/controlled-documents?...`
- `useControlledDocument(id)` → `GET /api/v2/controlled-documents/{id}`
- `useCreateControlledDocument()` → `POST /api/v2/controlled-documents`
- `useObsoleteControlledDocument()` → `PUT /api/v2/controlled-documents/{id}/obsolete`

**Verify:** `cd frontend/apps/web && npx tsc --noEmit` → exit 0

---

### Task 8.3 — Write `RegistryListPage.tsx` (replace `RegistryExplorerView.tsx`)

**File:** `frontend/apps/web/src/features/registry/RegistryListPage.tsx`

Master list table: Code, Title, Profile, Area, Owner, Status. Filter dropdowns: profile, area, status.
"New" button → `RegistryCreateDialog`. Row click → `RegistryDetailPage`.

Keep `RegistryExplorerView.tsx` in place until routing is switched — add TODO comment referencing this task.

**Verify:** `cd frontend/apps/web && npx tsc --noEmit` → exit 0

---

### Task 8.4 — Write `RegistryCreateDialog.tsx`

**File:** `frontend/apps/web/src/features/registry/RegistryCreateDialog.tsx`

Fields:
- Profile picker (dropdown from `useTaxonomyProfiles`)
- Area picker (dropdown from `useTaxonomyAreas`, filtered to user's areas)
- Title (text)
- Owner (user picker)
- Auto-code preview: `"{PROFILE.toUpperCase()}-NN"` (live preview, NN = estimated next seq from `GET /api/v2/taxonomy/profiles/{code}/next-seq-preview`)
- Admin-only toggle "Manual code" → reveals code input + reason textarea
- Admin-only toggle "Override template" → reveals template version picker + reason textarea

**Verify:** `cd frontend/apps/web && npx tsc --noEmit` → exit 0

---

### Task 8.5 — Write `RegistryDetailPage.tsx`

**File:** `frontend/apps/web/src/features/registry/RegistryDetailPage.tsx`

Shows: controlled doc metadata, status badge, list of `documents_v2` revisions linked to this CD. "Obsolete" action button (admin-only).

**Verify:** `cd frontend/apps/web && npx tsc --noEmit` → exit 0

---

### Task 8.6 — Write IAM `AreaMembershipAdminPage.tsx` + `MembershipGrantDialog.tsx`

**Files:**
- `frontend/apps/web/src/features/iam/AreaMembershipAdminPage.tsx`
- `frontend/apps/web/src/features/iam/MembershipGrantDialog.tsx`

`AreaMembershipAdminPage`: user × area × role matrix table. Filter by user or area. "Grant" button opens dialog.
`MembershipGrantDialog`: user picker, area picker, role selector (`viewer|editor|reviewer|approver`), effective_from date, reason field (→ `governance_events`).

**Hooks needed:**
- `useAreaMemberships(filter)` → `GET /api/v2/iam/area-memberships`
- `useGrantAreaMembership()` → `POST /api/v2/iam/area-memberships`
- `useRevokeAreaMembership()` → `DELETE /api/v2/iam/area-memberships/{id}`

Add to `frontend/apps/web/src/features/iam/api.ts` (new file).

**Verify:** `cd frontend/apps/web && npx tsc --noEmit` → exit 0

---

### Task 8.7 — Modify `DocumentCreatePage.tsx`

**File:** Locate existing document create page (search `frontend/apps/web/src/features/documents/` for create/new pattern).

Modify create flow:
1. First step: pick `ControlledDocument` from registry (`useControlledDocuments` filtered to user's areas)
2. Once CD selected: template auto-resolves (display resolved template name, read-only)
3. Admin-only toggle: "Override template" → reveals version picker
4. Submit: `{ controlled_document_id, name? }` → `POST /api/v2/documents`

**Verify:** `cd frontend/apps/web && npx tsc --noEmit` → exit 0

---

### Task 8.8 — Run full frontend type check + unit tests

```bash
cd frontend/apps/web && npx tsc --noEmit
cd frontend/apps/web && npx vitest run
```

Expected: 0 type errors, all unit tests `PASS`

**→ Phase 8 complete. Trigger Opus review.**

---

## Phase 9: Integration Tests + Backfill Execution

> **Model:** Codex

### Task 9.1 — Write registry integration test

**File:** `internal/modules/registry/application/integration_test.go` (build tag `integration`)

Full flow tests:
- `TestFullFlow_CreateProfile_SetTemplate_CreateCD_CreateDocument`: create profile → set default template → create controlled_doc → create `documents_v2` row → assert all snapshot fields populated
- `TestBackfill_SeedLegacyDocs_RunBackfill_AssertAllLinked`: seed 25 docs with NULL CD → run `BackfillLegacyDocuments` → assert count of NULL rows = 0
- `TestBackfill_ReRunIsNoop`: run backfill twice → second run inserts 0 rows
- `TestCrossProfileOverride_Rejected`: create CD with override template from different profile → `ErrTemplateProfileMismatch`
- `TestRename_Flow`: create new profile code → archive old → assert old documents keep old `profile_code_snapshot`

**Verify:** `go test ./internal/modules/registry/... -tags=integration -v` → all `PASS`

---

### Task 9.2 — Write tenant isolation integration test

**File:** `internal/modules/registry/application/tenant_isolation_test.go` (build tag `integration`)

Tests:
- Insert controlled_document row with `tenant_id = A`, `profile_code` belonging to tenant B → FK violation
- Insert `documents_v2` row with `controlled_document_id` from different tenant → trigger raises exception
- `GET /api/v2/controlled-documents` with tenant-A JWT → returns 0 rows from tenant-B data
- `GET /api/v2/taxonomy/profiles` with tenant-A JWT → returns 0 rows from tenant-B data
- `GET /api/v2/iam/area-memberships?userId=X` with tenant-A JWT → returns only tenant-A memberships for user X even if user X also has tenant-B memberships
- `POST /api/v2/controlled-documents` body references `profile_code` from tenant-B → 404 (not visible, not FK-joinable)

**Verify:** `go test ./internal/modules/registry/... -tags=integration -run TestTenantIsolation -v` → `PASS`

---

### Task 9.3 — Authz performance benchmark

**File:** `internal/modules/iam/application/authorization_bench_test.go`

**Protocol:** Seed 10 users each with 5 active area memberships (50 rows). Warmup: 100 calls discarded. Run: 1000 calls, measure wall time. Accept gate: p99 < 5ms measured via `testing.B` with `b.ReportMetric`. Benchmark script committed to `scripts/bench_authz.sh`.

**Verify:**
```bash
go test ./internal/modules/iam/... -tags=integration \
  -bench=BenchmarkAuthzCheck -benchtime=1000x -count=3
```
Expected: `ns/op` translates to < 5ms p99 across 3 runs.

---

### Task 9.4 — Sequence counter concurrency benchmark

**File:** `internal/modules/registry/domain/sequence_bench_test.go`

**Protocol:** Seed profile + counter row. Spawn 100 goroutines simultaneously (use `sync.WaitGroup` + buffered chan for coordination). Each goroutine creates one `controlled_document`. Accept gates: (a) no duplicate sequence numbers in result set; (b) all 100 codes present; (c) p99 < 50ms wall time. Run against local Postgres container only (build tag `integration`).

**Verify:**
```bash
go test ./internal/modules/registry/... -tags=integration \
  -bench=BenchmarkConcurrentCreate_100 -benchtime=5s
```

---

### Task 9.5 — Verify backfill completion before Phase C

Check NULL count via admin endpoint or direct query:
```bash
docker compose -f deploy/compose/docker-compose.yml exec postgres \
  psql -U metaldocs -d metaldocs \
  -c "SELECT COUNT(*) FROM documents_v2 WHERE controlled_document_id IS NULL;"
```
Expected: `0`. Only run migration 0129 after this confirms zero.

---

## Phase 10: E2E Tests (Playwright)

> **Model:** Codex

### Task 10.1 — Write `e2e/taxonomy.spec.ts`

**File:** `frontend/apps/web/e2e/taxonomy.spec.ts`

```
Test: admin creates profile via "Tipos documentais" → nav badge count updates
Test: set profile default template → picker shows only published versions of this profile
```

Use `makeProfile(code, name)` fixture helper that generates unique codes per test (no hardcoded 'po').

**Verify:** `cd frontend/apps/web && npx playwright test e2e/taxonomy.spec.ts` → passed

---

### Task 10.2 — Write `e2e/registry.spec.ts`

**File:** `frontend/apps/web/e2e/registry.spec.ts`

```
Test: auto-code increments per profile (create 3 CDs → codes are "XX-01","XX-02","XX-03")
Test: manual code + reason works
Test: duplicate code shows 409 error in UI
Test: missing reason for manual code shows validation error
```

**Verify:** `cd frontend/apps/web && npx playwright test e2e/registry.spec.ts` → passed

---

### Task 10.3 — Write `e2e/area-membership.spec.ts`

**File:** `frontend/apps/web/e2e/area-membership.spec.ts`

```
Test: admin grants user area/editor → user sees only assigned-area docs
Test: revoke membership → user gets 403 on create attempt
```

**Verify:** `cd frontend/apps/web && npx playwright test e2e/area-membership.spec.ts` → passed

---

### Task 10.4 — Write `e2e/document-from-registry.spec.ts`

**File:** `frontend/apps/web/e2e/document-from-registry.spec.ts`

```
Test: pick controlled doc → template auto-resolves → editable document created with snapshots
Test: controlled doc with no default template shows error prompting admin action
```

**Verify:** `cd frontend/apps/web && npx playwright test e2e/document-from-registry.spec.ts` → passed

---

### Task 10.5 — Write `e2e/sod.spec.ts`

**File:** `frontend/apps/web/e2e/sod.spec.ts`

```
Test: author submits template for review → same user cannot review (422 sod_violation shown in UI)
```

**Verify:** `cd frontend/apps/web && npx playwright test e2e/sod.spec.ts` → passed

---

### Task 10.6 — Write `e2e/rename-via-new-code.spec.ts`

**File:** `frontend/apps/web/e2e/rename-via-new-code.spec.ts`

```
Test: admin creates new profile, archives old → old documents render with historical profile code
Test: attempt direct code rename via API PUT → 409 (trigger blocks)
```

**Verify:** `cd frontend/apps/web && npx playwright test e2e/rename-via-new-code.spec.ts` → passed

---

### Task 10.7 — Run full E2E suite

```bash
cd frontend/apps/web && npx playwright test e2e/taxonomy.spec.ts e2e/registry.spec.ts \
  e2e/area-membership.spec.ts e2e/document-from-registry.spec.ts \
  e2e/sod.spec.ts e2e/rename-via-new-code.spec.ts
```

Expected: all passed, 0 failures

---

## Known Gaps (Codex round 2 — not blocking, address before Phase C cutover)

These issues were flagged across 2 Codex hardening rounds. None block Phase 1–8 execution, but items marked **structural** should be resolved before running migration 0129 in production.

| # | Scope | Issue | Where to fix |
|---|---|---|---|
| 1 | local | Missing HTTP boundary tests: no-tenant JWT → 401, malformed tenant UUID → 400, token-tenant vs resource-tenant mismatch → 403/404 | Add to Task 6.5 `contract_test.go` per route family |
| 2 | **structural** | Phase C cutover has no automated CI gate — Task 9.5 is a manual step | Add a pre-deploy script `scripts/phase-c-gate.sh` that queries NULL count and exits non-zero if > 0; wire into deploy pipeline before 0129 runs |
| 3 | local | Backfill partial-failure recovery untested: injected row error + rerun; concurrent startup advisory lock proves single-writer | Add two integration tests to Task 3.8: `TestBackfill_PartialError_RerunSucceeds`, `TestBackfill_ConcurrentStartup_OneWriter` |
| 4 | local | Contract test matrix for document create does not cover all spec error shapes (inactive CD, archived profile, archived area, override cross-profile) | Extend Task 6.5 `contract_test.go` with these 4 negative cases + exact HTTP status + `{ code }` field |
| 5 | **structural** | E2E has no migrated-tenant scenario — all specs start from empty tenant | Add `e2e/migrated-tenant.spec.ts`: seed legacy documents, run backfill via admin endpoint, assert post-backfill create/edit works |

---

## Manual Smoke Checklist (pre-merge)

Run against dev environment after all phases pass:

- [ ] Create profile via "Tipos documentais" → nav badge count updates
- [ ] Set profile default template → picker shows only published versions of THIS profile
- [ ] Create controlled doc (auto-code) → code reflects `{PROFILE.upper()}-{NN}` (zero-padded)
- [ ] Create controlled doc (manual code) → reason required, duplicate rejected
- [ ] Assign user to area → user sees only area-filtered registry list
- [ ] Attempt SoD violation (author reviews own) → blocked with 422
- [ ] Migrate 25 existing docs → all linked, no broken list entries
- [ ] Attempt direct SQL `UPDATE metaldocs.document_profiles SET code = ...` → DB trigger rejects

---

## Coverage Targets

| Module | Target |
|---|---|
| `registry/*` | 85% |
| `iam/authorization.go` | 90% |
| `taxonomy/*` | 75% |
| `documents_v2` bridge code | 80% |

Verify after Phase 9:
```bash
go test ./... -coverprofile=coverage.out
go tool cover -func=coverage.out | grep -E "registry|iam/application/authorization|taxonomy|documents_v2"
```

---

## Execution Handoff

Recommended: **Subagent-Driven Development** (`nexus:subagent-driven-development`)

Phases 1–5 are sequential (DB → domain → service — each depends on previous).
Phases 6–8 (HTTP delivery + frontend) can start in parallel once Phase 5 is complete.
Phase 9 (integration tests) requires the running DB stack.
Phase 10 (E2E) requires the full stack (API + frontend + DB).

Immediate next step: create worktree `feature/foundation-spec1`, then start Phase 1.
