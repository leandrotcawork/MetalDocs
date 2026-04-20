# Templates v2 — WYSIWYG authoring (eigenpal) design

Date: 2026-04-20
Status: Draft — design approved, pending user review before implementation plan
Owner: Product (founder) + engineering

## Motivation

MetalDocs is a company document controller. Its founder-stated goal is to let a company centralize its controlled documents — procedures, policies, forms, manuals — with standardized look-and-feel, area-scoped access, approval flow, and ISO 9001-grade audit.

The current template flow (W2 plan, 2026-04-18) requires administrators to upload a pre-authored `.docx` file and attach a JSON schema. That path explicitly deferred in-app template authoring (2026-04-04 design). During UAT on 2026-04-19 the founder flagged this as a regression from the product vision: **templates must be authored in-app, using the `@eigenpal/docx-js-editor` WYSIWYG editor, so that document controllers never leave MetalDocs to produce a template.** The editor library was chosen for exactly this capability.

This spec defines a new `templates_v2` module that builds WYSIWYG template authoring, fill-in document creation, versioning, approval flow, and ISO-grade audit on top of the existing eigenpal editor and area-based RBAC.

## Scope

### In scope (v1)

- In-app WYSIWYG template authoring via eigenpal `DocxEditor` in `template` mode.
- Structured metadata header (doc code, revision, dates, approver, retention, distribution) per ISO 9001.
- Placeholders (`{{field}}`) inserted via slash command or sidebar — typed (text/date/number/select/user), required/optional.
- Opt-in editable zones — template author marks regions where fillers may write free text.
- Strict-by-default body lock for fillers (edit placeholders + editable zones only).
- Two-stage approval flow: author → optional reviewer → required approver. Roles configured per template.
- Snapshot-on-create versioning: documents freeze template body + schema at creation; template version bumps do not retro-affect existing documents.
- Area-scoped `template_author` role; global doc types; per-template area binding and visibility tier (public / internal / specific).
- Append-only audit log.
- ISO segregation-of-duties enforcement (author ≠ reviewer ≠ approver).
- Document fill-in flow using eigenpal `DocxEditor` in `document` mode with placeholder chips + editable zones.
- Document state machine (draft → in_review → approved → published → obsolete) mirroring template lifecycle.

### Out of scope (v1)

- Automated migration of existing documents to newer template versions (may return as a later feature).
- Template-to-template inheritance.
- Multi-language templates.
- Cross-tenant template marketplace.
- Conditional placeholder visibility (show field X if field Y equals Z).
- Digital signatures on approval (can be added later).
- Auto-porting of legacy `modules/templates` templates into `templates_v2` (operators manually re-author the ones they want).

## Vision decisions (locked via brainstorming 2026-04-20)

| # | Decision | Choice |
|---|---|---|
| Q1 | Product framing | Company document controller (D). Multi-company infra reused; v1 focus on founder's company. |
| Q2 | ISO 9001 stance | Full compliance v1 (A). Required metadata, approval, audit, obsolete flagging. |
| Q3 | Template model | Hybrid (C). Structured metadata header + eigenpal WYSIWYG body with placeholders. |
| Q4 | Filler lock behavior | Strict default + opt-in editable zones per template (D). |
| Q5 | Template versioning | Snapshot-on-create, new documents use latest published version (D). |
| Q6 | Approval flow | Two-stage: author → optional reviewer → required approver, roles per doc type (C). |
| Q7 taxonomy | Doc type taxonomy | Global doc types + per-template area binding (C). Area + subarea hierarchy already implemented. |
| Q7 author | Who authors templates | Area-scoped `template_author` role (B). |
| Q8 | Implementation approach | New `templates_v2` module — clean rewrite mirroring `documents_v2` (B). |

## Architecture

### Backend module layout

```
internal/modules/templates_v2/
  domain/              Template, TemplateVersion, PlaceholderSchema,
                       MetadataSchema, EditableZone, ApprovalConfig,
                       state machine, error types
  application/         Commands: CreateTemplate, OpenAuthoring,
                       PresignAutosave, CommitAutosave,
                       SubmitForReview, Review, Approve, Reject,
                       Publish, Archive. Queries: ListTemplates,
                       GetTemplate, GetVersion, ListVersions.
  repository/          Postgres (pgx). Mirrors documents_v2 split —
                       aggregate reads + command-scoped writers.
  delivery/http/       Routes under /api/v2/templates.
```

Structure deliberately mirrors `internal/modules/documents_v2` so reviewers familiar with docs_v2 can navigate v2 templates without new mental model.

### Frontend layout

```
frontend/apps/web/src/features/templates/v2/
  api/                 fetchers for templates_v2 endpoints
  TemplatesListPage    list + filter by area/doc-type/status
  TemplateCreateDialog modal: key, name, doc-type, areas, visibility
  TemplateAuthoringPage metadata header form + placeholders sidebar +
                       DocxEditor (mode="template")
  TemplateReviewPage   read-only preview + approve/reject actions
  hooks/               useTemplateAutosave, useTemplatePlaceholders
```

