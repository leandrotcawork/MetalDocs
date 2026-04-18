# CK5 Pagination v3 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use `nexus:subagent-driven-development` (recommended) or `nexus:executing-plans` to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace broken overlay-frame pagination in CK5 author editor with single-paper CSS-gradient architecture + predictive page count + preventive break measurer.

**Architecture:** Editable gets `repeating-linear-gradient` background (white 297mm + gray 32px cycle). React hook `usePredictivePageCount` owns `--mddm-pages` via ResizeObserver. Footers portalled into editor root as absolute overlay. BreakMeasurer uses fixed-stride prediction + preventive break semantics + dynamic `spacerPx` inline margin-bottom.

**Tech Stack:** CKEditor 5 v48 DecoupledEditor, React 18, TypeScript, Vite, Vitest + jsdom.

**Spec:** `docs/superpowers/specs/2026-04-18-ck5-pagination-v3-design.md`.

**Branch:** current HEAD on `main` (v2 dirty tree in working copy — revert before starting or stash).

---

## Orchestration — Codex executes, Opus reviews

Codex (`gpt-5.3-codex`, reasoning=high) writes every task. Opus reviews between tasks via `git diff` (visual scan + line-count budget per task) and runs the final Vitest suite + validates preview manually. The `nexus:code-reviewer` subagent is optional — if available in the environment, use it for Task 5 and Task 1 review; otherwise rely on diff review.

| Task | Executor | Reviewer | Dependency |
|------|----------|----------|------------|
| 0 — Clean slate (stash v2 dirty tree) | Opus | — | none |
| 1 — BreakMeasurer preventive-break rewrite | Codex | Opus | 0 |
| 2 — `usePredictivePageCount` hook | Codex | Opus | 0 |
| 3 — `PageFooters` component | Codex | Opus | 0 |
| 4 — CSS gradient + min-height formula | Codex | Opus | 0 |
| 5 — AuthorEditor wire-up | Codex | Opus | 1,2,3,4 |
| 6 — Final Vitest + preview validation | Opus | — | 5 |

Tasks 1–4 can run in parallel (different files, no shared surface). Tasks 5 and 6 strictly sequential.

---

## File Structure

### Modified
- `frontend/apps/web/src/features/documents/ck5/plugins/MddmPaginationPlugin/BreakMeasurer.ts` — preventive break, hard-coded `page1TopY`, 50ms debounce, initial-sync flag, always measure from position [0].
- `frontend/apps/web/src/features/documents/ck5/react/AuthorEditor.tsx` — add `PageFooters` overlay + `usePredictivePageCount`, capture `editorRoot` after onReady. Keep `<PageCounter editor={editor} />` API unchanged.
- `frontend/apps/web/src/features/documents/ck5/react/AuthorEditor.module.css` — add repeating-gradient background, update editable `min-height` to stride formula, add `position: relative` to `.editable`, delete obsolete `[data-mddm-page-break-after]::before/::after` + `.mddm-page-overlay` rules.

### Added
- `frontend/apps/web/src/features/documents/ck5/react/usePredictivePageCount.ts`
- `frontend/apps/web/src/features/documents/ck5/react/__tests__/usePredictivePageCount.test.tsx`
- `frontend/apps/web/src/features/documents/ck5/react/PageFooters.tsx`
- `frontend/apps/web/src/features/documents/ck5/react/__tests__/PageFooters.test.tsx`

### Deleted
- None at plan level. (The v2 dirty tree introduced a `PageFrames.tsx` in the working copy; Task 0 stash removes it from the working tree. It was never committed to `main`, so nothing to delete after stash.)

### Unchanged (verify no regression)
- `frontend/apps/web/src/features/documents/ck5/plugins/MddmPaginationPlugin/PageOverlayView.ts` — already applies `style.marginBottom = spacerPx` inline.
- `shared/mddm-pagination-types/index.ts` — `ComputedBreak.spacerPx` stays.

---

## Task 0: Clean slate (Opus, manual)

**Goal:** Discard v2 dirty working tree so v3 starts from known-good `main`.

**Files:** working tree only.

- [ ] **Step 1: Check current git state**

```bash
cd "C:/Users/leandro.theodoro.MN-NTB-LEANDROT/Documents/MetalDocs"
git status --short
```

Expected: list of modified `BreakMeasurer.ts`, `PageOverlayView.ts`, CSS, AuthorEditor, tests, new `PageFrames.tsx`.

- [ ] **Step 2: Stash v2 dirty tree**

```bash
git stash push -u -m "v2-pagination-broken-for-reference" -- frontend/apps/web/src/features/documents/ck5 shared/mddm-pagination-types
```

Expected: `Saved working directory and index state On main: v2-pagination-broken-for-reference`.

- [ ] **Step 3: Verify clean**

```bash
git status --short
```

Expected: empty, or only `.superpowers/` untracked.

- [ ] **Step 4: Confirm baseline tests green**

```bash
cd "C:/Users/leandro.theodoro.MN-NTB-LEANDROT/Documents/MetalDocs/frontend/apps/web" && "/c/Program Files/nodejs/npx.cmd" vitest run src/features/documents/ck5/plugins/MddmPaginationPlugin src/features/documents/ck5/react --reporter=dot
```

Expected: all tests pass (pre-v2 counts). Note the baseline number.

---

## Task 1: BreakMeasurer preventive-break rewrite (Codex)

