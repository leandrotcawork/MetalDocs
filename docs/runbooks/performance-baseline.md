# Runbook: Performance Baseline (k6)

## Objetivo
Estabelecer baseline inicial de latencia e erro para a API MetalDocs no ambiente local.

## Pre-requisitos
- API MetalDocs em execucao.
- k6 instalado no host.
- Usuario com permissao de leitura (`admin-local` por padrao no script).

## Execucao
1. Rodar teste:
   - `k6 run scripts/perf/k6-baseline.js`
2. Opcional com variaveis:
   - `k6 run -e BASE_URL=http://192.168.0.3:8080/api/v1 -e USER_ID=admin-local scripts/perf/k6-baseline.js`

## Criterios (thresholds no script)
- `http_req_failed < 1%`
- `p95 < 1200ms`
- `p99 < 2000ms`

## Evidencia minima para gate Phase 2
- Data/hora da execucao.
- Throughput medio.
- `p95` e `p99`.
- Taxa de erro.

## Acoes se falhar
- Verificar endpoint `/api/v1/metrics` e logs estruturados.
- Identificar rotas com `errors` e maior `avgDurationMs`.
- Corrigir gargalo e repetir baseline.

