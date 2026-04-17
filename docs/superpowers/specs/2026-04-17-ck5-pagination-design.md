# CK5 Pagination Design

**Status:** Draft — brainstorm-approved, awaiting user review before plan-writing.
**Date:** 2026-04-17
**Branch:** migrate/ck5-plan-c
**Depends on:** shipped A4 canvas CSS (commit c3b297c / 52a587c lineage)

---

## Goal

Deliver Microsoft-Word-style pagination in the CK5 DecoupledEditor used for MetalDocs Author/Template views. Two concrete outcomes:

1. **Editor visual** — stacked A4 pages with a grey gutter gap between them, a shadow drop, and page numbers displayed in the gutter. User sees "page N of M" live while typing. Must feel professional, not a demo.
2. **Export parity** — when the author sees `N` pages in the editor, the downloaded `.docx` opens in Microsoft Word / WPS Office at exactly the same break positions, and the generated PDF matches byte-close.

No premium CKEditor plugins (GPL tier). No abandonment of CKEditor 5 (the OnlyOffice / Collabora alternatives are rejected — incompatible architectures).

Scale target: 500-page contracts remain typing-smooth.

---

## Architecture

### High-level layers

```
┌──────────────────────────── packages/mddm-layout-tokens ────────────────────────────┐
│  page dims · margins · Carlito font stack · line-height · spacing · SLO             │
└──────────────▲───────────────────────────────────────────────────▲──────────────────┘
               │ imports                                           │ imports
               │                                                   │
┌──────────────┴─────────────────────┐         ┌───────────────────┴──────────────────┐
│  EDITOR (browser, CK5 v48)         │         │  EXPORT (apps/ck5-export, Node)      │
│                                    │         │                                      │
│  BlockIdentityPlugin (NEW)         │  HTTP   │  paginate-with-chromium.ts (NEW)     │
│    └─ stamps data-mddm-bid         │ ──────▶ │   ├─ Paged.js in Playwright worker   │
│                                    │ getData │   ├─ sentinel injection              │
│  MddmPaginationPlugin (NEW)        │ ({pag}) │   └─ scrape + reconcile              │
│   ├─ requires BlockIdentityPlugin  │         │                                      │
│   ├─ BreakPlanner                  │         │  html-to-export-tree.ts (existing)   │
│   ├─ BreakMeasurer                 │         │  docx-emitter/ (existing + hygiene)  │
│   ├─ PageOverlayView (uiElement)   │         │  print-stylesheet/ (existing)        │
│   ├─ DirtyRangeTracker             │         │                                      │
│   └─ SectionScope                  │         │  Gotenberg → PDF bytes (unchanged)   │
└────────────────────────────────────┘         └──────────────────────────────────────┘
```

### Pattern inheritance from research

- **Client DOM measurement** for live editor visual = same pattern as CKEditor's paid Pagination plugin. Validated as industry-standard.
- **`data-pagination-page="N"` attribute** on serialized HTML = CKEditor industry convention. We adopt it to stay ecosystem-compatible.
- **Paged.js inside headless Chromium** for server-side break authority = proven production pattern (`pagedjs-cli`, Cloud Native PDF, Hocuspocus export).
- **DOCX hygiene recipe** (explicit `w:br` + `widowControl=0` + `autoHyphenation=false` + embedded Carlito TTF + pinned compat flags) = deterministic Word/WPS rendering per ECMA-376 §17.
- **CSS Paged Media for PDF** = already shipping via Gotenberg; no change needed.

---

## Components

### 1. `packages/mddm-layout-tokens` (new workspace package)

Single source of truth. Extract from `apps/ck5-export/src/layout-tokens.ts` (current location) into a shared workspace package so both editor and export service import from `@metaldocs/mddm-layout-tokens`.

Exports (additive to current `LayoutTokens`):
- `paginationSLO`: `{ maxBreakDeltaPer50Pages: 1 }`
- `blockIdentityAttr`: `'data-mddm-bid'`
- `pageBreakAttr`: `'data-pagination-page'` (industry-standard)
- `fontFallbackChain`: `['Carlito', 'Liberation Sans', 'Arial', 'sans-serif']`

