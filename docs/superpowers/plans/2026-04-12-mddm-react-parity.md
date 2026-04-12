# MDDM React Parity Layer — Implementation Plan

**Spec:** `docs/superpowers/specs/2026-04-12-mddm-react-parity-design.md`
**Branch:** `feat/mddm-react-parity`
**Estimated tasks:** 45
**Phases:** 7 (includes contract-freeze gate after Phase 1)

---

## File Map

### Modified files
```
frontend/apps/web/src/features/documents/mddm-editor/
├── MDDMEditor.tsx                          (Phase 1: token injection)
├── MDDMEditor.module.css                   (Phase 1: editor root styling)
├── MDDMViewer.tsx                          (Phase 5: viewer logic)
├── mddm-editor-global.css                 (Phase 1: chrome + bridge CSS)
├── adapter.ts                             (Phase 2: DataTable migration)
├── schema.ts                              (Phase 2: remove DataTableRow/Cell, change DataTable content type)
├── blocks/
│   ├── DataTable.tsx                       (Phase 2: rewrite to content:"table")
│   ├── DataTable.module.css                (Phase 2: table styling)
│   ├── Repeatable.tsx                      (Phase 3: stale closure fix)
│   ├── Repeatable.module.css               (Phase 3: CSS reshaping)
│   ├── RepeatableItem.tsx                  (Phase 3: numbered headers)
│   ├── RepeatableItem.module.css           (Phase 3: accent border)
│   ├── Section.tsx                         (Phase 4: numbering)
│   ├── Section.module.css                  (Phase 4: token consumption)
│   ├── Field.tsx                           (Phase 4: minor)
│   ├── Field.module.css                    (Phase 4: token consumption)
│   ├── FieldGroup.module.css               (Phase 4: token consumption)
│   ├── RichBlock.module.css                (Phase 4: token consumption)
│   └── __tests__/data-table-columns.test.ts (Phase 2: remove or rewrite)
├── engine/
│   ├── layout-ir/
│   │   ├── token-bridge.ts                 (Phase 1: NEW)
│   │   └── __tests__/token-bridge.test.ts  (Phase 1: NEW)
│   ├── layout-interpreter/
│   │   ├── types.ts                        (Phase 3: NEW)
│   │   ├── section-interpreter.ts          (Phase 4: NEW)
│   │   ├── repeatable-interpreter.ts       (Phase 3: NEW)
│   │   ├── field-interpreter.ts            (Phase 4: NEW)
│   │   └── __tests__/*.test.ts             (Phase 3-4: NEW)
│   ├── docx-emitter/emitters/
│   │   ├── data-table.ts                   (Phase 2: update for tableContent)
│   │   └── (data-table-row.ts, data-table-cell.ts → DELETE)
│   ├── external-html/
│   │   ├── data-table-html.tsx             (Phase 2: NEW, replaces data-table-cell-html)
│   │   ├── repeatable-html.tsx             (Phase 3: NEW)
│   │   ├── repeatable-item-html.tsx        (Phase 3: NEW)
│   │   ├── rich-block-html.tsx             (Phase 4: NEW)
│   │   └── section-html.tsx               (Phase 4: update numbering)
│   ├── export/export-pdf.ts               (Phase 5: verify wiring)
│   ├── canonicalize-migrate/pipeline.ts   (Phase 2: mddm_version 1→2 migration)
│   ├── completeness-gate/
│   │   ├── block-registry.ts              (Phase 2: remove DataTableRow/Cell)
│   │   └── __tests__/completeness.test.ts (Phase 6: enforce in CI)
│   └── golden/fixtures/
│       ├── 02-complex-table/input.mddm.json    (Phase 2: update to tableContent)
│       ├── 02-complex-table/expected.document.xml (Phase 2: regenerate)
│       └── 07-repeatable-nested/               (Phase 6: NEW fixture)

frontend/apps/web/src/features/documents/browser-editor/
├── BrowserDocumentEditorView.tsx           (Phase 5: PDF button, viewer logic)

internal/modules/documents/domain/mddm/
├── po_template.go                         (Phase 2: dataTable to tableContent)
```

### Deleted files
```
frontend/apps/web/src/features/documents/mddm-editor/blocks/DataTableRow.tsx
frontend/apps/web/src/features/documents/mddm-editor/blocks/DataTableRow.module.css
frontend/apps/web/src/features/documents/mddm-editor/blocks/DataTableCell.tsx
frontend/apps/web/src/features/documents/mddm-editor/blocks/DataTableCell.module.css
frontend/apps/web/src/features/documents/mddm-editor/engine/docx-emitter/emitters/data-table-row.ts
frontend/apps/web/src/features/documents/mddm-editor/engine/docx-emitter/emitters/data-table-cell.ts
frontend/apps/web/src/features/documents/mddm-editor/engine/external-html/data-table-cell-html.tsx
frontend/apps/web/src/features/documents/mddm-editor/engine/external-html/__tests__/data-table-cell-html.test.tsx
```

