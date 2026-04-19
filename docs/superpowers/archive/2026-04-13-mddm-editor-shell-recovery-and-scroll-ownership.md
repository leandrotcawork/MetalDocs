# MDDM Editor Shell Recovery and Scroll Ownership Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Restore full MDDM editor behavior (toolbar, editable sections/tables, template interactions) while fixing the dual-scroll bug through container hierarchy and scroll-owner corrections only.

**Architecture:** Preserve the MDDM unified engine contract (`MDDM React renderer` + `.docx` renderer parity) and keep editor business logic unchanged. Apply a behavior-first rollback to known-good interaction points, then reintroduce structural shell/layout changes in small verified steps. Enforce one scroll owner in the content-builder path and lock root viewport scrolling without mutating block schema, adapters, or rendering semantics.

**Tech Stack:** React 18, TypeScript, BlockNote, CSS Modules, Vitest + Testing Library.

---

## File Structure Map

- `frontend/apps/web/src/features/documents/mddm-editor/MDDMEditor.tsx`
  - Responsibility: MDDM editor runtime wiring, toolbar mounting, lock guards, BlockNote view configuration.
- `frontend/apps/web/src/features/documents/mddm-editor/MDDMEditor.module.css`
  - Responsibility: editor-local shell layout only (toolbar band, paper, viewport containment).
- `frontend/apps/web/src/features/documents/mddm-editor/mddm-editor-global.css`
  - Responsibility: scoped BlockNote overrides, no global page/viewport ownership.
- `frontend/apps/web/src/features/documents/browser-editor/BrowserDocumentEditorView.tsx`
  - Responsibility: browser editor structure and containment wrappers for the content-builder surface.
- `frontend/apps/web/src/features/documents/browser-editor/BrowserDocumentEditorView.module.css`
  - Responsibility: parent grid/flex containment and min-height/overflow boundaries.
- `frontend/apps/web/src/components/DocumentWorkspaceShell.module.css`
  - Responsibility: single scroll owner for content-builder workspace.
- `frontend/apps/web/src/styles/base.css`
  - Responsibility: root viewport lock (`html/body/#root`) and browser-scroll suppression.
- `frontend/apps/web/src/styles.css`
  - Responsibility: app shell viewport sizing fallback (`100vh/100dvh`) without behavior changes.
- `frontend/apps/web/src/features/documents/mddm-editor/__tests__/MDDMEditor.test.tsx`
  - Responsibility: regression tests for toolbar visibility and editability invariants.
- `frontend/apps/web/src/features/documents/browser-editor/__tests__/BrowserDocumentEditorView.structure.test.tsx`
  - Responsibility: structure-level assertions for containment wrappers.
- `frontend/apps/web/src/features/documents/mddm-editor/__tests__/editor-shell-contract.test.ts`
  - Responsibility: CSS contract checks for scoped overrides.
- `tasks/lessons.md`
  - Responsibility: append post-correction lesson with root cause and preventative rule.

---

### Task 1: Freeze baseline and isolate behavioral regression

**Files:**
- Modify: `frontend/apps/web/src/features/documents/mddm-editor/__tests__/MDDMEditor.test.tsx`
- Modify: `frontend/apps/web/src/features/documents/browser-editor/__tests__/BrowserDocumentEditorView.structure.test.tsx`
- Test: same files

- [ ] **Step 1: Write failing tests for missing toolbar/editability invariants**

```tsx
it("keeps formatting toolbar mounted when editor is editable", async () => {
  render(<MDDMEditor {...editableProps} />);
  expect(await screen.findByTestId("mddm-editor-toolbar")).toBeInTheDocument();
});

it("keeps contenteditable enabled on editable table cells", async () => {
  render(<MDDMEditor {...editableProps} />);
  const cell = await screen.findByTestId("mddm-table-editable-cell");
  expect(cell).toHaveAttribute("contenteditable", "true");
});
```

- [ ] **Step 2: Run tests to verify failure**

Run: `cd frontend/apps/web; npm.cmd run test -- src/features/documents/mddm-editor/__tests__/MDDMEditor.test.tsx src/features/documents/browser-editor/__tests__/BrowserDocumentEditorView.structure.test.tsx`
Expected: FAIL with missing toolbar mount and/or non-editable cell assertions.

- [ ] **Step 3: Add structure assertion to guard editor containment**

```tsx
it("renders browser editor viewport with surface and footer siblings", () => {
  render(<BrowserDocumentEditorView {...props} />);
  expect(screen.getByTestId("browser-editor-surface")).toBeInTheDocument();
  expect(screen.getByTestId("browser-editor-viewport")).toBeInTheDocument();
  expect(screen.getByTestId("browser-editor-footer")).toBeInTheDocument();
});
```