Fill-in document page (under `features/documents/v2`) gains a new `FillModeLayout` that renders placeholder sidebar + metadata form + DocxEditor (`mode="document"`). Editor shares the same component as today; only mode prop differs.

### Storage

- DOCX bodies → MinIO. Key pattern: `templates/{template_id}/versions/{version_number}.docx`. Same bucket as docs_v2 (simpler ops) with distinct prefix.
- Metadata, placeholder schemas, approval state, audit events → Postgres.

### Cross-module integration

`documents_v2_documents` gains a nullable FK column:

```sql
ALTER TABLE documents_v2_documents
  ADD COLUMN templates_v2_template_version_id uuid NULL
    REFERENCES templates_v2_template_version(id);
```

The document continues to store a **frozen snapshot** of the body, metadata schema, and placeholder schema in its own columns; the FK is purely for audit + reporting ("which template birthed this doc?"). Deleting or archiving a template does not cascade into documents.

### Eigenpal integration

The existing `DocxEditor` component gains a `mode` prop:

- `mode="template"` — template authoring. Slash command `/field` opens placeholder picker (choose existing or create new). Slash command `/zone` wraps selection in an editable-zone marker. Toolbar shows "Zone lock" toggle. Body otherwise fully editable.
- `mode="document"` — document fill-in. Placeholder chips render filled values; clicking a chip opens inline edit. Editable zones are the only unlocked prose regions. Everywhere else is hard read-only (no keystrokes, no paste).

Placeholders serialize as a custom inline run in the DOCX (eigenpal custom-element support). Editable zones serialize as a section with a bookmark attribute. Both survive round-trip through eigenpal save.

## Data model

### New tables

```sql
CREATE TABLE templates_v2_template (
  id                    uuid PRIMARY KEY,
  tenant_id             text NOT NULL,
  doc_type_code         text NOT NULL,           -- global doc type code
  key                   text NOT NULL,           -- slug, unique per tenant
  name                  text NOT NULL,
  description           text NOT NULL DEFAULT '',
  areas                 text[] NOT NULL DEFAULT '{}',    -- area codes; empty = all areas
  visibility            text NOT NULL,                    -- 'public' | 'internal' | 'specific'
  specific_areas        text[] NOT NULL DEFAULT '{}',    -- used when visibility = 'specific'
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
  status              text NOT NULL,         -- draft|in_review|approved|published|obsolete
  docx_storage_key    text NOT NULL,
  content_hash        text NOT NULL,
  metadata_schema     jsonb NOT NULL,        -- { doc_code_pattern, retention_days, distribution_default, required_metadata[] }
  placeholder_schema  jsonb NOT NULL,        -- [{ id, label, type, required, default }]
  editable_zones      jsonb NOT NULL,        -- [{ id, label, required }]
  author_id           text NOT NULL,
  reviewer_id         text NULL,
  approver_id         text NULL,
  submitted_at        timestamptz NULL,
  reviewed_at         timestamptz NULL,
  approved_at         timestamptz NULL,
  published_at        timestamptz NULL,
  obsoleted_at        timestamptz NULL,
  created_at          timestamptz NOT NULL DEFAULT now(),
  UNIQUE (template_id, version_number)
);

ALTER TABLE templates_v2_template
  ADD FOREIGN KEY (published_version_id) REFERENCES templates_v2_template_version(id);

CREATE TABLE templates_v2_approval_config (
  template_id     uuid PRIMARY KEY REFERENCES templates_v2_template(id),
  reviewer_role   text NULL,       -- optional stage
  approver_role   text NOT NULL    -- required stage
);

CREATE TABLE templates_v2_audit_log (
  id            bigserial PRIMARY KEY,
  tenant_id     text NOT NULL,
  template_id   uuid NOT NULL,
  version_id    uuid NULL,
  actor_id      text NOT NULL,
  action        text NOT NULL,     -- created|saved|submitted|reviewed|approved|rejected|published|obsoleted|archived
  details       jsonb NOT NULL DEFAULT '{}',
  occurred_at   timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX ON templates_v2_template (tenant_id, doc_type_code);
CREATE INDEX ON templates_v2_template_version (template_id, status);
CREATE INDEX ON templates_v2_audit_log (template_id, occurred_at DESC);
```

### Schema payload shapes

`metadata_schema`:
```json
{
  "doc_code_pattern": "PO-{area}-{seq:3}",
  "retention_days": 1825,
  "distribution_default": ["internal"],
  "required_metadata": ["effective_date", "approver", "distribution"]
}
```

`placeholder_schema`:
```json
[
  { "id": "customer_name", "label": "Customer name", "type": "text", "required": true },
  { "id": "effective_date", "label": "Effective date", "type": "date", "required": true },
  { "id": "revision",       "label": "Revision",       "type": "number", "default": 1 }
]
```

`editable_zones`:
```json
[
  { "id": "observations", "label": "Observations", "required": false }
]
```

## Lifecycles

### Template version state machine

```
draft  ──submit──▶  in_review  ──reviewer_approve──▶  approved  ──approver_publish──▶  published
  ▲                     │                                │
  └─────reject──────────┘                                │
                                                         │
                                 (new version published) ▼
                                                      obsolete
```

