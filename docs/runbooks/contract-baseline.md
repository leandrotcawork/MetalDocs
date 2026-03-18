# Runbook: Contract Baseline (OpenAPI Coverage Smoke)

## Objetivo
Garantir baseline de contrato da API com teste smoke dos endpoints v1 definidos no fluxo principal.

## Escopo atual
- `GET /api/v1/health/live`
- `GET /api/v1/health/ready`
- `GET /api/v1/metrics` (admin autenticado)
- `POST /api/v1/documents`
- `GET /api/v1/documents`
- `GET /api/v1/search/documents`
- `POST /api/v1/workflow/documents/{documentId}/transitions`
- `GET /api/v1/documents/{documentId}/versions`
- `POST /api/v1/iam/users/{userId}/roles`

## Execucao oficial
```powershell
powershell -ExecutionPolicy Bypass -File scripts/contract-baseline.ps1
```

## Evidencia
- JSON em `non_git/contract/contract_baseline_<timestamp>.json`
- Status esperado: `approved`

## Criterio de aceite
- `go test ./tests/contract -count=1` verde.
- Nenhum endpoint smoke retornando `404` ou erro inesperado no fluxo autenticado.
- `GET /api/v1/metrics` sem autenticacao retorna `401` no runtime oficial.

## Acoes se falhar
- Validar se houve drift entre OpenAPI e handlers.
- Corrigir rota/handler ou atualizar contrato OpenAPI.
- Reexecutar baseline.
