import { describe, expect, it } from "vitest";
import { rasterizePdfFirstPageToPng } from "../pixel-diff";

describe("pixel-diff smoke", () => {
  it("imports and exposes rasterizePdfFirstPageToPng", () => {
    expect(typeof rasterizePdfFirstPageToPng).toBe("function");
  });
});
