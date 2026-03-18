# Observability Local Runbook

## Objetivo
Validar rapidamente logs estruturados, readiness e metricas HTTP/runtime em ambiente local.

## Pre-requisitos
- API MetalDocs em execucao.
- Banco e migrations aplicadas para o ambiente usado.

## Passos
1. Executar requests de smoke:
   - `GET /api/v1/health/live`
   - `POST /api/v1/auth/login`
   - `GET /api/v1/auth/me`
   - `GET /api/v1/search/documents`
   - `POST /api/v1/documents`
2. Consultar metricas:
   - `GET /api/v1/metrics`
   - `GET /api/v1/health/ready`
3. Confirmar que as rotas aparecem com:
   - `requests > 0`
   - `errors > 0` para cenarios esperados de falha (`401`, `400`, `403`).
4. Confirmar no payload:
   - `/api/v1/health/ready` com `checks` para `repository`, `storage` e `auth`
   - `/api/v1/metrics` com bloco `runtime`
   - `runtime.auth.users`
   - `runtime.auth.sessions`
   - `runtime.worker.outbox`
5. Verificar logs no console da API:
   - evento `http_request`
   - campos `trace_id`, `user_id`, `method`, `path`, `route`, `status`, `duration_ms`.
6. Verificar logs do worker:
   - evento `worker_event`
   - campos `event_id`, `event_type`, `attempt_count`, `result`, `trace_id`
   - evento `worker_batch` com `processed`, `failed`, `dead_lettered`

## Sinais de sucesso
- `/api/v1/metrics` retorna `200` com `items` e `runtime`.
- `/api/v1/health/ready` retorna `200` com `checks` estruturados.
- `route` normalizada para endpoints com IDs em path.
- Logs estruturados emitidos para todas as requests.

## Troubleshooting rapido
- `/metrics` vazio:
  - confirmar se houve requests apos subir a API.
- `/health/ready` com `503`:
  - revisar conectividade com Postgres e provider de storage configurado.
- Campos faltando em log:
  - confirmar que o processo rodando e o binario mais recente.
- `401` inesperado:
  - revisar sessao/cookie e `Origin`/`Referer` no browser.

## Consultas operacionais da fila
Pendentes/claimable:
```sql
SELECT COUNT(*) AS pending_events
FROM metaldocs.outbox_events
WHERE published_at IS NULL
  AND dead_lettered_at IS NULL
  AND (next_attempt_at IS NULL OR next_attempt_at <= NOW());
```

Em DLQ:
```sql
SELECT COUNT(*) AS dead_lettered_events
FROM metaldocs.outbox_events
WHERE dead_lettered_at IS NOT NULL;
```

Ultimas falhas:
```sql
SELECT event_id, event_type, attempt_count, last_error, last_attempt_at, next_attempt_at, dead_lettered_at
FROM metaldocs.outbox_events
WHERE last_error IS NOT NULL
ORDER BY last_attempt_at DESC
LIMIT 20;
```
