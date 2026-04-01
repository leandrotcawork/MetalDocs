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
