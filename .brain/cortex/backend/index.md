---
id: cortex-backend-index
title: Backend Cortex Index
region: cortex/backend
tags: ["go", "api", "http", "iam", "documents", "carbone", "minio"]
weight: 0.70
created_at: "2026-03-26T10:00:00Z"
updated_at: "2026-03-26T10:00:00Z"
---

# Backend Cortex

## Stack

- **Language**: Go 1.24
- **HTTP**: `net/http` stdlib (no framework)
- **DB driver**: `pgx/v5` (raw SQL, no ORM)
- **Object storage**: `minio-go/v7`
- **IDs**: `rs/xid` (compact unique IDs)
- **JSON**: `goccy/go-json`

## Domain Modules

### documents
- CRUD for controlled documents (PO, IT, RG, FM profiles)
- Content versioning (native JSON content + Carbone PDF)
- `saveDocumentContentNative` → saves JSON → triggers Carbone render → stores PDF in MinIO → returns `pdfUrl`

### iam
- Identity and Access Management
- RBAC: roles + permissions
- `0002_init_iam_rbac.sql` schema

### audit
- Audit event recording per user action
- `0004_init_audit_events.sql` schema

## Key Endpoints

| Method | Pattern | Description |
|--------|---------|-------------|
| POST | `/documents/{id}/content/native` | Save form content + trigger PDF render |
| GET | `/documents/{id}/pdf` | Get latest PDF URL |
| POST | `/documents/{id}/pdf/render` | Force Carbone re-render |

## Carbone Integration

- Carbone Node.js server runs as a separate service
- Go API POSTs JSON data + template name to Carbone
- Carbone returns PDF → Go stores in MinIO
- MinIO presigned URL returned to frontend as `pdfUrl`

## Platform

- `internal/platform/db/` — pgx pool initialization, transaction helpers
- `internal/platform/storage/` — MinIO client wrapper
- `internal/platform/middleware/` — auth, logging, CORS
