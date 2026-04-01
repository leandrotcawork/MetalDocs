import { Document, Packer, Paragraph, TextRun } from "docx";
import { renderSection } from "./runtime/renderSection.js";
import type {
  ColumnDef,
  DocumentPayload,
  DocumentTypeSchema,
  DocumentValues,
  FieldDef,
  ScalarFieldType,
  SectionDef,
} from "./runtime/types.js";

function isObject(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}

const scalarTypes: ScalarFieldType[] = ["text", "textarea", "number", "date", "select", "checkbox"];

function asString(value: unknown): string | null {
  return typeof value === "string" ? value.trim() : null;
}

function invalid(code: string): never {
  throw new Error(code);
}

function assertNonEmptyString(value: unknown, code: string): string {
  const out = asString(value);
  if (!out) {
    invalid(code);
  }
  return out;
}

function validateColumnDef(column: ColumnDef): void {
  assertNonEmptyString(column.key, "DOCGEN_INVALID_SCHEMA");
  assertNonEmptyString(column.label, "DOCGEN_INVALID_SCHEMA");
  if (!scalarTypes.includes(column.type)) {
    invalid("DOCGEN_INVALID_SCHEMA");
  }
}

function validateFieldDef(field: FieldDef): void {
  assertNonEmptyString(field.key, "DOCGEN_INVALID_SCHEMA");
  assertNonEmptyString(field.label, "DOCGEN_INVALID_SCHEMA");
  if (!field.type) {
    invalid("DOCGEN_INVALID_SCHEMA");
  }

  if (field.type === "table") {
    if (!Array.isArray(field.columns) || field.columns.length === 0) {
      invalid("DOCGEN_INVALID_SCHEMA");
    }
    field.columns.forEach((column) => {
      if (!isObject(column)) {
        invalid("DOCGEN_INVALID_SCHEMA");
      }
      validateColumnDef(column as ColumnDef);
    });
    return;
  }
  if (field.type === "repeat") {
    if (!Array.isArray(field.itemFields) || field.itemFields.length === 0) {
      invalid("DOCGEN_INVALID_SCHEMA");
    }
    field.itemFields.forEach((itemField) => {
      if (!isObject(itemField)) {
        invalid("DOCGEN_INVALID_SCHEMA");
      }
      validateFieldDef(itemField as FieldDef);
    });
    return;
  }

  if (field.type === "rich") {
    return;
  }

  if (!scalarTypes.includes(field.type)) {
    invalid("DOCGEN_INVALID_SCHEMA");
  }
}

function validateSectionDef(section: SectionDef, values: DocumentValues): void {
  assertNonEmptyString(section.key, "DOCGEN_INVALID_SCHEMA");
  assertNonEmptyString(section.num, "DOCGEN_INVALID_SCHEMA");
  assertNonEmptyString(section.title, "DOCGEN_INVALID_SCHEMA");

  if (!Array.isArray(section.fields)) {
    invalid("DOCGEN_INVALID_SCHEMA");
  }
  if (section.fields.length === 0) {
    invalid("DOCGEN_INVALID_SCHEMA");
  }
  section.fields.forEach((field) => {
    if (!isObject(field)) {
      invalid("DOCGEN_INVALID_SCHEMA");
    }
    validateFieldDef(field as FieldDef);
  });

  const value = values[section.key];
  if (value !== undefined && !isObject(value)) {
    invalid("DOCGEN_INVALID_VALUES");
  }

  if (value !== undefined) {
    section.fields.forEach((field) => validateFieldValue(field as FieldDef, value as Record<string, unknown>));
  }
}

