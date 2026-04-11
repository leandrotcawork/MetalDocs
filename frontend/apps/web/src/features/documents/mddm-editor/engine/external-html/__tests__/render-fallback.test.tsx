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
    expect(html).toContain("inspect");
    expect(html.toLowerCase()).toContain("repeatable");
  });

  it("serializes a dataTable + dataTableRow + dataTableCell with cell text preserved", async () => {
    const envelope: MDDMEnvelope = {
      mddm_version: 1,
      template_ref: null,
      blocks: [
        {
          id: "t",
          type: "dataTable",
          props: {
            label: "Items",
            columns: [{ key: "c0", label: "Item" }],
            locked: true, minRows: 0, maxRows: 500, density: "normal",
          },
          children: [
            {
              id: "row1",
              type: "dataTableRow",
              props: {},
              children: [
                {
                  id: "cell1",
                  type: "dataTableCell",
                  props: { columnKey: "c0" },
                  children: [{ type: "text", text: "Parafuso" }],
                },
              ],
            },
          ],
        },
      ],
    };
    const html = await toHtml(envelope);
    expect(html).toContain("Parafuso");
  });
});
