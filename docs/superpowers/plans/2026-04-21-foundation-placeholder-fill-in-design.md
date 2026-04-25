# Foundation Placeholder Fill-In + Eigenpal Variable Fanout — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Ship an ISO 9001 §7.5.3 / §8.5.1 placeholder + editable-zone system with three-layer freeze (template → revision snapshot → render artifact), triple-hash (`schema_hash + values_hash + content_hash`), and DOCX→PDF fanout, matching the Spec 3 architecture exactly.

**Architecture:** Extend `templates_v2` domain with richer placeholder/zone schemas, snapshot template artifacts onto `documents_v2.documents` at create, persist fill-in values in two new tables, resolve computed placeholders through a typed registry at approval freeze, fan out a single-pass `processTemplate` pipeline (`apps/docgen-v2`) producing immutable DOCX + asynchronous PDF, expose a signed-URL viewer gated by Spec 1 area RBAC, and bind Spec 2 signatures to `(content_hash, values_hash, schema_hash)`.

**Tech Stack:** Go (backend), PostgreSQL (migrations numbered from 0152), Node/TypeScript (`apps/docgen-v2`), eigenpal `@eigenpal/docx-js-editor` (SDT + bookmarks), Gotenberg (PDF), S3 (artifacts), Azure Service Bus (existing `docgen_v2_pdf`), ProseMirror `filterTransaction` plugin (editor guardrails).

**Model delegation (per session policy):**
- **Codex gpt-5.3-codex — reasoning_effort: medium** — default for all Go/TS implementation tasks in every phase.
- **Haiku (claude-haiku-4-5-20251001)** — trivial typed struct additions, migration SQL stubs, handler routing wire-up (Phases 1, 2, 5/handlers, 17/quickwires).
- **Sonnet (claude-sonnet-4-6)** — medium-complexity orchestration, validation logic, cycle detection, repository glue (Phases 3, 4, 6, 9).
- **Opus (this model)** — end-of-phase review only. Does not implement.

Every phase ends with `rtk go test ./...` (backend) or `pnpm -C apps/docgen-v2 test` (fanout), zero-failure gate before commit. Every commit is TDD (failing test → implementation → green).

---

## File Structure

**Backend (Go)**

- `internal/modules/templates_v2/domain/schemas.go` — extended `Placeholder`, `EditableZone`, new `VisibilityCondition`, `ContentPolicy`, `CompositionConfig`, `PlaceholderType` constants.
- `internal/modules/templates_v2/application/schema.go` — extended validation (regex compile, numeric/date range sanity, `visible_if` topo-sort, unique IDs).
- `internal/modules/templates_v2/domain/errors.go` — extended with `ErrPlaceholderCycle`, `ErrUnknownResolver`, `ErrInvalidConstraint`.
- `internal/modules/documents_v2/domain/fillin.go` — new: `PlaceholderValue`, `ZoneContent` value objects; `ValuesHash` computation (canonical JSON).
- `internal/modules/documents_v2/domain/snapshot.go` — new: `TemplateSnapshot` struct grouping schema/body/composition hashes.
- `internal/modules/documents_v2/application/snapshot_service.go` — new: snapshots template on revision create.
- `internal/modules/documents_v2/application/fillin_service.go` — new: upserts placeholder values + zone content during draft.
- `internal/modules/documents_v2/application/freeze_service.go` — new: at approval transition — validate, resolve computed, compute `values_hash`, call fanout.
- `internal/modules/documents_v2/repository/fillin_repository.go` — new: `document_placeholder_values` + `document_editable_zone_content` CRUD.
- `internal/modules/documents_v2/repository/snapshot_repository.go` — new: read/write snapshot columns on `documents`.
- `internal/modules/documents_v2/http/fillin_handler.go` — new: `PUT /api/v2/documents/{id}/placeholders/{pid}` and `PUT …/zones/{zid}`.
- `internal/modules/documents_v2/http/view_handler.go` — new: `GET /api/v2/documents/{id}/view` returns signed PDF URL.
- `internal/modules/render/resolvers/resolver.go` — new: `ComputedResolver` interface + `Registry`.
- `internal/modules/render/resolvers/{doc_code,revision_number,effective_date,controlled_by_area,author,approvers,approval_date}.go` — new: seven v1 resolvers.
- `internal/modules/render/resolvers/registry.go` — new: typed registry map with `Version()` pinning.
- `internal/modules/render/fanout/client.go` — new: HTTP client calling `apps/docgen-v2` fanout endpoint.
- `internal/modules/render/fanout/pdf_dispatcher.go` — new: enqueues Service Bus `docgen_v2_pdf` job.
- `internal/modules/render/fanout/reconstruction.go` — new: append-only `reconstruction_attempts` writer.
- `migrations/0152_placeholder_fillin_columns.sql` — new: adds columns on `documents`, creates two tables.
- `migrations/0153_placeholder_values_ck_tenant.sql` — new: tenant-consistency triggers mirroring Spec 2 pattern.

**Fanout service (TypeScript, `apps/docgen-v2`)**

- `apps/docgen-v2/src/render/fanout.ts` — new: orchestrates placeholder fill + zone injection + composition in one `processTemplate` pass.
- `apps/docgen-v2/src/render/subblocks/registry.ts` — new: `SubBlockRenderer` registry.
- `apps/docgen-v2/src/render/subblocks/{doc_header_standard,revision_box,approval_signatures_block,footer_page_numbers,footer_controlled_copy_notice}.ts` — new: five v1 sub-blocks returning OOXML fragments.
- `apps/docgen-v2/src/render/zoneInjection.ts` — new: inject OOXML between `zone-start:<id>` bookmark pairs.
- `apps/docgen-v2/src/routes/fanout.ts` — new: `POST /render/fanout` accepts frozen snapshot + values, returns final DOCX bytes + content_hash.
- `apps/docgen-v2/src/routes/pdf.ts` — modify: PDF worker reads `final_docx_s3_key`, produces `final_pdf_s3_key` + `pdf_hash`.

**Frontend (eigenpal canvas)**

- `frontend/apps/web/src/editor-adapters/eigenpal-template-mode.ts` — extend: wrap frozen content in `lock: "sdtContentLocked"` SDT; emit typed `sdtType` per placeholder type.
- `frontend/apps/web/src/editor-adapters/filter-transaction-guard.ts` — new: ProseMirror `filterTransaction` plugin rejecting edits outside unlocked SDT/zone ranges.
- `frontend/apps/web/src/editor-adapters/__tests__/filter-transaction-guard.test.ts` — new.

Total: ~28 new files + ~6 modified. 18 phases, ~115 tasks.

---

## Phase 1 — Domain schema extensions (Haiku)

### Task 1.1: Extend `PlaceholderType` constants

**Files:**
- Modify: `internal/modules/templates_v2/domain/schemas.go`
- Test: `internal/modules/templates_v2/domain/schemas_test.go`

- [ ] **Step 1: Write failing test**

```go
// internal/modules/templates_v2/domain/schemas_test.go
package domain

import "testing"

func TestPlaceholderType_AllConstants(t *testing.T) {
    types := []PlaceholderType{PHText, PHDate, PHNumber, PHSelect, PHUser, PHPicture, PHComputed}
    wants := []string{"text", "date", "number", "select", "user", "picture", "computed"}
    for i, pt := range types {
        if string(pt) != wants[i] {
            t.Fatalf("PlaceholderType[%d] = %q, want %q", i, pt, wants[i])
        }
    }
}
```

- [ ] **Step 2: Run test — FAIL (PHPicture, PHComputed undefined)**

Run: `rtk go test ./internal/modules/templates_v2/domain/ -run TestPlaceholderType_AllConstants -v`
Expected: FAIL — undefined `PHPicture`, `PHComputed`.

- [ ] **Step 3: Add constants**

Edit `internal/modules/templates_v2/domain/schemas.go`:

```go
const (
    PHText     PlaceholderType = "text"
    PHDate     PlaceholderType = "date"
    PHNumber   PlaceholderType = "number"
    PHSelect   PlaceholderType = "select"
    PHUser     PlaceholderType = "user"
    PHPicture  PlaceholderType = "picture"
    PHComputed PlaceholderType = "computed"
)
```

- [ ] **Step 4: Run — PASS**

Run: `rtk go test ./internal/modules/templates_v2/domain/ -run TestPlaceholderType_AllConstants -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
rtk git add internal/modules/templates_v2/domain/schemas.go internal/modules/templates_v2/domain/schemas_test.go
rtk git commit -m "feat(templates_v2): add picture and computed placeholder types"
```

### Task 1.2: Extend `Placeholder` struct with validation + visibility + computed fields

**Files:**
- Modify: `internal/modules/templates_v2/domain/schemas.go`
- Test: `internal/modules/templates_v2/domain/schemas_test.go`

- [ ] **Step 1: Write failing test**

```go
func TestPlaceholder_JSONRoundTrip_AllFields(t *testing.T) {
    regex := "^[A-Z]{3}-\\d{4}$"
    mn, mx := 0.0, 100.0
    maxLen := 120
    rkey := "doc_code"
    ph := Placeholder{
        ID: "p1", Label: "Doc Code", Type: PHText, Required: true,
        Regex: &regex, MaxLength: &maxLen, MinNumber: &mn, MaxNumber: &mx,
        VisibleIf: &VisibilityCondition{PlaceholderID: "p0", Op: "eq", Value: "x"},
        Computed: true, ResolverKey: &rkey,
    }
    b, err := json.Marshal(ph)
    if err != nil { t.Fatal(err) }
    var back Placeholder
    if err := json.Unmarshal(b, &back); err != nil { t.Fatal(err) }
    if back.ID != "p1" || !back.Computed || back.ResolverKey == nil || *back.ResolverKey != "doc_code" {
        t.Fatalf("round-trip mismatch: %+v", back)
    }
    if back.VisibleIf == nil || back.VisibleIf.Op != "eq" {
        t.Fatalf("visible_if lost: %+v", back.VisibleIf)
    }
}
```

Add import `"encoding/json"` at top of test file.

- [ ] **Step 2: Run — FAIL (fields missing)**

Run: `rtk go test ./internal/modules/templates_v2/domain/ -run TestPlaceholder_JSONRoundTrip_AllFields -v`
Expected: FAIL — `Regex`, `VisibilityCondition`, etc. undefined.

- [ ] **Step 3: Implement extended struct**

Replace the `Placeholder` struct block in `schemas.go`:

```go
type VisibilityCondition struct {
    PlaceholderID string `json:"placeholder_id"`
    Op            string `json:"op"` // eq | neq | in | not_in
    Value         any    `json:"value"`
}

type Placeholder struct {
    ID       string          `json:"id"`
    Label    string          `json:"label"`
    Type     PlaceholderType `json:"type"`
    Required bool            `json:"required"`
    Default  any             `json:"default,omitempty"`
    Options  []string        `json:"options,omitempty"`

    Regex       *string              `json:"regex,omitempty"`
    MinNumber   *float64             `json:"min_number,omitempty"`
    MaxNumber   *float64             `json:"max_number,omitempty"`
    MinDate     *string              `json:"min_date,omitempty"`
    MaxDate     *string              `json:"max_date,omitempty"`
    MaxLength   *int                 `json:"max_length,omitempty"`
    VisibleIf   *VisibilityCondition `json:"visible_if,omitempty"`
    Computed    bool                 `json:"computed,omitempty"`
    ResolverKey *string              `json:"resolver_key,omitempty"`
}
```

- [ ] **Step 4: Run — PASS**

Run: `rtk go test ./internal/modules/templates_v2/... -v`
Expected: PASS (all existing tests still green — new fields all omitempty).

- [ ] **Step 5: Commit**

```bash
rtk git add internal/modules/templates_v2/domain/schemas.go internal/modules/templates_v2/domain/schemas_test.go
rtk git commit -m "feat(templates_v2): extend Placeholder with validation, visibility, computed"
```

### Task 1.3: Extend `EditableZone` + add `ContentPolicy`

**Files:**
- Modify: `internal/modules/templates_v2/domain/schemas.go`
- Test: `internal/modules/templates_v2/domain/schemas_test.go`

- [ ] **Step 1: Write failing test**

```go
func TestEditableZone_ContentPolicy_RoundTrip(t *testing.T) {
    ml := 5000
    z := EditableZone{
        ID: "z1", Label: "Intro", Required: true,
        ContentPolicy: ContentPolicy{AllowTables: true, AllowImages: false, AllowHeadings: true, AllowLists: true},
        MaxLength: &ml,
    }
    b, _ := json.Marshal(z)
    var back EditableZone
    if err := json.Unmarshal(b, &back); err != nil { t.Fatal(err) }
    if !back.ContentPolicy.AllowTables || back.ContentPolicy.AllowImages {
        t.Fatalf("content policy: %+v", back.ContentPolicy)
    }
    if back.MaxLength == nil || *back.MaxLength != 5000 {
        t.Fatalf("max_length: %v", back.MaxLength)
    }
}
```

- [ ] **Step 2: Run — FAIL (ContentPolicy missing)**

Run: `rtk go test ./internal/modules/templates_v2/domain/ -run TestEditableZone_ContentPolicy_RoundTrip -v`
Expected: FAIL.

- [ ] **Step 3: Implement**

In `schemas.go`:

```go
type ContentPolicy struct {
    AllowTables   bool `json:"allow_tables"`
    AllowImages   bool `json:"allow_images"`
    AllowHeadings bool `json:"allow_headings"`
    AllowLists    bool `json:"allow_lists"`
}

type EditableZone struct {
    ID            string        `json:"id"`
    Label         string        `json:"label"`
    Required      bool          `json:"required"`
    ContentPolicy ContentPolicy `json:"content_policy"`
    MaxLength     *int          `json:"max_length,omitempty"`
}
```

- [ ] **Step 4: Run — PASS**

Run: `rtk go test ./internal/modules/templates_v2/... -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
rtk git add internal/modules/templates_v2/domain/schemas.go internal/modules/templates_v2/domain/schemas_test.go
rtk git commit -m "feat(templates_v2): EditableZone ContentPolicy + MaxLength"
```

### Task 1.4: Add `CompositionConfig`

**Files:**
- Modify: `internal/modules/templates_v2/domain/schemas.go`
- Test: `internal/modules/templates_v2/domain/schemas_test.go`

- [ ] **Step 1: Write failing test**

```go
func TestCompositionConfig_RoundTrip(t *testing.T) {
    c := CompositionConfig{
        HeaderSubBlocks: []string{"doc_header_standard"},
        FooterSubBlocks: []string{"footer_page_numbers", "footer_controlled_copy_notice"},
        SubBlockParams: map[string]map[string]any{
            "doc_header_standard": {"show_logo": true},
        },
    }
    b, _ := json.Marshal(c)
    var back CompositionConfig
    if err := json.Unmarshal(b, &back); err != nil { t.Fatal(err) }
    if len(back.FooterSubBlocks) != 2 { t.Fatalf("footer: %+v", back.FooterSubBlocks) }
    if back.SubBlockParams["doc_header_standard"]["show_logo"] != true {
        t.Fatalf("params lost: %+v", back.SubBlockParams)
    }
}
```

- [ ] **Step 2: Run — FAIL**

- [ ] **Step 3: Implement**

```go
type CompositionConfig struct {
    HeaderSubBlocks []string                       `json:"header_sub_blocks"`
    FooterSubBlocks []string                       `json:"footer_sub_blocks"`
    SubBlockParams  map[string]map[string]any      `json:"sub_block_params"`
}
```

- [ ] **Step 4: Run — PASS**

- [ ] **Step 5: Commit**

```bash
rtk git add internal/modules/templates_v2/domain/
rtk git commit -m "feat(templates_v2): CompositionConfig type"
```

---

## Phase 2 — Migration (schema snapshot columns + value tables) (Haiku)

### Task 2.1: Write migration 0152

**Files:**
- Create: `migrations/0152_placeholder_fillin_columns.sql`
- Test: `migrations/0152_placeholder_fillin_columns_test.sql` (pg-verify style)

- [ ] **Step 1: Write failing integration test**

Create `tests/integration/migrations/migration_0152_test.go`:

```go
package migrations_test

import (
    "context"
    "testing"

    "github.com/metaldocs/metaldocs-api/tests/integration/testdb"
)

func TestMigration0152_DocumentsColumns(t *testing.T) {
    ctx := context.Background()
    db := testdb.Open(ctx, t)
    defer db.Close()

    cols := []string{
        "placeholder_schema_snapshot", "placeholder_schema_hash",
        "composition_config_snapshot", "composition_config_hash",
        "editable_zones_schema_snapshot",
        "body_docx_snapshot_s3_key", "body_docx_hash",
        "values_frozen_at", "values_hash",
        "final_docx_s3_key", "final_pdf_s3_key", "pdf_hash",
        "pdf_generated_at", "reconstruction_attempts",
    }
    for _, c := range cols {
        var exists bool
        err := db.QueryRowContext(ctx,
            `SELECT EXISTS (SELECT 1 FROM information_schema.columns
             WHERE table_schema='public' AND table_name='documents' AND column_name=$1)`,
            c).Scan(&exists)
        if err != nil || !exists {
            t.Fatalf("column documents.%s missing: err=%v exists=%v", c, err, exists)
        }
    }
}

func TestMigration0152_NewTables(t *testing.T) {
    ctx := context.Background()
    db := testdb.Open(ctx, t)
    defer db.Close()
    for _, tbl := range []string{"document_placeholder_values", "document_editable_zone_content"} {
        var exists bool
        err := db.QueryRowContext(ctx,
            `SELECT EXISTS (SELECT 1 FROM information_schema.tables
             WHERE table_schema='public' AND table_name=$1)`,
            tbl).Scan(&exists)
        if err != nil || !exists {
            t.Fatalf("table %s missing: err=%v", tbl, err)
        }
    }
}
```

