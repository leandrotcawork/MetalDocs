# Foundation Placeholder Fill-In + Eigenpal Variable Fanout Design

**Spec 3 of foundational MetalDocs QMS sprint.**
Depends on Spec 1 (taxonomy/RBAC/controlled document registry) and Spec 2 (6-state approval state machine with signatures).

## Goal

Ship an ISO 9001 §7.5.3 / §8.5.1 regulator-grade placeholder and editable-zone system that cleanly separates three concerns:

1. **Template** — authored once by QMS admin, defines structure, placeholders, editable zones, and composition.
2. **Document** — instance of a template with values bound at creation and frozen at approval.
3. **Render** — final DOCX and PDF artifacts produced deterministically from the frozen template + values.

Must deliver the Word-native authoring feel users expect (in-canvas editing, content controls, restrict-editing semantics) while giving the QMS audit trail regulators require (double-freeze, triple-hash, immutable signatures binding content).

Non-goals: canonical XML normalization, multi-language placeholders, versioned resolver migration (see Out of Scope).

## Architecture

Three-layer model:

```
┌──────────────────────────────────────────────────────────────┐
│  TEMPLATE (templates_v2)                                     │
│  - body DOCX (SDT-wrapped content, bookmarked zones)         │
│  - placeholder_schema                                        │
│  - editable_zones_schema                                     │
│  - composition_config (header/footer sub-blocks + params)    │
│  - metadata_schema                                           │
└──────────────────────────────────────────────────────────────┘
                          │ (snapshot at document create)
                          ▼
┌──────────────────────────────────────────────────────────────┐
│  DOCUMENT REVISION (documents_v2)                            │
│  - placeholder_schema_snapshot  + schema_hash                │
│  - body_docx_snapshot_s3_key    + body_hash                  │
│  - composition_config_snapshot  + composition_hash           │
│  - editable_zones_schema_snapshot                            │
│  - placeholder_values           (row-per-value table)        │
│  - editable_zone_content        (row-per-zone OOXML)         │
│  - values_hash                  (set at approval freeze)     │
└──────────────────────────────────────────────────────────────┘
                          │ (fanout at approval)
                          ▼
┌──────────────────────────────────────────────────────────────┐
│  RENDER ARTIFACTS                                            │
│  - final_docx_s3_key   + content_hash (immutable)            │
│  - final_pdf_s3_key    + pdf_hash                            │
│  - reconstruction_attempts JSONB (append-only, informational)│
└──────────────────────────────────────────────────────────────┘
```

Boundaries:
- Template authoring lives in `templates_v2`.
- Document instance + fill-in + approval lives in `documents_v2`.
- Fanout (resolvers + processTemplate + Gotenberg) lives in a new `render` module.
- Placeholders and zones use eigenpal-native primitives — no fork.

## Components

### Placeholders (eigenpal `inlineSdt`)

Every placeholder is a Word Structured Document Tag (SDT) content control. Uses eigenpal public API (`@eigenpal/docx-js-editor` core export) with full Word SDT type coverage.

Domain (`internal/modules/templates_v2/domain/schemas.go` — extends existing):

```go
type Placeholder struct {
    ID       string          `json:"id"`
    Label    string          `json:"label"`
    Type     PlaceholderType `json:"type"`
    Required bool            `json:"required"`
    Default  any             `json:"default,omitempty"`
    Options  []string        `json:"options,omitempty"`

    // Spec 3 extensions:
    Regex       *string             `json:"regex,omitempty"`
    MinNumber   *float64            `json:"min_number,omitempty"`
    MaxNumber   *float64            `json:"max_number,omitempty"`
    MinDate     *string             `json:"min_date,omitempty"`
    MaxDate     *string             `json:"max_date,omitempty"`
    MaxLength   *int                `json:"max_length,omitempty"`
    VisibleIf   *VisibilityCondition `json:"visible_if,omitempty"`
    Computed    bool                `json:"computed,omitempty"`
    ResolverKey *string             `json:"resolver_key,omitempty"`
}

type VisibilityCondition struct {
    PlaceholderID string `json:"placeholder_id"`
    Op            string `json:"op"`    // eq | neq | in | not_in
    Value         any    `json:"value"`
}
```

