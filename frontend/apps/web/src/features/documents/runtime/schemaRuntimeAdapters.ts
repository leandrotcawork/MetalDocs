import type {
  ChecklistItem,
  DocumentTypeSchema,
  SchemaDocumentEditorBundleResponse,
  SchemaDocumentEditorState,
  SchemaDocumentTypeBundleResponse,
  SchemaField,
  SchemaRepeatField,
  SchemaScalarField,
  SchemaSection,
  SchemaTableField,
  SchemaRuntimeStatus,
} from "./schemaRuntimeTypes";

function asRecord(value: unknown): Record<string, unknown> {
  if (!value || typeof value !== "object" || Array.isArray(value)) {
    return {};
  }
  return value as Record<string, unknown>;
}

function normalizeString(value: unknown, fallback = ""): string {
  return typeof value === "string" && value.trim() !== "" ? value : fallback;
}

function normalizeBoolean(value: unknown): boolean {
  return typeof value === "boolean" ? value : Boolean(value);
}

function normalizeFieldType(value: unknown): SchemaField["type"] {
  const allowed: SchemaField["type"][] = ["text", "textarea", "number", "date", "select", "checkbox", "checklist", "table", "rich", "repeat"];
  return typeof value === "string" && allowed.includes(value as SchemaField["type"]) ? (value as SchemaField["type"]) : "text";
}

function normalizeScalarField(value: Record<string, unknown>): SchemaScalarField {
  return {
    key: normalizeString(value.key),
    label: normalizeString(value.label, normalizeString(value.key)),
    required: normalizeBoolean(value.required),
    description: normalizeString(value.description, ""),
    itemType: normalizeString(value.itemType, ""),
    type: normalizeFieldType(value.type) as SchemaScalarField["type"],
    options: Array.isArray(value.options) ? value.options.filter((item): item is string => typeof item === "string") : [],
  };
}

function normalizeTableField(value: Record<string, unknown>): SchemaTableField {
  return {
    key: normalizeString(value.key),
    label: normalizeString(value.label, normalizeString(value.key)),
    required: normalizeBoolean(value.required),
    description: normalizeString(value.description, ""),
    itemType: normalizeString(value.itemType, ""),
    type: "table",
    columns: Array.isArray(value.columns) ? value.columns.map((column) => normalizeSchemaField(column)) : [],
  };
}

function normalizeRepeatField(value: Record<string, unknown>): SchemaRepeatField {
  return {
    key: normalizeString(value.key),
    label: normalizeString(value.label, normalizeString(value.key)),
    required: normalizeBoolean(value.required),
    description: normalizeString(value.description, ""),
    itemType: normalizeString(value.itemType, ""),
    type: "repeat",
    itemFields: Array.isArray(value.itemFields) ? value.itemFields.map((field) => normalizeSchemaField(field)) : [],
  };
}

function normalizeRichField(value: Record<string, unknown>): SchemaField {
  return {
    key: normalizeString(value.key),
    label: normalizeString(value.label, normalizeString(value.key)),
    required: normalizeBoolean(value.required),
    description: normalizeString(value.description, ""),
    itemType: normalizeString(value.itemType, ""),
    type: "rich",
  };
}

export function normalizeSchemaField(value: unknown): SchemaField {
  const record = asRecord(value);
  const fieldType = normalizeFieldType(record.type);

  if (fieldType === "table") {
    return normalizeTableField(record);
  }

  if (fieldType === "repeat") {
    return normalizeRepeatField(record);
  }

  if (fieldType === "rich") {
    return normalizeRichField(record);
  }

  return normalizeScalarField(record);
}

export function normalizeChecklistItems(value: unknown): ChecklistItem[] {
  if (!Array.isArray(value)) {
    return [];
  }

  return value.map((item) => {
    if (typeof item === "string") {
      return { label: item, checked: false };
    }

    const record = asRecord(item);
    return {
      label: normalizeString(record.label),
      checked: normalizeBoolean(record.checked),
    };
  });
}

export function normalizeSchemaSection(value: unknown): SchemaSection {
  const record = asRecord(value);
  return {
    key: normalizeString(record.key),
    num: normalizeString(record.num, ""),
    title: normalizeString(record.title, normalizeString(record.key)),
    color: normalizeString(record.color, ""),
    description: normalizeString(record.description, ""),
    fields: Array.isArray(record.fields) ? record.fields.map((field) => normalizeSchemaField(field)) : [],
  };
}

export function normalizeDocumentTypeSchema(value: unknown): DocumentTypeSchema {
  const record = asRecord(value);
  return {
    sections: Array.isArray(record.sections) ? record.sections.map((section) => normalizeSchemaSection(section)) : [],
  };
}

export function normalizeDocumentTypeBundle(value: unknown): SchemaDocumentTypeBundleResponse {
  const record = asRecord(value);
  return {
    typeKey: normalizeString(record.typeKey),
    name: normalizeString(record.name, ""),
    description: normalizeString(record.description, ""),
    activeVersion: typeof record.activeVersion === "number" ? record.activeVersion : null,
    schema: normalizeDocumentTypeSchema(record.schema),
  };
}

export function normalizeSchemaDocumentEditorBundle(value: unknown): SchemaDocumentEditorBundleResponse {
  const record = asRecord(value);
  const documentRecord = asRecord(record.document);
  return {
    document: {
      documentId: normalizeString(documentRecord.documentId),
      title: normalizeString(documentRecord.title),
      documentCode: normalizeString(documentRecord.documentCode),
      documentProfile: normalizeString(documentRecord.documentProfile),
      documentType: normalizeString(documentRecord.documentType),
      status: normalizeString(documentRecord.status, ""),
    },
    schema: normalizeDocumentTypeSchema(record.schema),
    values: asRecord(record.values),
    version: typeof record.version === "number" ? record.version : null,
    pdfUrl: normalizeString(record.pdfUrl, ""),
    typeKey: normalizeString(record.typeKey),
  };
}

export function createSchemaDocumentEditorState(
  input?: Partial<SchemaDocumentEditorState> & { status?: SchemaRuntimeStatus },
): SchemaDocumentEditorState {
  return {
    documentId: input?.documentId ?? "",
    typeKey: input?.typeKey ?? "",
    schema: input?.schema ?? null,
    values: input?.values ?? {},
    version: input?.version ?? null,
    pdfUrl: input?.pdfUrl ?? "",
    status: input?.status ?? "idle",
    error: input?.error ?? "",
    bundle: input?.bundle ?? null,
    document: input?.document ?? null,
  };
}

export function getActiveSchemaBundle(bundle: SchemaDocumentTypeBundleResponse | null) {
  if (!bundle) {
    return null;
  }
  return bundle.schema;
}
