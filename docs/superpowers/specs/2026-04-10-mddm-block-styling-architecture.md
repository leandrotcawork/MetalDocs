# MDDM Block Styling Architecture

**Date:** 2026-04-10
**Status:** Draft
**Depends on:** MDDM Foundational Design (merged), PO Template V2

## Goal

Build a scalable, token-driven styling system for the MDDM BlockNote editor that:

1. Makes the editor visually approximate the exported DOCX — same section headers, same field tables, same data tables, same numbering
2. Uses a unified MetalDocs visual language across all template types, architected for per-template theming later
3. Ensures both renderers (browser editor + DOCX exporter) read from the same theme specification — no manual CSS-to-Word synchronization
4. Follows the Pattern 3 architecture validated by Qualio, PandaDoc, and other document SaaS platforms: structured data as source of truth, theme-driven dual rendering

## Architecture Overview

```
Template Definition (JSON)
├── structure: blocks, sections, fields, tables
├── theme: accent colors, variant selections, density
│
├──→ Browser Renderer (BlockNote + CSS)
│    └── theme → CSS custom properties on editor root
│    └── variant props → data-variant attributes → CSS module selectors
│
└──→ DOCX Renderer (docx npm)
     └── theme → Word shading fills, font colors, table widths
     └── variant props → rendering strategy selection
```

Both renderers consume the same theme object. Changing the theme updates both outputs without code changes.

## 1. Template Theme Object

Added to the template definition schema. Stored in `definition_json` alongside the block tree.

```json
{
  "type": "page",
  "id": "po-mddm-root",
  "theme": {
    "accent": "#6b1f2a",
    "accentLight": "#f9f3f3",
    "accentDark": "#3e1018",
    "accentBorder": "#dfc8c8"
  },
  "children": [ ... ]
}
```

When `theme` is absent, the MetalDocs default palette applies (vinho). This is backward-compatible — existing templates work unchanged.

The theme object is intentionally minimal for v1. Future extensions (font family, spacing scale, radius) follow the same pattern — add a key, both renderers read it.

## 2. Design Token System

Semantic CSS custom properties on the editor root, injected at runtime from the template theme.

### Raw palette (defaults, overridden by theme)

```css
--mddm-raw-vinho: #6b1f2a;
--mddm-raw-vinho-dark: #3e1018;
--mddm-raw-vinho-light: #f9f3f3;
--mddm-raw-vinho-border: #dfc8c8;
--mddm-raw-white: #ffffff;
--mddm-raw-gray-50: #f9fafb;
--mddm-raw-gray-200: #e5e7eb;
--mddm-raw-gray-400: #9ca3af;
--mddm-raw-gray-700: #374151;
```

### Semantic tokens (what components reference)

```css
--mddm-accent: var(--mddm-raw-vinho);
--mddm-accent-dark: var(--mddm-raw-vinho-dark);
--mddm-accent-light: var(--mddm-raw-vinho-light);
--mddm-accent-border: var(--mddm-raw-vinho-border);

--mddm-section-header-bg: var(--mddm-accent);
--mddm-section-header-text: var(--mddm-raw-white);
--mddm-field-label-bg: var(--mddm-accent-light);
--mddm-field-label-text: var(--mddm-accent-dark);
--mddm-field-border: var(--mddm-accent-border);
--mddm-table-header-bg: var(--mddm-accent-light);
--mddm-table-border: var(--mddm-raw-gray-200);
```

### Layout tokens

```css
--mddm-radius: 4px;
--mddm-spacing-xs: 0.25rem;
--mddm-spacing-sm: 0.5rem;
--mddm-spacing-md: 1rem;
--mddm-spacing-lg: 1.5rem;
--mddm-content-max-width: 860px;
```

### Typography tokens

```css
--mddm-font-family: "Inter", -apple-system, sans-serif;
--mddm-font-size-sm: 0.84rem;
--mddm-font-size-base: 0.95rem;
--mddm-font-size-section: 0.85rem;
```

