# C4 Baseline

## Context
- Usuario interno acessa MetalDocs via navegador.
- MetalDocs integra com Postgres, Redis e MinIO.

## Container
- API: exposicao HTTP e regras de negocio.
- Worker: tarefas assincronas com retry, backoff e DLQ para eventos internos.
- Postgres: dados transacionais e auditoria.
- Redis: cache e filas leves.
- MinIO: storage oficial de blobs de documento no runtime real.

## Component
- Modulos de dominio: `documents`, `versions`, `workflow`, `iam`, `audit`, `search`.
- Plataforma compartilhada: `db`, `cache`, `messaging`, `security`, `observability`.
