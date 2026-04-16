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
import type { MDDMBlock } from "../../shared/adapter";
import { ptToHalfPt, mmToTwip } from "../../shared/helpers/units";
import { hexToFill } from "../../shared/helpers/color";
import { interpretDataTable } from "../../layout-interpreter";

type RawCell = { type?: string; text?: string; styles?: Record<string, boolean> };
type RawRow = { cells?: RawCell[][] };
type RawTableContent = {
  type?: string;
  columnWidths?: (number | null)[];
  headerRows?: number;
  rows?: RawRow[];
};

function readTableContent(content: unknown): RawTableContent | null {
  if (
    typeof content !== "object" ||
    content === null ||
    (content as any).type !== "tableContent"
  ) {
    return null;
  }
  return content as RawTableContent;
}

function cellToTextRun(cells: RawCell[][], tokens: LayoutTokens): TextRun[] {
  return cells.flatMap((cell) =>
    cell.map((run) =>
      new TextRun({
        text: run.text ?? "",
        bold: run.styles?.bold === true,
        italics: run.styles?.italic === true,
        underline: run.styles?.underline === true ? {} : undefined,
        size: ptToHalfPt(tokens.typography.baseSizePt),
        font: tokens.typography.exportFont,
      })
    )
  );
}

function buildRow(
  rowData: RawRow,
  tokens: LayoutTokens,
  isHeader: boolean,
  headerFill: string,
  borderColor: string,
): TableRow {
  const borders = {
    top:    { style: BorderStyle.SINGLE, size: 4, color: borderColor },
    bottom: { style: BorderStyle.SINGLE, size: 4, color: borderColor },
    left:   { style: BorderStyle.SINGLE, size: 4, color: borderColor },
    right:  { style: BorderStyle.SINGLE, size: 4, color: borderColor },
  };

  const cellsData = rowData.cells ?? [];
  const tableCells = cellsData.map((cellContent) => {
    const runs = cellToTextRun([cellContent], tokens);
    const shading = isHeader
      ? { fill: headerFill, type: "clear" as const, color: "auto" }
      : undefined;

    return new TableCell({
      borders,
      shading,
      children: [
        new Paragraph({
          children: runs.length > 0 ? runs : [new TextRun({ text: "" })],
        }),
      ],
    });
  });

  if (tableCells.length === 0) {
    tableCells.push(new TableCell({ children: [new Paragraph({ children: [] })] }));
  }

  return new TableRow({ children: tableCells, tableHeader: isHeader });
}

export function emitDataTable(block: MDDMBlock, tokens: LayoutTokens): Table[] {
  const vm = interpretDataTable(
    { props: block.props as Record<string, unknown> },
    tokens,
  );
  const headerFill = hexToFill(vm.headerBg);
  const borderColor = hexToFill(vm.cellBorderColor);

  const tableContent = readTableContent(block.content);

  if (!tableContent) {
    // Fallback: empty table when no tableContent (shouldn't happen after migration)
    const emptyTable = new Table({
      width: { size: mmToTwip(tokens.page.contentWidthMm), type: WidthType.DXA },
      rows: [new TableRow({ children: [new TableCell({ children: [new Paragraph({ children: [] })] })] })],
    });
    return [emptyTable];
  }

  const headerRows = tableContent.headerRows ?? 1;
  const rows = tableContent.rows ?? [];

  const builtRows = rows.map((row, i) => buildRow(row, tokens, i < headerRows, headerFill, borderColor));

  const safeRows =
    builtRows.length > 0
      ? builtRows
      : [new TableRow({ children: [new TableCell({ children: [new Paragraph({ children: [] })] })] })];

  const table = new Table({
    width: { size: mmToTwip(tokens.page.contentWidthMm), type: WidthType.DXA },
    rows: safeRows,
  });

  // Back-patch so tests can introspect via (out[0] as any).options.rows
  (table as unknown as { options: { rows: typeof safeRows } }).options = { rows: safeRows };

  return [table];
}

