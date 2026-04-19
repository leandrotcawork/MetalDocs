# FieldGroup/Field → Native BlockNote Table Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace custom field/fieldGroup blocks with BlockNote's native `table` block so fields render as proper bordered tables with headers, cell colors, and real table structure — matching the DOCX reference output.

**Architecture:** The MDDM data model keeps `fieldGroup`/`field` types for storage and DOCX export (which already renders them as tables). The adapter layer converts fieldGroup→native table at load time and table→fieldGroup at save time. The editor never sees field/fieldGroup blocks — only native tables. This gives us BlockNote's built-in table rendering (proper borders, header styling, cell selection, column resize) without changing the Go template, DOCX emitters, or database schema.

**Tech Stack:** BlockNote v0.47.3 advanced tables, TypeScript, Vitest

---

## File Structure

| File | Action | Responsibility |
|------|--------|---------------|
| `mddm-editor/MDDMEditor.tsx` | Modify | Enable `tables: { headers: true, cellBackgroundColor: true }` |
| `mddm-editor/adapter.ts` | Modify | Add `fieldGroupToTable()` and `tableToFieldGroup()` conversion functions |
| `mddm-editor/schema.ts` | Modify | Remove `field` and `fieldGroup` from `mddmSchemaBlockSpecs` |
| `mddm-editor/__tests__/adapter.test.ts` | Modify | Add tests for fieldGroup↔table conversion |
| `mddm-editor/blocks/Field.tsx` | Delete | No longer used — native table handles rendering |
| `mddm-editor/blocks/Field.module.css` | Delete | No longer used |
| `mddm-editor/blocks/FieldGroup.tsx` | Delete | No longer used |
| `mddm-editor/blocks/FieldGroup.module.css` | Delete | No longer used |
| `mddm-editor/mddm-editor-global.css` | Modify | Remove fieldGroup grid rules (lines 92-106) |

**Unchanged files (no modifications needed):**
- `po_template.go` — MDDM format keeps fieldGroup/field types
- `engine/emitters/field.ts` — DOCX emitter works on MDDM data, unaffected
- `engine/emitters/field-group.ts` — same, unaffected
- `engine/external-html/field-html.tsx` — external HTML no longer used (native table has its own)
- `engine/external-html/field-group-html.tsx` — same

---

### Task 1: Enable Advanced Tables in the Editor

**Files:**
- Modify: `frontend/apps/web/src/features/documents/mddm-editor/MDDMEditor.tsx:35-38`

- [ ] **Step 1: Add tables config to useCreateBlockNote**

```typescript
// MDDMEditor.tsx — change lines 35-38 from:
const editor = useCreateBlockNote({
  schema: mddmSchema,
  initialContent: initialContent?.length ? initialContent : undefined,
});

// to:
const editor = useCreateBlockNote({
  schema: mddmSchema,
  initialContent: initialContent?.length ? initialContent : undefined,
  tables: {
    headers: true,
    cellBackgroundColor: true,
  },
});
```

- [ ] **Step 2: Verify TypeScript compiles**

Run: `cd frontend/apps/web && npx tsc --noEmit`
Expected: exit 0

- [ ] **Step 3: Commit**

```bash
rtk git add frontend/apps/web/src/features/documents/mddm-editor/MDDMEditor.tsx
rtk git commit -m "feat(mddm): enable BlockNote advanced tables with headers and cell colors"
```

---

### Task 2: Write Failing Tests for fieldGroup→table Conversion

**Files:**
- Modify: `frontend/apps/web/src/features/documents/mddm-editor/__tests__/adapter.test.ts`

- [ ] **Step 1: Write test for 1-column fieldGroup converting to table**

Add this test after the existing roundtrip tests:

```typescript
describe("fieldGroup → native table conversion", () => {
  it("converts 1-column fieldGroup with inline fields to a table block", () => {
    const mddm: MDDMEnvelope = {
      mddm_version: 1,
      template_ref: null,
      blocks: [
        {
          id: "fg-1",
          type: "fieldGroup",
          props: { columns: 1, locked: true },
          children: [
            {
              id: "f-1",
              type: "field",
              props: { label: "Objetivo", valueMode: "inline", locked: true, layout: "grid" },
              children: [{ text: "Valor A" }],
            },
            {
              id: "f-2",
              type: "field",
              props: { label: "Escopo", valueMode: "inline", locked: true, layout: "grid" },
              children: [{ text: "Valor B" }],
            },
          ],
        },
      ],
    };

    const blocks = mddmToBlockNote(mddm);
    expect(blocks).toHaveLength(1);

    const table = blocks[0];
    expect(table.type).toBe("table");

    const content = table.content as any;
    expect(content.type).toBe("tableContent");
    expect(content.rows).toHaveLength(2);

    // Row 1: label cell with background + value cell
    const row1 = content.rows[0];
    expect(row1.cells).toHaveLength(2);
    // Label cell text
    expect(row1.cells[0]).toEqual(
      expect.arrayContaining([expect.objectContaining({ type: "text", text: "Objetivo" })]),
    );
    // Value cell text
    expect(row1.cells[1]).toEqual(
      expect.arrayContaining([expect.objectContaining({ type: "text", text: "Valor A" })]),
    );

    // Row 2
    const row2 = content.rows[1];
    expect(row2.cells[0]).toEqual(
      expect.arrayContaining([expect.objectContaining({ type: "text", text: "Escopo" })]),
    );
    expect(row2.cells[1]).toEqual(
      expect.arrayContaining([expect.objectContaining({ type: "text", text: "Valor B" })]),
    );

    // headerCols=1 marks the label column as a header
    expect(content.headerCols).toBe(1);
  });

  it("converts 2-column fieldGroup to a 4-column table", () => {
    const mddm: MDDMEnvelope = {
      mddm_version: 1,
      template_ref: null,
      blocks: [
        {
          id: "fg-1",
          type: "fieldGroup",
          props: { columns: 2, locked: true },
          children: [
            {
              id: "f-1",
              type: "field",
              props: { label: "Elaborado por", valueMode: "inline", locked: true, layout: "grid" },
              children: [{ text: "João" }],
            },
            {
              id: "f-2",
              type: "field",
              props: { label: "Aprovado por", valueMode: "inline", locked: true, layout: "grid" },
              children: [{ text: "Maria" }],
            },
            {
              id: "f-3",
              type: "field",
              props: { label: "Data de criação", valueMode: "inline", locked: true, layout: "grid" },
              children: [],
            },
            {
              id: "f-4",
              type: "field",
              props: { label: "Data de aprovação", valueMode: "inline", locked: true, layout: "grid" },
              children: [],
            },
          ],
        },
      ],
    };

    const blocks = mddmToBlockNote(mddm);
    const table = blocks[0];
    expect(table.type).toBe("table");

    const content = table.content as any;
    // 4 fields in 2-column layout = 2 rows of 4 cells each
    expect(content.rows).toHaveLength(2);
    expect(content.rows[0].cells).toHaveLength(4);
    expect(content.rows[1].cells).toHaveLength(4);

    // Row 1: Label1 | Value1 | Label2 | Value2
    expect(content.rows[0].cells[0]).toEqual(
      expect.arrayContaining([expect.objectContaining({ text: "Elaborado por" })]),
    );
    expect(content.rows[0].cells[1]).toEqual(
      expect.arrayContaining([expect.objectContaining({ text: "João" })]),
    );
    expect(content.rows[0].cells[2]).toEqual(
      expect.arrayContaining([expect.objectContaining({ text: "Aprovado por" })]),
    );
    expect(content.rows[0].cells[3]).toEqual(
      expect.arrayContaining([expect.objectContaining({ text: "Maria" })]),
    );
  });

  it("preserves __mddm_field_group metadata for roundtrip", () => {
    const mddm: MDDMEnvelope = {
      mddm_version: 1,
      template_ref: null,
      blocks: [
        {
          id: "fg-1",
          template_block_id: "tpl-fg-1",
          type: "fieldGroup",
          props: { columns: 1, locked: true },
          children: [
            {
              id: "f-1",
              template_block_id: "tpl-f-1",
              type: "field",
              props: { label: "Nome", valueMode: "inline", locked: true, layout: "grid" },
              children: [{ text: "Test" }],
            },
          ],
        },
      ],
    };

    const blocks = mddmToBlockNote(mddm);
    const table = blocks[0];

    // Metadata embedded in props for roundtrip
    expect(table.props?.__mddm_field_group).toBeTruthy();
    const meta = JSON.parse(table.props!.__mddm_field_group as string);
    expect(meta.id).toBe("fg-1");
    expect(meta.templateBlockId).toBe("tpl-fg-1");
    expect(meta.columns).toBe(1);
    expect(meta.fields).toHaveLength(1);
    expect(meta.fields[0].id).toBe("f-1");
    expect(meta.fields[0].label).toBe("Nome");
  });
});
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd frontend/apps/web && rtk vitest run src/features/documents/mddm-editor/__tests__/adapter.test.ts`
Expected: FAIL — `fieldGroupToTable` functions don't exist yet

