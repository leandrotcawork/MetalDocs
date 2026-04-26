# Purge Editable Zones — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Remove "editable zones" from MetalDocs end-to-end (frontend authoring UI, schema persistence, fill-in handlers, freeze/reconstruct, repository, DB columns). Eigenpal-pure: only placeholders remain.

**Architecture:** Surgical deletion. Zones (`[[id]]` tokens) are a MetalDocs layer above eigenpal that was never tested in the eigenpal spike. Eigenpal's `templatePlugin` covers placeholders natively (`{{uuid}}` tokens); zones add nothing eigenpal can't cover via additional placeholders if needed later. Strip them.

**Tech Stack:** React + TS (frontend), Go (api), Postgres (DB), eigenpal docx-js-editor.

---

## Pre-Flight

- [ ] **Confirm clean working tree:** `git status` shows only the existing diff from current branch. If unrelated changes — stash first.
- [ ] **Branch:** Create a new branch off `main`: `git checkout -b chore/purge-editable-zones`
- [ ] **Capture baseline:** Run `go build ./... && cd frontend/apps/web && pnpm tsc --noEmit` — record current state. Must be green before purge.

---

## Task 1: Remove zone frontend components

**Files:**
- Delete: `frontend/apps/web/src/features/templates/zone-chip.tsx`
- Delete: `frontend/apps/web/src/features/templates/zone-inspector.tsx`

- [ ] **Step 1: Delete files**

```bash
rm frontend/apps/web/src/features/templates/zone-chip.tsx
rm frontend/apps/web/src/features/templates/zone-inspector.tsx
```

- [ ] **Step 2: Verify no other imports**