Consumers: `apps/ck5-export`, `frontend/apps/web`, `apps/ck5-studio` (if it still exists post-plan-c).

### 2. `BlockIdentityPlugin` (new CK5 plugin, editor)

Location: `frontend/apps/web/src/features/documents/ck5/plugins/MddmBlockIdentityPlugin/`

Purpose: stamp every paginable block with a stable UUID, preserved through editor lifecycle.

Paginable block types:
- Native CK5: `paragraph`, `heading1-6`, `listItem`, `blockQuote`, `tableRow`, `imageBlock`, `mediaEmbed`
- MDDM widgets: `mddmSection`, `mddmRepeatable`, `mddmRepeatableItem`, `mddmDataTable`, `mddmFieldGroup`, `mddmRichBlock`

Implementation:
- Schema: extends each paginable element with `allowAttributes: ['mddmBid']`.
- Post-fixer: runs after every model change; mints UUID v4 for any paginable element missing `mddmBid`; mints a fresh UUID on split; keeps survivor's UUID on merge.
- Converters:
  - Upcast: reads `data-mddm-bid` → `mddmBid` attribute.
  - Data downcast: writes `mddmBid` → `data-mddm-bid` on the HTML element (always, not gated by pagination flag).
  - Editing downcast: same attribute mirrored for inspector visibility.
- Clipboard integration: registers a `clipboardInput` transformer that re-mints colliding IDs after upcast (prevents paste-from-same-doc ID duplicates).
- Dev overlay: hover-tooltip shows `bid` in dev builds (feature-flagged).

Schema migration: bump `data-mddm-schema="4"`. Legacy HTML (no bids) gets bids minted on first upcast; migration logs a `schema-upgrade` event.

### 3. `MddmPaginationPlugin` (new CK5 plugin, editor)

Location: `frontend/apps/web/src/features/documents/ck5/plugins/MddmPaginationPlugin/`

Sub-modules:

**`index.ts`** — plugin entrypoint. Requires `BlockIdentityPlugin`. Registers the view listener, wires sub-modules.

**`BreakPlanner.ts`** (Phase A — model walk, O(dirty range)):
- Input: `DirtyRangeTracker.dirtyStart` (model position).
- Walks model from dirty start to document end.
- Emits legal break candidates: positions *after* block boundaries, subject to `keep-with-next` rules (e.g., `h1..h3` never break-after).
- Table rows are candidates; figures are not (atomic via `break-inside: avoid`).
- Returns `BreakCandidate[] = Array<{ bid: string, position: ModelPosition }>`.

**`BreakMeasurer.ts`** (Phase B — DOM measure, O(candidates)):
- Input: `BreakCandidate[]`.
- Subscribes: `editor.editing.view.document.on('render')` + `200ms` debounce.
- For each candidate: reads `view.domConverter.mapViewToDom(candidate.viewElement).offsetTop` and `.offsetHeight`.
- Awaits any pending `HTMLImageElement.decode()` before measuring blocks containing images (prevents jitter on image load).
- Rounds to device pixels (`Math.round(px * devicePixelRatio) / devicePixelRatio`) to avoid subpixel noise.
- Accumulates heights against `contentHeightPx = tokens.page.heightMm - marginTopMm - marginBottomMm` converted via tokens' `mmToPx` helper.
- Produces `ComputedBreak[] = Array<{ afterBid: string, pageNumber: number }>`.

**`PageOverlayView.ts`** (view-layer chrome):
- Builds `uiElement`s in the editing view (never in data view, never in saved HTML).
- Renders one overlay per `ComputedBreak`: a horizontal bar in the editable's parent representing the page gutter (grey #e8eaed matching canvas bg), drop shadow, and page-number text ("Page N" at gutter-right, tokens' `theme.accent` color).
- Implementation: absolute-positioned overlays in a sibling div outside the `contenteditable` root, positioned from measured y-coordinates. `pointer-events: none` so they never steal clicks.

**`DirtyRangeTracker.ts`**:
- Subscribes to `editor.model.document.differ` via `change:data`.
- Reads `Differ.getChanges()` → identifies earliest modified position.
- Stores `dirtyStart` (cleared after pagination pass completes).
- Collapses multiple rapid changes within debounce window to single pass.

