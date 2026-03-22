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
- [x] `cd frontend/apps/web; npm.cmd run build`
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

- [x] T3: CSS Modules e tokens
      - Consolidar estilos no `AdminCenterView.module.css` e `ManagedUsersPanel.module.css`
      - Remover dependência de classes globais (catalog-*)
      commit: `refactor(frontend-admin): consolidate admin center styles`

- [x] T4: Adicionar barras de cor nos cards de resumo
      - Aplicar barra inferior colorida conforme referencia HTML
      commit: `fix(frontend-admin): add summary card accent bars`

- [x] T5: Ajustar card de usuarios online conforme referencia
      - Separadores, "Ver todos", e indicador online animado
      commit: `fix(frontend-admin): align online users card to reference`

- [x] T6: Ajustar card de atividades conforme referencia
      - Separadores, link "Audit trail" e layout de lista
      commit: `fix(frontend-admin): align activity card to reference`

- [x] T7: Alinhar card de atividades ao padrao do online
      - Titulos, separadores e colunas alinhadas
      commit: `fix(frontend-admin): align activity card spacing to online`

- [x] T8: Padronizar cards de lista (online + atividades)
      - Mesmo header, separadores e estrutura; variacoes só no conteudo
      commit: `refactor(frontend-admin): unify list card pattern`

- [x] T9: Equalizar altura dos headers dos cards
      - Mesmo wrapper de acoes e mesma altura minima para alinhar a divisoria
      commit: `fix(frontend-admin): align card header dividers`

- [x] T10: Alinhar barras horizontais entre cards
      - Fixar altura do header e baseline da divisoria nos dois cards
      commit: `fix(frontend-admin): align horizontal divider baselines`

- [x] T11: Centralizar corpo de atividades com usuarios
      - Equalizar altura visual do conteudo dos dois cards na mesma linha
      commit: `fix(frontend-admin): align activity body height with users`

- [x] T12: Ajustar empty state para inicio a esquerda
      - Manter centralizacao vertical e alinhar inicio horizontal na margem esquerda
      commit: `fix(frontend-admin): left-align activity empty row`

## Acceptance tests
- [x] `cd frontend/apps/web; npm.cmd run build`
- [ ] Visual: cards compactos, hierarquia clara e sem “cards gigantes”

---

## Feature: Admin Center UI - Gestao de Usuarios v2
Area: `frontend/apps/web/src/components/ManagedUsersPanel.*`  |  Risk: medium (UI full rewrite)  |  Goal: reconstruir Criar/Base/Editar conforme referencia HTML

Notes
- Reescrever markup e CSS mantendo handlers da store (`onCreateUser`, `onSaveManagedUser`, `onAdminResetPassword`, `onUnlockManagedUser`).
- Manter fluxo funcional sem alterar contratos de API.

## Tasks
- [x] T1: Recriar estrutura dos 3 cards (Criar, Base, Editar)
      - Substituir layout anterior e aplicar padrao da referencia
      commit: `refactor(frontend-admin): rebuild user management cards`

- [x] T2: Limitar base e equalizar altura dos cards
      - Exibir no maximo 10 usuarios, manter os 3 cards com mesma altura e incluir Departamento/Area de processo no criar usuario
      commit: `feat(frontend-admin): cap user list and add taxonomy fields`

- [x] T3: Usar spotlight + linha unica para Departamento/Area
      - Trocar selects nativos por `FilterDropdown` e alinhar ambos na mesma linha
      commit: `refactor(frontend-admin): use spotlight dropdowns in create form`

- [x] T4: Corrigir altura para nao cortar Criar/Editar
      - Igualar altura dos 3 cards sem truncar o conteudo de Criar e Editar
      commit: `fix(frontend-admin): normalize user cards full height`

- [x] T5: Equalizar altura pela grid sem altura rigida por coluna
      - Cards iguais via `grid-auto-rows` + `height: 100%`, sem clipping
      commit: `fix(frontend-admin): equalize user cards by grid row height`

- [x] T6: Reordenar campos de Editar e ajustar spacing do action stack
      - Departamento + Area na mesma linha; Perfil abaixo; reduzir gap do stack
      commit: `fix(frontend-admin): adjust edit fields layout and actions spacing`

- [x] T7: Ajustar altura fixa dos 3 cards
      - Base fica scrollavel e altura segue o editar_usuario sem estourar
      commit: `fix(frontend-admin): set user card fixed height`

- [x] T8: Deixar editar_usuario ditar altura
      - Remover altura fixa e evitar overflow no card de edicao
      commit: `fix(frontend-admin): let edit card drive height`

