---
id: cortex-database-index
title: Database Cortex Index
region: cortex/database
tags: ["postgresql", "migrations", "sql", "pgx", "schema"]
weight: 0.65
created_at: "2026-03-26T10:00:00Z"
updated_at: "2026-03-26T10:00:00Z"
---

# Database Cortex

## Stack

- **PostgreSQL** (version: inferred from pgx/v5 compatibility)
- **pgx/v5** — Go driver, raw SQL, no ORM
- **Migrations**: sequential numbered SQL files in `migrations/`

## Schema Overview

### migrations/

| File | Description |
|------|-------------|
| `0001_init_documents.sql` | Core documents table, profiles, content, versions |
| `0002_init_iam_rbac.sql` | Users, roles, permissions, role assignments |
| `0003_iam_role_code_constraint.sql` | Unique constraint on role codes |
| `0004_init_audit_events.sql` | Audit event log |
| `0005_grant_workflow_audit_privileges.sql` | Role privileges for audit |

## Key Tables

- `documents` — core document records (id, profile_code, title, status, version)
- `document_contents` — native JSON form content per version
- `document_pdfs` — MinIO references for rendered PDFs
- `users` / `roles` / `permissions` — IAM RBAC
- `audit_events` — user action log

## Conventions

- Forward-only migrations (no rollback scripts)
- All IDs: `xid` (compact, time-sortable)
- Timestamps: `TIMESTAMPTZ`
- Soft deletes via `status` field (not hard DELETE)

## Connection

- Pool managed by `internal/platform/db/`
- pgx pool with context-aware queries
- Transactions via platform helpers
