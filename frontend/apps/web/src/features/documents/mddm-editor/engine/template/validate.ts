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

/** Capability keys that are valid on each block type */
const BLOCK_TYPE_ALLOWED_CAPS: Record<string, Set<string>> = {
  section: new Set(["locked", "removable", "reorderable"]),
  richBlock: new Set(["locked", "removable", "reorderable", "editableZones"]),
  repeatableItem: new Set(["locked", "removable", "reorderable", "editableZones"]),
  repeatable: new Set(["locked", "removable", "reorderable", "addItems", "removeItems", "maxItems", "minItems"]),
  dataTable: new Set([
    "locked", "removable", "reorderable", "mode",
    "addRows", "removeRows", "addColumns", "removeColumns", "resizeColumns",
    "headerLocked", "editableZones", "maxRows",
  ]),
  paragraph: new Set(["locked", "removable", "reorderable"]),
  heading: new Set(["locked", "removable", "reorderable"]),
  bulletListItem: new Set(["locked", "removable", "reorderable"]),
  numberedListItem: new Set(["locked", "removable", "reorderable"]),
  image: new Set(["locked", "removable", "reorderable"]),
  quote: new Set(["locked", "removable", "reorderable"]),
  divider: new Set(["locked", "removable", "reorderable"]),
};

const VALID_DATATABLE_MODES = new Set(["fixed", "dynamic"]);

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

    if (block.capabilities) {
      validateCapabilities(block.type, block.capabilities, `${path}.capabilities`, errors);
    }

    if (block.children) {
      validateBlocks(block.children, `${path}.children`, errors);
    }
  }
}

function validateCapabilities(
  blockType: string,
  capabilities: Record<string, unknown>,
  path: string,
  errors: ValidationError[],
): void {
  const allowed = BLOCK_TYPE_ALLOWED_CAPS[blockType];
  if (!allowed) return;

  for (const key of Object.keys(capabilities)) {
    if (!allowed.has(key)) {
      errors.push({
        path: `${path}.${key}`,
        error: "invalid_capability_key",
        message: `Capability key "${key}" is not valid for block type "${blockType}"`,
      });
    }
  }

  if (blockType === "dataTable" && capabilities.mode !== undefined) {
    if (!VALID_DATATABLE_MODES.has(capabilities.mode as string)) {
      errors.push({
        path: `${path}.mode`,
        error: "invalid_capability",
        message: `dataTable.mode must be "fixed" or "dynamic", got "${capabilities.mode}"`,
      });
    }
  }

  if (blockType === "repeatable") {
    const maxItems = capabilities.maxItems;
    const minItems = capabilities.minItems;
    if (typeof maxItems === "number" && typeof minItems === "number" && maxItems < minItems) {
      errors.push({
        path: `${path}.maxItems`,
        error: "invalid_capability",
        message: `repeatable.maxItems (${maxItems}) must be >= minItems (${minItems})`,
      });
    }
  }
}
