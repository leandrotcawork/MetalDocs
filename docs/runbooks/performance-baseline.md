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
   - `k6 run -e BASE_URL=http://127.0.0.1:8080/api/v1 -e USER_ID=admin-local scripts/perf/k6-baseline.js`

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
- Verificar endpoint `/api/v1/metrics` com sessao administrativa e logs estruturados.
- Identificar rotas com `errors` e maior `avgDurationMs`.
- Corrigir gargalo e repetir baseline.

## Evidencia oficial registrada (review reminder incremental query)
Execucao oficial: `2026-03-18` (America/Sao_Paulo).

- Migration aplicada:
  - `migrations/0037_add_documents_review_reminder_index.sql`
- Indice validado:
  - `idx_documents_review_reminder_window` em `metaldocs.documents`
- Query alvo do worker reminder:
  - filtro por janela `expiry_at >= now` e `<= now + N dias`
  - filtro por `status IN ('APPROVED','PUBLISHED')`
  - ordenacao `expiry_at, created_at`
- Plano observado no ambiente local:
  - `Seq Scan` com custo baixo devido cardinalidade pequena do dataset local
  - comportamento esperado; em volume maior o indice parcial reduz varredura global
- Resultado de arquitetura:
  - worker/reminder nao depende mais de `ListDocuments` O(N) por tick
  - backend usa leitura incremental com predicados de janela temporal

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

## Runbook: Light concurrency (Task 083)
Objetivo:
- Medir degradacao leve com 5-20 req/s nos endpoints criticos do editor.

Script:
- `scripts/perf/k6-light-concurrency.js`

Comando:
```powershell
k6 run -e BASE_URL=http://127.0.0.1:8081/api/v1 `
  -e USER_ID=admin-local `
  -e PROFILE_CODE=po `
  -e DOCUMENT_ID=CHANGE_ME_DOCUMENT_ID `
  -e PDF_AVAILABLE=false `
  scripts/perf/k6-light-concurrency.js
```

Se precisar usar o header tecnico `X-User-Id`, suba a API com:
```powershell
powershell -ExecutionPolicy Bypass -File scripts/dev-api-perf.ps1
```

Comando completo:
```powershell
k6 run -e BASE_URL=http://127.0.0.1:8080/api/v1 `
  -e USER_ID=admin-local `
  -e PROFILE_CODE=po `
  -e DOCUMENT_ID=CHANGE_ME_DOCUMENT_ID `
  scripts/perf/k6-light-concurrency.js
```

Notas:
- Se `DOCUMENT_ID` nao for informado, o script testa apenas as rotas de registry/taxonomia.
- Para validação oficial, usar um documento real e salvar o output de p95/p99.

## Evidencia registrada (light concurrency - tentativa 1)
Execucao: `2026-03-20 12:15` ate `12:19` (America/Sao_Paulo).

- Comando:
  - `k6 run -e BASE_URL=http://127.0.0.1:8081/api/v1 -e USER_ID=admin-local -e PROFILE_CODE=po -e DOCUMENT_ID=<id> scripts/perf/k6-light-concurrency.js`
- Resultado:
  - `http_req_failed`: `100.00%` (threshold `< 1%`) - falhou
  - `http_req_duration p95`: `713.23µs`
  - `http_req_duration p99`: `939.73µs`
- Observacao:
  - Todas as checks falharam (status != 200). Precisa investigar auth/permissions ou endpoint base antes de validar a metrica oficial.

## Evidencia registrada (light concurrency - execucao oficial)
Execucao: `2026-03-20 13:14` ate `13:18` (America/Sao_Paulo).

- Comando:
  - `k6 run -e BASE_URL=http://127.0.0.1:8081/api/v1 -e USER_ID=admin-local -e PROFILE_CODE=po -e DOCUMENT_ID=<id> -e PDF_AVAILABLE=false scripts/perf/k6-light-concurrency.js`
- Resultado:
  - `http_req_failed`: `0.00%` (threshold `< 1%`) - aprovado
  - `http_req_duration p95`: `20.18ms`
  - `http_req_duration p99`: `74.37ms`

## Runbook: UX profile change timings (Task 094)
Objetivo:
- Medir tempo percebido ao trocar o tipo documental no create (start -> render pronto).

Pre-requisitos:
- Frontend em modo dev.
- Ativar tracing local (`?trace=1` na URL ou `localStorage.setItem("md_trace","1")`).

Passos:
1. Abrir a tela de criar documento.
2. Alternar o tipo documental (PO -> IT -> RG).
3. No console, observar o grupo `[md-ux] profile-change:<code>` com os deltas.

Evidencia a registrar:
- Data/hora local.
- Sequencia de tempos (ms):
  - `profile-change-start`
  - `profile-schema-loaded`
  - `profile-governance-loaded`
  - `profile-form-updated`
  - `profile-render-ready`

Resultado esperado:
- Latencia percebida < 300ms em ambiente local (apenas referencia).
