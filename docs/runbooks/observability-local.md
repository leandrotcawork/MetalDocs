# Observability Local Runbook

## Objetivo
Validar rapidamente logs estruturados e metricas HTTP RED (`rate`, `errors`, `duration`) em ambiente local.

## Pre-requisitos
- API MetalDocs em execucao.
- Banco e migrations aplicadas para o ambiente usado.

## Passos
1. Executar requests de smoke:
   - `GET /api/v1/health/live`
   - `GET /api/v1/search/documents` com e sem `X-User-Id`
   - `POST /api/v1/documents`
2. Consultar metricas:
   - `GET /api/v1/metrics`
3. Confirmar que as rotas aparecem com:
   - `requests > 0`
   - `errors > 0` para cenarios esperados de falha (`401`, `400`, `403`).
4. Verificar logs no console da API:
   - evento `http_request`
   - campos `trace_id`, `user_id`, `method`, `path`, `route`, `status`, `duration_ms`.

## Sinais de sucesso
- `/api/v1/metrics` retorna `200` com `items`.
- `route` normalizada para endpoints com IDs em path.
- Logs estruturados emitidos para todas as requests.

## Troubleshooting rapido
- `/metrics` vazio:
  - confirmar se houve requests apos subir a API.
- Campos faltando em log:
  - confirmar que o processo rodando e o binario mais recente.
- `401` inesperado:
  - revisar header `X-User-Id`.

