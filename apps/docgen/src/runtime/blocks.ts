import { AlignmentType, BorderStyle, ImageRun, Paragraph, Table, TableRow, TextRun, UnderlineType } from "docx";
import { CONTENT_WIDTH, makeCell, makeTable, paragraph, run, tableBorder } from "./docx.js";
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
    (richRun) =>
      new TextRun({
        text: richRun.text,
        bold: richRun.bold,
        italics: richRun.italic,
        underline: richRun.underline ? { type: UnderlineType.SINGLE } : undefined,
        color: richRun.color?.replace(/^#/, ""),
        font: "Arial",
        size: 20,
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

function renderTableBlock(block: Extract<RichBlock, { type: "table" }>): Table {
  const columnCount = block.rows.reduce((max, row) => Math.max(max, row.length), 0);
  const columnWidth = Math.floor(CONTENT_WIDTH / Math.max(columnCount, 1));
  const columnWidths = Array.from({ length: columnCount || 1 }, () => columnWidth);

  const rows = block.rows.map(
    (row, index) =>
      new TableRow({
        children: row.map((cell) =>
          makeCell({
            width: columnWidth,
            children: [
              paragraph([
                new TextRun({
                  text: asText(cell),
                  bold: block.header && index === 0,
                  font: "Arial",
                  size: 20,
                }),
              ]),
            ],
          })
        ),
      })
  );

  return makeTable(rows, columnWidths, {
    width: CONTENT_WIDTH,
    borders: tableBorder(BorderStyle.NONE),
  });
}

export function renderRichBlocks(blocks: RichBlock[]): Array<Paragraph | Table> {
  return blocks.flatMap((block) => {
    if (block.type === "text") {
      return [
        paragraph(renderRuns(block.runs)),
      ];
    }

    if (block.type === "image") {
      return [
        paragraph([
          new ImageRun({
            type: imageTypeFromMimeType(block.mimeType),
            data: decodeBase64(block.data),
            transformation: {
              width: block.width ?? 160,
              height: block.height ?? 160,
            },
            altText: block.altText ? { name: block.altText, title: block.altText } : undefined,
          }),
        ], {
          alignment: AlignmentType.CENTER,
        }),
      ];
    }

    if (block.type === "table") {
      return [renderTableBlock(block)];
    }

    return block.items.map((item, index) =>
      paragraph([
        run(block.ordered ? `${index + 1}. ${item}` : item),
      ], {
        bullet: block.ordered ? undefined : { level: 0 },
      })
    );
  });
}

export function renderScalarValue(value: unknown): string {
  return asText(value);
}

export function isRichBlockArray(value: unknown): value is RichBlock[] {
  return Array.isArray(value) && value.every((item) => isObject(item) && typeof item.type === "string");
}