**`SectionScope.ts`**:
- Each `mddmSection` widget defines an independent pagination context.
- BreakMeasurer receives per-section content-box dimensions (matches print-CSS `.mddm-section { break-before: page }`).
- Edits inside section B do not reflow section A. Dirty range is clamped to the containing section scope.

**Data contract — `getData({pagination: true})`**:
- Mirrors CKEditor's paid convention.
- Walks current `ComputedBreak[]` and emits `data-pagination-page="N"` attribute on the block (located by `bid`) that **starts** each new page.
- If invoked without `pagination: true`, no pagination attributes emitted (autosave uses this form).
- `bid` attributes are always emitted regardless.

### 4. `paginate-with-chromium.ts` (new export service module)

Location: `apps/ck5-export/src/pagination/paginate-with-chromium.ts`

Purpose: server-side authoritative pagination using headless Chromium (via existing Gotenberg pool).

Pipeline:
1. Receives `html` (with `data-mddm-bid` on every block, optionally `data-pagination-page` markers from editor).
2. Wraps with `print-stylesheet/wrap-print-document.ts` (existing) + injects `paged.polyfill.js` (MIT) in the shell.
3. Walks wrapped HTML: for each paginable block with `data-mddm-bid`, injects a zero-width sentinel `<span data-pb-marker="{bid}" style="display:inline"></span>` as first child.
4. Sends to Gotenberg with a modified endpoint variant (or spins a Playwright worker if Gotenberg doesn't expose `page.evaluate`; Gotenberg's `chromium.screenshot` route supports `waitForExpression` which we leverage).
5. Paged.js runs in-browser, fragments DOM into `.pagedjs_page`.
6. Evaluates: `document.querySelectorAll('[data-pb-marker]')` → for each, `element.closest('.pagedjs_page').dataset.pageNumber`.
7. Returns `ServerBreak[] = Array<{ bid: string, pageNumber: number }>`.

Implementation notes:
- **Gotenberg integration**: investigate (v1 task) whether Gotenberg's PDF route can be extended to return structured break data via a custom Chromium flag, or whether a separate dedicated Playwright worker is cheaper. Default: add a new `apps/ck5-export/src/routes/paginate.ts` endpoint running a direct Playwright worker, keeping Gotenberg for PDF rendering. Warm pool size: 3 (configurable `CHROMIUM_POOL_SIZE`).
- **Licensing**: Paged.js MIT, Playwright Apache-2, Chromium BSD-style. Zero GPL conflict with editor (ADR 0001 modular monolith boundary preserved).
- **Performance target**: ≤1.5s P95 per 50-page document on warm pool. Cache by `sha256(html)` with 5-min TTL to absorb rapid re-exports.

### 5. Reconciler (in `apps/ck5-export/src/pagination/reconcile.ts`, new)

Input: `editorBreaks: ComputedBreak[]`, `serverBreaks: ServerBreak[]`.

Algorithm:
```
resolved: ReconciledBreak[] = []
for each serverBreak:
  find matching editorBreak by bid
  if both agree on pageNumber → resolved.push(editor) (exact match, UX fidelity)
  else if |editor.page - server.page| <= 1 → resolved.push(editor) + log 'minor-drift'
  else → resolved.push(server) + log 'major-drift' (server is ground truth)
for each editorBreak not in server: log 'orphaned-editor-break' + drop
for each serverBreak not in editor: resolved.push(server) + log 'server-only-break'
```

Validator runs before reconcile:
- Fail export 422 if duplicate `bid`s in HTML.
- Fail export 422 if editor-reported `bid` missing from HTML.
- Warn (not fail) if paginable block missing `bid` (migration edge case; reconciler treats as server-authoritative).

Output: `ReconciledBreak[]` → injector inserts `<div class="mddm-page-break"></div>` after the matching `bid`'d block in the HTML.

### 6. `html-to-export-tree.ts` (existing, minor extension)

Already has `mddm-page-break` recognition per wiki 31 §4 parity matrix. Verify in implementation. If absent, add:
- Match `div.mddm-page-break` → emit `ExportNode { kind: 'pageBreak' }`.
- Visitor in `docx-emitter/block-dispatch.ts` emits `new Paragraph({ children: [new PageBreak()] })` on this node.

