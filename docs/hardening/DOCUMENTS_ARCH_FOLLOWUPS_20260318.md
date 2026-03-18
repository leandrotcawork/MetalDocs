# Documents Architecture Follow-ups (2026-03-18)

## Context
Este documento registra follow-ups da revisao externa do sistema documental (frontend + backend) focada em:
- estabilidade de contrato (OpenAPI v1)
- politica de schema/migrations
- boundaries por modulo (modular monolith)
- escalabilidade e padronizacao do frontend

O objetivo e transformar achados em tasks pequenas e rastreaveis.

## Findings (Resumo)
1. **Audit append-only violado por migration destrutiva**
- Evidencia: `migrations/0033_delete_legacy_document_registry.sql` deleta `metaldocs.audit_events`.
- Impacto: conflita com o invariavel "audit append-only" e com operacao segura (retencao, compliance, debug).
- Direcao: remover do fluxo oficial de migrations e tratar como utilitario de ambiente dev (script/runbook).

2. **Breaking change no contrato v1**
- Evidencia: `api/openapi/v1/openapi.yaml` define `DocumentProfileItem.alias` como `required`.
- Impacto: clientes estritos podem quebrar (mesmo internos/futuros) ao validar resposta com schema anterior.
- Direcao: servidor sempre fornece `alias`, mas o contrato v1 deve manter `alias` como opcional (nao-required) para evolucao compat.

3. **Boundary drift: `documents/domain` depende de `workflow/domain`**
- Evidencia: `internal/modules/documents/domain/port.go` referencia `workflowdomain.Approval` no `Repository`.
- Impacto: acopla invariantes e tipos entre modulos; dificulta evolucao e testes isolados.
- Direcao: `documents` define tipos/DTOs proprios ou porta dedicada; integracao com workflow via interface/evento.

4. **Boundary drift: `documents/application` depende de `iam/domain`**
- Evidencia: `internal/modules/documents/application/service.go` usa `iamdomain.UserIDFromContext` e roles.
- Impacto: acoplamento ao dominio de IAM para algo que deveria ser "contexto de autenticacao" de plataforma.
- Direcao: depender de `internal/platform/authn` (ou porta local) para extrair user/roles do contexto.

5. **Frontend: custo e padrao de composicao**
- Evidencia: `DocumentWorkspaceShell` agrega contagens por profile filtrando `documents` em render.
- Impacto: O(P*N) e recomputacao; degrada com acervo grande; mistura "summary" no shell.
- Direcao: memoizacao no minimo; idealmente endpoint/backend summary ou adapter layer dedicado.

6. **Frontend: organizacao de UI primitives vs features**
- Evidencia: componentes como dropdowns e views estao juntos em `frontend/apps/web/src/components`.
- Impacto: dificulta padronizacao e reuso; aumenta risco de drift visual e logica duplicada.
- Direcao: separar `components/ui` (primitives) de `features/*` (views e adapters) e manter `App.tsx` como composicao.

## Linked Tasks
Ver `docs/tasks/NEXT_CYCLE_BACKLOG.md` para tasks 039+.
