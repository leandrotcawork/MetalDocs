# CK5 Plan C — Production Readiness

**Date:** 2026-04-16
**Status:** approved
**Follows:** Plan A (frontend UI), Plan B (backend persistence)
**Goal:** Make CK5 the sole production document engine. Ship DOCX/PDF export, template publish flow, and delete the legacy BlockNote/MDDM stack.

---

## Decisions

| Question | Answer |
|---|---|
| Existing DB data | Clean slate — all test data, no migration needed |
| Template publish | Manual gate: draft → pending_review → published |
| BlockNote deletion | Full delete, single PR |
| Export runtime | Server-side primary, client-side `window.print()` for PDF preview |
| Export service | Standalone HTTP service (`apps/ck5-export/`) — replaces `apps/docgen/` |
| Commercial licenses | Zero — all GPL/MIT/Apache-2 |

---

## Architecture Overview

```
CK5 Author/Fill Page (React)
[Export DOCX] [Export PDF] [Print Preview] [Publish Template]
       │
       │ fetch /api/v1/documents/{id}/export/ck5/{docx|pdf}
       ▼
Go API (auth boundary)
  GET /documents/{id}/export/ck5/docx
  GET /documents/{id}/export/ck5/pdf
  POST /templates/{key}/publish
  POST /templates/{key}/approve
       │                    │
       ▼                    ▼
apps/ck5-export         Gotenberg (existing)
POST /render/docx       POST /forms/chromium/convert/html
POST /render/pdf-html
```

**Principles:**
- Go API = auth + routing only. Never exposes `ck5-export` directly to browser.
- `ck5-export` = internal-only service, no auth, trusted network.
- PDF reuses existing Gotenberg — zero new infrastructure.
- CK5 HTML is the single source of truth. IR is ephemeral on export only.

---

## PR1 — Additive (Export + UI)

### `apps/ck5-export/` — New Node.js HTTP Service

Replaces `apps/docgen/`. Single responsibility: transform CK5 HTML into export formats.

**Directory layout:**
```
apps/ck5-export/
  src/
    server.ts                     ← Hono, two routes
    export-node.ts                ← ExportNode type definitions
    html-to-export-tree.ts        ← linkedom DOM walk → ExportNode
    docx-emitter/                 ← MOVED from mddm-editor/engine/docx-emitter/
    asset-resolver/               ← MOVED from mddm-editor/engine/asset-resolver/
    print-stylesheet/             ← MOVED from mddm-editor/engine/print-stylesheet/
    inline-asset-rewriter.ts      ← MOVED from mddm-editor/engine/export/
  package.json                    ← docx, linkedom, hono
  tsconfig.json
```

**Routes:**
```
POST /render/docx
  body: { html: string }
  → htmlToExportTree(html)
  → collectImageUrls(tree)
  → AssetResolver.resolveAll(urls)
  → emitDocx(tree, assetMap) → Packer.toBuffer()
  → 200 + bytes
  Content-Type: application/vnd.openxmlformats-officedocument.wordprocessingml.document

POST /render/pdf-html
  body: { html: string }
  → inlineAssetRewriter(html)
  → wrapInPrintDocument(html)
  → 200 + wrapped HTML string
  Content-Type: text/html
```

**`html-to-export-tree.ts` — new file:**

Thin linkedom DOM walk mapping CK5 HTML shapes to `ExportNode`. Shape mapping:

| CK5 HTML | ExportNode |
|---|---|
| `<section class="mddm-section">` | `section` with variant, header, body |
| `<ol class="mddm-repeatable">` | `repeatable`, `items[]` |
| `<li>` inside repeatable | `repeatableItem` |
| `<figure class="table"><table>` | `table` (flat, variant=fixed\|dynamic) |
| `<tr>`, `<th>`, `<td>` | `tableRow`, `tableCell` |
| `<span class="mddm-field">` | `field` with id, type, value |
| `<div class="mddm-rich-block">` | unwrap — emit children |
| `<span class="restricted-editing-exception">` | unwrap |
| `<h1>`–`<h6>` | `heading` with level |
| `<p>` | `paragraph` with optional align |
| `<ul>` / `<ol>` / `<li>` | `list`, `listItem` |
| `<img>` | `image` with src, alt, width, height |
| `<a>` | `hyperlink` |
| `<strong>`, `<em>`, `<u>`, `<s>` | inline marks on `text` |
| `<br>` | `lineBreak` |
| `<blockquote>` | `blockquote` |

### Go — DOCX Export Handler

