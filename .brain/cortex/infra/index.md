---
id: cortex-infra-index
title: Infra Cortex Index
region: cortex/infra
tags: ["docker", "nginx", "minio", "carbone", "compose", "deployment"]
weight: 0.60
created_at: "2026-03-26T10:00:00Z"
updated_at: "2026-03-26T10:00:00Z"
---

# Infra Cortex

## Stack

- **Docker Compose** — local development and deployment
- **nginx** — reverse proxy (routes `/api` to Go, `/` to Vite SPA)
- **MinIO** — S3-compatible object storage (PDF files)
- **Carbone** — Node.js document rendering server (DOCX → PDF)

## Deploy Layout

```
deploy/
├── compose/   # docker-compose.yml and variants
├── docker/    # Dockerfiles for each service
└── nginx/     # nginx.conf, site configs
```

## Services

| Service | Port | Purpose |
|---------|------|---------|
| Go API | 8080 | Backend HTTP API |
| Vite dev | 5173 | Frontend dev server |
| nginx | 80/443 | Reverse proxy |
| PostgreSQL | 5432 | Database |
| MinIO | 9000/9001 | Object storage (API/console) |
| Carbone | 4000 | Document rendering |

## Carbone

- Runs as Node.js service
- Templates stored at `carbone/templates/`
- Receives JSON data + template name → returns PDF binary
- Go API intermediates: POST data → store PDF in MinIO → return presigned URL

## MinIO

- Bucket: `metaldocs-pdfs` (or similar)
- Presigned URLs for PDF access (time-limited)
- `minio-go/v7` client in Go

## Nginx Routing

- `GET /api/*` → Go API (`:8080`)
- `GET /*` → Vite SPA (`:5173` dev / static build prod)