### Runtime injection

`MDDMEditor` reads `templateSnapshot.definition.theme` and sets CSS variables on the editor root:

```tsx
const themeStyle = useMemo(() => {
  const t = templateTheme;
  if (!t) return {};
  return {
    "--mddm-accent": t.accent,
    "--mddm-accent-light": t.accentLight,
    "--mddm-accent-dark": t.accentDark,
    "--mddm-accent-border": t.accentBorder,
  } as React.CSSProperties;
}, [templateTheme]);

<div className={styles.editorRoot} style={themeStyle}>
```

## 3. Document Layout Shell

Content-width container — professional but responsive, not A4 page simulation.

```css
.editorRoot {
  max-width: var(--mddm-content-max-width);
  margin: 0 auto;
  background: var(--mddm-raw-white);
  padding: var(--mddm-spacing-lg) 2rem;
  min-height: 600px;
  font-family: var(--mddm-font-family);
  font-size: var(--mddm-font-size-base);
  color: var(--mddm-raw-gray-700);
  line-height: 1.6;
}
```

### Print contract (stub, not implemented)

```css
@media print {
  .editorRoot { max-width: none; padding: 0; box-shadow: none; }
}
```

## 4. Block Variant Architecture

Each structural block supports a `variant` prop that the template definition controls. CSS modules scope styles per variant via data attributes.

### Variant props per block

| Block | Prop | V1 Value | Future Values | Default |
|-------|------|----------|---------------|---------|
| Section | `variant` | `"bar"` | `"divider"`, `"outline"` | `"bar"` |
| Field | `layout` | `"grid"` | `"stacked"` | `"grid"` |
| FieldGroup | `columns` | `1`, `2` | — | `1` |
| DataTable | `density` | `"normal"` | `"compact"` | `"normal"` |
| RepeatableItem | `style` | `"bordered"` | `"minimal"` | `"bordered"` |
| RichBlock | `chrome` | `"labeled"` | `"plain"` | `"labeled"` |

Only v1 values are implemented. Others are defined in prop schema for future use.

### How variants flow

```
Template definition     →  MDDM block props      →  Editor render
(variant: "bar",           { variant: "bar",         <div data-variant="bar">
 color: "#6b1f2a")          color: "#6b1f2a" }          dark header bar
```

The adapter layer passes variant props through to BlockNote. Blocks render using data attributes. CSS modules target variants via attribute selectors.

## 5. Block Component Specifications

### Section

Dark header bar with white text, auto-numbered via CSS counter.

**Render:**
```tsx
<div className={styles.section} data-mddm-block="section"
     data-variant={props.block.props.variant || "bar"}
     data-locked={props.block.props.locked}>
  <div className={styles.sectionHeader}>
    <span className={styles.sectionTitle}>
      {props.block.props.title || "Section"}
    </span>
    {props.block.props.optional ? (
      <span className={styles.optionalBadge}>Opcional</span>
    ) : null}
  </div>
</div>
```

**Key CSS (variant: bar):**
```css
.section[data-variant="bar"] .sectionHeader {
  background: var(--mddm-section-header-bg);
  color: var(--mddm-section-header-text);
  padding: 8px 14px;
  font-size: var(--mddm-font-size-section);
  font-weight: 700;
  letter-spacing: 0.5px;
  text-transform: uppercase;
  display: flex;
  align-items: center;
  justify-content: space-between;
}
```

**Section numbering:** CSS counter on the editor root. `counter-reset: mddm-section` on `.editorRoot`, `counter-increment: mddm-section` on `.section`, `content: counter(mddm-section) ". "` on `.sectionTitle::before`. DOCX exporter computes numbers independently from sibling order.

### FieldGroup

Grid container. Columns controlled by `data-columns` attribute.

**Render:**
```tsx
<div className={styles.fieldGroup} data-mddm-block="fieldGroup"
     data-columns={props.block.props.columns}>
```

