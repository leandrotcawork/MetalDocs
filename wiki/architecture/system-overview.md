# System Overview

> **Last verified:** 2026-04-26
> **Scope:** Services, ports, data flow, infra at a glance.
> **Out of scope:** Per-module deep dives (see `modules/*`), DB schema details (see `data-model.md`).
> **Key files:**
> - `deploy/compose/docker-compose.yml` - service topology
> - `apps/api/cmd/metaldocs-api/main.go` - Go entry point
> - `frontend/apps/web/vite.config.ts` - frontend dev server
> - `internal/platform/bootstrap/` - service wiring

---

## Services

| Service | Port | Tech | Purpose |
|---------|------|------|---------|
| **metaldocs-api** | 8081 | Go | REST API. All business logic. |
| **metaldocs-web** | 4174 | React + Vite | SPA. Talks to api. |
| **postgres** | 5432 | Postgres 16 | Primary datastore |
| **minio** | 9000 (s3), 9001 (console) | MinIO | S3-compat blob store: DOCX bodies, final artifacts |
| **gotenberg** | 3000 (internal) | Gotenberg 8 | DOCX -> PDF rendering |

All run in Docker compose: `make up` -> `docker compose -f deploy/compose/docker-compose.yml up -d`.

## Module topology (backend)

Each module under `internal/modules/` is self-contained:
- `domain/` - types, value objects
- `application/` - services (use cases)
- `delivery/http/` - HTTP handlers + routes
- `repository/` or `infrastructure/` - Postgres adapters
- `module.go` - DI wiring

Modules:
- `templates_v2` - template authoring + schema versioning
- `documents_v2` - document instances, creation-time snapshots, freeze, view, approval
- `taxonomy` - profiles, areas, departments, subjects
- `iam` - users, roles, capabilities, area memberships
- `auth` - authn (token validation)
- `approval` (under documents_v2) - routes, signoffs
- `render/fanout` + `render/resolvers` - substitution + DOCX/PDF generation
- `registry` - controlled-document codes, sequence counters
- `workflow` - approval workflow definitions
- `jobs/*` - background jobs (effective-date publisher, idempotency janitor, scheduler, watchdog)

## Frontend topology

`frontend/apps/web/src/`:
- `App.tsx` - root, routing
- `routing/workspaceRoutes.ts` - route table
- `features/` - one folder per feature area
  - `templates/v2/` - template list + author page
  - `documents/v2/` - document list + create wizard + editor
  - `taxonomy/` - profile/area admin
  - `iam/` - user/membership admin
  - `approval/` - inbox, etag/mutation client
  - `auth/` - session
  - `notifications/` - bell + toasts
  - `documents/runtime/` - schema runtime adapters
- `components/` - shared UI
- `editor-adapters/` - eigenpal integration glue
- `api/` - fetch wrappers per domain

Shared packages:
- `packages/editor-ui/` - `MetalDocsEditor` wrapper around eigenpal
- `packages/shared-tokens/` - shared utilities (parser, OOXML, grammar, diff) - used by frontend + spike

## Data flow: template authoring -> document -> freeze

1. **Author** opens `TemplateAuthorPage` -> loads schema + body DOCX
2. Edits in eigenpal editor -> autosave (1500ms debounce) -> `PUT /templates/{id}/versions/{v}/body` (DOCX bytes) + schema PUT
3. Author submits -> `POST /templates/{id}/versions/{v}/submit` -> status=in_review
4. Reviewer approves -> `POST /approve` -> status=approved
5. **Document creation:** end user picks a controlled doc -> wizard creates the `documents` row. `application.SnapshotService`, wired via `documents_v2.Dependencies.SnapshotReader`/`SnapshotWriter`, populates `placeholder_schema_snapshot`, `placeholder_schema_hash`, `composition_config_snapshot`, `composition_config_hash`, `body_docx_snapshot_s3_key`, and `body_docx_hash`.
6. For catalog-only templates, `composition_config_snapshot` stores `{}`.
7. Submit -> `POST /documents/{id}/submit` -> `under_review`. Migration `0152`'s `enforce_snapshot_on_submit_trg` trigger requires all six snapshot columns to be non-NULL before draft -> under_review.
8. Approves -> `POST /documents/{id}/approve` -> triggers `freeze`:
   - Use the creation-time snapshots as the immutable render inputs
   - Call fanout -> renders final DOCX (substituted) + PDF
   - Upload to MinIO under `documents/{id}/final.{docx,pdf}`
   - Persist final_docx_s3_key, hashes
9. **View:** `GET /documents/{id}/view` -> returns signed URL for PDF

## Cross-refs

- Per-module deep dives: `modules/*.md`
- Workflow walkthroughs: `workflows/*.md`
- DB schema: `architecture/data-model.md` (TBD)