- [ ] **Step 3: Commit**

```bash
rtk git add frontend/apps/web/src/features/documents/mddm-editor/__tests__/adapter.test.ts
rtk git commit -m "test(mddm): add failing tests for fieldGroup→table conversion"
```

---

### Task 3: Implement fieldGroup→table Conversion in the Adapter

**Files:**
- Modify: `frontend/apps/web/src/features/documents/mddm-editor/adapter.ts`

- [ ] **Step 1: Add the fieldGroupToTable conversion function**

Add this function before the `toBlockNoteBlock` function (around line 195):

```typescript
/**
 * Convert a fieldGroup MDDM block (with field children) into a native
 * BlockNote table block. Each field becomes a row with 2 cells:
 *   - Cell 0: label text (will be styled as header via headerCols)
 *   - Cell 1: editable value (inline content)
 *
 * For columns=2, fields are paired into rows with 4 cells:
 *   - Cell 0: label1, Cell 1: value1, Cell 2: label2, Cell 3: value2
 *
 * A JSON metadata blob is stored in props.__mddm_field_group so the
 * reverse conversion (table→fieldGroup) can reconstruct the original
 * MDDM structure on save.
 */
function fieldGroupToTable(block: MDDMBlock): BlockNoteBlock {
  const columns = Number((block.props as any).columns) === 2 ? 2 : 1;
  const fields = (block.children ?? []) as MDDMBlock[];

  // Build metadata for roundtrip
  const meta = {
    id: block.id,
    templateBlockId: block.template_block_id ?? "",
    columns,
    locked: Boolean((block.props as any).locked),
    fields: fields.map((f) => ({
      id: f.id,
      templateBlockId: f.template_block_id ?? "",
      label: asString((f.props as any)?.label),
      valueMode: asString((f.props as any)?.valueMode) || "inline",
      locked: Boolean((f.props as any)?.locked),
      hint: asOptionalString((f.props as any)?.hint) ?? "",
      layout: asString((f.props as any)?.layout) || "grid",
    })),
  };

  // Convert each field's inline children to table cell content
  const fieldCells: { label: TableCellContent[]; value: TableCellContent[] }[] = fields.map((f) => {
    const label: TableCellContent[] = [
      { type: "text" as const, text: asString((f.props as any)?.label), styles: { bold: true } },
    ];
    const valueRuns = Array.isArray(f.children) ? (f.children as MDDMTextRun[]) : [];
    const value: TableCellContent[] = valueRuns.length > 0
      ? valueRuns.map((run) => ({
          type: "text" as const,
          text: run.text ?? "",
          ...(run.marks?.length ? { styles: marksToStyles(run.marks) } : {}),
        }))
      : [{ type: "text" as const, text: "" }];
    return { label, value };
  });

  // Arrange cells into rows based on column count
  const rows: TableRow[] = [];
  const cellsPerRow = columns * 2; // label+value per column

  for (let i = 0; i < fieldCells.length; i += columns) {
    const cells: TableCellContent[][] = [];
    for (let c = 0; c < columns; c++) {
      const fc = fieldCells[i + c];
      if (fc) {
        cells.push(fc.label);
        cells.push(fc.value);
      } else {
        // Pad with empty cells for odd field count
        cells.push([{ type: "text" as const, text: "" }]);
        cells.push([{ type: "text" as const, text: "" }]);
      }
    }
    rows.push({ cells });
  }

  // Column widths: label columns narrower, value columns wider
  const columnWidths: (number | null)[] = [];
  for (let c = 0; c < columns; c++) {
    columnWidths.push(200); // label column ~200px
    columnWidths.push(null); // value column auto
  }

  return {
    id: block.id,
    type: "table",
    props: {
      __mddm_field_group: JSON.stringify(meta),
    },
    content: {
      type: "tableContent",
      columnWidths,
      headerRows: 0,
      headerCols: columns, // marks label columns as headers (1 for 1-col, 2 for 2-col)
      rows,
    } as TableContent,
    children: [],
  };
}
```