- [ ] **Step 4: Run tests to verify expected failing point is explicit**

Run: `cd frontend/apps/web; npm.cmd run test -- src/features/documents/mddm-editor/__tests__/MDDMEditor.test.tsx src/features/documents/browser-editor/__tests__/BrowserDocumentEditorView.structure.test.tsx`
Expected: FAIL only on behavior regressions, PASS on static structure checks not impacted.

- [ ] **Step 5: Commit baseline regression lock**

```bash
git add frontend/apps/web/src/features/documents/mddm-editor/__tests__/MDDMEditor.test.tsx frontend/apps/web/src/features/documents/browser-editor/__tests__/BrowserDocumentEditorView.structure.test.tsx
git commit -m "test(mddm-editor): lock toolbar and editability regression invariants"
```

---

### Task 2: Restore editor behavior to pre-regression contract (no layout innovation)

**Files:**
- Modify: `frontend/apps/web/src/features/documents/mddm-editor/MDDMEditor.tsx`
- Modify: `frontend/apps/web/src/features/documents/mddm-editor/MDDMEditor.module.css`
- Test: `frontend/apps/web/src/features/documents/mddm-editor/__tests__/MDDMEditor.test.tsx`

- [ ] **Step 1: Write failing test for BlockNote interaction contract**

```tsx
it("renders BlockNoteView with editable interaction props", async () => {
  render(<MDDMEditor {...editableProps} />);
  const editorRoot = await screen.findByTestId("mddm-editor-root");
  expect(editorRoot).toHaveAttribute("data-mddm-editor-root", "true");
  expect(screen.getByTestId("mddm-editor-toolbar")).toBeVisible();
});
```

- [ ] **Step 2: Run test to verify it fails on current code**

Run: `cd frontend/apps/web; npm.cmd run test -- src/features/documents/mddm-editor/__tests__/MDDMEditor.test.tsx -t "editable interaction props"`
Expected: FAIL indicating missing toolbar and/or missing editor root contract.

- [ ] **Step 3: Restore minimal known-good editor wiring**

```tsx
<div className={styles.root} data-testid="mddm-editor-root" data-mddm-editor-root="true">
  <div className={styles.toolbarBand} data-testid="mddm-editor-toolbar">
    <FormattingToolbar controller={controller} />
  </div>
  <div className={styles.viewport}>
    <div className={styles.paper}>
      <BlockNoteView editor={editor} editable={editable} tableHandles={false} />
    </div>
  </div>
</div>
```

```css
.root { min-height: 0; height: 100%; display: flex; flex-direction: column; }
.toolbarBand { position: sticky; top: 0; z-index: 3; }
.viewport { min-height: 0; flex: 1 1 auto; overflow: visible; }
.paper { min-height: 100%; }
```

- [ ] **Step 4: Run test suite to verify behavior restoration**

Run: `cd frontend/apps/web; npm.cmd run test -- src/features/documents/mddm-editor/__tests__/MDDMEditor.test.tsx`
Expected: PASS for toolbar visibility and editable table behavior checks.

- [ ] **Step 5: Commit behavior restoration**

```bash
git add frontend/apps/web/src/features/documents/mddm-editor/MDDMEditor.tsx frontend/apps/web/src/features/documents/mddm-editor/MDDMEditor.module.css frontend/apps/web/src/features/documents/mddm-editor/__tests__/MDDMEditor.test.tsx
git commit -m "fix(mddm-editor): restore toolbar and editable template interactions"
```

---

### Task 3: Lock root viewport and remove browser-level scroll leakage

**Files:**
- Modify: `frontend/apps/web/src/styles/base.css`
- Modify: `frontend/apps/web/src/styles.css`
- Test: `frontend/apps/web/src/features/documents/mddm-editor/__tests__/editor-shell-contract.test.ts`

- [ ] **Step 1: Write failing CSS contract test for root lock**

```ts
it("locks html/body/root scrolling in workspace mode", () => {
  const css = fs.readFileSync(baseCssPath, "utf8");
  expect(css).toContain("html,");
  expect(css).toContain("body,");
  expect(css).toContain("#root");
  expect(css).toContain("overflow: hidden");
});
```

- [ ] **Step 2: Run test to verify failure**

Run: `cd frontend/apps/web; npm.cmd run test -- src/features/documents/mddm-editor/__tests__/editor-shell-contract.test.ts -t "locks html/body/root scrolling"`
Expected: FAIL if root lock is absent/incomplete.

- [ ] **Step 3: Implement root viewport lock and dvh fallback**

