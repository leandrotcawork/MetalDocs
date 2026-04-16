import { describe, expect, it } from "vitest";
import { emitRichBlock } from "../emitters/rich-block";
import { defaultLayoutTokens } from "../../layout-ir";
import type { MDDMBlock } from "../../shared/adapter";

describe("emitRichBlock", () => {
  it("emits an optional label paragraph plus rendered children", () => {
    const block: MDDMBlock = {
      id: "rb1",
      type: "richBlock",
      props: { label: "Notes", chrome: "labeled" },
      children: [
        { id: "p1", type: "paragraph", props: {}, children: [{ type: "text", text: "note" }] },
      ],
    };
    const renderedChildren: unknown[] = [{ marker: "p1" }];
    const out = emitRichBlock(block, defaultLayoutTokens, () => renderedChildren);
    // Label paragraph + at least 1 child element
    expect(out.length).toBeGreaterThanOrEqual(2);
  });

  it("skips the label paragraph when label is missing", () => {
    const block: MDDMBlock = {
      id: "rb2",
      type: "richBlock",
      props: {},
      children: [],
    };
    const out = emitRichBlock(block, defaultLayoutTokens, () => []);
    // No label, no children â†’ empty array
    expect(out).toEqual([]);
  });
});