Note: The `headerCols` value needs to account for 2-col layout. For columns=1, headerCols=1 (first column is header). For columns=2, we want columns 0 and 2 to be headers but `headerCols` is a count from the left, so headerCols=1 still works — only the first column in each "pair" would be header. Actually, for 2-column fieldGroups, the table is `Label1 | Value1 | Label2 | Value2` — we want both label columns (0 and 2) to be headers. Since BlockNote's `headerCols` counts consecutive columns from the left, we can't mark columns 0 and 2 independently. Instead, we'll rely on the bold text styling in label cells and accept headerCols=1 for now.

Update the `headerCols` line to:
```typescript
      headerCols: 1, // first column in each row is a label
```

- [ ] **Step 2: Hook fieldGroupToTable into toBlockNoteBlock**

In the `toBlockNoteBlock` function (around line 196), add this case BEFORE the existing fieldGroup handling. Add it right after the multiParagraph field handling (after line 253):

```typescript
  // fieldGroup → native table conversion
  if (block.type === "fieldGroup") {
    return fieldGroupToTable(block);
  }
```

Also add `"table"` to the `ALLOWED_MDDM_TYPES` set if not already there, and ensure `"field"` stays in `INLINE_BLOCK_TYPES`.

Since fieldGroup blocks are now fully consumed by `fieldGroupToTable()` (including their field children), the individual `field` type will never reach `toBlockNoteBlock` independently when nested inside a fieldGroup. However, for safety, keep `"field"` in `ALLOWED_MDDM_TYPES`.

- [ ] **Step 3: Run the new tests**

Run: `cd frontend/apps/web && rtk vitest run src/features/documents/mddm-editor/__tests__/adapter.test.ts`
Expected: The 3 new fieldGroup→table tests PASS

- [ ] **Step 4: Run all tests**

Run: `cd frontend/apps/web && rtk vitest run`
Expected: All tests pass (some existing tests may need adjustment if they test fieldGroup roundtrip — fix inline)

- [ ] **Step 5: Commit**

```bash
rtk git add frontend/apps/web/src/features/documents/mddm-editor/adapter.ts
rtk git commit -m "feat(mddm): convert fieldGroup/field to native table at load time"
```

---

### Task 4: Write Failing Tests for table→fieldGroup Reverse Conversion

**Files:**
- Modify: `frontend/apps/web/src/features/documents/mddm-editor/__tests__/adapter.test.ts`

- [ ] **Step 1: Write test for roundtrip fieldGroup→table→fieldGroup**

