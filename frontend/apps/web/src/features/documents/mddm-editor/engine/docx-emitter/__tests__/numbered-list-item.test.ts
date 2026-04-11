import { describe, expect, it } from "vitest";
import { Paragraph } from "docx";
import { emitNumberedListItem } from "../emitters/numbered-list-item";
import { defaultLayoutTokens } from "../../layout-ir";
import type { MDDMBlock } from "../../../adapter";

describe("emitNumberedListItem", () => {
  it("emits a Paragraph with numbering reference", () => {
    const block: MDDMBlock = {
      id: "n1",
      type: "numberedListItem",
      props: {},
      children: [{ type: "text", text: "Item 1" }],
    };
    const out = emitNumberedListItem(block, defaultLayoutTokens);
    expect(out).toHaveLength(1);
    expect(out[0]).toBeInstanceOf(Paragraph);
    expect((out[0] as any).options.numbering).toBeDefined();
    expect((out[0] as any).options.numbering.level).toBe(0);
  });
});
