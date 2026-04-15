import { describe, expect, it } from "vitest";
import {
  clampPageMarginMm,
  defaultTemplatePageSettings,
  readTemplatePageSettings,
  type TemplatePageSettings,
  writeTemplatePageSettings,
} from "../page-settings";

describe("page settings", () => {
  it("falls back to defaults when meta.page is missing", () => {
    const fromUndefined = readTemplatePageSettings(undefined);
    expect(fromUndefined).toEqual(defaultTemplatePageSettings);
    expect(fromUndefined).not.toBe(defaultTemplatePageSettings);
    expect(readTemplatePageSettings(null)).toEqual(defaultTemplatePageSettings);
    expect(readTemplatePageSettings({})).toEqual(defaultTemplatePageSettings);
  });

  it("clamps margins to supported range", () => {
    expect(clampPageMarginMm(1)).toBe(5);
    expect(clampPageMarginMm(5)).toBe(5);
    expect(clampPageMarginMm(26)).toBe(26);
    expect(clampPageMarginMm(50)).toBe(50);
    expect(clampPageMarginMm(88)).toBe(50);
  });

  it("reads page settings from meta and clamps invalid values", () => {
    expect(
      readTemplatePageSettings({
        page: {
          marginTopMm: 1,
          marginRightMm: 20,
          marginBottomMm: 72,
          marginLeftMm: 11,
        },
      }),
    ).toEqual({
      marginTopMm: 5,
      marginRightMm: 20,
      marginBottomMm: 50,
      marginLeftMm: 11,
    });
  });

  it("preserves unrelated meta keys when writing page settings", () => {
    expect(
      writeTemplatePageSettings(
        { audit: { source: "import" }, page: { customPreset: "narrow" } },
        { marginTopMm: 10, marginRightMm: 11, marginBottomMm: 12, marginLeftMm: 13 },
      ),
    ).toEqual({
      audit: { source: "import" },
      page: {
        customPreset: "narrow",
        marginTopMm: 10,
        marginRightMm: 11,
        marginBottomMm: 12,
        marginLeftMm: 13,
      },
    });
  });

  it("writes non-finite margins using field-specific defaults", () => {
    expect(
      writeTemplatePageSettings(
        { page: { customPreset: "tight" } },
        {
          marginTopMm: Number.NaN,
          marginRightMm: Number.POSITIVE_INFINITY,
          marginBottomMm: Number.NaN,
          marginLeftMm: Number.NEGATIVE_INFINITY,
        } as TemplatePageSettings,
      ),
    ).toEqual({
      page: {
        customPreset: "tight",
        marginTopMm: 25,
        marginRightMm: 20,
        marginBottomMm: 25,
        marginLeftMm: 25,
      },
    });
  });

  it("preserves malformed existing meta.page by replacing with typed margins", () => {
    expect(
      writeTemplatePageSettings(
        { audit: { source: "manual" }, page: "invalid-shape" },
        { marginTopMm: 8, marginRightMm: 9, marginBottomMm: 10, marginLeftMm: 11 },
      ),
    ).toEqual({
      audit: { source: "manual" },
      page: {
        marginTopMm: 8,
        marginRightMm: 9,
        marginBottomMm: 10,
        marginLeftMm: 11,
      },
    });
  });
});
