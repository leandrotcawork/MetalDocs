# Templates v2 WYSIWYG Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Ship in-app WYSIWYG template authoring via eigenpal on a new `templates_v2` module, with ISO-9001 metadata, approval flow, snapshot-on-create versioning, and document fill-in flow.

**Architecture:** New backend module `internal/modules/templates_v2` mirroring `documents_v2` (domain/app/repo/delivery). New frontend area `features/templates/v2`. Eigenpal `DocxEditor` gains `mode="template" | "document"` prop. Documents reference template version via FK, but snapshot body + schemas on creation.

**Tech Stack:** Go 1.22 + pgx/v5, React + TypeScript + TanStack Query, MinIO, PostgreSQL, @eigenpal/docx-js-editor, Vitest, Playwright.

**Spec:** [docs/superpowers/specs/2026-04-20-templates-v2-wysiwyg-design.md](../specs/2026-04-20-templates-v2-wysiwyg-design.md)

---

## Execution Workflow (MANDATORY)

Every coding task delegates to **Codex `gpt-5.3-codex`** via `mcp__codex__codex`. Claude Sonnet 4.6 only for trivial low-token edits explicitly marked `Sonnet-OK`. Opus coordinates + reviews per phase.

**Per-task procedure for the controller session:**

1. Read the task card — note `Files`, `Tests`, `Implementor`, and the **Codex prompt**.
2. If `Implementor: Codex`, call `mcp__codex__codex` with:
   - `model: "gpt-5.3-codex"`
   - `reasoning_effort: "medium"` (bump to `high` for state-machine / concurrency tasks explicitly flagged `high-effort`)
   - `prompt`: the caveman-register prompt from the card, **prepended** with this guardrail:
     > `Constraints: TDD. Write failing test first. Run it. Write minimal impl. Run tests green. No out-of-scope edits. Follow exact paths in prompt. No emojis. No new comments unless non-obvious.`
3. Read Codex output diff. If diff extends beyond stated files → reject, re-prompt tighter.
4. Controller runs tests (exact command in card). If green → controller commits with the commit message shown in the card.
5. If red → feed failure output back to Codex with one-line: `test red: <paste>. fix.`. Max 3 Codex rounds per task. After round 3, escalate to Opus.
6. At **end of each phase**: dispatch `nexus:code-reviewer` subagent (Opus) to review all diffs of that phase against the spec. Block next phase on un-addressed findings.

**Codex prompt register:** caveman-full (drop articles, fragments, imperative). Code blocks, paths, commands stay normal register.

---

## Phase 0 — Eigenpal capability spike

Goal: confirm `@eigenpal/docx-js-editor` can serialize custom inline runs (placeholder chips) and custom sections (editable zones) through DOCX save/load without data loss. Blocks rest of plan.

### Task 0.1: Spike eigenpal custom inline run

**Files:**
- Create: `frontend/apps/web/src/editor-adapters/__spike__/eigenpal-placeholder-spike.test.ts`
- Create: `frontend/apps/web/src/editor-adapters/eigenpal-template-mode.ts` (stub — only what spike needs)

**Implementor:** Codex (`gpt-5.3-codex`, `reasoning_effort: high`)

**Codex prompt:**
```
Goal: prove @eigenpal/docx-js-editor supports round-trip of a custom inline run representing a template placeholder chip.

Write test `frontend/apps/web/src/editor-adapters/__spike__/eigenpal-placeholder-spike.test.ts` (Vitest) that:
1. Imports DocxEditor + its DOCX serializer from @eigenpal/docx-js-editor.
2. Builds a minimal document model containing a custom inline run tagged as placeholder with id="customer_name".
3. Serializes to DOCX bytes, parses back, asserts the custom run survives with id intact.

Create `frontend/apps/web/src/editor-adapters/eigenpal-template-mode.ts` exposing:
  export type PlaceholderRun = { type: "placeholder"; id: string; label: string };
  export function placeholderToRun(p: PlaceholderRun): <eigenpal inline run node>;
  export function runToPlaceholder(node: <eigenpal inline run node>): PlaceholderRun | null;

Use whatever extension mechanism eigenpal exposes (inspect node_modules/@eigenpal/docx-js-editor/package.json + exports + README). If no extension point exists, the test MUST fail with a clear error and a comment documenting which eigenpal API is missing. Do NOT invent an API.

Constraints:
- Vitest only. No React render in this spike.
- Exact file paths above.
- Keep runToPlaceholder pure.
```

**Verify:**
```bash
cd frontend/apps/web && rtk pnpm vitest run src/editor-adapters/__spike__/eigenpal-placeholder-spike.test.ts
```
Expected: GREEN → eigenpal supports it, proceed. RED with documented missing API → stop plan, escalate to Opus for alternative (fork eigenpal or pivot to external chip overlay).

**Commit (on green):**
```
feat(templates-v2): P0.1 eigenpal placeholder round-trip spike
```

### Task 0.2: Spike eigenpal editable zone

**Files:**
- Create: `frontend/apps/web/src/editor-adapters/__spike__/eigenpal-zone-spike.test.ts`
- Modify: `frontend/apps/web/src/editor-adapters/eigenpal-template-mode.ts` — add `EditableZone` type + wrap/unwrap helpers.

**Implementor:** Codex (`gpt-5.3-codex`, high)

**Codex prompt:**
```
Goal: prove eigenpal supports wrapping a range of blocks in a custom section marker representing an editable zone.

Write test `frontend/apps/web/src/editor-adapters/__spike__/eigenpal-zone-spike.test.ts`:
1. Build doc with 3 paragraphs.
2. Wrap paragraph 2 in a zone with id="observations".
3. Serialize → DOCX → parse back. Assert zone marker survives; paragraph 2 still inside.

Extend `eigenpal-template-mode.ts`:
  export type EditableZone = { id: string; label: string };
  export function wrapZone(zoneId: string, blocks: BlockNode[]): ZoneNode;
  export function extractZone(node: ZoneNode): { zone: EditableZone; blocks: BlockNode[] } | null;

Same constraints as 0.1. If eigenpal can't wrap blocks, fail with documented missing API.
```

**Verify:** `rtk pnpm vitest run src/editor-adapters/__spike__/eigenpal-zone-spike.test.ts`
Expected: GREEN.

**Commit:** `feat(templates-v2): P0.2 eigenpal editable-zone spike`

### Task 0.3: Opus phase review

**Implementor:** Opus via `Agent(subagent_type="nexus:code-reviewer")`

**Prompt:**
```
Review Phase 0 (eigenpal spikes). Files: frontend/apps/web/src/editor-adapters/**/*. Spec: docs/superpowers/specs/2026-04-20-templates-v2-wysiwyg-design.md (Risks section, item 1).

Assess: (1) do the spikes truly prove round-trip, or just construct+deconstruct in-memory without DOCX bytes? (2) Any red flags in the eigenpal API the team should know before Phase 6? (3) Should the plan proceed or does the eigenpal extension story need rework?

Reply: PROCEED | BLOCK with specific concerns.
```

Block next phase if review returns BLOCK.

---

## Phase 1 — Backend domain + migration

Goal: DB schema + Go domain types + error set. No service or handlers yet.

### Task 1.1: Migration — core tables

**Files:**
- Create: `migrations/0118_templates_v2_init.sql`

**Implementor:** Codex (`gpt-5.3-codex`, medium)

**Codex prompt:**
```
Write PostgreSQL migration `migrations/0118_templates_v2_init.sql`. Use exact DDL below. No extra columns. No extra indexes beyond what shown.

CREATE TABLE templates_v2_template (
  id                    uuid PRIMARY KEY,
  tenant_id             text NOT NULL,
  doc_type_code         text NOT NULL,
  key                   text NOT NULL,
  name                  text NOT NULL,
  description           text NOT NULL DEFAULT '',
  areas                 text[] NOT NULL DEFAULT '{}',
  visibility            text NOT NULL,
  specific_areas        text[] NOT NULL DEFAULT '{}',
  latest_version        int NOT NULL DEFAULT 0,
  published_version_id  uuid NULL,
  created_by            text NOT NULL,
  created_at            timestamptz NOT NULL DEFAULT now(),
  archived_at           timestamptz NULL,
  UNIQUE (tenant_id, key)
);

CREATE TABLE templates_v2_template_version (
  id                  uuid PRIMARY KEY,
  template_id         uuid NOT NULL REFERENCES templates_v2_template(id),
  version_number      int  NOT NULL,
  status              text NOT NULL,
  docx_storage_key    text NOT NULL,
  content_hash        text NOT NULL,
  metadata_schema     jsonb NOT NULL,
  placeholder_schema  jsonb NOT NULL,
  editable_zones      jsonb NOT NULL,
  author_id               text NOT NULL,
  pending_reviewer_role   text NULL,
  pending_approver_role   text NOT NULL DEFAULT '',
  reviewer_id             text NULL,
  approver_id             text NULL,
  submitted_at        timestamptz NULL,
  reviewed_at         timestamptz NULL,
  approved_at         timestamptz NULL,
  published_at        timestamptz NULL,
  obsoleted_at        timestamptz NULL,
  created_at          timestamptz NOT NULL DEFAULT now(),
  UNIQUE (template_id, version_number)
);

ALTER TABLE templates_v2_template
  ADD CONSTRAINT fk_templates_v2_published_version
  FOREIGN KEY (published_version_id) REFERENCES templates_v2_template_version(id);

CREATE TABLE templates_v2_approval_config (
  template_id     uuid PRIMARY KEY REFERENCES templates_v2_template(id),
  reviewer_role   text NULL,
  approver_role   text NOT NULL
);

CREATE TABLE templates_v2_audit_log (
  id            bigserial PRIMARY KEY,
  tenant_id     text NOT NULL,
  template_id   uuid NOT NULL,
  version_id    uuid NULL,
  actor_id      text NOT NULL,
  action        text NOT NULL,
  details       jsonb NOT NULL DEFAULT '{}',
  occurred_at   timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_templates_v2_template_tenant_doctype ON templates_v2_template (tenant_id, doc_type_code);
CREATE INDEX idx_templates_v2_version_template_status ON templates_v2_template_version (template_id, status);
CREATE INDEX idx_templates_v2_audit_template_time ON templates_v2_audit_log (template_id, occurred_at DESC);
```

