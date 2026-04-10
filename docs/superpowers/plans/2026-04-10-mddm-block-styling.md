# MDDM Block Styling Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement the token-driven styling system from the spec at `docs/superpowers/specs/2026-04-10-mddm-block-styling-architecture.md`. All 9 custom BlockNote blocks get production-grade styling. DOCX exporter updated to read the same theme. CSS Modules per block, design tokens as CSS custom properties, variant architecture via data attributes.

**Spec reference:** `docs/superpowers/specs/2026-04-10-mddm-block-styling-architecture.md`

**File map:**

```
frontend/apps/web/src/features/documents/mddm-editor/
├── mddm-editor-global.css                    ← NEW: BlockNote bridge + design tokens
├── MDDMEditor.module.css                      ← MODIFY: update container styles
├── MDDMEditor.tsx                             ← MODIFY: import global CSS, inject theme vars, pass theme prop
├── schema.ts                                  ← NO CHANGE
├── adapter.ts                                 ← MODIFY: pass variant props through
├── blocks/
│   ├── Section.tsx                            ← MODIFY: styled render
│   ├── Section.module.css                     ← NEW
│   ├── FieldGroup.tsx                         ← MODIFY: grid container render
│   ├── FieldGroup.module.css                  ← NEW
│   ├── Field.tsx                              ← MODIFY: label/value render
│   ├── Field.module.css                       ← NEW
│   ├── Repeatable.tsx                         ← MODIFY: styled label
│   ├── Repeatable.module.css                  ← NEW
│   ├── RepeatableItem.tsx                     ← MODIFY: bordered item
│   ├── RepeatableItem.module.css              ← NEW
│   ├── RichBlock.tsx                          ← MODIFY: labeled content
│   ├── RichBlock.module.css                   ← NEW
│   ├── DataTable.tsx                          ← MODIFY: grid table + headers + add-row
│   ├── DataTable.module.css                   ← NEW
│   ├── DataTableRow.tsx                       ← MODIFY: row container
│   ├── DataTableRow.module.css                ← NEW
│   ├── DataTableCell.tsx                      ← MODIFY: cell styling
│   └── DataTableCell.module.css               ← NEW

frontend/apps/web/src/features/documents/browser-editor/
├── BrowserDocumentEditorView.tsx              ← MODIFY: pass theme to MDDMEditor

frontend/apps/web/src/lib.types.ts             ← MODIFY: add theme to snapshot type

apps/docgen/src/mddm/
├── exporter.ts                                ← MODIFY: theme-driven rendering
├── render-tables.ts                           ← MODIFY: themed field tables
├── render-data-table.ts                       ← MODIFY: themed data tables
```

---

# Phase 1 — Foundation (Tokens + Layout + Bridge)

## Task 1: Create the global CSS with design tokens and BlockNote bridge

**Files:**
- Create: `frontend/apps/web/src/features/documents/mddm-editor/mddm-editor-global.css`

- [ ] **Step 1: Create the global CSS file**

Create `frontend/apps/web/src/features/documents/mddm-editor/mddm-editor-global.css`:

```css
/* ══════════════════════════════════════════════════════
   MDDM Editor — Design Tokens & BlockNote Bridge
   ══════════════════════════════════════════════════════ */

/* ── Raw palette (defaults, overridden by theme injection) ── */
:root {
  --mddm-raw-vinho: #6b1f2a;
  --mddm-raw-vinho-dark: #3e1018;
  --mddm-raw-vinho-light: #f9f3f3;
  --mddm-raw-vinho-border: #dfc8c8;
  --mddm-raw-white: #ffffff;
  --mddm-raw-gray-50: #f9fafb;
  --mddm-raw-gray-200: #e5e7eb;
  --mddm-raw-gray-400: #9ca3af;
  --mddm-raw-gray-700: #374151;
}

/* ── Semantic tokens ── */
:root {
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

  --mddm-radius: 4px;
  --mddm-spacing-xs: 0.25rem;
  --mddm-spacing-sm: 0.5rem;
  --mddm-spacing-md: 1rem;
  --mddm-spacing-lg: 1.5rem;
  --mddm-content-max-width: 860px;

  --mddm-font-family: "Inter", -apple-system, sans-serif;
  --mddm-font-size-sm: 0.84rem;
  --mddm-font-size-base: 0.95rem;
  --mddm-font-size-section: 0.85rem;
}

/* ── BlockNote bridge ── */

/* Vertical rhythm */
.bn-container [data-content-type] {
  margin-bottom: 2px;
}

/* Hide drag handles on structural blocks */
.bn-container [data-content-type="section"] > .bn-block-outer > .bn-block > .bn-side-menu,
.bn-container [data-content-type="fieldGroup"] > .bn-block-outer > .bn-block > .bn-side-menu,
.bn-container [data-content-type="repeatable"] > .bn-block-outer > .bn-block > .bn-side-menu {
  display: none;
}

/* Typography reset for inline content */
.bn-container .bn-inline-content {
  font-family: var(--mddm-font-family);
  font-size: var(--mddm-font-size-base);
}

/* Remove extra left-padding on nested blocks inside structural containers */
.bn-container [data-content-type="section"] > .bn-block-group,
.bn-container [data-content-type="fieldGroup"] > .bn-block-group,
.bn-container [data-content-type="repeatable"] > .bn-block-group,
.bn-container [data-content-type="repeatableItem"] > .bn-block-group,
.bn-container [data-content-type="richBlock"] > .bn-block-group {
  padding-left: 0;
}

/* State: focus within a block */
[data-mddm-block]:focus-within {
  outline: 2px solid var(--mddm-accent-border);
  outline-offset: 1px;
  border-radius: var(--mddm-radius);
}

/* Section counter reset — applied by MDDMEditor.module.css on editorRoot */
```

