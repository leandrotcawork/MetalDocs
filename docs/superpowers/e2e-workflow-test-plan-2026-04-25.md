# E2E Workflow Test Plan — Full User Journey
**Date:** 2026-04-25 (updated post zone-purge + token-migration)
**Updated:** 2026-04-26 (post placeholder-fixed-catalog migration) | 2026-04-26 (Stage 10 live testing complete — 5 pipeline bugs fixed) | 2026-04-26 (Stage 10b PDF pipeline investigation — BLOCKED, architecture decision needed)
**Scope:** Validate full ISO flow — template authoring with fixed catalog tokens, auto-resolution, approval, fanout substitution, view.
**Method:** Manual UI simulation via preview tools (preview_click, preview_fill, preview_snapshot).
DB queries used ONLY for analysis/confirmation after UI actions — never as primary driver.

---

## Architecture note (2026-04-26)

- Placeholders are now a **fixed catalog of 7 computed tokens** — no user-fill types (text/date/number/select).
- Template author types `{token}` in the DOCX body; the catalog panel auto-detects and saves as `computed`.
- **`applyVariables` is NOT called at document open** — tokens stay literal in the editor. Backend substitutes at finalize/freeze via the fanout pipeline.
- No fill-in panel in the document editor. Stage 8 (fill-in flow) is replaced by a single no-panel check.

---

## Test Template Design

This template exercises all 7 fixed catalog tokens (all computed, all auto-resolved).

| Token | Resolver | Resolved value |
|-------|----------|----------------|
| `{doc_code}` | `doc_code` | CD code (e.g. `E2E-TEST-03`) |
| `{doc_title}` | `doc_title` | Document name |
| `{revision_number}` | `revision_number` | Revision number (e.g. `1`) |
| `{author}` | `author` | Creator display name |
| `{effective_date}` | `effective_date` | Creation date (or approval date post-publish) |
| `{approvers}` | `approvers` | `"[aguardando aprovação]"` until approved, then names |
| `{controlled_by_area}` | `controlled_by_area` | Area name |

All 7 tokens must appear in the DOCX body. Fanout must substitute all 7 correctly.

---

## Pre-flight

| Check | How | Pass |
|-------|-----|------|
| API up at :8081 | `preview_network` baseline | No connection errors |
| Web up at :4174 | `preview_start` / `preview_snapshot` | App renders |
| Login | `preview_fill` email+password, `preview_click` submit | Redirect to dashboard |
| DevTools baseline | `preview_console_logs` | 0 errors before any action |

---

## Stage 1 — Create Template

**Goal:** New template exists in DB, tenant correct, status=draft.

| # | UI Action | Tool | Expected |
|---|-----------|------|----------|
| 1.1 | Navigate to `/templates-v2` | `preview_eval window.location.hash = '#/templates-v2'` | List page renders |
| 1.2 | `preview_snapshot` | snapshot | No console errors, list visible |
| 1.3 | Click "New Template" button | `preview_click` | Dialog opens |
| 1.4 | `preview_snapshot` | snapshot | Key, Name, DocTypeCode, Visibility, ApproverRole inputs present |
| 1.5 | Fill key=`e2e-full-v1` | `preview_fill` | Input updated |
| 1.6 | Fill name=`E2E Full Test Template v1` | `preview_fill` | Input updated |
| 1.7 | Select ApproverRole=`admin` | `preview_click` select option | Dropdown value set |
| 1.8 | Click "Create Template" | `preview_click` | `POST /api/v2/templates → 201` |
| 1.9 | `preview_network` | network log | 201, response body has `id` |
| 1.10 | `preview_snapshot` | snapshot | Redirected to author page OR list shows new entry |

**DB Analysis (after UI):**
```sql
SELECT id, tenant_id, key, status, created_by
FROM templates_v2_template WHERE key = 'e2e-full-v1';
-- Expect: tenant_id = ffffffff-..., status = draft, created_by = e2e-admin
```

**Gate:** Capture template ID.

