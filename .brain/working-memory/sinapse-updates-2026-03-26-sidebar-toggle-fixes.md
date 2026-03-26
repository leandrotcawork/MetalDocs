---
task_id: 2026-03-26-sidebar-toggle-fixes
status: proposed
created_at: "2026-03-26T11:30:00Z"
---

## Proposed Sinapse Updates

### NEW LESSON candidate for cortex/frontend/lessons/

Pattern: "CSS grid 1fr collapse trap"
- When sidebar uses `grid-template-rows: 1fr auto` and nav content is conditionally removed, the header row expands to fill `1fr` and vertically centers its content.
- Fix: override to `grid-template-rows: auto` on the collapsed state.

Pattern: "SVG currentColor on dark surfaces"
- Use `color: #ffffff` (not rgba with low opacity) for icon buttons on dark backgrounds. Thin SVG strokes at 70% opacity become invisible on dark maroon (#4a141f).

**Developer approval required.**
