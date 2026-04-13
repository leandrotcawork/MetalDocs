# Targeted Editor Shell Cleanup Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Restructure only the browser editor shell so BlockNote chrome stays inside the editor viewport instead of leaking into the workspace top bar or phantom canvas area.

**Architecture:** `BrowserDocumentEditorView` becomes the sole page-shell owner for the browser editor and gets an explicit `editorViewport` containment boundary with stable DOM test ids. `MDDMEditor` is narrowed to a local BlockNote adapter with one local root, one sticky toolbar band, one paper surface, and explicit suppression of table-handle chrome that should never be visible for structural field tables.

**Tech Stack:** React 18, TypeScript, BlockNote 0.47, CSS Modules, Vitest/jsdom

---

## Known Baseline

- `frontend/apps/web/src/components/DocumentWorkspaceShell.module.css` already has unrelated local work in the current tree. Do not touch `DocumentWorkspaceShell.*` in this plan.
- `cd frontend/apps/web; npm.cmd run build` is currently blocked by unrelated TypeScript errors in `src/features/documents/runtime/fields/RichSlot.tsx` and `RichField.tsx`. This plan uses targeted Vitest suites plus manual browser verification and must not widen scope into those runtime files.

## File Structure & Ownership

| File | Action | Responsibility |
|------|--------|---------------|
| `frontend/apps/web/src/features/documents/browser-editor/BrowserDocumentEditorView.tsx` | Modify | Make the browser editor page the single shell owner and add an explicit `editorViewport` boundary with stable `data-testid`s |
| `frontend/apps/web/src/features/documents/browser-editor/BrowserDocumentEditorView.module.css` | Modify | Replace the current shell-inside-shell layout with `surface -> editorViewport -> documentFrame -> editorShell` and remove dead CKEditor-only rules |
| `frontend/apps/web/src/features/documents/browser-editor/__tests__/BrowserDocumentEditorView.structure.test.tsx` | Create | Lock the browser editor DOM contract so future layout edits cannot remove the containment boundary |
| `frontend/apps/web/src/features/documents/mddm-editor/MDDMEditor.tsx` | Modify | Reduce `MDDMEditor` to a local BlockNote adapter, add stable boundary test ids, and disable native BlockNote table handles |
| `frontend/apps/web/src/features/documents/mddm-editor/MDDMEditor.module.css` | Modify | Replace `pageShell/toolbarWrapper/editorRoot` with `root/chrome/toolbarInner/paperViewport/paperSurface` |
| `frontend/apps/web/src/features/documents/mddm-editor/__tests__/MDDMEditor.test.tsx` | Modify | Assert the new local-root structure and `BlockNoteView` props (`tableHandles={false}`) |
| `frontend/apps/web/src/features/documents/mddm-editor/__tests__/editor-shell-contract.test.ts` | Create | Guard the root attribute + bridge CSS selectors that keep table handle chrome suppressed |
| `frontend/apps/web/src/features/documents/mddm-editor/mddm-editor-global.css` | Modify | Keep BlockNote bridge CSS aligned with the new local root and explicitly suppress table-handle chrome |

**Unchanged on purpose:**
- `frontend/apps/web/src/components/DocumentWorkspaceShell.tsx`
- `frontend/apps/web/src/components/DocumentWorkspaceShell.module.css`
- `frontend/apps/web/src/styles/base.css`
- `frontend/apps/web/src/styles.css`

---

### Task 1: Add An Explicit Browser Editor Viewport Boundary

**Files:**
- Create: `frontend/apps/web/src/features/documents/browser-editor/__tests__/BrowserDocumentEditorView.structure.test.tsx`
- Modify: `frontend/apps/web/src/features/documents/browser-editor/BrowserDocumentEditorView.tsx`
- Modify: `frontend/apps/web/src/features/documents/browser-editor/BrowserDocumentEditorView.module.css`

- [ ] **Step 1: Write the failing structure test**