**Verify:**
```bash
cd apps/api && rtk go run ./cmd/metaldocs-api migrate up
rtk psql $DATABASE_URL -c "\dt templates_v2_*"
```
Expected: 4 tables listed.

**Commit:** `feat(templates-v2): P1.1 migration 0118 core tables`

### Task 1.2: Migration — documents_v2 FK

**Files:**
- Create: `migrations/0119_documents_v2_link_template_version.sql`

**Implementor:** Codex (medium)

**Codex prompt:**
```
Write migration `migrations/0119_documents_v2_link_template_version.sql`:

ALTER TABLE documents_v2_documents
  ADD COLUMN templates_v2_template_version_id uuid NULL
  REFERENCES templates_v2_template_version(id);

CREATE INDEX idx_documents_v2_template_version ON documents_v2_documents (templates_v2_template_version_id) WHERE templates_v2_template_version_id IS NOT NULL;
```

**Verify:**
```bash
cd apps/api && rtk go run ./cmd/metaldocs-api migrate up
rtk psql $DATABASE_URL -c "\d documents_v2_documents" | rtk grep templates_v2_template_version_id
```
Expected: column listed.

**Commit:** `feat(templates-v2): P1.2 migration 0119 docs_v2 FK`

### Task 1.3: Domain types — Template + Version

**Files:**
- Create: `internal/modules/templates_v2/domain/template.go`
- Create: `internal/modules/templates_v2/domain/version.go`
- Create: `internal/modules/templates_v2/domain/schemas.go`
- Create: `internal/modules/templates_v2/domain/template_test.go`
- Create: `internal/modules/templates_v2/domain/version_test.go`

**Implementor:** Codex (high)

**Codex prompt:**
```
Create Go domain package `internal/modules/templates_v2/domain` (module path: metaldocs/internal/modules/templates_v2/domain).

Files:

template.go — define:
  type Visibility string
  const (
    VisibilityPublic   Visibility = "public"
    VisibilityInternal Visibility = "internal"
    VisibilitySpecific Visibility = "specific"
  )

  type Template struct {
    ID                  string
    TenantID            string
    DocTypeCode         string
    Key                 string
    Name                string
    Description         string
    Areas               []string
    Visibility          Visibility
    SpecificAreas       []string
    LatestVersion       int
    PublishedVersionID  *string
    CreatedBy           string
    CreatedAt           time.Time
    ArchivedAt          *time.Time
  }

  func (t *Template) IsArchived() bool { return t.ArchivedAt != nil }

  Errors (var ErrX = errors.New("templates_v2: X")):
    ErrNotFound, ErrKeyConflict, ErrInvalidVisibility, ErrArchived.

version.go — define:
  type VersionStatus string
  const (
    StatusDraft     VersionStatus = "draft"
    StatusInReview  VersionStatus = "in_review"
    StatusApproved  VersionStatus = "approved"
    StatusPublished VersionStatus = "published"
    StatusObsolete  VersionStatus = "obsolete"
  )

  type TemplateVersion struct {
    ID, TemplateID string
    VersionNumber  int
    Status         VersionStatus
    DocxStorageKey string
    ContentHash    string
    MetadataSchema    MetadataSchema
    PlaceholderSchema []Placeholder
    EditableZones     []EditableZone
    AuthorID             string
    PendingReviewerRole  *string  // nil = no reviewer stage snapshot
    PendingApproverRole  string   // "" until submitted
    ReviewerID, ApproverID *string
    SubmittedAt, ReviewedAt, ApprovedAt, PublishedAt, ObsoletedAt *time.Time
    CreatedAt      time.Time
  }

  // State transition function — pure, returns new status or error.
  func (v *TemplateVersion) CanTransition(next VersionStatus, hasReviewer bool) error
  // Allowed:
  //   draft -> in_review
  //   in_review -> approved (if hasReviewer); or in_review -> published via approve when no reviewer (handled at app layer)
  //   approved -> published
  //   published -> obsolete
  //   in_review|approved -> draft (reject)
  //   any -> any otherwise returns ErrInvalidStateTransition

  Errors: ErrInvalidStateTransition, ErrContentHashMismatch, ErrStaleBase.

schemas.go — define:
  type MetadataSchema struct {
    DocCodePattern       string   `json:"doc_code_pattern"`
    RetentionDays        int      `json:"retention_days"`
    DistributionDefault  []string `json:"distribution_default"`
    RequiredMetadata     []string `json:"required_metadata"`
  }

  type PlaceholderType string
  const (
    PHText PlaceholderType = "text"
    PHDate PlaceholderType = "date"
    PHNumber PlaceholderType = "number"
    PHSelect PlaceholderType = "select"
    PHUser PlaceholderType = "user"
  )

  type Placeholder struct {
    ID       string          `json:"id"`
    Label    string          `json:"label"`
    Type     PlaceholderType `json:"type"`
    Required bool            `json:"required"`
    Default  any             `json:"default,omitempty"`
    Options  []string        `json:"options,omitempty"` // for PHSelect
  }

  type EditableZone struct {
    ID       string `json:"id"`
    Label    string `json:"label"`
    Required bool   `json:"required"`
  }

TDD: write `template_test.go` first with:
  - TestTemplate_IsArchived_TrueWhenSet
  - TestTemplate_IsArchived_FalseWhenNil

Write `version_test.go` with table-driven TestCanTransition covering every allowed + rejected transition.

Constraints: only files listed. No service/repo code. All exported symbols documented with one-line doc comments only when non-obvious.
```

**Verify:**
```bash
cd apps/api && rtk go test ./internal/modules/templates_v2/domain/...
```
Expected: PASS.

**Commit:** `feat(templates-v2): P1.3 domain types + state machine`

### Task 1.4: Domain — ApprovalConfig + segregation checks

**Files:**
- Create: `internal/modules/templates_v2/domain/approval.go`
- Create: `internal/modules/templates_v2/domain/approval_test.go`

**Implementor:** Codex (medium)

**Codex prompt:**
```
File `internal/modules/templates_v2/domain/approval.go`:

type ApprovalConfig struct {
  TemplateID     string
  ReviewerRole   *string
  ApproverRole   string
}

func (c ApprovalConfig) HasReviewer() bool { return c.ReviewerRole != nil && *c.ReviewerRole != "" }

// CheckSegregation enforces ISO segregation of duties.
// actorID = user attempting action. role = "reviewer" | "approver".
// authorID, reviewerID = already-recorded ids (reviewerID may be nil).
// Returns ErrISOSegregationViolation on conflict.
func CheckSegregation(role string, actorID, authorID string, reviewerID *string) error

Rules:
- role="reviewer": actorID != authorID.
- role="approver": actorID != authorID AND (reviewerID == nil OR actorID != *reviewerID).

Add errors:
  var ErrISOSegregationViolation = errors.New("templates_v2: iso_segregation_violation")
  var ErrForbiddenRole = errors.New("templates_v2: forbidden_role")

TDD: approval_test.go table-driven covering:
  reviewer OK when distinct from author
  reviewer fail when == author
  approver OK when distinct from both
  approver fail when == author
  approver fail when == reviewer
  approver OK when reviewerID nil and distinct from author
```

**Verify:** `rtk go test ./internal/modules/templates_v2/domain/...`
Expected: PASS.

**Commit:** `feat(templates-v2): P1.4 approval segregation rules`

### Task 1.5: Domain — audit event type

**Files:**
- Create: `internal/modules/templates_v2/domain/audit.go`

**Implementor:** Sonnet-OK (small, no tests needed — type-only)

**Prompt (Sonnet 4.6):**
```
Create internal/modules/templates_v2/domain/audit.go defining:

type AuditAction string
const (
  AuditCreated    AuditAction = "created"
  AuditSaved      AuditAction = "saved"
  AuditSubmitted  AuditAction = "submitted"
  AuditReviewed   AuditAction = "reviewed"
  AuditApproved   AuditAction = "approved"
  AuditRejected   AuditAction = "rejected"
  AuditPublished  AuditAction = "published"
  AuditObsoleted  AuditAction = "obsoleted"
  AuditArchived   AuditAction = "archived"
  AuditRestored   AuditAction = "restored"
)

type AuditEvent struct {
  TenantID   string
  TemplateID string
  VersionID  *string
  ActorID    string
  Action     AuditAction
  Details    map[string]any
  OccurredAt time.Time
}
```

**Verify:** `rtk go build ./internal/modules/templates_v2/domain/...`

**Commit:** `feat(templates-v2): P1.5 audit event type`

### Task 1.6: Opus phase review

**Implementor:** Opus via `nexus:code-reviewer`

**Prompt:**
```
Review Phase 1 (backend domain + migrations). Files: migrations/0118*, 0119*, internal/modules/templates_v2/domain/*. Spec: docs/superpowers/specs/2026-04-20-templates-v2-wysiwyg-design.md (Data model, Lifecycles, Approval flow).

Check: schema matches spec exactly; state machine covers all transitions in spec; segregation rules match spec rules; no out-of-scope columns or states.

Reply: PROCEED | BLOCK with specific concerns.
```

---

## Phase 2 — Backend application layer

Goal: service struct + commands + queries. Uses ports (interfaces) for repo + presigner + audit + clock + UUID. Pure, no pgx dependency.

### Task 2.1: Service scaffold + ports

**Files:**
- Create: `internal/modules/templates_v2/application/service.go`
- Create: `internal/modules/templates_v2/application/ports.go`

**Implementor:** Codex (medium)

