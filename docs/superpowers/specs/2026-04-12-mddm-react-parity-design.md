# MDDM React Parity Layer Design

## Goal

Close the visual parity gap between the React editor and the DOCX/PDF exports. The MDDM engine's DOCX emitters, Layout IR tokens, golden tests, shadow testing, and version pinning are production-grade. But the React block components don't consume the same tokens, don't render children inline, and don't match the DOCX output structurally. This design makes the editor look like the export and the export look like the editor — within formal tolerance thresholds.

### Success criteria

- Editor-to-PDF pixel diff: **< 2%** (enforced by Playwright visual parity tests)
- Editor-to-DOCX pixel diff: **< 5%** (enforced by Playwright visual parity tests)
- All 9 custom MDDM blocks have: `render()` + `toExternalHTML()` + DOCX emitter + golden fixture (enforced by completeness gate CI)
- DataTable renders as a proper interactive table with borders, cell editing, and column resize
- Repeatable items render visually inside the container with hierarchical numbering
- Editor chrome (drag handles, nesting lines, side menu) hidden by default, visible on hover/focus
- Released documents serve a stored DOCX artifact (never re-rendered)
- Two export buttons: "Exportar DOCX" (client-side) and "Exportar PDF" (Gotenberg)
- BlockNote v0.47.3 — no editor framework change

### Non-goals

- Dark mode support
- Landscape / custom page sizes
- Embedded fonts in DOCX
- Real-time collaboration
- Server-side DOCX generation
- Batch/bulk export
- ODT/RTF export formats
- Drag-to-reorder Repeatable items
- Offline PDF generation

## Architecture

### Three-Layer Rendering Stack

```
┌─────────────────────────────────────────────────────┐
│              MDDM Block (data model)                │
│         props, content, children                     │
└──────────────────────┬──────────────────────────────┘
                       │
                       ▼
┌─────────────────────────────────────────────────────┐
│           Layout Interpreter (per block type)        │
│                                                     │
│  Reads block + Layout IR tokens → ViewModel          │
│  Handles section numbering, item counting,           │
│  width resolution, style computation                 │
│                                                     │
│  One interpreter per MDDM block type                 │
└──────────┬─────────────────────┬────────────────────┘
           │                     │
           ▼                     ▼
┌──────────────────┐  ┌──────────────────┐
│  React Emitter   │  │  DOCX Emitter    │
│  (JSX output)    │  │  (docx.js output)│
│                  │  │                  │
│  Reads ViewModel │  │  Reads ViewModel │
│  + CSS variables │  │  + OOXML values  │
│  from Layout IR  │  │  from Layout IR  │
└──────────────────┘  └──────────────────┘
```

### Token Flow

```
Layout IR (tokens.ts) — SINGLE SOURCE OF TRUTH
    │
    ├──→ tokensToCssVars(tokens)
    │       Injected as CSS custom properties on .mddm-editor-root
    │       --mddm-section-gap: 6mm
    │       --mddm-field-label-width: 35%
    │       --mddm-accent: #6b1f2a
    │       + template theme overrides
    │       + BlockNote bridge: --bn-font-family, --bn-border-radius
    │
    ├──→ tokensToDocx(tokens)
    │       Used by DOCX emitters (already working)
    │       mmToTwip(6) → section gap
    │       35% of contentWidth → field label width
    │
    └──→ Direct import in toExternalHTML
            Used for PDF pipeline (no CSS variables in isolated React root)
            tokens.theme.accent → inline style
```

### Block Rendering Categories

| Category | Blocks | Rendering Strategy |
|---|---|---|
| **Container** | DataTable | `content: "table"` — BlockNote native table rendering. Custom chrome wrapper for label/density. |
| **Container** | Repeatable, RepeatableItem | `content: "none"` — CSS reshaping of `.bn-block-group` to pull children inside container visually. Numbered headers. |
| **Structural** | Section, FieldGroup, Field, RichBlock | Keep BlockNote child rendering. Consume Layout IR tokens for exact measurements via CSS variables. |
| **Content** | paragraph, heading, bulletListItem, numberedListItem, quote, image, divider | Keep BlockNote native rendering. Chrome cleanup + token-driven spacing only. |