```tsx
// @vitest-environment jsdom
import { createRoot } from "react-dom/client";
import { act } from "react-dom/test-utils";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

const {
  getDocumentBrowserEditorBundleMock,
  saveDocumentBrowserContentMock,
} = vi.hoisted(() => ({
  getDocumentBrowserEditorBundleMock: vi.fn(),
  saveDocumentBrowserContentMock: vi.fn(),
}));

vi.mock("../../../../api/documents", () => ({
  getDocumentBrowserEditorBundle: getDocumentBrowserEditorBundleMock,
  saveDocumentBrowserContent: saveDocumentBrowserContentMock,
}));

vi.mock("../DocumentEditorHeader", () => ({
  DocumentEditorHeader: () => <div data-testid="document-editor-header" />,
}));

vi.mock("../../mddm-editor/MDDMEditor", () => ({
  MDDMEditor: () => <div data-testid="mddm-editor-root" />,
}));

vi.mock("../../mddm-editor/MDDMViewer", () => ({
  MDDMViewer: () => <div data-testid="mddm-viewer-root" />,
}));

vi.mock("../SaveBeforeExportDialog", () => ({
  SaveBeforeExportDialog: () => null,
}));

import { BrowserDocumentEditorView } from "../BrowserDocumentEditorView";

const sampleDocument = {
  documentId: "doc-123",
  documentCode: "PO-001",
  title: "Documento teste",
  status: "DRAFT",
  documentProfile: "po",
} as any;

function sampleBundle() {
  return {
    document: sampleDocument,
    versions: [{ version: 1, createdAt: "2026-04-13T00:00:00Z", renderer_pin: null }],
    governance: {},
    templateSnapshot: {
      templateKey: "po-default",
      definition: { theme: null },
    },
    body: JSON.stringify({ mddm_version: 1, template_ref: null, blocks: [] }),
    draftToken: "draft-1",
  } as any;
}

describe("BrowserDocumentEditorView structure", () => {
  let host: HTMLDivElement;
  let root: ReturnType<typeof createRoot>;

  beforeEach(() => {
    host = document.createElement("div");
    document.body.appendChild(host);
    root = createRoot(host);
    getDocumentBrowserEditorBundleMock.mockResolvedValue(sampleBundle());
  });

  afterEach(() => {
    act(() => root.unmount());
    host.remove();
    vi.clearAllMocks();
  });

  it("renders a dedicated editor viewport between the surface and footer", async () => {
    act(() => {
      root.render(<BrowserDocumentEditorView document={sampleDocument} onBack={vi.fn()} />);
    });

    await act(async () => {
      await Promise.resolve();
    });

    const shell = host.querySelector('[data-testid="browser-document-editor"]');
    const surface = host.querySelector('[data-testid="browser-editor-surface"]');
    const viewport = host.querySelector('[data-testid="browser-editor-viewport"]');
    const footer = host.querySelector('[data-testid="browser-editor-footer"]');

    expect(shell).not.toBeNull();
    expect(surface).not.toBeNull();
    expect(viewport).not.toBeNull();
    expect(footer).not.toBeNull();
    expect(surface?.contains(viewport)).toBe(true);
    expect(shell?.lastElementChild).toBe(footer);
  });

  it("keeps the document header and editor root inside the viewport boundary", async () => {
    act(() => {
      root.render(<BrowserDocumentEditorView document={sampleDocument} onBack={vi.fn()} />);
    });

    await act(async () => {
      await Promise.resolve();
    });

    const viewport = host.querySelector('[data-testid="browser-editor-viewport"]');
    expect(viewport?.querySelector('[data-testid="document-editor-header"]')).not.toBeNull();
    expect(viewport?.querySelector('[data-testid="mddm-editor-root"]')).not.toBeNull();
  });
});
```

- [ ] **Step 2: Run the test and verify it fails**

Run: `cd frontend/apps/web; npm.cmd run test -- src/features/documents/browser-editor/__tests__/BrowserDocumentEditorView.structure.test.tsx`  
Expected: FAIL because `BrowserDocumentEditorView` does not yet render `browser-editor-surface`, `browser-editor-viewport`, or `browser-editor-footer`

- [ ] **Step 3: Implement the explicit viewport boundary and remove the dead nested shell CSS**

