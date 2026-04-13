import { describe, it, expect } from "vitest";
import { validateTemplate } from "../validate";
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

describe("validateTemplate — basic", () => {
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

describe("validateTemplate — repeatable capability invariants", () => {
  it("accepts repeatable with maxItems >= minItems", () => {
    const errors = validateTemplate(makeTemplate({
      blocks: [{ type: "repeatable", props: { label: "Steps" }, capabilities: { maxItems: 10, minItems: 1 } }],
    }));
    expect(errors).toHaveLength(0);
  });

  it("rejects repeatable with maxItems < minItems", () => {
    const errors = validateTemplate(makeTemplate({
      blocks: [{ type: "repeatable", props: { label: "Steps" }, capabilities: { maxItems: 2, minItems: 5 } }],
    }));
    expect(errors).toContainEqual(expect.objectContaining({ error: "invalid_capability", path: expect.stringContaining("maxItems") }));
  });

  it("accepts repeatable with maxItems === minItems === 0 (edge case)", () => {
    const errors = validateTemplate(makeTemplate({
      blocks: [{ type: "repeatable", props: { label: "Steps" }, capabilities: { maxItems: 0, minItems: 0 } }],
    }));
    expect(errors).toHaveLength(0);
  });
});

describe("validateTemplate — dataTable mode", () => {
  it("accepts dataTable with mode: fixed", () => {
    const errors = validateTemplate(makeTemplate({
      blocks: [{ type: "dataTable", props: { label: "T" }, capabilities: { mode: "fixed" } }],
    }));
    expect(errors).toHaveLength(0);
  });

  it("accepts dataTable with mode: dynamic", () => {
    const errors = validateTemplate(makeTemplate({
      blocks: [{ type: "dataTable", props: { label: "T" }, capabilities: { mode: "dynamic" } }],
    }));
    expect(errors).toHaveLength(0);
  });

  it("rejects dataTable with invalid mode", () => {
    const errors = validateTemplate(makeTemplate({
      blocks: [{ type: "dataTable", props: { label: "T" }, capabilities: { mode: "weird" } }],
    }));
    expect(errors).toContainEqual(expect.objectContaining({ error: "invalid_capability" }));
  });
});

describe("validateTemplate — per-type capability key restriction", () => {
  it("rejects addItems capability on a section block", () => {
    const errors = validateTemplate(makeTemplate({
      blocks: [{ type: "section", props: { title: "S" }, capabilities: { addItems: true } }],
    }));
    expect(errors).toContainEqual(expect.objectContaining({ error: "invalid_capability_key" }));
  });

  it("rejects addRows capability on a repeatable block", () => {
    const errors = validateTemplate(makeTemplate({
      blocks: [{ type: "repeatable", props: { label: "R" }, capabilities: { addRows: true } }],
    }));
    expect(errors).toContainEqual(expect.objectContaining({ error: "invalid_capability_key" }));
  });
});

describe("validateTemplate — nested children", () => {
  it("validates children recursively", () => {
    const errors = validateTemplate(makeTemplate({
      blocks: [{
        type: "section",
        props: { title: "S" },
        children: [{ type: "nonexistent", props: {} }],
      }],
    }));
    expect(errors).toContainEqual(expect.objectContaining({ error: "unknown_block_type" }));
  });
});