## Components

### 1. Layout Interpreter

One interpreter per MDDM block type. Takes a raw block + Layout IR tokens + context and produces a ViewModel — a plain object describing what to render without any rendering technology.

```typescript
// layout-interpreter/types.ts

export interface BlockLayoutInterpreter<TBlock, TViewModel> {
  interpret(block: TBlock, tokens: LayoutTokens, context: InterpretContext): TViewModel;
}

export type InterpretContext = {
  depth: number;
  sectionIndex?: number;
  parentType?: string;
};
```

#### SectionInterpreter

Computes sequential numbering from document position. Sub-items get hierarchical numbers via parent context:

```
Section (sectionIndex=1) → number: "1"
Section (sectionIndex=2) → number: "2"
  RepeatableItem[0] → sub-number: "2.1"
  RepeatableItem[1] → sub-number: "2.2"
Section (sectionIndex=3) → number: "3"
```

Produces `SectionViewModel`:
- `number: string` — "1", "2", "3"
- `title: string`
- `headerHeight: string` — "8mm"
- `headerBg: string` — resolved from tokens.theme.accent
- `headerColor: string` — "#ffffff"
- `headerFontSize: string` — "13pt"
- `optional: boolean`
- `locked: boolean`
- `childBlocks: MDDMBlock[]`

#### RepeatableInterpreter

Computes item numbering and add-item eligibility:

- `label: string`
- `itemPrefix: string`
- `items: RepeatableItemViewModel[]` — each with `title`, `number` (1-based), `accentBorder` color
- `locked: boolean`
- `canAddItem: boolean` — `!locked && items.length < maxItems`

#### FieldInterpreter

Resolves label width and value mode:

- `label: string`
- `labelWidthPct: number` — 35
- `valueWidthPct: number` — 65
- `labelBg: string`
- `borderColor: string`
- `minHeight: string` — "7mm"
- `valueMode: "inline" | "multiParagraph"`
- `locked: boolean`

DataTable does NOT need an interpreter — BlockNote's native TableContent handles the structure.

### 2. Token Bridge

Converts Layout IR tokens to CSS custom properties:

```typescript
// layout-ir/token-bridge.ts

export function tokensToCssVars(tokens: LayoutTokens): Record<string, string> {
  return {
    // Page
    '--mddm-page-width': `${tokens.page.widthMm}mm`,
    '--mddm-page-content-width': `${tokens.page.contentWidthMm}mm`,
    '--mddm-page-margin-top': `${tokens.page.marginTop}mm`,
    '--mddm-page-margin-right': `${tokens.page.marginRight}mm`,
    '--mddm-page-margin-bottom': `${tokens.page.marginBottom}mm`,
    '--mddm-page-margin-left': `${tokens.page.marginLeft}mm`,

    // Typography
    '--mddm-font-family': tokens.typography.editorFont,
    '--mddm-font-size-base': `${tokens.typography.baseSizePt}pt`,
    '--mddm-font-size-h1': `${tokens.typography.headingSizePt.h1}pt`,
    '--mddm-font-size-h2': `${tokens.typography.headingSizePt.h2}pt`,
    '--mddm-font-size-h3': `${tokens.typography.headingSizePt.h3}pt`,
    '--mddm-line-height': `${tokens.typography.lineHeightPt}pt`,
    '--mddm-font-size-label': `${tokens.typography.labelSizePt}pt`,

    // Spacing
    '--mddm-section-gap': `${tokens.spacing.sectionGapMm}mm`,
    '--mddm-field-gap': `${tokens.spacing.fieldGapMm}mm`,
    '--mddm-block-gap': `${tokens.spacing.blockGapMm}mm`,
    '--mddm-cell-padding': `${tokens.spacing.cellPaddingMm}mm`,

    // Theme
    '--mddm-accent': tokens.theme.accent,
    '--mddm-accent-light': tokens.theme.accentLight,
    '--mddm-accent-dark': tokens.theme.accentDark,
    '--mddm-accent-border': tokens.theme.accentBorder,

    // Component rules
    '--mddm-section-header-height': '8mm',
    '--mddm-section-header-font-size': '13pt',
    '--mddm-field-label-width': '35%',
    '--mddm-field-value-width': '65%',
    '--mddm-field-border-width': '0.5pt',
    '--mddm-field-min-height': '7mm',

    // BlockNote bridge
    '--bn-font-family': `"${tokens.typography.editorFont}", -apple-system, sans-serif`,
    '--bn-border-radius': '4px',
    '--bn-colors-side-menu': 'transparent',
  };
}
```

