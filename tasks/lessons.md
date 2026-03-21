# tasks/lessons.md
# Read at the start of EVERY session before touching any code.
# Add new lessons after every correction. Never delete existing lessons.

---

## Lesson A — Request flow is delivery → application → domain → infrastructure
Wrong:   Handler calls repository directly or contains business logic
Correct: Handler calls service; service calls domain + repo; domain has no IO
Rule:    delivery never bypasses application. application never bypasses domain.
Layer:   All layers

## Lesson B — Auth is read from context, not hardcoded
Wrong:   userID := "admin" or hardcoded actor
Correct: userID := authn.UserIDFromContext(r.Context()) — empty = 401
Rule:    IAM middleware resolves auth. Handler reads result via authn package.
Layer:   delivery/http

## Lesson C — Every new endpoint needs a permissions.go entry
Wrong:   New endpoint deployed without permission registration
Correct: Add to apps/api/cmd/metaldocs-api/permissions.go before shipping
Rule:    No endpoint is active without a permission mapping. Missing = unauthorized access.
Layer:   delivery + bootstrap

## Lesson D — Event idempotency_key is "event_type:aggregate_id"
Wrong:   IdempotencyKey: generateID() or empty
Correct: IdempotencyKey: "document.created:" + docID
Rule:    Idempotency key must be deterministic so retries do not duplicate events.
Layer:   application (service)

## Lesson E — Document versions are immutable — never UPDATE content
Wrong:   UPDATE document_versions SET content = $1 WHERE id = $2
Correct: INSERT new version row with incremented version number
Rule:    Version content is immutable by ADR-0013. New content = new row.
Layer:   infrastructure/postgres

## Lesson F — Audit events are append-only — never UPDATE or DELETE
Wrong:   DELETE FROM metaldocs.audit_events WHERE ...
Correct: Only INSERT to audit table. Audit is forever.
Rule:    Audit log is append-only by architectural contract.
Layer:   infrastructure/postgres

## Lesson G — No direct cross-module internal imports
Wrong:   import "metaldocs/internal/modules/documents/infrastructure/postgres"
Correct: Receive domain.Repository interface as parameter, or communicate via event
Rule:    Modules communicate via public interfaces and events only.
Layer:   All modules

## Lesson H — Every completed task needs a commit
Wrong:   Finishing tasks without committing, session ends with uncommitted work
Correct: git commit -m "feat(<scope>): <what>" after every task
Rule:    One commit per task. No uncommitted work at session end.
Layer:   Process

## Lesson I — ADR done only when acceptance test passes and committed
Wrong:   Mark ADR Accepted after writing the document
Correct: Write → implement → run acceptance test → commit "docs(adr): <n>-<title> — verified and closed"
Rule:    ADR is DONE only when acceptance test passes and git commit is made.
Layer:   Process

## Lesson J — Memory repo required alongside every postgres repo
Wrong:   Only postgres implementation, no memory double
Correct: infrastructure/postgres/repository.go + infrastructure/memory/repository.go
Rule:    Every postgres repo has a memory test double. Unit tests use memory, not DB.
Layer:   infrastructure

---
<!-- Project-specific lessons below this line -->

## Lesson K — Module boundaries enforced by imports
Date: 2026-03-21 | Trigger: correction
Wrong:   `internal/modules/iam/delivery/http/admin_handler.go` importing `metaldocs/internal/modules/auth/application`
Correct: `internal/modules/iam/delivery/http/admin_handler.go` depends on `UserAdminService` interface; bootstrap passes `*auth/application.Service`
Rule:    Cross-module dependencies must go through domain types or local interfaces, never another module's application/infrastructure internals.
Layer:   delivery

## Lesson L — Actor identity must come from authn context
Date: 2026-03-21 | Trigger: correction
Wrong:   `authenticatedActor()` reading `auth/domain.CurrentUserFromContext` (breaks legacy header flow; actor becomes `system`)
Correct: `authenticatedActor()` reads `authn.UserIDFromContext(r.Context())` (works with both cookie and legacy header mode)
Rule:    Handlers must read identity via `authn.UserIDFromContext(ctx)` so audit/log attribution stays correct.
Layer:   delivery

## Lesson M — Outbox idempotency_key must follow contract
Date: 2026-03-21 | Trigger: correction
Wrong:   idempotency keys like `doc-create-<id>` / `workflow-transition-<doc>-<to>` (harder to reason about and drifts from contract)
Correct: `"<event_type>:<aggregate_id>"` (e.g. `document.created:<docId>`, `document.version.created:<docId>:<version>`)
Rule:    Outbox idempotency keys are deterministic and contract-shaped so retries never duplicate events.
Layer:   application

## Lesson N — PowerShell scripts must declare param first
Date: 2026-03-21 | Trigger: correction
Wrong:   `scripts/dev-api-web.ps1` had `$ErrorActionPreference` before `param(...)` (PowerShell treats `param` as a command and fails)
Correct: `param(...)` is the first non-comment statement in the script, before any executable lines
Rule:    In PowerShell, `param` must come first or parameter binding breaks at runtime.
Layer:   process

## Lesson O — EPERM indicates sandbox permission issue
Date: 2026-03-21 | Trigger: correction
Wrong:   Retrying Playwright after `spawn EPERM` inside sandbox
Correct: Request escalated run when `EPERM` occurs during process spawn (outside sandbox permission required)
Rule:    `EPERM` from child process spawn means permission denied; run with escalation.
Layer:   process