### 7. DOCX hygiene layer (new concerns across existing emitter)

Additions in `apps/ck5-export/src/docx-emitter/`:

**`paragraph.ts`** — every `Paragraph` constructed with:
```ts
new Paragraph({
  ...existing,
  widowControl: false,  // NEW
})
```

**`emitter.ts`** — `Document` constructor gets:
```ts
new Document({
  ...existing,
  settings: {
    autoHyphenation: false,
    defaultTabStop: 720,                     // 0.5"
    characterSpacingControl: 'compressPunctuation',
    compatibility: {
      useWord2013TrackBottomHyphenation: true,
      doNotExpandShiftReturn: true,
    },
  },
  fonts: [{                                  // NEW — embed Carlito
    name: 'Carlito',
    data: readFileSync('fonts/Carlito-Regular.ttf'),
    altName: 'Calibri',
    embedRegular: true,
    embedBold: true,
    embedItalic: true,
    embedBoldItalic: true,
  }],
})
```

Carlito TTF bundled in `apps/ck5-export/fonts/` (OFL license, permissive, already Fedora-packaged).

### 8. PDF export (unchanged)

Current `apps/ck5-export/src/routes/render-pdf-html.ts` + `print-stylesheet/print-css.ts` already use CSS Paged Media via Gotenberg+Chromium.

Additive change only:
- Include `paged.polyfill.js` in `wrap-print-document.ts` shell (adds richer `@page :first`, running strings, `target-counter` support for future TOC).
- No break-marker wiring needed — CSS Paged Media handles breaks natively from the same print-css rules Paged.js consumes for DOCX reconciliation.

### 9. Parity probe (dev + CI)

`apps/ck5-export/src/pagination/parityDiff.ts` — emits JSON report per export:
```json
{
  "editorBreaks": [{"bid": "...", "page": 3}],
  "serverBreaks": [{"bid": "...", "page": 3}],
  "resolved": [...],
  "drift": {"exactMatches": 12, "minorDrift": 1, "majorDrift": 0, "orphaned": 0}
}
```

Dev mode: route exposes `/pagination-debug/:docId` returning this report + overlay rendered in editor behind `?debug=pagination` flag.

CI fixture set: `apps/ck5-export/src/__fixtures__/pagination/` — 10 canonical docs (short para, long para, heavy-table, image-heavy, nested-lists, section-rich, repeatable-rows, mixed-headings, edge-widow, 100-page-contract). Each has golden `ReconciledBreak[]`. CI gate:
- 0 major drift required.
- ≤1 break delta per 50 pages per block type.
- Break count exact match for non-drift fixtures.

### 10. Editor CSS font/margin unify (prereq)

Fix `frontend/apps/web/src/features/documents/ck5/react/AuthorEditor.module.css`:
- `padding: 25mm 20mm` → `padding: 25mm 25mm` (match tokens left margin).
- Add `@font-face` loading Carlito woff2 from assets, fallback `Liberation Sans, Arial, sans-serif`.
- `font-family: Carlito, Liberation Sans, Arial, sans-serif` on `.ck.ck-editor__editable_inline`.

Landing this before any pagination measurement avoids immediate drift.

---

## Data Flow

### Editor live session

```
user types
  ↓
CK5 model mutates (batch)
  ↓
Differ emits changes
  ↓
DirtyRangeTracker records earliest mutated position
  ↓
BlockIdentityPlugin post-fixer mints bids for any new paginable blocks
  ↓
view.document.render event fires (DOM flushed)
  ↓
MddmPaginationPlugin debounce 200ms
  ↓
BreakPlanner walks model from dirtyStart → BreakCandidate[]
  ↓
BreakMeasurer reads DOM offsetHeight at candidates → ComputedBreak[]
  ↓
PageOverlayView diffs previous vs current overlays, mutates uiElements
  ↓
Page counter UI updates ("Page N of M")
```

### Autosave

```
editor.getData()  # no pagination flag
  → HTML: <div data-mddm-bid="..." data-mddm-schema="4">content</div>
  → PUT /draft (lockVersion)
  → DB stores clean HTML (no pagination markers)
```

