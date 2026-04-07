# MetalDocs Document Model (MDDM) — Foundational Design

**Date:** 2026-04-07
**Status:** Locked architecture, ready for implementation
**Sprint:** Foundational, ~4-6 weeks
**Author:** Leandro Theodoro (with collaborative validation by Claude + Codex over 9 review rounds)

## Supersession notice

This document is the single source of truth for the MetalDocs document model and editor architecture. The following prior documents are **deprecated** and must not be implemented. They remain in the repository for historical context only:

- `2026-04-06-document-authoring-v1-freeze.md`
- `2026-04-06-po-browser-template-final-form.md`
- `2026-04-06-po-browser-template-redesign.md`
- `2026-04-06-po-browser-template-visual-polish.md`
- `2026-04-06-po-production-browser-template-design.md`
- `2026-04-04-template-assigned-browser-document-editor-design.md`
- `2026-04-02-governed-document-canvas-design.md`
- `2026-04-02-carbone-removal-docgen-unification-design.md`
- `2026-04-01-content-builder-fixes-design.md`

The CKEditor + RestrictedEditingMode + HTMLtoDOCX pipeline AND the schema-based docgen runtime (`apps/docgen/src/runtime/`) are both **deprecated and will be removed**. MDDM replaces both as the only document authoring + export pipeline.

---

## 1. Goal

Build the foundation for MetalDocs as a long-lived, professional quality control document system:

- **Templates and documents** authored in a block-based, structured editor
- **DOCX export** that produces professional Word output (real heading styles, proper tables, no font collapse)
- **Quality control workflows**: draft → approval → released → archived
- **Frozen historical records**: released DOCX files are immutable artifacts, never re-rendered
- **Audit-grade revision tracking**: structured diffs, time-stamped, FDA 21 CFR Part 11 / ISO 9001 ready
- **A platform that survives multi-year evolution**: versioned schema, forward migrations, deterministic canonical form, no third-party coupling

This is not a tactical fix to one template. It is the foundational rebuild of how MetalDocs handles documents.

## 2. Architecture overview

```
┌─────────────────────────────────────────────────────────────────┐
│                    FRONTEND (React + TS)                        │
│                                                                 │
│  ┌───────────────────────────────────────────────────────────┐  │
│  │  MDDM Editor (BlockNote + custom block schema)            │  │
│  │  ─────────────────────────────────────────────────────    │  │
│  │  Mounts BlockNote with our custom block definitions       │  │
│  │  Adapter layer translates MDDM ↔ BlockNote at boundaries  │  │
│  │  Editor only sees BlockNote-shaped JSON internally        │  │
│  └───────────────────────────────────────────────────────────┘  │
│                            │                                    │
│                            │ POST /documents/:id (draft saves)  │
│                            │ POST /documents/:id/release        │
│                            ▼                                    │
└─────────────────────────────────────────────────────────────────┘
                             │
┌─────────────────────────────────────────────────────────────────┐
│                    BACKEND (Go)                                 │
│                                                                 │
│  ┌───────────────────────────────────────────────────────────┐  │
│  │  Document Service                                         │  │
│  │  ─────────────────────────────────────────────────────    │  │
│  │  • normalizeMDDM (defensive)                              │  │
│  │  • Layer 1 validation (JSON Schema)                       │  │
│  │  • Load template via template_ref + hash verify           │  │
│  │  • Layer 2 validation (locks, grammar, business rules)    │  │
│  │  • Atomic transactions for save / release / archive       │  │
│  └───────────────────────────────────────────────────────────┘  │
│                                                                 │
│  ┌───────────────────────────────────────────────────────────┐  │
│  │  Image Service                                            │  │
│  │  ─────────────────────────────────────────────────────    │  │
│  │  • ImageStorage interface (pluggable backend)             │  │
│  │  • v1: PostgresByteaStorage                               │  │
│  │  • Per-image authorization via document references        │  │
│  │  • MIME sniffing from bytes, content-addressed dedup      │  │
│  └───────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────┘
                             │
                             │ POST /render/docx (only at release)
                             ▼
┌─────────────────────────────────────────────────────────────────┐
│                  DOCGEN SERVICE (Node.js)                       │
│                                                                 │
│  ┌───────────────────────────────────────────────────────────┐  │
│  │  MDDM → DOCX Compiler                                     │  │
│  │  ─────────────────────────────────────────────────────    │  │
│  │  Walks MDDM block tree, emits Word native elements via    │  │
│  │  the `docx` npm library (v9.6.1). Each block type has     │  │
│  │  an explicit mapping to DOCX primitives (Heading styles,  │  │
│  │  Tables, Numbering, Paragraphs, Runs, Hyperlinks).        │  │
│  │  Output is bytes streamed back to backend.                │  │
│  └───────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────┘
```

### Architectural commitments

1. **JSON is the only canonical format.** No HTML in the database. No mixed representations.
2. **MDDM is MetalDocs-owned.** BlockNote is an editor, not a contract. We define every block type, every prop, every rule.
3. **Templates and documents share the same model.** A template is just a document with `placeholder` content and `locked` blocks. Same editor edits both.
4. **The release event is a first-class concept.** Drafts are working copies. Released versions are frozen artifacts. Archives are immutable history.
5. **Released DOCX is rendered ONCE, at release time, and stored forever.** Never re-rendered. The frozen artifact is the official record.
6. **Server-side validation is non-negotiable.** Every save is validated structurally (JSON Schema) AND semantically (business rules) before persistence.
7. **Single-transaction atomicity for all multi-step state changes.** Saves, releases, archival, image reconciliation — all in one DB transaction.
8. **No filesystem.** Images live in PostgreSQL via a pluggable interface. Future S3 migration is a backend swap, not a model change.

## 3. Domain model and lifecycle

### Document states

A document version has exactly one of four states:

| State | Editable | docx_bytes | content_blocks | Visible to viewers |
|---|---|---|---|---|
| `draft` | ✅ in place | ❌ NULL | ✅ present | ❌ no (only author + admins) |
| `pending_approval` | ❌ frozen | ❌ NULL | ✅ present | ❌ no |
| `released` | ❌ no (clone to draft new revision) | ✅ frozen at release time | ✅ present (for cloning) | ✅ yes |
| `archived` | ❌ no | ✅ kept | ❌ NULL (discarded) | ✅ yes (read-only) |

### Cardinality invariants (DB-enforced)

```sql
-- At most ONE released version per document
CREATE UNIQUE INDEX idx_one_released_per_doc
  ON document_versions(document_id)
  WHERE status = 'released';

-- At most ONE active draft (draft or pending_approval) per document
CREATE UNIQUE INDEX idx_one_active_draft_per_doc
  ON document_versions(document_id)
  WHERE status IN ('draft', 'pending_approval');
```

These partial unique indexes make it **impossible** at the database level to violate cardinality. Concurrent attempts get unique constraint violations; the loser retries.

### Lifecycle walkthrough

