# MDDM Template Admin (Phase 2) Design

## Goal

Deliver a production-grade admin UI that lets non-technical quality managers create, edit, version, and publish MDDM templates visually — replacing the current workflow of editing Go code and running database migrations. The engine from Phase 1 stays untouched: Phase 2 only changes **where** template JSON comes from.

### Success criteria

- Quality managers can create a new template, edit its structure + per-block style + capabilities, preview it live, and publish it — without a developer
- Template editor reuses the existing MDDM BlockNote editor as the rendering surface; no second editor is built
- Publish is **fail-closed and deterministic**: zero silent coercion of published template data
- Draft-then-publish lifecycle with optimistic locking prevents concurrent admin overwrites
- RBAC scoped per profile: view / edit / publish / export permissions enforced server-side
- Import/export produces the exact same JSON format used by Phase 1
- Existing documents are never retroactively affected by template changes — `template_ref` is immutable
- The Phase 1 engine (interpreters, ViewModels, codecs, emitters) is not modified

### Non-goals

- Version diff / side-by-side comparison between template versions
- Undo/redo beyond what BlockNote provides natively
- Template usage analytics
- Template inheritance (one template extending another)
- Real-time collaboration / multi-cursor editing
- Template marketplace or cross-organization sharing
- Dark mode, landscape pages, custom fonts per template
- Drag-to-reorder sections at author time (template-locked)
- Automatic migration of existing documents when a template is republished
- Conditional blocks (show/hide based on document data)

### Relationship to existing specs

- **2026-04-13 MDDM Template Engine** — this spec is the Phase 2 companion. Phase 1 defined the engine, typed codecs, Layout Interpreters, ViewModels, React+DOCX emitters, and the declarative template JSON format. Phase 2 adds the admin UI + database storage + lifecycle service on top. The template JSON format is unchanged.
- **2026-04-12 React Parity Layer** — the embedded editor surface consumes the same interpreters and ViewModels. The admin's live preview IS the document author's future experience.
- **2026-04-10 Unified Document Engine** — Layout IR tokens, DOCX emitters, and version pinning are retained unchanged.

## Architecture

### Three-layer design

```
┌─────────────────────────────────────────────────┐
│  Template Editor View (React)                    │
│  Metadata Bar (name, profile, theme, status)     │
│  ┌───────────┬──────────────────┬──────────────┐ │
│  │ Block     │  MDDM Editor     │  Property    │ │
│  │ Palette   │  (live surface)   │  Sidebar     │ │
│  │           │                   │              │ │
│  │ Section   │  Existing Phase 1 │ Propriedades │ │
│  │ Field     │  BlockNote editor │ Estilo       │ │
│  │ Table     │  — UNMODIFIED     │ Capacidades  │ │
│  │ Rich      │                   │              │ │
│  │ Repeat    │                   │              │ │
│  └───────────┴──────────────────┴──────────────┘ │
└─────────────────────────────────────────────────┘
         │ REST API
         ▼
┌─────────────────────────────────────────────────┐
│  Template API (Go HTTP handlers)                 │
│  RBAC middleware + lockVersion validation        │
└─────────────────────────────────────────────────┘
         │
         ▼
┌─────────────────────────────────────────────────┐
│  Template Domain Service (Go)                    │
│  Lifecycle, validation (strict + lenient),       │
│  version management, audit events, locking       │
└─────────────────────────────────────────────────┘
         │
         ▼
┌─────────────────────────────────────────────────┐
│  PostgreSQL                                      │
│  document_template_versions (existing)           │
│  template_drafts (new)                           │
│  template_audit_log (new)                        │
└─────────────────────────────────────────────────┘
```

### Key architectural principles