```
GET /api/v1/documents/{id}/export/ck5/docx
│
├─ auth: isAllowed(CapabilityDocumentView) → 404 if denied
├─ fetch latest version where content_source = 'ck5_browser' → 404 if none
├─ POST ck5-export:PORT/render/docx  { html: version.Content }  timeout: 30s
├─ stream response bytes
│  Content-Type: application/vnd.openxml...
│  Content-Disposition: attachment; filename="{title}.docx"
└─ error mapping:
   ck5-export 400 → 422
   ck5-export 5xx / timeout → 502
```

### Go — PDF Export Handler

```
GET /api/v1/documents/{id}/export/ck5/pdf
│
├─ auth: isAllowed(CapabilityDocumentView) → 404 if denied
├─ fetch latest version where content_source = 'ck5_browser' → 404 if none
├─ POST ck5-export:PORT/render/pdf-html  { html: version.Content }
├─ POST Gotenberg /forms/chromium/convert/html
│   multipart: index.html (wrapped), print.css (static)
│   paperWidth: 8.27, paperHeight: 11.69, preferCSSPageSize: true
├─ stream PDF bytes
│  Content-Type: application/pdf
│  Content-Disposition: attachment; filename="{title}.pdf"
└─ error mapping: same as DOCX
```

### Frontend — Export UI

**Files created:**
```
ck5/react/components/ExportMenu.tsx         ← DOCX + PDF + Print Preview buttons
ck5/react/components/PublishButton.tsx      ← draft | pending_review | published states
ck5/persistence/exportApi.ts               ← triggerExport(docId, fmt) + clientPrint(editor)
ck5/persistence/templatePublishApi.ts      ← publishTemplate(key) + approveTemplate(key)
```

**Files modified:**
```
ck5/react/AuthorPage.tsx     ← add ExportMenu + PublishButton (template mode only)
ck5/react/FillPage.tsx       ← add ExportMenu
```

**`triggerExport`:**
```ts
async function triggerExport(docId: string, fmt: 'docx' | 'pdf') {
  const res = await fetch(`/api/v1/documents/${docId}/export/ck5/${fmt}`, {
    credentials: 'include',
  });
  if (!res.ok) throw new ExportError(res.status);
  const blob = await res.blob();
  const url = URL.createObjectURL(blob);
  const a = document.createElement('a');
  a.href = url; a.download = `document.${fmt}`; a.click();
  URL.revokeObjectURL(url);
}
```

**`clientPrint` (PDF preview, no server):**
```ts
function clientPrint(editor: DecoupledEditor) {
  const html = editor.getData();
  const iframe = document.createElement('iframe');
  iframe.style.cssText = 'position:fixed;inset:0;width:0;height:0;border:0';
  document.body.appendChild(iframe);
  iframe.contentDocument!.write(wrapInPrintDocument(html));
  iframe.contentDocument!.close();
  iframe.contentWindow!.print();
}
```

**PublishButton states:**
```
status=draft          → "Publish for Review" button (primary)
status=pending_review → "Awaiting Approval" badge + "Approve" button (admin only)
status=published      → "Published" badge (no action)
```

---

## PR2 — Destructive (Template Publish + Delete Legacy)

### Template Publish Flow

**DB migration:**
```sql
ALTER TABLE template_drafts
  ADD COLUMN published_html TEXT,
  ADD COLUMN status TEXT NOT NULL DEFAULT 'draft'
    CHECK (status IN ('draft', 'pending_review', 'published'));
```

No data migration required — existing rows get `status='draft'`, `published_html=NULL`.

**State machine:**
```
draft → pending_review (POST /publish)
pending_review → published (POST /approve)
```

**Go endpoints:**
```
POST /api/v1/templates/{key}/publish
├─ auth: isAllowedTemplate(CapabilityTemplateEdit)
├─ fetch TemplateDraft → read blocks_json._ck5.contentHtml
├─ validate: contentHtml not empty → 400 if empty
├─ assert status = 'draft' → 409 if already pending/published
├─ update status → 'pending_review'
└─ 200 OK

POST /api/v1/templates/{key}/approve
├─ auth: isAllowedTemplate(CapabilityTemplatePublish)
├─ fetch template, assert status = 'pending_review' → 409 if not
├─ update status → 'published'
├─ copy blocks_json._ck5.contentHtml → published_html column
└─ 200 OK
```

**Fill mode uses published snapshot:**
```
GET /api/v1/templates/{key}/ck5-draft
  if status = 'published' → return published_html (frozen)
  if status = 'draft' | 'pending_review' → return draft contentHtml (author preview only)
```

### Deletion Scope

**Delete entirely:**
```
frontend/apps/web/src/features/documents/mddm-editor/
apps/docgen/
```