- [ ] **Step 2: Commit**

```bash
git add frontend/apps/web/src/features/documents/mddm-editor/mddm-editor-global.css
git commit -m "feat(mddm): add design tokens and BlockNote bridge global CSS"
```

---

## Task 2: Update MDDMEditor container and theme injection

**Files:**
- Modify: `frontend/apps/web/src/features/documents/mddm-editor/MDDMEditor.module.css`
- Modify: `frontend/apps/web/src/features/documents/mddm-editor/MDDMEditor.tsx`

- [ ] **Step 1: Rewrite MDDMEditor.module.css**

Replace the entire content of `frontend/apps/web/src/features/documents/mddm-editor/MDDMEditor.module.css`:

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
  counter-reset: mddm-section;
}

@media print {
  .editorRoot {
    max-width: none;
    padding: 0;
    box-shadow: none;
  }
}
```

- [ ] **Step 2: Update MDDMEditor.tsx with theme injection**

Replace the entire content of `frontend/apps/web/src/features/documents/mddm-editor/MDDMEditor.tsx`:

```tsx
import { useMemo } from "react";
import { type PartialBlock } from "@blocknote/core";
import { BlockNoteView } from "@blocknote/mantine";
import { useCreateBlockNote } from "@blocknote/react";
import "@blocknote/core/fonts/inter.css";
import "@blocknote/mantine/style.css";
import "./mddm-editor-global.css";
import { mddmSchema } from "./schema";
import styles from "./MDDMEditor.module.css";

export type MDDMTheme = {
  accent?: string;
  accentLight?: string;
  accentDark?: string;
  accentBorder?: string;
};

export type MDDMEditorProps = {
  initialContent?: PartialBlock[];
  onChange?: (blocks: unknown[]) => void;
  readOnly?: boolean;
  theme?: MDDMTheme;
};

export function MDDMEditor({
  initialContent,
  onChange,
  readOnly,
  theme,
}: MDDMEditorProps) {
  const editor = useCreateBlockNote({
    schema: mddmSchema,
    initialContent: initialContent?.length ? initialContent : undefined,
  });

  const themeStyle = useMemo(() => {
    if (!theme) return undefined;
    const vars: Record<string, string> = {};
    if (theme.accent) vars["--mddm-accent"] = theme.accent;
    if (theme.accentLight) vars["--mddm-accent-light"] = theme.accentLight;
    if (theme.accentDark) vars["--mddm-accent-dark"] = theme.accentDark;
    if (theme.accentBorder) vars["--mddm-accent-border"] = theme.accentBorder;
    return Object.keys(vars).length > 0 ? vars as React.CSSProperties : undefined;
  }, [theme]);

  return (
    <div className={styles.editorRoot} style={themeStyle}>
      <BlockNoteView
        editor={editor}
        editable={!readOnly}
        onChange={(currentEditor) => onChange?.(currentEditor.document)}
      />
    </div>
  );
}
```

- [ ] **Step 3: Verify TypeScript compiles**

```bash
cd frontend/apps/web && npx tsc --noEmit
```

Expected: zero errors.

- [ ] **Step 4: Commit**

```bash
git add frontend/apps/web/src/features/documents/mddm-editor/MDDMEditor.module.css frontend/apps/web/src/features/documents/mddm-editor/MDDMEditor.tsx
git commit -m "feat(mddm): update editor container with theme injection and layout shell"
```

---

## Task 3: Wire theme from BrowserDocumentEditorView to MDDMEditor

**Files:**
- Modify: `frontend/apps/web/src/features/documents/browser-editor/BrowserDocumentEditorView.tsx`

- [ ] **Step 1: Extract theme from template snapshot and pass to MDDMEditor**

In `BrowserDocumentEditorView.tsx`, find the `<MDDMEditor` JSX usage. Add a `theme` prop derived from the bundle's template snapshot definition:

```tsx
// Add this useMemo near the other useMemo hooks:
const editorTheme = useMemo(() => {
  const def = bundle?.templateSnapshot?.definition as Record<string, unknown> | undefined;
  const t = def?.theme as Record<string, string> | undefined;
  if (!t) return undefined;
  return {
    accent: t.accent,
    accentLight: t.accentLight,
    accentDark: t.accentDark,
    accentBorder: t.accentBorder,
  };
}, [bundle?.templateSnapshot?.definition]);
```

Then pass it to `<MDDMEditor>`:

```tsx
<MDDMEditor
  initialContent={blockNoteDocument as PartialBlock[]}
  onChange={handleEditorChange}
  theme={editorTheme}
