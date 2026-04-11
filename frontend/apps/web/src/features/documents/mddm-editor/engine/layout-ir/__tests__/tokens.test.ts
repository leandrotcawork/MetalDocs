import { describe, expect, it } from "vitest";
import { defaultLayoutTokens, type LayoutTokens } from "../tokens";

describe("Layout IR tokens", () => {
  it("provides A4 page dimensions in mm", () => {
    expect(defaultLayoutTokens.page.widthMm).toBe(210);
    expect(defaultLayoutTokens.page.heightMm).toBe(297);
  });

  it("computes contentWidthMm from page width minus horizontal margins", () => {
    const { page } = defaultLayoutTokens;
    expect(page.contentWidthMm).toBe(page.widthMm - page.marginLeftMm - page.marginRightMm);
  });

  it("uses Carlito as the default exportFont", () => {
    expect(defaultLayoutTokens.typography.exportFont).toBe("Carlito");
  });

  it("uses absolute lineHeightPt (no unitless line-heights)", () => {
    expect(typeof defaultLayoutTokens.typography.lineHeightPt).toBe("number");
    expect(defaultLayoutTokens.typography.lineHeightPt).toBeGreaterThan(0);
  });

  it("has theme accent colors", () => {
    expect(defaultLayoutTokens.theme.accent).toMatch(/^#[0-9a-fA-F]{6}$/);
    expect(defaultLayoutTokens.theme.accentLight).toMatch(/^#[0-9a-fA-F]{6}$/);
    expect(defaultLayoutTokens.theme.accentDark).toMatch(/^#[0-9a-fA-F]{6}$/);
    expect(defaultLayoutTokens.theme.accentBorder).toMatch(/^#[0-9a-fA-F]{6}$/);
  });

  it("is readonly-typed (compile-time check)", () => {
    const tokens: LayoutTokens = defaultLayoutTokens;
    expect(tokens).toBe(defaultLayoutTokens);
  });
});
