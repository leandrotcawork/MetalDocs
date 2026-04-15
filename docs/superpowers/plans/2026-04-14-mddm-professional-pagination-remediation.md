# MDDM Professional Pagination Remediation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace current fake page-count overlay with a professional paged-authoring pipeline that only creates the next page when content actually overflows the writable area, while keeping page-two typing stable.

**Architecture:** Keep one canonical BlockNote/Tiptap document, but stop paginating from whole block boxes and stop treating page backgrounds as the source of truth. Instead, measure text and block fragments in document order, compute page breaks from fragment heights, and inject page-break spacer decorations at real document positions so content flow, caret movement, and visual pages stay aligned. Visual page gap remains, but it is presentation-only and never part of page-break math.

**Tech Stack:** React 18, TypeScript, BlockNote/Tiptap/ProseMirror, CSS Modules, Vitest, Playwright, Chrome.

---

## Root Cause Summary

Current implementation is not matching expected behavior for three concrete reasons:

1. **Pagination is block-box based, not line/fragment based.**
   - Current code measures whole block rects and computes pages from block bottoms, so it cannot wait until the real last writable line is crossed.
   - Evidence: [pagination.ts](/c:/Users/leandro.theodoro.MN-NTB-LEANDROT/Documents/MetalDocs/frontend/apps/web/src/features/documents/mddm-editor/pagination.ts:57) uses `block.topPx` / `block.heightPx` / `block.bottomPx` only.

2. **Pages are visual backgrounds only.**
   - Current code stretches one continuous `.bn-editor` over multiple page surfaces, so page-two typing is still one long DOM surface, not a truly paged flow.
   - Evidence: [MDDMEditor.tsx](/c:/Users/leandro.theodoro.MN-NTB-LEANDROT/Documents/MetalDocs/frontend/apps/web/src/features/documents/mddm-editor/MDDMEditor.tsx:519) renders many `mddm-editor-paper-surface` divs, but only one `BlockNoteViewEditor`.

3. **Visual page composition is coupled to layout approximation.**
   - Current page-count overlay and current break markers are derived from measured DOM after layout, rather than driving layout through explicit break positions.
   - Evidence: [MDDMEditor.tsx](/c:/Users/leandro.theodoro.MN-NTB-LEANDROT/Documents/MetalDocs/frontend/apps/web/src/features/documents/mddm-editor/MDDMEditor.tsx:193) measures DOM blocks, then [MDDMEditor.tsx](/c:/Users/leandro.theodoro.MN-NTB-LEANDROT/Documents/MetalDocs/frontend/apps/web/src/features/documents/mddm-editor/MDDMEditor.tsx:307) paints break offsets back onto already-laid-out content.

## Research Conclusions

Professional browser editors use one of three models:

