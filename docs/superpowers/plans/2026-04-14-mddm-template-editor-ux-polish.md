# MDDM Template Editor UX Polish Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make the template editing page behave like a document authoring surface: the document pane owns scrolling, the editor canvas reads clearly, the chrome is compact, and the screen can evolve into a true multi-page template workspace without breaking current MDDM behavior.

**Architecture:** Keep the existing route, editor schema, and save/publish flow intact. Limit this work to frontend composition and presentation: move scroll ownership to the document pane, introduce an explicit editor workspace layout around `MDDMEditor`, compact the metadata bar, and raise contrast in palette/sidebar controls. Do not redesign the template JSON model, API contracts, or MDDM block semantics.

**Tech Stack:** React 18, TypeScript, Zustand, Vite, Vitest, Playwright, BlockNote/Tiptap, CSS Modules, existing MetalDocs workspace shell.

---

## File Structure

**Modify**
- `frontend/apps/web/src/features/templates/TemplateEditorView.tsx`
  - Route-level composition for palette, document workspace, sidebar, and validation panel.
- `frontend/apps/web/src/features/templates/MetadataBar.tsx`
  - Compact action bar styling and clearer hierarchy.
- `frontend/apps/web/src/features/templates/PropertySidebar.tsx`
  - High-contrast sidebar chrome and denser controls.
- `frontend/apps/web/src/features/templates/BlockPalette.tsx`
  - High-contrast palette styling and denser spacing.
- `frontend/apps/web/src/features/documents/mddm-editor/MDDMEditor.tsx`
  - Editor wrapper structure for a dedicated scrollable document workspace.
- `frontend/apps/web/src/features/documents/mddm-editor/MDDMEditor.module.css`
  - Page workspace, paper stack, sticky toolbar, and internal document scroll behavior.
- `frontend/apps/web/src/features/documents/mddm-editor/mddm-editor-global.css`
  - Editor chrome contrast, spacing trims, and readability fixes.
- `frontend/apps/web/playwright/e2e/template-admin-editor.spec.ts`
  - Browser assertions for editor scroll ownership, compact chrome, and readable layout.

**Create**
- `frontend/apps/web/src/features/templates/TemplateEditorView.module.css`
  - Dedicated layout styles for the template editor route.
- `frontend/apps/web/src/features/templates/__tests__/TemplateEditorView.layout.test.tsx`
  - Render-level checks for layout ownership and compact metadata bar.

**Reference**
- `frontend/apps/web/src/components/DocumentWorkspaceShell.module.css`
  - Existing workspace shell scroll/container behavior.
- `docs/superpowers/specs/2026-04-13-mddm-template-engine-design.md`
  - Source of truth for editor behavior and current non-goals.
- `tasks/lessons.md`
  - Scroll-owner and editor initialization lessons already recorded.

---

### Task 1: Add Route-Level Editor Layout Module

**Files:**
- Create: `frontend/apps/web/src/features/templates/TemplateEditorView.module.css`
- Modify: `frontend/apps/web/src/features/templates/TemplateEditorView.tsx`
- Test: `frontend/apps/web/src/features/templates/__tests__/TemplateEditorView.layout.test.tsx`

- [ ] **Step 1: Write the failing layout test**

