# MetalDocs docx-editor Platform Design

## Goal

Replace the CKEditor 5 / MDDM document stack with a greenfield architecture built around `@eigenpal/docx-js-editor` (MIT, ProseMirror-based `.docx` editor). `.docx` files become the canonical visual source-of-truth for every template. Structured data captured via JSON Schema forms drives document generation. The new platform is designed as a multi-tenant SaaS foundation (tenant plumbing Day 1, runtime single-tenant) supporting Templates, Forms, and Documents with immutable content-addressed revisions, pessimistic editing locks, and a server-authoritative render pipeline.

The CK5 / MDDM code path is retired after the new path is end-to-end verified behind a feature flag; no data migration is performed.

## Architecture

### Service topology

```
┌─────────────────────────────────────────────────────────────┐
│ Browser (React + Vite, frontend/apps/web)                   │
│  ─ @metaldocs/editor-ui    — wrapper around docx-js-editor  │
│  ─ @metaldocs/form-ui      — rjsf/shadcn + Monaco schema    │
│  ─ @metaldocs/shared-tokens — docxtemplater token parser    │
│  ─ IndexedDB autosave (library hook)                        │
└───────────────────────┬─────────────────────────────────────┘
                        │ HTTPS + presigned S3 URLs
                        ▼
┌─────────────────────────────────────────────────────────────┐
│ Go API (apps/api) — existing, extended                      │
│  ─ auth / iam / audit (unchanged)                           │
│  ─ internal/modules/templates  (new)                        │
│  ─ internal/modules/documents  (new, replaces old module)   │
│  ─ internal/modules/editor_sessions (new)                   │
│  ─ internal/modules/document_revisions (new)                │
│  ─ orchestrates docgen-v2 + Gotenberg                       │
└───┬───────────────┬──────────────────────┬──────────────────┘
    │               │                      │
    ▼               ▼                      ▼
┌─────────┐ ┌───────────────┐ ┌────────────────────────┐
│ Postgres│ │  MinIO / S3   │ │ Node docgen-v2 (new)   │
│ (meta + │ │ (.docx, .json,│ │ apps/docgen-v2 Fastify │
│ JSONB)  │ │ .pdf blobs)   │ │ ─ processTemplate       │
└─────────┘ └───────────────┘ │ ─ parseDocxTokens       │
                              │ ─ ajv schema validate   │
                              │ ─ Gotenberg client      │
                              └────────────┬───────────┘
                                           │
                                           ▼
                                  ┌─────────────────┐
                                  │ Gotenberg       │
                                  │ (LibreOffice    │
                                  │ .docx → .pdf)   │
                                  └─────────────────┘
```

### Monorepo layout

```
apps/
  api/                        # Go, existing + new modules
  worker/                     # Go, existing (audit, notifications)
  docgen-v2/                  # NEW — Fastify, replaces apps/docgen + ck5-export
  # DELETED at W5 cutover: apps/docgen, apps/ck5-export, apps/ck5-studio

packages/
  docx-editor/                # Empty scaffold; subtree-ready for Phase 2 fork
  editor-ui/                  # NEW — MetalDocsEditor wrapper + plugins
  form-ui/                    # NEW — rjsf/shadcn + Monaco schema editor
  shared-tokens/              # NEW — canonical docxtemplater parser (TS + WASM)
  shared-types/               # NEW — DTOs shared across frontend + docgen-v2

frontend/apps/web/src/features/
  templates/                  # NEW — author screens
  documents/                  # NEW (replaces ck5/)
  # DELETED at W5: features/documents/ck5/

internal/modules/
  templates/                  # NEW
  documents/                  # REWRITTEN
  editor_sessions/            # NEW
  document_revisions/         # NEW
```

### Key architectural invariants

1. `.docx` and JSON Schema are authored in parallel and published as an atomic unit (TemplateVersion).
2. Documents always reference a specific immutable `template_version_id` — template evolution never retroactively breaks open documents.
3. All rendered `.docx` blobs are content-addressed and immutable; head state is a pointer in Postgres, never a mutable S3 key.
4. All server-side mutations on a Document require proof-of-session (pessimistic lock) + proof-of-base-revision (optimistic concurrency).
5. Server-side render is authoritative; client never calls `processTemplate` Day 1.
6. Token grammar is explicitly locked and enforced at publish; unsupported OOXML constructs are rejected rather than silently mis-rendered.