```
1. User creates document from template
   → DRAFT version 1 created (cloned from template)
   → status: draft, content_blocks: present, docx_bytes: NULL

2. User edits in BlockNote, saves multiple times
   → Each save: in-place UPDATE of the draft row
   → No new version rows for draft saves
   → Image references reconciled in same transaction

3. User submits for approval
   → status: draft → pending_approval (immutable)
   → Approver notified

4. Approver approves (the "release" event)
   → ATOMIC TRANSACTION:
     a. Render DOCX from current draft MDDM
     b. Find previous released version (if any)
     c. Archive the previous: status → archived, content_blocks → NULL
        (its previously-stored docx_bytes stays)
        Delete its document_version_images entries
     d. Promote the draft: status → released, write docx_bytes
     e. Compute revision_diff (canonical previous → canonical new)
     f. Cascade orphan image cleanup
   → New version is now THE released version, visible to all users

5. User starts a new revision (REV02)
   → New DRAFT created from current released MDDM
   → status: draft, NEW id, but same template_ref (same template snapshot)

6. Loop from step 2
```

### Storage profile per document over its lifetime

For a PO with 5 revisions over 2 years:

| Item | Count | Approx size |
|---|---|---|
| Latest released MDDM JSON | 1 | ~50 KB |
| Optional draft MDDM JSON | 0–1 | ~50 KB |
| Released DOCX (current) | 1 | ~50 KB |
| Archived DOCX (historical) | 4 | 4 × ~50 KB = 200 KB |
| Images (deduplicated, in document_images) | ~5 unique | ~2.5 MB |
| Revision diffs | 4 | ~10 KB |
| **TOTAL** | | **~3 MB** |

The MDDM JSON storage **does not accumulate** over time. It is bounded to at most 2 rows per document.

## 4. Block schema

MDDM has 17 block types in two categories.

### 4.1 Structural blocks (template skeleton, may have `template_block_id`)

These define the locked structure of a document. They are governed by the template and enforce the lock model.

#### `Section`

```ts
{
  id: string,                    // UUID v4
  template_block_id?: string,
  type: "section",
  props: {
    number: integer,             // computed at render, NOT stored
    title: string,               // 1-200 chars
    color: string,               // hex /^#[0-9a-fA-F]{6}$/
    locked: boolean              // default false
  },
  children: Block[]
}
```

**Allowed children**: `FieldGroup`, `Field`, `RichBlock`, `Repeatable`, `DataTable`, `Paragraph`, `Heading`, `BulletListItem`, `NumberedListItem`, `Image`, `Quote`, `Code`, `Divider`

**DOCX rendering**: Word `Heading 1` style with auto-numbered prefix. Optional colored border or background derived from `color`.

#### `FieldGroup`

```ts
{
  id, template_block_id?,
  type: "fieldGroup",
  props: {
    columns: 1 | 2,              // default 1
    locked: boolean              // default false
  },
  children: Field[]
}
```

**Allowed children**: `Field` only.

**DOCX rendering**: Real Word table. If `columns: 2`, renders as a 4-column table (label / value / label / value) so two field pairs per row.

#### `Field`

```ts
{
  id, template_block_id?,
  type: "field",
  props: {
    label: string,                                   // 1-100 chars
    valueMode: "inline" | "multiParagraph",          // default "inline"
    locked: boolean                                  // default false
  },
  children: InlineContent | Block[]                  // depends on valueMode
}
```

- `valueMode: "inline"` → children must be `InlineContent` only (single line of formatted text)
- `valueMode: "multiParagraph"` → children may be `Paragraph`, `BulletListItem`, `NumberedListItem`, `Quote`, `Divider` (no images, no sub-tables)

**DOCX rendering**: Inside a FieldGroup, renders as a labeled table cell. Standalone (rare), renders as a labeled paragraph block.

#### `Repeatable`

```ts
{
  id, template_block_id?,
  type: "repeatable",
  props: {
    label: string,               // 1-100 chars
    itemPrefix: string,          // 1-30 chars, e.g. "Etapa"
    locked: boolean,             // default false
    minItems: integer,           // >= 0, default 0
    maxItems: integer            // >= 1, default 100
  },
  children: RepeatableItem[]
}
```

**Allowed children**: `RepeatableItem` only. The number of children must satisfy `minItems <= count <= maxItems`.

**DOCX rendering**: Each item becomes an auto-numbered Word `Heading 2` (e.g., "5.1 Receber pedido", "5.2 Validar dados") followed by its body content.

#### `DataTable`

```ts
{
  id, template_block_id?,
  type: "dataTable",
  props: {
    label: string,               // 1-100 chars
    columns: ColumnDef[],        // 1-20 columns
    locked: boolean,
    minRows: integer,            // >= 0, default 0
    maxRows: integer             // >= 1, default 500
  },
  children: DataTableRow[]
}

ColumnDef = {
  key: string,                   // lowercase /^[a-z][a-z0-9_]*$/
  label: string,                 // 1-50 chars
  type: "text" | "number" | "date",  // default "text"
  required: boolean              // default false
}
```

**Allowed children**: `DataTableRow` only. Number of rows must satisfy `minRows <= count <= maxRows`.

**DOCX rendering**: Real Word table with header row from `columns[].label`, data rows from children.

#### `RichBlock`

```ts
{
  id, template_block_id?,
  type: "richBlock",
  props: {
    label: string,               // 1-100 chars
    locked: boolean
  },
  children: Block[]              // free zone, user content
}
```

**Allowed children**: `Paragraph`, `Heading`, `BulletListItem`, `NumberedListItem`, `Image`, `Quote`, `Code`, `Divider`.

**DOCX rendering**: Label as bold paragraph, then the children rendered as their natural Word equivalents.

### 4.2 Content blocks (user-fillable, no `template_block_id`)

These are what users add inside content slots (Field children, RepeatableItem children, RichBlock children, etc.).

#### `RepeatableItem`

```ts
{
  id,
  type: "repeatableItem",
  props: {
    title: string                // 0-200 chars
  },
  children: Block[]
}
```

**Allowed children**: `Paragraph`, `Heading`, `BulletListItem`, `NumberedListItem`, `Image`, `Quote`, `Code`, `Divider`, `RichBlock` (no nesting of `Section`/`FieldGroup`/`Field`/`Repeatable`/`DataTable`).

**DOCX rendering**: Title as auto-numbered Word `Heading 2`, body as the children's natural Word output.

#### `DataTableRow`

```ts
{
  id,
  type: "dataTableRow",
  props: {},
  children: DataTableCell[]
}
```

**Allowed children**: `DataTableCell` only, exactly one per parent DataTable column, in column order.

#### `DataTableCell`

```ts
{
  id,
  type: "dataTableCell",
  props: {
    columnKey: string            // matches a parent DataTable column key
  },
  children: InlineContent
}
```

**Allowed children**: `InlineContent` only.

#### `Paragraph`

```ts
{
  id,
  type: "paragraph",
  props: {},
  children: InlineContent
}
```

**DOCX rendering**: Word paragraph with `Normal` style.

#### `Heading`

```ts
{
  id,
  type: "heading",
  props: {
    level: 1 | 2 | 3
  },
  children: InlineContent
}
```

**DOCX rendering**: Word `Heading 1`, `Heading 2`, or `Heading 3` style. Note: top-level Section headers use Word Heading 1 by default; this block lets users add nested sub-headings inside free zones.

#### `BulletListItem` / `NumberedListItem`

```ts
{
  id,
  type: "bulletListItem" | "numberedListItem",
  props: {
    level: integer               // 0-6, controls nesting depth
  },
  children: InlineContent
}
```

**Nesting semantics**: consecutive list items at the same level form a list visually. Items at deeper levels nest under the previous shallower item. Invalid level jumps (e.g., 0 → 3) are clamped on canonicalization to `min(prev_level + 1, 6)`.