- [ ] **Step 2: Run — FAIL (migration not present)**

Run: `rtk go test ./tests/integration/migrations/ -run TestMigration0152 -v`
Expected: FAIL — columns do not exist.

- [ ] **Step 3: Create migration**

`migrations/0152_placeholder_fillin_columns.sql`:

```sql
BEGIN;

ALTER TABLE public.documents
    ADD COLUMN placeholder_schema_snapshot     JSONB,
    ADD COLUMN placeholder_schema_hash         BYTEA,
    ADD COLUMN composition_config_snapshot     JSONB,
    ADD COLUMN composition_config_hash         BYTEA,
    ADD COLUMN editable_zones_schema_snapshot  JSONB,
    ADD COLUMN body_docx_snapshot_s3_key       TEXT,
    ADD COLUMN body_docx_hash                  BYTEA,
    ADD COLUMN values_frozen_at                TIMESTAMPTZ,
    ADD COLUMN values_hash                     BYTEA,
    ADD COLUMN final_docx_s3_key               TEXT,
    ADD COLUMN final_pdf_s3_key                TEXT,
    ADD COLUMN pdf_hash                        BYTEA,
    ADD COLUMN pdf_generated_at                TIMESTAMPTZ,
    ADD COLUMN reconstruction_attempts         JSONB NOT NULL DEFAULT '[]'::jsonb;

-- Enforce snapshot presence for revisions created after this migration (existing rows left nullable for backfill safety).
CREATE OR REPLACE FUNCTION enforce_snapshot_on_submit() RETURNS trigger AS $$
BEGIN
    IF NEW.status IN ('under_review','approved','scheduled','published')
       AND (NEW.placeholder_schema_snapshot IS NULL
         OR NEW.placeholder_schema_hash IS NULL
         OR NEW.composition_config_snapshot IS NULL
         OR NEW.composition_config_hash IS NULL
         OR NEW.body_docx_snapshot_s3_key IS NULL
         OR NEW.body_docx_hash IS NULL) THEN
        RAISE EXCEPTION 'documents.% snapshot columns required for status=%',
            NEW.id, NEW.status
            USING ERRCODE = 'check_violation';
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS enforce_snapshot_on_submit_trg ON public.documents;
CREATE TRIGGER enforce_snapshot_on_submit_trg
    BEFORE INSERT OR UPDATE ON public.documents
    FOR EACH ROW EXECUTE FUNCTION enforce_snapshot_on_submit();

CREATE TABLE public.document_placeholder_values (
    tenant_id        TEXT        NOT NULL,
    revision_id      UUID        NOT NULL,
    placeholder_id   TEXT        NOT NULL,
    value_text       TEXT,
    value_typed      JSONB,
    source           TEXT        NOT NULL CHECK (source IN ('user','computed','default')),
    computed_from    TEXT,
    resolver_version INT,
    inputs_hash      BYTEA,
    validated_at     TIMESTAMPTZ,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (tenant_id, revision_id, placeholder_id),
    FOREIGN KEY (revision_id) REFERENCES public.documents(id) ON DELETE CASCADE
);

CREATE INDEX idx_dpv_revision ON public.document_placeholder_values(revision_id);

CREATE TABLE public.document_editable_zone_content (
    tenant_id     TEXT  NOT NULL,
    revision_id   UUID  NOT NULL,
    zone_id       TEXT  NOT NULL,
    content_ooxml TEXT  NOT NULL,
    content_hash  BYTEA NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (tenant_id, revision_id, zone_id),
    FOREIGN KEY (revision_id) REFERENCES public.documents(id) ON DELETE CASCADE
);

CREATE INDEX idx_dezc_revision ON public.document_editable_zone_content(revision_id);

COMMIT;
```

- [ ] **Step 4: Run — PASS**

Run: `rtk go test ./tests/integration/migrations/ -run TestMigration0152 -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
rtk git add migrations/0152_placeholder_fillin_columns.sql tests/integration/migrations/migration_0152_test.go
rtk git commit -m "feat(migrations): 0152 placeholder fillin columns and value tables"
```

### Task 2.2: Tenant-consistency triggers (0153)

**Files:**
- Create: `migrations/0153_placeholder_values_tenant_consistency.sql`

- [ ] **Step 1: Write failing test**

In `tests/integration/migrations/migration_0153_test.go`:

```go
func TestMigration0153_TenantConsistencyTrigger(t *testing.T) {
    ctx := context.Background()
    db := testdb.Open(ctx, t)
    defer db.Close()

    // Insert document with tenant A
    docID, tenantA := testdb.InsertDraftDocument(ctx, t, db, "tenantA")

    // Try insert placeholder_values with tenant B — expect error
    _, err := db.ExecContext(ctx,
        `INSERT INTO document_placeholder_values(tenant_id,revision_id,placeholder_id,value_text,source,created_at,updated_at)
         VALUES ($1,$2,'ph1','x','user',NOW(),NOW())`,
        "tenantB", docID)
    if err == nil {
        t.Fatal("expected tenant mismatch error, got nil")
    }
    _ = tenantA
}
```

(`testdb.InsertDraftDocument` is an existing helper — if missing, see tests/integration/testdb/helpers.go for pattern and add an equivalent; do not guess signature, grep first.)

- [ ] **Step 2: Run — FAIL**

- [ ] **Step 3: Implement**

```sql
BEGIN;
CREATE OR REPLACE FUNCTION enforce_placeholder_value_tenant_consistent() RETURNS trigger AS $$
DECLARE doc_tenant TEXT;
BEGIN
    SELECT tenant_id INTO doc_tenant FROM public.documents WHERE id = NEW.revision_id;
    IF doc_tenant IS NULL THEN
        RAISE EXCEPTION 'document % not found', NEW.revision_id USING ERRCODE = 'foreign_key_violation';
    END IF;
    IF doc_tenant <> NEW.tenant_id THEN
        RAISE EXCEPTION 'tenant mismatch: document=% value=%', doc_tenant, NEW.tenant_id
            USING ERRCODE = 'check_violation';
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS enforce_placeholder_value_tenant_trg ON public.document_placeholder_values;
CREATE TRIGGER enforce_placeholder_value_tenant_trg
    BEFORE INSERT OR UPDATE ON public.document_placeholder_values
    FOR EACH ROW EXECUTE FUNCTION enforce_placeholder_value_tenant_consistent();

DROP TRIGGER IF EXISTS enforce_zone_content_tenant_trg ON public.document_editable_zone_content;
CREATE TRIGGER enforce_zone_content_tenant_trg
    BEFORE INSERT OR UPDATE ON public.document_editable_zone_content
    FOR EACH ROW EXECUTE FUNCTION enforce_placeholder_value_tenant_consistent();
COMMIT;
```

- [ ] **Step 4: Run — PASS**

- [ ] **Step 5: Commit**

```bash
rtk git add migrations/0153_placeholder_values_tenant_consistency.sql tests/integration/migrations/migration_0153_test.go
rtk git commit -m "feat(migrations): 0153 tenant-consistency triggers for fillin tables"
```

---

## Phase 3 — Template schema validation (Sonnet)

### Task 3.1: Duplicate ID + required-field validation reuse test baseline

**Files:**
- Modify: `internal/modules/templates_v2/application/schema.go`
- Test: `internal/modules/templates_v2/application/schema_test.go`

- [ ] **Step 1: Write failing test**

```go
func TestValidatePlaceholders_DuplicateID_Error(t *testing.T) {
    phs := []domain.Placeholder{
        {ID: "p1", Label: "A", Type: domain.PHText, Required: true},
        {ID: "p1", Label: "B", Type: domain.PHText, Required: true},
    }
    err := ValidatePlaceholders(phs)
    if !errors.Is(err, domain.ErrDuplicatePlaceholderID) {
        t.Fatalf("got %v, want ErrDuplicatePlaceholderID", err)
    }
}
```

- [ ] **Step 2: Run — FAIL if function unexported / missing**

Run: `rtk go test ./internal/modules/templates_v2/application/ -run TestValidatePlaceholders_DuplicateID_Error -v`
(If helper exists under different name — grep `UpdateSchemas` for current validation entrypoint; re-expose `ValidatePlaceholders` as package-level func. Do NOT guess name; open `schema.go` and locate.)

- [ ] **Step 3: Extract and export**

In `schema.go`, pull inline duplicate-check into:

```go
func ValidatePlaceholders(phs []domain.Placeholder) error {
    seen := map[string]struct{}{}
    for _, p := range phs {
        if p.ID == "" {
            return domain.ErrPlaceholderIDEmpty
        }
        if _, ok := seen[p.ID]; ok {
            return fmt.Errorf("%w: %s", domain.ErrDuplicatePlaceholderID, p.ID)
        }
        seen[p.ID] = struct{}{}
    }
    return nil
}
```

Add matching errors to `domain/errors.go`:

```go
var (
    ErrPlaceholderIDEmpty     = errors.New("placeholder id empty")
    ErrDuplicatePlaceholderID = errors.New("duplicate placeholder id")
)
```

- [ ] **Step 4: Run — PASS**

- [ ] **Step 5: Commit**

```bash
rtk git add internal/modules/templates_v2/
rtk git commit -m "refactor(templates_v2): extract ValidatePlaceholders + named errors"
```

### Task 3.2: Regex compile validation

- [ ] **Step 1: Test**

```go
func TestValidatePlaceholders_InvalidRegex_Error(t *testing.T) {
    bad := "["
    phs := []domain.Placeholder{{ID: "p1", Type: domain.PHText, Regex: &bad}}
    err := ValidatePlaceholders(phs)
    if !errors.Is(err, domain.ErrInvalidConstraint) {
        t.Fatalf("got %v, want ErrInvalidConstraint", err)
    }
}
```

- [ ] **Step 2: Run — FAIL**

- [ ] **Step 3: Implement**

Add to `ValidatePlaceholders`:

```go
if p.Regex != nil {
    if _, err := regexp.Compile(*p.Regex); err != nil {
        return fmt.Errorf("%w: placeholder %s regex: %s", domain.ErrInvalidConstraint, p.ID, err)
    }
}
```

And in errors: `ErrInvalidConstraint = errors.New("invalid constraint")`.

- [ ] **Step 4: Run — PASS**

- [ ] **Step 5: Commit**

```bash
rtk git commit -am "feat(templates_v2): validate regex constraint at schema save"
```

### Task 3.3: Numeric / date range sanity (min ≤ max)

- [ ] **Step 1: Test**

```go
func TestValidatePlaceholders_NumberRangeInverted_Error(t *testing.T) {
    mn, mx := 10.0, 5.0
    phs := []domain.Placeholder{{ID: "p1", Type: domain.PHNumber, MinNumber: &mn, MaxNumber: &mx}}
    if err := ValidatePlaceholders(phs); !errors.Is(err, domain.ErrInvalidConstraint) {
        t.Fatalf("got %v", err)
    }
}

func TestValidatePlaceholders_DateRangeInverted_Error(t *testing.T) {
    a, b := "2030-01-01", "2020-01-01"
    phs := []domain.Placeholder{{ID: "p1", Type: domain.PHDate, MinDate: &a, MaxDate: &b}}
    if err := ValidatePlaceholders(phs); !errors.Is(err, domain.ErrInvalidConstraint) {
        t.Fatalf("got %v", err)
    }
}
```

- [ ] **Step 2: Run — FAIL**

- [ ] **Step 3: Implement** — add to `ValidatePlaceholders`:

```go
if p.MinNumber != nil && p.MaxNumber != nil && *p.MinNumber > *p.MaxNumber {
    return fmt.Errorf("%w: placeholder %s min_number > max_number", domain.ErrInvalidConstraint, p.ID)
}
if p.MinDate != nil && p.MaxDate != nil && *p.MinDate > *p.MaxDate {
    return fmt.Errorf("%w: placeholder %s min_date > max_date", domain.ErrInvalidConstraint, p.ID)
}
if p.MaxLength != nil && *p.MaxLength <= 0 {
    return fmt.Errorf("%w: placeholder %s max_length must be > 0", domain.ErrInvalidConstraint, p.ID)
}
```

Date comparison uses ISO-8601 string ordering (both must be YYYY-MM-DD); additional format validation step:

```go
if p.MinDate != nil {
    if _, err := time.Parse("2006-01-02", *p.MinDate); err != nil {
        return fmt.Errorf("%w: placeholder %s min_date not YYYY-MM-DD", domain.ErrInvalidConstraint, p.ID)
    }
}
if p.MaxDate != nil {
    if _, err := time.Parse("2006-01-02", *p.MaxDate); err != nil {
        return fmt.Errorf("%w: placeholder %s max_date not YYYY-MM-DD", domain.ErrInvalidConstraint, p.ID)
    }
}
```

- [ ] **Step 4: Run — PASS**

- [ ] **Step 5: Commit**

```bash
rtk git commit -am "feat(templates_v2): numeric and date range sanity validation"
```

### Task 3.4: `visible_if` cycle detection (topological sort)

**Files:**
- Create: `internal/modules/templates_v2/application/visibility_graph.go`
- Test: `internal/modules/templates_v2/application/visibility_graph_test.go`

- [ ] **Step 1: Write failing tests**

```go
package application

import (
    "errors"
    "testing"

    "github.com/metaldocs/metaldocs-api/internal/modules/templates_v2/domain"
)

func TestVisibilityGraph_SimpleCycle(t *testing.T) {
    phs := []domain.Placeholder{
        {ID: "a", VisibleIf: &domain.VisibilityCondition{PlaceholderID: "b", Op: "eq", Value: 1}},
        {ID: "b", VisibleIf: &domain.VisibilityCondition{PlaceholderID: "a", Op: "eq", Value: 1}},
    }
    err := DetectVisibilityCycle(phs)
    if !errors.Is(err, domain.ErrPlaceholderCycle) {
        t.Fatalf("want ErrPlaceholderCycle, got %v", err)
    }
}

func TestVisibilityGraph_LongCycle(t *testing.T) {
    phs := []domain.Placeholder{
        {ID: "a", VisibleIf: &domain.VisibilityCondition{PlaceholderID: "b", Op: "eq", Value: 1}},
        {ID: "b", VisibleIf: &domain.VisibilityCondition{PlaceholderID: "c", Op: "eq", Value: 1}},
        {ID: "c", VisibleIf: &domain.VisibilityCondition{PlaceholderID: "a", Op: "eq", Value: 1}},
    }
    if err := DetectVisibilityCycle(phs); !errors.Is(err, domain.ErrPlaceholderCycle) {
        t.Fatalf("got %v", err)
    }
}

func TestVisibilityGraph_Acyclic_OK(t *testing.T) {
    phs := []domain.Placeholder{
        {ID: "a"},
        {ID: "b", VisibleIf: &domain.VisibilityCondition{PlaceholderID: "a", Op: "eq", Value: 1}},
        {ID: "c", VisibleIf: &domain.VisibilityCondition{PlaceholderID: "b", Op: "eq", Value: 1}},
    }
    if err := DetectVisibilityCycle(phs); err != nil {
        t.Fatalf("unexpected %v", err)
    }
}

func TestVisibilityGraph_UnknownReference(t *testing.T) {
    phs := []domain.Placeholder{
        {ID: "a", VisibleIf: &domain.VisibilityCondition{PlaceholderID: "ghost", Op: "eq", Value: 1}},
    }
    if err := DetectVisibilityCycle(phs); !errors.Is(err, domain.ErrInvalidConstraint) {
        t.Fatalf("got %v", err)
    }
}
```

Add `ErrPlaceholderCycle = errors.New("placeholder visibility cycle")` to `domain/errors.go`.

- [ ] **Step 2: Run — FAIL**

- [ ] **Step 3: Implement**

```go
package application

import (
    "fmt"

    "github.com/metaldocs/metaldocs-api/internal/modules/templates_v2/domain"
)

// DetectVisibilityCycle walks the visible_if dependency graph using DFS with three-color marks
// and returns ErrPlaceholderCycle on back-edge detection, ErrInvalidConstraint on dangling ref.
func DetectVisibilityCycle(phs []domain.Placeholder) error {
    idx := make(map[string]*domain.Placeholder, len(phs))
    for i := range phs {
        idx[phs[i].ID] = &phs[i]
    }
    const (
        white = 0
        gray  = 1
        black = 2
    )
    color := make(map[string]int, len(phs))

    var visit func(id string, stack []string) error
    visit = func(id string, stack []string) error {
        if color[id] == gray {
            return fmt.Errorf("%w: %v", domain.ErrPlaceholderCycle, append(stack, id))
        }
        if color[id] == black {
            return nil
        }
        color[id] = gray
        if p, ok := idx[id]; ok && p.VisibleIf != nil {
            dep := p.VisibleIf.PlaceholderID
            if _, exists := idx[dep]; !exists {
                return fmt.Errorf("%w: placeholder %s visible_if references unknown %s",
                    domain.ErrInvalidConstraint, id, dep)
            }
            if err := visit(dep, append(stack, id)); err != nil {
                return err
            }
        }
        color[id] = black
        return nil
    }
    for _, p := range phs {
        if err := visit(p.ID, nil); err != nil {
            return err
        }
    }
    return nil
}
```

