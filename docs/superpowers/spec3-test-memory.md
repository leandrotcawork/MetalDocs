# Spec 3 — Test Memory & Architecture Reference

**Date:** 2026-04-24  
**Purpose:** Persistent memory for new sessions after context clear. Covers all three specs + Spec 3 E2E test status.

---

## MetalDocs Overview

SaaS ISO 9001 document controller for industrial companies.  
Core loop: **Template author → Document fill-in → Approval → Freeze → PDF artifact → Audit trail.**

**Stack:**
- Backend: Go, PostgreSQL, port 8081 (via `scripts/start-api.ps1`, APP_PORT=8081)
- Frontend: React 18 + Vite, port 4174 (`npm --prefix frontend/apps/web run dev -- --port 4174`)
- Auth: cookie-based, bootstrap admin: `admin` / `AdminMetalDocs123!`
- Editor: `@eigenpal/docx-js-editor` (ProseMirror-based DOCX editor)

---

## Three Specs

### Spec 1 — Foundation Taxonomy × RBAC
Plan: `docs/superpowers/plans/2026-04-21-foundation-taxonomy-rbac-spec1.md`  
10 phases, ~115 tasks.  
**Status:** Plan complete, execution deferred.  
Goal: Document type taxonomy, area-based RBAC, role/permission system.

### Spec 2 — Doc Approval State Machine
Plan: `docs/superpowers/plans/2026-04-21-foundation-doc-approval-state-machine.md`  
12 phases, ~137 tasks. Gaps: `docs/superpowers/plans/followups/spec2-gaps.md`  
**Status:** Plan complete + Codex+Opus review done. Ready to execute.  
Goal: Approval stages, signoffs, lease-epoch idempotency, Spec 2 signatures bound to `(content_hash, values_hash, schema_hash)`.

### Spec 3 — Placeholder Fill-In + Eigenpal Fanout
Plan: `docs/superpowers/plans/2026-04-21-foundation-placeholder-fill-in-design.md`  
18 phases, ~115 tasks.  
**Status:** Backend IMPLEMENTED. Frontend UI wired (this session). E2E testing IN PROGRESS.  
Goal: ISO 9001 §7.5.3/§8.5.1 placeholder + editable-zone system. Three-layer freeze. Triple-hash. DOCX→PDF fanout.

---

## Spec 3 Architecture

### Token Format in DOCX
- Placeholder tokens: `{{placeholder_uuid}}` (UUID, NOT label — labels can change)
- Zone markers: `[[zone_uuid]]` (tentative; backend uses `zone-start:<id>` bookmarks in OOXML)
- Eigenpal `DocumentAgent.getVariables()` reads detected variables from DOCX
- At freeze: `agent.setVariables({uuid: value}).applyVariables()` → substitution

### Three-Layer Freeze
1. **Schema hash** — hash of placeholder/zone schema at template publish time
2. **Values hash** — canonical JSON hash of filled placeholder values
3. **Content hash** — hash of final DOCX bytes after fanout

### Key Backend Files
- `internal/modules/templates_v2/domain/schemas.go` — Placeholder, EditableZone, ContentPolicy, CompositionConfig
- `internal/modules/templates_v2/delivery/http/handler.go` — route registration
- `internal/modules/templates_v2/delivery/http/routes_schema.go` — PUT `/api/v2/templates/{id}/versions/{n}/schema`
- `internal/modules/documents_v2/application/fillin_service.go` — placeholder value upsert
- `internal/modules/documents_v2/application/freeze_service.go` — approval freeze → fanout
- `internal/modules/render/fanout/client.go` — POST `/render/fanout` (docgen-v2 service)

### Key Frontend Files
- `frontend/apps/web/src/features/templates/v2/TemplateAuthorPage.tsx` — main author page
- `frontend/apps/web/src/features/templates/v2/api/templatesV2.ts` — API client (wire-format adapters)
- `frontend/apps/web/src/features/templates/v2/hooks/useTemplateSchemas.ts` — load/save schemas
- `frontend/apps/web/src/features/templates/placeholder-chip.tsx` — drag source chip
- `frontend/apps/web/src/features/templates/zone-chip.tsx` — drag source chip
- `frontend/apps/web/src/features/templates/placeholder-inspector.tsx` — right panel inspector
- `frontend/apps/web/src/features/templates/zone-inspector.tsx` — zone content policy editor
- `frontend/apps/web/src/features/templates/placeholder-types.ts` — TypeScript types

