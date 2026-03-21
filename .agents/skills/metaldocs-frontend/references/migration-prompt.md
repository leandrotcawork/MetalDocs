# Prompt — Migração Frontend MetalDocs

Cole esse prompt no Codex para iniciar a migração.

---

Leia antes de qualquer coisa:
1. `tasks/lessons.md`
2. `tasks/todo.md`
3. `AGENTS.md`

Use `$md` e `$metaldocs-frontend` para planejar e executar a migração do frontend MetalDocs.

## Contexto

O frontend atual em `frontend/apps/web/src/` tem 4 problemas estruturais que precisam ser corrigidos antes de qualquer nova feature:

1. **App.tsx God Component** — 1.287 linhas, 38 useState, 35 handlers. Toda a lógica de negócio em um arquivo.
2. **CSS monolítico** — styles.css com 4.200 linhas, zero CSS Modules. Tudo global.
3. **Prop drilling** — componentes recebendo 20+ props porque o estado está centralizado no App.tsx.
4. **lib.api.ts acumulando 3 responsabilidades** — transporte, endpoints e normalização em 634 linhas.

## O que quero

Use `$metaldocs-frontend` como referência de como o código deve ficar.

**Entre em plan mode.** Antes de qualquer código, produza um plano em `tasks/todo.md` com fases de migração ordenadas. Cada fase deve ser independente — o app tem que funcionar ao final de cada fase.

## Fases sugeridas (valide e ajuste no plano)

**Fase 1 — Separar API por domínio**
Criar `src/api/client.ts` (transporte puro do lib.api.ts atual) e `src/api/documents.ts`, `src/api/auth.ts`, `src/api/iam.ts`, `src/api/notifications.ts`, `src/api/registry.ts`. Sem mudar nada no App.tsx ainda. Apenas reorganizar onde o código vive.

**Fase 2 — Criar stores Zustand**
Criar `src/store/auth.store.ts`, `src/store/documents.store.ts`, `src/store/ui.store.ts`, `src/store/registry.store.ts`. Popular com os estados correspondentes do App.tsx. App.tsx ainda existe mas começa a usar os stores.

**Fase 3 — Desmontar App.tsx**
Migrar os handlers de cada domínio para hooks em `src/features/<domain>/use<Feature>.ts`. App.tsx vira um router que só lê stores.

**Fase 4 — CSS Modules nos componentes maiores**
DocumentsWorkspace, DocumentWorkspaceShell, RegistryExplorer — cada um ganha seu `.module.css`. Remover classes equivalentes do styles.css global.

## Critérios de aceite por fase

- `tsc --noEmit` passa
- App abre no browser sem erros de console
- Fluxo principal funciona (login → listagem → detalhe)
- Nenhuma regressão nos testes e2e

## Instrução

Escreva o plano completo em `tasks/todo.md` com tarefas, ordem, critérios de aceite e commit message por fase. Aguarde aprovação antes de começar qualquer implementação.