---

## Stage 2 — Author Template (Catalog Panel + Token Insertion)

**Goal:** All 7 catalog tokens typed into built-in ProseMirror editor, catalog panel auto-detects all 7, schema saved as 7 computed placeholders, persists on reload.

**Note (2026-04-26):** Template authoring uses the built-in ProseMirror editor — no DOCX upload. Authors type `{token_name}` directly in the canvas. DOCX is only generated at freeze/finalize time by docgen-v2 on the backend.

### 2A — Editor loads

| # | UI Action | Tool | Expected |
|---|-----------|------|----------|
| 2.1 | Template author page loads after create | `preview_snapshot` | Editor renders, left rail visible, catalog panel open |
| 2.2 | `preview_console_logs` | logs | 0 errors on load |

### 2B — Type tokens → catalog auto-detect

| # | UI Action | Tool | Expected |
|---|-----------|------|----------|
| 2.3 | Click into ProseMirror canvas, type all 7 tokens one per line: `{doc_code}`, `{doc_title}`, `{revision_number}`, `{author}`, `{effective_date}`, `{approvers}`, `{controlled_by_area}` | `preview_eval execCommand / ProseMirror API` | Text appears in editor |
| 2.4 | `preview_snapshot` | snapshot | Canvas shows all 7 literal tokens |
| 2.5 | Verify catalog panel visible (left rail) | `preview_snapshot` | "Placeholders disponíveis" panel shows 7 entries |
| 2.6 | `preview_snapshot` | snapshot | 7 catalog entries visible (doc_code, doc_title, revision_number, author, effective_date, approvers, controlled_by_area) |
| 2.7 | Verify no "+ Add manually" button | `preview_eval` | Button absent |
| 2.8 | Verify all 7 entries `data-detected="true"` | `preview_eval` | All 7 auto-detected |
| 2.10 | `preview_network` | network log | `PUT /schema → 200` after editor change |

### 2C — Verify persistence

| # | UI Action | Tool | Expected |
|---|-----------|------|----------|
| 2.11 | Reload page | `preview_eval window.location.reload()` | — |
| 2.12 | `preview_snapshot` | snapshot | Catalog panel still shows 7 detected entries |
| 2.13 | `preview_network` | network log | `GET /api/v2/templates/{id}/versions/1 → 200` |
| 2.14 | `preview_console_logs` | logs | 0 errors |

**DB Analysis (after UI):**
```sql
-- Schema is S3-stored. Verify via API:
-- GET /api/v2/templates/{id}/versions/1 → placeholders array
-- Expect: 7 items, all type=computed, resolverKey=name
```

**Gate:** Schema has 7 computed placeholders matching catalog.

---

## Stage 3 — Submit + Approve Template

**Goal:** Template status = approved, approved_by = e2e-admin.

| # | UI Action | Tool | Expected |
|---|-----------|------|----------|
| 3.1 | Click "Submit for Review" | `preview_click` | `POST /submit → 200` |
| 3.2 | `preview_network` | network log | 200, status=in_review |
| 3.3 | `preview_snapshot` | snapshot | UI reflects in_review state |
| 3.4 | Click "Approve" | `preview_click` | `POST /approve → 200` |
| 3.5 | `preview_network` | network log | 200, status=approved |
| 3.6 | `preview_snapshot` | snapshot | UI reflects approved state |
| 3.7 | `preview_console_logs` | logs | 0 errors |

**DB Analysis (after UI):**
```sql
SELECT status, approved_by, approved_at
FROM templates_v2_template_version WHERE template_id = '{id}' AND version_number = 1;
-- Expect: status = approved, approved_by = e2e-admin
```

---

## Stage 4 — Create Document Profile

**Goal:** Profile `e2e-full` exists, bound to approved template v1.

