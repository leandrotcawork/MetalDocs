export type SchemaRuntimeStatus = "idle" | "loading" | "saving" | "error";

export type SchemaScalarFieldType =
  | "text"
  | "textarea"
  | "number"
  | "date"
  | "select"
  | "checkbox"
  | "checklist"
  | "array";

export interface SchemaFieldBase {
  key: string;
  label: string;
  required?: boolean;
  description?: string;
  itemType?: string;
  options?: string[];
  columns?: SchemaField[];
  itemFields?: SchemaField[];
}

export interface SchemaScalarField extends SchemaFieldBase {
  type: SchemaScalarFieldType;
  options?: string[];
}

export interface SchemaTableField extends SchemaFieldBase {
  type: "table";
  columns: SchemaField[];
}

export interface SchemaRichField extends SchemaFieldBase {
  type: "rich";
}

export interface SchemaRepeatField extends SchemaFieldBase {
  type: "repeat";
  itemFields: SchemaField[];
}

export type SchemaField = SchemaScalarField | SchemaTableField | SchemaRichField | SchemaRepeatField;

export interface SchemaSection {
  key: string;
  num?: string;
  title?: string;
  color?: string;
  description?: string;
  fields: SchemaField[];
}

export interface DocumentTypeSchema {
  sections: SchemaSection[];
}

export interface ChecklistItem {
  label: string;
  checked: boolean;
}

export interface SchemaDocumentTypeBundleResponse {
  typeKey: string;
  schema: DocumentTypeSchema;
  name?: string;
  description?: string;
  activeVersion?: number | null;
}

export interface SchemaDocumentSnapshot {
  documentId: string;
  title: string;
  documentCode: string;
  documentProfile: string;
  documentType: string;
  status?: string;
}

export interface SchemaDocumentEditorBundleResponse {
  document: SchemaDocumentSnapshot;
  schema: DocumentTypeSchema;
  values: Record<string, unknown>;
  version: number | null;
  pdfUrl: string;
  typeKey: string;
}

export interface SchemaDocumentContentSaveResponse {
  documentId: string;
  version: number | null;
  pdfUrl: string;
  values: Record<string, unknown>;
}

export interface SchemaDocumentEditorState {
  documentId: string;
  typeKey: string;
  schema: DocumentTypeSchema | null;
  values: Record<string, unknown>;
  version: number | null;
  pdfUrl: string;
  status: SchemaRuntimeStatus;
  error: string;
  bundle: SchemaDocumentTypeBundleResponse | null;
  document: SchemaDocumentSnapshot | null;
}

export const emptyDocumentTypeSchema: DocumentTypeSchema = {
  sections: [],
};

export const emptySchemaDocumentEditorState: SchemaDocumentEditorState = {
  documentId: "",
  typeKey: "",
  schema: null,
  values: {},
  version: null,
  pdfUrl: "",
  status: "idle",
  error: "",
  bundle: null,
  document: null,
};

export type RuntimeFieldKind = "scalar" | "table" | "repeat" | "rich";

export type RuntimeScalarInput = "text" | "textarea" | "select" | "number" | "checkbox";

type RuntimeFieldBase = {
  key: string;
  label?: string;
  description?: string;
  required?: boolean;
};

export type RuntimeScalarField = RuntimeFieldBase & {
  kind: "scalar";
  input: RuntimeScalarInput;
  options: string[];
};

export type RuntimeTableField = RuntimeFieldBase & {
  kind: "table";
  columns: RuntimeScalarField[];
};

export type RuntimeRepeatField = RuntimeFieldBase & {
  kind: "repeat";
  itemLabel?: string;
  itemFields: RuntimeField[];
};

export type RuntimeRichField = RuntimeFieldBase & {
  kind: "rich";
};

export type RuntimeField = RuntimeScalarField | RuntimeTableField | RuntimeRepeatField | RuntimeRichField;

export type RuntimeSection = {
  key: string;
  title?: string;
  description?: string;
  fields: RuntimeField[];
};

export type RuntimeDocumentSchema = {
  sections: RuntimeSection[];
};

type RecordLike = Record<string, unknown>;

export function toRuntimeDocumentSchema(rawSchema: unknown): RuntimeDocumentSchema {
  const rawSections = getRecordArray(rawSchema, "sections");
  return {
    sections: rawSections.map(toRuntimeSection).filter((section): section is RuntimeSection => section !== null),
  };
}

