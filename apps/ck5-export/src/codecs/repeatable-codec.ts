import {
  safeParse,
  expectString,
  expectBoolean,
  expectNumber,
  stripUndefined,
  assertNoUnknownFields,
  expectNumberStrict,
  expectBooleanStrict,
  expectOptionalStringStrict,
} from "./codec-utils";

export type RepeatableStyle = {
  borderColor?: string;
  itemAccentBorder?: string;
  itemAccentWidth?: string;
};

export type RepeatableCapabilities = {
  locked: boolean;
  removable: boolean;
  addItems: boolean;
  removeItems: boolean;
  maxItems: number;
  minItems: number;
};

export const RepeatableCodec = {
  parseStyle(json: string): RepeatableStyle {
    const raw = safeParse(json, {});
    return {
      borderColor: expectString(raw.borderColor),
      itemAccentBorder: expectString(raw.itemAccentBorder),
      itemAccentWidth: expectString(raw.itemAccentWidth),
    };
  },
  parseCaps(json: string): RepeatableCapabilities {
    const raw = safeParse(json, {});
    return {
      locked: expectBoolean(raw.locked, true),
      removable: expectBoolean(raw.removable, false),
      addItems: expectBoolean(raw.addItems, true),
      removeItems: expectBoolean(raw.removeItems, true),
      maxItems: expectNumber(raw.maxItems, 100),
      minItems: expectNumber(raw.minItems, 0),
    };
  },
  defaultStyle(): RepeatableStyle { return {}; },
  defaultCaps(): RepeatableCapabilities {
    return { locked: true, removable: false, addItems: true, removeItems: true, maxItems: 100, minItems: 0 };
  },
  serializeStyle(style: RepeatableStyle): string { return JSON.stringify(stripUndefined(style as Record<string, unknown>)); },
  serializeCaps(caps: RepeatableCapabilities): string { return JSON.stringify(caps); },
};

// ---------------------------------------------------------------------------
// Strict parse functions
// ---------------------------------------------------------------------------

const REPEATABLE_STYLE_KEYS = ["borderColor", "itemAccentBorder", "itemAccentWidth"] as const;
const REPEATABLE_CAPS_KEYS = ["locked", "removable", "addItems", "removeItems", "maxItems", "minItems"] as const;

export function parseRepeatableStyleStrict(raw: Record<string, unknown>): RepeatableStyle {
  assertNoUnknownFields(raw, [...REPEATABLE_STYLE_KEYS], "style");
  return {
    borderColor: expectOptionalStringStrict(raw, "borderColor"),
    itemAccentBorder: expectOptionalStringStrict(raw, "itemAccentBorder"),
    itemAccentWidth: expectOptionalStringStrict(raw, "itemAccentWidth"),
  };
}

export function parseRepeatableCapsStrict(raw: Record<string, unknown>): RepeatableCapabilities {
  assertNoUnknownFields(raw, [...REPEATABLE_CAPS_KEYS], "caps");
  return {
    locked: expectBooleanStrict(raw, "locked"),
    removable: expectBooleanStrict(raw, "removable"),
    addItems: expectBooleanStrict(raw, "addItems"),
    removeItems: expectBooleanStrict(raw, "removeItems"),
    maxItems: expectNumberStrict(raw, "maxItems"),
    minItems: expectNumberStrict(raw, "minItems"),
  };
}

// ---------------------------------------------------------------------------
// Field schemas
// ---------------------------------------------------------------------------

export const repeatableStyleFieldSchema = [
  { key: "borderColor", label: "Cor da borda", type: "color", default: "" },
  { key: "itemAccentBorder", label: "Borda de destaque do item", type: "color", default: "" },
  { key: "itemAccentWidth", label: "Largura da borda de destaque", type: "string", default: "" },
] as const;

export const repeatableCapsFieldSchema = [
  { key: "locked", label: "Bloqueado", type: "toggle", default: true },
  { key: "removable", label: "Removível", type: "toggle", default: false },
  { key: "addItems", label: "Permitir adicionar itens", type: "toggle", default: true },
  { key: "removeItems", label: "Permitir remover itens", type: "toggle", default: true },
  { key: "maxItems", label: "Máximo de itens", type: "number", default: 100 },
  { key: "minItems", label: "Mínimo de itens", type: "number", default: 0 },
] as const;
