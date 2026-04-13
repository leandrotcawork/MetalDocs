import { safeParse, expectString, expectBoolean, stripUndefined } from "./codec-utils";

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
