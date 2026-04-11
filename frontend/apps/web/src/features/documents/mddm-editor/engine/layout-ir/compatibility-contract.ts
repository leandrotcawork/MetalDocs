// Render Compatibility Contract — three tiers governing editor/DOCX/PDF parity.
// See spec section "Render Compatibility Contract".

export type CompatibilityContract = Readonly<{
  tier1: Readonly<{
    blockStructure: "byte-exact";
    blockProps: "byte-exact";
    colors: "byte-exact";
    fontFamily: "byte-exact";
    columnProportions: "byte-exact";
  }>;
  tier2: Readonly<{
    pixelDiffEditorToPdf: number;
    pixelDiffEditorToDocx: number;
    verticalCellDriftPx: number;
    horizontalCharDriftPx: number;
  }>;
  forbidden: Readonly<{
    autoFitColumns: "error";
    unitlessLineHeight: "error";
    emLineHeight: "error";
    negativeMargins: "error";
    flexbox: "error";
    gridFrUnits: "error";
    nestedDataTableMaxDepth: number;
    percentageFontSize: "error";
    transforms: "error";
    filters: "error";
    fixedPositioning: "error";
    viewportUnits: "error";
    externalUrlsDuringPdfExport: "error";
  }>;
  degradation: Readonly<{
    logLevel: "warn";
    telemetry: boolean;
    userNotification: "toast";
  }>;
}>;

export const COMPATIBILITY_CONTRACT: CompatibilityContract = {
  tier1: {
    blockStructure: "byte-exact",
    blockProps: "byte-exact",
    colors: "byte-exact",
    fontFamily: "byte-exact",
    columnProportions: "byte-exact",
  },
  tier2: {
    pixelDiffEditorToPdf: 0.02,
    pixelDiffEditorToDocx: 0.05,
    verticalCellDriftPx: 3,
    horizontalCharDriftPx: 1,
  },
  forbidden: {
    autoFitColumns: "error",
    unitlessLineHeight: "error",
    emLineHeight: "error",
    negativeMargins: "error",
    flexbox: "error",
    gridFrUnits: "error",
    nestedDataTableMaxDepth: 2,
    percentageFontSize: "error",
    transforms: "error",
    filters: "error",
    fixedPositioning: "error",
    viewportUnits: "error",
    externalUrlsDuringPdfExport: "error",
  },
  degradation: {
    logLevel: "warn",
    telemetry: true,
    userNotification: "toast",
  },
};

export type ForbiddenConstruct = keyof CompatibilityContract["forbidden"];

const FORBIDDEN_SET: ReadonlySet<string> = new Set(Object.keys(COMPATIBILITY_CONTRACT.forbidden));

export function isForbiddenConstruct(name: string): boolean {
  return FORBIDDEN_SET.has(name) && name !== "nestedDataTableMaxDepth";
}