1. **The MDDM editor is embedded, not extended.** No `isAdminMode` prop, no admin-specific branches inside the editor. The wrapper intercepts selection events and uses the editor's public `updateBlock()` API to write codec-serialized props back.
2. **The template JSON format is frozen.** Phase 1 defined it; Phase 2 produces the same format. This is the contract between admin UI, database, and engine.
3. **Business rules live in the Go domain service.** Draft/publish/deprecate transitions, validation, RBAC, audit — all enforced server-side. The React admin UI is a thin client.
4. **Fail-closed publish.** Drafts are permissive, publish is strict. Any unrecognized field or invalid value blocks publish with a specific error — no silent coercion.
5. **Nested under profiles.** Templates are accessed through the existing RegistryExplorer profile view, not as a top-level feature. Matches the mental model: "I manage my document type, templates are part of that."

## Components

### 1. Template List (inside Profile view)

Embedded inside the existing profile detail page in `RegistryExplorer`. Renders a table:

| Column | Content |
|---|---|
| Nome | Template name |
| Status | `draft` / `published` / `deprecated` badge |
| Versão | Current version (v0 for never-published drafts) |
| Atualizado em | Last updated timestamp |
| Publicado por | Actor who last published (empty for drafts) |
| Ações | Edit / Clone / Export / Delete / Deprecate |

Above the table: **"Novo template"** and **"Importar"** buttons.

Action rules:
- **Novo template** — creates a blank draft pre-linked to this profile, redirects to editor
- **Edit** — opens full-screen `TemplateEditorView`
- **Clone** — creates a draft copy with a new key and same profile
- **Export** — downloads template JSON file (published version only)
- **Delete** — only drafts that were never published. Published templates can only be deprecated.
- **Deprecate** — published templates only. Deprecated templates don't appear in document creation.

Buttons hidden based on RBAC (see §4.4). Server enforces on every endpoint.

### 2. Template Editor View

Full-screen page at `/admin/profiles/:profileCode/templates/:templateKey`.

**Metadata Bar (top):**
- Template name (editable text field)
- Profile badge (read-only, from route param)
- Theme editor: accent color picker. Changes propagate live to the editor surface via CSS custom properties.
- Status badge: `draft` / `published` / `deprecated`
- Action buttons: "Salvar rascunho", "Publicar", "Visualizar DOCX", "Descartar rascunho"
- lockVersion indicator (small, technical — for debugging): "Edição #N"

**Block Palette (left, collapsible):**
- Block type buttons: Section, Field, FieldGroup, DataTable, Repeatable, RichBlock
- Each with icon + label
- Click to append to current container, or drag onto surface to insert at position
- Sections are top-level only; fields/tables/rich/repeatable are section-scoped
- Insertion rules enforced by the wrapper before calling `editor.insertBlocks()`

**Editor Surface (center):**
- The Phase 1 MDDM BlockNote editor, unmodified
- Renders template blocks using the same interpreters document authors see
- Admin authors are treated as having full editing capability — capabilities are data being authored, not being enforced at this level
- Visual admin indicators (lock icons, zone borders, "dynamic" badges on tables) overlaid by the wrapper via DOM decoration, not by the editor

**Property Sidebar (right):**
Appears when a block is selected. Three tabs:

**Propriedades** — block-specific props:
- Section: title
- Field: label, placeholder, value type
- FieldGroup: columns (1 or 2)
- DataTable: column definitions (key, label, width)
- Repeatable: item template configuration
- RichBlock: label

**Estilo** — codec-defined style fields:
- Color pickers for background/color fields
- Dropdowns for font weight, text alignment
- Number inputs for dimensions (with unit suffix: mm, pt, %)
- Toggles for boolean styles
- Only fields valid for the selected block type are shown (schema-driven from codec)
- "Reset to defaults" button per field

**Capacidades** — capability toggles:
- Universal: locked, removable, reorderable
- Per-block-type capabilities from Phase 1 spec: editableZones, mode (fixed/dynamic for tables), addRows/removeRows, maxItems/minItems, etc.
- Schema-driven from the codec's `capabilitiesSchema`

Changes in the sidebar immediately call `editor.updateBlock(blockId, { props: { styleJson, capabilitiesJson } })`, serialized via the block's codec. The editor re-renders through the interpreter → ViewModel pipeline. The admin sees changes live.

