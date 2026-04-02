import { AlignmentType, BorderStyle, Document, Footer, Packer, Paragraph, Table, TableRow } from "docx";
import { renderSection } from "./runtime/renderSection.js";
import {
  C,
  CONTENT_WIDTH,
  DEFAULT_FONT,
  DEFAULT_FONT_SIZE,
  HEADER_ROW_1,
  HEADER_ROW_2,
  HEADER_TITLE_WIDTH,
  PAGE_HEIGHT,
  PAGE_MARGIN,
  PAGE_WIDTH,
  cellBorder,
  makeCell,
  makePageNumberField,
  makeTable,
  paragraph,
  run,
  tableBorder,
} from "./runtime/docx.js";
import type {
  ColumnDef,
  DocumentMetadata,
  DocumentPayload,
  DocumentRevision,
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
const hexColor = /^#?[0-9a-fA-F]{6}$/;

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

function assertHexColor(value: unknown, code: string): void {
  if (value === undefined || value === null) {
    return;
  }
  if (typeof value !== "string" || !hexColor.test(value)) {
    invalid(code);
  }
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
  assertHexColor(section.color, "DOCGEN_INVALID_SCHEMA");

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
              assertHexColor(run.color, "DOCGEN_INVALID_VALUES");
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

  const { documentType, documentCode, title, version, status, schema, values } = input;

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
    version: typeof version === "string" && version.trim() ? version.trim() : undefined,
    status: typeof status === "string" && status.trim() ? status.trim() : undefined,
    schema: { sections: schema.sections as DocumentTypeSchema["sections"] },
    values: values as DocumentValues,
    metadata: isObject(input.metadata) ? input.metadata as unknown as DocumentPayload["metadata"] : undefined,
    revisions: Array.isArray(input.revisions) ? input.revisions as unknown as DocumentPayload["revisions"] : undefined,
  } satisfies DocumentPayload;

  if (payload.schema.sections.length === 0) {
    invalid("DOCGEN_INVALID_SCHEMA");
  }

  payload.schema.sections.forEach((section) => {
    if (!isObject(section)) {
      invalid("DOCGEN_INVALID_SCHEMA");
    }
    validateSectionDef(section as SectionDef, payload.values);
  });

  return payload;
}

function buildHeader(runtime: DocumentPayload): Table {
  const docVersion = runtime.version ?? "1.0";
  const docStatus = runtime.status ?? "Ativo";

  return makeTable(
    [
      new TableRow({
        children: [
          makeCell({
            width: HEADER_TITLE_WIDTH,
            fill: C.purple,
            children: [
              paragraph([run("PROCEDIMENTO OPERACIONAL", { bold: true, color: C.white })], {
                alignment: AlignmentType.CENTER,
              }),
              paragraph([run(runtime.title, { color: C.purpleLight, size: 19 })], {
                alignment: AlignmentType.CENTER,
              }),
            ],
          }),
          makeCell({
            width: HEADER_ROW_1[1],
            fill: C.purpleLight,
            children: [
              paragraph([run("C\u00F3digo", { color: C.purple })], { alignment: AlignmentType.CENTER }),
              paragraph([run(runtime.documentCode, { bold: true, color: C.purple })], {
                alignment: AlignmentType.CENTER,
              }),
            ],
          }),
          makeCell({
            width: HEADER_ROW_1[2],
            fill: C.purpleLight,
            children: [
              paragraph([run("Vers\u00E3o", { color: C.purple })], { alignment: AlignmentType.CENTER }),
              paragraph([run(docVersion, { bold: true, color: C.purple })], {
                alignment: AlignmentType.CENTER,
              }),
            ],
          }),
        ],
      }),
      new TableRow({
        children: [
          makeCell({
            width: HEADER_TITLE_WIDTH,
            fill: C.teal,
            children: [paragraph([run(runtime.documentType, { bold: true, color: C.white })], { alignment: AlignmentType.CENTER })],
          }),
          makeCell({
            width: HEADER_ROW_2[1],
            columnSpan: 2,
            fill: C.tealLight,
            children: [
              paragraph([run("Status", { color: C.teal })], { alignment: AlignmentType.CENTER }),
              paragraph([run(docStatus, { bold: true, color: C.teal })], { alignment: AlignmentType.CENTER }),
            ],
          }),
        ],
      }),
    ],
    HEADER_ROW_1,
    {
      width: CONTENT_WIDTH,
      borders: tableBorder(BorderStyle.NONE),
    }
  );
}

function buildFooter(runtime: DocumentPayload): Footer {
  const elaboradoPor = runtime.metadata?.elaboradoPor
    ?? (isObject(runtime.values.identificacao) ? asString(runtime.values.identificacao.elaboradoPor) ?? "\u2014" : "\u2014");

  return new Footer({
    children: [
      new Paragraph({
        alignment: AlignmentType.CENTER,
        border: {
          top: { style: BorderStyle.SINGLE, size: 4, color: C.grayMid },
        },
        children: [
          run(`Elaborado por: ${elaboradoPor}  |  P\u00E1gina `, {
            font: DEFAULT_FONT,
            size: DEFAULT_FONT_SIZE,
          }),
          ...(makePageNumberField() as any[]),
        ],
      }),
    ],
  });
}