Injected on the editor root via `style` attribute at mount time. Template theme overrides flow through the existing `theme` prop on MDDMEditor.

### 3. DataTable — Native Table Content

DataTable migrates from `content: "none"` with dataTableRow/dataTableCell children to `content: "table"` using BlockNote's native TableContent:

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
      <div className={styles.dataTable} data-mddm-block="dataTable"
           data-density={props.block.props.density}>
        <div className={styles.header}>
          <strong>{props.block.props.label || "Data Table"}</strong>
          <span>{props.block.content.rows.length} linhas</span>
        </div>
        <div className={styles.tableContainer} ref={props.contentRef} />
      </div>
    ),
  },
);
```

Native table features inherited from BlockNote:
- Proper grid with borders and cell separators
- Cell editing with cursor navigation
- Tab between cells
- Column resizing
- Header row support
- Cell background/text color
- colspan/rowspan
- Copy/paste of table data

DataTableRow and DataTableCell block types are removed entirely.

### 4. Repeatable — CSS Reshaping

Repeatable keeps `content: "none"` with children. CSS reshaping pulls the `.bn-block-group` (BlockNote's child container) inside the Repeatable wrapper visually:

```css
/* Repeatable.module.css */
.repeatable {
  border: 1px solid var(--mddm-accent-border);
  border-radius: var(--mddm-radius);
  overflow: hidden;
}

/* RepeatableItem styling via global bridge */
.bn-container .react-renderer.node-repeatableItem + .bn-block-group {
  margin-left: 0;
  padding-left: var(--mddm-cell-padding);
  border-left: 3pt solid var(--mddm-accent);
}
```

RepeatableItem gets a numbered header:
- Numbers computed by RepeatableInterpreter from section context
- "3.1 Etapa 1", "3.2 Etapa 2"

Stale closure bug fixed by reading editor state at click time:
```typescript
onClick={() => {
  const currentBlock = props.editor.getBlock(props.block.id);
  const currentChildren = currentBlock?.children ?? [];
  props.editor.updateBlock(props.block, {
    children: [...currentChildren, newItem],
  });
}}
```

### 5. Chrome Visibility

All BlockNote chrome hidden by default, visible on hover/focus:

```css
/* Side menu and drag handle: invisible by default */
.bn-container .bn-side-menu,
.bn-container .bn-drag-handle {
  opacity: 0;
  pointer-events: none;
  transition: opacity 150ms ease;
}

/* Visible on hover */
.bn-container .bn-block:hover > .bn-side-menu,
.bn-container .bn-block:hover > .bn-drag-handle {
  opacity: 1;
  pointer-events: auto;
}

/* Nesting lines removed */
.bn-container .bn-block-group {
  border-left: none;
}

