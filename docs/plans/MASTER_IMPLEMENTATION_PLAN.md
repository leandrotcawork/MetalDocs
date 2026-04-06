# MASTER IMPLEMENTATION PLAN (90 Days)

## Purpose
Plano diretor decision-complete para implementar MetalDocs sem desvio arquitetural.

## Locked Defaults
- Horizonte: 90 dias.
- Governanca: forte pragmatica.
- Deploy v1: Docker Compose single-node (`192.168.0.3`).
- Auth v1: RBAC local com login interno e interface pronta para SSO futuro.
- Escopo inicial: backend-first com UI minima.
- IA/LLM: fora de escopo v1.
- Document authoring v1: browser editor template-assigned (CKEditor5), with DOCX/PDF as derived backend artifacts.

## Execution Phases

### Phase 0 (Weeks 1-2) - Foundation and Guardrails
Deliverables:
- Governance docs e templates finalizados.
- ADRs 0001-0007 publicados.
- OpenAPI policy oficial definida.
- PR checklist e governance checks habilitados.

Acceptance metrics:
- 100% dos PRs novos usando checklist.
- Nenhuma mudanca sem aderencia a `AGENTS.md`.

Blockers to advance:
- Falta de policy documentada para API/contracts/ADRs.

### Phase 1 (Weeks 3-7) - Functional Core
Scope:
- Modules: `documents`, `versions`, `workflow`, `iam`, `audit`, `search`.
- Fluxo minimo E2E: criar documento -> gerar versao -> transicionar workflow -> auditar -> consultar.

Deliverables:
- Contratos internos de eventos e erros aplicados.
- OpenAPI v1 cobrindo fluxo core.
- Testes unit e integration para regras centrais.

Acceptance metrics:
- Fluxo E2E core operando em ambiente local.
- Contract drift = 0 entre handlers e OpenAPI.

Blockers to advance:
- Qualquer violacao de boundary.
- Falha de rastreabilidade de auditoria.

### Phase 2 (Weeks 8-10) - Operability
Scope:
- Observabilidade basica (logs estruturados + metricas + tracing).
- Seguranca operacional (segredos, hardening minimo, rate limits basicos).
- Backup/restore e runbooks.
- Baseline de performance.

Deliverables:
- Dashboards minimos e alertas base.
- Runbook de deploy/rollback revisado.
- Teste de restore validado.

Acceptance metrics:
- SLO baseline definido e medido.
- Backup e restore executados com sucesso.

Blockers to advance:
- Sem runbook operacional validado.
- Sem evidencias de restore.

### Phase 3 (Weeks 11-13) - Hardening
Scope:
- Confiabilidade, resiliencia e readiness para escala.
- Reducao de risco tecnico e debt critica.
- Endurecimento de testes contract/integration/e2e.

Deliverables:
- Checklist de release completo.
- Revisao arquitetural consolidada.
- Plano de extracao futura por modulo (se necessario).

Acceptance metrics:
- Gates de qualidade todos verdes.
- 0 critical findings abertos em seguranca.

Blockers to close cycle:
- Falhas recorrentes de contrato.
- Evidencias ausentes de operacao segura.

## Implementation Order (strict)
1. Governance and docs guardrails.
2. OpenAPI and API error contract alignment.
3. Domain core (`documents`, `versions`).
4. Workflow + IAM + Audit.
5. Search and async/event consistency.
6. Operability and hardening.

## Key Risks and Mitigations
- Risk: deriva arquitetural por velocidade.
  - Mitigation: ADR/RFC gate + weekly architecture review.
- Risk: inconsistencias entre DB e eventos.
  - Mitigation: outbox + idempotency policy obrigatoria.
- Risk: quebra de contrato API.
  - Mitigation: OpenAPI-first + contract checks em PR.
- Risk: operacao fraca em ambiente local.
  - Mitigation: runbooks, backup/restore e smoke tests por fase.

## Weekly Rituals
- Segunda: planejamento da semana por fase.
- Quarta: architecture review curta (30 min).
- Sexta: demo tecnica + status de riscos.