Wire into `ValidatePlaceholders` tail: `if err := DetectVisibilityCycle(phs); err != nil { return err }`.

- [ ] **Step 4: Run — PASS**

- [ ] **Step 5: Commit**

```bash
rtk git add internal/modules/templates_v2/
rtk git commit -m "feat(templates_v2): visible_if cycle detection with three-color DFS"
```

### Task 3.5: Computed placeholder requires `resolver_key`

- [ ] **Step 1: Test**

```go
func TestValidatePlaceholders_ComputedRequiresResolverKey(t *testing.T) {
    phs := []domain.Placeholder{{ID: "p1", Type: domain.PHComputed, Computed: true}}
    if err := ValidatePlaceholders(phs); !errors.Is(err, domain.ErrInvalidConstraint) {
        t.Fatalf("got %v", err)
    }
}
```

- [ ] **Step 2: Run — FAIL**

- [ ] **Step 3: Implement**

In `ValidatePlaceholders`, per placeholder:

```go
if p.Computed && (p.ResolverKey == nil || *p.ResolverKey == "") {
    return fmt.Errorf("%w: placeholder %s computed but resolver_key empty",
        domain.ErrInvalidConstraint, p.ID)
}
```

- [ ] **Step 4: Run — PASS**

- [ ] **Step 5: Commit**

```bash
rtk git commit -am "feat(templates_v2): computed placeholder requires resolver_key"
```

### Task 3.6: Unknown resolver key rejected at template save

- [ ] **Step 1: Test** — at template save time, unknown resolver_key must fail:

```go
func TestUpdateSchemas_UnknownResolverKey_Error(t *testing.T) {
    // build Service with stub ResolverRegistry returning knownKeys=["doc_code"]
    svc := newTestService(t, WithKnownResolvers("doc_code"))
    _, err := svc.UpdateSchemas(ctx, updateCmdWithComputed("p1", "missing_resolver"))
    if !errors.Is(err, domain.ErrUnknownResolver) {
        t.Fatalf("got %v", err)
    }
}
```

(`newTestService`, `WithKnownResolvers`, `updateCmdWithComputed` are test helpers — grep existing `fakes_test.go` patterns and add.)

- [ ] **Step 2: Run — FAIL**

- [ ] **Step 3: Implement**

Add port to `ports.go`:

```go
type ResolverRegistryReader interface {
    Known() map[string]int // key → version
}
```

Inject into `Service` struct and use in validation:

```go
for _, p := range phs {
    if p.ResolverKey != nil && *p.ResolverKey != "" {
        if _, ok := s.resolvers.Known()[*p.ResolverKey]; !ok {
            return fmt.Errorf("%w: placeholder %s resolver %s", domain.ErrUnknownResolver, p.ID, *p.ResolverKey)
        }
    }
}
```

Add `ErrUnknownResolver` to `domain/errors.go`.

- [ ] **Step 4: Run — PASS**

- [ ] **Step 5: Commit**

```bash
rtk git add internal/modules/templates_v2/
rtk git commit -m "feat(templates_v2): reject unknown resolver_key at template save"
```

---

## Phase 4 — Revision snapshot on create (Sonnet)

### Task 4.1: `TemplateSnapshot` value object + hash computation

**Files:**
- Create: `internal/modules/documents_v2/domain/snapshot.go`
- Test: `internal/modules/documents_v2/domain/snapshot_test.go`

- [ ] **Step 1: Write failing test**

```go
package domain

import (
    "encoding/hex"
    "testing"
)

func TestTemplateSnapshot_StableHash(t *testing.T) {
    s1 := TemplateSnapshot{
        PlaceholderSchemaJSON: []byte(`{"placeholders":[{"id":"a","type":"text"}]}`),
        CompositionJSON:       []byte(`{"header_sub_blocks":["h1"]}`),
        ZonesSchemaJSON:       []byte(`{"zones":[{"id":"z1"}]}`),
        BodyDocxBytes:         []byte("DOCXBYTES"),
    }
    h1 := s1.Hashes()
    h2 := s1.Hashes()
    if hex.EncodeToString(h1.PlaceholderSchemaHash) != hex.EncodeToString(h2.PlaceholderSchemaHash) {
        t.Fatal("hash not deterministic")
    }
    if len(h1.BodyDocxHash) != 32 {
        t.Fatalf("want 32-byte sha256, got %d", len(h1.BodyDocxHash))
    }
}
```

- [ ] **Step 2: Run — FAIL**

- [ ] **Step 3: Implement**

```go
package domain

import "crypto/sha256"

type TemplateSnapshot struct {
    PlaceholderSchemaJSON []byte
    CompositionJSON       []byte
    ZonesSchemaJSON       []byte
    BodyDocxBytes         []byte
    BodyDocxS3Key         string
}

type SnapshotHashes struct {
    PlaceholderSchemaHash []byte
    CompositionHash       []byte
    BodyDocxHash          []byte
}

func (s TemplateSnapshot) Hashes() SnapshotHashes {
    ph := sha256.Sum256(s.PlaceholderSchemaJSON)
    ch := sha256.Sum256(s.CompositionJSON)
    bh := sha256.Sum256(s.BodyDocxBytes)
    return SnapshotHashes{ph[:], ch[:], bh[:]}
}
```

- [ ] **Step 4: Run — PASS**

- [ ] **Step 5: Commit**

```bash
rtk git add internal/modules/documents_v2/domain/
rtk git commit -m "feat(documents_v2): TemplateSnapshot with deterministic hashes"
```

### Task 4.2: `SnapshotRepository` — write snapshot columns

**Files:**
- Create: `internal/modules/documents_v2/repository/snapshot_repository.go`
- Test: `internal/modules/documents_v2/repository/snapshot_repository_test.go`

- [ ] **Step 1: Write failing integration test**

```go
func TestSnapshotRepository_WriteAndRead(t *testing.T) {
    ctx := context.Background()
    db := testdb.Open(ctx, t)
    defer db.Close()
    docID, tenant := testdb.InsertDraftDocument(ctx, t, db, "tenant1")

    repo := repository.NewSnapshotRepository(db)
    snap := domain.TemplateSnapshot{
        PlaceholderSchemaJSON: []byte(`{}`),
        CompositionJSON:       []byte(`{}`),
        ZonesSchemaJSON:       []byte(`{}`),
        BodyDocxBytes:         []byte("x"),
        BodyDocxS3Key:         "s3://bucket/key",
    }
    if err := repo.WriteSnapshot(ctx, tenant, docID, snap); err != nil {
        t.Fatal(err)
    }
    got, err := repo.ReadSnapshot(ctx, tenant, docID)
    if err != nil {
        t.Fatal(err)
    }
    if got.BodyDocxS3Key != "s3://bucket/key" {
        t.Fatalf("got %+v", got)
    }
}
```

- [ ] **Step 2: Run — FAIL**

- [ ] **Step 3: Implement**

```go
package repository

import (
    "context"
    "database/sql"

    "github.com/metaldocs/metaldocs-api/internal/modules/documents_v2/domain"
)

type SnapshotRepository struct{ db *sql.DB }

func NewSnapshotRepository(db *sql.DB) *SnapshotRepository { return &SnapshotRepository{db} }

func (r *SnapshotRepository) WriteSnapshot(ctx context.Context, tenant, docID string, s domain.TemplateSnapshot) error {
    h := s.Hashes()
    _, err := r.db.ExecContext(ctx, `
        UPDATE public.documents
           SET placeholder_schema_snapshot   = $1,
               placeholder_schema_hash       = $2,
               composition_config_snapshot   = $3,
               composition_config_hash       = $4,
               editable_zones_schema_snapshot= $5,
               body_docx_snapshot_s3_key     = $6,
               body_docx_hash                = $7
         WHERE tenant_id=$8 AND id=$9`,
        s.PlaceholderSchemaJSON, h.PlaceholderSchemaHash,
        s.CompositionJSON, h.CompositionHash,
        s.ZonesSchemaJSON,
        s.BodyDocxS3Key, h.BodyDocxHash,
        tenant, docID,
    )
    return err
}

func (r *SnapshotRepository) ReadSnapshot(ctx context.Context, tenant, docID string) (domain.TemplateSnapshot, error) {
    var s domain.TemplateSnapshot
    err := r.db.QueryRowContext(ctx, `
        SELECT placeholder_schema_snapshot, composition_config_snapshot,
               editable_zones_schema_snapshot, body_docx_snapshot_s3_key
          FROM public.documents WHERE tenant_id=$1 AND id=$2`,
        tenant, docID,
    ).Scan(&s.PlaceholderSchemaJSON, &s.CompositionJSON, &s.ZonesSchemaJSON, &s.BodyDocxS3Key)
    return s, err
}
```

- [ ] **Step 4: Run — PASS**

- [ ] **Step 5: Commit**

```bash
rtk git add internal/modules/documents_v2/repository/
rtk git commit -m "feat(documents_v2): snapshot repository write/read"
```

### Task 4.3: `SnapshotService` — called at revision create

**Files:**
- Create: `internal/modules/documents_v2/application/snapshot_service.go`
- Test: `internal/modules/documents_v2/application/snapshot_service_test.go`

- [ ] **Step 1: Write failing test** — given a fake TemplateReader returning a template with placeholder/zone/composition schemas and body DOCX, when SnapshotService.SnapshotFromTemplate is invoked for a new revision, it writes all snapshot columns. Assert by calling `SnapshotRepository.ReadSnapshot` after and checking bytes equal template source bytes.

```go
func TestSnapshotService_CopiesTemplateToRevision(t *testing.T) {
    ctx := context.Background()
    db := testdb.Open(ctx, t)
    defer db.Close()
    docID, tenant := testdb.InsertDraftDocument(ctx, t, db, "tenant1")
    tmpl := fakeTemplate{
        PlaceholderSchema: []byte(`{"placeholders":[]}`),
        Composition:       []byte(`{"header_sub_blocks":[]}`),
        Zones:             []byte(`{"zones":[]}`),
        BodyDocx:          []byte("DOCX"),
        BodyDocxS3Key:     "s3://t/k",
    }
    svc := NewSnapshotService(fakeTemplateReader{tmpl}, repository.NewSnapshotRepository(db))
    if err := svc.SnapshotFromTemplate(ctx, tenant, docID, "tmpl-1"); err != nil {
        t.Fatal(err)
    }
    got, _ := repository.NewSnapshotRepository(db).ReadSnapshot(ctx, tenant, docID)
    if string(got.PlaceholderSchemaJSON) != `{"placeholders":[]}` {
        t.Fatalf("mismatch: %s", got.PlaceholderSchemaJSON)
    }
}
```

- [ ] **Step 2: Run — FAIL**

- [ ] **Step 3: Implement**

```go
package application

import (
    "context"

    "github.com/metaldocs/metaldocs-api/internal/modules/documents_v2/domain"
)

type TemplateReader interface {
    LoadForSnapshot(ctx context.Context, tenantID, templateID string) (domain.TemplateSnapshot, error)
}

type SnapshotWriter interface {
    WriteSnapshot(ctx context.Context, tenantID, revisionID string, s domain.TemplateSnapshot) error
}

type SnapshotService struct {
    templates TemplateReader
    writer    SnapshotWriter
}

func NewSnapshotService(t TemplateReader, w SnapshotWriter) *SnapshotService {
    return &SnapshotService{t, w}
}

func (s *SnapshotService) SnapshotFromTemplate(ctx context.Context, tenantID, revisionID, templateID string) error {
    snap, err := s.templates.LoadForSnapshot(ctx, tenantID, templateID)
    if err != nil {
        return err
    }
    return s.writer.WriteSnapshot(ctx, tenantID, revisionID, snap)
}
```

- [ ] **Step 4: Run — PASS**

- [ ] **Step 5: Commit**

```bash
rtk git add internal/modules/documents_v2/application/
rtk git commit -m "feat(documents_v2): SnapshotService copies template to revision at create"
```

### Task 4.4: Wire SnapshotService into revision create path

- [ ] **Step 1: Find existing create entrypoint** — `rtk grep -n "func.*Create.*Revision\|CreateDocument\|NewRevision" internal/modules/documents_v2/application/`.
- [ ] **Step 2: Write failing integration test** exercising `document_v2.Service.CreateRevisionFromTemplate(tenant, templateID)` then asserting snapshot columns populated.
- [ ] **Step 3: Call `SnapshotService.SnapshotFromTemplate` inside the create tx, same transaction as the `INSERT INTO documents`. Add pass-through constructor arg.
- [ ] **Step 4: Run full package — PASS.**
- [ ] **Step 5: Commit**

```bash
rtk git commit -am "feat(documents_v2): snapshot template at revision create"
```

### Task 4.5: Empty value rows seeded for required placeholders at create

- [ ] **Step 1: Write failing test** — after create, `document_placeholder_values` has one row per required placeholder with `source='default'`, `value_text IS NULL`.
- [ ] **Step 2: FAIL**
- [ ] **Step 3: Extend `SnapshotService` to accept a `PlaceholderValueSeeder` port and call `Seed(tenant, revisionID, []Placeholder)`**. Implementation inserts one row per required placeholder:

```go
func (r *FillInRepository) SeedDefaults(ctx context.Context, tenant, revisionID string, phs []domain.Placeholder) error {
    tx, err := r.db.BeginTx(ctx, nil)
    if err != nil { return err }
    defer tx.Rollback()
    for _, p := range phs {
        if !p.Required { continue }
        if _, err := tx.ExecContext(ctx, `
            INSERT INTO document_placeholder_values(tenant_id,revision_id,placeholder_id,source,created_at,updated_at)
            VALUES ($1,$2,$3,'default',NOW(),NOW())
            ON CONFLICT DO NOTHING`,
            tenant, revisionID, p.ID); err != nil {
            return err
        }
    }
    return tx.Commit()
}
```

- [ ] **Step 4: PASS**
- [ ] **Step 5: Commit**

```bash
rtk git commit -am "feat(documents_v2): seed default placeholder rows at revision create"
```

---

## Phase 5 — Fill-in repository + HTTP (Haiku for handlers, Codex for repo)

### Task 5.1: `FillInRepository.UpsertValue`

**Files:**
- Create: `internal/modules/documents_v2/repository/fillin_repository.go`
- Test: `internal/modules/documents_v2/repository/fillin_repository_test.go`

- [ ] **Step 1: Test** — insert then update the same (tenant, revision, placeholder) row; assert second read returns updated value and `updated_at > created_at`.
- [ ] **Step 2: FAIL**
- [ ] **Step 3: Implement**

```go
package repository

import (
    "context"
    "database/sql"
    "encoding/json"
)

type FillInRepository struct{ db *sql.DB }

func NewFillInRepository(db *sql.DB) *FillInRepository { return &FillInRepository{db} }

type PlaceholderValue struct {
    TenantID, RevisionID, PlaceholderID string
    ValueText   *string
    ValueTyped  map[string]any
    Source      string // "user" | "computed" | "default"
    ComputedFrom    *string
    ResolverVersion *int
    InputsHash      []byte
}

func (r *FillInRepository) UpsertValue(ctx context.Context, v PlaceholderValue) error {
    var typedJSON []byte
    if v.ValueTyped != nil {
        typedJSON, _ = json.Marshal(v.ValueTyped)
    }
    _, err := r.db.ExecContext(ctx, `
        INSERT INTO document_placeholder_values
            (tenant_id, revision_id, placeholder_id, value_text, value_typed,
             source, computed_from, resolver_version, inputs_hash,
             validated_at, created_at, updated_at)
        VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,NOW(),NOW(),NOW())
        ON CONFLICT (tenant_id, revision_id, placeholder_id) DO UPDATE SET
            value_text       = EXCLUDED.value_text,
            value_typed      = EXCLUDED.value_typed,
            source           = EXCLUDED.source,
            computed_from    = EXCLUDED.computed_from,
            resolver_version = EXCLUDED.resolver_version,
            inputs_hash      = EXCLUDED.inputs_hash,
            validated_at     = NOW(),
            updated_at       = NOW()`,
        v.TenantID, v.RevisionID, v.PlaceholderID, v.ValueText, typedJSON,
        v.Source, v.ComputedFrom, v.ResolverVersion, v.InputsHash,
    )
    return err
}

func (r *FillInRepository) ListValues(ctx context.Context, tenantID, revisionID string) ([]PlaceholderValue, error) {
    rows, err := r.db.QueryContext(ctx, `
        SELECT placeholder_id, value_text, value_typed, source, computed_from,
               resolver_version, inputs_hash
          FROM document_placeholder_values
         WHERE tenant_id=$1 AND revision_id=$2`,
        tenantID, revisionID)
    if err != nil { return nil, err }
    defer rows.Close()
    var out []PlaceholderValue
    for rows.Next() {
        var v PlaceholderValue
        var typedJSON []byte
        if err := rows.Scan(&v.PlaceholderID, &v.ValueText, &typedJSON,
            &v.Source, &v.ComputedFrom, &v.ResolverVersion, &v.InputsHash); err != nil {
            return nil, err
        }
        if len(typedJSON) > 0 {
            _ = json.Unmarshal(typedJSON, &v.ValueTyped)
        }
        v.TenantID, v.RevisionID = tenantID, revisionID
        out = append(out, v)
    }
    return out, rows.Err()
}
```