### DOCX export

```
editor.getData({ pagination: true })
  → HTML with bids + data-pagination-page="N" on break-start blocks
  ↓
POST /api/v1/documents/:id/export/ck5/docx
  ↓
Go API → ck5-export service
  ↓
paginate-with-chromium.ts:
  inject <span data-pb-marker="{bid}"> sentinels
  run Paged.js in Chromium
  scrape serverBreaks
  ↓
reconcile.ts:
  validate bids (fail 422 on collision/missing)
  join editor + server by bid
  honor editor ±1, override on >1 drift
  ↓
inject <div class="mddm-page-break"> at reconciled positions
  ↓
html-to-export-tree.ts (existing)
  → ExportNode tree with pageBreak nodes
  ↓
docx-emitter with hygiene:
  widowControl=0 on every Paragraph
  settings.xml pinned (autoHyphenation=false, compat flags)
  fontTable.xml embeds Carlito TTF (embedRegular/Bold/Italic/BoldItalic)
  PageBreak runs at markers
  ↓
Packer.toBuffer() → .docx bytes → stream to browser
```

### PDF export

```
editor.getData()  # no pagination flag needed
  ↓
POST /api/v1/documents/:id/export/ck5/pdf
  ↓
ck5-export route /render-pdf-html (existing)
  ↓
wrap-print-document.ts shell + print-css.ts + paged.polyfill.js (NEW in shell)
  ↓
Gotenberg → Chromium → PDF bytes
  (CSS Paged Media handles pagination, widowControl, break-before/inside,
   table-header-group, orphans/widows; no marker wiring needed)
```

---

## Error Handling

### Editor

| Condition | Response |
|---|---|
| `ResizeObserver` throws during measure | Swallow, schedule retry on next render event. |
| `image.decode()` rejects (broken asset) | Skip measure for containing block, use last-known height, log `pagination-measure-skip`. |
| `contentHeightPx` computes to ≤0 (bad tokens) | Refuse to paginate, show diagnostic banner in dev builds. |
| Section scope change mid-measure | Abort current pass, full-scope recompute on next debounce. |
| Deep nesting >20 levels (malformed HTML) | Cap walk depth, log `pagination-depth-cap`, don't crash. |
| Font not yet loaded (`document.fonts.ready`) | Await readiness before first measurement pass. |

### Block identity

| Condition | Response |
|---|---|
| Legacy HTML without bids loaded | Schema migration mints bids on upcast, logs `schema-upgrade-v4`. |
| Paste creates bid collision | Clipboard transformer re-mints, logs `paste-remint` with count. |
| Server validator sees duplicate bid | Export fails 422 with `{error: "bid-collision", bids: [...]}`; client shows user-facing error "Document content invalid, please refresh editor". |
| Server validator sees editor-reported bid not in HTML | Export fails 422 `{error: "editor-server-desync"}`; rare, indicates serialization bug. |
| Non-paginable block has bid (e.g., inline text) | Ignored, no harm. |
| Paginable block missing bid | Warn, reconciler treats as server-authoritative for that block. |

### Server paginator

| Condition | Response |
|---|---|
| Chromium worker crash | Retry once on different worker; on second crash, fail 503 `{error: "paginator-unavailable"}`. |
| Paged.js runtime error | Log stderr, fall back to "server-only" mode (ignore editor markers, use CSS Paged Media output directly). |
| Paged.js timeout (>15s) | Abort, fail 504 `{error: "paginator-timeout"}`. |
| Major drift rate >10% of breaks | Succeed but log `pagination-quality-degraded`, alert ops. |
| Asset (image) fetch fails | Use existing `inline-asset-rewriter.ts` fallback, treat image as zero-height for measurement. |

### DOCX hygiene

| Condition | Response |
|---|---|
| Carlito TTF file missing at deploy | Startup check fails `/readyz`, container won't serve. |
| `docx` library version mismatch on settings | Log compat warning, continue with best-effort subset. |
| `fontTable.xml` emission fails (encoding) | Fail export 500, surface internal error; do not ship a half-embedded DOCX. |

