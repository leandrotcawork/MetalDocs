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
- [x] Add global split (minimal): `frontend/apps/web/src/styles/tokens.css` + `frontend/apps/web/src/styles/base.css` (keep `styles.css` only as temporary bridge).
- [x] For each target: create `<Component>.module.css` and migrate classes from global CSS.
- [x] Remove migrated selectors from `frontend/apps/web/src/styles.css` as each component converts.
- [x] Ensure styles use tokens (`var(--...)`) rather than hardcoded values.

Acceptance
- [x] `cd frontend/apps/web; npm run build` passes
- [x] App opens with no console errors in main flows
- [x] login -> listagem -> detalhe works
- [x] `cd frontend/apps/web; npm run e2e:smoke` passes
commit: `refactor(frontend-css): introduce css modules for workspaces`

---

## Feature: Content Builder UX + consistency
Area: `frontend/apps/web/src/components/create/` + `frontend/apps/web/src/components/content-builder/`  |  Risk: medium (UI flow + state transitions)  |  Goal: align flows and polish UX

Notes
- Scope is frontend-only (no API contract changes).
- Keep diffs per task; one commit per task (Lesson H).

## Tasks
- [x] T1: Split "Salvar e ir" behavior by mode
      - Default path: button navigates to editor without persisting document until the user fills it
      - Exception: if user selects "Usar Template Word", it must save/prepare first
      commit: `fix(frontend-create): navigate-to-editor without save by default`

- [x] T2: Reuse progress sidebar on "preencher documento" screen
      - Extract sidebar from create flow into a reusable component (used in Create + Fill/Edit)
      commit: `refactor(frontend-ui): extract progress sidebar component`

- [x] T3: Normalize Content Builder action buttons (size + color)
      - Back / Save draft / Generate PDF match the Create flow visual standard
      commit: `fix(frontend-ui): normalize content builder buttons`

- [x] T4: Make PDF preview collapsible/expandable
      - Persist state in UI store if it’s shared; otherwise local state in feature
      commit: `feat(frontend-content-builder): add collapsible pdf preview`

- [x] T5: Remove "Voltar" button from content builder topbar
      - Keep only the correct navigation affordance (per design decision)
      commit: `fix(frontend-content-builder): remove topbar back button`

- [x] T6: Improve document bar text hierarchy and format
      - Title becomes `PO-XX-<Nome do Documento>`
      - Version and Status become clearer and consistent with typography
      commit: `fix(frontend-content-builder): improve document bar hierarchy`

- [x] T7: Refine docbar + preview controls
      - Remove document subtitle, make back button vinho
      - Preview toggle on the left with arrow icons and smaller size
      commit: `fix(frontend-content-builder): refine docbar and preview controls`

- [x] T8: Align collapsed preview toggle to top
      - Avoid centered placement when collapsed
      commit: `fix(frontend-content-builder): align collapsed preview toggle`

- [x] T9: Center collapsed preview toggle horizontally
      - Keep top alignment while centering horizontally
      commit: `fix(frontend-content-builder): center collapsed preview toggle`

- [x] T10: Back from content builder returns to create
      - Replace library navigation with create flow
      commit: `fix(frontend-content-builder): route back to create`

- [x] T11: Defer document creation until editor PDF
      - Back restores form fields from the edited document
      - "Ir para editor" opens draft; create on "Gerar PDF"
      commit: `fix(frontend-content-builder): defer create to editor`

- [x] T12: Load schema in editor for draft documents
      - Draft editor loads profile schema even before documentId exists
      commit: `fix(frontend-content-builder): load draft schema`

## Acceptance tests (run after each task)
- [x] `cd frontend/apps/web; npm.cmd run build`
- [ ] Manual: Create doc -> reach editor via both paths (no-template vs Word template)
- [ ] Manual: In editor, verify buttons are consistent + PDF preview toggles cleanly
- [ ] No console errors in browser during the flows above

---

## Feature: Admin Center (usuarios + online + atividade)
Area: `frontend/apps/web/src/features/iam/` + `internal/modules/iam/`  |  Risk: medium (new data surface)  |  Goal: admin view com cadastro, presenca e atividade recente

Notes
- Reusar CRUD de usuarios existente (ManagedUsersPanel) dentro do Admin Center.
- Dados de online/atividade dependem de endpoint backend; se nao existir, adicionar.

## Tasks
- [x] T1: Contrato de dados (OpenAPI)
      - Definir endpoint(s) para: usuarios online, ultima atividade, ultimo login
      commit: `docs(api): add admin center contracts`

- [x] T2: Backend IAM (se necessario)
      - Service + handler para overview/admin dashboard
      - Permission registrada em `permissions.go`
      commit: `feat(iam): add admin dashboard overview`

- [x] T3: Store de admin dashboard
      - `frontend/apps/web/src/store/admin.store.ts` com estado de online/atividade
      commit: `feat(frontend-admin): add admin dashboard store`

- [x] T4: Feature view + CSS Modules
      - `frontend/apps/web/src/features/iam/AdminCenterView.tsx`
      - `frontend/apps/web/src/features/iam/AdminCenterView.module.css`
      commit: `feat(frontend-admin): add admin center view`

- [x] T5: Integracao no App + navegacao
      - Novo `activeView` ou substituicao do painel admin atual
      commit: `feat(frontend-admin): wire admin center navigation`
      
- [x] T6: Sincronizar formulario de usuario no Admin Center
      - Evita acoplamento com App e corrige build
      commit: `fix(frontend-admin): sync managed user form`

## Acceptance tests
- [ ] `cd frontend/apps/web; npm.cmd run build`
- [ ] Admin ve: lista de usuarios, ultimo login, online, ultimas atividades
- [ ] Nenhum erro de console ao navegar para Admin Center

---

## Feature: Admin Center UI (HTML ref: metaldocs-admin-sample2)
Area: `frontend/apps/web/src/features/iam/` + `frontend/apps/web/src/components/ManagedUsersPanel.tsx`  |  Risk: low (UI/CSS only)  |  Goal: aplicar layout e hierarquia do HTML de referencia

Notes
- Somente front-end e CSS Modules, sem alterar dados/fluxo.
- Seguir tokens (`var(--...)`) e evitar estilos inline.

## Tasks
- [x] T1: Mapear estrutura do HTML no AdminCenterView
      - Header do workspace (kicker/title/description) alinhado ao layout
      - KPI row com cards compactos e subtitulos
      - Cards de Online e Atividades com badge no header
      commit: `refactor(frontend-admin): align admin center layout to html`

- [x] T2: Refatorar ManagedUsersPanel para layout de 3 colunas do HTML
      - Criar estrutura de sections e headers equivalentes
      - Ajustar tipografia, chips e ações
      commit: `refactor(frontend-admin): align managed users layout to html`

- [ ] T3: CSS Modules e tokens
      - Consolidar estilos no `AdminCenterView.module.css` e `ManagedUsersPanel.module.css`
      - Remover dependência de classes globais (catalog-*)
      commit: `refactor(frontend-admin): consolidate admin center styles`

## Acceptance tests
- [ ] `cd frontend/apps/web; npm.cmd run build`
- [ ] Visual: cards compactos, hierarquia clara e sem “cards gigantes”

