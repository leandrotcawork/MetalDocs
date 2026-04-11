import { describe, expect, it } from "vitest";
import { Paragraph } from "docx";
import { emitQuote } from "../emitters/quote";
import { defaultLayoutTokens } from "../../layout-ir";
import type { MDDMBlock } from "../../../adapter";

describe("emitQuote", () => {
  it("emits a Paragraph with left indentation and italic styling", () => {
    const block: MDDMBlock = {
      id: "q1",
      type: "quote",
      props: {},
      children: [{ type: "text", text: "Quoted text" }],
    };
    const out = emitQuote(block, defaultLayoutTokens);
    expect(out).toHaveLength(1);
    expect(out[0]).toBeInstanceOf(Paragraph);
    const opts = (out[0] as any).options;
    expect(opts.indent).toBeDefined();
    expect(opts.indent.left).toBeGreaterThan(0);
  });
});