```tsx
import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { TemplateEditorView } from "../TemplateEditorView";

vi.mock("../useTemplateDraft", () => ({
  useTemplateDraft: () => ({
    draft: {
      templateKey: "tpl-ux",
      profileCode: "po",
      name: "Template UX",
      status: "draft",
      lockVersion: 1,
      hasStrippedFields: false,
      blocks: [],
    },
    isLoading: false,
    error: null,
    saveDraft: vi.fn(),
    publish: vi.fn(),
    discardDraft: vi.fn(),
    replaceDraft: vi.fn(),
  }),
}));

describe("TemplateEditorView layout", () => {
  it("renders a dedicated document workspace shell instead of relying on page scroll", () => {
    render(<TemplateEditorView profileCode="po" templateKey="tpl-ux" />);

    expect(screen.getByTestId("template-editor-layout")).toBeInTheDocument();
    expect(screen.getByTestId("template-editor-document-pane")).toBeInTheDocument();
    expect(screen.getByTestId("template-editor-sidebars")).toBeInTheDocument();
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `npm.cmd --prefix frontend/apps/web test -- --run src/features/templates/__tests__/TemplateEditorView.layout.test.tsx`

Expected: FAIL because `template-editor-layout` and related test ids do not exist yet.

- [ ] **Step 3: Add minimal route layout module and wire test ids**

```css
/* frontend/apps/web/src/features/templates/TemplateEditorView.module.css */
.layout {
  display: grid;
  grid-template-columns: 240px minmax(0, 1fr) 320px;
  min-height: 0;
  height: 100%;
  overflow: hidden;
  background: linear-gradient(180deg, #f3eee8 0%, #efe7df 100%);
}

.documentPane {
  min-width: 0;
  min-height: 0;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

.sidebars {
  min-height: 0;
  display: contents;
}
```

```tsx
// frontend/apps/web/src/features/templates/TemplateEditorView.tsx
import styles from "./TemplateEditorView.module.css";

// ...

<div className={styles.layout} data-testid="template-editor-layout">
  <BlockPalette editor={editorInstance} />
  <div className={styles.documentPane} data-testid="template-editor-document-pane">
    <MDDMEditor
      initialContent={Array.isArray(draft.blocks) ? (draft.blocks as PartialBlock[]) : undefined}
      onEditorReady={handleEditorReady}
      onChange={handleChange}
      onSelectionChange={setSelectedBlock}
    />
  </div>
  <div className={styles.sidebars} data-testid="template-editor-sidebars">
    <PropertySidebar editor={editorInstance} selectedBlockId={selectedBlockId} />
  </div>
  {validationErrors.length > 0 && (
    <ValidationPanel
      errors={validationErrors}
      onSelectBlock={handleValidationSelectBlock}
      onDismiss={handleValidationDismiss}
    />
  )}
</div>
```

- [ ] **Step 4: Run test to verify it passes**

Run: `npm.cmd --prefix frontend/apps/web test -- --run src/features/templates/__tests__/TemplateEditorView.layout.test.tsx`

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add frontend/apps/web/src/features/templates/TemplateEditorView.tsx frontend/apps/web/src/features/templates/TemplateEditorView.module.css frontend/apps/web/src/features/templates/__tests__/TemplateEditorView.layout.test.tsx
git commit -m "fix(templates): add dedicated editor layout shell"
```

---

### Task 2: Move Scroll Ownership Into the Document Pane

**Files:**
- Modify: `frontend/apps/web/src/features/documents/mddm-editor/MDDMEditor.tsx`
- Modify: `frontend/apps/web/src/features/documents/mddm-editor/MDDMEditor.module.css`
- Test: `frontend/apps/web/playwright/e2e/template-admin-editor.spec.ts`

- [ ] **Step 1: Add the failing browser check for document-owned scroll**

```ts
test("template editor keeps scroll inside the document pane", async ({ page }) => {
  await loginAsAdmin(page);
  await openTemplateEditor(page, "po", "tpl-scroll-owner");

  const scrollState = await page.evaluate(() => {
    const pane = document.querySelector('[data-testid="mddm-editor-scroll-shell"]') as HTMLElement | null;
    const paper = document.querySelector('[data-testid="mddm-editor-paper"]') as HTMLElement | null;
    return {
      paneOverflowY: pane ? getComputedStyle(pane).overflowY : null,
      paneScrollable: pane ? pane.scrollHeight > pane.clientHeight : false,
      paperExists: Boolean(paper),
    };
  });

  expect(scrollState.paperExists).toBe(true);
  expect(scrollState.paneOverflowY).toBe("auto");
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `npm.cmd --prefix frontend/apps/web run e2e:template-admin -- --grep "keeps scroll inside the document pane"`

Expected: FAIL because `mddm-editor-scroll-shell` does not exist and current scroll is owned by the outer page.

- [ ] **Step 3: Wrap the editor with a scroll shell and page stack**

```tsx
// frontend/apps/web/src/features/documents/mddm-editor/MDDMEditor.tsx
<div className={styles.pageShell} data-testid="mddm-editor-root">
  <div className={styles.scrollShell} data-testid="mddm-editor-scroll-shell">
    {!readOnly && (
      <div className={styles.toolbarWrapper} data-testid="mddm-editor-toolbar">
        <FormattingToolbar>{/* existing buttons */}</FormattingToolbar>
      </div>
    )}
    <div className={styles.pageStack} data-testid="mddm-editor-page-stack">
      <div
        className={styles.editorRoot}
        style={cssVars as CSSProperties}
        data-editable={!readOnly}
        data-mddm-editor-root="true"
        data-testid="mddm-editor-paper"
      >
        <BlockNoteViewEditor />
      </div>
    </div>
  </div>
</div>
```

```css
/* frontend/apps/web/src/features/documents/mddm-editor/MDDMEditor.module.css */
.pageShell {
  height: 100%;
  min-height: 0;
  background: linear-gradient(180deg, #efe7df 0%, #e7ddd2 100%);
}

.scrollShell {
  height: 100%;
  min-height: 0;
  overflow-y: auto;
  overflow-x: hidden;
  overscroll-behavior: contain;
}

.pageStack {
  display: grid;
  justify-items: center;
  gap: 24px;
  padding: 16px 20px 32px;
}
```

- [ ] **Step 4: Run the browser test to verify it passes**

Run: `npm.cmd --prefix frontend/apps/web run e2e:template-admin -- --grep "keeps scroll inside the document pane"`

Expected: PASS with `overflowY === "auto"` on `mddm-editor-scroll-shell`.

- [ ] **Step 5: Commit**

```bash
git add frontend/apps/web/src/features/documents/mddm-editor/MDDMEditor.tsx frontend/apps/web/src/features/documents/mddm-editor/MDDMEditor.module.css frontend/apps/web/playwright/e2e/template-admin-editor.spec.ts
git commit -m "fix(templates): move scroll ownership into editor workspace"
```

---

### Task 3: Tighten the Paper Surface and Remove Redundant Outer Margins

**Files:**
- Modify: `frontend/apps/web/src/features/documents/mddm-editor/MDDMEditor.module.css`
- Modify: `frontend/apps/web/src/features/documents/mddm-editor/mddm-editor-global.css`
- Test: `frontend/apps/web/playwright/e2e/template-admin-editor.spec.ts`

- [ ] **Step 1: Add the failing visual density check**

```ts
test("template editor uses a centered paper stack without oversized outer margins", async ({ page }) => {
  await loginAsAdmin(page);
  await openTemplateEditor(page, "po", "tpl-paper-density");

  const metrics = await page.evaluate(() => {
    const pageStack = document.querySelector('[data-testid="mddm-editor-page-stack"]') as HTMLElement | null;
    const paper = document.querySelector('[data-testid="mddm-editor-paper"]') as HTMLElement | null;
    if (!pageStack || !paper) return null;
    const stackBox = pageStack.getBoundingClientRect();
    const paperBox = paper.getBoundingClientRect();
    return {
      leftInset: paperBox.left - stackBox.left,
      stackPaddingTop: parseFloat(getComputedStyle(pageStack).paddingTop),
    };
  });

  expect(metrics).not.toBeNull();
  expect(metrics!.stackPaddingTop).toBeLessThanOrEqual(20);
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `npm.cmd --prefix frontend/apps/web run e2e:template-admin -- --grep "centered paper stack without oversized outer margins"`

Expected: FAIL because current desk/paper spacing is still too loose.

- [ ] **Step 3: Reduce duplicated whitespace and trim BlockNote chrome offsets**

```css
/* frontend/apps/web/src/features/documents/mddm-editor/MDDMEditor.module.css */
.toolbarWrapper {
  position: sticky;
  top: 0;
  z-index: 20;
  padding: 10px 16px;
  margin-bottom: 0;
  background: rgba(239, 231, 223, 0.94);
  backdrop-filter: blur(10px);
  border-bottom: 1px solid #d8cec2;
}

.editorRoot {
  width: min(var(--mddm-page-width, 210mm), calc(100vw - 120px));
  padding: 24mm 20mm 24mm;
  min-height: var(--mddm-page-height, 297mm);
  border: 1px solid rgba(90, 63, 49, 0.08);
  border-radius: 10px;
}
```

```css
/* frontend/apps/web/src/features/documents/mddm-editor/mddm-editor-global.css */
.bn-container .bn-editor {
  padding-left: 0 !important;
  padding-right: 0 !important;
  padding-top: 0 !important;
}

.bn-container .bn-block {
  margin-bottom: var(--mddm-block-gap, 2px);
}
```

- [ ] **Step 4: Run the browser test to verify it passes**

Run: `npm.cmd --prefix frontend/apps/web run e2e:template-admin -- --grep "centered paper stack without oversized outer margins"`

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add frontend/apps/web/src/features/documents/mddm-editor/MDDMEditor.module.css frontend/apps/web/src/features/documents/mddm-editor/mddm-editor-global.css frontend/apps/web/playwright/e2e/template-admin-editor.spec.ts
git commit -m "fix(templates): tighten editor paper spacing"
```

---

### Task 4: Compact the Metadata Bar and Action Buttons

**Files:**
- Modify: `frontend/apps/web/src/features/templates/MetadataBar.tsx`
- Test: `frontend/apps/web/src/features/templates/__tests__/TemplateEditorView.layout.test.tsx`

- [ ] **Step 1: Extend the failing render test for compact actions**

```tsx
it("renders compact template actions instead of large workspace CTAs", () => {
  render(<TemplateEditorView profileCode="po" templateKey="tpl-ux" />);

  const bar = screen.getByTestId("metadata-bar");
  expect(bar).toHaveAttribute("data-density", "compact");
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `npm.cmd --prefix frontend/apps/web test -- --run src/features/templates/__tests__/TemplateEditorView.layout.test.tsx`

Expected: FAIL because `MetadataBar` does not expose compact density yet.

- [ ] **Step 3: Restyle the metadata bar as compact editor chrome**

```tsx
// frontend/apps/web/src/features/templates/MetadataBar.tsx
<div
  data-testid="metadata-bar"
  data-density="compact"
  style={{
    display: "flex",
    alignItems: "center",
    gap: "0.5rem",
    padding: "0.375rem 0.75rem",
    borderBottom: "1px solid rgba(255,255,255,0.08)",
    background: "#1f2031",
    flexShrink: 0,
    flexWrap: "wrap",
    minHeight: "42px",
  }}
>
```

```tsx
// same file, action button style
style={{
  fontSize: "12px",
  lineHeight: 1.1,
  padding: "0.45rem 0.8rem",
  minHeight: "34px",
  borderRadius: "10px",
}}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `npm.cmd --prefix frontend/apps/web test -- --run src/features/templates/__tests__/TemplateEditorView.layout.test.tsx`

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add frontend/apps/web/src/features/templates/MetadataBar.tsx frontend/apps/web/src/features/templates/__tests__/TemplateEditorView.layout.test.tsx
git commit -m "fix(templates): compact metadata editor chrome"
```

---

### Task 5: Raise Contrast and Density in Palette and Property Sidebar

**Files:**
- Modify: `frontend/apps/web/src/features/templates/BlockPalette.tsx`
- Modify: `frontend/apps/web/src/features/templates/PropertySidebar.tsx`
- Test: `frontend/apps/web/src/features/templates/__tests__/TemplateEditorView.layout.test.tsx`

- [ ] **Step 1: Add the failing contrast-oriented render assertion**

```tsx
it("renders readable editing side panels", () => {
  render(<TemplateEditorView profileCode="po" templateKey="tpl-ux" />);

  expect(screen.getByTestId("block-palette")).toHaveAttribute("data-contrast", "high");
  expect(screen.getByTestId("property-sidebar")).toHaveAttribute("data-contrast", "high");
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `npm.cmd --prefix frontend/apps/web test -- --run src/features/templates/__tests__/TemplateEditorView.layout.test.tsx`

Expected: FAIL because neither panel declares the new chrome mode.

- [ ] **Step 3: Apply high-contrast, denser side-panel styles**

```tsx
// frontend/apps/web/src/features/templates/BlockPalette.tsx
<div
  data-testid="block-palette"
  data-contrast="high"
  style={{
    width: "220px",
    background: "#20222c",
    color: "rgba(255,255,255,0.92)",
    borderRight: "1px solid rgba(255,255,255,0.08)",
  }}
>
```

```tsx
// frontend/apps/web/src/features/templates/PropertySidebar.tsx
const sidebarStyle: React.CSSProperties = {
  width: "320px",
  minWidth: "320px",
  borderLeft: "1px solid rgba(255,255,255,0.08)",
  background: "#d8d1cb",
  color: "#2b211d",
  display: "flex",
  flexDirection: "column",
  overflow: "hidden",
};
```

```tsx
// same file
<div style={sidebarStyle} data-testid="property-sidebar" data-contrast="high">
```

- [ ] **Step 4: Run test to verify it passes**

Run: `npm.cmd --prefix frontend/apps/web test -- --run src/features/templates/__tests__/TemplateEditorView.layout.test.tsx`

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add frontend/apps/web/src/features/templates/BlockPalette.tsx frontend/apps/web/src/features/templates/PropertySidebar.tsx frontend/apps/web/src/features/templates/__tests__/TemplateEditorView.layout.test.tsx
git commit -m "fix(templates): improve editor panel contrast"
```

---

### Task 6: Add a Minimal Page-Stack Contract for Future Multi-Page Support

**Files:**
- Modify: `frontend/apps/web/src/features/documents/mddm-editor/MDDMEditor.tsx`
- Modify: `frontend/apps/web/src/features/documents/mddm-editor/MDDMEditor.module.css`
- Test: `frontend/apps/web/playwright/e2e/template-admin-editor.spec.ts`

- [ ] **Step 1: Add the failing browser assertion for page stack presence**

```ts
test("template editor exposes a page stack container for long-form authoring", async ({ page }) => {
  await loginAsAdmin(page);
  await openTemplateEditor(page, "po", "tpl-page-stack");

  await expect(page.getByTestId("mddm-editor-page-stack")).toBeVisible();
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `npm.cmd --prefix frontend/apps/web run e2e:template-admin -- --grep "page stack container for long-form authoring"`

Expected: FAIL if the page stack wrapper has not been added or is not visible.

- [ ] **Step 3: Keep the first release simple: one visible page stack, not true pagination**

```css
/* frontend/apps/web/src/features/documents/mddm-editor/MDDMEditor.module.css */
.pageStack::after {
  content: "";
  width: min(var(--mddm-page-width, 210mm), calc(100vw - 120px));
  height: 24px;
}
```

This step intentionally does not implement real content pagination. It creates the structural hook so a later task can split overflow into multiple visual pages without reworking the route layout again.

- [ ] **Step 4: Run the browser assertion to verify it passes**

Run: `npm.cmd --prefix frontend/apps/web run e2e:template-admin -- --grep "page stack container for long-form authoring"`

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add frontend/apps/web/src/features/documents/mddm-editor/MDDMEditor.tsx frontend/apps/web/src/features/documents/mddm-editor/MDDMEditor.module.css frontend/apps/web/playwright/e2e/template-admin-editor.spec.ts
git commit -m "refactor(templates): add page stack hook for editor workspace"
```

---

### Task 7: Full Verification and Evidence Pass

**Files:**
- Modify: `tasks/lessons.md` (only if new durable lessons emerge during implementation)
- Verify: `frontend/apps/web/playwright/e2e/template-admin-editor.spec.ts`
- Verify: `frontend/apps/web/src/features/templates/__tests__/TemplateEditorView.layout.test.tsx`

- [ ] **Step 1: Run focused unit tests**

Run: `npm.cmd --prefix frontend/apps/web test -- --run src/features/templates/__tests__/TemplateEditorView.layout.test.tsx src/features/templates/__tests__/BlockPaletteRules.test.ts src/features/templates/__tests__/useTemplateDraft.test.tsx`

Expected: PASS.

- [ ] **Step 2: Run the template-admin browser suite**

Run: `npm.cmd --prefix frontend/apps/web run e2e:template-admin`

Expected: PASS with the new scroll/layout assertions green.

- [ ] **Step 3: Manual visual smoke**

Run this workflow in Chrome:

```text
1. Login as admin.
2. Open a template draft from registry.
3. Verify page scroll stays stable while the editor pane scrolls.
4. Verify the paper is fully visible and not clipped.
5. Verify palette/sidebar text is readable without low-contrast fields.
6. Verify top actions read as compact editor controls, not oversized CTAs.
7. Add enough blocks to force a long document and confirm the page stack stays centered.
```

Expected: No clipping, no outer-page scroll confusion, no unreadable controls.

- [ ] **Step 4: Record any new durable lesson**

If a durable lesson emerges, append it to `tasks/lessons.md` in the required format:

```md
## Lesson N - Scroll ownership must match authoring surface
Date: 2026-04-14 | Trigger: correction
Wrong:   Template editor relied on outer workspace scrolling, clipping the document authoring surface.
Correct: The dedicated editor pane now owns vertical scrolling and keeps the workspace shell fixed.
Rule:    Document-authoring UIs must assign scroll ownership to the document workspace, not the outer app shell.
Layer:   process
```

- [ ] **Step 5: Commit**

```bash
git add frontend/apps/web/playwright/e2e/template-admin-editor.spec.ts frontend/apps/web/src/features/templates/__tests__/TemplateEditorView.layout.test.tsx tasks/lessons.md
git commit -m "test(templates): verify editor workspace ux polish"
```

---

## Self-Review

### Spec coverage
- Scroll ownership and visual authoring quality are covered by Tasks 1-3.
- Compact editor chrome is covered by Task 4.
- Palette/sidebar readability is covered by Task 5.
- The user's request for future multi-page behavior is acknowledged without overbuilding in Task 6.
- Verification and lesson capture are covered by Task 7.

### Placeholder scan
- No `TODO`, `TBD`, or “implement later” markers remain in executable steps.
- Every task has exact files, concrete commands, and explicit expected outcomes.

### Type consistency
- New route layout test ids are consistent across Tasks 1-7:
  - `template-editor-layout`
  - `template-editor-document-pane`
  - `template-editor-sidebars`
  - `mddm-editor-scroll-shell`
  - `mddm-editor-page-stack`
- The plan preserves the current `MDDMEditor` and `TemplateEditorView` interfaces and does not invent new backend contracts.

