import type { TemplateDefinition, TemplateBlock } from "./types";

export type ValidationError = {
  path: string;
  error: string;
  message: string;
};

const KNOWN_BLOCK_TYPES = new Set([
  "section", "dataTable", "repeatable", "repeatableItem",
  "richBlock", "paragraph", "heading", "bulletListItem",
  "numberedListItem", "image", "quote", "divider",
]);

const REQUIRED_PROPS: Record<string, string[]> = {
  section: ["title"],
  dataTable: ["label"],
  repeatable: ["label"],
  richBlock: ["label"],
};

export function validateTemplate(template: TemplateDefinition): ValidationError[] {
  const errors: ValidationError[] = [];
  validateBlocks(template.blocks, "blocks", errors);
  return errors;
}

function validateBlocks(blocks: TemplateBlock[], basePath: string, errors: ValidationError[]): void {
  for (let i = 0; i < blocks.length; i++) {
    const block = blocks[i];
    const path = `${basePath}[${i}]`;

    if (!KNOWN_BLOCK_TYPES.has(block.type)) {
      errors.push({ path, error: "unknown_block_type", message: `Unknown block type: ${block.type}` });
      continue;
    }

    const required = REQUIRED_PROPS[block.type];
    if (required) {
      for (const prop of required) {
        if (!block.props[prop]) {
          errors.push({ path: `${path}.props.${prop}`, error: "missing_required_prop", message: `Missing required prop: ${prop}` });
        }
      }
    }

    if (block.children) {
      validateBlocks(block.children, `${path}.children`, errors);
    }
  }
}