- [ ] **Step 4: PASS**
- [ ] **Step 5: Commit**

```bash
rtk git commit -am "feat(documents_v2): FillInRepository upsert and list"
```

### Task 5.2: Zone content CRUD on same repo

- [ ] **Step 1: Test** — upsert zone content twice with different OOXML, second read returns latest and `content_hash` changes.
- [ ] **Step 2: FAIL**
- [ ] **Step 3: Implement**

```go
type ZoneContent struct {
    TenantID, RevisionID, ZoneID string
    ContentOOXML string
    ContentHash  []byte
}

func (r *FillInRepository) UpsertZoneContent(ctx context.Context, z ZoneContent) error {
    h := sha256.Sum256([]byte(z.ContentOOXML))
    z.ContentHash = h[:]
    _, err := r.db.ExecContext(ctx, `
        INSERT INTO document_editable_zone_content
            (tenant_id, revision_id, zone_id, content_ooxml, content_hash, created_at, updated_at)
        VALUES ($1,$2,$3,$4,$5,NOW(),NOW())
        ON CONFLICT (tenant_id, revision_id, zone_id) DO UPDATE SET
            content_ooxml = EXCLUDED.content_ooxml,
            content_hash  = EXCLUDED.content_hash,
            updated_at    = NOW()`,
        z.TenantID, z.RevisionID, z.ZoneID, z.ContentOOXML, z.ContentHash,
    )
    return err
}

func (r *FillInRepository) ListZoneContent(ctx context.Context, tenantID, revisionID string) ([]ZoneContent, error) {
    rows, err := r.db.QueryContext(ctx, `
        SELECT zone_id, content_ooxml, content_hash
          FROM document_editable_zone_content
         WHERE tenant_id=$1 AND revision_id=$2`,
        tenantID, revisionID)
    if err != nil { return nil, err }
    defer rows.Close()
    var out []ZoneContent
    for rows.Next() {
        var z ZoneContent
        if err := rows.Scan(&z.ZoneID, &z.ContentOOXML, &z.ContentHash); err != nil { return nil, err }
        z.TenantID, z.RevisionID = tenantID, revisionID
        out = append(out, z)
    }
    return out, rows.Err()
}
```

Add `import "crypto/sha256"` at top.

- [ ] **Step 4: PASS**
- [ ] **Step 5: Commit**

```bash
rtk git commit -am "feat(documents_v2): zone content upsert with content hash"
```

### Task 5.3: `FillInService.SetPlaceholderValue` — validates against snapshot schema

**Files:**
- Create: `internal/modules/documents_v2/application/fillin_service.go`
- Test: `internal/modules/documents_v2/application/fillin_service_test.go`

- [ ] **Step 1: Test**

```go
func TestFillInService_ValueRejectedIfFailsRegex(t *testing.T) {
    re := "^[A-Z]{3}$"
    schema := []domain.Placeholder{{ID: "p1", Type: domain.PHText, Regex: &re}}
    svc := NewFillInService(fakeSnapshotReader{schema: schema}, fakeFillInWriter{})
    err := svc.SetPlaceholderValue(ctx, "tenant", "rev", "p1", "abc")
    if !errors.Is(err, domain.ErrValidationFailed) {
        t.Fatalf("got %v", err)
    }
}
func TestFillInService_ValueAcceptedIfMatches(t *testing.T) {
    re := "^[A-Z]{3}$"
    schema := []domain.Placeholder{{ID: "p1", Type: domain.PHText, Regex: &re}}
    writer := &fakeFillInWriter{}
    svc := NewFillInService(fakeSnapshotReader{schema: schema}, writer)
    if err := svc.SetPlaceholderValue(ctx, "tenant", "rev", "p1", "ABC"); err != nil {
        t.Fatal(err)
    }
    if len(writer.Upserts) != 1 || *writer.Upserts[0].ValueText != "ABC" {
        t.Fatalf("bad upsert: %+v", writer.Upserts)
    }
}
```

Add `ErrValidationFailed = errors.New("placeholder value validation failed")` to documents_v2 domain errors.

- [ ] **Step 2: FAIL**
- [ ] **Step 3: Implement**

```go
package application

import (
    "context"
    "fmt"
    "regexp"
    "strconv"
    "time"

    v2domain "github.com/metaldocs/metaldocs-api/internal/modules/documents_v2/domain"
    "github.com/metaldocs/metaldocs-api/internal/modules/documents_v2/repository"
    tmpldom "github.com/metaldocs/metaldocs-api/internal/modules/templates_v2/domain"
)

type SchemaReader interface {
    LoadPlaceholderSchema(ctx context.Context, tenantID, revisionID string) ([]tmpldom.Placeholder, error)
    LoadZonesSchema(ctx context.Context, tenantID, revisionID string) ([]tmpldom.EditableZone, error)
}

type FillInWriter interface {
    UpsertValue(ctx context.Context, v repository.PlaceholderValue) error
    UpsertZoneContent(ctx context.Context, z repository.ZoneContent) error
}

type FillInService struct {
    schemas SchemaReader
    writer  FillInWriter
}

func NewFillInService(s SchemaReader, w FillInWriter) *FillInService {
    return &FillInService{s, w}
}

func (s *FillInService) SetPlaceholderValue(ctx context.Context, tenantID, revisionID, placeholderID, raw string) error {
    schema, err := s.schemas.LoadPlaceholderSchema(ctx, tenantID, revisionID)
    if err != nil { return err }
    ph, ok := findPlaceholder(schema, placeholderID)
    if !ok {
        return fmt.Errorf("%w: unknown placeholder %s", v2domain.ErrValidationFailed, placeholderID)
    }
    if err := validateValue(ph, raw); err != nil {
        return err
    }
    val := raw
    return s.writer.UpsertValue(ctx, repository.PlaceholderValue{
        TenantID:      tenantID,
        RevisionID:    revisionID,
        PlaceholderID: placeholderID,
        ValueText:     &val,
        Source:        "user",
    })
}

func findPlaceholder(phs []tmpldom.Placeholder, id string) (tmpldom.Placeholder, bool) {
    for _, p := range phs { if p.ID == id { return p, true } }
    return tmpldom.Placeholder{}, false
}

func validateValue(p tmpldom.Placeholder, raw string) error {
    if p.Required && raw == "" {
        return fmt.Errorf("%w: %s required", v2domain.ErrValidationFailed, p.ID)
    }
    if p.MaxLength != nil && len(raw) > *p.MaxLength {
        return fmt.Errorf("%w: %s max_length exceeded", v2domain.ErrValidationFailed, p.ID)
    }
    if p.Regex != nil {
        re, err := regexp.Compile(*p.Regex)
        if err != nil { return err }
        if !re.MatchString(raw) {
            return fmt.Errorf("%w: %s regex mismatch", v2domain.ErrValidationFailed, p.ID)
        }
    }
    switch p.Type {
    case tmpldom.PHNumber:
        n, err := strconv.ParseFloat(raw, 64)
        if err != nil { return fmt.Errorf("%w: %s not a number", v2domain.ErrValidationFailed, p.ID) }
        if p.MinNumber != nil && n < *p.MinNumber {
            return fmt.Errorf("%w: %s < min_number", v2domain.ErrValidationFailed, p.ID)
        }
        if p.MaxNumber != nil && n > *p.MaxNumber {
            return fmt.Errorf("%w: %s > max_number", v2domain.ErrValidationFailed, p.ID)
        }
    case tmpldom.PHDate:
        _, err := time.Parse("2006-01-02", raw)
        if err != nil { return fmt.Errorf("%w: %s not YYYY-MM-DD", v2domain.ErrValidationFailed, p.ID) }
        if p.MinDate != nil && raw < *p.MinDate {
            return fmt.Errorf("%w: %s < min_date", v2domain.ErrValidationFailed, p.ID)
        }
        if p.MaxDate != nil && raw > *p.MaxDate {
            return fmt.Errorf("%w: %s > max_date", v2domain.ErrValidationFailed, p.ID)
        }
    case tmpldom.PHSelect:
        found := false
        for _, opt := range p.Options { if opt == raw { found = true; break } }
        if !found {
            return fmt.Errorf("%w: %s not in options", v2domain.ErrValidationFailed, p.ID)
        }
    }
    return nil
}
```

- [ ] **Step 4: PASS**
- [ ] **Step 5: Commit**

```bash
rtk git commit -am "feat(documents_v2): FillInService validates values against snapshot schema"
```

### Task 5.4: `FillInService.SetZoneContent` — validate ContentPolicy

- [ ] **Step 1: Test** — zone marked `AllowTables=false` receiving OOXML with `<w:tbl>` element rejected; OOXML without tables accepted.
- [ ] **Step 2: FAIL**
- [ ] **Step 3: Implement** — simple substring/regex checks on OOXML for `<w:tbl`, `<w:drawing`, `<w:pStyle w:val="Heading`, `<w:numPr`. This is acknowledged as a best-effort coarse gate before freeze; deep OOXML validation is out-of-scope.

```go
func (s *FillInService) SetZoneContent(ctx context.Context, tenantID, revisionID, zoneID, ooxml string) error {
    zones, err := s.schemas.LoadZonesSchema(ctx, tenantID, revisionID)
    if err != nil { return err }
    var zone tmpldom.EditableZone
    found := false
    for _, z := range zones { if z.ID == zoneID { zone = z; found = true; break } }
    if !found {
        return fmt.Errorf("%w: unknown zone %s", v2domain.ErrValidationFailed, zoneID)
    }
    if zone.MaxLength != nil && len(ooxml) > *zone.MaxLength {
        return fmt.Errorf("%w: zone %s exceeds max_length", v2domain.ErrValidationFailed, zoneID)
    }
    if !zone.ContentPolicy.AllowTables && strings.Contains(ooxml, "<w:tbl") {
        return fmt.Errorf("%w: zone %s disallows tables", v2domain.ErrValidationFailed, zoneID)
    }
    if !zone.ContentPolicy.AllowImages && strings.Contains(ooxml, "<w:drawing") {
        return fmt.Errorf("%w: zone %s disallows images", v2domain.ErrValidationFailed, zoneID)
    }
    if !zone.ContentPolicy.AllowHeadings && strings.Contains(ooxml, `<w:pStyle w:val="Heading`) {
        return fmt.Errorf("%w: zone %s disallows headings", v2domain.ErrValidationFailed, zoneID)
    }
    if !zone.ContentPolicy.AllowLists && strings.Contains(ooxml, "<w:numPr") {
        return fmt.Errorf("%w: zone %s disallows lists", v2domain.ErrValidationFailed, zoneID)
    }
    return s.writer.UpsertZoneContent(ctx, repository.ZoneContent{
        TenantID: tenantID, RevisionID: revisionID, ZoneID: zoneID, ContentOOXML: ooxml,
    })
}
```

Add `import "strings"`.

- [ ] **Step 4: PASS**
- [ ] **Step 5: Commit**

```bash
rtk git commit -am "feat(documents_v2): zone content policy enforcement"
```

### Task 5.5: HTTP handler — `PUT /api/v2/documents/{id}/placeholders/{pid}`

**Files:**
- Create: `internal/modules/documents_v2/http/fillin_handler.go`
- Test: `internal/modules/documents_v2/http/fillin_handler_test.go`

- [ ] **Step 1: Test** — POST JSON `{"value":"ABC"}`, authorized user, existing revision in `draft`; assert 200 + row exists.

- [ ] **Step 2: FAIL**

- [ ] **Step 3: Implement** — follow exact pattern of `internal/modules/documents_v2/approval/http/submit_handler.go`. Route: `PUT /api/v2/documents/{id}/placeholders/{pid}` with chi `URLParam`. Requires capability `doc.edit_draft` (add to `metaldocs.role_capabilities` seed in Task 5.7). Returns 200 with `{"placeholder_id":"pid","updated_at":"..."}`. 403 on missing capability, 404 if revision not found, 409 if revision status not in `{draft}`, 422 on validation failure.

- [ ] **Step 4: PASS**
- [ ] **Step 5: Commit**

```bash
rtk git commit -am "feat(documents_v2): PUT placeholder value endpoint"
```

### Task 5.6: HTTP handler — `PUT /api/v2/documents/{id}/zones/{zid}`

Same shape as 5.5. Body `{"content_ooxml": "<w:p>...</w:p>"}`. Commit:

```bash
rtk git commit -am "feat(documents_v2): PUT zone content endpoint"
```

### Task 5.7: Capability seed — add `doc.edit_draft`

- [ ] **Step 1: Test** — after applying 0154, `metaldocs.role_capabilities` contains `(doc.edit_draft, author)` and `(doc.edit_draft, qms_admin)`.
- [ ] **Step 2: FAIL**
- [ ] **Step 3: Migration `0154_capability_doc_edit_draft.sql`:**

```sql
BEGIN;
INSERT INTO metaldocs.role_capabilities(capability, role)
VALUES ('doc.edit_draft','author'),('doc.edit_draft','qms_admin')
ON CONFLICT DO NOTHING;
COMMIT;
```

- [ ] **Step 4: PASS**
- [ ] **Step 5: Commit**

```bash
rtk git commit -am "feat(migrations): 0154 doc.edit_draft capability"
```

---

## Phase 6 — Zone round-trip verification (Sonnet)

(zone repo already landed in Phase 5; this phase hardens boundary preservation)

### Task 6.1: Zone bookmark boundary preservation test

- [ ] **Step 1: Failing test** — build a DOCX with two zones, run through `extractZones` (existing frontend helper), assert pairs come back with same IDs, same ordering, content between preserved:

```ts
// frontend/apps/web/src/editor-adapters/__tests__/zone-round-trip.test.ts
import { wrapZone, extractZones, type EditableZone } from "../eigenpal-template-mode";

test("zone round-trip preserves boundaries and content", () => {
  const zones: EditableZone[] = [
    { id: "intro", label: "Intro" },
    { id: "body",  label: "Body"  },
  ];
  const doc = wrapZone(/* build minimal blocks per helper's API */);
  const extracted = extractZones(doc);
  expect(extracted.map(z => z.id)).toEqual(["intro", "body"]);
});
```

(Read current `eigenpal-template-mode.ts` helper signatures before finalizing test shape. Do not guess argument names.)

- [ ] **Step 2: FAIL only if helper drops IDs — otherwise PASS; if PASS, still keep as regression test.**
- [ ] **Step 3: If FAIL, fix `wrapZone`/`extractZones` to preserve IDs.**
- [ ] **Step 4: PASS**
- [ ] **Step 5: Commit**

```bash
rtk git commit -am "test(editor-adapters): zone round-trip regression coverage"
```

---

## Phase 7 — Computed resolver registry + 7 v1 resolvers (Codex medium)

### Task 7.1: Resolver interface + port types

**Files:**
- Create: `internal/modules/render/resolvers/resolver.go`
- Test: `internal/modules/render/resolvers/resolver_test.go`

- [ ] **Step 1: Failing compile test** for interface adherence + `Resolve` contract.

```go
package resolvers

import (
    "context"
    "testing"
    "time"
)

type fixedResolver struct{}

func (fixedResolver) Key() string { return "doc_code" }
func (fixedResolver) Version() int { return 1 }
func (fixedResolver) Resolve(ctx context.Context, in ResolveInput) (ResolvedValue, error) {
    return ResolvedValue{
        Value: "QMS-0001", ResolverKey: "doc_code", ResolverVer: 1,
        InputsHash: []byte("abc"), ComputedAt: time.Unix(0, 0).UTC(),
    }, nil
}

func TestResolver_InterfaceShape(t *testing.T) {
    var r ComputedResolver = fixedResolver{}
    v, err := r.Resolve(context.Background(), ResolveInput{})
    if err != nil { t.Fatal(err) }
    if v.ResolverKey != "doc_code" { t.Fatalf("got %s", v.ResolverKey) }
}
```

- [ ] **Step 2: FAIL**
- [ ] **Step 3: Implement**

```go
package resolvers

import (
    "context"
    "time"

    "github.com/metaldocs/metaldocs-api/internal/modules/registry"
    "github.com/metaldocs/metaldocs-api/internal/modules/taxonomy"
    "github.com/metaldocs/metaldocs-api/internal/modules/workflow"
    v2docs "github.com/metaldocs/metaldocs-api/internal/modules/documents_v2"
)

type ResolveInput struct {
    TenantID, RevisionID, ControlledDocumentID string
    ProfileCodeSnapshot, AreaCodeSnapshot      string
    RegistryReader  registry.Reader
    RevisionReader  v2docs.RevisionReader
    WorkflowReader  workflow.Reader
    TaxonomyReader  taxonomy.Reader
}

type ResolvedValue struct {
    Value       any
    ResolverKey string
    ResolverVer int
    InputsHash  []byte
    ComputedAt  time.Time
}