**Key CSS:**
```css
.fieldGroup {
  display: grid;
  grid-template-columns: 1fr;
  border: 1px solid var(--mddm-field-border);
  border-radius: var(--mddm-radius);
  overflow: hidden;
  margin-bottom: var(--mddm-spacing-sm);
}
.fieldGroup[data-columns="2"] {
  grid-template-columns: 1fr 1fr;
}
```

### Field

Label/value split. Label has colored background, value is editable.

**Render:**
```tsx
<div className={styles.field} data-mddm-block="field"
     data-layout={props.block.props.layout || "grid"}
     data-locked={props.block.props.locked}>
  <div className={styles.fieldLabel}>
    {props.block.props.label || "Field"}
  </div>
  <div className={styles.fieldValue} ref={props.contentRef} />
</div>
```

**Key CSS (layout: grid):**
```css
.field[data-layout="grid"] {
  display: grid;
  grid-template-columns: 35% 65%;
  min-height: 2.25rem;
  border-bottom: 1px solid var(--mddm-field-border);
}
.fieldLabel {
  background: var(--mddm-field-label-bg);
  color: var(--mddm-field-label-text);
  padding: var(--mddm-spacing-sm) 0.75rem;
  font-weight: 600;
  font-size: var(--mddm-font-size-sm);
  display: flex;
  align-items: center;
  border-right: 1px solid var(--mddm-field-border);
}
.fieldValue {
  padding: var(--mddm-spacing-sm) 0.75rem;
  background: var(--mddm-raw-white);
  display: flex;
  align-items: center;
}
```

### Repeatable

Labeled container for repeatable items.

**Render:**
```tsx
<div className={styles.repeatable} data-mddm-block="repeatable">
  <div className={styles.repeatableLabel}>
    {props.block.props.label || "Repeatable"}
  </div>
</div>
```

**Key CSS:**
```css
.repeatableLabel {
  font-weight: 600;
  font-size: var(--mddm-font-size-sm);
  color: var(--mddm-accent-dark);
  margin-bottom: var(--mddm-spacing-sm);
  text-transform: uppercase;
  letter-spacing: 0.3px;
}
```

### RepeatableItem

Left accent border with header.

**Render:**
```tsx
<div className={styles.repeatableItem} data-mddm-block="repeatableItem"
     data-style={props.block.props.style || "bordered"}>
  <div className={styles.itemHeader}>
    {props.block.props.title || "Item"}
  </div>
</div>
```

**Key CSS (style: bordered):**
```css
.repeatableItem[data-style="bordered"] {
  border-left: 3px solid var(--mddm-accent);
  padding-left: var(--mddm-spacing-md);
  margin-bottom: var(--mddm-spacing-md);
}
.itemHeader {
  font-weight: 600;
  font-size: var(--mddm-font-size-base);
  color: var(--mddm-accent-dark);
  margin-bottom: var(--mddm-spacing-sm);
  padding-bottom: var(--mddm-spacing-xs);
  border-bottom: 1px solid var(--mddm-field-border);
}
```

### RichBlock

Labeled content area for rich text (paragraphs, lists, images).

**Render:**
```tsx
<div className={styles.richBlock} data-mddm-block="richBlock"
     data-chrome={props.block.props.chrome || "labeled"}>
  <div className={styles.richBlockLabel}>
    {props.block.props.label || "Content"}
  </div>
</div>
```

**Key CSS (chrome: labeled):**
```css
.richBlock[data-chrome="labeled"] .richBlockLabel {
  font-weight: 600;
  font-size: var(--mddm-font-size-sm);
  color: var(--mddm-accent-dark);
  margin-bottom: var(--mddm-spacing-xs);
}
```

### DataTable

CSS Grid-based table with column headers from template definition and add-row button.

