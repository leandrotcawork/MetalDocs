import { BorderStyle, Paragraph, Table, TableCell, TableRow, TextRun, UnderlineType } from "docx";
import { CONTENT_WIDTH, cellBorder, fieldParagraph, makeCell, makeTable, paragraph, tableBorder } from "../runtime/docx.js";
import type { InlineRun, MDDMBlock, MDDMTemplateTheme } from "./types.js";

const ONE_COLUMN_WIDTHS = [30, 70] as const;
const TWO_COLUMN_WIDTHS = [22, 28, 22, 28] as const;

function isObject(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}

function isInlineRun(value: unknown): value is InlineRun {
  return isObject(value) && typeof value.text === "string";
}

function isInlineRunArray(value: unknown): value is InlineRun[] {
  return Array.isArray(value) && value.every(isInlineRun);
}

function isMDDMBlock(value: unknown): value is MDDMBlock {
  return isObject(value) && typeof value.type === "string" && isObject(value.props);
}

function normalizeWidthParts(parts: readonly number[], totalWidth = CONTENT_WIDTH): number[] {
  const totalParts = parts.reduce((sum, part) => sum + part, 0);
  if (totalParts <= 0) {
    return parts.map(() => 0);
  }
  const rawWidths = parts.map((part) => Math.floor((totalWidth * part) / totalParts));
  const consumed = rawWidths.reduce((sum, width) => sum + width, 0);
  if (rawWidths.length > 0) {
    rawWidths[rawWidths.length - 1] += totalWidth - consumed;
  }
  return rawWidths;
}

function inlineRunsToParagraph(runs: InlineRun[]): Paragraph {
  return paragraph(runs.map((runValue) => runToTextRun(runValue)));
}

function runToTextRun(runValue: InlineRun): TextRun {
  const marks = new Set(
    (runValue.marks ?? [])
      .filter((mark): mark is { type: string } => isObject(mark) && typeof mark.type === "string")
      .map((mark) => mark.type),
  );
  return new TextRun({
    text: runValue.text,
    bold: marks.has("bold"),
    italics: marks.has("italic"),
    underline: marks.has("underline") ? { type: UnderlineType.SINGLE } : undefined,
    strike: marks.has("strike"),
  });
}

function renderBlockContent(block: MDDMBlock): Paragraph[] {
  if (block.type === "divider") {
    return [paragraph([])];
  }

  if (isInlineRunArray(block.children)) {
    return [inlineRunsToParagraph(block.children)];
  }

  if (Array.isArray(block.children) && block.children.every(isMDDMBlock)) {
    return block.children.flatMap((child) => renderBlockContent(child));
  }

  return [paragraph([])];
}

function renderFieldValue(block: MDDMBlock): Paragraph[] {
  const valueMode = typeof block.props.valueMode === "string" ? block.props.valueMode : "inline";

  if (valueMode === "inline") {
    if (isInlineRunArray(block.children)) {
      return [inlineRunsToParagraph(block.children)];
    }
    return [paragraph([])];
  }

  if (Array.isArray(block.children) && block.children.every(isMDDMBlock)) {
    const paragraphs = block.children.flatMap((child) => renderBlockContent(child));
    return paragraphs.length > 0 ? paragraphs : [paragraph([])];
  }

  return [paragraph([])];
}

function fieldToCells(
  field: MDDMBlock,
  labelWidth: number,
  valueWidth: number,
  theme: Required<MDDMTemplateTheme>,
): [TableCell, TableCell] {
  const label = typeof field.props.label === "string" ? field.props.label : "";

  return [
    makeCell({
      width: labelWidth,
      fill: theme.accentLight,
      borders: cellBorder(BorderStyle.SINGLE, normalizeHex(theme.accentBorder)),
      verticalAlign: "top",
      children: [fieldParagraph(label, { bold: true, color: theme.accentDark, size: 18 })],
    }),
    makeCell({
      width: valueWidth,
      borders: cellBorder(BorderStyle.SINGLE, normalizeHex(theme.accentBorder)),
      verticalAlign: "top",
      children: renderFieldValue(field),
    }),
  ];
}