- Reviewer stage is skipped when `approval_config.reviewer_role IS NULL`; `in_review` transitions directly on approver approval.
- Reject from any stage returns to `draft` and records the rejection reason in `audit_log.details`.
- Publishing version *n+1* auto-obsoletes the currently-published version (only one `published` per template at any time).

### Document state machine

Identical shape. Publishing a new document version auto-obsoletes the prior published version of the same document.

## Approval flow details

- `approval_config` is created with the template and editable by admins or the template's author (before first publish). Once a template version has published, changing `approver_role` requires admin.
- Reviewer / approver are **resolved at submit time** — the system records the specific `reviewer_id` / `approver_id` of the user who acted, not the role.
- ISO segregation-of-duties checks:
  - At submit: `author_id != reviewer_id` when reviewer is chosen.
  - At approve: `approver_id != author_id` AND `approver_id != reviewer_id` (if a reviewer acted).
  - Violations return `403` with error code `iso_segregation_violation`.
- Document approval uses the same `approval_config` as its source template version (can be overridden per document later; out of scope v1).

## RBAC

| Role | Template actions | Document actions |
|------|------------------|------------------|
| `reader` (area X) | View published templates visible to X | View published documents visible to X |
| `author` (area X) | — | Create draft from published template, edit own drafts |
| `template_author` (area X) | Create + edit template drafts in X (inherits `author`) | — |
| `reviewer` (doc type T) | Review template versions | Review document versions |
| `approver` (doc type T) | Approve + publish template versions | Approve + publish document versions |
| `admin` (global) | Any action, any area | Any action, any area |

Enforcement: new HTTP middleware `authorizeTemplateScope`, mirroring `authorizeDocumentScope` already present in documents_v2. Role source is the existing auth context (`X-User-Roles` / `X-User-Areas` headers today; production auth later).

## Audit

Append-only. One row per state transition and per autosave commit. Events:

`created | saved | submitted | reviewed | approved | rejected | published | obsoleted | archived | restored`

Each event persists actor id, timestamp, version id, and a `details` jsonb (rejection reason, content hash, prior/new status). Audit log is write-once — there is no update or delete path. Retention follows `metadata_schema.retention_days` once the version obsoletes.

## HTTP API surface (v1)

```
POST   /api/v2/templates                               create template + draft v1
GET    /api/v2/templates                               list (filterable)
GET    /api/v2/templates/{id}                          aggregate
POST   /api/v2/templates/{id}/versions                 create new version (from published+1)
GET    /api/v2/templates/{id}/versions/{n}             read version
POST   /api/v2/templates/{id}/versions/{n}/autosave/presign
POST   /api/v2/templates/{id}/versions/{n}/autosave/commit
PUT    /api/v2/templates/{id}/versions/{n}/schema       update metadata_schema / placeholder_schema / editable_zones
POST   /api/v2/templates/{id}/versions/{n}/submit
POST   /api/v2/templates/{id}/versions/{n}/review       body: { accept | reject, reason? }
POST   /api/v2/templates/{id}/versions/{n}/approve      body: { accept | reject, reason? }  — publishes on accept
POST   /api/v2/templates/{id}/archive
GET    /api/v2/templates/{id}/audit                     paginated audit log
PUT    /api/v2/templates/{id}/approval-config
```

Responses mirror documents_v2 envelope style (`{ data, meta }`). Error codes extend existing families (e.g., `invalid_state_transition`, `iso_segregation_violation`, `stale_base`, `content_hash_mismatch`).

## Migration

- `modules/templates` (legacy) becomes read-only via handler-level gate. Existing docs continue to reference it until superseded.
- No automated porting. Operators re-author templates in v2 manually; this is explicit product direction, not a limitation.
- A temporary `/templates` → `/templates-v2` redirect on the frontend nudges template authors to the new surface.

## Testing

- Go: table-driven tests for each command in `application` (success + every reject branch). Repository tests hit a real Postgres (docker-compose). State-machine tests exhaustively cover transitions.
- Frontend: Vitest for hooks (`useTemplateAutosave`, `useTemplatePlaceholders`). Playwright E2E for one golden path: create template → add placeholders + editable zone → submit → approve → create document → fill → submit → approve → verify DOCX exports correctly.
- Contract tests against the HTTP handlers using the same fixtures as docs_v2.

## Risks and open questions

1. **Eigenpal extensibility for placeholder chips / editable zones.** We need to confirm the library supports custom inline runs that round-trip through DOCX save/load. If not, an eigenpal upstream patch is required. Blocks template authoring UX. Should be validated in the implementation plan's first phase as a technical spike.
2. **Doc code sequence counter under concurrency.** Per-(tenant, doc_type, area) sequence must be monotonic. Use a Postgres sequence table with `SELECT ... FOR UPDATE` or advisory locks; decide in plan.
3. **Backfill of audit events.** Legacy templates won't carry v2 audit rows. Out of scope — legacy audit lives in old module.
4. **Role grant UI.** This spec assumes `template_author` role can be granted from the existing admin screens; confirm before implementation plan.

## Next step

Hand off to `writing-plans` to produce a phased implementation plan.