### 3. Validation Panel

Slides in from the bottom when "Publicar" is clicked and validation fails.

Each error entry:
- Plain-language message (PT-BR)
- Block reference (block type + human-readable position, e.g., "Seção 3 — Entradas e Saídas")
- Click → selects the offending block in the editor, scrolls into view, focuses the relevant sidebar tab

Error categories surfaced:
- Missing required props (section without title)
- Invalid capabilities (addRows on a non-table block)
- Column widths not summing to 100% on DataTable
- maxItems < minItems on Repeatable
- Theme references pointing to undefined keys
- **Strict codec failures** (unknown fields, invalid values — see §5)

Publish is blocked until all errors are cleared. "Salvar rascunho" always works.

### 4. Preview

Two modes:

**Live preview (always on):**
The editor surface IS the preview. Because the embedded MDDM editor uses the same interpreters and ViewModels as the document author experience, what the admin sees is what authors will see. No second rendering path.

**Export preview (on-demand):**
"Visualizar DOCX" button triggers the existing DOCX export pipeline on the current template blocks, downloads a `.docx` file. The admin opens it in Word to verify pixel-level output. PDF preview uses the existing `toExternalHTML` → Gotenberg pipeline.

Both export paths use the same Phase 1 emitters — no preview-specific rendering code.

### 5. Import / Export UI

**Export:**
- Action button on any template row
- Calls `GET /api/v1/templates/:key/export`
- Downloads `.json` file with clean template format (internal fields stripped)

**Import:**
- "Importar" button on profile's template list
- File picker accepts `.json`
- On upload: server parses, validates via **lenient codec pass**, creates a draft
- If any fields were stripped, the returned draft is flagged `hasStrippedFields: true`
- Editor opens the draft with a **banner**: "Esta importação contém campos não reconhecidos que foram removidos. Revise o template antes de publicar."
- The banner persists until the admin clicks "Reconheço as alterações" — only then can they publish

## Data Flow

### Template CRUD Lifecycle

```
Create:
  Admin clicks "Novo template"
  → POST /api/v1/templates { profileCode, name }
  → Service creates draft (base_version 0, status "draft")
  → Returns draft + lockVersion: 1
  → Redirects to TemplateEditorView

Edit draft:
  Admin edits blocks, style, capabilities in editor
  → "Salvar rascunho" (manual; no auto-save in v1)
  → PUT /api/v1/templates/:key/draft { blocks, theme, meta, lockVersion }
  → Service validates lockVersion (optimistic lock)
  → Service runs lenient codec pass on blocks (strip unknowns, default invalids)
  → Saves, increments lockVersion, returns new lockVersion

Publish:
  Admin clicks "Publicar"
  → Frontend runs validateTemplate() client-side (fast feedback)
  → POST /api/v1/templates/:key/publish { lockVersion }
  → Service re-loads draft from DB (source of truth)
  → Service runs STRICT codec pass: any unknown field or invalid value → 422
  → Service runs validateTemplate() with all invariants
  → If all pass: bumps version, inserts row into document_template_versions, deletes draft row, writes audit event
  → Returns published version

Edit published:
  Admin opens a published template that has no draft
  → GET /api/v1/templates/:key returns published
  → Admin clicks "Editar"
  → POST /api/v1/templates/:key/edit (auto-creates draft from published)
  → Service deep-clones published blocks into template_drafts with base_version = current published version
  → Returns draft, lockVersion: 1
  → Editor reloads in draft mode

Clone:
  → POST /api/v1/templates/:key/clone { newName }
  → Service generates new templateKey, deep-clones blocks
  → Creates draft, version 0, status "draft"
  → Writes audit event
  → Returns new key, admin redirected

Deprecate:
  → POST /api/v1/templates/:key/deprecate
  → Latest published version's status → "deprecated"
  → Writes audit event
  → Existing documents with template_ref unaffected

Delete:
  → DELETE /api/v1/templates/:key
  → Only allowed if no published version exists (draft only)
  → Deletes draft row
  → Writes audit event
```

