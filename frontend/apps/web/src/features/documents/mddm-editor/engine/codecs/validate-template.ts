import type { PublishErrorDTO } from "../../../../../api/templates";
import { CodecStrictError, safeParse } from "./codec-utils";
import { SectionCodec, parseSectionStyleStrict, parseSectionCapsStrict } from "./section-codec";
import { DataTableCodec, parseDataTableStyleStrict, parseDataTableCapsStrict } from "./data-table-codec";
import { RepeatableCodec, parseRepeatableStyleStrict, parseRepeatableCapsStrict } from "./repeatable-codec";
import { RepeatableItemCodec, parseRepeatableItemStyleStrict, parseRepeatableItemCapsStrict } from "./repeatable-item-codec";
import { RichBlockCodec, parseRichBlockStyleStrict, parseRichBlockCapsStrict } from "./rich-block-codec";

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

type RawBlock = {
  id?: string;
  type?: string;
  props?: Record<string, unknown>;
  children?: unknown[];
};

type StrictCodec = {
  defaultStyle: () => Record<string, unknown>;
  defaultCaps: () => Record<string, unknown>;
  parseStyleStrict: (raw: Record<string, unknown>) => unknown;
  parseCapsStrict: (raw: Record<string, unknown>) => unknown;
};

const PASS_THROUGH_BLOCK_TYPES = new Set([
  "paragraph",
  "heading",
  "bulletListItem",
  "numberedListItem",
  "image",
  "quote",
  "divider",
]);

const STRICT_CODECS: Record<string, StrictCodec> = {
  section: {
    defaultStyle: () => SectionCodec.defaultStyle() as Record<string, unknown>,
    defaultCaps: () => SectionCodec.defaultCaps() as Record<string, unknown>,
    parseStyleStrict: parseSectionStyleStrict,
    parseCapsStrict: parseSectionCapsStrict,
  },
  dataTable: {
    defaultStyle: () => DataTableCodec.defaultStyle() as Record<string, unknown>,
    defaultCaps: () => DataTableCodec.defaultCaps() as Record<string, unknown>,
    parseStyleStrict: parseDataTableStyleStrict,
    parseCapsStrict: parseDataTableCapsStrict,
  },
  repeatable: {
    defaultStyle: () => RepeatableCodec.defaultStyle() as Record<string, unknown>,
    defaultCaps: () => RepeatableCodec.defaultCaps() as Record<string, unknown>,
    parseStyleStrict: parseRepeatableStyleStrict,
    parseCapsStrict: parseRepeatableCapsStrict,
  },
  repeatableItem: {
    defaultStyle: () => RepeatableItemCodec.defaultStyle() as Record<string, unknown>,
    defaultCaps: () => RepeatableItemCodec.defaultCaps() as Record<string, unknown>,
    parseStyleStrict: parseRepeatableItemStyleStrict,
    parseCapsStrict: parseRepeatableItemCapsStrict,
  },
  richBlock: {
    defaultStyle: () => RichBlockCodec.defaultStyle() as Record<string, unknown>,
    defaultCaps: () => RichBlockCodec.defaultCaps() as Record<string, unknown>,
    parseStyleStrict: parseRichBlockStyleStrict,
    parseCapsStrict: parseRichBlockCapsStrict,
  },
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
      errors.push(...validateTemplate(b.children.filter(isBlockNode)));
    }
  }

  return errors;
}

// ---------------------------------------------------------------------------
// Block dispatcher
// ---------------------------------------------------------------------------

function validateBlock(block: RawBlock): void {
  const props = block.props ?? {};
  const codec = block.type ? STRICT_CODECS[block.type] : undefined;

  if (block.type && PASS_THROUGH_BLOCK_TYPES.has(block.type)) {
    return;
  }

  if (!codec) {
    throw new CodecStrictError("type", `unknown block type: ${block.type}`);
  }

  const rawStyle = readStoredRecord(props, "styleJson", "style");
  const rawCaps = readStoredRecord(props, "capabilitiesJson", "caps");

  codec.parseStyleStrict({ ...codec.defaultStyle(), ...rawStyle });
  codec.parseCapsStrict({ ...codec.defaultCaps(), ...rawCaps });
}

function readStoredRecord(
  props: Record<string, unknown>,
  jsonKey: "styleJson" | "capabilitiesJson",
  legacyKey: "style" | "caps",
): Record<string, unknown> {
  const jsonValue = props[jsonKey];
  if (typeof jsonValue === "string") {
    return safeParse(jsonValue, {});
  }

  const legacyValue = props[legacyKey];
  if (isRecord(legacyValue)) {
    return legacyValue;
  }

  return {};
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}

function isBlockNode(value: unknown): value is RawBlock {
  return isRecord(value) && typeof value.type === "string" && !("text" in value);
}