type ComputedResolver interface {
    Key() string
    Version() int
    Resolve(ctx context.Context, in ResolveInput) (ResolvedValue, error)
}
```

(If imported packages lack the Reader interfaces, add minimal interfaces in this file consuming concrete readers — Codex reviews which pattern already exists in Spec 1/2; do not invent.)

- [ ] **Step 4: PASS**
- [ ] **Step 5: Commit**

```bash
rtk git commit -am "feat(render): ComputedResolver interface + ResolveInput/Value types"
```

### Task 7.2: `Registry` map + `Known()` for template validation

- [ ] **Step 1: Test**

```go
func TestRegistry_Known_ReturnsAllResolvers(t *testing.T) {
    reg := NewRegistry()
    reg.Register(fixedResolver{})
    known := reg.Known()
    if known["doc_code"] != 1 {
        t.Fatalf("got %v", known)
    }
}
```

- [ ] **Step 2: FAIL**
- [ ] **Step 3: Implement**

```go
type Registry struct {
    items map[string]ComputedResolver
}

func NewRegistry() *Registry { return &Registry{items: map[string]ComputedResolver{}} }

func (r *Registry) Register(cr ComputedResolver) {
    r.items[cr.Key()] = cr
}

func (r *Registry) Get(key string) (ComputedResolver, bool) {
    cr, ok := r.items[key]; return cr, ok
}

func (r *Registry) Known() map[string]int {
    out := make(map[string]int, len(r.items))
    for k, v := range r.items { out[k] = v.Version() }
    return out
}
```

- [ ] **Step 4: PASS**
- [ ] **Step 5: Commit**

```bash
rtk git commit -am "feat(render): resolver Registry with Known() version map"
```

### Tasks 7.3 – 7.9: Seven v1 resolvers

For **each** of: `doc_code`, `revision_number`, `effective_date`, `controlled_by_area`, `author`, `approvers`, `approval_date`:

- [ ] **Step 1: Write failing test** with fake RegistryReader/RevisionReader returning canned fixtures. Assert `Resolve` returns expected value and stable `InputsHash` across two calls with identical input.
- [ ] **Step 2: FAIL**
- [ ] **Step 3: Implement**

Template shape (copy per resolver; no reuse):

```go
// internal/modules/render/resolvers/doc_code.go
package resolvers

import (
    "context"
    "crypto/sha256"
    "time"
)

type DocCodeResolver struct{}

func (DocCodeResolver) Key() string    { return "doc_code" }
func (DocCodeResolver) Version() int   { return 1 }

func (DocCodeResolver) Resolve(ctx context.Context, in ResolveInput) (ResolvedValue, error) {
    rec, err := in.RegistryReader.GetControlledDocument(ctx, in.TenantID, in.ControlledDocumentID)
    if err != nil { return ResolvedValue{}, err }
    hasher := sha256.New()
    hasher.Write([]byte(rec.DocCode))
    return ResolvedValue{
        Value: rec.DocCode, ResolverKey: "doc_code", ResolverVer: 1,
        InputsHash: hasher.Sum(nil),
        ComputedAt: time.Now().UTC(),
    }, nil
}
```

Specifics for each resolver:

- **`doc_code`** — reads `registry.GetControlledDocument(tenant, controlledDocumentID).DocCode`.
- **`revision_number`** — reads `v2docs.RevisionReader.GetRevisionNumber(tenant, revisionID)`.
- **`effective_date`** — reads `v2docs.RevisionReader.GetEffectiveFrom(tenant, revisionID)`; returns `YYYY-MM-DD` string.
- **`controlled_by_area`** — uses `in.AreaCodeSnapshot` directly (no reader needed).
- **`author`** — reads `v2docs.RevisionReader.GetAuthor(tenant, revisionID)` → `{user_id, display_name}`.
- **`approvers`** — reads `workflow.GetApprovers(tenant, revisionID)` → `[{user_id, display_name, signed_at}]` in signoff order.
- **`approval_date`** — reads `workflow.GetFinalApprovalDate(tenant, revisionID)`; returns `YYYY-MM-DD` string.

Each `InputsHash` concatenates canonical JSON of inputs and hashes with sha256.

- [ ] **Step 4: PASS** — each resolver isolated.
- [ ] **Step 5: Commit per resolver**

```bash
rtk git commit -am "feat(render): <resolver_key> resolver v1"
```

### Task 7.10: `RegisterBuiltins` — wires all 7 into Registry

```go
func RegisterBuiltins(r *Registry) {
    r.Register(DocCodeResolver{})
    r.Register(RevisionNumberResolver{})
    r.Register(EffectiveDateResolver{})
    r.Register(ControlledByAreaResolver{})
    r.Register(AuthorResolver{})
    r.Register(ApproversResolver{})
    r.Register(ApprovalDateResolver{})
}
```

Test: builtins registry has exactly 7 entries.

Commit:

```bash
rtk git commit -am "feat(render): RegisterBuiltins wires 7 v1 resolvers"
```

---

## Phase 8 — SubBlockRenderer registry + 5 v1 sub-blocks (Codex medium)

(TypeScript, `apps/docgen-v2/src/render/subblocks/`)

### Task 8.1: `SubBlockRenderer` interface + `registry.ts`

- [ ] **Step 1: Failing test**

```ts
// apps/docgen-v2/src/render/subblocks/__tests__/registry.test.ts
import { describe, expect, test } from "vitest";
import { SubBlockRegistry } from "../registry";

describe("SubBlockRegistry", () => {
  test("register and render returns ooxml", async () => {
    const reg = new SubBlockRegistry();
    reg.register({
      key: "k1",
      render: async () => "<w:p>X</w:p>",
    });
    const out = await reg.render("k1", { params: {} });
    expect(out).toBe("<w:p>X</w:p>");
  });

  test("unknown key throws", async () => {
    const reg = new SubBlockRegistry();
    await expect(reg.render("nope", { params: {} })).rejects.toThrow(/unknown/);
  });
});
```

- [ ] **Step 2: FAIL**
- [ ] **Step 3: Implement**

```ts
// apps/docgen-v2/src/render/subblocks/registry.ts
export interface SubBlockContext {
  params: Record<string, unknown>;
  values: Record<string, unknown>;
}

export interface SubBlockRenderer {
  key: string;
  render(ctx: SubBlockContext): Promise<string>;
}

export class SubBlockRegistry {
  private map = new Map<string, SubBlockRenderer>();
  register(r: SubBlockRenderer) { this.map.set(r.key, r); }
  async render(key: string, ctx: SubBlockContext): Promise<string> {
    const r = this.map.get(key);
    if (!r) throw new Error(`unknown sub-block: ${key}`);
    return r.render(ctx);
  }
  keys(): string[] { return Array.from(this.map.keys()); }
}
```

- [ ] **Step 4: PASS**
- [ ] **Step 5: Commit**

```bash
rtk git commit -am "feat(docgen-v2): SubBlockRegistry scaffolding"
```

### Tasks 8.2 – 8.6: Five v1 sub-blocks

For each of: `doc_header_standard`, `revision_box`, `approval_signatures_block`, `footer_page_numbers`, `footer_controlled_copy_notice`:

- [ ] **Step 1: Failing test** — given a `SubBlockContext` with representative params/values, the renderer emits OOXML containing expected anchors (e.g. `doc_code` placeholder text for header; `Page X of Y` OOXML `<w:fldSimple w:instr="PAGE">` for footer_page_numbers).
- [ ] **Step 2: FAIL**
- [ ] **Step 3: Implement** with concrete OOXML string builder. Example for `footer_page_numbers`:

```ts
// apps/docgen-v2/src/render/subblocks/footer_page_numbers.ts
import { SubBlockRenderer } from "./registry";

export const FooterPageNumbers: SubBlockRenderer = {
  key: "footer_page_numbers",
  async render(): Promise<string> {
    return `<w:p><w:r><w:t xml:space="preserve">Page </w:t></w:r>` +
           `<w:fldSimple w:instr="PAGE"><w:r><w:t>1</w:t></w:r></w:fldSimple>` +
           `<w:r><w:t xml:space="preserve"> of </w:t></w:r>` +
           `<w:fldSimple w:instr="NUMPAGES"><w:r><w:t>1</w:t></w:r></w:fldSimple></w:p>`;
  },
};
```

For `approval_signatures_block`: render a `<w:tbl>` with one row per signer consumed from `ctx.values.approvers`.

For `doc_header_standard`: render title + doc code + effective date + revision number in a three-row table, each field pulling from `ctx.values`.

For `revision_box`: render a one-row table with columns `[Rev, Date, Description]` consuming `ctx.values.revision_history` (if empty, render placeholder "—").

For `footer_controlled_copy_notice`: fixed OOXML with tenant's copy-notice string read from `ctx.params.notice_text` (fallback "CONTROLLED COPY — WHEN PRINTED").

- [ ] **Step 4: PASS**
- [ ] **Step 5: Commit per sub-block**

```bash
rtk git commit -am "feat(docgen-v2): <subblock_key> sub-block renderer"
```

### Task 8.7: `registerV1Builtins(reg)` wires all 5

```ts
// apps/docgen-v2/src/render/subblocks/builtins.ts
import { SubBlockRegistry } from "./registry";
import { DocHeaderStandard } from "./doc_header_standard";
import { RevisionBox } from "./revision_box";
import { ApprovalSignaturesBlock } from "./approval_signatures_block";
import { FooterPageNumbers } from "./footer_page_numbers";
import { FooterControlledCopyNotice } from "./footer_controlled_copy_notice";

export function registerV1Builtins(r: SubBlockRegistry): void {
  r.register(DocHeaderStandard);
  r.register(RevisionBox);
  r.register(ApprovalSignaturesBlock);
  r.register(FooterPageNumbers);
  r.register(FooterControlledCopyNotice);
}
```

Test: after `registerV1Builtins(new SubBlockRegistry())`, `keys()` returns the exact 5.

Commit:

```bash
rtk git commit -am "feat(docgen-v2): registerV1Builtins wires 5 v1 sub-blocks"
```

---

## Phase 9 — Freeze service (Codex medium + Opus phase review)

### Task 9.1: `values_hash` canonical JSON computation

**Files:**
- Create: `internal/modules/documents_v2/domain/values_hash.go`
- Test: `internal/modules/documents_v2/domain/values_hash_test.go`

- [ ] **Step 1: Failing test**

```go
func TestValuesHash_OrderIndependent(t *testing.T) {
    a := map[string]any{"p1": "x", "p2": "y"}
    b := map[string]any{"p2": "y", "p1": "x"}
    if ComputeValuesHash(a) != ComputeValuesHash(b) {
        t.Fatal("hash must be order-independent")
    }
}
func TestValuesHash_ChangesOnValueChange(t *testing.T) {
    if ComputeValuesHash(map[string]any{"p1": "x"}) == ComputeValuesHash(map[string]any{"p1": "y"}) {
        t.Fatal("hash must differ on value change")
    }
}
```

- [ ] **Step 2: FAIL**
- [ ] **Step 3: Implement**

```go
package domain

import (
    "crypto/sha256"
    "encoding/hex"
    "encoding/json"
    "sort"
)

func ComputeValuesHash(values map[string]any) string {
    keys := make([]string, 0, len(values))
    for k := range values { keys = append(keys, k) }
    sort.Strings(keys)
    h := sha256.New()
    for _, k := range keys {
        v, _ := json.Marshal(values[k])
        h.Write([]byte(k))
        h.Write([]byte{0})
        h.Write(v)
        h.Write([]byte{0})
    }
    return hex.EncodeToString(h.Sum(nil))
}
```

- [ ] **Step 4: PASS**
- [ ] **Step 5: Commit**

```bash
rtk git commit -am "feat(documents_v2): canonical values_hash computation"
```

### Task 9.2: `FreezeService.Freeze` — validate + resolve computed + write values_hash

**Files:**
- Create: `internal/modules/documents_v2/application/freeze_service.go`
- Test: `internal/modules/documents_v2/application/freeze_service_test.go`

- [ ] **Step 1: Failing test** — given a revision with schema containing one user placeholder + one computed placeholder with resolver_key `doc_code`, when Freeze is called: (a) validates all required filled, (b) resolves the computed, (c) writes computed value row with `source='computed'`, (d) computes `values_hash` and stores it, (e) sets `values_frozen_at`.

- [ ] **Step 2: FAIL**
- [ ] **Step 3: Implement**

```go
package application

import (
    "context"
    "fmt"
    "time"

    v2dom "github.com/metaldocs/metaldocs-api/internal/modules/documents_v2/domain"
    "github.com/metaldocs/metaldocs-api/internal/modules/documents_v2/repository"
    "github.com/metaldocs/metaldocs-api/internal/modules/render/resolvers"
    tmpldom "github.com/metaldocs/metaldocs-api/internal/modules/templates_v2/domain"
)

type FreezeFinalizer interface {
    WriteFreeze(ctx context.Context, tenantID, revisionID string, valuesHash []byte, frozenAt time.Time) error
}

type FreezeService struct {
    schemas    SchemaReader
    values     FillInWriter
    valuesRead interface {
        ListValues(ctx context.Context, tenantID, revisionID string) ([]repository.PlaceholderValue, error)
    }
    resolvers *resolvers.Registry
    finalize  FreezeFinalizer
    resolveCtx ResolverContextBuilder
}

type ResolverContextBuilder interface {
    Build(ctx context.Context, tenantID, revisionID string) (resolvers.ResolveInput, error)
}

func NewFreezeService(
    schemas SchemaReader, values FillInWriter,
    valuesRead interface {
        ListValues(ctx context.Context, tenantID, revisionID string) ([]repository.PlaceholderValue, error)
    },
    reg *resolvers.Registry, final FreezeFinalizer, ctxBuilder ResolverContextBuilder,
) *FreezeService {
    return &FreezeService{schemas, values, valuesRead, reg, final, ctxBuilder}
}

func (s *FreezeService) Freeze(ctx context.Context, tenantID, revisionID string) error {
    schema, err := s.schemas.LoadPlaceholderSchema(ctx, tenantID, revisionID)
    if err != nil { return err }
    existing, err := s.valuesRead.ListValues(ctx, tenantID, revisionID)
    if err != nil { return err }
    byID := map[string]repository.PlaceholderValue{}
    for _, v := range existing { byID[v.PlaceholderID] = v }

    // Validate required are filled for all non-computed
    for _, p := range schema {
        if !p.Required || p.Computed { continue }
        v, ok := byID[p.ID]
        if !ok || v.ValueText == nil || *v.ValueText == "" {
            return fmt.Errorf("%w: placeholder %s required", v2dom.ErrValidationFailed, p.ID)
        }
    }

    // Resolve computed
    resolveIn, err := s.resolveCtx.Build(ctx, tenantID, revisionID)
    if err != nil { return err }
    for _, p := range schema {
        if !p.Computed { continue }
        if p.ResolverKey == nil {
            return fmt.Errorf("%w: placeholder %s computed without resolver_key",
                v2dom.ErrValidationFailed, p.ID)
        }
        r, ok := s.resolvers.Get(*p.ResolverKey)
        if !ok {
            return fmt.Errorf("%w: placeholder %s resolver %s",
                tmpldom.ErrUnknownResolver, p.ID, *p.ResolverKey)
        }
        rv, err := r.Resolve(ctx, resolveIn)
        if err != nil {
            return fmt.Errorf("resolver %s failed: %w", *p.ResolverKey, err)
        }
        strVal := fmt.Sprintf("%v", rv.Value)
        key, ver := *p.ResolverKey, rv.ResolverVer
        if err := s.values.UpsertValue(ctx, repository.PlaceholderValue{
            TenantID: tenantID, RevisionID: revisionID, PlaceholderID: p.ID,
            ValueText: &strVal, Source: "computed",
            ComputedFrom: &key, ResolverVersion: &ver,
            InputsHash: rv.InputsHash,
        }); err != nil {
            return err
        }
        byID[p.ID] = repository.PlaceholderValue{ValueText: &strVal}
    }

    // Compute values_hash
    valMap := make(map[string]any, len(byID))
    for _, p := range schema {
        if v, ok := byID[p.ID]; ok && v.ValueText != nil {
            valMap[p.ID] = *v.ValueText
        }
    }
    hashHex := v2dom.ComputeValuesHash(valMap)
    hashBytes, _ := hex.DecodeString(hashHex)
    return s.finalize.WriteFreeze(ctx, tenantID, revisionID, hashBytes, time.Now().UTC())
}
```

Add `import "encoding/hex"`.

- [ ] **Step 4: PASS**
- [ ] **Step 5: Commit**

```bash
rtk git commit -am "feat(documents_v2): FreezeService validates, resolves, hashes"
```

### Task 9.3: `SnapshotRepository.WriteFreeze`

- [ ] **Step 1: Test** — `WriteFreeze` sets `values_hash` + `values_frozen_at`; subsequent read returns them.
- [ ] **Step 2: FAIL**
- [ ] **Step 3: Add method**

```go
func (r *SnapshotRepository) WriteFreeze(ctx context.Context, tenant, docID string, valuesHash []byte, frozenAt time.Time) error {
    _, err := r.db.ExecContext(ctx, `
        UPDATE public.documents
           SET values_hash=$1, values_frozen_at=$2
         WHERE tenant_id=$3 AND id=$4`,
        valuesHash, frozenAt, tenant, docID)
    return err
}
```

- [ ] **Step 4: PASS**
- [ ] **Step 5: Commit**

```bash
rtk git commit -am "feat(documents_v2): SnapshotRepository WriteFreeze"
```

### Task 9.4: Opus phase-review checkpoint (no implementation)

Opus reviews Phases 1–9 output against spec §Freeze Model + §Data Flow + §Components. Produce `docs/superpowers/plans/reviews/placeholder-phase9-review.md` (only if Opus finds issues). No code change.

---

## Phase 10 — DOCX fanout pipeline (Codex medium)

### Task 10.1: `fanout.ts` single-pass orchestration (TypeScript)

**Files:**
- Create: `apps/docgen-v2/src/render/fanout.ts`
- Test: `apps/docgen-v2/src/render/__tests__/fanout.test.ts`

- [ ] **Step 1: Failing test** — given a minimal template DOCX bytes with one SDT `placeholder:doc_code` and one zone `zone-start:intro`, and values `{doc_code:"ABC-001"}` plus zone content `"<w:p><w:r><w:t>Hello</w:t></w:r></w:p>"`, and composition config `{header_sub_blocks:["doc_header_standard"],footer_sub_blocks:[]}`, calling `fanout({...})` returns DOCX bytes whose content_hash is stable and whose extracted text contains `ABC-001` and `Hello` and header content.

- [ ] **Step 2: FAIL**
- [ ] **Step 3: Implement**

```ts
// apps/docgen-v2/src/render/fanout.ts
import { createHash } from 'node:crypto';
import { processTemplateDetailed } from '@eigenpal/docx-js-editor';
import { injectZones } from './zoneInjection';
import { SubBlockRegistry } from './subblocks/registry';
import { registerV1Builtins } from './subblocks/builtins';

