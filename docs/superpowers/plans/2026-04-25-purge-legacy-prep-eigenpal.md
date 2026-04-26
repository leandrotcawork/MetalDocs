# Plan: Purge Legacy + Quality Blockers Before Eigenpal Port

**Date:** 2026-04-25
**Feature:** Remove dead modules, dep orphans, and fix structural blockers in active code so the eigenpal port lands on clean ground.
**Successor plan:** `2026-04-24-eigenpal-outline-metadata-port.md` (P0→P4) executes after this one.

**Scope basis:** 4-agent parallel audit (backend legacy, frontend legacy, dep orphans, code quality).
**Constraint:** dev-only, no live consumers, breaking contracts is fine. No need for migrations, deprecation warnings, or compatibility shims.

---

## Guiding rules

1. Each phase ends with `rtk tsc --noEmit` clean and `go build ./...` clean. Do not start the next phase until green.
2. Delete callers before removing the called code, OR delete bottom-up if no callers exist.
3. Domain types of v1 still imported by notifications/search/workflow → migrate those callers to `documents_v2` types as part of phase B2.
4. Tests for deleted code: delete the test file in the same commit. Don't leave orphan tests.

---

## Phase A — Quality Blockers in Code We Keep

These touch ACTIVE modules and should land first so subsequent legacy deletes don't fight churn.

### A.1 — Replace `tx interface{}` with typed `DBExecutor` in registry sequence
**File:** `internal/modules/registry/domain/sequence.go:6`
**Change:** Define `type DBExecutor interface { QueryRowContext(ctx context.Context, sql string, args ...any) *sql.Row; ExecContext(ctx context.Context, sql string, args ...any) (sql.Result, error) }` (or pgx equivalent — check what the impl actually calls). Replace `tx interface{}` in `NextAndIncrement` with `DBExecutor`. Update implementation + every caller's type assertion.
**Verify:** `go build ./...` clean, registry tests pass.

### A.2 — Delete empty `TaxonomyReader interface{}`
**File:** `internal/modules/render/resolvers/resolver.go:61`
**Change:** Delete the empty interface declaration. Grep for `TaxonomyReader` to confirm zero callers; if any exist, delete those references too.
**Verify:** `go build ./...` clean.

### A.3 — DROPPED per Codex review
The two `MapErr` functions diverge too much to share without becoming a leaky abstraction. Templates `MapErr` is template-domain-specific (~30 LOC, `ErrInvalidVisibility` etc.); approval `MapErr` is broad (~110 LOC, authz + signatures + strict-body). Keep separate. Move only the shared `WriteJSON`/`readJSON` helpers (A.4) into `internal/platform/httpresponse`.

### A.4 — Consolidate `writeJSON` / `readJSON` helpers
**Files:**
- `internal/modules/templates_v2/delivery/http/handler.go:46-54` (local helpers)
- `internal/modules/documents_v2/approval/http/errors.go:161` (`WriteJSON` + `_, _ = w.Write(payload)`)
- NEW: `internal/platform/httpresponse/response.go`

