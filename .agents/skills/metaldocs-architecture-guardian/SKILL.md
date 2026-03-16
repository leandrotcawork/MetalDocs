# metaldocs-architecture-guardian

## Purpose
Keep every MetalDocs change aligned with long-term architecture, contract safety, and operability.

## When to use
- Any feature touching `internal/modules`, `api/openapi`, `migrations`, or auth/RBAC.
- Any change that may affect contracts, boundaries, data integrity, or rollout risk.

## Non-negotiable checks
1. Boundary integrity:
- `delivery -> application -> domain -> infrastructure`.
- No cross-module internals import.

2. Contract safety:
- API change requires OpenAPI update in same PR.
- Error payload must keep `error.code/message/details/trace_id`.

3. Data safety:
- Additive-first migrations.
- Destructive change requires ADR + rollback notes.

4. Auth/RBAC safety:
- Backend authorization required for protected endpoints.
- Never trust client role if DB-backed role resolution exists.

5. Observability and ops:
- Update runbook for infra/ops impacts.
- Keep smoke test path documented.

## Required output format for every task
- Summary of what changed.
- Files touched.
- Validation commands.
- Risks and follow-up.

## PR checklist shortcut
- [ ] OpenAPI updated if API changed
- [ ] Tests updated for behavior change
- [ ] Migration policy respected
- [ ] Runbook updated if needed
- [ ] No boundary violations
