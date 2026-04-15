# Template Page Margins And Auto Pagination Design

Date: 2026-04-14
Status: Proposed
Area: `frontend/apps/web/src/features/templates/` + `frontend/apps/web/src/features/documents/mddm-editor/`

## Summary

Add professional page-authoring controls to the template editor:

- template-level page margin controls for `top`, `right`, `bottom`, and `left`
- automatic visual page creation when content exceeds the writable height of the current page
- Word/Google Docs-like authoring feel with a single continuous editor model underneath

The editor will remain one BlockNote/ProseMirror document. Pagination will be visual and computed from the rendered layout, not by splitting the editor into multiple editors or true page DOM fragments.

## Goals

- Let authors set page margins in `mm`
- Apply margin settings consistently to screen editor, print CSS, and DOCX export
- Show additional pages automatically as content grows past the current page
- Keep cursor movement, selection, block insertion, and toolbar behavior stable
- Preserve existing template save/load APIs by using already-available `meta` payload storage

## Non-Goals

- Manual page-break insertion
- Header/footer editing
- Separate per-section page settings
- True DOM pagination with block splitting across page containers
- Exact Word parity for oversized blocks that are taller than one page

## User Experience

### Margin Controls

Page settings live in the right property sidebar as template-level controls. This is the most consistent placement for editor configuration because the sidebar already owns editable settings and keeps the author in-context while changing layout.

When no block is selected, the sidebar shows a `Page` settings panel instead of only an empty placeholder. When a block is selected, the `Page` settings card remains visible above the block-specific tabs so margin control stays accessible without forcing deselection.

Fields:

- `Top` margin (`mm`)
- `Right` margin (`mm`)
- `Bottom` margin (`mm`)
- `Left` margin (`mm`)

Validation:

- each value must stay in `5..50mm`
- computed writable width/height must remain above a minimum safe threshold
- invalid values are clamped in UI before persistence

### Automatic Pages

The editor presents a centered paper stack. Page 1 is always visible. As soon as content exceeds the writable range of the current page, the next page appears automatically below it, matching the expected Word-like interaction.

Typing, pressing `Enter`, inserting blocks, and editing nested content keep using one continuous editor. The page illusion comes from computed spacing and repeated paper surfaces, not from splitting the document model.

## Recommended Approach

Use a hybrid Word/Google Docs model:

- single continuous BlockNote document model
- visual pages in the editor
- automatic page appearance as content overflows
- shared page settings contract for screen and export

This is the best professional fit for the current stack because it keeps editing stable while still delivering page-based authoring UX. It avoids the fragility of true DOM pagination in ProseMirror/BlockNote.

## Data Model

Persist page settings under `draft.meta.page`.

```ts
type TemplatePageSettings = {
  marginTopMm: number;
  marginRightMm: number;
  marginBottomMm: number;
  marginLeftMm: number;
};
```

Stored shape:

```ts
type TemplateMeta = {
  page?: TemplatePageSettings;
  [key: string]: unknown;
};
```

Rules:

- if `meta.page` is absent, use `defaultLayoutTokens.page.*`
- existing drafts without page settings keep rendering exactly as today
- saving a draft includes current `meta` together with edited `blocks`

## Layout Token Integration

The editor already has a page token contract:

- `marginTopMm`
- `marginRightMm`
- `marginBottomMm`
- `marginLeftMm`

These values already flow into:

- DOCX page margins
- print stylesheet margins
- CSS custom properties

Required work is to merge persisted template `meta.page` values into runtime `LayoutTokens` before building editor CSS variables and export payloads.

## Editor Rendering Design

### Paper Geometry

Replace hardcoded editor paper padding with CSS variables derived from layout tokens:

- `padding-top: var(--mddm-margin-top)`
- `padding-right: var(--mddm-margin-right)`
- `padding-bottom: var(--mddm-margin-bottom)`
- `padding-left: var(--mddm-margin-left)`

This keeps visible writable area aligned with export margins.

### Visual Pagination

