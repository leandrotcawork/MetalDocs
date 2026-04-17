// Unit conversions for OOXML / docx.js output.
// Reference: ECMA-376 Part 1 §17.18.74 (ST_TwipsMeasure),
// §17.3.1.13 (ST_HalfPoint), §20.1.2.1 (ST_EmuAbsMeasure).

// 1 inch = 25.4 mm = 1440 twips (twentieths of a point)
const TWIPS_PER_MM = 1440 / 25.4;

// 1 inch = 914400 EMU (English Metric Units)
const EMU_PER_MM = 914400 / 25.4;

// 1 inch = 72 points
const POINTS_PER_MM = 72 / 25.4;

export function mmToTwip(mm: number): number {
  return Math.round(mm * TWIPS_PER_MM);
}

export function mmToEmu(mm: number): number {
  return Math.round(mm * EMU_PER_MM);
}

export function mmToPt(mm: number): number {
  return mm * POINTS_PER_MM;
}

// OOXML font sizes are stored in half-points (so size 22 = 11pt)
export function ptToHalfPt(pt: number): number {
  return Math.round(pt * 2);
}

// OOXML table column widths can be expressed in fiftieths of a percent (0-5000)
// when type="pct". 100% = 5000.
//
// WARNING: Do NOT use this with docx.js WidthType.PERCENTAGE. That API
// accepts a plain integer percentage (0-100) and formats it as "${size}%"
// internally. Passing fiftieths (e.g. 1750) would produce "1750%" — invalid
// OOXML. This function is only valid when writing raw OOXML w:w attributes
// with type="pct" directly (i.e. bypassing docx.js width helpers).
export function percentToTablePct(percent: number): number {
  const clamped = Math.max(0, Math.min(100, percent));
  return Math.round(clamped * 50);
}
