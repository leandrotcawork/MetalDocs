# Phase 3 Release Checklist (Hardening)

## Objective
Checklist final para release com foco em confiabilidade, operacao segura e readiness de escala.

## 1. Quality Gates (must pass)
- [ ] `go test ./...` verde.
- [ ] `scripts/contract-baseline.ps1` aprovado.
- [ ] `scripts/security-baseline.ps1 -SkipGovulncheck` aprovado (`gosec` sem issues).
- [ ] `scripts/phase3-hardening-gate.ps1` aprovado.
- [ ] Baselines de performance (read + write concurrency) aprovados e registrados em runbook.

## 2. Contract and Architecture
- [ ] OpenAPI alinhada com handlers (sem drift).
- [ ] Nenhuma quebra de boundary em `internal/modules/*`.
- [ ] Eventos de outbox presentes para mutacoes relevantes.
- [ ] Auditoria append-only validada nos fluxos mutaveis.

## 3. Operability
- [ ] Backup/restore gate aprovado com evidencias em `backups/evidence/`.
- [ ] Runbooks atualizados: deploy, observability, security, backup/restore, performance.
- [ ] Health checks `live` e `ready` validados apos deploy.
- [ ] `/api/v1/metrics` respondendo e com rotas criticas observaveis.

## 4. Security
- [ ] `metaldocs_app` mantido em least-privilege.
- [ ] Role de backup dedicada (`metaldocs_backup`) ativa.
- [ ] Sem segredo hardcoded no repositorio.
- [ ] Pendencias de `govulncheck` documentadas quando ambiente bloquear internet.

## 5. Release Decision
- [ ] Go/No-Go aprovado com evidencias anexadas.
- [ ] Responsavel de operacao definido.
- [ ] Plano de rollback confirmado e testado.