```tsx
// BrowserDocumentEditorView.tsx
return (
  <section className={styles.root} data-testid="browser-document-editor">
    <header className={styles.topbar} data-testid="browser-editor-topbar">
      ...
    </header>

    <div className={styles.metaBar} data-testid="browser-editor-meta">
      ...
    </div>

    {bundle ? (
      <div className={styles.surface} data-testid="browser-editor-surface">
        {showInlineError ? (
          <div className={styles.errorBanner} role="alert">
            ...
          </div>
        ) : null}

        <div className={styles.editorViewport} data-testid="browser-editor-viewport">
          <div className={styles.documentFrame}>
            <DocumentEditorHeader bundle={bundle} />
            <div className={styles.editorShell}>
              {blockNoteDocument ? (
                isViewOnly ? (
                  <MDDMViewer ... />
                ) : (
                  <MDDMEditor ... />
                )
              ) : (
                <div className={styles.stateCard} role="alert">
                  ...
                </div>
              )}
            </div>
          </div>
        </div>
      </div>
    ) : (
      <div className={styles.statePanel}>
        ...
      </div>
    )}

    <footer className={styles.footer} data-testid="browser-editor-footer">
      ...
    </footer>
    ...
  </section>
);
```

```css
/* BrowserDocumentEditorView.module.css */
.surface {
  min-height: 0;
  display: grid;
  grid-template-rows: auto minmax(0, 1fr);
  padding: 1rem 1rem 0;
  gap: 0.75rem;
}

.editorViewport {
  position: relative;
  min-height: 0;
  isolation: isolate;
  padding: 0.25rem;
  border-radius: 1.5rem 1.5rem 0 0;
  background:
    linear-gradient(180deg, rgba(237, 233, 228, 0.98), rgba(237, 233, 228, 0.9));
  overflow: clip;
}

.documentFrame {
  min-height: 0;
  display: grid;
  gap: 0.75rem;
  align-content: start;
}

.editorShell {
  min-height: 0;
  border-radius: 1.4rem;
  overflow: clip;
  background: transparent;
  border: 0;
}
```

Delete the unused `.toolbarShell` block and the old `.editorShell :global(.ck...)` selectors from `BrowserDocumentEditorView.module.css`; they belong to the removed CKEditor path and keep the current file harder to reason about.

- [ ] **Step 4: Run the browser editor structure test again**

Run: `cd frontend/apps/web; npm.cmd run test -- src/features/documents/browser-editor/__tests__/BrowserDocumentEditorView.structure.test.tsx`  
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add frontend/apps/web/src/features/documents/browser-editor/BrowserDocumentEditorView.tsx frontend/apps/web/src/features/documents/browser-editor/BrowserDocumentEditorView.module.css frontend/apps/web/src/features/documents/browser-editor/__tests__/BrowserDocumentEditorView.structure.test.tsx
git commit -m "refactor(frontend-editor): add explicit browser editor viewport boundary"
```

---

### Task 2: Reduce MDDMEditor To One Local Root, One Toolbar Band, One Paper Surface

**Files:**
- Modify: `frontend/apps/web/src/features/documents/mddm-editor/__tests__/MDDMEditor.test.tsx`
- Modify: `frontend/apps/web/src/features/documents/mddm-editor/MDDMEditor.tsx`
- Modify: `frontend/apps/web/src/features/documents/mddm-editor/MDDMEditor.module.css`

- [ ] **Step 1: Extend the editor test with the failing local-root and table-handle assertions**

```tsx
// MDDMEditor.test.tsx - extend the hoisted mocks
const {
  useCreateBlockNoteMock,
  uploadAttachmentMock,
  getAttachmentDownloadURLMock,
  blockNoteViewPropsMock,
} = vi.hoisted(() => ({
  useCreateBlockNoteMock: vi.fn(() => editor),
  uploadAttachmentMock: vi.fn(),
  getAttachmentDownloadURLMock: vi.fn(),
  blockNoteViewPropsMock: vi.fn(),
}));

vi.mock("@blocknote/mantine", () => ({
  BlockNoteView: ({ children, ...props }: { children?: import("react").ReactNode }) => {
    blockNoteViewPropsMock(props);
    return children ?? null;
  },
}));
```

```tsx
// MDDMEditor.test.tsx - add the new tests
it("renders one local editor boundary with toolbar and paper surface", () => {
  const host = document.createElement("div");
  document.body.appendChild(host);
  const root = createRoot(host);

  act(() => {
    root.render(<MDDMEditor />);
  });

  expect(host.querySelector('[data-testid="mddm-editor-root"]')).not.toBeNull();
  expect(host.querySelector('[data-testid="mddm-editor-toolbar"]')).not.toBeNull();
  expect(host.querySelector('[data-testid="mddm-editor-paper"]')).not.toBeNull();

  act(() => {
    root.unmount();
  });
  host.remove();
});

