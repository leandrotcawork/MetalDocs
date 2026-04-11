import { describe, expect, it } from "vitest";
import { Paragraph } from "docx";
import { emitBulletListItem } from "../emitters/bullet-list-item";
import { defaultLayoutTokens } from "../../layout-ir";
import type { MDDMBlock } from "../../../adapter";

describe("emitBulletListItem", () => {
  it("emits a Paragraph with bullet numbering", () => {
    const block: MDDMBlock = {
      id: "b1",
      type: "bulletListItem",
      props: {},
      children: [{ type: "text", text: "First" }],
    };
    const out = emitBulletListItem(block, defaultLayoutTokens);
    expect(out).toHaveLength(1);
    expect(out[0]).toBeInstanceOf(Paragraph);
    expect((out[0] as any).options.bullet).toBeDefined();
    expect((out[0] as any).options.bullet.level).toBe(0);
  });

  it("preserves marks on text runs", () => {
    const block: MDDMBlock = {
      id: "b2",
      type: "bulletListItem",
      props: {},
      children: [{ type: "text", text: "Bold", marks: [{ type: "bold" }] }],
    };
    const out = emitBulletListItem(block, defaultLayoutTokens);
    const run = (out[0] as any).options.children[0];
    expect(run.options).toMatchObject({ bold: true });
  });
});
