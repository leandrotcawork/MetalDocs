// MDDM Layout IR — design tokens shared between React, DOCX, and PDF renderers.
// All dimensions are absolute (mm/pt). No unitless or relative values.

export type LayoutTokens = Readonly<{
  page: Readonly<{
    widthMm: number;
    heightMm: number;
    marginTopMm: number;
    marginRightMm: number;
    marginBottomMm: number;
    marginLeftMm: number;
    contentWidthMm: number;
  }>;
  typography: Readonly<{
    editorFont: string;
    exportFont: string;
    exportFontFallbacks: readonly string[];
    baseSizePt: number;
    headingSizePt: Readonly<{ h1: number; h2: number; h3: number }>;
    lineHeightPt: number;
    labelSizePt: number;
  }>;
  spacing: Readonly<{
    sectionGapMm: number;
    fieldGapMm: number;
    blockGapMm: number;
    cellPaddingMm: number;
  }>;
  theme: Readonly<{
    accent: string;
    accentLight: string;
    accentDark: string;
    accentBorder: string;
  }>;
}>;

const PAGE_WIDTH_MM = 210;
const PAGE_HEIGHT_MM = 297;
const MARGIN_TOP_MM = 25;
const MARGIN_RIGHT_MM = 20;
const MARGIN_BOTTOM_MM = 25;
const MARGIN_LEFT_MM = 25;

export const defaultLayoutTokens: LayoutTokens = {
  page: {
    widthMm: PAGE_WIDTH_MM,
    heightMm: PAGE_HEIGHT_MM,
    marginTopMm: MARGIN_TOP_MM,
    marginRightMm: MARGIN_RIGHT_MM,
    marginBottomMm: MARGIN_BOTTOM_MM,
    marginLeftMm: MARGIN_LEFT_MM,
    contentWidthMm: PAGE_WIDTH_MM - MARGIN_LEFT_MM - MARGIN_RIGHT_MM,
  },
  typography: {
    editorFont: "Inter",
    exportFont: "Carlito",
    exportFontFallbacks: ["Liberation Sans", "Arial", "sans-serif"],
    baseSizePt: 11,
    headingSizePt: { h1: 18, h2: 15, h3: 13 },
    lineHeightPt: 15,
    labelSizePt: 9,
  },
  spacing: {
    sectionGapMm: 6,
    fieldGapMm: 3,
    blockGapMm: 2,
    cellPaddingMm: 2,
  },
  theme: {
    accent: "#6b1f2a",
    accentLight: "#f9f3f3",
    accentDark: "#3e1018",
    accentBorder: "#dfc8c8",
  },
};