it("disables native BlockNote table handles for the browser document editor shell", () => {
  const host = document.createElement("div");
  document.body.appendChild(host);
  const root = createRoot(host);

  act(() => {
    root.render(<MDDMEditor />);
  });

  const props = blockNoteViewPropsMock.mock.calls[0]?.[0] as { tableHandles?: boolean } | undefined;
  expect(props?.tableHandles).toBe(false);

  act(() => {
    root.unmount();
  });
  host.remove();
});

it("omits the local toolbar band in readOnly mode", () => {
  const host = document.createElement("div");
  document.body.appendChild(host);
  const root = createRoot(host);

  act(() => {
    root.render(<MDDMEditor readOnly />);
  });

  expect(host.querySelector('[data-testid="mddm-editor-toolbar"]')).toBeNull();

  act(() => {
    root.unmount();
  });
  host.remove();
});
```

- [ ] **Step 2: Run the editor test and verify it fails**

Run: `cd frontend/apps/web; npm.cmd run test -- src/features/documents/mddm-editor/__tests__/MDDMEditor.test.tsx`  
Expected: FAIL because `MDDMEditor` still renders `pageShell/toolbarWrapper/editorRoot` and does not pass `tableHandles={false}`

- [ ] **Step 3: Implement the local root and disable table handles**

```tsx
// MDDMEditor.tsx
return (
  <div
    className={styles.root}
    style={cssVars as CSSProperties}
    data-testid="mddm-editor-root"
    data-mddm-editor-root="true"
    data-editable={!readOnly}
  >
    <BlockNoteView
      editor={editor}
      editable={!readOnly}
      formattingToolbar={false}
      renderEditor={false}
      tableHandles={false}
      onChange={(currentEditor) => onChange?.(currentEditor.document)}
    >
      {!readOnly && (
        <div className={styles.chrome} data-testid="mddm-editor-toolbar">
          <div className={styles.toolbarInner}>
            <FormattingToolbar>
              {getFormattingToolbarItems()}
            </FormattingToolbar>
          </div>
        </div>
      )}

      <div className={styles.paperViewport}>
        <div className={styles.paperSurface} data-testid="mddm-editor-paper">
          <BlockNoteViewEditor />
        </div>
      </div>
    </BlockNoteView>
  </div>
);
```

```css
/* MDDMEditor.module.css */
.root {
  position: relative;
  min-height: 0;
  isolation: isolate;
  background: #ede9e4;
  padding: 0 0 2rem;
}

.chrome {
  position: sticky;
  top: 0;
  z-index: 3;
  padding: 0.5rem 1rem;
  background: #ede9e4;
  border-bottom: 1px solid #d4cfc9;
}

.toolbarInner {
  display: flex;
  justify-content: center;
}

.paperViewport {
  min-height: 0;
  padding: 1.5rem 1rem 0;
}

.paperSurface {
  max-width: var(--mddm-content-max-width);
  margin: 0 auto;
  background: var(--mddm-raw-white);
  padding: 2.5rem 3rem;
  min-height: min(842px, calc(100dvh - 16rem));
  font-family: var(--mddm-font-family);
  font-size: var(--mddm-font-size-base);
  color: var(--mddm-raw-gray-700);
  line-height: 1.6;
  counter-reset: mddm-section;
  box-shadow:
    0 1px 2px rgba(0, 0, 0, 0.06),
    0 4px 12px rgba(0, 0, 0, 0.06),
    0 12px 40px rgba(0, 0, 0, 0.05);
}
```

Delete the old `.pageShell`, `.toolbarWrapper`, and `.editorRoot` rules from `MDDMEditor.module.css`; after this task they should not exist anywhere in the file.

- [ ] **Step 4: Run the editor test again**

Run: `cd frontend/apps/web; npm.cmd run test -- src/features/documents/mddm-editor/__tests__/MDDMEditor.test.tsx`  
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add frontend/apps/web/src/features/documents/mddm-editor/MDDMEditor.tsx frontend/apps/web/src/features/documents/mddm-editor/MDDMEditor.module.css frontend/apps/web/src/features/documents/mddm-editor/__tests__/MDDMEditor.test.tsx
git commit -m "refactor(frontend-editor): narrow mddm editor to one local shell"
```

