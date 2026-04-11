import { TableRow } from "docx";
import type { LayoutTokens } from "../../layout-ir";
import type { MDDMBlock } from "../../../adapter";
import { emitDataTableCell } from "./data-table-cell";

function isCellBlock(child: unknown): child is MDDMBlock {
  return typeof child === "object" && child !== null && (child as MDDMBlock).type === "dataTableCell";
}

export function emitDataTableRow(block: MDDMBlock, tokens: LayoutTokens): TableRow {
  const allChildren = (block.children ?? []) as unknown[];
  const cells = allChildren.filter(isCellBlock).map((c) => emitDataTableCell(c as MDDMBlock, tokens));
  return new TableRow({ children: cells });
}
