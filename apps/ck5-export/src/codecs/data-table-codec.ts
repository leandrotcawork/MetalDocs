import {
  safeParse,
  expectString,
  expectBoolean,
  expectNumber,
  stripUndefined,
  CodecStrictError,
  assertNoUnknownFields,
  expectStringStrict,
  expectNumberStrict,
  expectBooleanStrict,
  expectOptionalStringStrict,
} from "./codec-utils";

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

// ---------------------------------------------------------------------------
// Strict parse functions
// ---------------------------------------------------------------------------

const DATA_TABLE_STYLE_KEYS = [
  "headerBackground",
  "headerColor",
  "headerFontWeight",
  "cellBorderColor",
  "cellPadding",
  "density",
] as const;

const DATA_TABLE_CAPS_KEYS = [
  "locked",
  "removable",
  "mode",
  "editableZones",
  "addRows",
  "removeRows",
  "addColumns",
  "removeColumns",
  "resizeColumns",
  "headerLocked",
  "maxRows",
] as const;

export function parseDataTableStyleStrict(raw: Record<string, unknown>): DataTableStyle {
  assertNoUnknownFields(raw, [...DATA_TABLE_STYLE_KEYS], "style");
  const density = expectOptionalStringStrict(raw, "density");
  if (density !== undefined && !VALID_DENSITIES.includes(density as "normal" | "compact")) {
    throw new CodecStrictError("style.density", `invalid density value: ${density}`);
  }
  return {
    headerBackground: expectOptionalStringStrict(raw, "headerBackground"),
    headerColor: expectOptionalStringStrict(raw, "headerColor"),
    headerFontWeight: expectOptionalStringStrict(raw, "headerFontWeight"),
    cellBorderColor: expectOptionalStringStrict(raw, "cellBorderColor"),
    cellPadding: expectOptionalStringStrict(raw, "cellPadding"),
    density: density as "normal" | "compact" | undefined,
  };
}

export function parseDataTableCapsStrict(raw: Record<string, unknown>): DataTableCapabilities {
  assertNoUnknownFields(raw, [...DATA_TABLE_CAPS_KEYS], "caps");
  const mode = expectStringStrict(raw, "mode");
  if (!VALID_MODES.includes(mode as "fixed" | "dynamic")) {
    throw new CodecStrictError("caps.mode", `invalid mode value: ${mode}`);
  }
  const editableZones = raw.editableZones;
  if (!Array.isArray(editableZones) || !editableZones.every((z) => typeof z === "string")) {
    throw new CodecStrictError("caps.editableZones", "expected string[]");
  }
  return {
    locked: expectBooleanStrict(raw, "locked"),
    removable: expectBooleanStrict(raw, "removable"),
    mode: mode as "fixed" | "dynamic",
    editableZones: editableZones as string[],
    addRows: expectBooleanStrict(raw, "addRows"),
    removeRows: expectBooleanStrict(raw, "removeRows"),
    addColumns: expectBooleanStrict(raw, "addColumns"),
    removeColumns: expectBooleanStrict(raw, "removeColumns"),
    resizeColumns: expectBooleanStrict(raw, "resizeColumns"),
    headerLocked: expectBooleanStrict(raw, "headerLocked"),
    maxRows: expectNumberStrict(raw, "maxRows"),
  };
}

// ---------------------------------------------------------------------------
// Field schemas
// ---------------------------------------------------------------------------

export const dataTableStyleFieldSchema = [
  { key: "headerBackground", label: "Fundo do cabeçalho", type: "color", default: "" },
  { key: "headerColor", label: "Cor do texto do cabeçalho", type: "color", default: "" },
  { key: "headerFontWeight", label: "Peso da fonte do cabeçalho", type: "string", default: "" },
  { key: "cellBorderColor", label: "Cor da borda das células", type: "color", default: "" },
  { key: "cellPadding", label: "Padding das células", type: "string", default: "" },
  { key: "density", label: "Densidade", type: "select", options: ["normal", "compact"], default: "normal" },
] as const;

export const dataTableCapsFieldSchema = [
  { key: "locked", label: "Bloqueado", type: "toggle", default: false },
  { key: "removable", label: "Removível", type: "toggle", default: false },
  { key: "mode", label: "Modo", type: "select", options: ["fixed", "dynamic"], default: "dynamic" },
  { key: "editableZones", label: "Zonas editáveis", type: "string[]", default: ["cells"] },
  { key: "addRows", label: "Permitir adicionar linhas", type: "toggle", default: true },
  { key: "removeRows", label: "Permitir remover linhas", type: "toggle", default: true },
  { key: "addColumns", label: "Permitir adicionar colunas", type: "toggle", default: false },
  { key: "removeColumns", label: "Permitir remover colunas", type: "toggle", default: false },
  { key: "resizeColumns", label: "Permitir redimensionar colunas", type: "toggle", default: false },
  { key: "headerLocked", label: "Cabeçalho bloqueado", type: "toggle", default: true },
  { key: "maxRows", label: "Máximo de linhas", type: "number", default: 100 },
] as const;
