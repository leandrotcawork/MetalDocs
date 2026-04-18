# CK5 Pagination v3 — Single-Paper Gradient Architecture

**Date:** 2026-04-18
**Status:** Approved design, awaiting implementation plan
**Supersedes:** `2026-04-17-ck5-pagination-design.md` (v1/v2 overlay-frame approach — proven broken in preview)

## Problem

Visual pagination in the CKEditor 5 DecoupledEditor (MetalDocs author editor) is broken. Two prior iterations tried an overlay-frame architecture: N absolute-positioned white paper frames behind a transparent editable, with a dynamic `margin-bottom` spacer on break paragraphs to force content alignment. Both failed in preview. User reports:

- "Page 1 starts increasing size" when pressing Enter — content flows past the first paper frame before the measurer emits a break, and since the editable sits on top of a gray shell with transparent background, paragraphs appear to "leak" into the gutter zone below the frame.
- "No gap between pages" — when the measurer does emit a break, frame stride drifts because frame geometry is read live from DOM groupings, which go stale between measurer cycles.
- Partial last page rendered shorter than a full A4.

Root cause: the overlay-frame architecture couples paint to JavaScript measurement. Any lag between content edit and measurer firing leaves the paper frames desynchronized with content. Attempts to fix via debounce tuning, speculative rendering, and dynamic spacers keep creating new race conditions.

## Goals

1. Paper (white A4 background) CANNOT become visually wrong regardless of typing speed, DOM mutation timing, or measurer state.
2. Gap between pages (32px gray gutter) always visually present at every 297mm cycle.
3. "Page N" footer label inside each page's bottom-margin zone (not in gutter).
4. Partial last page still renders full A4 height.
5. Break metadata (`data-mddm-page-break-after`, `data-mddm-next-page`) remains emitted for the DOCX / PDF export pipeline.

## Non-goals

1. Splitting the CKEditor model into separate per-page roots (Google Docs approach). Out of scope — requires custom DecoupledEditor fork.
2. Content-aware table / image splitting across page boundaries. Oversized single elements (> MM(247)) overflow the page; manual break is the escape hatch. Word behaves identically.
3. Header / page-title rendering per page. Future work.
4. Print-from-browser fidelity. Users export to DOCX or PDF; browser print is not a supported surface.

## Architecture — single paper + CSS gradient

### Paint layer (immutable)

The editable root (`.ck.ck-editor__editable.ck-editor__editable_inline`) gets a `repeating-linear-gradient` background:

```
background: repeating-linear-gradient(
  to bottom,
  #ffffff 0,
  #ffffff 297mm,
  #e8eaed 297mm,
  #e8eaed calc(297mm + 32px)
);
```

Each cycle = one A4 paper (297mm white) + one gutter (32px gray). The editable's `min-height` is `calc(var(--mddm-pages, 1) * (297mm + 32px) - 32px)` — last gutter trimmed so the editable doesn't leave a stray gray band below the last page.

Paper can never "grow wrong" because it is a mathematical cycle. No DOM measurement feeds into paint. No race condition is possible at the paint layer.

### Page-count layer (predictive, synchronous)

A React `usePredictivePageCount` hook owns `--mddm-pages`:

1. Subscribes to `ResizeObserver` on the editable root.
2. On each resize, reads `editable.scrollHeight`.
3. Computes `pages = max(1, Math.ceil(scrollHeight / (MM(297) + 32)))`.
4. If `pages` changed, writes `editable.style.setProperty('--mddm-pages', String(pages))` and updates React state.

ResizeObserver is synchronous with layout, no 200 ms debounce. Page count tracks content height within one animation frame.

### Footer layer (absolute overlay)

React `<PageFooters pages={n}>` renders `n` absolutely-positioned `<div class="pageFooter">` elements as a sibling of the editable, inside the `.editable` flex container.

Footers are absolutely positioned relative to the editor root (`.ck-editor__editable`), not the outer `.editable` flex container. The hook mounts them via a portal into the editor root so the 0,0 origin aligns with the gradient cycles.