```typescript
describe("table → fieldGroup reverse conversion (save)", () => {
  it("roundtrips fieldGroup through table and back to MDDM", () => {
    const original: MDDMEnvelope = {
      mddm_version: 1,
      template_ref: null,
      blocks: [
        {
          id: "fg-1",
          template_block_id: "tpl-fg-1",
          type: "fieldGroup",
          props: { columns: 2, locked: true },
          children: [
            {
              id: "f-1",
              template_block_id: "tpl-f-1",
              type: "field",
              props: { label: "Elaborado por", valueMode: "inline", locked: true, layout: "grid" },
              children: [{ text: "João" }],
            },
            {
              id: "f-2",
              template_block_id: "tpl-f-2",
              type: "field",
              props: { label: "Aprovado por", valueMode: "inline", locked: true, layout: "grid" },
              children: [{ text: "Maria" }],
            },
          ],
        },
      ],
    };

    // Load: MDDM → BlockNote (fieldGroup becomes table)
    const blocks = mddmToBlockNote(original);
    expect(blocks[0].type).toBe("table");

    // Save: BlockNote → MDDM (table becomes fieldGroup)
    const saved = blockNoteToMDDM(blocks);
    expect(saved.blocks).toHaveLength(1);

    const fg = saved.blocks[0];
    expect(fg.type).toBe("fieldGroup");
    expect(fg.id).toBe("fg-1");
    expect(fg.template_block_id).toBe("tpl-fg-1");
    expect((fg.props as any).columns).toBe(2);

    const fields = fg.children as MDDMBlock[];
    expect(fields).toHaveLength(2);
    expect(fields[0].type).toBe("field");
    expect(fields[0].id).toBe("f-1");
    expect((fields[0].props as any).label).toBe("Elaborado por");
    // Value text preserved
    const textRuns = fields[0].children as MDDMTextRun[];
    expect(textRuns[0].text).toBe("João");
  });

  it("does NOT convert regular tables (without __mddm_field_group)", () => {
    const blocks: BlockNoteBlock[] = [
      {
        id: "t-1",
        type: "table",
        props: {},
        content: {
          type: "tableContent",
          columnWidths: [null, null],
          headerRows: 1,
          rows: [
            { cells: [[{ type: "text", text: "Header" }], [{ type: "text", text: "Value" }]] },
          ],
        },
        children: [],
      },
    ];

    // Regular table without marker should stay as-is (not become fieldGroup)
    // This should throw because "table" is not in ALLOWED_MDDM_TYPES
    // unless we add special handling
    expect(() => blockNoteToMDDM(blocks as any)).toThrow(/unsupported block type/i);
  });
});
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd frontend/apps/web && rtk vitest run src/features/documents/mddm-editor/__tests__/adapter.test.ts`
Expected: FAIL — reverse conversion not implemented yet

- [ ] **Step 3: Commit**

```bash
rtk git add frontend/apps/web/src/features/documents/mddm-editor/__tests__/adapter.test.ts
rtk git commit -m "test(mddm): add failing tests for table→fieldGroup reverse conversion"
```

---

### Task 5: Implement table→fieldGroup Reverse Conversion

**Files:**
- Modify: `frontend/apps/web/src/features/documents/mddm-editor/adapter.ts`

- [ ] **Step 1: Add the tableToFieldGroup conversion function**

Add this function near `fieldGroupToTable`:

```typescript
/**
 * Reverse conversion: BlockNote table with __mddm_field_group metadata
 * back to MDDM fieldGroup + field blocks for storage.
 *
 * Returns null if the table is a regular table (no metadata marker).
 */
function tableToFieldGroup(block: BlockNoteBlock): MDDMBlock | null {
  const metaJson = block.props?.__mddm_field_group;
  if (typeof metaJson !== "string") return null;

  let meta: {
    id: string;
    templateBlockId: string;
    columns: number;
    locked: boolean;
    fields: {
      id: string;
      templateBlockId: string;
      label: string;
      valueMode: string;
      locked: boolean;
      hint: string;
      layout: string;
    }[];
  };
  try {
    meta = JSON.parse(metaJson);
  } catch {
    return null;
  }

  const content = block.content as TableContent | undefined;
  const rows = content?.rows ?? [];

  // Rebuild field blocks from table rows + metadata
  const fields: MDDMBlock[] = [];
  let fieldIdx = 0;
  const cols = meta.columns;

  for (const row of rows) {
    for (let c = 0; c < cols; c++) {
      const labelCellIdx = c * 2;
      const valueCellIdx = c * 2 + 1;
      const fieldMeta = meta.fields[fieldIdx];
      if (!fieldMeta) break;

      // Extract value cell inline content → MDDM text runs
      const valueCellContent = row.cells?.[valueCellIdx] ?? [];
      const valueRuns: MDDMTextRun[] = (valueCellContent as TableCellContent[])
        .filter((c) => c.text !== "")
        .map((c) => {
          const run: MDDMTextRun = { text: c.text };
          if (c.styles) {
            const marks = stylesToMarks(c.styles);
            if (marks.length > 0) run.marks = marks;
          }
          return run;
        });

      const fieldBlock: MDDMBlock = {
        id: fieldMeta.id,
        type: "field",
        props: {
          label: fieldMeta.label,
          valueMode: fieldMeta.valueMode,
          locked: fieldMeta.locked,
          ...(fieldMeta.hint ? { hint: fieldMeta.hint } : {}),
          layout: fieldMeta.layout || "grid",
        },
        children: valueRuns,
      };
      if (fieldMeta.templateBlockId) {
        fieldBlock.template_block_id = fieldMeta.templateBlockId;
      }
      fields.push(fieldBlock);
      fieldIdx++;
    }
  }

  const output: MDDMBlock = {
    id: meta.id,
    type: "fieldGroup",
    props: {
      columns: meta.columns,
      locked: meta.locked,
    },
    children: fields,
  };
  if (meta.templateBlockId) {
    output.template_block_id = meta.templateBlockId;
  }

  return output;
}
```

- [ ] **Step 2: Hook tableToFieldGroup into toMDDMBlock**

In the `toMDDMBlock` function, add a check at the top (before the existing mddmType resolution):

```typescript
function toMDDMBlock(block: BlockNoteBlock): MDDMBlock {
  // Table with field group metadata → convert back to fieldGroup
  if (block.type === "table") {
    const converted = tableToFieldGroup(block);
    if (converted) return converted;
    // Regular tables without __mddm_field_group fall through to error below
  }

  const mddmType = toMDDMType(block.type);
  // ... rest of function
```

- [ ] **Step 3: Run the reverse conversion tests**

Run: `cd frontend/apps/web && rtk vitest run src/features/documents/mddm-editor/__tests__/adapter.test.ts`
Expected: All fieldGroup↔table tests PASS

- [ ] **Step 4: Run all tests**

Run: `cd frontend/apps/web && rtk vitest run`
Expected: All 208+ tests pass

- [ ] **Step 5: Commit**

```bash
rtk git add frontend/apps/web/src/features/documents/mddm-editor/adapter.ts
rtk git commit -m "feat(mddm): reverse table→fieldGroup conversion for MDDM storage"
```

---

### Task 6: Remove field/fieldGroup from BlockNote Schema

**Files:**
- Modify: `frontend/apps/web/src/features/documents/mddm-editor/schema.ts`
- Modify: `frontend/apps/web/src/features/documents/mddm-editor/mddm-editor-global.css`

- [ ] **Step 1: Remove field and fieldGroup from mddmSchemaBlockSpecs**