export interface FanoutInput {
  bodyDocx: Uint8Array;
  placeholderValues: Record<string, string>;
  zoneContent: Record<string, string>; // zoneId → OOXML
  compositionConfig: {
    header_sub_blocks: string[];
    footer_sub_blocks: string[];
    sub_block_params: Record<string, Record<string, unknown>>;
  };
  resolvedValues: Record<string, unknown>; // all values including computed for sub-blocks
}

export interface FanoutResult {
  buffer: Uint8Array;
  contentHash: string;
  unreplacedVars: string[];
}

export async function fanout(input: FanoutInput): Promise<FanoutResult> {
  const subReg = new SubBlockRegistry();
  registerV1Builtins(subReg);

  // 1. Render header/footer sub-blocks into a composed values map for processTemplate.
  const headerOoxml = (await Promise.all(
    input.compositionConfig.header_sub_blocks.map(k =>
      subReg.render(k, {
        params: input.compositionConfig.sub_block_params[k] ?? {},
        values: input.resolvedValues,
      }),
    ),
  )).join("");
  const footerOoxml = (await Promise.all(
    input.compositionConfig.footer_sub_blocks.map(k =>
      subReg.render(k, {
        params: input.compositionConfig.sub_block_params[k] ?? {},
        values: input.resolvedValues,
      }),
    ),
  )).join("");

  // 2. Inject zone OOXML into body bytes between bookmark pairs.
  const withZones = injectZones(input.bodyDocx, input.zoneContent);

  // 3. processTemplate for placeholders + inject header/footer slots via well-known tags.
  const variables: Record<string, string> = {
    ...input.placeholderValues,
    __header_composition__: headerOoxml,
    __footer_composition__: footerOoxml,
  };
  const result = processTemplateDetailed(
    withZones.buffer.slice(
      withZones.byteOffset,
      withZones.byteOffset + withZones.byteLength,
    ) as ArrayBuffer,
    variables,
    { nullGetter: 'empty' },
  );

  const buf = new Uint8Array(result.buffer);
  const contentHash = createHash('sha256').update(buf).digest('hex');
  return { buffer: buf, contentHash, unreplacedVars: result.unreplacedVariables ?? [] };
}
```

- [ ] **Step 4: PASS** — run `pnpm -C apps/docgen-v2 test`
- [ ] **Step 5: Commit**

```bash
rtk git commit -am "feat(docgen-v2): single-pass fanout with zone + subblock composition"
```

### Task 10.2: `zoneInjection.ts` — inject OOXML between bookmark pairs

- [ ] **Step 1: Failing test** — given a DOCX with bookmark pair `zone-start:intro` / `zone-end:intro` and inject map `{intro:"<w:p>X</w:p>"}`, result DOCX contains `<w:p>X</w:p>` between the bookmarks.
- [ ] **Step 2: FAIL**
- [ ] **Step 3: Implement** — unzip DOCX, parse `word/document.xml`, find `<w:bookmarkStart w:name="zone-start:<id>"/>`, insert given OOXML immediately after, re-zip.

```ts
// apps/docgen-v2/src/render/zoneInjection.ts
import JSZip from "jszip";

export async function injectZones(docx: Uint8Array, zones: Record<string, string>): Promise<Uint8Array> {
  const zip = await JSZip.loadAsync(docx);
  const file = zip.file("word/document.xml");
  if (!file) throw new Error("malformed DOCX: missing word/document.xml");
  let xml = await file.async("string");
  for (const [zoneId, ooxml] of Object.entries(zones)) {
    const startMarker = new RegExp(
      `(<w:bookmarkStart[^/]*w:name="zone-start:${escapeRegex(zoneId)}"[^/]*/>)`,
    );
    xml = xml.replace(startMarker, `$1${ooxml}`);
  }
  zip.file("word/document.xml", xml);
  const out = await zip.generateAsync({ type: "uint8array" });
  return out;
}

function escapeRegex(s: string): string { return s.replace(/[.*+?^${}()|[\]\\]/g, "\\$&"); }
```

Add `jszip` to `apps/docgen-v2/package.json` dependencies if missing: `pnpm -C apps/docgen-v2 add jszip`.

- [ ] **Step 4: PASS**
- [ ] **Step 5: Commit**

```bash
rtk git commit -am "feat(docgen-v2): zone OOXML injection between bookmark pairs"
```

### Task 10.3: `POST /render/fanout` route

**Files:**
- Create: `apps/docgen-v2/src/routes/fanout.ts`
- Test: `apps/docgen-v2/src/routes/__tests__/fanout.test.ts`

- [ ] **Step 1: Failing supertest** — POST body with base64 DOCX, values map, zones map, composition, assert 200 and JSON response `{ content_hash, final_docx_s3_key, unreplaced_vars }`.
- [ ] **Step 2: FAIL**
- [ ] **Step 3: Implement** Fastify route (match existing handler conventions in `routes/`). Upload result to S3 via existing `s3.ts` helper. Return content_hash + S3 key + unreplaced vars list.
- [ ] **Step 4: PASS**
- [ ] **Step 5: Commit**

```bash
rtk git commit -am "feat(docgen-v2): POST /render/fanout endpoint"
```

---

## Phase 11 — Backend fanout client + dispatch (Codex medium)

### Task 11.1: `fanout.Client` HTTP client

**Files:**
- Create: `internal/modules/render/fanout/client.go`
- Test: `internal/modules/render/fanout/client_test.go`

- [ ] **Step 1: Failing test** with httptest server returning canned response.
- [ ] **Step 2: FAIL**
- [ ] **Step 3: Implement**

```go
package fanout

import (
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    "net/http"
)

type FanoutRequest struct {
    TenantID          string            `json:"tenant_id"`
    RevisionID        string            `json:"revision_id"`
    BodyDocxS3Key     string            `json:"body_docx_s3_key"`
    PlaceholderValues map[string]string `json:"placeholder_values"`
    ZoneContent       map[string]string `json:"zone_content"`
    Composition       json.RawMessage   `json:"composition_config"`
    ResolvedValues    map[string]any    `json:"resolved_values"`
}

type FanoutResponse struct {
    ContentHash     string   `json:"content_hash"`
    FinalDocxS3Key  string   `json:"final_docx_s3_key"`
    UnreplacedVars  []string `json:"unreplaced_vars"`
}

type Client struct {
    baseURL string
    http    *http.Client
}

func NewClient(baseURL string, h *http.Client) *Client { return &Client{baseURL, h} }

func (c *Client) Fanout(ctx context.Context, req FanoutRequest) (FanoutResponse, error) {
    body, _ := json.Marshal(req)
    httpReq, _ := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/render/fanout", bytes.NewReader(body))
    httpReq.Header.Set("content-type", "application/json")
    resp, err := c.http.Do(httpReq)
    if err != nil { return FanoutResponse{}, err }
    defer resp.Body.Close()
    if resp.StatusCode != http.StatusOK {
        return FanoutResponse{}, fmt.Errorf("fanout status %d", resp.StatusCode)
    }
    var out FanoutResponse
    if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
        return FanoutResponse{}, err
    }
    return out, nil
}
```

- [ ] **Step 4: PASS**
- [ ] **Step 5: Commit**

```bash
rtk git commit -am "feat(render): fanout HTTP client"
```

### Task 11.2: `SnapshotRepository.WriteFinalDocx`

- [ ] **Step 1: Test** — writing final_docx_s3_key + content_hash persists.
- [ ] **Step 2: FAIL**
- [ ] **Step 3: Implement**

```go
func (r *SnapshotRepository) WriteFinalDocx(ctx context.Context, tenant, docID, s3Key string, contentHash []byte) error {
    _, err := r.db.ExecContext(ctx, `
        UPDATE public.documents
           SET final_docx_s3_key=$1, content_hash=$2
         WHERE tenant_id=$3 AND id=$4`,
        s3Key, contentHash, tenant, docID)
    return err
}
```

(`content_hash` already exists on documents from Spec 2 — confirm via `rtk grep -n "content_hash" migrations/ | head -5`. If not present, add in 0152 migration retroactively, else reuse.)

- [ ] **Step 4: PASS**
- [ ] **Step 5: Commit**

```bash
rtk git commit -am "feat(documents_v2): persist final_docx_s3_key + content_hash"
```

### Task 11.3: `PDFDispatcher` — enqueue `docgen_v2_pdf` service bus job

**Files:**
- Create: `internal/modules/render/fanout/pdf_dispatcher.go`
- Test: `internal/modules/render/fanout/pdf_dispatcher_test.go`

- [ ] **Step 1: Failing test** with fake bus implementing `Publish(topic, body)`.
- [ ] **Step 2: FAIL**
- [ ] **Step 3: Implement** — publishes `{tenant_id, revision_id, final_docx_s3_key}` to `docgen_v2_pdf` topic using existing platform service-bus wrapper (grep `internal/platform/` for existing publisher; reuse).
- [ ] **Step 4: PASS**
- [ ] **Step 5: Commit**

```bash
rtk git commit -am "feat(render): PDFDispatcher enqueues docgen_v2_pdf job"
```

### Task 11.4: Extend `FreezeService` to call `Fanout` only (PDF dispatch is post-commit, not inside Freeze)

**Invariant:** `FreezeService.Freeze` runs synchronously under the approval transaction (see Task 16.1). Its sole side-effects under the tx are:
1. Validate required placeholders filled.
2. Resolve computed placeholders (with approver context).
3. Compute `values_hash`, write `values_frozen_at` + `values_hash`.
4. Call `fanoutClient.Fanout(...)` (synchronous DOCX build).
5. `snapshotRepo.WriteFinalDocx(...)` — persist `final_docx_s3_key` + `content_hash`.

`FreezeService.Freeze` MUST NOT call `PDFDispatcher.Dispatch`. PDF dispatch happens in `DecisionService.RecordSignoff` **after** the approval transaction commits (Task 16.1). This preserves the spec invariant that PDF is a best-effort artifact whose failure never blocks approval.

- [ ] **Step 1: Failing integration test** — exercises `FreezeService.Freeze` with a stubbed fanout client (returns `{content_hash, final_docx_s3_key}`) and asserts:
  (a) `values_hash`, `values_frozen_at`, `final_docx_s3_key`, `content_hash` all populated after Freeze;
  (b) `PDFDispatcher.Dispatch` is NEVER invoked by `Freeze` (inject a counting fake — assert count == 0);
  (c) when `fanoutClient.Fanout` returns an error, `Freeze` returns the error and writes NOTHING (no partial state — assert `values_frozen_at IS NULL` after error).

- [ ] **Step 2: FAIL**

- [ ] **Step 3: Implement** — append to `FreezeService.Freeze` after the existing `WriteFreeze` call:

```go
// (validation + computed resolve + WriteFreeze already happened above.)

// Load snapshot + values + zones for fanout input.
snap, _ := s.snapshots.ReadSnapshot(ctx, tenantID, revisionID)
vals, _ := s.valuesRead.ListValues(ctx, tenantID, revisionID)
zones, _ := s.zonesRead.ListZoneContent(ctx, tenantID, revisionID)

valMap := map[string]string{}
resolvedForSubblocks := map[string]any{}
for _, v := range vals {
    if v.ValueText != nil {
        valMap[v.PlaceholderID] = *v.ValueText
        resolvedForSubblocks[v.PlaceholderID] = *v.ValueText
    }
}
zoneMap := map[string]string{}
for _, z := range zones { zoneMap[z.ZoneID] = z.ContentOOXML }

resp, err := s.fanout.Fanout(ctx, fanout.FanoutRequest{
    TenantID: tenantID, RevisionID: revisionID,
    BodyDocxS3Key:     snap.BodyDocxS3Key,
    PlaceholderValues: valMap,
    ZoneContent:       zoneMap,
    Composition:       snap.CompositionJSON,
    ResolvedValues:    resolvedForSubblocks,
})
if err != nil {
    return fmt.Errorf("fanout: %w", err) // caller's tx rolls back; values_frozen_at stays NULL via its earlier WriteFreeze being part of the same tx
}

contentHashBytes, _ := hex.DecodeString(resp.ContentHash)
if err := s.snapshots.WriteFinalDocx(ctx, tenantID, revisionID, resp.FinalDocxS3Key, contentHashBytes); err != nil {
    return err
}
// Explicitly: NO PDFDispatcher call here. PDF is dispatched by DecisionService AFTER commit.
return nil
```

Both `WriteFreeze` and `WriteFinalDocx` must run on the same `*sql.Tx` as the signoff; extend `SnapshotRepository` method signatures to accept an optional `*sql.Tx`. If `tx == nil`, fall back to `r.db`.

- [ ] **Step 4: PASS**
- [ ] **Step 5: Commit**

```bash
rtk git commit -am "feat(documents_v2): FreezeService runs fanout synchronously (PDF dispatched post-commit)"
```

---

## Phase 12 — PDF worker (Codex medium)

### Task 12.1: `docgen_v2_pdf` worker reads DOCX, calls Gotenberg, writes PDF

**Files:**
- Modify: `apps/docgen-v2/src/pdf/` + `apps/docgen-v2/src/routes/` (existing scaffold under `pdf/version.ts`)
- Create: `apps/docgen-v2/src/pdf/worker.ts`
- Test: `apps/docgen-v2/src/pdf/__tests__/worker.test.ts`

- [ ] **Step 1: Failing test** — message `{tenant_id, revision_id, final_docx_s3_key}` arrives, worker reads DOCX from S3, POSTs to Gotenberg `/forms/libreoffice/convert`, writes PDF to S3 at `<key>.pdf`, returns `{final_pdf_s3_key, pdf_hash, pdf_generated_at}`.
- [ ] **Step 2: FAIL**
- [ ] **Step 3: Implement.** Use existing `internal/platform/render/gotenberg/client.go` pattern — adapt for TS client. Retry on 5xx with exponential backoff (3 attempts). On final failure, throw for message redelivery.
- [ ] **Step 4: PASS**
- [ ] **Step 5: Commit**

```bash
rtk git commit -am "feat(docgen-v2): docgen_v2_pdf worker converts DOCX to PDF"
```

### Task 12.2: Backend endpoint to receive PDF completion webhook → persist `pdf_hash`

**Files:**
- Create: `internal/modules/documents_v2/http/pdf_webhook_handler.go`

- [ ] **Step 1: Failing test** — POST `/api/v2/documents/{id}/pdf-complete` with HMAC-signed body persists `final_pdf_s3_key`, `pdf_hash`, `pdf_generated_at`; unsigned rejected 401.
- [ ] **Step 2: FAIL**
- [ ] **Step 3: Implement** — verify HMAC using shared secret env var `DOCGEN_V2_WEBHOOK_SECRET`; call `SnapshotRepository.WritePDF`.
- [ ] **Step 4: PASS**
- [ ] **Step 5: Commit**

```bash
rtk git commit -am "feat(documents_v2): PDF completion webhook handler"
```

---

## Phase 13 — Viewer endpoint + area RBAC (Codex medium)

### Task 13.1: `GET /api/v2/documents/{id}/view` returns signed PDF URL

**Files:**
- Create: `internal/modules/documents_v2/http/view_handler.go`
- Test: `internal/modules/documents_v2/http/view_handler_test.go`

- [ ] **Step 1: Failing tests (three cases)** —
  (a) consumer in user's area hits `/view` on `status=approved` revision → 200 + `{signed_url}` (spec's viewer flow is explicitly for `approved`);
  (b) consumer in user's area hits `/view` on `status=published` revision → 200 + `{signed_url}` (published is a superset state of approved for viewing);
  (c) draft/under_review revision → 404;
  (d) consumer without area grant → 403.
- [ ] **Step 2: FAIL**
- [ ] **Step 3: Implement** — requires capability `doc.view_published` (add seed in this task's migration 0155). Handler accepts `status IN ('approved','scheduled','published')`; returns 404 otherwise. Reads `final_pdf_s3_key`; returns 404 if PDF not yet generated (with body `{error:"pdf_pending"}` to distinguish). Uses existing `s3.PresignGet` helper (grep `internal/platform/s3/` first).
- [ ] **Step 4: PASS**
- [ ] **Step 5: Commit**

```bash
rtk git commit -am "feat(documents_v2): GET /view signed PDF URL with area RBAC"
```

### Task 13.2: Capability `doc.view_published` seed

Migration `0155_capability_doc_view_published.sql`:

```sql
INSERT INTO metaldocs.role_capabilities(capability, role)
VALUES ('doc.view_published','reader'),
       ('doc.view_published','author'),
       ('doc.view_published','reviewer'),
       ('doc.view_published','signer'),
       ('doc.view_published','area_admin'),
       ('doc.view_published','qms_admin')
