# Runbook: Deploy Local (192.168.0.3)

## Pre-requisitos
- Docker e Docker Compose instalados.
- Portas 80, 5433, 6379, 9000, 9001 liberadas no host.

## Topologia
- `gateway` publica `http://192.168.0.3/`
- `web` serve a SPA operacional
- `api` fica atras do gateway em `/api/v1`
- `worker` processa notificacoes e lembretes
- `minio` e o storage oficial de anexos/blobs
- `postgres` persiste em volume Docker nomeado (`metaldocs_postgres_data`)

## Passos
1. Copiar `.env.example` para `.env` e ajustar valores.
   - Atualizar obrigatoriamente:
     - `POSTGRES_HOST_PORT`
     - `POSTGRES_PASSWORD`
     - `PGPASSWORD`
     - `MINIO_ROOT_PASSWORD`
      - `METALDOCS_ATTACHMENTS_SIGNING_SECRET`
      - `METALDOCS_MINIO_SECRET_KEY`
      - `METALDOCS_AUTH_SESSION_SECRET`
      - `METALDOCS_BOOTSTRAP_ADMIN_PASSWORD`
      - `METALDOCS_AUTH_TRUSTED_ORIGINS`
   - Para hardening basico, habilitar:
     - `METALDOCS_RATE_LIMIT_ENABLED=true`
     - ajustar `METALDOCS_RATE_LIMIT_WINDOW_SECONDS` e `METALDOCS_RATE_LIMIT_MAX_REQUESTS`.
   - Confirmar runtime oficial:
     - `PGPORT=5433`
      - `METALDOCS_STORAGE_PROVIDER=minio`
      - `METALDOCS_MINIO_BUCKET=metaldocs-attachments`
      - `METALDOCS_MINIO_AUTO_CREATE_BUCKET=true`
      - `METALDOCS_AUTH_LEGACY_HEADER_ENABLED=false`
      - `METALDOCS_AUTH_ORIGIN_PROTECTION_ENABLED=true`
2. Subir stack:
   - `docker compose -f deploy/compose/docker-compose.yml --env-file .env up -d --build`
3. Validar saude:
   - `curl http://192.168.0.3/api/v1/health/live`
   - `curl http://192.168.0.3/api/v1/health/ready`
4. Validar UI:
   - abrir `http://192.168.0.3/`
   - login inicial:
     - username: `admin`
     - senha: valor de `METALDOCS_BOOTSTRAP_ADMIN_PASSWORD`
   - trocar a senha no primeiro acesso
5. Validar storage:
   - abrir `http://192.168.0.3/api/v1/health/live`
   - confirmar bucket `metaldocs-attachments` no console do MinIO `http://192.168.0.3:9001/`
6. Validar worker:
   - `docker compose -f deploy/compose/docker-compose.yml --env-file .env logs worker --tail 100`
   - confirmar logs `worker_batch result=completed`

## Observacoes
- O modelo preferencial e same-origin via `gateway`; CORS fica desabilitado por padrao no deploy local.
- A protecao de origem para sessao por cookie deve permanecer habilitada no runtime oficial.
- A API nao precisa ficar exposta diretamente no host fora do gateway.
- O Postgres Docker publica a porta de host configurada em `POSTGRES_HOST_PORT`; para evitar conflito com instalacoes locais, o padrao recomendado e `5433`.
- Storage local em filesystem nao e runtime oficial da stack Docker.
- `X-User-Id` existe apenas para testes tecnicos controlados e deve permanecer desligado no runtime oficial.
- Parar o container do Postgres nao apaga o banco; os dados permanecem no volume Docker.
- `docker compose down` preserva o volume do banco; `docker compose down -v` remove o volume e apaga os dados locais.
- Escopo atual nao inclui IA.
