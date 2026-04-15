---
title: Backend API contracts
status: draft
area: backend
priority: HIGH
---

# 35 — Backend API contracts

How templates and documents are stored, what payloads flow over the wire,
how versioning and concurrency are negotiated, and which auth scopes gate
which operations. Grounded in the HTML-as-SoT decision from
[25 — MDDM IR bridge](./25-mddm-ir-bridge.md) and the data-format rules
from [24 — Data format](./24-data-format.md).

⚠ uncertain — which existing MetalDocs backend framework we target
(Nest, Express, Fastify, or the Python sidecar). The shapes below are
framework-agnostic; route prefixes and middleware hooks adjust per host.

---

## Recommended data model

One row per template, one row per document. No IR column. No ir-hash.

### `templates`

| Column             | Type        | Notes                                                                       |
|--------------------|-------------|-----------------------------------------------------------------------------|
| `id`               | UUID PK     |                                                                             |
| `org_id`           | UUID FK     | Scoping key (see §7).                                                       |
| `name`             | TEXT        | Human label, unique per org.                                                |
| `description`     | TEXT        | Optional.                                                                   |
| `version`          | INT         | Monotonic; bumps on publish.                                                |
| `status`           | ENUM        | `draft` \| `published`.                                                     |
| `content_html`     | TEXT        | CK5 `getData()` output. Canonicalized (see §4).                             |
| `manifest`         | JSONB       | Field definitions, field groups, title style, variant settings.             |
| `created_by`       | UUID        |                                                                             |
| `created_at`       | TIMESTAMPTZ |                                                                             |
| `updated_at`       | TIMESTAMPTZ |                                                                             |

### `documents`

| Column              | Type        | Notes                                                                      |
|---------------------|-------------|----------------------------------------------------------------------------|
| `id`                | UUID PK     |                                                                            |
| `org_id`            | UUID FK     |                                                                            |
| `owner_id`          | UUID FK     |                                                                            |
| `template_id`       | UUID FK     |                                                                            |
| `template_version`  | INT         | Snapshot of template version at instantiation.                             |
| `content_html`      | TEXT        | CK5 data-view HTML; source of truth.                                       |
| `field_values`      | JSONB       | Derived cache `{ fieldId: value }`. Always regeneratable from HTML.        |
| `status`            | ENUM        | `draft` \| `filled` \| `submitted` \| `archived`.                          |
| `lock_version`      | INT         | Optimistic concurrency counter (see §6).                                   |
| `created_at`        | TIMESTAMPTZ |                                                                            |
| `updated_at`        | TIMESTAMPTZ |                                                                            |

Explicitly **not** in the schema: `content_ir`, `ir_hash`, `blocks`,
`layout_json`. Page 25 removes those.

---

## Endpoint list

### Templates

| Method | Path                                 | Purpose                                       |
|--------|--------------------------------------|-----------------------------------------------|
| POST   | `/templates`                         | Create draft template.                        |
| GET    | `/templates`                         | List templates (org-scoped).                  |
| GET    | `/templates/:id`                     | Fetch one template.                           |
| PATCH  | `/templates/:id`                     | Autosave draft; bumps `updated_at`.           |
| POST   | `/templates/:id/publish`             | Promote draft → published; bumps `version`.   |
| POST   | `/templates/:id/clone`               | Duplicate template (new id, `draft`).         |

### Documents

| Method | Path                                   | Purpose                                                 |
|--------|----------------------------------------|---------------------------------------------------------|
| POST   | `/documents`                           | Instantiate from `template_id` (+ optional pre-fill).   |
| GET    | `/documents`                           | List documents for caller (owner + org scope).          |
| GET    | `/documents/:id`                       | Fetch one document.                                     |
| PATCH  | `/documents/:id`                       | Autosave; requires `lock_version`.                      |
| POST   | `/documents/:id/submit`                | Transition `filled` → `submitted`.                      |
| POST   | `/documents/:id/export/docx`           | Generate DOCX (sync or async 202).                      |
| POST   | `/documents/:id/export/pdf`            | Generate PDF (sync or async 202).                       |

### Assets

| Method | Path        | Purpose                                                        |
|--------|-------------|----------------------------------------------------------------|
| POST   | `/assets`   | Upload image/binary; returns CDN URL. See §11.                 |

---

## 3. Payload shapes

All payloads are JSON. `content_html` is a plain string, not a wrapped
structure — this matches `editor.getData()` output byte-for-byte after
server canonicalization.

### Create template

```http
POST /templates
Content-Type: application/json

{
  "name": "Purchase Order v3",
  "description": "Standard PO for metals division",
  "content_html": "<div data-mddm-schema=\"v1\"><h1>Purchase Order</h1>…</div>",
  "manifest": {
    "fields": [
      { "id": "supplier_name", "type": "text", "required": true },
      { "id": "po_number",     "type": "text", "required": true }
    ],
    "fieldGroups": [],
    "titleStyle": "heading-1",
    "variant": "mixed"
  }
}
```