ON CONFLICT DO NOTHING;
```

- [ ] Commit

```bash
rtk git commit -am "feat(migrations): 0155 doc.view_published capability"
```

---

## Phase 14 — Reconstruction (forensic, admin only) (Codex medium)

### Task 14.1: `ReconstructService.Reconstruct` — never overwrites originals

**Files:**
- Create: `internal/modules/render/fanout/reconstruction.go`
- Test: `internal/modules/render/fanout/reconstruction_test.go`

- [ ] **Step 1: Failing test** — given a revision with populated `final_docx_s3_key` + `content_hash`, calling `Reconstruct` (a) re-runs fanout, (b) appends an entry to `reconstruction_attempts` containing `rendered_at`, `eigenpal_ver`, `docxtemplater_ver`, `bytes_hash`, `matches_original`, (c) does NOT modify `final_docx_s3_key` or `content_hash`.

- [ ] **Step 2: FAIL**
- [ ] **Step 3: Implement** — repo method `AppendReconstruction(ctx, tenant, docID, entry)`:

```go
func (r *SnapshotRepository) AppendReconstruction(ctx context.Context, tenant, docID string, entry []byte) error {
    _, err := r.db.ExecContext(ctx, `
        UPDATE public.documents
           SET reconstruction_attempts = reconstruction_attempts || $1::jsonb
         WHERE tenant_id=$2 AND id=$3`,
        entry, tenant, docID)
    return err
}
```

Service runs fanout, hashes result, compares to stored `content_hash`, builds JSON entry, appends. Never writes `content_hash` or `final_docx_s3_key`.

- [ ] **Step 4: PASS**
- [ ] **Step 5: Commit**

```bash
rtk git commit -am "feat(render): Reconstruct appends attempt, never overwrites originals"
```

### Task 14.2: Admin endpoint `POST /api/v2/documents/{id}/reconstruct`

Requires capability `doc.reconstruct` (seed via migration 0156 for `qms_admin` only). Returns the appended entry.

Commit:

```bash
rtk git commit -am "feat(documents_v2): admin reconstruct endpoint"
```

---

## Phase 15 — Editor UX guardrails (Codex medium)

### Task 15.1: Emit typed `sdtType` per placeholder type

**Files:**
- Modify: `frontend/apps/web/src/editor-adapters/eigenpal-template-mode.ts`
- Test: `frontend/apps/web/src/editor-adapters/__tests__/eigenpal-template-mode.sdt-type.test.ts`

- [ ] **Step 1: Failing test** — `placeholderToRun({type:"placeholder", id:"d1", label:"Date", placeholderType:"date"})` returns node with `properties.sdtType === "date"`; similar for `select` → `"dropdown"` with `listItems`; `picture` → `"picture"`; `computed` → `"plainText"` with `lock: "sdtContentLocked"`; default/text → `"richText"`.

- [ ] **Step 2: FAIL**
- [ ] **Step 3: Extend `PlaceholderRun` type**:

```ts
export type PlaceholderRun = {
  type: "placeholder";
  id: string;
  label: string;
  placeholderType?: "text" | "date" | "number" | "select" | "user" | "picture" | "computed";
  options?: string[];
};
```

Map in `placeholderToRun`:

```ts
const sdtType =
  p.placeholderType === "date" ? "date" :
  p.placeholderType === "select" ? "dropdown" :
  p.placeholderType === "picture" ? "picture" :
  p.placeholderType === "computed" ? "plainText" :
  "richText";
const props: any = {
  sdtType, tag: `${PLACEHOLDER_TAG_PREFIX}${p.id}`, alias: p.label, placeholder: p.label,
};
if (p.placeholderType === "select" && p.options) props.listItems = p.options.map(o => ({ displayText: o, value: o }));
if (p.placeholderType === "computed") props.lock = "sdtContentLocked";
```

- [ ] **Step 4: PASS**
- [ ] **Step 5: Commit**

```bash
rtk git commit -am "feat(editor-adapters): typed sdtType per placeholder type"
```

### Task 15.2: Wrap frozen template content in locked SDT

- [ ] **Step 1: Failing test** — helper `wrapFrozenContent(blocks)` returns content nested in a block-level SDT with `lock: "sdtContentLocked"`.
- [ ] **Step 2: FAIL**
- [ ] **Step 3: Implement helper** in `eigenpal-template-mode.ts`.
- [ ] **Step 4: PASS**
- [ ] **Step 5: Commit**

```bash
rtk git commit -am "feat(editor-adapters): wrapFrozenContent with sdtContentLocked"
```

### Task 15.3: `filterTransaction` plugin

**Files:**
- Create: `frontend/apps/web/src/editor-adapters/filter-transaction-guard.ts`
- Test: `frontend/apps/web/src/editor-adapters/__tests__/filter-transaction-guard.test.ts`

- [ ] **Step 1: Failing test** — using ProseMirror test util, build a doc with one locked SDT and one unlocked SDT; simulate a transaction replacing text inside locked → assert transaction rejected; inside unlocked → accepted.
- [ ] **Step 2: FAIL**
- [ ] **Step 3: Implement**

```ts
// frontend/apps/web/src/editor-adapters/filter-transaction-guard.ts
import { Plugin } from "prosemirror-state";

export function filterTransactionGuard() {
  return new Plugin({
    filterTransaction(tr, state) {
      if (!tr.docChanged) return true;
      let allowed = true;
      tr.steps.forEach((step, i) => {
        const map = tr.mapping.maps[i];
        map.forEach((oldStart, oldEnd) => {
          state.doc.nodesBetween(oldStart, Math.min(oldEnd, state.doc.content.size), node => {
            if (node.attrs?.sdtLock === "sdtContentLocked") {
              allowed = false;
              return false;
            }
            return true;
          });
        });
      });
      return allowed;
    },
  });
}
```

(Assumes SDT node attrs surface `sdtLock` in the schema — if eigenpal adapter does not yet expose this, extend the node spec in `eigenpal-template-mode.ts` to copy `properties.lock` onto the PM node attrs. Add a prior micro-step to the task if needed.)

- [ ] **Step 4: PASS**
- [ ] **Step 5: Commit**

```bash
rtk git commit -am "feat(editor-adapters): filterTransaction guard for locked SDT ranges"
```

---

## Phase 16 — Wire Spec 2 approval transition → freeze (Codex medium)

### Task 16.1: `DecisionService.RecordSignoff` triggers `FreezeService.Freeze` atomically on final approval

**Files:**
- Modify: `internal/modules/documents_v2/approval/application/decision_service.go`
- Test: `internal/modules/documents_v2/approval/application/decision_service_test.go`

- [ ] **Step 1: Failing tests (four cases)** —
  (a) quorum-approved signoff on last stage transitions doc to `approved` AND invokes `FreezeService.Freeze` exactly once, and the approver's identity + capabilities are propagated to `ResolverContextBuilder`;
  (b) when `FreezeService.Freeze` returns an error (fanout failure), the entire approval transaction rolls back — doc remains in `under_review`, stage instance is NOT marked complete, NO signature row is committed. Assert post-rollback DB state;
  (c) when the synchronous DOCX fanout succeeds but PDF dispatch returns a transient error, the approval is still committed (PDF is best-effort) — assert doc status=`approved`, `final_docx_s3_key` set, `pdf_hash` remains NULL;
  (d) idempotency — calling the signoff twice (same idempotency key) invokes `Freeze` at most once.

- [ ] **Step 2: FAIL**

- [ ] **Step 3: Implement** — inject `FreezeInvoker` and `ApproverContext` into `DecisionService`. Restructure the signoff tx boundary:

```go
// Inside DecisionService.RecordSignoff, within the existing tx:
if quorumReached && isLastStage {
    // Run freeze BEFORE marking doc approved, within the same transaction.
    // Freeze does: validate, resolve computed (with approver context), compute values_hash,
    // call fanout (synchronous DOCX build), write final_docx_s3_key + content_hash.
    approver := ApproverContext{
        UserID:      signoff.ActorID,
        Capabilities: signoff.Capabilities, // populated by authz at handler entry
    }
    if err := s.freeze.Freeze(ctx, tx, tenantID, doc.ID, approver); err != nil {
        return fmt.Errorf("freeze: %w", err)  // tx rolls back, doc stays under_review
    }
    // Only now mark doc approved + stage complete + commit.
    if _, err := tx.ExecContext(ctx, `UPDATE documents SET status='approved' WHERE ...`); err != nil {
        return err
    }
    // PDF dispatch is AFTER commit (non-blocking):
    defer func() {
        if commitErr == nil {
            _ = s.pdfDispatcher.Dispatch(ctx, tenantID, doc.ID) // fire-and-forget; errors logged, not returned
        }
    }()
}
```

Key invariants for the reviewer agent to preserve:
- `FreezeService.Freeze` now takes a `*sql.Tx` and runs the synchronous DOCX fanout under the approval transaction. Fanout failure rolls the whole approval back.
- `PDFDispatcher.Dispatch` is called *after* `tx.Commit()` succeeds. PDF failures never block approval; the `docgen_v2_pdf` worker retries via Service Bus redelivery.
- `ApproverContext` carries signer identity and capabilities; `ResolverContextBuilder.Build` now takes `(ctx, tenantID, revisionID, approver ApproverContext)` so sensitive resolvers (e.g. `approvers`) authorize reads via the approver's `workflow.approve` capability per spec §Computed resolvers.

Contract: `FreezeService.Freeze` is **idempotent** — on entry it reads `values_frozen_at`; if set, returns nil without re-running.

- [ ] **Step 4: PASS**
- [ ] **Step 5: Commit**

```bash
rtk git commit -am "feat(approval): invoke FreezeService on quorum approval"
```

### Task 16.2: Signoff payload binds `(content_hash, values_hash, schema_hash)`

**Files:**
- Modify: `internal/modules/documents_v2/approval/application/content_hash.go` (or equivalent)
- Test: `internal/modules/documents_v2/approval/application/content_hash_test.go`

- [ ] **Step 1: Failing test** — signoff payload builder now includes all three hashes; changing `values_hash` produces a different payload_hash.
- [ ] **Step 2: FAIL**
- [ ] **Step 3: Extend the payload struct used by the signature module:**

```go
type SignaturePayload struct {
    RevisionID     string `json:"revision_id"`
    ContentHash    string `json:"content_hash"`
    ValuesHash     string `json:"values_hash"`
    SchemaHash     string `json:"schema_hash"`
    StageInstanceID string `json:"stage_instance_id"`
    SignerUserID   string `json:"signer_user_id"`
    SignedAt       string `json:"signed_at"`
}
```

Update `ComputePayloadHash` to include the new fields. Update signoff handler to populate them from the snapshot row.

- [ ] **Step 4: PASS**
- [ ] **Step 5: Commit**

```bash
rtk git commit -am "feat(approval): signature payload binds triple-hash"
```

### Task 16.3: Freeze idempotency guard

- [ ] **Step 1: Failing test** — calling `Freeze` twice on same revision produces identical `values_hash` and a single `values_frozen_at`; second invocation must NOT overwrite `final_docx_s3_key`.
- [ ] **Step 2: FAIL**
- [ ] **Step 3: Add early-return in `FreezeService.Freeze`:**

```go
snap, err := s.snapshots.ReadSnapshot(ctx, tenantID, revisionID)
if err != nil { return err }
if snap.ValuesFrozenAt != nil {
    return nil // already frozen
}
```

(Extend `ReadSnapshot` to return `ValuesFrozenAt *time.Time`.)

- [ ] **Step 4: PASS**
- [ ] **Step 5: Commit**

```bash
rtk git commit -am "feat(documents_v2): FreezeService idempotency guard"
```

---

## Phase 17 — End-to-end + round-trip tests (Codex medium)

### Task 17.1: Full Draft → Submit → Signoff → Approve → Freeze → Publish E2E

**Files:**
- Create: `tests/integration/scenarios/placeholder_fanout_e2e_test.go`

- [ ] **Step 1: Failing test** exercising the full HTTP flow with real Postgres:
  1. Create template with `doc_code` computed placeholder + one user placeholder + one zone + one header sub-block.
  2. POST `/api/v2/documents` (create revision from template).
  3. PUT placeholder value + zone content.
  4. POST `/submit` (Spec 2).
  5. POST signoff (Spec 2) completing quorum.
  6. Assert documents row has `values_hash`, `final_docx_s3_key`, status=`approved`.
  7. POST `/publish`; assert status=`published`.
  8. GET `/view` as reader → 200 + signed URL.

Use LocalStack S3 and a mocked docgen-v2 HTTP server returning canned response.

- [ ] **Step 2: FAIL**
- [ ] **Step 3: Implement** scenario test with existing testdb harness.
- [ ] **Step 4: PASS** — `rtk go test ./tests/integration/scenarios/ -run TestPlaceholderFanoutE2E -v`
- [ ] **Step 5: Commit**

```bash
rtk git commit -am "test(integration): placeholder fanout E2E scenario"
```

### Task 17.2: DOCX round-trip (eigenpal inlineSdt preservation)

**Files:**
- Create: `frontend/apps/web/src/editor-adapters/__tests__/inline-sdt-round-trip.test.ts`

- [ ] **Step 1: Failing test** — build a paragraph with one placeholder via `placeholderToRun`, serialize to DOCX via eigenpal core, parse the DOCX back through eigenpal, walk nodes, find the SDT, assert `tag`, `alias`, `sdtType` all preserved.
- [ ] **Step 2: FAIL**
- [ ] **Step 3: Implement test.**
- [ ] **Step 4: PASS**
- [ ] **Step 5: Commit**

```bash
rtk git commit -am "test(editor-adapters): inline SDT round-trip through DOCX"
```

### Task 17.3: Resolver contract tests (all 7)

One test per resolver with fixed fixtures asserting stable `InputsHash` across two invocations.

- [ ] Commit each:

```bash
rtk git commit -am "test(render): <resolver> contract test"
```

### Task 17.4: RBAC tests — `/view` and `/reconstruct`

- [ ] **Step 1: Failing tests** — reader without area grant → 403; qms_admin gets 200 on reconstruct; any other role on reconstruct → 403.
- [ ] **Step 2: FAIL**
- [ ] **Step 3: Implement.**
- [ ] **Step 4: PASS**
- [ ] **Step 5: Commit**

```bash
rtk git commit -am "test(documents_v2): view and reconstruct RBAC"
```

### Task 17.5: Reconstruction drift detection

- [ ] **Step 1: Failing test** — seed a revision with known `content_hash`; run `Reconstruct` with a fanout stub returning different bytes; assert `reconstruction_attempts[-1].matches_original == false` AND original `content_hash` unchanged.
- [ ] **Step 2: FAIL**
- [ ] **Step 3: Implement.**
- [ ] **Step 4: PASS**
- [ ] **Step 5: Commit**

```bash
rtk git commit -am "test(render): reconstruction drift detection"
```

---

## Phase 5b — Draft-time computed placeholder resolution (Codex medium)

Spec §Fill-in flow: computed placeholders resolve on load and on dependency change during draft with `inputs_hash` cache reuse.

### Task 5b.1: `DraftResolverService.ResolveComputedIfStale`

**Files:**
- Create: `internal/modules/documents_v2/application/draft_resolver_service.go`
- Test: `internal/modules/documents_v2/application/draft_resolver_service_test.go`

- [ ] **Step 1: Failing tests** —
  (a) first call for a computed placeholder with empty current value writes `source='computed'` + `inputs_hash` + `resolver_version`;
  (b) second call with unchanged upstream values reads stored `inputs_hash`, matches it to a freshly-computed inputs hash, and SKIPS the resolver `Resolve` call (fake resolver's counter stays at 1);
  (c) third call after an upstream user-placeholder change (different inputs hash) re-invokes `Resolve` and writes new value + new `inputs_hash`.

- [ ] **Step 2: FAIL**

- [ ] **Step 3: Implement**

```go
package application

import (
    "context"
    "bytes"

    "github.com/metaldocs/metaldocs-api/internal/modules/documents_v2/repository"
    "github.com/metaldocs/metaldocs-api/internal/modules/render/resolvers"
    tmpldom "github.com/metaldocs/metaldocs-api/internal/modules/templates_v2/domain"
)

type DraftResolverService struct {
    schemas    SchemaReader
    values     FillInWriter
    valuesRead interface {
        ListValues(ctx context.Context, tenantID, revisionID string) ([]repository.PlaceholderValue, error)
    }
    resolvers *resolvers.Registry
    resolveCtx ResolverContextBuilder
}

