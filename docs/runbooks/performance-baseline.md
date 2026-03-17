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

## Evidencia oficial registrada
Execucao oficial: `2026-03-16 22:09:02` ate `22:09:06` (America/Sao_Paulo), com duracao total do teste de 5 minutos.

- Comando executado:
  - `C:\Program Files\k6\k6.exe run scripts/perf/k6-baseline.js`
- Ambiente:
  - API em `http://localhost:8080/api/v1`
  - `METALDOCS_REPOSITORY=postgres`
  - `PGUSER=metaldocs_app`
- Resultado:
  - `http_req_failed`: `0.00%` (threshold `< 1%`) - aprovado
  - `http_req_duration p95`: `1.88ms` (threshold `< 1200ms`) - aprovado
  - `http_req_duration p99`: `2.52ms` (threshold `< 2000ms`) - aprovado
  - `http_reqs`: `17098` (56.99 req/s)
  - `iterations`: `8549` (28.50 iter/s)
- Status do gate de performance:
  - `aprovado`

## Acoes se falhar
- Verificar endpoint `/api/v1/metrics` e logs estruturados.
- Identificar rotas com `errors` e maior `avgDurationMs`.
- Corrigir gargalo e repetir baseline.

## Evidencia oficial registrada (write concurrency)
Execucao oficial: `2026-03-16 22:19` ate `22:23` (America/Sao_Paulo), com duracao total de ~4 minutos.

- Script:
  - `scripts/perf/k6-write-concurrency.js`
- Comando:
  - `C:\Program Files\k6\k6.exe run --summary-export non_git/k6-write-summary.json scripts/perf/k6-write-concurrency.js`
- Fluxo validado por iteracao:
  - `POST /documents` (cria documento)
  - `POST /workflow/documents/{documentId}/transitions` (DRAFT -> IN_REVIEW)
  - `GET /documents/{documentId}/versions` (confere versao)
- Resultado:
  - `iterations`: `3202`
  - `http_reqs`: `9606` (~40.38 req/s)
  - `http_req_failed`: `0.00%` (threshold `< 1%`) - aprovado
  - `http_req_duration p95`: `185.14ms` (threshold `< 1500ms`) - aprovado
  - `http_req_duration p99`: `482.77ms` (threshold `< 2500ms`) - aprovado
  - `create status 201`: `3202/3202`
  - `transition status 200`: `3202/3202`
  - `versions status 200`: `3202/3202`
- Status:
  - `aprovado`