**Goal:** Rewrite `BreakMeasurer.measure()` with fixed-stride prediction, preventive break, hard-coded `page1TopY = MM(25)`, 50 ms debounce, initial-sync flag.

**Files:**
- Modify: `frontend/apps/web/src/features/documents/ck5/plugins/MddmPaginationPlugin/BreakMeasurer.ts`
- Test: `frontend/apps/web/src/features/documents/ck5/plugins/MddmPaginationPlugin/__tests__/BreakMeasurer.test.ts`

- [ ] **Step 1: Write failing test — preventive break fires on current candidate when it would overflow**

Append to `__tests__/BreakMeasurer.test.ts` (inside the existing `describe('BreakMeasurer', () => { ... })` block, after the two existing tests). The test is fully self-contained — it does NOT rely on any helpers; it patches `offsetTop` / `offsetHeight` on the rendered DOM elements directly:

```ts
  it('emits preventive break when next candidate would overflow current page', async () => {
    editor.setData(
      '<p data-mddm-bid="a">A</p><p data-mddm-bid="b">B</p><p data-mddm-bid="c">C</p>',
    );

    // Patch rendered DOM offsets. Page 1 cap = MM(25) + MM(247) = 1028.03px.
    // A at y=94.49 (= MM(25)), 100px tall → A.bot = 194.49 (fits).
    // B at y=194.49, 900px tall → B.bot = 1094.49 (overflows → trigger preventive break on A).
    // C at y=1094.49, 50px tall.
    const root = editor.editing.view.getDomRoot() as HTMLElement;
    const fakes: Array<[string, number, number]> = [
      ['a', 94.49, 100],
      ['b', 194.49, 900],
      ['c', 1094.49, 50],
    ];
    for (const [bid, top, height] of fakes) {
      const el = root.querySelector(`[data-mddm-bid="${bid}"]`) as HTMLElement | null;
      if (!el) throw new Error(`missing bid ${bid}`);
      Object.defineProperty(el, 'offsetTop', { configurable: true, get: () => top });
      Object.defineProperty(el, 'offsetHeight', { configurable: true, get: () => height });
    }

    const tracker = new DirtyRangeTracker(editor);
    const measurer = new BreakMeasurer(editor, tracker, { debounceMs: 10 });
    const emitted: Array<{ afterBid: string; pageNumber: number; spacerPx: number }> = [];
    measurer.onBreaks(batch => emitted.push(...batch));

    // Trigger a render cycle: insert a no-op text change that forces view.render.
    editor.model.change(writer => {
      const r = editor.model.document.getRoot()!;
      writer.insertText(' ', r.getChild(0) as never, 'end');
    });
    await new Promise(r => setTimeout(r, 40));

    // Expect preventive break on 'a' (not on 'b', because B would land in overflow zone).
    expect(emitted.length).toBeGreaterThan(0);
    expect(emitted[emitted.length - 1]).toMatchObject({ afterBid: 'a', pageNumber: 2 });
    // spacerPx ≈ targetNextTop - A.bot = (MM(25) + stride) - 194.49.
    // stride = MM(297) + 32 = 1122.52 + 32 = 1154.52. targetNextTop = 94.49 + 1154.52 = 1249.01.
    // spacerPx ≈ 1249.01 - 194.49 = 1054.52.
    expect(emitted[emitted.length - 1].spacerPx).toBeCloseTo(1054.52, 0);

    measurer.destroy();
    tracker.destroy();
  });
```

Note: the test matches the existing file style — no external helpers, direct editor + DOM patching.

- [ ] **Step 2: Run test to verify it fails**

```bash
cd frontend/apps/web && "/c/Program Files/nodejs/npx.cmd" vitest run src/features/documents/ck5/plugins/MddmPaginationPlugin/__tests__/BreakMeasurer.test.ts
```

Expected: the new test fails — current measurer emits break at `B` not `A`, or spacer wrong.

- [ ] **Step 3: Rewrite `BreakMeasurer.ts` `measure()` method**

Replace the entire class body with:

```ts
import type { Editor } from 'ckeditor5';
import { defaultLayoutTokens } from '@metaldocs/mddm-layout-tokens';
import { planBreaks } from './BreakPlanner';
import type { DirtyRangeTracker } from './DirtyRangeTracker';
import type { ComputedBreak } from './types';

const MM_PER_INCH = 25.4;
const PX_PER_INCH = 96;
const mmToPx = (mm: number) => (mm / MM_PER_INCH) * PX_PER_INCH;
const PAGE_HEIGHT_PX = mmToPx(defaultLayoutTokens.page.heightMm);
const PAGE_GAP_PX = 32;
const STRIDE_PX = PAGE_HEIGHT_PX + PAGE_GAP_PX;
const MARGIN_TOP_PX = mmToPx(defaultLayoutTokens.page.marginTopMm);

type Listener = (breaks: ComputedBreak[]) => void;

export class BreakMeasurer {
  private readonly listeners = new Set<Listener>();
  private timer: ReturnType<typeof setTimeout> | null = null;
  private readonly renderHandler: () => void;
  private fontsReady = false;
  private initialSyncDone = false;

  public constructor(
    private readonly editor: Editor,
    private readonly tracker: DirtyRangeTracker,
    private readonly opts: { debounceMs: number } = { debounceMs: 50 },
  ) {
    this.renderHandler = () => this.schedule();
    (this.editor.editing.view as unknown as { on: (n: string, f: () => void) => void }).on(
      'render',
      this.renderHandler,
    );
  }

  public destroy(): void {
    (this.editor.editing.view as unknown as { off: (n: string, f: () => void) => void }).off(
      'render',
      this.renderHandler,
    );
    if (this.timer !== null) {
      clearTimeout(this.timer);
      this.timer = null;
    }
    this.listeners.clear();
  }

  public onBreaks(fn: Listener): () => void {
    this.listeners.add(fn);
    return () => this.listeners.delete(fn);
  }

  private schedule(): void {
    // Initial render: skip debounce for fast first paint.
    if (!this.initialSyncDone) {
      this.initialSyncDone = true;
      void this.measure();
      return;
    }
    if (this.timer !== null) clearTimeout(this.timer);
    this.timer = setTimeout(() => {
      this.timer = null;
      void this.measure();
    }, this.opts.debounceMs);
  }

  private async measure(): Promise<void> {
    if (!this.fontsReady) {
      try {
        if (typeof document !== 'undefined' && document.fonts?.ready) {
          await document.fonts.ready;
        }
      } catch {
        /* jsdom — no fonts API */
      }
      this.fontsReady = true;
    }

    const root = this.editor.model.document.getRoot();
    if (!root) return;

    // Always measure the full document from position [0]. Correctness over
    // incremental-perf: `currentPage = 1` seed is only valid when we start at
    // document top. Starting from a dirty range mid-doc would mis-seed the
    // page counter and miss breaks when the first dirty candidate overflows
    // (prevAfterBid null at loop start). Debounce (50 ms) + ceiling of O(blocks)
    // keeps this bounded; `tracker` is kept for future incremental reuse.
    const from = this.editor.model.createPositionFromPath(root, [0]);
    void this.tracker.snapshot(); // intentional no-op read to keep tracker hot
    const candidates = planBreaks(this.editor, from, undefined);

    const dpr = (typeof window !== 'undefined' && window.devicePixelRatio) || 1;
    const contentHeightPx = mmToPx(
      defaultLayoutTokens.page.heightMm -
        defaultLayoutTokens.page.marginTopMm -
        defaultLayoutTokens.page.marginBottomMm,
    );

    // Hard-coded page1TopY — editor padding-top = MARGIN_TOP_PX. Does NOT depend on first candidate.
    const page1TopY = MARGIN_TOP_PX;

    const breaks: ComputedBreak[] = [];
    let currentPage = 1;
    let prevAfterBid: string | null = null;
    let prevCandidateBot = 0;

    for (const c of candidates) {
      const pos = this.editor.model.createPositionFromPath(root, c.modelPath);
      const modelEl = pos.nodeBefore;
      if (!modelEl || !modelEl.is('element')) continue;

      const viewEl = this.editor.editing.mapper.toViewElement(modelEl);
      if (!viewEl) continue;
      const domEl = this.editor.editing.view.domConverter.mapViewToDom(viewEl) as
        | HTMLElement
        | undefined;
      if (!domEl) continue;

      const imgs = domEl.querySelectorAll('img');
      for (const img of Array.from(imgs)) {
        try {
          await (img as HTMLImageElement).decode();
        } catch {
          console.warn('mddm:pagination-measure-skip', c.afterBid);
        }
      }

      const top = domEl.offsetTop;
      const bot = Math.round((top + domEl.offsetHeight) * dpr) / dpr;

      const predictedPageTop = page1TopY + (currentPage - 1) * STRIDE_PX;
      const predictedPageCap = predictedPageTop + contentHeightPx;

      // Preventive break: if current candidate bot would overflow this page, break on prev.
      if (bot > predictedPageCap && prevAfterBid !== null) {
        currentPage += 1;
        const targetNextTop = page1TopY + (currentPage - 1) * STRIDE_PX;
        const spacerPx = Math.max(0, targetNextTop - prevCandidateBot);
        breaks.push({
          afterBid: prevAfterBid,
          pageNumber: currentPage,
          yPx: prevCandidateBot,
          spacerPx,
        });
      }

      prevAfterBid = c.afterBid;
      prevCandidateBot = bot;
    }

    for (const l of this.listeners) l(breaks);
  }
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
cd frontend/apps/web && "/c/Program Files/nodejs/npx.cmd" vitest run src/features/documents/ck5/plugins/MddmPaginationPlugin/__tests__/BreakMeasurer.test.ts
```

Expected: all tests in file pass (existing + new).

- [ ] **Step 5: Run full pagination suite**

```bash
cd frontend/apps/web && "/c/Program Files/nodejs/npx.cmd" vitest run src/features/documents/ck5/plugins/MddmPaginationPlugin --reporter=dot
```

Expected: all suites pass.

- [ ] **Step 6: Commit**

```bash
git add frontend/apps/web/src/features/documents/ck5/plugins/MddmPaginationPlugin/BreakMeasurer.ts frontend/apps/web/src/features/documents/ck5/plugins/MddmPaginationPlugin/__tests__/BreakMeasurer.test.ts
git -c commit.gpgsign=false commit -m "feat(ck5-pagination): preventive break + fixed-stride prediction in measurer"
```

---

## Task 2: `usePredictivePageCount` hook (Codex)

**Goal:** React hook owning `--mddm-pages` CSS var via `ResizeObserver` on editable root.

**Files:**
- Create: `frontend/apps/web/src/features/documents/ck5/react/usePredictivePageCount.ts`
- Test: `frontend/apps/web/src/features/documents/ck5/react/__tests__/usePredictivePageCount.test.tsx`

