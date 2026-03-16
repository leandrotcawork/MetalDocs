# Runbook: Deploy Local (192.168.0.3)

## Pre-requisitos
- Docker e Docker Compose instalados.
- Portas 80, 5432, 6379, 9000, 9001, 8080 liberadas no host.

## Passos
1. Copiar `.env.example` para `.env` e ajustar valores.
   - Para hardening basico, habilitar:
     - `METALDOCS_RATE_LIMIT_ENABLED=true`
     - ajustar `METALDOCS_RATE_LIMIT_WINDOW_SECONDS` e `METALDOCS_RATE_LIMIT_MAX_REQUESTS`.
2. Subir stack:
   - `docker compose -f deploy/compose/docker-compose.yml --env-file .env up -d`
3. Validar saude:
   - `curl http://192.168.0.3:8080/api/v1/health/live`
   - `curl http://192.168.0.3:8080/api/v1/health/ready`

## Observacoes
- `api` e `worker` estao como placeholder ate o bootstrap da aplicacao.
- Escopo atual nao inclui IA.