| # | UI Action | Tool | Expected |
|---|-----------|------|----------|
| 4.1 | Navigate to `/taxonomy/profiles` | `preview_eval` | Profile list page |
| 4.2 | `preview_snapshot` | snapshot | List visible |
| 4.3 | Click "New Profile" | `preview_click` | Dialog opens |
| 4.4 | Fill code=`e2e-full`, name=`E2E Full Profile` | `preview_fill` | Inputs updated |
| 4.5 | Submit | `preview_click` | `POST → 201` |
| 4.6 | `preview_network` | network log | 201, profile ID in response |
| 4.7 | Click Edit on new profile | `preview_click` | Edit dialog opens |
| 4.8 | Bind default template = template from Stage 1 | `preview_click` / `preview_fill` | Template version ID set |
| 4.9 | Save | `preview_click` | `PUT → 200` |
| 4.10 | `preview_network` | network log | 200, default_template_version_id updated |
| 4.11 | `preview_snapshot` | snapshot | Profile card shows template binding |

**DB Analysis (after UI):**
```sql
SELECT code, default_template_version_id
FROM metaldocs.document_profiles WHERE code = 'e2e-full';
-- Expect: default_template_version_id = {version_id from Stage 1}
```

---

## Stage 5 — Create Controlled Document Code

**Goal:** Controlled document `E2E-XXXX` exists, status=active, bound to `e2e-full`.

| # | UI Action | Tool | Expected |
|---|-----------|------|----------|
| 5.1 | Navigate to controlled documents | `preview_eval` | List page |
| 5.2 | Click "New" controlled document | `preview_click` | Dialog opens |
| 5.3 | Select profile `e2e-full` | `preview_click` | Profile selected |
| 5.4 | Submit | `preview_click` | `POST → 201` |
| 5.5 | `preview_network` | network log | 201, code = `E2E-XXXX` auto-generated |
| 5.6 | `preview_snapshot` | snapshot | New CD appears in list |

**DB Analysis (after UI):**
```sql
SELECT code, status, profile_code
FROM public.controlled_documents WHERE profile_code = 'e2e-full';
-- Expect: code auto-generated, status = active
```

**Gate:** Capture CD code for Stage 6.

---

## Stage 6 — Create Document from Template

**Goal:** Document instance created, status=draft, bound to template version with 8-placeholder schema snapshot.

| # | UI Action | Tool | Expected |
|---|-----------|------|----------|
| 6.1 | Navigate to `/documents-v2` | `preview_eval` | Documents list |
| 6.2 | Click "New Document" | `preview_click` | Wizard Step 1 |
| 6.3 | Select CD `E2E-XXXX` | `preview_click` | Step 2 unlocked |
| 6.4 | Fill name=`E2E Full Test Document v1` | `preview_fill` | Input updated |
| 6.5 | Click Create | `preview_click` | `POST /api/v2/documents → 201` |
| 6.6 | `preview_network` | network log | 201, document ID in response |
| 6.7 | `preview_snapshot` | snapshot | Redirected to `/documents-v2/{docID}` editor |
| 6.8 | `preview_console_logs` | logs | 0 errors |

**DB Analysis (after UI):**
```sql
SELECT id, template_version_id, name, status, created_by
FROM public.documents WHERE name = 'E2E Full Test Document v1';
-- Expect: status=draft, template_version_id={v1 id}
```

**Gate:** Capture document ID.

---

## Stage 7 — Document Editor: Plugins + Metadata Badges

**Goal:** Code chip + StateBadge render in title bar, OutlinePlugin visible.

| # | UI Action | Tool | Expected |
|---|-----------|------|----------|
| 7.1 | Document editor loaded | `preview_snapshot` | Full editor renders |
| 7.2 | **Code chip** — title bar | `preview_snapshot` | `E2E-XXXX` gray pill left of Checkpoints btn |
| 7.3 | **StateBadge** — title bar | `preview_snapshot` | "Rascunho" gray badge visible |
| 7.4 | `preview_inspect` code chip | inspect | font-size:11, background:#f1f5f9, border present |
| 7.5 | **OutlinePlugin** — left panel | `preview_snapshot` | Headings from template DOCX visible |
| 7.6 | Click outline heading | `preview_click` | Editor scrolls to heading |
| 7.7 | `preview_console_logs` | logs | 0 errors |