### New files
```
frontend/apps/web/src/features/documents/mddm-editor/engine/layout-ir/token-bridge.ts
frontend/apps/web/src/features/documents/mddm-editor/engine/layout-ir/__tests__/token-bridge.test.ts
frontend/apps/web/src/features/documents/mddm-editor/engine/layout-interpreter/types.ts
frontend/apps/web/src/features/documents/mddm-editor/engine/layout-interpreter/section-interpreter.ts
frontend/apps/web/src/features/documents/mddm-editor/engine/layout-interpreter/repeatable-interpreter.ts
frontend/apps/web/src/features/documents/mddm-editor/engine/layout-interpreter/field-interpreter.ts
frontend/apps/web/src/features/documents/mddm-editor/engine/layout-interpreter/__tests__/section-interpreter.test.ts
frontend/apps/web/src/features/documents/mddm-editor/engine/layout-interpreter/__tests__/repeatable-interpreter.test.ts
frontend/apps/web/src/features/documents/mddm-editor/engine/layout-interpreter/__tests__/field-interpreter.test.ts
frontend/apps/web/src/features/documents/mddm-editor/engine/external-html/data-table-html.tsx
frontend/apps/web/src/features/documents/mddm-editor/engine/external-html/repeatable-html.tsx
frontend/apps/web/src/features/documents/mddm-editor/engine/external-html/repeatable-item-html.tsx
frontend/apps/web/src/features/documents/mddm-editor/engine/external-html/rich-block-html.tsx
frontend/apps/web/src/features/documents/mddm-editor/engine/golden/fixtures/07-repeatable-nested/input.mddm.json
frontend/apps/web/src/features/documents/mddm-editor/engine/golden/fixtures/07-repeatable-nested/expected.document.xml
migrations/0072_add_release_artifact_columns.sql
tests/visual-parity/parity.spec.ts
```

---

# Phase 1 — Foundation (Token Bridge + Chrome + Global CSS)

## Task 1: Create the token bridge function

**Files:**
- Create: `frontend/apps/web/src/features/documents/mddm-editor/engine/layout-ir/token-bridge.ts`
- Create: `frontend/apps/web/src/features/documents/mddm-editor/engine/layout-ir/__tests__/token-bridge.test.ts`

**Test first** — write a test that imports `tokensToCssVars` and asserts:
- It returns an object with keys starting with `--mddm-` and `--bn-`
- `--mddm-accent` equals `tokens.theme.accent`
- `--mddm-field-label-width` equals `"35%"`
- `--mddm-section-gap` equals `"6mm"`
- `--bn-font-family` contains `tokens.typography.editorFont`
- Changing `tokens.theme.accent` to `"#ff0000"` changes `--mddm-accent` to `"#ff0000"`

**Then implement** `tokensToCssVars(tokens: LayoutTokens): Record<string, string>`:
- Map all page tokens: `--mddm-page-width`, `--mddm-page-content-width`, margins
- Map all typography tokens: `--mddm-font-family`, `--mddm-font-size-base`, heading sizes, line height, label size
- Map all spacing tokens: `--mddm-section-gap`, `--mddm-field-gap`, `--mddm-block-gap`, `--mddm-cell-padding`
- Map all theme tokens: `--mddm-accent`, `--mddm-accent-light`, `--mddm-accent-dark`, `--mddm-accent-border`
- Map component rules: `--mddm-section-header-height: 8mm`, `--mddm-section-header-font-size: 13pt`, `--mddm-field-label-width: 35%`, `--mddm-field-value-width: 65%`, `--mddm-field-border-width: 0.5pt`, `--mddm-field-min-height: 7mm`
- Map BlockNote bridge: `--bn-font-family`, `--bn-border-radius: 4px`, `--bn-colors-side-menu: transparent`

Export from `layout-ir/index.ts`.

```bash
cd frontend/apps/web && npx vitest run src/features/documents/mddm-editor/engine/layout-ir/__tests__/token-bridge.test.ts
# Expected: all tests pass
```

---

## Task 2: Inject tokens into MDDMEditor

**Files:**
- Modify: `frontend/apps/web/src/features/documents/mddm-editor/MDDMEditor.tsx`

Import `tokensToCssVars` and `getDefaultTokens` (or `defaultLayoutTokens`). In the component:

1. Create a `useMemo` that builds tokens with template theme overrides:
```typescript
const tokens = useMemo(() => {
  const base = { ...defaultLayoutTokens };
  if (theme) {
    base.theme = { ...base.theme };
    if (theme.accent) base.theme.accent = theme.accent;
    if (theme.accentLight) base.theme.accentLight = theme.accentLight;
    if (theme.accentDark) base.theme.accentDark = theme.accentDark;
    if (theme.accentBorder) base.theme.accentBorder = theme.accentBorder;
  }
  return base;
}, [theme]);
```

2. Create a `useMemo` that calls `tokensToCssVars(tokens)`.

3. Spread the CSS vars object onto the `style` attribute of the editor root `<div>`.

4. Add `data-editable={!readOnly}` attribute to the root div for CSS targeting.

**Verify:** The editor still renders. Open DevTools → inspect the root div → confirm CSS variables are present.

---

## Task 3: Chrome visibility CSS

**Files:**
- Modify: `frontend/apps/web/src/features/documents/mddm-editor/mddm-editor-global.css`

Add the following rules at the end of the file:

```css
/* ── Chrome: hidden by default, visible on interaction ── */

.bn-container .bn-side-menu {
  opacity: 0;
  pointer-events: none;
  transition: opacity 150ms ease;
}

.bn-container .bn-block:hover > .bn-side-menu,
.bn-container .bn-block:focus-within > .bn-side-menu {
  opacity: 1;
  pointer-events: auto;
}

.bn-container .bn-drag-handle {
  opacity: 0;
  pointer-events: none;
  transition: opacity 150ms ease;
}

.bn-container .bn-block:hover .bn-drag-handle,
.bn-container .bn-block:focus-within .bn-drag-handle {
  opacity: 0.6;
  pointer-events: auto;
}

.bn-container .bn-drag-handle:hover {
  opacity: 1;
}

/* Remove nesting lines */
.bn-container .bn-block-group {
  border-left: none;
}

/* Spacing from tokens */
.bn-container .bn-block {
  margin-bottom: var(--mddm-block-gap, 2px);
}

.bn-container .bn-block-content[data-content-type="section"] {
  margin-top: var(--mddm-section-gap, 6mm);
}

/* Read-only: all chrome removed */
[data-editable="false"] .bn-side-menu,
[data-editable="false"] .bn-drag-handle,
[data-editable="false"] .bn-formatting-toolbar {
  display: none !important;
}

[data-editable="false"] .bn-inline-content {
  cursor: default;
}

[data-editable="false"] [data-mddm-block]:focus-within {
  outline: none;
}
```

**Verify:** Editor loads. No drag handles or + buttons visible. Hover over a block → chrome appears. Move away → chrome fades.

---

## Task 4: Update block CSS to consume tokens

**Files:**
- Modify: `frontend/apps/web/src/features/documents/mddm-editor/blocks/Section.module.css`
- Modify: `frontend/apps/web/src/features/documents/mddm-editor/blocks/Field.module.css`
- Modify: `frontend/apps/web/src/features/documents/mddm-editor/blocks/FieldGroup.module.css`
- Modify: `frontend/apps/web/src/features/documents/mddm-editor/blocks/RichBlock.module.css`

For each file, replace hardcoded values with `var()` references:

**Section.module.css** — key changes:
- `.sectionHeader` height → `var(--mddm-section-header-height, 8mm)`
- `.sectionHeader` background → `var(--mddm-accent)`
- `.sectionHeader` font-size → `var(--mddm-section-header-font-size, 13pt)`

**Field.module.css** — key changes:
- `.field` grid-template-columns → `var(--mddm-field-label-width, 35%) var(--mddm-field-value-width, 65%)`
- `.field` min-height → `var(--mddm-field-min-height, 7mm)`
- `.label` background → `var(--mddm-accent-light)`
- `.field` border → `var(--mddm-field-border-width, 0.5pt) solid var(--mddm-accent-border)`

**FieldGroup.module.css** — already mostly using tokens, verify gap uses `var(--mddm-field-gap)`.

**RichBlock.module.css** — key changes:
- `.label` background → `var(--mddm-accent-light)`
- border → `var(--mddm-accent-border)`

**Verify:** Editor renders. Sections have correct accent background. Fields show 35%/65% split. All borders use accent-border color.

---

## Task 5: Commit Phase 1

```bash
git add -A
git commit -m "feat(mddm): token bridge + chrome visibility + block CSS token consumption

- Add tokensToCssVars() bridge from Layout IR to CSS custom properties
- Inject tokens at runtime on MDDMEditor root
- Hide BlockNote chrome by default, show on hover/focus
- Remove nesting lines
- Update Section, Field, FieldGroup, RichBlock CSS to consume tokens

Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>"
```

---

# Phase 1.5 — Contract Freeze Gate

## Task 5.5: Lock shared interfaces before parallel phases

Before Phases 2, 3, and 4 start (they can run in parallel), freeze these contracts:

1. **Block registry shape** — confirm the final list of block types after DataTableRow/Cell removal:
   `section, fieldGroup, field, repeatable, repeatableItem, richBlock, dataTable, paragraph, heading, bulletListItem, numberedListItem, quote, image, divider`

2. **Adapter responsibilities** — the adapter handles:
   - Old DataTable children → tableContent conversion (on read)
   - BlockNote ↔ MDDM serialization for all blocks
   - No mddm_version bump yet — adapter converts internally, version stays at 1 until all phases stabilize

3. **toExternalHTML contract** — every implementation must:
   - Use `<table>` for layout (no flexbox/grid)
   - Use inline styles only
   - Import tokens directly (not CSS variables)
   - Include `data-mddm-block` attribute

4. **Layout interpreter extension point** — `InterpretContext` type is frozen. New interpreters must conform to the `BlockLayoutInterpreter<TBlock, TViewModel>` interface.

Document these in a brief comment block at the top of:
- `adapter.ts` (adapter responsibilities)
- `external-html/index.ts` (toExternalHTML contract)
- `layout-interpreter/types.ts` (interpreter contract)

This is a documentation task, not a code change. No commit needed — these comments are added alongside the code in subsequent phases.

---

# Phase 2 — DataTable Migration (content:"table")

## Task 6: Interaction spike — verify content:"table" works with createReactBlockSpec

**Files:**
- Modify: `frontend/apps/web/src/features/documents/mddm-editor/blocks/DataTable.tsx` (temporary test)

Before committing to the full migration, verify that `content: "table"` works with `createReactBlockSpec`:

1. Create a minimal test block:
```typescript
const TestTable = createReactBlockSpec(
  {
    type: "testTable",
    propSchema: { label: { default: "test" } },
    content: "table",
  },
  {
    render: (props) => (
      <div>
        <strong>{props.block.props.label}</strong>
        <div ref={props.contentRef} />
      </div>
    ),
  },
);
```

