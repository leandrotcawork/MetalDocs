# Runbook: Manual PostgreSQL Setup for MetalDocs

Este procedimento e manual para executar no seu servidor PostgreSQL.

## 1) Criar role, database e schema isolados

```sql
CREATE ROLE metaldocs_app LOGIN PASSWORD ''CHANGE_ME_STRONG_PASSWORD'';
CREATE DATABASE metaldocs OWNER metaldocs_app;
\c metaldocs
CREATE SCHEMA IF NOT EXISTS metaldocs AUTHORIZATION metaldocs_app;
ALTER ROLE metaldocs_app IN DATABASE metaldocs
  SET search_path = metaldocs, public;
```

## 2) Aplicar migrations

Execute em ordem:
1. `migrations/0001_init_documents.sql`
2. `migrations/0002_init_iam_rbac.sql`
3. `migrations/0003_iam_role_code_constraint.sql`

## 3) Seed inicial de RBAC para teste

```sql
INSERT INTO metaldocs.iam_users (user_id, display_name)
VALUES ('admin-local', 'Admin Local')
ON CONFLICT (user_id) DO NOTHING;

INSERT INTO metaldocs.iam_user_roles (user_id, role_code, assigned_by)
VALUES ('admin-local', 'admin', 'manual-seed')
ON CONFLICT (user_id, role_code) DO NOTHING;
```

## 4) Configurar MetalDocs

```env
METALDOCS_REPOSITORY=postgres
PGHOST=127.0.0.1
PGPORT=5432
PGDATABASE=metaldocs
PGUSER=metaldocs_app
PGPASSWORD=CHANGE_ME_STRONG_PASSWORD
PGSSLMODE=disable
METALDOCS_AUTH_ENABLED=true
METALDOCS_AUTHZ_CACHE_TTL_SECONDS=30
```

## 5) Smoke auth

```bash
curl http://localhost:8080/api/v1/health/ready
curl http://localhost:8080/api/v1/documents
curl -H "X-User-Id: admin-local" http://localhost:8080/api/v1/documents
```

## 6) Smoke admin IAM

```bash
curl -X POST \
  -H "X-User-Id: admin-local" \
  -H "Content-Type: application/json" \
  -d '{"displayName":"Editor User","role":"editor"}' \
  http://localhost:8080/api/v1/iam/users/editor-user/roles
```

Apos atribuir role, o cache do usuario e invalidado automaticamente no processo.