**DOCX rendering**: Mapped to `docx` library's numbering primitives with proper bullet/numbered nesting.

#### `Image`

```ts
{
  id,
  type: "image",
  props: {
    src: string,                 // /^\/api\/images\/[a-f0-9-]{36}$/
    alt: string,                 // 0-500 chars
    caption: string              // 0-500 chars
  }
}
```

Leaf block (no children). The `src` references an image stored via the `ImageStorage` interface (PostgreSQL bytea in v1).

**DOCX rendering**: Embedded image with the bytes pulled from `document_images.bytes` at render time. Alt text and caption become Word's built-in image description and caption.

#### `Quote`

```ts
{
  id,
  type: "quote",
  props: {},
  children: Paragraph[]
}
```

**Allowed children**: `Paragraph` only.

**DOCX rendering**: Word `Quote` style.

#### `Code`

```ts
{
  id,
  type: "code",
  props: {
    language: string             // 0-30 chars, e.g., "python", ""
  },
  children: { type: "text", text: string }[]
}
```

**Allowed children**: text-only objects, no marks, no links, byte-exact preserved (no whitespace touching, no NFC).

**DOCX rendering**: Monospace paragraph with light gray background.

#### `Divider`

```ts
{
  id,
  type: "divider",
  props: {}
}
```

Leaf block (no children).

**DOCX rendering**: Horizontal rule.

### 4.3 Inline content (frozen schema)

```ts
type InlineContent = TextRun[]

type TextRun = {
  text: string,
  marks?: Mark[],
  link?: {
    href: string,
    title?: string
  },
  document_ref?: {
    target_document_id: string,
    target_revision_label?: string
  }
}

type Mark = {
  type: "bold" | "italic" | "underline" | "strike" | "code"
}
```

Used by: `Paragraph` children, `Heading` children, `Field` children (when `valueMode=inline`), `DataTableCell` children, `BulletListItem`/`NumberedListItem` children.

### 4.4 Cross-document references

`TextRun.document_ref` is the inline mention mechanism for "see PO-117" type references.

**Validation at save time**:
- `target_document_id` must exist in the `documents` table
- The user submitting the save must have read access to the target document
- If validation fails: `CROSS_DOC_REF_NOT_FOUND` (400) or `CROSS_DOC_REF_FORBIDDEN` (403)

**Editor rendering**: chip/badge showing auto-fetched title + revision label (e.g., "PO-117 — Atendimento (REV03)").

**DOCX rendering**: hyperlink with text "[PO-117 — Atendimento (REV03)]" pointing to the appropriate URL.

**Broken refs at render time** (target document deleted after this document was saved): render as italic placeholder "[Documento removido: PO-117]" + log a warning. The save is not rejected (the reference was valid when saved); the stale reference is degraded gracefully at render.

### 4.5 Auto-numbering

Numbers like Section "5" or RepeatableItem "5.1" are **never stored**. They are computed from sibling order at render time, both in the editor and in the DOCX exporter.

Renaming a Section automatically renumbers all its descendants. Adding a new Section in the middle shifts all subsequent numbers. This is the only correct way to handle hierarchical numbering — storing computed values would create drift.

### 4.6 Block grammar (parent → allowed children)

Encoded into JSON Schema via type-aware `oneOf`/`if`/`then` constraints, with Layer 2 (Go business rules) defense-in-depth.

| Parent | Allowed children |
|---|---|
| Document root | `Section`, `Paragraph`, `Heading`, `Image`, `Divider` |
| `Section` | `FieldGroup`, `Field`, `RichBlock`, `Repeatable`, `DataTable`, `Paragraph`, `Heading`, `BulletListItem`, `NumberedListItem`, `Image`, `Quote`, `Code`, `Divider` |
| `FieldGroup` | `Field` only |
| `Field` (inline) | `InlineContent` only |
| `Field` (multiParagraph) | `Paragraph`, `BulletListItem`, `NumberedListItem`, `Quote`, `Divider` |
| `Repeatable` | `RepeatableItem` only |
| `RepeatableItem` | `Paragraph`, `Heading`, `BulletListItem`, `NumberedListItem`, `Image`, `Quote`, `Code`, `Divider`, `RichBlock` |
| `DataTable` | `DataTableRow` only |
| `DataTableRow` | `DataTableCell` only (one per column, in column order) |
| `DataTableCell` | `InlineContent` only |
| `RichBlock` | `Paragraph`, `Heading`, `BulletListItem`, `NumberedListItem`, `Image`, `Quote`, `Code`, `Divider` |
| `Paragraph` | `InlineContent` only |
| `Heading` | `InlineContent` only |
| `BulletListItem` | `InlineContent` only |
| `NumberedListItem` | `InlineContent` only |
| `Quote` | `Paragraph` only |
| `Code` | text-only objects (no marks, no links) |
| `Image` | leaf, no children |
| `Divider` | leaf, no children |

## 5. Identity model

Every block has two identifiers:

### `id: string` (always present)

- UUID v4
- Document-local identity
- Generated by MetalDocs (NOT by BlockNote)
- Immutable for the lifetime of the block within a document version
- Regenerated on template instantiation (a cloned block has a new `id`)
- Used for: diff engine targeting, validation error reporting, editor state tracking, locked-block matching within a version

### `template_block_id?: string` (optional)

- UUID v4
- Present **only** on STRUCTURAL blocks instantiated from a template
- Absent on user-added content blocks
- Immutable across the document's lifetime (preserved through saves, releases, migrations)
- Used for: locked-block enforcement (matches template blocks to document blocks), template snapshot binding

### Server-side ID continuity enforcement

On every save, the backend compares the incoming Block tree against the previous version:

1. For every block in the new submission with `template_block_id` X:
   - The previous version MUST have a block with the same `template_block_id` X
   - That previous block's `id` MUST equal the new block's `id`
   - If different → reject with `BLOCK_ID_REWRITE_FORBIDDEN` (HTTP 422)
2. For every templated block that existed in the previous version:
   - It MUST exist in the new version (with the same `template_block_id`)
   - If missing → reject with `LOCKED_BLOCK_DELETED` (HTTP 422)

User-added blocks (no `template_block_id`) get best-effort continuity in v1: stable when the adapter behaves correctly, but the server does not enforce. Diffs for user-authored content are documented as "best-effort move detection". v2 may add stricter enforcement via parent+position heuristics.

## 6. Canonicalization contract

The `normalizeMDDM(envelope)` function produces a deterministic canonical form. It runs at every boundary:

- Frontend before POST
- Backend before validation, persistence, diff computation, lock check, hash computation
- Exporter input
- Image GC scan input

### Canonical form rules (precise)

