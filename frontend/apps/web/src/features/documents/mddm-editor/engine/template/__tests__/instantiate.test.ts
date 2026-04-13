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
    {
      type: "section",
      props: { title: "IDENTIFICAÇÃO" },
      style: { headerBackground: "#6b1f2a" },
      capabilities: { locked: true, removable: false },
      children: [],
    },
    {
      type: "repeatable",
      props: { label: "Etapas", itemPrefix: "Etapa" },
      style: {},
      capabilities: { locked: false, addItems: true, maxItems: 50, minItems: 1 },
      children: [],
    },
  ],
};

describe("instantiateTemplate", () => {
  it("creates an envelope with template_ref", () => {
    const envelope = instantiateTemplate(template);
    expect(envelope.template_ref.templateKey).toBe("po-standard");
    expect(envelope.template_ref.templateVersion).toBe(1);
    expect(envelope.template_ref.instantiatedAt).toBeTruthy();
  });

  it("maps top-level style to props.styleJson", () => {
    const envelope = instantiateTemplate(template);
    const parsed = JSON.parse(envelope.blocks[0].props.styleJson as string);
    expect(parsed.headerBackground).toBe("#6b1f2a");
  });

  it("maps top-level capabilities to props.capabilitiesJson", () => {
    const envelope = instantiateTemplate(template);
    const parsed = JSON.parse(envelope.blocks[0].props.capabilitiesJson as string);
    expect(parsed.locked).toBe(true);
    expect(parsed.removable).toBe(false);
  });

  it("maps empty style to '{}'", () => {
    const envelope = instantiateTemplate(template);
    expect(envelope.blocks[1].props.styleJson).toBe("{}");
  });

  it("omits top-level style/capabilities fields from instantiated blocks", () => {
    const envelope = instantiateTemplate(template);
    expect((envelope.blocks[0] as any).style).toBeUndefined();
    expect((envelope.blocks[0] as any).capabilities).toBeUndefined();
  });

  it("deep clones blocks (mutation-safe)", () => {
    const envelope = instantiateTemplate(template);
    (envelope.blocks[0].props as any).title = "MODIFIED";
    expect(template.blocks[0].props.title).toBe("IDENTIFICAÇÃO");
  });

  it("recursively maps children", () => {
    const templateWithChild: TemplateDefinition = {
      ...template,
      blocks: [{
        type: "section",
        props: { title: "S1" },
        capabilities: { locked: true },
        children: [{
          type: "richBlock",
          props: { label: "Obj" },
          capabilities: { locked: true, editableZones: ["content"] },
        }],
      }],
    };
    const envelope = instantiateTemplate(templateWithChild);
    const childCaps = JSON.parse(envelope.blocks[0].children![0].props.capabilitiesJson as string);
    expect(childCaps.locked).toBe(true);
    expect(childCaps.editableZones).toEqual(["content"]);
  });
});