/* Read-only: completely removed */
.bn-container[data-editable="false"] .bn-side-menu,
.bn-container[data-editable="false"] .bn-drag-handle {
  display: none !important;
}
```

### 6. MDDMViewer

MDDMEditor with `readOnly={true}`. Not a separate component:

```typescript
export function MDDMViewer({ blocks, theme }: MDDMViewerProps) {
  return <MDDMEditor initialContent={blocks} theme={theme} readOnly={true} />;
}
```

Document opening logic in BrowserDocumentEditorView:
- DRAFT → MDDMEditor (editable)
- PENDING_APPROVAL → MDDMEditor (readOnly)
- RELEASED → MDDMViewer
- ARCHIVED → MDDMViewer

### 7. Export Buttons

Two buttons in the header:

```typescript
<button onClick={handleExportPdf}>Exportar PDF</button>
<button onClick={handleExportDocx}>Exportar DOCX</button>
{!isViewOnly && <button onClick={handleSave}>Salvar rascunho</button>}
```

DOCX export: client-side via docx.js (native export at 100% rollout).
PDF export: `toExternalHTML` → `wrapInPrintDocument` → POST to Gotenberg → stream PDF back.

### 8. toExternalHTML — Full Coverage

All 9 custom blocks get `toExternalHTML` implementations:
- Use `<table>` for layout (forbidden: flexbox, grid — per compatibility contract)
- Use inline styles (no CSS Modules — rendered in isolated React root)
- Read token values directly from Layout IR (not CSS variables)
- Include `data-mddm-block` attribute for print stylesheet targeting

Blocks requiring new implementations: DataTable, Repeatable, RepeatableItem, RichBlock.
Blocks with existing implementations: Section, Field, FieldGroup, DataTableCell (removed — merged into DataTable).

## Data Flow

### DataTable Migration

Old MDDM format (children):
```json
{
  "type": "dataTable",
  "props": { "columnsJson": "[{\"key\":\"item\",\"label\":\"Item\"}]" },
  "children": [
    { "type": "dataTableRow", "children": [
      { "type": "dataTableCell", "props": { "columnKey": "item" },
        "children": [{ "text": "Registro inicial" }] }
    ]}
  ]
}
```

New MDDM format (tableContent):
```json
{
  "type": "dataTable",
  "props": { "label": "Checklist da etapa" },
  "content": {
    "type": "tableContent",
    "columnWidths": [200],
    "headerRows": 1,
    "rows": [
      { "cells": [{ "content": [{ "type": "text", "text": "Item" }], "props": {} }] },
      { "cells": [{ "content": [{ "type": "text", "text": "Registro inicial" }], "props": {} }] }
    ]
  },
  "children": []
}
```

Migration handled in the MDDM adapter (`adapter.ts`) on read:
- Detect old format (has children with `dataTableRow` type)
- Convert `columnsJson` + children → `tableContent` with header row + data rows
- No database migration needed — conversion happens on read, new format saved on next write

This is an `mddm_version` bump from 1 → 2 registered in the canonicalize-migrate pipeline.

### Template Update

`po_template.go` updated: `dataTableBlock()` helper produces the new format with `tableContent` instead of children. `dataTableRowBlock()` and `dataTableCellBlock()` helpers removed.

### Removed Block Types

- `DataTableRow` — block type, component, CSS, emitter all deleted
- `DataTableCell` — block type, component, CSS, emitter all deleted

### Existing Documents

All existing documents (including test POs) will be deleted and recreated from the updated template. No backward compatibility required for old data.

## Release-Time Determinism

When a document transitions DRAFT → RELEASED:

1. Run `canonicalizeAndMigrate(envelope)` at current mddm_version
2. Generate DOCX with current renderer (client-side)
3. Store the DOCX blob as an immutable artifact (`release_artifact_key` in version record)
4. Store the canonical MDDM snapshot (`canonical_mddm_snapshot` JSONB in version record)
5. Capture RendererPin (version, IR hash, template ref)

After release:
- **Exportar DOCX**: Serve the stored artifact. No re-rendering.
- **Exportar PDF**: Re-render from canonical snapshot via toExternalHTML → Gotenberg (Tier 2 tolerance).
- **Viewer**: MDDMViewer loads canonical snapshot with current components (Tier 3 — visual only).

Backend schema addition:
```sql
ALTER TABLE metaldocs.document_versions
  ADD COLUMN release_artifact_key TEXT,
  ADD COLUMN canonical_mddm_snapshot JSONB;
