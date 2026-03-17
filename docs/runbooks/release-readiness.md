# Runbook: Release Readiness (Phase 3)

## Objetivo
Executar validacao final de Go/No-Go para merge/release com um comando unico.

## O que valida
- `check-governance` (conformidade de mudanca com regras do projeto).
- `phase3-hardening-gate` (testes + contract baseline + security baseline).

## Execucao oficial
```powershell
powershell -ExecutionPolicy Bypass -File scripts/phase3-release-readiness.ps1 -BaseRef origin/main
```

## Execucao local (sem remoto)
```powershell
powershell -ExecutionPolicy Bypass -File scripts/phase3-release-readiness.ps1 -BaseRef HEAD~1
```

## Execucao via GitHub Actions (manual)
Workflow: `release-readiness` (`.github/workflows/release-readiness.yml`)

Inputs:
- `base_ref` (default: `origin/main`)
- `skip_govulncheck` (default: `true`)

Resultado esperado:
- Job `readiness` verde.
- Artifact `phase3-release-evidence` com JSONs de evidencia.

## Evidencia
- JSON final:
  - `non_git/release/phase3_release_readiness_<timestamp>.json`
- Referencias internas:
  - evidencia do hardening gate em `non_git/hardening/*.json`
  - evidencia de contract em `non_git/contract/*.json`
  - evidencia de security em `non_git/security/*.json`

## Criterio de aceite
- Status final `approved`.
- `governance_check = approved`.
- `hardening_gate = approved`.

## Acoes se falhar
- Corrigir check que reprovou.
- Reexecutar `phase3-release-readiness`.
- So aprovar merge/release com status final `approved`.