function toRuntimeSection(rawSection: unknown): RuntimeSection | null {
  if (!isRecordLike(rawSection)) return null;
  const key = String(rawSection.key ?? "").trim();
  if (!key) return null;

  const rawFields = getRecordArray(rawSection, "fields");
  return {
    key,
    title: asOptionalString(rawSection.title),
    description: asOptionalString(rawSection.description),
    fields: rawFields.map(toRuntimeField).filter((field): field is RuntimeField => field !== null),
  };
}

function toRuntimeField(rawField: unknown): RuntimeField | null {
  if (!isRecordLike(rawField)) return null;
  const kind = resolveFieldKind(rawField);
  const key = String(rawField.key ?? "").trim();
  if (!key) return null;

  if (kind === "table") {
    return {
      kind,
      key,
      label: asOptionalString(rawField.label),
      description: asOptionalString(rawField.description),
      required: Boolean(rawField.required),
      columns: getRecordArray(rawField, "columns")
        .map((column) => toRuntimeScalarField(column))
        .filter((column): column is RuntimeScalarField => column !== null),
    };
  }

  if (kind === "repeat") {
    const rawItemFields = getRecordArray(rawField, "itemFields");
    const itemFields =
      rawItemFields.length > 0
        ? rawItemFields.map(toRuntimeField).filter((field): field is RuntimeField => field !== null)
        : buildLegacyRepeatFields(rawField);

    return {
      kind,
      key,
      label: asOptionalString(rawField.label),
      description: asOptionalString(rawField.description),
      required: Boolean(rawField.required),
      itemLabel: asOptionalString(rawField.itemLabel) ?? asOptionalString(rawField.label),
      itemFields,
    };
  }

  if (kind === "rich") {
    return {
      kind,
      key,
      label: asOptionalString(rawField.label),
      description: asOptionalString(rawField.description),
      required: Boolean(rawField.required),
    };
  }

  return toRuntimeScalarField(rawField, key);
}

function toRuntimeScalarField(rawField: unknown, forcedKey?: string): RuntimeScalarField | null {
  if (!isRecordLike(rawField)) return null;
  const key = forcedKey ?? String(rawField.key ?? "").trim();
  if (!key) return null;

  return {
    kind: "scalar",
    key,
    label: asOptionalString(rawField.label),
    description: asOptionalString(rawField.description),
    required: Boolean(rawField.required),
    input: resolveScalarInput(rawField),
    options: getStringArray(rawField, "options"),
  };
}

function resolveFieldKind(rawField: RecordLike): RuntimeFieldKind {
  const explicitKind = asOptionalString(rawField.kind);
  if (explicitKind === "scalar" || explicitKind === "table" || explicitKind === "repeat" || explicitKind === "rich") {
    return explicitKind;
  }

  const legacyType = asOptionalString(rawField.type);
  if (legacyType === "table") return "table";
  if (legacyType === "array" || legacyType === "checklist") return "repeat";
  if (legacyType === "rich" || legacyType === "richtext" || legacyType === "rich_text") return "rich";
  return "scalar";
}

function resolveScalarInput(rawField: RecordLike): RuntimeScalarInput {
  const explicitInput = asOptionalString(rawField.input);
  if (explicitInput === "text" || explicitInput === "textarea" || explicitInput === "select" || explicitInput === "number" || explicitInput === "checkbox") {
    return explicitInput;
  }

  const legacyType = asOptionalString(rawField.type);
  if (legacyType === "textarea" || legacyType === "select" || legacyType === "number") {
    return legacyType;
  }

  return "text";
}

function buildLegacyRepeatFields(rawField: RecordLike): RuntimeField[] {
  const legacyType = asOptionalString(rawField.type);
  if (legacyType === "checklist") {
    return [
      {
        kind: "scalar",
        key: "label",
        label: "Item",
        input: "text",
        options: [],
      },
      {
        kind: "scalar",
        key: "checked",
        label: "Concluido",
        input: "checkbox",
        options: [],
      },
    ];
  }

  return [
    {
      kind: "scalar",
      key: "value",
      label: asOptionalString(rawField.itemLabel) ?? asOptionalString(rawField.label) ?? "Item",
      input: resolveScalarInput(rawField),
      options: getStringArray(rawField, "options"),
    },
  ];
}

function getRecordArray(raw: unknown, key: string): unknown[] {
  if (!isRecordLike(raw)) return [];
  const value = raw[key];
  return Array.isArray(value) ? value : [];
}

function getStringArray(raw: unknown, key: string): string[] {
  return getRecordArray(raw, key).filter((item): item is string => typeof item === "string");
}

function asOptionalString(value: unknown): string | undefined {
  return typeof value === "string" && value.trim() ? value : undefined;
}

function isRecordLike(value: unknown): value is RecordLike {
  return Boolean(value) && typeof value === "object" && !Array.isArray(value);
}
