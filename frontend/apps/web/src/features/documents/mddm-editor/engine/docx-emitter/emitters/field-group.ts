import { Table, TableRow, TableCell, WidthType, BorderStyle } from "docx";
import type { LayoutTokens } from "../../layout-ir";
import type { MDDMBlock } from "../../../adapter";
import { emitField } from "./field";

const NO_BORDER = {
  top: { style: BorderStyle.NONE, size: 0, color: "auto" },
  bottom: { style: BorderStyle.NONE, size: 0, color: "auto" },
  left: { style: BorderStyle.NONE, size: 0, color: "auto" },
  right: { style: BorderStyle.NONE, size: 0, color: "auto" },
} as const;

function attachOptions<T, O extends object>(instance: T, options: O): T {
  (instance as unknown as { options: O }).options = options;
  return instance;
}

function isFieldBlock(child: unknown): child is MDDMBlock {
  return (
    typeof child === "object" &&
    child !== null &&
    (child as MDDMBlock).type === "field"
  );
}

export function emitFieldGroup(block: MDDMBlock, tokens: LayoutTokens): Table[] {
  const columns = (block.props as { columns?: number }).columns === 1 ? 1 : 2;
  const allChildren = (block.children ?? []) as unknown[];
  const fields = allChildren.filter(isFieldBlock) as MDDMBlock[];

  if (fields.length === 0) {
    const emptyCellOptions = { borders: NO_BORDER, children: [] };
    const emptyCell = attachOptions(new TableCell(emptyCellOptions), emptyCellOptions);
    const emptyRowOptions = { children: [emptyCell] };
    const emptyRow = attachOptions(new TableRow(emptyRowOptions), emptyRowOptions);
    const emptyTableOptions = {
      width: { size: 100, type: WidthType.PERCENTAGE },
      rows: [emptyRow],
    };
    return [attachOptions(new Table(emptyTableOptions), emptyTableOptions)];
  }

  const cellWidthPct = Math.floor(5000 / columns);
  const rows: TableRow[] = [];

  for (let i = 0; i < fields.length; i += columns) {
    const rowCells: TableCell[] = [];
    for (let c = 0; c < columns; c++) {
      const field = fields[i + c];
      const fieldTable = field ? emitField(field, tokens) : [];
      const cellOptions = {
        width: { size: cellWidthPct, type: WidthType.PERCENTAGE },
        borders: NO_BORDER,
        children: fieldTable,
      };
      rowCells.push(attachOptions(new TableCell(cellOptions), cellOptions));
    }
    const rowOptions = { children: rowCells };
    rows.push(attachOptions(new TableRow(rowOptions), rowOptions));
  }

  const tableOptions = {
    width: { size: 100, type: WidthType.PERCENTAGE },
    rows,
  };
  return [attachOptions(new Table(tableOptions), tableOptions)];
}
