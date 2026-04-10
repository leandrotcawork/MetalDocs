import { BorderStyle, Paragraph, ShadingType, Table, TableCell, TableRow, TextRun, WidthType } from "docx";
import { cellBorder, tableBorder } from "../runtime/docx.js";
import type { InlineRun, MDDMBlock, MDDMTemplateTheme } from "./types.js";

function isObject(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}

function isInlineRun(value: unknown): value is InlineRun {
  return isObject(value) && typeof value.text === "string";
}

function isMDDMBlock(value: unknown): value is MDDMBlock {
  return isObject(value) && typeof value.type === "string" && isObject(value.props);
}

function renderInlineRuns(runs: InlineRun[]): Paragraph {
  return new Paragraph({
    children: runs.map((run) => new TextRun({ text: run.text })),
  });
}

function renderCell(block: MDDMBlock, theme: Required<MDDMTemplateTheme>): TableCell {
  const runs = Array.isArray(block.children) && block.children.every(isInlineRun) ? block.children : [];
  return new TableCell({
    borders: cellBorder(BorderStyle.SINGLE, normalizeHex(theme.accentBorder)),
    children: [renderInlineRuns(runs)],
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

function renderHeaderCell(column: unknown, theme: Required<MDDMTemplateTheme>): TableCell {
  const label = isObject(column) && typeof column.label === "string" ? column.label : "";
  return new TableCell({
    shading: {
      type: ShadingType.CLEAR,
      fill: normalizeHex(theme.accentLight),
    },
    borders: cellBorder(BorderStyle.SINGLE, normalizeHex(theme.accentBorder)),
    children: [new Paragraph({ children: [new TextRun({ text: label, bold: true, color: normalizeHex(theme.accentDark) })] })],
  });
}

export function renderDataTable(block: MDDMBlock, theme?: MDDMTemplateTheme): Table {
  const resolvedTheme = resolveTheme(theme);
  const columns = Array.isArray(block.props.columns) ? block.props.columns : [];
  const rows = Array.isArray(block.children) && block.children.every(isMDDMBlock) ? block.children : [];

  const headerRow = new TableRow({
    tableHeader: true,
    children: columns.map((column) => renderHeaderCell(column, resolvedTheme)),
  });

  const bodyRows = rows.map((row) => {
    const cells = Array.isArray(row.children) ? row.children : [];
    return new TableRow({
      children: cells.map((cell) => renderCell(cell as MDDMBlock, resolvedTheme)),
    });
  });

  return new Table({
    width: {
      size: 100,
      type: WidthType.PERCENTAGE,
    },
    borders: tableBorder(BorderStyle.SINGLE, normalizeHex(resolvedTheme.accentBorder)),
    rows: [headerRow, ...bodyRows],
  });
}
