import { describe, expect, it } from "vitest";
import { Paragraph, HeadingLevel } from "docx";
import { emitHeading } from "../emitters/heading";
import { defaultLayoutTokens } from "../../layout-ir";
import type { MDDMBlock } from "../../shared/adapter";

describe("emitHeading", () => {
  it("emits a Paragraph with HEADING_1 for level 1", () => {
    const block: MDDMBlock = {
      id: "h1",
      type: "heading",
      props: { level: 1 },
      children: [{ type: "text", text: "Title" }],
    };
    const out = emitHeading(block, defaultLayoutTokens);
    expect(out).toHaveLength(1);
    expect(out[0]).toBeInstanceOf(Paragraph);
    expect((out[0] as any).options.heading).toBe(HeadingLevel.HEADING_1);
  });

  it("emits HEADING_2 for level 2", () => {
    const block: MDDMBlock = {
      id: "h2",
      type: "heading",
      props: { level: 2 },
      children: [{ type: "text", text: "Sub" }],
    };
    const out = emitHeading(block, defaultLayoutTokens);
    expect((out[0] as any).options.heading).toBe(HeadingLevel.HEADING_2);
  });

  it("defaults to HEADING_1 when level is missing or invalid", () => {
    const block: MDDMBlock = {
      id: "h3",
      type: "heading",
      props: {},
      children: [{ type: "text", text: "Default" }],
    };
    const out = emitHeading(block, defaultLayoutTokens);
    expect((out[0] as any).options.heading).toBe(HeadingLevel.HEADING_1);
  });
});

