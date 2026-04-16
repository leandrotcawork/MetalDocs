import { describe, expect, it } from "vitest";
import { Table } from "docx";
import { emitDataTable } from "../emitters/data-table";
import { defaultLayoutTokens } from "../../layout-ir";
import type { MDDMBlock } from "../../shared/adapter";

function makeTableContent(headerRows: number, rowCount: number, colCount: number) {
  return {
    type: "tableContent",
    columnWidths: Array(colCount).fill(null),
    headerRows,
    rows: Array.from({ length: rowCount }, (_, r) => ({
      cells: Array.from({ length: colCount }, (_, c) => [
        { type: "text", text: `r${r}c${c}` },
      ]),
    })),
  };
}

describe("emitDataTable", () => {
  it("emits a single Table with header row + data rows", () => {
    const block: MDDMBlock = {
      id: "t1",
      type: "dataTable",
      props: { label: "Items" },
      content: makeTableContent(1, 3, 2), // 1 header + 2 data rows
      children: [],
    };
    const out = emitDataTable(block, defaultLayoutTokens);
    expect(out).toHaveLength(1);
    expect(out[0]).toBeInstanceOf(Table);

    // header row + 2 data rows = 3 total
    const rows = (out[0] as any).options.rows;
    expect(rows).toHaveLength(3);
  });

  it("renders empty table when there is no tableContent", () => {
    const block: MDDMBlock = {
      id: "t2",
      type: "dataTable",
      props: { label: "X" },
      children: [],
    };
    const out = emitDataTable(block, defaultLayoutTokens);
    expect(out).toHaveLength(1);
    expect(out[0]).toBeInstanceOf(Table);
  });

  it("renders empty table when rows is empty", () => {
    const block: MDDMBlock = {
      id: "t3",
      type: "dataTable",
      props: { label: "X" },
      content: { type: "tableContent", headerRows: 1, rows: [] },
      children: [],
    };
    const out = emitDataTable(block, defaultLayoutTokens);
    expect(out).toHaveLength(1);
    expect(out[0]).toBeInstanceOf(Table);
  });

  it("does not throw when content is present but rows have no cells", () => {
    const block: MDDMBlock = {
      id: "t4",
      type: "dataTable",
      props: { label: "X" },
      content: {
        type: "tableContent",
        headerRows: 1,
        rows: [{ cells: [] }, { cells: [] }],
      },
      children: [],
    };
    expect(() => emitDataTable(block, defaultLayoutTokens)).not.toThrow();
  });
});