**Delete from Go (`handler.go`, related service files):**
```
handleDocumentExportDocx
handleDocumentContentDocx
handleDocumentTemplateDocx
handleDocumentContentRenderPDF
handleDocumentContentBrowserPost
DocgenClient + docgen URL config
```

**Delete from `launch.json`:**
```
metaldocs-docgen entry → replace with ck5-export entry
```

**Delete from `package.json`:**
```
@blocknote/* all packages
```

**Keep:**
```
frontend/apps/web/src/features/documents/browser-editor/  ← separate, unrelated
frontend/apps/web/src/features/documents/ck5/             ← new stack
apps/ck5-export/                                          ← new, built in PR1
Go: handleDocumentContentNativeGet/Post                   ← keep
Go: handleDocumentContentBrowserGet                       ← keep (read path)
Go: /export/ck5/docx + /export/ck5/pdf                   ← new, PR1
Go: /templates/{key}/publish + /approve                  ← new, PR2
```

**Pre-delete verification (must pass before PR2 merges):**
```bash
grep -r "from.*mddm-editor" src/    # 0 results
go build ./...                       # clean
rtk vitest run                       # all pass
rtk go test ./...                    # all pass
rtk pnpm build                       # clean, zero @blocknote in bundle
```

---

## Testing

### `apps/ck5-export/` (Node)
```
__tests__/
  html-to-export-tree.test.ts   ← unit: HTML fixtures → ExportNode snapshots
  docx-emitter.test.ts          ← goldens re-anchored: CK5 HTML → docx AST
  server.test.ts                ← integration: POST /render/docx → valid .docx bytes

__fixtures__/
  section-with-fields.html
  table-fixed.html
  repeatable.html
  rich-block.html

__goldens__/                    ← migrated from mddm-editor/engine/docx-emitter/__tests__/
```

**Golden migration strategy:**
1. For each existing golden: add equivalent CK5 HTML fixture
2. Run both old IR path + new ExportNode path — assert same docx AST
3. Once parity confirmed → delete IR path input
4. Retire IR-only goldens with no HTML equivalent — log reason in PR

### Go Backend
```
handler_ck5_export_test.go
  ← mock ck5-export + mock Gotenberg via httptest.NewServer
  ← assert 200 + correct Content-Type
  ← assert 404 when no ck5_browser version exists
  ← assert 502 on ck5-export timeout

handler_ck5_template_publish_test.go
  ← POST /publish → 200, status=pending_review
  ← POST /approve → 200, status=published, published_html set
  ← GET /ck5-draft on published → returns published_html
  ← 401 no auth, 403 insufficient role (publish=edit, approve=publish cap), 409 wrong status
```

### Frontend
```
exportApi.test.ts               ← vitest: fetch mock, blob + anchor trigger
templatePublishApi.test.ts      ← vitest: publish + approve fetch contracts
ExportMenu.test.tsx             ← render, button click triggers export
PublishButton.test.tsx          ← status prop → correct UI state per state
```

### Preview Full Workflow Validation

**Setup:**
```
preview_start ck5-plan-c-api  (port 8082)
preview_start ck5-plan-c-web  (port 4174)
VITE_CK5_PERSISTENCE=api in .env.local
```

**Author flow:**
```
1. navigate → #/test-harness/ck5?mode=author&tpl=sandbox
2. preview_snapshot → AuthorPage rendered, toolbar visible
3. window.__ck5.save() → preview_network → PUT /templates/sandbox/ck5-draft 200
4. click Export DOCX → preview_network → GET /export/ck5/docx 200
                        preview_network → POST ck5-export /render/docx 200
5. click Print Preview → no network call, window.print fires
6. click Publish → preview_network → POST /templates/sandbox/publish 200
7. preview_snapshot → PublishButton shows "Awaiting Approval"
```

**Fill flow:**
```
1. navigate → #/test-harness/ck5?mode=fill&tpl=sandbox&doc=sandbox-doc
2. preview_snapshot → FillPage rendered, fields editable
3. window.__ck5.save() → preview_network → POST /documents/sandbox-doc/content/ck5 201
4. click Export PDF → preview_network → GET /export/ck5/pdf 200
                      preview_network → Gotenberg called with HTML + print.css
```

**Template publish approval flow:**
```
1. login as admin
2. GET /api/v1/templates/sandbox → assert status=pending_review
3. POST /templates/sandbox/approve → 200
4. navigate fill mode → preview_network → GET /ck5-draft returns published_html
```

**Deletion smoke (PR2 only):**
```
5. preview_snapshot → zero mddm-editor components in DOM
6. rtk pnpm build → zero @blocknote/* in bundle output
7. preview_console_logs → zero errors referencing mddm-editor or docgen
```