- [x] T9: Base scrollavel sem altura fixa de card
      - Remover altura compartilhada dos cards; manter scroll apenas na lista da Base de usuarios
      commit: `fix(frontend-admin): make base list scrollable with auto card heights`

- [x] T10: Remover contador da base e equalizar altura dinamica
      - Remover texto "Exibindo X de Y usuarios" e manter os tres cards com a altura do maior
      commit: `fix(frontend-admin): remove base footer and equalize dynamic card heights`

- [x] T11: Reativar scroll visivel na Base de usuarios
      - Manter estrutura/altura atual e ajustar apenas overflow da lista para scroll
      commit: `fix(frontend-admin): restore base users list scroll behavior`

- [x] T12: Remover limite artificial da Base de usuarios
      - Eliminar corte em 10 itens para permitir scroll real com toda a base
      commit: `fix(frontend-admin): remove user list cap in admin base`

- [x] T13: Fixar base pela altura do editar com scroll interno
      - Base de usuarios passa a rolar apenas na lista, sem ditar a altura do card
      commit: `fix(frontend-admin): constrain base list height to edit card`

- [x] T14: Vincular altura da Base ao card de Edicao
      - Medir `Editar usuario` em runtime e aplicar a mesma altura no card Base com scroll interno
      commit: `fix(frontend-admin): sync base card height with edit card`

- [x] T15: Remover stretch residual dos cards de usuarios
      - Eliminar `height: 100%` global para o Editar ditar sua propria altura e a Base espelhar essa medida
      commit: `fix(frontend-admin): remove residual card stretch in user admin`

- [x] T16: Sincronizar Criar e corrigir canto da scrollbar da Base
      - Aplicar a altura do Editar tambem no card Criar e clipar a Base para respeitar a curvatura do card
      commit: `fix(frontend-admin): sync create card height and clip base scrollbar corner`

- [x] T17: Ancorar acao de criar na base do card
      - Remover `+` do label e usar o espaco livre para manter o botao no rodape do card Criar
      commit: `fix(frontend-admin): anchor create user action to card footer`

- [x] T18: Acoes de edicao na mesma linha
      - Resetar, Desbloquear e Desativar ficam alinhados horizontalmente no card Editar
      commit: `fix(frontend-admin): align edit actions on one row`

- [x] T19: Aumentar gap entre senha e botoes
      - Aumentar espaco entre "Nova senha temporaria" e a linha de botoes de acoes
      commit: `fix(frontend-admin): increase spacing before edit actions`

- [x] T20: Ocultar estado de loading na gestao de usuarios
      - Remover o card "Atualizando base de usuarios" durante atualizacoes
      commit: `fix(frontend-admin): hide managed users loading panel`

- [x] T21: Fixar altura dos cards de Online e Auditoria com scroll
      - Definir altura fixa para ambos e rolagem interna das listas, mantendo itens ancorados no topo
      commit: `fix(frontend-admin): fix panel height and scroll for online and activity`

- [x] T22: Adicionar padding na lista da Base de usuarios
      - Aplicar padding interno no `ul` para manter o mesmo respiro visual dos demais paines
      commit: `fix(frontend-admin): add inner padding to base users list`

- [x] T23: Padronizar Base de usuarios com estrutura de painel
      - Trocar `article` por `div` e separar header, actions e lista no mesmo shape dos cards de auditoria
      commit: `refactor(frontend-admin): align base users card structure with audit panel`

- [x] T24: Adicionar padding no grid do Admin Center
      - Aplicar padding ao container do grid para manter respiro externo
      commit: `fix(frontend-admin): add padding to admin grid container`

- [x] T25: Aplicar padding no root da Base de usuarios
      - Mover o espacamento para o container principal da Base e ajustar paddings internos
      commit: `fix(frontend-admin): move base users spacing to root container`

- [x] T26: Aplicar estrutura de padding no Criar e Editar
      - Adotar padding no root dos cards Criar/Editar e simplificar paddings internos
      commit: `refactor(frontend-admin): align create and edit card spacing model`

- [x] T27: Padronizar semantica dos 3 cards como article
      - Converter Base de usuarios para `article` para unificar semantica com Criar e Editar
      commit: `refactor(frontend-admin): standardize managed user cards as article`

## Acceptance tests
- [x] `cd frontend/apps/web; npm.cmd run build`
- [ ] Visual: cards de criacao/base/edicao sem alongamento e no padrao da referencia

---

