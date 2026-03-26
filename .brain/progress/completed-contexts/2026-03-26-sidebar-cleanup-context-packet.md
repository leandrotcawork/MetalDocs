---
task_id: 2026-03-26-sidebar-cleanup
model: haiku
domain: frontend
complexity: 15
created_at: "2026-03-26T10:00:00Z"
---

## Task
1. Remove `.workspace-context-pill` div (and divider after it) from `DocumentWorkspaceShell.tsx`
2. Remove `.workspace-runtime-card` div (child of `.workspace-sidebar-footer`) from the same file
3. Make the sidebar collapsible via a toggle button, repurposing the footer area

## Target File
`frontend/apps/web/src/components/DocumentWorkspaceShell.tsx` (CSS modules)
`frontend/apps/web/src/components/DocumentWorkspaceShell.module.css`

## Key Facts
- Component uses CSS Modules (`styles["class-name"]` pattern)
- Sidebar is `<aside className={styles["workspace-sidebar"]}>` with `grid-template-rows: 1fr auto`
- `workspace-sidebar-scroll` = scrollable nav content (1fr row)
- `workspace-sidebar-footer` = fixed bottom area (auto row)
- Context pill: lines 363-379, divider at line 381 — REMOVE BOTH
- Runtime card: lines 478-483 inside footer — REMOVE (replace footer content with toggle button)
- Add `sidebarCollapsed` state (useState, default false)
- Toggle button goes in the footer — always visible
- When collapsed: sidebar width = 44px, scroll area hidden
- When expanded: full 240px width, all content visible

## Conventions
- Named exports only
- No default exports
- CSS module classes via `styles["class-name"]`
- State via useState for simple UI state
- SVG icons inline

## Acceptance Criteria
- [ ] Context pill removed
- [ ] Runtime card removed
- [ ] Toggle button in footer
- [ ] Collapsed state: 44px, nav hidden
- [ ] Expanded state: 240px, full nav visible
- [ ] No TypeScript errors