**Codex prompt:**
```
Create application package internal/modules/templates_v2/application.

ports.go: define interfaces.

type Repository interface {
  CreateTemplate(ctx context.Context, t *domain.Template) error
  GetTemplate(ctx context.Context, tenantID, id string) (*domain.Template, error)
  GetTemplateByKey(ctx context.Context, tenantID, key string) (*domain.Template, error)
  ListTemplates(ctx context.Context, f ListFilter) ([]*domain.Template, error)
  UpdateTemplate(ctx context.Context, t *domain.Template) error

  CreateVersion(ctx context.Context, v *domain.TemplateVersion) error
  GetVersion(ctx context.Context, templateID string, n int) (*domain.TemplateVersion, error)
  GetVersionByID(ctx context.Context, id string) (*domain.TemplateVersion, error)
  UpdateVersion(ctx context.Context, v *domain.TemplateVersion) error
  ObsoletePreviousPublished(ctx context.Context, templateID, keepVersionID string) error

  GetApprovalConfig(ctx context.Context, templateID string) (*domain.ApprovalConfig, error)
  UpsertApprovalConfig(ctx context.Context, c *domain.ApprovalConfig) error

  AppendAudit(ctx context.Context, e *domain.AuditEvent) error
  ListAudit(ctx context.Context, templateID string, limit, offset int) ([]*domain.AuditEvent, error)
}

type Presigner interface {
  PresignPUT(ctx context.Context, key string, expires time.Duration) (url string, err error)
  HeadContentHash(ctx context.Context, key string) (string, error)
  Delete(ctx context.Context, key string) error
}

type Clock interface { Now() time.Time }
type UUIDGen interface { New() string }

type ListFilter struct {
  TenantID      string
  AreaAny       []string   // OR match against areas[]
  DocTypeCode   *string
  Status        *domain.VersionStatus
  Limit, Offset int
}

service.go:

type Service struct {
  repo Repository
  presign Presigner
  clock Clock
  uuid UUIDGen
}

func New(repo Repository, presign Presigner, clock Clock, uuid UUIDGen) *Service {
  return &Service{repo: repo, presign: presign, clock: clock, uuid: uuid}
}

No commands yet — just struct + ports. Build must succeed.
```

**Verify:** `rtk go build ./internal/modules/templates_v2/application/...`

**Commit:** `feat(templates-v2): P2.1 application scaffold + ports`

### Task 2.2: CreateTemplate + CreateVersion

**Files:**
- Create: `internal/modules/templates_v2/application/create.go`
- Create: `internal/modules/templates_v2/application/create_test.go`
- Create: `internal/modules/templates_v2/application/fakes_test.go`

**Implementor:** Codex (high)

**Codex prompt:**
```
Files: internal/modules/templates_v2/application/create.go, create_test.go, fakes_test.go.

In create.go implement:

type CreateTemplateCmd struct {
  TenantID      string
  ActorUserID   string
  DocTypeCode   string
  Key           string
  Name          string
  Description   string
  Areas         []string
  Visibility    domain.Visibility
  SpecificAreas []string
  ApproverRole  string
  ReviewerRole  *string  // nil = no reviewer stage
}

type CreateTemplateResult struct {
  Template *domain.Template
  Version  *domain.TemplateVersion
}

func (s *Service) CreateTemplate(ctx context.Context, cmd CreateTemplateCmd) (*CreateTemplateResult, error)

Behavior:
1. Validate cmd.Visibility in {public, internal, specific}. If invalid return domain.ErrInvalidVisibility.
2. Check key uniqueness: repo.GetTemplateByKey; if found return domain.ErrKeyConflict.
3. Build Template with id=s.uuid.New(), CreatedBy=ActorUserID, CreatedAt=s.clock.Now(), LatestVersion=1, PublishedVersionID=nil.
4. Build version 1 with: status=draft, VersionNumber=1, AuthorID=ActorUserID, DocxStorageKey=fmt.Sprintf("templates/%s/versions/1.docx", template.ID), ContentHash="", empty schemas {}/[]/[], CreatedAt=s.clock.Now().
5. repo.CreateTemplate then repo.CreateVersion (not transactional at app layer; repo may wrap). 
6. repo.UpsertApprovalConfig with cmd.ApproverRole + ReviewerRole.
7. repo.AppendAudit with action=created, ActorID, Details={}.
8. Return {Template, Version}.

Also implement:

type CreateVersionCmd struct {
  TenantID    string
  ActorUserID string
  TemplateID  string
  // body + schemas cloned from latest published if exists else from latest version
}

func (s *Service) CreateNextVersion(ctx context.Context, cmd CreateVersionCmd) (*domain.TemplateVersion, error)

Behavior:
1. Load template. If archived: ErrArchived.
2. Find source version: template.PublishedVersionID if set else latest version (template.LatestVersion).
3. Build new version: number=template.LatestVersion+1, status=draft, AuthorID=ActorUserID, schemas+docxStorageKey cloned; ContentHash="", DocxStorageKey=fmt.Sprintf("templates/%s/versions/%d.docx", templateID, newNum).
4. repo.CreateVersion, update template.LatestVersion, repo.UpdateTemplate, repo.AppendAudit(created, VersionID=newID).

TDD create_test.go:
- TestCreateTemplate_Happy — ok, returns template + v1, audit appended, approval config persisted.
- TestCreateTemplate_KeyConflict — repo.GetTemplateByKey returns existing; expect ErrKeyConflict.
- TestCreateTemplate_InvalidVisibility — visibility="weird"; expect ErrInvalidVisibility.
- TestCreateNextVersion_FromPublished — template with PublishedVersionID set; new version clones published.
- TestCreateNextVersion_NoPublished_ClonesLatest — template with only draft v1; new v2 clones v1 schemas.
- TestCreateNextVersion_Archived — ArchivedAt set; expect ErrArchived.

Write fakes_test.go with fakeRepo, fakePresigner, fakeClock, fakeUUID supporting calls needed by all application tests in this phase. Use a map[string]*domain.Template keyed by id, map for versions, slice for audit events. Counters for uuid (returns fmt.Sprintf("id_%d", n)). Clock returns fixed time.Date(2026,4,20,12,0,0,0,time.UTC) unless overridden.
```

**Verify:** `rtk go test ./internal/modules/templates_v2/application/...`
Expected: PASS.

**Commit:** `feat(templates-v2): P2.2 CreateTemplate + CreateNextVersion`

### Task 2.3: Update schemas (metadata, placeholders, zones)

**Files:**
- Create: `internal/modules/templates_v2/application/schema.go`
- Create: `internal/modules/templates_v2/application/schema_test.go`

**Implementor:** Codex (medium)

**Codex prompt:**
```
File internal/modules/templates_v2/application/schema.go.

type UpdateSchemasCmd struct {
  TenantID, ActorUserID, TemplateID string
  VersionNumber                     int
  MetadataSchema                    domain.MetadataSchema
  PlaceholderSchema                 []domain.Placeholder
  EditableZones                     []domain.EditableZone
  ExpectedContentHash               string  // optimistic lock; empty = no check
}

func (s *Service) UpdateSchemas(ctx context.Context, cmd UpdateSchemasCmd) (*domain.TemplateVersion, error)

Behavior:
1. Load version. If not found: domain.ErrNotFound.
2. Version.Status must be draft else domain.ErrInvalidStateTransition.
3. If cmd.ExpectedContentHash != "" and != v.ContentHash: ErrStaleBase.
4. Validate placeholder ids unique; error "duplicate_placeholder_id".
5. Validate zone ids unique; error "duplicate_zone_id".
6. Assign schemas to version; repo.UpdateVersion; AppendAudit(saved, details={"kind":"schema"}).

TDD schema_test.go:
  Happy (draft state, no hash check, no dupes): OK.
  Non-draft: ErrInvalidStateTransition.
  Stale hash: ErrStaleBase.
  Duplicate placeholder id: error contains "duplicate_placeholder_id".
  Duplicate zone id: error contains "duplicate_zone_id".
```

**Verify:** `rtk go test ./internal/modules/templates_v2/application/...`

**Commit:** `feat(templates-v2): P2.3 UpdateSchemas`

### Task 2.4: Autosave presign + commit

**Files:**
- Create: `internal/modules/templates_v2/application/autosave.go`
- Create: `internal/modules/templates_v2/application/autosave_test.go`

**Implementor:** Codex (high)

**Codex prompt:**
```
File internal/modules/templates_v2/application/autosave.go.

type PresignAutosaveCmd struct {
  TenantID, ActorUserID, TemplateID string
  VersionNumber                     int
}
type PresignAutosaveResult struct {
  UploadURL   string
  StorageKey  string
  ExpiresAt   time.Time
}
func (s *Service) PresignAutosave(ctx context.Context, cmd PresignAutosaveCmd) (*PresignAutosaveResult, error)

Behavior:
1. Load template + version. Check version.Status == draft else ErrInvalidStateTransition.
2. key := version.DocxStorageKey.
3. url, err := s.presign.PresignPUT(ctx, key, 10*time.Minute).
4. Return {url, key, now+10min}.

type CommitAutosaveCmd struct {
  TenantID, ActorUserID, TemplateID string
  VersionNumber                     int
  ExpectedContentHash               string  // client-computed hash of body
}
func (s *Service) CommitAutosave(ctx context.Context, cmd CommitAutosaveCmd) (*domain.TemplateVersion, error)

Behavior:
1. Load version. Status must be draft.
2. actualHash, err := s.presign.HeadContentHash(ctx, version.DocxStorageKey). If err == domain.ErrUploadMissing return ErrUploadMissing.
3. If actualHash != cmd.ExpectedContentHash return domain.ErrContentHashMismatch AND call s.presign.Delete(ctx, key) as orphan cleanup.
4. version.ContentHash = actualHash; repo.UpdateVersion; AppendAudit(saved, details={"content_hash":actualHash}).
5. Return updated version.

Reuse ErrUploadMissing from domain (add if missing: var ErrUploadMissing = errors.New("templates_v2: upload_missing")).

TDD autosave_test.go:
  Presign happy: draft → url + key returned.
  Presign on non-draft: ErrInvalidStateTransition.
  Commit happy: hash matches, version.ContentHash updated.
  Commit hash mismatch: ErrContentHashMismatch, orphan Delete called exactly once.
  Commit upload missing: ErrUploadMissing, no Delete.
```

