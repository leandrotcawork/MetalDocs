import { describe, expect, it } from "vitest";
import { defaultComponentRules, type ComponentRules } from "../components";

describe("Layout IR component rules", () => {
  it("defines Section with fixed 8mm header height and full width", () => {
    expect(defaultComponentRules.section.headerHeightMm).toBe(8);
    expect(defaultComponentRules.section.fullWidth).toBe(true);
    expect(defaultComponentRules.section.headerFontSizePt).toBeGreaterThan(0);
  });

  it("defines Field with 35/65 label/value split", () => {
    expect(defaultComponentRules.field.labelWidthPercent).toBe(35);
    expect(defaultComponentRules.field.valueWidthPercent).toBe(65);
    expect(defaultComponentRules.field.labelWidthPercent + defaultComponentRules.field.valueWidthPercent).toBe(100);
  });

  it("defines FieldGroup with valid column counts", () => {
    expect([1, 2]).toContain(defaultComponentRules.fieldGroup.defaultColumns);
    expect(defaultComponentRules.fieldGroup.fullWidth).toBe(true);
  });

  it("exports the ComponentRules type", () => {
    const rules: ComponentRules = defaultComponentRules;
    expect(rules).toBe(defaultComponentRules);
  });
});