---

## Stage 8 — No Fill-In Panel (catalog model)

**Goal:** Document editor has NO fill-in panel. All tokens are computed; no user input required. Tokens stay literal `{token}` in the editor — backend substitutes at finalize/freeze.

| # | UI Action | Tool | Expected |
|---|-----------|------|----------|
| 8.1 | Document editor loaded | `preview_snapshot` | NO `<aside>` fill panel on right side |
| 8.2 | `preview_snapshot` | snapshot | Editor is full-width; no PlaceholderForm inputs visible |
| 8.3 | `preview_network` | network log | NO `GET /placeholders` request fired on load |
| 8.4 | Tokens visible literal in canvas | `preview_snapshot` | `{doc_code}`, `{doc_title}` etc displayed as text, not substituted |
| 8.5 | `preview_console_logs` | logs | 0 errors |

**Note:** No DB analysis needed — there are no user-fill placeholder value rows.

---

## Stage 9 — Approval + Freeze (Fanout)

**Goal:** Document approved → freeze runs → docgen-v2 substitutes all 7 catalog tokens → DOCX uploaded to S3.

**Fanout substitution map must contain:**
- `doc_code` → resolved from CD code (e.g. `"E2E-TEST-03"`)
- `doc_title` → `"E2E Catalog Test"` (document name)
- `revision_number` → `"1"`
- `author` → e2e-admin display name
- `effective_date` → creation date (or approval date post-publish)
- `approvers` → `"[aguardando aprovação]"` (pre-approval) or approver names after sign
- `controlled_by_area` → area name (e.g. `"General"`)

| # | UI Action | Tool | Expected |
|---|-----------|------|----------|
| 9.1 | Click Finalize / Submit for approval | `preview_click` | `POST /submit → 200` |
| 9.2 | `preview_network` | network log | 200, status=in_review |
| 9.3 | `preview_snapshot` | snapshot | StateBadge → "Em Revisão" |
| 9.4 | Click Approve | `preview_click` | `POST /approve → 200` |
| 9.5 | `preview_network` | network log | 200, status=approved |
| 9.6 | `preview_snapshot` | snapshot | StateBadge → "Aprovado" |
| 9.7 | `preview_console_logs` | logs | 0 errors during freeze |

**DB Analysis (after UI):**
```sql
SELECT status, content_hash, values_hash, schema_hash, final_docx_s3_key
FROM public.documents WHERE id = '{docID}';
-- Expect: status=approved, all hashes non-null, final_docx_s3_key set
```

**Note:** If fanout fails (docgen-v2 down or token mismatch), `preview_network` shows 500 on approve. Do NOT proceed to Stage 10b.

---

## Stage 10 — Signoff → Freeze Pipeline (Backend E2E)

**Date tested:** 2026-04-26
**Status:** COMPLETE — all 5 bugs found and fixed.

**Scope:** This section documents live API testing of the full signoff → auto-freeze pipeline (the backend pipeline that runs before the View step). Testing used document `088636b8-b924-4d7e-ba58-6ee65d3be1db` (tenant `ffffffff-...`) which was already under review.

### Pipeline steps (what happens on signoff)

1. `POST /api/v2/documents/{id}/signoff` receives `decision`, `password`, `content_hash`
2. Backend verifies password and content hash
3. Document status set to `approved`
4. Freeze triggered: `values_frozen_at` stamped, `schema_hash`/`values_hash` recorded
5. Fanout call dispatched to docgen-v2 with substitution map (all 7 catalog tokens resolved)
6. docgen-v2 fetches template DOCX from `metaldocs-attachments` bucket, applies substitutions, uploads frozen DOCX to `metaldocs-docx-v2` bucket
7. `final_docx_s3_key` written to document row

