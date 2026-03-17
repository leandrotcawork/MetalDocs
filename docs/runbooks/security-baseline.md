# Runbook: Security Baseline (SAST)

## Objetivo
Executar baseline de seguranca estatica no backend Go com:
- `gosec` (analise de padroes inseguros no codigo)
- `govulncheck` (vulnerabilidades conhecidas em dependencias e chamadas)

## Pre-requisitos
- Go instalado.
- Ferramentas instaladas no host:
  - `go install github.com/securego/gosec/v2/cmd/gosec@latest`
  - `go install golang.org/x/vuln/cmd/govulncheck@latest`
- Repositorio atualizado e compilavel.

## Execucao oficial
```powershell
powershell -ExecutionPolicy Bypass -File scripts/security-baseline.ps1
```

## Opcional
- Rodar somente `gosec`:
  - `powershell -ExecutionPolicy Bypass -File scripts/security-baseline.ps1 -SkipGovulncheck`
- Rodar somente `govulncheck`:
  - `powershell -ExecutionPolicy Bypass -File scripts/security-baseline.ps1 -SkipGosec`

## Evidencia minima para gate
- Arquivo JSON em `non_git/security/security_baseline_<timestamp>.json`.
- Saidas brutas em:
  - `non_git/security/gosec_<timestamp>.txt`
  - `non_git/security/govulncheck_<timestamp>.txt`
- Status final: `approved` ou `rejected`.

## Criterio de aceite v1
- `gosec` sem findings `HIGH/CRITICAL`.
- `govulncheck` sem vulnerabilidades exploraveis no caminho de execucao principal.
- Pipeline sem erro de execucao de ferramenta.

## Acoes se falhar
- Tratar findings por severidade (critical/high primeiro).
- Abrir issue com owner e prazo.
- Reexecutar baseline apos correcao.

## Evidencia oficial registrada
Execucao oficial (gosec): `2026-03-16 22:40` (America/Sao_Paulo).

- Comando:
  - `powershell -ExecutionPolicy Bypass -File scripts/security-baseline.ps1 -SkipGovulncheck`
- Resultado:
  - `gosec`: `approved`, `Issues: 0`
  - Evidencia: `non_git/security/security_baseline_20260317T014047Z.json`

Observacao operacional:
- `govulncheck` depende de acesso a `https://vuln.go.dev` para baixar a base de vulnerabilidades.
- Sem conectividade externa liberada, o passo `govulncheck` fica bloqueado por ambiente, nao por codigo.