```

Existing released documents: deleted and recreated fresh. No retroactive backfill needed.

## Error Handling

### DataTable migration errors
- Malformed `columnsJson`: fall back to empty table (0 columns, 0 rows)
- Missing children: produce table with header row only
- Log warning to telemetry, never crash the editor

### Export errors
- DOCX generation failure: show toast "Falha ao gerar DOCX. Tente novamente.", log stack trace
- PDF generation failure (Gotenberg down): show toast "Serviço de PDF indisponível. Tente exportar em DOCX.", offer DOCX fallback
- Asset resolution failure: skip failed asset, log warning, continue export
- Timeout (DOCX > 30s, PDF > 60s): show toast with specific message

### Released doc artifact errors
- Artifact missing: fall back to re-render from canonical snapshot with pinned renderer
- Canonical snapshot missing: fall back to re-render from raw MDDM JSON with current renderer, log error

### Chrome visibility
- If CSS selectors fail to match (BlockNote class name change on version update): chrome stays visible (fail-safe — visible is better than invisible)

## Testing Approach

### Layer 1: Unit tests (Vitest)
- Layout interpreter tests: section numbering, repeatable item counting, field width resolution
- Token bridge tests: `tokensToCssVars` produces correct values, template theme overrides apply
- DataTable migration tests: old children format → tableContent roundtrip
- toExternalHTML snapshot tests: each block's HTML output matches expected structure

### Layer 2: Golden file tests
- Existing 6 fixtures updated for new DataTable tableContent format
- New fixture: `07-repeatable-nested` — Repeatable with multiple items containing mixed blocks
- Each fixture gets `expected.external.html` alongside `expected.document.xml`
- Byte-exact comparison after normalization

### Layer 3: Visual parity tests (Playwright)
- Load each fixture in test harness (`/test-harness/mddm?doc=<fixture>`)
- Screenshot editor content area
- Export PDF, rasterize first page, compare → < 2% pixel diff
- Export DOCX, rasterize via LibreOffice, compare → < 5% pixel diff

Deterministic parity harness:
- Viewport: 1280 × 900, 1x DPI
- Fonts: Inter (editor), Carlito (export) — installed in CI container
- Browser: Chromium (Playwright default, same engine as Gotenberg)
- Editor width: `--mddm-page-content-width: 165mm`

### Layer 4: Completeness gate (CI)
Every MDDM block type must have:
- `render()` implementation
- `toExternalHTML()` implementation
- DOCX emitter registered
- At least one golden fixture exercising the block

Enforced on every PR. Blocks missing any renderer cannot be merged.

### Layer 5: End-to-end browser testing (Claude in Chrome)
46-test matrix covering:
- Document creation & template (5 tests)
- DataTable native table (7 tests)
- Repeatable (6 tests)
- Editor chrome (5 tests)
- Save flow (4 tests)
- DOCX export (6 tests)
- PDF export (5 tests)
- Document status & viewer (5 tests)
- Shadow testing & telemetry (3 tests)

Execution protocol:
1. Start all services (API, frontend, docgen, Gotenberg)
2. Run tests 1-46 sequentially in Chrome
3. Screenshot each result
4. All critical/major tests must pass before marking complete

Failure severity:
- Critical (DataTable rendering, export downloads): blocks release
- Major (Repeatable, save, DOCX/PDF content): fix before release unless within Tier 2 tolerance
- Minor (chrome visibility, telemetry): can ship with known imperfections

## Out of Scope

- Dark mode support
- Landscape / custom page sizes
- Embedded fonts in DOCX
- Real-time collaboration
- Server-side DOCX generation
- Batch/bulk export
- ODT/RTF export formats
- Image optimization (resize, compress)
- Custom numbering styles (roman, alpha)
- Drag-to-reorder Repeatable items
- Offline PDF generation
- Undo/redo enhancements beyond BlockNote native