**Verify:** `rtk go test ./internal/modules/templates_v2/application/...`

**Commit:** `feat(templates-v2): P2.4 autosave presign + commit`

### Task 2.5: Lifecycle — submit/review/approve/reject/publish/archive

**Files:**
- Create: `internal/modules/templates_v2/application/lifecycle.go`
- Create: `internal/modules/templates_v2/application/lifecycle_test.go`

**Implementor:** Codex (high)

**Codex prompt:**
```
File internal/modules/templates_v2/application/lifecycle.go.

Shared loader helper (unexported) loadVersionDraftLike(ctx, tenantID, templateID, n) loads both template + version, asserts tenant match else domain.ErrNotFound.

Commands:

SubmitForReviewCmd { TenantID, ActorUserID, TemplateID; VersionNumber int }
func (s *Service) SubmitForReview(ctx, cmd) (*domain.TemplateVersion, error)
Rules:
  1. version.Status must be draft.
  2. Load approval_config; snapshot onto version:
       version.PendingReviewerRole = config.ReviewerRole (clone)
       version.PendingApproverRole = config.ApproverRole
  3. Transition → in_review. Set SubmittedAt=now. repo.UpdateVersion. Audit action=submitted with details={"reviewer_role":config.ReviewerRole,"approver_role":config.ApproverRole}.

ReviewCmd { TenantID, ActorUserID, ActorRoles []string, TemplateID; VersionNumber int; Accept bool; Reason string }
func (s *Service) Review(ctx, cmd) (*domain.TemplateVersion, error)
Rules: use snapshot on version. If version.PendingReviewerRole == nil return ErrInvalidStateTransition (no reviewer stage snapshotted).
Version.Status must be in_review.
Actor must hold role == *version.PendingReviewerRole else ErrForbiddenRole (new domain error).
Enforce CheckSegregation("reviewer", ActorUserID, version.AuthorID, nil).
On Accept=true: transition → approved; ReviewerID=&ActorUserID; ReviewedAt=now; audit=reviewed.
On Accept=false: transition → draft; audit=rejected with details={"reason":Reason,"stage":"reviewer"}. Clear SubmittedAt.

ApproveCmd { TenantID, ActorUserID, ActorRoles []string, TemplateID; VersionNumber int; Accept bool; Reason string }
func (s *Service) Approve(ctx, cmd) (*domain.TemplateVersion, error)
Rules: use snapshot on version. Required status:
  - If version.PendingReviewerRole != nil: version.Status must be approved.
  - Else: version.Status must be in_review.
Actor must hold role == version.PendingApproverRole else ErrForbiddenRole.
Enforce CheckSegregation("approver", ActorUserID, version.AuthorID, version.ReviewerID).
On Accept=true:
  - transition → published; ApproverID=&ActorUserID; ApprovedAt=now; PublishedAt=now.
  - repo.ObsoletePreviousPublished(templateID, keepVersionID=version.ID) — sets prior published row to status=obsolete, obsoleted_at=now.
  - template.PublishedVersionID = &version.ID; repo.UpdateTemplate.
  - Audit=published.
On Accept=false:
  - transition → draft. Audit=rejected with details={"reason":Reason,"stage":"approver"}. Clear SubmittedAt+ReviewedAt+ApprovedAt.

ArchiveCmd { TenantID, ActorUserID, TemplateID }
func (s *Service) ArchiveTemplate(ctx, cmd) (*domain.Template, error)
Rules: template.ArchivedAt must be nil else already-archived (ignore — idempotent). Set ArchivedAt=now. repo.UpdateTemplate. Audit=archived (VersionID=nil).

TDD lifecycle_test.go covers every branch above + every segregation violation + every invalid transition. Table-driven where possible.
```

**Verify:** `rtk go test ./internal/modules/templates_v2/application/...`
Expected: PASS.

**Commit:** `feat(templates-v2): P2.5 lifecycle commands`

### Task 2.6: Queries

**Files:**
- Create: `internal/modules/templates_v2/application/queries.go`
- Create: `internal/modules/templates_v2/application/queries_test.go`

**Implementor:** Codex (medium)

**Codex prompt:**
```
File internal/modules/templates_v2/application/queries.go.

func (s *Service) GetTemplate(ctx, tenantID, id string) (*domain.Template, error)
func (s *Service) GetVersion(ctx, tenantID, templateID string, n int) (*domain.TemplateVersion, error)
func (s *Service) ListTemplates(ctx, f ListFilter) ([]*domain.Template, error)
func (s *Service) ListAudit(ctx, tenantID, templateID string, limit, offset int) ([]*domain.AuditEvent, error)

All are pure pass-through to repo plus tenant check (re-load template if needed to confirm tenant owns resource; return ErrNotFound otherwise).

TDD queries_test.go: happy path for each + cross-tenant returns ErrNotFound.
```

**Verify:** `rtk go test ./internal/modules/templates_v2/application/...`

**Commit:** `feat(templates-v2): P2.6 queries`

### Task 2.7: Opus phase review

**Implementor:** Opus via `nexus:code-reviewer`

**Prompt:**
```
Review Phase 2 (application layer). Files: internal/modules/templates_v2/application/*. Spec sections: Lifecycles, Approval flow, HTTP API (request shapes). Check: every command maps to spec; state transitions correct; segregation enforced; tests cover reject branches + stale hash + orphan delete; ports interface minimal and mockable.

Reply: PROCEED | BLOCK.
```

---

## Phase 3 — Backend repository (Postgres)

### Task 3.1: pgx Repository implementation

**Files:**
- Create: `internal/modules/templates_v2/repository/postgres.go`
- Create: `internal/modules/templates_v2/repository/mappers.go`
- Create: `internal/modules/templates_v2/repository/postgres_test.go`

**Implementor:** Codex (high)

**Codex prompt:**
```
Implement Postgres repository at internal/modules/templates_v2/repository.

postgres.go:

type Repository struct { pool *pgxpool.Pool }
func New(pool *pgxpool.Pool) *Repository { return &Repository{pool: pool} }

Implement every method of application.Repository interface. Use parameterized queries only. For UUID errors map pg SQLSTATE 22P02 to domain.ErrNotFound (mirror documents_v2 repo pattern — check internal/modules/documents_v2/repository/postgres.go for isInvalidUUID helper, copy it).

For ObsoletePreviousPublished(templateID, keepVersionID):
  UPDATE templates_v2_template_version
  SET status='obsolete', obsoleted_at=now()
  WHERE template_id=$1 AND status='published' AND id <> $2

For ListTemplates with ListFilter:
  WHERE tenant_id=$1
  AND ($2::text IS NULL OR doc_type_code=$2)
  AND (cardinality($3::text[])=0 OR areas && $3::text[])
  ORDER BY created_at DESC
  LIMIT $4 OFFSET $5

mappers.go: row→domain and vice versa. Handle jsonb Marshal/Unmarshal for MetadataSchema, []Placeholder, []EditableZone.

postgres_test.go uses testcontainers-go or existing test harness (check apps/api/test/*); reuse whichever documents_v2 repo tests use. Minimum tests:
  CreateTemplate + GetTemplate round-trip.
  GetTemplate cross-tenant returns ErrNotFound.
  GetTemplate with malformed uuid returns ErrNotFound.
  CreateVersion + GetVersion round-trip.
  UpdateVersion persists schema jsonb correctly (re-read + compare).
  ObsoletePreviousPublished updates only the prior published row.
  AppendAudit + ListAudit ordered DESC.

Before writing tests, read internal/modules/documents_v2/repository/postgres_test.go to mirror connection setup. Do not change test harness.
```

**Verify:**
```bash
cd apps/api && rtk go test ./internal/modules/templates_v2/repository/...
```
Expected: PASS.

**Commit:** `feat(templates-v2): P3.1 postgres repository`

### Task 3.2: Opus phase review

**Implementor:** Opus via `nexus:code-reviewer`

**Prompt:**
```
Review Phase 3 (repository). Files: internal/modules/templates_v2/repository/*. Check: parameterized queries only (no string concat), tenant predicates present on every read, uuid error mapping mirrors documents_v2 pattern, ObsoletePreviousPublished handles the "no prior published" case safely. Reply PROCEED | BLOCK.
```

---

## Phase 4 — Backend HTTP delivery

### Task 4.1: Handler scaffold + error mapper

**Files:**
- Create: `internal/modules/templates_v2/delivery/http/handler.go`
- Create: `internal/modules/templates_v2/delivery/http/errors.go`
- Create: `internal/modules/templates_v2/delivery/http/errors_test.go`

**Implementor:** Codex (medium)

**Codex prompt:**
```
Files:

errors.go: func MapErr(err error) (httpStatus int, code string) mirroring internal/modules/documents_v2/delivery/http/errors.go. Mapping:
  domain.ErrNotFound → 404 "not_found"
  domain.ErrKeyConflict → 409 "key_conflict"
  domain.ErrInvalidVisibility → 400 "invalid_visibility"
  domain.ErrInvalidStateTransition → 409 "invalid_state_transition"
  domain.ErrStaleBase → 409 "stale_base"
  domain.ErrContentHashMismatch → 409 "content_hash_mismatch"
  domain.ErrUploadMissing → 409 "upload_missing"
  domain.ErrISOSegregationViolation → 403 "iso_segregation_violation"
  domain.ErrArchived → 409 "archived"
  default → 500 "internal_error"

Response body JSON: {"error":{"code":"X","message":"..."}}.

errors_test.go: table-driven over each mapping.

handler.go: struct Handler holding *application.Service plus authorizer. Constructor func New(svc *application.Service, authz AuthzFunc) *Handler. Method Register(mux *http.ServeMux) that will attach routes (add in subsequent tasks — empty body for now). Expose helper writeJSON(w, status, body) and readJSON(r, v). authorizer type:
  type AuthzFunc func(r *http.Request, tenantID, area string, action string) error
```

