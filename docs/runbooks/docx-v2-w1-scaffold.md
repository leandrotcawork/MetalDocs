# Runbook — docx-v2 W1 scaffold

This runbook covers the greenfield docx-editor platform scaffold introduced
in W1 (see `docs/superpowers/plans/2026-04-18-docx-editor-w1-scaffold.md`).
It is referenced by `scripts/check-governance.ps1` which requires a runbook
entry for any `deploy/` or `scripts/` change.

## What W1 adds

- New Fastify service `apps/docgen-v2` exposing only `/health` (port 3100).
- New Postgres tables 0101–0108 (templates, template_versions, documents_v2,
  editor_sessions, document_revisions, autosave_pending_uploads,
  document_checkpoints, template_audit_log).
- New `METALDOCS_DOCX_V2_ENABLED` feature flag wired through Go config and
  the `GET /api/v1/feature-flags` response.
- Empty npm workspace packages under `packages/*` (business logic arrives
  in W2–W4 plans).

## Operator bring-up

```
export DOCGEN_V2_SERVICE_TOKEN=$(openssl rand -hex 24)
docker compose -f deploy/compose/docker-compose.yml \
  up -d postgres minio gotenberg docgen-v2
bash scripts/docx-v2-verify-migrations.sh
bash scripts/docx-v2-seed-minio.sh
curl -f http://127.0.0.1:3100/health
```

## Rollback

W1 is pure-additive: no existing table altered, no existing service
modified. Rollback = drop the 8 new tables in reverse FK order:

```
DROP TABLE template_audit_log, document_checkpoints, autosave_pending_uploads,
           document_revisions, editor_sessions, documents_v2,
           template_versions, templates;
```

…and remove `docgen-v2` from the compose file. No data loss beyond new-path
state.

## Known limits (carried to W2)

- Per-tenant flag resolution not implemented; `METALDOCS_DOCX_V2_ENABLED`
  is global-only.
- `/render/docx` and other docgen-v2 routes return 404 by design.
- OOXML parser / validators arrive in W2.
