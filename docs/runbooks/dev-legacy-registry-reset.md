# Runbook: Dev Legacy Registry Reset

## Objective
Reset legacy document registry data in local/dev environments without shipping destructive SQL in the official migration chain.

## Scope
- Local/dev only.
- Not allowed for production environments.
- Destructive operation: deletes legacy documents and related records.

## Inputs
- Running Postgres container/service for local environment.
- `scripts/sql/dev_reset_legacy_document_registry.sql`.

## Execution (Docker local)
```powershell
Get-Content -Raw scripts/sql/dev_reset_legacy_document_registry.sql |
  docker exec -i metaldocs-postgres psql -U metaldocs_app -d metaldocs -v ON_ERROR_STOP=1
```

## Validation
```powershell
docker exec -i metaldocs-postgres psql -U metaldocs_app -d metaldocs -c "select code, name from metaldocs.document_profiles order by code;"
docker exec -i metaldocs-postgres psql -U metaldocs_app -d metaldocs -c "select code, name from metaldocs.document_types order by code;"
```

Expected:
- `po`, `it`, `rg` only for profiles/types.
- No legacy profile/type rows.

## Safety Notes
- This runbook is intentionally outside official migrations to preserve:
  - additive-first migration policy (`ADR-0007`)
  - append-only audit invariants.
