# Review Lenses — MetalDocs

## L1 — Request flow (Critical)
- delivery → application → domain → infrastructure enforced?
- Handler contains business logic? (must not)
- Domain imports infrastructure? (must not)
- Application bypasses domain? (must not)

## L2 — Auth and permissions (Critical)
- `authn.UserIDFromContext(ctx)` used in every handler that needs identity?
- New endpoint registered in `permissions.go`?
- Any endpoint reachable without auth that should require it?
- Permission check at correct layer (backend, not frontend)?

## L3 — Module boundaries (Critical)
- Any module importing internal packages of another module?
- Any direct SQL query on another module's tables?
- Cross-module communication via interface or event only?

## L4 — Events and outbox
- Every relevant mutation publishes a domain event?
- `IdempotencyKey` set: `"event_type:aggregate_id"`?
- Worker consumers are idempotent (ON CONFLICT DO NOTHING)?
- Event payload contains all fields consumers need?

## L5 — Immutability contracts (Critical for document domain)
- Document version content never updated (INSERT only)?
- Audit events append-only (no UPDATE/DELETE)?
- New content = new version row (not overwrite)?

## L6 — OpenAPI
- Every new or changed endpoint in `api/openapi/v1/openapi.yaml`?
- Response shape matches OpenAPI schema?
- Error responses follow standard format (`error.code`, `error.message`, `error.trace_id`)?
- Breaking change in v1? (never — use v2)

## L7 — Data integrity
- Migration additive-first?
- Destructive migration has ADR + rollback plan?
- Migrations numbered sequentially?
- Grants included for API/worker users?

## L8 — Observability
- HTTP observability middleware wraps all routes (already in main.go)?
- Structured logs include `trace_id`, `user_id`, `module`, `action`?
- Errors surfaced with enough context to debug?
- Silent failure paths? (errors caught without log)

## L9 — Test coverage (minimum)
- Unit test for domain invariants?
- Unit test for application service with memory repo?
- Integration test for handler with real DB?
- Contract test: response matches OpenAPI schema?

## L10 — Code quality
- No secrets or credentials in code?
- No broad refactor mixed with feature?
- Smallest safe diff?
- Error codes stable and documented?

## Severity mapping
| Severity | Lenses |
|---|---|
| Critical | L1 flow, L2 auth, L3 boundaries, L5 immutability |
| High | L6 OpenAPI drift, L4 missing idempotency, L8 silent failures |
| Medium | L9 missing tests, L7 migration without ADR, L10 mixed concerns |
| Low | naming, log verbosity, test coverage gaps |
