import { describe, expect, it } from "vitest";
import { TableRow } from "docx";
import { emitDataTableRow } from "../emitters/data-table-row";
import { defaultLayoutTokens } from "../../layout-ir";
import type { MDDMBlock } from "../../../adapter";

describe("emitDataTableRow", () => {
  it("emits a TableRow containing one cell per dataTableCell child", () => {
    const block: MDDMBlock = {
      id: "r1",
      type: "dataTableRow",
      props: {},
      children: [
        { id: "c1", type: "dataTableCell", props: { columnKey: "a" }, children: [{ type: "text", text: "1" }] },
        { id: "c2", type: "dataTableCell", props: { columnKey: "b" }, children: [{ type: "text", text: "2" }] },
      ],
    };
    const out = emitDataTableRow(block, defaultLayoutTokens);
    expect(out).toBeInstanceOf(TableRow);
    expect((out as any).options.children).toHaveLength(2);
  });
});
