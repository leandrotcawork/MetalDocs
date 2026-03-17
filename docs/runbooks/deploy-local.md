# Runbook: Deploy Local (192.168.0.3)

## Pre-requisitos
- Docker e Docker Compose instalados.
- Portas 80, 5432, 6379, 9000, 9001 liberadas no host.

## Topologia
- `gateway` publica `http://192.168.0.3/`
- `web` serve a SPA operacional
- `api` fica atras do gateway em `/api/v1`
- `worker` processa notificacoes e lembretes

## Passos
1. Copiar `.env.example` para `.env` e ajustar valores.
   - Atualizar obrigatoriamente:
     - `POSTGRES_PASSWORD`
     - `PGPASSWORD`
     - `MINIO_ROOT_PASSWORD`
     - `METALDOCS_ATTACHMENTS_SIGNING_SECRET`
   - Para hardening basico, habilitar:
     - `METALDOCS_RATE_LIMIT_ENABLED=true`
     - ajustar `METALDOCS_RATE_LIMIT_WINDOW_SECONDS` e `METALDOCS_RATE_LIMIT_MAX_REQUESTS`.
2. Subir stack:
   - `docker compose -f deploy/compose/docker-compose.yml --env-file .env up -d --build`
3. Validar saude:
   - `curl http://192.168.0.3/api/v1/health/live`
   - `curl http://192.168.0.3/api/v1/health/ready`
4. Validar UI:
   - abrir `http://192.168.0.3/`

## Observacoes
- O modelo preferencial e same-origin via `gateway`; CORS fica desabilitado por padrao no deploy local.
- A API nao precisa ficar exposta diretamente no host fora do gateway.
- Escopo atual nao inclui IA.