**API call used:**
```
POST /api/v2/documents/{id}/signoff
{ "decision": "approve", "password": "test1234", "content_hash": "5fbc68..." }
```

### Bugs found during live test

**Bug 1 — `area_code` → `process_area_code_snapshot` column name mismatch**
- 5 Go files queried `area_code` column from the `documents` table; actual column name is `process_area_code_snapshot`.
- Fixed in: `context_builder.go`, `fillin_authz.go`, `view_service.go`, `cancel_service.go`, `obsolete_service.go`

**Bug 2 — Placeholder schema JSON format mismatch**
- eigenpal stores placeholder schema as a raw JSON array `[{...}]` but Go parsers expected a wrapped object `{"placeholders":[...]}`.
- Fixed by adding a `parsePlaceholderSchema()` helper in `fillin_service.go` that tries the raw array format first and falls back to the wrapped format. `snapshot_service.go` updated to delegate to the same helper.

**Bug 3 — `composition_config` required fields in docgen-v2 fanout route**
- docgen-v2 fanout route required `header_sub_blocks`, `footer_sub_blocks`, and `sub_block_params` even when empty, causing validation errors on minimal payloads.
- Fixed in `apps/docgen-v2/src/routes/fanout.ts`: made all three fields optional with defaults `[]` / `{}`.

**Bug 4 — Template DOCX missing from `metaldocs-docx-v2` bucket**
- The `metaldocs-docx-v2` bucket was empty; the source DOCX lives in `metaldocs-attachments`.
- Root cause: local dev setup does not copy the template DOCX automatically.
- **Dev setup requirement:** manually copy the template DOCX from `metaldocs-attachments/templates/{id}/versions/` into `metaldocs-docx-v2/templates/{id}/versions/` before running the freeze pipeline locally.

**Bug 5 — Fanout error body swallowed**
- `fanout/client.go` only surfaced the HTTP status code in errors; the response body (which contains the actual error detail) was discarded.
- Fixed: `fanout/client.go` now includes the full response body in the returned error.

### Verification result

After all 5 fixes:
- Signoff returned `{"outcome":"approved"}`
- DB: `status=approved`, `final_docx_s3_key=tenants/.../frozen.docx`, `values_frozen_at` stamped (non-null)
- MinIO: `frozen.docx` exists in `metaldocs-docx-v2` bucket

**DB confirmation query:**
```sql
SELECT status, final_docx_s3_key, values_frozen_at
FROM public.documents WHERE id = '088636b8-b924-4d7e-ba58-6ee65d3be1db';
-- Result: status=approved, final_docx_s3_key set, values_frozen_at non-null
```

---

## Stage 10b — View (PDF)

**Status: READY TO TEST — pipeline fully wired as of 2026-04-26**

**Goal:** After signoff, worker generates PDF → `GET /api/v2/documents/{id}/view` returns presigned PDF URL. PDF must contain all 7 substituted catalog tokens — no raw `{token}` strings remaining.

### Architecture (implemented)

Outbox + worker pattern. No webhook — docgen-v2 `/convert/pdf` is synchronous; worker writes result directly.

1. Signoff → `DecisionService.RecordSignoff` calls `pdfDispatchAdapter.Dispatch(ctx, tenantID, revisionID)` post-commit
2. `PDFDispatchAdapter` reads `final_docx_s3_key` from DB, calls `PDFDispatcher.Dispatch`
3. `PDFDispatcher` publishes `docgen_v2_pdf` event to `messaging_outbox` table
4. Worker polls outbox → `PDFJobRunner.Handle` picks up event
5. `PDFJobRunner` calls docgen-v2 `/convert/pdf` synchronously with `final_docx_s3_key`
6. docgen-v2 converts DOCX → PDF, uploads to MinIO at `tenants/{id}/revisions/{id}/final.pdf`, returns `OutputKey` + `ContentHash`
7. `PDFJobRunner` calls `WritePDF` → stamps `final_pdf_s3_key` + `pdf_content_hash` on document row
8. `GET /api/v2/documents/{id}/view` reads `final_pdf_s3_key`, returns presigned URL