## Feature: Frontend UI Standardization
Area: `frontend/apps/web/src/components/ui/*` + `frontend/apps/web/src/components/ManagedUsersPanel.*`  |  Risk: low  |  Goal: padronizar fields e dropdowns repetidos

## Tasks
- [x] T1: Criar componentes unificados de campo
      - `TextFieldBox` e `DropdownFieldBox` como base de boxes de entrada/seleção
      commit: `refactor(frontend-ui): standardize form fields and dropdown usage`

- [x] T2: Migrar Gestao de Usuarios para os componentes padrao
      - Criar/Base/Editar com os mesmos componentes de input/dropdown
      commit: `refactor(frontend-admin): standardize user management fields`

- [x] T3: Documentar padrão de componente reutilizado
      - Documento em `docs/standards/FRONTEND_COMPONENT_STANDARDIZATION.md`
      commit: `docs(frontend): add component standardization guide`

## Acceptance tests
- [x] `cd frontend/apps/web; npm.cmd run build`
- [ ] Visual: campos e dropdowns do Admin Center sem variacao de estilo

---

## Feature: Documents Hub (Todos documentos redesign)
Area: `frontend/apps/web/src/features/documents/` + `frontend/apps/web/src/store/documents.store.ts`  |  Risk: medium  |  Goal: substituir a tela "Todos documentos" por um fluxo em 3 telas (overview -> listagem -> detalhe) mantendo o tema MetalDocs

Notes
- Frontend-first: reutilizar endpoints existentes (`search/documents`, `process-areas`, `document-profiles`) sempre que possivel.
- Se faltar dado (ex: "abertos recentemente"): implementar via store + `localStorage` (sem mudar backend).
- Se em algum ponto for necessario mudar contrato/endpoints: parar e adicionar T1 (OpenAPI) com `$metaldocs-openapi`.

## Tasks
- [x] T1: Mapear fluxo atual e pontos de entrada
      - Identificar onde "Todos documentos" e renderizado hoje (view/state)
      - Definir chaves de navegacao para: `DocumentsHub` (overview), `DocumentsCollection` (list), `DocumentOverview` (detalhe)
      commit: `chore(frontend-docs): define documents hub navigation model`

- [x] T2: Tela 1 (Overview) "Todos documentos"
      - KPIs: total, vigentes, em revisao, atencao (placeholder se nao houver dado)
      - Grid de Areas (cards)
      - Grid de Tipos de documento (cards)
      - Lista "Abertos recentemente" (persistido no client)
      commit: `feat(frontend-docs): add documents hub overview screen`

- [ ] T3: Persistir "Abertos recentemente"
      - Registrar documento ao abrir/entrar no detalhe
      - Persistir lista curta no `localStorage` e hidratar no load
      commit: `feat(frontend-docs): persist recently opened documents`

- [ ] T4: Tela 2 (Collection) por Area/Tipo
      - Header com breadcrumb + titulo + contagem
      - Tabs: Todos, Draft, Em revisao, Aprovados
      - Toggle Card/List + busca/ordenacao basica
      commit: `feat(frontend-docs): add documents collection screen`

- [ ] T5: Padrao de item (Card e Row)
      - Mesma informacao nos dois modos: codigo/titulo, tipo, area, owner, status, proxima revisao
      - Reutilizar componentes de badge/chip existentes quando possivel
      commit: `refactor(frontend-docs): unify collection item primitives`

- [ ] T6: Tela 3 (Document overview)
      - Card principal com header + status + meta (area/processo/versao/owner/proxima revisao)
      - Secoes: Classificacao, Governanca, Colaboracao, Diff (placeholders se nao houver dado)
      - Acoes: Abrir documento, Enviar para revisao, Duplicar, Historico de versoes (wire onde existir; desabilitar onde nao existir)
      commit: `feat(frontend-docs): add document overview screen`

- [ ] T7: Substituir a tela atual por este fluxo
      - "Todos documentos" passa a abrir a Tela 1
      - Navegacao entre as 3 telas sem reload e sem regressao no fluxo principal
      commit: `feat(frontend-docs): replace all-documents flow with hub`

## Acceptance tests
- [ ] `cd frontend/apps/web; npm.cmd run build`
- [ ] Manual: abrir "Todos documentos" -> ver overview com Areas/Tipos/Recentes
- [ ] Manual: clicar em Area/Tipo -> ver listagem + filtros + toggle card/list
- [ ] Manual: clicar em documento -> ver overview do documento + acoes principais
- [ ] No console errors durante o fluxo acima