## Components

### Entities

| Entity | Purpose | Primary storage |
|-|-|-|
| Template | Logical template (e.g. "Purchase Order"). | `templates` row |
| TemplateVersion | Immutable-when-published snapshot binding a `.docx` and a JSON Schema. States: `draft`, `published`, `deprecated`. | `template_versions` row + 2 S3 objects |
| Document | Instance built from a TemplateVersion + form data. States: `draft`, `finalized`, `archived`. | `documents` row |
| DocumentRevision | Immutable content-addressed snapshot of a Document. Head pointer = `documents.current_revision_id`. | `document_revisions` row + S3 object |
| DocumentCheckpoint | User-labeled immutable milestone pointing at a DocumentRevision. | `document_checkpoints` row |
| EditorSession | Pessimistic lock + lineage tracker for one user editing one Document. | `editor_sessions` row |
| AutosavePendingUpload | Server-issued presigned-upload reservation binding session+base+hash. | `autosave_pending_uploads` row |
| FormData | JSON instance conforming to TemplateVersion schema. Stored inline on `documents` + snapshotted in revisions. | JSONB column |
| TemplateAuditLog | Append-only history. | `template_audit_log` table |

### Database schema (Postgres, greenfield migrations 0101+)

```sql
CREATE TABLE templates (
  id                            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id                     UUID NOT NULL,
  key                           TEXT NOT NULL,
  name                          TEXT NOT NULL,
  description                   TEXT,
  current_published_version_id  UUID,
  created_at                    TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at                    TIMESTAMPTZ NOT NULL DEFAULT now(),
  created_by                    UUID NOT NULL,
  UNIQUE (tenant_id, key)
);

CREATE TABLE template_versions (
  id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  template_id           UUID NOT NULL REFERENCES templates(id) ON DELETE CASCADE,
  version_num           INT NOT NULL,
  status                TEXT NOT NULL CHECK (status IN ('draft','published','deprecated')),
  grammar_version       INT NOT NULL DEFAULT 1,
  docx_storage_key      TEXT NOT NULL,
  schema_storage_key    TEXT NOT NULL,
  docx_content_hash     TEXT NOT NULL,
  schema_content_hash   TEXT NOT NULL,
  published_at          TIMESTAMPTZ,
  published_by          UUID,
  deprecated_at         TIMESTAMPTZ,
  lock_version          INT NOT NULL DEFAULT 0,
  created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
  created_by            UUID NOT NULL,
  UNIQUE (template_id, version_num)
);

CREATE UNIQUE INDEX idx_one_draft_per_template
  ON template_versions (template_id) WHERE status = 'draft';

ALTER TABLE templates
  ADD CONSTRAINT fk_current_published
  FOREIGN KEY (current_published_version_id) REFERENCES template_versions(id);

CREATE TABLE documents (
  id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id             UUID NOT NULL,
  template_version_id   UUID NOT NULL REFERENCES template_versions(id),
  name                  TEXT NOT NULL,
  status                TEXT NOT NULL CHECK (status IN ('draft','finalized','archived')),
  form_data_json        JSONB NOT NULL,
  current_revision_id   UUID,  -- FK set after first revision inserted
  active_session_id     UUID,  -- FK to editor_sessions
  finalized_at          TIMESTAMPTZ,
  archived_at           TIMESTAMPTZ,
  created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
  created_by            UUID NOT NULL
);

CREATE INDEX idx_documents_tenant_status ON documents (tenant_id, status);
CREATE INDEX idx_documents_template_version ON documents (template_version_id);
CREATE INDEX idx_documents_form_data_gin ON documents USING GIN (form_data_json jsonb_path_ops);

CREATE TABLE editor_sessions (
  id                              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  document_id                     UUID NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
  user_id                         UUID NOT NULL,
  acquired_at                     TIMESTAMPTZ NOT NULL DEFAULT now(),
  expires_at                      TIMESTAMPTZ NOT NULL,
  released_at                     TIMESTAMPTZ,
  last_acknowledged_revision_id   UUID NOT NULL,  -- mutable, advances on commit
  status                          TEXT NOT NULL CHECK (status IN ('active','expired','released','force_released'))
);

CREATE UNIQUE INDEX idx_one_active_session_per_doc
  ON editor_sessions (document_id) WHERE status = 'active';

CREATE TABLE document_revisions (
  id                     UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  document_id            UUID NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
  revision_num           BIGSERIAL,
  parent_revision_id     UUID REFERENCES document_revisions(id),
  session_id             UUID NOT NULL REFERENCES editor_sessions(id),
  storage_key            TEXT NOT NULL,
  content_hash           TEXT NOT NULL,
  form_data_snapshot     JSONB,
  created_at             TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (document_id, content_hash)
);

CREATE INDEX idx_revisions_doc_num ON document_revisions (document_id, revision_num DESC);

ALTER TABLE documents
  ADD CONSTRAINT fk_current_revision
    FOREIGN KEY (current_revision_id) REFERENCES document_revisions(id),
  ADD CONSTRAINT fk_active_session
    FOREIGN KEY (active_session_id) REFERENCES editor_sessions(id);

CREATE TABLE autosave_pending_uploads (
  id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  session_id           UUID NOT NULL REFERENCES editor_sessions(id) ON DELETE CASCADE,
  document_id          UUID NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
  base_revision_id     UUID NOT NULL REFERENCES document_revisions(id),
  content_hash         TEXT NOT NULL,
  storage_key          TEXT NOT NULL,
  presigned_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
  expires_at           TIMESTAMPTZ NOT NULL,
  consumed_at          TIMESTAMPTZ,
  UNIQUE (session_id, base_revision_id, content_hash)
);

CREATE INDEX idx_pending_expired
  ON autosave_pending_uploads (expires_at)
  WHERE consumed_at IS NULL;

CREATE TABLE document_checkpoints (
  id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  document_id        UUID NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
  revision_id        UUID NOT NULL REFERENCES document_revisions(id),
  version_num        INT NOT NULL,
  label              TEXT,
  created_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
  created_by         UUID NOT NULL,
  UNIQUE (document_id, version_num)
);

CREATE TABLE template_audit_log (
  id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id            UUID NOT NULL,
  template_id          UUID,
  template_version_id  UUID,
  document_id          UUID,
  action               TEXT NOT NULL,
  actor_user_id        UUID NOT NULL,
  metadata_json        JSONB,
  created_at           TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_audit_tenant_created
  ON template_audit_log (tenant_id, created_at DESC);
```

