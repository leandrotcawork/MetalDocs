import { describe, expect, it } from "vitest";
import { Paragraph } from "docx";
import { emitParagraph } from "../emitters/paragraph";
import { defaultLayoutTokens } from "../../layout-ir";
import type { MDDMBlock } from "../../../adapter";

describe("emitParagraph", () => {
  it("emits one docx Paragraph for a paragraph block with text runs", () => {
    const block: MDDMBlock = {
      id: "p1",
      type: "paragraph",
      props: {},
      children: [{ type: "text", text: "Hello" }],
    };
    const out = emitParagraph(block, defaultLayoutTokens);
    expect(out).toHaveLength(1);
    expect(out[0]).toBeInstanceOf(Paragraph);
  });

  it("honors bold marks from children text runs", () => {
    const block: MDDMBlock = {
      id: "p2",
      type: "paragraph",
      props: {},
      children: [{ type: "text", text: "Bold", marks: [{ type: "bold" }] }],
    };
    const out = emitParagraph(block, defaultLayoutTokens);
    expect(out).toHaveLength(1);
    expect((out[0] as any).options.children[0].options).toMatchObject({ bold: true });
  });

  it("emits an empty Paragraph when children is empty or missing", () => {
    const emptyChildren: MDDMBlock = { id: "p3", type: "paragraph", props: {}, children: [] };
    const noChildren: MDDMBlock = { id: "p4", type: "paragraph", props: {} };
    expect(emitParagraph(emptyChildren, defaultLayoutTokens)).toHaveLength(1);
    expect(emitParagraph(noChildren, defaultLayoutTokens)).toHaveLength(1);
  });
});
