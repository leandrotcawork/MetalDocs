import {
  safeParse,
  expectString,
  expectBoolean,
  stripUndefined,
  CodecStrictError,
  assertNoUnknownFields,
  expectBooleanStrict,
} from "./codec-utils";

export type SectionStyle = {
  headerHeight?: string;
  headerBackground?: string;
  headerColor?: string;
  headerFontSize?: string;
  headerFontWeight?: string;
};

export type SectionCapabilities = {
  locked: boolean;
  removable: boolean;
  reorderable: boolean;
};

export const SectionCodec = {
  parseStyle(json: string): SectionStyle {
    const raw = safeParse(json, {});
    return {
      headerHeight: expectString(raw.headerHeight),
      headerBackground: expectString(raw.headerBackground),
      headerColor: expectString(raw.headerColor),
      headerFontSize: expectString(raw.headerFontSize),
      headerFontWeight: expectString(raw.headerFontWeight),
    };
  },

  parseCaps(json: string): SectionCapabilities {
    const raw = safeParse(json, {});
    return {
      locked: expectBoolean(raw.locked, true),
      removable: expectBoolean(raw.removable, false),
      reorderable: expectBoolean(raw.reorderable, false),
    };
  },

  defaultStyle(): SectionStyle {
    return {};
  },

  defaultCaps(): SectionCapabilities {
    return { locked: true, removable: false, reorderable: false };
  },

  serializeStyle(style: SectionStyle): string {
    return JSON.stringify(stripUndefined(style as Record<string, unknown>));
  },

  serializeCaps(caps: SectionCapabilities): string {
    return JSON.stringify(caps);
  },
};

// ---------------------------------------------------------------------------
// Strict parse functions — throw CodecStrictError on unknown or invalid fields
// ---------------------------------------------------------------------------

const SECTION_STYLE_KEYS = [
  "headerHeight",
  "headerBackground",
  "headerColor",
  "headerFontSize",
  "headerFontWeight",
] as const;

const SECTION_CAPS_KEYS = ["locked", "removable", "reorderable"] as const;

export function parseSectionStyleStrict(raw: Record<string, unknown>): SectionStyle {
  assertNoUnknownFields(raw, [...SECTION_STYLE_KEYS], "style");
  return {
    headerHeight: typeof raw.headerHeight === "string" ? raw.headerHeight : undefined,
    headerBackground: typeof raw.headerBackground === "string" ? raw.headerBackground : undefined,
    headerColor: typeof raw.headerColor === "string" ? raw.headerColor : undefined,
    headerFontSize: typeof raw.headerFontSize === "string" ? raw.headerFontSize : undefined,
    headerFontWeight: typeof raw.headerFontWeight === "string" ? raw.headerFontWeight : undefined,
  };
}

export function parseSectionCapsStrict(raw: Record<string, unknown>): SectionCapabilities {
  assertNoUnknownFields(raw, [...SECTION_CAPS_KEYS], "caps");
  return {
    locked: expectBooleanStrict(raw, "locked"),
    removable: expectBooleanStrict(raw, "removable"),
    reorderable: expectBooleanStrict(raw, "reorderable"),
  };
}

// ---------------------------------------------------------------------------
// Field schemas — consumed by the Property Sidebar (Phase 11)
// ---------------------------------------------------------------------------

export const sectionStyleFieldSchema = [
  { key: "headerHeight", label: "Altura do cabeçalho", type: "string", default: "" },
  { key: "headerBackground", label: "Fundo do cabeçalho", type: "color", default: "" },
  { key: "headerColor", label: "Cor do texto do cabeçalho", type: "color", default: "" },
  { key: "headerFontSize", label: "Tamanho da fonte", type: "string", default: "" },
  { key: "headerFontWeight", label: "Peso da fonte", type: "string", default: "" },
] as const;

export const sectionCapsFieldSchema = [
  { key: "locked", label: "Bloqueado", type: "toggle", default: true },
  { key: "removable", label: "Removível", type: "toggle", default: false },
  { key: "reorderable", label: "Reordenável", type: "toggle", default: false },
] as const;

