import { defaultLayoutTokens } from "../layout-tokens";

const { exportFont, exportFontFallbacks } = defaultLayoutTokens.typography;
const fontFamily = [`"${exportFont}"`, ...exportFontFallbacks.map((f) => (f.includes(" ") ? `"${f}"` : f))].join(", ");

export const PRINT_STYLESHEET = `
@page {
  size: A4;
  margin: ${defaultLayoutTokens.page.marginTopMm}mm ${defaultLayoutTokens.page.marginRightMm}mm ${defaultLayoutTokens.page.marginBottomMm}mm ${defaultLayoutTokens.page.marginLeftMm}mm;
}

html, body {
  margin: 0;
  padding: 0;
  font-family: ${fontFamily};
  font-size: ${defaultLayoutTokens.typography.baseSizePt}pt;
  line-height: ${defaultLayoutTokens.typography.lineHeightPt}pt;

  font-kerning: normal;
  font-feature-settings: "liga" 1, "kern" 1;
  font-synthesis: none;
}

@media print { * { -webkit-print-color-adjust: exact !important; color-adjust: exact !important; } }

.mddm-section-header,
.mddm-field,
.mddm-field-group {
  page-break-inside: avoid;
}

/* Hide editor-only chrome in case any leaks through. */
.bn-side-menu,
.bn-formatting-toolbar,
.bn-slash-menu,
.bn-drag-handle {
  display: none !important;
}

/* MDDM block base styling used alongside inline styles from toExternalHTML. */
[data-mddm-block] {
  box-sizing: border-box;
}
`;