Runtime enforcement: `GRANT INSERT, SELECT ON template_audit_log TO metaldocs_app;` (no UPDATE/DELETE).

### Storage (S3 / MinIO) key scheme

```
tenants/{tid}/templates/{template_id}/v{n}.docx
tenants/{tid}/templates/{template_id}/v{n}.schema.json
tenants/{tid}/documents/{document_id}/revisions/{content_hash}.docx
tenants/{tid}/documents/{document_id}/exports/{composite_hash}.pdf
```

`composite_hash = sha256(docx_content_hash || template_version_id || grammar_version || docgen_v2_version || canonical_json(render_opts))`.

Bucket policy: block public access, SSE-S3 (SSE-KMS Phase 2), versioning OFF (revision immutability handled in app layer). `/uploads/presign` enforces Content-Type whitelist and max-size per kind (template docx 10MB, document docx 25MB, schema 256KB, form_data 1MB).

### Token grammar (`packages/shared-tokens`)

```ebnf
token   = "{" ident "}"
        | "{#" ident "}" ... "{/" ident "}"
        | "{^" ident "}" ... "{/" ident "}"
ident   = [a-zA-Z_][a-zA-Z0-9_]*
```

**Day 1 restrictions (enforced at publish):**
- No field paths with dots (`{client.name}` rejected; use `{client_name}`).
- No nested sections beyond 1 level.
- No filters, expressions, triple-brace.
- Idents reserved for docgen internals are rejected.

