import { describe, expect, it } from "vitest";
import { BlockNoteSchema, defaultBlockSpecs } from "@blocknote/core";
import { ServerBlockNoteEditor } from "@blocknote/server-util";
import { mddmSchemaBlockSpecs } from "../../../schema";
import { mddmToBlockNote, type MDDMEnvelope } from "../../../adapter";

async function toHtml(envelope: MDDMEnvelope): Promise<string> {
  const schema = BlockNoteSchema.create({
    blockSpecs: {
      ...defaultBlockSpecs,
      ...mddmSchemaBlockSpecs,
    },
  });
  const editor = ServerBlockNoteEditor.create({ schema });
  const blocks = mddmToBlockNote(envelope);
  return await editor.blocksToFullHTML(blocks as any);
}

describe('blocksToFullHTML render fallback for MDDM content:"none" blocks', () => {
  it("serializes a repeatable + repeatableItem + nested paragraph with text preserved", async () => {
    const envelope: MDDMEnvelope = {
      mddm_version: 1,
      template_ref: null,
      blocks: [
        {
          id: "r",
          type: "repeatable",
          props: { label: "Steps", itemPrefix: "Step" },
          children: [
            {
              id: "ri",
              type: "repeatableItem",
              props: { title: "Step 1" },
              children: [
                { id: "p", type: "paragraph", props: {}, children: [{ type: "text", text: "inspect" }] },
              ],
            },
          ],
        },
      ],
    };
    const html = await toHtml(envelope);
    // Primary check: leaf text preserved through content:"none" block nesting.
    expect(html).toContain("inspect");
  });

  it("serializes a dataTable with tableContent (new format) with cell text preserved", async () => {
    const envelope: MDDMEnvelope = {
      mddm_version: 1,
      template_ref: null,
      blocks: [
        {
          id: "t",
          type: "dataTable",
          props: {
            label: "Items",
            locked: true,
            density: "normal",
          },
          content: {
            type: "tableContent",
            columnWidths: [null],
            headerRows: 1,
            rows: [
              { cells: [[{ type: "text" as const, text: "Item" }]] },
              { cells: [[{ type: "text" as const, text: "Parafuso" }]] },
            ],
          },
          children: [],
        },
      ],
    };
    const html = await toHtml(envelope);
    expect(html).toContain("Parafuso");
  });

});