**Verify:** `rtk go build ./internal/modules/templates_v2/delivery/http/... && rtk go test ./internal/modules/templates_v2/delivery/http/...`

**Commit:** `feat(templates-v2): P4.1 handler scaffold + error mapper`

### Task 4.2: Create + list + get routes

**Files:**
- Create: `internal/modules/templates_v2/delivery/http/routes_create.go`
- Create: `internal/modules/templates_v2/delivery/http/routes_query.go`
- Create: `internal/modules/templates_v2/delivery/http/routes_create_test.go`

**Implementor:** Codex (high)

**Codex prompt:**
```
In routes_create.go add to Handler:
  POST /api/v2/templates            createTemplate
  POST /api/v2/templates/{id}/versions  createNextVersion

Request for createTemplate: JSON with fields matching application.CreateTemplateCmd (minus TenantID+ActorUserID which come from headers X-Tenant-ID, X-User-ID). Authorize: authz(r, tenantID, area="*", action="template.create").

Response 201: {"data":{"template":{...},"version":{...}}}.

In routes_query.go:
  GET /api/v2/templates             list (query params: area, doc_type, limit, offset)
  GET /api/v2/templates/{id}        get aggregate (returns template + latest version + approval_config)
  GET /api/v2/templates/{id}/versions/{n}  getVersion
  GET /api/v2/templates/{id}/audit  list audit (paginated)

Register all in handler.go Register().

TDD routes_create_test.go:
  POST creates template, returns 201, body has template.id + version.version_number=1.
  POST with visibility="weird" returns 400 invalid_visibility.
  POST same key twice returns 409 key_conflict.

Use httptest.NewServer + in-memory fake repo/presigner/clock/uuid from phase 2.
```

**Verify:** `rtk go test ./internal/modules/templates_v2/delivery/http/...`

**Commit:** `feat(templates-v2): P4.2 create + query routes`

### Task 4.3: Schema + autosave routes

**Files:**
- Create: `internal/modules/templates_v2/delivery/http/routes_schema.go`
- Create: `internal/modules/templates_v2/delivery/http/routes_autosave.go`
- Create: `internal/modules/templates_v2/delivery/http/routes_autosave_test.go`

**Implementor:** Codex (high)

**Codex prompt:**
```
routes_schema.go:
  PUT /api/v2/templates/{id}/versions/{n}/schema → UpdateSchemas

routes_autosave.go:
  POST /api/v2/templates/{id}/versions/{n}/autosave/presign → PresignAutosave
  POST /api/v2/templates/{id}/versions/{n}/autosave/commit  → CommitAutosave

All request/response shapes mirror documents_v2 equivalents exactly. Authorization: autosave + schema changes require action="template.edit" on template.areas (any-match).

TDD routes_autosave_test.go: presign returns URL + key; commit hash-match persists; commit hash-mismatch returns 409 content_hash_mismatch and calls presigner.Delete once.
```

**Verify:** `rtk go test ./internal/modules/templates_v2/delivery/http/...`

**Commit:** `feat(templates-v2): P4.3 schema + autosave routes`

### Task 4.4: Lifecycle routes

**Files:**
- Create: `internal/modules/templates_v2/delivery/http/routes_lifecycle.go`
- Create: `internal/modules/templates_v2/delivery/http/routes_lifecycle_test.go`

**Implementor:** Codex (high)

**Codex prompt:**
```
routes_lifecycle.go adds:
  POST /api/v2/templates/{id}/versions/{n}/submit
  POST /api/v2/templates/{id}/versions/{n}/review    body: {accept, reason?}
  POST /api/v2/templates/{id}/versions/{n}/approve   body: {accept, reason?}
  POST /api/v2/templates/{id}/archive
  PUT  /api/v2/templates/{id}/approval-config        body: {reviewer_role?, approver_role}

Authorization actions:
  submit: template.edit (author)
  review: template.review
  approve: template.approve
  archive: template.archive (admin)
  approval-config: template.admin

TDD routes_lifecycle_test.go: one happy path per endpoint + one failure path each (wrong state, segregation violation).
```

**Verify:** `rtk go test ./internal/modules/templates_v2/delivery/http/...`

**Commit:** `feat(templates-v2): P4.4 lifecycle routes`

### Task 4.5: Wire handler into main

**Files:**
- Modify: `apps/api/cmd/metaldocs-api/main.go` — add `templates_v2` handler registration.

**Implementor:** Codex (medium)

**Codex prompt:**
```
Modify apps/api/cmd/metaldocs-api/main.go. Find the documents_v2 handler registration (search for "documents_v2"). Directly after it, add equivalent for templates_v2:

Imports needed:
  tmplv2app "metaldocs/internal/modules/templates_v2/application"
  tmplv2repo "metaldocs/internal/modules/templates_v2/repository"
  tmplv2http "metaldocs/internal/modules/templates_v2/delivery/http"

Wiring block:
  tmplRepo := tmplv2repo.New(pool)
  tmplSvc := tmplv2app.New(tmplRepo, minioPresigner, realClock{}, uuidGen{})
  tmplHandler := tmplv2http.New(tmplSvc, authz)
  tmplHandler.Register(mux)

Reuse existing minioPresigner + realClock + uuidGen + authz identifiers already in main.go (inspect first; do not redeclare).

No new test file — covered by existing smoke test (apps/api/cmd/metaldocs-api/main_smoke_test.go if present). If none, skip tests.
```

**Verify:**
```bash
cd apps/api && rtk go build ./... && rtk go vet ./...
```
Expected: no errors.

**Commit:** `feat(templates-v2): P4.5 wire handler into main`

### Task 4.6: Opus phase review

**Implementor:** Opus via `nexus:code-reviewer`

**Prompt:**
```
Review Phase 4 (HTTP delivery). Files: internal/modules/templates_v2/delivery/http/*, apps/api/cmd/metaldocs-api/main.go. Check: every spec endpoint exists; auth actions sensible; error codes match spec; request/response shapes mirror docs_v2 where relevant. Reply PROCEED | BLOCK.
```

---

## Phase 5 — Frontend list + create modal

### Task 5.1: API client

**Files:**
- Create: `frontend/apps/web/src/features/templates/v2/api/templatesV2.ts`
- Create: `frontend/apps/web/src/features/templates/v2/types.ts`
- Create: `frontend/apps/web/src/features/templates/v2/api/__tests__/templatesV2.test.ts`

**Implementor:** Codex (medium)

**Codex prompt:**
```
Mirror structure of frontend/apps/web/src/features/documents/v2/api/documentsV2.ts (read it first).

types.ts: TS types matching every Go domain type (Template, TemplateVersion, MetadataSchema, Placeholder, EditableZone, ApprovalConfig, AuditEvent, VersionStatus union).

templatesV2.ts: fetcher functions
  createTemplate(input): Promise<{template, version}>
  listTemplates(filter): Promise<Template[]>
  getTemplate(id): Promise<{template, latest_version, approval_config}>
  getVersion(id, n): Promise<TemplateVersion>
  updateSchemas(id, n, body): Promise<TemplateVersion>
  presignAutosave(id, n): Promise<{upload_url, storage_key, expires_at}>
  commitAutosave(id, n, body): Promise<TemplateVersion>
  submit(id, n): Promise<TemplateVersion>
  review(id, n, body): Promise<TemplateVersion>
  approve(id, n, body): Promise<TemplateVersion>
  archive(id): Promise<Template>
  upsertApprovalConfig(id, body): Promise<ApprovalConfig>
  listAudit(id, page): Promise<AuditEvent[]>
  createNextVersion(id): Promise<TemplateVersion>

Use the shared apiFetch helper (find it by searching imports in documentsV2.ts). No retries, no custom error handling beyond apiFetch defaults.

TDD test uses MSW (match how documentsV2 tests do) — one test per fetcher at minimum: builds correct request (method + url + body) and returns parsed response.
```

**Verify:**
```bash
cd frontend/apps/web && rtk pnpm vitest run src/features/templates/v2/api
```

**Commit:** `feat(templates-v2): P5.1 frontend API client`

### Task 5.2: Templates list page

**Files:**
- Create: `frontend/apps/web/src/features/templates/v2/TemplatesListPage.tsx`
- Create: `frontend/apps/web/src/features/templates/v2/__tests__/TemplatesListPage.test.tsx`

**Implementor:** Codex (medium)

**Codex prompt:**
```
Component TemplatesListPage. Uses TanStack Query useQuery(['templates-v2','list',filter], () => listTemplates(filter)).

Layout:
  - Top bar: [New template] button (opens TemplateCreateDialog — import placeholder; will be created next task).
  - Filter row: area dropdown, doc type dropdown, status dropdown.
  - Table: columns Name, Key, Doc Type, Areas, Latest Version, Status, Updated.
  - Row click → navigate to `#/templates-v2/{id}/edit` for draft, `#/templates-v2/{id}` for published.

Use existing TableUI primitives — check documents_v2 list page for the library in use (probably ShadCN or similar). Mirror styling.

Empty state: "No templates yet. Click New template to create one."

