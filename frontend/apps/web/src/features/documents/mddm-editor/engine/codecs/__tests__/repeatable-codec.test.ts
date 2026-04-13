import { describe, it, expect } from "vitest";
import { RepeatableCodec } from "../repeatable-codec";

describe("RepeatableCodec.parseCaps", () => {
  it("parses addItems/removeItems/maxItems/minItems", () => {
    const caps = RepeatableCodec.parseCaps(JSON.stringify({
      addItems: true, removeItems: false, maxItems: 20, minItems: 1,
    }));
    expect(caps.addItems).toBe(true);
    expect(caps.removeItems).toBe(false);
    expect(caps.maxItems).toBe(20);
    expect(caps.minItems).toBe(1);
  });
  it("defaults maxItems to 100, minItems to 0", () => {
    const caps = RepeatableCodec.parseCaps("{}");
    expect(caps.maxItems).toBe(100);
    expect(caps.minItems).toBe(0);
  });
});

describe("RepeatableCodec.parseStyle", () => {
  it("parses border and accent styles", () => {
    const style = RepeatableCodec.parseStyle(JSON.stringify({
      borderColor: "#dfc8c8", itemAccentBorder: "#6b1f2a", itemAccentWidth: "3pt",
    }));
    expect(style.borderColor).toBe("#dfc8c8");
    expect(style.itemAccentBorder).toBe("#6b1f2a");
  });
});