Types supported (all eigenpal-native): `text | date | number | select | user | picture | computed`.

`sdtType` mapping: `text → richText`, `date → date`, `number → richText` (validated client+server), `select → dropdown` with `listItems`, `user → dropdown` populated from IAM, `picture → picture`, `computed → plainText locked`.

Tag convention (already in production at `frontend/apps/web/src/editor-adapters/eigenpal-template-mode.ts`):
```
tag = "placeholder:<placeholder_id>"
alias = <label>
```

### Editable zones (bookmarks)

Zones mark regions where document authors may insert rich OOXML content (paragraphs, tables, images) during fill-in.

Domain:

```go
type EditableZone struct {
    ID            string        `json:"id"`
    Label         string        `json:"label"`
    Required      bool          `json:"required"`
    ContentPolicy ContentPolicy `json:"content_policy"`
    MaxLength     *int          `json:"max_length,omitempty"`
}

type ContentPolicy struct {
    AllowTables    bool `json:"allow_tables"`
    AllowImages    bool `json:"allow_images"`
    AllowHeadings  bool `json:"allow_headings"`
    AllowLists     bool `json:"allow_lists"`
}
```

DOCX representation: OOXML `bookmarkStart` / `bookmarkEnd` pairs with name prefix `zone-start:<zone_id>`. Already implemented at `frontend/apps/web/src/editor-adapters/eigenpal-template-mode.ts` (`wrapZone`, `extractZones`).

### Composition config

Header/footer composition via toggleable sub-block catalogue.

```go
type CompositionConfig struct {
    HeaderSubBlocks []string                       `json:"header_sub_blocks"`
    FooterSubBlocks []string                       `json:"footer_sub_blocks"`
    SubBlockParams  map[string]map[string]any      `json:"sub_block_params"`
}
```

Sub-blocks rendered via `SubBlockRenderer` registry — each returns OOXML fragment. Catalogue v1: `doc_header_standard`, `revision_box`, `approval_signatures_block`, `footer_page_numbers`, `footer_controlled_copy_notice`.

### Computed resolvers

Typed registry populates `computed` placeholders at freeze time.

```go
type ComputedResolver interface {
    Key() string
    Version() int
    Resolve(ctx context.Context, in ResolveInput) (ResolvedValue, error)
}

type ResolveInput struct {
    TenantID, RevisionID, ControlledDocumentID string
    ProfileCodeSnapshot, AreaCodeSnapshot      string
    RegistryReader  registry.Reader
    RevisionReader  documents_v2.RevisionReader
    WorkflowReader  workflow.Reader
    TaxonomyReader  taxonomy.Reader
}

type ResolvedValue struct {
    Value       any
    ResolverKey string
    ResolverVer int
    InputsHash  string
    ComputedAt  time.Time
}
```

v1 resolvers: `doc_code`, `revision_number`, `effective_date`, `controlled_by_area`, `author`, `approvers`, `approval_date`.

Module location: `internal/modules/render/resolvers/`.

### Editor UX (in-canvas)

Authoring and fill-in both happen inside the eigenpal canvas — no side form panel.

- **Frozen template content** (outside placeholders and zones) wrapped in SDT with `lock: "sdtContentLocked"`.
- **Placeholders** wrapped in SDT with `lock: "unlocked"`, typed by placeholder type.
- **Zones** bookmarked regions with `lock: "unlocked"`, content constrained by `ContentPolicy`.
- **filterTransaction** plugin (ProseMirror public extension seam) rejects transactions whose changes touch ranges outside an unlocked SDT or zone — belt-and-suspenders alongside SDT locks.

## Data Flow

### Template author flow

1. QMS admin opens template v2 editor.
2. Drags placeholders and zones into canvas → SDT + bookmark nodes.
3. Edits schema side-panel (validation rules, `visible_if`, `computed` resolver key).
4. Edits composition config (toggle header/footer sub-blocks).
5. `UpdateSchemas` persists; body DOCX persisted separately via existing templates_v2 upload pipeline.

### Document create flow