---

### Task 3: Lock The Bridge CSS Against Returning Table-Handle Chrome

**Files:**
- Create: `frontend/apps/web/src/features/documents/mddm-editor/__tests__/editor-shell-contract.test.ts`
- Modify: `frontend/apps/web/src/features/documents/mddm-editor/mddm-editor-global.css`

- [ ] **Step 1: Write the failing contract test for the root attribute and handle selectors**

```ts
/// <reference types="node" />

import { readFileSync } from "node:fs";
import { dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";
import { describe, expect, it } from "vitest";

const __dirname = dirname(fileURLToPath(import.meta.url));
const mddmEditorDir = resolve(__dirname, "..");
const browserEditorDir = resolve(mddmEditorDir, "../browser-editor");

function readUtf8(path: string): string {
  return readFileSync(path, "utf8");
}

function normalize(value: string): string {
  return value.replace(/\s+/g, " ").trim();
}

describe("editor shell contracts", () => {
  it("keeps the browser editor viewport contract explicit", () => {
    const tsx = normalize(readUtf8(resolve(browserEditorDir, "BrowserDocumentEditorView.tsx")));
    const css = normalize(readUtf8(resolve(browserEditorDir, "BrowserDocumentEditorView.module.css")));

    expect(tsx).toContain('data-testid="browser-editor-viewport"');
    expect(css).toContain(".editorViewport");
    expect(css).toContain("isolation: isolate");
    expect(css).toContain("overflow: clip");
  });

  it("guards the bridge css against table handle chrome leaking back in", () => {
    const tsx = normalize(readUtf8(resolve(mddmEditorDir, "MDDMEditor.tsx")));
    const css = normalize(readUtf8(resolve(mddmEditorDir, "mddm-editor-global.css")));

    expect(tsx).toContain('data-mddm-editor-root="true"');
    expect(css).toContain('[data-mddm-editor-root="true"] .bn-table-handle');
    expect(css).toContain('[data-mddm-editor-root="true"] .bn-table-handle-menu');
    expect(css).toContain('[data-mddm-editor-root="true"] .bn-table-cell-handle');
  });
});
```

- [ ] **Step 2: Run the contract test and verify it fails**

Run: `cd frontend/apps/web; npm.cmd run test -- src/features/documents/mddm-editor/__tests__/editor-shell-contract.test.ts`  
Expected: FAIL because the global bridge CSS does not yet contain the root-scoped table-handle selectors

- [ ] **Step 3: Add the root-scoped guard selectors to the BlockNote bridge CSS**

```css
/* mddm-editor-global.css */

/* The local MDDM editor root is the only allowed containment boundary for
   BlockNote chrome on the browser editor surface. */
[data-mddm-editor-root="true"] .bn-container {
  position: relative;
  color: var(--mddm-raw-gray-700);
  font-family: var(--mddm-font-family);
}

/* FieldGroup-origin native tables are structural metadata, not user-created
   freeform tables. Keep row/column handle chrome suppressed even if a future
   BlockNote upgrade re-enables the controller. */
[data-mddm-editor-root="true"] .bn-table-handle,
[data-mddm-editor-root="true"] .bn-table-handle-menu,
[data-mddm-editor-root="true"] .bn-table-cell-handle {
  display: none !important;
}

/* Keep any remaining menus layered inside the isolated editor shell rather
   than competing with the workspace top bar. */
[data-mddm-editor-root="true"] .bn-formatting-toolbar,
[data-mddm-editor-root="true"] .bn-side-menu,
[data-mddm-editor-root="true"] .bn-drag-handle-menu {
  z-index: 2;
}
```

Keep the existing `.bn-extend-button` and `.bn-table-drop-cursor` suppression rules. They are still valid defense-in-depth for the same structural table surface.

- [ ] **Step 4: Run the contract test and the two editor suites together**