```css
html,
body,
#root {
  height: 100%;
  min-height: 100%;
  overflow: hidden;
}
```

```css
.app-shell.is-workspace {
  min-height: 100vh;
  min-height: 100dvh;
  height: 100vh;
  height: 100dvh;
  overflow: hidden;
}
```

- [ ] **Step 4: Run CSS contract tests**

Run: `cd frontend/apps/web; npm.cmd run test -- src/features/documents/mddm-editor/__tests__/editor-shell-contract.test.ts`
Expected: PASS with explicit root lock assertions.

- [ ] **Step 5: Commit viewport lock**

```bash
git add frontend/apps/web/src/styles/base.css frontend/apps/web/src/styles.css frontend/apps/web/src/features/documents/mddm-editor/__tests__/editor-shell-contract.test.ts
git commit -m "fix(frontend-shell): lock root viewport to prevent browser scroll leak"
```

---

### Task 4: Enforce single scroll owner in workspace content-builder

**Files:**
- Modify: `frontend/apps/web/src/components/DocumentWorkspaceShell.module.css`
- Modify: `frontend/apps/web/src/features/documents/browser-editor/BrowserDocumentEditorView.module.css`
- Test: `frontend/apps/web/src/features/documents/browser-editor/__tests__/BrowserDocumentEditorView.structure.test.tsx`

- [ ] **Step 1: Write failing structure test for scroll-owner contract**

```tsx
it("keeps editor descendants non-scrolling while workspace owns scrolling", () => {
  render(<BrowserDocumentEditorView {...props} />);
  const viewport = screen.getByTestId("browser-editor-viewport");
  expect(viewport).toBeInTheDocument();
});
```

- [ ] **Step 2: Run structure tests**

Run: `cd frontend/apps/web; npm.cmd run test -- src/features/documents/browser-editor/__tests__/BrowserDocumentEditorView.structure.test.tsx`
Expected: FAIL if markup or expected containment hooks are missing.

- [ ] **Step 3: Apply scroll-owner CSS boundaries**

```css
/* DocumentWorkspaceShell.module.css */
.workspace-main.is-content-builder-view {
  display: flex;
  flex-direction: column;
  min-height: 0;
  overflow: auto;
  overscroll-behavior: contain;
}

.workspace-main.is-content-builder-view > * {
  min-height: 0;
  flex: 1 1 auto;
}
```

```css
/* BrowserDocumentEditorView.module.css */
.root { min-height: 0; height: 100%; overflow: hidden; }
.surface { min-height: 0; height: 100%; overflow: hidden; }
.editorShell { min-height: 0; overflow: visible; }
```

- [ ] **Step 4: Run focused tests**

Run: `cd frontend/apps/web; npm.cmd run test -- src/features/documents/browser-editor/__tests__/BrowserDocumentEditorView.structure.test.tsx src/features/documents/mddm-editor/__tests__/editor-shell-contract.test.ts`
Expected: PASS with structure and CSS contract green.

- [ ] **Step 5: Commit single-scroll-owner implementation**

```bash
git add frontend/apps/web/src/components/DocumentWorkspaceShell.module.css frontend/apps/web/src/features/documents/browser-editor/BrowserDocumentEditorView.module.css frontend/apps/web/src/features/documents/browser-editor/__tests__/BrowserDocumentEditorView.structure.test.tsx
git commit -m "fix(content-builder): enforce single workspace scroll owner"
```

---

### Task 5: Scope BlockNote overrides to editor root and protect interactions

**Files:**
- Modify: `frontend/apps/web/src/features/documents/mddm-editor/mddm-editor-global.css`
- Modify: `frontend/apps/web/src/features/documents/mddm-editor/__tests__/editor-shell-contract.test.ts`
- Test: same file

- [ ] **Step 1: Write failing contract test for scoped BlockNote selectors**

```ts
it("scopes BlockNote override selectors under data-mddm-editor-root", () => {
  const css = fs.readFileSync(globalCssPath, "utf8");
  expect(css).toContain('[data-mddm-editor-root="true"] .bn-table-handle');
  expect(css).toContain('[data-mddm-editor-root="true"] .bn-table-handle-menu');
});
```

- [ ] **Step 2: Run test to verify failure**

Run: `cd frontend/apps/web; npm.cmd run test -- src/features/documents/mddm-editor/__tests__/editor-shell-contract.test.ts -t "scopes BlockNote override selectors"`
Expected: FAIL if selectors are global/non-scoped.

- [ ] **Step 3: Implement scoped overrides only**

```css
[data-mddm-editor-root="true"] .bn-table-handle,
[data-mddm-editor-root="true"] .bn-table-handle-menu,
[data-mddm-editor-root="true"] .bn-table-cell-handle {
  display: none !important;
}
```