- [ ] **Step 1: Write failing test**

Create `__tests__/usePredictivePageCount.test.tsx`:

```tsx
/**
 * @vitest-environment jsdom
 */
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { renderHook, act } from '@testing-library/react';
import { usePredictivePageCount } from '../usePredictivePageCount';

const MM = (mm: number) => (mm / 25.4) * 96;
const STRIDE = MM(297) + 32;

class FakeResizeObserver {
  public static instances: FakeResizeObserver[] = [];
  public callback: ResizeObserverCallback;
  public constructor(cb: ResizeObserverCallback) {
    this.callback = cb;
    FakeResizeObserver.instances.push(this);
  }
  public observe = vi.fn();
  public unobserve = vi.fn();
  public disconnect = vi.fn();
  public fire(height: number): void {
    this.callback(
      [{ contentRect: { height } } as unknown as ResizeObserverEntry],
      this as unknown as ResizeObserver,
    );
  }
}

beforeEach(() => {
  FakeResizeObserver.instances = [];
  (globalThis as any).ResizeObserver = FakeResizeObserver;
});

describe('usePredictivePageCount', () => {
  it('defaults to 1 when editable null', () => {
    const { result } = renderHook(() => usePredictivePageCount(null));
    expect(result.current).toBe(1);
  });

  it('updates pages on resize — scrollHeight/stride ceil', () => {
    const el = document.createElement('div');
    Object.defineProperty(el, 'scrollHeight', { get: () => STRIDE * 2.1, configurable: true });
    el.style.setProperty = vi.fn();

    const { result } = renderHook(() => usePredictivePageCount(el));
    act(() => {
      FakeResizeObserver.instances[0].fire(STRIDE * 2.1);
    });
    expect(result.current).toBe(3);
    expect(el.style.setProperty).toHaveBeenCalledWith('--mddm-pages', '3');
  });

  it('minimum 1 page even for empty content — and writes CSS var', () => {
    const el = document.createElement('div');
    Object.defineProperty(el, 'scrollHeight', { get: () => 0, configurable: true });
    el.style.setProperty = vi.fn();

    const { result } = renderHook(() => usePredictivePageCount(el));
    act(() => {
      FakeResizeObserver.instances[0].fire(0);
    });
    expect(result.current).toBe(1);
    // CSS var must be set even when next === initial state (1). Downstream
    // min-height calc depends on the var being present, not on React state.
    expect(el.style.setProperty).toHaveBeenCalledWith('--mddm-pages', '1');
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd frontend/apps/web && "/c/Program Files/nodejs/npx.cmd" vitest run src/features/documents/ck5/react/__tests__/usePredictivePageCount.test.tsx
```

Expected: FAIL — module not found.

- [ ] **Step 3: Create `usePredictivePageCount.ts`**

```ts
import { useEffect, useState } from 'react';

const MM_PER_INCH = 25.4;
const PX_PER_INCH = 96;
const PAGE_HEIGHT_PX = (297 / MM_PER_INCH) * PX_PER_INCH;
const PAGE_GAP_PX = 32;
const STRIDE_PX = PAGE_HEIGHT_PX + PAGE_GAP_PX;

export function usePredictivePageCount(editable: HTMLElement | null): number {
  const [pages, setPages] = useState(1);

  useEffect(() => {
    if (!editable) return;
    if (typeof ResizeObserver === 'undefined') {
      // Fallback: static 1 page.
      return;
    }
    const compute = () => {
      const height = editable.scrollHeight;
      const next = Math.max(1, Math.ceil(height / STRIDE_PX));
      // Always write the CSS var, even on initial 1-page state, so consumers
      // downstream (CSS min-height calc) never read a missing `--mddm-pages`.
      editable.style.setProperty('--mddm-pages', String(next));
      setPages((prev) => (prev !== next ? next : prev));
    };
    const ro = new ResizeObserver(compute);
    ro.observe(editable);
    compute(); // initial sync
    return () => ro.disconnect();
  }, [editable]);

  return pages;
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
cd frontend/apps/web && "/c/Program Files/nodejs/npx.cmd" vitest run src/features/documents/ck5/react/__tests__/usePredictivePageCount.test.tsx
```

Expected: 3 tests pass.

- [ ] **Step 5: Commit**

```bash
git add frontend/apps/web/src/features/documents/ck5/react/usePredictivePageCount.ts frontend/apps/web/src/features/documents/ck5/react/__tests__/usePredictivePageCount.test.tsx
git -c commit.gpgsign=false commit -m "feat(ck5-pagination): usePredictivePageCount hook via ResizeObserver"
```

---

## Task 3: `PageFooters` component (Codex)

**Goal:** Absolute-positioned overlay of N footer labels inside editor root via React portal.

**Files:**
- Create: `frontend/apps/web/src/features/documents/ck5/react/PageFooters.tsx`
- Test: `frontend/apps/web/src/features/documents/ck5/react/__tests__/PageFooters.test.tsx`

- [ ] **Step 1: Write failing test**

Create `__tests__/PageFooters.test.tsx`:

```tsx
/**
 * @vitest-environment jsdom
 */
import { describe, it, expect } from 'vitest';
import { render } from '@testing-library/react';
import { PageFooters } from '../PageFooters';

describe('PageFooters', () => {
  it('returns null when portal target is null', () => {
    const { container } = render(<PageFooters pages={3} portalTarget={null} />);
    expect(container.textContent).toBe('');
  });

  it('renders N footers via portal into target wrapper', () => {
    const target = document.createElement('div');
    document.body.appendChild(target);
    render(<PageFooters pages={3} portalTarget={target} />);
    const footers = target.querySelectorAll('[data-mddm-page-footer]');
    expect(footers).toHaveLength(3);
    expect(footers[0].textContent).toBe('Page 1');
    expect(footers[2].textContent).toBe('Page 3');
  });

  it('positions each footer at correct y', () => {
    const target = document.createElement('div');
    document.body.appendChild(target);
    render(<PageFooters pages={2} portalTarget={target} />);
    const footers = target.querySelectorAll<HTMLElement>('[data-mddm-page-footer]');
    // Numeric compare with tolerance — don't assert exact floating-point string.
    // Page 1 footer: top = 0*STRIDE + MM(287) = 1084.72px.
    // Page 2 footer: top = 1*STRIDE + MM(287) = 2238.72px.
    const MM = (mm: number) => (mm / 25.4) * 96;
    const STRIDE = MM(297) + 32;
    expect(parseFloat(footers[0].style.top)).toBeCloseTo(MM(287), 2);
    expect(parseFloat(footers[1].style.top)).toBeCloseTo(STRIDE + MM(287), 2);
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd frontend/apps/web && "/c/Program Files/nodejs/npx.cmd" vitest run src/features/documents/ck5/react/__tests__/PageFooters.test.tsx
```

Expected: FAIL — module not found.

- [ ] **Step 3: Create `PageFooters.tsx`**

```tsx
import { createPortal } from 'react-dom';

const MM = (mm: number) => (mm / 25.4) * 96;
const STRIDE_PX = MM(297) + 32;
const FOOTER_Y_OFFSET_PX = MM(287); // 10mm above gray band inside page bottom margin

const footerStyle = (pageIndex: number): React.CSSProperties => ({
  position: 'absolute',
  top: `${pageIndex * STRIDE_PX + FOOTER_Y_OFFSET_PX}px`,
  right: `${MM(20)}px`,
  color: '#6b7280',
  font: '11px/1 Carlito, "Liberation Sans", Arial, sans-serif',
  pointerEvents: 'none',
  zIndex: 1,
});

export interface PageFootersProps {
  pages: number;
  /** React-owned wrapper (e.g. `paperWrapperRef.current`). MUST NOT be the
   *  CKEditor DOM root — CK5's renderer reconciles its subtree and can remove
   *  non-editor children. Pass a plain div sized to the editor geometry. */
  portalTarget: HTMLElement | null;
}

export function PageFooters({ pages, portalTarget }: PageFootersProps): JSX.Element | null {
  if (!portalTarget) return null;
  const safeCount = Math.max(1, pages | 0);
  const nodes = Array.from({ length: safeCount }, (_, i) => (
    <div key={i} data-mddm-page-footer style={footerStyle(i)}>
      Page {i + 1}
    </div>
  ));
  return createPortal(<>{nodes}</>, portalTarget);
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
cd frontend/apps/web && "/c/Program Files/nodejs/npx.cmd" vitest run src/features/documents/ck5/react/__tests__/PageFooters.test.tsx
```

Expected: 3 tests pass. If `top` values differ from test expectation by >0.01, read the actual printed values and correct test literals (floating point precision).

- [ ] **Step 5: Commit**

```bash
git add frontend/apps/web/src/features/documents/ck5/react/PageFooters.tsx frontend/apps/web/src/features/documents/ck5/react/__tests__/PageFooters.test.tsx
git -c commit.gpgsign=false commit -m "feat(ck5-pagination): PageFooters overlay portalled into editor root"
```

---

## Task 4: CSS gradient + min-height formula (Codex)

**Goal:** Editable gets repeating-linear-gradient paper, min-height snaps to full A4 slots, remove obsolete `[data-mddm-page-break-after]::before/::after` rules (replaced by gradient + `PageFooters`). Surgical edits against `main` baseline.

**Files:**
- Modify: `frontend/apps/web/src/features/documents/ck5/react/AuthorEditor.module.css`

Baseline (post Task 0 stash) contains an editable block rule starting with `.editable :global(.ck.ck-editor__editable.ck-editor__editable_inline) { … min-height: 297mm; … background: #fff; … }` plus `[data-mddm-page-break-after]::before/::after` rules and a `.mddm-page-overlay` rule.

- [ ] **Step 1: Read baseline to confirm current rule set**

```bash
cd "C:/Users/leandro.theodoro.MN-NTB-LEANDROT/Documents/MetalDocs" && git show HEAD:frontend/apps/web/src/features/documents/ck5/react/AuthorEditor.module.css
```

Confirm it ends with 3 rules: the focus rule, `[data-mddm-page-break-after]` block (with margin-bottom + ::before + ::after), and `.mddm-page-overlay`.

- [ ] **Step 2: Edit the `.editable` container — add `position: relative`**

Exact Edit #1:

```
OLD:
.editable {
  flex: 1;
  overflow: auto;
  padding: 32px 24px;
  display: flex;
  justify-content: center;
  align-items: flex-start;
}
NEW:
.editable {
  flex: 1;
  overflow: auto;
  padding: 32px 24px;
  display: flex;
  justify-content: center;
  align-items: flex-start;
  position: relative;
}
```

- [ ] **Step 3: Edit the editable inline rule — change min-height, swap background, add box-sizing/z-index**