func NewDraftResolverService(
    schemas SchemaReader, values FillInWriter,
    valuesRead interface {
        ListValues(ctx context.Context, tenantID, revisionID string) ([]repository.PlaceholderValue, error)
    },
    reg *resolvers.Registry, ctxBuilder ResolverContextBuilder,
) *DraftResolverService {
    return &DraftResolverService{schemas, values, valuesRead, reg, ctxBuilder}
}

// ResolveComputedIfStale resolves all computed placeholders whose inputs_hash differs from the
// currently stored value. Called on draft load and after any user-placeholder upsert.
func (s *DraftResolverService) ResolveComputedIfStale(ctx context.Context, tenantID, revisionID string) error {
    schema, err := s.schemas.LoadPlaceholderSchema(ctx, tenantID, revisionID)
    if err != nil { return err }
    existing, err := s.valuesRead.ListValues(ctx, tenantID, revisionID)
    if err != nil { return err }
    byID := map[string]repository.PlaceholderValue{}
    for _, v := range existing { byID[v.PlaceholderID] = v }

    // Draft-time resolver context: no approver yet — uses revision-author context.
    rin, err := s.resolveCtx.BuildForDraft(ctx, tenantID, revisionID)
    if err != nil { return err }

    for _, p := range schema {
        if !p.Computed || p.ResolverKey == nil { continue }
        r, ok := s.resolvers.Get(*p.ResolverKey)
        if !ok {
            return fmt.Errorf("%w: %s", tmpldom.ErrUnknownResolver, *p.ResolverKey)
        }
        rv, err := r.Resolve(ctx, rin)
        if err != nil { return err }
        if cur, ok := byID[p.ID]; ok && bytes.Equal(cur.InputsHash, rv.InputsHash) {
            continue // cache hit — skip write
        }
        strVal := fmt.Sprintf("%v", rv.Value)
        key, ver := *p.ResolverKey, rv.ResolverVer
        if err := s.values.UpsertValue(ctx, repository.PlaceholderValue{
            TenantID: tenantID, RevisionID: revisionID, PlaceholderID: p.ID,
            ValueText: &strVal, Source: "computed",
            ComputedFrom: &key, ResolverVersion: &ver,
            InputsHash: rv.InputsHash,
        }); err != nil {
            return err
        }
    }
    return nil
}
```

Add `BuildForDraft` method to `ResolverContextBuilder` alongside `Build`. Draft-time authz uses revision-author identity.

- [ ] **Step 4: PASS**
- [ ] **Step 5: Commit**

```bash
rtk git commit -am "feat(documents_v2): draft-time computed placeholder resolution with inputs_hash cache"
```

### Task 5b.2: Hook draft resolver into fill-in endpoints + revision load

- [ ] **Step 1: Failing integration test** —
  (a) on `GET /api/v2/documents/{id}` (draft), response includes computed values resolved from schema;
  (b) after `PUT /placeholders/{pid}` changing a user-placeholder that a computed placeholder depends on via `visible_if` (or by resolver contract — e.g. `doc_code` after registry change), GET returns updated computed value with new `inputs_hash`.

- [ ] **Step 2: FAIL**

- [ ] **Step 3: Implement** — inject `DraftResolverService` into document read handler and `FillInService.SetPlaceholderValue` tail. Every user placeholder upsert followed by `ResolveComputedIfStale` (best-effort; resolver errors logged, not fatal during draft).

- [ ] **Step 4: PASS**

- [ ] **Step 5: Commit**

```bash
rtk git commit -am "feat(documents_v2): wire draft resolver into read and upsert paths"
```

---

## Phase 7b — IAM user options + user-placeholder validation (Codex medium)

Spec §Placeholders: `user → dropdown populated from IAM`.

### Task 7b.1: `IAMUserOptionsReader` port + implementation

**Files:**
- Create: `internal/modules/documents_v2/application/iam_user_options.go`
- Test: `internal/modules/documents_v2/application/iam_user_options_test.go`

- [ ] **Step 1: Failing test** — given a tenant with three IAM users, `ListUserOptions(ctx, tenant)` returns `[{user_id, display_name}]` sorted by display_name.

- [ ] **Step 2: FAIL**

- [ ] **Step 3: Implement** — port consumes existing `iam.Reader` (grep `internal/modules/iam/`), maps to UI option rows.

```go
type IAMUserOptionsReader interface {
    ListUserOptions(ctx context.Context, tenantID string) ([]UserOption, error)
}

type UserOption struct {
    UserID      string `json:"user_id"`
    DisplayName string `json:"display_name"`
}
```

- [ ] **Step 4: PASS**

- [ ] **Step 5: Commit**

```bash
rtk git commit -am "feat(documents_v2): IAM user options reader"
```

### Task 7b.2: `GET /api/v2/documents/{id}/placeholder-options/{pid}` returns user options for `user` placeholder

- [ ] **Step 1: Failing test** — GET on a `user`-typed placeholder returns `{options:[{user_id,display_name}]}`; GET on a `text` placeholder → 400 `not_a_choice_placeholder`.

- [ ] **Step 2: FAIL**

- [ ] **Step 3: Implement** handler. Uses `IAMUserOptionsReader` for `user` type; `select` type returns `options` from schema directly.

- [ ] **Step 4: PASS**

- [ ] **Step 5: Commit**

```bash
rtk git commit -am "feat(documents_v2): placeholder options endpoint"
```

### Task 7b.3: Extend `validateValue` — `user` placeholder must be a known IAM user

- [ ] **Step 1: Failing test** — `SetPlaceholderValue` for `user` placeholder with unknown `user_id` → 422 `unknown_user`; with known → success.

- [ ] **Step 2: FAIL**

- [ ] **Step 3: Inject `IAMUserOptionsReader` into `FillInService`. Extend `validateValue`:**

```go
case tmpldom.PHUser:
    opts, err := s.iam.ListUserOptions(ctx, tenantID)
    if err != nil { return err }
    found := false
    for _, o := range opts { if o.UserID == raw { found = true; break } }
    if !found {
        return fmt.Errorf("%w: %s unknown user %s", v2domain.ErrValidationFailed, p.ID, raw)
    }
```

(Threading `ctx, tenantID` through `validateValue` requires changing the helper signature; apply that refactor.)

- [ ] **Step 4: PASS**

- [ ] **Step 5: Commit**

```bash
rtk git commit -am "feat(documents_v2): validate user placeholders against IAM"
```

---

## Phase 19 — Template authoring UI (Codex medium)

Spec §Template author flow: QMS admin edits placeholder schema (validation rules, `visible_if`, `computed` resolver key), composition config (header/footer sub-block toggles), and zone ContentPolicy — all in-canvas / side-panel.

### Task 19.1: Placeholder inspector panel

**Files:**
- Create: `frontend/apps/web/src/features/templates/placeholder-inspector.tsx`
- Test: `frontend/apps/web/src/features/templates/__tests__/placeholder-inspector.test.tsx`

- [ ] **Step 1: Failing test** — render with a selected placeholder of type `text`; interacting with inputs for `required`, `regex`, `max_length`, `visible_if` fires `onChange` with a merged `Placeholder` object; switching type to `number` reveals `min_number`/`max_number` inputs and hides `regex`; switching to `computed` reveals resolver-key `<select>` populated from a `resolvers` prop (array of `{key, version}`).

- [ ] **Step 2: FAIL**

- [ ] **Step 3: Implement** React component rendering type-specific inputs. Props: `{value: Placeholder, resolvers: {key:string,version:number}[], onChange}`. Uses existing UI kit (grep `frontend/apps/web/src/components/ui/` for the design system library).

- [ ] **Step 4: PASS** — `pnpm -C frontend/apps/web test`

- [ ] **Step 5: Commit**

```bash
rtk git commit -am "feat(templates-ui): placeholder inspector panel"
```

### Task 19.2: Zone inspector panel (ContentPolicy toggles + MaxLength)

- [ ] **Step 1: Failing test** — four toggles for tables/images/headings/lists and a numeric `max_length` input; interactions fire `onChange(EditableZone)`.

- [ ] **Step 2: FAIL**

- [ ] **Step 3: Implement.**

- [ ] **Step 4: PASS**

- [ ] **Step 5: Commit**

```bash
rtk git commit -am "feat(templates-ui): zone inspector with content policy toggles"
```

### Task 19.3: Composition config panel (header/footer toggles + params)

- [ ] **Step 1: Failing test** — given a `subBlockCatalogue: [{key,label,params:[{name,type}]}]` prop, toggling a header sub-block adds it to `headerSubBlocks` array; editing a param writes into `subBlockParams[key][name]`.

- [ ] **Step 2: FAIL**

- [ ] **Step 3: Implement.**

- [ ] **Step 4: PASS**

- [ ] **Step 5: Commit**

```bash
rtk git commit -am "feat(templates-ui): composition config panel"
```

### Task 19.4: Drag-to-insert placeholder into canvas

- [ ] **Step 1: Failing test** — simulating a drag of a placeholder chip onto the canvas inserts a new inlineSdt at the drop position via `placeholderToRun`.

- [ ] **Step 2: FAIL**

- [ ] **Step 3: Implement** — extend existing template-mode canvas wrapper with drop-handler.

- [ ] **Step 4: PASS**

- [ ] **Step 5: Commit**

```bash
rtk git commit -am "feat(templates-ui): drag-to-insert placeholder in canvas"
```

### Task 19.5: Drag-to-insert zone bookmarks into canvas

Same shape; inserts zone-start/zone-end bookmark pair via existing `wrapZone`.

- [ ] Commit:

```bash
rtk git commit -am "feat(templates-ui): drag-to-insert editable zone"
```

### Task 19.6: Wire inspector saves to `PUT /api/v2/templates/{id}/schemas`

Existing endpoint handles schema updates (grep `UpdateSchemas` handler in templates_v2). Task = client hook `useTemplateSchemas(templateId)` with mutate fn.

- [ ] **Step 1: Failing test** using MSW to mock endpoint.
- [ ] **Step 2: FAIL**
- [ ] **Step 3: Implement.**
- [ ] **Step 4: PASS**
- [ ] **Step 5: Commit**

```bash
rtk git commit -am "feat(templates-ui): schema save hook"
```

---

## Phase 20 — Document fill-in UI (Codex medium)

Spec §Fill-in flow: eigenpal canvas loads snapshotted body DOCX; user edits placeholders (typed inputs per `sdtType`) and zones; each edit upserts row via new endpoints.

### Task 20.1: Load snapshotted DOCX + values + zones on document open

**Files:**
- Create: `frontend/apps/web/src/features/documents/fill-in-loader.ts`
- Test: `frontend/apps/web/src/features/documents/__tests__/fill-in-loader.test.ts`

- [ ] **Step 1: Failing test** — given mocked responses for `GET /api/v2/documents/{id}` (returns snapshot schema refs + S3 presigned URL for body), `GET /placeholders` (returns current values), `GET /zones` (returns OOXML), loader returns `{bodyDocx: Uint8Array, placeholders, zones, schema}`.

- [ ] **Step 2: FAIL**

- [ ] **Step 3: Implement** loader. Uses `fetch` with presigned S3 URL for body; concurrent requests via `Promise.all`.

- [ ] **Step 4: PASS**

- [ ] **Step 5: Commit**

```bash
rtk git commit -am "feat(documents-ui): fill-in loader"
```

### Task 20.2: Placeholder edit → `PUT /placeholders/{pid}` with optimistic update

- [ ] **Step 1: Failing test** — typing into an SDT whose `sdtType === "date"` validates YYYY-MM-DD client-side; debounced PUT after 400ms; on 422 response, shows inline error with `rule` + `message` from backend.

- [ ] **Step 2: FAIL**

- [ ] **Step 3: Implement** — hook `usePlaceholderValue(docId, placeholderId)` returning `{value, setValue, error, saving}`. Debounce + optimistic update + rollback on error.

- [ ] **Step 4: PASS**

- [ ] **Step 5: Commit**

```bash
rtk git commit -am "feat(documents-ui): placeholder edit with debounced save + validation errors"
```

### Task 20.3: Zone edit → `PUT /zones/{zid}` with ContentPolicy-aware toolbar

- [ ] **Step 1: Failing test** — zone with `AllowTables=false` hides the table-insert toolbar button; on edit, zone content OOXML upserts via PUT.

- [ ] **Step 2: FAIL**

- [ ] **Step 3: Implement** — zone toolbar component reads `ContentPolicy`; eigenpal canvas receives restricted command list per zone.

- [ ] **Step 4: PASS**

- [ ] **Step 5: Commit**

```bash
rtk git commit -am "feat(documents-ui): zone edit with content policy toolbar"
```

### Task 20.4: Computed placeholder live refresh on user-placeholder change

- [ ] **Step 1: Failing test** — after user placeholder `p_area` change, computed `controlled_by_area` refetches via `GET /api/v2/documents/{id}/placeholders` and its SDT display updates without page reload.

- [ ] **Step 2: FAIL**

- [ ] **Step 3: Implement** — after every successful PUT of a user placeholder, invalidate the placeholders query; backend's `FillInService.SetPlaceholderValue` tail (Task 5b.2) has already run `DraftResolverService.ResolveComputedIfStale`.

- [ ] **Step 4: PASS**

- [ ] **Step 5: Commit**

```bash
rtk git commit -am "feat(documents-ui): computed placeholder live refresh"
```

### Task 20.5: Submit-for-approval button validates all required filled client-side before POST

- [ ] **Step 1: Failing test** — with one required placeholder empty, submit button is disabled + shows error list; all filled → enabled + POST `/api/v2/documents/{id}/submit` succeeds.

- [ ] **Step 2: FAIL**

- [ ] **Step 3: Implement.**

- [ ] **Step 4: PASS**

- [ ] **Step 5: Commit**

```bash
rtk git commit -am "feat(documents-ui): submit with client-side required validation"
```

### Task 20.6: E2E Playwright scenario (Draft → Fill → Submit → Signoff → Viewer)

- [ ] **Step 1: Failing Playwright spec** — author fills placeholders + zone, submits; reviewer signs; admin publishes; consumer opens viewer and sees PDF iframe.

- [ ] **Step 2: FAIL**

- [ ] **Step 3: Implement** — extend existing Playwright config in `frontend/apps/web/tests/e2e/`.

- [ ] **Step 4: PASS** — `rtk playwright test`

- [ ] **Step 5: Commit**

```bash
rtk git commit -am "test(e2e): placeholder fill-in + approval + viewer"
```

---

## Phase 18 — Final review gate (Opus)

**Non-implementing phase.** Opus reviews:

1. Every spec §section maps to ≥ 1 task (spec-coverage check).
2. No `content_hash` / `final_docx_s3_key` mutation outside Phase 11 create path (triple-hash invariant).
3. `FreezeService.Freeze` is invoked exactly once per revision approval (idempotency).
4. All 7 resolvers registered and version-pinned.
5. `filterTransaction` + SDT locks both present (belt-and-suspenders).

If issues found, produce `docs/superpowers/plans/followups/placeholder-fanout-gaps.md` with punch list; otherwise proceed to handoff.

---

## Self-Review Checklist (run before Codex handoff)

1. **Spec coverage:**
   - §Placeholders → Phase 1 + 3 (types + validation).
   - §Editable zones → Phase 1 + 5 + 6.
   - §Composition config → Phase 1 + 8.
   - §Computed resolvers → Phase 7.
   - §Editor UX → Phase 15.
   - §Template author flow → covered by existing templates_v2 UpdateSchemas + Phase 3 validation extensions.
   - §Document create flow → Phase 4.
   - §Fill-in flow → Phase 5.
   - §Approval freeze + fanout → Phases 9, 10, 11, 16.
   - §Viewer flow → Phase 13.
   - §Re-render → Phase 14.
   - §Schema (new columns + tables) → Phase 2.
   - §Freeze Model (double-freeze + triple-hash) → Phase 2 (columns) + Phase 4 (first freeze) + Phase 9/16 (second freeze + binding).
   - §Error Handling → Phases 3 (template-save errors), 5 (fill-in errors), 9 (freeze errors), 14 (reconstruction).
   - §Testing Approach → Phase 6 + 17.
   - §Invariant "content_hash immutable post-approval" → Phase 14 enforces via append-only.

2. **Placeholder scan:** no "TBD", no "similar to", no "implement later" in this document.

3. **Type consistency:** `Placeholder`, `EditableZone`, `ContentPolicy`, `CompositionConfig`, `VisibilityCondition`, `TemplateSnapshot`, `SnapshotHashes`, `PlaceholderValue`, `ZoneContent`, `ResolveInput`, `ResolvedValue`, `ComputedResolver`, `Registry`, `SubBlockRenderer`, `SubBlockRegistry`, `FanoutInput`, `FanoutResult`, `FanoutRequest`, `FanoutResponse`, `SignaturePayload` — all referenced consistently across tasks.

---

## Execution Handoff

Plan complete and saved to `docs/superpowers/plans/2026-04-21-foundation-placeholder-fill-in-design.md`. Two execution options:

1. **Subagent-Driven (recommended)** — fresh Codex subagent per task, Opus reviews between phases, fast iteration.
2. **Inline Execution** — tasks executed in this session using `superpowers:executing-plans`, batch with checkpoints.

Which approach?
