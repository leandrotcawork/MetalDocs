# Workflow: Freeze and Fanout

> **Last verified:** 2026-04-27
> **Scope:** The full pipeline from signoff approval → computed value resolution → DOCX substitution → frozen artifact stored in S3 → async PDF generation via outbox worker.
> **Out of scope:** Approval routing and signoff rules (see `workflows/approval.md`), editor-side substitution deferral (see `modules/editor-ui-eigenpal.md`).
> **Key files:**
> - `internal/modules/documents_v2/approval/application/decision_service.go` — triggers freeze on signoff
> - `internal/modules/documents_v2/application/freeze_service.go` — FreezeService.Freeze orchestration
> - `internal/modules/documents_v2/application/context_builder.go` — builds resolver input context
> - `internal/modules/render/fanout/client.go` — HTTP client calling docgen-v2
> - `internal/modules/render/resolvers/builtins.go` — registered resolver implementations
> - `apps/docgen-v2/src/routes/fanout.ts` — docgen-v2 fanout route, Zod request schema
> - `apps/docgen-v2/src/render/fanout.ts` — eigenpal headless token substitution
> - `internal/modules/render/fanout/pdf_dispatcher.go` — PDFDispatcher: publishes docgen_v2_pdf outbox event
> - `internal/modules/render/fanout/pdf_dispatch_adapter.go` — PDFDispatchAdapter: bridges PDFDispatchInvoker interface
> - `internal/platform/worker/pdf_job_runner.go` — PDFJobRunner: handles docgen_v2_pdf events end-to-end

---

## Trigger

`RecordSignoff` in `decision_service.go` calls `s.freezeInvoker.Freeze(ctx, tx, ...)` when a signoff is recorded and the approval condition is satisfied (all required signoffs received). Freeze runs inside the same database transaction as the signoff write.

## Pipeline steps

### 1. Read snapshot

`FreezeService.Freeze` calls `ReadSnapshotWithFreezeAt` to load the document version row, including all snapshot columns populated at document creation time.

### 2. Idempotency check

If `values_frozen_at` is already set, `Freeze` returns early without error. Safe to retry — double-calling freeze on an already-frozen document is a no-op.

### 3. Load placeholder schema

