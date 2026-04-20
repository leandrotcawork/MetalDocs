# Eigenpal migration — UAT run log (2026-04-19)

Walkthrough of all user workflows after CK5 → `@eigenpal/docx-js-editor` swap.
Tester: Claude (driving `preview_*`). Account: `leandro_theodoro` / admin.
Frontend: http://localhost:4174 · Backend: http://localhost:8081.

Rule: log mistakes as encountered, keep walking; stop to fix only when blocker prevents next step.

---

## Legend
- ✅ pass
- ⚠️ defect (non-blocker, walk continues)
- 🛑 blocker (stopped to fix)
- 🩹 fix applied mid-run

---

## 0. Bootstrap

🛑 **Vite fail on boot** — `Failed to resolve import "@eigenpal/docx-js-editor/styles.css" from packages/editor-ui/src/MetalDocsEditor.tsx`.
  - Cause: plain-object alias `'@eigenpal/docx-js-editor': <dir>` did not rewrite subpath `/styles.css` correctly.
  - 🩹 Fix: converted `vite.config.ts` `resolve.alias` to array form; added regex subpath rule mapping `/styles.css` → `dist/styles.css`.
  - Server restarted clean.

## 1. Auth + landing
- ✅ Session cookie persisted from prior login — landed directly on Dashboard as `Leandro Theodoro · Administrador`.
- ✅ Dashboard rendered: "25 Documentos Ativos", "0 Em Revisao", etc.
- ✅ Sidebar groups present: VISAO GERAL, DOCUMENTOS, POR TIPO, WORKSPACE, ADMIN, TEMPLATES V2.

⚠️ Pre-existing backend gaps (not caused by this sprint, but noise in every page load):
- `GET /api/v1/document-profiles` → 404
- `GET /api/v1/process-areas` → 404
- `GET /api/v1/document-departments` → 404
- `GET /api/v1/document-subjects` → 404
- `GET /api/v1/operations/stream` → 401

## 2. Templates V2
- ✅ List page loads, shows `Contrato Padrao (contrato-padrao) — v2`.
- ✅ "New template" button present.
- ✅ Opening template shows editor panel: DOCX URL, autosave idle, schema JSON, "Publish version 2" action.

## 3. Documents V2

⚠️ **Sidebar "Novo documento" routes to legacy `/create` wizard**, not `/documents-v2/new`. Legacy wizard relies on 404'd profile endpoints — broken user path.

⚠️ **No v2 entry point in sidebar**. User has no way to reach `DocumentCreatePage` without typing URL or coming from another flow.

⚠️ **Router footgun**: App uses `HashRouter` (main.tsx:3). All links must use `#/...` path. Several direct pushes to `window.location.pathname` didn't route. Not a bug per se, but easy trap for future dev who expects `BrowserRouter` (vite-config-style aliases suggest SPA).

✅ **API create works**: `POST /api/v2/documents` → 201, returned `document_id`, `initial_revision_id`, `session_id`.

✅ **Editor loads** at `#/documents-v2/{uuid}` — eigenpal `DocxEditor` mounts with full toolbar:
- File / Format / Insert / Checkpoints / Export / Download .docx / Export PDF / Finalize
- Merge field sidebar lists `{client_name}`, `{total_amount}`.

✅ **Autosave works end-to-end**. Typed " UAT TEST INSERTED" into ProseMirror. After ~1500ms debounce:
  1. `POST /api/v2/documents/{id}/autosave/presign` → 200
  2. `PUT` pre-signed URL to MinIO (`/revisions/{sha}.docx`) → 200
  3. `POST /api/v2/documents/{id}/autosave/commit` → 200
  4. Heartbeat continues (204).

⚠️ **MinIO PUT logs `[FAILED: net::ERR_ABORTED]` alongside 200**. Commit succeeds → data persisted. Likely fetch abort after body flush; harmless but noisy. Track.

⚠️ **`GET /api/v2/documents/{id}/comments` → 404** despite commit 141fd40 (P5.1 backend). Route not wired in handler.

⚠️ **External font fetch blocked** — eigenpal requests `fonts.googleapis.com/css2?family=Calibri%20Light` → `net::ERR_BLOCKED_BY_ORB`. `Carlito` fallback loads fine. Cosmetic.

## 4. Exports

🛑 **Export endpoints missing — feature broken end-to-end**.
  - `GET /api/v2/documents/{id}/export/docx-url` → 404
  - `POST /api/v2/documents/{id}/export/pdf` → 404
  - `ExportMenu` buttons disabled at mount (`canExport=false` — `sessionPhase` ≠ `writer|readonly`). Even if enabled, fetch would 404.
  - Frontend code in [exportsV2.ts](frontend/apps/web/src/features/documents/v2/api/exportsV2.ts) hits `/export/docx-url` + `/export/pdf`; handler never registered server-side.
  - Not a walk blocker (editor itself still works, autosave persists). Logged; continue.

⚠️ `ExportMenuButton` uses native `<details>`/`<summary>` — no explicit Portuguese label, not styled for workspace theme. Cosmetic.

## 5. Comments (P5.1)