### Graceful degradation ladder

1. Full pipeline works → editor markers + server reconcile + DOCX hygiene = ideal output.
2. Server paginator degraded → fallback to editor-markers-only (CKEditor-paid equivalent), log `server-paginator-fallback`.
3. Editor markers missing → server paginator authoritative, no reconciliation.
4. Both missing → DOCX still generated with Word's native reflow (last resort, documented as "approximate" to user).

---

## Testing Approach

### Unit tests

Per module, colocated `__tests__/`:
- `BlockIdentityPlugin/__tests__/postFixer.test.ts` — mint on insert, preserve on edit, fresh on split, survivor on merge.
- `BlockIdentityPlugin/__tests__/clipboard.test.ts` — collision remint for same-doc paste.
- `BlockIdentityPlugin/__tests__/converters.test.ts` — upcast/downcast round-trip, schema v3→v4 migration.
- `MddmPaginationPlugin/__tests__/BreakPlanner.test.ts` — candidate enumeration, keep-with-next rules.
- `MddmPaginationPlugin/__tests__/BreakMeasurer.test.ts` — debounce, device-px rounding, image-decode await.
- `MddmPaginationPlugin/__tests__/PageOverlayView.test.ts` — uiElement emission, never in data view.
- `MddmPaginationPlugin/__tests__/DirtyRangeTracker.test.ts` — differ-based invalidation, range clamping.
- `MddmPaginationPlugin/__tests__/dataContract.test.ts` — `getData({pagination:true})` attribute emission matches ComputedBreak[].
- `paginate-with-chromium/__tests__/sentinel.test.ts` — sentinel injection for every paginable block.
- `paginate-with-chromium/__tests__/scrape.test.ts` — mock Paged.js DOM, assert bid→page mapping.
- `reconcile/__tests__/reconcile.test.ts` — all 4 branches (agree, ±1 drift, >1 drift, orphaned).
- `reconcile/__tests__/validator.test.ts` — 422 on collision/missing, warn on paginable-missing-bid.
- `docx-emitter/__tests__/hygiene.test.ts` — widowControl, settings, fontTable embedded.

### Integration tests

`apps/ck5-export/src/__tests__/pagination-e2e.test.ts`:
- Spin up Chromium pool with test fixture.
- Send 10-page canonical HTML with editor markers.
- Assert reconcile output matches golden, DOCX bytes match golden (parsed, not byte-exact — Carlito embed changes bytes but structure deterministic).
- Open generated DOCX with `docx4j` or `python-docx` in CI, assert page count exact.

### Golden fixtures

`apps/ck5-export/src/__fixtures__/pagination/*.html`:
1. `short-para.html` — 1 page, no breaks.
2. `long-para.html` — 3 pages, natural breaks only.
3. `heavy-table.html` — 5 pages, tr splits with repeated header.
4. `image-heavy.html` — 4 pages with figure break-inside: avoid.
5. `nested-lists.html` — 3 pages with indented lists.
6. `section-rich.html` — 6 pages with `mddmSection` break-before: page.
7. `repeatable-rows.html` — 8 pages of repeatable items.
8. `mixed-headings.html` — 4 pages with keep-with-next for headings.
9. `edge-widow.html` — widow/orphan boundary cases.
10. `100-page-contract.html` — stress test, scalability gate.

Each fixture + matching `.reconciled.json` golden.

### Block identity invariance matrix (CI)

`apps/ck5-export/src/__tests__/bid-invariance.test.ts`:
- Forced edit sequences (headless CK5 via jsdom — wiki 36 note: jsdom works for model ops, just not rendering):
  - List: bullet → ordered conversion, indent +3, outdent +3, split at midpoint, merge adjacent pairs.
  - Table: insert 5 rows, delete rows 2+4, merge cells (1,1)+(1,2), split merged, paste whole table from clipboard fixture.
  - Widget: insert section, add 3 repeatable items, drag-reorder items, delete middle, undo 5 times.
  - Paste: from same doc, from other CK5 doc (with bids), from Word clipboard HTML (no bids), from plain text.
