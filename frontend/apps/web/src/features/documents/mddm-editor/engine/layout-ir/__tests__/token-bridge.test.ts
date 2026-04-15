import { describe, it, expect } from "vitest";
import { tokensToCssVars } from "../token-bridge";
import { defaultLayoutTokens } from "../tokens";

describe("tokensToCssVars", () => {
  const vars = tokensToCssVars(defaultLayoutTokens);

  it("maps theme colors correctly", () => {
    expect(vars["--mddm-accent"]).toBe(defaultLayoutTokens.theme.accent);
    expect(vars["--mddm-accent-light"]).toBe(defaultLayoutTokens.theme.accentLight);
    expect(vars["--mddm-accent-dark"]).toBe(defaultLayoutTokens.theme.accentDark);
    expect(vars["--mddm-accent-border"]).toBe(defaultLayoutTokens.theme.accentBorder);
  });

  it("maps typography correctly", () => {
    expect(vars["--mddm-font-family"]).toContain(defaultLayoutTokens.typography.editorFont);
    expect(vars["--mddm-font-size-base"]).toBe("11pt");
  });

  it("maps spacing to screen values (not print mm)", () => {
    // token-bridge uses screenSpacing for CSS vars; DOCX/PDF emitters use spacing.*Mm directly
    expect(vars["--mddm-section-gap"]).toBe(defaultLayoutTokens.screenSpacing.sectionGap);
    expect(vars["--mddm-field-gap"]).toBe(defaultLayoutTokens.screenSpacing.fieldGap);
    expect(vars["--mddm-block-gap"]).toBe(defaultLayoutTokens.screenSpacing.blockGap);
    expect(vars["--mddm-cell-padding"]).toBe(defaultLayoutTokens.screenSpacing.cellPadding);
  });

  it("maps component rules correctly", () => {
    expect(vars["--mddm-field-label-width"]).toBe("35%");
    expect(vars["--mddm-field-value-width"]).toBe("65%");
    expect(vars["--mddm-field-min-height"]).toBe("7mm");
    expect(vars["--mddm-section-header-height"]).toBe("8mm");
    expect(vars["--mddm-section-header-font-size"]).toBe("13pt");
  });

  it("maps BlockNote bridge vars correctly", () => {
    expect(vars["--bn-font-family"]).toContain(defaultLayoutTokens.typography.editorFont);
    expect(vars["--bn-border-radius"]).toBe("4px");
  });

  it("all keys start with --mddm- or --bn-", () => {
    for (const key of Object.keys(vars)) {
      expect(key.startsWith("--mddm-") || key.startsWith("--bn-")).toBe(true);
    }
  });

  it("reflects changed tokens dynamically", () => {
    const modified = {
      ...defaultLayoutTokens,
      theme: { ...defaultLayoutTokens.theme, accent: "#ff0000" },
    };
    const modifiedVars = tokensToCssVars(modified);
    expect(modifiedVars["--mddm-accent"]).toBe("#ff0000");
  });

  it("maps overridden page margins and content width vars", () => {
    const marginLeftMm = 30;
    const marginRightMm = 28;
    const modified = {
      ...defaultLayoutTokens,
      page: {
        ...defaultLayoutTokens.page,
        marginTopMm: 15,
        marginRightMm,
        marginBottomMm: 35,
        marginLeftMm,
        contentWidthMm:
          defaultLayoutTokens.page.widthMm - marginLeftMm - marginRightMm,
      },
    };

    const modifiedVars = tokensToCssVars(modified);
    expect(modifiedVars["--mddm-margin-top"]).toBe("15mm");
    expect(modifiedVars["--mddm-margin-right"]).toBe("28mm");
    expect(modifiedVars["--mddm-margin-bottom"]).toBe("35mm");
    expect(modifiedVars["--mddm-margin-left"]).toBe("30mm");
    expect(modifiedVars["--mddm-page-content-width"]).toBe("152mm");
  });
});