### Backend API Endpoints (templates_v2)
```
POST   /api/v2/templates                              — create template
POST   /api/v2/templates/{id}/versions                — create next version
PUT    /api/v2/templates/{id}/versions/{n}/schema     — save placeholder/zone schema (NO trailing 's')
POST   /api/v2/templates/{id}/versions/{n}/autosave/presign
POST   /api/v2/templates/{id}/versions/{n}/autosave/commit
POST   /api/v2/templates/{id}/versions/{n}/submit     — submit for review
POST   /api/v2/templates/{id}/versions/{n}/review
POST   /api/v2/templates/{id}/versions/{n}/approve
POST   /api/v2/templates/{id}/archive
PUT    /api/v2/templates/{id}/approval-config
GET    /api/v2/templates
GET    /api/v2/templates/{id}
GET    /api/v2/templates/{id}/versions/{n}            — ALSO used as getSchemas (extracts placeholder_schema, editable_zones)
GET    /api/v2/templates/{id}/versions/{n}/docx-url
GET    /api/v2/templates/{id}/audit
```

### Wire Format Bug Fixed (this session)
Frontend types use camelCase; backend uses snake_case JSON.  
Fix in `templatesV2.ts`: added `zoneFromWire/zoneToWire/placeholderFromWire/placeholderToWire` adapters.  
- `contentPolicy.allowTables` ↔ `content_policy.allow_tables`
- `maxLength` ↔ `max_length`
- `resolverKey` ↔ `resolver_key`
- `visibleIf.placeholderID` ↔ `visible_if.placeholder_id`
- `visibleIf.operator` ↔ `visible_if.op`

`getTemplateSchemas` now calls `GET /versions/{n}` (not `/schemas`).  
`putTemplateSchemas` now calls `PUT /versions/{n}/schema` (singular, no trailing s).

---

## Spec 3 E2E Test Map

### F1 — Template Authoring ✅ VERIFIED (this session)

| Test | Status | Notes |
|------|--------|-------|
| F1.1 Add placeholder chip | ✅ | Click "+ Add placeholder" |
| F1.2 Rename label | ✅ | Inspector label input → debounce 400ms → PUT /schema 200 |
| F1.3 Change type (text→date) | ✅ | Inspector select → persisted |
| F1.4 Required toggle | ✅ | Checkbox → persisted |
| F1.5 Add zone | ✅ | Click "+ Add zone" |
| F1.6 Zone content policy | ✅ | Allow tables/images/headings/lists toggles → persisted |
| F1.7 Drag placeholder to DOCX | ✅ | Drop on canvas → `{{uuid}}` inserted at ProseMirror cursor |
| F1.8 Schema reload on refresh | ✅ | GET /versions/{n} extracts and maps arrays correctly |

**Drag insert implementation:**  
`editorRef.current.getEditorRef().getView().dispatch(state.tr.insertText('{{id}}', from, to))`  
Via `DocxEditorRef.getEditorRef()` → `PagedEditorRef.getView()` → `EditorView.dispatch(Transaction)`.

**NOT yet verified in F1:**
- Delete placeholder/zone
- Reorder
- Select dropdown (options list)
- visibleIf conditional logic
- Drag zone to canvas (`[[id]]` insert)
- Publish gate (schema validation before publish)

### F2 — Fill-In Flow ⬜ NOT STARTED

**What's needed:** Wire Spec 3 components into `DocumentEditorPage`.  
Components exist with unit tests but zero runtime imports.

Components to wire:
- `SubmitButton` — triggers `PUT /api/v2/documents/{id}/placeholders/{pid}` per field
- `ZoneToolbar` — rich-text editing controls for editable zones
- `usePlaceholderValue` hook — manages per-placeholder field state
- Fill-in form loader — reads schema from template snapshot, renders form fields

Flow:
1. User opens document (created from template)
2. Document has snapshot of template schema (placeholder_schema, editable_zones)
3. User fills form fields → each save calls PUT placeholder value
4. Computed placeholders auto-resolve (DraftResolverService)
5. User clicks Submit → `POST /api/v2/documents/{id}/submit`

### F3 — Approval + Freeze ⬜ NOT STARTED

Backend exists. Frontend approval panel likely needs wiring.

