// MDDM component layout rules. Reference absolute dimensions so React and
// docx.js produce equivalent output. See spec section "Layout IR".

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