Exact Edit #2 — replace the entire rule starting with `.editable :global(.ck.ck-editor__editable.ck-editor__editable_inline) {` up through its closing `}` (the FIRST occurrence only — there are two similar selectors, only replace the non-`:focus` one):

```css
.editable :global(.ck.ck-editor__editable.ck-editor__editable_inline) {
  width: 210mm;
  min-height: calc(var(--mddm-pages, 1) * (297mm + 32px) - 32px);
  padding: 25mm 20mm;
  font-family: Carlito, 'Liberation Sans', Arial, sans-serif;
  box-sizing: border-box;
  margin: 0;
  flex: 0 0 auto;
  overflow: visible;
  position: relative;
  z-index: 1;
  border: 1px solid rgba(0, 0, 0, 0.08);
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.15);
  border-radius: 2px;
  background:
    repeating-linear-gradient(
      to bottom,
      #ffffff 0,
      #ffffff 297mm,
      #e8eaed 297mm,
      #e8eaed calc(297mm + 32px)
    );
}
```

Key deltas vs baseline:
- `min-height: 297mm` → stride formula with last-gutter trim.
- `background: #fff` → `repeating-linear-gradient(...)`.
- Added `box-sizing: border-box`, `position: relative`, `z-index: 1`.

- [ ] **Step 4: Delete obsolete `[data-mddm-page-break-after]` rules + `.mddm-page-overlay`**

Baseline has three rules that must be removed (they were the v1 static-CSS-based page-break paint — now redundant with gradient + `PageFooters` + inline `margin-bottom` from measurer):

Exact Delete #1 — remove this block:

```css
.editable :global(.ck.ck-editor__editable.ck-editor__editable_inline [data-mddm-page-break-after]) {
  position: relative;
  margin-bottom: 32px;
}
```

Exact Delete #2 — remove this block:

```css
.editable :global(.ck.ck-editor__editable.ck-editor__editable_inline [data-mddm-page-break-after]::before) {
  content: '';
  position: absolute;
  bottom: -32px;
  left: -20mm;
  right: -20mm;
  height: 32px;
  background: #e8eaed;
  pointer-events: none;
}
```

Exact Delete #3 — remove this block:

```css
.editable :global(.ck.ck-editor__editable.ck-editor__editable_inline [data-mddm-page-break-after]::after) {
  content: 'Page ' attr(data-mddm-next-page);
  position: absolute;
  left: 0;
  right: 0;
  top: calc(100% + 8px);
  text-align: right;
  color: #6b7280;
  font: 11px/1 Carlito, sans-serif;
  pointer-events: none;
}
```

Exact Delete #4 — remove this block:

```css
.editable :global(.mddm-page-overlay) {
  background: #e8eaed;
  color: #5f6368;
  font: 11px/1 Carlito, sans-serif;
  padding: 6px 12px;
  text-align: right;
  box-shadow: 0 -1px 3px rgba(0, 0, 0, 0.08);
  pointer-events: none;
}
```

Leave all other rules (font-face blocks, `.shell`, `.toolbar`, the `:focus`/`.ck-focused` rule) untouched.

- [ ] **Step 4a: Add `.paperWrapper` rule (for PageFooters portal target)**

