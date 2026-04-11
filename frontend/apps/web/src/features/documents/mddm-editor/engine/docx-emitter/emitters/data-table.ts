import {
  Table,
  TableRow,
  TableCell,
  Paragraph,
  TextRun,
  WidthType,
  BorderStyle,
} from "docx";
import type { LayoutTokens } from "../../layout-ir";
import type { MDDMBlock } from "../../../adapter";
import { emitDataTableRow } from "./data-table-row";
import { ptToHalfPt, mmToTwip } from "../../helpers/units";

type ColumnSpec = { key: string; label: string };

function readColumns(props: Record<string, unknown>): ColumnSpec[] {
  const columns = props.columns;
  if (!Array.isArray(columns)) return [];
  const out: ColumnSpec[] = [];
  for (const column of columns) {
    if (!column || typeof column !== "object") continue;
    const key = typeof (column as { key?: unknown }).key === "string" ? (column as { key: string }).key : "";
    const label = typeof (column as { label?: unknown }).label === "string" ? (column as { label: string }).label : "";
    if (key && label) out.push({ key, label });
  }
  return out;
}

function hexToFill(hex: string): string {
  return hex.replace(/^#/, "").toUpperCase();
}

function isRowBlock(child: unknown): child is MDDMBlock {
  return typeof child === "object" && child !== null && (child as MDDMBlock).type === "dataTableRow";
}

function buildHeaderRow(columns: ColumnSpec[], tokens: LayoutTokens): TableRow {
  const headerFill = hexToFill(tokens.theme.accentLight);
  const borderColor = hexToFill(tokens.theme.accentBorder);
  const borders = {
    top:    { style: BorderStyle.SINGLE, size: 4, color: borderColor },
    bottom: { style: BorderStyle.SINGLE, size: 4, color: borderColor },
    left:   { style: BorderStyle.SINGLE, size: 4, color: borderColor },
    right:  { style: BorderStyle.SINGLE, size: 4, color: borderColor },
  };

  const cells = columns.map((col) => new TableCell({
    shading: { fill: headerFill, type: "clear", color: "auto" },
    borders,
    children: [
      new Paragraph({
        children: [
          new TextRun({
            text: col.label,
            bold: true,
            size: ptToHalfPt(tokens.typography.baseSizePt),
            font: tokens.typography.exportFont,
          }),
        ],
      }),
    ],
  }));

  return new TableRow({ children: cells });
}

export function emitDataTable(block: MDDMBlock, tokens: LayoutTokens): Table[] {
  const columns = readColumns(block.props as Record<string, unknown>);
  const rowChildren = ((block.children ?? []) as unknown[]).filter(isRowBlock) as MDDMBlock[];

  const headerRow = columns.length > 0 ? [buildHeaderRow(columns, tokens)] : [];
  const dataRows = rowChildren.map((r) => emitDataTableRow(r, tokens));
  const rows = [...headerRow, ...dataRows];

  // docx's Table constructor throws "Invalid array length" when rows is empty
  // (columnWidths uses Math.max(...[]) = -Infinity). Guard with a placeholder row.
  const safeRows =
    rows.length > 0
      ? rows
      : [new TableRow({ children: [new TableCell({ children: [new Paragraph({ children: [] })] })] })];

  const tableOptions = {
    width: { size: mmToTwip(tokens.page.contentWidthMm), type: WidthType.DXA },
    rows: safeRows,
  } as const;

  const table = new Table(tableOptions);
  // Back-patch so tests can introspect via (out[0] as any).options.rows
  (table as unknown as { options: { rows: typeof safeRows } }).options = { rows: safeRows };

  return [table];
}
