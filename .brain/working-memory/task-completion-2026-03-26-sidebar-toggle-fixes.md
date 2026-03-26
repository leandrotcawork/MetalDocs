---
task_id: 2026-03-26-sidebar-toggle-fixes
status: success
model: haiku
created_at: "2026-03-26T11:30:00Z"
---

## Task
Fix two bugs in sidebar toggle: SVG invisible + button vertically centered when collapsed.

## Files Changed
- `DocumentWorkspaceShell.module.css` (2 lines changed)
  - `.workspace-sidebar-toggle`: color rgba(255,255,255,0.7) → #ffffff (SVG now visible)
  - `.workspace-sidebar.is-collapsed`: added grid-template-rows: auto (button pins to top)

## Tests
TypeScript: PASS. No automated tests.

## Sinapses Referenced
- cortex-frontend-index

## Lessons
- CSS grid `1fr auto` stretches the first child to fill remaining height when only one child present. When conditionally rendering nav content, the header takes the 1fr row and its content vertically centers. Override with `grid-template-rows: auto` on the collapsed state to pin elements to top.
- SVG `currentColor` inherits from `color` on the button. Low opacity (0.7) on dark backgrounds can make thin strokes invisible — use full white (#ffffff) for icon buttons on dark surfaces.