2. Register it temporarily in schema.ts.
3. Open the editor, add a testTable block via slash menu.
4. Verify: table renders with editable cells, Tab navigates between cells, typing works, column resize works.
5. If this works → proceed with full migration. If not → document the failure and use approach B (self-render children) as fallback.

**This is a gate.** If the spike fails, Tasks 7-14 change to the self-render approach described in the spec as a contingency.

---

## Task 7: Update adapter — DataTable migration on read

**Files:**
- Modify: `frontend/apps/web/src/features/documents/mddm-editor/adapter.ts`

Add a `migrateDataTableToTableContent` function:
- Detect old format: `block.type === "dataTable"` AND `block.children?.length > 0` AND `block.children[0]?.type === "dataTableRow"`
- Parse `columnsJson` from props to get column definitions
- Build header row: cells from column labels
- Build data rows: iterate `dataTableRow` children → for each row, iterate `dataTableCell` children → extract inline content per columnKey
- Return block with `content: { type: "tableContent", columnWidths: [...], headerRows: 1, rows: [...] }` and `children: []`
- Remove `columnsJson` from props, keep `label`, `locked`, `density`, `__template_block_id`

Update `toBlockNoteBlock`:
- For `dataTable` blocks, call `migrateDataTableToTableContent` if old format detected
- For new format, pass through directly

Update `toMDDMBlock`:
- For `dataTable` blocks, store `content` field (tableContent) in the MDDM output
- Remove children-based serialization for dataTable

**Test:** Write a unit test in adapter tests:
- Input: old-format DataTable with 2 columns and 3 rows
- Expected: new-format DataTable with tableContent, headerRows: 1, 4 total rows (1 header + 3 data)
- Verify roundtrip: toBlockNote → toMDDM preserves content

---

## Task 8: Rewrite DataTable component for content:"table"

**Files:**
- Modify: `frontend/apps/web/src/features/documents/mddm-editor/blocks/DataTable.tsx`
- Modify: `frontend/apps/web/src/features/documents/mddm-editor/blocks/DataTable.module.css`

Rewrite DataTable to use `content: "table"`:

```typescript
export const DataTable = createReactBlockSpec(
  {
    type: "dataTable",
    propSchema: {
      label: { default: "" },
      locked: { default: true },
      density: { default: "normal" },
      __template_block_id: { default: "" },
    },
    content: "table",
  },
  {
    render: (props) => (
      <div
        className={styles.dataTable}
        data-mddm-block="dataTable"
        data-density={props.block.props.density || "normal"}
        data-locked={props.block.props.locked}
      >
        <div className={styles.dataTableHeader}>
          <strong className={styles.tableLabel}>
            {props.block.props.label || "Data Table"}
          </strong>
        </div>
        <div className={styles.tableContainer} ref={props.contentRef} />
      </div>
    ),
  },
);
```

Remove `parseDataTableColumns` function (no longer needed).

Update `DataTable.module.css`:
- Keep `.dataTable` wrapper styles (border, border-radius, background)
- Keep `.dataTableHeader` styles
- Remove `.tableGrid`, `.tableHeaderCell`, `.addRowButton` (BlockNote handles these natively)
- Add `.tableContainer` styles for proper fit within the wrapper

---

## Task 9: Update schema — remove DataTableRow/DataTableCell

**Files:**
- Modify: `frontend/apps/web/src/features/documents/mddm-editor/schema.ts`

1. Remove imports for `DataTableRow` and `DataTableCell`
2. Remove their entries from the `blockSpecs` object
3. Update DataTable's entry to use the new component

---

## Task 10: Delete DataTableRow and DataTableCell files

**Files:**
- Delete: `frontend/apps/web/src/features/documents/mddm-editor/blocks/DataTableRow.tsx`
- Delete: `frontend/apps/web/src/features/documents/mddm-editor/blocks/DataTableRow.module.css`
- Delete: `frontend/apps/web/src/features/documents/mddm-editor/blocks/DataTableCell.tsx`
- Delete: `frontend/apps/web/src/features/documents/mddm-editor/blocks/DataTableCell.module.css`
- Delete: `frontend/apps/web/src/features/documents/mddm-editor/engine/external-html/data-table-cell-html.tsx`
- Delete: `frontend/apps/web/src/features/documents/mddm-editor/engine/external-html/__tests__/data-table-cell-html.test.tsx`

Also remove exports from `external-html/index.ts` if they reference DataTableCell.

---

## Task 11: Update DataTable DOCX emitter for tableContent

**Files:**
- Modify: `frontend/apps/web/src/features/documents/mddm-editor/engine/docx-emitter/emitters/data-table.ts`
- Delete: `frontend/apps/web/src/features/documents/mddm-editor/engine/docx-emitter/emitters/data-table-row.ts`
- Delete: `frontend/apps/web/src/features/documents/mddm-editor/engine/docx-emitter/emitters/data-table-cell.ts`
- Modify: `frontend/apps/web/src/features/documents/mddm-editor/engine/docx-emitter/emitter.ts` (remove row/cell emitter registrations)

Rewrite `data-table.ts`:
- Read `block.content.rows` instead of `block.children`
- First row (if `headerRows >= 1`) gets header styling (accent-light background, bold)
- Data rows get standard cell styling
- Column widths from `block.content.columnWidths`
- Cell borders from `tokens.theme.accentBorder`