- [ ] **Step 4: Run contract tests**

Run: `cd frontend/apps/web; npm.cmd run test -- src/features/documents/mddm-editor/__tests__/editor-shell-contract.test.ts`
Expected: PASS with all scope checks green.

- [ ] **Step 5: Commit scoped override hardening**

```bash
git add frontend/apps/web/src/features/documents/mddm-editor/mddm-editor-global.css frontend/apps/web/src/features/documents/mddm-editor/__tests__/editor-shell-contract.test.ts
git commit -m "fix(mddm-editor): scope BlockNote overrides to editor root"
```

---

### Task 6: Full verification and manual QA protocol (no further code changes)

**Files:**
- Modify: `docs/superpowers/reports/2026-04-13-mddm-editor-scroll-regression-verification.md` (create if missing)
- Modify: `tasks/lessons.md`

- [ ] **Step 1: Run automated regression pack**

Run: `cd frontend/apps/web; npm.cmd run test -- src/features/documents/mddm-editor/__tests__/MDDMEditor.test.tsx src/features/documents/browser-editor/__tests__/BrowserDocumentEditorView.structure.test.tsx src/features/documents/mddm-editor/__tests__/editor-shell-contract.test.ts`
Expected: PASS (all tests green).

Run: `cd frontend/apps/web; npm.cmd run build`
Expected: PASS with no TypeScript or CSS module errors.

- [ ] **Step 2: Run manual browser protocol and record result**

Run in DevTools console during content-builder usage:

```js
console.log({
  innerHeight: window.innerHeight,
  docElScrollHeight: document.documentElement.scrollHeight,
  bodyScrollHeight: document.body.scrollHeight,
  rootScrollHeight: document.getElementById("root")?.scrollHeight
});
```

Expected:
- `docElScrollHeight === innerHeight`
- `bodyScrollHeight === innerHeight`
- only `.workspace-main.is-content-builder-view` changes `scrollTop`

- [ ] **Step 3: Save verification report with exact outcomes**

```md
# MDDM Editor Scroll Regression Verification (2026-04-13)
- Automated tests: PASS
- Build: PASS
- Manual QA:
  - Toolbar visible: PASS
  - Sections render: PASS
  - Table cells editable: PASS
  - Right grey browser scrollbar leak: PASS (not reproducible)
  - Root scroll metrics check: PASS
```

- [ ] **Step 4: Append lesson entry for process hardening**

```md
## Lesson N - Scroll/layout fixes must preserve editor interaction contract
Date: 2026-04-13 | Trigger: correction
Wrong:   Scroll containment changes were merged without preserving toolbar/editability invariants in MDDM editor runtime.
Correct: Behavior-critical invariants now have regression tests and layout fixes are applied only after behavior baseline passes.
Rule:    For editor UX bugs, restore and lock user interaction invariants before applying structural scroll fixes.
Layer:   process
```

- [ ] **Step 5: Commit verification artifacts**

```bash
git add docs/superpowers/reports/2026-04-13-mddm-editor-scroll-regression-verification.md tasks/lessons.md
git commit -m "docs(mddm-editor): record scroll-regression verification evidence"
```

---

## Self-Review

### 1. Spec coverage
- Preserves MDDM architecture and renderer parity contract from `docs/superpowers/specs/2026-04-10-mddm-unified-document-engine-design.md`: covered by Tasks 2, 5 (no logic/schema changes, runtime behavior restoration).
- Solves dual-scroll leak with clear scroll ownership: covered by Tasks 3 and 4.
- Restores broken editor UX (toolbar, sections, table editability): covered by Tasks 1 and 2.
- Ensures repeatable verification and auditability: covered by Task 6.

### 2. Placeholder scan
- No `TODO`, `TBD`, or “implement later”.
- Each task has explicit files, concrete code snippets, commands, and expected outcomes.

### 3. Type/signature consistency
- Test IDs and selectors are consistent across tasks:
  - `mddm-editor-root`
  - `mddm-editor-toolbar`
  - `browser-editor-surface`
  - `browser-editor-viewport`
  - `browser-editor-footer`
- Scoped selector key remains consistent:
  - `[data-mddm-editor-root="true"]`

---

Plan complete and saved to `docs/superpowers/plans/2026-04-13-mddm-editor-shell-recovery-and-scroll-ownership.md`. Two execution options:

**1. Subagent-Driven (recommended)** - I dispatch a fresh subagent per task, review between tasks, fast iteration

**2. Inline Execution** - Execute tasks in this session using executing-plans, batch execution with checkpoints

Which approach?
