import { safeParse, expectString, expectBoolean, stripUndefined } from "./codec-utils";

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