Update emitter registry in `emitter.ts`:
- Remove `dataTableRow` and `dataTableCell` entries
- Update `dataTable` to use the new emitter

**Test:** Run existing golden test `02-complex-table` — it will fail because the input fixture needs updating (Task 14).

---

## Task 12: Create DataTable toExternalHTML

**Files:**
- Create: `frontend/apps/web/src/features/documents/mddm-editor/engine/external-html/data-table-html.tsx`
- Modify: `frontend/apps/web/src/features/documents/mddm-editor/engine/external-html/index.ts`

Implement `DataTableExternalHTML`:
- Render as `<table>` with inline styles
- Header row: accent-light background, bold font
- Data rows: standard cells with borders
- All styling from direct token import (not CSS variables)
- `data-mddm-block="dataTable"` attribute

Wire into the DataTable block spec's `toExternalHTML` option.

---

## Task 13: Update Go template for tableContent

**Files:**
- Modify: `internal/modules/documents/domain/mddm/po_template.go`

Rewrite `dataTableBlock()`:
- Output `"content"` field with `"type": "tableContent"`, `"columnWidths"`, `"headerRows": 1`, `"rows"`
- Build header row from column definitions
- Build data rows from previous row data
- Remove `"children"` (set to empty array)
- Remove `dataTableRowBlock()` and `dataTableCellBlock()` helper functions

---

## Task 14: Update golden fixture 02-complex-table

**Files:**
- Modify: `frontend/apps/web/src/features/documents/mddm-editor/engine/golden/fixtures/02-complex-table/input.mddm.json`
- Regenerate: `frontend/apps/web/src/features/documents/mddm-editor/engine/golden/fixtures/02-complex-table/expected.document.xml`

Update `input.mddm.json` to use `tableContent` format instead of children format. Set `mddm_version: 2`.

Regenerate `expected.document.xml` by running the golden test in update mode:
```bash
cd frontend/apps/web && GOLDEN_UPDATE=1 npx vitest run src/features/documents/mddm-editor/engine/golden/__tests__/golden-02-complex-table.test.ts
```

Review the generated XML to confirm proper table structure.

---

## Task 15: DataTable migration in adapter (no version bump yet)

**Files:**
- Modify: `frontend/apps/web/src/features/documents/mddm-editor/adapter.ts`

The adapter handles old→new DataTable format conversion internally on read WITHOUT bumping `mddm_version`. The version bump (1→2) is deferred to Phase 6 (Task 37.5) after all phases stabilize, so v2 represents the full parity schema — not just DataTable.

The adapter's `toBlockNoteBlock` function detects old-format DataTable blocks (with `dataTableRow` children) and converts them to `tableContent` format. On save (`toMDDMBlock`), the new tableContent format is persisted. Old documents gradually migrate as they're opened and saved.

No changes to `canonicalize-migrate/pipeline.ts` in this phase.

---

## Task 16: Update block registry — remove DataTableRow/Cell

**Files:**
- Modify: `frontend/apps/web/src/features/documents/mddm-editor/engine/completeness-gate/block-registry.ts`

Remove `dataTableRow` and `dataTableCell` entries from `BLOCK_REGISTRY`.

---

## Task 17: Commit Phase 2

```bash
git add -A
git commit -m "feat(mddm): migrate DataTable to native content:table

- DataTable uses BlockNote's native TableContent (proper grid, cell editing, resize)
- Remove DataTableRow and DataTableCell block types
- Adapter auto-migrates old children format to tableContent on read
- DOCX emitter reads content.rows instead of block.children
- New toExternalHTML for DataTable (table-based, inline styles)
- Go template updated to produce tableContent format
- Golden fixture 02-complex-table updated
- mddm_version bumped 1→2 with registered migration

Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>"
```

---

# Phase 3 — Repeatable Polish

## Task 18: Create layout interpreter types

**Files:**
- Create: `frontend/apps/web/src/features/documents/mddm-editor/engine/layout-interpreter/types.ts`

Define:
```typescript
export interface BlockLayoutInterpreter<TBlock, TViewModel> {
  interpret(block: TBlock, tokens: LayoutTokens, context: InterpretContext): TViewModel;
}

export type InterpretContext = {
  depth: number;
  sectionIndex?: number;
  parentNumber?: string;
  parentType?: string;
};
```

---

## Task 19: Create RepeatableInterpreter

**Files:**
- Create: `frontend/apps/web/src/features/documents/mddm-editor/engine/layout-interpreter/repeatable-interpreter.ts`
- Create: `frontend/apps/web/src/features/documents/mddm-editor/engine/layout-interpreter/__tests__/repeatable-interpreter.test.ts`

**Test first:**
- Input: Repeatable with 3 RepeatableItem children, parentNumber "4"
- Expected: `items[0].number = 1`, `items[0].displayNumber = "4.1"`, `items[1].displayNumber = "4.2"`, `items[2].displayNumber = "4.3"`
- `canAddItem = true` when `!locked && items.length < maxItems`
- `canAddItem = false` when `locked` or `items.length >= maxItems`

