import type { PublishErrorDTO } from "../../../../../api/templates";
import { CodecStrictError } from "./codec-utils";
import { parseSectionStyleStrict, parseSectionCapsStrict } from "./section-codec";
import { parseDataTableStyleStrict, parseDataTableCapsStrict } from "./data-table-codec";
import { parseRepeatableStyleStrict, parseRepeatableCapsStrict } from "./repeatable-codec";
import { parseRepeatableItemStyleStrict, parseRepeatableItemCapsStrict } from "./repeatable-item-codec";
import { parseRichBlockStyleStrict, parseRichBlockCapsStrict } from "./rich-block-codec";

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

type RawBlock = {
  id?: string;
  type?: string;
  props?: Record<string, unknown>;
  children?: unknown[];
};

// ---------------------------------------------------------------------------
// Public API
// ---------------------------------------------------------------------------

/**
 * Aggregate strict validation across all blocks in a template document.
 * Returns the same error shape the server returns on a 422 publish validation response.
 */
export function validateTemplate(blocks: unknown[]): PublishErrorDTO[] {
  const errors: PublishErrorDTO[] = [];

  for (const block of blocks) {
    const b = block as RawBlock;
    const blockId = b.id ?? "";
    const blockType = b.type ?? "unknown";

    try {
      validateBlock(b);
    } catch (err) {
      if (err instanceof CodecStrictError) {
        errors.push({ blockId, blockType, field: err.field, reason: err.reason });
      } else {
        errors.push({ blockId, blockType, field: "", reason: String(err) });
      }
    }

    // Recurse into children
    if (Array.isArray(b.children)) {
      errors.push(...validateTemplate(b.children));
    }
  }

  return errors;
}

// ---------------------------------------------------------------------------
// Block dispatcher
// ---------------------------------------------------------------------------

function validateBlock(block: RawBlock): void {
  const props = block.props ?? {};
  const style = (props.style ?? {}) as Record<string, unknown>;
  const caps = (props.caps ?? {}) as Record<string, unknown>;

  switch (block.type) {
    case "section":
      parseSectionStyleStrict(style);
      parseSectionCapsStrict(caps);
      break;

    case "dataTable":
      parseDataTableStyleStrict(style);
      parseDataTableCapsStrict(caps);
      break;

    case "repeatable":
      parseRepeatableStyleStrict(style);
      parseRepeatableCapsStrict(caps);
      break;

    case "repeatableItem":
      parseRepeatableItemStyleStrict(style);
      parseRepeatableItemCapsStrict(caps);
      break;

    case "richBlock":
      parseRichBlockStyleStrict(style);
      parseRichBlockCapsStrict(caps);
      break;

    default:
      throw new CodecStrictError("type", `unknown block type: ${block.type}`);
  }
}
