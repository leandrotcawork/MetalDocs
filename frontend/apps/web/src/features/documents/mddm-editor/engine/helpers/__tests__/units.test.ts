import { describe, expect, it } from "vitest";
import { mmToTwip, ptToHalfPt, mmToEmu, mmToPt, percentToTablePct } from "../units";

describe("Unit conversions", () => {
  it("mmToTwip: 25mm equals 1417 twips (OOXML twentieths of a point)", () => {
    expect(mmToTwip(25)).toBe(1417);
  });

  it("mmToTwip: 0mm is 0 twips", () => {
    expect(mmToTwip(0)).toBe(0);
  });

  it("ptToHalfPt: 11pt equals 22 half-points", () => {
    expect(ptToHalfPt(11)).toBe(22);
  });

  it("ptToHalfPt: rounds to nearest integer", () => {
    expect(ptToHalfPt(11.25)).toBe(23);
  });

  it("mmToEmu: 10mm equals 360000 EMU", () => {
    expect(mmToEmu(10)).toBe(360000);
  });

  it("mmToPt: 10mm equals 28.35pt approximately", () => {
    expect(mmToPt(10)).toBeCloseTo(28.35, 2);
  });

  it("percentToTablePct: 35 percent equals 1750 fiftieths", () => {
    expect(percentToTablePct(35)).toBe(1750);
  });

  it("percentToTablePct: clamps out-of-range values", () => {
    expect(percentToTablePct(150)).toBe(5000);
    expect(percentToTablePct(-10)).toBe(0);
  });
});
