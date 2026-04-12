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

  it("maps spacing correctly", () => {
    expect(vars["--mddm-section-gap"]).toBe("6mm");
    expect(vars["--mddm-field-gap"]).toBe("3mm");
    expect(vars["--mddm-block-gap"]).toBe("2mm");
    expect(vars["--mddm-cell-padding"]).toBe("2mm");
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
});
