# Preview Panel Zoom Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make the Ao Vivo live preview render the document at a readable, proportional size that adapts to the panel width via two CSS zoom levels.

**Architecture:** Remove the hardcoded `width: 380px` cap on `.content-builder-preview` so the split-pane drag actually changes the preview width. Set `.preview-document-page` to a fixed design width of 700px and use a CSS container query on `.preview-panel-body` to apply `zoom: 0.6` (medium) or `zoom: 0.88` (full) based on available width. Bump `minRightWidth` in `ResizableSplitPane` from 280 to 340 so the panel never collapses too narrow.

**Tech Stack:** CSS (container queries, `zoom` property), React/TypeScript (one prop default change only).

---

## File map

- Modify: `frontend/apps/web/src/styles.css` — remove fixed width cap, add container-type, set 700px design width, add zoom rules
- Modify: `frontend/apps/web/src/components/content-builder/ResizableSplitPane.tsx:18` — bump minRightWidth default

---

### Task 1: CSS — unblock preview width and apply zoom

**Files:**
- Modify: `frontend/apps/web/src/styles.css:2239-2249` (`.content-builder-preview` block)
- Modify: `frontend/apps/web/src/styles.css:2456-2460` (`@media (max-width: 1200px)` block — remove `.content-builder-preview` override)
- Modify: `frontend/apps/web/src/styles.css:3343-3349` (`.preview-panel-body` block)
- Modify: `frontend/apps/web/src/styles.css:3361-3366` (`.preview-document` block)
- Modify: `frontend/apps/web/src/styles.css:3368-3374` (`.preview-document-page` block)

- [ ] **Step 1: Remove the hardcoded `width: 380px` from `.content-builder-preview`**

Find this block (line ~2239):
```css
.content-builder-preview {
  width: 380px;
  border-left: 1px solid var(--border);
  background: var(--surface);
  display: flex;
  flex-direction: column;
  min-height: 0;
  position: relative;
  align-self: stretch;
  transition: width 0.2s ease;
}
```

Replace with:
```css
.content-builder-preview {
  width: 100%;
  border-left: 1px solid var(--border);
  background: var(--surface);
  display: flex;
  flex-direction: column;
  min-height: 0;
  position: relative;
  align-self: stretch;
  transition: width 0.2s ease;
}
```

- [ ] **Step 2: Remove the `@media (max-width: 1200px)` override that set `width: 320px`**

Find this block (line ~2456):
```css
@media (max-width: 1200px) {
  .content-builder-preview {
    width: 320px;
  }
}
```

Delete the three inner lines so only the media wrapper remains with no `.content-builder-preview` rule inside it. If `.content-builder-preview` is the only rule inside that media block, delete the entire block.

- [ ] **Step 3: Add `container-type: inline-size` to `.preview-panel-body`**

Find this block (line ~3343):
```css
.preview-panel-body {
  flex: 1;
  overflow-y: auto;
  overflow-x: hidden;
  padding: 16px;
  background: var(--bg);
}
```

Replace with:
```css
.preview-panel-body {
  flex: 1;
  overflow-y: auto;
  overflow-x: hidden;
  padding: 16px;
  background: var(--bg);
  container-type: inline-size;
}
```

- [ ] **Step 4: Add default `zoom: 0.6` to `.preview-document`**

Find this block (line ~3361):
```css
.preview-document {
  display: flex;
  flex-direction: column;
  align-items: stretch;
  gap: 16px;
}
```

Replace with:
```css
.preview-document {
  display: flex;
  flex-direction: column;
  align-items: stretch;
  gap: 16px;
  zoom: 0.6;
}
```

- [ ] **Step 5: Set `.preview-document-page` to 700px fixed design width**

Find this block (line ~3368):
```css
.preview-document-page {
  background: var(--surface);
  border-radius: 4px;
  box-shadow: 0 2px 12px rgba(74, 33, 33, 0.08);
  padding: 32px 28px 24px;
  width: 100%;
  max-width: 100%;
  font-size: 11px;
  line-height: 1.55;
  color: var(--text);
```

Change only `width` and `max-width`:
```css
  width: 700px;
  max-width: none;
```

- [ ] **Step 6: Add the container query for full zoom at 520px**

Immediately after the `.preview-document` block (after the closing `}` of `.preview-document`), add:
```css
@container (min-width: 520px) {
  .preview-document {
    zoom: 0.88;
  }
}
```

- [ ] **Step 7: Build and verify**

```bash
cd frontend/apps/web && npm run build
```

Expected: `✓ built in X.XXs` with no errors.

Open `http://127.0.0.1:4173/#/content-builder` (run `npm run preview` in `frontend/apps/web` if not already running).

Check:
- At default panel width (~420px): document page is visibly wider than before, tables have breathing room, text wraps less aggressively than it did at 347px.
- Drag the split handle to widen the right panel past ~520px: the document jumps to a larger zoom, appearing nearly full-size.
- `zoom: 0.6` at 700px design width = 420px rendered. Fits in a 380px+ panel after padding.
- `zoom: 0.88` at 700px design width = 616px rendered. Fits in a 650px+ panel.

- [ ] **Step 8: Commit**

```bash
git add frontend/apps/web/src/styles.css
git commit -m "feat(preview): two-level zoom via CSS container query

Remove hardcoded 380px cap, set 700px design width, apply zoom 0.6
(medium) / 0.88 (full) via container query breakpoint at 520px."
```

---

### Task 2: Bump `minRightWidth` default in ResizableSplitPane

**Files:**
- Modify: `frontend/apps/web/src/components/content-builder/ResizableSplitPane.tsx:18`

- [ ] **Step 1: Change the default from 280 to 340**

Find (line ~16-18):
```typescript
  defaultRightWidth = 420,
  minLeftWidth = 400,
  minRightWidth = 280,
```

Replace with:
```typescript
  defaultRightWidth = 420,
  minLeftWidth = 400,
  minRightWidth = 340,
```

- [ ] **Step 2: Build and verify**

```bash
cd frontend/apps/web && npm run build
```

Expected: `✓ built in X.XXs` with no errors.

In the browser at `http://127.0.0.1:4173/#/content-builder`, drag the split handle all the way left. The right panel should stop at 340px instead of 280px.

- [ ] **Step 3: Commit**

```bash
git add frontend/apps/web/src/components/content-builder/ResizableSplitPane.tsx
git commit -m "fix(split-pane): raise minRightWidth default to 340px

Prevents preview panel from collapsing so narrow that the zoomed
document becomes unreadable."
```