| Rule | Applies to | Notes |
|---|---|---|
| Property keys ordered alphabetically | All JSON objects | Deterministic serialization |
| All defaults stored explicitly | Every block prop | NO elision; if `locked: false` is the default, it's still stored |
| Empty arrays/objects stored explicitly | All collections | NO elision; if `children: []` is the default, it's still stored |
| Inline content marks sorted alphabetically by name | `TextRun.marks` | Deterministic mark order |
| Adjacent runs with identical marks merged | `InlineContent` arrays | `[{text: "foo", marks: [bold]}, {text: "bar", marks: [bold]}]` → `[{text: "foobar", marks: [bold]}]` |
| Whitespace normalization | JSON formatting + inline-content run boundaries | NEVER touches user text inside runs |
| Code block content preserved byte-exact | `Code` block children | NO whitespace touching, NO merging, NO NFC |
| URL canonicalization | `Image.src` only | Strip query strings, fragments, trailing slashes from internal `/api/images/` URLs |
| User-authored link hrefs preserved | `TextRun.link.href` | NEVER canonicalized (could break signed URLs) |
| Strings NFC unicode normalized | All strings EXCEPT Code block text | Standard NFC normalization |
| Integers stored as integers | Number-typed props | No `1.0` floats |
| List item levels clamped | `BulletListItem.level`, `NumberedListItem.level` | Invalid jumps (0 → 3) clamped to `min(prev_level + 1, 6)` |

### Defaults policy

All defaults are part of the **versioned MDDM contract**. Changing a default requires:

1. Bumping `mddm_version` (e.g., 1 → 2)
2. Writing a forward migration that materializes the old default explicitly into existing documents

This guarantees document immutability across MDDM evolution. Old documents will never silently behave differently because a default changed.

### Implementation

- TypeScript: `shared/schemas/canonicalize.ts`
- Go: `internal/modules/documents/domain/mddm/canonicalize.go`
- Test fixture suite verifies **byte-identical output** across both implementations

The written canonical form spec lives at `shared/schemas/mddm-canonical-form.md` with field-by-field rules.

## 7. Validation

### Layer 1 — JSON Schema (structural)

- Single source of truth: `shared/schemas/mddm.schema.json` (JSON Schema 2020-12)
- TypeScript: AJV at runtime, `json-schema-to-typescript` for build-time types
- Go: `github.com/santhosh-tekuri/jsonschema/v6` at runtime
- Catches: wrong block type, missing required fields, invalid prop types, wrong children for parent, ID format violations, image src format violations

### Layer 2 — Go business rules (semantic)

Implemented in `internal/modules/documents/domain/mddm/rules.go`:

1. **Locked-block immutability** via `template_block_id` matching after hash verification
2. **Position checks** on locked structural blocks (same parent + same sibling order among templated siblings)
3. **`minItems`/`maxItems`** on `Repeatable` and `DataTable`
4. **DataTable cell-column consistency** (every row has exactly one cell per column, in order)
5. **Image existence + auth** (every Image src must point to an existing `document_images` row that the user can read)
6. **Cross-document reference target existence + auth** (every `TextRun.document_ref` must point to an existing document the user can read)
7. **Template_block_id uniqueness** within a document
8. **ID uniqueness** within a document
9. **Size limits** (see below)
10. **Parent → children grammar** (defense-in-depth, also enforced in Layer 1)
11. **Block ID continuity** across saves (`BLOCK_ID_REWRITE_FORBIDDEN`, `LOCKED_BLOCK_DELETED`)

### Both layers run on every save server-side

The frontend also runs Layer 1 for inline UX feedback, but **the server is the only authority**.

### Size limits (server-enforced)

| Limit | Value |
|---|---|
| `max_blocks_per_document` | 5000 |
| `max_nesting_depth` | 20 |
| `max_children_per_block` | 1000 |
| `max_data_table_rows` | 500 |
| `max_repeatable_items` | 200 |
| `max_payload_bytes` | 5 MB |
| `max_inline_text_length` | 10000 chars per run |
| `max_image_size` | 10 MB |

## 8. Storage layout

### Tables

```sql
-- Documents (logical entities)
CREATE TABLE documents (
  id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  code        TEXT NOT NULL UNIQUE,            -- e.g., "PO-118"
  title       TEXT NOT NULL,
  profile     TEXT NOT NULL,                   -- e.g., "po"
  created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
  created_by  UUID NOT NULL REFERENCES users(id)
);

-- Document versions (one per draft + released + archived)
CREATE TABLE document_versions (
  id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  document_id     UUID NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
  version_number  INTEGER NOT NULL,            -- 1, 2, 3, ...
  revision_label  TEXT NOT NULL,               -- "REV01", "REV02", ...
  status          TEXT NOT NULL CHECK (status IN ('draft','pending_approval','released','archived')),
  content_blocks  JSONB,                       -- present when status != 'archived'
  docx_bytes      BYTEA,                       -- present when status IN ('released','archived')
  template_ref    JSONB,                       -- snapshot ref to template
  content_hash    TEXT,                        -- sha256 of canonical content_blocks
  revision_diff   JSONB,                       -- JSON Patch from previous released version (set at release time)
  change_summary  TEXT,                        -- user-provided "what changed" at release time
  created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
  created_by      UUID NOT NULL REFERENCES users(id),
  approved_at     TIMESTAMPTZ,
  approved_by     UUID REFERENCES users(id),
  UNIQUE (document_id, version_number)
);

-- Cardinality enforcement
CREATE UNIQUE INDEX idx_one_released_per_doc
  ON document_versions(document_id)
  WHERE status = 'released';

CREATE UNIQUE INDEX idx_one_active_draft_per_doc
  ON document_versions(document_id)
  WHERE status IN ('draft', 'pending_approval');

-- Image storage (deduplicated by content hash)
CREATE TABLE document_images (
  id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  sha256      TEXT NOT NULL UNIQUE,
  mime_type   TEXT NOT NULL,
  byte_size   INTEGER NOT NULL,
  bytes       BYTEA NOT NULL,
  created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- M:N references for image lifecycle
CREATE TABLE document_version_images (
  document_version_id UUID NOT NULL REFERENCES document_versions(id) ON DELETE CASCADE,
  image_id            UUID NOT NULL REFERENCES document_images(id),
  PRIMARY KEY (document_version_id, image_id)
);
CREATE INDEX idx_dvi_image ON document_version_images(image_id);

-- Templates (independently versioned)
CREATE TABLE document_template_versions (
  id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  template_id     UUID NOT NULL,
  version         INTEGER NOT NULL,
  mddm_version    INTEGER NOT NULL,
  content_blocks  JSONB NOT NULL,
  content_hash    TEXT NOT NULL,
  is_published    BOOLEAN NOT NULL DEFAULT false,
  created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (template_id, version)
);

-- DB trigger preventing UPDATE of published template content
CREATE OR REPLACE FUNCTION prevent_published_template_mutation()
RETURNS TRIGGER AS $$
BEGIN
  IF OLD.is_published = true AND NEW.content_blocks IS DISTINCT FROM OLD.content_blocks THEN
    RAISE EXCEPTION 'Cannot modify content_blocks of a published template version';
  END IF;
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_template_immutable
  BEFORE UPDATE ON document_template_versions
  FOR EACH ROW
  EXECUTE FUNCTION prevent_published_template_mutation();
```

## 9. Templates

### Template versioning

- Templates have their own version history (`document_template_versions`)
- Each version has its own `mddm_version` (templates evolve through MDDM versions like documents do)
- Once `is_published = true`, the row is **immutable** (DB trigger enforces this)
- Editing a template creates a NEW `template_version` row, never modifies a published one
- Foreign key from `document_versions.template_ref → document_template_versions(id, version)` prevents deletion of referenced template versions

### Template snapshot binding

Every document version stores:

```json
{
  "template_ref": {
    "template_id": "uuid",
    "template_version": 7,
    "template_mddm_version": 1,
    "template_content_hash": "sha256-hex-string"
  }
}
```

