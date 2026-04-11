import { describe, expect, it } from "vitest";
import { TableCell } from "docx";
import { emitDataTableCell } from "../emitters/data-table-cell";
import { defaultLayoutTokens } from "../../layout-ir";
import type { MDDMBlock } from "../../../adapter";

describe("emitDataTableCell", () => {
  it("emits a TableCell containing a Paragraph with text runs", () => {
    const block: MDDMBlock = {
      id: "c1",
      type: "dataTableCell",
      props: { columnKey: "qty" },
      children: [{ type: "text", text: "100" }],
    };
    const out = emitDataTableCell(block, defaultLayoutTokens);
    expect(out).toBeInstanceOf(TableCell);
    expect((out as any).options.children).toHaveLength(1);
  });

  it("renders empty cell when there are no text runs", () => {
    const block: MDDMBlock = {
      id: "c2",
      type: "dataTableCell",
      props: { columnKey: "x" },
      children: [],
    };
    const out = emitDataTableCell(block, defaultLayoutTokens);
    expect(out).toBeInstanceOf(TableCell);
  });
});
