# MetalDocs

MetalDocs e uma plataforma interna para centralizacao, versionamento, workflow e auditoria de documentos.

## Objetivo
- Ter controle total de documentos e historico de mudancas.
- Garantir rastreabilidade (quem fez, o que mudou, quando mudou).
- Escalar sem reescrita estrutural.

## Escopo v1
- Cadastro de documentos e metadados.
- Versionamento imutavel.
- Workflow de aprovacao.
- Busca e consulta.
- Auditoria append-only.
- RBAC no backend.

## Fora de escopo v1
- Funcionalidades de IA generativa, NLP ou agentes no produto.
- Sugestoes automaticas baseadas em IA.
- Qualquer processamento em runtime dependente de LLM.

## Source of Truth (ordem de prioridade)
1. `AGENTS.md`
2. `docs/plans/MASTER_IMPLEMENTATION_PLAN.md`
3. `docs/architecture/ARCHITECTURE_GUARDRAILS.md`
4. `docs/standards/ENGINEERING_STANDARDS.md`
5. `docs/adr/*.md`
6. `api/openapi/v1/openapi.yaml`

## Documentos chave
- Plano mestre: `docs/plans/MASTER_IMPLEMENTATION_PLAN.md`
- Guardrails de arquitetura: `docs/architecture/ARCHITECTURE_GUARDRAILS.md`
- Standards de engenharia: `docs/standards/ENGINEERING_STANDARDS.md`
- Contratos internos (eventos/erros): `docs/contracts/INTERNAL_EVENTS_AND_ERRORS.md`
- Setup dev Go: `docs/runbooks/dev-setup.md`
- Setup manual Postgres: `docs/runbooks/postgres-manual-setup.md`
- Security baseline (SAST): `docs/runbooks/security-baseline.md`
- Contract baseline: `docs/runbooks/contract-baseline.md`
- Branch protection: `docs/runbooks/branch-protection.md`
- Release readiness: `docs/runbooks/release-readiness.md`
- Phase 3 hardening checklist: `docs/hardening/PHASE3_RELEASE_CHECKLIST.md`
- Architecture review (Q1): `docs/hardening/ARCHITECTURE_REVIEW_2026Q1.md`
- Module extraction strategy: `docs/hardening/MODULE_EXTRACTION_PLAN.md`
- Templates: `docs/templates/`
- Prompts oficiais: `docs/prompts/`

## Dependency Management (Go)
- O projeto nao usa `venv` ou `requirements.txt`.
- Dependencias sao geridas por `go.mod` e travadas em `go.sum`.

## Repository Modes
- `METALDOCS_REPOSITORY=memory` (default para desenvolvimento rapido)
- `METALDOCS_REPOSITORY=postgres` (persistencia real)

## Ambiente inicial
- Host local: `192.168.0.3`.
- Deploy inicial: Docker Compose single-node.
