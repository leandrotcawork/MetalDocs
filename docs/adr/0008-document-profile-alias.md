# ADR-0008: Document Profile Alias

## Status
Accepted

## Context
The document registry now uses formal profile names such as `Procedimento Operacional` and `Instrucao de Trabalho`.
These names are correct for registry, governance, and authoring flows, but they are too long for compact navigation surfaces such as the sidebar and chips.

Hardcoding short labels in the frontend would violate the registry-as-source-of-truth rule and create drift between UI, API, and persistence.

This change also affects a public API contract and a persisted schema, so it must be frozen explicitly.

## Decision
Add a canonical `alias` field to `document_profiles` and expose it through the document registry API.

Rules:
- `name` remains the formal profile name.
- `alias` is the short display label for compact surfaces.
- `alias` is mandatory.
- `alias` is validated in the backend as trimmed, non-empty, and at most 24 characters.
- `alias` is persisted in PostgreSQL and exposed in the OpenAPI contract.
- Compact UI surfaces prefer `alias`; formal/detail surfaces keep using `name`.
- Future write operations for document profiles must require `alias` and remain admin-governed.

Initial aliases:
- `po -> Procedimentos`
- `it -> Instrucoes`
- `rg -> Registros`

## Consequences
- Positive:
  - Sidebar and compact UI become readable without inventing frontend-only labels.
  - Registry remains the source of truth for naming.
  - Future admin CRUD can govern formal and short names independently.
- Negative:
  - Adds one more required field to the registry contract and database schema.
  - Existing environments need a migration before the API can read profiles from PostgreSQL.

## Alternatives Considered
- Option A: Shorten labels only in the frontend.
  - Rejected because it duplicates naming rules outside the backend and creates drift.
- Option B: Increase sidebar width and keep only formal names.
  - Rejected because it weakens navigation density and does not scale for future profile names.