function validateFieldValue(field: FieldDef, container: Record<string, unknown>): void {
  const rawValue = container[field.key];
  if (rawValue === undefined || rawValue === null) {
    return;
  }

  switch (field.type) {
    case "text":
    case "textarea":
    case "select":
      if (typeof rawValue !== "string") {
        invalid("DOCGEN_INVALID_VALUES");
      }
      return;
    case "number":
      if (typeof rawValue !== "number") {
        invalid("DOCGEN_INVALID_VALUES");
      }
      return;
    case "checkbox":
      if (typeof rawValue !== "boolean") {
        invalid("DOCGEN_INVALID_VALUES");
      }
      return;
    case "date":
      if (typeof rawValue !== "string") {
        invalid("DOCGEN_INVALID_VALUES");
      }
      return;
    case "table":
      if (!Array.isArray(rawValue)) {
        invalid("DOCGEN_INVALID_VALUES");
      }
      rawValue.forEach((row) => {
        if (!isObject(row)) {
          invalid("DOCGEN_INVALID_VALUES");
        }
        field.columns.forEach((column) => validateFieldValue(column, row as Record<string, unknown>));
      });
      return;
    case "repeat":
      if (!Array.isArray(rawValue)) {
        invalid("DOCGEN_INVALID_VALUES");
      }
      rawValue.forEach((item) => {
        if (!isObject(item)) {
          invalid("DOCGEN_INVALID_VALUES");
        }
        field.itemFields.forEach((nested) => validateFieldValue(nested, item as Record<string, unknown>));
      });
      return;
    case "rich":
      if (!Array.isArray(rawValue)) {
        invalid("DOCGEN_INVALID_VALUES");
      }
      rawValue.forEach((block) => {
        if (!isObject(block) || typeof block.type !== "string") {
          invalid("DOCGEN_INVALID_VALUES");
        }
        switch (block.type) {
          case "text":
            if (!Array.isArray(block.runs)) {
              invalid("DOCGEN_INVALID_VALUES");
            }
            block.runs.forEach((run) => {
              if (!isObject(run) || typeof run.text !== "string") {
                invalid("DOCGEN_INVALID_VALUES");
              }
            });
            return;
          case "image":
            if (typeof block.data !== "string") {
              invalid("DOCGEN_INVALID_VALUES");
            }
            if (block.mimeType !== undefined && typeof block.mimeType !== "string") {
              invalid("DOCGEN_INVALID_VALUES");
            }
            return;
          case "table":
            if (!Array.isArray(block.rows)) {
              invalid("DOCGEN_INVALID_VALUES");
            }
            block.rows.forEach((row) => {
              if (!Array.isArray(row)) {
                invalid("DOCGEN_INVALID_VALUES");
              }
            });
            return;
          case "list":
            if (!Array.isArray(block.items)) {
              invalid("DOCGEN_INVALID_VALUES");
            }
            block.items.forEach((item) => {
              if (typeof item !== "string") {
                invalid("DOCGEN_INVALID_VALUES");
              }
            });
            return;
          default:
            invalid("DOCGEN_INVALID_VALUES");
        }
      });
      return;
  }
}

function normalizeDocumentPayload(input: unknown): DocumentPayload {
  if (!isObject(input)) {
    invalid("DOCGEN_INVALID_PAYLOAD");
  }

  const { documentType, documentCode, title, schema, values } = input;

  if (
    typeof documentType !== "string" ||
    typeof documentCode !== "string" ||
    typeof title !== "string" ||
    !isObject(schema) ||
    !Array.isArray(schema.sections) ||
    !isObject(values)
  ) {
    invalid("DOCGEN_INVALID_PAYLOAD");
  }

  const payload = {
    documentType,
    documentCode,
    title,
    schema: { sections: schema.sections as DocumentTypeSchema["sections"] },
    values: values as DocumentValues,
  } satisfies DocumentPayload;

  if (payload.schema.sections.length === 0) {
    invalid("DOCGEN_INVALID_SCHEMA");
  }

  payload.schema.sections.forEach((section) => validateSectionDef(section, payload.values));

  return payload;
}

export async function generateDocx(payload: unknown): Promise<Uint8Array> {
  const runtime = normalizeDocumentPayload(payload);

  const children = [
    new Paragraph({
      children: [new TextRun({ text: runtime.title, bold: true, size: 30 })],
    }),
    new Paragraph({
      children: [
        new TextRun({
          text: `${runtime.documentType} | ${runtime.documentCode}`,
          italics: true,
          size: 20,
        }),
      ],
    }),
    ...runtime.schema.sections.flatMap((section) =>
      renderSection(section, runtime.values[section.key] ?? {})
    ),
  ];

  const doc = new Document({
    sections: [
      {
        properties: {},
        children,
      },
    ],
  });

  return Packer.toBuffer(doc);
}
