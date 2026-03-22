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

## Lesson AJ - Equal card height works better at grid level than fixed per card
Date: 2026-03-22 | Trigger: correction
Wrong:   Fixed heights per card class caused visual mismatch and clipping risk
Correct: Equalize heights using grid row sizing (`grid-auto-rows`) and `height: 100%` on cards
Rule:    For sibling card parity, prefer layout-level equalization over per-card hardcoded heights.
Layer:   frontend

## Lesson AK - Repeated form controls must use shared components
Date: 2026-03-22 | Trigger: correction
Wrong:   Same screen mixing native inputs/selects and spotlight dropdown variants
Correct: Use `TextFieldBox` and `DropdownFieldBox` for repeated fields in the same flow
Rule:    Repeated controls should be composed from shared components to prevent style drift.
Layer:   frontend

## Lesson AL - Card layout must guard against overflow at smaller zooms
Date: 2026-03-22 | Trigger: correction
Wrong:   Grid/card content overflowed when zooming out because min-width and width constraints were missing
Correct: Add `min-width: 0` and `width: 100%` guards on cards and spotlight dropdowns
Rule:    Ensure grid children can shrink without overflow by enforcing min-width and width constraints.
Layer:   frontend

## Lesson AM - Set shared card height from a single source
Date: 2026-03-22 | Trigger: correction
Wrong:   Card heights drifted because the tallest column dictated the row
Correct: Use a shared CSS variable for card height so base/edit/create stay aligned
Rule:    For sibling cards that must match height, set a single shared height token.
Layer:   frontend

## Lesson AN - Scroll only where needed, not on the whole card
Date: 2026-03-22 | Trigger: correction
Wrong:   Forcing fixed height on all user cards to make Base de usuarios scroll
Correct: Keep cards with auto height; apply scroll constraint only to the Base list container
Rule:    When one panel needs scrolling, constrain that panel's list area instead of hard-fixing sibling card heights.
Layer:   frontend

## Lesson AO - Equal-height cards should follow grid stretch, not fixed tokens
Date: 2026-03-22 | Trigger: correction
Wrong:   Mixing fixed card heights and per-card exceptions caused inconsistent panel sizing
Correct: Let the grid row stretch items so all cards match the tallest sibling, while only the list area scrolls
Rule:    For same-row admin cards, use grid stretch for height parity and local overflow on scrollable content only.
Layer:   frontend

## Lesson AP - Scroll behavior can be restored without layout refactor
Date: 2026-03-22 | Trigger: correction
Wrong:   Reworking card structure when only scroll feedback was requested
Correct: Keep existing card sizing and change only list overflow mode in Base de usuarios
Rule:    For UI corrections scoped to scroll behavior, apply the smallest CSS-only diff possible.
Layer:   frontend

## Lesson AQ - Visible scroll requires real overflow data
Date: 2026-03-22 | Trigger: correction
Wrong:   Keeping `slice(0, 10)` while expecting a scroll bar with larger datasets
Correct: Remove hard cap when requirement is full-base scrolling
Rule:    Scroll behavior depends on rendered content exceeding the viewport; avoid client-side caps when full list navigation is expected.
Layer:   frontend

## Lesson AR - Constrain list height inside the card to avoid row blowout
Date: 2026-03-22 | Trigger: correction
Wrong:   Letting the Base list grow freely, making the whole row height explode
Correct: Use `grid-template-rows: auto auto minmax(0, 1fr)` on the Base card and scroll the list area
Rule:    When a sibling card should set row height, constrain list overflow inside its own card layout.
Layer:   frontend

## Lesson AS - Dynamic sibling height may require runtime measurement
Date: 2026-03-22 | Trigger: correction
Wrong:   Expecting CSS grid alone to keep Base exactly equal to Editar while Base has much more content
Correct: Observe the Editar card height and apply it directly to the Base card, then scroll only the list region
Rule:    When one panel must exactly mirror another panel's auto height, use runtime measurement instead of CSS guesswork.
Layer:   frontend

## Lesson AT - A shared `height: 100%` can invalidate measured auto-height layouts
Date: 2026-03-22 | Trigger: correction
Wrong:   Measuring one card while a common `.card { height: 100%; }` still forces all cards to fill the grid row
Correct: Keep common cards at `height: auto` when one sibling's intrinsic height is the source of truth
Rule:    Runtime height sync only works if generic stretch rules are removed from the measured elements.
Layer:   frontend

## Lesson AU - Shared measured height should cover all sibling cards
Date: 2026-03-22 | Trigger: correction
Wrong:   Syncing only `Base de usuarios` to `Editar usuario` and leaving `Criar usuario` unsynchronized
Correct: Reuse the measured edit height across every sibling card that must visually match
Rule:    When one card defines the canonical height of a row, every peer card in that row should consume the same measured value.
Layer:   frontend

## Lesson AV - Scroll containers inside rounded cards must be clipped by the parent
Date: 2026-03-22 | Trigger: correction
Wrong:   Letting the list scrollbar render outside the rounded clipping area, creating a visual step in the bottom corner
Correct: Apply `overflow: hidden` on the rounded card and keep the inner list scrollable
Rule:    Rounded cards with internal scroll regions must clip overflow at the card boundary to preserve corner curvature.
Layer:   frontend

## Lesson AW - Footer actions in equal-height cards should consume spare space
Date: 2026-03-22 | Trigger: correction
Wrong:   Leaving the primary action inside the normal field flow, creating a large dead gap under it
Correct: Make the create form body a column layout and push the primary action to the bottom with `margin-top: auto`
Rule:    In fixed-height form cards, the primary CTA should anchor to the footer rather than float above leftover space.
Layer:   frontend

## Lesson AX - Edit actions should align in a single row
Date: 2026-03-22 | Trigger: correction
Wrong:   Stacking Resetar, Desbloquear e Desativar vertically, consuming too much height
Correct: Use a three-column row in the actions area so all three sit on one line
Rule:    Related edit actions should be grouped horizontally when space allows.
Layer:   frontend

## Lesson AY - Keep edit action spacing consistent with form fields
Date: 2026-03-22 | Trigger: correction
Wrong:   Action row sits too close to the password field, breaking spacing rhythm
Correct: Add explicit margin above the action row to match surrounding gaps
Rule:    Action groups should follow the same vertical rhythm as the form fields.
Layer:   frontend

## Lesson AZ - Avoid blocking UI with loading placeholder panels during inline updates
Date: 2026-03-22 | Trigger: correction
Wrong:   Showing the workspace loading placeholder over the management form during save/refresh
Correct: Skip the loading placeholder while keeping empty and error states visible
Rule:    Inline updates should not replace active forms with blocking placeholders.
Layer:   frontend

## Lesson BA - Peer dashboard panels need fixed list regions
Date: 2026-03-22 | Trigger: correction
Wrong:   Letting Online and Auditoria cards grow with item count, causing uneven layout and drifting content
Correct: Give both panels fixed heights and make only their list regions scrollable
Rule:    In side-by-side dashboard cards, growth should happen inside list containers, not at panel level.
Layer:   frontend
