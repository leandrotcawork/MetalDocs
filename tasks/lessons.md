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

## Lesson P â€” useEffect must not depend on unstable hook objects
Date: 2026-03-21 | Trigger: correction
Wrong:   `useEffect(..., [authSession])` and `useEffect(..., [notificationsApi, documentsWorkspace])` causing re-bootstrap/re-subscribe on every render
Correct: Depend on stable callbacks only (e.g. `bootstrap`, `subscribeOperations`, `refreshOperationalSignals`)
Rule:    Effects that trigger I/O must use stable function dependencies, never whole hook return objects.
Layer:   frontend

## Lesson O — EPERM indicates sandbox permission issue
Date: 2026-03-21 | Trigger: correction
Wrong:   Retrying Playwright after `spawn EPERM` inside sandbox
Correct: Request escalated run when `EPERM` occurs during process spawn (outside sandbox permission required)
Rule:    `EPERM` from child process spawn means permission denied; run with escalation.
Layer:   process

## Lesson Q — Content builder docbar layout must match spec
Date: 2026-03-21 | Trigger: correction
Wrong:   Docbar left-aligned title and missing right-side back button
Correct: Tags on the left, title centered and larger, back button on the right
Rule:    Docbar layout follows UI spec before marking T6 complete.
Layer:   frontend

## Lesson R — Preview toggle uses arrows, compact and vinho
Date: 2026-03-21 | Trigger: correction
Wrong:   Preview toggle labeled "Recolher" on the right and oversized
Correct: Compact wine toggle on the left with arrow icons (right = recolher, left = expandir)
Rule:    Preview controls follow the interaction spec and sizing.
Layer:   frontend

## Lesson S — Preview expand button aligns to top when collapsed
Date: 2026-03-21 | Trigger: correction
Wrong:   Collapsed preview shows expand button centered
Correct: Collapsed preview aligns expand button to the top
Rule:    Collapsed controls should follow top alignment for visibility.
Layer:   frontend

## Lesson T — Collapsed preview aligns horizontally centered
Date: 2026-03-21 | Trigger: correction
Wrong:   Collapsed preview button is horizontally offset
Correct: Collapsed preview button is horizontally centered
Rule:    Collapsed controls must be centered horizontally.
Layer:   frontend

## Lesson U — Content builder back route returns to create
Date: 2026-03-21 | Trigger: correction
Wrong:   Back from content builder navigates to library
Correct: Back from content builder navigates to create view
Rule:    Back navigation must return to the originating create flow.
Layer:   frontend

## Lesson V — Document creation deferred until editor PDF
Date: 2026-03-21 | Trigger: correction
Wrong:   "Ir para editor" creates the document immediately
Correct: Create only when the user generates PDF in the editor
Rule:    Draft editor must not create documents prematurely.
Layer:   frontend

## Lesson W — Draft editor must still load schema
Date: 2026-03-21 | Trigger: correction
Wrong:   Editor without documentId skips schema and renders no sections
Correct: Editor loads profile schema for draft documents
Rule:    Draft mode must show sections and schema.
Layer:   frontend

## Lesson X — Admin form sync lives with admin view
Date: 2026-03-21 | Trigger: correction
Wrong:   App-level effect referenced removed managed user state, breaking build
Correct: AdminCenterView syncs managed user form when the list refreshes
Rule:    View-specific state sync stays inside the view.
Layer:   frontend

## Lesson Y â€” Workspace main must not reserve removed toolbar row
Date: 2026-03-22 | Trigger: correction
Wrong:   Removed toolbar markup but kept `grid-template-rows: 44px minmax(0, 1fr)` causing content to render in a 44px row
Correct: `grid-template-rows: minmax(0, 1fr)` when toolbar is removed
Rule:    When removing structural rows, update the grid template to avoid zero-visibility content.
Layer:   frontend

## Lesson Z - WorkspaceViewFrame must stay the single frame
Date: 2026-03-22 | Trigger: correction
Wrong:   `AdminCenterView` added extra `container/card` wrappers inside `WorkspaceViewFrame`, duplicating padding and card boundaries
Correct: `WorkspaceViewFrame` provides the outer frame; the feature view adds only one inner shell for max-width and content gap
Rule:    Workspace views should not nest a second page frame inside the shared frame component.
Layer:   frontend

## Lesson AA - Inner shells must not recentralize workspace content
Date: 2026-03-22 | Trigger: correction
Wrong:   `AdminCenterView.module.css` applied `width` and `margin: 0 auto` on the inner `.shell`, shifting only the content block
Correct: Inner shells handle layout gaps only; outer workspace frame owns page width and horizontal rhythm
Rule:    Do not center an inner content shell when the shared workspace frame already defines the page container.
Layer:   frontend

## Lesson AB - CSS edits must preserve valid block structure
Date: 2026-03-22 | Trigger: correction
Wrong:   Extra closing brace in `AdminCenterView.module.css` causing Vite CSS parse error
Correct: Keep selector blocks balanced; remove stray brace before next selector
Rule:    Every CSS block must have balanced braces to avoid build-time parsing errors.
Layer:   frontend

## Lesson AC - Empty states must respect list alignment
Date: 2026-03-22 | Trigger: correction
Wrong:   Activity empty state rendered as a plain paragraph, breaking header/row alignment
Correct: Render empty state within the list layout to keep row spacing and separators aligned
Rule:    Empty states inside list panels must reuse list layout structure for visual consistency.
Layer:   frontend

## Lesson AD - Shared dividers require equal header height
Date: 2026-03-22 | Trigger: correction
Wrong:   Similar cards used different header content structures, so the divider line sat at different vertical positions
Correct: Cards that share a visual pattern use the same action wrapper and a fixed minimum header height
Rule:    When the divider belongs to the header container, equalize header structure and height across sibling cards.
Layer:   frontend

## Lesson AE - Divider baseline needs fixed header height
Date: 2026-03-22 | Trigger: correction
Wrong:   Header used only `min-height`, allowing content-driven height drift and misaligned horizontal dividers
Correct: Shared cards use explicit header `height` + identical title wrapper structure
Rule:    For pixel-aligned dividers across sibling cards, use fixed header height, not only minimum height.
Layer:   frontend

## Lesson AF - Shared cards need shared row height
Date: 2026-03-22 | Trigger: correction
Wrong:   Online and activity rows used different effective heights, making cards look vertically unbalanced
Correct: Use one shared `min-height` for list rows and center empty-state row content
Rule:    Sibling cards in the same grid row should share row metrics to keep visual balance.
Layer:   frontend

## Lesson AG - Empty row can be vertically centered and left aligned
Date: 2026-03-22 | Trigger: correction
Wrong:   Empty activity row centered both axes, drifting away from left margin rhythm
Correct: Keep `align-items: center` and use `justify-content: flex-start` for left start alignment
Rule:    In list panels, empty-state content should follow the same left edge rhythm as filled rows.
Layer:   frontend

## Lesson AH - UI-only fields should not break API contracts
Date: 2026-03-22 | Trigger: correction
Wrong:   Expanding create form by changing API payload shape without backend support
Correct: Keep new UX fields local to UI when contract is unchanged, preserve existing create payload
Rule:    Frontend form redesign must preserve stable API contracts unless backend changes are planned.
Layer:   frontend

## Lesson AI - List limits must apply to both filtered and unfiltered states
Date: 2026-03-22 | Trigger: correction
Wrong:   User list cap applied only when search had query, showing full list otherwise
Correct: Apply `slice(0, limit)` after filtering logic in every state
Rule:    UI limits should be enforced consistently regardless of filter input.
Layer:   frontend
