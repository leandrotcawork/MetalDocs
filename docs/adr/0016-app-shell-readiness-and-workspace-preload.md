# ADR-0016: App Shell Readiness and Workspace Preload

## Status
Accepted

## Context
When the user returns to the app, "session validation" feels slow. The current bootstrap path waits for `me()` and then blocks UI readiness on workspace preload (profiles, taxonomy, lists). This creates a perceived authentication delay and a poor UX, even though auth is already valid.

## Decision
Render the application shell as soon as `me()` succeeds and handle workspace preload asynchronously.
We will:
- Set `authState=ready` immediately after `me()` resolves.
- Start `loadWorkspace()` in the background and expose a `loadState` to show skeletons/inline loading per section.
- Keep authorization and permission checks in the backend as the source of truth.

## Consequences
- Positive:
  - Faster perceived login/return flow and improved UX.
  - Clear separation between auth readiness and data readiness.
  - Enables progressive loading of heavy resources.
- Negative:
  - UI must handle transient empty/loading states more carefully.
  - Slightly more complexity in client state management.

## Alternatives Considered
- Keep blocking bootstrap until workspace preload completes.
- Add a single global loading screen (still blocks the shell).