**Render:**
```tsx
render: (props) => {
  const columns = parseColumns(props.block.props.columnsJson);
  return (
    <div className={styles.dataTable} data-mddm-block="dataTable"
         data-density={props.block.props.density || "normal"}>
      <div className={styles.tableLabel}>
        {props.block.props.label || "Tabela"}
      </div>
      <div className={styles.tableGrid}
           style={{ gridTemplateColumns: `repeat(${columns.length}, 1fr)` }}>
        {columns.map((col) => (
          <div key={col.key} className={styles.th}>{col.label}</div>
        ))}
      </div>
      <button type="button" className={styles.addRowButton}>
        + Adicionar linha
      </button>
    </div>
  );
}
```

**Key CSS (density: normal):**
```css
.tableGrid {
  display: grid;
  border: 1px solid var(--mddm-table-border);
  font-size: var(--mddm-font-size-sm);
}
.th {
  background: var(--mddm-table-header-bg);
  color: var(--mddm-field-label-text);
  padding: var(--mddm-spacing-sm) 0.75rem;
  font-weight: 600;
  text-align: left;
  border: 1px solid var(--mddm-table-border);
}
.addRowButton {
  display: block;
  width: 100%;
  padding: var(--mddm-spacing-xs) var(--mddm-spacing-sm);
  margin-top: var(--mddm-spacing-xs);
  background: none;
  border: 1px dashed var(--mddm-table-border);
  border-radius: var(--mddm-radius);
  color: var(--mddm-raw-gray-400);
  font-size: var(--mddm-font-size-sm);
  cursor: pointer;
}
.addRowButton:hover {
  color: var(--mddm-accent);
  border-color: var(--mddm-accent-border);
}
```

### DataTableRow

Row container using CSS grid placement.

```tsx
<div className={styles.row} data-mddm-block="dataTableRow" role="row" />
```

```css
.row { border-bottom: 1px solid var(--mddm-table-border); }
.row:hover { background: var(--mddm-raw-gray-50); }
```

### DataTableCell

Cell with editable inline content.

```tsx
<td className={styles.cell} data-mddm-block="dataTableCell"
    data-column-key={props.block.props.columnKey} role="cell">
  <div ref={props.contentRef} />
</td>
```

```css
.cell {
  padding: var(--mddm-spacing-sm) 0.75rem;
  border: 1px solid var(--mddm-table-border);
  vertical-align: top;
}
```

## 6. DOCX Export Styling

The exporter at `apps/docgen/src/mddm/exporter.ts` reads the same theme object and variant props from the MDDM block tree.

### Theme consumption

```ts
function resolveTheme(envelope: MDDMEnvelope): ExportTheme {
  const t = envelope.template_ref?.theme;
  return {
    accent: t?.accent ?? "#6b1f2a",
    accentLight: t?.accentLight ?? "#f9f3f3",
    accentDark: t?.accentDark ?? "#3e1018",
    accentBorder: t?.accentBorder ?? "#dfc8c8",
  };
}
```

### Block rendering with theme

**Section (variant: bar):**
```ts
new Paragraph({
  children: [new TextRun({ text: `${index}. ${title}`, bold: true, color: "FFFFFF", size: 20 })],
  shading: { type: ShadingType.CLEAR, fill: theme.accent.replace("#", "") },
  spacing: { before: 240, after: 120 },
  heading: HeadingLevel.HEADING_1,
})
```

**Field (layout: grid):**
```ts
new TableRow({
  children: [
    new TableCell({
      children: [new Paragraph({ children: [new TextRun({ text: label, bold: true, size: 18, color: theme.accentDark.replace("#", "") })] })],
      shading: { fill: theme.accentLight.replace("#", "") },
      width: { size: 35, type: WidthType.PERCENTAGE },
    }),
    new TableCell({
      children: [valueParagraph],
      width: { size: 65, type: WidthType.PERCENTAGE },
    }),
  ],
})
```

**DataTable:**
```ts
new Table({
  rows: [
    new TableRow({
      children: columns.map(col => new TableCell({
        children: [new Paragraph({ children: [new TextRun({ text: col.label, bold: true })] })],
        shading: { fill: theme.accentLight.replace("#", "") },
      })),
    }),
    ...dataRows,
  ],
  width: { size: 100, type: WidthType.PERCENTAGE },
})
```