```bash
grep -rn "zone-chip\|zone-inspector\|ZoneChip\|ZoneInspector" frontend/
```
Expected: only matches inside `TemplateAuthorPage.tsx` (we'll fix in Task 2).

---

## Task 2: Strip zones from TemplateAuthorPage

**Files:**
- Modify: `frontend/apps/web/src/features/templates/v2/TemplateAuthorPage.tsx`

- [ ] **Step 1: Open file, identify all zone references**

Search for: `zone`, `Zone`, `zones`, `Zones`, `ZONES`, `[[`, `editableZones`, `editable_zones`, `EditableZone`, `updateZone`, `addZone`, `removeZone`.

- [ ] **Step 2: Remove imports**

Remove `import { ZoneChip } from '../zone-chip'` and `import { ZoneInspector } from '../zone-inspector'`.

- [ ] **Step 3: Remove zone state + handlers**

Delete `localSchemas.zones`, `updateZone`, `addZone`, `removeZone`, `zonesById` (or whatever the names are — read the file first). Keep only placeholder-related state.

- [ ] **Step 4: Remove ZONES section in Variables panel JSX**

Delete the `<section>` rendering ZONES heading, the `+ Add zone` button, and the zone list mapping.

- [ ] **Step 5: Remove zone branch in Inspector**

Right inspector should only handle placeholder selection. Delete `if (selectedZone) return <ZoneInspector ... />`.

- [ ] **Step 6: Remove zone-token insertion logic**

Any `[[id]]` token insertion or zone-to-canvas drag handlers — delete.

- [ ] **Step 7: Build check**

```bash
cd frontend/apps/web && pnpm tsc --noEmit
```
Expected: 0 errors.

- [ ] **Step 8: Commit**

```bash
git add frontend/apps/web/src/features/templates/
git commit -m "refactor(templates): remove editable zones from author page"
```

---

## Task 3: Strip zones from schema hooks + types

**Files:**
- Modify: `frontend/apps/web/src/features/templates/v2/hooks/useTemplateSchemas.ts`
- Modify: `frontend/apps/web/src/features/templates/v2/api/templatesV2.ts` (if it has zones types)
- Modify: `frontend/apps/web/src/features/taxonomy/types.ts` (already has changes — check for EditableZone)
- Search + clean: `frontend/apps/web/src/features/templates/v2/types*` (if exists)

- [ ] **Step 1: Drop `zones` field from PutTemplateSchemas request type**

Remove `zones?: EditableZone[]` (or similar) from the request body type. Caller no longer sends it.

- [ ] **Step 2: Drop `zones` from local schemas state shape**

Wherever `localSchemas` is typed, remove zones field.

- [ ] **Step 3: Delete `EditableZone` type definition**

Find and delete the type. Search: `grep -rn "EditableZone\b" frontend/`.

- [ ] **Step 4: Build check**

```bash
cd frontend/apps/web && pnpm tsc --noEmit
```
Expected: 0 errors.

- [ ] **Step 5: Commit**

```bash
git add frontend/apps/web/src/
git commit -m "refactor(templates): drop EditableZone type and schema field"
```

---

## Task 4: Strip zones from frontend fill-in loader

**Files:**
- Modify: `frontend/apps/web/src/features/documents/fill-in-loader.ts`
- Modify: `frontend/apps/web/src/features/documents/v2/DocumentCreatePage.tsx` (if it references zones)
- Modify: `frontend/apps/web/src/features/taxonomy/ProfileEditDialog.tsx` (already in current diff — check)

- [ ] **Step 1: Find all zone references in fill-in flow**

```bash
grep -rn "zone\|Zone" frontend/apps/web/src/features/documents/
```

- [ ] **Step 2: Remove `getZoneContents`, `putZoneContent`, `setZoneContent` API calls**

Delete the API call functions and any places that consume them.

- [ ] **Step 3: Remove zone form state from fill-in form components**

If the placeholder form supports zones, remove zone fields from form state, validation, submission.

- [ ] **Step 4: Build check**

```bash
cd frontend/apps/web && pnpm tsc --noEmit
```
Expected: 0 errors.

- [ ] **Step 5: Commit**

```bash
git add frontend/apps/web/src/features/documents/
git commit -m "refactor(documents): remove zone fill-in from frontend"
```

---

## Task 5: Strip zones from documents_v2 fill-in handler

**Files:**
- Modify: `internal/modules/documents_v2/http/fillin_handler.go`
- Modify: `internal/modules/documents_v2/http/fillin_handler_test.go`

- [ ] **Step 1: Remove zone routes**

Delete (lines ~39, 41 — verify by reading file first):
- `r.Get("/documents/{id}/zones", h.ListZoneContents)`
- `r.Put("/documents/{id}/zones/{zid}", h.PutZoneContent)`

- [ ] **Step 2: Delete handler methods**

Delete `ListZoneContents` (lines ~85–100) and `PutZoneContent` (lines ~130–155). Verify line ranges by reading the file.

- [ ] **Step 3: Remove zone test cases**

In `fillin_handler_test.go`, delete tests targeting `/zones` and `/zones/{zid}`.

- [ ] **Step 4: Build check**

```bash
go build ./...
go test ./internal/modules/documents_v2/...
```
Expected: 0 errors, all passing.

- [ ] **Step 5: Commit**

```bash
git add internal/modules/documents_v2/http/
git commit -m "refactor(api): remove zone routes from fillin handler"
```

---

## Task 6: Strip zones from documents_v2 fill-in service

**Files:**
- Modify: `internal/modules/documents_v2/application/fillin_service.go`
- Modify: any service test file

- [ ] **Step 1: Delete `SetZoneContent` method**

Lines ~239–282. Verify by reading file.

- [ ] **Step 2: Delete `LoadZonesSchema` method**

Lines ~98–118. Verify by reading file.

- [ ] **Step 3: Remove zone fields from service struct + interface**

If service struct has `zoneRepo` or similar — remove. Update constructor to not require it.

- [ ] **Step 4: Update interface in module.go**

`internal/modules/documents_v2/module.go` likely wires the service — drop zone repo dependency from the constructor call.

- [ ] **Step 5: Build check**

```bash
go build ./...
go test ./internal/modules/documents_v2/...
```

- [ ] **Step 6: Commit**

```bash
git add internal/modules/documents_v2/
git commit -m "refactor(api): remove zone methods from fillin service"
```

---

## Task 7: Strip zones from freeze + reconstruct

**Files:**
- Modify: `internal/modules/documents_v2/application/freeze_service.go`
- Modify: `internal/modules/documents_v2/http/reconstruct_handler.go`
- Modify: `internal/modules/documents_v2/http/view_handler.go`

- [ ] **Step 1: Drop `zoneMap` from freeze**

In freeze_service.go (~lines 182–197, 210), delete:
- Zone loading code
- `zoneMap` variable
- Zone inclusion in the reconstruction payload

Freeze should only compute hashes/payload from placeholder values now.

- [ ] **Step 2: Drop zone handling from reconstruct_handler.go**

Read file, find any zone references, remove. Output payload no longer carries zone OOXML.

- [ ] **Step 3: Drop zone references from view_handler.go**

Same — read, find, remove.

- [ ] **Step 4: Build check**

```bash
go build ./...
go test ./internal/modules/documents_v2/...
```

- [ ] **Step 5: Commit**

```bash
git add internal/modules/documents_v2/
git commit -m "refactor(api): remove zones from freeze and reconstruct"
```

---

## Task 8: Strip zones from documents repository

**Files:**
- Modify: `internal/modules/documents_v2/repository/` — find all zone-related files and methods
- Modify: `internal/modules/documents_v2/repository/resolver_readers.go` (already in diff — check)

- [ ] **Step 1: Find zone repo methods**

```bash
grep -rn "Zone" internal/modules/documents_v2/repository/
```

- [ ] **Step 2: Delete `UpsertZoneContent`, `LoadZoneContents`, `LoadZonesSchema` and similar**

Remove method definitions, interface entries, mock implementations.

- [ ] **Step 3: Remove zone-related struct fields**

If repo has zone tables/columns referenced — remove.

- [ ] **Step 4: Build check**

```bash
go build ./...
```

- [ ] **Step 5: Commit**

```bash
git add internal/modules/documents_v2/repository/
git commit -m "refactor(api): remove zone repository methods"
```

---

## Task 9: Strip zones from templates_v2 schema endpoint

**Files:**
- Modify: `internal/modules/templates_v2/delivery/http/handler.go` (already in diff — check)
- Modify: `internal/modules/templates_v2/delivery/http/routes_schema.go`
- Modify: `internal/modules/templates_v2/application/...` — any UpdateSchemasCmd
- Modify: `internal/modules/templates_v2/repository/...`

- [ ] **Step 1: Find all zone references in templates_v2**

```bash
grep -rn "EditableZone\|editable_zone\|zones" internal/modules/templates_v2/
```

- [ ] **Step 2: Drop `EditableZones` field from UpdateSchemasCmd**

Schema endpoint should only accept `placeholders` now.

- [ ] **Step 3: Drop `editable_zones_schema` column references in repo**

Remove from SQL writes/reads. Migration in next task drops the column.

- [ ] **Step 4: Drop EditableZones from domain types**

In `internal/modules/templates_v2/domain/`, delete EditableZone struct.

- [ ] **Step 5: Build check**

```bash
go build ./...
go test ./internal/modules/templates_v2/...
```

- [ ] **Step 6: Commit**

```bash
git add internal/modules/templates_v2/
git commit -m "refactor(api): remove EditableZones from templates schema"
```

---

## Task 10: Drop zone DB columns and tables

**Files:**
- Create: `internal/db/migrations/<NNNN>_drop_editable_zones.sql` (or whatever migration tool/path the project uses — verify)

- [ ] **Step 1: Identify migration tool**

Check `internal/db/migrations/` (or `db/migrations/` or wherever migrations live). Read the latest migration filename to learn naming convention.

- [ ] **Step 2: Inventory zone DB objects**

Connect to DB or grep schema dumps for:
- `editable_zones_schema_snapshot` column on `templates_v2_template_version`
- `editable_zones_schema` (if separate from snapshot)
- `document_zone_contents` table (or whatever stores fill-in zone content) — verify by reading repo SQL

- [ ] **Step 3: Write down migration**

```sql
-- DOWN intentionally not provided; this is a forward-only purge.
ALTER TABLE templates_v2_template_version
  DROP COLUMN IF EXISTS editable_zones_schema_snapshot;

-- Adjust table name based on Task 8 findings:
DROP TABLE IF EXISTS document_zone_contents;
```

- [ ] **Step 4: Run migration locally**

Use whatever the project's migration runner is (e.g., `make migrate-up` or `go run ./cmd/migrate up`).

- [ ] **Step 5: Build + smoke check**

```bash
go build ./...
go test ./...
```

- [ ] **Step 6: Commit**

```bash
git add internal/db/migrations/
git commit -m "refactor(db): drop editable_zones columns and tables"
```

---

## Task 11: Strip zone references from contract + integration tests

**Files:**
- Modify: `internal/modules/taxonomy/delivery/http/routes_profiles_contract_test.go` (already in diff — check)
- Search: any `*_test.go` referencing zones

- [ ] **Step 1: Find all zone test references**

```bash
grep -rn "zone\|Zone" internal/ --include="*_test.go"
```

- [ ] **Step 2: Delete zone-targeted test cases**

For each match, decide: delete (zone-only test) or update (test that mentioned zones incidentally).

- [ ] **Step 3: Build + test check**

```bash
go build ./...
go test ./...
```
Expected: all green.

- [ ] **Step 4: Commit**

```bash
git add internal/
git commit -m "test: remove zone-specific test cases"
```

---

## Task 12: Final regression sweep

- [ ] **Step 1: Full build**

```bash
go build ./...
cd frontend/apps/web && pnpm tsc --noEmit
cd ../../packages/editor-ui && pnpm tsc --noEmit
```
Expected: 0 errors.

- [ ] **Step 2: Full test suite**

```bash
go test ./...
cd frontend && pnpm test
```
Expected: all green.

- [ ] **Step 3: Grep for orphan references**

```bash
grep -rn "EditableZone\|editable_zone\|zone-chip\|zone-inspector\|ZoneChip\|ZoneInspector\|/zones/\|/zones\"" \
  --include="*.ts" --include="*.tsx" --include="*.go" --include="*.sql" .
```
Expected: 0 matches (or only matches in this plan file itself).

- [ ] **Step 4: Manual smoke — start servers, open template author page**

Run `metaldocs-api` + `metaldocs-web`. Open `/templates-v2`, click an existing template, verify:
- Variables panel shows only PLACEHOLDERS section (no ZONES)
- No console errors
- Schema autosave still works on placeholder edit

- [ ] **Step 5: Push branch + open PR**

```bash
git push -u origin chore/purge-editable-zones
```
Open PR with title: `chore: purge legacy editable zones`. Body: link to this plan.

---

## Self-Review Notes

- All 12 tasks have explicit file paths or grep commands to find them
- Each task ends with build + test verification
- Each task ends with a focused commit
- DB migration is forward-only (no DOWN) — zones are gone for good
- Frontend tasks (1–4) → backend tasks (5–9) → DB (10) → tests (11) → regression (12)
- Order matters: deleting backend before frontend would break the running app; we go frontend-first so the API stays serving while the client stops calling
