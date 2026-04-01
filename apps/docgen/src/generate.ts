import { Document, Packer, Paragraph, TextRun } from "docx";
import { renderSection } from "./runtime/renderSection.js";
import type { DocumentPayload, DocumentTypeSchema, DocumentValues } from "./runtime/types.js";

function isObject(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}

function normalizeDocumentPayload(input: unknown): DocumentPayload {
  if (!isObject(input)) {
    throw new Error("DOCGEN_INVALID_PAYLOAD");
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
    throw new Error("DOCGEN_INVALID_PAYLOAD");
  }

  return {
    documentType,
    documentCode,
    title,
    schema: { sections: schema.sections as DocumentTypeSchema["sections"] },
    values: values as DocumentValues,
  } satisfies DocumentPayload;
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
