---
name: md
description: MetalDocs master orchestrator. Use for any implementation task. Enters plan mode, validates module structure and architecture before any code, orchestrates specialist skills. Usage: "use $md to implement X".
---

# MetalDocs — Master Orchestrator

**Never write code before Phase 1 and Phase 2 are complete and approved.**

---

## Phase 1 — Architectural thinking (always before code)

Read first: `tasks/lessons.md` → `AGENTS.md` → `docs/architecture/ARCHITECTURE_GUARDRAILS.md`

Answer these questions explicitly before proceeding:

**1. Which module(s) does this touch?**
`documents` | `versions` | `workflow` | `iam` | `audit` | `search` | `notifications` | new module

**2. Folder structure** — state before creating any file:
```
internal/modules/<n>/
  domain/
    model.go       ← entities, value objects, invariants
    port.go        ← interfaces (Repository, Writer, Reader)
    errors.go      ← sentinel errors
  application/
    service.go     ← use case orchestration, no HTTP, no direct DB
  infrastructure/
    postgres/
      repository.go  ← implements domain port
    memory/
      repository.go  ← test double (in-memory impl)
  delivery/
    http/
      handler.go   ← parse request → auth check → service → write response
```
See `references/folder-patterns.md` for all variants.

**3. Auth pattern** — state explicitly:
- IAM middleware already validates auth on every request
- Handler reads user: `authn.UserIDFromContext(ctx)` and `authn.RolesFromContext(ctx)`
- New endpoint permission registered in `apps/api/cmd/metaldocs-api/permissions.go`

**4. Event pattern** — if mutation fires events:
- Use `messaging.Publisher.Publish(ctx, messaging.Event{...})`
- `IdempotencyKey`: `"event_type:aggregate_id"`
- Outbox handles delivery — `ON CONFLICT (idempotency_key) DO NOTHING`

**5. Risks** — identify before coding:
- Cross-module dependency? Use interface or event, never direct import
- Immutability constraint? (document versions = immutable, audit = append-only)
- New permission needed? Register in `permissions.go`
- Migration needed? Additive or destructive? Destructive requires ADR

**6. Level scope:**
Level 1 = feature works, tests pass, OpenAPI updated.
State what is deferred.

---

## Phase 2 — Plan (write tasks/todo.md, wait for approval)

```markdown
## Feature: <n>
Module(s): <list>  |  Events: <yes: types | no>  |  ADR: <yes: number | no>
Migration: <yes: additive/destructive | no>

## Tasks
- [ ] T1: OpenAPI contract — $metaldocs-openapi
      commit: "docs(api): add <endpoint> to OpenAPI"
- [ ] T2: domain + port — $metaldocs-module
      commit: "feat(<m>): add domain model and port"
- [ ] T3: infrastructure (postgres + memory) — $metaldocs-module
      commit: "feat(<m>): implement postgres repository"
- [ ] T4: application service — $metaldocs-module
      commit: "feat(<m>): implement application service"
- [ ] T5: delivery handler + permissions — $metaldocs-module
      commit: "feat(<m>): implement HTTP handler and register permission"
- [ ] T6: tests — $metaldocs-module
      commit: "test(<m>): add unit and integration tests"
- [ ] T7: migration (if needed)
      commit: "chore(db): add migration <n>_<description>"
- [ ] T8: ADR — $metaldocs-adr (if needed)
      commit: "docs(adr): ADR-XXXX <title> — verified and closed"

## Acceptance tests
- [ ] go build ./... passes
- [ ] go test ./... passes
- [ ] OpenAPI updated and contract test passes
- [ ] e2e-smoke.ps1 passes
- [ ] no cross-module boundary violation
```

**Present plan. Wait for explicit approval. Then begin T1.**

---

## Phase 3 — Execute (one task at a time, commit after each)

### T1 — OpenAPI → use `$metaldocs-openapi`
File: `api/openapi/v1/openapi.yaml`
Add path, method, request/response schema. No endpoint without spec entry.

### T2–T5 — Go module → use `$metaldocs-module`