TDD test renders with mocked query returning 2 templates; asserts both rows visible; clicking New template calls onCreate prop.
```

**Verify:** `rtk pnpm vitest run src/features/templates/v2/__tests__/TemplatesListPage`

**Commit:** `feat(templates-v2): P5.2 templates list page`

### Task 5.3: Create dialog

**Files:**
- Create: `frontend/apps/web/src/features/templates/v2/TemplateCreateDialog.tsx`
- Create: `frontend/apps/web/src/features/templates/v2/__tests__/TemplateCreateDialog.test.tsx`

**Implementor:** Codex (medium)

**Codex prompt:**
```
Component TemplateCreateDialog (controlled open/onOpenChange props). Fields:
  key (text, required, slug validation /^[a-z0-9-]+$/)
  name (text, required)
  description (textarea)
  doc_type_code (select — options from TODO provider; for now hardcoded list ["PO","POLITICA","FORMULARIO","MANUAL"] — replace with API later; wire via prop `docTypes`)
  areas (multi-select — prop `availableAreas: Area[]`)
  visibility (radio: public/internal/specific)
  specific_areas (multi-select, visible only when visibility=specific)
  approver_role (text, required)
  reviewer_role (text, optional)

On submit: call createTemplate; on success navigate to `#/templates-v2/{newId}/edit`. On error show error banner.

TDD test fills all fields, submits, asserts createTemplate called with correct payload.
```

**Verify:** `rtk pnpm vitest run src/features/templates/v2/__tests__/TemplateCreateDialog`

**Commit:** `feat(templates-v2): P5.3 template create dialog`

### Task 5.4: Route wiring

**Files:**
- Modify: `frontend/apps/web/src/routing/workspaceRoutes.ts` — add template editor/review path helpers.
- Modify: `frontend/apps/web/src/App.tsx` (or router root — find it) — mount TemplatesListPage at `/templates-v2` and the editor page (placeholder component) at `/templates-v2/:id/edit`.

**Implementor:** Sonnet-OK (pattern-match to documents_v2 route wiring).

**Prompt (Sonnet 4.6):**
```
In workspaceRoutes.ts: templates_v2 view already exists. Add helpers:
  export function templateEditorPath(id: string): string
  export function templateReviewPath(id: string): string
  export function parseTemplateV2Route(pathname: string): { view: "list" } | { view: "edit"; id: string } | { view: "review"; id: string }

Pattern /^\/templates-v2$/ = list; /^\/templates-v2\/([^/]+)\/edit$/ = edit; /^\/templates-v2\/([^/]+)$/ = review.

Mount TemplatesListPage + a stub TemplateAuthoringPage (empty div for now) in the router. Leave TemplateReviewPage as stub.
```

**Verify:** `rtk pnpm tsc --noEmit`
Expected: clean.

**Commit:** `feat(templates-v2): P5.4 route wiring`

### Task 5.5: Opus phase review

**Implementor:** Opus via `nexus:code-reviewer`.

---

## Phase 6 — Eigenpal template mode + authoring page

### Task 6.1: Eigenpal template-mode adapter

**Files:**
- Modify: `frontend/apps/web/src/editor-adapters/eigenpal-template-mode.ts`
- Create: `frontend/apps/web/src/editor-adapters/__tests__/eigenpal-template-mode.test.ts`

**Implementor:** Codex (high)

**Codex prompt:**
```
Extend eigenpal-template-mode.ts (built in Phase 0) with:

export type TemplateModeOptions = {
  placeholders: Placeholder[];
  editableZones: EditableZone[];
  onInsertPlaceholder(id: string): void;
  onWrapZone(id: string): void;
};

export function registerTemplateCommands(editor: DocxEditorInstance, opts: TemplateModeOptions): () => void

Registers slash commands /field and /zone in eigenpal. /field opens a picker populated from opts.placeholders; selecting inserts placeholder inline run via helpers from Phase 0. /zone wraps current selection via wrapZone helper. Returns teardown fn.

TDD test: construct a stub editor (eigenpal exports a test utility or we stub the slash command API observed in Phase 0 spike); invoke registered command; assert onInsertPlaceholder called with chosen id + inline run inserted.
```

**Verify:** `rtk pnpm vitest run src/editor-adapters/__tests__/eigenpal-template-mode`

**Commit:** `feat(templates-v2): P6.1 eigenpal template commands`

### Task 6.2: Autosave hook

**Files:**
- Create: `frontend/apps/web/src/features/templates/v2/hooks/useTemplateAutosave.ts`
- Create: `frontend/apps/web/src/features/templates/v2/hooks/__tests__/useTemplateAutosave.test.tsx`

**Implementor:** Codex (high)

**Codex prompt:**
```
Port documents/v2 useAutosave hook (read frontend/apps/web/src/features/documents/v2/hooks/useDocumentAutosave.ts or equivalent) for templates.

Contract:
  useTemplateAutosave({ templateId, versionNumber, editor })
    debounces body-changed events; on debounce expiry calls presignAutosave → PUT to MinIO URL → computes body hash client-side → commitAutosave(expectedContentHash).
    Returns {status: 'idle'|'saving'|'saved'|'error', lastSavedAt, error}.

TDD: fake timers + MSW. Test cases: typing triggers save after debounce; commit success moves state to saved; hash mismatch surfaces error; non-draft state from server returns invalid_state_transition and hook shows error.
```

**Verify:** `rtk pnpm vitest run src/features/templates/v2/hooks`

**Commit:** `feat(templates-v2): P6.2 useTemplateAutosave hook`

### Task 6.3: Placeholder sidebar

**Files:**
- Create: `frontend/apps/web/src/features/templates/v2/components/PlaceholderSidebar.tsx`
- Create: `frontend/apps/web/src/features/templates/v2/hooks/useTemplatePlaceholders.ts`
- Create: tests for each.

**Implementor:** Codex (medium)

**Codex prompt:**
```
useTemplatePlaceholders({ templateId, versionNumber, initial }): manages placeholder array with optimistic updates; calls updateSchemas(id,n,{...,placeholder_schema}) on change.

PlaceholderSidebar component: list + "[+ Add]" button that opens an inline form (label, type, required). On add, update via hook. On delete, confirm if placeholder id referenced in body (parent supplies referenced ids via prop).

TDD: add → PUT /schema called; delete referenced shows confirm modal.
```

**Verify:** `rtk pnpm vitest run src/features/templates/v2/components/__tests__/PlaceholderSidebar`

**Commit:** `feat(templates-v2): P6.3 placeholder sidebar`

### Task 6.4: Metadata schema form + editable zones panel

**Files:**
- Create: `frontend/apps/web/src/features/templates/v2/components/MetadataSchemaForm.tsx`
- Create: `frontend/apps/web/src/features/templates/v2/components/EditableZonesPanel.tsx`
- Tests.

**Implementor:** Codex (medium)

**Codex prompt:**
```
MetadataSchemaForm: form with inputs for doc_code_pattern, retention_days, distribution_default (chips), required_metadata (chips with a small library of well-known keys: effective_date, approver, distribution, revision). onChange emits MetadataSchema.

EditableZonesPanel: list of zones with label + required toggle; buttons to add/remove (removal refuses if zone already referenced in body — parent supplies referenced ids via prop).

Both persist via useTemplatePlaceholders style (split: extract shared hook useTemplateSchemas if needed — one hook per field group is fine).

TDD each: typing → onChange called with new value.
```

**Verify:** `rtk pnpm vitest run src/features/templates/v2/components/__tests__/Metadata src/features/templates/v2/components/__tests__/EditableZonesPanel`

**Commit:** `feat(templates-v2): P6.4 metadata + zones forms`

### Task 6.5: Authoring page composition

**Files:**
- Create: `frontend/apps/web/src/features/templates/v2/TemplateAuthoringPage.tsx` (replace stub from 5.4)
- Create: tests.

**Implementor:** Codex (high)

**Codex prompt:**
```
Compose the authoring page per spec Section 3 layout:
  - Top bar: template name + version + status badge + [Submit for review] button.
  - Metadata schema panel (collapsible, from 6.4).
  - Two-column body: PlaceholderSidebar (left) + DocxEditor mode="template" (right).
  - Below body: EditableZonesPanel.

Wire:
  - Load template + latest version via useQuery.
  - Mount DocxEditor with mode="template"; register commands from 6.1 passing live placeholders + zones.
  - Autosave via 6.2.
  - Submit button disabled unless status=draft; on click calls submit(id,n) and navigates to list.

Read-only if status != draft (editor readOnly=true, schema panels disabled).

TDD: render with fake data; type → autosave triggered; submit → API called + navigation.
```

**Verify:** `rtk pnpm vitest run src/features/templates/v2/__tests__/TemplateAuthoringPage`

**Commit:** `feat(templates-v2): P6.5 authoring page`

### Task 6.6: Opus phase review

**Prompt:** Review Phase 6 with emphasis on eigenpal integration correctness and the read-only state after submit.

---

## Phase 7 — Review / approve surface

### Task 7.1: Review page

**Files:**
- Create: `frontend/apps/web/src/features/templates/v2/TemplateReviewPage.tsx`
- Tests.

**Implementor:** Codex (medium)

**Codex prompt:**
```
Page for reviewer/approver. Loads version (status in_review or approved). Renders read-only DocxEditor (mode="document", readOnly=true), metadata panel, placeholder list (all read-only).

Top bar shows version status + actions based on user role:
  - Reviewer (on in_review, approval_config.reviewer_role): [Approve review] + [Reject] buttons.
  - Approver (on in_review when no reviewer configured, OR on approved): [Publish] + [Reject] buttons.

Reject opens a reason textarea modal; submit posts to /review or /approve with accept=false + reason.

TDD: renders correct buttons per status+role combination; submit calls correct endpoint with correct body.
```

**Verify:** `rtk pnpm vitest run src/features/templates/v2/__tests__/TemplateReviewPage`

**Commit:** `feat(templates-v2): P7.1 review page`

### Task 7.2: Opus phase review.

---

## Phase 8 — Document fill-in flow

### Task 8.1: Extend documents_v2 create flow to snapshot template_v2

**Files:**
- Modify: `internal/modules/documents_v2/application/create.go` — new flag / branch for template source.
- Modify: `internal/modules/documents_v2/repository/postgres.go` — write FK column.
- Add tests.

**Implementor:** Codex (high)

**Codex prompt:**
```
Extend documents_v2 CreateDocument command:

