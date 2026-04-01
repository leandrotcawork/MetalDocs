# Preview Panel Zoom Design

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make the Ao Vivo live preview render the document at a proportional, readable size by applying a two-level zoom that adapts to the panel width.

**Architecture:** The document page is given a fixed design width of 700px and scaled via CSS `zoom`. A CSS container query on `.preview-panel-body` switches between medium (0.6×) and full (0.88×) zoom at a 520px breakpoint. The preview aside's hardcoded `width: 380px` is removed so the split-pane drag actually changes the preview width.

**Tech Stack:** CSS container queries (`container-type: inline-size`), CSS `zoom` property, React (no JS changes needed beyond one prop default).

---

## Problem

- `.content-builder-preview` has a hardcoded `width: 380px` that ignores the split-pane drag handle — making the drag useless for the preview.
- `.preview-document-page` uses `width: 100%`, so the document content is squeezed to ~347px — too narrow for tables and multi-column content to render correctly.
- Text wraps far more aggressively than it would in the generated document, misleading the user about real line breaks and table widths.

## Solution

### Two zoom levels

| Level  | Zoom  | Doc renders at | Applies when panel is |
|--------|-------|----------------|-----------------------|
| Medium | 0.6×  | 420px          | < 520px               |
| Full   | 0.88× | 616px          | ≥ 520px               |

### Files to change

**`frontend/apps/web/src/styles.css`**

1. `.content-builder-preview`: remove `width: 380px`, replace with `width: 100%`.
2. Remove the `@media (max-width: 1200px)` override that set `.content-builder-preview { width: 320px }`.
3. `.preview-panel-body`: add `container-type: inline-size`.
4. `.preview-document-page`: change `width: 100%; max-width: 100%` → `width: 700px; max-width: none`.
5. `.preview-document`: add default `zoom: 0.6`.
6. Add container query: `@container (min-width: 520px) { .preview-document { zoom: 0.88; } }`.

**`frontend/apps/web/src/components/content-builder/ResizableSplitPane.tsx`**

1. Change `minRightWidth` default prop from `280` to `340` so the panel never collapses so narrow the document becomes unreadable.

### What does NOT change

- No React component logic changes.
- No prop threading.
- No ResizeObserver JS.
- The split-pane drag behavior is unchanged — it already works; we're just unblocking the preview from ignoring it.
- PDF tab is unaffected.

## Expected outcome

- At default panel width (~420px): document renders at 0.6× → 420px content area. Tables visible and proportional.
- Dragged to ~700px: zoom switches to 0.88 → 616px content area. Near full-size, readable like a real document.
- "Objetivo" field: 4 words no longer wrap into 2 lines for content that would fit on one line in the real document.