**Implement** `RepeatableInterpreter`:
- Read `block.props.label`, `block.props.itemPrefix`, `block.props.locked`, `block.props.minItems`, `block.props.maxItems`
- Map `block.children` filtered to `type === "repeatableItem"` → items with index-based numbering
- Each item gets `displayNumber` from `context.parentNumber + "." + (index + 1)`
- Compute `canAddItem`

---

## Task 20: Fix Repeatable stale closure bug + use interpreter

**Files:**
- Modify: `frontend/apps/web/src/features/documents/mddm-editor/blocks/Repeatable.tsx`

1. Import `RepeatableInterpreter`
2. In the `onClick` handler, read current children from editor state:
```typescript
onClick={() => {
  const currentBlock = props.editor.getBlock(props.block.id);
  const currentChildren = currentBlock?.children ?? [];
  props.editor.updateBlock(props.block, {
    children: [...currentChildren, newItem],
  });
}}
```
3. Use interpreter to compute `canAddItem` for conditional button rendering
4. Pass `context.parentNumber` for display (if available via props or data attribute)

---

## Task 21: Repeatable CSS reshaping

**Files:**
- Modify: `frontend/apps/web/src/features/documents/mddm-editor/blocks/Repeatable.module.css`
- Modify: `frontend/apps/web/src/features/documents/mddm-editor/mddm-editor-global.css`

**Repeatable.module.css:**
- `.repeatable` → border, border-radius, overflow hidden, background white
- `.repeatableHeader` → accent-light background, padding, font-weight
- `.addItemButton` → dashed border top, full width, accent color on hover

**Global CSS additions** — pull RepeatableItem children inside the container:
```css
.bn-container .react-renderer.node-repeatable + .bn-block-group {
  margin-left: 0;
  padding: 0 var(--mddm-spacing-sm);
  padding-bottom: var(--mddm-spacing-sm);
}
```

---

## Task 22: RepeatableItem numbered headers + accent border

**Files:**
- Modify: `frontend/apps/web/src/features/documents/mddm-editor/blocks/RepeatableItem.tsx`
- Modify: `frontend/apps/web/src/features/documents/mddm-editor/blocks/RepeatableItem.module.css`

**RepeatableItem.tsx:**
- Display numbered title: use index from parent's children array
- Render title bar with number and title text

**RepeatableItem.module.css:**
- `.repeatableItem` → left border accent bar (3pt solid accent), margin-bottom
- `.itemHeader` → bold title with number, padding, accent-dark color
- Children render below via BlockNote's `.bn-block-group`

Global CSS addition:
```css
.bn-container .react-renderer.node-repeatableItem + .bn-block-group {
  margin-left: 0;
  padding-left: var(--mddm-cell-padding);
  border-left: 3pt solid var(--mddm-accent);
}
```

---

## Task 23: Create Repeatable + RepeatableItem toExternalHTML

**Files:**
- Create: `frontend/apps/web/src/features/documents/mddm-editor/engine/external-html/repeatable-html.tsx`
- Create: `frontend/apps/web/src/features/documents/mddm-editor/engine/external-html/repeatable-item-html.tsx`
- Modify: `frontend/apps/web/src/features/documents/mddm-editor/engine/external-html/index.ts`

**repeatable-html.tsx:**
- `<table>` with header row (label, accent-light bg)
- Children rendered below in a content cell

**repeatable-item-html.tsx:**
- `<table>` with left border accent bar (3pt)
- Title row with number + title
- Content cell with child blocks

Wire both into their block specs' `toExternalHTML`.

---

## Task 24: Commit Phase 3

```bash
git add -A
git commit -m "feat(mddm): repeatable polish — CSS reshaping, numbering, stale closure fix

- Fix stale closure bug (read editor state at click time)
- RepeatableInterpreter for item numbering and add-item eligibility
- CSS reshaping pulls items inside container visually
- RepeatableItem gets numbered header with accent border
- toExternalHTML for Repeatable and RepeatableItem
- Global CSS bridge for Repeatable/RepeatableItem block-group

Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>"
```

---

# Phase 4 — Section Numbering + Structural Block Polish

## Task 25: Create SectionInterpreter

**Files:**
- Create: `frontend/apps/web/src/features/documents/mddm-editor/engine/layout-interpreter/section-interpreter.ts`
- Create: `frontend/apps/web/src/features/documents/mddm-editor/engine/layout-interpreter/__tests__/section-interpreter.test.ts`

**Test first:**
- Input: list of 4 section blocks
- Expected: numbers "1", "2", "3", "4"
- Resolves header styling from tokens: bg = accent, color = white, height = 8mm, fontSize = 13pt

**Implement** `SectionInterpreter`:
- `interpret(block, tokens, context)` → `SectionViewModel`
- Number from `context.sectionIndex`
- Header styling from `tokens.theme` and `tokens.components.section`

---

## Task 26: Update Section component — auto-numbering

**Files:**
- Modify: `frontend/apps/web/src/features/documents/mddm-editor/blocks/Section.tsx`

Currently the title is just `props.block.props.title`. Add section numbering:
- The section needs to know its index. Two options:
  - Option A: Compute from the editor's document blocks at render time
  - Option B: Store the number as a computed data attribute

Use Option A: read `props.editor.document` to find this section's index among all top-level section blocks.

Display: `{sectionNumber}. {title}` → "1. Identificação do Processo"

---

## Task 27: Update Section toExternalHTML with numbering

**Files:**
- Modify: `frontend/apps/web/src/features/documents/mddm-editor/engine/external-html/section-html.tsx`