Response: `201 Created` with the full row including `id`, `version: 1`,
`status: "draft"`.

### Create document (instantiate)

```http
POST /documents
Content-Type: application/json

{
  "template_id": "8e1f…",
  "prefill": {
    "supplier_name": "Acme Corp"
  }
}
```

Server loads the template's `content_html`, applies the prefill map by
writing values into the matching `restricted-editing-exception` spans
(by `data-field-id`; see page 25 §8), and persists the result as the new
document's `content_html`. Response: `201 Created` with the full document,
`status: "draft"`, `lock_version: 0`.

### Autosave PATCH

```http
PATCH /documents/:id
Content-Type: application/json
If-Match: "lock_version=7"

{
  "content_html": "<div data-mddm-schema=\"v1\">…edited…</div>",
  "lock_version": 7
}
```

Success: `200 OK` with `{ "lock_version": 8, "updated_at": "…",
"field_values": { … } }`. Conflict: `409` (see §6).

---

## 4. Ingress validation

Every write that carries `content_html` runs the payload through a
canonicalization step **before** the row is stored. Pipeline:

1. **Parse** with linkedom (per [25 §2](./25-mddm-ir-bridge.md)).
2. **Sanitize** — strip scripts, inline event handlers, `javascript:` URLs,
   unknown elements outside the GHS allow-list.
3. **Schema check** against the template manifest:
   - required fields present (by `data-field-id`);
   - no nested tables (CKEditor figure-wrap issue blocks DOCX export);
   - restricted-editing exception composition matches the template
     (no new exception regions introduced on document writes);
   - no unknown `data-mddm-*` attributes outside the registered set.
4. **Re-serialize** the sanitized DOM as the canonical form.

Reject with `422` on violations, returning a structured error listing the
offending nodes. Direction for [37 — Validation](./37-validation.md):
that page owns the sanitizer allow-list, the schema-check rules, and the
error envelope; this page only names the ingress hook.

---

## 5. Versioning strategy

The root wrapper element carries `data-mddm-schema="v1"` (per page 25 §9).

- On `PATCH` / `POST`, the server reads the schema attribute.
- If it matches `CURRENT`, proceed.
- If older, run the upcast migration chain located in
  `backend/src/content/migrations/` (⚠ uncertain — final path depends on
  framework choice). Migrations are pure HTML-to-HTML, idempotent at their
  own version, and versioned sequentially.
- If newer than `CURRENT`, reject `409` and return
  `{ error: "client_newer_than_server", server_schema: "v1" }`. The
  client must downgrade or the deployment must roll forward.

No silent migration on **read**. Read returns whatever schema the row is
stored at; the client upcasts during `setData()`.

---

## 6. Concurrency

Optimistic locking via `lock_version`:

- Every document row has an `INT lock_version`, starting at `0`.
- Autosave `PATCH` includes the client's last-known `lock_version` in the
  JSON body (and, optionally, as an `If-Match` header for clients that
  prefer HTTP-native semantics).
- Server: `UPDATE documents SET content_html = $1, lock_version =
  lock_version + 1 WHERE id = $2 AND lock_version = $3`. Zero rows
  affected → `409 Conflict` with the current server state:
  ```json
  { "error": "lock_version_mismatch",
    "server_lock_version": 9,
    "server_updated_at": "2026-04-15T14:22:11Z" }
  ```
- Client handles `409` by reloading the document and prompting the user
  (see [26 — Autosave & persistence](./26-autosave-persistence.md)).

Templates use the same mechanism for draft edits; published templates
are immutable (clone-to-edit).

---

## 7. Auth scopes

| Resource               | Read                           | Write / autosave              | Publish / admin          |
|------------------------|--------------------------------|-------------------------------|--------------------------|
| Template (draft)       | org member                     | template author + org admin   | org admin                |
| Template (published)   | org member                     | — (immutable)                 | org admin (archive)      |
| Document               | owner + org admin              | owner                         | owner (submit)           |
| Asset                  | org member with signed URL     | org member                    | —                        |

Publish (`POST /templates/:id/publish`) requires the `template:publish`
scope, gated to the org-admin role. Cross-org access is never granted,
regardless of role.

---

## 8. Derived fields cache

`field_values JSONB` is recomputed on every successful write:

1. Parse the canonicalized `content_html`.
2. Walk `span[data-field-id]` and `div[data-field-id]` nodes within
   restricted-editing exceptions.
3. Emit `{ [fieldId]: textContent }`.

