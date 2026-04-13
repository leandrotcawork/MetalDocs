import { safeParse, expectString, expectBoolean, stripUndefined } from "./codec-utils";

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