### Draft Editing Interaction Flow

```
Load:
  GET /api/v1/templates/:key
  → Returns { blocks, theme, meta, status, lockVersion, hasStrippedFields? }
  → Frontend initializes editor with blocks
  → Sidebar populated with theme, meta

Block selection:
  Admin clicks a block in editor
  → BlockNote fires selection change event
  → Wrapper reads block.props.styleJson, block.props.capabilitiesJson
  → Wrapper parses via codec (lenient) → populates Estilo, Capacidades tabs
  → Wrapper reads type-specific props → populates Propriedades tab

Style change:
  Admin changes a color in the Estilo tab
  → Sidebar state updates
  → Codec serializes the updated style object
  → editor.updateBlock(blockId, { props: { styleJson: serialized } })
  → Editor re-renders the block via interpretSection() → new ViewModel → new colors
  → Change is local until "Salvar rascunho"

Save:
  → Frontend reads all blocks from editor state
  → PUT /api/v1/templates/:key/draft with lockVersion
  → On 409: toast "Template editado por outro administrador. Recarregar."
  → On 200: updates local lockVersion, toast "Rascunho salvo"
```

### Document Creation (unchanged from Phase 1)

```
User clicks "Novo documento"
  → GET /api/v1/document-templates?profileCode=po
  → Returns published templates for profile
  → User picks template (or default used)
  → instantiateTemplate(template) — structuredClone(blocks)
  → POST /api/v1/documents with template_ref
  → Editor opens with cloned blocks, Phase 1 capability enforcement
```

The engine and document creation flow do not change. Phase 2 only changes the origin of template JSON.

## Backend Domain Service & API

### Domain Service (Go)

Location: `internal/modules/documents/domain/template_lifecycle.go` (new) alongside existing `template.go`.

Key types:

```go
type TemplateStatus string
const (
    TemplateStatusDraft      TemplateStatus = "draft"
    TemplateStatusPublished  TemplateStatus = "published"
    TemplateStatusDeprecated TemplateStatus = "deprecated"
)

type TemplateDraft struct {
    TemplateKey  string
    ProfileCode  string
    BaseVersion  int              // 0 for new, N when editing published vN
    Name         string
    Theme        json.RawMessage
    Blocks       json.RawMessage  // MDDM blocks — same format as Phase 1
    LockVersion  int              // optimistic locking, starts at 1
    HasStrippedFields bool        // set by Import if lenient codec dropped fields
    CreatedBy    string
    UpdatedAt    time.Time
}
```

Service methods (all take actor for RBAC + audit):

| Method | Validations |
|---|---|
| `CreateDraft(profileCode, name, actor)` | Profile exists, actor has `template:edit` |
| `SaveDraft(key, blocks, theme, meta, lockVersion, actor)` | lockVersion match, actor has `template:edit`, lenient codec pass |
| `Publish(key, lockVersion, actor)` | lockVersion match, actor has `template:publish`, **strict codec pass**, full `validateTemplate()`, `hasStrippedFields == false` |
| `Deprecate(key, actor)` | Latest version is published, actor has `template:publish` |
| `Clone(sourceKey, newName, actor)` | Source exists, actor has `template:view` on source + `template:edit` on destination |
| `EditPublished(key, actor)` | Published version exists, actor has `template:edit` |
| `GetTemplate(key)` | Returns draft if exists, else latest published |
| `ListByProfile(profileCode, actor)` | Filters by `template:view` |
| `Delete(key, actor)` | Draft only, no published version exists, actor has `template:edit` |
| `Export(key, actor)` | Published only, actor has `template:export` |
| `Import(profileCode, jsonBytes, actor)` | Lenient codec pass, sets `HasStrippedFields` if any fields dropped, actor has `template:edit` |
| `DiscardDraft(key, actor)` | Deletes draft row (published version preserved), actor has `template:edit` |

### Codec: Lenient vs Strict

Every block codec from Phase 1 gains a strict variant:

