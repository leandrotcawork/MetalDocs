import { describe, it, expect } from "vitest";
import { validateTemplate, type ValidationError } from "../validate";
import type { TemplateDefinition } from "../types";

function makeTemplate(overrides: Partial<TemplateDefinition> = {}): TemplateDefinition {
  return {
    templateKey: "test",
    version: 1,
    profileCode: "po",
    status: "published",
    meta: { name: "Test", description: "Test", createdAt: "", updatedAt: "" },
    theme: { accent: "#6b1f2a", accentLight: "#f9f3f3", accentDark: "#3e1018", accentBorder: "#dfc8c8" },
    blocks: [],
    ...overrides,
  };
}

describe("validateTemplate", () => {
  it("accepts a valid empty template", () => {
    expect(validateTemplate(makeTemplate())).toHaveLength(0);
  });

  it("accepts a template with valid section block", () => {
    const errors = validateTemplate(makeTemplate({
      blocks: [{ type: "section", props: { title: "TEST" } }],
    }));
    expect(errors).toHaveLength(0);
  });

  it("rejects unknown block type", () => {
    const errors = validateTemplate(makeTemplate({
      blocks: [{ type: "nonexistent", props: {} }],
    }));
    expect(errors).toContainEqual(expect.objectContaining({ error: "unknown_block_type" }));
  });

  it("rejects section without title", () => {
    const errors = validateTemplate(makeTemplate({
      blocks: [{ type: "section", props: {} }],
    }));
    expect(errors).toContainEqual(expect.objectContaining({ error: "missing_required_prop" }));
  });
});