import { describe, expect, it } from "vitest";
import { Paragraph } from "docx";
import { emitDivider } from "../emitters/divider";
import { defaultLayoutTokens } from "../../layout-ir";
import type { MDDMBlock } from "../../shared/adapter";

describe("emitDivider", () => {
  it("emits a Paragraph with a bottom border (horizontal rule)", () => {
    const block: MDDMBlock = { id: "d1", type: "divider", props: {}, children: [] };
    const out = emitDivider(block, defaultLayoutTokens);
    expect(out).toHaveLength(1);
    expect(out[0]).toBeInstanceOf(Paragraph);
    const opts = (out[0] as any).options;
    expect(opts.border).toBeDefined();
    expect(opts.border.bottom).toBeDefined();
    // defaultLayoutTokens.theme.accentBorder = "#dfc8c8" â†’ stripped + uppercased
    expect(opts.border.bottom.color).toBe("DFC8C8");
  });
});