**Pass criteria:**
```
All network assertions: correct status codes + endpoints
Zero console errors
Bundle contains zero @blocknote/* after PR2
Export buttons produce downloadable blob
Published template loads published_html in fill mode
```

---

## File Inventory Summary

### New files
```
apps/ck5-export/src/server.ts
apps/ck5-export/src/export-node.ts
apps/ck5-export/src/html-to-export-tree.ts
apps/ck5-export/src/html-to-export-tree.test.ts
apps/ck5-export/package.json
apps/ck5-export/tsconfig.json
internal/modules/documents/delivery/http/handler_ck5_export.go
internal/modules/documents/delivery/http/handler_ck5_export_test.go
internal/modules/documents/delivery/http/handler_ck5_template_publish.go
internal/modules/documents/delivery/http/handler_ck5_template_publish_test.go
internal/modules/documents/application/service_ck5_export.go
internal/modules/documents/application/service_ck5_template_publish.go
frontend/apps/web/src/features/documents/ck5/react/components/ExportMenu.tsx
frontend/apps/web/src/features/documents/ck5/react/components/PublishButton.tsx
frontend/apps/web/src/features/documents/ck5/persistence/exportApi.ts
frontend/apps/web/src/features/documents/ck5/persistence/templatePublishApi.ts
```

### Moved (mddm-editor → ck5-export)
```
mddm-editor/engine/docx-emitter/     → apps/ck5-export/src/docx-emitter/
mddm-editor/engine/asset-resolver/   → apps/ck5-export/src/asset-resolver/
mddm-editor/engine/print-stylesheet/ → apps/ck5-export/src/print-stylesheet/
mddm-editor/engine/export/inline-asset-rewriter.ts → apps/ck5-export/src/inline-asset-rewriter.ts
```

### Modified
```
internal/modules/documents/delivery/http/handler.go          ← wire new export routes
internal/modules/documents/delivery/http/template_admin_handler.go ← wire publish/approve
frontend/apps/web/src/features/documents/ck5/react/AuthorPage.tsx
frontend/apps/web/src/features/documents/ck5/react/FillPage.tsx
.claude/launch.json                                           ← add ck5-export, remove docgen
```

### Deleted (PR2)
```
frontend/apps/web/src/features/documents/mddm-editor/   (entire directory)
apps/docgen/                                             (entire directory)
```

### DB migration (PR2)
```sql
ALTER TABLE template_drafts
  ADD COLUMN published_html TEXT,
  ADD COLUMN status TEXT NOT NULL DEFAULT 'draft'
    CHECK (status IN ('draft', 'pending_review', 'published'));
```

---

## License Audit

| Dependency | License | Usage |
|---|---|---|
| CKEditor5 core | GPL-2.0+ | editor |
| docx npm | MIT | DOCX generation |
| linkedom | MIT | HTML parsing in ck5-export |
| Paged.js | MIT | PDF pagination polyfill |
| Gotenberg | Apache-2 | headless Chromium PDF |
| Hono | MIT | ck5-export HTTP server |
| All custom plugins | own code | — |

Zero commercial / premium CKEditor licenses.

---

## Amendments (post-ship)

### 2026-04-16 — Migration column name: `draft_status`

Spec §PR2 specified column name `status`. Shipped migration `0077_add_template_publish_state.sql` uses `draft_status` to avoid a naming collision with the existing `template_versions.status` column, which tracks a different state machine (draft / published / deprecated lifecycle vs the review workflow). Functional behavior identical.

### 2026-04-16 — IR golden retirement deferred, then obsoleted

Spec §Testing step 3 planned a dual-path golden-parity migration: run both the IR → docx and the HTML → docx paths against equivalent goldens, confirm parity, then retire the IR-only goldens. The dual-path phase was skipped during implementation; both paths shipped together without a parity check. The post-audit sweep (see below) deleted the IR path entirely, making the retirement moot. Any IR-only golden fixtures were removed as part of the `docx-emitter/` directory deletion and are not recoverable without restoring the legacy pipeline.

### 2026-04-16 — Post-audit dead-code sweep

An outsider audit found `apps/ck5-export/src/{docx-emitter,codecs,layout-interpreter}/` shipped but unreachable from any route. The `layout-ir/` directory framed design tokens as a persistent IR in violation of the "IR is ephemeral" principle. All three directories + `layout-ir/` were deleted in plan `docs/superpowers/plans/2026-04-16-ck5-plan-c-post-audit-fixes.md`. The surviving `ck5-docx-emitter.ts` god file was split into `docx-emitter/` (block-per-file) and `layout-ir/` was collapsed to a single `layout-tokens.ts`. Correctness bugs (missing block-exception button, field label not rendered, hyperlink color hardcoded) were fixed at the same time.