This binds the document FOREVER to the exact template snapshot it was created from. Editing a template creates a new template_version; old documents remain governed by their original snapshot.

### Hash verification

Every template resolution (lock check, export, etc.) runs:

```go
template := loadTemplateVersion(ref.template_id, ref.template_version)
canonical := normalizeMDDM(template.content_blocks)  // canonicalize at template_mddm_version
actualHash := sha256(canonical)
if actualHash != ref.template_content_hash {
    return ErrTemplateSnapshotMismatch  // HTTP 422
}
// THEN migrate template forward to current MDDM version for use
migrated := migrateForward(canonical, template.mddm_version, currentMDDMVersion)
useTemplate(migrated)
```

**Critical ordering**: hash verification happens BEFORE migration. The hash is permanently bound to the template's original canonical bytes at its original mddm_version. Migration runs only after the hash check passes.

### Error codes

- `TEMPLATE_SNAPSHOT_MISMATCH` (HTTP 422): hash mismatch detected. Document is unloadable until snapshot is restored or admin rebinds.
- `TEMPLATE_SNAPSHOT_MISSING` (HTTP 422): template version not found in DB. Same handling.

Both are hard domain errors. Never warnings, never fall back.

## 10. Image storage

### Pluggable interface

```go
type ImageStorage interface {
    // Put stores image bytes idempotently. Returns the image ID. If an image
    // with the same sha256 already exists, returns the existing ID.
    Put(ctx context.Context, sha256 string, mimeType string, bytes []byte) (uuid.UUID, error)

    // Get retrieves image bytes by ID. Returns ErrImageNotFound if missing.
    Get(ctx context.Context, id uuid.UUID) (bytes []byte, mimeType string, err error)

    // Delete removes an image by ID. Used by orphan cleanup.
    Delete(ctx context.Context, id uuid.UUID) error

    // Exists checks if an image with this sha256 already exists.
    Exists(ctx context.Context, sha256 string) (id uuid.UUID, exists bool, err error)
}
```

### v1 implementation: PostgresByteaStorage

Uses the `document_images` table directly. Simple, ACID, single backup.

### Future: S3 backend (Phase 2 or later)

Implements the same interface against S3-compatible object storage. Configured via:

```bash
MDDM_IMAGE_STORAGE=postgres_bytea  # default for v1
MDDM_IMAGE_STORAGE=s3              # future
```

The MDDM JSON never knows which backend is in use. The image URL stays as `/api/images/{uuid}`.

### Upload flow

1. User pastes/drops an image in BlockNote editor
2. Frontend POSTs the file to `/api/uploads/images` with `document_id` for auth context
3. Backend reads bytes (max 10 MB cap)
4. **MIME type detected from bytes** via `http.DetectContentType` (NOT trusted from filename)
5. Whitelist check: `image/png`, `image/jpeg`, `image/webp`, `image/gif`. Reject others with HTTP 415.
6. Compute SHA-256 of bytes
7. Call `storage.Put(sha256, mimeType, bytes)` → returns image_id (deduplicated)
8. Return `{ image_id, mime_type }` to frontend
9. Frontend inserts an Image block with `src: "/api/images/{image_id}"`

### Read flow

1. GET `/api/images/{id}` arrives
2. Backend checks: does the user have read access to ANY document_version that references this image_id (via `document_version_images` join)?
3. If yes: load bytes from storage backend, stream with `Content-Type: {mime_type}`, `Cache-Control: private, max-age=3600`, `ETag: "{id}"`
4. If no: return 403

### Reference reconciliation on every draft save

```sql
BEGIN;
  -- Update draft content
  UPDATE document_versions
  SET content_blocks = $1, content_hash = $2, ...
  WHERE id = $version_id AND status = 'draft';

  -- Compute new image set from canonical MDDM (extracted in Go before this transaction)
  -- Reconcile join table:
  DELETE FROM document_version_images
  WHERE document_version_id = $version_id
    AND image_id != ALL($new_image_set::uuid[]);

  INSERT INTO document_version_images (document_version_id, image_id)
  SELECT $version_id, image_id
  FROM unnest($new_image_set::uuid[]) AS image_id
  ON CONFLICT DO NOTHING;
COMMIT;
```

### Orphan cleanup at release transition

Runs in the same transaction as the release:

```sql
DELETE FROM document_images
WHERE NOT EXISTS (
  SELECT 1 FROM document_version_images WHERE image_id = document_images.id
);
```

Any `document_images` row with no remaining references gets deleted. The cascade is automatic, no race conditions, no grace period needed (the entire release is one transaction).

## 11. Adapter layer

The adapter translates between MDDM (canonical) and BlockNote (editor working format). It is a thin layer (~200 lines for v1) with explicit per-block-type translation.

```typescript
// shared/adapters/blocknote-adapter.ts

export function mddmToBlockNote(envelope: MDDMEnvelope): BlockNoteBlock[] {
  return envelope.blocks.map(blockToBlockNote);
}

export function blockNoteToMDDM(blocks: BlockNoteBlock[]): MDDMEnvelope {
  return {
    mddm_version: 1,
    blocks: blocks.map(blockNoteToBlock),
    template_ref: getCurrentTemplateRef()
  };
}
```

