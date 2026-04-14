import { defaultComponentRules } from "./components";
import type { LayoutTokens } from "./tokens";

export function tokensToCssVars(tokens: LayoutTokens): Record<string, string> {
  const rules = defaultComponentRules;
  return {
    // Page dimensions
    "--mddm-page-width": `${tokens.page.widthMm}mm`,
    "--mddm-page-height": `${tokens.page.heightMm}mm`,
    "--mddm-page-content-width": `${tokens.page.contentWidthMm}mm`,
    "--mddm-margin-top": `${tokens.page.marginTopMm}mm`,
    "--mddm-margin-right": `${tokens.page.marginRightMm}mm`,
    "--mddm-margin-bottom": `${tokens.page.marginBottomMm}mm`,
    "--mddm-margin-left": `${tokens.page.marginLeftMm}mm`,
    // Typography
    "--mddm-font-family": `"${tokens.typography.editorFont}", -apple-system, sans-serif`,
    "--mddm-font-size-base": `${tokens.typography.baseSizePt}pt`,
    "--mddm-font-size-h1": `${tokens.typography.headingSizePt.h1}pt`,
    "--mddm-font-size-h2": `${tokens.typography.headingSizePt.h2}pt`,
    "--mddm-font-size-h3": `${tokens.typography.headingSizePt.h3}pt`,
    "--mddm-line-height": `${tokens.typography.lineHeightPt}pt`,
    "--mddm-label-font-size": `${tokens.typography.labelSizePt}pt`,
    // Spacing — screen values (rem/px); print mm values live in tokens.spacing for DOCX/PDF emitters
    "--mddm-section-gap": tokens.screenSpacing.sectionGap,
    "--mddm-field-gap": tokens.screenSpacing.fieldGap,
    "--mddm-block-gap": tokens.screenSpacing.blockGap,
    "--mddm-cell-padding": tokens.screenSpacing.cellPadding,
    // Theme colors
    "--mddm-accent": tokens.theme.accent,
    "--mddm-accent-light": tokens.theme.accentLight,
    "--mddm-accent-dark": tokens.theme.accentDark,
    "--mddm-accent-border": tokens.theme.accentBorder,
    // Component rules (derived from ComponentRules)
    "--mddm-section-header-height": `${rules.section.headerHeightMm}mm`,
    "--mddm-section-header-font-size": `${rules.section.headerFontSizePt}pt`,
    "--mddm-section-header-color": rules.section.headerFontColor,
    "--mddm-field-label-width": `${rules.field.labelWidthPercent}%`,
    "--mddm-field-value-width": `${rules.field.valueWidthPercent}%`,
    "--mddm-field-border-width": `${rules.field.borderWidthPt}pt`,
    "--mddm-field-min-height": `${rules.field.minHeightMm}mm`,
    "--mddm-field-label-font-size": `${rules.field.labelFontSizePt}pt`,
    // BlockNote bridge vars
    "--bn-font-family": `"${tokens.typography.editorFont}", -apple-system, sans-serif`,
    "--bn-border-radius": "4px",
    "--bn-colors-side-menu": "transparent",
  };
}