Run: `cd frontend/apps/web; npm.cmd run test -- src/features/documents/browser-editor/__tests__/BrowserDocumentEditorView.structure.test.tsx src/features/documents/mddm-editor/__tests__/MDDMEditor.test.tsx src/features/documents/mddm-editor/__tests__/editor-shell-contract.test.ts`  
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add frontend/apps/web/src/features/documents/mddm-editor/mddm-editor-global.css frontend/apps/web/src/features/documents/mddm-editor/__tests__/editor-shell-contract.test.ts
git commit -m "fix(frontend-editor): suppress structural table handle chrome"
```

---

## Final Verification

### Automated

Run: `cd frontend/apps/web; npm.cmd run test -- src/features/documents/browser-editor/__tests__/BrowserDocumentEditorView.structure.test.tsx src/features/documents/mddm-editor/__tests__/MDDMEditor.test.tsx src/features/documents/mddm-editor/__tests__/editor-shell-contract.test.ts`  
Expected: PASS

Run: `cd frontend/apps/web; npm.cmd run test -- src/features/documents/mddm-editor/__tests__/styling-contract.test.ts`  
Expected: PASS

### Manual Browser Validation

1. Start the web app with the usual local stack.
2. Open the browser document editor for the same document flow that reproduced the bug.
3. Scroll up and down repeatedly through the long document.
4. Click inside native field tables and hover near row/column edges where the white chrome used to appear.
5. Confirm:
   - no white BlockNote table chrome appears inside the global workspace top bar
   - no gray phantom area appears below the paper surface
   - the sticky formatting toolbar remains inside the editor desk band
   - the document header and white paper stay visually inside the `browser-editor-viewport`

### Console Diagnostics If The Bug Persists

Run this in DevTools before touching `DocumentWorkspaceShell` or `base.css`:

```js
(() => {
  const topbarBottom =
    document.querySelector('[class*="workspace-topbar"]')?.getBoundingClientRect().bottom ?? 0;

  return [...document.querySelectorAll("*")]
    .filter((node) => {
      const style = getComputedStyle(node);
      if (!["absolute", "fixed", "sticky"].includes(style.position)) return false;
      const rect = node.getBoundingClientRect();
      return rect.width > 0 && rect.height > 0 && rect.top < topbarBottom;
    })
    .map((node) => ({
      tag: node.tagName,
      testid: node.getAttribute("data-testid"),
      className: node.className,
      position: getComputedStyle(node).position,
      rect: node.getBoundingClientRect().toJSON(),
    }));
})();
```

Run this scroll-owner snapshot in the same failing state:

```js
(() => {
  const candidates = [
    document.scrollingElement,
    document.querySelector('[class*="workspace-main"]'),
    document.querySelector('[data-testid="browser-editor-viewport"]'),
    document.querySelector('[data-testid="mddm-editor-root"]'),
  ].filter(Boolean);

  return candidates.map((node) => ({
    node:
      node === document.scrollingElement
        ? "document.scrollingElement"
        : node.getAttribute?.("data-testid") || node.className,
    clientHeight: node.clientHeight,
    scrollHeight: node.scrollHeight,
    scrollTop: node.scrollTop,
    overflowY: getComputedStyle(node).overflowY,
  }));
})();
```

If either script shows escaped editor chrome above the workspace top bar after Tasks 1-3, stop and write a follow-up spec for `DocumentWorkspaceShell` instead of expanding this plan ad hoc.

---

## Self-Review Checklist

1. **Spec coverage**
   - Single visual shell owner: Task 1
   - Explicit editor viewport boundary: Task 1
   - Reduce `MDDMEditor` responsibility: Task 2
   - Suppress/contain floating structural table chrome: Tasks 2 and 3
   - Manual validation of scroll + table interactions: Final Verification
   - Out-of-scope guardrails preserved: `DocumentWorkspaceShell` and root viewport files remain untouched

2. **Placeholder scan**
   - No `TODO`, `TBD`, or “similar to previous task” shortcuts remain
   - Every task includes exact paths, code snippets, commands, and commit messages
   - Manual diagnostics include copy/paste-ready console scripts

3. **Type consistency**
   - DOM ids are consistent across tasks: `browser-editor-viewport`, `browser-editor-surface`, `browser-editor-footer`, `mddm-editor-root`, `mddm-editor-toolbar`, `mddm-editor-paper`
   - The containment attribute is consistent: `data-mddm-editor-root="true"`
   - Table chrome suppression is consistent across behavior and CSS: `tableHandles={false}` plus `.bn-table-handle*` guards