1. User selects template → creates revision.
2. `documents_v2` snapshots from template:
   - `placeholder_schema_snapshot` + `placeholder_schema_hash`
   - `body_docx_snapshot_s3_key` + `body_hash`
   - `composition_config_snapshot` + `composition_hash`
   - `editable_zones_schema_snapshot`
3. Placeholder value rows created empty for all required placeholders.
4. Revision status = `draft`.

### Fill-in flow (draft)

1. Eigenpal canvas loads snapshotted body DOCX.
2. User edits placeholders (typed inputs per `sdtType`) and zones (rich OOXML).
3. Each edit upserts `document_placeholder_values` or `document_editable_zone_content` row.
4. Computed placeholders resolve on load + on dependency change; stored with `inputs_hash` so reveal is cache-friendly.
5. No DOCX is produced during draft — UI state and DB rows only.

### Approval freeze + fanout

Triggered by Spec 2's transition into `approved`:

1. Validate all required placeholders filled + constraints satisfied.
2. Resolve all `computed` placeholders; store `resolver_key`, `resolver_ver`, `inputs_hash`.
3. Compute `values_hash = sha256(canonical_json(all_values_by_id))`.
4. Single-pass `processTemplate` over body DOCX:
   - Replace SDT placeholders with their resolved values.
   - Inject zone OOXML between bookmark pairs.
   - Apply composition config (header/footer sub-blocks via `SubBlockRenderer`).
5. Persist final DOCX → S3 at `final_docx_s3_key`, store `content_hash`.
6. Enqueue `docgen_v2_pdf` service bus job → Gotenberg DOCX→PDF.
7. PDF worker writes `final_pdf_s3_key`, `pdf_hash`, `pdf_generated_at`.
8. Spec 2 signature binds `content_hash` (DOCX-only). PDF is a convenience artifact; if regenerated, new `pdf_hash` is acceptable — signature is not attached to PDF.

### Viewer flow

1. Consumer opens `/documents/:id/view`.
2. Backend checks area RBAC (Spec 1), revision status = `approved`.
3. Returns signed S3 URL to `final_pdf_s3_key`.
4. Consumer views read-only PDF. No eigenpal load. No re-render.

### Re-render (forensic only)

Only triggered by explicit admin action (e.g. library recovery). Never in happy path.

1. Re-run fanout from snapshots + frozen values.
2. Hash new bytes.
3. Append to `revisions.reconstruction_attempts`:
   ```json
   {
     "rendered_at": "...",
     "eigenpal_ver": "...",
     "docxtemplater_ver": "...",
     "bytes_hash": "...",
     "matches_original": true|false
   }
   ```
4. **Never** overwrite original `content_hash` or `final_docx_s3_key`. Signature stays valid against original bytes.

### Schema — new columns and tables

```sql
ALTER TABLE documents_v2_revisions ADD COLUMN
  placeholder_schema_snapshot     JSONB       NOT NULL,
  placeholder_schema_hash         BYTEA       NOT NULL,
  composition_config_snapshot     JSONB       NOT NULL,
  composition_config_hash         BYTEA       NOT NULL,
  editable_zones_schema_snapshot  JSONB       NOT NULL,
  body_docx_snapshot_s3_key       TEXT        NOT NULL,
  body_docx_hash                  BYTEA       NOT NULL,
  values_frozen_at                TIMESTAMPTZ,
  values_hash                     BYTEA,
  final_docx_s3_key               TEXT,
  final_pdf_s3_key                TEXT,
  pdf_hash                        BYTEA,
  pdf_generated_at                TIMESTAMPTZ,
  reconstruction_attempts         JSONB       NOT NULL DEFAULT '[]'::jsonb;

CREATE TABLE document_placeholder_values (
  tenant_id          TEXT         NOT NULL,
  revision_id        UUID         NOT NULL,
  placeholder_id     TEXT         NOT NULL,
  value_text         TEXT,
  value_typed        JSONB,
  source             TEXT         NOT NULL,        -- user | computed | default
  computed_from      TEXT,                         -- resolver_key if computed
  resolver_version   INT,
  inputs_hash        BYTEA,
  validated_at       TIMESTAMPTZ,
  created_at         TIMESTAMPTZ  NOT NULL,
  updated_at         TIMESTAMPTZ  NOT NULL,
  PRIMARY KEY (tenant_id, revision_id, placeholder_id)
);

CREATE TABLE document_editable_zone_content (
  tenant_id     TEXT   NOT NULL,
  revision_id   UUID   NOT NULL,
  zone_id       TEXT   NOT NULL,
  content_ooxml TEXT   NOT NULL,
  content_hash  BYTEA  NOT NULL,
  PRIMARY KEY (tenant_id, revision_id, zone_id)
);
```