1. **Pageless edit + paged preview**
   - Google Docs officially supports `Pages` and `Pageless` as separate document modes; page settings only apply in pages mode.
   - Source: Google Docs Help, “Change page settings on Google Docs” (<https://support.google.com/docs/answer/10296604?hl=en>).

2. **Decoupled pagination preview aligned to export**
   - CKEditor 5 pagination is a decoupled editor feature that shows where breaks will be after export; its configuration must exactly match export configuration and browser/content styling.
   - Source: CKEditor docs, “Pagination overview” (<https://ckeditor.com/docs/ckeditor5/latest/features/pagination/pagination.html>), especially config and troubleshooting sections.

3. **Custom layout/pagination pipeline on top of document model**
   - ProseMirror community experience is consistent: naïve rendering-time pagination in standard DOM editor view is not scalable, and serious page-oriented editing needs a custom layout pipeline.
   - Sources:
     - ProseMirror forum, “Implementing pagination with ProseMirror” (<https://discuss.prosemirror.net/t/implementing-pagination-with-prosemirror/6336>)
     - ProseMirror forum, “Building a Canvas-Based Editor on Top of ProseMirror’s State and Plugin System” (<https://discuss.prosemirror.net/t/building-a-canvas-based-editor-on-top-of-prosemirror-s-state-and-plugin-system/8982>)

## Chosen Approach

For MetalDocs, best fit is:

- **Keep one canonical BlockNote/ProseMirror document**
- **Replace block-box paginator with fragment-based paginator**
- **Inject real page-break spacers into content flow with ProseMirror decorations**
- **Keep visual page gap**
- **Do not split document into page nodes**
- **Do not keep current “background pages only” model**
- **Do not build a fully custom canvas editor**

This is the best tradeoff:

- More professional than current approximation
- Keeps current editor stack and extension model
- Supports exact “new page only after overflow” behavior
- Keeps page gap without corrupting content coordinates
- Avoids full custom runtime complexity and accessibility regression

---

## File Structure

**Create**
- `frontend/apps/web/src/features/documents/mddm-editor/page-fragments.ts`
- `frontend/apps/web/src/features/documents/mddm-editor/page-fragments-dom.ts`
- `frontend/apps/web/src/features/documents/mddm-editor/pagination-decorations.ts`
- `frontend/apps/web/src/features/documents/mddm-editor/__tests__/page-fragments.test.ts`
- `frontend/apps/web/src/features/documents/mddm-editor/__tests__/pagination-decorations.test.ts`

**Modify**
- `frontend/apps/web/src/features/documents/mddm-editor/pagination.ts`
- `frontend/apps/web/src/features/documents/mddm-editor/MDDMEditor.tsx`
- `frontend/apps/web/src/features/documents/mddm-editor/MDDMEditor.module.css`
- `frontend/apps/web/src/features/documents/mddm-editor/mddm-editor-global.css`
- `frontend/apps/web/src/features/documents/mddm-editor/__tests__/MDDMEditor.test.tsx`
- `frontend/apps/web/playwright/e2e/template-admin-editor.spec.ts`
- `tasks/lessons.md`

**Responsibilities**
- `page-fragments.ts`: pure types + fragment pagination algorithm
- `page-fragments-dom.ts`: DOM measurement from live editor into fragments with document positions
- `pagination-decorations.ts`: ProseMirror decoration widgets for page-break spacers and page markers
- `pagination.ts`: reduced to thin compatibility wrapper or deleted-by-replacement logic, depending on final diff safety
- `MDDMEditor.tsx`: lifecycle wiring, measurement scheduling, decoration plugin registration, page surface count
- `MDDMEditor.module.css`: visual page stack + page gap + spacer styling contract
- `mddm-editor-global.css`: editor-global selectors for break widgets/markers
- `template-admin-editor.spec.ts`: real Chrome regression coverage for exact overflow and page-two stability

---

### Task 1: Lock Browser Repro Into Failing Tests

**Files:**
- Modify: `frontend/apps/web/playwright/e2e/template-admin-editor.spec.ts`

- [ ] **Step 1: Write failing browser test for “page 2 only after real overflow”**

```ts
test("template editor keeps one page until content actually crosses writable height", async ({ page }) => {
  const templateKey = "tpl-pagination-overflow-threshold";
  const draft = makeDraft({
    templateKey,
    lockVersion: 1,
    blocks: [],
    meta: {
      page: {
        marginTopMm: 5,
        marginRightMm: 5,
        marginBottomMm: 5,
        marginLeftMm: 5,
      },
    },
  });

  await page.route("**/api/v1/templates/**", async (route) => {
    const url = new URL(route.request().url());
    const method = route.request().method();
    if (url.pathname === `/api/v1/templates/${templateKey}` && method === "GET") {
      await fulfillJson(route, 200, draft);
      return;
    }
    await route.continue();
  });

  await loginAsAdmin(page);
  await openTemplateEditor(page, "po", templateKey);
  await page.locator(".bn-editor").click();

  let switchAt = -1;
  for (let i = 1; i <= 240; i++) {
    await page.keyboard.type("x");
    await page.keyboard.press("Enter");

    const pageCount = await page.evaluate(() => {
      const paper = document.querySelector('[data-testid="mddm-editor-paper"]') as HTMLElement | null;
      return Number(paper?.dataset.pageCount ?? "1");
    });

    if (pageCount >= 2) {
      switchAt = i;
      break;
    }
  }

  const metrics = await page.evaluate(() => {
    const paper = document.querySelector('[data-testid="mddm-editor-paper"]') as HTMLElement | null;
    const editor = document.querySelector(".bn-editor") as HTMLElement | null;
    if (!paper || !editor) return null;

    const style = getComputedStyle(paper);
    const pageHeightPx = parseFloat(style.getPropertyValue("--mddm-page-height")) * 96 / 25.4;
    const topMarginPx = parseFloat(style.paddingTop);
    const bottomMarginPx = parseFloat(style.paddingBottom);
    const writableHeightPx = pageHeightPx - topMarginPx - bottomMarginPx;

    return {
      pageCount: Number(paper.dataset.pageCount ?? "1"),
      scrollHeight: editor.scrollHeight,
      writableHeightPx,
    };
  });

  expect(switchAt).toBeGreaterThan(0);
  expect(metrics).not.toBeNull();
  expect(metrics?.pageCount).toBe(2);
  expect((metrics?.scrollHeight ?? 0) - (metrics?.writableHeightPx ?? 0)).toBeGreaterThanOrEqual(0);
  expect((metrics?.scrollHeight ?? 0) - (metrics?.writableHeightPx ?? 0)).toBeLessThan(32);
});
```

- [ ] **Step 2: Write failing browser test for “page two typing surface stays stable”**

```ts
test("template editor keeps page-two writing surface aligned after automatic break", async ({ page }) => {
  const templateKey = "tpl-pagination-page-two-stability";
  const draft = makeDraft({
    templateKey,
    lockVersion: 1,
    blocks: [],
    meta: {
      page: {
        marginTopMm: 5,
        marginRightMm: 5,
        marginBottomMm: 5,
        marginLeftMm: 5,
      },
    },
  });

  await page.route("**/api/v1/templates/**", async (route) => {
    const url = new URL(route.request().url());
    const method = route.request().method();
    if (url.pathname === `/api/v1/templates/${templateKey}` && method === "GET") {
      await fulfillJson(route, 200, draft);
      return;
    }
    await route.continue();
  });

  await loginAsAdmin(page);
  await openTemplateEditor(page, "po", templateKey);
  await page.locator(".bn-editor").click();

  while (true) {
    await page.keyboard.type("x");
    await page.keyboard.press("Enter");

    const pageCount = await page.evaluate(() => {
      const paper = document.querySelector('[data-testid="mddm-editor-paper"]') as HTMLElement | null;
      return Number(paper?.dataset.pageCount ?? "1");
    });
    if (pageCount >= 2) break;
  }

  for (let i = 0; i < 10; i++) {
    await page.keyboard.type("x");
    await page.keyboard.press("Enter");
  }

  const metrics = await page.evaluate(() => {
    const surfaces = Array.from(document.querySelectorAll<HTMLElement>('[data-testid="mddm-editor-paper-surface"]'));
    const lastBlock = Array.from(document.querySelectorAll<HTMLElement>(".bn-editor .bn-block")).at(-1);
    if (surfaces.length < 2 || !lastBlock) return null;

    const secondSurface = surfaces[1].getBoundingClientRect();
    const lastRect = lastBlock.getBoundingClientRect();
    return {
      pageGapPx: secondSurface.top - surfaces[0].getBoundingClientRect().bottom,
      lastBlockTopVsSecondPage: lastRect.top - secondSurface.top,
      lastBlockBottomVsSecondPage: lastRect.bottom - secondSurface.bottom,
    };
  });

  expect(metrics).not.toBeNull();
  expect(metrics?.pageGapPx ?? 0).toBeGreaterThan(8);
  expect(metrics?.lastBlockTopVsSecondPage ?? -1).toBeGreaterThanOrEqual(0);
  expect(metrics?.lastBlockBottomVsSecondPage ?? 1).toBeLessThanOrEqual(0);
});
```

- [ ] **Step 3: Run browser tests to verify they fail on current implementation**

Run: `cd frontend/apps/web; .\\node_modules\\.bin\\playwright.cmd test tests/e2e/template-admin-editor.spec.ts --project=chrome --grep "overflow-threshold|page-two-stability"`

Expected: FAIL because current block-box overlay cannot hold exact overflow threshold or stable page-two writing bounds.

- [ ] **Step 4: Commit failing spec guard**

```bash
git add frontend/apps/web/playwright/e2e/template-admin-editor.spec.ts
git commit -m "test(mddm-editor): lock real paged authoring regressions"
```

---

### Task 2: Add Pure Fragment Pagination Model

**Files:**
- Create: `frontend/apps/web/src/features/documents/mddm-editor/page-fragments.ts`
- Create: `frontend/apps/web/src/features/documents/mddm-editor/__tests__/page-fragments.test.ts`
- Modify: `frontend/apps/web/src/features/documents/mddm-editor/pagination.ts`

- [ ] **Step 1: Write failing unit tests for fragment pagination**

```ts
import { describe, expect, it } from "vitest";
import { paginateMeasuredFragments } from "../page-fragments";

describe("paginateMeasuredFragments", () => {
  it("does not create next page before first overflowing line", () => {
    const result = paginateMeasuredFragments({
      pageHeightPx: 1000,
      topMarginPx: 50,
      bottomMarginPx: 50,
      pageGapPx: 20,
      fragments: [
        { key: "l1", pos: 1, heightPx: 300, kind: "line" },
        { key: "l2", pos: 2, heightPx: 300, kind: "line" },
        { key: "l3", pos: 3, heightPx: 299, kind: "line" },
      ],
    });

    expect(result.pageCount).toBe(1);
    expect(result.breaks).toEqual([]);
  });

  it("creates next page exactly before first overflowing fragment", () => {
    const result = paginateMeasuredFragments({
      pageHeightPx: 1000,
      topMarginPx: 50,
      bottomMarginPx: 50,
      pageGapPx: 20,
      fragments: [
        { key: "l1", pos: 1, heightPx: 300, kind: "line" },
        { key: "l2", pos: 2, heightPx: 300, kind: "line" },
        { key: "l3", pos: 3, heightPx: 300, kind: "line" },
      ],
    });

    expect(result.pageCount).toBe(2);
    expect(result.breaks).toEqual([
      { pos: 3, pageIndex: 1, spacerBeforePx: 120 },
    ]);
  });

  it("moves atomic block wholly to next page", () => {
    const result = paginateMeasuredFragments({
      pageHeightPx: 1000,
      topMarginPx: 50,
      bottomMarginPx: 50,
      pageGapPx: 20,
      fragments: [
        { key: "l1", pos: 1, heightPx: 780, kind: "line" },
        { key: "tbl", pos: 5, heightPx: 200, kind: "atomic" },
      ],
    });

    expect(result.breaks[0]?.pos).toBe(5);
    expect(result.pageCount).toBe(2);
  });
});
```

- [ ] **Step 2: Run unit test to verify it fails**

Run: `npm.cmd --prefix frontend/apps/web test -- --run src/features/documents/mddm-editor/__tests__/page-fragments.test.ts`

Expected: FAIL because `page-fragments.ts` does not exist yet.

- [ ] **Step 3: Write minimal pure fragment model**

```ts
export type MeasuredFragment = {
  key: string;
  pos: number;
  heightPx: number;
  kind: "line" | "atomic";
  keepWithNext?: boolean;
};

export type PaginationBreak = {
  pos: number;
  pageIndex: number;
  spacerBeforePx: number;
};

export type PaginatedFragments = {
  pageCount: number;
  breaks: PaginationBreak[];
};

export function paginateMeasuredFragments(input: {
  pageHeightPx: number;
  topMarginPx: number;
  bottomMarginPx: number;
  pageGapPx: number;
  fragments: ReadonlyArray<MeasuredFragment>;
}): PaginatedFragments {
  const writableHeightPx = input.pageHeightPx - input.topMarginPx - input.bottomMarginPx;
  let cursorPx = 0;
  let pageIndex = 0;
  const breaks: PaginationBreak[] = [];

  for (const fragment of input.fragments) {
    if (fragment.heightPx <= 0) continue;

    const overflows = cursorPx + fragment.heightPx > writableHeightPx;
    if (!overflows) {
      cursorPx += fragment.heightPx;
      continue;
    }

    pageIndex += 1;
    breaks.push({
      pos: fragment.pos,
      pageIndex,
      spacerBeforePx: input.pageGapPx + input.topMarginPx,
    });
    cursorPx = fragment.heightPx;
  }

  return {
    pageCount: Math.max(1, pageIndex + 1),
    breaks,
  };
}
```

- [ ] **Step 4: Make old `pagination.ts` a compatibility wrapper**

```ts
import { paginateMeasuredFragments } from "./page-fragments";

export function computePageLayout(input: ComputePageLayoutInput): PageLayout {
  const fragments = input.blocks
    .sort((a, b) => a.topPx - b.topPx)
    .map((block) => ({
      key: block.id,
      pos: 0,
      heightPx: block.heightPx,
      kind: "atomic" as const,
    }));

  const result = paginateMeasuredFragments({
    pageHeightPx: input.pageHeightPx,
    topMarginPx: input.topMarginPx,
    bottomMarginPx: input.bottomMarginPx,
    pageGapPx: 0,
    fragments,
  });

  return {
    pageCount: result.pageCount,
    breakOffsetsByBlockId: {},
  };
}
```

- [ ] **Step 5: Run unit test to verify it passes**

Run: `npm.cmd --prefix frontend/apps/web test -- --run src/features/documents/mddm-editor/__tests__/page-fragments.test.ts src/features/documents/mddm-editor/__tests__/pagination.test.ts`

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add frontend/apps/web/src/features/documents/mddm-editor/page-fragments.ts frontend/apps/web/src/features/documents/mddm-editor/__tests__/page-fragments.test.ts frontend/apps/web/src/features/documents/mddm-editor/pagination.ts
git commit -m "feat(mddm-editor): add fragment pagination model"
```

---

### Task 3: Measure Real DOM Fragments With ProseMirror Positions

**Files:**
- Create: `frontend/apps/web/src/features/documents/mddm-editor/page-fragments-dom.ts`
- Modify: `frontend/apps/web/src/features/documents/mddm-editor/MDDMEditor.tsx`

- [ ] **Step 1: Add failing integration assertion in editor test**

```ts
it("publishes measured fragments with document positions for pagination", async () => {
  const measured = vi.fn();
  render(
    <MDDMEditor
      initialContent={[]}
      onMeasuredFragments={measured}
    />,
  );

  await waitFor(() => expect(measured).toHaveBeenCalled());
  expect(measured.mock.calls.at(-1)?.[0]).toEqual(
    expect.arrayContaining([
      expect.objectContaining({
        pos: expect.any(Number),
        heightPx: expect.any(Number),
      }),
    ]),
  );
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `npm.cmd --prefix frontend/apps/web test -- --run src/features/documents/mddm-editor/__tests__/MDDMEditor.test.tsx`

Expected: FAIL because fragment measurement callback/measurer does not exist.

- [ ] **Step 3: Add DOM fragment measurer**

```ts
import type { EditorView } from "@tiptap/pm/view";
import type { MeasuredFragment } from "./page-fragments";

export function measureFragmentsFromEditorDom(args: {
  view: EditorView;
  paperElement: HTMLElement;
  pageTopPaddingPx: number;
}): MeasuredFragment[] {
  const { view, paperElement, pageTopPaddingPx } = args;
  const editorDom = view.dom as HTMLElement;
  const paperTopPx = paperElement.getBoundingClientRect().top + pageTopPaddingPx;
  const fragments: MeasuredFragment[] = [];

  editorDom.querySelectorAll<HTMLElement>(".bn-block").forEach((blockEl) => {
    const blockPos = view.posAtDOM(blockEl, 0);
    const textNode = blockEl.querySelector("[data-content-type], .bn-inline-content");

    if (!textNode) {
      const rect = blockEl.getBoundingClientRect();
      fragments.push({
        key: `atomic:${blockPos}`,
        pos: blockPos,
        heightPx: rect.height,
        kind: "atomic",
      });
      return;
    }

    const range = document.createRange();
    range.selectNodeContents(textNode);
    const lineRects = Array.from(range.getClientRects())
      .filter((rect) => rect.height > 0)
      .map((rect, index) => ({
        key: `line:${blockPos}:${index}`,
        pos: blockPos,
        heightPx: rect.height,
        kind: "line" as const,
      }));

    fragments.push(...lineRects);
  });

  return fragments;
}
```

- [ ] **Step 4: Wire measurer into editor lifecycle**

```ts
const measuredFragments = measureFragmentsFromEditorDom({
  view: tiptapView,
  paperElement,
  pageTopPaddingPx: topMarginPx,
});
```

- [ ] **Step 5: Run integration test to verify it passes**

Run: `npm.cmd --prefix frontend/apps/web test -- --run src/features/documents/mddm-editor/__tests__/MDDMEditor.test.tsx`

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add frontend/apps/web/src/features/documents/mddm-editor/page-fragments-dom.ts frontend/apps/web/src/features/documents/mddm-editor/MDDMEditor.tsx frontend/apps/web/src/features/documents/mddm-editor/__tests__/MDDMEditor.test.tsx
git commit -m "feat(mddm-editor): measure editor fragments with positions"
```

---

### Task 4: Inject Real Page-Break Spacers With Decorations

**Files:**
- Create: `frontend/apps/web/src/features/documents/mddm-editor/pagination-decorations.ts`
- Create: `frontend/apps/web/src/features/documents/mddm-editor/__tests__/pagination-decorations.test.ts`
- Modify: `frontend/apps/web/src/features/documents/mddm-editor/MDDMEditor.tsx`
- Modify: `frontend/apps/web/src/features/documents/mddm-editor/mddm-editor-global.css`

- [ ] **Step 1: Write failing unit test for decoration widgets**

```ts
import { describe, expect, it } from "vitest";
import { createPaginationDecorations } from "../pagination-decorations";

describe("createPaginationDecorations", () => {
  it("creates one spacer widget per computed page break", () => {
    const decorations = createPaginationDecorations({
      doc: schema.node("doc", null, [schema.node("paragraph")]),
      breaks: [
        { pos: 1, pageIndex: 1, spacerBeforePx: 120 },
      ],
    });

    expect(decorations.find()).toHaveLength(1);
  });
});
```

- [ ] **Step 2: Run unit test to verify it fails**

Run: `npm.cmd --prefix frontend/apps/web test -- --run src/features/documents/mddm-editor/__tests__/pagination-decorations.test.ts`

Expected: FAIL because decoration helper does not exist yet.

- [ ] **Step 3: Create pagination decoration helper**

```ts
import { Decoration, DecorationSet } from "@tiptap/pm/view";
import type { Node as PMNode } from "@tiptap/pm/model";
import type { PaginationBreak } from "./page-fragments";

export function createPaginationDecorations(input: {
  doc: PMNode;
  breaks: ReadonlyArray<PaginationBreak>;
}): DecorationSet {
  const decorations = input.breaks.map((pageBreak) =>
    Decoration.widget(pageBreak.pos, () => {
      const spacer = document.createElement("div");
      spacer.dataset.mddmPageSpacer = "true";
      spacer.dataset.mddmPageIndex = String(pageBreak.pageIndex);
      spacer.style.height = `${pageBreak.spacerBeforePx}px`;
      return spacer;
    }, { side: -1 }),
  );

  return DecorationSet.create(input.doc, decorations);
}
```

- [ ] **Step 4: Register decoration plugin in `MDDMEditor`**

```ts
const paginationPlugin = new Plugin({
  key: new PluginKey("mddm-pagination-spacers"),
  props: {
    decorations(state) {
      return createPaginationDecorations({
        doc: state.doc,
        breaks: pageLayout.breaks,
      });
    },
  },
});
```

- [ ] **Step 5: Style spacers without affecting semantics**

```css
.bn-container [data-mddm-page-spacer="true"] {
  display: block;
  width: 100%;
  pointer-events: none;
}
```

- [ ] **Step 6: Run tests to verify they pass**

Run: `npm.cmd --prefix frontend/apps/web test -- --run src/features/documents/mddm-editor/__tests__/pagination-decorations.test.ts src/features/documents/mddm-editor/__tests__/MDDMEditor.test.tsx`

Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add frontend/apps/web/src/features/documents/mddm-editor/pagination-decorations.ts frontend/apps/web/src/features/documents/mddm-editor/__tests__/pagination-decorations.test.ts frontend/apps/web/src/features/documents/mddm-editor/MDDMEditor.tsx frontend/apps/web/src/features/documents/mddm-editor/mddm-editor-global.css
git commit -m "feat(mddm-editor): insert real page-break spacers"
```

---

### Task 5: Restore Visual Page Gap As Presentation-Only

**Files:**
- Modify: `frontend/apps/web/src/features/documents/mddm-editor/MDDMEditor.tsx`
- Modify: `frontend/apps/web/src/features/documents/mddm-editor/MDDMEditor.module.css`
- Modify: `frontend/apps/web/playwright/e2e/template-admin-editor.spec.ts`

- [ ] **Step 1: Update editor page stack to derive count from break decisions**

```ts
const pageGapPx = 20;
const visualPageCount = Math.max(1, pageLayout.pageCount);
```

```tsx
<div className={styles.surfaceStack} style={{ ["--mddm-page-gap-px" as any]: `${pageGapPx}px` }}>
  {Array.from({ length: visualPageCount }).map((_, pageIndex) => (
    <div
      key={`surface-${pageIndex}`}
      className={styles.paperSurface}
      data-testid="mddm-editor-paper-surface"
    />
  ))}
</div>
```

- [ ] **Step 2: Restore visual page gap in CSS**

```css
.surfaceStack {
  display: grid;
  justify-items: center;
  gap: var(--mddm-page-gap-px, 20px);
  width: 100%;
  pointer-events: none;
}
```

- [ ] **Step 3: Remove obsolete block-level break painting**

```ts
// Delete old code that writes data-mddm-page-break onto block wrappers
// because real break positions now come from decoration widgets.
```

- [ ] **Step 4: Run browser tests to verify page gap stays visible but threshold remains correct**

Run: `cd frontend/apps/web; .\\node_modules\\.bin\\playwright.cmd test tests/e2e/template-admin-editor.spec.ts --project=chrome --grep "overflow-threshold|page-two-stability|keeps scroll on the document pane"`

Expected: PASS, with page gap still visible and no early page creation.

- [ ] **Step 5: Commit**

```bash
git add frontend/apps/web/src/features/documents/mddm-editor/MDDMEditor.tsx frontend/apps/web/src/features/documents/mddm-editor/MDDMEditor.module.css frontend/apps/web/playwright/e2e/template-admin-editor.spec.ts
git commit -m "feat(mddm-editor): separate page gap from page-break math"
```

---

### Task 6: Final Verification And Lessons

**Files:**
- Modify: `tasks/lessons.md`

- [ ] **Step 1: Write correction lesson**

```md
## Lesson N - Paged editor visuals must be driven by real break positions
Date: 2026-04-14 | Trigger: correction
Wrong:   MDDM pagination derived page count from block bounding boxes and painted page backgrounds behind one continuous editor surface.
Correct: MDDM pagination now computes fragment-level page breaks and injects spacer decorations at real document positions while keeping visual page gap presentation-only.
Rule:    Word-like browser pagination requires break positions inside content flow, not page backgrounds layered behind a continuous editor DOM.
Layer:   delivery
```

- [ ] **Step 2: Run full targeted verification**

Run: `npm.cmd --prefix frontend/apps/web test -- --run src/features/documents/mddm-editor/__tests__/page-fragments.test.ts src/features/documents/mddm-editor/__tests__/pagination-decorations.test.ts src/features/documents/mddm-editor/__tests__/pagination.test.ts src/features/documents/mddm-editor/__tests__/MDDMEditor.test.tsx src/features/templates/__tests__/PropertySidebar.test.tsx`

Expected: PASS.

- [ ] **Step 3: Run Chrome browser verification**

Run: `cd frontend/apps/web; .\\node_modules\\.bin\\playwright.cmd test tests/e2e/template-admin-editor.spec.ts --project=chrome --grep "overflow-threshold|page-two-stability|page margin inputs commit on blur|preserves page margins after save and reload|keeps scroll on the document pane" --workers=1`

Expected: PASS.

- [ ] **Step 4: Capture visual evidence**

Run one-off Chrome script or Playwright screenshot flow and save:
- `frontend/apps/web/tmp/visual-checks/pagination-overflow-threshold.png`
- `frontend/apps/web/tmp/visual-checks/pagination-page-two-stability.png`

Expected:
- page gap visible
- page 2 only appears after true overflow
- page-two typing remains inside second page bounds

- [ ] **Step 5: Run build**

Run: `npm.cmd --prefix frontend/apps/web run build`

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add tasks/lessons.md frontend/apps/web/playwright/e2e/template-admin-editor.spec.ts frontend/apps/web/tmp/visual-checks
git commit -m "fix(mddm-editor): make paged authoring break at real overflow"
```

---

## Self-Review

**1. Spec coverage**
- Root cause investigation included: yes
- Professional approach comparison included: yes
- Best-fit approach selected for MetalDocs: yes
- Full implementation plan with exact files/tests/commands: yes
- Real-browser validation loop included: yes

**2. Placeholder scan**
- No `TODO` / `TBD`
- Every task has exact files
- Every task has exact commands
- Code shown for tests and implementation steps

**3. Type consistency**
- Core planned types:
  - `MeasuredFragment`
  - `PaginationBreak`
  - `PaginatedFragments`
  - `paginateMeasuredFragments`
  - `measureFragmentsFromEditorDom`
  - `createPaginationDecorations`
- Names remain consistent across tasks

## Execution Handoff

Plan complete and saved to `docs/superpowers/plans/2026-04-14-mddm-professional-pagination-remediation.md`. Two execution options:

**1. Subagent-Driven (recommended)** - I dispatch a fresh subagent per task, review between tasks, fast iteration

**2. Inline Execution** - Execute tasks in this session using executing-plans, batch execution with checkpoints

**Which approach?**
