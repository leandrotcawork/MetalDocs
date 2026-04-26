# Local Dev Credentials

**Last verified:** 2026-04-26

## API login

Login endpoint: `POST /api/v1/auth/login`
Body field is `identifier` (not `username`).

| identifier | password | role | notes |
|---|---|---|---|
| `admin` | `AdminMetalDocs123!` | admin | bootstrapped on first start when no admin role exists |

Bootstrap triggers when: API starts and `metaldocs.iam_user_roles` has no `admin` role.
To re-bootstrap: truncate `metaldocs.auth_identities`, `metaldocs.iam_user_roles`, `metaldocs.iam_users` and restart API.

## API startup

Port: `8081`. Binary: `metaldocs-api.exe` (compiled from `./apps/api/cmd/metaldocs-api/...`).
Critical env vars that must be set explicitly via PowerShell (bash corrupts `PGPASSWORD` due to `<>` chars):

```
APP_PORT=8081
PGHOST=127.0.0.1
PGPORT=5433
PGDATABASE=metaldocs
PGUSER=metaldocs_app
PGPASSWORD=Lepa12<>!   ← set via $env:PGPASSWORD in PowerShell, never via bash source .env
```

See `.env` for full var list. Use `scripts/start-api-ps.ps1` if it exists, otherwise set manually.

## DB access

```
docker exec metaldocs-postgres psql -U metaldocs_app -d metaldocs -c "<query>"
```

User tables: `metaldocs.auth_identities`, `metaldocs.iam_users`, `metaldocs.iam_user_roles`
Document tables: `public.documents_v2`, `public.controlled_documents`
Template tables: `public.templates_v2_template`, `public.templates_v2_template_version`

## Process-area roles (approval authz)

The approval authz system (`authz.Require`) resolves capabilities via:

```sql
SELECT rc.capability
FROM metaldocs.role_capabilities rc
JOIN metaldocs.user_process_areas upa ON upa.role = rc.role
WHERE upa.user_id = ? AND upa.area_code = ? AND upa.effective_to IS NULL
```

Dev seed: `admin` user has **qms_admin** role in `general` area (applied by migration 0158).

| role | key capabilities |
|---|---|
| `author` | doc.submit, doc.edit_draft |
| `reviewer` | doc.signoff, doc.submit |
| `signer` | doc.signoff |
| `area_admin` | doc.submit, doc.signoff, doc.publish, membership.grant |
| `qms_admin` | all of the above + doc.obsolete, doc.reconstruct, route.admin |

**Historical note:** `user_process_areas_role_check` originally only allowed `viewer/editor/reviewer/approver` (0125) while `role_capabilities` used a different set. Migration 0158 widens the constraint to align them. See `decisions/` for the ADR.