## Freeze Model

**Double-freeze:**
1. At document create — schema + body + composition snapshotted to revision row. Template changes after create do not affect existing revisions.
2. At approval — values + zone_content + `values_hash` frozen. Spec 2 signature binds `content_hash + values_hash + schema_hash`.

**Triple-hash:**
- `placeholder_schema_hash` — structure contract
- `values_hash` — filled data
- `content_hash` — final DOCX bytes

All three persisted, all three referenced by signature payload. Any drift detectable.

## Error Handling

- **Validation failures** (regex / min / max / length / required) → 422 with `{placeholder_id, rule, message}`. Approval blocked.
- **Cyclic `visible_if`** → detected at template save via topo-sort; 422 with cycle path.
- **Resolver failure at freeze** → approval blocked; surface `{resolver_key, error}`. Retry allowed.
- **Unknown `resolver_key`** at freeze → approval blocked with `unknown_resolver` error; template requires fix.
- **Duplicate placeholder_id / zone_id** → already handled in `UpdateSchemas` (existing code at `internal/modules/templates_v2/application/schema.go`).
- **`processTemplate` failure** → approval aborts, revision stays in pre-approval state, error logged with template + revision IDs.
- **Gotenberg timeout or failure** → service bus retry policy (existing `docgen_v2_pdf` worker). DOCX is already persisted; PDF regenerated on retry. Approval not blocked by PDF failure — PDF is best-effort artifact.
- **Stale base at fill-in** (template updated after snapshot) → no effect; revision uses its snapshot.

## Testing Approach

- **Round-trip test** — serialize placeholder → write DOCX → parse DOCX → assert same placeholder ID, type, alias round-tripped through eigenpal `inlineSdt`.
- **Zone round-trip** — wrap zone → write DOCX → `extractZones` → assert boundaries preserved.
- **filterTransaction test** — attempt edit inside locked SDT → assert transaction rejected; attempt edit inside zone → assert accepted.
- **Freeze idempotency** — approve same revision twice (hypothetical) → identical DOCX bytes expected (documented behavior, acceptable minor drift under library upgrade per `reconstruction_attempts`).
- **Resolver contract test** — each v1 resolver given fixed `ResolveInput` → asserted `ResolvedValue` with stable `inputs_hash`.
- **Cycle detection test** — `visible_if` A→B→A at schema save → 422.
- **RBAC test** — consumer without area grant hits `/view` → 403.
- **Viewer test** — approved revision → signed PDF URL returned; draft revision → 404 for consumer.
- **Reconstruction test** — mock eigenpal/docxtemplater version bump → re-render appends `reconstruction_attempts` row with `matches_original: false`; `content_hash` and signature intact.

## Out of Scope (v1)

- **Canonical XML normalization** for byte-deterministic re-render (Codex Call 2 flagged — deferred to Future Hardening; `reconstruction_attempts` makes drift observable without blocking v1).
- **Multi-language placeholders** — labels and content are tenant's primary language only.
- **Nested conditional sub-blocks** — composition config is flat toggle list; no visible_if on sub-blocks.
- **Versioned resolver migration** — all v1 resolvers pinned at version 1; upgrade path designed later.
- **Real-time collaborative fill-in** — single-author edit at a time, optimistic concurrency via `ExpectedContentHash` like existing `UpdateSchemas`.
- **Pre-Spec-3 approved documents** — stay as-is, no backfill, no fanout. Only new revisions use the double-freeze pipeline.
- **PDF signature binding** — signature binds DOCX only. PDF is a convenience artifact, regeneratable.
- **Signed DOCX (XAdES / CAdES)** — v1 relies on Spec 2's PKI-like password re-auth signature over hashes; in-artifact digital signatures deferred.
