import { BorderStyle, Paragraph, Table, TableCell, TableRow, WidthType } from "docx"
import type { ResolvedAsset } from "../asset-resolver"
import type {
  Table as ExportTable,
  TableCell as ExportTableCell,
  TableRow as ExportTableRow,
} from "../export-node"
import type { LayoutTokens } from "../layout-tokens"
import { hexToFill, type EmitBlocks } from "./helpers"

export function emitTableCell(
  node: ExportTableCell,
  tokens: LayoutTokens,
  assetMap: ReadonlyMap<string, ResolvedAsset>,
  emitBlocks: EmitBlocks,
): TableCell {
  const borderColor = hexToFill(tokens.theme.accentBorder)
  const children = emitBlocks(node.children, tokens, assetMap)

  return new TableCell({
    columnSpan: node.colspan,
    rowSpan: node.rowspan,
    shading: node.isHeader ? { fill: hexToFill(tokens.theme.accentLight), type: "clear", color: "auto" } : undefined,
    borders: {
      top: { style: BorderStyle.SINGLE, size: 4, color: borderColor },
      bottom: { style: BorderStyle.SINGLE, size: 4, color: borderColor },
      left: { style: BorderStyle.SINGLE, size: 4, color: borderColor },
      right: { style: BorderStyle.SINGLE, size: 4, color: borderColor },
    },
    children: children.length > 0 ? children : [new Paragraph({ children: [] })],
  })
}

export function emitTableRow(
  node: ExportTableRow,
  tokens: LayoutTokens,
  assetMap: ReadonlyMap<string, ResolvedAsset>,
  emitBlocks: EmitBlocks,
): TableRow {
  return new TableRow({
    children: node.cells.map((cell) => emitTableCell(cell, tokens, assetMap, emitBlocks)),
  })
}

export function emitTable(
  node: ExportTable,
  tokens: LayoutTokens,
  assetMap: ReadonlyMap<string, ResolvedAsset>,
  emitBlocks: EmitBlocks,
): Table[] {
  return [
    new Table({
      width: { size: 100, type: WidthType.PERCENTAGE },
      rows: node.rows.map((row) => emitTableRow(row, tokens, assetMap, emitBlocks)),
    }),
  ]
}