- Assertion after each step:
  - Every paginable block has `mddmBid`.
  - No duplicate `mddmBid` values.
  - IDs that existed before a non-destructive edit still exist.
  - IDs of blocks that were merged (not survivors) are removed.

### Drift SLO gate

CI step: run full fixture set, compute drift stats, fail build if:
- Any `major-drift` (>1 block delta).
- More than 1 `minor-drift` per 50 pages per block type across all fixtures.
- Any `bid-collision` or `editor-server-desync`.

### Manual QA

Preview-mode E2E in `frontend/apps/web/tests/e2e/pagination.spec.ts`:
- Open template editor, type 10 pages of text, assert page counter reads "Page 10 of 10".
- Export DOCX, open in headless LibreOffice (`soffice --headless --convert-to pdf`), assert page count.
- Toggle dev `?debug=pagination`, assert overlay shows drift stats.

---

## Out of Scope

Explicitly deferred, tracked as follow-up tickets:

1. **Real-time collaboration pagination** — Codex-flagged concern about bid stability under concurrent edits. V1 uses optimistic locking (`lockVersion`), single-author-per-session. CK Cloud collab, track-changes, per-user cursor sharing all out of v1. When collab lands, revisit bid mint authority + conflict resolution.
2. **Track-changes export** — wiki 31 §4 parity matrix already marks this ❌ v1.
3. **Comments export** — same.
4. **Table of contents with page numbers** — needs `target-counter` (Paged.js supports, but TOC is a template-authoring feature we haven't designed yet).
5. **Headers/footers content** — CSS Paged Media already supports `@top-*`/`@bottom-*` margin boxes; populating them with document-specific content (logo, PO number) is template-engine territory, separate spec.
6. **Firefox/Safari full pagination support** — matches CKEditor paid restriction (Blink-only default). Firefox/Safari show A4 canvas but no live page-break overlays. Opt-in flag for later.
7. **Paged.js advanced features** — footnotes (`float: footnote`), multi-column layouts, named pages. Defer until a template requires them.
8. **PDF/A compliance** — Gotenberg flag investigation deferred.
9. **DOCX SDT / content controls for fields** — wiki 31 §5 marks `docx` lib SDT support as uncertain; defer.
10. **Autosave pagination refresh** — pagination overlay recomputes on edit, not on save; no server round-trip during edit. If a future feature needs saved-state pagination (e.g. shareable "go to page N" URLs), add it then.
11. **Inline image cropping/resizing affecting pagination** — v1 treats images as fixed-size after paste. Editor resize handle wiring to re-measure is a follow-up.
12. **Multi-document concurrent export** — Chromium pool handles, but no prioritization / fairness. Default FIFO v1; queue policy later.

---

## Open questions flagged for implementation phase

- Gotenberg's current API supports PDF rendering but may not expose `page.evaluate` for break scraping. Implementation decision: either extend Gotenberg (contribute upstream) or run a parallel Playwright worker pool dedicated to pagination. Default plan: dedicated Playwright worker, keep Gotenberg for final PDF bytes.
- Exact shape of `Font` config in the `docx` npm library — verify `embedRegular`/`embedBold`/etc. work on the pinned version; if not, embed via low-level XML override.
- Carlito TTF licensing for redistribution — OFL permits bundling; confirm we attribute correctly in `NOTICES`.
- Performance of BlockIdentityPlugin post-fixer on docs with 1000+ paginable blocks — profile before locking debounce defaults.

---

## Cross-references

- ADR 0001 — modular monolith
- ADR 0006 — deploy-v1 Compose single node
- ADR 0022 — browser document editor v1
- Wiki 03 — engine model (view.document.render hook)
- Wiki 11 — widgets (uiElement pattern, restricted editing composition)
- Wiki 25 — MDDM IR bridge (HTML as SoT, no parallel IR)
- Wiki 31 — DOCX export (existing emitter, `mddm-page-break` contract)
- Wiki 32 — PDF export (CSS Paged Media via Gotenberg)
- Wiki 34 — golden tests
- Wiki 40 — performance (this spec sets baselines where wiki was empty)
- CKEditor Pagination plugin docs (paid, reference only)
- ECMA-376 §17.3 (DOCX determinism recipe)
- Paged.js (MIT) — https://pagedjs.org