function buildIdentificationSection(runtime: DocumentPayload): (Paragraph | Table)[] {
  if (!runtime.metadata) {
    return [];
  }
  const meta = runtime.metadata;
  const widths = [2200, 2480, 2200, 2480] as const;

  return [
    paragraph([]),
    makeTable(
      [
        new TableRow({
          children: [
            makeCell({
              width: CONTENT_WIDTH,
              columnSpan: 4,
              fill: C.purple,
              children: [
                paragraph([run("1 \u2014 IDENTIFICA\u00C7\u00C3O", { bold: true, color: C.white })]),
              ],
            }),
          ],
        }),
        new TableRow({
          children: [
            makeCell({
              width: widths[0],
              fill: C.purpleLight,
              children: [paragraph([run("Elaborado por", { bold: true })])],
            }),
            makeCell({
              width: widths[1],
              children: [paragraph([run(meta.elaboradoPor || "\u2014")])],
            }),
            makeCell({
              width: widths[2],
              fill: C.purpleLight,
              children: [paragraph([run("Aprovado por", { bold: true })])],
            }),
            makeCell({
              width: widths[3],
              children: [paragraph([run(meta.aprovadoPor || "\u2014")])],
            }),
          ],
        }),
        new TableRow({
          children: [
            makeCell({
              width: widths[0],
              fill: C.purpleLight,
              children: [paragraph([run("Data de cria\u00E7\u00E3o", { bold: true })])],
            }),
            makeCell({
              width: widths[1],
              children: [paragraph([run(meta.createdAt || "\u2014")])],
            }),
            makeCell({
              width: widths[2],
              fill: C.purpleLight,
              children: [paragraph([run("Data de aprova\u00E7\u00E3o", { bold: true })])],
            }),
            makeCell({
              width: widths[3],
              children: [paragraph([run(meta.approvedAt || "\u2014")])],
            }),
          ],
        }),
      ],
      widths,
      { width: CONTENT_WIDTH, borders: tableBorder(BorderStyle.NONE) },
    ),
  ];
}

function buildRevisionHistorySection(runtime: DocumentPayload): (Paragraph | Table)[] {
  if (!runtime.revisions || runtime.revisions.length === 0) {
    return [];
  }
  const widths = [1400, 1800, 4360, 1800] as const;

  return [
    paragraph([]),
    makeTable(
      [
        new TableRow({
          children: [
            makeCell({
              width: CONTENT_WIDTH,
              columnSpan: 4,
              fill: C.gray,
              children: [
                paragraph([run("10 \u2014 HIST\u00D3RICO DE REVIS\u00D5ES", { bold: true, color: C.white })]),
              ],
            }),
          ],
        }),
        new TableRow({
          children: [
            makeCell({
              width: widths[0],
              fill: C.grayLight,
              children: [paragraph([run("Vers\u00E3o", { bold: true })])],
            }),
            makeCell({
              width: widths[1],
              fill: C.grayLight,
              children: [paragraph([run("Data", { bold: true })])],
            }),
            makeCell({
              width: widths[2],
              fill: C.grayLight,
              children: [paragraph([run("O que foi alterado", { bold: true })])],
            }),
            makeCell({
              width: widths[3],
              fill: C.grayLight,
              children: [paragraph([run("Por", { bold: true })])],
            }),
          ],
        }),
        ...runtime.revisions.map(
          (rev) =>
            new TableRow({
              children: [
                makeCell({
                  width: widths[0],
                  children: [paragraph([run(rev.versao || "\u2014")])],
                }),
                makeCell({
                  width: widths[1],
                  children: [paragraph([run(rev.data || "\u2014")])],
                }),
                makeCell({
                  width: widths[2],
                  children: [paragraph([run(rev.descricao || "\u2014")])],
                }),
                makeCell({
                  width: widths[3],
                  children: [paragraph([run(rev.por || "\u2014")])],
                }),
              ],
            }),
        ),
      ],
      widths,
      { width: CONTENT_WIDTH, borders: tableBorder(BorderStyle.NONE) },
    ),
  ];
}

export async function generateDocx(payload: unknown): Promise<Uint8Array> {
  const runtime = normalizeDocumentPayload(payload);

  const doc = new Document({
    sections: [
      {
        footers: {
          default: buildFooter(runtime),
        },
        properties: {
          page: {
            size: { width: PAGE_WIDTH, height: PAGE_HEIGHT },
            margin: {
              top: PAGE_MARGIN,
              right: PAGE_MARGIN,
              bottom: PAGE_MARGIN,
              left: PAGE_MARGIN,
              header: PAGE_MARGIN,
              footer: PAGE_MARGIN,
              gutter: 0,
            },
          },
        },
        children: [
          buildHeader(runtime),
          ...buildIdentificationSection(runtime),
          ...runtime.schema.sections.flatMap((section) => renderSection(section, runtime.values[section.key] ?? {})),
          ...buildRevisionHistorySection(runtime),
        ],
      },
    ],
    features: {
      updateFields: true,
    },
    styles: {
      default: {
        document: {
          run: {
            font: DEFAULT_FONT,
            size: DEFAULT_FONT_SIZE,
          },
        },
      },
    },
  });

  return Packer.toBuffer(doc);
}
