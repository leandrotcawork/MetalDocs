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

async function toHtml(blocks: any[]) {
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

describe("MDDM block toExternalHTML runtime token threading", () => {
  it("threads runtime tokens through Field, DataTable, Repeatable, RepeatableItem, and RichBlock", async () => {
    const accentBorderRgb = hexToRgbString(runtimeTokens.theme.accentBorder);
    const accentLightRgb = hexToRgbString(runtimeTokens.theme.accentLight);
    const accentRgb = hexToRgbString(runtimeTokens.theme.accent);
    const defaultAccentBorderRgb = hexToRgbString(defaultLayoutTokens.theme.accentBorder);
    const defaultAccentLightRgb = hexToRgbString(defaultLayoutTokens.theme.accentLight);
    const defaultAccentRgb = hexToRgbString(defaultLayoutTokens.theme.accent);
    const html = await toHtml([
      {
        id: "field-1",
        type: "field",
        props: { label: "Owner" },
        content: [{ type: "text", text: "Leandro", styles: {} }],
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

    expect(html.toLowerCase()).toContain(accentBorderRgb);
    expect(html.toLowerCase()).toContain(accentLightRgb);
    expect(html.toLowerCase()).toContain(accentRgb);
    expect(html).toContain("2. Inspect");
    expect(html.toLowerCase()).not.toContain(defaultAccentBorderRgb);
    expect(html.toLowerCase()).not.toContain(defaultAccentLightRgb);
    expect(html.toLowerCase()).not.toContain(defaultAccentRgb);
  });

  it("uses runtime tokens and section position from editor.document for Section", async () => {
    const accentRgb = hexToRgbString(runtimeTokens.theme.accent);
    const defaultAccentRgb = hexToRgbString(defaultLayoutTokens.theme.accent);
    const html = await toHtml([
      { id: "section-1", type: "section", props: { title: "Intro" } },
      { id: "paragraph-1", type: "paragraph", props: {} },
      { id: "section-2", type: "section", props: { title: "Safety" } },
    ]);

    expect(html).toContain("2. Safety");
    expect(html.toLowerCase()).toContain(accentRgb);
    expect(html.toLowerCase()).not.toContain(defaultAccentRgb);
  });
});