### Zod fix (committed 2026-04-26)

`render_opts` in docgen-v2 `/convert/pdf` Zod schema was required; made optional. Go client uses `omitempty` so this was causing 400. Fixed in `apps/docgen-v2/src/routes/convert-pdf.ts`.

### Pre-requisites

| # | Prerequisite | Status |
|---|---|---|
| 1 | API wired with PDFDispatchAdapter | ✅ cdaf7625 |
| 2 | Worker PDFJobRunner wired | ✅ cdaf7625 |
| 3 | PDFDispatchAdapter bridging interface mismatch | ✅ 00c8f24a |
| 4 | Worker routes docgen_v2_pdf events | ✅ bfbdbbbb |
| 5 | Zod render_opts optional in /convert/pdf | ✅ 79adcd3c |
| 6 | Worker binary running alongside API | ⚠️ must start separately |

### Local dev setup requirement

Worker is a separate binary. Must run both:
```powershell
.\scripts\start-api.ps1      # API on :8081
.\scripts\start-worker.ps1   # Worker (if script exists) OR go run ./apps/worker/...
```

### Test steps

| # | Action | Tool | Expected |
|---|--------|------|----------|
| 10.1 | Complete Stage 9 (signoff) | — | `final_docx_s3_key` set in DB |
| 10.2 | Wait for worker to drain outbox (poll interval ~10s) | wait | — |
| 10.3 | Check worker log | worker stdout | `worker_event event_type=docgen_v2_pdf result=published` |
| 10.4 | DB: `final_pdf_s3_key` set | SQL | non-null |
| 10.5 | Click View / PDF button | `preview_click` | `GET /api/v2/documents/{id}/view → 200` |
| 10.6 | `preview_network` | network log | 200, `signed_url` in response |
| 10.7 | Open signed URL | browser | PDF renders with all 7 token values substituted — no `{doc_code}` raw strings |

**DB verification:**
```sql
SELECT final_pdf_s3_key, pdf_content_hash
FROM public.documents WHERE id = '{docID}';
-- Expect: both non-null after worker processes event
```

---

## Stage 11 — Regression Checks

**Goal:** Post-migration runtime clean — no zone refs, no `{{uuid}}` tokens anywhere.

| # | Check | Method | Expected |
|---|-------|--------|----------|
| 11.1 | App boots without errors | `preview_console_logs` fresh load | 0 errors |
| 11.2 | Template list loads | `preview_snapshot` `/templates-v2` | No panics, no zone-related errors |
| 11.3 | Document list loads | `preview_snapshot` `/documents-v2` | No errors |
| 11.4 | `go build ./...` | Bash | Clean |
| 11.5 | `pnpm tsc --noEmit` (web) | Bash | 0 errors |
| 11.6 | No `{{uuid}}` in DOCX body | Download frozen DOCX, grep body XML | 0 matches |
| 11.7 | No `{raw_token}` unreplaced vars in fanout response | `preview_network` approve response | `unreplaced_vars: []` |
| 11.8 | `preview_console_logs` after full workflow | logs | 0 unhandled errors total |

---

## Failure Protocol

On any RED step:
1. `preview_screenshot` + `preview_console_logs` + `preview_network` — capture full state
2. Stop progression
3. Root cause investigate (systematic-debugging skill)
4. Fix. Re-verify failed step passes
5. Resume from next step (do NOT restart from Stage 1 unless state is corrupted)

---

## Success Criteria

- All 11 stages green
- 0 console errors at end of each stage
- No 4xx/5xx on happy-path network calls
- Frozen DOCX contains all 7 substituted catalog values (no raw `{token}` strings)
- `unreplaced_vars: []` in fanout response
- `go build ./...` + `pnpm tsc --noEmit` clean
- StateBadge + code chip visible in document editor title bar
- No fill-in panel in document editor
- Catalog panel in template author shows 7 entries with correct auto-detect
