import { describe, expect, it } from "vitest";
import { Table } from "docx";
import { emitField } from "../emitters/field";
import { defaultLayoutTokens } from "../../layout-ir";
import type { MDDMBlock } from "../../../adapter";

describe("emitField", () => {
  it("emits a Table with two cells using 35/65 split", () => {
    const block: MDDMBlock = {
      id: "f1",
      type: "field",
      props: { label: "Responsável" },
      children: [{ type: "text", text: "João Silva" }],
    };
    const out = emitField(block, defaultLayoutTokens);
    expect(out).toHaveLength(1);
    expect(out[0]).toBeInstanceOf(Table);

    const tableOptions = (out[0] as any).options;
    const firstRow = tableOptions.rows[0];
    const cells = firstRow.options.children;
    expect(cells).toHaveLength(2);

    // Cell widths use raw integer percentages for WidthType.PERCENTAGE (docx.js formats as "${size}%")
    expect(cells[0].options.width.size).toBe(35);
    expect(cells[1].options.width.size).toBe(65);
  });

  it("applies the accentLight background to the label cell", () => {
    const block: MDDMBlock = {
      id: "f2",
      type: "field",
      props: { label: "Label" },
      children: [],
    };
    const tokens = {
      ...defaultLayoutTokens,
      theme: { ...defaultLayoutTokens.theme, accentLight: "#ffeeff" },
    };
    const out = emitField(block, tokens);
    const labelCell = (out[0] as any).options.rows[0].options.children[0];
    expect(labelCell.options.shading.fill).toBe("FFEEFF");
  });

  it("renders the value cell with inline text runs from block.children", () => {
    const block: MDDMBlock = {
      id: "f3",
      type: "field",
      props: { label: "L" },
      children: [
        { type: "text", text: "Bold part", marks: [{ type: "bold" }] },
      ],
    };
    const out = emitField(block, defaultLayoutTokens);
    const valueCell = (out[0] as any).options.rows[0].options.children[1];
    const valueParagraph = valueCell.options.children[0];
    expect(valueParagraph.options.children[0].options).toMatchObject({ bold: true });
  });
});