The `placeholder_schema_snapshot` column is read and parsed by `LoadPlaceholderSchema`. This column stores the eigenpal-native format — see [Storage format](#storage-format) below.

### 4. Validate required placeholders

Any non-computed placeholder marked required is checked for a filled value. Computed placeholders skip this check — they are resolved in the next step.

### 5. Resolve computed placeholders

Each computed placeholder carries a `resolver_key` (e.g. `doc_code`, `approvers`). `FreezeService` looks up the resolver from `resolvers.Registry` and calls it. The context passed to each resolver is built by `context_builder.go`, which queries fields including `process_area_code_snapshot`.

Registered resolvers (`builtins.go`):

| Resolver key | Resolved value |
|---|---|
| `doc_code` | Document code from profile sequence counter |
| `doc_title` | Document title field |
| `revision_number` | Revision counter |
| `author` | Display name of document creator |
| `effective_date` | Effective date set during approval/freeze |
| `approvers` | Approver names joined by `", "`; falls back to `"[aguardando aprovação]"` if none |
| `controlled_by_area` | Area name from taxonomy binding |
| `approval_date` | Date the final approval signoff was recorded |

### 6. Write computed values

Resolved values are written to the database via `UpsertValue` — one row per token.

### 7. Compute values_hash

A hash is computed over all placeholder values (resolved + any pre-filled). Stored as `values_hash`. See `concepts/freeze-and-hashing.md`.

### 8. WriteFreeze

`WriteFreeze` stamps `values_frozen_at` on the document version row, making the freeze visible to readers.

### 9. Fanout to docgen-v2

`fanout.Client.Fanout` sends `POST {METALDOCS_FANOUT_URL}/render/fanout` with the revision ID, tenant ID, and the resolved `{name: value}` map.

`docgen-v2` (`apps/docgen-v2/src/routes/fanout.ts`) receives the request, validates it against a Zod schema, and delegates to `apps/docgen-v2/src/render/fanout.ts`.

### 10. Eigenpal headless substitution

`fanout.ts` in docgen-v2:
1. Loads the template body DOCX from S3 bucket `metaldocs-docx-v2` at key `templates/{templateID}/versions/{n}.docx`.
2. Calls eigenpal's headless substitution API, passing the `{name: value}` map (docxtemplater-compatible).
3. Uploads the substituted `frozen.docx` to `metaldocs-docx-v2` at key `tenants/{tenantID}/revisions/{revisionID}/frozen.docx`.

### 11. WriteFinalDocx

Back in `FreezeService`, `WriteFinalDocx` stamps `final_docx_s3_key` and `content_hash` on the document version row, completing the pipeline.

## S3 bucket mapping

| Object | Bucket | Key pattern |
|---|---|---|
| Template DOCX (source) | `metaldocs-docx-v2` | `templates/{templateID}/versions/{n}.docx` |
| Frozen DOCX (output) | `metaldocs-docx-v2` | `tenants/{tenantID}/revisions/{revisionID}/frozen.docx` |
| Attachments / uploads | `metaldocs-attachments` | (separate bucket — NOT used by freeze) |

## Storage format

`placeholder_schema_snapshot` in the `documents` table stores eigenpal-native JSON:

```json
[
  { "id": "...", "type": "computed", "resolver_key": "doc_code" },
  { "id": "...", "type": "computed", "resolver_key": "approvers" }
]
```

This is a **raw JSON array** — NOT wrapped as `{"placeholders": [...]}`. `parsePlaceholderSchema()` in `internal/modules/documents_v2/application/fillin_service.go` handles both formats for backward compatibility with legacy rows.

## Gotchas

- **Wrong S3 bucket in local dev:** docgen-v2 reads template DOCX from `metaldocs-docx-v2`, not `metaldocs-attachments`. Template DOCX must exist in `metaldocs-docx-v2` in the local MinIO instance.
- **`composition_config` defaults:** `header_sub_blocks`, `footer_sub_blocks`, and `sub_block_params` default to empty — templates without sub-blocks work fine without explicit values.
- **Freeze is idempotent:** `values_frozen_at` already set → early return, no duplicate writes, no error.
- **Freeze runs inside signoff transaction:** if the freeze step fails, the entire signoff is rolled back.

## PDF Generation Pipeline (Steps 12–16)

> **Status as of 2026-04-26:** Fully implemented and wired. PDF is generated asynchronously after freeze via outbox + worker pattern. No webhook — docgen-v2 `/convert/pdf` is synchronous; the worker writes the result directly.

### Flow

| Step | Component | What happens |
|---|---|---|
| 12 | `DecisionService.RecordSignoff` | Post-commit: calls `pdfDispatchAdapter.Dispatch(ctx, tenantID, revisionID)` |
| 13 | `PDFDispatchAdapter` | Reads `final_docx_s3_key` from DB, delegates to `PDFDispatcher.Dispatch` |
| 14 | `PDFDispatcher` | Publishes `docgen_v2_pdf` event to `messaging_outbox` table |
| 15 | Worker (`PDFJobRunner.Handle`) | Picks up event, calls docgen-v2 `/convert/pdf` synchronously |
| 16 | docgen-v2 | Converts DOCX→PDF, uploads to `tenants/{id}/revisions/{id}/final.pdf`, returns `OutputKey` + `ContentHash` |
| 17 | `PDFJobRunner.Handle` (cont.) | Calls `WritePDF` — stamps `final_pdf_s3_key` and `pdf_content_hash` on document row |

After step 17, `GET /api/v2/documents/{id}/view` can return a presigned URL for the PDF.

### S3 key pattern

| Object | Bucket | Key pattern |
|---|---|---|
| Frozen DOCX (input) | `metaldocs-attachments` (local) / `metaldocs-docx-v2` (prod) | `tenants/{tenantID}/revisions/{revisionID}/frozen.docx` |
| Final PDF (output) | same bucket | `tenants/{tenantID}/revisions/{revisionID}/final.pdf` |

### Zod fix (2026-04-26)

`apps/docgen-v2/src/routes/convert-pdf.ts`: `render_opts` was required in the Zod schema; Go client omits it via `omitempty` → 400. Made `render_opts` and its fields optional.

### Key files

- `apps/api/cmd/metaldocs-api/main.go` — wires `PDFDispatchAdapter` into `NewDecisionService`
- `internal/modules/render/fanout/pdf_dispatch_adapter.go` — `PDFDispatchAdapter`: reads `final_docx_s3_key` from DB, calls `PDFDispatcher`
- `internal/modules/render/fanout/pdf_dispatcher.go` — `PDFDispatcher`: publishes `docgen_v2_pdf` outbox event
- `internal/modules/documents_v2/approval/application/decision_service.go` — `PDFDispatchInvoker` interface + post-commit dispatch call
- `internal/platform/worker/pdf_job_runner.go` — `PDFJobRunner`: handles `docgen_v2_pdf` events end-to-end
- `internal/platform/worker/service.go` — routes `docgen_v2_pdf` to `PDFJobRunner`
- `internal/platform/bootstrap/worker.go` — builds `DocgenV2Client` + exposes `SQLDB` in `WorkerDependencies`
- `apps/worker/cmd/metaldocs-worker/main.go` — conditionally wires `PDFJobRunner` via `WithPDFRunner`
- `internal/modules/documents_v2/http/view_handler.go` — view endpoint, reads `final_pdf_s3_key`

---

## Cross-refs

- [concepts/placeholders.md](../concepts/placeholders.md) — fixed 7-token catalog, resolver keys
- [concepts/freeze-and-hashing.md](../concepts/freeze-and-hashing.md) — content_hash, values_hash, immutability
- [modules/render-fanout.md](../modules/render-fanout.md) — substitution engine internals
- [modules/editor-ui-eigenpal.md](../modules/editor-ui-eigenpal.md) — why substitution is deferred to freeze (not done in editor)
- [workflows/approval.md](approval.md) — signoff flow that triggers freeze
