import { Paragraph, Table, TableCell, TableRow, TextRun, WidthType } from "docx";
import type { InlineRun, MDDMBlock } from "./types.js";

const HEADER_FILL = "F9F3F3";

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

function renderCell(block: MDDMBlock): TableCell {
  const runs = Array.isArray(block.children) && block.children.every(isInlineRun) ? block.children : [];
  return new TableCell({
    children: [renderInlineRuns(runs)],
  });
}

function renderHeaderCell(column: unknown): TableCell {
  const label = isObject(column) && typeof column.label === "string" ? column.label : "";
  return new TableCell({
    shading: {
      fill: HEADER_FILL,
    },
    children: [new Paragraph({ children: [new TextRun({ text: label, bold: true })] })],
  });
}

export function renderDataTable(block: MDDMBlock): Table {
  const columns = Array.isArray(block.props.columns) ? block.props.columns : [];
  const rows = Array.isArray(block.children) && block.children.every(isMDDMBlock) ? block.children : [];

  const headerRow = new TableRow({
    tableHeader: true,
    children: columns.map(renderHeaderCell),
  });

  const bodyRows = rows.map((row) => {
    const cells = Array.isArray(row.children) ? row.children : [];
    return new TableRow({
      children: cells.map((cell) => renderCell(cell as MDDMBlock)),
    });
  });

  return new Table({
    width: {
      size: 100,
      type: WidthType.PERCENTAGE,
    },
    rows: [headerRow, ...bodyRows],
  });
}