function blankFieldCells(
  labelWidth: number,
  valueWidth: number,
  theme: Required<MDDMTemplateTheme>,
): [TableCell, TableCell] {
  return [
    makeCell({
      width: labelWidth,
      fill: theme.accentLight,
      borders: cellBorder(BorderStyle.SINGLE, normalizeHex(theme.accentBorder)),
      verticalAlign: "top",
      children: [fieldParagraph("", { bold: true, color: theme.accentDark, size: 18 })],
    }),
    makeCell({
      width: valueWidth,
      borders: cellBorder(BorderStyle.SINGLE, normalizeHex(theme.accentBorder)),
      verticalAlign: "top",
      children: [paragraph([])],
    }),
  ];
}

function renderFieldRow(field: MDDMBlock, theme: Required<MDDMTemplateTheme>): TableRow {
  const [labelWidth, valueWidth] = normalizeWidthParts(ONE_COLUMN_WIDTHS) as [number, number];
  const [labelCell, valueCell] = fieldToCells(field, labelWidth, valueWidth, theme);
  return new TableRow({ children: [labelCell, valueCell] });
}

function renderFieldPairRow(left: MDDMBlock, theme: Required<MDDMTemplateTheme>, right?: MDDMBlock): TableRow {
  const [labelWidth, valueWidth] = normalizeWidthParts([22, 28], CONTENT_WIDTH / 2) as [number, number];
  const leftCells = fieldToCells(left, labelWidth, valueWidth, theme);
  const rightCells = right ? fieldToCells(right, labelWidth, valueWidth, theme) : blankFieldCells(labelWidth, valueWidth, theme);

  return new TableRow({
    children: [leftCells[0], leftCells[1], rightCells[0], rightCells[1]],
  });
}

function resolveTheme(theme?: MDDMTemplateTheme): Required<MDDMTemplateTheme> {
  return {
    accent: theme?.accent ?? "6B1F2A",
    accentLight: theme?.accentLight ?? "F9F3F3",
    accentDark: theme?.accentDark ?? "3E1018",
    accentBorder: theme?.accentBorder ?? "DFC8C8",
  };
}

function normalizeHex(value: string): string {
  return value.replace(/^#/, "").toUpperCase();
}

export function renderFieldGroup(block: MDDMBlock, theme?: MDDMTemplateTheme): Table {
  const resolvedTheme = resolveTheme(theme);
  const columns = typeof block.props.columns === "number" && block.props.columns === 2 ? 2 : 1;
  const fields = Array.isArray(block.children) && block.children.every(isMDDMBlock) ? block.children : [];
  const rows: TableRow[] = [];

  if (columns === 2) {
    for (let index = 0; index < fields.length; index += 2) {
      rows.push(renderFieldPairRow(fields[index], resolvedTheme, fields[index + 1]));
    }
    if (fields.length === 0) {
      rows.push(renderFieldPairRow({ id: "", type: "field", props: {}, children: [] }, resolvedTheme, undefined));
    }
  } else {
    for (const field of fields) {
      rows.push(renderFieldRow(field, resolvedTheme));
    }
    if (fields.length === 0) {
      rows.push(renderFieldRow({ id: "", type: "field", props: {}, children: [] }, resolvedTheme));
    }
  }

  const columnWidths =
    columns === 2
      ? (normalizeWidthParts(TWO_COLUMN_WIDTHS) as [number, number, number, number])
      : (normalizeWidthParts(ONE_COLUMN_WIDTHS) as [number, number]);

  return makeTable(rows, columnWidths, {
    width: CONTENT_WIDTH,
    borders: tableBorder(BorderStyle.NONE, normalizeHex(resolvedTheme.accentBorder)),
  });
}
