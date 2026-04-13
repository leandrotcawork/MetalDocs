import { safeParse, expectString, expectBoolean, expectNumber, stripUndefined } from "./codec-utils";

export type DataTableStyle = {
  headerBackground?: string;
  headerColor?: string;
  headerFontWeight?: string;
  cellBorderColor?: string;
  cellPadding?: string;
  density?: "normal" | "compact";
};

export type DataTableCapabilities = {
  locked: boolean;
  removable: boolean;
  mode: "fixed" | "dynamic";
  editableZones: string[];
  addRows: boolean;
  removeRows: boolean;
  addColumns: boolean;
  removeColumns: boolean;
  resizeColumns: boolean;
  headerLocked: boolean;
  maxRows: number;
};

const VALID_DENSITIES = ["normal", "compact"] as const;
const VALID_MODES = ["fixed", "dynamic"] as const;

export const DataTableCodec = {
  parseStyle(json: string): DataTableStyle {
    const raw = safeParse(json, {});
    const density = expectString(raw.density);
    return {
      headerBackground: expectString(raw.headerBackground),
      headerColor: expectString(raw.headerColor),
      headerFontWeight: expectString(raw.headerFontWeight),
      cellBorderColor: expectString(raw.cellBorderColor),
      cellPadding: expectString(raw.cellPadding),
      density: density && VALID_DENSITIES.includes(density as any) ? (density as "normal" | "compact") : undefined,
    };
  },

  parseCaps(json: string): DataTableCapabilities {
    const raw = safeParse(json, {});
    const mode = expectString(raw.mode);
    return {
      locked: expectBoolean(raw.locked, false),
      removable: expectBoolean(raw.removable, false),
      mode: mode && VALID_MODES.includes(mode as any) ? (mode as "fixed" | "dynamic") : "dynamic",
      editableZones: Array.isArray(raw.editableZones) ? raw.editableZones.filter((z: unknown) => typeof z === "string") : ["cells"],
      addRows: expectBoolean(raw.addRows, true),
      removeRows: expectBoolean(raw.removeRows, true),
      addColumns: expectBoolean(raw.addColumns, false),
      removeColumns: expectBoolean(raw.removeColumns, false),
      resizeColumns: expectBoolean(raw.resizeColumns, false),
      headerLocked: expectBoolean(raw.headerLocked, true),
      maxRows: expectNumber(raw.maxRows, 100),
    };
  },

  defaultStyle(): DataTableStyle {
    return {};
  },
  defaultCaps(): DataTableCapabilities {
    return {
      locked: false,
      removable: false,
      mode: "dynamic",
      editableZones: ["cells"],
      addRows: true,
      removeRows: true,
      addColumns: false,
      removeColumns: false,
      resizeColumns: false,
      headerLocked: true,
      maxRows: 100,
    };
  },

  serializeStyle(style: DataTableStyle): string {
    return JSON.stringify(stripUndefined(style as Record<string, unknown>));
  },
  serializeCaps(caps: DataTableCapabilities): string {
    return JSON.stringify(caps);
  },
};