```typescript
// schema.ts — remove field and fieldGroup imports and entries

import { BlockNoteSchema, defaultBlockSpecs } from "@blocknote/core";
import { DataTable } from "./blocks/DataTable";
// REMOVED: import { Field } from "./blocks/Field";
// REMOVED: import { FieldGroup } from "./blocks/FieldGroup";
import { Repeatable } from "./blocks/Repeatable";
import { RepeatableItem } from "./blocks/RepeatableItem";
import { RichBlock } from "./blocks/RichBlock";
import { Section } from "./blocks/Section";

const {
  paragraph,
  heading,
  bulletListItem,
  numberedListItem,
  image,
  quote,
  divider,
  codeBlock,
  table,
} = defaultBlockSpecs;

export const mddmSchemaBlockSpecs = {
  section: Section(),
  // REMOVED: fieldGroup: FieldGroup(),
  // REMOVED: field: Field(),
  repeatable: Repeatable(),
  repeatableItem: RepeatableItem(),
  dataTable: DataTable(),
  richBlock: RichBlock(),
};

export const mddmSchema = BlockNoteSchema.create({
  blockSpecs: {
    paragraph,
    heading,
    bulletListItem,
    numberedListItem,
    image,
    quote,
    divider,
    codeBlock,
    table,
    ...mddmSchemaBlockSpecs,
  },
});
```

- [ ] **Step 2: Remove fieldGroup grid CSS from global stylesheet**

In `mddm-editor-global.css`, remove the fieldGroup-specific rules that are no longer needed (the 2-column grid, the `node-fieldGroup` selectors). Specifically remove:

```css
/* REMOVE these rules — fieldGroup no longer renders as a custom block */

.bn-container .react-renderer.node-fieldGroup:has([data-columns]) + .bn-block-group { ... }
.bn-container .react-renderer.node-fieldGroup:has([data-columns="2"]) + .bn-block-group { ... }
.bn-container .react-renderer.node-fieldGroup:has([data-columns="2"]) + .bn-block-group > .bn-block-outer:nth-child(odd) { ... }
```

Also remove the `node-fieldGroup` entry from the block-group margin-left reset (line 57):
```css
/* BEFORE */
.bn-container .react-renderer.node-fieldGroup + .bn-block-group { margin-left: 0; padding-left: 0; }
/* AFTER — remove this line entirely */
```

And the `fieldGroup` entries from the force-fill rule (lines 71-72):
```css
/* REMOVE these two lines from the selector group */
.bn-container .bn-block-content[data-content-type="fieldGroup"] > *,
.bn-container .bn-block-content[data-content-type="field"] > *,
```

And the fieldGroup side-menu hide rule (line 103):
```css
/* REMOVE fieldGroup from this selector */
.bn-container [data-content-type="fieldGroup"] > .bn-block-outer > .bn-block > .bn-side-menu,
```

- [ ] **Step 3: Verify TypeScript compiles**

Run: `cd frontend/apps/web && npx tsc --noEmit`
Expected: exit 0

- [ ] **Step 4: Run all tests**

Run: `cd frontend/apps/web && rtk vitest run`
Expected: All tests pass. If any tests reference Field/FieldGroup imports, update them.

- [ ] **Step 5: Commit**

```bash
rtk git add frontend/apps/web/src/features/documents/mddm-editor/schema.ts
rtk git add frontend/apps/web/src/features/documents/mddm-editor/mddm-editor-global.css
rtk git commit -m "refactor(mddm): remove field/fieldGroup from BlockNote schema — native table handles rendering"
```

---

### Task 7: Delete Unused Field/FieldGroup Block Files

**Files:**
- Delete: `frontend/apps/web/src/features/documents/mddm-editor/blocks/Field.tsx`
- Delete: `frontend/apps/web/src/features/documents/mddm-editor/blocks/Field.module.css`
- Delete: `frontend/apps/web/src/features/documents/mddm-editor/blocks/FieldGroup.tsx`
- Delete: `frontend/apps/web/src/features/documents/mddm-editor/blocks/FieldGroup.module.css`

- [ ] **Step 1: Delete the files**

