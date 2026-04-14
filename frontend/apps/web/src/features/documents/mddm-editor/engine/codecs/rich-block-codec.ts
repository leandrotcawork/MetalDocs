import {
  safeParse,
  expectString,
  expectBoolean,
  stripUndefined,
  CodecStrictError,
  assertNoUnknownFields,
  expectBooleanStrict,
} from "./codec-utils";

export type RichBlockStyle = {
  labelBackground?: string;
  labelFontSize?: string;
  labelColor?: string;
  borderColor?: string;
};

export type RichBlockCapabilities = {
  locked: boolean;
  removable: boolean;
  editableZones: string[];
};

export const RichBlockCodec = {
  parseStyle(json: string): RichBlockStyle {
    const raw = safeParse(json, {});
    return {
      labelBackground: expectString(raw.labelBackground),
      labelFontSize: expectString(raw.labelFontSize),
      labelColor: expectString(raw.labelColor),
      borderColor: expectString(raw.borderColor),
    };
  },
  parseCaps(json: string): RichBlockCapabilities {
    const raw = safeParse(json, {});
    return {
      locked: expectBoolean(raw.locked, true),
      removable: expectBoolean(raw.removable, false),
      editableZones: Array.isArray(raw.editableZones) ? raw.editableZones.filter((z: unknown) => typeof z === "string") : ["content"],
    };
  },
  defaultStyle(): RichBlockStyle { return {}; },
  defaultCaps(): RichBlockCapabilities {
    return { locked: true, removable: false, editableZones: ["content"] };
  },
  serializeStyle(style: RichBlockStyle): string { return JSON.stringify(stripUndefined(style as Record<string, unknown>)); },
  serializeCaps(caps: RichBlockCapabilities): string { return JSON.stringify(caps); },
};

// ---------------------------------------------------------------------------
// Strict parse functions
// ---------------------------------------------------------------------------

const RICH_BLOCK_STYLE_KEYS = ["labelBackground", "labelFontSize", "labelColor", "borderColor"] as const;
const RICH_BLOCK_CAPS_KEYS = ["locked", "removable", "editableZones"] as const;

export function parseRichBlockStyleStrict(raw: Record<string, unknown>): RichBlockStyle {
  assertNoUnknownFields(raw, [...RICH_BLOCK_STYLE_KEYS], "style");
  return {
    labelBackground: typeof raw.labelBackground === "string" ? raw.labelBackground : undefined,
    labelFontSize: typeof raw.labelFontSize === "string" ? raw.labelFontSize : undefined,
    labelColor: typeof raw.labelColor === "string" ? raw.labelColor : undefined,
    borderColor: typeof raw.borderColor === "string" ? raw.borderColor : undefined,
  };
}

export function parseRichBlockCapsStrict(raw: Record<string, unknown>): RichBlockCapabilities {
  assertNoUnknownFields(raw, [...RICH_BLOCK_CAPS_KEYS], "caps");
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

export const richBlockStyleFieldSchema = [
  { key: "labelBackground", label: "Fundo do rótulo", type: "color", default: "" },
  { key: "labelFontSize", label: "Tamanho da fonte do rótulo", type: "string", default: "" },
  { key: "labelColor", label: "Cor do rótulo", type: "color", default: "" },
  { key: "borderColor", label: "Cor da borda", type: "color", default: "" },
] as const;

export const richBlockCapsFieldSchema = [
  { key: "locked", label: "Bloqueado", type: "toggle", default: true },
  { key: "removable", label: "Removível", type: "toggle", default: false },
  { key: "editableZones", label: "Zonas editáveis", type: "string[]", default: ["content"] },
] as const;