Flow:
1. Reviewer sees document in `in_review` state
2. Reviewer approves → `POST /api/v2/documents/{id}/approve`
3. FreezeService runs: validate required fields, resolve computed, compute values_hash
4. Calls fanout service `POST /render/fanout` with `{placeholder_values: {uuid: value}, zone_content: {uuid: ooxml}}`
5. Returns `content_hash`, `final_docx_s3_key`
6. Async PDF generation via Service Bus `docgen_v2_pdf`

### F4 — View Frozen Artifact ⬜ NOT STARTED

`GET /api/v2/documents/{id}/view` → returns signed URL for final PDF.  
Frontend needs viewer component (react-pdf or iframe).

### Cross-Cutting Tests ⬜ NOT STARTED

- ETag/idempotency on schema PUT
- Console zero-error baseline
- Network: no 4xx on happy path
- content_hash mismatch → 409 Conflict
- Required placeholder missing at freeze → validation error

---

## Dev Server Setup

```bash
# API
powershell -ExecutionPolicy Bypass -File scripts/start-api.ps1
# → port 8081, reads .env

# Frontend
npm --prefix frontend/apps/web run dev -- --port 4174 --strictPort
# → port 4174, proxies /api/v2 → :8081

# Login
# URL: http://localhost:4174
# Username: admin
# Password: AdminMetalDocs123!

# Navigate to template author
# http://localhost:4174/#/templates-v2/{templateId}/versions/{n}/author
# State-based routing (not URL) — use "Open" button from list page
```

### Existing Test Templates (as of 2026-04-24)
- `34b7b0da-38bc-40d0-9d45-76c9e8f88b74` — "sadasd" (test template, has 4 placeholders + 1 zone)
- `99bdf19e-d4f9-46b0-8927-c218865fa5fb` — "Purchase Order"

**sadasd placeholders (saved state):**
- `b6e73ea9-...` — "New placeholder", text, required=false
- `dbb456e5-...` — "Customer Name", date, required=true ← tested token
- `a5b9bd7e-...` — "New placeholder", text
- `45782e5b-...` — "New placeholder", text

**sadasd zones:**
- `e3813647-...` — "New zone", allow_tables=true, others=false

---

## Key Bugs Fixed This Session

1. **Import path typo**: `TemplateAuthorPage.tsx` line 6 had 4 dots (`../../../../editor-adapters/...`), fixed to 3 dots.
2. **Schemas endpoint mismatch**: Frontend called `/schemas` (plural), backend only has `/schema` (singular). Fixed: `getTemplateSchemas` reads from `GET /versions/{n}`, `putTemplateSchemas` writes to `PUT /versions/{n}/schema`.
3. **camelCase↔snake_case wire bug**: Zone `contentPolicy` fields sent as camelCase → backend stored all false. Fixed with wire adapters.
4. **Drag insert stub**: `insertPlaceholder` was `console.info + toast` only. Fixed to use `PagedEditorRef.getView().dispatch(tr.insertText(...))`.

---

## Next Session TODO

1. **F2 wire-in**: Find `DocumentEditorPage`, identify Spec 3 fill-in components, wire them (likely needs Codex MCP dispatch)
2. **F2 verify**: Fill form fields → PUT values → verify DB
3. **F3 verify**: Submit + approve → freeze → check fanout response + final_docx_s3_key
4. **F4 verify**: GET view → signed PDF URL → render
5. **Commit**: All changes from this session uncommitted (TemplateAuthorPage.tsx, TemplateAuthorPage.module.css, api/templatesV2.ts)

---

## Session Git State (2026-04-24)

```
Branch: main
Modified (unstaged):
  frontend/apps/web/src/features/templates/v2/TemplateAuthorPage.module.css
  frontend/apps/web/src/features/templates/v2/TemplateAuthorPage.tsx
  frontend/apps/web/src/features/templates/v2/api/templatesV2.ts
```

**Changes in this session:**
- `TemplateAuthorPage.tsx`: Fixed import path (3 dots), wired `insertTokenAtCursor` via `PagedEditorRef`
- `TemplateAuthorPage.module.css`: CSS added by Codex for sidePanel/rightPanel/inspector
- `api/templatesV2.ts`: Replaced `/schemas` endpoint calls with `/versions/{n}` + `/schema`, added full wire-format adapters (camelCase↔snake_case)
