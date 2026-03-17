# Architecture Review (2026-Q1)

## Executive Summary
- Status geral: **aprovado com observacoes**.
- Arquitetura alvo (modular monolith) esta aderente no fluxo principal.
- Principais riscos residuais estao em conectividade para `govulncheck` e maturidade de observabilidade distribuida.

## Review Scope
- Boundaries por modulo (`documents`, `versions`, `workflow`, `iam`, `audit`, `search`).
- Contratos imutaveis (OpenAPI, append-only audit, immutable versions, backend authz).
- Operabilidade (backup/restore, observabilidade, performance baseline, security baseline).

## Findings
1. **Boundary discipline**
- Evidencia: organizacao por `domain/application/infrastructure/delivery` mantida.
- Status: conforme guardrails.

2. **Contract stability**
- Evidencia: OpenAPI v1 definida e rotas principais implementadas.
- Evidencia adicional: check de governanca refinado para evitar falso positivo em mudancas de bootstrap.
- Status: conforme.

3. **Boundary enforcement**
- Evidencia: gate dedicado `module-boundaries` no CI e script `check-module-boundaries.ps1`.
- Status: conforme guardrails, com bloqueio automatico em PR.

4. **Event consistency**
- Evidencia: outbox aplicado para eventos de dominio criticos.
- Status: conforme ADR de idempotencia e outbox.

5. **Operational readiness**
- Evidencia: backup/restore gate aprovado com restore em DB isolado.
- Evidencia: performance read/write baseline aprovadas em k6.
- Evidencia: `gosec` zerado apos hardening.
- Evidencia adicional: workflow manual `release-readiness` com artifact de evidencias.
- Risco residual: `govulncheck` depende de rede externa.

## Decisions and Actions
- D1: Manter modular monolith para ciclo atual (sem extracao imediata).
- D2: Manter hardening gate como obrigatorio antes de release.
- D3: Manter gate de boundary como status check obrigatorio na branch `main`.
- D4: Registrar politica de execucao `govulncheck` em ambiente com saida para internet.
- D5: Evoluir observabilidade para stack OTEL/Prometheus no proximo ciclo.

## Exit Criteria (Phase 3)
- [x] Checklist de release completo publicado.
- [x] Revisao arquitetural consolidada publicada.
- [x] Plano de extracao por modulo publicado.
- [x] Gate de hardening executavel implementado.
- [x] Gate de boundary dedicado no CI publicado.
- [x] Gate de release readiness com evidencias publicado.
