// MDDM Layout IR - design tokens shared between React, DOCX, and PDF renderers.
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
  screenSpacing: Readonly<{
    sectionGap: string;
    fieldGap: string;
    blockGap: string;
    cellPadding: string;
  }>;
  theme: Readonly<{
    accent: string;
    accentLight: string;
    accentDark: string;
    accentBorder: string;
    hyperlink: string;
  }>;
}>;

export type SectionRule = Readonly<{
  headerHeightMm: number;
  headerFontSizePt: number;
  headerFontWeight: "bold" | "normal";
  headerFontColor: string;
  headerBackgroundToken: "theme.accent";
  fullWidth: true;
}>;

export type FieldRule = Readonly<{
  labelWidthPercent: number;
  valueWidthPercent: number;
  labelBackgroundToken: "theme.accentLight";
  labelFontSizePt: number;
  borderColorToken: "theme.accentBorder";
  borderWidthPt: number;
  minHeightMm: number;
}>;

export type FieldGroupRule = Readonly<{
  defaultColumns: 1 | 2;
  gapMm: number;
  fullWidth: true;
}>;

export type DataTableRule = Readonly<{
  headerBackgroundToken: "theme.accentLight";
  headerFontColor: string;
  headerFontWeight: "bold" | "normal";
  cellBorderColorToken: "theme.accentBorder";
  cellPaddingMm: number;
  defaultDensity: "normal" | "compact";
}>;

export type RepeatableRule = Readonly<{
  borderColorToken: "theme.accentBorder";
  itemAccentBorderToken: "theme.accent";
  itemAccentWidthPt: number;
}>;

export type RichBlockRule = Readonly<{
  labelBackgroundToken: "theme.accentLight";
  labelFontSizePt: number;
  labelFontColor: string;
  borderColorToken: "theme.accentBorder";
}>;

export type ComponentRules = Readonly<{
  section: SectionRule;
  field: FieldRule;
  fieldGroup: FieldGroupRule;
  dataTable: DataTableRule;
  repeatable: RepeatableRule;
  richBlock: RichBlockRule;
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
  screenSpacing: {
    sectionGap: "1rem",
    fieldGap: "0.75rem",
    blockGap: "2px",
    cellPadding: "0.5rem",
  },
  theme: {
    accent: "#6b1f2a",
    accentLight: "#f9f3f3",
    accentDark: "#3e1018",
    accentBorder: "#dfc8c8",
    hyperlink: "#0563C1",
  },
};

export const defaultComponentRules: ComponentRules = {
  section: {
    headerHeightMm: 8,
    headerFontSizePt: 13,
    headerFontWeight: "bold",
    headerFontColor: "#ffffff",
    headerBackgroundToken: "theme.accent",
    fullWidth: true,
  },
  field: {
    labelWidthPercent: 35,
    valueWidthPercent: 65,
    labelBackgroundToken: "theme.accentLight",
    labelFontSizePt: 9,
    borderColorToken: "theme.accentBorder",
    borderWidthPt: 0.5,
    minHeightMm: 7,
  },
  fieldGroup: {
    defaultColumns: 2,
    gapMm: 0,
    fullWidth: true,
  },
  dataTable: {
    headerBackgroundToken: "theme.accentLight",
    headerFontColor: "#3e1018",
    headerFontWeight: "bold",
    cellBorderColorToken: "theme.accentBorder",
    cellPaddingMm: 2,
    defaultDensity: "normal",
  },
  repeatable: {
    borderColorToken: "theme.accentBorder",
    itemAccentBorderToken: "theme.accent",
    itemAccentWidthPt: 3,
  },
  richBlock: {
    labelBackgroundToken: "theme.accentLight",
    labelFontSizePt: 9,
    labelFontColor: "#3e1018",
    borderColorToken: "theme.accentBorder",
  },
};
