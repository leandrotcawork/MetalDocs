import { describe, it, expect } from "vitest";
import { RichBlockCodec } from "../rich-block-codec";

describe("RichBlockCodec.parseCaps", () => {
  it("parses editableZones", () => {
    const caps = RichBlockCodec.parseCaps(JSON.stringify({ editableZones: ["content"] }));
    expect(caps.editableZones).toEqual(["content"]);
  });
  it("defaults locked to true", () => {
    const caps = RichBlockCodec.parseCaps("{}");
    expect(caps.locked).toBe(true);
  });
});

describe("RichBlockCodec.parseStyle", () => {
  it("parses label styling", () => {
    const style = RichBlockCodec.parseStyle(JSON.stringify({
      labelBackground: "#f9f3f3", labelFontSize: "10pt",
    }));
    expect(style.labelBackground).toBe("#f9f3f3");
  });
});
