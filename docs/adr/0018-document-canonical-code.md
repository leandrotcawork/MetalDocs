# ADR-0018: Canonical Document Code

## Status
Accepted

## Context
Document names must follow a stable canonical code across the system (e.g. `PO-001`), derived from the document profile and a sequential number.
Today, the UI infers codes from `documentId` or ad-hoc metadata, which is inconsistent and not guaranteed to be stable.
We need a server-side, canonical code that is generated once and returned in all document responses.

This change adds new fields to the document schema and API responses, which is a public contract and schema change.

## Decision
- Add `document_sequence` and `document_code` to `metaldocs.documents`.
- Generate a monotonic sequence per `document_profile_code` and derive `document_code` as `UPPER(profile_code) + '-' + 3-digit sequence`.
- Persist `document_code` and expose it in all document list/detail/search responses.
- Frontend uses `document_code` to render `PO-001-<Title>` consistently; fallback only if missing.

## Consequences
- Positive:
  - One canonical code across all views and consumers.
  - No UI heuristics based on `documentId`.
  - Deterministic backfill for existing documents.
- Negative:
  - Adds DB columns and a new sequence table.
  - Minor migration cost and additional data fields in responses.

## Acceptance test
- `go test ./...`
- `cd frontend/apps/web; npm run build`
