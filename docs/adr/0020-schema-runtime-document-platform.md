# ADR-0020 Schema Runtime Document Platform

## Decision

Adopt a schema-driven document runtime inside the existing `documents` module.

## Consequences

- `document_profiles` stop being the active editor contract
- schema + values become the canonical editor and export contract
- admin schema authoring is deferred to a later phase
