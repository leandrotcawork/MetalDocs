# AGENTS — MetalDocs

## On every session start
1. Read `tasks/lessons.md` — apply every rule before touching code
2. Read `tasks/todo.md` — know current state
3. After any correction: write lesson to `tasks/lessons.md` immediately

## Engineering bar
Every decision passes this filter:
*"Would a Stripe or Google senior engineer approve this in code review?"*
- Names are self-documenting — no comment needed
- Errors carry structured codes: `MODULE_ENTITY_REASON`
- Every request logs `trace_id`, `user_id`, `module`, `action`, `result`, `duration_ms`
- Every write event uses `idempotency_key` and outbox pattern
- Authorization always validated in backend — never skipped

## Absolute rules — violation = stop and fix immediately

**Request flow (frozen)**
- `delivery → application → domain → infrastructure`
- Handler never contains business logic
- Domain never depends on infrastructure
- Modules never import internals of other modules directly

**Auth and authorization**
- IAM middleware handles auth — handler reads via `authn.UserIDFromContext(ctx)`
- Permission check registered in `permissions.go` for every new endpoint
- Never bypass authorization in handler

**Events and outbox**
- Every relevant mutation publishes a domain event via `messaging.Publisher`
- Outbox `ON CONFLICT (idempotency_key) DO NOTHING` — idempotent by design
- `idempotency_key` format: `event_type:aggregate_id`
- Worker consumers must be idempotent — retry-safe

**Data**
- Document versions are immutable — never update existing version content
- Audit log is append-only — never update or delete audit events
- Migrations additive-first — destructive migration requires ADR

**API**
- `api/openapi/v1/openapi.yaml` is source of truth
- No endpoint without OpenAPI entry
- Breaking change only in `/api/v2`
- OpenAPI updated in same PR as API change

**Code**
- No secrets or credentials in code — env var only
- No business logic in delivery layer
- No direct table access across module boundaries
- Smallest safe diff — never mix feature + broad refactor

## Skill map

| Task | Skill |
|---|---|
| Any implementation | `$md` |
| New Go module | `$metaldocs-module` |
| OpenAPI contract | `$metaldocs-openapi` |
| ADR lifecycle | `$metaldocs-adr` |
| Review | `$metaldocs-review` |

## Commit format
`<type>(<scope>): <what>` — feat | fix | docs | chore | refactor | test
One commit per completed task. No uncommitted work at session end.

## Lesson format (write to tasks/lessons.md after every correction)
```
## Lesson N — <title>
Date: YYYY-MM-DD | Trigger: <correction | review | build failure>
Wrong:   <exact code or decision>
Correct: <exact code or decision>
Rule:    <one sentence>
Layer:   <delivery | application | domain | infrastructure | process>
```