## 7. State-Based Styling

Data attributes for interactive states, targeted by CSS.

```css
/* Focus within a block */
[data-mddm-block]:focus-within {
  outline: 2px solid var(--mddm-accent-border);
  outline-offset: 1px;
}

/* Locked indicator */
[data-locked="true"] { position: relative; }

/* Future: validation error */
[data-mddm-block="field"][data-invalid="true"] .fieldLabel {
  color: #dc2626;
  background: #fef2f2;
}

/* Readonly mode */
.editorRoot[data-readonly="true"] [data-mddm-block] { cursor: default; }
```

## 8. Global Bridge Layer

One thin `mddm-editor-global.css` file that bridges BlockNote defaults with MDDM styling.

```css
/* Vertical rhythm */
.bn-container [data-content-type] { margin-bottom: 2px; }

/* Hide drag handles on structural blocks */
.bn-container [data-content-type="section"] .bn-side-menu,
.bn-container [data-content-type="fieldGroup"] .bn-side-menu,
.bn-container [data-content-type="repeatable"] .bn-side-menu {
  display: none;
}

/* Typography reset */
.bn-container .bn-inline-content {
  font-family: var(--mddm-font-family);
  font-size: var(--mddm-font-size-base);
}

/* Remove extra padding on nested blocks inside structural containers */
.bn-container [data-content-type="section"] > .bn-block-group,
.bn-container [data-content-type="fieldGroup"] > .bn-block-group {
  padding-left: 0;
}
```

## 9. File Structure

### New files

```
mddm-editor/
  mddm-editor-global.css
  blocks/
    Section.module.css
    FieldGroup.module.css
    Field.module.css
    Repeatable.module.css
    RepeatableItem.module.css
    RichBlock.module.css
    DataTable.module.css
    DataTableRow.module.css
    DataTableCell.module.css
```

### Modified files

```
mddm-editor/
  MDDMEditor.module.css          (updated container styles)
  MDDMEditor.tsx                 (import global CSS, inject theme vars)
  blocks/
    Section.tsx                  (styled render)
    FieldGroup.tsx               (grid container)
    Field.tsx                    (label/value split)
    Repeatable.tsx               (styled label)
    RepeatableItem.tsx           (left border accent)
    RichBlock.tsx                (styled label)
    DataTable.tsx                (grid table, headers, add-row)
    DataTableRow.tsx             (row container)
    DataTableCell.tsx            (cell with content)

apps/docgen/src/mddm/
  exporter.ts                   (theme-driven Word rendering)
```

### No backend changes needed for v1

The theme object lives inside the existing `definition_json` field. No schema migration required.

## 10. In Scope

- Design token system with runtime injection from template theme
- Document layout shell (content-width container)
- Variant architecture (prop schema + data attributes)
- All 9 block components styled with v1 variant
- DOCX exporter updated to read theme and produce matching output
- CSS counters for section numbering
- State styling (focus, locked, hover)
- DataTable add-row button
- Global bridge layer for BlockNote overrides

## 11. Out of Scope

- Per-template theming UI (theme object must be set manually in template JSON for now)
- Additional variants beyond v1 defaults
- Add/remove repeatable items UI
- Delete row on DataTable
- Field `valueMode: "multiParagraph"` support
- Print/export CSS implementation
- Visual regression tests
- Template generator UI
- Reduced-motion transitions

## 12. Success Criteria

1. A new PO document in the editor shows dark section headers, labeled field tables, real data tables, accent-bordered repeatable items — visually approximating the reference DOCX
2. Exported DOCX from the same document shows matching structure: same section header style, same field table layout, same data table format, same colors
3. Changing the template theme `accent` color to blue updates both editor and DOCX output without code changes
4. TypeScript compiles with zero errors
5. No new npm dependencies
