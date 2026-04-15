import { describe, expect, it } from "vitest";
import { computePageLayout } from "../pagination";

describe("computePageLayout", () => {
  it("returns a single page with no break offsets when content does not overflow", () => {
    const result = computePageLayout({
      pageHeightPx: 1000,
      topMarginPx: 100,
      bottomMarginPx: 100,
      blocks: [
        { id: "a", topPx: 0, heightPx: 260 },
        { id: "b", topPx: 260, heightPx: 300 },
      ],
    });

    expect(result.pageCount).toBe(1);
    expect(result.breakOffsetsByBlockId).toEqual({});
  });

  it("creates a second page and reports break offset for the overflowing block", () => {
    const result = computePageLayout({
      pageHeightPx: 1000,
      topMarginPx: 100,
      bottomMarginPx: 100,
      blocks: [
        { id: "a", topPx: 0, heightPx: 320 },
        { id: "b", topPx: 320, heightPx: 320 },
        { id: "c", topPx: 640, heightPx: 220 },
      ],
    });

    expect(result.pageCount).toBe(2);
    expect(result.breakOffsetsByBlockId).toEqual({
      c: 160,
    });
  });

  it("keeps oversized blocks offset-free while still counting visual pages", () => {
    const result = computePageLayout({
      pageHeightPx: 1000,
      topMarginPx: 100,
      bottomMarginPx: 100,
      blocks: [{ id: "oversized", topPx: 0, heightPx: 1200 }],
    });

    expect(result.pageCount).toBe(2);
    expect(result.breakOffsetsByBlockId).toEqual({});
  });
});
