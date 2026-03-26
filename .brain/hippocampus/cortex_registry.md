---
id: hippocampus-cortex-registry
title: MetalDocs Cortex Registry
region: hippocampus
tags: ["registry", "cortex", "regions"]
weight: 0.80
created_at: "2026-03-26T10:00:00Z"
updated_at: "2026-03-26T10:00:00Z"
---

# Cortex Registry

Maps each cortex region to its domain, key files, and index sinapse.

## backend

- **Domain**: Go HTTP API, IAM, document logic, Carbone integration
- **Key paths**: `api/`, `internal/modules/`, `internal/platform/`
- **Index sinapse**: `cortex/backend/index.md` (id: `cortex-backend-index`)
- **Lessons path**: `cortex/backend/lessons/`

## frontend

- **Domain**: React SPA — dashboard, content-builder, document creation
- **Key paths**: `frontend/apps/web/src/`
- **Index sinapse**: `cortex/frontend/index.md` (id: `cortex-frontend-index`)
- **Lessons path**: `cortex/frontend/lessons/`

## database

- **Domain**: PostgreSQL schema, migrations, queries
- **Key paths**: `migrations/`, `internal/platform/db/`
- **Index sinapse**: `cortex/database/index.md` (id: `cortex-database-index`)
- **Lessons path**: `cortex/database/lessons/`

## infra

- **Domain**: Docker Compose, nginx, MinIO, Carbone server
- **Key paths**: `deploy/`, `carbone/`
- **Index sinapse**: `cortex/infra/index.md` (id: `cortex-infra-index`)
- **Lessons path**: `cortex/infra/lessons/`
