# MetalDocs AGENTS.md

## How To Work
- Use targeted search first (`rg -n "symbol|text"`).
- Keep diffs minimal and scoped to the request.
- Do not introduce unrelated refactors.

## Architecture Rules
- Frontend never executes business rules directly.
- Business rules live in `internal/modules/<module>/application`.
- Domain invariants live in `internal/modules/<module>/domain`.
- Infra adapters live in `internal/modules/<module>/infrastructure`.
- Cross-cutting concerns live in `internal/platform`.

## Non-Negotiable Contracts
- API contract is OpenAPI source of truth in `api/openapi/v1/openapi.yaml`.
- Document versions are immutable.
- Audit is append-only.
- Permission checks are always server-side.
- Do not overwrite populated values with null/empty values.

## Scope Guardrails
- No AI/LLM features in v1.
- Keep placeholders documented only; do not implement AI pipelines.

## Delivery Format
Always report:
- Summary of what changed.
- File list touched.
- Validation commands executed.
- Risks and follow-up notes.
