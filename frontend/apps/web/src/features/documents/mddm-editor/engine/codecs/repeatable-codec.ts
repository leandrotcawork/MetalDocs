import { safeParse, expectString, expectBoolean, expectNumber, stripUndefined } from "./codec-utils";

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
