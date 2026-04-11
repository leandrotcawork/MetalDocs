import { TableRow, TableCell, Paragraph } from "docx";
import type { LayoutTokens } from "../../layout-ir";
import type { MDDMBlock } from "../../../adapter";
import { emitDataTableCell } from "./data-table-cell";

function isCellBlock(child: unknown): child is MDDMBlock {
  return typeof child === "object" && child !== null && (child as MDDMBlock).type === "dataTableCell";
}

// Symmetric with the empty-rows guard in data-table.ts: docx serialization
// may crash on a row with zero cells. Use a placeholder when none are found.
const EMPTY_CELL = new TableCell({ children: [new Paragraph({ children: [] })] });

export function emitDataTableRow(block: MDDMBlock, tokens: LayoutTokens): TableRow {
  const allChildren = (block.children ?? []) as unknown[];
  const cells = allChildren.filter(isCellBlock).map((c) => emitDataTableCell(c as MDDMBlock, tokens));
  return new TableRow({ children: cells.length > 0 ? cells : [EMPTY_CELL] });
}
