import { describe, it, expect } from "vitest";
import { validateTemplate } from "../validate";
import { poStandardTemplate } from "../../../templates/po-standard";

describe("PO Standard Template", () => {
  it("passes validation", () => {
    const errors = validateTemplate(poStandardTemplate);
    expect(errors).toHaveLength(0);
  });

  it("has 10 top-level sections", () => {
    const sections = poStandardTemplate.blocks.filter(b => b.type === "section");
    expect(sections.length).toBe(10);
  });

  it("has the correct theme", () => {
    expect(poStandardTemplate.theme.accent).toBe("#6b1f2a");
  });
});