```go
type BlockCodec interface {
    ParseStyleLenient(raw json.RawMessage) (Style, []StrippedField)
    ParseStyleStrict(raw json.RawMessage) (Style, error)
    ParseCapsLenient(raw json.RawMessage) (Caps, []StrippedField)
    ParseCapsStrict(raw json.RawMessage) (Caps, error)
    SerializeStyle(Style) json.RawMessage
    SerializeCaps(Caps) json.RawMessage
    DefaultStyle() Style
    DefaultCaps() Caps
}
```

- **Lenient** — current Phase 1 behavior. Unknown fields stripped (returned as `StrippedField[]` for reporting), invalid values replaced with defaults. Used for: loading drafts, loading old published templates, import, any read path that must not crash on bad data.
- **Strict** — fails on any unknown field or invalid value. Returns a specific error naming the field and block. Used for: **publish validation only**.

Publish runs every block through strict parse. If any block fails: 422 with array of errors, each naming the block ID and field.

### Round-trip Invariant

CI test enforces: for every published template, `serialize(parseStrict(serialize(blocks)))` produces byte-identical output to the stored `blocks_json`. This catches emitter drift that would silently alter published templates.

### Optimistic Locking

Every mutation on a draft requires `lockVersion`. Flow:

1. Client loads draft → receives `lockVersion: 5`
2. Client saves → sends `lockVersion: 5`
3. Server checks: stored lockVersion == 5? If yes → save, set to 6, return 6
4. Another admin saved in between (lockVersion now 6) → 409 Conflict
5. Client shows: "Este template foi editado por outro administrador. Recarregue para ver as alterações."

Simple integer increment. No merge, no conflict resolution. First writer wins, second writer reloads.

### REST API

All endpoints under `/api/v1/templates`:

| Method | Path | Body | Returns |
|---|---|---|---|
| `GET` | `/templates?profileCode=X` | — | Template list scoped to actor's profile permissions |
| `POST` | `/templates` | `{ profileCode, name }` | `201 { templateKey, lockVersion }` |
| `GET` | `/templates/:key` | — | `{ ...template, lockVersion, hasStrippedFields? }` |
| `PUT` | `/templates/:key/draft` | `{ blocks, theme, meta, lockVersion }` | `{ lockVersion }` or `409` |
| `POST` | `/templates/:key/publish` | `{ lockVersion }` | `{ version }` or `422 { errors[] }` or `409` |
| `POST` | `/templates/:key/edit` | — | `{ lockVersion }` — creates draft from published |
| `POST` | `/templates/:key/deprecate` | — | `{ status: "deprecated" }` |
| `POST` | `/templates/:key/clone` | `{ newName }` | `201 { templateKey, lockVersion }` |
| `DELETE` | `/templates/:key` | — | `204` |
| `POST` | `/templates/:key/discard-draft` | — | `204` |
| `GET` | `/templates/:key/export` | — | JSON file (Content-Disposition: attachment) |
| `POST` | `/templates/import` | multipart file + profileCode | `201 { templateKey, hasStrippedFields, strippedFields[] }` |

### RBAC

Permissions scoped per profile:

| Permission | Granted to | Enables |
|---|---|---|
| `template:view` | Any admin with profile access | List, GET, Export UI visible |
| `template:edit` | Template editors for this profile | Create, SaveDraft, Edit, Clone, Import, Delete (drafts), Discard |
| `template:publish` | Template publishers for this profile | Publish, Deprecate |
| `template:export` | Same as `template:view` | Export endpoint |

Enforced in middleware on every endpoint. Frontend reads the actor's permissions on load and hides disallowed buttons, but this is UX only — server is authoritative.

### Audit Log

```sql
CREATE TABLE template_audit_log (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    template_key TEXT NOT NULL,
    version      INT,
    action       TEXT NOT NULL,  -- created | draft_saved | published | deprecated | cloned | imported | deleted | draft_discarded | edit_started
    actor_id     TEXT NOT NULL,
    diff_summary TEXT,           -- human-readable PT-BR summary
    trace_id     TEXT,           -- request trace ID for correlation
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_template_audit_log_key ON template_audit_log(template_key);
CREATE INDEX idx_template_audit_log_actor ON template_audit_log(actor_id);
```

