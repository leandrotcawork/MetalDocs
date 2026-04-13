import { describe, it, expect } from "vitest";
import { instantiateTemplate } from "../instantiate";
import type { TemplateDefinition } from "../types";

const template: TemplateDefinition = {
  templateKey: "po-standard",
  version: 1,
  profileCode: "po",
  status: "published",
  meta: { name: "PO", description: "PO", createdAt: "", updatedAt: "" },
  theme: { accent: "#6b1f2a", accentLight: "#f9f3f3", accentDark: "#3e1018", accentBorder: "#dfc8c8" },
  blocks: [
    { type: "section", props: { title: "IDENTIFICAÇÃO" }, capabilities: { locked: true } },
  ],
};

describe("instantiateTemplate", () => {
  it("creates an envelope with template_ref", () => {
    const envelope = instantiateTemplate(template);
    expect(envelope.template_ref.templateKey).toBe("po-standard");
    expect(envelope.template_ref.templateVersion).toBe(1);
    expect(envelope.template_ref.instantiatedAt).toBeTruthy();
  });

  it("deep clones blocks (mutation-safe)", () => {
    const envelope = instantiateTemplate(template);
    envelope.blocks[0].props.title = "MODIFIED";
    expect(template.blocks[0].props.title).toBe("IDENTIFICAÇÃO");
  });

  it("preserves capabilities on cloned blocks", () => {
    const envelope = instantiateTemplate(template);
    expect(envelope.blocks[0].capabilities).toEqual({ locked: true });
  });
});