Each block type has an explicit translator function. For v1, most are near-identity (BlockNote's JSON shape is similar to MDDM), but the seam exists for future-proofing.

**Round-trip invariant**: `normalizeMDDM(blockNoteToMDDM(mddmToBlockNote(envelope))) === normalizeMDDM(envelope)` for any valid envelope. Tested via golden fixtures.

## 12. Diff engine

```typescript
type Diff = {
  added: { id: string, type: string, parentId: string | null }[],
  removed: { id: string }[],
  moved: { id: string, oldParentId: string | null, newParentId: string | null }[],
  modified: { id: string, propChanges: PropChange[] }[]
}
```

- Operates **only on canonicalized envelopes**
- Computed at release time (not at every save), comparing the new released version against the previous released version
- ~100 lines of TypeScript or Go (one implementation, server-side)
- Output stored as JSON in `document_versions.revision_diff`

**Why at release time only**: drafts have no audit value (they're working copies). The audit-relevant moments are release transitions. Computing diffs only at releases is far more efficient and matches the user-meaningful events.

## 13. Migration framework

```go
type EnvelopeMigration func(envelope MDDMEnvelopeVN) (MDDMEnvelopeVN1, error)

var migrations = map[int]EnvelopeMigration{
    1: migrateV1toV2,  // when v2 exists
    2: migrateV2toV3,  // when v3 exists
    // ...
}

func migrateForward(envelope MDDMEnvelope, fromVersion, toVersion int) (MDDMEnvelope, error) {
    for v := fromVersion; v < toVersion; v++ {
        migration, ok := migrations[v]
        if !ok {
            return envelope, fmt.Errorf("missing migration from v%d", v)
        }
        next, err := migration(envelope)
        if err != nil {
            return envelope, err
        }
        envelope = next
    }
    return envelope, nil
}
```

- **Forward-only**: migrations go v1→v2→v3, never backwards
- **Pure functions**: no side effects, no I/O, deterministic
- **Operate on the full envelope**, not just blocks (so they can update `template_ref`, add new envelope-level fields, etc.)
- **Tested with golden fixtures**: every migration has `(input, expected_output)` test fixtures + a property test asserting purity

## 14. Data flows

### 14.1 Load document for editing

```
1. GET /api/documents/:id (with Accept: application/json)
2. Backend: SELECT * FROM document_versions WHERE document_id = ? AND status IN ('draft', 'released') ORDER BY ...
   - If draft exists for this user: return draft
   - Else: return latest released
3. Parse JSONB → MDDMEnvelope { mddm_version, blocks, template_ref }
4. Check mddm_version, run forward migrations if needed (envelope-level)
5. Run normalizeMDDM (defensive)
6. Send envelope to frontend
7. Frontend: mddmToBlockNote(envelope) → mount BlockNote editor
```

### 14.2 Draft save

```
1. User edits in BlockNote, presses Ctrl+S
2. Frontend: blocks = editor.document
   envelope = blockNoteToMDDM(blocks)
   envelope = normalizeMDDM(envelope)
   POST /api/documents/:id/draft with body { content: envelope, base_version: 4 }
3. Backend:
   a. Receive envelope
   b. normalizeMDDM (defensive)
   c. Layer 1 validation (JSON Schema) — reject 400 with structured issues if fail
   d. Load template via envelope.template_ref
   e. Verify template_content_hash → reject 422 if mismatch
   f. Migrate template forward to current mddm_version
   g. Layer 2 validation (locks, grammar, business rules) — reject 422 if fail
   h. Compare against previous draft state for ID continuity
4. If validation passes: ATOMIC TRANSACTION:
   a. UPDATE document_versions SET content_blocks = $1, content_hash = $2 WHERE id = $draft_id AND status = 'draft'
   b. Compute image set from canonical MDDM
   c. DELETE document_version_images WHERE document_version_id = $draft_id AND image_id NOT IN $new_set
   d. INSERT document_version_images for new image_ids ON CONFLICT DO NOTHING
   e. Verify all referenced images exist (defense)
   COMMIT
5. Return 200 with new content_hash
```

### 14.3 Release approval (the critical transaction)

```
1. POST /api/documents/:id/release (admin/approver only)
2. Backend authorization check
3. Load current draft (status='pending_approval')
4. Render DOCX from draft MDDM via docgen service
   - If render fails: return 500, draft stays in pending_approval
5. ATOMIC TRANSACTION (critical sequence to satisfy partial unique indexes):
   a. UPDATE prev_released SET status = 'archived', content_blocks = NULL WHERE document_id = $doc AND status = 'released' (if exists)
      ↑ Archives the previous released version FIRST so the partial unique index allows the new release
   b. UPDATE document_versions SET status = 'released', docx_bytes = $rendered_bytes, approved_at = now(), approved_by = $user WHERE id = $draft_id
   c. Compute revision_diff against previous released version's canonical content
   d. UPDATE document_versions SET revision_diff = $diff WHERE id = $draft_id
   e. DELETE FROM document_version_images WHERE document_version_id = $prev_released_id (only the just-archived version)
   f. DELETE FROM document_images WHERE NOT EXISTS (SELECT 1 FROM document_version_images WHERE image_id = document_images.id) — orphan cleanup
   COMMIT
6. Return 200 with the new released version metadata
```

**Why archive-then-promote ordering**: the `idx_one_released_per_doc` partial unique index allows at most one row with `status = 'released'` per document. We must archive the old one BEFORE promoting the new one, otherwise the INSERT/UPDATE would violate the constraint.

### 14.4 Export DOCX

```
1. GET /api/documents/:id/export/docx?version_id=X (default: latest released)
2. Backend authorization check
3. Load the specified document_version
4. Branch on status:
   a. status = 'released' OR status = 'archived':
      → ALWAYS serve docx_bytes directly (NEVER re-render)
      → Set Content-Type: application/vnd.openxmlformats-officedocument.wordprocessingml.document
      → Stream bytes
   b. status = 'draft' OR status = 'pending_approval' (admin/owner only):
      → Render fresh DOCX from MDDM via docgen (debug mode allowed)
      → Stream bytes
5. PDF rendering: GET /api/documents/:id/export/pdf?version_id=X
   → Get the DOCX (from above flow)
   → LibreOffice converts DOCX → PDF on demand (not stored)
   → Stream PDF bytes
```

**Repair/backfill mode** (admin-only, separate endpoint):

```
POST /api/admin/documents/:id/versions/:version_id/rerender
```

This re-renders DOCX for a released or archived version using the current renderer. It's logged as a special audit event with the reason. **Never used in normal flow**. Reserved for cases like "renderer bug fix needs to update historical artifacts" (extremely rare).

### 14.5 Create document from template

```
1. POST /api/documents { template_id, title, profile, ... }
2. Backend:
   a. Load latest published template_version
   b. Verify hash (consistency check at creation time)
   c. Migrate template forward to current mddm_version
   d. instantiate(template):
      - Walk template tree
      - For each STRUCTURAL block: assign new id, copy template's id to template_block_id
      - For each block INSIDE a content slot (Field children, Repeatable children, etc.): assign new id, NO template_block_id, treated as user-owned
      - Snapshot template_ref onto the new envelope: { template_id, template_version, template_mddm_version, template_content_hash }
   e. normalizeMDDM
   f. INSERT documents row (allocate code: "PO-121", etc.)
   g. INSERT document_versions row with status='draft', content_blocks=instantiated, version_number=1, revision_label='REV01'
   h. Insert document_version_images entries for any images in the template (rare; templates usually don't have images)
3. Return 201 with the new document id and code
4. Frontend redirects to /documents/:id (which loads in editor as in 14.1)
```

## 15. Error handling

### Validation errors

400 with structured response:

```json
{
  "error": "validation_failed",
  "issues": [
    {
      "blockId": "abc-123-def",
      "path": "/blocks/3/children/2/props/title",
      "code": "minLength",
      "message": "Title must not be empty",
      "limit": 1,
      "actual": 0
    }
  ]
}
```

Frontend uses `blockId` to scroll-to and visually mark the offending block.

### Network errors

- Connection lost: non-blocking banner "Conexão perdida. Salvando localmente." Editor maintains in-memory dirty state. Auto-retry on reconnect.
- 5xx: exponential backoff retry (1s, 3s, 9s). After 3 failures, error toast with manual retry.
- Auth expired (401): redirect to login, preserve URL + dirty state in sessionStorage.

### Concurrency

- Optimistic locking via `If-Match: "version-N"` header on draft updates
- Mismatch → 409 with `current_version` + `your_base_version` + structured prompt
- Cardinality enforced by partial unique indexes; concurrent release attempts get unique constraint violation, loser retries
- **No auto-merge**. Quality control documents must never silently merge concurrent changes.

### Image upload failures

- 413 (too large) — clear message with max size
- 415 (unsupported type) — clear message with allowed types
- 500 (storage failure) — retry button, image NOT inserted into the document

### DOCX export failures

- **Released/archived export**: NEVER fails. Serves cached `docx_bytes`. The frozen artifact is always available.
- **Draft preview export** (production mode, default): any block render error fails the entire export with structured 500 pointing to the failing block.
- **Draft preview export** (debug mode, admin/dev only): per-block try/catch, failed blocks become red error placeholders, export continues.

### Migration failures

500 with structured error. Document is marked unloadable until manual repair by admin. This indicates a code bug, not user data.

### Template snapshot errors

- `TEMPLATE_SNAPSHOT_MISMATCH` (422): hash mismatch detected on template resolution
- `TEMPLATE_SNAPSHOT_MISSING` (422): template version not found in DB

Both fail closed. Document is unloadable until snapshot is restored or admin explicitly rebinds (Phase 2 feature).

### Block identity errors

- `BLOCK_ID_REWRITE_FORBIDDEN` (422): client/adapter regenerated an existing template block's id
- `LOCKED_BLOCK_DELETED` (422): client tried to delete a templated block

### Cross-doc reference errors

- `CROSS_DOC_REF_NOT_FOUND` (400): target document doesn't exist at save time
- `CROSS_DOC_REF_FORBIDDEN` (403): user doesn't have read access to the target

### Logging

Every error path emits a structured log entry:

```json
{
  "level": "error",
  "request_id": "req-uuid",
  "user_id": "user-uuid",
  "document_id": "doc-uuid",
  "document_version": "v3",
  "blockId": "block-uuid",
  "error_type": "validation_failed",
  "error_code": "BLOCK_ID_REWRITE_FORBIDDEN",
  "message": "..."
}
```

## 16. Testing

### 16.1 Schema validation tests (foundational)

`shared/schemas/test-fixtures/`:

```
valid/
  empty-po.json
  full-po.json
  multi-paragraph-fields.json
  etapas-with-images.json
  data-table-with-rows.json
  cross-doc-references.json

invalid/
  missing-section-title.json
  locked-block-modified.json
  repeatable-below-min-items.json
  invalid-color-format.json
  block-id-rewrite.json
  templated-block-deleted.json
```

Both languages run identical fixtures:

- TypeScript: `shared/schemas/__tests__/schema.test.ts` via AJV
- Go: `internal/modules/documents/domain/mddm/schema_test.go` via santhosh-tekuri/jsonschema

If TS and Go disagree about a fixture, the test fails. **Drift is impossible.**

### 16.2 Canonicalization tests

For every fixture in `valid/`, both TS and Go canonicalize and produce **byte-identical** output. Verified via golden output files.

### 16.3 Block component tests (frontend)

`frontend/apps/web/src/features/mddm-editor/blocks/__tests__/`:

Vitest + React Testing Library + BlockNote test utilities. Tests cover:

- Rendering with various props (locked vs unlocked, empty vs populated)
- Edit interactions (typing, adding/removing children for `Repeatable`/`DataTable`)
- Locked-state behavior (cannot edit, no controls shown)
- Auto-numbering computation
- ID preservation through edit cycles

### 16.4 DOCX export golden-file tests

`apps/docgen/__tests__/exporter/`:

For each fixture in `valid/`, the exporter produces a DOCX. We extract and assert specific properties of the XML:

```typescript
test("full-po renders correctly", async () => {
  const blocks = loadFixture("full-po.json");
  const docx = await exporter.export(blocks);
  const doc = await unzipAndParse(docx);

  // Margins
  expect(doc.pageMargins).toEqual({ top: 900, right: 900, bottom: 900, left: 900 });

  // Heading styles
  const h1 = doc.findFirst("w:p", p => p.style === "Heading1");
  expect(h1.text).toBe("1. Identificação do Processo");
  expect(h1.fontSize).toBe(28);  // half-points = 14pt

  // FieldGroup table
  const table = doc.findFirst("w:tbl");
  expect(table.rows[0].cells[0].text).toBe("Objetivo");
  expect(table.rows[0].cells[0].fontSize).toBe(20);  // 10pt — not 5pt

  // Repeatable item heading
  const h2s = doc.findAll("w:p", p => p.style === "Heading2");
  expect(h2s[0].text).toMatch(/^5\.1\s/);  // auto-numbered

  // Hyperlinks for cross-doc references
  const links = doc.findAll("w:hyperlink");
  expect(links.length).toBeGreaterThan(0);
});
```

Visual regression nightly: LibreOffice renders test DOCX → PDF for eyeballed review in PR.

### 16.5 Backend API integration tests

`internal/modules/documents/api/__tests__/` using Go testing + testcontainers Postgres:

- `POST /documents` — happy path, validation failures
- `POST /documents/:id/draft` — happy path, validation 400, version conflict 409
- `POST /documents/:id/release` — happy path, cardinality enforcement, atomic rollback on failure
- `GET /documents/:id/export/docx` — released serves cached bytes, draft renders fresh
- `POST /uploads/images` — happy path, 413, 415, MIME spoofing rejected
- Concurrency: simultaneous release attempts → exactly one succeeds, others get 409

### 16.6 E2E tests (Playwright)

5 critical user journeys:

1. Create from template → fill all fields → save draft → reload → verify persistence
2. Add 3 etapas with bullet lists + images → save → publish → approve → verify released DOCX has all 3 etapas with images
3. Two users editing same document → second saver gets 409 conflict modal
4. Delete a required field → save rejected with inline error highlighting the bad block
5. Cross-doc reference: create PO-A, save → create PO-B with `[see PO-A]` mention → verify hyperlink in DOCX

### 16.7 Adapter round-trip tests

`blockNoteToMDDM(mddmToBlockNote(blocks)) → normalizeMDDM === normalizeMDDM(blocks)` for every fixture.

### 16.8 Diff engine golden fixtures

For each `(oldBlocks, newBlocks)` pair, expected diff output is captured as JSON.

### 16.9 Migration property tests

Each migration `migrateVN_VN1` has:

- Input fixture (vN envelope)
- Expected output fixture (vN+1 envelope)
- Property test: migration is pure (same input → same output) and forward-only (no v→v-1 path)

### 16.10 Compatibility tests

Old golden documents from `mddm_version 1` still load, validate, and export correctly after schema bumps.

### 16.11 Locked-block enforcement tests

Golden fixtures of attempted illicit mutations:

- Modify a locked prop → rejected with `blockId`
- Delete a templated block → rejected
- Reorder structural siblings → rejected
- Add a non-templated block at a structural position → rejected
- Modify content inside a content slot → allowed

### 16.12 Template snapshot integrity tests

- Hash mismatch → `TEMPLATE_SNAPSHOT_MISMATCH` returned
- Missing template → `TEMPLATE_SNAPSHOT_MISSING` returned
- Migration runs only after hash verification (test ordering)

### 16.13 Template immutability tests

DB trigger test: UPDATE on a published template's `content_blocks` raises an exception.

### 16.14 Image storage tests

- Dedup: same bytes uploaded twice → same image_id
- MIME sniffing: `.png` filename with JPEG bytes → 415
- Reference reconciliation: draft save with removed image → join row deleted
- Orphan cleanup: image referenced only by archived version → deleted at release transition
- Pluggable interface: PostgresByteaStorage implements ImageStorage correctly

### 16.15 Block ID continuity tests

- Rewriting a templated block's id → 422
- Deleting a templated block → 422
- Rewriting a user-added block's id → allowed in v1 (best-effort)

### 16.16 Cross-doc reference tests

- Reference to non-existent doc at save time → 400
- Reference to forbidden doc at save time → 403
- Reference to deleted doc at render time → degraded placeholder + log warning

### 16.17 Size limit tests

Fixtures hitting each limit (max blocks, max nesting, max payload) — rejected with structured error.

### 16.18 Released DOCX immutability test

After release, querying `/api/documents/:id/export/docx?version_id=X` returns the SAME bytes regardless of how the renderer changes between calls. Verified with byte-level comparison.

### 16.19 Cardinality unique index tests

Concurrent release attempts on the same document → exactly one succeeds, others get unique constraint violation.

### 16.20 CI matrix

```
On every PR:
- Lint (TS + Go)
- Schema validation tests (TS + Go, identical fixtures)
- Canonicalization byte-identity tests
- Frontend unit tests (block components)
- Backend unit tests (validators, business rules)
- DOCX export golden-file tests
- Backend API integration tests with testcontainers Postgres
- Adapter round-trip tests
- Migration property tests
- Cardinality enforcement tests
- Template immutability tests

Nightly + on main:
- Everything above
- E2E Playwright tests (full stack)
- Visual regression via LibreOffice render
- Performance baselines
```

## 17. Out of scope for v1

Explicitly NOT in scope, deferred to Phase 2 or later:

- Real-time collaborative editing (CRDT/Yjs/operational transform)
- Mobile/responsive editor
- Offline editing (no service worker, no IndexedDB)
- AI-assisted authoring (auto-fill, generation, summarization)
- Word import (reverse direction)
- Cross-document references for navigation (existence + hyperlink IS in scope; richer features deferred)
- Per-block permissions
- Comments, suggestions, track changes
- In-document full-text search
- Template designer UI (templates edited via JSON in v1)
- Pixel-perfect WYSIWYG editor (editor approximates DOCX layout but is not pixel-identical)
- Document upgrades to new template versions (documents stay bound to original template snapshot forever in v1)
- Template snapshot recovery / manual rebind UI
- S3 image storage backend (interface is ready, implementation is Phase 2)
- Full user-block ID continuity enforcement (best-effort in v1)
- PDF storage (rendered on demand from DOCX in v1)
- Branching workflows (variant of REV03 for region X)

## 18. In scope for v1 (the foundational sprint)

- Clean-slate migration: delete all existing test PO documents and templates
- Hand-author a new PO template in MDDM JSON exercising every block type
- Schema documentation page with parent → children grammar table
- BlockNote editor integration with custom block schema
- Adapter layer (~200 lines) — explicit per-block-type translation
- Migration framework: forward-only, full-envelope, v1 baseline
- Diff engine (~100 lines) — operates on canonicalized envelopes
- Locked-block enforcer with position checks
- Canonicalization in TS + Go (precise spec, byte-identical implementations)
- DB triggers for template immutability
- Hash verification helpers
- `PostgresByteaStorage` implementation of `ImageStorage` interface
- `document_images` and `document_version_images` tables with cascade cleanup
- MIME sniffing for image uploads
- Cross-document reference inline mentions
- Two-mode export (production fail-closed for drafts, cached-bytes for released/archived)
- Per-document image authorization
- Single-transaction save and release operations
- Image reference reconciliation on every draft save
- Partial unique indexes for version cardinality
- Approval workflow integration with existing role-based access
- New PO template hand-authored in MDDM JSON

## 19. Phasing

This is a foundational sprint, ~4-6 weeks. It is treated as a single coherent investment, not a series of small features.

**Week 1**: Schema, canonicalization, validation foundations
- Define `mddm.schema.json`
- Implement `normalizeMDDM` in TS and Go (with byte-identity tests)
- Set up `document_versions` table with new columns and indexes
- Set up `document_images` and `document_version_images` tables
- DB triggers and partial unique indexes
- Migration framework skeleton

**Week 2**: Backend services
- Document service: load, draft save, release, archive operations
- Image service: upload, store, retrieve, reconciliation, orphan cleanup
- Template service: hash verification, snapshot binding
- Layer 2 business rule validators
- API endpoint handlers
- Backend integration tests with testcontainers

**Week 3**: Editor and adapter
- BlockNote integration with custom block schema
- Adapter layer (`mddmToBlockNote` / `blockNoteToMDDM`)
- Custom block React components for all 17 block types
- Frontend unit tests for blocks
- Cross-doc reference UI

**Week 4**: Docgen and DOCX export
- MDDM → DOCX compiler in `apps/docgen` using `docx` library
- Per-block-type render functions
- Auto-numbering, heading styles, table rendering
- Image embedding, hyperlinks for cross-doc refs
- Golden-file export tests

**Week 5**: Integration, end-to-end flows
- Create from template flow
- Draft → release → archive workflow
- Approval integration with existing role system
- Hand-author the new PO template in MDDM JSON
- Clean-slate migration (delete old test data)
- E2E Playwright tests

**Week 6**: Polish, documentation, deployment
- Schema documentation page
- Operations runbook (backups, restore, image lifecycle)
- Performance baseline tests
- Visual regression with LibreOffice
- Production deployment + monitoring

## 20. Validation history

This design was validated through nine adversarial review rounds with Codex (gpt-5.4) plus an industry-comparison research phase. Each round identified real foundational concerns; each fix tightened the design without adding architectural complexity.

- **Round 1**: Third-party coupling → MetalDocs-owned canonical schema + adapter layer + version migrations
- **Round 2**: Missing block identity → required immutable `id` on every block
- **Round 3**: Canonical normalization, template snapshot binding, size limits, frozen inline content schema
- **Round 4**: `template_block_id` separation (resolved instantiation/lock contradiction)
- **Round 5**: Template snapshot immutability + hash verification with correct ordering
- **Round 6**: Image/blob immutability via content-addressed storage (later replaced by release-based discarding model)
- **Round 7**: Block ID continuity enforcement across saves
- **Round 8** (comprehensive): canonicalization consistency, template mddm_version, frozen defaults, full block grammar, list semantics, MIME sniffing, fail-closed export, immutable export provenance, plus 9 other refinements
- **Round 9** (release-based pivot): introduction of draft/released/archived lifecycle, image reference reconciliation on draft save, DB-level cardinality enforcement, frozen DOCX at release time
- **Industry comparison**: validated against Notion (block model + PostgreSQL), Confluence (versioning), FDA 21 CFR Part 11, ISO 9001, BlockNote production usage (EU government Docs project), Google Docs (correctly diverged on event sourcing for our single-user use case)
- **Final round**: READY verdict with two non-blocking notes incorporated into this spec

## 21. References

- [Notion's data model](https://www.notion.com/blog/data-model-behind-notion) — block-based architecture in PostgreSQL
- [Confluence document versioning](https://www.atlassian.com/work-management/knowledge-sharing/documentation/storage-tracking)
- [FDA 21 CFR Part 11](https://www.fda.gov/regulatory-information/search-fda-guidance-documents/part-11-electronic-records-electronic-signatures-scope-and-application) — electronic records and audit trails
- [ISO 9001 QMS documentation](https://advisera.com/9001academy/knowledgebase/how-to-structure-quality-management-system-documentation/)
- [BlockNote](https://www.blocknotejs.org/) — block editor framework, used by EU government Docs project
- [docx npm library v9.6.1](https://github.com/dolanmiu/docx) — Word document generation
- [PostgreSQL bytea storage](https://wiki.postgresql.org/wiki/BinaryFilesInDB)
- [JSON Schema 2020-12](https://json-schema.org/draft/2020-12/schema)
- [santhosh-tekuri/jsonschema v6](https://github.com/santhosh-tekuri/jsonschema) — Go JSON Schema validator

---

**End of foundational design document.**