Add the section number to the HTML output. The toExternalHTML receives the block and editor — compute the section index the same way as the render function.

---

## Task 28: Create FieldInterpreter

**Files:**
- Create: `frontend/apps/web/src/features/documents/mddm-editor/engine/layout-interpreter/field-interpreter.ts`
- Create: `frontend/apps/web/src/features/documents/mddm-editor/engine/layout-interpreter/__tests__/field-interpreter.test.ts`

**Test first:**
- Input: Field block with `valueMode: "inline"`
- Expected: `labelWidthPct: 35`, `valueWidthPct: 65`, `labelBg` from tokens, `borderColor` from tokens

Simple interpreter — mostly resolves token values for the Field component.

---

## Task 29: Create RichBlock toExternalHTML

**Files:**
- Create: `frontend/apps/web/src/features/documents/mddm-editor/engine/external-html/rich-block-html.tsx`
- Modify: `frontend/apps/web/src/features/documents/mddm-editor/engine/external-html/index.ts`

**rich-block-html.tsx:**
- `<table>` with optional label row (accent-light bg, bold)
- Content cell with child blocks
- All inline styles from direct token import

Wire into RichBlock's `toExternalHTML`.

---

## Task 30: Commit Phase 4

```bash
git add -A
git commit -m "feat(mddm): section auto-numbering + field/richBlock polish

- SectionInterpreter computes sequential section numbers
- Section component displays numbered titles (1. Title, 2. Title)
- Section toExternalHTML includes numbering
- FieldInterpreter resolves token values
- RichBlock toExternalHTML implementation
- All structural blocks consume Layout IR tokens via CSS variables

Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>"
```

---

# Phase 5 — Viewer + Export Flow + Release Determinism

## Task 31: MDDMViewer for released/archived documents

**Files:**
- Modify: `frontend/apps/web/src/features/documents/mddm-editor/MDDMViewer.tsx`

Ensure MDDMViewer passes `readOnly={true}` to MDDMEditor. If it already does this, verify it works with the new `data-editable="false"` attribute and chrome CSS.

---

## Task 32: Add PDF export button to BrowserDocumentEditorView

**Files:**
- Modify: `frontend/apps/web/src/features/documents/browser-editor/BrowserDocumentEditorView.tsx`

1. Add `isExportingPdf` state
2. Add `handleExportPdf` function:
   - Parse envelope from editorData
   - Call `exportPdf()` from the export module
   - Trigger blob download with `.pdf` extension
   - Error handling: 503 → "Serviço de PDF indisponível. Tente exportar em DOCX."
3. Add "Exportar PDF" button in the actions div, before "Exportar DOCX"
4. Both buttons disabled during any export operation

---

## Task 33: Document status → viewer logic

**Files:**
- Modify: `frontend/apps/web/src/features/documents/browser-editor/BrowserDocumentEditorView.tsx`

Add status-based rendering:
```typescript
const isViewOnly = document.status === "RELEASED" || document.status === "ARCHIVED";
const isPendingApproval = document.status === "PENDING_APPROVAL";
```

- `isViewOnly` → render `<MDDMViewer>` instead of `<MDDMEditor>`, hide "Salvar rascunho" button
- `isPendingApproval` → render `<MDDMEditor readOnly>`, hide "Salvar rascunho"
- DRAFT → current behavior (editor, save button)

---

## Task 34: Keep native DOCX at 0% rollout for now

Native DOCX export stays at 0% rollout during development. The 100% cutover is Task 43.5 in Phase 7 — AFTER all testing passes. This prevents shipping a format/export change before validation.

---

## Task 35: Release determinism — DB migration for artifact storage

**Files:**
- Create: `migrations/0072_add_release_artifact_columns.sql`

```sql
ALTER TABLE metaldocs.document_versions
  ADD COLUMN IF NOT EXISTS release_artifact_key TEXT,
  ADD COLUMN IF NOT EXISTS canonical_mddm_snapshot JSONB;

COMMENT ON COLUMN metaldocs.document_versions.release_artifact_key IS
  'Storage key for the immutable DOCX artifact generated at release time';
COMMENT ON COLUMN metaldocs.document_versions.canonical_mddm_snapshot IS
  'Frozen MDDM envelope JSON captured at release time (post-migration, post-canonicalization)';
```

---

## Task 36: Commit Phase 5

```bash
git add -A
git commit -m "feat(mddm): viewer + PDF export + native DOCX + release artifact schema

- MDDMViewer for RELEASED/ARCHIVED documents (readOnly)
- PDF export button (Gotenberg HTML→Chromium→PDF)
- Document status drives editor vs viewer rendering
- Native DOCX export enabled at 100% rollout
- DB migration for release_artifact_key and canonical_mddm_snapshot

Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>"
```

---

# Phase 6 — Testing + Validation

## Task 37: Update all golden fixtures for mddm_version 2

**Files:**
- Modify: all `input.mddm.json` files in `engine/golden/fixtures/` that contain DataTable blocks
- Regenerate corresponding `expected.document.xml` files

Set `mddm_version: 2` on all fixtures. Convert any DataTable blocks from children format to tableContent format.

Run golden tests:
```bash
cd frontend/apps/web && npx vitest run src/features/documents/mddm-editor/engine/golden/
# Expected: all golden tests pass
```