Append-only. Service writes one row per mutation. Never UPDATE, never DELETE.

### Database Schema

The existing `document_template_versions` table (from Phase 1 seed migrations) stores published versions — unchanged.

New tables:

```sql
CREATE TABLE template_drafts (
    template_key   TEXT PRIMARY KEY,
    profile_code   TEXT NOT NULL REFERENCES document_profiles(code),
    base_version   INT NOT NULL DEFAULT 0,
    name           TEXT NOT NULL,
    theme_json     JSONB NOT NULL DEFAULT '{}'::jsonb,
    meta_json      JSONB NOT NULL DEFAULT '{}'::jsonb,
    blocks_json    JSONB NOT NULL,
    lock_version   INT NOT NULL DEFAULT 1,
    has_stripped_fields BOOLEAN NOT NULL DEFAULT false,
    stripped_fields_json JSONB,  -- details for banner display
    created_by     TEXT NOT NULL,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_template_drafts_profile ON template_drafts(profile_code);

CREATE TABLE template_audit_log ( ... as above ... );
```

On publish: service performs in a transaction — deletes the draft row, inserts into `document_template_versions` with `version = base_version + 1`, writes audit event.

## Error Handling

### Client-side

| Scenario | Behavior |
|---|---|
| Validation fails on publish (422) | Validation panel appears with clickable errors naming blocks and fields. Publish blocked. Drafts still saveable. |
| Optimistic lock conflict (409) | Toast: "Este template foi editado por outro administrador. Recarregue para ver as alterações." Reload button. No auto-merge. |
| Network error on save | Toast with retry button. Unsaved changes preserved in editor state. |
| Import file invalid JSON | Inline error in import dialog. File rejected. |
| Import succeeded but fields stripped | Editor opens with persistent banner. Admin must click "Reconheço as alterações" (clears `hasStrippedFields` flag on server) before publish is allowed. |
| Template not found (404) | Redirect to profile's template list with toast. |
| Permission denied (403) | Toast. Buttons proactively hidden from that user. |

### Server-side

| Scenario | Behavior |
|---|---|
| Lenient codec parse | Strips unknowns, defaults invalids, never crashes. Drafts and imports tolerate messy input. |
| Strict codec parse at publish | 422 with structured array: `{ blockId, blockType, field, reason }`. Publish blocked. |
| validateTemplate() fails at publish | 422 with structured errors — empty template, bad column widths, missing required props, etc. |
| Profile doesn't exist | 400: "Perfil documental não encontrado." |
| Delete a published template | 400: "Templates publicados não podem ser excluídos. Use 'Depreciar'." |
| Deprecate already-deprecated | 400 with specific message |
| Database constraint violation | 500, fully logged server-side with trace_id. Generic message to user. |
| hasStrippedFields true on publish | 422: "Este template foi importado e contém campos removidos. Reconheça as alterações antes de publicar." |

### Data Integrity Guarantees

- **Codec is the gatekeeper.** All JSON read/write paths go through typed codecs. No raw `json.Unmarshal` on template data anywhere.
- **Publish is fail-closed and deterministic.** Zero silent coercion. Any field that would be stripped or defaulted at publish time blocks the publish with a specific error.
- **Server-side validation is authoritative.** Client-side validation is for fast feedback; server re-runs all checks on publish before persisting.
- **Round-trip invariant enforced in CI.** For published templates: `serialize(parseStrict(blocks_json))` is byte-identical to `blocks_json`. Protects against emitter drift corrupting published data.
- **Drafts can be messy.** Only publish enforces strict validation. Admins can save incomplete work.
- **Published versions are immutable.** Once inserted into `document_template_versions`, a row is never updated. Editing creates a draft; publishing inserts a new version row.
- **Documents are immutable against template changes.** `template_ref` on a document points to the specific version used at instantiation time. Republishing a template does not affect existing documents.

