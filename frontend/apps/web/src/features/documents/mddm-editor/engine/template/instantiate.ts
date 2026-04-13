import { validateTemplate } from "./validate";
import type { TemplateDefinition, TemplateRef, TemplateBlock } from "./types";

export type MDDMTemplateEnvelope = {
  mddm_version: number;
  template_ref: TemplateRef;
  blocks: TemplateBlock[];
};

const CURRENT_MDDM_VERSION = 1;

/**
 * Maps a canonical TemplateBlock (with top-level style/capabilities) to a
 * BlockNote-compatible block where those fields are serialized into
 * props.styleJson / props.capabilitiesJson.
 *
 * This is the editor boundary: templates are authored in canonical format;
 * BlockNote only accepts primitive props.
 */
function toBlockNoteBlock(block: TemplateBlock): TemplateBlock {
  const { style, capabilities, children, ...rest } = block;
  return {
    ...rest,
    props: {
      ...block.props,
      styleJson: JSON.stringify(style ?? {}),
      capabilitiesJson: JSON.stringify(capabilities ?? {}),
    },
    children: children?.map(toBlockNoteBlock),
  };
}

/**
 * Instantiate a template into a document envelope.
 *
 * Validates the template first (throws on errors), then maps each block's
 * canonical style/capabilities to BlockNote prop strings.
 */
export function instantiateTemplate(template: TemplateDefinition): MDDMTemplateEnvelope {
  const errors = validateTemplate(template);
  if (errors.length > 0) {
    throw new Error(
      `Cannot instantiate invalid template "${template.templateKey}": ` +
      errors.map((e) => `${e.path} — ${e.message}`).join("; "),
    );
  }

  return {
    mddm_version: CURRENT_MDDM_VERSION,
    template_ref: {
      templateKey: template.templateKey,
      templateVersion: template.version,
      instantiatedAt: new Date().toISOString(),
    },
    blocks: template.blocks.map(toBlockNoteBlock),
  };
}
