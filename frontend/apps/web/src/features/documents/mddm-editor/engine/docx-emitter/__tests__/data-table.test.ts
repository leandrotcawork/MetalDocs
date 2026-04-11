import { describe, expect, it } from "vitest";
import { Table } from "docx";
import { emitDataTable } from "../emitters/data-table";
import { defaultLayoutTokens } from "../../layout-ir";
import type { MDDMBlock } from "../../../adapter";

function makeRow(id: string, cellCount: number): MDDMBlock {
  return {
    id,
    type: "dataTableRow",
    props: {},
    children: Array.from({ length: cellCount }, (_, i) => ({
      id: `${id}-c${i}`,
      type: "dataTableCell",
      props: { columnKey: `col${i}` },
      children: [{ type: "text", text: `r${id}c${i}` }],
    })),
  };
}

describe("emitDataTable", () => {
  it("emits a single Table with header row + data rows", () => {
    const block: MDDMBlock = {
      id: "t1",
      type: "dataTable",
      props: {
        label: "Items",
        columns: [
          { key: "col0", label: "Item" },
          { key: "col1", label: "Qty" },
        ],
      },
      children: [makeRow("r1", 2), makeRow("r2", 2)],
    };
    const out = emitDataTable(block, defaultLayoutTokens);
    expect(out).toHaveLength(1);
    expect(out[0]).toBeInstanceOf(Table);

    // header row + 2 data rows
    const rows = (out[0] as any).options.rows;
    expect(rows).toHaveLength(3);
  });

  it("renders empty table when there are no columns and no rows", () => {
    const block: MDDMBlock = {
      id: "t2",
      type: "dataTable",
      props: { label: "X", columns: [] },
      children: [],
    };
    const out = emitDataTable(block, defaultLayoutTokens);
    expect(out).toHaveLength(1);
    expect(out[0]).toBeInstanceOf(Table);
  });

  it("falls back gracefully when columns prop is missing or not an array", () => {
    const block: MDDMBlock = {
      id: "t3",
      type: "dataTable",
      props: { label: "X" },
      children: [makeRow("r1", 1)],
    };
    expect(() => emitDataTable(block, defaultLayoutTokens)).not.toThrow();
  });
});