## Testing Approach

### Backend (Go)

**Domain Service unit tests:**
- Lifecycle transitions: valid (draft→publish→deprecate) and invalid (publish→draft, deprecate→publish) rejected
- Optimistic locking: concurrent saves — second returns 409
- Clone: deep independence of blocks, new key, draft status, correct profile
- Import: valid JSON accepted, invalid JSON rejected, stripped fields recorded, `hasStrippedFields` flag set
- Publish preconditions: `hasStrippedFields` blocks publish, strict codec failures block publish, validateTemplate invariants block publish
- Delete: only drafts without published versions
- RBAC: each method checks required permission, unauthorized returns error

**API integration tests:**
- Full CRUD cycle through HTTP endpoints
- 409 on lock conflict
- 422 on strict codec failure at publish (specific error structure)
- 403 on unauthorized actions
- Import → edit → publish round trip
- Export → re-import produces equivalent draft (ignoring stripped-fields flag on clean data)

**Strict codec tests (per block type):**
- Valid block → strict parse succeeds
- Block with unknown field → strict parse fails with specific error
- Block with invalid value type → strict parse fails with specific error
- Same block through lenient parse → succeeds, returns stripped field list

**Round-trip invariant test (CI):**
- For every seed template and every published template in the database:
- `serialize(parseStrict(blocks_json))` == `blocks_json` (byte-equal)
- Fails CI if an emitter change would silently alter existing published data

### Frontend (Vitest)

**Property Sidebar unit tests:**
- Section block → renders correct style controls (color picker, font size, height)
- DataTable block → renders mode toggle (fixed/dynamic), column editor, row limits
- Value change → calls `updateBlock()` with codec-serialized output
- Unknown block type → shows generic info panel, no crash
- Capability toggle → updates `capabilitiesJson` via codec

**Template List tests:**
- Published templates show Edit, Clone, Export, Deprecate. No Delete.
- Draft templates show Edit, Delete, Discard. No Deprecate.
- Deprecated templates show Export, Clone. No Edit.
- Permission-based button visibility: user without `template:publish` sees no Publish/Deprecate buttons

**Validation Panel tests:**
- Renders errors with human-readable messages
- Click error → fires block selection callback with correct block ID
- No errors → panel hidden, Publish enabled
- Stripped-fields banner persistence until acknowledge

**Template Editor integration tests (Vitest + Testing Library):**
- Create editor with template blocks → select a block → sidebar populates from codec → change a style value → block re-renders with new value → serialize all blocks → codec parse produces the updated value
- Full CRUD flow with mocked API: create → edit → save draft → publish → verify published version has correct blocks
- 409 lock conflict handling
- Import with stripped fields → banner appears → publish blocked until acknowledged

### Golden Fixtures (existing, extended)

Phase 1 fixtures continue unchanged. New fixtures:
- Template with all block types → interpret → emit → compare DOCX/HTML against golden
- Template with custom theme → verify theme colors propagate through interpreters
- Template with mixed capabilities (some locked, some dynamic) → verify correct emitted output
- Imported-then-published template → verify stripped fields did not leak into emitted output

### What We Don't Test Automatically

- Pixel-perfect visual rendering — manual review + browser automation where available
- BlockNote framework internals — trusted third-party
- MDDM engine behavior — Phase 1 tests cover this, unchanged
- Drag-and-drop interaction in block palette — manual QA
- Color picker usability — manual QA

## Out of Scope

- Version diff / side-by-side comparison
- Undo/redo beyond BlockNote native
- Template usage analytics
- Template inheritance
- Real-time collaboration
- Auto-save (v1 is manual save only)
- Template marketplace
- Dark mode, landscape pages, custom fonts per template
- Drag-to-reorder sections at author time
- Automatic document migration on template republish
- Conditional blocks (show/hide on data)
- Multi-language template content (templates are PT-BR in v1)
