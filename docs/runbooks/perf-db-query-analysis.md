# Runbook: DB Query Analysis (profiles/schemas/metadata)

## Objective
Identify slow queries related to document profiles, schemas, and document metadata.

## Preconditions
- Postgres Docker running (dev local uses port `5433`).
- `psql` available on host.
- A few documents/profiles seeded (Metal Nobre seeds are enough).

## What this runbook covers
- `ListDocumentProfiles` query with active schema join.
- `ListDocumentProfileSchemas` query.
- `GetDocument` query that carries `metadata_json`.
- Index inventory for profiles/schemas/documents.

## How to run
1. Open a PowerShell in the repo root.
2. Update the variables inside the SQL file if needed:
   - `profile_code` (default: `po`)
   - `document_id` (set any existing document ID)

3. Execute:
```powershell
psql -h 127.0.0.1 -p 5433 -U metaldocs_app -d metaldocs -f scripts/sql/perf_db_query_analysis.sql
```

## How to interpret results
Focus on:
- `Seq Scan` on `document_profile_schema_versions` or `documents` when the table grows.
- `Sort` steps with high memory or disk usage.
- `Rows Removed by Filter` large numbers for basic lookups.
- `actual time` for p95/p99 endpoints (compare with HTTP logs).

## Evidence to capture
- `EXPLAIN (ANALYZE, BUFFERS)` outputs for the 3 queries.
- Index list for `document_profiles`, `document_profile_schema_versions`, `documents`.
- A short summary of any slow steps and candidate indexes.

## Next steps if slow
- Add or adjust indexes based on predicates and ordering.
- Re-run the file and confirm plan shift (index scan vs seq scan).
- Record findings in `docs/runbooks/performance-baseline.md` if changes are approved.
