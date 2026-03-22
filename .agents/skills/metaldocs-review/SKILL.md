---
name: metaldocs-review
description: "Architecture and implementation review for MetalDocs. Two levels: (1) compliance — does code follow current patterns, (2) quality — are the patterns themselves professional and scalable at big-tech level. Always runs both. Produces severity-ordered findings and verdict."
---

# MetalDocs Review

Use this skill as the quality gate. `$md` remains the workflow owner for planning, approval, and task sequencing.

## Two levels — always run both

**Level 1 — Compliance:** does the code follow current repo patterns?
**Level 2 — Quality:** are the patterns themselves professional and scalable?

MetalDocs has a strong foundation (clean domain, memory doubles, typed errors).
Level 2 catches the places where it still falls short of big-tech standard.

---

## Level 1 — Compliance (quick check first)

If any fail → CRITICAL → stop:
- [ ] Request flow: delivery → application → domain → infrastructure?
- [ ] Handler contains no business logic?
- [ ] `authn.UserIDFromContext(ctx)` used — not hardcoded user?
- [ ] New endpoint registered in `permissions.go`?
- [ ] New handler registered in `main.go`?
- [ ] Events use `idempotency_key: "event_type:aggregate_id"`?
- [ ] No direct cross-module internal imports?
- [ ] OpenAPI updated for any API change?
- [ ] Document versions immutable (no UPDATE on content)?
- [ ] Audit events append-only?

Full compliance lenses: `references/compliance-lenses.md`

---

## Level 2 — Quality (are the patterns themselves good?)

### Q1 — Domain is pure Go?
Does `domain/` import anything from `infrastructure/`, `delivery/`, or any IO package?
- **Target:** `domain/` imports only stdlib. No `database/sql`, no `net/http`, no platform packages.
- **Why it matters:** domain logic must be testable with zero infrastructure. If it imports sql, you can't unit test it.
- **Flag:** any non-stdlib import in `domain/model.go`, `domain/port.go`, or `domain/errors.go`.

### Q2 — Invariants live in domain, not in service or handler
Does the service or handler contain validation that belongs in the domain model?
- **Problem:** `if strings.TrimSpace(doc.Title) == ""` in service = domain rule leaking up.
- **Target:** `doc.Validate()` in domain, called by service. Service orchestrates, domain guards.
- **Flag:** field-level validation in `application/service.go` or `delivery/http/handler.go`.

### Q3 — Service is actually thin
Does the service contain SQL fragments, HTTP concerns, or framework imports?
- **Target:** service calls domain + repo interface + publisher. That's it.
- **Flag:** any `sql.`, `http.`, or direct repo construction inside `application/service.go`.

### Q4 — Memory double matches postgres interface
Is the memory repo implementing the exact same interface as postgres repo?
- **Problem:** if memory drifts from postgres, unit tests pass but integration fails.
- **Target:** both implement the same `domain.Repository` interface. Verified by Go compiler.
- **Flag:** any method in postgres repo not present in memory repo.

### Q5 — Error mapping is explicit, not generic
Does the handler catch specific domain errors and map them to HTTP status?
- **Problem:** `if err != nil { writeError(w, 500, ...) }` swallows domain errors as 500.
- **Target:** `errors.Is(err, domain.ErrDocumentNotFound)` → 404. Known errors = explicit codes.
- **Flag:** any handler that returns 500 for an error that could be a 404, 409, or 422.

### Q6 — Events carry enough payload for consumers
Does the event payload contain all fields a worker or consumer will realistically need?
- **Problem:** sparse payload forces consumer to make a second query to get context.
- **Target:** payload is self-contained for the primary consumer use case.
- **Flag:** event published with only `{"id": "..."}` when downstream needs title, status, actor, etc.

### Q7 — Tests cover the business rules, not just happy path
Does the test only cover the success case?
- **Target:** unit tests cover: invalid input, not found, permission denied, invariant violation.
- **Flag:** any `TestCreate` that only tests successful creation with no error cases.

### Q8 — No N+1 in list endpoints
Does any list endpoint issue one query per item in a loop?
- **Problem:** `for _, id := range ids { repo.GetByID(id) }` = N+1.
- **Target:** single `SELECT ... WHERE id = ANY($1)` or JOIN.
- **Flag:** any loop containing a repo or DB call in application or infrastructure layer.

### Q9 — Migrations are safe to run twice
Can the migration be re-run without error?
- **Target:** `CREATE TABLE IF NOT EXISTS`, `CREATE INDEX IF NOT EXISTS`, idempotent operations.
- **Flag:** any migration without `IF NOT EXISTS` that would fail on second run.

### Q10 — OpenAPI response schema matches actual handler response
Does the handler return fields not in OpenAPI, or omit fields that are in OpenAPI?
- **Problem:** OpenAPI says `effectiveAt` is optional string, handler returns `null` vs `""` inconsistently.
- **Target:** handler response is mechanically derived from a typed struct that matches OpenAPI.
- **Flag:** any response field not present in the OpenAPI schema, or vice versa.

---

## Finding format

```
CRITICAL — <title>
  File: <path>:<function>
  Rule: <ADR or architectural principle>
  Impact: <what breaks>
  Fix: <specific action>

QUALITY — <title>
  File: <path>:<function>
  Pattern: <what weak pattern this perpetuates>
  Target: <what professional code looks like>
  Priority: fix now | next cycle | track as debt
```

## Verdict

- **ALIGNED** — compliant + no new quality debt → commit
- **COMPLIANT BUT WEAK** — passes compliance, adds quality issues → commit + log to `tasks/quality-debt.md`
- **MISALIGNED** — compliance failures → stop, fix, do not commit

## References
- `references/compliance-lenses.md` — full compliance checklist
- `tasks/quality-debt.md` — running list of known quality issues