Indexed for search: `CREATE INDEX ON documents USING GIN (field_values)`.
Never queried by the editor — exists purely for server-side list views
and reports, so queries never require a CK5 runtime.

If `field_values` drifts from `content_html` (e.g. after a hot migration),
a backfill job re-extracts from HTML. HTML stays authoritative.

---

## 9. Export endpoints

DOCX and PDF generation lives server-side (see
[31 — DOCX export](./31-docx-export.md) and
[32 — PDF export](./32-pdf-export.md) for the pipeline).

- **Small documents** (⚠ uncertain threshold — suggest < 10 pages / <
  500 KB HTML): handle synchronously, return `200 OK` with the binary.
- **Large documents**: enqueue a background job, return `202 Accepted`
  with a job id and poll URL:
  ```json
  { "job_id": "exp_8a…", "status_url": "/jobs/exp_8a…" }
  ```
- Exports never mutate `content_html`. They read the canonical HTML as of
  the request timestamp; later edits do not affect in-flight jobs.

Jobs write artefacts to object storage; the response includes a signed
URL valid for 15 minutes.

---

## 10. Audit log

Every write (template `PATCH`/publish, document `PATCH`/submit) appends
to `audit_events`:

| Column        | Type        | Notes                                         |
|---------------|-------------|-----------------------------------------------|
| `id`          | UUID PK     |                                               |
| `actor_id`    | UUID        |                                               |
| `resource`    | TEXT        | `template:<id>` or `document:<id>`.           |
| `action`      | TEXT        | `patch`, `publish`, `submit`, etc.            |
| `content_hash`| TEXT        | SHA-256 of the new `content_html`.            |
| `diff_bytes`  | INT         | `len(new) - len(old)`; cheap drift signal.    |
| `at`          | TIMESTAMPTZ |                                               |

We do not store full HTML diffs — too large, and the hash + byte delta is
enough for compliance and drift alerting. Full-history retention lives in
a separate snapshot table (⚠ uncertain — out of scope for v1).

---

## 11. Storage considerations

- `content_html` is TEXT. Target p95 size: ~500 KB; hard cap: 5 MB. Beyond
  that, writes return `413 Payload Too Large`.
- **Images are never base64-embedded.** Client uploads via `POST /assets`
  first; the editor inserts `<img src="https://cdn.metaldocs…">` referring
  to the returned URL. This keeps `content_html` text-only and cacheable.
- Asset upload is multipart:
  ```http
  POST /assets
  Content-Type: multipart/form-data

  file=<binary>
  ```
  Response: `{ "url": "https://cdn…/abc.png", "bytes": 41230, "mime":
  "image/png" }`. The asset row is owned by the uploading user + org; a
  garbage-collection sweep removes assets not referenced by any document
  after 30 days (⚠ uncertain — retention window pending product input).
- `manifest` and `field_values` are JSONB for native GIN indexing.

---

## 12. Backward compatibility

- **Breaking HTML schema changes bump `data-mddm-schema`** (v1 → v2) and
  ship with a migration under `migrations/html/v1-to-v2.ts`.
- **Migration job**: on deploy, a batch worker walks `documents` and
  `templates` with older schemas and rewrites them in place. Idempotent;
  safe to rerun.
- **Never silent**: any HTML change that drops or renames a
  `data-mddm-*` attribute requires an explicit migration. The ingress
  sanitizer rejects unknown attributes, so a missing migration manifests
  as `422` rather than silent data loss.
- **API versioning**: REST routes live under `/api/v1/…`. A v2 prefix is
  introduced only when payload shapes change incompatibly — the schema
  attribute handles HTML evolution without touching the REST surface.

---

## Open questions

- ⚠ uncertain — backend framework (Nest vs Fastify vs Python sidecar).
  Routes and middleware hooks adjust per choice; JSON shapes do not.
- ⚠ uncertain — sync/async export threshold. Needs a benchmark on
  realistic template sizes before fixing a number.
- ⚠ uncertain — whether templates need their own `lock_version` or
  whether the `draft`-only edit window makes it unnecessary.
- ⚠ uncertain — asset retention window and GC policy.
- ⚠ uncertain — whether `manifest` belongs in the template row or in a
  sidecar `template_manifests` table for easier schema evolution.

---

## Sources & cross-refs

- [24 — Data format](./24-data-format.md)
- [25 — MDDM IR bridge](./25-mddm-ir-bridge.md)
- [26 — Autosave & persistence](./26-autosave-persistence.md)
- [28 — Template authoring](./28-template-authoring.md)
- [29 — Template instantiation](./29-template-instantiation.md)
- [31 — DOCX export](./31-docx-export.md)
- [32 — PDF export](./32-pdf-export.md)
- [37 — Validation](./37-validation.md)