**OOXML whitelist (parser supported):**
`<w:p>`, `<w:r>`, `<w:t>`, `<w:tab>`, `<w:br>`, `<w:tbl>`, `<w:tr>`, `<w:tc>`, `<w:pPr>`, `<w:rPr>`, `<w:hyperlink>`, `<w:drawing>`, `<w:hdr>`, `<w:ftr>`, `<w:sectPr>`.

**OOXML rejected (publish returns `unsupported_construct`):**
`<w:ins>`, `<w:del>`, `<w:moveFrom>`, `<w:moveTo>`, `<w:sdt>`/`<w:sdtContent>`, `<w:comment*>` inside body, `<w:bookmarkStart/End>` intersecting any token, `<w:bidi>`/`<w:rtl>` inside a token run, `<w:proofErr>` intersecting any token, `<w:smartTag>`, `<w:fldSimple>`, `<w:fldChar>`, `<w:object>`, `<w:pict>`, `<w:altChunk>`, nested `<w:tbl>` inside `<w:tc>`.

**Parser result type:**

```ts
interface ParseResult {
  tokens: Array<{ kind: 'var' | 'section' | 'inverted' | 'closing';
                  ident: string; start: number; end: number; run_id: string }>;
  errors: Array<
    | { type: 'split_across_runs'; run_ids: string[]; token_text: string; auto_fixable: true }
    | { type: 'unsupported_construct'; element: string; location: string; auto_fixable: false }
    | { type: 'reserved_ident'; ident: string; location: string }
    | { type: 'malformed_token'; raw: string; location: string }
    | { type: 'nested_section_too_deep'; ident: string; depth: number }
    | { type: 'unmatched_closing'; ident: string; location: string }
  >;
}
```

Grammar version is locked per TemplateVersion (`template_versions.grammar_version`). Parser upgrades are opt-in for new drafts only; existing published versions continue to use the grammar they were published under.

### Editor wrapper (`packages/editor-ui`)

Public API:

```ts
export { MetalDocsEditor } from './MetalDocsEditor';
export type { MetalDocsEditorProps, MetalDocsEditorRef } from './types';
export { mergefieldPlugin } from './plugins/mergefieldPlugin';
export { brandTheme } from './theme';
```

```ts
interface MetalDocsEditorProps {
  documentId?: string;
  documentBuffer?: ArrayBuffer;
  mode: 'template-draft' | 'document-edit' | 'readonly';
  schema?: JSONSchema;
  onAutoSave?: (buf: ArrayBuffer) => Promise<void>;
  onLockLost?: () => void;
  userId: string;
}
```

