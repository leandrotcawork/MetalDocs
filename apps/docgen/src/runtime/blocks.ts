import {
  AlignmentType,
  ImageRun,
  Paragraph,
  Table,
  TableCell,
  TableRow,
  TextRun,
  UnderlineType,
  WidthType,
} from "docx";
import type { RichBlock, RichTextRun } from "./types.js";

function isObject(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}

function asText(value: unknown): string {
  if (value === null || value === undefined) {
    return "N/A";
  }
  if (typeof value === "boolean") {
    return value ? "Sim" : "Nao";
  }
  if (typeof value === "number") {
    return String(value);
  }
  if (typeof value === "string") {
    return value;
  }
  if (value instanceof Date) {
    return value.toISOString().slice(0, 10);
  }
  return JSON.stringify(value);
}

function renderRuns(runs: RichTextRun[]): TextRun[] {
  return runs.map(
    (run) =>
      new TextRun({
        text: run.text,
        bold: run.bold,
        italics: run.italic,
        underline: run.underline ? { type: UnderlineType.SINGLE } : undefined,
        color: run.color?.replace(/^#/, ""),
      })
  );
}

function decodeBase64(data: string): Buffer {
  const payload = data.startsWith("data:") ? data.split(",")[1] ?? "" : data;
  return Buffer.from(payload, "base64");
}

function imageTypeFromMimeType(mimeType?: string): "jpg" | "png" | "gif" | "bmp" {
  switch (mimeType) {
    case "image/jpeg":
    case "image/jpg":
      return "jpg";
    case "image/gif":
      return "gif";
    case "image/bmp":
      return "bmp";
    default:
      return "png";
  }
}

export function renderRichBlocks(blocks: RichBlock[]): Array<Paragraph | Table> {
  return blocks.flatMap((block) => {
    if (block.type === "text") {
      return [
        new Paragraph({
          children: renderRuns(block.runs),
        }),
      ];
    }

    if (block.type === "image") {
      return [
        new Paragraph({
          alignment: AlignmentType.CENTER,
          children: [
            new ImageRun({
              type: imageTypeFromMimeType(block.mimeType),
              data: decodeBase64(block.data),
              transformation: {
                width: block.width ?? 160,
                height: block.height ?? 160,
              },
              altText: block.altText ? { name: block.altText, title: block.altText } : undefined,
            }),
          ],
        }),
      ];
    }

    if (block.type === "table") {
      const rows = block.rows.map(
        (row, index) =>
          new TableRow({
            children: row.map(
              (cell) =>
                new TableCell({
                  children: [
                    new Paragraph({
                      children: [
                        new TextRun({
                          text: asText(cell),
                          bold: block.header && index === 0,
                        }),
                      ],
                    }),
                  ],
                })
            ),
          })
      );

      return [
        new Table({
          width: { size: 100, type: WidthType.PERCENTAGE },
          rows,
        }),
      ];
    }

    const rows = block.items.map((item, index) =>
      new Paragraph({
        bullet: block.ordered ? undefined : { level: 0 },
        children: [new TextRun({ text: block.ordered ? `${index + 1}. ${item}` : item })],
      })
    );

    return rows;
  });
}

export function renderScalarValue(value: unknown): string {
  return asText(value);
}

export function isRichBlockArray(value: unknown): value is RichBlock[] {
  return Array.isArray(value) && value.every((item) => isObject(item) && typeof item.type === "string");
}
