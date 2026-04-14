import {
  safeParse,
  expectString,
  expectBoolean,
  stripUndefined,
  CodecStrictError,
  assertNoUnknownFields,
  expectBooleanStrict,
} from "./codec-utils";

export type RepeatableItemStyle = {
  accentBorderColor?: string;
  accentBorderWidth?: string;
};

export type RepeatableItemCapabilities = {
  locked: boolean;
  removable: boolean;
  editableZones: string[];
};

export const RepeatableItemCodec = {
  parseStyle(json: string): RepeatableItemStyle {
    const raw = safeParse(json, {});
    return {
      accentBorderColor: expectString(raw.accentBorderColor),
      accentBorderWidth: expectString(raw.accentBorderWidth),
    };
  },
  parseCaps(json: string): RepeatableItemCapabilities {
    const raw = safeParse(json, {});
    return {
      locked: expectBoolean(raw.locked, false),
      removable: expectBoolean(raw.removable, true),
      editableZones: Array.isArray(raw.editableZones) ? raw.editableZones.filter((z: unknown) => typeof z === "string") : ["content"],
    };
  },
  defaultStyle(): RepeatableItemStyle { return {}; },
  defaultCaps(): RepeatableItemCapabilities {
    return { locked: false, removable: true, editableZones: ["content"] };
  },
  serializeStyle(style: RepeatableItemStyle): string { return JSON.stringify(stripUndefined(style as Record<string, unknown>)); },
  serializeCaps(caps: RepeatableItemCapabilities): string { return JSON.stringify(caps); },
};

// ---------------------------------------------------------------------------
// Strict parse functions
// ---------------------------------------------------------------------------

const REPEATABLE_ITEM_STYLE_KEYS = ["accentBorderColor", "accentBorderWidth"] as const;
const REPEATABLE_ITEM_CAPS_KEYS = ["locked", "removable", "editableZones"] as const;

export function parseRepeatableItemStyleStrict(raw: Record<string, unknown>): RepeatableItemStyle {
  assertNoUnknownFields(raw, [...REPEATABLE_ITEM_STYLE_KEYS], "style");
  return {
    accentBorderColor: typeof raw.accentBorderColor === "string" ? raw.accentBorderColor : undefined,
    accentBorderWidth: typeof raw.accentBorderWidth === "string" ? raw.accentBorderWidth : undefined,
  };
}

export function parseRepeatableItemCapsStrict(raw: Record<string, unknown>): RepeatableItemCapabilities {
  assertNoUnknownFields(raw, [...REPEATABLE_ITEM_CAPS_KEYS], "caps");
  const editableZones = raw.editableZones;
  if (!Array.isArray(editableZones) || !editableZones.every((z) => typeof z === "string")) {
    throw new CodecStrictError("caps.editableZones", "expected string[]");
  }
  return {
    locked: expectBooleanStrict(raw, "locked"),
    removable: expectBooleanStrict(raw, "removable"),
    editableZones: editableZones as string[],
  };
}

// ---------------------------------------------------------------------------
// Field schemas
// ---------------------------------------------------------------------------

export const repeatableItemStyleFieldSchema = [
  { key: "accentBorderColor", label: "Cor da borda de destaque", type: "color", default: "" },
  { key: "accentBorderWidth", label: "Largura da borda de destaque", type: "string", default: "" },
] as const;

export const repeatableItemCapsFieldSchema = [
  { key: "locked", label: "Bloqueado", type: "toggle", default: false },
  { key: "removable", label: "Removível", type: "toggle", default: true },
  { key: "editableZones", label: "Zonas editáveis", type: "string[]", default: ["content"] },
] as const;
