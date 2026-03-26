---
task_id: 2026-03-26-sidebar-toggle-fixes
model: haiku
domain: frontend
complexity: 15
created_at: "2026-03-26T11:30:00Z"
---

## Bugs
1. SVG arrow invisible — stroke color blending with dark sidebar background → force white stroke
2. Collapsed sidebar: button vertically centered, should be pinned to top

## Target File
`DocumentWorkspaceShell.module.css` (CSS Modules)

## Fix 1
`.workspace-sidebar-toggle` — add `color: rgba(255,255,255,0.85)` explicitly, or ensure stroke is white

## Fix 2
Sidebar uses `grid-template-rows: 1fr auto`. When collapsed, scroll area is unmounted (hidden).
The header becomes the only child, but `1fr auto` grid still reserves the 1fr row.
Fix: when collapsed, sidebar grid should be `auto` only, or use flex with align-items: flex-start.
Simplest: `.workspace-sidebar.is-collapsed` → `grid-template-rows: auto` so header sits at top.
