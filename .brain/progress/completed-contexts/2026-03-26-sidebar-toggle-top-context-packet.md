---
task_id: 2026-03-26-sidebar-toggle-top
model: haiku
domain: frontend
complexity: 15
created_at: "2026-03-26T11:00:00Z"
---

## Task
Move sidebar collapse toggle button from bottom footer to top-right corner.
Spotify "Sua Biblioteca" style: header row with title on left + arrow button on right.
When collapsed: just the arrow button at top, centered.

## Reference
Spotify pattern: sidebar header = "Sua Biblioteca" label (left) + ↗ arrow button (right corner).
Collapsed: narrow strip with single arrow button at top.

## Target Files
- `DocumentWorkspaceShell.tsx` (CSS Modules)
- `DocumentWorkspaceShell.module.css`

## Current Structure
- Toggle button is in `workspace-sidebar-footer` (bottom)
- Footer is the auto-row in the sidebar grid (grid-template-rows: 1fr auto)

## Acceptance Criteria
- [ ] New `workspace-sidebar-header` div at TOP of aside
- [ ] Header: label "Navegacao" (left) + toggle button (right) when expanded
- [ ] Header: just toggle button (centered) when collapsed
- [ ] Toggle button uses diagonal expand/collapse arrow (Spotify style)
- [ ] Footer div removed or emptied
- [ ] CSS Modules pattern (styles["class-name"])
- [ ] TypeScript clean