```bash
rtk git rm frontend/apps/web/src/features/documents/mddm-editor/blocks/Field.tsx
rtk git rm frontend/apps/web/src/features/documents/mddm-editor/blocks/Field.module.css
rtk git rm frontend/apps/web/src/features/documents/mddm-editor/blocks/FieldGroup.tsx
rtk git rm frontend/apps/web/src/features/documents/mddm-editor/blocks/FieldGroup.module.css
```

- [ ] **Step 2: Check for any remaining imports**

Run: `grep -r "Field\|FieldGroup" frontend/apps/web/src/features/documents/mddm-editor/ --include="*.ts" --include="*.tsx" | grep -v "__tests__" | grep -v "node_modules" | grep -v "external-html" | grep -v adapter`

Fix any broken imports. The external-html renderers (`field-html.tsx`, `field-group-html.tsx`) should be kept — they're used by the DOCX/HTML export pipeline, not the editor.

- [ ] **Step 3: Verify TypeScript compiles**

Run: `cd frontend/apps/web && npx tsc --noEmit`
Expected: exit 0

- [ ] **Step 4: Run all tests**

Run: `cd frontend/apps/web && rtk vitest run`
Expected: All tests pass

- [ ] **Step 5: Commit**

```bash
rtk git commit -m "chore(mddm): delete unused Field/FieldGroup block components"
```

---

### Task 8: Visual Verification in Browser

**Files:** None (manual testing)

- [ ] **Step 1: Start API and dev server**

```bash
cd /c/Users/leandro.theodoro.MN-NTB-LEANDROT/Documents/MetalDocs
env $(cat .env | grep -v '^#' | xargs) ./metaldocs-api.exe &
npm run dev &
```

- [ ] **Step 2: Create a new PO document and verify field rendering**

Navigate to `http://localhost:4175`, create a new document. Verify:
- Section 1 "IDENTIFICAÇÃO" renders fields as a **native BlockNote table** with:
  - Visible horizontal and vertical borders
  - Label column with header styling (bold text, background color)
  - Value column editable
  - 4-column layout for 2-column fieldGroups (Label | Value | Label | Value)
- Section 2 "IDENTIFICAÇÃO DO PROCESSO" fields also render as bordered table
- All other sections render correctly

- [ ] **Step 3: Test editing**

Click in a value cell and type text. Verify:
- Text appears in the table cell
- Cell selection works (click, shift-click)
- Tab navigation between cells works

- [ ] **Step 4: Test save and reload**

Click "Salvar rascunho", reload the page. Verify:
- Field values are preserved
- Table structure is maintained
- No data loss

- [ ] **Step 5: Test DOCX export**

Click "Exportar DOCX". Open the downloaded file. Verify:
- Fields still render as bordered tables in DOCX
- Label cells have background color
- Values are preserved

---

## Scope Decisions

**In scope:**
- Converting inline fields (valueMode="inline") to native table cells
- Preserving all field metadata for roundtrip (id, template_block_id, label, locked, hint)
- Proper 1-column and 2-column layouts

**Out of scope (future work):**
- MultiParagraph fields (valueMode="multiParagraph") — these have child blocks that don't fit in table cells. For now, their text content is flattened to inline. A future task should handle these with a richer approach (possibly a different block type or expanding BlockNote's table cells).
- Column resize persistence — BlockNote handles this natively but we may want to lock label column widths
- Cell color customization UI — enabled but not actively exposed to users yet

---

## Risks & Mitigations

| Risk | Mitigation |
|------|-----------|
| MultiParagraph field data loss | Adapter flattens children to inline text with warning log. Existing multiParagraph content is preserved in MDDM storage — only the editor rendering simplifies it. |
| Existing documents with field/fieldGroup | Adapter converts at load time. Original MDDM data is never modified on disk unless user saves. |
| Regular `table` blocks (e.g., user inserts a table) collide with field tables | `__mddm_field_group` metadata distinguishes field-origin tables. Regular tables without this prop are handled normally. |