Each footer at:
- `top: i * (MM(297) + 32) + MM(287)` (10 mm above the gray band = inside page i+1's bottom-margin zone)
- `right: MM(20)` (matches page right margin)
- `font: 11px/1 Carlito, Arial, sans-serif; color: #6b7280; pointer-events: none; z-index: 1`

Footer position is pure arithmetic on page index. No DOM reads, no measurer coupling.

### Content-alignment layer (BreakMeasurer + dynamic spacer)

The paint and footer layers always look correct regardless of content position. The remaining responsibility is **ensuring typed content lands at the right Y coordinate** — specifically, past the gray band into the next page's top-margin zone.

BreakMeasurer rewrite:

1. Subscribes to CKEditor `view.render` event.
2. Debounce reduced from 200 ms to 50 ms. On initial load (first render after editor ready), runs synchronously without debounce.
3. Uses fixed-stride prediction, not measured pageTop:
   - `page1TopY = MM(25)` — hard-coded to editor's padding-top. Does NOT depend on first candidate's position (first block may be a non-candidate like a table, which would make `firstCandidate.offsetTop > MM(25)` and break the prediction).
   - For each candidate K:
     - `predictedPageTop = page1TopY + (currentPage - 1) * (MM(297) + 32)`
     - `predictedPageCap = predictedPageTop + MM(247)`
     - If `K.offsetTop + K.offsetHeight > predictedPageCap` AND `prevAfterBid != null`:
       - Emit break on previous candidate.
       - `currentPage += 1`.
       - `targetNextTop = page1TopY + (currentPage - 1) * (MM(297) + 32)` (next page's first-paragraph offsetTop)
       - `spacerPx = Math.max(0, targetNextTop - prevCandidate.bot)`
       - Push `{ afterBid, pageNumber, yPx, spacerPx }` to breaks.

Preventive semantics: break triggers **before** K is placed on overflowing page, so content never lands in the bottom margin of the current page or crosses the gray band.

PageOverlayView applies the inline `style.margin-bottom = spacerPx + 'px'` on each break paragraph. Stale break markers get their attributes and inline style cleared.

### Metadata layer (export)

`data-mddm-page-break-after=""` + `data-mddm-next-page={N}` on each break paragraph (unchanged from current v2 implementation). The DOCX and PDF export pipelines read these attributes. This layer is a side effect of the content-alignment layer, not a separate subsystem.

## Data flow

```
User edit
    |
    v
CKEditor renders view
    |
    +---> ResizeObserver fires (editable.scrollHeight changed)
    |         |
    |         v
    |     usePredictivePageCount recomputes pages → --mddm-pages updated
    |         |
    |         v
    |     editable min-height grows to next full A4 slot
    |         |
    |         v
    |     <PageFooters> re-renders with new N
    |
    +---> view 'render' event
              |
              v (50 ms debounce, skipped on first render)
          BreakMeasurer.measure
              |
              v
          planBreaks → candidates
              |
              v
          For each candidate: predict page, detect overflow, compute spacerPx
              |
              v
          Emit ComputedBreak[] → listeners
              |
              v
          PageOverlayView.update: set attrs + inline margin-bottom
              |
              v
          view 'render' refires — idempotent (same attrs, same spacers → no-op)
```

## Error handling

1. **ResizeObserver unsupported** (ancient browser): fall back to fixed `pages = 1` and log console warning. MetalDocs targets Chrome/Edge/Firefox current, all support RO.
2. **Oversized element** (paragraph / image / table > MM(247)): measurer does not emit break for it; element overflows into gray band and next page. Acceptable — matches Word behavior. User inserts manual page break as escape hatch.
3. **Measurer runs before fonts loaded**: existing `document.fonts.ready` gate stays. Skip first measurement if fonts still loading.
4. **Empty editable** (0 paragraphs): `planBreaks` returns empty, no breaks emitted, `pages = 1` from ResizeObserver floor. Editor shows 1 blank paper.
5. **Spacer negative** (content overshoots predicted target, e.g. font metric variance): clamp to `Math.max(0, spacerPx)`. Small visual overlap of content into the gutter is preferable to pulling next paragraph up.

## Testing strategy

### Unit tests (Vitest + jsdom)

- `BreakMeasurer.test.ts` — verify fixed-stride prediction, preventive break trigger, spacerPx math. Existing tests updated for new ComputedBreak shape.
- `PageOverlayView.test.ts` — verify inline `margin-bottom` set and cleared.
- `usePredictivePageCount.test.ts` — NEW, mock ResizeObserver, verify page count formula.
- `PageFooters.test.tsx` — NEW, verify N footers rendered at correct absolute positions.
- `data-contract.test.ts` — verify `ComputedBreak.spacerPx` field contract.

### Integration

- `exceptionRoundTrip.integration.test.ts` — existing CKEditor-round-trip test confirms break metadata survives save/load.
- Manual preview walkthrough on http://localhost:4175: type Enter until page 2 appears, verify gutter visible, footer correct, content starts at page 2 top margin. Repeat to page 3, page 4.

### Visual regression

Out of scope for this spec (no visual regression tooling in project). Manual preview review is sufficient.

## File changes

### Modified

- `frontend/apps/web/src/features/documents/ck5/plugins/MddmPaginationPlugin/BreakMeasurer.ts`
  - Keep existing fixed-stride prediction from v2.
  - Add preventive break semantics (emit break on candidate K-1 when K would overflow).
  - Reduce debounce 200 → 50 ms, add initial-render zero-debounce flag.

- `frontend/apps/web/src/features/documents/ck5/plugins/MddmPaginationPlugin/PageOverlayView.ts`
  - Keep v2 behavior (inline `margin-bottom` from `spacerPx`).
  - No functional change.

- `frontend/apps/web/src/features/documents/ck5/react/AuthorEditor.module.css`
  - Add `background: repeating-linear-gradient(...)` to editable.
  - Update `min-height` formula: `calc(var(--mddm-pages, 1) * (297mm + 32px) - 32px)`.
  - Remove `.pageFrames` + `.pageFrame` rules.
  - Remove editable `background: transparent` override.

- `frontend/apps/web/src/features/documents/ck5/react/AuthorEditor.tsx`
  - Replace `<PageFrames count={pages} />` with `<PageFooters pages={pages} />`.
  - Add `usePredictivePageCount(editorRef)` hook call. Drop current measurer-based `pages` state if not needed for footers.
  - Measurer subscription retained only for break-metadata updates; page-count state now sourced from predictive hook.

- `shared/mddm-pagination-types/index.ts`
  - No change. `ComputedBreak.spacerPx` stays.

### Added

- `frontend/apps/web/src/features/documents/ck5/react/PageFooters.tsx`
  - N absolute-positioned footer labels, pure arithmetic positioning.

- `frontend/apps/web/src/features/documents/ck5/react/usePredictivePageCount.ts`
  - React hook wrapping ResizeObserver.

- Test files for both new components.

### Deleted

- `frontend/apps/web/src/features/documents/ck5/react/PageFrames.tsx` (replaced).

## Success criteria

1. Type Enter 100× fast in author editor preview. Gray band always at y = N × (297mm + 32px). Never appears elsewhere. Content never flows past bottom margin of current page into gray band.
2. Page count (visible frames on screen) = `Math.ceil(scrollHeight / (MM(297) + 32))` and updates synchronously on content change.
3. Partial last page renders full A4 (297mm white) regardless of content length.
4. "Page N" footer visible at bottom-right of each page, 10 mm above gray band.
5. All existing Vitest suites pass (30 pagination-related tests minimum).
6. DOCX export from preview includes correct `<p data-mddm-page-break-after>` markers matching server pagination.

## Migration

No database, schema, or export format changes. Only editor rendering layer. Zero-touch for server and other consumers.

## Rollback plan

If v3 regresses in preview, revert:
```
git checkout HEAD~1 -- frontend/apps/web/src/features/documents/ck5/
```
v2 dirty tree is already in working copy (spacer-based overlay frames) but was also broken; revert target is `main` HEAD which has the committed 33-task plan v1.
