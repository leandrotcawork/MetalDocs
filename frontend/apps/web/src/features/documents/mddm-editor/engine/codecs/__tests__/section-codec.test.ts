import { describe, it, expect } from "vitest";
import { SectionCodec } from "../section-codec";

describe("SectionCodec.parseStyle", () => {
  it("parses valid style JSON", () => {
    const style = SectionCodec.parseStyle('{"headerHeight":"12mm","headerBackground":"#ff0000"}');
    expect(style.headerHeight).toBe("12mm");
    expect(style.headerBackground).toBe("#ff0000");
  });

  it("returns empty style for empty JSON", () => {
    const style = SectionCodec.parseStyle("{}");
    expect(style.headerHeight).toBeUndefined();
    expect(style.headerBackground).toBeUndefined();
  });

  it("strips unknown fields", () => {
    const style = SectionCodec.parseStyle('{"headerHeight":"12mm","unknownField":"value"}');
    expect(style.headerHeight).toBe("12mm");
    expect((style as any).unknownField).toBeUndefined();
  });

  it("ignores non-string values for string fields", () => {
    const style = SectionCodec.parseStyle('{"headerHeight":42}');
    expect(style.headerHeight).toBeUndefined();
  });

  it("handles malformed JSON gracefully", () => {
    const style = SectionCodec.parseStyle("not json");
    expect(style).toEqual(SectionCodec.defaultStyle());
  });
});

describe("SectionCodec.parseCaps", () => {
  it("parses valid capabilities", () => {
    const caps = SectionCodec.parseCaps('{"locked":false,"removable":true}');
    expect(caps.locked).toBe(false);
    expect(caps.removable).toBe(true);
  });

  it("applies defaults for missing fields", () => {
    const caps = SectionCodec.parseCaps("{}");
    expect(caps.locked).toBe(true);
    expect(caps.removable).toBe(false);
    expect(caps.reorderable).toBe(false);
  });

  it("handles malformed JSON gracefully", () => {
    const caps = SectionCodec.parseCaps("broken");
    expect(caps).toEqual(SectionCodec.defaultCaps());
  });
});

describe("SectionCodec.serializeStyle", () => {
  it("round-trips through parse", () => {
    const original = { headerHeight: "12mm", headerBackground: "#ff0000" };
    const serialized = SectionCodec.serializeStyle(original);
    const parsed = SectionCodec.parseStyle(serialized);
    expect(parsed.headerHeight).toBe("12mm");
    expect(parsed.headerBackground).toBe("#ff0000");
  });

  it("strips undefined values", () => {
    const serialized = SectionCodec.serializeStyle({ headerHeight: "12mm" });
    expect(serialized).not.toContain("undefined");
  });
});
