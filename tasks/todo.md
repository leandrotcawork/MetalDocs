# tasks/todo.md
# Frontend migration plan. Execute one phase at a time; app must work after each phase.

## Feature: Frontend migration (debt paydown)
Area: `frontend/apps/web/src/`  |  Risk: medium (wide refactor)  |  Goal: unblock new features safely

## Guardrails (hard)
- App must run after every phase (no "big bang").
- No behavior change intended unless explicitly called out.
- Prefer additive moves + re-exports to keep diffs safe.
- After each phase: run checks + commit.

## Baseline (do once, before Phase 1)
- [ ] Confirm current local dev boot: `powershell -ExecutionPolicy Bypass -File scripts/dev-local.ps1`
- [ ] Confirm web builds: `cd frontend/apps/web; npm run build`
- [ ] Confirm e2e smoke (when infra is up): `cd frontend/apps/web; npm run e2e:smoke`
- [ ] Capture “golden path” manual smoke: login → listagem → detalhe (note any current quirks; do not fix yet)
commit: `chore(frontend): record baseline checks`

## Phase 1 — Split API by domain (no App.tsx changes yet)
Goal: remove `lib.api.ts` as god-module by extracting transport + domain files, but keep the same call sites working.

Tasks
- [x] Create `frontend/apps/web/src/api/client.ts` (transport only: fetch wrapper, trace helpers, error normalization).
- [x] Create domain endpoint modules:
  - `frontend/apps/web/src/api/auth.ts`
  - `frontend/apps/web/src/api/documents.ts`
  - `frontend/apps/web/src/api/iam.ts`
  - `frontend/apps/web/src/api/notifications.ts`
  - `frontend/apps/web/src/api/registry.ts`
  - (optional, if used) `frontend/apps/web/src/api/workflow.ts`
- [x] Keep compatibility: `frontend/apps/web/src/lib.api.ts` becomes a thin facade re-exporting the old `api` object shape (delegating into `src/api/*`).
- [x] Update imports (only where needed) to keep build green.

Acceptance
- [x] `cd frontend/apps/web; npm run build` passes
- [x] App opens with no console errors in main flows
- [x] login → listagem → detalhe works
- [x] `cd frontend/apps/web; npm run e2e:smoke` passes
commit: `refactor(frontend-api): split lib.api by domain`

## Phase 2 — Introduce Zustand stores (reduce prop drilling)
Goal: move shared state out of `App.tsx` into domain stores, while keeping UI largely the same.

Tasks
- [x] Add `frontend/apps/web/src/store/auth.store.ts` (session/user, auth state, must-change-password, etc.)
- [x] Add `frontend/apps/web/src/store/documents.store.ts` (selected doc, lists, loading flags, filters/search state)
- [x] Add `frontend/apps/web/src/store/registry.store.ts` (registry explorer state)
- [x] Add `frontend/apps/web/src/store/notifications.store.ts` (unread, items, loading)
- [x] Add `frontend/apps/web/src/store/ui.store.ts` (workspace tab/view, modals, toasts, ephemeral UI)
- [x] Wire minimal set: `App.tsx` reads from stores instead of holding domain state in `useState`.
- [x] Keep mutations in `App.tsx` for now (Phase 3 moves them).

Acceptance
- [x] `cd frontend/apps/web; npm run build` passes
- [x] App opens with no console errors in main flows
- [x] login → listagem → detalhe works
- [x] `cd frontend/apps/web; npm run e2e:smoke` passes
commit: `refactor(frontend-store): introduce zustand domain stores`

## Phase 3 — Decompose App.tsx into feature hooks + shells
Goal: App becomes “router/shell only”; domain logic lives in hooks under `src/features/<domain>/`.

Tasks
- [x] Create hooks per domain (compose store + api; no JSX):
  - `frontend/apps/web/src/features/auth/useAuthSession.ts`
  - `frontend/apps/web/src/features/documents/useDocumentsWorkspace.ts`
  - `frontend/apps/web/src/features/documents/useDocumentDetail.ts`
  - `frontend/apps/web/src/features/registry/useRegistryExplorer.ts`
  - `frontend/apps/web/src/features/notifications/useNotifications.ts`
- [x] Move handlers out of `App.tsx` into those hooks (keep signatures stable; update call sites).
- [x] Move big UI blocks out of `App.tsx` into feature components:
  - `frontend/apps/web/src/features/documents/DocumentsWorkspaceView.tsx`
  - `frontend/apps/web/src/features/registry/RegistryExplorerView.tsx`
  - `frontend/apps/web/src/features/shell/WorkspaceShell.tsx`
- [x] `App.tsx` becomes a thin shell deciding what to render based on store state.

Acceptance
- [x] `cd frontend/apps/web; npm run build` passes
- [x] App opens with no console errors in main flows
- [x] login -> listagem -> detalhe works
- [x] `cd frontend/apps/web; npm run e2e:smoke` passes
commit: `refactor(frontend-app): dismantle App.tsx into features`

## Phase 4 — CSS Modules for the biggest components
Goal: stop leaking global styles; begin domain-scoped styling with `.module.css`.

Targets (start with these)
- `DocumentsWorkspace` (or `DocumentsWorkspaceView`)
- `DocumentWorkspaceShell` / `WorkspaceShell`
- `RegistryExplorer` (or `RegistryExplorerView`)

Tasks
- [ ] Add global split (minimal): `frontend/apps/web/src/styles/tokens.css` + `frontend/apps/web/src/styles/base.css` (keep `styles.css` only as temporary bridge).
- [ ] For each target: create `<Component>.module.css` and migrate classes from global CSS.
- [ ] Remove migrated selectors from `frontend/apps/web/src/styles.css` as each component converts.
- [ ] Ensure styles use tokens (`var(--...)`) rather than hardcoded values.

Acceptance
- [ ] `cd frontend/apps/web; npm run build` passes
- [x] App opens with no console errors in main flows
- [x] login -> listagem -> detalhe works
- [x] `cd frontend/apps/web; npm run e2e:smoke` passes
commit: `refactor(frontend-css): introduce css modules for workspaces`