**Domain port (interfaces)**
```go
// domain/port.go
type Repository interface {
    Create(ctx context.Context, entity Entity) error
    GetByID(ctx context.Context, id string) (Entity, error)
}
```

**Infrastructure — postgres**
```go
// infrastructure/postgres/repository.go
func (r *Repository) Create(ctx context.Context, entity domain.Entity) error {
    const q = `INSERT INTO metaldocs.<table> (...) VALUES (...)`
    if _, err := r.db.ExecContext(ctx, q, ...); err != nil {
        return fmt.Errorf("create entity: %w", err)
    }
    return nil
}
```

**Infrastructure — memory (test double)**
```go
// infrastructure/memory/repository.go — simple map, used in unit tests
```

**Application service**
```go
// application/service.go
func (s *Service) CreateX(ctx context.Context, cmd CreateXCommand) (domain.X, error) {
    // 1. validate invariants
    // 2. call repo
    // 3. publish event if needed
    event := messaging.Event{
        EventType:      "x.created",
        AggregateType:  "x",
        AggregateID:    x.ID,
        IdempotencyKey: "x.created:" + x.ID,
        TraceID:        cmd.TraceID,
        Payload:        map[string]any{"id": x.ID},
    }
    s.publisher.Publish(ctx, event)
    return x, nil
}
```

**Delivery handler**
```go
// delivery/http/handler.go
func (h *Handler) handleCreate(w http.ResponseWriter, r *http.Request) {
    traceID := requestTraceID(r)

    // Auth resolved by IAM middleware — just read context
    userID := authn.UserIDFromContext(r.Context())
    if userID == "" {
        writeError(w, 401, "AUTH_REQUIRED", "Authentication required", traceID)
        return
    }

    // Parse request
    var req CreateRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        writeError(w, 400, "INVALID_REQUEST", "Invalid request body", traceID)
        return
    }

    // Call service
    result, err := h.service.CreateX(r.Context(), application.CreateXCommand{
        ActorID: userID, TraceID: traceID, ...
    })
    if err != nil {
        writeError(w, 500, "INTERNAL_ERROR", "Failed to create X", traceID)
        return
    }
    writeJSON(w, 201, toResponse(result))
}
```

**Register permission** (for every new endpoint):
```go
// apps/api/cmd/metaldocs-api/permissions.go
if method == http.MethodPost && path == "/api/v1/<resource>" {
    return iamdomain.PermXCreate, true
}
```

**Register handler in main.go:**
```go
xHandler := xdelivery.NewHandler(xService)
xHandler.RegisterRoutes(mux)
```

### T6 — Tests
- Unit: domain invariants + application service with memory repo
- Integration: handler + real DB (use `*_test.go` alongside the file)
- Contract: validate response matches OpenAPI schema

### T7 — Migration (if needed)
```sql
-- migrations/XXXX_<description>.sql
CREATE TABLE metaldocs.<table> (...);
GRANT SELECT, INSERT, UPDATE ON metaldocs.<table> TO metaldocs_api;
```

### T8 — ADR → use `$metaldocs-adr`

### After every task
1. `go build ./...` passes
2. `go test ./...` passes (or targeted: `go test ./internal/modules/<n>/...`)
3. Mark `[x]` in `tasks/todo.md`
4. `git commit -m "<type>(<scope>): <what>"`

---

## Phase 4 — Review before declaring done

Quick self-check:
- [ ] Request flow: delivery → application → domain → infrastructure?
- [ ] No business logic in handler?
- [ ] `authn.UserIDFromContext` used (not hardcoded user)?
- [ ] New endpoint registered in `permissions.go`?
- [ ] Events use `idempotency_key: "event_type:aggregate_id"`?
- [ ] No direct cross-module internal imports?
- [ ] OpenAPI updated?
- [ ] Tests added?
- [ ] Document versions treated as immutable?

Run `$metaldocs-review` for full review.

---

## After any correction
Write lesson to `tasks/lessons.md` immediately.

## References
- `references/folder-patterns.md` — module structure variants
- `skills/metaldocs-implement/references/go-patterns.md` — Go code patterns
