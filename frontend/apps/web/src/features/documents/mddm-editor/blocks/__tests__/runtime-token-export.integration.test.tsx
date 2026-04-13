import { BlockNoteSchema, defaultBlockSpecs } from "@blocknote/core";
import { ServerBlockNoteEditor } from "@blocknote/server-util";
import { describe, expect, it } from "vitest";
import { setEditorTokens } from "../../engine/editor-tokens";
import { defaultLayoutTokens } from "../../engine/layout-ir";
import { mddmSchemaBlockSpecs } from "../../schema";

const runtimeTokens = {
  ...defaultLayoutTokens,
  theme: {
    ...defaultLayoutTokens.theme,
    accent: "#123456",
    accentLight: "#ddeeff",
    accentDark: "#112233",
    accentBorder: "#654321",
  },
};

function hexToRgbString(hex: string): string {
  const normalized = hex.replace("#", "");
  const [r, g, b] = [
    normalized.slice(0, 2),
    normalized.slice(2, 4),
    normalized.slice(4, 6),
  ].map((value) => parseInt(value, 16));
  return `rgb(${r}, ${g}, ${b})`;
}

async function exportLossyHTML(blocks: any[]): Promise<string> {
  const schema = BlockNoteSchema.create({
    blockSpecs: {
      ...defaultBlockSpecs,
      ...mddmSchemaBlockSpecs,
    },
  });
  const editor = ServerBlockNoteEditor.create({ schema });
  Object.defineProperty(editor.editor, "document", {
    configurable: true,
    value: blocks,
  });
  setEditorTokens(editor.editor, runtimeTokens);
  return editor.blocksToHTMLLossy(blocks);
}

describe("blocksToHTMLLossy runtime token export threading", () => {
  it("threads runtime accent tokens through block exports and preserves repeatable item numbering", async () => {
    const html = await exportLossyHTML([
      {
        id: "section-1",
        type: "section",
        props: { title: "Introduction" },
      },
      {
        id: "table-1",
        type: "dataTable",
        props: { label: "Items", locked: true, density: "normal" },
        content: {
          type: "tableContent",
          headerRows: 1,
          rows: [
            { cells: [[{ type: "text", text: "Name" }]] },
            { cells: [[{ type: "text", text: "Bolt" }]] },
          ],
        },
      },
      {
        id: "repeatable-1",
        type: "repeatable",
        props: { label: "Steps", itemPrefix: "Step" },
        children: [
          { id: "item-1", type: "repeatableItem", props: { title: "Prepare" }, children: [] },
          { id: "item-2", type: "repeatableItem", props: { title: "Inspect" }, children: [] },
        ],
      },
      {
        id: "rich-1",
        type: "richBlock",
        props: { label: "Notes", chrome: "labeled" },
        children: [],
      },
    ]);

    expect(html.toLowerCase()).toContain(hexToRgbString(runtimeTokens.theme.accent));
    expect(html.toLowerCase()).toContain(hexToRgbString(runtimeTokens.theme.accentLight));
    expect(html.toLowerCase()).toContain(hexToRgbString(runtimeTokens.theme.accentDark));
    expect(html.toLowerCase()).toContain(hexToRgbString(runtimeTokens.theme.accentBorder));
    expect(html).toContain("2. Inspect");
    expect(html).not.toContain("1. Inspect");
  });
});
