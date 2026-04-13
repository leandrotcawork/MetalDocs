import type { TemplateDefinition, TemplateRef, TemplateBlock } from "./types";

export type MDDMTemplateEnvelope = {
  mddm_version: number;
  template_ref: TemplateRef;
  blocks: TemplateBlock[];
};

const CURRENT_MDDM_VERSION = 1;

export function instantiateTemplate(template: TemplateDefinition): MDDMTemplateEnvelope {
  return {
    mddm_version: CURRENT_MDDM_VERSION,
    template_ref: {
      templateKey: template.templateKey,
      templateVersion: template.version,
      instantiatedAt: new Date().toISOString(),
    },
    blocks: structuredClone(template.blocks),
  };
}