# Runbook: Branch Protection (main)

## Objetivo
Garantir que nenhuma mudanca entre em `main` sem passar pelos gates de governanca e hardening da Phase 3.

## Regras recomendadas para `main`
- Require a pull request before merging.
- Require approvals: minimo 1.
- Dismiss stale pull request approvals when new commits are pushed.
- Require status checks to pass before merging.
- Require branches to be up to date before merging.
- Include administrators: enabled.
- Restrict who can push directly to `main`: enabled.

## Status checks obrigatorios
- `governance-check / check`
- `phase3-hardening-gate / hardening`

## Como configurar (GitHub)
1. Abrir repo -> `Settings` -> `Branches`.
2. Em `Branch protection rules`, criar regra para `main`.
3. Marcar as opcoes acima.
4. Em `Status checks`, selecionar:
   - `governance-check / check`
   - `phase3-hardening-gate / hardening`
5. Salvar regra.

## Evidencia minima
- Screenshot da regra ativa em `Branches`.
- PR de teste bloqueado quando check falha.
- PR de teste liberado quando checks passam.