Add a page-layout manager inside the MDDM editor surface.

Responsibilities:

- measure rendered structural blocks in DOM order
- compute writable page height:
  - `pageHeightMm - marginTopMm - marginBottomMm`
- compute where each next page must start
- expose page count and per-page break offsets to the page stack renderer

Implementation strategy:

1. Observe the rendered editor DOM with `ResizeObserver` and mutation-safe recalculation hooks
2. Measure block top/bottom positions relative to paper content origin
3. When a block would cross the current writable page boundary:
   - move the next writable start to the next page
   - inject synthetic offset before that block via page-break CSS metadata
4. Render enough visual page surfaces for the calculated page count

The document itself remains continuous. Pagination metadata only affects visual spacing and paper rendering.

### Synthetic Break Mechanism

Use a lightweight page-break attribute/class on block wrappers, for example:

```html
<div class="bn-block-outer" data-page-break-before="132mm">
```

Then apply top spacing through CSS custom properties. This avoids rewriting editor content and keeps pagination logic external to document semantics.

### Oversized Blocks

If a single block is taller than one writable page:

- do not split the block in this phase
- allow it to continue through the page boundary
- still compute subsequent pages correctly below it

This is acceptable for the first professional version because true in-block splitting is materially more complex and risky.

## Sidebar Design

Enhance `PropertySidebar` with a template-level `Page` card.

Behavior:

- shown when no block is selected
- also shown above block tabs when a block is selected
- updates local template draft meta immediately in memory
- margin changes reflect in editor layout live without requiring manual save

Control style:

- numeric inputs with `mm` suffix
- small range hints
- compact explanatory note: margins affect both editor page and export

## Save And Load Flow

`useTemplateDraft` must preserve and persist `meta`:

- initial load reads `draft.meta`
- page settings edits update in-memory draft meta
- save/publish sends both `blocks` and `meta`

This work must not overwrite unrelated meta fields.

## Testing Strategy

### Unit Tests

- parse persisted `meta.page` settings into runtime layout tokens
- fallback to defaults when `meta.page` is absent or partial
- margin update helpers preserve unrelated `meta` keys
- pagination math returns expected page count and break offsets for known geometries

### Browser Tests

Real Chromium verification for the exact workflow:

- changing `Top/Right/Bottom/Left` updates visible paper padding
- only document pane scrolls
- long content creates a second page automatically
- page 2 appears when content exceeds page 1 writable area
- pressing `Enter` repeatedly on page 1 eventually shows next page
- first block on the next page aligns with that page's writable top margin

### Build Verification

- targeted Playwright spec for page settings + auto pages
- web build

## Risks

### Risk: pagination jitter during typing

Mitigation:

- debounce expensive measurement work into animation frame / micro-batched recalculation
- only write pagination metadata when computed offsets actually changed

### Risk: BlockNote wrapper DOM changes

Mitigation:

- target stable editor test ids and structural wrappers already exercised in existing visual tests
- add a focused regression test around pagination metadata placement

### Risk: hidden editor chrome offsets

Mitigation:

- keep DOM instrumentation in tests
- validate real computed geometry in browser, not only snapshot structure

## Acceptance Criteria

- Authors can set all four page margins from the template editor sidebar
- Margin settings persist after save/reload
- Visible paper padding changes immediately when margins change
- Long content shows additional pages automatically
- Editor remains one continuous authoring surface with stable cursor behavior
- Only the document pane scrolls
- Targeted Playwright verification and web build pass

## Implementation Notes

Preferred implementation sequence:

1. Introduce typed page-meta helpers and token merge
2. Add failing unit tests for margin persistence and token application
3. Add failing browser tests for margin controls and auto page appearance
4. Implement sidebar page controls
5. Implement editor token override from template meta
6. Implement pagination measurement + visual page rendering
7. Re-run browser and build verification

## Scope Decision

This spec intentionally chooses professional and reliable behavior over risky deep pagination. The result should feel like Word/Google Docs for authors while remaining technically safe in the current BlockNote-based architecture.