/>
```

Update the import to include `MDDMTheme` if needed for type safety.

- [ ] **Step 2: Verify TypeScript compiles**

```bash
cd frontend/apps/web && npx tsc --noEmit
```

Expected: zero errors.

- [ ] **Step 3: Commit**

```bash
git add frontend/apps/web/src/features/documents/browser-editor/BrowserDocumentEditorView.tsx
git commit -m "feat(mddm): wire template theme to editor via BrowserDocumentEditorView"
```

---

# Phase 2 — Block Component Styling (Editor)

## Task 4: Style Section block with dark header bar and CSS counter

**Files:**
- Create: `frontend/apps/web/src/features/documents/mddm-editor/blocks/Section.module.css`
- Modify: `frontend/apps/web/src/features/documents/mddm-editor/blocks/Section.tsx`

- [ ] **Step 1: Create Section.module.css**

Create `frontend/apps/web/src/features/documents/mddm-editor/blocks/Section.module.css`:

```css
.section {
  margin-top: var(--mddm-spacing-lg);
  counter-increment: mddm-section;
}

.sectionHeader {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

/* Variant: bar */
.section[data-variant="bar"] .sectionHeader {
  background: var(--mddm-section-header-bg);
  color: var(--mddm-section-header-text);
  padding: 8px 14px;
  font-size: var(--mddm-font-size-section);
  font-weight: 700;
  letter-spacing: 0.5px;
  text-transform: uppercase;
}

.sectionTitle::before {
  content: counter(mddm-section) ". ";
}

.optionalBadge {
  font-size: 0.75rem;
  font-weight: 400;
  text-transform: none;
  letter-spacing: 0;
  opacity: 0.7;
  padding-left: var(--mddm-spacing-md);
  white-space: nowrap;
}
```

- [ ] **Step 2: Update Section.tsx render**

Replace the entire content of `frontend/apps/web/src/features/documents/mddm-editor/blocks/Section.tsx`:

```tsx
import { createReactBlockSpec } from "@blocknote/react";
import styles from "./Section.module.css";

export const Section = createReactBlockSpec(
  {
    type: "section",
    propSchema: {
      title: { default: "" },
      color: { default: "#6b1f2a" },
      locked: { default: true },
      optional: { default: false },
      variant: { default: "bar" },
      __template_block_id: { default: "" },
    },
    content: "none",
  },
  {
    render: (props) => (
      <div
        className={styles.section}
        data-mddm-block="section"
        data-variant={props.block.props.variant || "bar"}
        data-locked={props.block.props.locked}
      >
        <div className={styles.sectionHeader}>
          <span className={styles.sectionTitle}>
            {props.block.props.title || "Section"}
          </span>
          {props.block.props.optional ? (
            <span className={styles.optionalBadge}>Opcional</span>
          ) : null}
        </div>
      </div>
    ),
  },
);
```

- [ ] **Step 3: Verify TypeScript compiles**

```bash
cd frontend/apps/web && npx tsc --noEmit
```

- [ ] **Step 4: Commit**

```bash
git add frontend/apps/web/src/features/documents/mddm-editor/blocks/Section.tsx frontend/apps/web/src/features/documents/mddm-editor/blocks/Section.module.css
git commit -m "feat(mddm): style Section block with dark header bar and CSS counter"
```

---

## Task 5: Style FieldGroup block with grid container

**Files:**
- Create: `frontend/apps/web/src/features/documents/mddm-editor/blocks/FieldGroup.module.css`
- Modify: `frontend/apps/web/src/features/documents/mddm-editor/blocks/FieldGroup.tsx`

- [ ] **Step 1: Create FieldGroup.module.css**

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

- [ ] **Step 2: Update FieldGroup.tsx render**

```tsx
import { createReactBlockSpec } from "@blocknote/react";
import styles from "./FieldGroup.module.css";

export const FieldGroup = createReactBlockSpec(
  {
    type: "fieldGroup",
    propSchema: {
      columns: { default: 1, values: [1, 2] as const },
      locked: { default: true },
      __template_block_id: { default: "" },
    },
    content: "none",
  },
  {
    render: (props) => (
      <div
        className={styles.fieldGroup}
        data-mddm-block="fieldGroup"
        data-columns={props.block.props.columns}
        data-locked={props.block.props.locked}
      />
    ),
  },
);
```

- [ ] **Step 3: Commit**

```bash
git add frontend/apps/web/src/features/documents/mddm-editor/blocks/FieldGroup.tsx frontend/apps/web/src/features/documents/mddm-editor/blocks/FieldGroup.module.css
git commit -m "feat(mddm): style FieldGroup block with grid container"
```

---

## Task 6: Style Field block with label/value split

**Files:**
- Create: `frontend/apps/web/src/features/documents/mddm-editor/blocks/Field.module.css`
- Modify: `frontend/apps/web/src/features/documents/mddm-editor/blocks/Field.tsx`

- [ ] **Step 1: Create Field.module.css**

```css
.field {
  border-bottom: 1px solid var(--mddm-field-border);
}

.field:last-child {
  border-bottom: none;
}

/* Layout: grid */
.field[data-layout="grid"] {
  display: grid;
  grid-template-columns: 35% 65%;
  min-height: 2.25rem;
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

.fieldHint {
  display: block;
  font-weight: 400;
  font-size: 0.75rem;
  opacity: 0.7;
  margin-top: 2px;
}

.fieldValue {
  padding: var(--mddm-spacing-sm) 0.75rem;
  background: var(--mddm-raw-white);
  display: flex;
  align-items: center;
  min-height: 2.25rem;
}
```

- [ ] **Step 2: Update Field.tsx render**

```tsx
import { createReactBlockSpec } from "@blocknote/react";
import styles from "./Field.module.css";

export const Field = createReactBlockSpec(
  {
    type: "field",
    propSchema: {
      label: { default: "" },
      valueMode: { default: "inline", values: ["inline"] as const },
      locked: { default: true },
      hint: { default: "" },
      layout: { default: "grid" },
      __template_block_id: { default: "" },
    },
    content: "inline",
  },
  {
    render: (props) => (
      <div
        className={styles.field}
        data-mddm-block="field"
        data-layout={props.block.props.layout || "grid"}
        data-locked={props.block.props.locked}
      >
        <div className={styles.fieldLabel}>
          <span>
            {props.block.props.label || "Field"}
            {props.block.props.hint ? (
              <small className={styles.fieldHint}>{props.block.props.hint}</small>
            ) : null}
          </span>
        </div>
        <div className={styles.fieldValue} ref={props.contentRef} />
      </div>
    ),
  },
);
```

- [ ] **Step 3: Commit**

```bash
git add frontend/apps/web/src/features/documents/mddm-editor/blocks/Field.tsx frontend/apps/web/src/features/documents/mddm-editor/blocks/Field.module.css
git commit -m "feat(mddm): style Field block with label/value grid layout"
```

---

## Task 7: Style Repeatable and RepeatableItem blocks

**Files:**
- Create: `frontend/apps/web/src/features/documents/mddm-editor/blocks/Repeatable.module.css`
- Create: `frontend/apps/web/src/features/documents/mddm-editor/blocks/RepeatableItem.module.css`
- Modify: `frontend/apps/web/src/features/documents/mddm-editor/blocks/Repeatable.tsx`
- Modify: `frontend/apps/web/src/features/documents/mddm-editor/blocks/RepeatableItem.tsx`

- [ ] **Step 1: Create Repeatable.module.css**

```css
.repeatable {
  margin: var(--mddm-spacing-md) 0;
}

.repeatableLabel {
  font-weight: 600;
  font-size: var(--mddm-font-size-sm);
  color: var(--mddm-accent-dark);
  margin-bottom: var(--mddm-spacing-sm);
  text-transform: uppercase;
  letter-spacing: 0.3px;
}
```

- [ ] **Step 2: Update Repeatable.tsx render**

```tsx
import { createReactBlockSpec } from "@blocknote/react";
import styles from "./Repeatable.module.css";

export const Repeatable = createReactBlockSpec(
  {
    type: "repeatable",
    propSchema: {
      label: { default: "" },
      itemPrefix: { default: "Item" },
      locked: { default: true },
      minItems: { default: 0 },
      maxItems: { default: 100 },
      __template_block_id: { default: "" },
    },
    content: "none",
  },
  {
    render: (props) => (
      <div className={styles.repeatable} data-mddm-block="repeatable">
        <div className={styles.repeatableLabel}>
          {props.block.props.label || "Repeatable"}
        </div>
      </div>
    ),
  },
);
```

- [ ] **Step 3: Create RepeatableItem.module.css**

```css
.repeatableItem {
  margin-bottom: var(--mddm-spacing-md);
}

/* Style: bordered */
.repeatableItem[data-style="bordered"] {
  border-left: 3px solid var(--mddm-accent);
  padding-left: var(--mddm-spacing-md);
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

- [ ] **Step 4: Update RepeatableItem.tsx render**

```tsx
import { createReactBlockSpec } from "@blocknote/react";
import styles from "./RepeatableItem.module.css";

export const RepeatableItem = createReactBlockSpec(
  {
    type: "repeatableItem",
    propSchema: {
      title: { default: "" },
      style: { default: "bordered" },
    },
    content: "none",
  },
  {
    render: (props) => (
      <div
        className={styles.repeatableItem}
        data-mddm-block="repeatableItem"
        data-style={props.block.props.style || "bordered"}
      >
        <div className={styles.itemHeader}>
          {props.block.props.title || "Item"}
        </div>
      </div>
    ),
  },
);
```

- [ ] **Step 5: Commit**

```bash
git add frontend/apps/web/src/features/documents/mddm-editor/blocks/Repeatable.tsx frontend/apps/web/src/features/documents/mddm-editor/blocks/Repeatable.module.css frontend/apps/web/src/features/documents/mddm-editor/blocks/RepeatableItem.tsx frontend/apps/web/src/features/documents/mddm-editor/blocks/RepeatableItem.module.css
git commit -m "feat(mddm): style Repeatable and RepeatableItem blocks"
```

---

## Task 8: Style RichBlock

**Files:**
- Create: `frontend/apps/web/src/features/documents/mddm-editor/blocks/RichBlock.module.css`
- Modify: `frontend/apps/web/src/features/documents/mddm-editor/blocks/RichBlock.tsx`

- [ ] **Step 1: Create RichBlock.module.css**

```css
.richBlock {
  margin: var(--mddm-spacing-sm) 0;
}

/* Chrome: labeled */
.richBlock[data-chrome="labeled"] .richBlockLabel {
  font-weight: 600;
  font-size: var(--mddm-font-size-sm);
  color: var(--mddm-accent-dark);
  margin-bottom: var(--mddm-spacing-xs);
}
```

- [ ] **Step 2: Update RichBlock.tsx render**

```tsx
import { createReactBlockSpec } from "@blocknote/react";
import styles from "./RichBlock.module.css";

export const RichBlock = createReactBlockSpec(
  {
    type: "richBlock",
    propSchema: {
      label: { default: "" },
      locked: { default: true },
      chrome: { default: "labeled" },
      __template_block_id: { default: "" },
    },
    content: "none",
  },
  {
    render: (props) => (
      <div
        className={styles.richBlock}
        data-mddm-block="richBlock"
        data-chrome={props.block.props.chrome || "labeled"}
      >
        <div className={styles.richBlockLabel}>
          {props.block.props.label || "Conteúdo"}
        </div>
      </div>
    ),
  },
);
```

- [ ] **Step 3: Commit**

```bash
git add frontend/apps/web/src/features/documents/mddm-editor/blocks/RichBlock.tsx frontend/apps/web/src/features/documents/mddm-editor/blocks/RichBlock.module.css
git commit -m "feat(mddm): style RichBlock with labeled chrome"
```

---

## Task 9: Style DataTable with CSS grid, headers, and add-row button

**Files:**
- Create: `frontend/apps/web/src/features/documents/mddm-editor/blocks/DataTable.module.css`
- Modify: `frontend/apps/web/src/features/documents/mddm-editor/blocks/DataTable.tsx`

- [ ] **Step 1: Create DataTable.module.css**

```css
.dataTable {
  margin: var(--mddm-spacing-sm) 0 var(--mddm-spacing-md);
}

.tableLabel {
  font-weight: 600;
  font-size: var(--mddm-font-size-sm);
  color: var(--mddm-accent-dark);
  margin-bottom: var(--mddm-spacing-xs);
}

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
  border-bottom: 1px solid var(--mddm-table-border);
  border-right: 1px solid var(--mddm-table-border);
}

.th:last-child {
  border-right: none;
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
  transition: color 0.15s, border-color 0.15s;
}

.addRowButton:hover {
  color: var(--mddm-accent);
  border-color: var(--mddm-accent-border);
}
```

- [ ] **Step 2: Update DataTable.tsx render**

```tsx
import { createReactBlockSpec } from "@blocknote/react";
import styles from "./DataTable.module.css";

type Column = { key: string; label: string };

function parseColumns(json: string): Column[] {
  try {
    const parsed = JSON.parse(json);
    return Array.isArray(parsed) ? parsed.filter((c) => c && typeof c.key === "string") : [];
  } catch {
    return [];
  }
}

export const DataTable = createReactBlockSpec(
  {
    type: "dataTable",
    propSchema: {
      label: { default: "" },
      columnsJson: { default: "[]" },
      locked: { default: true },
      minRows: { default: 0 },
      maxRows: { default: 500 },
      density: { default: "normal" },
      __template_block_id: { default: "" },
    },
    content: "none",
  },
  {
    render: (props) => {
      const columns = parseColumns(props.block.props.columnsJson);
      return (
        <div
          className={styles.dataTable}
          data-mddm-block="dataTable"
          data-density={props.block.props.density || "normal"}
        >
          <div className={styles.tableLabel}>
            {props.block.props.label || "Tabela"}
          </div>
          {columns.length > 0 ? (
            <div
              className={styles.tableGrid}
              style={{ gridTemplateColumns: `repeat(${columns.length}, 1fr)` }}
            >
              {columns.map((col) => (
                <div key={col.key} className={styles.th}>
                  {col.label}
                </div>
              ))}
            </div>
          ) : null}
          <button type="button" className={styles.addRowButton}>
            + Adicionar linha
          </button>
        </div>
      );
    },
  },
);
```

- [ ] **Step 3: Commit**

```bash
git add frontend/apps/web/src/features/documents/mddm-editor/blocks/DataTable.tsx frontend/apps/web/src/features/documents/mddm-editor/blocks/DataTable.module.css
git commit -m "feat(mddm): style DataTable block with grid headers and add-row button"
```

---

## Task 10: Style DataTableRow and DataTableCell

**Files:**
- Create: `frontend/apps/web/src/features/documents/mddm-editor/blocks/DataTableRow.module.css`
- Create: `frontend/apps/web/src/features/documents/mddm-editor/blocks/DataTableCell.module.css`
- Modify: `frontend/apps/web/src/features/documents/mddm-editor/blocks/DataTableRow.tsx`
- Modify: `frontend/apps/web/src/features/documents/mddm-editor/blocks/DataTableCell.tsx`

- [ ] **Step 1: Create DataTableRow.module.css**

```css
.row {
  border-bottom: 1px solid var(--mddm-table-border);
}

.row:hover {
  background: var(--mddm-raw-gray-50);
}
```

- [ ] **Step 2: Update DataTableRow.tsx**

```tsx
import { createReactBlockSpec } from "@blocknote/react";
import styles from "./DataTableRow.module.css";

export const DataTableRow = createReactBlockSpec(
  {
    type: "dataTableRow",
    propSchema: {},
    content: "none",
  },
  {
    render: () => (
      <div className={styles.row} data-mddm-block="dataTableRow" role="row" />
    ),
  },
);
```

- [ ] **Step 3: Create DataTableCell.module.css**

```css
.cell {
  padding: var(--mddm-spacing-sm) 0.75rem;
  border-right: 1px solid var(--mddm-table-border);
  vertical-align: top;
  min-height: 2rem;
}

.cell:last-child {
  border-right: none;
}
```

- [ ] **Step 4: Update DataTableCell.tsx**

```tsx
import { createReactBlockSpec } from "@blocknote/react";
import styles from "./DataTableCell.module.css";

export const DataTableCell = createReactBlockSpec(
  {
    type: "dataTableCell",
    propSchema: {
      columnKey: { default: "" },
    },
    content: "inline",
  },
  {
    render: (props) => (
      <div
        className={styles.cell}
        data-mddm-block="dataTableCell"
        data-column-key={props.block.props.columnKey}
        role="cell"
      >
        <div ref={props.contentRef} />
      </div>
    ),
  },
);
```

- [ ] **Step 5: Verify full TypeScript build**

```bash
cd frontend/apps/web && npx tsc --noEmit
```

Expected: zero errors.

- [ ] **Step 6: Commit**

```bash
git add frontend/apps/web/src/features/documents/mddm-editor/blocks/DataTableRow.tsx frontend/apps/web/src/features/documents/mddm-editor/blocks/DataTableRow.module.css frontend/apps/web/src/features/documents/mddm-editor/blocks/DataTableCell.tsx frontend/apps/web/src/features/documents/mddm-editor/blocks/DataTableCell.module.css
git commit -m "feat(mddm): style DataTableRow and DataTableCell blocks"
```

---

# Phase 3 — Adapter Passthrough + Template Theme Seed

## Task 11: Update adapter to pass variant props through to BlockNote

**Files:**
- Modify: `frontend/apps/web/src/features/documents/mddm-editor/adapter.ts`

- [ ] **Step 1: Ensure variant props are mapped in mddmToBlockNote**

In `adapter.ts`, find the function that maps MDDM block props to BlockNote block props. Ensure these props are passed through for each block type:

- `section`: `variant` (default `"bar"`)
- `field`: `layout` (default `"grid"`)
- `repeatableItem`: `style` (default `"bordered"`)
- `richBlock`: `chrome` (default `"labeled"`)
- `dataTable`: `density` (default `"normal"`)

The adapter should copy any prop from the MDDM block that exists in the BlockNote propSchema. If the prop isn't present in the MDDM data, the BlockNote default applies.

- [ ] **Step 2: Verify TypeScript compiles**

```bash
cd frontend/apps/web && npx tsc --noEmit
```

- [ ] **Step 3: Commit**

```bash
git add frontend/apps/web/src/features/documents/mddm-editor/adapter.ts
git commit -m "feat(mddm): pass variant props through adapter to BlockNote blocks"
```

---

## Task 12: Add theme to PO template definition in database

**Files:**
- Create: `migrations/0068_add_theme_to_po_mddm_template.sql`

- [ ] **Step 1: Write migration to add theme object to PO template definition**

Create `migrations/0068_add_theme_to_po_mddm_template.sql`:

```sql
-- 0068_add_theme_to_po_mddm_template.sql
-- Adds the theme object to the po-mddm-canvas template definition_json.
-- Both the editor and DOCX exporter read this to produce matching visuals.

UPDATE metaldocs.document_template_versions
SET definition_json = definition_json || '{"theme": {"accent": "#6b1f2a", "accentLight": "#f9f3f3", "accentDark": "#3e1018", "accentBorder": "#dfc8c8"}}'::jsonb
WHERE template_key = 'po-mddm-canvas'
  AND version = 1;
```

- [ ] **Step 2: Apply migration locally**

```bash
PGPASSWORD='Lepa12<>!' "C:/Program Files/PostgreSQL/16/bin/psql.exe" -h 127.0.0.1 -p 5433 -U metaldocs_app -d metaldocs -f migrations/0068_add_theme_to_po_mddm_template.sql
```

Expected: `UPDATE 1`

- [ ] **Step 3: Commit**

```bash
git add migrations/0068_add_theme_to_po_mddm_template.sql
git commit -m "feat(mddm): add theme object to PO template definition"
```

---

# Phase 4 — DOCX Exporter Theme Alignment

## Task 13: Add theme resolution to DOCX exporter

**Files:**
- Modify: `apps/docgen/src/mddm/exporter.ts`

- [ ] **Step 1: Add theme type and resolver function**

At the top of `exporter.ts`, add:

```ts
type ExportTheme = {
  accent: string;
  accentLight: string;
  accentDark: string;
  accentBorder: string;
};

const DEFAULT_THEME: ExportTheme = {
  accent: "#6b1f2a",
  accentLight: "#f9f3f3",
  accentDark: "#3e1018",
  accentBorder: "#dfc8c8",
};

function resolveTheme(request: MDDMExportRequest): ExportTheme {
  const t = request.templateTheme;
  if (!t || typeof t !== "object") return DEFAULT_THEME;
  return {
    accent: typeof t.accent === "string" ? t.accent : DEFAULT_THEME.accent,
    accentLight: typeof t.accentLight === "string" ? t.accentLight : DEFAULT_THEME.accentLight,
    accentDark: typeof t.accentDark === "string" ? t.accentDark : DEFAULT_THEME.accentDark,
    accentBorder: typeof t.accentBorder === "string" ? t.accentBorder : DEFAULT_THEME.accentBorder,
  };
}

function hexToDocx(hex: string): string {
  return hex.replace("#", "").toUpperCase();
}
```

- [ ] **Step 2: Update section rendering to use theme**

Find where sections are rendered as `Paragraph` with `HeadingLevel`. Update to use theme colors:

```ts
// Replace hardcoded colors with:
new Paragraph({
  children: [new TextRun({ text: `${index}. ${title}`, bold: true, color: "FFFFFF", size: 20 })],
  shading: { type: ShadingType.CLEAR, fill: hexToDocx(theme.accent) },
  spacing: { before: 240, after: 120 },
  heading: HeadingLevel.HEADING_1,
})
```

- [ ] **Step 3: Update types.ts to include templateTheme in request**

Add `templateTheme?: Record<string, string>` to the `MDDMExportRequest` type in `apps/docgen/src/mddm/types.ts`.

- [ ] **Step 4: Commit**

```bash
git add apps/docgen/src/mddm/exporter.ts apps/docgen/src/mddm/types.ts
git commit -m "feat(mddm): add theme resolution to DOCX exporter"
```

---

## Task 14: Update field table and data table exporters with theme colors

**Files:**
- Modify: `apps/docgen/src/mddm/render-tables.ts`
- Modify: `apps/docgen/src/mddm/render-data-table.ts`

- [ ] **Step 1: Update render-tables.ts to accept theme and use it for field label shading**

Pass `ExportTheme` to `renderFieldGroup`. Use `theme.accentLight` for label cell shading, `theme.accentDark` for label text color.

- [ ] **Step 2: Update render-data-table.ts to accept theme and use it for header row shading**

Pass `ExportTheme` to `renderDataTable`. Use `theme.accentLight` for header cell shading.

- [ ] **Step 3: Commit**

```bash
git add apps/docgen/src/mddm/render-tables.ts apps/docgen/src/mddm/render-data-table.ts
git commit -m "feat(mddm): apply theme colors to field and data table DOCX renderers"
```

---

## Task 15: Wire template theme from Go service to docgen request

**Files:**
- Modify: `internal/modules/documents/application/service_content_docx.go`

- [ ] **Step 1: Read theme from template definition and include in docgen payload**

When building the `MDDMExportRequest` payload for the docgen service, extract the `theme` object from the template definition and include it as `templateTheme`.

- [ ] **Step 2: Verify Go builds**

```bash
go build ./...
```

- [ ] **Step 3: Commit**

```bash
git add internal/modules/documents/application/service_content_docx.go
git commit -m "feat(mddm): pass template theme from Go service to docgen export request"
```

---

# Phase 5 — Visual Verification

## Task 16: End-to-end visual verification in browser

- [ ] **Step 1: Rebuild Go API and restart**

```bash
go build -o metaldocs-api.exe ./apps/api/cmd/metaldocs-api/
# Restart from the terminal with correct env vars
```

- [ ] **Step 2: Verify TypeScript compiles clean**

```bash
cd frontend/apps/web && npx tsc --noEmit
```

Expected: zero errors.

- [ ] **Step 3: Create a new PO document in the browser**

Navigate to `http://localhost:4173`, click "Novo documento", fill title, click "Ir para o editor".

Verify:
- Dark vinho header bars on each section with white uppercase text
- Section numbering (1. Identificação, 2. Identificação do Processo, ...)
- Optional sections show "Opcional" badge
- Fields render as label/value grid with colored label background
- FieldGroups show 2-column layout where specified
- RepeatableItems have left border accent
- DataTable shows column headers with light background
- Add-row button appears below DataTable
- No BlockNote vertical lines or raw structural text

- [ ] **Step 4: Export DOCX and verify visual alignment**

Click "Exportar DOCX", open the downloaded file, verify:
- Section headers have dark background shading matching editor
- Field tables have colored label cells
- Data tables have header rows with shading
- Colors match between editor and DOCX

- [ ] **Step 5: Final commit if any adjustments needed**

```bash
git add -A
git commit -m "fix(mddm): visual adjustments from end-to-end verification"
```