**Change:** Move both `writeJSON` and `readJSON` into `internal/platform/httpresponse/response.go`. Single signature: `WriteJSON(w, status, body) error`. Return write errors (don't swallow `_, _`). Callers log at top level if write fails. Each handler keeps its own `MapErr` (per A.3 decision).
**Verify:** `go build ./...` clean. Response bodies match prior shape.

### A.5 — Stop discarding `actorID` in approval read service
**File:** `internal/modules/documents_v2/approval/application/read_service.go:24` (and `:76-82`)
**Change:** Either route `actorID` to authorization filter OR return `errors.New("not implemented")` from stubbed `ListPendingForActor` so callers fail loud. Pick whichever matches the route's current behavior — don't silently succeed.
**Verify:** Read service tests reflect chosen behavior.

### A.6 — Tighten `route_admin_handler` constructor
**File:** `internal/modules/documents_v2/approval/http/route_admin_handler.go:33-40`
**Change:** Pick one injection mode — direct service OR services container. Delete the runtime fallback. Update construction site in `module.go` accordingly.
**Verify:** `go build ./...` clean.

### A.7 — Type the v2 documents API client
**File:** `frontend/apps/web/src/features/documents/v2/api/documentsV2.ts:30-31`
**Change:** Define `DocumentResponse` + `RenameResponse` matching the Go domain. Replace `Promise<any>` returns. Propagate types through callers (DocumentEditorPage etc.); cast at boundary if backend types drift.
**Verify:** `rtk tsc --noEmit` clean in `frontend/apps/web`.

---

## Phase B — Backend Legacy Purge

Bottom-up: delete leaves first, then trunks.

### B.1 — Delete empty placeholder modules
**Paths:**
- `internal/modules/editor_sessions/` (6 LOC, blank `_` import in `main.go:21`) — confirmed by Codex
- `internal/modules/document_revisions/` (6 LOC, blank `_` import in `main.go:20`) — confirmed by Codex
- `internal/modules/versions/` (13 LOC, orphan domain port) — Codex notes NO blank import; just confirm zero callers via `rg "modules/versions"` before deleting

**Change:** Remove the directories. Remove the two confirmed blank imports from `apps/api/cmd/metaldocs-api/main.go`. Grep for `versions` stragglers.
**Verify:** `go build ./...` clean.

**B.1 plus — DB migration cross-reference (REVISED per Codex v2 review):**
Earlier draft proposed dropping `editor_sessions` / `document_revisions` tables. **DO NOT drop the tables.** Codex v2 confirmed `documents_v2/repository/repository.go:52,60,219,471` actively queries those tables. The v1 Go modules are empty scaffolds (delete those), but the tables themselves are owned by v2. No migration changes in this phase.

Action: only delete the 3 empty Go module directories + the 2 blank `_` imports. Tables remain. Migrations 0104/0105/0113/0110 untouched.

**B.1 plus 2 — Smoke test fix (Codex v5):**
`tests/docx_v2/scaffold_smoke_test.go:11-12,20-24` imports + asserts `editor_sessions.New() != nil` and `document_revisions.New() != nil`. Deleting the modules breaks this test. Fix by removing those two assertions from the smoke test (modules are empty placeholders, the assertions test nothing meaningful).
Note: `tests/integration/testdb/fixtures.go` uses the *table names* `editor_sessions`/`document_revisions` in raw SQL — NOT the Go modules. Tables remain (per above), so fixtures.go stays untouched.

### B.2 — Migrate v1 documents domain consumers, then delete v1 infra+delivery
**Path:** `internal/modules/documents/`

**⚠️ RISK FLAG (per Codex #4):** v1 status enum (`DRAFT/IN_REVIEW/APPROVED/PUBLISHED/ARCHIVED`) does NOT match v2 (`draft/finalized/archived`). This is NOT a trivial type swap. Migration requires explicit status mapping per caller.

**Confirmed v1 importers (Codex #4 + Codex v5):**
- `internal/modules/notifications/application/service.go:12`
- `internal/modules/search/infrastructure/documents/reader.go:6`
- `internal/modules/workflow/application/service.go:13`
- `internal/modules/workflow/delivery/http/handler.go:10`
- `internal/platform/bootstrap/api.go:19` (Codex v5)
- `internal/platform/bootstrap/worker.go` (Codex v5 — verify line)

Step B.2.0 — Status enum mapping decision (NEW):
Build an explicit mapping table:
| v1 status | v2 status |
|---|---|
| DRAFT | draft |
| IN_REVIEW | (no v2 equivalent — was approval flow inline; now external in approval module) |
| APPROVED | finalized |
| PUBLISHED | finalized (or new v2 state — verify with backend lead) |
| ARCHIVED | archived |

If `IN_REVIEW` and `PUBLISHED` have no clean v2 mapping → STOP. Either (a) extend v2 enum, or (b) defer this whole phase B.2 to a separate PR with backend stakeholder review. Do NOT silently collapse states.

Step B.2.1 — For each of the 6 importers:
- Replace v1 type imports with v2 equivalents.
- Apply status mapping at the boundary (helper function `v1StatusToV2(s string) string` if there's a single transition point, otherwise inline).
- Update tests to use v2 status strings.

Step B.2.2 — Delete v1 infra + delivery:
- `internal/modules/documents/infrastructure/`
- `internal/modules/documents/delivery/` (already a `.gitkeep` per audit)
- `internal/modules/documents/application/` if no remaining callers

Step B.2.3 — Decide v1 domain fate:
- If all 6 callers migrated → delete `internal/modules/documents/domain/`.
- Bootstrap in `main.go` instantiates v1 postgres repo unused → delete that block.
- Per Codex #10b: also audit DI wiring at `main.go:114-119` and `main.go:153-155` — these wire search/workflow/notifications using v1 types. Update to v2 in lockstep.

**Verify:** `go build ./...` clean. Backend tests pass. `rg "modules/documents\""` shows zero non-v2 references. Notification/search/workflow integration tests still meaningful.

### B.3 — Delete v1 templates module
**Path:** `internal/modules/templates/`
**Change:**
- Delete entire `internal/modules/templates/` directory.
- Update `tests/docx_v2/scaffold_smoke_test.go` to drop the v1 import (port to v2 or delete v1 sub-test).
- Per Codex #5: also delete v1 import in `tests/docx_v2/templates_integration_test.go:12-13` (port to v2 or delete).
- Confirm `main.go:181` already says "Legacy templates module routes removed" — if any `templates` ref remains, drop it.
**Verify:** `go build ./...` clean. Smoke + integration tests still meaningful or replaced.

### B.4 — Delete `apps/docgen/` (old Node service)
**Path:** `apps/docgen/`
**Change:**
- Confirm no `docker-compose.yml` reference (only `docgen-v2` is deployed).
- Per Codex #6: update `.github/workflows/governance-check.yml:34` — it still references `apps/docgen`. Either drop the reference or repoint to `docgen-v2`.
- Delete the directory + any workspace entry in root `package.json`.
**Verify:** `pnpm install` clean. No build script references it. `gh workflow run governance-check` (or local act) passes.

### B.5 — Delete one-shot seed binary
**File:** `cmd/seed-spec1-template/main.go`
**Change:** Delete the file. Confirm `//go:build ignore` so no main.go side-effect.
**Verify:** `go build ./...` clean.

---

## Phase C — Frontend Legacy Purge (low-risk batch)

Defer `DocumentsHubView` (1,102 LOC) to a separate plan — needs v2 hub designed first. This phase deletes the orphaned trees only.

### C.1 — Delete confirmed-orphan files (REVISED v4)
**Path:**
- `frontend/apps/web/src/features/documents/useDocumentDetail.ts` (32 LOC) — truly unimported

**DEFERRED from C.1 to follow-up plan:**
- `documents/canvas/*` — imports `documents/runtime/` types
- `documents/runtime/*` — `schemaRuntimeTypes.ts` + `schemaRuntimeAdapters.ts` actively imported by `src/api/documents.ts:30-31` and `src/store/documents.store.ts:2` (Codex v4 finding)

To delete canvas+runtime, the chain `api/documents.ts` (v1) → `store/documents.store.ts` (v1) must first be migrated/deleted. That's a separate plan with its own caller audit. Not blocking eigenpal port.

**Change:** Delete `useDocumentDetail.ts` only.
**Verify:** `rtk tsc --noEmit` clean.

### C.2 — Delete orphan v1 templates UI
**⚠️ Per Codex #7 — re-verify each path before delete. Some files in the original audit are wrong:**
- `composition-config-panel.tsx` IS imported by `templates/v2/TemplateAuthorPage.tsx:12` → **DO NOT DELETE**
- `BlockPalette.tsx` and `MetadataBar.tsx` paths flagged as suspect — confirm actual location with `rg --files | rg -i blockpalette` first

**Pre-delete check (mandatory for each file):**
```
rg -l "from.*<filename>" frontend/apps/web/src --type tsx --type ts
```
Only delete if the only hits are the file itself + its own test.

**Confirmed-orphan delete list (Codex v3 spot-checked):**
- `frontend/apps/web/src/features/templates/controls/` (158 LOC) — orphan confirmed
- `frontend/apps/web/src/features/templates/tabs/PropriedadesTab.tsx` (167 LOC) — orphan confirmed
- `frontend/apps/web/src/features/templates/ValidationPanel.tsx` (119 LOC) — orphan confirmed
- `frontend/apps/web/src/features/templates/StrippedFieldsBanner.tsx` (71 LOC) — orphan confirmed
- Their tests under `features/templates/__tests__/`

**REMOVED from delete list (still in use by v2 TemplateAuthorPage):**
- `composition-config-panel.tsx` — imported by `templates/v2/TemplateAuthorPage.tsx:12` (Codex v2)
- `placeholder-inspector.tsx` — imported by `templates/v2/TemplateAuthorPage.tsx:10`, used at `:403` (Codex v2)
- `placeholder-chip.tsx` — imported by `templates/v2/TemplateAuthorPage.tsx:8` (Codex v3)
- `zone-chip.tsx` — imported by `templates/v2/TemplateAuthorPage.tsx:9` (Codex v3)
- `zone-inspector.tsx` — imported by `templates/v2/TemplateAuthorPage.tsx:11` (Codex v3)

**Lesson:** original frontend audit's grep missed v2 `TemplateAuthorPage` imports for 5 files. Before any future "delete unused" pass, run import grep that covers `templates/v2/**`.

**Re-verify before delete:**
- `BlockPalette.tsx` — Codex says path may be wrong; confirm exists at `templates/` root (not under `documents/v2/`)
- `MetadataBar.tsx` — same caveat

**Change:** For each verified-orphan file: delete + its test in lockstep. After delete, grep again to confirm no broken imports.
**Verify:** `rtk tsc --noEmit` clean. `pnpm test` clean.

### C.3 — Delete TemplateEditorView stub + CSS + legacy App.tsx routes
**Files:**
- `frontend/apps/web/src/features/templates/TemplateEditorView.tsx` (9 LOC stub)
- `frontend/apps/web/src/features/templates/TemplateEditorView.module.css`

**App.tsx route cleanup (per Codex #10a, refined v3):**
Legacy refs confirmed at `App.tsx:24, 189, 434, 573` — `TemplateEditorView` import + `templateEditorParams` route plumbing. Delete these.
**DO NOT touch line 574** — Codex v3 confirms it's v2 `tplRoute.kind === 'author'` edit-mode logic, not legacy.

**Change:** Delete files + remove all 4 line refs from App.tsx in lockstep.
**Verify:** `rtk tsc --noEmit` clean. Manual: hitting old `/registry/profiles/.../templates/.../edit` URL gives 404, not stale stub.

### C.4 — Verify `useTemplateDraft` v1 → v2 parity, then delete v1
**Files:**
- `frontend/apps/web/src/features/templates/useTemplateDraft.ts` (180 LOC)
- `frontend/apps/web/src/features/templates/__tests__/useTemplateDraft.test.tsx`

**Change:** Compare against `v2/hooks/useTemplateDraft` (audit flagged MEDIUM). If v2 covers all behaviors → delete v1 + its test. If not → port missing behaviors to v2 first.
**Verify:** `rtk tsc --noEmit` clean. v2 template editing flows still work.

---

## Phase D — Dependency Pruning

### D.1 — Drop dead frontend deps
**File:** `frontend/apps/web/package.json`
**Remove:** `@ckeditor/ckeditor5-react`, `ckeditor5`, `pdf-img-convert`
**Verify:** `pnpm install` rebuilds lockfile clean. `rtk tsc --noEmit` clean. `pnpm dev` boots.

### D.2 — `react-pdf` vs `pdfjs-dist` — KEEP BOTH (per Codex #8)
- `react-pdf` imported at `PdfPreview.tsx:2`
- `pdfjs-dist` imported at `PdfPreview.tsx:3` and `e2e/helpers/pixel-diff.ts:4`
Different layers, both active. No action.

### D.3 — Run `go mod tidy` (REVISED per Codex v5)
**Action:** Run `go mod tidy`. Expect NO removals.
**Note:** Original audit flagged `gopkg.in/yaml.v3` as unused. Codex v5 confirmed it IS used by `ops/canary/controller.go:19`. Keep it.
**Verify:** `go build ./...` + `go test ./...` clean. `go.mod` diff is empty or whitespace-only.

---

## Phase E — Final Audit

### E.1 — Self-check post-purge
- `go build ./...` and `go test ./...` clean
- `rtk tsc --noEmit` clean (web + editor-ui)
- `pnpm test` clean
- `git diff --stat` — confirm LOC delta roughly matches estimate (~1,700 frontend + ~3,880 backend)
- `rg -i "blocknote|ckeditor|todo: remove|deprecated|legacy"` — sweep for stragglers

### E.2 — Handoff to eigenpal port plan
**File:** `docs/superpowers/plans/2026-04-24-eigenpal-outline-metadata-port.md` (P0 onward)
The eigenpal port plan assumes a clean tree from this purge. After E.1 green, kick off P0.

---

## Execution Order + Estimates (REVISED per Codex #9)

Order: safe B deletes → A refactors → risky B.2 (status enum migration) → C frontend → D deps → E final.

| # | Phase | Task | Estimate | Risk |
|---|---|---|---|---|
| 1 | B.1 | Placeholder modules + migration cross-ref | 20 min | L |
| 2 | B.3 | Delete v1 templates + tests | 25 min | L |
| 3 | B.4 | Delete apps/docgen + governance workflow | 15 min | L |
| 4 | B.5 | Delete seed binary | 5 min | L |
| 5 | A.1 | DBExecutor port | 30 min | M |
| 6 | A.2 | Delete TaxonomyReader | 5 min | L |
| 7 | A.3 | DROPPED | — | — |
| 8 | A.4 | Consolidate writeJSON | 20 min | L |
| 9 | A.5 | actorID stub | 15 min | L |
| 10 | A.6 | route_admin constructor | 15 min | L |
| 11 | A.7 | Type documentsV2 client | 25 min | L |
| 12 | B.2 | v1 docs status enum mapping + caller migration + DI audit | 120 min | **H** |
| 13 | C.1 | Orphan editor trees | 10 min | L |
| 14 | C.2 | Orphan templates UI (with re-verify) | 40 min | L |
| 15 | C.3 | TemplateEditorView stub + App.tsx routes | 15 min | L |
| 16 | C.4 | useTemplateDraft parity | 30 min | M |
| 17 | D.1 | Drop CKEditor + pdf-img-convert | 10 min | L |
| 18 | D.3 | go mod tidy | 5 min | L |
| 19 | E.1 | Self-check | 30 min | — |

**Total: ~7-8 hours active work. B.2 is the gate — if status enum doesn't map cleanly, defer B.2 to separate PR with backend lead review and ship the rest.**

---

## Success Criteria

1. All 4 audit categories green: backend legacy purged, frontend orphans purged, dep orphans purged, quality blockers fixed.
2. `go build ./...`, `go test ./...`, `rtk tsc --noEmit`, `pnpm test` all clean.
3. Total LOC delta ≈ 4,300-4,500 deleted (backend ~3,880 + frontend ~525 after Codex v3+v4 removed 5 v2-imported files plus deferred canvas/runtime cascade) minus the small additions in Phase A.
4. No `blocknote`, `ckeditor`, `documents/` v1 (except migrated domain shared types if any), `templates/` v1 references in active source tree.
5. Eigenpal port plan P0 starts on a green tree.

---

## What this plan does NOT do

- Port `DocumentsHubView` to v2 (1,102 LOC, separate plan — needs UX decision: merge into shell sidebar vs dedicated v2 hub).
- Delete `documents/canvas/` and `documents/runtime/` — deferred (Codex v4): runtime types still imported by v1 `api/documents.ts` + `store/documents.store.ts`. Separate plan needed to audit v1 api/store cascade.
- Drop `editor_sessions` / `document_revisions` DB tables — owned by v2 repo, keep.
- Touch `internal/modules/render/` — actively used by fanout, keep.
- Touch `apps/worker/` — actively used.
- Database migrations — schema already cleaned per audit (migrations 0032/0033/0064/0130/0132/0142 already drop legacy registry+template FKs).