**Plugins Day 1:**
- `mergefieldPlugin(schema)` — runs `parseDocxTokens` on every change, cross-references schema idents, renders sidebar (used/missing/orphan), provides `insertField(ident)` command.
- `brandThemePlugin` — CSS variable overlay (colors, fonts).
- `overrides.css` — forces `.layout-page { min-height: var(--docx-page-min-h) !important }` (fixes library's content-fit last-page bug); custom scrollbars + focus rings matching shadcn tokens.

**Library version:** `@eigenpal/docx-js-editor` pinned at **`0.0.34`** exact in frontend wrapper + docgen-v2 + shared-tokens host. CI: `pnpm install --frozen-lockfile`.

### Form UI (`packages/form-ui`)

- Runtime renderer: `@rjsf/shadcn` with custom widgets (`ClientPickerWidget`, `ItemsTableWidget`, `SignatureWidget` Phase 2).
- Validation client-side: `ajv` with meta-schema validation + per-field errors.
- Schema editor Day 1: Monaco editor with JSON Schema meta-schema loaded for autocomplete.
- Schema builder drag-drop: **Phase 2**.

### Docgen-v2 (`apps/docgen-v2`)

Node 20 + Fastify + TypeScript. Internal-only HTTP, shared-secret `X-Service-Token` auth, not exposed via public ingress.

| Method | Path | Body | Returns |
|-|-|-|-|
| POST | `/render/docx` | `{ template_docx_key, schema_key, form_data, output_key }` | `{ output_key, content_hash, size_bytes, warnings[], unreplaced_vars[] }` |
| POST | `/convert/pdf` | `{ docx_key, output_key }` | `{ output_key, content_hash, size_bytes }` |
| POST | `/validate/schema` | `{ schema_key, form_data }` | `{ valid, errors[] }` |
| POST | `/validate/template` | `{ docx_key, schema_key }` | `{ valid, parse_errors[], missing_tokens[], orphan_tokens[] }` |
| GET | `/health` | — | `{ status, version }` |

Render pipeline: fetch template docx + schema from S3 → `parseDocxTokens` → reject on any parse error → `ajv.validate(schema, form_data)` → `processTemplate(buffer, form_data, { nullGetter: 'empty' })` → compute `content_hash` → S3 PUT → return metadata.

PDF pipeline: fetch `.docx` from S3 → multipart POST to `http://gotenberg:3000/forms/libreoffice/convert` → S3 PUT response → return metadata.

## Data Flow

### Template authoring (W2 vertical)

```
1. Author opens /templates → POST /templates → creates template + v1 draft
2. Split-pane editor loads:
   LEFT   <MetalDocsEditor mode="template-draft" documentBuffer={v1.docx}>
   RIGHT  Tabs: [Schema] Monaco editor | [Preview] rjsf rendering of current schema
3. Author types tokens in docx; mergefieldPlugin live-parses and sync-checks with schema.
4. Draft autosave (every 15s idle):
     docx side: presigned PUT → PUT /templates/{id}/versions/{n}/docx (content_hash, lock_version)
     schema side: PUT /templates/{id}/versions/{n}/schema (same optimistic lock_version)
5. Author clicks [Publish]:
     POST /templates/{id}/versions/{n}/publish
       → docgen-v2 /validate/template (parseDocxTokens + diffTokensVsSchema + schema meta-validate)
       → if valid: INSERT new immutable row status='published', version_num++, update
         templates.current_published_version_id
       → audit: template.published
     On success, opens a fresh draft automatically for next iteration.
```

### Document fill + edit (W3 vertical)

```
1. End user opens /documents/new → picks template.
2. <FormRenderer schema={published_schema}> auto-renders inputs from the schema.
3. User fills; ajv validates client-side.
4. Clicks [Generate Document]:
     POST /documents { template_version_id, name, form_data }
       → Go API: gojsonschema validate form_data
       → docgen-v2 /render/docx → revisions/{content_hash}.docx in S3
       → acquire session:
           INSERT editor_sessions { last_acknowledged_revision_id = new_revision.id, ... }
           UPDATE documents SET active_session_id = session.id, current_revision_id = revision.id
       → audit: document.created
       → returns { document_id, session_id, initial_revision_id }
5. Client navigates to /documents/{id}; editor loads via signed URL to revision blob.
```

### Atomic autosave (document editing)

```
Client autosave trigger (library onChange debounced 500ms local + 15s server sync):
  content_hash = sha256(editor.saveDocument())

  POST /documents/{id}/autosave/presign
    { session_id, base_revision_id = last_ack_revision, content_hash }

  Server (single DB tx):
    SELECT editor_sessions WHERE id=:session FOR UPDATE
    → reject if not active, not holder of documents.active_session_id,
      or session.last_acknowledged_revision_id != base_revision_id
    INSERT autosave_pending_uploads ON CONFLICT (session,base,hash) DO NOTHING
      RETURNING id, storage_key
    Issue presigned PUT for storage_key with Content-Type + Content-Length caps
    → { upload_url, pending_upload_id, expires_at=now()+15min }

  Client: PUT docx bytes to S3

  POST /documents/{id}/autosave/commit
    { session_id, pending_upload_id }

  Server (single DB tx):
    SELECT pending FOR UPDATE
      → 409 misbound if session_id mismatch
      → 200 idempotent_replay if already consumed (returns existing revision)
      → 410 expired_upload if expires_at < now()
    Re-verify session active + holder + last_ack == pending.base_revision
    Stream S3 object revisions/{hash}.docx → recompute sha256
      → 422 content_hash_mismatch if ≠ pending.content_hash (delete orphan S3 obj)
    INSERT document_revisions
      { parent = pending.base_revision, session_id, storage_key, content_hash, form_data_snapshot }
      RETURNING id
    UPDATE documents SET current_revision_id = new_revision_id
    UPDATE editor_sessions SET last_acknowledged_revision_id = new_revision_id
    UPDATE autosave_pending_uploads SET consumed_at = now()
    → 200 { revision_id, revision_num }
    audit: document.autosaved
```

Client advances its local `last_ack_revision_id = revision_id` on 200.

### Session lifecycle

- **Acquire** (`POST /documents/{id}/session/acquire`): if no active session, INSERT with `last_acknowledged_revision_id = documents.current_revision_id`, UPDATE `documents.active_session_id`. If caller already holds → refresh. If another user holds → returns `{ mode: 'readonly', held_by, held_until }`.
- **Heartbeat** (`POST .../session/heartbeat`, 30s): UPDATE `expires_at = now() + 5min` only if still active holder.
- **Release** (`POST .../session/release`): sets `status='released'`, `released_at=now()`, clears `documents.active_session_id`.
- **Expire** (background job): any `active` session with `expires_at < now()` → `status='expired'`, clear pointer.
- **Force-release** (admin, `POST .../session/force-release`): sets `status='force_released'`. Stale tab's next commit hits 409 `session_inactive`.

### Checkpoint + restore

- **Create** (`POST /documents/{id}/checkpoints { label }`): INSERT `document_checkpoints { revision_id = documents.current_revision_id, version_num = next, label }`. No S3 copy; immutable revision already exists.
- **Restore** (`POST /documents/{id}/checkpoints/{versionNum}/restore`): treated as a normal commit. Client sees the checkpoint revision content, adopts it as its `base_revision_id`, and immediately performs an autosave creating a new revision whose `parent_revision_id = checkpoint.revision_id`. Head advances forward; history preserved.

### PDF export

```
POST /documents/{id}/export/pdf
  → Go API:
    composite_hash = sha256(
        current_revision.content_hash
      || template_version_id
      || template_version.grammar_version
      || docgen_v2_version
      || canonical_json(render_opts)
    )
    HEAD S3 exports/{composite_hash}.pdf
      exists  → return signed GET URL (cached)
      missing → docgen-v2 /convert/pdf
                  → Gotenberg → S3 PUT
                  → return signed GET URL
  audit: export.pdf_generated
```

### RBAC + tenancy

Every handler: (1) existing IAM middleware resolves user + tenant → context; (2) `requireRole()` check against matrix below; (3) every SQL SELECT/UPDATE/DELETE filters by `tenant_id`; (4) S3 keys prefixed `tenants/{tid}/…`, signed URLs scoped to prefix.

| Action | admin | template_author | document_filler |
|-|:-:|:-:|:-:|
| List / view published templates | ✓ | ✓ | ✓ |
| Create / edit template draft | ✓ | ✓ | — |
| Publish template version | ✓ | ✓ | — |
| Deprecate / delete template | ✓ | — | — |
| Create / edit own documents | ✓ | — | ✓ |
| Edit others' documents | ✓ | — | — |
| Finalize / archive document | ✓ | — | ✓ (own) |
| Force-release session | ✓ | — | — |
| View audit log | ✓ | — | — |

### HTTP surface (Go API)

Templates: `GET/POST /templates`, `GET/PATCH/DELETE /templates/{id}`, `GET/POST /templates/{id}/versions`, `GET /templates/{id}/versions/{n}`, `PUT /templates/{id}/versions/{n}/{docx|schema}`, `POST /templates/{id}/versions/{n}/{publish|deprecate}`, `GET /templates/{id}/versions/{n}/{docx-url|schema-url}`.

Documents: `GET/POST /documents`, `GET /documents/{id}`, `PATCH /documents/{id}`, `POST /documents/{id}/{finalize|archive}`, `DELETE /documents/{id}` (admin).

Sessions: `POST /documents/{id}/session/{acquire|heartbeat|release|force-release}`.

Autosave: `POST /documents/{id}/autosave/{presign|commit}`.

Checkpoints: `GET/POST /documents/{id}/checkpoints`, `POST /documents/{id}/checkpoints/{n}/restore`.

Render + export: `POST /documents/{id}/render`, `GET /documents/{id}/export/docx-url`, `POST /documents/{id}/export/pdf`.

Upload: `POST /uploads/presign`.

Audit: `GET /audit/templates/{id}`, `GET /audit/documents/{id}` (admin).

Error codes: `403` role/tenant denied; `409 Conflict` (`session_inactive`, `session_lost`, `lock_lost`, `stale_base`, `misbound`); `410 Gone` expired presign; `422 Unprocessable` (`schema_invalid`, `content_hash_mismatch`, token parse errors, missing/orphan tokens).

### Rate limits

- `/uploads/presign`, `/documents/{id}/autosave/presign`: 60 req/min/user
- `/documents/{id}/render`: 30 req/min/user
- `/documents/{id}/export/pdf`: 20 req/min/user
- `/documents/{id}/autosave/commit`: 30 req/min/user

## Error Handling

### Layered defense

| Layer | Validates |
|-|-|
| Client rjsf + ajv | UX feedback per keystroke |
| Go API (gojsonschema) | All `POST`/`PATCH` with `form_data` |
| Docgen-v2 (ajv) | Re-validates before render (independent codepath) |
| parseDocxTokens (shared-tokens) | Runs in docgen-v2 at publish + render; runs in editor-ui live |

### Specific failure modes

- **Stale tab autosave:** 409 `stale_base` or `session_inactive`. Editor displays "You're behind / session taken over" banner, offers reload. IndexedDB local copy kept for manual export if user wants to reconcile.
- **Content-hash mismatch:** orphan S3 object deleted, pending row left consumed=false + expires_at; client retries with re-presign.
- **Presign expired:** 410 `expired_upload`; client re-presigns + re-PUTs. Idempotent on same `(session, base, hash)`.
- **Session force-released by admin:** next commit hits 409 `session_inactive`. Client prompts user, offers export of local state before reload.
- **Publish with unsupported OOXML:** 422 with typed `parse_errors` array; editor shows red sidebar banner and disables publish button.
- **Publish with split tokens only:** 422 `split_tokens`; editor offers "Fix tokens" one-click auto-merge (PM transaction) then retry publish.
- **Docgen-v2 unreachable:** Go API 502 with `retry-after`. Client retries once with backoff; surfaces generic error after.
- **Gotenberg unreachable / conversion failed:** 502 from `/export/pdf`; no cache populated; user can retry. PDF path is non-blocking for core editing.
- **Optimistic lock mismatch on template draft:** 409 `template_draft_stale`; editor reloads draft state.
- **Finalize attempt on non-draft document:** 409 `invalid_state_transition`.
- **Cross-tenant access attempt:** 404 (not 403, to avoid tenancy disclosure).

### Orphan cleanup

- **Hourly job:** `DELETE FROM autosave_pending_uploads WHERE expires_at < now() - interval '24h' AND consumed_at IS NULL`.
- **Weekly job:** list S3 `tenants/*/documents/*/revisions/*` → left-join with `document_revisions.storage_key` → delete S3 objects with no DB row older than 24h.
- **Session expiry sweep:** every minute, any `active` session with `expires_at < now()` → `status='expired'`, clear `documents.active_session_id`.

### Audit log actions

`template.created`, `template.draft_saved`, `template.published`, `template.deprecated`, `template.deleted`, `document.created`, `document.autosaved`, `document.checkpoint_created`, `document.checkpoint_restored`, `document.finalized`, `document.archived`, `document.deleted`, `session.acquired`, `session.released`, `session.force_released`, `export.pdf_generated`, `export.docx_downloaded`.

## Testing Approach

### Go backend (unit + integration)

- Unit: domain state transitions (template publish/deprecate, document draft/finalized/archived, session acquire/heartbeat/expire/force-release).
- Integration (testcontainers: Postgres + MinIO): full CRUD per module + S3 round-trips; optimistic lock conflict cases; partial unique index behavior.
- Autosave commit tx tests (table-driven): each of the 8 rejection branches (stale session, wrong holder, stale base, missing pending, consumed pending, expired pending, S3 missing, hash mismatch) asserted to leave DB state untouched.
- Cascade tests: template delete → versions + revisions + checkpoints gone; document delete → sessions + pending uploads gone.

### Docgen-v2 (unit + golden file)

- Fixture template set: 5 canonical templates (PO, invoice, NDA, letter, multi-page-with-table). Golden-file regression: rendered output `content_hash` compared to checked-in golden per (lib_version × template × form_data) matrix. Failing golden blocks merge.
- parseDocxTokens: parametric over all OOXML rejection cases — each must return the right typed error.
- Schema validation: 20 test cases covering required, nested arrays, conditionals, cross-field logic.
- Gotenberg contract: mock server for unit; real container for CI integration.

### Frontend

- Vitest: mergefield sidebar diff logic, autosave state machine (session heartbeat, 409 handling, 410 re-presign), form validation, IndexedDB restore.
- Testing Library component tests: FormRenderer rendering all widget types, MetalDocsEditor lock-loss flow, template editor split-pane.
- Playwright E2E:
  - **author-happy-path:** create template → upload docx → author schema → publish → appears in list.
  - **filler-happy-path:** pick template → fill form → generate → edit → checkpoint → finalize → export PDF.
  - **conflict-two-tabs:** two browsers same document — second readonly; first releases; second acquires.
  - **autosave-crash:** type → kill tab → reopen → IndexedDB restores + server commit catches up.
  - **stale-base:** force second tab's commit after first commits → 409 → reload flow.
  - **force-release:** admin takeover → stale tab sees banner → no data loss.
  - **publish-rejects-tracked-changes:** author inserts tracked-change fixture → publish returns 422.

### Performance smoke tests

- Render of 50KB template + 10KB form_data → docgen-v2 latency p95 < 500ms (single pod).
- PDF conversion 10-page doc → p95 < 5s.
- Document load (editor mount → first paint with 5MB docx) < 2s.

### Phased rollout (4 weeks build + 1 week cutover)

- **W1** — scaffold new packages + modules *alongside* old code. New DB tables introduced; no drops. Feature flag `feature.docx_v2_enabled` (per-tenant, default OFF). New routes under `/api/v2/...`. Docker Compose extended: MinIO + docgen-v2 + Gotenberg.
- **W2** — templates vertical behind flag. E2E author-happy-path green.
- **W3** — documents vertical + docgen-v2 behind flag. E2E filler-happy-path + autosave-crash + conflict-two-tabs green.
- **W4** — PDF export, RBAC hardening, rate limits, full E2E suite green. Internal dogfood behind flag for 5 business days.
- **W5 (cutover)** — flip `feature.docx_v2_enabled=on` globally. One-week soak. Then destructive migration deleting old tables + folders + packages: `apps/docgen`, `apps/ck5-export`, `apps/ck5-studio`, `frontend/apps/web/src/features/documents/ck5/`, `internal/modules/documents/*/service_ck5_*.go`, old `document_template_versions`, `template_drafts`, `blocks_json` artifacts. Main branch stays shippable throughout; destructive commit lands only after flag flip holds steady.

## Out of Scope

Explicitly not built Day 1 (Phase 2+ candidates):

- Multi-tenant runtime (tenant provisioning UI, billing, tenant admin). Schema is tenant-ready; runtime is single-tenant seed.
- Real-time collaborative editing (Yjs / CRDT). Day 1 is pessimistic lock only.
- Visual drag-drop form builder. Day 1 is Monaco JSON Schema editor.
- Approver workflow / signature requests / email notifications.
- Public REST API, auth keys, OpenAPI SDK. Internal API only.
- Template marketplace, MetalDocs-curated catalog.
- AI autofill from email / PDF / LLM function-calling integration.
- Fork of `@eigenpal/docx-js-editor` and any feature requiring internals (restricted-cell editing, custom toolbar buttons, custom node schemas). Scaffold is ready under `packages/docx-editor/`; fork is triggered by first confirmed blocker.
- PDF import / in-browser PDF editing.
- `.doc` (binary) / `.odt` / `.rtf` upload.
- Mobile apps (responsive web only).
- Visual version-diff rendering beyond checkpoint list.
- `form_data` history beyond revision snapshots.
- SSE-KMS, cross-region S3 replication, mTLS between internal services.
- Client-side `processTemplate` preview (deferred; Phase 2 behind explicit "may differ from canonical" banner).
- Grammar extensions beyond Day 1 EBNF (field paths with dots, deep-nested sections, filters, expressions).