🛑 **All CRUD endpoints 404** despite source code in [handler.go:75-78](internal/modules/templates/delivery/http/handler.go):
  - `GET /api/v2/documents/{id}/comments` → 404
  - `POST /api/v2/documents/{id}/comments` → 404
  - Routes correctly wired via `docMod.RegisterRoutes(mux)` in [main.go:142](apps/api/cmd/metaldocs-api/main.go).
  - **Root cause: running backend binary is stale.** Process started 2026-04-19T17:45:43Z — before P5.1 code (commit 141fd40) was compiled into the current binary. Rebuild required.
  - Note: export 404s have a different cause — see §4 (deps `ExportPresign`/`ExportDocgen` never passed in `main.go:134-141`, so `ExportHandler` stays `nil` and its routes are never registered).
  - Frontend hook `useDocumentComments` polls on mount → repeating 404 in network pane.
  - Walk continues. Feature-level blocker for comments; rebuild backend to retest.

## 6. Documents list (mine/recent/library)

- ✅ `/documents` → "Todos documentos" — 25 total, recent list populated.
- ✅ `/documents/mine` → "Meus documentos" renders.
- ✅ `/documents/recent` → "Recentes" renders.
- ⚠️ "Tipos de documento" panel empty on all scopes — downstream of 404 on `/document-profiles` etc. Pre-existing v1 gap.

## 7. Legacy shells retirement notice

- ✅ `/content-builder` shows "Editor legado removido. Use a rota /documents-v2 para editar documentos."
- ✅ Template legacy editor shows "Editor de templates legado removido. Use a rota /templates-v2 para editar templates." ([TemplateEditorView.tsx:5-6](frontend/apps/web/src/features/templates/TemplateEditorView.tsx:5)).
- P6 retirement intent landed cleanly.

## 8. Summary

### Blockers (🛑)
1. ~~**Comments (P5.1) — backend binary stale**~~ → 🩹 **Resolved**. Two issues stacked:
   - Running binary predated commit 141fd40 → rebuilt + restarted.
   - Migration `0118_docx_v2_document_comments.sql` never applied to local Postgres (`schema_migrations` only had 0112-0113). Ran `psql -f 0114..0118` manually; `document_comments` table now exists. `GET /api/v2/documents/{id}/comments` → 200 `[]`.
2. ~~**Exports (PDF + DOCX download) — deps not wired**~~ → 🩹 **Code wired, activation pending ops config**. Fixes landed:
   - [main.go](apps/api/cmd/metaldocs-api/main.go): pass `ExportPresign` (re-uses `docPresigner`) + conditionally `ExportDocgen` when `DocgenV2Client != nil` (avoids typed-nil interface trap).
   - [config/docgen_v2.go](internal/platform/config/docgen_v2.go): read `METALDOCS_DOCGEN_V2_URL` (was `..._API_URL`) to match `.env.example` + all docs/plans.
   - Activation still requires ops: set `METALDOCS_DOCGEN_V2_URL` + `METALDOCS_DOCGEN_V2_SERVICE_TOKEN` in `.env` against a running docgen-v2 instance; restart API. Not a code blocker.

### Defects (⚠️)
- Sidebar "Novo documento" routes to legacy `/create` wizard (which itself depends on 404'd v1 profile endpoints).
- No sidebar entry for `/documents-v2/new` flow.
- MinIO PUT logs `[FAILED: net::ERR_ABORTED]` despite 200 (cosmetic; commit succeeds).
- eigenpal fetches `fonts.googleapis.com/css2?family=Calibri%20Light` → blocked by ORB (Carlito fallback loads).
- Pre-existing v1 404s: `document-profiles`, `process-areas`, `document-departments`, `document-subjects`.
- `GET /api/v1/operations/stream` → 401 on every page.
- HashRouter footgun: navigation via `window.location.pathname` silently fails; must set `location.hash`. Document for future devs.
- `ExportMenuButton` uses raw `<details>`/`<summary>` — no PT label, not themed.

### Passes (✅)
- Vite alias fix for eigenpal subpath resolution.
- Templates V2 list / editor panel renders.
- `POST /api/v2/documents` → 201 creates document with initial revision + session.
- eigenpal `DocxEditor` mounts with full toolbar + merge-field sidebar.
- Autosave end-to-end: presign → MinIO PUT → commit.
- Documents list scopes (library/mine/recent) render.
- Legacy editor shells show retirement stub.

### Recommended next actions
1. ~~Rebuild backend API binary + retest comments CRUD~~ — done.
2. ~~Wire `ExportPresign` + `ExportDocgen` in `main.go`~~ — done. Still TODO: set `METALDOCS_DOCGEN_V2_URL` + `METALDOCS_DOCGEN_V2_SERVICE_TOKEN` in `.env` pointing at running docgen-v2, restart API, retest UI.
3. Make migrations auto-run on API boot (or wire into `dev-local.ps1`) — current UX requires manual `dev-migrate.ps1` after every pull; caused this UAT's 500s.
4. Add `/documents-v2/new` sidebar entry; deprecate legacy `/create` wizard or repair v1 profile endpoints.
5. Clean up `net::ERR_ABORTED` on MinIO PUT (likely fetch AbortController race) — low priority.
