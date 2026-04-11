import { describe, expect, it } from "vitest";
import { COMPATIBILITY_CONTRACT, isForbiddenConstruct } from "../compatibility-contract";

describe("Compatibility contract", () => {
  it("defines tier2 tolerance < 2% for editor vs PDF", () => {
    expect(COMPATIBILITY_CONTRACT.tier2.pixelDiffEditorToPdf).toBeLessThan(0.02 + Number.EPSILON);
  });

  it("defines tier2 tolerance < 5% for editor vs DOCX", () => {
    expect(COMPATIBILITY_CONTRACT.tier2.pixelDiffEditorToDocx).toBeLessThan(0.05 + Number.EPSILON);
  });

  it("forbids auto-fit columns", () => {
    expect(isForbiddenConstruct("autoFitColumns")).toBe(true);
  });

  it("forbids Flexbox layouts", () => {
    expect(isForbiddenConstruct("flexbox")).toBe(true);
  });

  it("forbids unitless line-heights", () => {
    expect(isForbiddenConstruct("unitlessLineHeight")).toBe(true);
  });

  it("caps nested DataTable depth at 2 levels", () => {
    expect(COMPATIBILITY_CONTRACT.forbidden.nestedDataTableMaxDepth).toBe(2);
  });

  it("allows known-safe constructs", () => {
    expect(isForbiddenConstruct("absoluteLineHeight")).toBe(false);
    expect(isForbiddenConstruct("explicitColumnWidths")).toBe(false);
  });
});