---

## Task 38: Create golden fixture 07-repeatable-nested

**Files:**
- Create: `frontend/apps/web/src/features/documents/mddm-editor/engine/golden/fixtures/07-repeatable-nested/input.mddm.json`
- Create: `frontend/apps/web/src/features/documents/mddm-editor/engine/golden/fixtures/07-repeatable-nested/expected.document.xml`

Input fixture: Repeatable block with 2 RepeatableItems, each containing:
- A paragraph block
- A DataTable (tableContent format) with 2 columns and 2 rows
- A bulletListItem block

Generate expected XML:
```bash
cd frontend/apps/web && GOLDEN_UPDATE=1 npx vitest run src/features/documents/mddm-editor/engine/golden/__tests__/golden-runner.test.ts
```

Review generated XML manually.

---

## Task 39: Enforce completeness gate in CI

**Files:**
- Modify: `frontend/apps/web/src/features/documents/mddm-editor/engine/completeness-gate/__tests__/completeness.test.ts`

Update the test to verify for every block type in the registry:
1. `render()` is defined
2. `toExternalHTML()` is defined
3. DOCX emitter is registered
4. At least one golden fixture exercises the block

This test runs as part of the regular vitest suite. Any PR missing a renderer blocks the merge.

---

## Task 40: Delete old documents and recreate from template

Connect to the database and delete all existing PO documents:
```sql
DELETE FROM metaldocs.document_versions;
DELETE FROM metaldocs.documents;
```

Apply the new migration (0072). Restart the API to re-seed the template with the updated `po_template.go`.

Create a new PO document from the template to verify everything works end-to-end.

---

## Task 41: Run vitest full suite

```bash
cd frontend/apps/web && npx vitest run
# Expected: all tests pass including golden tests, completeness gate, token bridge
```

Fix any failures before proceeding to E2E.

---

## Task 42: E2E browser testing (Claude in Chrome)

Execute the 46-test matrix from the spec:
- Tests 1-5: Document creation & template
- Tests 6-12: DataTable native table
- Tests 13-18: Repeatable
- Tests 19-23: Editor chrome
- Tests 24-27: Save flow
- Tests 28-33: DOCX export
- Tests 34-38: PDF export
- Tests 39-43: Document status & viewer
- Tests 44-46: Shadow testing & telemetry

Screenshot each test result. All critical/major tests must pass.

---

---

# Phase 7 — Version Bump + Rollout Cutover

## Task 43.5: Register mddm_version 1→2 migration (full parity schema)

**Files:**
- Modify: `frontend/apps/web/src/features/documents/mddm-editor/engine/canonicalize-migrate/pipeline.ts`

Now that all phases are stable and tested:

1. Change `CURRENT_MDDM_VERSION` from `1` to `2`
2. Add migration `1 → 2` in the `MIGRATIONS` map:
```typescript
const MIGRATIONS: Record<number, Migration> = {
  1: (envelope) => {
    const migratedBlocks = envelope.blocks.map(block => migrateBlockRecursive(block));
    return { ...envelope, mddm_version: 2, blocks: migratedBlocks };
  },
};
```

Version 2 represents the full parity schema: DataTable with tableContent, all toExternalHTML implementations, section numbering, Layout IR token consumption.

Update all golden fixture `input.mddm.json` files to `mddm_version: 2`.

Re-run golden tests:
```bash
cd frontend/apps/web && npx vitest run src/features/documents/mddm-editor/engine/golden/
```

---

## Task 43.6: Enable native DOCX export at 100% rollout

**Files:**
- Modify: `frontend/apps/web/src/features/featureFlags.ts` (or wherever `MDDM_NATIVE_EXPORT_ROLLOUT_PCT` is configured)

Set rollout to 100%. All users now get client-side DOCX via docx.js.

**Verify:** Click "Exportar DOCX" → no network call to `/export/docx` → file downloads from client-side generation.

---

## Task 44: Final commit + branch ready for PR

```bash
git add -A
git commit -m "feat(mddm): bump mddm_version to 2, enable native DOCX at 100%

- Register mddm_version 1→2 migration (full parity schema)
- Enable native DOCX export at 100% rollout
- All golden fixtures at mddm_version 2
- Full vitest suite passes
- E2E browser tests validated

Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>"
```

---

# Execution Handoff

This plan has 45 tasks across 7 phases. Recommended execution:

1. **Subagent-Driven Development** (recommended) — one subagent per task, spec compliance review after each, code quality review after each phase
2. **Inline Execution** — execute tasks sequentially in the current session

Phase dependencies:
```
Phase 1 (Foundation) → Phase 1.5 (Contract Freeze) → Phases 2, 3, 4 (parallel OK)
                                                          ↓
                                                     Phase 5 (Viewer + Export)
                                                          ↓
                                                     Phase 6 (Testing)
                                                          ↓
                                                     Phase 7 (Version Bump + Rollout)
```

- Phase 1 must complete first (token bridge is consumed by all later phases)
- Phase 1.5 freezes contracts before parallel work begins
- Phases 2, 3, 4 can run in parallel after Phase 1.5 (DataTable, Repeatable, Section are independent)
- Phase 5 depends on Phases 2-4 (viewer needs blocks working)
- Phase 6 depends on all prior phases (testing validates everything)
- Phase 7 depends on Phase 6 passing (version bump + rollout only after validation)