CreateDocumentCmd already exists. Add field: TemplatesV2VersionID *string (optional).

If set, service must:
1. Load templates_v2 version (inject templates_v2 application service or a narrow read port — add a new interface TemplatesV2ReadPort { GetPublishedVersion(ctx, id string) (*tmplv2domain.TemplateVersion, error) }). Implementation in apps/api wires templates_v2 repo via this port.
2. Version must have status=published. Else return a new error domain.ErrTemplateNotPublished.
3. Copy version.DocxStorageKey as starting object (copy object in MinIO to documents/{docID}/versions/1.docx via s.presign.Copy — add Copy method to docs_v2 presigner interface if missing).
4. Snapshot version.MetadataSchema + PlaceholderSchema + EditableZones into the new document's body_payload JSON (documents_v2 already has a payload jsonb — add these keys: templates_v2_metadata_schema, templates_v2_placeholder_schema, templates_v2_editable_zones).
5. Store FK documents_v2_documents.templates_v2_template_version_id.

TDD in documents_v2/application/create_test.go — use a fake TemplatesV2ReadPort:
  Happy: published version → doc created with FK + payload keys.
  Non-published version → ErrTemplateNotPublished.
```

**Verify:** `rtk go test ./internal/modules/documents_v2/...`

**Commit:** `feat(templates-v2): P8.1 documents_v2 template snapshot on create`

### Task 8.2: Wire TemplatesV2ReadPort into main.go

**Files:**
- Modify: `apps/api/cmd/metaldocs-api/main.go`
- Create: small adapter in `internal/modules/templates_v2/application/read_port_adapter.go` exposing GetPublishedVersion using the existing service.

**Implementor:** Sonnet-OK (plumbing).

**Prompt:** Implement adapter struct that wraps `*templates_v2_application.Service` and exposes the method `GetPublishedVersion` used by documents_v2. Pass adapter into `documents_v2_application.New(...)` wiring.

**Verify:** `rtk go build ./...`

**Commit:** `feat(templates-v2): P8.2 wire templates read-port into documents_v2`

### Task 8.3: Frontend fill-mode layout

**Files:**
- Create: `frontend/apps/web/src/features/documents/v2/FillModeLayout.tsx`
- Modify: `frontend/apps/web/src/features/documents/v2/DocumentEditorPage.tsx` — render FillModeLayout when document has templates_v2 snapshot keys.
- Tests.

**Implementor:** Codex (high)

**Codex prompt:**
```
FillModeLayout props: { document, versionMetadataSchema, versionPlaceholderSchema, versionEditableZones, onFieldChange(id,value), onZoneChange(id,blocks), readOnly }.

Layout:
  - Metadata form (reuse/fork MetadataSchemaForm but all inputs required per required_metadata; fields defined implicitly: doc_code, effective_date, approver, distribution, etc from required_metadata list).
  - Placeholder sidebar (read-only list of placeholders; click → focus field in body or inline edit).
  - DocxEditor mode="document".

DocumentEditorPage: detect templates_v2 snapshot by presence of payload key templates_v2_metadata_schema. If present → render FillModeLayout. Else → existing editor (unchanged).

TDD: render DocumentEditorPage with mock doc containing snapshot keys → FillModeLayout shown.
```

**Verify:** `rtk pnpm vitest run src/features/documents/v2/__tests__`

**Commit:** `feat(templates-v2): P8.3 fill-mode layout`

### Task 8.4: Document create flow picks published templates

**Files:**
- Modify: `frontend/apps/web/src/features/documents/v2/DocumentCreatePage.tsx` — add "From template" dropdown that lists published templates visible to user.
- Tests.

**Implementor:** Codex (medium)

**Codex prompt:**
```
Extend DocumentCreatePage:

Add field "Template" — a searchable dropdown populated by listTemplates({status:'published', visibleTo: user areas}). Optional. On submit, pass templates_v2_version_id = selected template.published_version_id.

TDD: pick template → create call receives templates_v2_version_id.
```

**Verify:** `rtk pnpm vitest run src/features/documents/v2/__tests__/DocumentCreatePage`

**Commit:** `feat(templates-v2): P8.4 document create picks template`

### Task 8.5: Opus phase review

---

## Phase 9 — Legacy templates migration banner + legacy read-only gate

### Task 9.1: Gate legacy templates handler to read-only

**Files:**
- Modify: `internal/modules/templates/delivery/http/handler.go` — block all mutating methods (POST/PUT/DELETE) with 410 Gone { code: "legacy_templates_readonly", message: "use templates_v2" }.

**Implementor:** Codex (medium)

**Codex prompt:**
```
Wrap legacy templates handler: any request method != GET returns 410 with JSON error envelope. Log once per process (not per request).

