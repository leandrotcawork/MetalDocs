import { describe, it, expect } from "vitest";
import { DataTableCodec } from "../data-table-codec";

describe("DataTableCodec.parseStyle", () => {
  it("parses all style fields", () => {
    const style = DataTableCodec.parseStyle(
      JSON.stringify({
        headerBackground: "#f9f3f3",
        headerColor: "#3e1018",
        headerFontWeight: "bold",
        cellBorderColor: "#dfc8c8",
        cellPadding: "2mm",
        density: "compact",
      }),
    );
    expect(style.headerBackground).toBe("#f9f3f3");
    expect(style.density).toBe("compact");
  });

  it("rejects invalid density values", () => {
    const style = DataTableCodec.parseStyle('{"density":"huge"}');
    expect(style.density).toBeUndefined();
  });
});

describe("DataTableCodec.parseCaps", () => {
  it("parses fixed mode", () => {
    const caps = DataTableCodec.parseCaps(
      JSON.stringify({
        locked: true,
        mode: "fixed",
        addRows: false,
        removeRows: false,
      }),
    );
    expect(caps.mode).toBe("fixed");
    expect(caps.addRows).toBe(false);
  });

  it("parses dynamic mode with maxRows", () => {
    const caps = DataTableCodec.parseCaps(
      JSON.stringify({
        mode: "dynamic",
        addRows: true,
        removeRows: true,
        maxRows: 50,
      }),
    );
    expect(caps.mode).toBe("dynamic");
    expect(caps.addRows).toBe(true);
    expect(caps.maxRows).toBe(50);
  });

  it("defaults to dynamic mode", () => {
    const caps = DataTableCodec.parseCaps("{}");
    expect(caps.mode).toBe("dynamic");
    expect(caps.maxRows).toBe(100);
  });

  it("rejects invalid mode values", () => {
    const caps = DataTableCodec.parseCaps('{"mode":"weird"}');
    expect(caps.mode).toBe("dynamic");
  });
});

describe("DataTableCodec round-trip", () => {
  it("serializes and parses back identically", () => {
    const original = { headerBackground: "#ff0000", density: "compact" as const };
    const serialized = DataTableCodec.serializeStyle(original);
    const parsed = DataTableCodec.parseStyle(serialized);
    expect(parsed.headerBackground).toBe("#ff0000");
    expect(parsed.density).toBe("compact");
  });
});