`PageFooters` portals into a React-owned wrapper div (not CKEditor's root) to avoid CK5 renderer reconciliation removing the footer nodes. The wrapper must be 210mm wide with `position: relative` so footer absolute-positioning maps to editor geometry.

Exact Insert — append this rule immediately before the editable inline rule (right after `.toolbar`):

```css
.paperWrapper {
  position: relative;
  width: 210mm;
  flex: 0 0 auto;
}
```

- [ ] **Step 5: Diff-scope check**

```bash
git diff frontend/apps/web/src/features/documents/ck5/react/AuthorEditor.module.css
```

Expected hunks:
1. `.editable` rule gains `position: relative`.
2. New `.paperWrapper` rule inserted.
3. Editable inline rule — replaced block per Step 3.
4. `[data-mddm-page-break-after]` (3 rules) deleted.
5. `.mddm-page-overlay` rule deleted.

No other hunks. If font-face rules or the focus rule show diffs, revert and redo surgically.

- [ ] **Step 6: Run any CSS-module-affected tests**

```bash
cd frontend/apps/web && "/c/Program Files/nodejs/npx.cmd" vitest run src/features/documents/ck5/react --reporter=dot
```

Expected: all react suites pass. CSS changes don't break JS tests (CSS modules just export class names).

- [ ] **Step 7: Commit**

```bash
git add frontend/apps/web/src/features/documents/ck5/react/AuthorEditor.module.css
git -c commit.gpgsign=false commit -m "feat(ck5-pagination): editable repeating-gradient background + stride min-height"
```

---

## Task 5: AuthorEditor wire-up (Codex)

**Goal:** Add `PageFooters` overlay + `usePredictivePageCount` to `AuthorEditor.tsx`. Minimal surgical edits against the `main` baseline (post Task 0 stash). Do NOT rewrite the file — use Edit-style diffs only.

**Important — portal target:** Footers are portalled into a **React-owned** wrapper div (`paperWrapperRef`), NOT into CKEditor's DOM root (`view.getDomRoot()`). CKEditor owns the renderer for its root subtree and may remove non-editor children on re-render cycles. The paper wrapper is a plain div, identically sized to the editor (210mm wide, `position: relative`), so footer absolute positioning maps 1:1 to editor geometry without being owned by CK5.

**Files:**
- Modify: `frontend/apps/web/src/features/documents/ck5/react/AuthorEditor.tsx`
- Modify: `frontend/apps/web/src/features/documents/ck5/react/AuthorEditor.module.css` (add `.paperWrapper` rule — done in Task 4 Step 4a below, NOT in Task 5; skip this bullet if already done)

Baseline reference (post Task 0 stash, from `main` HEAD):

```tsx
import { useMemo, useRef, useState } from 'react';
import { CKEditor } from '@ckeditor/ckeditor5-react';
import { DecoupledEditor } from 'ckeditor5';
import type { ClassicEditor } from 'ckeditor5';
import type { DecoupledEditorUIView } from 'ckeditor5';
import 'ckeditor5/ckeditor5.css';
import { createAuthorConfig } from '../config/editorConfig';
import { PageCounter } from './PageCounter';
import { PaginationDebugOverlay } from './PaginationDebugOverlay';
import styles from './AuthorEditor.module.css';
```
…and JSX passes `<PageCounter editor={editor} />`. Keep that API untouched — `PageCounter` reads the measurer directly. Our new `pages` from the hook is for `--mddm-pages` + `PageFooters` only.

- [ ] **Step 1a: Edit import block — add useEffect, PageFooters, usePredictivePageCount**

Exact Edit #1 — replace the first import line:

```
OLD: import { useMemo, useRef, useState } from 'react';
NEW: import { useEffect, useMemo, useRef, useState } from 'react';
```

Exact Edit #2 — insert after `import { PageCounter } from './PageCounter';`:

```
import { PageFooters } from './PageFooters';
import { usePredictivePageCount } from './usePredictivePageCount';
```

- [ ] **Step 1b: Edit function body — add paperWrapperRef, editorRoot state, hook call, root-capture effect**

Exact Edit #3 — insert after `const [editor, setEditor] = useState<ClassicEditor | null>(null);`:

```tsx
  const paperWrapperRef = useRef<HTMLDivElement | null>(null);
  const [editorRoot, setEditorRoot] = useState<HTMLElement | null>(null);
  const [portalTarget, setPortalTarget] = useState<HTMLElement | null>(null);
  const pages = usePredictivePageCount(editorRoot);

  useEffect(() => {
    if (!editor) return;
    const root = editor.editing?.view?.getDomRoot?.() as HTMLElement | null | undefined;
    setEditorRoot(root ?? null);
    setPortalTarget(paperWrapperRef.current);
  }, [editor]);
```

`editorRoot` is used by `usePredictivePageCount` for `ResizeObserver` (safe — read-only observe). `portalTarget` is the React-owned `paperWrapperRef.current` passed to `<PageFooters>` for `createPortal` (NOT the CKEditor DOM root).

(The `pages` variable drives `--mddm-pages` via the hook side effect + `PageFooters` count. ESLint `no-unused-vars` may flag; if so, reference it via `void pages` or suppress. Prefer keeping it named for readability.)

- [ ] **Step 1c: Edit JSX — wrap CKEditor in `<div ref={paperWrapperRef} className={styles.paperWrapper}>` + inject `<PageFooters>` inside wrapper**

Exact Edit #4 — change the JSX block for `.editable`. Current baseline:

```tsx
      <div className={styles.editable} data-ck5-role="editable">
        <CKEditor ... />
      </div>
```

New JSX:

```tsx
      <div className={styles.editable} data-ck5-role="editable">
        <div ref={paperWrapperRef} className={styles.paperWrapper}>
          <CKEditor ... />
          <PageFooters pages={pages} portalTarget={portalTarget} />
        </div>
      </div>
```

The `paperWrapper` div is 210mm wide, `position: relative`, matches the CK5 editor bounding box. Footers portal into it, not into the editor. CK5 renderer only owns the `<CKEditor>` subtree — the sibling `<PageFooters>` portalled into `paperWrapperRef.current` is safe from CK5 DOM reconciliation.

- [ ] **Step 1d: Diff-scope check — verify only expected lines changed**

```bash
cd "C:/Users/leandro.theodoro.MN-NTB-LEANDROT/Documents/MetalDocs" && git diff --stat frontend/apps/web/src/features/documents/ck5/react/AuthorEditor.tsx
```

Expected: ~12–15 insertions, 1 deletion (the import line modification). No deletions elsewhere. If more deletions appear, revert and redo edits.

```bash
git diff frontend/apps/web/src/features/documents/ck5/react/AuthorEditor.tsx
```

Visual scan: every changed hunk must correspond to Step 1a / 1b / 1c. No other hunks allowed. In particular the `<PageCounter editor={editor} />` line must remain untouched.

- [ ] **Step 2: Run AuthorEditor tests**

```bash
cd frontend/apps/web && "/c/Program Files/nodejs/npx.cmd" vitest run src/features/documents/ck5/react/__tests__/AuthorEditor.test.tsx
```

Expected: existing 2 tests pass (render toolbar + onChange fires).

- [ ] **Step 3: Run full React + pagination suites**

```bash
cd frontend/apps/web && "/c/Program Files/nodejs/npx.cmd" vitest run src/features/documents/ck5/plugins/MddmPaginationPlugin src/features/documents/ck5/react --reporter=dot
```

Expected: all suites green.

- [ ] **Step 4: Commit**

```bash
git add frontend/apps/web/src/features/documents/ck5/react/AuthorEditor.tsx
git -c commit.gpgsign=false commit -m "feat(ck5-pagination): wire AuthorEditor to usePredictivePageCount + PageFooters"
```

---

## Task 6: Final Vitest + preview validation (Opus manual)

**Goal:** Verify v3 behaves correctly both in tests and in the running dev preview.

**Files:** none (verification only).

- [ ] **Step 1: Full ck5 suite run**

```bash
cd frontend/apps/web && "/c/Program Files/nodejs/npx.cmd" vitest run src/features/documents/ck5 --reporter=dot
```

Expected: every test in `src/features/documents/ck5/` passes (baseline count from Task 0 Step 4 + 3 new PageFooters tests + 3 new usePredictivePageCount tests + 1 new BreakMeasurer preventive-break test = baseline+7).

- [ ] **Step 2: Start preview**

From `launch.json` config `ck5-plan-c-web` (port 4175), start preview via Claude Preview MCP or manually:

```bash
cd frontend/apps/web && "/c/Program Files/nodejs/npx.cmd" vite --port 4175
```

- [ ] **Step 3: Manual UX checks in preview**

In browser at `http://localhost:4175`, open the author editor with an existing template or new one:

1. Type Enter 30× rapidly. Expected: gray band always at y = N × (297mm + 32px). Paper never "grows" beyond the gradient cycle. Content stays inside white regions.
2. Type a long paragraph that spans page 1 → page 2. Expected: content naturally stops at page 1 bottom, resumes at page 2 top margin. Gray band visible between. "Page 1" footer visible bottom-right of page 1. "Page 2" footer visible bottom-right of page 2.
3. Delete content from middle of document. Expected: pages shrink synchronously. Gray band re-positions. No ghost frames.
4. Reach page 4 via content. Expected: all 4 pages at full A4 height even if page 4 content is < MM(247).
5. DevTools inspect: `.ck-editor__editable` has `background: repeating-linear-gradient(...)`, `--mddm-pages` CSS var matches visual count.

- [ ] **Step 4: If any UX check fails**

Record the repro in a new `docs/superpowers/notes/YYYY-MM-DD-ck5-v3-regression.md` with: keystroke sequence, screenshot, expected vs actual, editable.scrollHeight, `--mddm-pages` value. Root-cause before patching — do NOT patch blindly. If `mcp__claudewatch__get_blockers` is available, also log there; otherwise file is sufficient. Reopen Codex with the repro note as context for a targeted fix.

- [ ] **Step 5: If all UX checks pass — unstash reference tree**

```bash
git stash list  # Confirm v2 stash still there.
```

Leave v2 stash in place for comparison reference. Do NOT drop it — user may want to diff behavior later.

- [ ] **Step 6: Summary commit**

No code change, just a tag commit marking v3 complete. Skip if no tracked-file changes pending.

```bash
git log --oneline -10
```

Expected: last 5 commits are tasks 1–5.

---

## Success criteria (mirrors spec)

1. ✅ Type Enter 100× fast → gray band always at y = N × (297mm + 32px), never elsewhere.
2. ✅ Page count = `Math.ceil(scrollHeight / (MM(297) + 32))`, updates synchronously (ResizeObserver, not debounce).
3. ✅ Partial last page renders full A4 height.
4. ✅ "Page N" footer visible inside each page's bottom margin zone, 10 mm above gray band.
5. ✅ All Vitest suites in `src/features/documents/ck5/` pass (baseline+7).
6. ✅ DOCX export from preview includes `<p data-mddm-page-break-after>` markers matching server.

## Known tradeoffs (deferred)

### BreakMeasurer full-doc scan

The v3 measurer always scans from document position [0] on every render cycle. This was a deliberate correctness fix: previously starting from a `DirtyRangeTracker` snapshot mis-seeded `currentPage = 1` when the dirty range began mid-doc, causing missed breaks when the first dirty candidate already overflowed.

**Cost:** O(N) block walk per measure cycle. On small docs (< 50 blocks) this is sub-millisecond. On large docs (500+ blocks) it could exceed a 16 ms frame budget during fast typing.

**Mitigation for now:** 50 ms debounce caps work to ~20 Hz. Initial-sync flag fires the first measure synchronously. This is acceptable for MetalDocs' typical doc size (PO / technical reports, usually < 100 blocks).

**Future work (deferred, not in this plan):**
1. Incremental cache in `BreakMeasurer`: persist `{ page cursor, bot }` per `afterBid` between cycles; on mutation, find the mutated bid's cached cursor and recompute only the suffix. Requires `DirtyRangeTracker` to return the nearest cached `afterBid`, not a raw position.
2. Schedule measure via `requestAnimationFrame` coalescing instead of `setTimeout` debounce. Pins work to frame boundaries; measurer runs at most 60 Hz.

Add those as a follow-up task under `docs/superpowers/plans/YYYY-MM-DD-ck5-pagination-v3-perf.md` once v3 ships and real-world doc sizes are measured.

### PageCounter fork

`PageCounter` (toolbar label) still derives its count from `measurer.onBreaks` events (v1 behavior) while `PageFooters` + `--mddm-pages` derive from `usePredictivePageCount` (scrollHeight-based). These two sources can diverge transiently (measurer debounce vs ResizeObserver sync). For v3 this is acceptable — the values converge within one debounce cycle and the visible layout is driven by the hook, not the counter. Follow-up: migrate `PageCounter` to accept `pages` prop from the hook so there's a single source of truth.

## Rollback

```bash
git reset --hard <commit-before-task-1>
git stash pop  # restore v2 tree if needed for reference
```