TDD: POST to any legacy templates route returns 410.
```

**Verify:** `rtk go test ./internal/modules/templates/...`

**Commit:** `feat(templates-v2): P9.1 legacy templates read-only`

### Task 9.2: Frontend redirect + banner

**Files:**
- Modify: `frontend/apps/web/src/features/templates/TemplateEditorView.tsx` — at mount, if route is legacy `/templates/...`, show banner "Template editing has moved. Click here to use the new editor." linking to `/templates-v2`.

**Implementor:** Sonnet-OK.

**Verify:** `rtk pnpm tsc --noEmit`

**Commit:** `feat(templates-v2): P9.2 legacy banner`

### Task 9.3: Opus phase review

---

## Phase 10 — E2E Playwright golden path

### Task 10.1: Playwright test — full lifecycle

**Files:**
- Create: `frontend/apps/web/e2e/templates-v2-golden-path.spec.ts`

**Implementor:** Codex (high)

**Codex prompt:**
```
Playwright test mirroring the golden path in spec Testing section. Use existing Playwright harness (check frontend/apps/web/e2e/*.spec.ts for fixtures). Steps:

1. Log in as template_author of area "buying".
2. Navigate /templates-v2; click New template.
3. Fill: key=e2e-po, name=E2E PO, doc_type=PO, areas=[buying], visibility=internal, approver_role=quality_manager.
4. On authoring page: add placeholder customer_name (text, required) + effective_date (date, required).
5. Type body content in DocxEditor: "Cliente: {{customer_name}}, vigência {{effective_date}}."
6. Mark a paragraph as editable zone "observations".
7. Click Submit for review.
8. Log in as quality_manager.
9. Navigate /templates-v2/{id}; click Publish.
10. Log in as author of buying.
11. Create new document: pick template e2e-po.
12. Fill placeholders + metadata; submit.
13. Log in as quality_manager; approve document.
14. Export DOCX; verify zip contains expected text (use e2e utility).

Use test.step() for each major phase. Add data-testid where needed (coordinate with Codex review if components missing test ids).
```

**Verify:** `rtk pnpm playwright test templates-v2-golden-path`

**Commit:** `test(templates-v2): P10.1 E2E golden path`

### Task 10.2: Final Opus review + spec compliance audit

**Implementor:** Opus via `nexus:code-reviewer`

**Prompt:**
```
Final audit of templates-v2 implementation against spec docs/superpowers/specs/2026-04-20-templates-v2-wysiwyg-design.md. Walk every section of the spec; for each, cite the commits or files that satisfy it. Flag any gaps. Flag any out-of-scope code that crept in. Reply with a compliance table.
```

---

## Self-review

**Spec coverage:**
- In-scope items mapped: WYSIWYG authoring (P6), metadata (P6.4), placeholders (P6.3), zones (P6.4), strict lock (P8.3), approval (P2.5 + P7), snapshot versioning (P8.1), area RBAC (P4 authz hooks), audit (P1.5 + P2 appends), segregation (P1.4 + enforced in P2.5 + E2E in P10), fill-in (P8), state machines (P1.3 + P2.5), HTTP API (P4.1-P4.5), migration (P1.1 + P1.2), data model (P1.1), eigenpal integration (P0 spike + P6.1), legacy read-only (P9).
- Out-of-scope items NOT included (correctly).
- Risks addressed: eigenpal extensibility = Phase 0 spike (blocks rest); doc code concurrency = handled in CreateDocument application layer with seq table + FOR UPDATE (call out explicitly — TBD: not in current plan). **Gap:** doc code sequence counter. Add as Task 8.1a.

**Placeholder scan:** no TBD/TODO/"similar to".

**Type consistency:** Go types used consistently (Template, TemplateVersion, VersionStatus, MetadataSchema, Placeholder, EditableZone, ApprovalConfig, AuditEvent). TS types mirror 1:1. ListFilter consistent. No drift.

**Fix inline — Task 8.1a added:**

### Task 8.1a: Doc code sequence allocator

**Files:**
- Create: `migrations/0120_doc_code_sequence.sql` — `CREATE TABLE doc_code_sequence(tenant_id text, doc_type_code text, area_code text, next_val int NOT NULL, PRIMARY KEY(tenant_id, doc_type_code, area_code));`
- Modify: `internal/modules/documents_v2/application/create.go` — when MetadataSchema.DocCodePattern present and document metadata requires doc_code, allocate via `SELECT ... FOR UPDATE; UPDATE ...` in a transaction; format via pattern.

**Implementor:** Codex (high). Inline in P8.1.

---

## Codex revision round 1 (2026-04-20) — added tasks

These tasks + edits supersede or extend the phases above. Apply in-phase per task number.

### Task 2.2a: Validation edge cases in CreateTemplate + UpdateSchemas

**Files:** modify `application/create.go`, `application/schema.go`, extend `*_test.go`.

**Implementor:** Codex (medium)

**Codex prompt:**
```
Extend validation:

In CreateTemplate:
  - If cmd.Visibility == "specific" and len(cmd.SpecificAreas) == 0 → return domain.ErrInvalidVisibility with wrapped error text "specific_visibility_requires_areas".
  - If cmd.Visibility != "specific" and len(cmd.SpecificAreas) > 0 → silently ignore (clear the slice before persist).
  - If len(cmd.Areas) == 0 AND cmd.Visibility == "specific" → error "specific_visibility_requires_areas".

In UpdateSchemas, for every placeholder p in cmd.PlaceholderSchema:
  - If p.Type == "select" AND len(p.Options) == 0 → error "select_placeholder_requires_options".
  - If p.Type != "select" AND len(p.Options) > 0 → error "options_allowed_only_for_select".

Tests cover every new branch.
```

**Commit:** `feat(templates-v2): P2.2a validation edge cases`

### Task 2.6a: ListTemplates visibility-tier enforcement

**Files:** modify `application/queries.go`, `application/ports.go` (extend ListFilter), tests.

**Implementor:** Codex (medium)

**Codex prompt:**
```
Extend ListFilter with:
  ActorAreas []string  // areas the caller belongs to
  IsExternalViewer bool // true for unauthenticated/public-only callers

Repository-side filter:
  visibility='public' always included.
  visibility='internal' included unless IsExternalViewer.
  visibility='specific' included only if specific_areas && ActorAreas is non-empty.

Queries test covers each tier. Repository test (Phase 3) adds parallel coverage.

For GetTemplate aggregate, same rules: if caller lacks visibility, return ErrNotFound (never reveal existence).
```

**Commit:** `feat(templates-v2): P2.6a visibility filtering`

### Task 2.8: UpsertApprovalConfig with publish-state rules

**Files:** create `application/approval_config.go` + tests.

**Implementor:** Codex (high)

**Codex prompt:**
```
Command:

type UpsertApprovalConfigCmd {
  TenantID, ActorUserID, TemplateID string
  ActorRoles []string             // must include "admin" after first publish
  ReviewerRole *string
  ApproverRole string
}

Rules:
  1. Load template. If archived → ErrArchived.
  2. Determine "has ever published" = template.PublishedVersionID != nil OR exists any version with published_at != nil.
  3. If has-ever-published:
       - Actor must include "admin" role → else ErrForbidden.
  4. Else (no published history):
       - Actor must be template author (created_by) OR admin → else ErrForbidden.
  5. Validate ApproverRole non-empty; else ErrInvalidApprovalConfig.
  6. repo.UpsertApprovalConfig; AppendAudit(action="approval_config_updated", details={"reviewer_role":rr,"approver_role":ar}).

Add errors: ErrForbidden, ErrInvalidApprovalConfig (domain).

TDD covers all branches.
```

**Commit:** `feat(templates-v2): P2.8 UpsertApprovalConfig`

### Task 3.1a: Repo visibility clause

Extend `ListTemplates` SQL with visibility filter clause matching 2.6a. Additional repo test per tier.

**Implementor:** Codex (medium). Extend existing postgres_test.go.

**Commit:** `feat(templates-v2): P3.1a repo visibility filter`

### Task 8.0: Snapshot approval_config on document creation

**Files:** modify `internal/modules/documents_v2/application/create.go` (extends P8.1), `documents_v2` migration.

**Implementor:** Codex (high)

**Codex prompt:**
```
Extend CreateDocument (P8.1):

After loading template version + pulling MetadataSchema/PlaceholderSchema/EditableZones, also load the template's approval_config via TemplatesV2ReadPort.GetApprovalConfig(ctx, templateID).

Snapshot into documents_v2_documents new columns (add via new migration `migrations/0121_docs_v2_template_approval_snapshot.sql`):
  ALTER TABLE documents_v2_documents
    ADD COLUMN template_approval_reviewer_role text NULL,
    ADD COLUMN template_approval_approver_role text NOT NULL DEFAULT '';

Extend TemplatesV2ReadPort in documents_v2 with:
  GetApprovalConfig(ctx, templateID string) (*tmplv2domain.ApprovalConfig, error)

Documents_v2 review/approve flow (existing) must consult these snapshot columns:
  - Reviewer stage gated by snapshot reviewer role (if non-null).
  - Approver stage gated by snapshot approver role (always required).
  - Actor role checks + segregation rules identical to templates_v2.

Add unit tests for:
  Document create snapshots approval_config correctly.
  Reviewer without snapshot role → ErrForbiddenRole.
  Approver without snapshot role → ErrForbiddenRole.
  Segregation still enforced for docs.
```

**Commit:** `feat(templates-v2): P8.0 document approval snapshot`

### Task 9.2 (REVISION): Legacy → v2 redirect

Replace banner with actual route redirect. When user hits `/templates` or `/templates/*` route, router redirects to `/templates-v2`. Preserve query string if any. Remove banner component.

**Implementor:** Sonnet-OK.

**Prompt:**
```
In frontend routing root (locate router via `grep -rn HashRouter frontend/apps/web/src`), add a redirect: any path matching /^\/templates(\/|$)/ → /templates-v2. Delete TemplateEditorView banner changes from P9.2 original task.

Add a Vitest that simulates navigating to /templates/foo and asserts the router ends on /templates-v2.
```

**Commit:** `feat(templates-v2): P9.2 revised — redirect /templates → /templates-v2`

### Task 6.5a + 10.1a: Strict-lock negative tests

**Files:** extend `TemplateAuthoringPage.test.tsx`, `templates-v2-golden-path.spec.ts`.

**Implementor:** Codex (medium)

**Codex prompt:**
```
TemplateAuthoringPage.test.tsx:
  Add test: when version.status='published', DocxEditor renders with readOnly=true and user keystroke events are not captured (assert editor prop readOnly=true + simulate paste event + check no onChange fire).

templates-v2-golden-path.spec.ts:
  After publishing the template, create a document. In fill-in page:
   - Attempt to type inside a paragraph outside placeholders/zones → assert content unchanged.
   - Attempt to paste → assert no change.
   - Type inside an editable zone → content updates.
   - Fill placeholders via sidebar → chips update.
```

**Commit:** `test(templates-v2): P10.1a strict-lock negative paths`

### Task 4.4a: PUT /api/v2/templates/{id}/approval-config wiring

Wire 2.8 command into HTTP route. Request body `{reviewer_role?, approver_role}`. Response = updated config. Auth header X-User-Roles supplies ActorRoles.

**Implementor:** Codex (medium)

**Commit:** `feat(templates-v2): P4.4a approval-config route`

### Task 8.0a: Extend TemplatesV2ReadPort adapter for approval-config

**Files:** modify `internal/modules/templates_v2/application/read_port_adapter.go`, `apps/api/cmd/metaldocs-api/main.go`, tests.

**Implementor:** Sonnet-OK (plumbing).

**Prompt:**
```
The TemplatesV2ReadPort interface consumed by documents_v2 now has two methods:
  GetPublishedVersion(ctx, id) (*tmplv2domain.TemplateVersion, error)
  GetApprovalConfig(ctx, templateID string) (*tmplv2domain.ApprovalConfig, error)

Extend the adapter struct in templates_v2/application/read_port_adapter.go to expose GetApprovalConfig, delegating to the underlying Service.GetApprovalConfig (add that query to templates_v2/application/queries.go if missing — simple repo pass-through + tenant check).

Verify main.go wiring: no change needed (adapter is same instance), but run go build to confirm.
```

**Commit:** `feat(templates-v2): P8.0a adapter exposes approval-config read`

### Task 8.0b: Enforce snapshotted approval on document review/approve

**Files:** modify `internal/modules/documents_v2/application/lifecycle.go` (or wherever Review+Approve live — verify via grep), `internal/modules/documents_v2/domain/` (add `ErrForbiddenRole` if not already present or import from shared errors package), tests + delivery tests.

**Implementor:** Codex (high)

**Codex prompt:**
```
Extend documents_v2 Review and Approve commands:

Both commands must accept ActorRoles []string on their Cmd struct.

Review:
  - Require document.TemplateApprovalReviewerRole != nil else ErrInvalidStateTransition (no reviewer stage).
  - ActorRoles must include *document.TemplateApprovalReviewerRole else ErrForbiddenRole.
  - Existing segregation checks still apply (actor != author).

Approve:
  - If document.TemplateApprovalReviewerRole != nil: expect document.Status=approved (post-review).
  - Else: expect document.Status=in_review.
  - ActorRoles must include document.TemplateApprovalApproverRole else ErrForbiddenRole.
  - Segregation: actor != author AND (reviewer_id nil OR actor != reviewer).

Add domain.ErrForbiddenRole in documents_v2 domain (or import shared).

HTTP delivery: lifecycle routes (review/approve) must read X-User-Roles header into ActorRoles.

Tests:
  Happy: actor with correct role → OK.
  Wrong role: → 403 forbidden_role (map via errors.go).
  Segregation violation: unchanged coverage.
  Reviewer stage absent (nil snapshot): Review returns invalid_state_transition; Approve goes straight.
```

**Commit:** `feat(templates-v2): P8.0b document review/approve snapshot enforcement`

### Task 4.6a: Visibility-aware route tests

Extend P4.2 query route tests to cover each visibility tier (public/internal/specific) + external viewer scenario → returns only permitted rows / 404 for hidden.

**Implementor:** Codex (medium)

**Commit:** `test(templates-v2): P4.6a visibility route coverage`

---

## Execution handoff

Plan saved to [docs/superpowers/plans/2026-04-20-templates-v2-wysiwyg.md](2026-04-20-templates-v2-wysiwyg.md). Two execution options:

**1. Subagent-Driven (recommended)** — Controller dispatches Codex (`gpt-5.3-codex`) per task, Opus reviews between tasks and at phase ends. REQUIRED SUB-SKILL: `superpowers:subagent-driven-development`.

**2. Inline Execution** — Controller invokes `mcp__codex__codex` synchronously per task from this session. REQUIRED SUB-SKILL: `superpowers:executing-plans`.

Which approach?
